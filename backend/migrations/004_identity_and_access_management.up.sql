-- Migration: 004_identity_and_access_management.up.sql
-- Category: Identity & Access Management (IAM)
-- Purpose: Organizational units, permissions, and access control tables

-- Drop existing tables if migration was partially applied
DROP TABLE IF EXISTS org_unit_access CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS org_units CASCADE;

-- ============================================================================
-- 1. ORGANIZATIONAL UNITS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS org_units (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    parent_org_unit_id UUID REFERENCES org_units(id) ON DELETE SET NULL,
    
    -- Department/team information
    department_name VARCHAR(100),
    manager_email VARCHAR(255),
    team_size INTEGER,
    
    -- Resource allocation
    resource_quota_cpu_cores DECIMAL(10,2) DEFAULT 10,
    resource_quota_memory_gb DECIMAL(10,2) DEFAULT 20,
    resource_quota_storage_gb DECIMAL(10,2) DEFAULT 100,
    resource_quota_concurrent_builds INTEGER DEFAULT 5,
    
    -- Configuration
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, deleted
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, slug),
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS idx_org_units_company_id ON org_units(company_id);
CREATE INDEX IF NOT EXISTS idx_org_units_parent_id ON org_units(parent_org_unit_id);
CREATE INDEX IF NOT EXISTS idx_org_units_status ON org_units(status);

-- ============================================================================
-- 2. PERMISSIONS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource VARCHAR(100) NOT NULL, -- e.g., 'users', 'projects', 'builds', 'images'
    action VARCHAR(50) NOT NULL, -- e.g., 'create', 'read', 'update', 'delete'
    description TEXT,
    
    -- Category for grouping
    category VARCHAR(50), -- e.g., 'user_management', 'build_management', 'security'
    
    -- Is this a system permission
    is_system_permission BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(resource, action)
);

CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource);
CREATE INDEX IF NOT EXISTS idx_permissions_action ON permissions(action);
CREATE INDEX IF NOT EXISTS idx_permissions_category ON permissions(category);

-- ============================================================================
-- 2. ROLE PERMISSIONS MAPPING
-- ============================================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);

-- ============================================================================
-- 3. ORG UNIT ACCESS CONTROLS
-- ============================================================================
CREATE TABLE IF NOT EXISTS org_unit_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_unit_id UUID NOT NULL REFERENCES org_units(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Access level for this specific org unit
    access_level VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member, viewer
    
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    
    UNIQUE(org_unit_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_unit_access_org_unit_id ON org_unit_access(org_unit_id);
CREATE INDEX IF NOT EXISTS idx_org_unit_access_user_id ON org_unit_access(user_id);
CREATE INDEX IF NOT EXISTS idx_org_unit_access_level ON org_unit_access(access_level);

-- ============================================================================
-- 4. TENANT-SCOPED GROUPS (for tenant-specific RBAC)
-- ============================================================================
-- Each tenant automatically gets 4 groups: viewer, developer, operator, administrator
CREATE TABLE IF NOT EXISTS tenant_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Role type that this group represents
    role_type VARCHAR(50) NOT NULL, -- 'viewer', 'developer', 'operator', 'administrator'
    
    -- System-managed flag (auto-created groups can't be deleted)
    is_system_group BOOLEAN DEFAULT false,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, deleted
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, slug),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tenant_groups_tenant_id ON tenant_groups(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_groups_role_type ON tenant_groups(role_type);
CREATE INDEX IF NOT EXISTS idx_tenant_groups_status ON tenant_groups(status);

-- ============================================================================
-- 5. GROUP MEMBERS TABLE (users assigned to groups)
-- ============================================================================
CREATE TABLE IF NOT EXISTS group_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL REFERENCES tenant_groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Admin of the group
    is_group_admin BOOLEAN DEFAULT false,
    
    -- Timestamps
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    removed_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_group_members_is_admin ON group_members(is_group_admin);

-- ============================================================================
-- TRIGGER: Update timestamps
-- ============================================================================
CREATE TRIGGER update_org_units_updated_at
    BEFORE UPDATE ON org_units
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_tenant_groups_updated_at 
    BEFORE UPDATE ON tenant_groups 
    FOR EACH ROW 
    EXECUTE FUNCTION update_timestamp();
