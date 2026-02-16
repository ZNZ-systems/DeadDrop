CREATE TABLE inbound_email_raws (
    inbound_email_id BIGINT PRIMARY KEY REFERENCES inbound_emails(id) ON DELETE CASCADE,
    raw_source       TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE inbound_email_attachments (
    id               BIGSERIAL PRIMARY KEY,
    inbound_email_id BIGINT NOT NULL REFERENCES inbound_emails(id) ON DELETE CASCADE,
    file_name        TEXT NOT NULL,
    content_type     TEXT NOT NULL,
    size_bytes       BIGINT NOT NULL,
    content          BYTEA NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_email_attachments_email_id
ON inbound_email_attachments(inbound_email_id);
