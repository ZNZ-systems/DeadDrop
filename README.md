# DeadDrop

**Own your contact forms. Own your conversations. Own your data.**

DeadDrop is a self-hosted platform for receiving messages from your website visitors and managing email conversations — without handing your data to a third party. Drop a single script tag on your site, point your MX records at your server, and you have a complete contact and support system running on hardware you control.

---

## Why DeadDrop?

Most contact form services are SaaS black boxes. Your visitors' messages flow through someone else's servers, get stored in someone else's database, and are subject to someone else's pricing changes and privacy policies.

DeadDrop flips this. It's a single Go binary + PostgreSQL that gives you:

- **Embeddable contact widget** — one `<script>` tag, Shadow DOM isolated, works on any site
- **Inbound SMTP server** — receive emails directly, no third-party forwarding
- **Conversation threads** — reply to visitors from your dashboard, track open/closed status
- **Multi-domain support** — manage contact forms across all your domains from one place
- **DNS-verified domain ownership** — prove you own a domain before receiving messages for it
- **Runs on a Raspberry Pi** — ARM64 builds included, because your homelab deserves nice things

## Quick Start

### Docker Compose (recommended)

```bash
git clone https://github.com/ZNZ-systems/DeadDrop.git
cd DeadDrop/docker
cp .env.example .env
docker compose up
```

The app is now running at **http://localhost:8080**. Sign up, add a domain, and you're live.

### From Source

```bash
git clone https://github.com/ZNZ-systems/DeadDrop.git
cd DeadDrop

# Requires Go 1.25+ and a running PostgreSQL instance
export DATABASE_URL="postgres://user:pass@localhost:5432/deaddrop?sslmode=disable"
go build -o deaddrop ./cmd/deaddrop
./deaddrop
```

## Embed the Widget

Once you've added and verified a domain in the dashboard, drop this on your website:

```html
<script
  src="https://your-deaddrop-instance.com/static/widget.js"
  data-deaddrop-id="YOUR_DOMAIN_ID">
</script>
```

That's it. A floating contact button appears in the bottom-right corner. Messages land in your dashboard instantly.

The widget:
- Uses **Shadow DOM** — zero CSS conflicts with your site
- Includes a **honeypot field** for bot filtering
- Works on any site, any framework, any static host
- No cookies, no tracking, no third-party requests

## Receive Email via SMTP

DeadDrop includes a built-in inbound SMTP server. Create a mailbox, add an email stream address (e.g. `support@yourdomain.com`), point your MX records, and incoming emails automatically become conversations in your dashboard.

```bash
# Enable inbound SMTP in your environment
INBOUND_SMTP_ADDR=":25"
INBOUND_SMTP_DOMAIN="yourdomain.com"
```

## Production Deployment

The production Docker Compose setup includes Caddy for automatic HTTPS:

```bash
cd docker
cp .env.example .env.prod
# Edit .env.prod with your production values

export DOMAIN=deaddrop.yourdomain.com
export POSTGRES_USER=deaddrop
export POSTGRES_PASSWORD=<strong-password>
export POSTGRES_DB=deaddrop

docker compose -f docker-compose.prod.yml up -d
```

This gives you:
- Caddy reverse proxy with **automatic TLS** via Let's Encrypt
- Health-checked app container with auto-restart
- Persistent PostgreSQL with Docker volumes
- Bundled outbound SMTP relay for mailbox replies (no external SMTP creds required)

If your host blocks outbound port `25`, configure `RELAYHOST` credentials in `.env.prod` to use a smarthost (for example Mailgun/SES/Resend SMTP).

### ARM64 / Raspberry Pi

Build a multi-platform image and deploy to your Pi:

```bash
docker buildx build --platform linux/arm64 -t deaddrop:latest -f docker/Dockerfile .
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Clients                              │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ Contact Form │  │  Dashboard   │  │   Email (SMTP)    │  │
│  │   Widget     │  │   Browser    │  │   Inbound Mail    │  │
│  └──────┬───────┘  └──────┬───────┘  └────────┬──────────┘  │
└─────────┼─────────────────┼───────────────────┼─────────────┘
          │ POST /api/v1    │ HTML + HTMX       │ SMTP :25
          │ /messages       │                   │
┌─────────▼─────────────────▼───────────────────▼─────────────┐
│                      DeadDrop Server                        │
│                                                             │
│  ┌─────────┐  ┌──────────┐  ┌────────────┐  ┌───────────┐  │
│  │ Chi     │  │ Auth &   │  │ Domain     │  │ Inbound   │  │
│  │ Router  │  │ Sessions │  │ Verify     │  │ SMTP      │  │
│  │ + CSRF  │  │          │  │ (DNS TXT)  │  │ Server    │  │
│  └────┬────┘  └──────────┘  └────────────┘  └─────┬─────┘  │
│       │                                           │         │
│  ┌────▼───────────────────────────────────────────▼──────┐  │
│  │              Service Layer                            │  │
│  │  Mailbox · Conversation · Message · Mail · Domain     │  │
│  └───────────────────────┬───────────────────────────────┘  │
│                          │                                  │
│  ┌───────────────────────▼───────────────────────────────┐  │
│  │           PostgreSQL Store (Repository Pattern)       │  │
│  └───────────────────────┬───────────────────────────────┘  │
└──────────────────────────┼──────────────────────────────────┘
                           │
                    ┌──────▼──────┐
                    │ PostgreSQL  │
                    │     16      │
                    └─────────────┘
```

### Key concepts

| Concept | Description |
|---|---|
| **Domain** | A verified website domain (e.g. `example.com`). Verified via DNS TXT record. |
| **Mailbox** | A named inbox under a domain (e.g. "Support", "Sales"). |
| **Stream** | A message source attached to a mailbox — either a `form` widget or an `email` address. |
| **Conversation** | A thread of messages between a visitor and you. Created from widget submissions or inbound emails. |

### Data flow

1. Visitor submits a form via the widget or sends an email to a stream address
2. DeadDrop creates a conversation in the matching mailbox
3. Owner gets an email notification (if SMTP is configured)
4. Owner views and replies from the dashboard
5. Reply is sent back to the visitor via email

## Configuration

All configuration is through environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://...localhost...` | PostgreSQL connection string |
| `BASE_URL` | `http://localhost:8080` | Public URL of your instance |
| `SECURE_COOKIES` | `true` | Set `false` for local development |
| `SMTP_HOST` | `smtp-relay` | Outbound SMTP host (bundled relay by default) |
| `SMTP_PORT` | `25` | Outbound SMTP port |
| `SMTP_ENABLED` | `true` | Set to `false` to disable outbound email entirely |
| `SMTP_USER` | *(empty)* | SMTP username |
| `SMTP_PASS` | *(empty)* | SMTP password |
| `SMTP_FROM` | `deaddrop@localhost` | "From" address for outbound notifications |
| `RELAYHOST` | *(empty)* | Optional smarthost (e.g. `[smtp.mailgun.org]:587`) for postfix relay container |
| `RELAYHOST_USERNAME` | *(empty)* | Optional smarthost username |
| `RELAYHOST_PASSWORD` | *(empty)* | Optional smarthost password |
| `INBOUND_SMTP_ADDR` | *(empty)* | Inbound SMTP listen address — enables SMTP server when set (e.g. `:25`) |
| `INBOUND_SMTP_DOMAIN` | `localhost` | Domain for inbound SMTP |
| `RATE_LIMIT_RPS` | `2` | API rate limit (requests/sec) |
| `RATE_LIMIT_BURST` | `5` | API rate limit burst |
| `SESSION_MAX_AGE_HOURS` | `72` | Session expiry |

## Development

```bash
# Run the full stack locally
cd docker && docker compose up

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/auth/ -v
go test ./internal/domain/ -v

# Run end-to-end tests
bash e2e/run-e2e.sh
```

### Project layout

```
cmd/deaddrop/          Entry point — wires all dependencies
internal/
  auth/                Authentication & session management
  config/              Environment-based configuration
  conversation/        Conversation threads & replies
  domain/              Domain registration & DNS verification
  inbound/             Inbound SMTP server
  mail/                Outbound email notifications
  mailbox/             Mailbox CRUD
  message/             Legacy contact form messages
  models/              Data models
  ratelimit/           Token bucket rate limiter
  store/               Repository interfaces
    postgres/          PostgreSQL implementations
  web/
    handlers/          HTTP request handlers
    middleware/        Auth, CSRF, CORS, security headers, rate limiting
    render/            Go template renderer
migrations/            SQL migrations (golang-migrate format)
static/                Widget JS and static assets
templates/             Go HTML templates (server-side rendered)
e2e/                   End-to-end test suite
docker/                Docker and Compose configs
```

### Database migrations

Migrations live in `migrations/` and use [golang-migrate](https://github.com/golang-migrate/migrate) format (`NNN_name.up.sql` / `NNN_name.down.sql`). They run automatically on startup.

## Tech Stack

- **Go** with [Chi](https://github.com/go-chi/chi) router
- **PostgreSQL 16** for storage
- **HTMX** for dynamic dashboard interactions
- **Shadow DOM** widget — zero dependencies, zero conflicts
- **Caddy** for production reverse proxy + auto-HTTPS
- **Docker** with multi-platform builds (amd64 + arm64)

## Contributing

Contributions are welcome! Please open an issue first to discuss what you'd like to change.

## License

TBD
