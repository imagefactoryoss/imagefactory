-- ============================================================================
-- Infrastructure Providers Seed Data
-- ============================================================================
-- This file seeds sample infrastructure providers for testing
-- Run this after the infrastructure_providers migration (048)
-- ============================================================================

-- Get the first tenant ID for seeding
DO $$
DECLARE
    first_tenant_id UUID;
BEGIN
    SELECT id INTO first_tenant_id FROM tenants LIMIT 1;

    -- Seed Kubernetes provider
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
        first_tenant_id,
        false,
        'kubernetes',
        'dev-k8s-cluster',
        'Development Kubernetes Cluster',
        '{
            "namespace": "build-system",
            "system_namespace": "imagefactory-system",
            "runtime_auth": {
                "auth_method": "token",
                "apiServer": "https://dev-k8s.example.com:6443",
                "endpoint": "https://dev-cluster.example.com",
                "token": "eyJhbGciOiJSUzI1NiIsImtpZCI6..."
            },
            "bootstrap_auth": {
                "auth_method": "token",
                "apiServer": "https://dev-k8s.example.com:6443",
                "endpoint": "https://dev-cluster.example.com",
                "token": "eyJhbGciOiJSUzI1NiIsImtpZCI6..."
            }
        }'::jsonb,
        'online',
        '["gpu", "high-memory", "arm64"]'::jsonb,
        (SELECT id FROM users WHERE email = 'admin@imagefactory.local' LIMIT 1)
    ) ON CONFLICT DO NOTHING;

    -- Seed Build Nodes provider
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
        first_tenant_id,
        false,
        'build_nodes',
        'dev-build-nodes',
        'Development Build Nodes',
        '{
            "host": "build-node-01.dev.example.com",
            "port": 22,
            "username": "builduser",
            "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
            "workDir": "/opt/builds"
        }'::jsonb,
        'online',
        '["docker", "high-cpu", "large-disk"]'::jsonb,
        (SELECT id FROM users WHERE email = 'admin@imagefactory.local' LIMIT 1)
    ) ON CONFLICT DO NOTHING;

    -- Grant configure permissions to admin tenant for all providers
    INSERT INTO provider_permissions (
        provider_id,
        tenant_id,
        permission,
        granted_by
    )
    SELECT
        ip.id,
        t.id,
        'infrastructure:configure',
        u.id
    FROM infrastructure_providers ip
    CROSS JOIN tenants t
    CROSS JOIN users u
    WHERE u.email = 'admin@imagefactory.local'
    ON CONFLICT DO NOTHING;

    -- Grant selection permissions to all tenants for online providers
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
        u.id
    FROM infrastructure_providers ip
    CROSS JOIN tenants t
    CROSS JOIN users u
    WHERE ip.status = 'online' AND u.email = 'admin@imagefactory.local'
    ON CONFLICT DO NOTHING;

END $$;
