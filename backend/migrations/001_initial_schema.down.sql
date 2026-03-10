-- Migration: 001_initial_schema.down.sql
-- Rollback: Initial schema

DROP TRIGGER IF EXISTS update_approval_requests_updated_at ON approval_requests;
DROP TRIGGER IF EXISTS update_approval_workflows_updated_at ON approval_workflows;
DROP TRIGGER IF EXISTS update_security_policies_updated_at ON security_policies;
DROP TRIGGER IF EXISTS update_builds_updated_at ON builds;
DROP TRIGGER IF EXISTS update_catalog_images_updated_at ON catalog_images;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_user_sessions_updated_at ON user_sessions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

DROP TABLE IF EXISTS approval_requests CASCADE;
DROP TABLE IF EXISTS approval_workflows CASCADE;
DROP TABLE IF EXISTS security_policies CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP INDEX IF EXISTS idx_builds_dispatch_next_run_at;
DROP TABLE IF EXISTS builds CASCADE;
DROP TABLE IF EXISTS catalog_image_tags CASCADE;
DROP TABLE IF EXISTS catalog_image_versions CASCADE;
DROP TABLE IF EXISTS catalog_images CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS user_role_assignments CASCADE;
DROP TABLE IF EXISTS user_sessions CASCADE;
DROP INDEX IF EXISTS idx_users_tenant_id;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

DROP FUNCTION IF EXISTS update_timestamp() CASCADE;
