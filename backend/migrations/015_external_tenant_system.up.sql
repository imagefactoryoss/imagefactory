-- Create external_tenants table for simulating external system integration
-- This schema represents tenant data from an external system (e.g., enterprise directory, partner API)

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS external_tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(8) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    contact_email VARCHAR(255),
    industry VARCHAR(100),
    country VARCHAR(2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on tenant_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_external_tenants_tenant_id ON external_tenants(tenant_id);
CREATE INDEX IF NOT EXISTS idx_external_tenants_slug ON external_tenants(slug);

-- Add comment for documentation

-- Add comment for documentation
COMMENT ON TABLE external_tenants IS 'Represents tenants from external systems';
COMMENT ON COLUMN external_tenants.tenant_id IS 'Unique 8-digit identifier for the tenant';
COMMENT ON COLUMN external_tenants.slug IS 'URL-friendly identifier for the tenant';
