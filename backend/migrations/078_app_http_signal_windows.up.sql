CREATE TABLE IF NOT EXISTS app_http_signal_windows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_count BIGINT NOT NULL DEFAULT 0,
    server_error_count BIGINT NOT NULL DEFAULT 0,
    client_error_count BIGINT NOT NULL DEFAULT 0,
    total_latency_ms BIGINT NOT NULL DEFAULT 0,
    average_latency_ms BIGINT NOT NULL DEFAULT 0,
    max_latency_ms BIGINT NOT NULL DEFAULT 0,
    window_started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    window_ended_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_http_signal_windows_ended_at
    ON app_http_signal_windows(window_ended_at DESC);

COMMENT ON TABLE app_http_signal_windows IS 'Short-window app-level HTTP traffic, error, and latency snapshots used by SRE Smart Bot golden-signal analysis.';
