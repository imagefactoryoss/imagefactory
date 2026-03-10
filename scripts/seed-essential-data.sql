-- ============================================================================
-- Essential System Data - REQUIRED FOR SYSTEM FUNCTION
-- ============================================================================
-- This file contains ONLY the data required for the system to function.
-- No demo/sample data should be included here.
--
-- Data sources:
-- - Migration 003: Companies (minimal), System Users (minimal), System Roles
-- - Migration 011: Core Permissions, Basic Deployment Environments
-- - Migration 022: RBAC Roles and essential permissions
-- ============================================================================

-- ============================================================================
-- SEED MINIMAL COMPANY (required for deployment environments)
-- ============================================================================
INSERT INTO companies (name, description, industry, subscription_tier, status) VALUES
('ImageFactory', 'Internal Image Factory Platform', 'Technology', 'enterprise', 'active')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED ESSENTIAL SYSTEM USERS (minimal required users)
-- ============================================================================
INSERT INTO users (email, first_name, last_name, is_ldap_user, status, email_verified) VALUES
('admin@imagefactory.local', 'Admin', 'User', true, 'active', true),
('system@imagefactory.local', 'System', 'User', false, 'active', true)
ON CONFLICT (email) DO NOTHING;

-- ============================================================================
-- SEED CORE PERMISSIONS (required for RBAC system)
-- ============================================================================
-- ============================================================================
-- SEED BASIC DEPLOYMENT ENVIRONMENTS (required for builds)
-- ============================================================================
INSERT INTO deployment_environments (company_id, name, display_name, environment_type, infrastructure_provider, requires_approval, status)
SELECT c.id, 'development', 'Development', 'development', 'kubernetes', false, 'active' FROM companies c WHERE c.name = 'ImageFactory'
ON CONFLICT (company_id, name) DO NOTHING;

-- ============================================================================
-- SEED GIT PROVIDERS (required for repository configuration)
-- ============================================================================
INSERT INTO git_providers (provider_key, display_name, provider_type, api_base_url, supports_api) VALUES
('generic', 'Generic Git', 'generic', NULL, false),
('github', 'GitHub', 'hosted', 'https://api.github.com', true),
('gitlab', 'GitLab', 'hosted', 'https://gitlab.com/api/v4', true),
('bitbucket', 'Bitbucket', 'hosted', 'https://api.bitbucket.org/2.0', true),
('gitea', 'Gitea', 'hosted', NULL, false)
ON CONFLICT (provider_key) DO NOTHING;

-- System administrator tenant bootstrap is owned by:
--   backend/cmd/essential-config-seeder (ensureSystemBootstrap)
-- This SQL seed file remains schema-agnostic and non-opinionated about tenant IDs.

-- ============================================================================
-- SEED ESSENTIAL RBAC ROLES (required for access control)
-- ============================================================================
INSERT INTO rbac_roles (tenant_id, name, description, is_system, created_at, updated_at) VALUES
(NULL, 'System Administrator', 'Full system access with all permissions', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(NULL, 'Owner', 'Owner-level access for tenant management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(NULL, 'Developer', 'Developer access for building and deploying', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(NULL, 'Viewer', 'Read-only access for monitoring', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- ============================================================================
-- SEED ESSENTIAL PERMISSIONS (required for RBAC)
-- ============================================================================
INSERT INTO permissions (resource, action, description, category, is_system_permission, created_at, updated_at) VALUES
-- System (essential)
('system', 'read', 'Read system information', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'write', 'Write system information', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'manage_config', 'Manage system configuration', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Tenant (essential)
('tenant', 'create', 'Create new tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'read', 'View tenant details', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'list', 'List all tenants', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'update', 'Update tenant settings', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'delete', 'Delete tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'activate', 'Activate tenant', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'manage_users', 'Manage tenant users and roles', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- User (essential)
('user', 'create', 'Create new user', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'read', 'View user details', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'list', 'List users', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'update', 'Update user information', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'delete', 'Delete user', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'manage_roles', 'Manage user roles and permissions', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Role (essential)
('role', 'read', 'View role details', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'list', 'List roles', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'create', 'Create new role', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'update', 'Update role', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'delete', 'Delete role', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'manage_permissions', 'Manage role permissions', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Permissions management (complete CRUD for admin)
('permissions', 'create', 'Create permission', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('permissions', 'read', 'Read permission details', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('permissions', 'update', 'Update permission', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('permissions', 'delete', 'Delete permission', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('permissions', 'manage', 'Manage all permissions', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Build (core feature)
('build', 'create', 'Create new build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'read', 'View build details', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'list', 'List builds', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'cancel', 'Cancel running build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'delete', 'Delete build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'manage_triggers', 'Manage build triggers', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'view_logs', 'View build logs', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Image (core feature)
('image', 'create', 'Create image', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'read', 'Read image information', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'list', 'List images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'update', 'Update image metadata', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'delete', 'Delete image', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Registry (core feature)
('registry', 'read', 'Read registry information', 'registry_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'list', 'List registry items', 'registry_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Deployment (core feature)
('deployment', 'read', 'Read deployment information', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'list', 'List deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'create', 'Create deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Projects (core feature)
('projects', 'create', 'Create new project', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'read', 'View project details', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'update', 'Update project', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'delete', 'Delete project', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'view_members', 'View project members', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'add_member', 'Add member to project', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'remove_member', 'Remove member from project', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'update_member_role', 'Update project member role', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'manage_repository_auth', 'Manage repository authentication', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Configuration
('config', 'read', 'Read configuration', 'configuration', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('config', 'write', 'Write configuration', 'configuration', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Infrastructure
('infrastructure', 'read', 'Read infrastructure information', 'infrastructure_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('infrastructure', 'create', 'Create infrastructure', 'infrastructure_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('infrastructure', 'update', 'Update infrastructure', 'infrastructure_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('infrastructure', 'delete', 'Delete infrastructure', 'infrastructure_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('infrastructure', 'select', 'Select infrastructure', 'infrastructure_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Worker management
('workers', 'register', 'Register worker', 'worker_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('workers', 'unregister', 'Unregister worker', 'worker_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('workers', 'heartbeat', 'Worker heartbeat', 'worker_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('workers', 'read', 'Read worker information', 'worker_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Queue management
('queue', 'enqueue', 'Enqueue task', 'queue_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('queue', 'read', 'Read queue information', 'queue_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('queue', 'assign', 'Assign queue task', 'queue_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('queue', 'complete', 'Complete queue task', 'queue_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('queue', 'fail', 'Fail queue task', 'queue_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Analytics
('analytics', 'read', 'Read analytics data', 'analytics', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Groups
('groups', 'read', 'Read group information', 'group_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('groups', 'write', 'Write group information', 'group_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Invitations
('invitation', 'create', 'Create invitation', 'invitation_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('invitation', 'list', 'List invitations', 'invitation_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('invitation', 'delete', 'Delete invitation', 'invitation_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- MFA (Multi-Factor Authentication)
('mfa', 'read', 'Read MFA settings', 'mfa_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('mfa', 'write', 'Write MFA settings', 'mfa_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- SSO (Single Sign-On)
('sso', 'read', 'Read SSO settings', 'sso_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('sso', 'write', 'Write SSO settings', 'sso_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (resource, action) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO SYSTEM ADMINISTRATOR ROLE (essential)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r
CROSS JOIN permissions p
WHERE r.name = 'System Administrator'
  AND r.is_system = true
  AND r.tenant_id IS NULL
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO OWNER ROLE (essential - tenant admin)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Owner' AND r.is_system = true AND r.tenant_id IS NULL
  AND (
    -- Tenant management (full access except delete)
    (p.resource = 'tenant' AND p.action IN ('read', 'list', 'update', 'activate', 'manage_users'))
    -- User management (full access except delete)
    OR (p.resource = 'user' AND p.action IN ('create', 'read', 'list', 'update', 'manage_roles'))
    -- Role management (full access)
    OR (p.resource = 'role' AND p.action IN ('create', 'read', 'list', 'update', 'delete', 'manage_permissions'))
    -- Permissions management
    OR (p.resource = 'permissions' AND p.action = 'manage')
    -- Project management (full access)
    OR (p.resource = 'projects' AND p.action IN ('create', 'read', 'update', 'delete', 'view_members', 'add_member', 'remove_member', 'update_member_role', 'manage_repository_auth'))
    -- Build management (full access)
    OR (p.resource = 'build' AND p.action IN ('create', 'read', 'list', 'cancel', 'delete', 'manage_triggers', 'view_logs'))
    -- Image management (full access)
    OR (p.resource = 'image' AND p.action IN ('create', 'read', 'list', 'update', 'delete'))
    -- Registry management (full access)
    OR (p.resource = 'registry' AND p.action IN ('read', 'list'))
    -- Deployment management (full access)
    OR (p.resource = 'deployment' AND p.action IN ('read', 'list', 'create'))
    -- Infrastructure management (full access)
    OR (p.resource = 'infrastructure' AND p.action IN ('read', 'create', 'update', 'delete', 'select'))
    -- Configuration (full access)
    OR (p.resource = 'config' AND p.action IN ('read', 'write'))
    -- Groups management
    OR (p.resource = 'groups' AND p.action IN ('read', 'write'))
    -- Invitations
    OR (p.resource = 'invitation' AND p.action IN ('create', 'list', 'delete'))
    -- MFA settings (read access)
    OR (p.resource = 'mfa' AND p.action IN ('read', 'write'))
    -- Analytics
    OR (p.resource = 'analytics' AND p.action = 'read')
    -- System read access
    OR (p.resource = 'system' AND p.action IN ('read'))
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO DEVELOPER ROLE (essential - development access)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Developer' AND r.is_system = true AND r.tenant_id IS NULL
  AND (
    -- Tenant read access
    (p.resource = 'tenant' AND p.action IN ('read', 'list'))
    -- User read access
    OR (p.resource = 'user' AND p.action IN ('read', 'list'))
    -- Role read access
    OR (p.resource = 'role' AND p.action IN ('read', 'list'))
    -- Project management (create, read, manage members)
    OR (p.resource = 'projects' AND p.action IN ('create', 'read', 'update', 'view_members', 'add_member', 'remove_member', 'update_member_role', 'manage_repository_auth'))
    -- Build management (full development access)
    OR (p.resource = 'build' AND p.action IN ('create', 'read', 'list', 'cancel', 'delete', 'manage_triggers', 'view_logs'))
    -- Image management (create and read)
    OR (p.resource = 'image' AND p.action IN ('create', 'read', 'list', 'update'))
    -- Registry management (full access for CI/CD)
    OR (p.resource = 'registry' AND p.action IN ('read', 'list'))
    -- Deployment management (create and read)
    OR (p.resource = 'deployment' AND p.action IN ('read', 'list', 'create'))
    -- Infrastructure management (read and select)
    OR (p.resource = 'infrastructure' AND p.action IN ('read', 'select'))
    -- Configuration (read for build configs)
    OR (p.resource = 'config' AND p.action IN ('read', 'write'))
    -- Groups (read)
    OR (p.resource = 'groups' AND p.action IN ('read'))
    -- Invitations (can invite others)
    OR (p.resource = 'invitation' AND p.action IN ('create', 'list'))
    -- Analytics (read for build insights)
    OR (p.resource = 'analytics' AND p.action = 'read')
    -- System read access
    OR (p.resource = 'system' AND p.action IN ('read'))
    -- Workers (read for build worker info)
    OR (p.resource = 'workers' AND p.action IN ('read'))
    -- Queue (can enqueue builds)
    OR (p.resource = 'queue' AND p.action IN ('read', 'enqueue'))
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO VIEWER ROLE (essential - read-only access)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Viewer' AND r.is_system = true AND r.tenant_id IS NULL
  AND (
    -- Tenant read access
    (p.resource = 'tenant' AND p.action IN ('read', 'list'))
    -- User read access
    OR (p.resource = 'user' AND p.action IN ('read', 'list'))
    -- Role read access
    OR (p.resource = 'role' AND p.action IN ('read', 'list'))
    -- Project read access
    OR (p.resource = 'projects' AND p.action IN ('read', 'view_members'))
    -- Build read access
    OR (p.resource = 'build' AND p.action IN ('read', 'list', 'view_logs'))
    -- Image read access
    OR (p.resource = 'image' AND p.action IN ('read', 'list'))
    -- Registry read access
    OR (p.resource = 'registry' AND p.action IN ('read', 'list'))
    -- Deployment read access
    OR (p.resource = 'deployment' AND p.action IN ('read', 'list'))
    -- Infrastructure read access
    OR (p.resource = 'infrastructure' AND p.action IN ('read'))
    -- Analytics read access
    OR (p.resource = 'analytics' AND p.action = 'read')
    -- System read access
    OR (p.resource = 'system' AND p.action IN ('read'))
    -- Groups (read)
    OR (p.resource = 'groups' AND p.action IN ('read'))
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;


-- ============================================================================
-- ASSIGN ADMIN USER TO SYSTEM ADMINISTRATOR ROLE (essential)
-- ============================================================================
INSERT INTO user_role_assignments (user_id, role_id, assigned_by_user_id, assigned_at, created_at, updated_at)
SELECT u.id, r.id, u.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM users u
CROSS JOIN rbac_roles r
WHERE u.email = 'admin@imagefactory.local'
  AND r.name = 'System Administrator'
  AND r.is_system = true
  AND r.tenant_id IS NULL
ON CONFLICT (user_id, role_id, tenant_id) DO NOTHING;
