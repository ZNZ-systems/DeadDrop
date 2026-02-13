#!/usr/bin/env bash
set -euo pipefail

# ─── Configuration ───────────────────────────────────────────────────
PI_HOST="${PI_HOST:-pi@192.168.86.79}"
PI_DIR="${PI_DIR:-/home/pi/deaddrop}"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE_NAME="deaddrop:e2e"
IMAGE_TAR="deaddrop-e2e.tar"

echo "━━━ DeadDrop E2E: Build → Deploy → Test ━━━"
echo "  Pi: $PI_HOST"
echo "  Project: $PROJECT_ROOT"
echo ""

# ─── Step 1: Clean the Pi ───────────────────────────────────────────
echo "▸ Cleaning Pi..."
ssh "$PI_HOST" bash -s <<'REMOTE'
    # Stop any running deaddrop containers
    cd ~/deaddrop 2>/dev/null && docker compose down -v 2>/dev/null || true
    # Remove old deaddrop images (keep postgres)
    docker images --format '{{.Repository}}:{{.Tag}} {{.ID}}' | grep -i deaddrop | awk '{print $2}' | xargs -r docker rmi -f 2>/dev/null || true
    # Clean up deployment dir
    rm -rf ~/deaddrop
    mkdir -p ~/deaddrop/config
    echo "  Pi cleaned."
REMOTE

# ─── Step 2: Build arm64 Docker image ───────────────────────────────
echo "▸ Building arm64 Docker image..."
cd "$PROJECT_ROOT"
docker buildx build \
    --platform linux/arm64 \
    -t "$IMAGE_NAME" \
    -f docker/Dockerfile \
    --output "type=docker,dest=${IMAGE_TAR}" \
    .
echo "  Image built: $(du -h "$IMAGE_TAR" | cut -f1)"

# ─── Step 3: Transfer files to Pi ───────────────────────────────────
echo "▸ Transferring files to Pi..."
scp "$IMAGE_TAR" "$PI_HOST:${PI_DIR}/"
scp e2e/docker-compose.yml "$PI_HOST:${PI_DIR}/"
scp e2e/e2e-tests.sh "$PI_HOST:${PI_DIR}/"
# Create empty DNS overrides file (tests will populate it)
ssh "$PI_HOST" "touch ${PI_DIR}/config/dns-overrides.txt"
echo "  Files transferred."

# Clean up local tarball
rm -f "$IMAGE_TAR"

# ─── Step 4: Load image and start containers ────────────────────────
echo "▸ Starting containers on Pi..."
ssh "$PI_HOST" bash -s <<REMOTE
    cd ${PI_DIR}
    docker load < ${IMAGE_TAR}
    rm -f ${IMAGE_TAR}
    docker compose up -d
    echo "  Containers starting..."
REMOTE

# ─── Step 5: Wait for health ────────────────────────────────────────
echo "▸ Waiting for app to be healthy..."
for i in $(seq 1 60); do
    if ssh "$PI_HOST" "curl -sf http://localhost:8080/health" > /dev/null 2>&1; then
        echo "  App is healthy! (${i}s)"
        break
    fi
    if [[ $i -eq 60 ]]; then
        echo "  FATAL: App not healthy after 60s"
        ssh "$PI_HOST" "cd ${PI_DIR} && docker compose logs app"
        exit 1
    fi
    sleep 1
done

# ─── Step 6: Install swaks if missing ───────────────────────────────
echo "▸ Ensuring test dependencies..."
ssh "$PI_HOST" "command -v swaks > /dev/null || sudo apt-get install -y swaks" 2>/dev/null

# ─── Step 7: Run tests ──────────────────────────────────────────────
echo ""
echo "━━━ Running E2E Tests ━━━"
ssh "$PI_HOST" "cd ${PI_DIR} && chmod +x e2e-tests.sh && bash e2e-tests.sh"
TEST_EXIT=$?

# ─── Step 8: Show logs on failure ────────────────────────────────────
if [[ $TEST_EXIT -ne 0 ]]; then
    echo ""
    echo "━━━ App Logs (last 50 lines) ━━━"
    ssh "$PI_HOST" "cd ${PI_DIR} && docker compose logs --tail=50 app"
fi

exit $TEST_EXIT
