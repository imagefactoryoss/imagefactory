-- Migration: 075_cluster_resource_metrics_snapshots.down.sql

DROP TABLE IF EXISTS cluster_pod_metrics_snapshots CASCADE;
DROP TABLE IF EXISTS cluster_node_metrics_snapshots CASCADE;
