-- Migration: 032_create_workers_pool.up.sql
-- Purpose: Create worker pool table for build execution management
-- Phase: Phase 3 - Build Queue & Worker Management

-- ============================================================================
-- CREATE WORKERS TABLE
-- ============================================================================
-- Tracks available build workers (Docker, Kubernetes, Lambda)
-- Used by build queue to assign builds to available capacity

CREATE TABLE IF NOT EXISTS workers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Worker identification
    worker_name VARCHAR(255) NOT NULL,
    worker_type VARCHAR(50) NOT NULL,  -- docker|kubernetes|lambda
    
    -- Capacity management
    capacity INTEGER NOT NULL DEFAULT 4,           -- Max concurrent builds
    current_load INTEGER NOT NULL DEFAULT 0,       -- Current active builds
    
    -- Health tracking
    status VARCHAR(50) NOT NULL DEFAULT 'available',  -- available|busy|offline
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_worker_type CHECK (worker_type IN ('docker', 'kubernetes', 'lambda')),
    CONSTRAINT valid_status CHECK (status IN ('available', 'busy', 'offline')),
    CONSTRAINT positive_capacity CHECK (capacity > 0),
    CONSTRAINT non_negative_load CHECK (current_load >= 0),
    CONSTRAINT valid_load CHECK (current_load <= capacity)
);

-- ============================================================================
-- TRIGGER FUNCTION FOR UPDATED_AT
-- ============================================================================
CREATE OR REPLACE FUNCTION update_workers_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INDEXES FOR WORKERS TABLE
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_workers_tenant_id ON workers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status);
CREATE INDEX IF NOT EXISTS idx_workers_available_capacity ON workers(current_load ASC) WHERE status = 'available';
CREATE INDEX IF NOT EXISTS idx_workers_last_heartbeat ON workers(last_heartbeat DESC);
CREATE INDEX IF NOT EXISTS idx_workers_type ON workers(worker_type);

-- ============================================================================
-- TRIGGER FOR UPDATED_AT
-- ============================================================================
CREATE TRIGGER update_workers_updated_at
BEFORE UPDATE ON workers
FOR EACH ROW
EXECUTE FUNCTION update_workers_timestamp();

-- ============================================================================
-- COMMENT
-- ============================================================================
COMMENT ON TABLE workers IS 'Worker pool for distributed build execution';
COMMENT ON COLUMN workers.capacity IS 'Maximum concurrent builds this worker can handle';
COMMENT ON COLUMN workers.current_load IS 'Number of builds currently running on this worker';
COMMENT ON COLUMN workers.consecutive_failures IS 'Count of consecutive failures for health monitoring';
