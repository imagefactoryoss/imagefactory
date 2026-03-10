-- Migration: 051_git_providers.up.sql
-- Category: Feature Implementation
-- Purpose: Add git provider catalog for project repository configuration

-- ============================================================================
-- GIT PROVIDERS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS git_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_key VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('generic', 'hosted')),
    api_base_url VARCHAR(255),
    supports_api BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_git_providers_active ON git_providers(is_active);

DROP TRIGGER IF EXISTS update_git_providers_timestamp ON git_providers;
CREATE TRIGGER update_git_providers_timestamp
    BEFORE UPDATE ON git_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

-- ============================================================================
-- PROJECTS: ADD GIT PROVIDER KEY
-- ============================================================================
ALTER TABLE projects
ADD COLUMN IF NOT EXISTS git_provider_key VARCHAR(50) NOT NULL DEFAULT 'generic';

CREATE INDEX IF NOT EXISTS idx_projects_git_provider_key ON projects(git_provider_key);

COMMENT ON TABLE git_providers IS 'Catalog of supported Git providers for repository configuration';
COMMENT ON COLUMN projects.git_provider_key IS 'Selected Git provider key for repository operations';
