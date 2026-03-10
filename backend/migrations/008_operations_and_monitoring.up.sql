-- Migration: 008_operations_and_monitoring.up.sql
-- Category: Operations & Monitoring
-- Purpose: Notification channels, templates, resource quotas, usage tracking, and environment access

-- ============================================================================
-- 1. NOTIFICATION CHANNELS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS notification_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Channel information
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Channel type
    channel_type VARCHAR(50) NOT NULL, -- email, slack, teams, webhook, pagerduty, opsgenie, sms
    
    -- Configuration
    config_json TEXT NOT NULL, -- JSON with provider-specific config
    -- For email: { "recipients": ["email@example.com"] }
    -- For slack: { "webhook_url": "https://...", "channel": "#alerts" }
    -- For webhook: { "url": "https://...", "headers": {...} }
    
    -- Authentication
    api_key_encrypted VARCHAR(500),
    api_secret_encrypted VARCHAR(500),
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, testing, disabled
    
    -- Verification
    last_verified_at TIMESTAMP WITH TIME ZONE,
    verification_status VARCHAR(50), -- unverified, verified, failed
    
    -- Scope
    available_for_alerts BOOLEAN DEFAULT true,
    available_for_notifications BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_channels_company_id ON notification_channels(company_id);
CREATE INDEX IF NOT EXISTS idx_notification_channels_type ON notification_channels(channel_type);
CREATE INDEX IF NOT EXISTS idx_notification_channels_status ON notification_channels(status);

-- ============================================================================
-- 2. NOTIFICATION TEMPLATES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Template information
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Template type
    template_type VARCHAR(50) NOT NULL, -- build_started, build_failed, build_succeeded, deployment_started, deployment_failed, security_alert, compliance_alert
    
    -- Content
    subject_template VARCHAR(500),
    body_template TEXT, -- Can contain variables like {{build_id}}, {{error_message}}, etc.
    html_template TEXT,
    
    -- Formatting
    template_variables TEXT, -- JSON array of variable names and descriptions
    
    -- Usage
    is_default BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    
    -- Customization
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS idx_notification_templates_company_id ON notification_templates(company_id);
CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates(template_type);

-- ============================================================================
-- 3. RESOURCE QUOTAS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS resource_quotas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    org_unit_id UUID REFERENCES org_units(id) ON DELETE CASCADE,
    
    -- Quota scope (company-wide or org unit specific)
    scope VARCHAR(50) NOT NULL DEFAULT 'company', -- company, org_unit
    
    -- Compute quotas
    cpu_cores_limit DECIMAL(10,2),
    memory_gb_limit DECIMAL(10,2),
    
    -- Storage quotas
    storage_gb_limit DECIMAL(10,2),
    artifact_storage_gb_limit DECIMAL(10,2),
    
    -- Build quotas
    concurrent_builds_limit INTEGER,
    monthly_builds_limit INTEGER,
    
    -- Image and registry quotas
    images_per_repository_limit INTEGER,
    repositories_limit INTEGER,
    
    -- API quotas
    api_calls_per_minute_limit INTEGER,
    api_calls_per_month_limit INTEGER,
    
    -- Deployment quotas
    concurrent_deployments_limit INTEGER,
    
    -- Enforcement
    enforce_hard_limit BOOLEAN DEFAULT true, -- If true, reject requests over quota
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, suspended, deleted
    
    -- Dates
    effective_from DATE,
    effective_until DATE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_resource_quotas_company_id ON resource_quotas(company_id);
CREATE INDEX IF NOT EXISTS idx_resource_quotas_org_unit_id ON resource_quotas(org_unit_id);
CREATE INDEX IF NOT EXISTS idx_resource_quotas_scope ON resource_quotas(scope);

-- ============================================================================
-- 4. USAGE TRACKING TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS usage_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    org_unit_id UUID REFERENCES org_units(id) ON DELETE CASCADE,
    
    -- Time period
    usage_date DATE NOT NULL,
    usage_month VARCHAR(7) NOT NULL, -- YYYY-MM format
    
    -- Resource usage
    cpu_core_hours DECIMAL(10,2) DEFAULT 0,
    memory_gb_hours DECIMAL(10,2) DEFAULT 0,
    storage_gb_days DECIMAL(10,2) DEFAULT 0,
    
    -- Build metrics
    builds_executed INTEGER DEFAULT 0,
    builds_succeeded INTEGER DEFAULT 0,
    builds_failed INTEGER DEFAULT 0,
    total_build_minutes INTEGER DEFAULT 0,
    
    -- Deployment metrics
    deployments_executed INTEGER DEFAULT 0,
    deployments_succeeded INTEGER DEFAULT 0,
    deployments_failed INTEGER DEFAULT 0,
    
    -- Image metrics
    images_created INTEGER DEFAULT 0,
    images_total_size_gb DECIMAL(10,2) DEFAULT 0,
    
    -- API metrics
    api_calls_count BIGINT DEFAULT 0,
    api_errors_count BIGINT DEFAULT 0,
    
    -- Data transfer
    data_transferred_gb DECIMAL(10,2) DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, usage_date),
    UNIQUE(company_id, usage_month)
);

CREATE INDEX IF NOT EXISTS idx_usage_tracking_company_id ON usage_tracking(company_id);
CREATE INDEX IF NOT EXISTS idx_usage_tracking_org_unit_id ON usage_tracking(org_unit_id);
CREATE INDEX IF NOT EXISTS idx_usage_tracking_date ON usage_tracking(usage_date DESC);

-- ============================================================================
-- 5. ENVIRONMENT ACCESS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS environment_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deployment_environment_id UUID NOT NULL REFERENCES deployment_environments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Access level
    access_level VARCHAR(50) NOT NULL DEFAULT 'viewer', -- owner, admin, deployer, viewer
    
    -- Restriction
    ip_whitelist TEXT, -- JSON array of IP addresses/CIDR ranges
    time_restriction VARCHAR(100), -- e.g., 'business_hours', 'utc_09_to_17'
    
    -- Approval requirement
    requires_approval_for_deployment BOOLEAN DEFAULT false,
    
    -- Access validity
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    access_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(deployment_environment_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_environment_access_environment_id ON environment_access(deployment_environment_id);
CREATE INDEX IF NOT EXISTS idx_environment_access_user_id ON environment_access(user_id);
CREATE INDEX IF NOT EXISTS idx_environment_access_expires_at ON environment_access(expires_at);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_notification_channels_updated_at
    BEFORE UPDATE ON notification_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_notification_templates_updated_at
    BEFORE UPDATE ON notification_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_resource_quotas_updated_at
    BEFORE UPDATE ON resource_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_environment_access_updated_at
    BEFORE UPDATE ON environment_access
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
