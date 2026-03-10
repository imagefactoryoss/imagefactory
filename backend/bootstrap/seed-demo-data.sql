-- ============================================================================
-- Demo/Sample Data - OPTIONAL FOR DEVELOPMENT/TESTING
-- ============================================================================
-- This file contains demo and sample data for development and testing.
-- Run this AFTER seed-essential-data.sql to populate sample data.
--
-- Contains:
-- - External tenants (20 sample tenants)
-- - Additional deployment environments (staging, production)
-- - System configurations and build policies
-- - Sample infrastructure nodes
-- ============================================================================

-- NOTE:
-- Demo seed intentionally does NOT provision users.
-- User provisioning is handled by bootstrap/setup and runtime auth flows.

-- ============================================================================
-- SEED ADDITIONAL DEPLOYMENT ENVIRONMENTS (demo data)
-- ============================================================================
INSERT INTO deployment_environments (company_id, name, display_name, environment_type, infrastructure_provider, requires_approval, status)
SELECT c.id, 'staging', 'Staging', 'staging', 'kubernetes', true, 'active' FROM companies c WHERE c.name = 'ImageFactory'
ON CONFLICT (company_id, name) DO NOTHING;

INSERT INTO deployment_environments (company_id, name, display_name, environment_type, infrastructure_provider, requires_approval, status)
SELECT c.id, 'production', 'Production', 'production', 'kubernetes', true, 'active' FROM companies c WHERE c.name = 'ImageFactory'
ON CONFLICT (company_id, name) DO NOTHING;

-- System administrator tenant/group membership is seeded by:
--   backend/cmd/essential-config-seeder (ensureSystemBootstrap)

-- ============================================================================
-- SEED EXTERNAL TENANTS (demo data)
-- ============================================================================
INSERT INTO external_tenants (tenant_id, name, slug, description, contact_email, industry, country) VALUES
('10001', 'Engineering Team', 'engineering-team', 'Core engineering and development team', 'eng@imagefactory.local', 'Technology', 'US'),
('10002', 'DevOps Team', 'devops-team', 'Infrastructure and deployment operations', 'devops@imagefactory.local', 'Technology', 'US'),
('10003', 'Product Team', 'product-team', 'Product management and strategy', 'product@imagefactory.local', 'Technology', 'US'),
('10004', 'QA & Testing Team', 'qa-testing-team', 'Quality assurance and testing', 'qa@imagefactory.local', 'Technology', 'US'),
('10005', 'Frontend Team', 'frontend-team', 'User interface and frontend development', 'frontend@imagefactory.local', 'Technology', 'US'),
('10006', 'Backend Team', 'backend-team', 'Backend services and APIs', 'backend@imagefactory.local', 'Technology', 'US'),
('10007', 'Data Science Team', 'data-science-team', 'Analytics and machine learning', 'datasci@imagefactory.local', 'Technology', 'US'),
('10008', 'Security Team', 'security-team', 'Information security and compliance', 'security@imagefactory.local', 'Technology', 'US'),
('10009', 'Platform Team', 'platform-team', 'Internal platform and tools', 'platform@imagefactory.local', 'Technology', 'US'),
('10010', 'Finance Team', 'finance-team', 'Financial planning and operations', 'finance@imagefactory.local', 'Finance', 'US'),
('10011', 'Human Resources', 'human-resources', 'HR and talent management', 'hr@imagefactory.local', 'Administration', 'US'),
('10012', 'Sales Team', 'sales-team', 'Enterprise sales and business development', 'sales@imagefactory.local', 'Sales', 'US'),
('10013', 'Marketing Team', 'marketing-team', 'Marketing and communications', 'marketing@imagefactory.local', 'Marketing', 'US'),
('10014', 'Customer Success', 'customer-success', 'Customer support and success', 'support@imagefactory.local', 'Support', 'US'),
('10015', 'Legal Team', 'legal-team', 'Legal and compliance', 'legal@imagefactory.local', 'Legal', 'US'),
('10016', 'Infrastructure Team', 'infrastructure-team', 'Cloud and infrastructure management', 'infra@imagefactory.local', 'Technology', 'US'),
('10017', 'Mobile Team', 'mobile-team', 'Mobile app development', 'mobile@imagefactory.local', 'Technology', 'US'),
('10018', 'API Team', 'api-team', 'API design and integration', 'api@imagefactory.local', 'Technology', 'US'),
('10019', 'Database Team', 'database-team', 'Database administration and optimization', 'database@imagefactory.local', 'Technology', 'US'),
('10020', 'Research Team', 'research-team', 'Research and innovation', 'research@imagefactory.local', 'Technology', 'US')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED SYSTEM CONFIGS (demo data)
-- ============================================================================
INSERT INTO system_configs (
  tenant_id,
  config_type,
  config_key,
  config_value,
  status,
  description,
  created_by,
  updated_by
)
SELECT
  NULL,
  'tool_settings',
  'tool_availability',
  '{"build_methods":{"container":true,"packer":true,"paketo":true,"kaniko":true,"buildx":true,"nix":true},"sbom_tools":{"syft":true,"grype":true,"trivy":true},"scan_tools":{"trivy":true,"clair":false,"grype":true,"snyk":false},"registry_types":{"s3":true,"harbor":false,"quay":false,"artifactory":false},"secret_managers":{"vault":false,"aws_secretsmanager":true,"azure_keyvault":false,"gcp_secretmanager":false}}',
  'active',
  'Global tool availability configuration',
  u.id,
  u.id
FROM users u
WHERE u.email = 'admin@imagefactory.local'
ON CONFLICT DO NOTHING;

INSERT INTO system_configs (
  tenant_id,
  config_type,
  config_key,
  config_value,
  status,
  description,
  created_by,
  updated_by
)
SELECT
  NULL,
  'build',
  'build',
  '{"default_timeout_minutes":30,"max_concurrent_jobs":10,"worker_pool_size":5,"max_queue_size":100,"artifact_retention_days":30,"tekton_enabled":true}',
  'active',
  'Global build configuration',
  u.id,
  u.id
FROM users u
WHERE u.email = 'admin@imagefactory.local'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED ADDITIONAL PERMISSIONS (demo data - extended permissions)
-- ============================================================================
INSERT INTO permissions (resource, action, description, category, is_system_permission, created_at, updated_at) VALUES
-- Additional system permissions
('system', 'view_status', 'View system status', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'run_maintenance', 'Run system maintenance', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'manage_certificates', 'Manage SSL certificates', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'view_logs', 'View system logs', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional tenant permissions
('tenant', 'delete', 'Delete tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'suspend', 'Suspend tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'activate', 'Activate tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional user permissions
('user', 'delete', 'Delete user', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'suspend', 'Suspend user account', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'activate', 'Activate user account', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'reset_password', 'Reset user password', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'manage_roles', 'Manage user roles', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional role permissions
('role', 'create', 'Create new role', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'update', 'Update role settings', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'delete', 'Delete role', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'manage_permissions', 'Manage role permissions', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'assign_users', 'Assign users to roles', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional build permissions
('build', 'retry', 'Retry failed build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'delete', 'Delete build', 'build_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'manage_triggers', 'Manage build triggers', 'build_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional image permissions
('image', 'push', 'Push images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'pull', 'Pull images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'update', 'Update images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'delete', 'Delete images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'scan', 'Scan images for vulnerabilities', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'sign', 'Sign images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional registry permissions
('registry', 'create', 'Create registry', 'registry_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'update', 'Update registry settings', 'registry_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'delete', 'Delete registry', 'registry_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional deployment permissions
('deployment', 'update', 'Update deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'delete', 'Delete deployments', 'deployment_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'rollback', 'Rollback deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Additional project permissions
('projects', 'update', 'Update project settings', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'delete', 'Delete project', 'project_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'archive', 'Archive project', 'project_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'add_member', 'Add project member', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'remove_member', 'Remove project member', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'update_member_role', 'Update member role', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (resource, action) DO NOTHING;

-- ============================================================================
-- SEED SYSTEM CONFIGURATIONS (demo data)
-- ============================================================================
-- Insert default security configurations for all existing tenants
INSERT INTO system_configs (tenant_id, config_type, config_key, config_value, description, is_default, created_by, updated_by)
SELECT
    t.id as tenant_id,
    'security' as config_type,
    config_key,
    config_value::jsonb,
    defaults.description,
    true as is_default,
    (SELECT u.id FROM users u JOIN user_role_assignments ura ON u.id = ura.user_id WHERE ura.tenant_id = t.id LIMIT 1) as created_by,
    (SELECT u.id FROM users u JOIN user_role_assignments ura ON u.id = ura.user_id WHERE ura.tenant_id = t.id LIMIT 1) as updated_by
FROM tenants t
CROSS JOIN (
    VALUES
        ('jwt_expiration_hours', '24', 'JWT access token expiration in hours'),
        ('refresh_token_hours', '168', 'Refresh token expiration in hours (7 days)'),
        ('max_login_attempts', '5', 'Maximum failed login attempts before lockout'),
        ('account_lock_duration_minutes', '30', 'Account lock duration in minutes'),
        ('password_min_length', '8', 'Minimum password length'),
        ('require_special_chars', 'true', 'Require special characters in passwords'),
        ('require_numbers', 'true', 'Require numbers in passwords'),
        ('require_uppercase', 'true', 'Require uppercase letters in passwords'),
        ('session_timeout_minutes', '60', 'Session timeout in minutes')
) AS defaults(config_key, config_value, description)
WHERE EXISTS (SELECT 1 FROM user_role_assignments ura WHERE ura.tenant_id = t.id)
ON CONFLICT (tenant_id, config_type, config_key) DO NOTHING;

-- Insert default build configurations for all existing tenants
INSERT INTO system_configs (tenant_id, config_type, config_key, config_value, description, is_default, created_by, updated_by)
SELECT
    t.id as tenant_id,
    'build' as config_type,
    config_key,
    config_value::jsonb,
    defaults.description,
    true as is_default,
    (SELECT u.id FROM users u JOIN user_role_assignments ura ON u.id = ura.user_id WHERE ura.tenant_id = t.id LIMIT 1) as created_by,
    (SELECT u.id FROM users u JOIN user_role_assignments ura ON u.id = ura.user_id WHERE ura.tenant_id = t.id LIMIT 1) as updated_by
FROM tenants t
CROSS JOIN (
    VALUES
        ('default_timeout_minutes', '30', 'Default build timeout in minutes'),
        ('max_concurrent_jobs', '10', 'Maximum concurrent build jobs'),
        ('worker_pool_size', '5', 'Worker pool size for builds'),
        ('max_queue_size', '100', 'Maximum build queue size'),
        ('artifact_retention_days', '30', 'Build artifact retention in days')
) AS defaults(config_key, config_value, description)
WHERE EXISTS (SELECT 1 FROM user_role_assignments ura WHERE ura.tenant_id = t.id)
ON CONFLICT (tenant_id, config_type, config_key) DO NOTHING;

-- ============================================================================
-- SEED BUILD POLICIES (demo data)
-- ============================================================================
INSERT INTO build_policies (tenant_id, policy_type, policy_key, policy_value, description, is_active)
SELECT
    (SELECT id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1) AS tenant_id,
    v.policy_type,
    v.policy_key,
    v.policy_value::jsonb,
    v.description,
    v.is_active
FROM (
    VALUES
-- Resource Limits
    ('resource_limit', 'max_build_duration', '{"value": 2, "unit": "hours"}', 'Maximum duration a single build can run', true),
    ('resource_limit', 'concurrent_builds_per_tenant', '{"value": 5}', 'Maximum simultaneous builds allowed per tenant', true),
    ('resource_limit', 'storage_quota_per_build', '{"value": 10, "unit": "GB"}', 'Maximum disk space a build can use', true),
-- Scheduling Rules
    ('scheduling_rule', 'maintenance_windows', '{"schedule": "weekends 2-4 AM", "timezone": "UTC"}', 'Scheduled maintenance periods when builds are paused', true),
    ('scheduling_rule', 'priority_queuing', '{"algorithm": "priority-based", "levels": ["low", "normal", "high", "urgent"]}', 'How builds are prioritized in the queue', true),
-- Approval Workflows
    ('approval_workflow', 'approval_required', '{"enabled": false, "conditions": ["production_deployment", "privileged_access"]}', 'Whether builds require approval before execution', true),
    ('approval_workflow', 'auto_approval_threshold', '{"max_duration": "1h", "max_resources": "medium"}', 'Criteria for automatic approval of builds', true)
) AS v(policy_type, policy_key, policy_value, description, is_active)
ON CONFLICT (tenant_id, policy_key) DO NOTHING;

-- ============================================================================
-- SEED INFRASTRUCTURE PROVIDERS (demo data)
-- ============================================================================
INSERT INTO infrastructure_providers (
    tenant_id,
    is_global,
    provider_type,
    name,
    display_name,
    config,
    status,
    capabilities,
    created_by
) VALUES (
    (SELECT id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1), -- System admin tenant
    true,
    'kubernetes',
    'rancher-desktop',
    'Rancher Desktop Kubernetes',
    '{
        "namespace": "image_factory",
        "system_namespace": "imagefactory-system",
        "runtime_auth": {
          "auth_method": "token",
          "apiServer": "https://localhost:6443",
          "token": "<redacted-service-account-token>"
        },
        "bootstrap_auth": {
          "auth_method": "token",
          "apiServer": "https://localhost:6443",
          "token": "<redacted-service-account-token>"
        }
    }'::jsonb,
    'online',
    '["gpu", "high-memory", "arm64", "x86_64"]'::jsonb,
    (SELECT id FROM users WHERE email = 'admin@imagefactory.local' LIMIT 1)
) ON CONFLICT DO NOTHING;

-- Grant permissions for the rancher-desktop provider to all tenants
INSERT INTO provider_permissions (
    provider_id,
    tenant_id,
    permission,
    granted_by
)
SELECT
    ip.id,
    t.id,
    'infrastructure:select',
    (SELECT id FROM users WHERE email = 'admin@imagefactory.local' LIMIT 1)
FROM infrastructure_providers ip
CROSS JOIN tenants t
WHERE ip.name = 'rancher-desktop' AND ip.status = 'online'
ON CONFLICT DO NOTHING;

-- Grant configure permission for the rancher-desktop provider to system tenant
INSERT INTO provider_permissions (
    provider_id,
    tenant_id,
    permission,
    granted_by
)
SELECT
    ip.id,
    (SELECT id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1),
    'infrastructure:configure',
    (SELECT id FROM users WHERE email = 'admin@imagefactory.local' LIMIT 1)
FROM infrastructure_providers ip
WHERE ip.name = 'rancher-desktop'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED INFRASTRUCTURE NODES (demo data)
-- ============================================================================
INSERT INTO infrastructure_nodes (id, tenant_id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, last_heartbeat, maintenance_mode, labels) VALUES
('550e8400-e29b-41d4-a716-446655440001', (SELECT id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1), 'build-node-01', 'ready', 8, 16, 100, NOW(), false, '{"type": "build", "region": "us-west", "environment": "development"}'),
('550e8400-e29b-41d4-a716-446655440002', (SELECT id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1), 'build-node-02', 'ready', 4, 8, 50, NOW(), false, '{"type": "build", "region": "us-east", "environment": "development"}'),
('550e8400-e29b-41d4-a716-446655440003', (SELECT id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1), 'build-node-03', 'maintenance', 16, 32, 200, NOW() - interval '1 hour', true, '{"type": "build", "region": "eu-west", "environment": "development"}')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED CATALOG IMAGE DEMO DATA (demo data)
-- ============================================================================
DO $$
DECLARE
    v_admin_user_id UUID;
    v_tenant_id UUID;
    v_company_id UUID;
    v_img_nginx UUID;
    v_img_node UUID;
    v_img_python UUID;
    v_sbom_nginx UUID;
    v_sbom_node UUID;
    v_sbom_python UUID;
BEGIN
    SELECT id INTO v_admin_user_id FROM users WHERE email = 'admin@imagefactory.local' LIMIT 1;
    SELECT id INTO v_tenant_id FROM tenants WHERE tenant_code = 'sysadmin' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM tenants ORDER BY created_at ASC LIMIT 1;
    END IF;
    SELECT id INTO v_company_id FROM companies WHERE name = 'ImageFactory' LIMIT 1;

    IF v_admin_user_id IS NULL OR v_tenant_id IS NULL THEN
        RAISE NOTICE 'Skipping image catalog demo seed: missing admin user or tenant';
        RETURN;
    END IF;

    -- ------------------------------------------------------------------------
    -- Catalog images
    -- ------------------------------------------------------------------------
    INSERT INTO catalog_images (
        tenant_id, name, description, visibility, status, repository_url, registry_provider,
        architecture, os, language, framework, version, tags, metadata, size_bytes, pull_count,
        created_by, updated_by
    ) VALUES (
        v_tenant_id,
        'nginx-runtime',
        'Hardened NGINX runtime image for production reverse-proxy workloads.',
        'public',
        'published',
        'registry.example.com/platform/nginx-runtime',
        'harbor',
        'amd64',
        'linux',
        'c',
        'nginx',
        '1.25.3',
        '["nginx","runtime","alpine","prod-ready"]'::jsonb,
        '{"maintainer":"platform-team","lifecycle":"stable","compliance":["cis","soc2"]}'::jsonb,
        76349440,
        15420,
        v_admin_user_id,
        v_admin_user_id
    )
    ON CONFLICT (tenant_id, name) DO UPDATE SET
        description = EXCLUDED.description,
        visibility = EXCLUDED.visibility,
        status = EXCLUDED.status,
        repository_url = EXCLUDED.repository_url,
        registry_provider = EXCLUDED.registry_provider,
        architecture = EXCLUDED.architecture,
        os = EXCLUDED.os,
        language = EXCLUDED.language,
        framework = EXCLUDED.framework,
        version = EXCLUDED.version,
        tags = EXCLUDED.tags,
        metadata = EXCLUDED.metadata,
        size_bytes = EXCLUDED.size_bytes,
        pull_count = EXCLUDED.pull_count,
        updated_by = EXCLUDED.updated_by,
        updated_at = CURRENT_TIMESTAMP
    RETURNING id INTO v_img_nginx;

    INSERT INTO catalog_images (
        tenant_id, name, description, visibility, status, repository_url, registry_provider,
        architecture, os, language, framework, version, tags, metadata, size_bytes, pull_count,
        created_by, updated_by
    ) VALUES (
        v_tenant_id,
        'nodejs-builder',
        'Node.js build image with npm/pnpm toolchain for CI pipelines.',
        'tenant',
        'published',
        'registry.example.com/platform/nodejs-builder',
        'harbor',
        'amd64',
        'linux',
        'javascript',
        'nodejs',
        '20.11.1',
        '["nodejs","builder","ci","npm","pnpm"]'::jsonb,
        '{"maintainer":"devx-team","lifecycle":"stable","supports":["npm","pnpm","yarn"]}'::jsonb,
        214958080,
        8920,
        v_admin_user_id,
        v_admin_user_id
    )
    ON CONFLICT (tenant_id, name) DO UPDATE SET
        description = EXCLUDED.description,
        visibility = EXCLUDED.visibility,
        status = EXCLUDED.status,
        repository_url = EXCLUDED.repository_url,
        registry_provider = EXCLUDED.registry_provider,
        architecture = EXCLUDED.architecture,
        os = EXCLUDED.os,
        language = EXCLUDED.language,
        framework = EXCLUDED.framework,
        version = EXCLUDED.version,
        tags = EXCLUDED.tags,
        metadata = EXCLUDED.metadata,
        size_bytes = EXCLUDED.size_bytes,
        pull_count = EXCLUDED.pull_count,
        updated_by = EXCLUDED.updated_by,
        updated_at = CURRENT_TIMESTAMP
    RETURNING id INTO v_img_node;

    INSERT INTO catalog_images (
        tenant_id, name, description, visibility, status, repository_url, registry_provider,
        architecture, os, language, framework, version, tags, metadata, size_bytes, pull_count,
        created_by, updated_by
    ) VALUES (
        v_tenant_id,
        'python-ml-runtime',
        'Python runtime with common ML dependencies for inference services.',
        'tenant',
        'published',
        'registry.example.com/platform/python-ml-runtime',
        'harbor',
        'amd64',
        'linux',
        'python',
        'fastapi',
        '3.11.8',
        '["python","ml","runtime","fastapi","inference"]'::jsonb,
        '{"maintainer":"ml-platform","lifecycle":"active","cuda":"optional"}'::jsonb,
        498073600,
        4760,
        v_admin_user_id,
        v_admin_user_id
    )
    ON CONFLICT (tenant_id, name) DO UPDATE SET
        description = EXCLUDED.description,
        visibility = EXCLUDED.visibility,
        status = EXCLUDED.status,
        repository_url = EXCLUDED.repository_url,
        registry_provider = EXCLUDED.registry_provider,
        architecture = EXCLUDED.architecture,
        os = EXCLUDED.os,
        language = EXCLUDED.language,
        framework = EXCLUDED.framework,
        version = EXCLUDED.version,
        tags = EXCLUDED.tags,
        metadata = EXCLUDED.metadata,
        size_bytes = EXCLUDED.size_bytes,
        pull_count = EXCLUDED.pull_count,
        updated_by = EXCLUDED.updated_by,
        updated_at = CURRENT_TIMESTAMP
    RETURNING id INTO v_img_python;

    -- ------------------------------------------------------------------------
    -- Catalog versions
    -- ------------------------------------------------------------------------
    INSERT INTO catalog_image_versions (catalog_image_id, version, description, digest, size_bytes, created_by, manifest, config, layers)
    VALUES
        (v_img_nginx, '1.25.2', 'Previous stable nginx runtime', 'sha256:nginx1252demo', 74211328, v_admin_user_id, '{"schemaVersion":2}'::jsonb, '{"user":"nginx"}'::jsonb, '[{"digest":"sha256:ngbase"},{"digest":"sha256:nginxbin"}]'::jsonb),
        (v_img_nginx, '1.25.3', 'Current stable nginx runtime',  'sha256:nginx1253demo', 76349440, v_admin_user_id, '{"schemaVersion":2}'::jsonb, '{"user":"nginx"}'::jsonb, '[{"digest":"sha256:ngbase"},{"digest":"sha256:nginxbin"},{"digest":"sha256:nginxcfg"}]'::jsonb),
        (v_img_node,  '20.10.0', 'Node builder previous',       'sha256:node2010demo', 205520896, v_admin_user_id, '{"schemaVersion":2}'::jsonb, '{"user":"node"}'::jsonb,  '[{"digest":"sha256:nodebase"},{"digest":"sha256:nodetools"}]'::jsonb),
        (v_img_node,  '20.11.1', 'Node builder current',        'sha256:node2011demo', 214958080, v_admin_user_id, '{"schemaVersion":2}'::jsonb, '{"user":"node"}'::jsonb,  '[{"digest":"sha256:nodebase"},{"digest":"sha256:nodetools"},{"digest":"sha256:pnpm"}]'::jsonb),
        (v_img_python,'3.11.7',  'Python ML previous',          'sha256:py3117demo', 485490688, v_admin_user_id, '{"schemaVersion":2}'::jsonb, '{"user":"app"}'::jsonb,    '[{"digest":"sha256:pybase"},{"digest":"sha256:pypkgs"}]'::jsonb),
        (v_img_python,'3.11.8',  'Python ML current',           'sha256:py3118demo', 498073600, v_admin_user_id, '{"schemaVersion":2}'::jsonb, '{"user":"app"}'::jsonb,    '[{"digest":"sha256:pybase"},{"digest":"sha256:pypkgs"},{"digest":"sha256:fastapi"}]'::jsonb)
    ON CONFLICT (catalog_image_id, version) DO UPDATE SET
        description = EXCLUDED.description,
        digest = EXCLUDED.digest,
        size_bytes = EXCLUDED.size_bytes,
        created_by = EXCLUDED.created_by,
        manifest = EXCLUDED.manifest,
        config = EXCLUDED.config,
        layers = EXCLUDED.layers;

    -- ------------------------------------------------------------------------
    -- Catalog tags
    -- ------------------------------------------------------------------------
    INSERT INTO catalog_image_tags (catalog_image_id, tag, category, created_by)
    VALUES
        (v_img_nginx, 'production', 'system', v_admin_user_id),
        (v_img_nginx, 'web', 'system', v_admin_user_id),
        (v_img_node, 'ci', 'system', v_admin_user_id),
        (v_img_node, 'build', 'system', v_admin_user_id),
        (v_img_python, 'ml', 'system', v_admin_user_id),
        (v_img_python, 'inference', 'system', v_admin_user_id)
    ON CONFLICT (catalog_image_id, tag) DO UPDATE SET
        category = EXCLUDED.category,
        created_by = EXCLUDED.created_by;

    -- ------------------------------------------------------------------------
    -- Layer + metadata details
    -- ------------------------------------------------------------------------
    INSERT INTO catalog_image_layers (
        image_id, layer_number, layer_digest, layer_size_bytes, media_type,
        is_base_layer, base_image_name, base_image_tag, used_in_builds_count, last_used_in_build_at
    ) VALUES
        (v_img_nginx, 1, 'sha256:nginxlayerbase', 14548992, 'application/vnd.oci.image.layer.v1.tar+gzip', true, 'alpine', '3.19', 122, CURRENT_TIMESTAMP),
        (v_img_nginx, 2, 'sha256:nginxlayerbin',  45219840, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 122, CURRENT_TIMESTAMP),
        (v_img_nginx, 3, 'sha256:nginxlayercfg',  16580608, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 122, CURRENT_TIMESTAMP),
        (v_img_nginx, 4, 'sha256:nginxlayerssl',   6291456, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 122, CURRENT_TIMESTAMP),
        (v_img_nginx, 5, 'sha256:nginxlayermods',  7340032, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 122, CURRENT_TIMESTAMP),
        (v_img_node,  1, 'sha256:nodelayerbase',  18874368, 'application/vnd.oci.image.layer.v1.tar+gzip', true, 'debian', 'bookworm-slim', 91, CURRENT_TIMESTAMP),
        (v_img_node,  2, 'sha256:nodelayertools', 125829120, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 91, CURRENT_TIMESTAMP),
        (v_img_node,  3, 'sha256:nodelayerpnpm',  70254592, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 91, CURRENT_TIMESTAMP),
        (v_img_node,  4, 'sha256:nodelayernpm',   26214400, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 91, CURRENT_TIMESTAMP),
        (v_img_node,  5, 'sha256:nodelayerca',     8388608, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 91, CURRENT_TIMESTAMP),
        (v_img_python,1, 'sha256:pylayerbase',    25165824, 'application/vnd.oci.image.layer.v1.tar+gzip', true, 'python', '3.11-slim', 57, CURRENT_TIMESTAMP),
        (v_img_python,2, 'sha256:pylayerpkgs',    387973120, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 57, CURRENT_TIMESTAMP),
        (v_img_python,3, 'sha256:pylayerfastapi', 84934656, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 57, CURRENT_TIMESTAMP),
        (v_img_python,4, 'sha256:pylayernumpy',   50331648, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 57, CURRENT_TIMESTAMP),
        (v_img_python,5, 'sha256:pylayermodels',  75497472, 'application/vnd.oci.image.layer.v1.tar+gzip', false, NULL, NULL, 57, CURRENT_TIMESTAMP)
    ON CONFLICT (image_id, layer_number) DO UPDATE SET
        layer_digest = EXCLUDED.layer_digest,
        layer_size_bytes = EXCLUDED.layer_size_bytes,
        media_type = EXCLUDED.media_type,
        is_base_layer = EXCLUDED.is_base_layer,
        base_image_name = EXCLUDED.base_image_name,
        base_image_tag = EXCLUDED.base_image_tag,
        used_in_builds_count = EXCLUDED.used_in_builds_count,
        last_used_in_build_at = EXCLUDED.last_used_in_build_at;

    INSERT INTO catalog_image_metadata (
        image_id, docker_config_digest, docker_manifest_digest, total_layer_count,
        compressed_size_bytes, uncompressed_size_bytes, packages_count,
        vulnerabilities_high_count, vulnerabilities_medium_count, vulnerabilities_low_count,
        entrypoint, cmd, env_vars, working_dir, labels, last_scanned_at, scan_tool
    ) VALUES
        (v_img_nginx,  'sha256:cfgnginx', 'sha256:manifestnginx', 5, 89653248, 181403648, 96, 1, 3, 6, '["nginx"]', '["-g","daemon off;"]', '{"NGINX_ENV":"prod"}', '/etc/nginx', '{"org.opencontainers.image.source":"platform/nginx-runtime"}', CURRENT_TIMESTAMP, 'trivy'),
        (v_img_node,   'sha256:cfgnode',  'sha256:manifestnode',  5, 249561088, 471859200, 524, 2, 8, 18, '["node"]', '["npm","--version"]', '{"NODE_ENV":"ci"}', '/workspace', '{"org.opencontainers.image.source":"platform/nodejs-builder"}', CURRENT_TIMESTAMP, 'trivy'),
        (v_img_python, 'sha256:cfgpy',    'sha256:manifestpy',    5, 609222656, 1184890880, 784, 3, 10, 24, '["python"]', '["-m","uvicorn","app:app"]', '{"PYTHONUNBUFFERED":"1"}', '/app', '{"org.opencontainers.image.source":"platform/python-ml-runtime"}', CURRENT_TIMESTAMP, 'grype')
    ON CONFLICT (image_id) DO UPDATE SET
        docker_config_digest = EXCLUDED.docker_config_digest,
        docker_manifest_digest = EXCLUDED.docker_manifest_digest,
        total_layer_count = EXCLUDED.total_layer_count,
        compressed_size_bytes = EXCLUDED.compressed_size_bytes,
        uncompressed_size_bytes = EXCLUDED.uncompressed_size_bytes,
        packages_count = EXCLUDED.packages_count,
        vulnerabilities_high_count = EXCLUDED.vulnerabilities_high_count,
        vulnerabilities_medium_count = EXCLUDED.vulnerabilities_medium_count,
        vulnerabilities_low_count = EXCLUDED.vulnerabilities_low_count,
        entrypoint = EXCLUDED.entrypoint,
        cmd = EXCLUDED.cmd,
        env_vars = EXCLUDED.env_vars,
        working_dir = EXCLUDED.working_dir,
        labels = EXCLUDED.labels,
        last_scanned_at = EXCLUDED.last_scanned_at,
        scan_tool = EXCLUDED.scan_tool,
        updated_at = CURRENT_TIMESTAMP;

    -- ------------------------------------------------------------------------
    -- Vulnerability and SBOM details
    -- ------------------------------------------------------------------------
    INSERT INTO cve_database (cve_id, cve_description, cvss_v3_score, cvss_v3_vector, cvss_v3_severity, published_date, modified_date, cwe_id, "references", is_exploited_in_wild, exploit_count)
    VALUES
        ('CVE-2024-11111', 'OpenSSL buffer validation issue in specific TLS flows.', 8.1, 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N', 'HIGH', DATE '2024-01-12', DATE '2024-02-20', 'CWE-120', '["https://nvd.nist.gov/vuln/detail/CVE-2024-11111"]', false, 1),
        ('CVE-2024-22222', 'glibc iconv out-of-bounds read under malformed input.', 7.4, 'CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N', 'HIGH', DATE '2024-03-02', DATE '2024-04-09', 'CWE-125', '["https://nvd.nist.gov/vuln/detail/CVE-2024-22222"]', false, 0),
        ('CVE-2024-33333', 'urllib3 header parsing weakness allowing request smuggling edge cases.', 6.5, 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:N', 'MEDIUM', DATE '2024-05-18', DATE '2024-06-04', 'CWE-444', '["https://nvd.nist.gov/vuln/detail/CVE-2024-33333"]', false, 0),
        ('CVE-2024-44444', 'busybox ash command injection in crafted environment expansion.', 9.3, 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H', 'CRITICAL', DATE '2024-06-10', DATE '2024-06-21', 'CWE-78', '["https://nvd.nist.gov/vuln/detail/CVE-2024-44444"]', true, 2),
        ('CVE-2024-55555', 'npm tar package path traversal on malformed archive extraction.', 8.0, 'CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:N', 'HIGH', DATE '2024-07-01', DATE '2024-07-12', 'CWE-22', '["https://nvd.nist.gov/vuln/detail/CVE-2024-55555"]', false, 1),
        ('CVE-2024-66666', 'numpy integer overflow in ndarray shape parsing under untrusted input.', 7.1, 'CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N', 'HIGH', DATE '2024-08-02', DATE '2024-08-17', 'CWE-190', '["https://nvd.nist.gov/vuln/detail/CVE-2024-66666"]', false, 0),
        ('CVE-2024-77777', 'curl HTTP/2 stream reset handling allows request smuggling scenarios.', 7.8, 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L/A:N', 'HIGH', DATE '2024-09-03', DATE '2024-09-19', 'CWE-444', '["https://nvd.nist.gov/vuln/detail/CVE-2024-77777"]', false, 0),
        ('CVE-2024-88888', 'fastapi dependency pin mismatch enables unsafe deserialization path.', 8.6, 'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N', 'HIGH', DATE '2024-10-11', DATE '2024-10-27', 'CWE-502', '["https://nvd.nist.gov/vuln/detail/CVE-2024-88888"]', false, 1)
    ON CONFLICT (cve_id) DO UPDATE SET
        cve_description = EXCLUDED.cve_description,
        cvss_v3_score = EXCLUDED.cvss_v3_score,
        cvss_v3_vector = EXCLUDED.cvss_v3_vector,
        cvss_v3_severity = EXCLUDED.cvss_v3_severity,
        modified_date = EXCLUDED.modified_date,
        cwe_id = EXCLUDED.cwe_id,
        "references" = EXCLUDED."references",
        is_exploited_in_wild = EXCLUDED.is_exploited_in_wild,
        exploit_count = EXCLUDED.exploit_count,
        updated_at = CURRENT_TIMESTAMP;

    INSERT INTO package_vulnerabilities (package_name, package_type, package_version, cve_id, vulnerable_version_range, patched_version, source_database, discovered_at)
    VALUES
        ('openssl', 'apk', '3.0.12-r2', 'CVE-2024-11111', '<=3.0.12-r2', '3.0.13-r0', 'nvd', CURRENT_TIMESTAMP),
        ('glibc', 'deb', '2.36-9+deb12u3', 'CVE-2024-22222', '<=2.36-9+deb12u3', '2.36-9+deb12u5', 'nvd', CURRENT_TIMESTAMP),
        ('urllib3', 'pip', '2.1.0', 'CVE-2024-33333', '<2.2.2', '2.2.2', 'nvd', CURRENT_TIMESTAMP),
        ('busybox', 'apk', '1.36.1-r2', 'CVE-2024-44444', '<=1.36.1-r2', '1.36.1-r5', 'nvd', CURRENT_TIMESTAMP),
        ('npm', 'npm', '10.7.0', 'CVE-2024-55555', '<10.8.1', '10.8.1', 'nvd', CURRENT_TIMESTAMP),
        ('numpy', 'pip', '1.26.4', 'CVE-2024-66666', '<1.26.5', '1.26.5', 'nvd', CURRENT_TIMESTAMP),
        ('curl', 'deb', '8.5.0-2', 'CVE-2024-77777', '<=8.5.0-2', '8.6.0-1', 'nvd', CURRENT_TIMESTAMP),
        ('fastapi', 'pip', '0.110.0', 'CVE-2024-88888', '<0.110.2', '0.110.2', 'nvd', CURRENT_TIMESTAMP)
    ON CONFLICT (package_name, package_type, package_version, cve_id) DO UPDATE SET
        vulnerable_version_range = EXCLUDED.vulnerable_version_range,
        patched_version = EXCLUDED.patched_version,
        source_database = EXCLUDED.source_database,
        discovered_at = EXCLUDED.discovered_at;

    INSERT INTO catalog_image_sbom (
        image_id, sbom_format, sbom_version, sbom_content, generated_by_tool, tool_version,
        sbom_checksum, scan_timestamp, scan_duration_seconds, status
    ) VALUES
        (v_img_nginx, 'spdx', '2.3', '{"packages":[{"name":"nginx"},{"name":"openssl"}]}'::text, 'syft', '1.2.0', 'sha256:sbomnginx', CURRENT_TIMESTAMP, 18, 'valid'),
        (v_img_node, 'cyclonedx', '1.5', '{"components":[{"name":"node"},{"name":"npm"}]}'::text, 'syft', '1.2.0', 'sha256:sbomnode', CURRENT_TIMESTAMP, 25, 'valid'),
        (v_img_python, 'spdx', '2.3', '{"packages":[{"name":"python"},{"name":"urllib3"}]}'::text, 'syft', '1.2.0', 'sha256:sbompython', CURRENT_TIMESTAMP, 31, 'valid')
    ON CONFLICT (image_id) DO UPDATE SET
        sbom_format = EXCLUDED.sbom_format,
        sbom_version = EXCLUDED.sbom_version,
        sbom_content = EXCLUDED.sbom_content,
        generated_by_tool = EXCLUDED.generated_by_tool,
        tool_version = EXCLUDED.tool_version,
        sbom_checksum = EXCLUDED.sbom_checksum,
        scan_timestamp = EXCLUDED.scan_timestamp,
        scan_duration_seconds = EXCLUDED.scan_duration_seconds,
        status = EXCLUDED.status,
        updated_at = CURRENT_TIMESTAMP;

    SELECT id INTO v_sbom_nginx FROM catalog_image_sbom WHERE image_id = v_img_nginx;
    SELECT id INTO v_sbom_node FROM catalog_image_sbom WHERE image_id = v_img_node;
    SELECT id INTO v_sbom_python FROM catalog_image_sbom WHERE image_id = v_img_python;

    INSERT INTO sbom_packages (
        image_sbom_id, image_id, package_name, package_version, package_type, package_url,
        homepage_url, license_name, license_spdx_id, package_path, known_vulnerabilities_count, critical_vulnerabilities_count
    ) VALUES
        (v_sbom_nginx, v_img_nginx, 'openssl', '3.0.12-r2', 'library', 'pkg:apk/alpine/openssl@3.0.12-r2', 'https://www.openssl.org', 'Apache-2.0', 'Apache-2.0', '/lib', 1, 0),
        (v_sbom_nginx, v_img_nginx, 'busybox', '1.36.1-r2', 'os', 'pkg:apk/alpine/busybox@1.36.1-r2', 'https://busybox.net', 'GPL-2.0-only', 'GPL-2.0-only', '/bin', 1, 1),
        (v_sbom_nginx, v_img_nginx, 'zlib', '1.3.1-r0', 'library', 'pkg:apk/alpine/zlib@1.3.1-r0', 'https://zlib.net', 'Zlib', 'Zlib', '/lib', 0, 0),
        (v_sbom_node, v_img_node, 'glibc', '2.36-9+deb12u3', 'os', 'pkg:deb/debian/glibc@2.36-9+deb12u3', 'https://www.gnu.org/software/libc', 'LGPL-2.1-or-later', 'LGPL-2.1-or-later', '/usr/lib', 1, 0),
        (v_sbom_node, v_img_node, 'npm', '10.7.0', 'library', 'pkg:npm/npm@10.7.0', 'https://www.npmjs.com/package/npm', 'Artistic-2.0', 'Artistic-2.0', '/usr/local/lib/node_modules', 1, 0),
        (v_sbom_node, v_img_node, 'curl', '8.5.0-2', 'os', 'pkg:deb/debian/curl@8.5.0-2', 'https://curl.se', 'curl', 'curl', '/usr/bin', 1, 0),
        (v_sbom_python, v_img_python, 'urllib3', '2.1.0', 'library', 'pkg:pypi/urllib3@2.1.0', 'https://urllib3.readthedocs.io', 'MIT', 'MIT', '/usr/local/lib/python3.11/site-packages', 1, 0)
        ,(v_sbom_python, v_img_python, 'numpy', '1.26.4', 'library', 'pkg:pypi/numpy@1.26.4', 'https://numpy.org', 'BSD-3-Clause', 'BSD-3-Clause', '/usr/local/lib/python3.11/site-packages', 1, 0)
        ,(v_sbom_python, v_img_python, 'fastapi', '0.110.0', 'library', 'pkg:pypi/fastapi@0.110.0', 'https://fastapi.tiangolo.com', 'MIT', 'MIT', '/usr/local/lib/python3.11/site-packages', 1, 0)
    ON CONFLICT (image_sbom_id, package_name, package_version) DO UPDATE SET
        package_type = EXCLUDED.package_type,
        package_url = EXCLUDED.package_url,
        homepage_url = EXCLUDED.homepage_url,
        license_name = EXCLUDED.license_name,
        license_spdx_id = EXCLUDED.license_spdx_id,
        package_path = EXCLUDED.package_path,
        known_vulnerabilities_count = EXCLUDED.known_vulnerabilities_count,
        critical_vulnerabilities_count = EXCLUDED.critical_vulnerabilities_count;

    INSERT INTO catalog_image_vulnerability_scans (
        id, image_id, build_id, scan_tool, tool_version, scan_status,
        started_at, completed_at, duration_seconds,
        vulnerabilities_critical, vulnerabilities_high, vulnerabilities_medium,
        vulnerabilities_low, vulnerabilities_negligible, vulnerabilities_unknown,
        pass_fail_result, compliance_check_passed, scan_report_location, scan_report_json, error_message
    ) VALUES
        ('d9238d6f-7d8f-4424-8d8a-8a74b5551001', v_img_nginx, NULL, 'trivy', '0.52.0', 'completed', CURRENT_TIMESTAMP - INTERVAL '3 minutes', CURRENT_TIMESTAMP - INTERVAL '2 minutes', 62, 1, 2, 3, 6, 4, 0, 'WARNING', false, 's3://reports/nginx-runtime-trivy.json', '{"summary":{"critical":1,"high":2,"medium":3}}'::text, NULL),
        ('d9238d6f-7d8f-4424-8d8a-8a74b5551002', v_img_node, NULL, 'trivy', '0.52.0', 'completed', CURRENT_TIMESTAMP - INTERVAL '5 minutes', CURRENT_TIMESTAMP - INTERVAL '3 minutes', 121, 0, 3, 8, 18, 10, 0, 'WARNING', false, 's3://reports/nodejs-builder-trivy.json', '{"summary":{"high":3,"medium":8}}'::text, NULL),
        ('d9238d6f-7d8f-4424-8d8a-8a74b5551003', v_img_python, NULL, 'grype', '0.76.0', 'completed', CURRENT_TIMESTAMP - INTERVAL '6 minutes', CURRENT_TIMESTAMP - INTERVAL '4 minutes', 144, 0, 4, 10, 24, 11, 0, 'WARNING', false, 's3://reports/python-ml-runtime-grype.json', '{"summary":{"high":4,"medium":10}}'::text, NULL)
    ON CONFLICT (id) DO UPDATE SET
        image_id = EXCLUDED.image_id,
        scan_tool = EXCLUDED.scan_tool,
        tool_version = EXCLUDED.tool_version,
        scan_status = EXCLUDED.scan_status,
        started_at = EXCLUDED.started_at,
        completed_at = EXCLUDED.completed_at,
        duration_seconds = EXCLUDED.duration_seconds,
        vulnerabilities_critical = EXCLUDED.vulnerabilities_critical,
        vulnerabilities_high = EXCLUDED.vulnerabilities_high,
        vulnerabilities_medium = EXCLUDED.vulnerabilities_medium,
        vulnerabilities_low = EXCLUDED.vulnerabilities_low,
        vulnerabilities_negligible = EXCLUDED.vulnerabilities_negligible,
        vulnerabilities_unknown = EXCLUDED.vulnerabilities_unknown,
        pass_fail_result = EXCLUDED.pass_fail_result,
        compliance_check_passed = EXCLUDED.compliance_check_passed,
        scan_report_location = EXCLUDED.scan_report_location,
        scan_report_json = EXCLUDED.scan_report_json,
        error_message = EXCLUDED.error_message;

    IF v_company_id IS NOT NULL THEN
        INSERT INTO vulnerability_suppressions (
            id, company_id, cve_id, package_name, suppression_scope, project_id, image_id,
            reason, justification, suppressed_by_user_id, suppressed_at, expires_at,
            approved_by_user_id, approved_at, status
        ) VALUES (
            'd9238d6f-7d8f-4424-8d8a-8a74b5551004',
            v_company_id,
            'CVE-2024-33333',
            'urllib3',
            'image',
            NULL,
            v_img_python,
            'Temporary suppress until upstream patch is validated in staging.',
            'in_remediation',
            v_admin_user_id,
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP + INTERVAL '30 days',
            v_admin_user_id,
            CURRENT_TIMESTAMP,
            'active'
        )
        ON CONFLICT (id) DO UPDATE SET
            company_id = EXCLUDED.company_id,
            cve_id = EXCLUDED.cve_id,
            package_name = EXCLUDED.package_name,
            suppression_scope = EXCLUDED.suppression_scope,
            project_id = EXCLUDED.project_id,
            image_id = EXCLUDED.image_id,
            reason = EXCLUDED.reason,
            justification = EXCLUDED.justification,
            suppressed_by_user_id = EXCLUDED.suppressed_by_user_id,
            suppressed_at = EXCLUDED.suppressed_at,
            expires_at = EXCLUDED.expires_at,
            approved_by_user_id = EXCLUDED.approved_by_user_id,
            approved_at = EXCLUDED.approved_at,
            status = EXCLUDED.status;
    END IF;

    RAISE NOTICE 'Catalog image demo data seeded: %, %, %', v_img_nginx, v_img_node, v_img_python;
END $$;
