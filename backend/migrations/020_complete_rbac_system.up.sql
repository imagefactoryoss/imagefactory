-- Migration: Consolidated RBAC System Setup
-- Purpose: Set up complete RBAC infrastructure including user-role assignments table
-- This migration sets up the core RBAC system schema (data seeding handled by seed scripts)

-- ============================================================================
-- 1. USER ROLE ASSIGNMENTS TABLE
-- ============================================================================
-- Create the table to link users to rbac_roles (system and tenant-scoped)
CREATE TABLE IF NOT EXISTS user_role_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES rbac_roles(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    -- Assignment tracking
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Validity
    expires_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, role_id, tenant_id)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_id ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role_id ON user_role_assignments(role_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_tenant_id ON user_role_assignments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_role ON user_role_assignments(user_id, role_id);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_user_role_assignments_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_user_role_assignments_updated_at ON user_role_assignments;
CREATE TRIGGER update_user_role_assignments_updated_at
    BEFORE UPDATE ON user_role_assignments
    FOR EACH ROW
    EXECUTE FUNCTION update_user_role_assignments_timestamp();

-- ============================================================================
-- 2. CLEAN UP AND CONSOLIDATE RBAC ROLES
-- ============================================================================
-- First, delete all duplicate roles (keep only one of each)
DELETE FROM rbac_roles r1
WHERE r1.tenant_id IS NULL
  AND r1.name IN ('System Administrator', 'Owner', 'Developer', 'Viewer', 'Operator', 'Administrator')
  AND EXISTS (
    SELECT 1 FROM rbac_roles r2
    WHERE r2.tenant_id IS NULL
      AND r2.name = r1.name
      AND r2.id < r1.id
  );

-- Rename any remaining "Administrator" roles to "Owner"
UPDATE rbac_roles
SET name = 'Owner', is_system = false, updated_at = CURRENT_TIMESTAMP
WHERE name = 'Administrator' AND tenant_id IS NULL;

-- Delete any remaining "Administrator" roles (shouldn't be any system-wide)
DELETE FROM rbac_roles
WHERE name = 'Administrator' AND tenant_id IS NULL;

-- ============================================================================
-- 3. ENSURE ALL REQUIRED SYSTEM ROLES EXIST
-- ============================================================================
INSERT INTO rbac_roles (tenant_id, name, description, is_system, created_at, updated_at)
VALUES (
    NULL,
    'Operator',
    'Operator role for system management and audit access',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (tenant_id, name) DO NOTHING;

