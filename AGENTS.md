# AGENTS.md

Guidance for coding agents and contributors working in this repository.

## Mission

DeadDrop is a self-hosted shared inbox for many domains. The core promise is:

- one dashboard,
- one deployable stack,
- domain onboarding inside the app,
- no mandatory third-party SaaS dependencies.

When making changes, optimize for self-host reliability and operator clarity.

## Product Truths (Do Not Break)

1. No default login credentials. Users sign up in-app.
2. Domain is onboarded in dashboard, not required at initial installer runtime.
3. `data-deaddrop-id` in widget snippet is the **form stream widget ID**, not domain ID.
4. Main navigation is sectioned as:
   - `Domains` at `/`
   - `Mailboxes` at `/mailboxes`
5. Catch-all behavior depends on DNS + SMTP plumbing, not only dashboard config.

## Local Commands

- Build:

```bash
go build -o deaddrop ./cmd/deaddrop
```

- Test:

```bash
go test ./...
```

- Local stack:

```bash
cd docker && docker compose up -d
```

- E2E harness:

```bash
bash e2e/run-e2e.sh
```

## Key Files

- Entrypoint: `/Users/pz/CodeProjects/DeadDrop/cmd/deaddrop/main.go`
- Router: `/Users/pz/CodeProjects/DeadDrop/internal/web/router.go`
- Template renderer: `/Users/pz/CodeProjects/DeadDrop/internal/web/render/render.go`
- Domain handlers: `/Users/pz/CodeProjects/DeadDrop/internal/web/handlers/domains.go`
- Mailbox handlers: `/Users/pz/CodeProjects/DeadDrop/internal/web/handlers/mailboxes.go`
- Widget script: `/Users/pz/CodeProjects/DeadDrop/static/widget.js`
- Templates: `/Users/pz/CodeProjects/DeadDrop/templates`
- Installer: `/Users/pz/CodeProjects/DeadDrop/docker/install.sh`
- Production compose: `/Users/pz/CodeProjects/DeadDrop/docker/docker-compose.prod.yml`

## Implementation Standards

1. Keep dependencies minimal; prefer standard library and existing packages.
2. Preserve environment-variable configuration style (`internal/config/config.go`).
3. Respect repository pattern boundaries (`internal/store` interfaces).
4. Add/maintain tests for behavioral changes, especially handlers/services.
5. If changing schema behavior, add migration files in `/migrations`.
6. Keep installer output step-based with explicit validation checks.

## Self-Host UX Requirements

When editing install/deploy/onboarding flows:

1. Dashboard must be reachable by explicit URL after install.
2. Instructions must be sequential and verifiable by operators.
3. Docs must include both:
   - IP-based HTTP testing path
   - domain + HTTPS public path
4. Any DNS step must specify record Type + Host/Name + Value.

## SMTP & DNS Safety Notes

- Inbound mail requires `INBOUND_SMTP_ADDR` and port publication from host to app SMTP listener.
- Outbound delivery quality depends on SPF/DKIM/DMARC alignment and server reputation.
- Do not claim guaranteed inbox placement; keep deliverability language precise.

## Change Process

1. Understand the current behavior from code, not assumptions.
2. Implement minimal coherent change.
3. Run `go test ./...`.
4. For deploy-impacting changes, update README in same PR.
5. Include operator-facing verification steps in PR notes.

## Definition Of Done

A change is done when:

1. Tests pass locally.
2. Documentation reflects new behavior.
3. No contradictions remain between installer, compose files, and README.
4. Self-host operator can execute steps without tribal knowledge.
