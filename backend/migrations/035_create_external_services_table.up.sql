-- Migration: 041_create_external_services_table.up.sql
-- Category: System Configuration
-- Purpose: Create external_services table for managing external service integrations

-- ============================================================================
-- EXTERNAL SERVICES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS external_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Service metadata
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    url VARCHAR(500) NOT NULL,
    api_key VARCHAR(1000), -- Encrypted API key
    headers JSONB DEFAULT '{}'::jsonb, -- Custom HTTP headers for authentication and other purposes
    enabled BOOLEAN NOT NULL DEFAULT true,

    -- Audit fields
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Optimistic locking
    version INTEGER NOT NULL DEFAULT 1
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_external_services_name ON external_services(name);
CREATE INDEX IF NOT EXISTS idx_external_services_enabled ON external_services(enabled);
CREATE INDEX IF NOT EXISTS idx_external_services_created_at ON external_services(created_at DESC);

-- Updated at trigger
CREATE TRIGGER update_external_services_timestamp
    BEFORE UPDATE ON external_services
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();