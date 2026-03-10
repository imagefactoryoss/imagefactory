-- Migration: 025_system_configuration_table.up.sql
-- Category: System Configuration
-- Purpose: Create system_configs table for runtime configuration management

-- ============================================================================
-- SYSTEM CONFIGS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS system_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Tenant scope (NULL for universal configs, UUID for tenant-specific)
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    -- Configuration metadata
    config_type VARCHAR(50) NOT NULL, -- security, build, ldap, smtp, general, rate_limit, feature_flags, tool_settings, external_services, messaging, runtime_services
    config_key VARCHAR(255) NOT NULL,
    config_value JSONB NOT NULL,

    -- Status and management
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, inactive, testing
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT false,

    -- Audit fields
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Optimistic locking
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    UNIQUE(tenant_id, config_type, config_key),
    CHECK (status IN ('active', 'inactive', 'testing')),
    CHECK (config_type IN ('security', 'build', 'tekton', 'ldap', 'smtp', 'general', 'rate_limit', 'feature_flags', 'tool_settings', 'external_services', 'messaging', 'runtime_services'))
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_system_configs_tenant_id ON system_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_system_configs_config_type ON system_configs(config_type);
CREATE INDEX IF NOT EXISTS idx_system_configs_config_key ON system_configs(config_key);
CREATE INDEX IF NOT EXISTS idx_system_configs_status ON system_configs(status);

-- Composite indexes for uniqueness constraints
-- For tenant-specific configs (tenant_id IS NOT NULL), ensure uniqueness within tenant
CREATE UNIQUE INDEX idx_system_configs_tenant_type_key
ON system_configs(tenant_id, config_type, config_key)
WHERE tenant_id IS NOT NULL;

-- For universal configs (tenant_id IS NULL), ensure global uniqueness
CREATE UNIQUE INDEX idx_system_configs_universal_type_key
ON system_configs(config_type, config_key)
WHERE tenant_id IS NULL;

-- Updated at trigger
CREATE OR REPLACE FUNCTION update_system_configs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_system_configs_updated_at
    BEFORE UPDATE ON system_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_system_configs_updated_at();
