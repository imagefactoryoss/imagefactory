-- Drop triggers
DROP TRIGGER IF EXISTS update_build_executions_timestamp ON build_executions;

-- Drop indexes
DROP INDEX IF EXISTS idx_build_executions_build_id;
DROP INDEX IF EXISTS idx_build_executions_config_id;
DROP INDEX IF EXISTS idx_build_executions_status;
DROP INDEX IF EXISTS idx_build_executions_created_at;
DROP INDEX IF EXISTS idx_build_execution_logs_execution_id;
DROP INDEX IF EXISTS idx_build_execution_logs_timestamp;

-- Drop tables
DROP TABLE IF EXISTS build_execution_logs;
DROP TABLE IF EXISTS build_executions;
