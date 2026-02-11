CREATE TABLE messages (
    id           BIGSERIAL PRIMARY KEY,
    public_id    UUID NOT NULL UNIQUE,
    domain_id    BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    sender_name  TEXT NOT NULL DEFAULT '',
    sender_email TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL,
    is_read      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_domain_created ON messages(domain_id, created_at DESC);
CREATE INDEX idx_messages_domain_read ON messages(domain_id, is_read);
