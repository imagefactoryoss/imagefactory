-- Migration: 056_add_external_tenant_fields.up.sql
-- Purpose: Add AppHQ external fields to tenants table for richer metadata

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS contact_email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS company VARCHAR(100),
    ADD COLUMN IF NOT EXISTS critical_app VARCHAR(10),
    ADD COLUMN IF NOT EXISTS org VARCHAR(100),
    ADD COLUMN IF NOT EXISTS app_strategy VARCHAR(100),
    ADD COLUMN IF NOT EXISTS record_type VARCHAR(100),
    ADD COLUMN IF NOT EXISTS internal_flag VARCHAR(100),
    ADD COLUMN IF NOT EXISTS prod_date VARCHAR(32),
    ADD COLUMN IF NOT EXISTS tech_exec_email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS lob_primary_email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS app_mgr_netid VARCHAR(100),
    ADD COLUMN IF NOT EXISTS app_mgr_first_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS app_mgr_last_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS app_mgr_email VARCHAR(255);

-- Note: status, description, name, slug, and tenant_code already exist.
-- If you want to persist all 18 fields, add more columns as needed.
