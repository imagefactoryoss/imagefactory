-- +migrate Up
-- Infrastructure Usage Tracking - Up Migration
-- Purpose: Create infrastructure usage table for analytics and monitoring

-- Create infrastructure_usage table
-- Tracks usage of different infrastructure types for analytics
CREATE TABLE IF NOT EXISTS infrastructure_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    build_execution_id UUID REFERENCES build_executions(id) ON DELETE SET NULL,
    infrastructure_type VARCHAR(50) NOT NULL CHECK (infrastructure_type IN ('kubernetes', 'build_nodes')),
    provider_type VARCHAR(50),
    cluster_name VARCHAR(255),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    cost_cents INTEGER DEFAULT 0,
    resource_usage JSONB, -- CPU, memory, disk usage
    success BOOLEAN,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_infrastructure_usage_tenant ON infrastructure_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_infrastructure_usage_type_time ON infrastructure_usage(infrastructure_type, start_time);
CREATE INDEX IF NOT EXISTS idx_infrastructure_usage_build ON infrastructure_usage(build_execution_id);