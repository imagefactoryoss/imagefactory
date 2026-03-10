-- Migration: 008_operations_and_monitoring.down.sql
-- Rollback: Operations & Monitoring tables

DROP TRIGGER IF EXISTS update_environment_access_updated_at ON environment_access;
DROP TRIGGER IF EXISTS update_resource_quotas_updated_at ON resource_quotas;
DROP TRIGGER IF EXISTS update_notification_templates_updated_at ON notification_templates;
DROP TRIGGER IF EXISTS update_notification_channels_updated_at ON notification_channels;

DROP TABLE IF EXISTS environment_access CASCADE;
DROP TABLE IF EXISTS usage_tracking CASCADE;
DROP TABLE IF EXISTS resource_quotas CASCADE;
DROP TABLE IF EXISTS notification_templates CASCADE;
DROP TABLE IF EXISTS notification_channels CASCADE;
