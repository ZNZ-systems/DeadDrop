# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is DeadDrop

DeadDrop is a Go web application for managing contact forms and email conversations. Users register domains, create mailboxes with streams (form widgets or email addresses), and manage conversations through a web dashboard. It includes an inbound SMTP server for receiving emails and an embeddable JavaScript widget for contact forms.

## Build & Run Commands

```bash
# Build binary
go build -o deaddrop ./cmd/deaddrop

# Run all tests
go test ./...

# Run a single package's tests
go test ./internal/domain/ -v

# Run with Docker Compose (starts app + PostgreSQL)
cd docker && docker compose up

# Multi-platform Docker build (for arm64 Pi deployment)
docker buildx build --platform linux/arm64 -t deaddrop:e2e -f docker/Dockerfile .

# E2E tests (run on Pi)
bash e2e/run-e2e.sh
```

## Architecture

### Layered Structure

```
cmd/deaddrop/main.go          → Entry point, wires all dependencies
internal/web/router.go         → Chi router, middleware stack, all route definitions
internal/web/handlers/         → HTTP handlers (one file per resource)
internal/web/middleware/        → Auth, CSRF, CORS, rate limiting, security headers
internal/web/render/            → Go html/template renderer
internal/{service}/             → Business logic services (auth, domain, mail, mailbox, message, conversation)
internal/store/store.go         → Repository interfaces (UserStore, SessionStore, DomainStore, etc.)
internal/store/postgres/        → PostgreSQL implementations of all store interfaces
internal/models/models.go       → All data models
internal/config/config.go       → Env-var based configuration
internal/inbound/               → Inbound SMTP server (receives emails → creates conversations)
```

### Key Patterns

- **Dependency injection via constructors**: All services and handlers receive dependencies as constructor args, wired in `main.go`.
- **Repository interfaces in `internal/store/store.go`**: All data access goes through interfaces (`UserStore`, `DomainStore`, `MailboxStore`, `StreamStore`, `ConversationStore`, etc.), with implementations in `internal/store/postgres/`.
- **Notifier/Sender interfaces**: Email sending uses `message.Notifier`, `conversation.Notifier`, and `conversation.Sender` interfaces. Production uses `mail.Service`; tests use `NoopNotifier`/`NoopSender`.
- **Pluggable DNS resolver**: `domain.DNSResolver` interface with `NetResolver` (production) and `FileResolver` (e2e testing via `DNS_OVERRIDE_FILE` env var).
- **Embedded filesystems**: `static/`, `templates/`, and `migrations/` use Go `embed.FS` for single-binary deployment.

### Data Model

Core entities: User → Domain → Mailbox → Stream (form/email) → Conversation → ConversationMessage. Messages (legacy contact form submissions) are linked directly to Domain.

### Migrations

SQL migrations in `migrations/` use golang-migrate format (`NNN_name.up.sql` / `NNN_name.down.sql`). Migrations run automatically on startup in `main.go`.

## Environment Configuration

All config is via environment variables (see `docker/.env.example`). Key vars:
- `DATABASE_URL` — Postgres connection string
- `SMTP_HOST` — enables outbound email when set
- `INBOUND_SMTP_ADDR` — enables inbound SMTP server when set (e.g., `:2525`)
- `INBOUND_SMTP_DOMAIN` — domain for inbound SMTP
- `DNS_OVERRIDE_FILE` — switches to file-based DNS resolver for e2e testing
- `SECURE_COOKIES` — set `false` for local dev (defaults to `true`)

## Testing

Tests use the standard Go testing package. Service-level tests mock store interfaces. Handler tests use `httptest` and test helpers in `internal/web/handlers/test_helpers_test.go`. E2E tests in `e2e/` use curl, swaks, and jq against a running Docker environment.
