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
    VERIFY_TOKEN=$(echo "$page" | grep -oP 'class="val">\K[^<]+' | head -1)
fi

if [[ -n "$VERIFY_TOKEN" ]]; then
    pass "Verification token extracted: ${VERIFY_TOKEN:0:8}..."
else
    fail "Verification token extraction" "could not find token in response"
fi

# Extract domain public ID from URL
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
    sleep 1
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
