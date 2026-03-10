CREATE TABLE IF NOT EXISTS tenant_notification_trigger_prefs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trigger_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    channels JSONB NOT NULL DEFAULT '["in_app"]'::jsonb,
    recipient_policy TEXT NOT NULL DEFAULT 'initiator',
    custom_recipient_user_ids JSONB NULL,
    severity_override TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tenant_notification_trigger_prefs_trigger_chk CHECK (trigger_id ~ '^BN-[0-9]{3}$'),
    CONSTRAINT tenant_notification_trigger_prefs_recipient_policy_chk CHECK (recipient_policy IN ('initiator', 'project_members', 'tenant_admins', 'custom_users')),
    CONSTRAINT tenant_notification_trigger_prefs_severity_chk CHECK (severity_override IS NULL OR severity_override IN ('low', 'normal', 'high')),
    CONSTRAINT tenant_notification_trigger_prefs_channels_array_chk CHECK (jsonb_typeof(channels) = 'array'),
    CONSTRAINT tenant_notification_trigger_prefs_custom_users_array_chk CHECK (custom_recipient_user_ids IS NULL OR jsonb_typeof(custom_recipient_user_ids) = 'array')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_notification_trigger_prefs_tenant_trigger
    ON tenant_notification_trigger_prefs(tenant_id, trigger_id);

