-- ============================================================================
-- Essential Development Seed Data
-- ============================================================================
-- This file populates core data needed for development
-- Should be run AFTER schema migrations are applied
-- 
-- Data sources:
-- - Migration 003: Companies, System Users, System Roles, User Role Assignments
-- - Migration 011: Permissions, Deployment Environments
-- - Migration 012: System Admin Group (from first tenant)
-- - Migration 013: LDAP Users
-- - Migration 015: External Tenants
-- - Migration 018: System Admin Tenant and Group
-- - Migration 025: Default Tenant Groups (viewers, developers, operators, owners)
-- - Migration 039: Infrastructure Nodes (sample build nodes)
-- ============================================================================

-- ============================================================================
-- SEED COMPANIES (from migration 003)
-- ============================================================================
INSERT INTO companies (name, description, industry, subscription_tier, status) VALUES
('ImageFactory', 'Internal Image Factory Platform', 'Technology', 'enterprise', 'active')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED SYSTEM USERS (from migrations 003, 013)
-- ============================================================================
INSERT INTO users (email, first_name, last_name, is_ldap_user, status, email_verified) VALUES
('admin@imagefactory.local', 'Admin', 'User', true, 'active', true),
('system@imagefactory.local', 'System', 'User', false, 'active', true),
-- LDAP Users from migration 013
('michael.richardson@imagefactory.local', 'Michael', 'Richardson', true, 'active', true),
('alice.johnson@imagefactory.local', 'Alice', 'Johnson', true, 'active', true),
('david.wilson@imagefactory.local', 'David', 'Wilson', true, 'active', true),
('eve.martinez@imagefactory.local', 'Eve', 'Martinez', true, 'active', true),
('frank.thompson@imagefactory.local', 'Frank', 'Thompson', true, 'active', true),
('grace.lee@imagefactory.local', 'Grace', 'Lee', true, 'active', true),
('carol.davis@imagefactory.local', 'Carol', 'Davis', true, 'active', true),
('bob.smith@imagefactory.local', 'Bob', 'Smith', true, 'active', true),
('sarah.mitchell@imagefactory.local', 'Sarah', 'Mitchell', true, 'active', true),
('mark.anderson@imagefactory.local', 'Mark', 'Anderson', true, 'active', true),
('jennifer.chang@imagefactory.local', 'Jennifer', 'Chang', true, 'active', true),
('lisa.taylor@imagefactory.local', 'Lisa', 'Taylor', true, 'active', true),
('thomas.brown@imagefactory.local', 'Thomas', 'Brown', true, 'active', true)
ON CONFLICT (email) DO NOTHING;

-- ============================================================================
-- SEED PERMISSIONS (from migration 011)
-- ============================================================================
-- System-level permissions (true): users, roles, permissions, org_units, companies, security, api_keys, system
-- Tenant-level permissions (false): tenant, projects, builds, images, deployments, environments, registries
-- ============================================================================
INSERT INTO permissions (resource, action, category, is_system_permission) VALUES
-- System-level user management
('users', 'create', 'user_management', true),
('users', 'read', 'user_management', true),
('users', 'update', 'user_management', true),
('users', 'delete', 'user_management', true),
('users', 'activate', 'user_management', true),
('users', 'deactivate', 'user_management', true),
('users', 'reset_password', 'user_management', true),
('users', 'manage_mfa', 'user_management', true),
-- System-level access control
('roles', 'create', 'access_control', true),
('roles', 'read', 'access_control', true),
('roles', 'update', 'access_control', true),
('roles', 'delete', 'access_control', true),
('roles', 'manage_permissions', 'access_control', true),
('permissions', 'read', 'access_control', true),
('permissions', 'assign', 'access_control', true),
('permissions', 'manage', 'access_control', true),
-- System-level organization
('org_units', 'create', 'organization', true),
('org_units', 'read', 'organization', true),
('org_units', 'update', 'organization', true),
('org_units', 'delete', 'organization', true),
-- System-level company management
('companies', 'read', 'company_management', true),
('companies', 'update', 'company_management', true),
('companies', 'manage_subscription', 'company_management', true),
-- Tenant-level project management (scoped to tenant)
('projects', 'create', 'project_management', false),
('projects', 'read', 'project_management', false),
('projects', 'update', 'project_management', false),
('projects', 'delete', 'project_management', false),
('projects', 'archive', 'project_management', false),
('projects', 'add_member', 'project_management', false),
('projects', 'remove_member', 'project_management', false),
('projects', 'view_members', 'project_management', false),
('projects', 'update_member_role', 'project_management', false),
-- Tenant-level build management (scoped to tenant)
('builds', 'create', 'build_management', false),
('builds', 'read', 'build_management', false),
('builds', 'cancel', 'build_management', false),
('builds', 'retry', 'build_management', false),
('builds', 'delete', 'build_management', false),
-- Tenant-level image management (scoped to tenant)
('images', 'read', 'image_management', false),
('images', 'delete', 'image_management', false),
('images', 'scan', 'image_management', false),
('images', 'sign', 'image_management', false),
-- Tenant-level registry management (scoped to tenant)
('registries', 'create', 'registry_management', false),
('registries', 'read', 'registry_management', false),
('registries', 'update', 'registry_management', false),
('registries', 'delete', 'registry_management', false),
-- Tenant-level deployment management (scoped to tenant)
('deployments', 'create', 'deployment_management', false),
('deployments', 'read', 'deployment_management', false),
('deployments', 'execute', 'deployment_management', false),
('deployments', 'rollback', 'deployment_management', false),
-- Tenant-level environment management (scoped to tenant)
('environments', 'create', 'environment_management', false),
('environments', 'read', 'environment_management', false),
('environments', 'update', 'environment_management', false),
('environments', 'delete', 'environment_management', false),
-- System-level security management
('security', 'read', 'security', true),
('security', 'manage_keys', 'security', true),
('security', 'manage_policies', 'security', true),
('security', 'view_audit_logs', 'security', true),
-- System-level API key management
('api_keys', 'create', 'api_management', true),
('api_keys', 'read', 'api_management', true),
('api_keys', 'revoke', 'api_management', true),
-- System-level configuration management
('system', 'read_config', 'system_management', true),
('system', 'update_config', 'system_management', true),
('system', 'manage_workers', 'system_management', true),
('system', 'read', 'system_management', true),
('system', 'manage_config', 'system_management', true),
-- Tenant-level infrastructure management (scoped to tenant)
('infrastructure', 'read', 'infrastructure_management', false),
('infrastructure', 'create', 'infrastructure_management', false),
('infrastructure', 'update', 'infrastructure_management', false),
('infrastructure', 'delete', 'infrastructure_management', false),
('infrastructure', 'configure', 'infrastructure_management', false),
('infrastructure', 'select', 'infrastructure_management', false)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED DEPLOYMENT ENVIRONMENTS (from migration 011)
-- ============================================================================
INSERT INTO deployment_environments (company_id, name, display_name, environment_type, infrastructure_provider, requires_approval, status)
SELECT c.id, 'development', 'Development', 'development', 'kubernetes', false, 'active' FROM companies c WHERE c.name = 'ImageFactory'
ON CONFLICT (company_id, name) DO NOTHING;

INSERT INTO deployment_environments (company_id, name, display_name, environment_type, infrastructure_provider, requires_approval, status)
SELECT c.id, 'staging', 'Staging', 'staging', 'kubernetes', true, 'active' FROM companies c WHERE c.name = 'ImageFactory'
ON CONFLICT (company_id, name) DO NOTHING;

INSERT INTO deployment_environments (company_id, name, display_name, environment_type, infrastructure_provider, requires_approval, status)
SELECT c.id, 'production', 'Production', 'production', 'kubernetes', true, 'active' FROM companies c WHERE c.name = 'ImageFactory'
ON CONFLICT (company_id, name) DO NOTHING;

-- ============================================================================
-- SEED SYSTEM TENANT
-- ============================================================================
INSERT INTO tenants (id, tenant_code, name, slug, description, status)
VALUES ('00000000-0000-0000-0000-000000000000', 'SYS', 'System', 'system', 'System tenant for global administration', 'active')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SEED SYSTEM ADMIN TENANT (from migration 018)
-- ============================================================================
INSERT INTO tenants (tenant_code, name, slug, description, status)
VALUES ('sysadmin', 'System Administrators', 'system-admin', 'System administrator group tenant', 'active')
ON CONFLICT (tenant_code) DO NOTHING;

-- ============================================================================
-- SEED SYSTEM ADMIN GROUP (from migrations 012, 018)
-- ============================================================================
INSERT INTO tenant_groups (tenant_id, name, slug, description, role_type, is_system_group, status)
SELECT t.id, 'System Administrators', 'system-admin', 'System administration group', 'system_administrator', true, 'active'
FROM tenants t
WHERE t.tenant_code = 'sysadmin'
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- ============================================================================
-- ASSIGN ADMIN USER TO SYSTEM ADMIN GROUP (from migrations 012, 018)
-- ============================================================================
INSERT INTO group_members (group_id, user_id, is_group_admin, added_at)
SELECT tg.id, u.id, true, CURRENT_TIMESTAMP
FROM tenant_groups tg, users u
WHERE tg.role_type = 'system_administrator'
  AND u.email = 'admin@imagefactory.local'
ON CONFLICT (group_id, user_id) DO NOTHING;

-- ============================================================================
-- SEED EXTERNAL TENANTS (from migration 015)
-- ============================================================================
INSERT INTO external_tenants (tenant_id, name, slug, description, contact_email, industry, country) VALUES
('10001', 'Engineering Team', 'engineering-team', 'Core engineering and development team', 'eng@imagefactory.local', 'Technology', 'US'),
('10002', 'DevOps Team', 'devops-team', 'Infrastructure and deployment operations', 'devops@imagefactory.local', 'Technology', 'US'),
('10003', 'Product Team', 'product-team', 'Product management and strategy', 'product@imagefactory.local', 'Technology', 'US'),
('10004', 'QA & Testing Team', 'qa-testing-team', 'Quality assurance and testing', 'qa@imagefactory.local', 'Technology', 'US'),
('10005', 'Frontend Team', 'frontend-team', 'User interface and frontend development', 'frontend@imagefactory.local', 'Technology', 'US'),
('10006', 'Backend Team', 'backend-team', 'Backend services and APIs', 'backend@imagefactory.local', 'Technology', 'US'),
('10007', 'Data Science Team', 'data-science-team', 'Analytics and machine learning', 'datasci@imagefactory.local', 'Technology', 'US'),
('10008', 'Security Team', 'security-team', 'Information security and compliance', 'security@imagefactory.local', 'Technology', 'US'),
('10009', 'Platform Team', 'platform-team', 'Internal platform and tools', 'platform@imagefactory.local', 'Technology', 'US'),
('10010', 'Finance Team', 'finance-team', 'Financial planning and operations', 'finance@imagefactory.local', 'Finance', 'US'),
('10011', 'Human Resources', 'human-resources', 'HR and talent management', 'hr@imagefactory.local', 'Administration', 'US'),
('10012', 'Sales Team', 'sales-team', 'Enterprise sales and business development', 'sales@imagefactory.local', 'Sales', 'US'),
('10013', 'Marketing Team', 'marketing-team', 'Marketing and communications', 'marketing@imagefactory.local', 'Marketing', 'US'),
('10014', 'Customer Success', 'customer-success', 'Customer support and success', 'support@imagefactory.local', 'Support', 'US'),
('10015', 'Legal Team', 'legal-team', 'Legal and compliance', 'legal@imagefactory.local', 'Legal', 'US'),
('10016', 'Infrastructure Team', 'infrastructure-team', 'Cloud and infrastructure management', 'infra@imagefactory.local', 'Technology', 'US'),
('10017', 'Mobile Team', 'mobile-team', 'Mobile app development', 'mobile@imagefactory.local', 'Technology', 'US'),
('10018', 'API Team', 'api-team', 'API design and integration', 'api@imagefactory.local', 'Technology', 'US'),
('10019', 'Database Team', 'database-team', 'Database administration and optimization', 'database@imagefactory.local', 'Technology', 'US'),
('10020', 'Research Team', 'research-team', 'Research and innovation', 'research@imagefactory.local', 'Technology', 'US')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED SYSTEM CONFIGS (from migration 026)
-- ============================================================================
-- Global tool availability configuration
-- NOTE: admin user ID is a7c8e1f2-4b5e-4a7f-8c2a-1b3d4e5f6g7h (placeholder, will be replaced by actual ID in database)
-- Using the pattern: INSERT with admin user as created_by/updated_by
INSERT INTO system_configs (
  tenant_id, 
  config_type,
  config_key, 
  config_value, 
  status,
  description,
  created_by,
  updated_by
)
SELECT 
  NULL,
  'build',
  'tool_availability',
  '{"build_methods":{"container":true,"packer":true,"paketo":true,"kaniko":true,"buildx":true,"nix":true},"sbom_tools":{"syft":true,"grype":true,"trivy":true},"scan_tools":{"trivy":true,"clair":false,"grype":true,"snyk":false},"registry_types":{"s3":true,"harbor":false,"quay":false,"artifactory":false},"secret_managers":{"vault":false,"aws_secretsmanager":true,"azure_keyvault":false,"gcp_secretmanager":false}}',
  'active',
  'Global tool availability configuration',
  u.id,
  u.id
FROM users u
WHERE u.email = 'admin@imagefactory.local'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED RBAC ROLES (from migration 022)
-- ============================================================================
INSERT INTO rbac_roles (tenant_id, name, description, is_system, created_at, updated_at) VALUES
(NULL, 'System Administrator', 'Full system access with all permissions', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(NULL, 'Owner', 'Owner-level access for tenant management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(NULL, 'Developer', 'Developer access for building and deploying', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
(NULL, 'Viewer', 'Read-only access for monitoring', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- ============================================================================
-- SEED PERMISSIONS (from migration 022)
-- ============================================================================
INSERT INTO permissions (resource, action, description, category, is_system_permission, created_at, updated_at) VALUES
-- System (6)
('system', 'read', 'Read system information', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'view_status', 'View system status', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'manage_config', 'Manage system configuration', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'run_maintenance', 'Run system maintenance', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'manage_certificates', 'Manage SSL certificates', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('system', 'view_logs', 'View system logs', 'system', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Tenant (7)
('tenant', 'create', 'Create new tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'read', 'View tenant details', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'list', 'List all tenants', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'update', 'Update tenant settings', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'delete', 'Delete tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'suspend', 'Suspend tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'activate', 'Activate tenant', 'tenant_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('tenant', 'manage_users', 'Manage tenant users and roles', 'tenant_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- User (9)
('user', 'create', 'Create new user', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'read', 'View user details', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'list', 'List users', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'update', 'Update user information', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'delete', 'Delete user', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'suspend', 'Suspend user account', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'activate', 'Activate user account', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'reset_password', 'Reset user password', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('user', 'manage_roles', 'Manage user roles', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Role (7)
('role', 'create', 'Create new role', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'read', 'View role details', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'list', 'List roles', 'access_control', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'update', 'Update role settings', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'delete', 'Delete role', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'manage_permissions', 'Manage role permissions', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('role', 'assign_users', 'Assign users to roles', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Build (9)
('build', 'create', 'Create new build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'read', 'View build details', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'list', 'List builds', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'cancel', 'Cancel running build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'view_logs', 'View build logs', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'retry', 'Retry failed build', 'build_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'delete', 'Delete build', 'build_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('build', 'manage_triggers', 'Manage build triggers', 'build_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Audit (4)
('audit', 'read', 'Read audit logs', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('audit', 'list', 'List audit events', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('audit', 'export', 'Export audit logs', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('audit', 'manage_retention', 'Manage audit retention', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Registry (4)
('registry', 'read', 'Read registry information', 'registry_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'list', 'List registry items', 'registry_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'create', 'Create registry', 'registry_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'update', 'Update registry settings', 'registry_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('registry', 'delete', 'Delete registry', 'registry_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Image (6)
('image', 'read', 'Read image information', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'list', 'List images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'push', 'Push images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'pull', 'Pull images', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'delete', 'Delete images', 'image_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('image', 'scan', 'Scan images for vulnerabilities', 'image_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Container (5)
('container', 'read', 'Read container information', 'container_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('container', 'list', 'List containers', 'container_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('container', 'start', 'Start containers', 'container_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('container', 'stop', 'Stop containers', 'container_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('container', 'delete', 'Delete containers', 'container_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Deployment (5)
('deployment', 'read', 'Read deployment information', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'list', 'List deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'create', 'Create deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'update', 'Update deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'delete', 'Delete deployments', 'deployment_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('deployment', 'rollback', 'Rollback deployments', 'deployment_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Settings (2)
('settings', 'read', 'Read system settings', 'system_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('settings', 'update', 'Update system settings', 'system_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('settings', 'manage_secrets', 'Manage system secrets', 'system_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Admin (2)
('admin', 'access', 'Access admin panel', 'administration', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('admin', 'manage_system', 'Manage system settings', 'administration', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Invitation (4)
('invitation', 'create', 'Create invitations', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('invitation', 'read', 'Read invitations', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('invitation', 'list', 'List invitations', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('invitation', 'delete', 'Delete invitations', 'user_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('invitation', 'accept', 'Accept invitations', 'user_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Permissions (1)
('permissions', 'manage', 'Manage permissions system', 'access_control', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- SSO (2)
('sso', 'read', 'Read SSO configuration', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('sso', 'write', 'Write SSO configuration', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- MFA (2)
('mfa', 'read', 'Read MFA settings', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('mfa', 'write', 'Write MFA settings', 'security', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Groups (4)
('groups', 'list', 'List groups', 'organization', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('groups', 'read', 'Read group details', 'organization', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('groups', 'view_members', 'View group members', 'organization', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('groups', 'manage_members', 'Manage group membership', 'organization', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
-- Projects (7)
('projects', 'create', 'Create new project', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'read', 'View project details', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'update', 'Update project settings', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'delete', 'Delete project', 'project_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'archive', 'Archive project', 'project_management', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'add_member', 'Add project member', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'remove_member', 'Remove project member', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'view_members', 'View project members', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('projects', 'update_member_role', 'Update member role', 'project_management', false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (resource, action) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO SYSTEM ADMINISTRATOR ROLE (from migration 022)
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
-- ASSIGN PERMISSIONS TO OWNER ROLE (from migration 022)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Owner' AND r.is_system = true AND r.tenant_id IS NULL
  AND (p.resource IN ('tenant', 'user', 'build', 'deployment', 'image', 'registry', 'container', 'notification', 'settings', 'invitation', 'groups', 'projects', 'role')
       OR (p.resource = 'audit' AND p.action IN ('read', 'list')))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO DEVELOPER ROLE (from migration 022)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Developer' AND r.is_system = true AND r.tenant_id IS NULL
  AND ((p.resource = 'build' AND p.action IN ('list', 'read', 'create', 'cancel', 'view_logs', 'retry'))
       OR (p.resource = 'image' AND p.action IN ('list', 'read'))
       OR (p.resource = 'projects' AND p.action IN ('read', 'view_members'))
       OR (p.resource = 'tenant' AND p.action IN ('read', 'list')))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO OPERATOR ROLE (from migration 022)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Operator' AND r.is_system = true AND r.tenant_id IS NULL
  AND ((p.resource = 'build' AND p.action IN ('list', 'read', 'view_logs', 'cancel'))
       OR (p.resource = 'projects' AND p.action IN ('read', 'view_members'))
       OR (p.resource = 'tenant' AND p.action IN ('read', 'list')))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN PERMISSIONS TO VIEWER ROLE (from migration 022)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Viewer' AND r.is_system = true AND r.tenant_id IS NULL
  AND ((p.resource = 'build' AND p.action IN ('list', 'read', 'view_logs'))
       OR (p.resource = 'projects' AND p.action IN ('read', 'view_members'))
       OR (p.resource = 'tenant' AND p.action IN ('read', 'list')))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- ASSIGN ADMIN USER TO SYSTEM ADMINISTRATOR ROLE (from migration 022)
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

-- ============================================================================
-- SEED SYSTEM CONFIGURATIONS (from migration 023)
-- ============================================================================
-- Insert default security configurations for all existing tenants
INSERT INTO system_configs (tenant_id, config_type, config_key, config_value, description, is_default, created_by, updated_by)
SELECT
    t.id as tenant_id,
    'security' as config_type,
    config_key,
    config_value::jsonb,
    defaults.description,
    true as is_default,
    (SELECT id FROM users WHERE tenant_id = t.id LIMIT 1) as created_by,
    (SELECT id FROM users WHERE tenant_id = t.id LIMIT 1) as updated_by
FROM tenants t
CROSS JOIN (
    VALUES
        ('jwt_expiration_hours', '24', 'JWT access token expiration in hours'),
        ('refresh_token_hours', '168', 'Refresh token expiration in hours (7 days)'),
        ('max_login_attempts', '5', 'Maximum failed login attempts before lockout'),
        ('account_lock_duration_minutes', '30', 'Account lock duration in minutes'),
        ('password_min_length', '8', 'Minimum password length'),
        ('require_special_chars', 'true', 'Require special characters in passwords'),
        ('require_numbers', 'true', 'Require numbers in passwords'),
        ('require_uppercase', 'true', 'Require uppercase letters in passwords'),
        ('session_timeout_minutes', '60', 'Session timeout in minutes')
) AS defaults(config_key, config_value, description)
WHERE EXISTS (SELECT 1 FROM user_role_assignments ura WHERE ura.tenant_id = t.id)
ON CONFLICT (tenant_id, config_type, config_key) DO NOTHING;

-- Insert default build configurations for all existing tenants
INSERT INTO system_configs (tenant_id, config_type, config_key, config_value, description, is_default, created_by, updated_by)
SELECT
    t.id as tenant_id,
    'build' as config_type,
    config_key,
    (config_value::integer)::text::jsonb,
    defaults.description,
    true as is_default,
    (SELECT id FROM users WHERE tenant_id = t.id LIMIT 1) as created_by,
    (SELECT id FROM users WHERE tenant_id = t.id LIMIT 1) as updated_by
FROM tenants t
CROSS JOIN (
    VALUES
        ('default_timeout_minutes', '30', 'Default build timeout in minutes'),
        ('max_concurrent_jobs', '10', 'Maximum concurrent build jobs'),
        ('worker_pool_size', '5', 'Worker pool size for builds'),
        ('max_queue_size', '100', 'Maximum build queue size'),
        ('artifact_retention_days', '30', 'Build artifact retention in days')
) AS defaults(config_key, config_value, description)
WHERE EXISTS (SELECT 1 FROM users u WHERE u.tenant_id = t.id)
ON CONFLICT (tenant_id, config_type, config_key) DO NOTHING;

-- ============================================================================
-- ADD PROJECT PERMISSIONS TO DEVELOPER ROLE (from migration 028)
-- ============================================================================
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r, permissions p
WHERE r.name = 'Developer' AND r.is_system = true AND r.tenant_id IS NULL
  AND p.resource = 'projects' AND p.action IN ('read', 'create')
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================================
-- BACKFILL PROJECT MEMBERS (from migration 029)
-- ============================================================================
-- Add all existing tenant users to all projects in their tenant
-- This ensures existing data works with the new project scope system
-- Only adds if user is not already in project_members

INSERT INTO project_members (project_id, user_id, role_id, assigned_by_user_id, created_at)
SELECT 
    p.id,
    ura.user_id,
    ura.role_id,
    (SELECT id FROM users WHERE email ILIKE '%system%' LIMIT 1),
    CURRENT_TIMESTAMP
FROM projects p
INNER JOIN tenants t ON p.tenant_id = t.id
INNER JOIN user_role_assignments ura ON ura.tenant_id = t.id
WHERE NOT EXISTS (
    SELECT 1 FROM project_members pm 
    WHERE pm.project_id = p.id AND pm.user_id = ura.user_id
)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- ============================================================================
-- SEED BUILD POLICIES (default system policies)
-- ============================================================================
-- Default build policies for the system tenant
INSERT INTO build_policies (tenant_id, policy_type, policy_key, policy_value, description, is_active) VALUES
-- Resource Limits
('00000000-0000-0000-0000-000000000000', 'resource_limit', 'max_build_duration', '{"value": 2, "unit": "hours"}', 'Maximum duration a single build can run', true),
('00000000-0000-0000-0000-000000000000', 'resource_limit', 'concurrent_builds_per_tenant', '{"value": 5}', 'Maximum simultaneous builds allowed per tenant', true),
('00000000-0000-0000-0000-000000000000', 'resource_limit', 'storage_quota_per_build', '{"value": 10, "unit": "GB"}', 'Maximum disk space a build can use', true),

-- Scheduling Rules
('00000000-0000-0000-0000-000000000000', 'scheduling_rule', 'maintenance_windows', '{"schedule": "weekends 2-4 AM", "timezone": "UTC"}', 'Scheduled maintenance periods when builds are paused', true),
('00000000-0000-0000-0000-000000000000', 'scheduling_rule', 'priority_queuing', '{"algorithm": "priority-based", "levels": ["low", "normal", "high", "urgent"]}', 'How builds are prioritized in the queue', true),

-- Approval Workflows
('00000000-0000-0000-0000-000000000000', 'approval_workflow', 'approval_required', '{"enabled": false, "conditions": ["production_deployment", "privileged_access"]}', 'Whether builds require approval before execution', true),
('00000000-0000-0000-0000-000000000000', 'approval_workflow', 'auto_approval_threshold', '{"max_duration": "1h", "max_resources": "medium"}', 'Criteria for automatic approval of builds', true)
ON CONFLICT (tenant_id, policy_key) DO NOTHING;

-- ============================================================================
-- SEED INFRASTRUCTURE NODES (for development/demo)
-- ============================================================================
-- Add sample build nodes for development and testing
INSERT INTO infrastructure_nodes (id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, last_heartbeat, maintenance_mode, labels) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'build-node-01', 'ready', 8, 16, 100, NOW(), false, '{"type": "build", "region": "us-west", "environment": "development"}'),
('550e8400-e29b-41d4-a716-446655440002', 'build-node-02', 'ready', 4, 8, 50, NOW(), false, '{"type": "build", "region": "us-east", "environment": "development"}'),
('550e8400-e29b-41d4-a716-446655440003', 'build-node-03', 'maintenance', 16, 32, 200, NOW() - interval '1 hour', true, '{"type": "build", "region": "eu-west", "environment": "development"}')
ON CONFLICT (id) DO NOTHING;

