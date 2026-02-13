# Mailbox Flows Design

## Overview

Transform DeadDrop from a passive contact form collector into a self-hosted mailbox system with inbound SMTP, conversation threading, and dashboard-based replies.

## Requirements

- **Mailboxes** are standalone, top-level entities belonging to a user
- **Incoming streams** feed into mailboxes (many-to-one): contact form widgets and inbound email
- **Outbox**: users reply to conversations from the dashboard; replies sent Resend-style from any address on a verified domain
- **Inbound SMTP**: self-hosted Go SMTP server bundled in the same binary, exposed via Docker
- **Deployment**: Docker Compose, fully self-hostable OSS

## Data Model

```
User (existing)
 +-- Domain (existing, refocused: DNS verification + sending authorization)
 |    +-- SPF/DKIM verification status
 |
 +-- Mailbox (NEW)
      +-- name: "Support", "Sales"
      +-- from_address: "support@example.com"
      +-- domain_id: FK -> Domain (for sending authorization)
      |
      +-- Stream[] (NEW, many-to-one)
      |    +-- type: "form" | "email"
      |    +-- address: "contact@example.com" (email streams)
      |    +-- widget_id: UUID (form streams)
      |    +-- enabled: bool
      |
      +-- Conversation[] (NEW)
           +-- subject: string
           +-- status: "open" | "closed"
           +-- stream_id: FK -> Stream (how it arrived)
           |
           +-- Message[] (refactored)
                +-- direction: "inbound" | "outbound"
                +-- sender_address: string
                +-- body: text
                +-- created_at: timestamp
```

### Key Relationships

- A Mailbox has one `from_address` for outbound replies, tied to a verified Domain
- A Stream defines one input channel (form widget or email address) and feeds exactly one Mailbox
- A Conversation groups an initial message + all replies
- A Message is either inbound or outbound, always part of a Conversation

### Migration Strategy

New tables alongside existing ones. Existing `messages` data migrated into `conversations` + `messages`. Existing `domains` remain but gain DKIM/SPF columns.

## SMTP Architecture

### Inbound SMTP Server

A Go SMTP server (`github.com/emersion/go-smtp`) runs as a goroutine within the same binary.

**Flow:**
1. External sender connects on port 25
2. Go SMTP server parses envelope (RCPT TO)
3. Look up Stream by recipient address
4. Parse MIME body (`github.com/emersion/go-message`)
5. Create Conversation + inbound Message
6. Trigger notification to mailbox owner

### Outbound SMTP (Replies)

When a user replies from the dashboard:
1. POST to reply endpoint
2. Create outbound Message in Conversation
3. Send via SMTP with From: mailbox's `from_address`
4. Relies on verified domain DNS (SPF/DKIM)

### Docker Compose

```yaml
services:
  deaddrop:
    ports:
      - "8080:8080"   # HTTP
      - "25:2525"     # SMTP inbound
    environment:
      - SMTP_LISTEN_ADDR=:2525
      - SMTP_DOMAIN=mail.deaddrop.example
```

## Service Layer

| Service | Responsibility |
|---------|---------------|
| `mailbox.Service` | CRUD for mailboxes, validate from_address against verified domains |
| `stream.Service` | CRUD for streams, generate widget IDs, register email addresses |
| `conversation.Service` | Create conversations, add replies, manage open/closed status |
| `inbound.Service` | SMTP session handler: receive email, resolve stream, create conversation |

Existing `message.Service` replaced by `conversation.Service`. Existing `domain.Service` stays, gains SPF/DKIM guidance methods.

## Routes

```
# Mailbox management
GET    /mailboxes                          -> list user's mailboxes
GET    /mailboxes/new                      -> new mailbox form
POST   /mailboxes                          -> create mailbox
GET    /mailboxes/{id}                     -> mailbox detail (conversations)
POST   /mailboxes/{id}/delete              -> delete mailbox

# Stream management
POST   /mailboxes/{id}/streams             -> add stream
POST   /mailboxes/{id}/streams/{sid}/delete -> remove stream

# Conversations
GET    /mailboxes/{id}/conversations/{cid} -> view thread
POST   /mailboxes/{id}/conversations/{cid}/reply -> send reply
POST   /mailboxes/{id}/conversations/{cid}/close -> close conversation

# Public API
POST   /api/v1/messages                    -> form widget (creates conversation)
```

## Dashboard Changes

- Dashboard shifts from "domains list" to "mailboxes list"
- Mailbox view: name, from address, unread count
- Conversation list within mailbox (most recent first)
- Conversation detail: threaded messages, reply form at bottom
- Stream management in mailbox settings

## Notifications

- New conversation: email notification to user's account email (not mailbox from_address)
- Include link to conversation in dashboard
- No notification for replies to existing conversations (configurable later)
- Fire-and-forget, errors logged

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Inbound email to unknown address | SMTP rejects with 550 |
| Inbound email to disabled stream | SMTP rejects with 550 |
| Outbound reply fails | Store as "failed", show error in UI, allow retry |
| Domain not verified for outbox | Block reply with error |
| SMTP overloaded | Rate limit connections |

## Testing Strategy

- Unit tests for each new service
- Integration test for SMTP inbound flow
- Existing tests updated for conversations migration
