-- Migration: 051_git_providers.down.sql
-- Category: Feature Implementation
-- Purpose: Remove git provider catalog

ALTER TABLE projects
DROP COLUMN IF EXISTS git_provider_key;

DROP TRIGGER IF EXISTS update_git_providers_timestamp ON git_providers;
DROP TABLE IF EXISTS git_providers;
