-- Migration: 006_container_lifecycle_management.down.sql
-- Rollback: Container Lifecycle Management tables

DROP TRIGGER IF EXISTS update_deployment_environments_updated_at ON deployment_environments;
DROP TRIGGER IF EXISTS update_deployments_updated_at ON deployments;
DROP TRIGGER IF EXISTS update_git_repositories_updated_at ON git_repositories;
DROP TRIGGER IF EXISTS update_container_repositories_updated_at ON container_repositories;
DROP TRIGGER IF EXISTS update_container_registries_updated_at ON container_registries;

DROP TABLE IF EXISTS deployment_environments CASCADE;
DROP TABLE IF EXISTS deployments CASCADE;
DROP TABLE IF EXISTS git_repositories CASCADE;
DROP TABLE IF EXISTS container_repositories CASCADE;
DROP TABLE IF EXISTS container_registries CASCADE;
