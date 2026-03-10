-- Migration: 002_companies_schema.down.sql
-- Rollback: Companies schema

DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS update_companies_updated_at ON companies;

DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS companies CASCADE;
