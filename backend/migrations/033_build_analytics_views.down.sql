-- Migration: 033_build_analytics_views.down.sql
-- Purpose: Rollback views created in up migration

DROP VIEW IF EXISTS v_build_failure_rate_by_project CASCADE;
DROP VIEW IF EXISTS v_build_failure_reasons CASCADE;
DROP VIEW IF EXISTS v_build_slowest_builds CASCADE;
DROP VIEW IF EXISTS v_build_failures CASCADE;
DROP VIEW IF EXISTS v_build_performance_trends CASCADE;
DROP VIEW IF EXISTS v_build_analytics CASCADE;

DROP INDEX IF EXISTS idx_builds_status_completed_at;
DROP INDEX IF EXISTS idx_builds_created_at_completed_at;
DROP INDEX IF EXISTS idx_builds_tenant_status_created;
DROP INDEX IF EXISTS idx_builds_project_status_created;
DROP INDEX IF EXISTS idx_builds_status_created_at;
