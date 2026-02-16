# DeadDrop Catch-All Email Task List

Status date: 2026-02-16

Goal: provide a production-ready catch-all inbox for many domains without buying separate mailboxes.

## Current Build Status
- [x] Core catch-all flow is implemented and working in app.
- [x] Tests currently pass (`go test ./...` and `go test -race ./...`).
- [ ] Public-launch hardening is complete.

## Launch Blockers (Must-Haves Before Public Launch)
- [ ] Move raw email + attachments to durable object storage (S3/R2) and enforce retention/quotas.
- [ ] Add async ingest/parse pipeline with retry + dead-letter queue (remove parse work from request path).
- [ ] Add abuse/auth signals: SPF, DKIM, DMARC capture + sender/domain/IP rate limits.
- [ ] Add spam/malware controls and suspicious-email labeling in inbox.
- [ ] Add production observability: metrics, alerts, structured logs, runbooks.
- [ ] Finalize legal + abuse process (privacy/AUP/abuse reporting path).

## Phase 0: Product + Architecture Baseline
- [x] v1 scope in code is receive-only catch-all inbox (not full outbound mailbox).
- [ ] Define retention policy (days + archive/delete lifecycle).
- [ ] Define SLA targets (latency, uptime, max message size).
- [ ] Document inbound provider strategy for production.
- [ ] Write ADR for queue + parser + storage architecture.

## Phase 1: Data Model + Migrations
- [x] `inbound_emails` table with dedupe + inbox indexes.
- [x] `inbound_domain_configs` table (MX target + verification state).
- [x] `inbound_recipient_rules` table (`exact`/`wildcard`, `inbox`/`drop`).
- [x] `inbound_email_raws` table.
- [x] `inbound_email_attachments` table.
- [ ] `mailboxes` table and mailbox-targeted routing.
- [ ] `email_recipients` normalized table (to/cc/bcc/envelope breakdown).

## Phase 2: Domain Onboarding
- [x] Domain email setup UI/API.
- [x] MX/DNS instructions per domain.
- [x] MX verification endpoint and checks.
- [x] Ingest gated on domain verified + MX verified.
- [ ] Automated status polling/recheck flow.

## Phase 3: Inbound Ingest Pipeline
- [x] Token-authenticated inbound endpoint (`POST /api/v1/inbound/emails`).
- [x] Multiple-recipient handling.
- [x] Recipient rule evaluation (`exact` then `wildcard`).
- [x] Catch-all fallback when no rules exist.
- [x] Duplicate protection via unique index for non-empty message IDs.
- [x] Request body size limit on inbound API.
- [ ] Persist raw source to durable storage before parse.
- [ ] Queue parse jobs asynchronously.
- [ ] Retry/dead-letter handling for failures.

## Phase 4: MIME Parsing + Normalization
- [x] Parse sender/subject/message-id/recipients from RFC822.
- [x] Extract plain text + HTML bodies.
- [x] Parse and persist attachment metadata + content.
- [x] Preserve raw source for debugging/download use.
- [ ] Sanitize HTML before rendering.
- [ ] Add parser failure state + retry flow.
- [ ] Add fuzz/property tests for malformed MIME hardening.

## Phase 5: Mail Auth + Abuse Protection
- [ ] Record SPF results.
- [ ] Record DKIM verification results.
- [ ] Record DMARC alignment results.
- [ ] Add spam scoring pipeline.
- [ ] Add abuse blocklists + rate limits.
- [ ] Add attachment malware scanning hook.

## Phase 6: Inbox UX
- [x] Inbox navigation + list view + detail view.
- [x] Read/unread + delete actions.
- [x] Attachment listing + download.
- [x] Search/filter endpoints and UI controls.
- [ ] Archive/restore lifecycle.
- [ ] Pagination/infinite scroll.
- [ ] Spam/suspicious badges and filtering.

## Phase 7: Rules + Routing
- [x] Rule UI/API for exact + wildcard recipient patterns.
- [x] Domain-level catch-all behavior (when no rules are configured).
- [x] `drop` action with auditability via stored inbound rows only for accepted messages.
- [ ] Forward-to-external action.
- [ ] Rule conflict detection + explicit precedence diagnostics.

## Phase 8: Notifications + Forwarding
- [ ] Queue-based notification worker.
- [ ] Inbound email notifications/digests.
- [ ] Forwarding with retries + bounce tracking.
- [ ] Per-user notification preferences.

## Phase 9: Security, Compliance, Reliability
- [ ] Encryption-at-rest strategy for raw + attachment blobs.
- [ ] Secret rotation/management for inbound provider credentials.
- [ ] Audit logs for domain/rule/security changes.
- [ ] Backup + restore drills for mail data.
- [ ] Data export + deletion workflows.

## Phase 10: Observability + Operations
- [ ] Metrics: ingest rate, parse failures, spam ratio, queue depth, notification failures.
- [ ] Tracing across ingest -> parse -> store -> notify.
- [ ] Dashboards + alert thresholds.
- [ ] Correlation IDs in logs.
- [ ] Incident runbooks.

## Phase 11: Testing + QA Matrix
- [x] Unit/integration tests for implemented inbound flow and routing.
- [ ] Fuzz/property tests for MIME parser.
- [ ] Load tests for burst traffic.
- [ ] Security tests for malformed/oversized payloads.
- [ ] Cross-client rendering verification.

## Phase 12: Rollout Plan
- [ ] Internal alpha on owned domains.
- [ ] Private beta with selected users.
- [ ] Gradual rollout with per-account caps.
- [ ] Public launch checklist completed.

## Definition of Done for Catch-All v1
- [x] User can add domain, set MX, verify, and receive catch-all mail.
- [x] Messages are parsed and viewable with attachments.
- [x] Search/filter is available in inbox.
- [ ] Abuse controls and auth signals (SPF/DKIM/DMARC) are active.
- [ ] System has production-grade reliability/visibility under expected traffic.
