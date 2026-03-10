-- Migration: 033_create_build_history.down.sql
-- Purpose: Rollback build history table

DROP TABLE IF EXISTS build_history CASCADE;
