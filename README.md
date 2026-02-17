# DeadDrop

Self-hosted shared inbox for all your domains.

DeadDrop lets you collect inbound email and website form submissions into one dashboard you control. It is built for the “many projects, many domains, one mailbox stack” workflow.

## What You Get

- Domain ownership verification via DNS TXT.
- Mailboxes with two stream types:
  - `form` stream for website widget submissions.
  - `email` stream for inbound SMTP delivery.
- Conversation inbox with open/closed states and in-dashboard replies.
- Embeddable widget (`/static/widget.js`) that works on any site.
- One-command self-host installer for Linux servers.
- No external SaaS dependency required for core operation.

## Core Model

- `Domain`: a verified domain you control (for example `openclaw.london`).
- `Mailbox`: a team inbox under a domain (for example `Support`).
- `Stream`: a channel connected to a mailbox:
  - `form` stream has a `widget_id` for the JS embed.
  - `email` stream has an email address (for example `contact@openclaw.london`).
- `Conversation`: a thread created from a form submission or inbound email.

## Quick Start (Self-Hosted)

### Option A: One-Command Installer (Recommended)

Run this on your server:

```bash
curl -fsSL https://raw.githubusercontent.com/ZNZ-systems/DeadDrop/master/docker/install.sh | INSTALL_DIR=deaddrop DASHBOARD_PORT=8080 bash
```

What this does:

1. Installs Docker if missing (Linux).
2. Downloads production compose + env template.
3. Generates DB credentials and `DATABASE_URL`.
4. Starts Postgres, app, and Caddy.
5. Validates `/health` before finishing.

After install, open the printed dashboard URL (usually `http://<server-ip>:8080`).

### Option B: Local Dev Stack

```bash
git clone https://github.com/ZNZ-systems/DeadDrop.git
cd DeadDrop/docker
cp .env.example .env
docker compose up -d
```

Open [http://localhost:8080](http://localhost:8080).

## First-Time Dashboard Setup

1. Sign up (there are no default credentials).
2. Create a domain in `Domains`.
3. Add the DNS TXT record shown in the domain page.
4. Click `Check Verification`.
5. Create a mailbox in `Mailboxes` (example from-address: `contact@yourdomain.com`).
6. Use the generated streams:
   - `form` stream: copy widget snippet to your website.
   - `email` stream: route MX to your server and send mail to that address.

## Widget Setup (Important)

Use the **form stream widget ID**, not the domain ID.

```html
<script
  src="https://YOUR-DEADDROP-HOST/static/widget.js"
  data-deaddrop-id="FORM_STREAM_WIDGET_ID">
</script>
```

Where to find `FORM_STREAM_WIDGET_ID`:

- Dashboard → `Mailboxes` → open mailbox → `Streams` → `form` stream.

## DNS Setup Reference

For a domain like `openclaw.london`:

1. TXT verification record (from dashboard):
   - Type: `TXT`
   - Host/Name: `@`
   - Value: `deaddrop-verify=<token-from-dashboard>`

2. MX routing for inbound email:
   - Create an A record for your mail host (example `mx.openclaw.london -> <server-ip>`).
   - Add MX:
     - Host/Name: `@`
     - Value: `mx.openclaw.london`
     - Priority: `10`

3. Recommended deliverability records:
   - SPF: `v=spf1 mx a:mx.openclaw.london ip4:<server-ip> -all`
   - DMARC: `_dmarc` TXT like `v=DMARC1; p=none; rua=mailto:dmarc@yourdomain`

## Inbound SMTP Enablement

Inbound email is handled by the app’s SMTP listener (`INBOUND_SMTP_ADDR`).

In development compose, this is already wired (`25 -> 2525`).
For production-like setups, ensure both are true:

- `INBOUND_SMTP_ADDR` is set (for example `:2525`).
- Host port `25` is published to app container port `2525`.

Example compose override:

```yaml
services:
  app:
    ports:
      - "25:2525"
    environment:
      - INBOUND_SMTP_ADDR=:2525
      - INBOUND_SMTP_DOMAIN=yourdomain.com
```

## Outbound Email (Replies)

DeadDrop can send mailbox replies via SMTP.

Default approach:

- Use bundled `smtp-relay` container.
- App points at `SMTP_HOST=smtp-relay` and `SMTP_PORT=25`.

If your provider blocks direct port 25 egress, configure smarthost relay envs:

- `RELAYHOST`
- `RELAYHOST_USERNAME`
- `RELAYHOST_PASSWORD`

## Configuration

Primary env vars:

- `PORT` (default `8080`)
- `DATABASE_URL`
- `BASE_URL`
- `SECURE_COOKIES` (`false` for plain HTTP/IP testing)
- `SESSION_MAX_AGE_HOURS`
- `SMTP_ENABLED`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`
- `INBOUND_SMTP_ADDR`, `INBOUND_SMTP_DOMAIN`
- `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`
- `DNS_OVERRIDE_FILE` (used for deterministic e2e DNS tests)

## Running Tests

```bash
go test ./...
```

E2E harness:

```bash
bash e2e/run-e2e.sh
```

## Architecture

- Go + Chi router
- Server-rendered HTML templates + HTMX
- PostgreSQL (repository pattern)
- Embedded static assets/templates/migrations
- Optional inbound SMTP server in-process

Main entrypoint: `/Users/pz/CodeProjects/DeadDrop/cmd/deaddrop/main.go`

## Project Structure

- `/Users/pz/CodeProjects/DeadDrop/cmd/deaddrop` - app entrypoint
- `/Users/pz/CodeProjects/DeadDrop/internal/web` - router, handlers, middleware, renderer
- `/Users/pz/CodeProjects/DeadDrop/internal/domain` - domain verification service
- `/Users/pz/CodeProjects/DeadDrop/internal/mailbox` - mailbox logic
- `/Users/pz/CodeProjects/DeadDrop/internal/conversation` - inbox threads + replies
- `/Users/pz/CodeProjects/DeadDrop/internal/inbound` - SMTP inbound server
- `/Users/pz/CodeProjects/DeadDrop/internal/store/postgres` - persistence layer
- `/Users/pz/CodeProjects/DeadDrop/static` - widget and static files
- `/Users/pz/CodeProjects/DeadDrop/templates` - HTML templates
- `/Users/pz/CodeProjects/DeadDrop/docker` - docker compose + installer

## Troubleshooting

- `CSRF token mismatch`:
  - Usually `BASE_URL` / `SECURE_COOKIES` mismatch.
  - For IP + HTTP testing, set `BASE_URL=http://<ip>:8080` and `SECURE_COOKIES=false`.

- Widget script loads but no messages:
  - Confirm `data-deaddrop-id` is a **form stream widget ID**.
  - Confirm browser can reach `https://YOUR-DEADDROP-HOST/static/widget.js`.

- DNS verification not passing:
  - Verify TXT record value is exact.
  - Wait for propagation.
  - Re-run `Check Verification` in domain page.

- Inbound email not appearing:
  - Confirm MX points to host with A record.
  - Confirm SMTP port 25 reachable externally.
  - Confirm `INBOUND_SMTP_ADDR` is enabled.

## Contributing

1. Create a branch.
2. Make changes.
3. Run `go test ./...`.
4. Open a PR with scope and test notes.

## License

TBD
