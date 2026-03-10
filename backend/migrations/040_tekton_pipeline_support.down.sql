-- +migrate Down
-- Remove Tekton pipeline support tables

-- Drop indexes
DROP INDEX IF EXISTS idx_pipeline_run_artifacts_run_id;
DROP INDEX IF EXISTS idx_pipeline_run_logs_timestamp;
DROP INDEX IF EXISTS idx_pipeline_run_logs_run_id;
DROP INDEX IF EXISTS idx_pipeline_templates_active;
DROP INDEX IF EXISTS idx_pipeline_templates_build_method;
DROP INDEX IF EXISTS idx_pipeline_runs_status;
DROP INDEX IF EXISTS idx_pipeline_runs_tenant_id;
DROP INDEX IF EXISTS idx_pipeline_runs_build_execution_id;

-- Drop tables
DROP TABLE IF EXISTS pipeline_run_artifacts;
DROP TABLE IF EXISTS pipeline_run_logs;
DROP TABLE IF EXISTS pipeline_runs;
DROP TABLE IF EXISTS pipeline_templates;