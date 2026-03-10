-- Migration: 030_add_build_configs_table.down.sql
-- Rollback: Remove build_configs table

BEGIN;

DROP TRIGGER IF EXISTS update_build_configs_updated_at ON build_configs;
DROP TABLE IF EXISTS build_configs;

COMMIT;
