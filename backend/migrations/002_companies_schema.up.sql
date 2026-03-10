-- Migration: 002_companies_schema.up.sql
-- Category: Multi-Tenancy Foundation
-- Purpose: Create company structure to support multi-tenant architecture

-- ============================================================================
-- 1. COMPANIES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    industry VARCHAR(100),
    website_url VARCHAR(255),
    size VARCHAR(50), -- e.g., 'small', 'medium', 'large', 'enterprise'
    headquarters_country VARCHAR(100),
    
    -- Billing and subscription
    subscription_tier VARCHAR(50) NOT NULL DEFAULT 'standard', -- basic, standard, enterprise
    billing_contact_email VARCHAR(255),
    
    -- Configuration
    enforce_mfa BOOLEAN DEFAULT false,
    enforce_image_signing BOOLEAN DEFAULT true,
    max_concurrent_builds INTEGER DEFAULT 10,
    
    -- Status and timestamps
    status VARCHAR(50) DEFAULT 'active', -- active, suspended, deleted
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_companies_status ON companies(status);
CREATE INDEX IF NOT EXISTS idx_companies_created_at ON companies(created_at);

-- ============================================================================
-- 2. ROLES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Role scope - where this role applies
    scope VARCHAR(50) NOT NULL DEFAULT 'company', -- system, company, org_unit, project
    
    -- Predefined or custom
    is_system_role BOOLEAN DEFAULT false,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, deleted
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS idx_roles_company_id ON roles(company_id);
CREATE INDEX IF NOT EXISTS idx_roles_scope ON roles(scope);
CREATE INDEX IF NOT EXISTS idx_roles_status ON roles(status);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
