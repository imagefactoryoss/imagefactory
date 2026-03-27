-- Migration: 082_packer_target_profiles.up.sql
-- Purpose: Admin-managed Packer target profiles with deterministic validation status

CREATE TABLE IF NOT EXISTS packer_target_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    is_global BOOLEAN NOT NULL DEFAULT false,
    name VARCHAR(120) NOT NULL,
    provider VARCHAR(32) NOT NULL CHECK (provider IN ('vmware', 'aws', 'azure', 'gcp')),
    description TEXT,
    secret_ref VARCHAR(255) NOT NULL,
    options JSONB NOT NULL DEFAULT '{}'::jsonb,
    validation_status VARCHAR(16) NOT NULL DEFAULT 'untested' CHECK (validation_status IN ('untested', 'valid', 'invalid')),
    last_validated_at TIMESTAMP WITH TIME ZONE,
    last_validation_message TEXT,
    last_remediation_hints JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_packer_target_profiles_tenant_provider
    ON packer_target_profiles(tenant_id, provider);

CREATE INDEX IF NOT EXISTS idx_packer_target_profiles_validation_status
    ON packer_target_profiles(validation_status);

CREATE INDEX IF NOT EXISTS idx_packer_target_profiles_is_global
    ON packer_target_profiles(is_global);
