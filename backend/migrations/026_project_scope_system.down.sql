-- Migration: 032_project_scope_system.down.sql
-- Purpose: Rollback project member table and associated structures
-- Date: 2026-01-29

-- ============================================================================
-- 1. REMOVE PROJECT MEMBERS TABLE
-- ============================================================================
-- Drop trigger first (before dropping table)
DROP TRIGGER IF EXISTS update_project_members_updated_at ON project_members;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_project_members_timestamp();

-- Drop indexes
DROP INDEX IF EXISTS idx_project_members_project_id;
DROP INDEX IF EXISTS idx_project_members_user_id;
DROP INDEX IF EXISTS idx_project_members_role_id;

-- Drop the table
DROP TABLE IF EXISTS project_members;

-- ============================================================================
-- 2. REMOVE AUDIT COLUMN FROM PROJECTS
-- ============================================================================
-- Remove created_by column from projects table
ALTER TABLE projects DROP COLUMN IF EXISTS created_by;

-- Drop associated index
DROP INDEX IF EXISTS idx_projects_created_by;
