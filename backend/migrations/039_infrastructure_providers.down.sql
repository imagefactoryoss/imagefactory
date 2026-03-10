-- Migration: 048_infrastructure_providers.down.sql
-- Category: Infrastructure Management
-- Purpose: Drop infrastructure providers tables

-- Drop indexes
DROP INDEX IF EXISTS idx_provider_permissions_provider;
DROP INDEX IF EXISTS idx_provider_permissions_tenant;
DROP INDEX IF EXISTS idx_infrastructure_providers_is_global;
DROP INDEX IF EXISTS idx_providers_tenant_type;

-- Drop tables
DROP TABLE IF EXISTS provider_permissions;
DROP TABLE IF EXISTS infrastructure_providers;
