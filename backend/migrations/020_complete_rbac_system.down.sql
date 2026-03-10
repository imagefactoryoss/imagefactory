-- Consolidated rollback for RBAC system, permissions, user management, and templates

-- Remove notification templates
DELETE FROM notification_templates 
WHERE template_type IN ('user_invitation', 'user_added_to_group');

-- Remove default tenant
DELETE FROM tenants WHERE tenant_code = 'default';

-- Drop user management tables
DROP TABLE IF EXISTS bulk_operations;
DROP TABLE IF EXISTS user_invitations;
DROP TABLE IF EXISTS password_reset_tokens;

-- Remove role permissions and roles
DELETE FROM role_permissions WHERE 1=1;
DELETE FROM user_role_assignments WHERE 1=1;
DELETE FROM permissions WHERE 1=1;
DELETE FROM rbac_roles WHERE tenant_id IS NULL;

-- Drop RBAC tables
DROP TABLE IF EXISTS user_role_assignments;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS rbac_roles;
