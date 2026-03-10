-- Migration: 011_seed_reference_data.down.sql
-- Rollback: Placeholder - seed data has been moved to separate file

-- This down migration is now a no-op since seed data has been moved to scripts/seed-essential-data.sql
-- When rolling back the up migration, no seed data removal is needed
    'build_log_retention_days', 'scan_frequency', 'enable_audit_logging',
    'require_image_signing', 'default_registry_verify_ssl'
);
DELETE FROM notification_templates WHERE template_type IN ('build_succeeded', 'build_failed') AND created_at >= '2024-01-01'::timestamp;
DELETE FROM compliance_frameworks WHERE framework_type = 'ISO27001' AND created_at >= '2024-01-01'::timestamp;
