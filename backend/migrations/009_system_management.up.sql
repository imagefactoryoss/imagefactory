-- Migration: 009_system_management.up.sql
-- Category: System Management
-- Purpose: API keys, system configuration, and git integration management

-- ============================================================================
-- 1. API KEYS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_revoked_at ON api_keys(revoked_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);

-- Constraint: key_hash must not be empty
ALTER TABLE api_keys ADD CONSTRAINT ck_key_hash_not_empty CHECK (key_hash != '');

-- Constraint: name must not be empty
ALTER TABLE api_keys ADD CONSTRAINT ck_name_not_empty CHECK (name != '');

-- Add comment to describe table purpose
COMMENT ON TABLE api_keys IS 'External API keys for tenant-to-tenant authentication';
COMMENT ON COLUMN api_keys.key_hash IS 'Bcrypt-hashed API key for secure storage';
COMMENT ON COLUMN api_keys.scopes IS 'Array of permission scopes for this API key';
COMMENT ON COLUMN api_keys.revoked_at IS 'Timestamp when key was revoked (soft delete)';
COMMENT ON COLUMN api_keys.last_used_at IS 'Track last successful usage for security audit';

-- ============================================================================
-- 2. GIT INTEGRATION TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS git_integration (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    git_repository_id UUID NOT NULL REFERENCES git_repositories(id) ON DELETE CASCADE,
    
    -- Integration type
    integration_type VARCHAR(50) NOT NULL, -- webhook, ci_cd, auto_build, dependency_scan
    
    -- Configuration
    is_enabled BOOLEAN DEFAULT true,
    
    -- For webhook integrations
    webhook_event_types TEXT, -- JSON array of events to listen to: push, pull_request, release, tag
    webhook_url VARCHAR(500),
    webhook_secret_encrypted VARCHAR(500),
    
    -- For CI/CD integrations
    ci_cd_provider VARCHAR(50), -- github_actions, gitlab_ci, jenkins, circleci
    ci_cd_config_file_path VARCHAR(255),
    
    -- Build trigger configuration
    auto_build_on_push BOOLEAN DEFAULT false,
    auto_build_branches TEXT, -- JSON array of branch patterns
    auto_build_dockerfile_path VARCHAR(255),
    
    -- Scan configuration
    run_security_scans BOOLEAN DEFAULT false,
    run_compliance_checks BOOLEAN DEFAULT false,
    scan_frequency VARCHAR(50), -- on_push, daily, weekly
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, error, disabled
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_sync_status VARCHAR(50), -- success, failure, partial
    last_error_message TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(git_repository_id, integration_type)
);

CREATE INDEX IF NOT EXISTS idx_git_integration_company_id ON git_integration(company_id);
CREATE INDEX IF NOT EXISTS idx_git_integration_repository_id ON git_integration(git_repository_id);
CREATE INDEX IF NOT EXISTS idx_git_integration_is_enabled ON git_integration(is_enabled);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_git_integration_updated_at
    BEFORE UPDATE ON git_integration
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
