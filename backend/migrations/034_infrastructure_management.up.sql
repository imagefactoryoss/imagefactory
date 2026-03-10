-- Migration: 034_infrastructure_management.up.sql
-- Purpose: Create tenant-aware infrastructure tables and views for queue monitoring and infrastructure status

ALTER TABLE builds ADD COLUMN IF NOT EXISTS assigned_node_id UUID;
ALTER TABLE builds ADD COLUMN IF NOT EXISTS cpu_required NUMERIC(10, 2) DEFAULT 1;
ALTER TABLE builds ADD COLUMN IF NOT EXISTS memory_required_gb NUMERIC(10, 2) DEFAULT 2;
ALTER TABLE builds ADD COLUMN IF NOT EXISTS finished_at TIMESTAMP WITH TIME ZONE;

CREATE TABLE IF NOT EXISTS infrastructure_nodes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ready',

    total_cpu_cores NUMERIC(10, 2) NOT NULL,
    total_memory_gb NUMERIC(10, 2) NOT NULL,
    total_disk_gb NUMERIC(10, 2) NOT NULL,

    last_heartbeat TIMESTAMP WITH TIME ZONE,
    maintenance_mode BOOLEAN DEFAULT FALSE,

    labels JSONB,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, name)
);

CREATE TABLE IF NOT EXISTS node_resource_usage (
    id UUID PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES infrastructure_nodes(id) ON DELETE CASCADE,
    used_cpu_cores NUMERIC(10, 2) NOT NULL DEFAULT 0,
    used_memory_gb NUMERIC(10, 2) NOT NULL DEFAULT 0,
    used_disk_gb NUMERIC(10, 2) NOT NULL DEFAULT 0,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_node_resource_usage_node FOREIGN KEY (node_id) REFERENCES infrastructure_nodes(id)
);

CREATE OR REPLACE VIEW v_infrastructure_nodes AS
SELECT
    n.id,
    n.tenant_id,
    n.name,
    n.status,
    n.last_heartbeat,
    EXTRACT(EPOCH FROM (NOW() - n.last_heartbeat))::INTEGER AS heartbeat_age_seconds,
    n.total_cpu_cores AS total_cpu_capacity,
    n.total_memory_gb AS total_memory_capacity_gb,
    n.total_disk_gb AS total_disk_capacity_gb,
    n.maintenance_mode,
    n.labels,

    COALESCE(SUM(CASE WHEN b.status = 'running' THEN 1 ELSE 0 END), 0)::INTEGER AS current_builds,
    COALESCE(SUM(CASE WHEN b.status = 'running' THEN b.cpu_required ELSE 0 END), 0)::NUMERIC(10, 2) AS used_cpu_cores,
    COALESCE(SUM(CASE WHEN b.status = 'running' THEN b.memory_required_gb ELSE 0 END), 0)::NUMERIC(10, 2) AS used_memory_gb,

    (n.total_cpu_cores - COALESCE(SUM(CASE WHEN b.status = 'running' THEN b.cpu_required ELSE 0 END), 0))::NUMERIC(10, 2) AS available_cpu_cores,
    (n.total_memory_gb - COALESCE(SUM(CASE WHEN b.status = 'running' THEN b.memory_required_gb ELSE 0 END), 0))::NUMERIC(10, 2) AS available_memory_gb,

    CASE
        WHEN n.total_cpu_cores > 0 THEN
            ROUND(COALESCE(SUM(CASE WHEN b.status = 'running' THEN b.cpu_required ELSE 0 END), 0)::NUMERIC / n.total_cpu_cores * 100, 2)
        ELSE 0
    END::NUMERIC(5, 2) AS cpu_usage_percent,

    CASE
        WHEN n.total_memory_gb > 0 THEN
            ROUND(COALESCE(SUM(CASE WHEN b.status = 'running' THEN b.memory_required_gb ELSE 0 END), 0)::NUMERIC / n.total_memory_gb * 100, 2)
        ELSE 0
    END::NUMERIC(5, 2) AS memory_usage_percent,

    n.created_at,
    n.updated_at
FROM infrastructure_nodes n
LEFT JOIN builds b ON n.id = b.assigned_node_id
    AND b.tenant_id = n.tenant_id
GROUP BY n.id;

CREATE OR REPLACE VIEW v_infrastructure_health AS
SELECT
    tenant_id,
    COUNT(*)::INTEGER AS total_nodes,
    COALESCE(SUM(CASE WHEN status = 'ready' AND maintenance_mode = FALSE THEN 1 ELSE 0 END), 0)::INTEGER AS healthy_nodes,
    COALESCE(SUM(CASE WHEN status = 'offline' THEN 1 ELSE 0 END), 0)::INTEGER AS offline_nodes,
    COALESCE(SUM(CASE WHEN maintenance_mode = TRUE THEN 1 ELSE 0 END), 0)::INTEGER AS maintenance_nodes,

    SUM(total_cpu_cores)::NUMERIC(10, 2) AS total_cpu_capacity,
    SUM(total_memory_gb)::NUMERIC(10, 2) AS total_memory_capacity_gb,
    SUM(total_disk_gb)::NUMERIC(10, 2) AS total_disk_capacity_gb,

    0::NUMERIC(10, 2) AS used_cpu_cores,
    0::NUMERIC(10, 2) AS used_memory_gb,
    0::NUMERIC(10, 2) AS used_disk_gb,

    0::NUMERIC(5, 2) AS avg_cpu_usage_percent,
    0::NUMERIC(5, 2) AS avg_memory_usage_percent,
    0::NUMERIC(5, 2) AS avg_disk_usage_percent
FROM infrastructure_nodes
GROUP BY tenant_id;

CREATE INDEX IF NOT EXISTS idx_builds_assigned_node
    ON builds(assigned_node_id)
    WHERE status = 'running';

CREATE INDEX IF NOT EXISTS idx_infrastructure_nodes_tenant_id
    ON infrastructure_nodes(tenant_id);

CREATE INDEX IF NOT EXISTS idx_infrastructure_nodes_status
    ON infrastructure_nodes(status);

CREATE INDEX IF NOT EXISTS idx_infrastructure_nodes_last_heartbeat
    ON infrastructure_nodes(last_heartbeat DESC);

CREATE INDEX IF NOT EXISTS idx_node_resource_usage_node_id_recorded
    ON node_resource_usage(node_id, recorded_at DESC);

COMMENT ON TABLE infrastructure_nodes IS 'Tenant-scoped build nodes/runners managed by the system';
COMMENT ON VIEW v_infrastructure_nodes IS 'Tenant-scoped real-time view of node status and resource usage';
COMMENT ON VIEW v_infrastructure_health IS 'Tenant-scoped infrastructure health summary';
