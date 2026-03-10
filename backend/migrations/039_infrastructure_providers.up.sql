-- Migration: 048_infrastructure_providers.up.sql
-- Category: Infrastructure Management
-- Purpose: Create unified infrastructure providers table for both K8s and build nodes

-- Create infrastructure_providers table
-- Unified table for managing all types of infrastructure providers
CREATE TABLE IF NOT EXISTS infrastructure_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    is_global BOOLEAN NOT NULL DEFAULT false,
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('kubernetes', 'aws-eks', 'gcp-gke', 'azure-aks', 'oci-oke', 'vmware-vks', 'openshift', 'rancher', 'build_nodes')),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}', -- Encrypted provider-specific configuration
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('online', 'offline', 'maintenance', 'pending')),
    capabilities JSONB DEFAULT '[]',
    last_health_check TIMESTAMP WITH TIME ZONE,
    health_status VARCHAR(50),
    readiness_status VARCHAR(50),
    readiness_last_checked TIMESTAMP WITH TIME ZONE,
    readiness_missing_prereqs JSONB NOT NULL DEFAULT '[]'::jsonb,
    bootstrap_mode VARCHAR(64) NOT NULL DEFAULT 'image_factory_managed' CHECK (bootstrap_mode IN ('image_factory_managed', 'self_managed')),
    credential_scope VARCHAR(64) NOT NULL DEFAULT 'unknown' CHECK (credential_scope IN ('cluster_admin', 'namespace_admin', 'read_only', 'unknown')),
    target_namespace VARCHAR(255),
    is_schedulable BOOLEAN NOT NULL DEFAULT false,
    schedulable_reason TEXT,
    blocked_by JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create provider_permissions table
-- Manages which tenants have access to which providers
CREATE TABLE IF NOT EXISTS provider_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES infrastructure_providers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(provider_id, tenant_id, permission)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_providers_tenant_type ON infrastructure_providers(tenant_id, provider_type);
CREATE INDEX IF NOT EXISTS idx_infrastructure_providers_is_global ON infrastructure_providers(is_global);
CREATE INDEX IF NOT EXISTS idx_infrastructure_providers_readiness_last_checked ON infrastructure_providers (readiness_last_checked DESC);
CREATE INDEX IF NOT EXISTS idx_provider_permissions_tenant ON provider_permissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_provider_permissions_provider ON provider_permissions(provider_id);
