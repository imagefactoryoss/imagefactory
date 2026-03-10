-- Migration: 030_add_build_configs_table.up.sql
-- Category: Build Configuration
-- Purpose: Add dedicated table for build configuration with method-specific validation
-- Reason: Current JSONB approach doesn't validate per method (Kaniko vs Buildx vs Paketo vs Packer)

BEGIN;

-- ============================================================================
-- BUILD CONFIGS TABLE - Method-specific configuration storage
-- ============================================================================
CREATE TABLE IF NOT EXISTS build_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE UNIQUE,
    source_id UUID REFERENCES project_sources(id) ON DELETE SET NULL,
    ref_policy VARCHAR(30) NOT NULL DEFAULT 'source_default',
    fixed_ref VARCHAR(255),
    
    -- Build method identifier (determines which fields are used)
    build_method VARCHAR(50) NOT NULL,
    CONSTRAINT valid_build_method CHECK (build_method IN ('kaniko', 'buildx', 'container', 'docker', 'nix', 'paketo', 'packer')),
    CONSTRAINT build_configs_ref_policy_check CHECK (ref_policy IN ('source_default', 'fixed', 'event_ref')),
    
    -- ========================================================================
    -- SHARED FIELDS (used by multiple methods)
    -- ========================================================================
    sbom_tool VARCHAR(50),      -- SBOM tool selection (syft, grype, trivy)
    scan_tool VARCHAR(50),      -- Security scanner selection (trivy, clair, grype, snyk)
    registry_type VARCHAR(50),  -- Registry backend (s3, harbor, quay, artifactory)
    secret_manager_type VARCHAR(50), -- Secret manager (vault, aws_secretsmanager, azure_keyvault, gcp_secretmanager)
    build_args JSONB,           -- Build arguments (ENV variables passed to build)
    environment JSONB,          -- Environment variables
    secrets JSONB,              -- Encrypted secrets (handled by backend encryption)
    metadata JSONB,             -- Method-specific metadata not covered by schema
    
    -- ========================================================================
    -- KANIKO-SPECIFIC FIELDS
    -- (Used when: build_method = 'kaniko')
    -- Kaniko builds container images using a Dockerfile
    -- ========================================================================
    dockerfile TEXT,            -- Dockerfile content or path
    build_context VARCHAR(255), -- Build context directory (usually ".")
    cache_enabled BOOLEAN,      -- Enable layer caching
    cache_repo VARCHAR(255),    -- Cache repository for layer reuse (e.g., Harbor, Docker registry)
    
    -- ========================================================================
    -- BUILDX-SPECIFIC FIELDS
    -- (Used when: build_method = 'buildx')
    -- Buildx provides multi-platform builds and advanced caching
    -- ========================================================================
    platforms JSONB,            -- Array of target platforms ["linux/amd64", "linux/arm64", "darwin/amd64"]
    cache_from JSONB,           -- Array of cache sources to pull from
    cache_to VARCHAR(255),      -- Cache export destination (where to push layer cache)
    
    -- ========================================================================
    -- CONTAINER (Docker) SPECIFIC FIELDS
    -- (Used when: build_method = 'container')
    -- Standard Docker build approach
    -- ========================================================================
    target_stage VARCHAR(255),  -- Target stage in multi-stage Dockerfile build
    
    -- ========================================================================
    -- PAKETO-SPECIFIC FIELDS
    -- (Used when: build_method = 'paketo')
    -- Paketo buildpacks provide automated building for various languages
    -- ========================================================================
    builder VARCHAR(255),       -- Builder image (e.g., "paketobuildpacks/builder:base")
    buildpacks JSONB,           -- Array of buildpack image URIs
    
    -- ========================================================================
    -- PACKER-SPECIFIC FIELDS
    -- (Used when: build_method = 'packer')
    -- Packer builds machine images (AMIs, VMs, etc)
    -- ========================================================================
    packer_template TEXT,       -- Packer HCL template content
    
    -- ========================================================================
    -- AUDIT & TIMESTAMPS
    -- ========================================================================
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- INDEXES for performance
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_build_configs_build_id ON build_configs(build_id);
CREATE INDEX IF NOT EXISTS idx_build_configs_build_method ON build_configs(build_method);
CREATE INDEX IF NOT EXISTS idx_build_configs_source_id ON build_configs(source_id);
CREATE INDEX IF NOT EXISTS idx_build_configs_ref_policy ON build_configs(ref_policy);

-- ============================================================================
-- TRIGGER to update updated_at timestamp automatically
-- ============================================================================
CREATE TRIGGER update_build_configs_updated_at
    BEFORE UPDATE ON build_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

COMMIT;
