#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-ZNZ-systems/DeadDrop}"
BRANCH="${BRANCH:-master}"
REMOTE_DOCKER_BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/docker"
INSTALL_DIR="${INSTALL_DIR:-deaddrop}"
ENV_FILE=".env.prod"
DASHBOARD_PORT="${DASHBOARD_PORT:-8080}"
TOTAL_STEPS=7
STEP_NUM=0

info() {
  printf '%s\n' "$*"
}

warn() {
  printf 'WARN: %s\n' "$*" >&2
}

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

step() {
  STEP_NUM=$((STEP_NUM + 1))
  info ""
  info "Step ${STEP_NUM}/${TOTAL_STEPS}: $1"
  info "$2"
}

validate() {
  description="$1"
  shift
  if "$@"; then
    info "Validation: PASS - ${description}"
  else
    fail "Validation: FAIL - ${description}"
  fi
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif has_cmd sudo; then
    sudo "$@"
  else
    fail "This step requires root and sudo is not available: $*"
  fi
}

ensure_curl() {
  if has_cmd curl; then
    return 0
  fi
  warn "curl not found. Attempting to install curl."
  case "$(uname -s)" in
    Linux)
      if has_cmd apt-get; then
        as_root apt-get update
        as_root apt-get install -y curl
      elif has_cmd dnf; then
        as_root dnf install -y curl
      elif has_cmd yum; then
        as_root yum install -y curl
      elif has_cmd apk; then
        as_root apk add --no-cache curl
      else
        fail "Unsupported Linux package manager. Install curl manually."
      fi
      ;;
    *)
      fail "Unsupported OS for automatic curl install. Install curl manually."
      ;;
  esac
}

ensure_docker() {
  if has_cmd docker; then
    return 0
  fi

  info "Docker is not installed. Installing Docker..."
  case "$(uname -s)" in
    Linux)
      ensure_curl
      tmp_script="$(mktemp)"
      curl -fsSL https://get.docker.com -o "${tmp_script}"
      as_root sh "${tmp_script}"
      rm -f "${tmp_script}"
      if has_cmd systemctl; then
        as_root systemctl enable --now docker >/dev/null 2>&1 || true
      fi
      ;;
    Darwin)
      fail "Automatic Docker install is not supported on macOS. Install Docker Desktop and rerun."
      ;;
    *)
      fail "Unsupported OS for automatic Docker install."
      ;;
  esac
}

configure_docker_command() {
  if docker version >/dev/null 2>&1; then
    DOCKER_CMD=(docker)
    return 0
  fi
  if sudo docker version >/dev/null 2>&1; then
    DOCKER_CMD=(sudo docker)
    warn "Using sudo for Docker commands (current user lacks docker socket permissions)."
    return 0
  fi
  fail "Docker is installed but cannot be used by this user."
}

configure_compose_command() {
  if "${DOCKER_CMD[@]}" compose version >/dev/null 2>&1; then
    COMPOSE_CMD=("${DOCKER_CMD[@]}" compose)
    return 0
  fi
  if has_cmd docker-compose; then
    COMPOSE_CMD=(docker-compose)
    return 0
  fi
  fail "Docker Compose is not available."
}

download_stack_files() {
  info "Downloading production deployment files..."
  curl -fsSL "${REMOTE_DOCKER_BASE_URL}/docker-compose.prod.yml" -o docker-compose.prod.yml
  curl -fsSL "${REMOTE_DOCKER_BASE_URL}/Caddyfile" -o Caddyfile
  curl -fsSL "${REMOTE_DOCKER_BASE_URL}/.env.prod.example" -o .env.prod.example
}

download_source_archive() {
  info "Downloading source archive for local app build..."
  tmp_dir="$(mktemp -d)"
  archive_path="${tmp_dir}/repo.tgz"
  archive_url="https://github.com/${REPO}/archive/refs/heads/${BRANCH}.tar.gz"

  curl -fsSL "${archive_url}" -o "${archive_path}"
  tar -xzf "${archive_path}" -C "${tmp_dir}"

  extracted_root="$(find "${tmp_dir}" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  [ -n "${extracted_root}" ] || fail "Could not locate extracted source archive directory."

  rm -rf src
  mkdir -p src
  cp -R "${extracted_root}"/. src/
  rm -rf "${tmp_dir}"
}

normalize_compose_for_installer() {
  if grep -qE '^[[:space:]]*build:[[:space:]]*$' docker-compose.prod.yml; then
    if grep -qE '^[[:space:]]*context:[[:space:]]*\.\.[[:space:]]*$' docker-compose.prod.yml; then
      download_source_archive

      sed -i.bak -E \
        's#(^[[:space:]]*context:)[[:space:]]*\.\.[[:space:]]*$#\1 ./src#' \
        docker-compose.prod.yml
      sed -i.bak -E \
        's#(^[[:space:]]*dockerfile:)[[:space:]]*\./src/docker/Dockerfile[[:space:]]*$#\1 docker/Dockerfile#' \
        docker-compose.prod.yml
      sed -i.bak -E \
        's#(^[[:space:]]*dockerfile:)[[:space:]]*src/docker/Dockerfile[[:space:]]*$#\1 docker/Dockerfile#' \
        docker-compose.prod.yml
      rm -f docker-compose.prod.yml.bak
    fi
  fi

  # Expose dashboard on an explicit host port so users have a concrete URL.
  sed -i.bak -E \
    "s#(^[[:space:]]*-[[:space:]]*\")80:80(\"[[:space:]]*\$)#\\1${DASHBOARD_PORT}:80\\2#" \
    docker-compose.prod.yml
  sed -i.bak -E \
    's#(^[[:space:]]*-[[:space:]]*)SECURE_COOKIES=true([[:space:]]*$)#\1SECURE_COOKIES=${SECURE_COOKIES}\2#' \
    docker-compose.prod.yml
  sed -i.bak -E \
    's#(^[[:space:]]*-[[:space:]]*)BASE_URL=https://\$\{DOMAIN\}([[:space:]]*$)#\1BASE_URL=${BASE_URL}\2#' \
    docker-compose.prod.yml
  rm -f docker-compose.prod.yml.bak
}

upsert_env() {
  key="$1"
  value="$2"

  if grep -q "^${key}=" "${ENV_FILE}"; then
    sed -i.bak "s|^${key}=.*$|${key}=${value}|" "${ENV_FILE}"
    rm -f "${ENV_FILE}.bak"
  else
    printf '%s=%s\n' "${key}" "${value}" >>"${ENV_FILE}"
  fi
}

read_env_value() {
  key="$1"
  if [ ! -f "${ENV_FILE}" ]; then
    return 0
  fi
  awk -F= -v k="${key}" '$1 == k {sub(/^[^=]*=/, "", $0); print $0; exit}' "${ENV_FILE}"
}

random_password() {
  if has_cmd openssl; then
    openssl rand -hex 24
  else
    od -An -N24 -tx1 /dev/urandom | tr -d ' \n'
  fi
}

detect_local_ip() {
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  if [ -n "${ip}" ]; then
    echo "${ip}"
    return 0
  fi
  echo "localhost"
}

normalize_caddy_for_installer() {
  cat > Caddyfile <<'EOF'
:80 {
	reverse_proxy app:8080
}
EOF
}

mask_database_url() {
  echo "$1" | sed -E 's#(postgres://[^:]+:)[^@]+#\1***#'
}

seed_environment() {
  if [ ! -f "${ENV_FILE}" ]; then
    if db_volume_exists; then
      fail "Found existing DB volume '$(db_volume_name)' but no ${ENV_FILE}. This can cause password mismatch. Run: docker volume rm $(db_volume_name) (or restore your previous ${ENV_FILE}) and rerun installer."
    fi
    cp .env.prod.example "${ENV_FILE}"
    info "Created ${ENV_FILE} from template."
  else
    info "${ENV_FILE} already exists; updating required values."
  fi

  domain="${DOMAIN:-$(read_env_value DOMAIN)}"
  if [ -z "${domain}" ] || [ "${domain}" = "deaddrop.example.com" ]; then
    domain="localhost"
  fi

  pg_user="$(read_env_value POSTGRES_USER)"
  if [ -z "${pg_user}" ]; then
    pg_user="deaddrop"
  fi

  pg_db="$(read_env_value POSTGRES_DB)"
  if [ -z "${pg_db}" ]; then
    pg_db="deaddrop"
  fi

  pg_pass="$(read_env_value POSTGRES_PASSWORD)"
  case "${pg_pass}" in
    ""|"CHANGE_ME_TO_A_STRONG_PASSWORD")
      pg_pass="$(random_password)"
      ;;
  esac

  base_url="${APP_BASE_URL:-${BASE_URL:-$(read_env_value BASE_URL)}}"
  if [ -z "${base_url}" ] || \
    echo "${base_url}" | grep -q '\${DOMAIN}' || \
    [ "${base_url}" = "https://${domain}" ] || \
    [ "${base_url}" = "http://${domain}" ] || \
    [ "${base_url}" = "https://deaddrop.example.com" ] || \
    [ "${base_url}" = "http://deaddrop.example.com" ]; then
    base_url="http://$(detect_local_ip):${DASHBOARD_PORT}"
  fi

  if echo "${base_url}" | grep -q '^https://'; then
    secure_cookies="true"
  else
    secure_cookies="false"
  fi

  db_url="postgres://${pg_user}:${pg_pass}@db:5432/${pg_db}?sslmode=disable"

  upsert_env DOMAIN "${domain}"
  upsert_env POSTGRES_USER "${pg_user}"
  upsert_env POSTGRES_PASSWORD "${pg_pass}"
  upsert_env POSTGRES_DB "${pg_db}"
  upsert_env DATABASE_URL "${db_url}"
  upsert_env BASE_URL "${base_url}"
  upsert_env SECURE_COOKIES "${secure_cookies}"
}

validate_file_exists() {
  [ -e "$1" ]
}

validate_file_nonempty() {
  [ -s "$1" ]
}

compose_project_name() {
  basename "$(pwd)"
}

db_volume_name() {
  echo "$(compose_project_name)_pgdata"
}

db_volume_exists() {
  "${DOCKER_CMD[@]}" volume inspect "$(db_volume_name)" >/dev/null 2>&1
}

validate_compose_file_has_local_build_context() {
  grep -qE '^[[:space:]]*context:[[:space:]]*\./src[[:space:]]*$' docker-compose.prod.yml
}

validate_compose_dashboard_port_mapping() {
  grep -q "\"${DASHBOARD_PORT}:80\"" docker-compose.prod.yml
}

validate_env_key_nonempty() {
  [ -n "$(read_env_value "$1")" ]
}

validate_docker_ready() {
  "${DOCKER_CMD[@]}" version >/dev/null 2>&1
}

validate_compose_ready() {
  "${COMPOSE_CMD[@]}" version >/dev/null 2>&1
}

validate_service_running() {
  service="$1"
  [ -n "$("${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" ps -q "${service}")" ]
}

validate_service_healthy() {
  service="$1"
  cid="$("${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" ps -q "${service}")"
  [ -n "${cid}" ] || return 1
  status="$("${DOCKER_CMD[@]}" inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${cid}")"
  [ "${status}" = "healthy" ] || [ "${status}" = "running" ]
}

validate_db_app_credentials() {
  db_user="$(read_env_value POSTGRES_USER)"
  db_name="$(read_env_value POSTGRES_DB)"
  db_pass="$(read_env_value POSTGRES_PASSWORD)"
  PGPASSWORD="${db_pass}" "${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" exec -T db \
    psql -h 127.0.0.1 -U "${db_user}" -d "${db_name}" -c 'select 1;' >/dev/null
}

validate_app_health() {
  "${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" exec -T app \
    wget -qO- http://localhost:8080/health | grep -q '"status":"ok"'
}

wait_for_db() {
  tries=60
  info "Starting Postgres..."
  "${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" up -d db

  info "Waiting for Postgres to become ready..."
  while [ "${tries}" -gt 0 ]; do
    if "${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" exec -T db \
      pg_isready -U "$(read_env_value POSTGRES_USER)" -d "$(read_env_value POSTGRES_DB)" >/dev/null 2>&1; then
      info "Postgres is ready."
      return 0
    fi
    tries=$((tries - 1))
    sleep 2
  done

  fail "Postgres did not become ready in time."
}

start_stack() {
  info "Starting application stack..."
  "${COMPOSE_CMD[@]}" -f docker-compose.prod.yml --env-file "${ENV_FILE}" up -d
}

main() {
  info "DeadDrop — Self-hosted install"
  info "=============================="
  info ""

  step "Check Runtime Requirements" \
    "Instruction: Ensure curl, Docker, and Docker Compose are available (auto-installs Docker on Linux if needed)."
  ensure_curl
  ensure_docker
  configure_docker_command
  configure_compose_command
  validate "curl command exists" has_cmd curl
  validate "docker command works" validate_docker_ready
  validate "docker compose command works" validate_compose_ready

  step "Create Install Directory" \
    "Instruction: Create the install path and switch into it."
  mkdir -p "${INSTALL_DIR}"
  cd "${INSTALL_DIR}"
  validate "install directory exists" validate_file_exists "."

  step "Download Production Files" \
    "Instruction: Download compose file, Caddy config, and environment template."
  download_stack_files
  normalize_compose_for_installer
  validate "docker-compose.prod.yml downloaded" validate_file_nonempty "docker-compose.prod.yml"
  validate "Caddyfile downloaded" validate_file_nonempty "Caddyfile"
  validate ".env.prod.example downloaded" validate_file_nonempty ".env.prod.example"
  validate "dashboard host port mapped (${DASHBOARD_PORT}:80)" validate_compose_dashboard_port_mapping
  if grep -qE '^[[:space:]]*build:[[:space:]]*$' docker-compose.prod.yml; then
    validate "local build context rewritten for installer layout" validate_compose_file_has_local_build_context
    validate "local build Dockerfile is present" validate_file_nonempty "src/docker/Dockerfile"
  fi
  normalize_caddy_for_installer
  validate "Caddyfile set to host-only reverse proxy mode (:80)" grep -q '^:80 {' Caddyfile

  step "Generate Application Environment" \
    "Instruction: Create/update .env.prod with generated DB password and DATABASE_URL."
  seed_environment
  validate ".env.prod exists" validate_file_nonempty "${ENV_FILE}"
  validate "BASE_URL is set" validate_env_key_nonempty "BASE_URL"
  validate "DATABASE_URL is set" validate_env_key_nonempty "DATABASE_URL"
  validate "POSTGRES_PASSWORD is set" validate_env_key_nonempty "POSTGRES_PASSWORD"

  step "Start And Validate Postgres" \
    "Instruction: Boot db service first and wait until it responds to pg_isready."
  wait_for_db
  validate "db service is running" validate_service_running "db"
  validate "db service is healthy" validate_service_healthy "db"
  validate "app DB credentials authenticate against Postgres" validate_db_app_credentials

  step "Start And Validate Full Stack" \
    "Instruction: Start app + caddy and ensure required services are running."
  start_stack
  validate "app service is running" validate_service_running "app"
  validate "app service is healthy/running" validate_service_healthy "app"
  validate "caddy service is running" validate_service_running "caddy"

  step "Validate Application Health Endpoint" \
    "Instruction: Run an in-container health request to verify the app is serving traffic."
  validate "app /health responds with status ok" validate_app_health

  info ""
  info "Install complete."
  info "Directory: $(pwd)"
  info "Dashboard URL: $(read_env_value BASE_URL)"
  info "DATABASE_URL: $(mask_database_url "$(read_env_value DATABASE_URL)")"
  info ""
  info "Next setup in dashboard (domain onboarding happens here):"
  info "  1. Open dashboard at $(read_env_value BASE_URL)"
  info "     Validation: curl -sS $(read_env_value BASE_URL)/health"
  info "  2. Sign in and create a domain in the Domains screen."
  info "     Validation: Domain appears as Pending Verification."
  info "  3. Add the DNS records shown in the dashboard (TXT + MX, optionally SPF/DMARC)."
  info "     Validation: Click Verify in dashboard and status becomes Verified."
  info "  4. Send test mail to any address on that domain."
  info "     Validation: Message appears in the Inbox view."
  info ""
  info "Useful commands:"
  info "  ${COMPOSE_CMD[*]} -f docker-compose.prod.yml --env-file ${ENV_FILE} ps"
  info "  ${COMPOSE_CMD[*]} -f docker-compose.prod.yml --env-file ${ENV_FILE} logs -f app"
}

main "$@"
