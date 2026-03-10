CREATE TABLE IF NOT EXISTS image_import_notification_receipts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(120) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id, event_type, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_image_import_notification_receipts_created_at
    ON image_import_notification_receipts(created_at DESC);
