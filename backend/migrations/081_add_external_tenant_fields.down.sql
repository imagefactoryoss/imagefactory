-- Migration: 081_add_external_tenant_fields.down.sql
-- Purpose: Roll back AppHQ external metadata fields from tenants table

ALTER TABLE tenants
    DROP COLUMN IF EXISTS app_mgr_email,
    DROP COLUMN IF EXISTS app_mgr_last_name,
    DROP COLUMN IF EXISTS app_mgr_first_name,
    DROP COLUMN IF EXISTS app_mgr_netid,
    DROP COLUMN IF EXISTS lob_primary_email,
    DROP COLUMN IF EXISTS tech_exec_email,
    DROP COLUMN IF EXISTS prod_date,
    DROP COLUMN IF EXISTS internal_flag,
    DROP COLUMN IF EXISTS record_type,
    DROP COLUMN IF EXISTS app_strategy,
    DROP COLUMN IF EXISTS org,
    DROP COLUMN IF EXISTS critical_app,
    DROP COLUMN IF EXISTS company,
    DROP COLUMN IF EXISTS contact_email;
