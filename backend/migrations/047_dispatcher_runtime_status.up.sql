CREATE TABLE IF NOT EXISTS dispatcher_runtime_status (
    instance_id VARCHAR(128) PRIMARY KEY,
    mode VARCHAR(32) NOT NULL,
    running BOOLEAN NOT NULL DEFAULT FALSE,
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dispatcher_runtime_mode_heartbeat
    ON dispatcher_runtime_status (mode, last_heartbeat DESC);

CREATE INDEX IF NOT EXISTS idx_dispatcher_runtime_updated_at
    ON dispatcher_runtime_status (updated_at DESC);

