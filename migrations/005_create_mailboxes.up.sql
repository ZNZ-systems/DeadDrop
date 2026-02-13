CREATE TABLE mailboxes (
    id           BIGSERIAL PRIMARY KEY,
    public_id    UUID NOT NULL UNIQUE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id    BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    from_address TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX idx_mailboxes_domain_id ON mailboxes(domain_id);
CREATE INDEX idx_mailboxes_public_id ON mailboxes(public_id);
