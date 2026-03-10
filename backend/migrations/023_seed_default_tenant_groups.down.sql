-- Migration: 025_seed_default_tenant_groups.down.sql
-- Category: Seed Data (moved to scripts/seed-essential-data.sql)
-- Purpose: Placeholder - seed data has been moved to separate file

-- This down migration is now a no-op since seed data has been moved to scripts/seed-essential-data.sql
-- When rolling back the up migration, no seed data removal is needed
  AND is_system_group = true
  AND slug = 'developers'
  AND name = 'Developers';

-- Remove operator groups
DELETE FROM tenant_groups
WHERE role_type = 'operator'
  AND is_system_group = true
  AND slug = 'operators'
  AND name = 'Operators';

-- Remove owner groups
DELETE FROM tenant_groups
WHERE role_type = 'owner'
  AND is_system_group = true
  AND slug = 'owners'
  AND name = 'Owners';

-- Remove old administrator groups (legacy)
DELETE FROM tenant_groups
WHERE role_type = 'administrator'
  AND is_system_group = true;