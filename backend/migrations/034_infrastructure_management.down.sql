-- Migration: 034_infrastructure_management.down.sql
-- Purpose: Rollback infrastructure tables and views

DROP VIEW IF EXISTS v_infrastructure_health CASCADE;
DROP VIEW IF EXISTS v_infrastructure_nodes CASCADE;

DROP TABLE IF EXISTS node_resource_usage CASCADE;
DROP TABLE IF EXISTS infrastructure_nodes CASCADE;

ALTER TABLE builds DROP COLUMN IF EXISTS assigned_node_id;
ALTER TABLE builds DROP COLUMN IF EXISTS cpu_required;
ALTER TABLE builds DROP COLUMN IF EXISTS memory_required_gb;
ALTER TABLE builds DROP COLUMN IF EXISTS finished_at;
