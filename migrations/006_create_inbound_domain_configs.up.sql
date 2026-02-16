CREATE TABLE inbound_domain_configs (
    domain_id     BIGINT PRIMARY KEY REFERENCES domains(id) ON DELETE CASCADE,
    mx_target     TEXT NOT NULL,
    mx_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    last_error    TEXT NOT NULL DEFAULT '',
    checked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_domain_configs_mx_verified
ON inbound_domain_configs(mx_verified);
