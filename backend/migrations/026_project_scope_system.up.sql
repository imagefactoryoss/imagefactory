-- Migration: 032_project_scope_system.up.sql
-- Purpose: Add project member table for project-level access control
-- Date: 2026-01-29
-- Notes: Enables users to be assigned to specific projects with optional role overrides

-- ============================================================================
-- 1. PROJECT MEMBERS TABLE
-- ============================================================================
-- Track which users have access to which projects and their role in each project
-- This enables project-scoped access control separate from tenant-level access

CREATE TABLE IF NOT EXISTS project_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Project and User relationship
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Optional project-level role override
    -- NULL means use tenant-level role
    -- Non-null means this user has this specific role IN THIS PROJECT ONLY
    role_id UUID REFERENCES rbac_roles(id) ON DELETE SET NULL,
    
    -- Audit trail
    assigned_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure no duplicate assignments for same project+user combination
    UNIQUE(project_id, user_id)
);

-- Create indexes for query performance
CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_role_id ON project_members(role_id);

-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION update_project_members_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if it exists to avoid conflicts
DROP TRIGGER IF EXISTS update_project_members_updated_at ON project_members;

-- Create trigger to automatically update the updated_at timestamp
CREATE TRIGGER update_project_members_updated_at
    BEFORE UPDATE ON project_members
    FOR EACH ROW
    EXECUTE FUNCTION update_project_members_timestamp();

-- ============================================================================
-- 2. ADD AUDIT COLUMN TO PROJECTS TABLE
-- ============================================================================
-- Track who created each project for audit purposes

ALTER TABLE projects 
ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Create index for queries filtering by creator
CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects(created_by);
