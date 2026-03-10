-- Migration: 025_system_configuration_table.down.sql
-- Category: System Configuration
-- Purpose: Remove system_configs table and related objects

-- Drop trigger first
DROP TRIGGER IF EXISTS update_system_configs_updated_at ON system_configs;

-- Drop function
DROP FUNCTION IF EXISTS update_system_configs_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_system_configs_universal_type_key;
DROP INDEX IF EXISTS idx_system_configs_tenant_type_key;
DROP INDEX IF EXISTS idx_system_configs_status;
DROP INDEX IF EXISTS idx_system_configs_config_key;
DROP INDEX IF EXISTS idx_system_configs_config_type;
DROP INDEX IF EXISTS idx_system_configs_tenant_id;

-- Drop table
DROP TABLE IF EXISTS system_configs;