-- Migration: 006_container_lifecycle_management.up.sql
-- Category: Container Lifecycle Management
-- Purpose: Container registries, repositories, deployments, and GitOps

-- ============================================================================
-- 1. CONTAINER REGISTRIES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS container_registries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Registry information
    name VARCHAR(255) NOT NULL,
    description TEXT,
    registry_type VARCHAR(50) NOT NULL, -- docker_hub, ecr, gcr, harbor, quay, artifactory, nexus, private
    
    -- Connection details
    registry_url VARCHAR(500) NOT NULL UNIQUE,
    auth_method VARCHAR(50) NOT NULL DEFAULT 'credentials', -- credentials, oidc, api_key
    username VARCHAR(255),
    password_encrypted VARCHAR(500), -- Encrypted password
    api_token_encrypted VARCHAR(500), -- Encrypted API token
    
    -- TLS/SSL
    verify_ssl BOOLEAN DEFAULT true,
    ca_certificate_pem TEXT,
    
    -- Capabilities
    supports_push BOOLEAN DEFAULT true,
    supports_pull BOOLEAN DEFAULT true,
    supports_delete BOOLEAN DEFAULT false,
    
    -- Status and configuration
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, deleted
    is_default BOOLEAN DEFAULT false,
    
    -- Test and verification
    last_connectivity_check_at TIMESTAMP WITH TIME ZONE,
    last_connectivity_status VARCHAR(50), -- success, failed
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_container_registries_company_id ON container_registries(company_id);
CREATE INDEX IF NOT EXISTS idx_container_registries_type ON container_registries(registry_type);
CREATE INDEX IF NOT EXISTS idx_container_registries_status ON container_registries(status);

-- ============================================================================
-- 2. CONTAINER REPOSITORIES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS container_repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    registry_id UUID NOT NULL REFERENCES container_registries(id) ON DELETE CASCADE,
    
    -- Repository information
    name VARCHAR(255) NOT NULL,
    full_name VARCHAR(500) NOT NULL, -- e.g., 'my-org/my-repo'
    description TEXT,
    
    -- Ownership
    owned_by_org_unit_id UUID REFERENCES org_units(id) ON DELETE SET NULL,
    
    -- Access controls
    visibility VARCHAR(50) DEFAULT 'private', -- private, internal, public
    
    -- Image storage
    image_count INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    
    -- Configuration
    immutable_tags BOOLEAN DEFAULT false, -- Prevent tag overwrite
    cleanup_policy VARCHAR(50), -- daily, weekly, monthly, never
    max_retention_days INTEGER,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, archived, deleted
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(registry_id, full_name)
);

CREATE INDEX IF NOT EXISTS idx_container_repositories_company_id ON container_repositories(company_id);
CREATE INDEX IF NOT EXISTS idx_container_repositories_registry_id ON container_repositories(registry_id);
CREATE INDEX IF NOT EXISTS idx_container_repositories_org_unit_id ON container_repositories(owned_by_org_unit_id);

-- ============================================================================
-- 3. GIT REPOSITORIES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS git_repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Repository information
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL,
    description TEXT,
    
    -- Git provider
    provider VARCHAR(50) NOT NULL, -- github, gitlab, gitea, bitbucket, azure_devops
    provider_repo_id VARCHAR(255), -- ID from the provider (e.g., GitHub repo ID)
    
    -- Authentication
    ssh_key_name VARCHAR(255), -- Reference to stored SSH key
    personal_access_token_encrypted VARCHAR(500),
    
    -- Default branch
    default_branch VARCHAR(100) DEFAULT 'main',
    
    -- Webhook information
    webhook_url VARCHAR(500),
    webhook_secret_encrypted VARCHAR(500),
    webhook_enabled BOOLEAN DEFAULT true,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, archived, deleted
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, url)
);

CREATE INDEX IF NOT EXISTS idx_git_repositories_company_id ON git_repositories(company_id);
CREATE INDEX IF NOT EXISTS idx_git_repositories_provider ON git_repositories(provider);

-- ============================================================================
-- 4. DEPLOYMENT ENVIRONMENTS TABLE (must be created before deployments)
-- ============================================================================
CREATE TABLE IF NOT EXISTS deployment_environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Environment information
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- Environment type
    environment_type VARCHAR(50) NOT NULL, -- development, staging, production, custom
    
    -- Owner
    owned_by_org_unit_id UUID REFERENCES org_units(id) ON DELETE SET NULL,
    
    -- Infrastructure details
    infrastructure_provider VARCHAR(50), -- kubernetes, docker_swarm, lambda, ecs, cloudrun, custom
    cluster_name VARCHAR(255),
    namespace VARCHAR(100),
    
    -- Configuration
    requires_approval BOOLEAN DEFAULT false,
    max_concurrent_deployments INTEGER DEFAULT 1,
    auto_rollback_on_failure BOOLEAN DEFAULT true,
    
    -- Resource limits
    cpu_limit VARCHAR(50),
    memory_limit VARCHAR(50),
    storage_limit_gb INTEGER,
    
    -- Monitoring and logging
    log_aggregation_enabled BOOLEAN DEFAULT true,
    metrics_collection_enabled BOOLEAN DEFAULT true,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, maintenance, deleted
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS idx_deployment_environments_company_id ON deployment_environments(company_id);
CREATE INDEX IF NOT EXISTS idx_deployment_environments_org_unit_id ON deployment_environments(owned_by_org_unit_id);

-- ============================================================================
-- 5. DEPLOYMENTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS deployments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Deployment details
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- What's being deployed
    image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE RESTRICT,
    
    -- Where it's deployed
    deployment_environment_id UUID NOT NULL REFERENCES deployment_environments(id) ON DELETE CASCADE,
    
    -- Deployment specification
    config_manifest TEXT, -- JSON or YAML manifest
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, in_progress, success, failed, rolled_back
    
    -- Triggering information
    triggered_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Execution timeline
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Rollout progress
    desired_replicas INTEGER DEFAULT 1,
    ready_replicas INTEGER DEFAULT 0,
    updated_replicas INTEGER DEFAULT 0,
    
    -- Rollback information
    previous_deployment_id UUID REFERENCES deployments(id) ON DELETE SET NULL,
    
    -- Monitoring
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_deployments_company_id ON deployments(company_id);
CREATE INDEX IF NOT EXISTS idx_deployments_image_id ON deployments(image_id);
CREATE INDEX IF NOT EXISTS idx_deployments_environment_id ON deployments(deployment_environment_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at DESC);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_container_registries_updated_at
    BEFORE UPDATE ON container_registries
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_container_repositories_updated_at
    BEFORE UPDATE ON container_repositories
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_git_repositories_updated_at
    BEFORE UPDATE ON git_repositories
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_deployments_updated_at
    BEFORE UPDATE ON deployments
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_deployment_environments_updated_at
    BEFORE UPDATE ON deployment_environments
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
