-- Migration: 004_identity_and_access_management.down.sql
-- Rollback: Identity & Access Management tables

DROP TRIGGER IF EXISTS update_tenant_groups_updated_at ON tenant_groups;
DROP TRIGGER IF EXISTS update_org_units_updated_at ON org_units;

DROP TABLE IF EXISTS group_members CASCADE;
DROP TABLE IF EXISTS tenant_groups CASCADE;
DROP TABLE IF EXISTS org_unit_access CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS org_units CASCADE;
