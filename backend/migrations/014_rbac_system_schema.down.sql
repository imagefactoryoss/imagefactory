-- Migration: 013_rbac_system_schema.down.sql
-- Purpose: Rollback RBAC system schema changes

-- Drop user invitations table and related objects
DROP TABLE IF EXISTS user_invitations CASCADE;
DROP TRIGGER IF EXISTS update_user_invitations_updated_at ON user_invitations;
DROP FUNCTION IF EXISTS update_user_invitations_timestamp();

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_rbac_roles_timestamp();

-- Remove added columns from roles table
ALTER TABLE roles
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS is_system,
DROP COLUMN IF EXISTS permissions,
DROP COLUMN IF EXISTS tenant_id;
