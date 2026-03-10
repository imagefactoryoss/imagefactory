-- Migration: 009_system_management.down.sql
-- Rollback: System Management tables

DROP TRIGGER IF EXISTS update_git_integration_updated_at ON git_integration;

DROP TABLE IF EXISTS git_integration CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
