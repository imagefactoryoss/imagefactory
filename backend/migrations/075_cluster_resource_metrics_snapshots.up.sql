-- Migration: 075_cluster_resource_metrics_snapshots.up.sql
-- Purpose: Store lightweight Kubernetes node and pod resource metric snapshots

CREATE TABLE IF NOT EXISTS cluster_node_metrics_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cluster_name VARCHAR(255) NOT NULL,
    node_name VARCHAR(255) NOT NULL,
    cpu_usage_millicores BIGINT NOT NULL DEFAULT 0,
    memory_usage_bytes BIGINT NOT NULL DEFAULT 0,
    cpu_allocatable_millicores BIGINT NOT NULL DEFAULT 0,
    memory_allocatable_bytes BIGINT NOT NULL DEFAULT 0,
    ephemeral_storage_allocatable_bytes BIGINT NOT NULL DEFAULT 0,
    window_seconds INTEGER NOT NULL DEFAULT 0,
    collected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cluster_node_metrics_cluster_collected
    ON cluster_node_metrics_snapshots(cluster_name, collected_at DESC);

CREATE INDEX IF NOT EXISTS idx_cluster_node_metrics_node_collected
    ON cluster_node_metrics_snapshots(node_name, collected_at DESC);

CREATE TABLE IF NOT EXISTS cluster_pod_metrics_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cluster_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    pod_name VARCHAR(255) NOT NULL,
    node_name VARCHAR(255),
    container_count INTEGER NOT NULL DEFAULT 0,
    cpu_usage_millicores BIGINT NOT NULL DEFAULT 0,
    memory_usage_bytes BIGINT NOT NULL DEFAULT 0,
    window_seconds INTEGER NOT NULL DEFAULT 0,
    collected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cluster_pod_metrics_cluster_collected
    ON cluster_pod_metrics_snapshots(cluster_name, collected_at DESC);

CREATE INDEX IF NOT EXISTS idx_cluster_pod_metrics_namespace_collected
    ON cluster_pod_metrics_snapshots(namespace, collected_at DESC);

CREATE INDEX IF NOT EXISTS idx_cluster_pod_metrics_pod_collected
    ON cluster_pod_metrics_snapshots(pod_name, collected_at DESC);

COMMENT ON TABLE cluster_node_metrics_snapshots IS 'Lightweight snapshots of Kubernetes node resource usage from metrics-server';
COMMENT ON TABLE cluster_pod_metrics_snapshots IS 'Lightweight snapshots of Kubernetes pod resource usage from metrics-server';
