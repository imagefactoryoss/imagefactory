CREATE TABLE IF NOT EXISTS messaging_outbox (
    id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    tenant_id UUID NULL,
    source TEXT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    schema_version TEXT NOT NULL,
    publish_attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messaging_outbox_pending
    ON messaging_outbox (next_attempt_at)
    WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_messaging_outbox_type_created
    ON messaging_outbox (event_type, created_at);

