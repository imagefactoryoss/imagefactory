-- Migration: 031_create_build_triggers.down.sql
-- Purpose: Rollback build_triggers table and associated objects
-- Date: February 1, 2026

BEGIN;

DROP INDEX IF EXISTS idx_webhook_receipts_provider_status;
DROP INDEX IF EXISTS idx_webhook_receipts_project_received_at;
DROP INDEX IF EXISTS idx_webhook_receipts_project_provider_delivery;
DROP TABLE IF EXISTS webhook_receipts;

-- Drop trigger and function
DROP TRIGGER IF EXISTS update_build_triggers_timestamp ON build_triggers;
DROP FUNCTION IF EXISTS update_build_triggers_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_build_triggers_git_event;
DROP INDEX IF EXISTS idx_build_triggers_webhook_url;
DROP INDEX IF EXISTS idx_build_triggers_scheduled_active;
DROP INDEX IF EXISTS idx_build_triggers_build_id;
DROP INDEX IF EXISTS idx_build_triggers_project_id;
DROP INDEX IF EXISTS idx_build_triggers_tenant_id;

-- Drop table
DROP TABLE IF EXISTS build_triggers;

COMMIT;
