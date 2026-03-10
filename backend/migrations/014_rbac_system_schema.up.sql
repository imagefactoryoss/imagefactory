-- Migration: 013_rbac_system_schema.up.sql
-- Purpose: Add RBAC system support to roles table and create necessary RBAC tables

-- ============================================================================
-- 1. ALTER ROLES TABLE TO SUPPORT RBAC SYSTEM
-- ============================================================================

-- Add missing columns to roles table if they don't exist
ALTER TABLE roles 
ADD COLUMN IF NOT EXISTS tenant_id UUID,
ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1;

-- Drop the old company_id constraint and update it to use tenant_id
-- Note: This is a migration to align with RBAC domain model
UPDATE roles SET tenant_id = company_id WHERE tenant_id IS NULL AND company_id IS NOT NULL;

-- If there are system roles (is_system_role = true), mark them appropriately
UPDATE roles SET is_system = true WHERE is_system_role = true AND is_system = false;

-- Create RBAC roles table for system-wide roles (tenant_id is null for system roles)
CREATE TABLE IF NOT EXISTS rbac_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1,
    
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_rbac_roles_tenant_id ON rbac_roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rbac_roles_is_system ON rbac_roles(is_system);
CREATE INDEX IF NOT EXISTS idx_rbac_roles_name ON rbac_roles(name);

-- Create trigger for rbac_roles updated_at
CREATE OR REPLACE FUNCTION update_rbac_roles_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_rbac_roles_updated_at ON rbac_roles;
CREATE TRIGGER update_rbac_roles_updated_at
    BEFORE UPDATE ON rbac_roles
    FOR EACH ROW
    EXECUTE FUNCTION update_rbac_roles_timestamp();

-- ============================================================================
-- 3. USER INVITATIONS TABLE
-- ============================================================================
-- Stores user invitations for tenant onboarding and role assignment
CREATE TABLE IF NOT EXISTS user_invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id UUID REFERENCES rbac_roles(id) ON DELETE SET NULL,
    invite_token VARCHAR(255) NOT NULL UNIQUE,
    invited_by_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'pending', -- pending, accepted, rejected, expired
    accepted_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    message TEXT, -- Optional personal message from inviter
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email, tenant_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_invitations_email ON user_invitations(email);
CREATE INDEX IF NOT EXISTS idx_user_invitations_tenant_id ON user_invitations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_invitations_status ON user_invitations(status);
CREATE INDEX IF NOT EXISTS idx_user_invitations_invite_token ON user_invitations(invite_token);

-- Create trigger for user_invitations updated_at
CREATE OR REPLACE FUNCTION update_user_invitations_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_user_invitations_updated_at ON user_invitations;
CREATE TRIGGER update_user_invitations_updated_at
    BEFORE UPDATE ON user_invitations
    FOR EACH ROW
    EXECUTE FUNCTION update_user_invitations_timestamp();
