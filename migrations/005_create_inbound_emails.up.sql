CREATE TABLE inbound_emails (
    id            BIGSERIAL PRIMARY KEY,
    public_id     UUID NOT NULL UNIQUE,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id     BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    recipient     TEXT NOT NULL,
    sender        TEXT NOT NULL,
    subject       TEXT NOT NULL DEFAULT '',
    text_body     TEXT NOT NULL DEFAULT '',
    html_body     TEXT NOT NULL DEFAULT '',
    message_id    TEXT NOT NULL DEFAULT '',
    is_read       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent duplicate deliveries for the same message to the same recipient.
CREATE UNIQUE INDEX uq_inbound_emails_message_recipient
ON inbound_emails(message_id, recipient)
WHERE message_id <> '';

CREATE INDEX idx_inbound_emails_user_created
ON inbound_emails(user_id, created_at DESC);

CREATE INDEX idx_inbound_emails_user_read
ON inbound_emails(user_id, is_read);

CREATE INDEX idx_inbound_emails_domain_created
ON inbound_emails(domain_id, created_at DESC);
