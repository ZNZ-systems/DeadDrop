CREATE TABLE domains (
    id                 BIGSERIAL PRIMARY KEY,
    public_id          UUID NOT NULL UNIQUE,
    user_id            BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    verification_token TEXT NOT NULL,
    verified           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_domains_public_id ON domains(public_id);
CREATE INDEX idx_domains_user_id ON domains(user_id);
CREATE INDEX idx_domains_name ON domains(name);
