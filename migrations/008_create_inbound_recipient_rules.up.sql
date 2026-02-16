CREATE TABLE inbound_recipient_rules (
    id         BIGSERIAL PRIMARY KEY,
    domain_id  BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    rule_type  TEXT NOT NULL CHECK (rule_type IN ('exact', 'wildcard')),
    pattern    TEXT NOT NULL,
    action     TEXT NOT NULL CHECK (action IN ('inbox', 'drop')) DEFAULT 'inbox',
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(domain_id, rule_type, pattern)
);

CREATE INDEX idx_inbound_recipient_rules_domain_active
ON inbound_recipient_rules(domain_id, is_active);
