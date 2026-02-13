# E2E Testing on Raspberry Pi — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deploy DeadDrop to a Raspberry Pi 5 on the LAN and run comprehensive e2e tests covering signup, domain verification (via file-based DNS), mailbox/stream management, form API submissions, inbound SMTP, and edge cases.

**Architecture:** Add a `FileResolver` that reads DNS TXT overrides from a file (activated via env var). Cross-compile a Docker image for arm64, deploy to the Pi with docker-compose, and run a bash test suite using curl/swaks.

**Tech Stack:** Go, Docker buildx (arm64), bash, curl, swaks, jq, docker compose

---

### Task 1: Add FileResolver for DNS TXT Override

**Files:**
- Create: `internal/domain/file_resolver.go`
- Test: `internal/domain/service_test.go` (add test cases)

**Step 1: Write the failing test**

Add to `internal/domain/service_test.go`:

```go
func TestFileResolver_LookupTXT(t *testing.T) {
	// Create temp file with overrides
	f, err := os.CreateTemp("", "dns-overrides-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, _ = f.WriteString("example.com=deaddrop-verify=abc-123\n")
	_, _ = f.WriteString("other.com=deaddrop-verify=def-456\n")
	f.Close()

	resolver, err := NewFileResolver(f.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records, err := resolver.LookupTXT("example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(records) != 1 || records[0] != "deaddrop-verify=abc-123" {
		t.Errorf("expected [deaddrop-verify=abc-123], got %v", records)
	}

	records, err = resolver.LookupTXT("nonexistent.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty records, got %v", records)
	}
}

func TestFileResolver_ReloadsFile(t *testing.T) {
	f, err := os.CreateTemp("", "dns-overrides-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	resolver, err := NewFileResolver(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Initially empty
	records, _ := resolver.LookupTXT("example.com")
	if len(records) != 0 {
		t.Errorf("expected empty, got %v", records)
	}

	// Write a record
	os.WriteFile(f.Name(), []byte("example.com=deaddrop-verify=new-token\n"), 0644)

	// Should pick up the new record
	records, _ = resolver.LookupTXT("example.com")
	if len(records) != 1 || records[0] != "deaddrop-verify=new-token" {
		t.Errorf("expected [deaddrop-verify=new-token], got %v", records)
	}
}
```

Add `"os"` to the import block.

**Step 2: Run test to verify it fails**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/domain/ -run TestFileResolver -v`
Expected: FAIL — `NewFileResolver` not defined

**Step 3: Write FileResolver implementation**

Create `internal/domain/file_resolver.go`:

```go
package domain

import (
	"bufio"
	"os"
	"strings"
)

// FileResolver reads DNS TXT overrides from a file. Each line is
// "domain=value". The file is re-read on every LookupTXT call so
// records can be added at runtime (e.g., by e2e test scripts).
type FileResolver struct {
	path string
}

// NewFileResolver creates a FileResolver that reads from the given file path.
func NewFileResolver(path string) (*FileResolver, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &FileResolver{path: path}, nil
}

// LookupTXT returns all TXT record values for the given host.
func (r *FileResolver) LookupTXT(host string) ([]string, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && parts[0] == host {
			records = append(records, parts[1])
		}
	}
	return records, scanner.Err()
}
```

**IMPORTANT NOTE on file format:** Each line is `domain=value`. The `SplitN(line, "=", 2)` splits on the FIRST `=` only, so `example.com=deaddrop-verify=abc` correctly yields domain=`example.com`, value=`deaddrop-verify=abc`.

**Step 4: Run test to verify it passes**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/domain/ -run TestFileResolver -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/file_resolver.go internal/domain/service_test.go
git commit -m "feat: add FileResolver for DNS TXT override in e2e tests"
```

---

### Task 2: Wire FileResolver into Config and Main

**Files:**
- Modify: `internal/config/config.go` (add DNSOverrideFile field)
- Modify: `cmd/deaddrop/main.go` (use FileResolver when configured)

**Step 1: Add config field**

In `internal/config/config.go`, add to the `Config` struct after `InboundSMTPEnabled`:

```go
	DNSOverrideFile string
```

In the `Load()` function, before the return statement, add:

```go
	dnsOverrideFile := getEnv("DNS_OVERRIDE_FILE", "")
```

And add to the returned Config:

```go
		DNSOverrideFile:    dnsOverrideFile,
```

**Step 2: Wire in main.go**

In `cmd/deaddrop/main.go`, replace line 64:
```go
	domainService := domain.NewService(domainStore, &domain.NetResolver{})
```

With:
```go
	var dnsResolver domain.DNSResolver
	if cfg.DNSOverrideFile != "" {
		r, err := domain.NewFileResolver(cfg.DNSOverrideFile)
		if err != nil {
			slog.Error("failed to open DNS override file", "path", cfg.DNSOverrideFile, "error", err)
			os.Exit(1)
		}
		slog.Info("using file-based DNS resolver", "path", cfg.DNSOverrideFile)
		dnsResolver = r
	} else {
		dnsResolver = &domain.NetResolver{}
	}
	domainService := domain.NewService(domainStore, dnsResolver)
```

**Step 3: Run all tests**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./... -v`
Expected: All tests pass (no behavior change when env var is unset)

**Step 4: Commit**

```bash
git add internal/config/config.go cmd/deaddrop/main.go
git commit -m "feat: wire FileResolver via DNS_OVERRIDE_FILE env var"
```

---

### Task 3: Create E2E Docker Compose for Pi

**Files:**
- Create: `e2e/docker-compose.yml`
- Create: `e2e/.env`

**Step 1: Create compose file**

Create `e2e/docker-compose.yml`:

```yaml
services:
  app:
    image: deaddrop:e2e
    ports:
      - "8080:8080"
      - "25:2525"
    depends_on:
      db:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://deaddrop:deaddrop@db:5432/deaddrop?sslmode=disable
      SECURE_COOKIES: "false"
      BASE_URL: http://localhost:8080
      INBOUND_SMTP_ADDR: ":2525"
      INBOUND_SMTP_DOMAIN: test.example.com
      DNS_OVERRIDE_FILE: /config/dns-overrides.txt
      RATE_LIMIT_RPS: "100"
      RATE_LIMIT_BURST: "200"
    volumes:
      - ./config:/config
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: deaddrop
      POSTGRES_PASSWORD: deaddrop
      POSTGRES_DB: deaddrop
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U deaddrop"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

volumes:
  pgdata:
```

**Key decisions:**
- `SECURE_COOKIES: false` — Pi is HTTP-only on LAN, no HTTPS
- `RATE_LIMIT_RPS: 100` — High limit so tests don't get rate-limited during normal flow (edge case test handles rate limit separately)
- `DNS_OVERRIDE_FILE: /config/dns-overrides.txt` — Mounted volume, writable by test script
- Health check on both app and db ensures proper startup ordering

**Step 2: Create initial empty config dir**

Create `e2e/config/.gitkeep` (empty file to ensure the config directory exists in git).

**Step 3: Commit**

```bash
git add e2e/docker-compose.yml e2e/config/.gitkeep
git commit -m "feat: add e2e docker-compose for Pi deployment"
```

---

### Task 4: Create E2E Test Script

**Files:**
- Create: `e2e/e2e-tests.sh`

**Step 1: Write the test script**

Create `e2e/e2e-tests.sh` (full content below). This is the core test runner.

```bash
#!/usr/bin/env bash
set -euo pipefail

# ─── Configuration ───────────────────────────────────────────────────
BASE_URL="${BASE_URL:-http://localhost:8080}"
SMTP_HOST="${SMTP_HOST:-localhost}"
SMTP_PORT="${SMTP_PORT:-25}"
DNS_FILE="${DNS_FILE:-/home/pi/deaddrop/config/dns-overrides.txt}"
COOKIE_JAR=$(mktemp /tmp/e2e-cookies.XXXXXX)
trap 'rm -f "$COOKIE_JAR"' EXIT

PASS=0
FAIL=0
TOTAL=0

# ─── Helpers ─────────────────────────────────────────────────────────
pass() { ((PASS++)); ((TOTAL++)); echo "  ✓ $1"; }
fail() { ((FAIL++)); ((TOTAL++)); echo "  ✗ $1: $2"; }

assert_status() {
    local desc="$1" expected="$2" actual="$3"
    if [[ "$actual" == "$expected" ]]; then
        pass "$desc"
    else
        fail "$desc" "expected HTTP $expected, got $actual"
    fi
}

assert_contains() {
    local desc="$1" haystack="$2" needle="$3"
    if echo "$haystack" | grep -q "$needle"; then
        pass "$desc"
    else
        fail "$desc" "response does not contain '$needle'"
    fi
}

# curl with cookies, following redirects, capturing headers+body
# Usage: response=$(do_get "/path")
do_get() {
    curl -s -L -b "$COOKIE_JAR" -c "$COOKIE_JAR" "${BASE_URL}$1"
}

# POST with form data — automatically injects CSRF token from cookie jar
do_post() {
    local path="$1"; shift
    local csrf
    csrf=$(grep csrf_token "$COOKIE_JAR" 2>/dev/null | awk '{print $NF}' || echo "")
    curl -s -L -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
         -d "csrf_token=${csrf}" "$@" "${BASE_URL}${path}"
}

# POST returning only HTTP status code (no redirect follow)
do_post_status() {
    local path="$1"; shift
    local csrf
    csrf=$(grep csrf_token "$COOKIE_JAR" 2>/dev/null | awk '{print $NF}' || echo "")
    curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
         -d "csrf_token=${csrf}" "$@" "${BASE_URL}${path}"
}

# GET returning only HTTP status code (no redirect follow)
do_get_status() {
    curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE_JAR" -c "$COOKIE_JAR" "${BASE_URL}$1"
}

section() {
    echo ""
    echo "━━━ $1 ━━━"
}

# ─── Wait for app ────────────────────────────────────────────────────
section "Waiting for app to be healthy"
for i in $(seq 1 30); do
    if curl -sf "${BASE_URL}/health" > /dev/null 2>&1; then
        pass "App is healthy"
        break
    fi
    if [[ $i -eq 30 ]]; then
        fail "App health check" "timed out after 30s"
        echo "FATAL: App not ready. Aborting."
        exit 1
    fi
    sleep 1
done

# ─── Flow 1: Full Lifecycle ─────────────────────────────────────────
section "Flow 1: Full Lifecycle"

# 1.1 — Get signup page (establishes CSRF cookie)
page=$(do_get "/signup")
assert_contains "GET /signup loads" "$page" "Sign Up"

# 1.2 — Signup
page=$(do_post "/signup" \
    -d "email=test@e2e.local" \
    -d "password=TestPass123!" \
    -d "password_confirm=TestPass123!")
assert_contains "Signup + auto-login succeeds" "$page" "Dashboard\|Domains\|DeadDrop"

# 1.3 — Add domain
page=$(do_post "/domains" -d "name=test.example.com")
assert_contains "Domain created" "$page" "test.example.com"

# Extract verification token from domain detail page
VERIFY_TOKEN=$(echo "$page" | grep -oP 'deaddrop-verify=\K[a-f0-9-]+' | head -1)
if [[ -z "$VERIFY_TOKEN" ]]; then
    # Fallback: try extracting from val span
    VERIFY_TOKEN=$(echo "$page" | grep -oP 'class="val">\K[^<]+' | head -1)
fi

if [[ -n "$VERIFY_TOKEN" ]]; then
    pass "Verification token extracted: ${VERIFY_TOKEN:0:8}..."
else
    fail "Verification token extraction" "could not find token in response"
fi

# Extract domain public ID from URL (the page we're on after redirect)
DOMAIN_PUBLIC_ID=$(echo "$page" | grep -oP '/domains/\K[a-f0-9-]{36}' | head -1)
if [[ -n "$DOMAIN_PUBLIC_ID" ]]; then
    pass "Domain public ID: ${DOMAIN_PUBLIC_ID:0:8}..."
else
    fail "Domain public ID extraction" "could not find domain UUID"
fi

# 1.4 — Write DNS override file for verification
echo "test.example.com=deaddrop-verify=${VERIFY_TOKEN}" > "$DNS_FILE"
pass "DNS override file written"

# 1.5 — Verify domain
page=$(do_post "/domains/${DOMAIN_PUBLIC_ID}/verify")
assert_contains "Domain verification succeeds" "$page" "verified\|Widget Embed Code"

# 1.6 — Get mailbox creation form (need internal domain ID)
page=$(do_get "/mailboxes/new")
DOMAIN_INTERNAL_ID=$(echo "$page" | grep -oP 'value="(\d+)".*?test\.example\.com' | grep -oP '\d+' | head -1)
if [[ -z "$DOMAIN_INTERNAL_ID" ]]; then
    # Broader fallback
    DOMAIN_INTERNAL_ID=$(echo "$page" | grep 'test.example.com' | grep -oP 'value="(\d+)"' | grep -oP '\d+' | head -1)
fi

if [[ -n "$DOMAIN_INTERNAL_ID" ]]; then
    pass "Domain internal ID: $DOMAIN_INTERNAL_ID"
else
    fail "Domain internal ID extraction" "could not find internal ID in select"
fi

# 1.7 — Create mailbox
page=$(do_post "/mailboxes" \
    -d "name=Support" \
    -d "domain_id=${DOMAIN_INTERNAL_ID}" \
    -d "from_address=support@test.example.com")
assert_contains "Mailbox created" "$page" "Support"

# Extract mailbox public ID
MAILBOX_PUBLIC_ID=$(echo "$page" | grep -oP '/mailboxes/\K[a-f0-9-]{36}' | head -1)
if [[ -n "$MAILBOX_PUBLIC_ID" ]]; then
    pass "Mailbox public ID: ${MAILBOX_PUBLIC_ID:0:8}..."
else
    fail "Mailbox public ID extraction" "could not find mailbox UUID"
fi

# 1.8 — Add form stream
page=$(do_post "/mailboxes/${MAILBOX_PUBLIC_ID}/streams" \
    -d "type=form" \
    -d "address=")
assert_contains "Form stream created" "$page" "Widget:"

# Extract widget ID for public API
WIDGET_ID=$(echo "$page" | grep -oP 'Widget: \K[a-f0-9-]{36}' | head -1)
if [[ -n "$WIDGET_ID" ]]; then
    pass "Widget ID: ${WIDGET_ID:0:8}..."
else
    fail "Widget ID extraction" "could not find widget UUID"
fi

# 1.9 — Submit contact form via public API
API_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/messages" \
    -d "domain_id=${WIDGET_ID}" \
    -d "name=Jane Doe" \
    -d "email=jane@visitor.com" \
    -d "subject=Hello from e2e test" \
    -d "message=This is a test message from the e2e suite.")
assert_contains "Public API accepts message" "$API_RESPONSE" '"ok":true'

# 1.10 — Check conversation appeared
page=$(do_get "/mailboxes/${MAILBOX_PUBLIC_ID}")
assert_contains "Conversation visible" "$page" "Hello from e2e test"

# Extract conversation public ID
CONV_PUBLIC_ID=$(echo "$page" | grep -oP '/conversations/\K[a-f0-9-]{36}' | head -1)
if [[ -n "$CONV_PUBLIC_ID" ]]; then
    pass "Conversation public ID: ${CONV_PUBLIC_ID:0:8}..."
else
    fail "Conversation public ID extraction" "could not find conversation UUID"
fi

# 1.11 — View conversation detail
page=$(do_get "/mailboxes/${MAILBOX_PUBLIC_ID}/conversations/${CONV_PUBLIC_ID}")
assert_contains "Conversation shows message body" "$page" "This is a test message"
assert_contains "Conversation shows sender" "$page" "Jane Doe\|jane@visitor.com"

# 1.12 — Reply to conversation
page=$(do_post "/mailboxes/${MAILBOX_PUBLIC_ID}/conversations/${CONV_PUBLIC_ID}/reply" \
    -d "body=Thanks for reaching out!")
assert_contains "Reply sent" "$page" "Thanks for reaching out!\|Reply sent"

# 1.13 — Close conversation
page=$(do_post "/mailboxes/${MAILBOX_PUBLIC_ID}/conversations/${CONV_PUBLIC_ID}/close")
assert_contains "Conversation closed" "$page" "Closed\|closed"

# ─── Flow 2: Inbound SMTP ───────────────────────────────────────────
section "Flow 2: Inbound SMTP"

# 2.1 — Add email stream
page=$(do_post "/mailboxes/${MAILBOX_PUBLIC_ID}/streams" \
    -d "type=email" \
    -d "address=inbox@test.example.com")
assert_contains "Email stream created" "$page" "inbox@test.example.com"

# 2.2 — Send email via SMTP using swaks
if command -v swaks &> /dev/null; then
    SWAKS_OUT=$(swaks \
        --to inbox@test.example.com \
        --from sender@external.com \
        --server "${SMTP_HOST}:${SMTP_PORT}" \
        --header "Subject: SMTP e2e test" \
        --body "This message was sent via SMTP" \
        --timeout 10 2>&1) || true

    if echo "$SWAKS_OUT" | grep -q "250 "; then
        pass "SMTP message accepted"
    else
        fail "SMTP message delivery" "swaks output: $(echo "$SWAKS_OUT" | tail -3)"
    fi

    # 2.3 — Check conversation created from SMTP
    sleep 1  # Brief pause for async processing
    page=$(do_get "/mailboxes/${MAILBOX_PUBLIC_ID}")
    assert_contains "SMTP conversation visible" "$page" "SMTP e2e test"

    # 2.4 — View SMTP conversation
    SMTP_CONV_ID=$(echo "$page" | grep -oP '/conversations/\K[a-f0-9-]{36}' | head -1)
    if [[ -n "$SMTP_CONV_ID" && "$SMTP_CONV_ID" != "$CONV_PUBLIC_ID" ]]; then
        page=$(do_get "/mailboxes/${MAILBOX_PUBLIC_ID}/conversations/${SMTP_CONV_ID}")
        assert_contains "SMTP message body visible" "$page" "SMTP\|sent via SMTP"
        pass "SMTP conversation detail loaded"
    else
        fail "SMTP conversation ID" "could not find new conversation from SMTP"
    fi
else
    echo "  ⚠ swaks not installed — skipping SMTP tests"
    echo "    Install with: sudo apt-get install -y swaks"
fi

# ─── Flow 3: Edge Cases ─────────────────────────────────────────────
section "Flow 3: Edge Cases"

# 3.1 — Health check
status=$(do_get_status "/health")
assert_status "Health check returns 200" "200" "$status"

# 3.2 — Honeypot: submit with _gotcha field filled
API_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/messages" \
    -d "domain_id=${WIDGET_ID}" \
    -d "name=Bot" \
    -d "email=bot@spam.com" \
    -d "subject=Spam" \
    -d "message=Buy stuff" \
    -d "_gotcha=I am a bot")
assert_contains "Honeypot silently accepts" "$API_RESPONSE" '"ok":true'

# Check no conversation created for honeypot
page=$(do_get "/mailboxes/${MAILBOX_PUBLIC_ID}")
if echo "$page" | grep -q "Spam"; then
    fail "Honeypot blocks conversation" "spam conversation was created"
else
    pass "Honeypot blocks conversation creation"
fi

# 3.3 — Missing message body
API_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/messages" \
    -d "domain_id=${WIDGET_ID}" \
    -d "name=Test" \
    -d "email=test@test.com")
assert_contains "Missing message rejected" "$API_RESPONSE" '"error"'

# 3.4 — Invalid domain_id
API_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/messages" \
    -d "domain_id=00000000-0000-0000-0000-000000000000" \
    -d "message=test")
assert_contains "Invalid domain_id rejected" "$API_RESPONSE" '"error"'

# 3.5 — CSRF protection: POST without token
CSRF_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BASE_URL}/domains" \
    -b "$COOKIE_JAR" \
    -d "name=nope.com")
assert_status "POST without CSRF token returns 403" "403" "$CSRF_STATUS"

# 3.6 — Unauthenticated access
UNAUTH_JAR=$(mktemp /tmp/e2e-unauth.XXXXXX)
trap 'rm -f "$COOKIE_JAR" "$UNAUTH_JAR"' EXIT
UNAUTH_PAGE=$(curl -s -L -b "$UNAUTH_JAR" -c "$UNAUTH_JAR" "${BASE_URL}/mailboxes/new")
assert_contains "Unauthenticated redirects to login" "$UNAUTH_PAGE" "Log In\|login"

# ─── Flow 4: Cleanup Cascades ───────────────────────────────────────
section "Flow 4: Cleanup & Cascades"

# 4.1 — Submit another message for cascade testing
API_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/messages" \
    -d "domain_id=${WIDGET_ID}" \
    -d "name=Cascade Test" \
    -d "email=cascade@test.com" \
    -d "subject=Cascade test msg" \
    -d "message=Will this survive deletion?")
assert_contains "Pre-delete message created" "$API_RESPONSE" '"ok":true'

# 4.2 — Delete mailbox (cascades conversations)
page=$(do_post "/mailboxes/${MAILBOX_PUBLIC_ID}/delete")
assert_contains "Mailbox deleted, back at dashboard" "$page" "Dashboard\|New Mailbox\|DeadDrop"

# 4.3 — Delete domain
page=$(do_post "/domains/${DOMAIN_PUBLIC_ID}/delete")
assert_contains "Domain deleted" "$page" "Dashboard\|DeadDrop"

# ─── Summary ─────────────────────────────────────────────────────────
section "Results"
echo ""
echo "  Total: $TOTAL"
echo "  Pass:  $PASS"
echo "  Fail:  $FAIL"
echo ""

if [[ $FAIL -gt 0 ]]; then
    echo "  ✗ SOME TESTS FAILED"
    exit 1
else
    echo "  ✓ ALL TESTS PASSED"
    exit 0
fi
```

**Step 2: Make it executable**

```bash
chmod +x e2e/e2e-tests.sh
```

**Step 3: Commit**

```bash
git add e2e/e2e-tests.sh
git commit -m "feat: add e2e test script for Pi deployment"
```

---

### Task 5: Create Orchestration Script

**Files:**
- Create: `e2e/run-e2e.sh`

**Step 1: Write the orchestration script**

This runs from the local Mac and handles the full build → deploy → test cycle.

```bash
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
```

**Step 2: Make it executable**

```bash
chmod +x e2e/run-e2e.sh
```

**Step 3: Commit**

```bash
git add e2e/run-e2e.sh
git commit -m "feat: add e2e orchestration script for Pi deployment"
```

---

### Task 6: Update Dockerfile for Arm64 Compatibility

**Files:**
- Modify: `docker/Dockerfile`

**Step 1: Verify current Dockerfile works with buildx**

The current Dockerfile uses `CGO_ENABLED=0 GOOS=linux` without GOARCH. Docker buildx with `--platform linux/arm64` sets TARGETPLATFORM but doesn't automatically pass GOARCH to `go build`. We need to use Docker's automatic platform args.

Replace the full `docker/Dockerfile` with:

```dockerfile
# Stage 1: Build
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /deaddrop ./cmd/deaddrop

# Stage 2: Runtime
FROM alpine:3.20

COPY --from=builder /deaddrop /deaddrop

EXPOSE 8080

ENTRYPOINT ["/deaddrop"]
```

**Key change:** `FROM --platform=$BUILDPLATFORM` ensures the builder runs on the host platform (amd64 Mac), while `TARGETOS`/`TARGETARCH` cross-compile the Go binary for the target (arm64). This is much faster than emulating arm64 for compilation.

**Step 2: Verify it still builds for local platform**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && docker build -f docker/Dockerfile -t deaddrop:test .`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add docker/Dockerfile
git commit -m "feat: update Dockerfile for multi-platform buildx support"
```

---

### Task 7: Run the Full E2E Suite

**Step 1: Execute orchestration from local machine**

```bash
cd /Volumes/BigStorage/CodeWorld/DeadDrop && bash e2e/run-e2e.sh
```

Expected output:
```
━━━ DeadDrop E2E: Build → Deploy → Test ━━━
  Pi: pi@192.168.86.79
▸ Cleaning Pi...
▸ Building arm64 Docker image...
▸ Transferring files to Pi...
▸ Starting containers on Pi...
▸ Waiting for app to be healthy...

━━━ Running E2E Tests ━━━
━━━ Flow 1: Full Lifecycle ━━━
  ✓ GET /signup loads
  ✓ Signup + auto-login succeeds
  ✓ Domain created
  ...
━━━ Results ━━━
  ✓ ALL TESTS PASSED
```

**Step 2: If tests fail, debug**

- Check app logs: `ssh pi@192.168.86.79 "cd ~/deaddrop && docker compose logs app"`
- Check db: `ssh pi@192.168.86.79 "cd ~/deaddrop && docker compose exec db psql -U deaddrop -c '\dt'"`
- Run single test manually: `ssh pi@192.168.86.79 "cd ~/deaddrop && curl -v http://localhost:8080/health"`

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat: complete e2e testing infrastructure for Pi deployment"
```
