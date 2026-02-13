CREATE TABLE conversations (
    id         BIGSERIAL PRIMARY KEY,
    public_id  UUID NOT NULL UNIQUE,
    mailbox_id BIGINT NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    stream_id  BIGINT NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    subject    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_mailbox_id ON conversations(mailbox_id, created_at DESC);
CREATE INDEX idx_conversations_status ON conversations(mailbox_id, status);

CREATE TABLE conversation_messages (
    id              BIGSERIAL PRIMARY KEY,
    public_id       UUID NOT NULL UNIQUE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    direction       TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    sender_address  TEXT NOT NULL DEFAULT '',
    sender_name     TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conv_messages_conversation ON conversation_messages(conversation_id, created_at ASC);
