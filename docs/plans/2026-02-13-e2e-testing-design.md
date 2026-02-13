# E2E Testing Design: DeadDrop on Raspberry Pi

## Overview

End-to-end testing of the DeadDrop mailbox system by deploying to a Raspberry Pi 5 (pi@192.168.86.79) on the LAN. Tests validate the complete user journey from signup through domain verification, mailbox management, inbound message handling (both HTTP form API and SMTP), and conversation management.

## Architecture

### File-Based DNS Resolver

Add `FileResolver` to `internal/domain/dns.go` that reads TXT record overrides from a local file. Activated via `DNS_OVERRIDE_FILE` env var. Falls back to `NetResolver` when unset.

**File format** (`dns-overrides.txt`):
```
test.example.com=deaddrop-verify=<token>
```

One domain per line, `domain=value` pairs. The resolver returns matching values as TXT records for `LookupTXT()` calls.

**Wiring**: In `main.go`, check `DNS_OVERRIDE_FILE` env var. If set, pass `FileResolver` to `domain.NewService()` instead of `NetResolver`.

### Docker Image

Cross-compile for `linux/arm64` using Docker buildx. Export as tarball, transfer via SCP, load on Pi.

### Deployment Stack (on Pi)

- `deaddrop:e2e` app container (port 8080 HTTP, port 25→2525 SMTP)
- `postgres:16-alpine` database container (already cached on Pi)
- Volume mount for DNS override file at `/config/dns-overrides.txt`
- Volume mount for test scripts

### Test Runner

Bash script (`e2e-tests.sh`) using `curl` for HTTP and `swaks` for SMTP. Runs on the Pi after deployment. Reports pass/fail per test with colored output.

## E2E Test Flows

### Flow 1: Full Lifecycle (Happy Path)

| Step | Method | Endpoint | Validates |
|------|--------|----------|-----------|
| 1 | POST | /signup | User creation |
| 2 | POST | /login | Session cookie returned |
| 3 | POST | /domains | Domain created with verification token |
| 4 | Write | dns-overrides.txt | Add token for test domain |
| 5 | POST | /domains/{id}/verify | Domain verified via file resolver |
| 6 | GET | /domains/{id} | Domain shows as verified |
| 7 | POST | /mailboxes | Mailbox created on verified domain |
| 8 | POST | /mailboxes/{id}/streams | Form stream created |
| 9 | POST | /api/v1/messages | Contact form submission creates conversation |
| 10 | GET | /mailboxes/{id} | Conversation visible in mailbox |
| 11 | GET | /mailboxes/{id}/conversations/{cid} | Conversation detail with message |
| 12 | POST | /mailboxes/{id}/conversations/{cid}/reply | Reply sent |
| 13 | POST | /mailboxes/{id}/conversations/{cid}/close | Conversation closed |

### Flow 2: Inbound SMTP

| Step | Method | Target | Validates |
|------|--------|--------|-----------|
| 1 | POST | /mailboxes/{id}/streams | Email stream created |
| 2 | swaks | localhost:25 | SMTP accepts email |
| 3 | GET | /mailboxes/{id} | Conversation created from SMTP |
| 4 | GET | /mailboxes/{id}/conversations/{cid} | Email content matches sent message |

### Flow 3: Edge Cases

| Test | Action | Expected |
|------|--------|----------|
| Disabled stream | POST /api/v1/messages to disabled stream | Rejection / error |
| Closed conversation reply | POST reply to closed conversation | Failure |
| Rate limiting | Rapid POST /api/v1/messages | HTTP 429 |
| CSRF protection | POST without CSRF token | HTTP 403 |
| Auth required | GET /mailboxes unauthenticated | Redirect to /login |
| Honeypot | POST /api/v1/messages with _gotcha field | Silent accept, no conversation |

### Flow 4: Health & Cleanup

| Test | Action | Expected |
|------|--------|----------|
| Health check | GET /health | HTTP 200 |
| Mailbox cascade delete | Delete mailbox | Conversations gone |
| Domain cascade delete | Delete domain | Mailboxes gone |

## Deployment Script (`e2e/run-e2e.sh`)

Orchestration script run from the local machine:

1. **Clean Pi**: SSH in, stop containers, prune volumes/images
2. **Build image**: `docker buildx build --platform linux/arm64`
3. **Transfer**: SCP tarball + compose + tests + DNS file to Pi
4. **Deploy**: `docker load` + `docker compose up -d`
5. **Wait**: Poll `/health` until 200
6. **Test**: SSH into Pi, run `e2e-tests.sh`
7. **Report**: Show pass/fail summary

## Files to Create/Modify

### New Files
- `internal/domain/file_resolver.go` — FileResolver implementation
- `e2e/docker-compose.yml` — Pi deployment compose file
- `e2e/dns-overrides.txt` — Template DNS override file (tokens filled at runtime)
- `e2e/e2e-tests.sh` — Test runner script
- `e2e/run-e2e.sh` — Local orchestration script

### Modified Files
- `cmd/deaddrop/main.go` — Wire FileResolver when DNS_OVERRIDE_FILE is set
- `internal/config/config.go` — Add DNSOverrideFile config field

## Pi Target

- **Host**: pi@192.168.86.79
- **Hardware**: Raspberry Pi 5, 8GB RAM, 49GB free disk
- **Docker**: Compose v5.0.2
- **Cached images**: postgres:16-alpine, old deaddrop:latest
- **Current state**: No running DeadDrop containers, no deployment directory
