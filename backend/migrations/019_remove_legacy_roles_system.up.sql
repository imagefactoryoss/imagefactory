-- Migration: Remove legacy roles system
-- Purpose: Drop unused roles and user_role_assignments tables in favor of rbac_roles
-- Rationale: The system has migrated to rbac_roles (migration 014) which provides
--           tenant-aware permissions via RBAC. The legacy roles table is company-scoped
--           and is no longer used. Tenant-scoped roles are now handled via rbac_roles
--           and tenant_groups with group_members for user assignments.

-- Drop user_role_assignments first (foreign key dependency)
DROP TABLE IF EXISTS user_role_assignments CASCADE;

-- Drop legacy roles table
DROP TABLE IF EXISTS roles CASCADE;

-- Drop any associated indexes that may have been left behind
DROP INDEX IF EXISTS idx_roles_company_id CASCADE;
DROP INDEX IF EXISTS idx_roles_scope CASCADE;
DROP INDEX IF EXISTS idx_user_role_assignments_user_id CASCADE;
DROP INDEX IF EXISTS idx_user_role_assignments_role_id CASCADE;
DROP INDEX IF EXISTS idx_user_role_assignments_tenant_id CASCADE;
