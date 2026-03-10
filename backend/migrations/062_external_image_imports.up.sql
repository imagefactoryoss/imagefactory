CREATE TABLE IF NOT EXISTS external_image_imports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    sor_record_id VARCHAR(128) NOT NULL,
    source_registry VARCHAR(255) NOT NULL,
    source_image_ref VARCHAR(1024) NOT NULL,
    registry_auth_id UUID REFERENCES registry_auth(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    internal_image_ref VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_external_image_import_status
        CHECK (status IN ('pending', 'approved', 'importing', 'success', 'failed', 'quarantined'))
);

CREATE INDEX IF NOT EXISTS idx_external_image_imports_tenant_created_at
    ON external_image_imports (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_external_image_imports_tenant_status
    ON external_image_imports (tenant_id, status);
