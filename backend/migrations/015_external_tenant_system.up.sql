-- Migration 015: external_tenants table removed.
-- Tenant lookup is now handled inline via AppHQ integration (runtime_services config).
-- This migration intentionally left empty; the table is dropped below if it exists from
-- a previous schema version.
DROP TABLE IF EXISTS external_tenants CASCADE;
