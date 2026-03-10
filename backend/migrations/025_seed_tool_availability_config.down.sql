-- Migration: 027_seed_tool_availability_config.down.sql
-- Category: System Configuration (moved to scripts/seed-essential-data.sql)
-- Purpose: Placeholder - seed data has been moved to separate file

-- This down migration is now a no-op since seed data has been moved to scripts/seed-essential-data.sql
-- When rolling back the up migration, no seed data removal is needed
  AND config_key = 'tool_availability'
  AND tenant_id IS NULL
  AND is_default = true;