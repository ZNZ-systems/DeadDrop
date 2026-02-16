#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-ZNZ-systems/DeadDrop}"
BRANCH="${BRANCH:-master}"
BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/docker"
INSTALL_DIR="${INSTALL_DIR:-deaddrop}"
ENV_FILE=".env.prod"

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
  curl -fsSL "${BASE_URL}/docker-compose.prod.yml" -o docker-compose.prod.yml
  curl -fsSL "${BASE_URL}/Caddyfile" -o Caddyfile
  curl -fsSL "${BASE_URL}/.env.prod.example" -o .env.prod.example
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

seed_environment() {
  if [ ! -f "${ENV_FILE}" ]; then
    cp .env.prod.example "${ENV_FILE}"
    info "Created ${ENV_FILE} from template."
  else
    info "${ENV_FILE} already exists; updating required values."
  fi

  domain="${DOMAIN:-$(read_env_value DOMAIN)}"
  if [ -z "${domain}" ]; then
    domain="localhost"
    warn "DOMAIN was not set. Defaulting to localhost."
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

  if [ "${domain}" = "localhost" ] || [ "${domain}" = "127.0.0.1" ]; then
    base_url="http://${domain}"
    secure_cookies="false"
  else
    base_url="https://${domain}"
    secure_cookies="true"
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

  ensure_curl
  ensure_docker
  configure_docker_command
  configure_compose_command

  mkdir -p "${INSTALL_DIR}"
  cd "${INSTALL_DIR}"

  download_stack_files
  seed_environment
  wait_for_db
  start_stack

  info ""
  info "Install complete."
  info "Directory: $(pwd)"
  info "Domain: $(read_env_value DOMAIN)"
  info "DATABASE_URL: $(read_env_value DATABASE_URL)"
  info ""
  info "Useful commands:"
  info "  ${COMPOSE_CMD[*]} -f docker-compose.prod.yml --env-file ${ENV_FILE} ps"
  info "  ${COMPOSE_CMD[*]} -f docker-compose.prod.yml --env-file ${ENV_FILE} logs -f app"
}

main "$@"
