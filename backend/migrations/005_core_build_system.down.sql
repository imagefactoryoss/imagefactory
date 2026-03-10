-- Migration: 005_core_build_system.down.sql
-- Rollback: Core Build System tables

DROP TRIGGER IF EXISTS update_catalog_image_metadata_updated_at ON catalog_image_metadata;

DROP TABLE IF EXISTS catalog_image_layers CASCADE;
DROP TABLE IF EXISTS catalog_image_metadata CASCADE;
DROP TABLE IF EXISTS build_artifacts CASCADE;
DROP TABLE IF EXISTS build_steps CASCADE;
DROP TABLE IF EXISTS build_metrics CASCADE;
