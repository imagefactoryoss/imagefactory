-- Migration: 005_core_build_system.up.sql
-- Category: Core Build System
-- Purpose: Build, image, and container artifact tracking

-- ============================================================================
-- 1. BUILD METRICS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS build_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    
    -- Execution metrics
    total_duration_seconds INTEGER,
    docker_build_duration_seconds INTEGER,
    docker_push_duration_seconds INTEGER,
    
    -- Resource usage
    peak_memory_usage_mb INTEGER,
    cpu_usage_percent DECIMAL(5,2),
    disk_read_bytes BIGINT,
    disk_write_bytes BIGINT,
    
    -- Layer information
    total_layers INTEGER,
    reused_layers INTEGER,
    new_layers INTEGER,
    
    -- Image characteristics
    final_image_size_bytes BIGINT,
    uncompressed_size_bytes BIGINT,
    compression_ratio DECIMAL(5,2),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_build_metrics_build_id ON build_metrics(build_id);

-- ============================================================================
-- 2. BUILD STEPS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS build_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    
    -- Step information
    step_number INTEGER NOT NULL,
    step_name VARCHAR(255),
    instruction_type VARCHAR(50), -- FROM, RUN, COPY, ADD, ENV, EXPOSE, etc.
    instruction_line VARCHAR(2000),
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, success, failed, skipped
    
    -- Execution details
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    
    -- Layer information (from docker build)
    layer_digest VARCHAR(255), -- SHA256 digest of the layer
    layer_size_bytes BIGINT,
    cached BOOLEAN DEFAULT false,
    
    -- Error details if failed
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Output logs
    stdout TEXT,
    stderr TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(build_id, step_number)
);

CREATE INDEX IF NOT EXISTS idx_build_steps_build_id ON build_steps(build_id);
CREATE INDEX IF NOT EXISTS idx_build_steps_status ON build_steps(status);

-- ============================================================================
-- 4. BUILD ARTIFACTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS build_artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    
    -- Artifact type
    artifact_type VARCHAR(50) NOT NULL, -- docker_image, sbom, test_report, build_log, security_scan
    
    -- Reference to the artifact
    artifact_name VARCHAR(255) NOT NULL,
    artifact_version VARCHAR(100),
    artifact_location VARCHAR(500), -- URL or path to artifact
    
    -- Artifact details
    artifact_mime_type VARCHAR(100),
    artifact_size_bytes BIGINT,
    
    -- Integrity
    sha256_digest VARCHAR(64),
    
    -- Availability
    is_available BOOLEAN DEFAULT true,
    retention_policy VARCHAR(50), -- permanent, days_30, days_90, days_365, delete_on_success
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Relationships
    image_id UUID REFERENCES catalog_images(id) ON DELETE SET NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_build_artifacts_build_id ON build_artifacts(build_id);
CREATE INDEX IF NOT EXISTS idx_build_artifacts_type ON build_artifacts(artifact_type);
CREATE INDEX IF NOT EXISTS idx_build_artifacts_image_id ON build_artifacts(image_id);

-- ============================================================================
-- 5. CATALOG IMAGE METADATA TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS catalog_image_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE UNIQUE,
    
    -- Docker image information
    docker_config_digest VARCHAR(255),
    docker_manifest_digest VARCHAR(255),
    
    -- Image details
    total_layer_count INTEGER,
    compressed_size_bytes BIGINT,
    uncompressed_size_bytes BIGINT,
    
    -- Content information
    packages_count INTEGER, -- number of installed packages
    vulnerabilities_high_count INTEGER DEFAULT 0,
    vulnerabilities_medium_count INTEGER DEFAULT 0,
    vulnerabilities_low_count INTEGER DEFAULT 0,
    
    -- Runtime information
    entrypoint TEXT, -- JSON array of entrypoint
    cmd TEXT, -- JSON array of default command
    env_vars TEXT, -- JSON object of environment variables
    working_dir VARCHAR(500),
    
    -- Labels (OCI/Docker labels)
    labels TEXT, -- JSON object
    
    -- Scan information
    last_scanned_at TIMESTAMP WITH TIME ZONE,
    scan_tool VARCHAR(100), -- e.g., 'trivy', 'grype', 'snyk'
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_metadata_image_id ON catalog_image_metadata(image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_metadata_vulnerabilities ON catalog_image_metadata(vulnerabilities_high_count, vulnerabilities_medium_count);

-- ============================================================================
-- 6. CATALOG IMAGE LAYERS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS catalog_image_layers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE,
    
    -- Layer information
    layer_number INTEGER NOT NULL,
    layer_digest VARCHAR(255) NOT NULL, -- SHA256 digest
    layer_size_bytes BIGINT,
    
    -- Layer content information
    media_type VARCHAR(100),
    
    -- Base image tracking
    is_base_layer BOOLEAN DEFAULT false,
    base_image_name VARCHAR(255),
    base_image_tag VARCHAR(100),
    
    -- Reuse tracking (for optimization)
    used_in_builds_count INTEGER DEFAULT 1,
    last_used_in_build_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(image_id, layer_number),
    UNIQUE(image_id, layer_digest)
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_layers_image_id ON catalog_image_layers(image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_layers_digest ON catalog_image_layers(layer_digest);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_catalog_image_metadata_updated_at
    BEFORE UPDATE ON catalog_image_metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
