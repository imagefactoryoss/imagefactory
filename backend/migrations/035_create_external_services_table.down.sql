-- Migration: 041_create_external_services_table.down.sql
-- Category: System Configuration
-- Purpose: Remove external_services table

-- Drop trigger
DROP TRIGGER IF EXISTS update_external_services_timestamp ON external_services;

-- Drop indexes
DROP INDEX IF EXISTS idx_external_services_name;
DROP INDEX IF EXISTS idx_external_services_enabled;
DROP INDEX IF EXISTS idx_external_services_created_at;

-- Drop table
DROP TABLE IF EXISTS external_services;