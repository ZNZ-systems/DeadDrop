CREATE TABLE inbound_ingest_jobs (
    id            BIGSERIAL PRIMARY KEY,
    status        TEXT NOT NULL CHECK (status IN ('queued', 'processing', 'done', 'failed')) DEFAULT 'queued',
    payload       JSONB NOT NULL,
    attempts      INTEGER NOT NULL DEFAULT 0,
    max_attempts  INTEGER NOT NULL DEFAULT 5,
    available_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at     TIMESTAMPTZ,
    last_error    TEXT NOT NULL DEFAULT '',
    accepted      INTEGER NOT NULL DEFAULT 0,
    dropped       INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    done_at       TIMESTAMPTZ
);

CREATE INDEX idx_inbound_ingest_jobs_claim
ON inbound_ingest_jobs(status, available_at, id)
WHERE status = 'queued';

CREATE INDEX idx_inbound_ingest_jobs_failed
ON inbound_ingest_jobs(status, updated_at DESC)
WHERE status = 'failed';
