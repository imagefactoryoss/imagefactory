CREATE TABLE IF NOT EXISTS registry_auth (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    registry_type VARCHAR(50) NOT NULL,
    auth_type VARCHAR(50) NOT NULL,
    registry_host VARCHAR(255) NOT NULL,
    credential_data BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_registry_auth_type CHECK (auth_type IN ('basic_auth', 'token', 'dockerconfigjson')),
    CONSTRAINT chk_registry_auth_scope CHECK (
        (project_id IS NULL) OR (project_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_registry_auth_tenant_name
    ON registry_auth (tenant_id, name)
    WHERE project_id IS NULL;

CREATE UNIQUE INDEX uq_registry_auth_project_name
    ON registry_auth (project_id, name)
    WHERE project_id IS NOT NULL;

CREATE UNIQUE INDEX uq_registry_auth_default_tenant
    ON registry_auth (tenant_id)
    WHERE project_id IS NULL AND is_default = true AND is_active = true;

CREATE UNIQUE INDEX uq_registry_auth_default_project
    ON registry_auth (project_id)
    WHERE project_id IS NOT NULL AND is_default = true AND is_active = true;

CREATE INDEX IF NOT EXISTS idx_registry_auth_tenant ON registry_auth (tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_registry_auth_project ON registry_auth (project_id, is_active);
