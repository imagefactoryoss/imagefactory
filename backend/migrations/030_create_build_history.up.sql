-- Migration: 033_create_build_history.up.sql
-- Purpose: Create build history table for ETA calculation and metrics
-- Phase: Phase 3 - Build Queue & Worker Management

-- ============================================================================
-- CREATE BUILD_HISTORY TABLE
-- ============================================================================
-- Tracks historical build metrics for ETA learning and performance analysis
-- Used by ETAService to predict build durations based on method, project, and history

CREATE TABLE IF NOT EXISTS build_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Build reference
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    
    -- Execution metrics
    build_method VARCHAR(50) NOT NULL,  -- kaniko|buildx|container|paketo|packer
    worker_id UUID REFERENCES workers(id) ON DELETE SET NULL,
    
    -- Duration tracking
    duration_seconds INTEGER NOT NULL,
    success BOOLEAN NOT NULL DEFAULT true,
    
    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_build_method CHECK (build_method IN ('kaniko', 'buildx', 'container', 'paketo', 'packer')),
    CONSTRAINT positive_duration CHECK (duration_seconds > 0)
);

-- ============================================================================
-- INDEXES FOR BUILD_HISTORY TABLE
-- ============================================================================
-- Optimize queries for ETA calculation and metrics analysis
CREATE INDEX IF NOT EXISTS idx_build_history_build_id ON build_history(build_id);
CREATE INDEX IF NOT EXISTS idx_build_history_tenant_id ON build_history(tenant_id);
CREATE INDEX IF NOT EXISTS idx_build_history_project_id ON build_history(project_id);
CREATE INDEX IF NOT EXISTS idx_build_history_build_method ON build_history(build_method);
CREATE INDEX IF NOT EXISTS idx_build_history_completed_at ON build_history(completed_at DESC);

-- Composite indexes for ETA queries
CREATE INDEX IF NOT EXISTS idx_build_history_method_success ON build_history(build_method, success, completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_build_history_project_method ON build_history(project_id, build_method, completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_build_history_tenant_method ON build_history(tenant_id, build_method, completed_at DESC);

-- ============================================================================
-- COMMENT
-- ============================================================================
COMMENT ON TABLE build_history IS 'Historical build metrics for ETA prediction and performance analysis';
COMMENT ON COLUMN build_history.build_method IS 'Build method used (for grouping by method in ETA calculations)';
COMMENT ON COLUMN build_history.duration_seconds IS 'Actual build execution time in seconds';
COMMENT ON COLUMN build_history.success IS 'Whether build succeeded (for filtering successful builds in ETA)';
