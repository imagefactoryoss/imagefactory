-- +migrate Down
-- Infrastructure Usage Tracking - Down Migration
-- Purpose: Rollback infrastructure usage table

-- Drop indexes
DROP INDEX IF EXISTS idx_infrastructure_usage_build;
DROP INDEX IF EXISTS idx_infrastructure_usage_type_time;
DROP INDEX IF EXISTS idx_infrastructure_usage_tenant;

-- Drop table
DROP TABLE IF EXISTS infrastructure_usage CASCADE;