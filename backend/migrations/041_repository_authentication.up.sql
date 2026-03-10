-- Migration: 050_repository_authentication.up.sql
-- Category: Feature Implementation
-- Purpose: Add repository authentication support for private source code repositories

-- ============================================================================
-- REPOSITORY AUTHENTICATION TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS repository_auth (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,
    auth_type VARCHAR(50) NOT NULL CHECK (auth_type IN ('ssh_key', 'token', 'basic_auth', 'oauth')),

    -- Encrypted credential data
    credential_data BYTEA NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(project_id, name)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_repository_auth_project_id ON repository_auth(project_id);
CREATE INDEX IF NOT EXISTS idx_repository_auth_active ON repository_auth(is_active);
CREATE INDEX IF NOT EXISTS idx_repository_auth_created_by ON repository_auth(created_by);

-- Updated at trigger
DROP TRIGGER IF EXISTS update_repository_auth_timestamp ON repository_auth;
CREATE TRIGGER update_repository_auth_timestamp
    BEFORE UPDATE ON repository_auth
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

-- ============================================================================
-- ADD REPOSITORY AUTH REFERENCE TO PROJECTS
-- ============================================================================
ALTER TABLE projects
ADD COLUMN IF NOT EXISTS repository_auth_id UUID REFERENCES repository_auth(id);

-- Add index for the foreign key
CREATE INDEX IF NOT EXISTS idx_projects_repository_auth_id ON projects(repository_auth_id);

-- Add comment for documentation
COMMENT ON TABLE repository_auth IS 'Stores encrypted authentication credentials for private source code repositories';
COMMENT ON COLUMN repository_auth.credential_data IS 'AES-256-GCM encrypted JSON containing authentication credentials';
COMMENT ON COLUMN projects.repository_auth_id IS 'Reference to the active repository authentication configuration';
