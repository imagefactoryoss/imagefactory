-- Migration: 050_repository_authentication.down.sql
-- Category: Feature Implementation
-- Purpose: Remove repository authentication support

-- ============================================================================
-- REMOVE REPOSITORY AUTH REFERENCE FROM PROJECTS
-- ============================================================================
ALTER TABLE projects
DROP COLUMN IF EXISTS repository_auth_id;

-- ============================================================================
-- DROP REPOSITORY AUTHENTICATION TABLE
-- ============================================================================
DROP TRIGGER IF EXISTS update_repository_auth_timestamp ON repository_auth;
DROP TABLE IF EXISTS repository_auth;