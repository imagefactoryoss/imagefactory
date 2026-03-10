-- Migration: 032_create_workers_pool.down.sql
-- Purpose: Rollback worker pool table

DROP TRIGGER IF EXISTS update_workers_updated_at ON workers;
DROP TABLE IF EXISTS workers CASCADE;
