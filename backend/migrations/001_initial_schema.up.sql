-- Migration: 001_initial_schema.up.sql
-- Category: Core Foundation
-- Purpose: Create foundational tables for multi-tenancy and user management

-- ============================================================================
-- EXTENSIONS
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION generate_numeric_id()
RETURNS INTEGER AS $$
DECLARE
    v_id INTEGER;
BEGIN
    LOOP
        v_id := (FLOOR(RANDOM() * (999999 - 100000 + 1)) + 100000)::INTEGER;
        IF NOT EXISTS (SELECT 1 FROM tenants WHERE numeric_id = v_id) THEN
            RETURN v_id;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION set_numeric_id()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.numeric_id IS NULL THEN
        NEW.numeric_id := generate_numeric_id();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 1. TENANTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_code VARCHAR(8) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active', -- active, suspended, deleted
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_tenant_code ON tenants(tenant_code);

-- ============================================================================
-- 2. USERS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    
    -- Authentication
    password_hash VARCHAR(255),
    is_ldap_user BOOLEAN DEFAULT false,
    ldap_username VARCHAR(255),
    auth_method VARCHAR(50) DEFAULT 'credentials', -- credentials, ldap, oidc, api_key
    
    -- MFA
    mfa_enabled BOOLEAN DEFAULT false,
    mfa_type VARCHAR(50), -- totp, sms, email, backup_code, or null/none
    mfa_secret VARCHAR(255),
    backup_codes TEXT, -- JSON array
    
    -- Account status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, suspended, deleted
    email_verified BOOLEAN DEFAULT false,
    email_verified_at TIMESTAMP WITH TIME ZONE,
    
    -- Profile
    profile_picture_url VARCHAR(500),
    timezone VARCHAR(50),
    preferred_language VARCHAR(10),
    
    -- Audit
    last_login_at TIMESTAMP WITH TIME ZONE,
    password_changed_at TIMESTAMP WITH TIME ZONE,
    
    -- Security & Account Locking
    failed_login_count INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    version INTEGER DEFAULT 1,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_ldap_username ON users(ldap_username);
CREATE INDEX IF NOT EXISTS idx_users_auth_method ON users(auth_method);
CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users(locked_until) WHERE locked_until IS NOT NULL;

-- ============================================================================
-- 3. USER SESSIONS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Session information
    access_token VARCHAR(500),
    refresh_token VARCHAR(500),
    
    -- Session tracking
    ip_address VARCHAR(45),
    user_agent VARCHAR(500),
    
    -- Validity
    expires_at TIMESTAMP WITH TIME ZONE,
    refreshed_at TIMESTAMP WITH TIME ZONE,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_id ON user_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active);

-- ============================================================================
-- 4. USER ROLE ASSIGNMENTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_role_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL, -- FK to roles table (created in migration 004)
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Scope
    scope VARCHAR(50) NOT NULL DEFAULT 'system', -- system, company, org_unit
    
    -- Assignment tracking
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Validity
    expires_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id, role_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_id ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role_id ON user_role_assignments(role_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_tenant_id ON user_role_assignments(tenant_id);

-- ============================================================================
-- 5. PROJECTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Project information
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Docker build configuration
    dockerfile_path VARCHAR(255) DEFAULT 'Dockerfile',
    build_context_path VARCHAR(255) DEFAULT '.',
    
    -- Image naming
    image_name_prefix VARCHAR(100),
    
    -- Build settings
    enable_cache BOOLEAN DEFAULT true,
    build_timeout_minutes INTEGER DEFAULT 30,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, archived, deleted
    visibility VARCHAR(50) DEFAULT 'private', -- private, internal, public

    -- Draft/ownership
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    is_draft BOOLEAN DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(tenant_id, slug),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);

-- ============================================================================
-- 5b. PROJECT SOURCES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS project_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    -- Source information
    name VARCHAR(120) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'generic',
    repository_url VARCHAR(500) NOT NULL,
    default_branch VARCHAR(120) NOT NULL DEFAULT 'main',
    repository_auth_id UUID,

    -- Status
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(project_id, name)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_sources_one_default
    ON project_sources(project_id)
    WHERE is_default = true AND is_active = true;
CREATE INDEX IF NOT EXISTS idx_project_sources_project_id ON project_sources(project_id);
CREATE INDEX IF NOT EXISTS idx_project_sources_tenant_id ON project_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_project_sources_repository_url ON project_sources(repository_url);

-- ============================================================================
-- 6. CATALOG IMAGES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS catalog_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Image identification
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Image details
    repository_url VARCHAR(500),
    registry_provider VARCHAR(50),
    
    -- Build information
    architecture VARCHAR(50),
    os VARCHAR(50),
    language VARCHAR(50),
    framework VARCHAR(100),
    version VARCHAR(50),
    
    -- Image size
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    size_bytes BIGINT,
    pull_count BIGINT DEFAULT 0,
    
    -- Status
    visibility VARCHAR(50) NOT NULL DEFAULT 'tenant', -- public, tenant, private
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- draft, published, deprecated, archived
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    deprecated_at TIMESTAMP WITH TIME ZONE,
    archived_at TIMESTAMP WITH TIME ZONE,

    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_catalog_images_tenant_id ON catalog_images(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_images_visibility ON catalog_images(visibility);
CREATE INDEX IF NOT EXISTS idx_catalog_images_status ON catalog_images(status);
CREATE INDEX IF NOT EXISTS idx_catalog_images_created_at ON catalog_images(created_at DESC);

-- ============================================================================
-- 7. CATALOG IMAGE VERSIONS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS catalog_image_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    catalog_image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE,
    version VARCHAR(100) NOT NULL,
    description TEXT,
    digest VARCHAR(255),
    size_bytes BIGINT,
    manifest JSONB DEFAULT '{}',
    config JSONB DEFAULT '{}',
    layers JSONB DEFAULT '[]',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(catalog_image_id, version)
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_versions_catalog_image_id ON catalog_image_versions(catalog_image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_versions_created_at ON catalog_image_versions(created_at DESC);

-- ============================================================================
-- 8. CATALOG IMAGE TAGS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS catalog_image_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    catalog_image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE,
    tag VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(catalog_image_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_tags_catalog_image_id ON catalog_image_tags(catalog_image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_tags_tag ON catalog_image_tags(tag);

-- ============================================================================
-- 9. BUILDS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS builds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    image_id UUID REFERENCES catalog_images(id) ON DELETE SET NULL,
    
    -- Build identification
    build_number INTEGER NOT NULL,
    
    -- Trigger information
    triggered_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    triggered_by_git_event VARCHAR(50), -- push, pull_request, manual, webhook
    
    -- Git information
    git_commit VARCHAR(40),
    git_branch VARCHAR(100),
    git_author_name VARCHAR(255),
    git_author_email VARCHAR(255),
    git_message TEXT,
    
    -- Build execution
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'queued', -- queued, in_progress, success, failed, cancelled

    -- Infrastructure selection
    infrastructure_type VARCHAR(50),
    infrastructure_reason TEXT,
    infrastructure_provider_id UUID,
    selected_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT valid_infrastructure_type
        CHECK (infrastructure_type IS NULL OR infrastructure_type IN ('kubernetes', 'build_node')),
    
    -- Results
    error_message TEXT,
    build_log_url VARCHAR(500),
    
    -- Cleanup
    cleanup_at TIMESTAMP WITH TIME ZONE,
    
    -- Dispatcher retry mechanism
    dispatch_attempts INTEGER NOT NULL DEFAULT 0,
    dispatch_next_run_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(project_id, build_number)
);

CREATE INDEX IF NOT EXISTS idx_builds_tenant_id ON builds(tenant_id);
CREATE INDEX IF NOT EXISTS idx_builds_project_id ON builds(project_id);
CREATE INDEX IF NOT EXISTS idx_builds_image_id ON builds(image_id);
CREATE INDEX IF NOT EXISTS idx_builds_status ON builds(status);
CREATE INDEX IF NOT EXISTS idx_builds_tenant_id_status ON builds(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_builds_created_at ON builds(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_builds_dispatch_next_run_at ON builds(dispatch_next_run_at);
CREATE INDEX IF NOT EXISTS idx_builds_infrastructure_type ON builds(infrastructure_type);
CREATE INDEX IF NOT EXISTS idx_builds_infrastructure_provider_id ON builds(infrastructure_provider_id);
CREATE INDEX IF NOT EXISTS idx_builds_selected_at ON builds(selected_at);

-- ============================================================================
-- 10. NOTIFICATIONS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Notification content
    title VARCHAR(255),
    message TEXT,
    
    -- Type
    notification_type VARCHAR(50), -- build_complete, build_failed, deployment, security_alert
    
    -- Reference
    related_resource_type VARCHAR(100),
    related_resource_id UUID,
    
    -- Status
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP WITH TIME ZONE,
    
    -- Channel
    channel VARCHAR(50), -- in_app, email, slack, teams
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_id ON notifications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);

-- ============================================================================
-- 12. SECURITY POLICIES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS security_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Policy information
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Policy type
    policy_type VARCHAR(50) NOT NULL, -- image_signing, vulnerability_scanning, registry_scan
    
    -- Enforcement
    is_enabled BOOLEAN DEFAULT true,
    enforce_on_deployment BOOLEAN DEFAULT true,
    
    -- Configuration
    policy_config TEXT, -- JSON with policy-specific config
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, deleted
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_security_policies_tenant_id ON security_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_policies_type ON security_policies(policy_type);

-- ============================================================================
-- 13. APPROVAL WORKFLOWS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS approval_workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Workflow information
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Workflow type
    workflow_type VARCHAR(50) NOT NULL, -- image_deployment, security_exception, policy_change
    
    -- Approval configuration
    required_approvers_count INTEGER DEFAULT 1,
    approver_roles TEXT, -- JSON array of role IDs
    
    -- Timeout
    approval_timeout_hours INTEGER DEFAULT 24,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_approval_workflows_tenant_id ON approval_workflows(tenant_id);

-- ============================================================================
-- 14. APPROVAL REQUESTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    approval_workflow_id UUID NOT NULL REFERENCES approval_workflows(id) ON DELETE CASCADE,
    
    -- Request details
    request_type VARCHAR(50),
    resource_type VARCHAR(100),
    resource_id UUID,
    
    -- Requester
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Context
    request_context TEXT, -- JSON with request details
    
    -- Approval status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, cancelled
    
    -- Approvals
    approved_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    
    -- Expiry
    expires_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_approval_requests_workflow_id ON approval_requests(approval_workflow_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_requests_requested_at ON approval_requests(requested_at DESC);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_user_sessions_updated_at
    BEFORE UPDATE ON user_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_catalog_images_updated_at
    BEFORE UPDATE ON catalog_images
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_builds_updated_at
    BEFORE UPDATE ON builds
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_security_policies_updated_at
    BEFORE UPDATE ON security_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_approval_workflows_updated_at
    BEFORE UPDATE ON approval_workflows
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_approval_requests_updated_at
    BEFORE UPDATE ON approval_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
