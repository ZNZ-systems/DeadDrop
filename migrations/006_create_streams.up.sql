CREATE TABLE streams (
    id         BIGSERIAL PRIMARY KEY,
    public_id  UUID NOT NULL UNIQUE,
    mailbox_id BIGINT NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('form', 'email')),
    address    TEXT NOT NULL DEFAULT '',
    widget_id  UUID UNIQUE,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_streams_mailbox_id ON streams(mailbox_id);
CREATE INDEX idx_streams_widget_id ON streams(widget_id);
CREATE INDEX idx_streams_address ON streams(address) WHERE address != '';
