-- Validation script to verify role permissions are correctly assigned
-- This helps ensure all roles have the permissions they need

SELECT 'ROLE PERMISSION SUMMARY' as validation;
SELECT '';

-- Check all roles and their permission counts
SELECT 
    r.name as role_name,
    COUNT(rp.permission_id) as permission_count,
    STRING_AGG(DISTINCT p.resource || ':' || p.action, ', ' ORDER BY p.resource || ':' || p.action) as permissions
FROM rbac_roles r
LEFT JOIN role_permissions rp ON r.id = rp.role_id
LEFT JOIN permissions p ON rp.permission_id = p.id
WHERE r.tenant_id IS NULL
GROUP BY r.id, r.name
ORDER BY r.name;

SELECT '';
SELECT 'TENANT RESOURCE PERMISSIONS' as validation;
SELECT '';

-- Check which roles have tenant resource permissions
SELECT 
    r.name as role_name,
    p.action,
    p.is_system_permission,
    p.category
FROM rbac_roles r
INNER JOIN role_permissions rp ON r.id = rp.role_id
INNER JOIN permissions p ON rp.permission_id = p.id
WHERE r.tenant_id IS NULL
AND p.resource = 'tenant'
ORDER BY r.name, p.action;

SELECT '';
SELECT 'MISSING TENANT PERMISSIONS' as validation;
SELECT '';

-- Check if any role should have tenant access but doesn't
WITH roles_without_tenant AS (
    SELECT DISTINCT r.id, r.name
    FROM rbac_roles r
    WHERE r.tenant_id IS NULL
    AND r.name IN ('Owner', 'Developer', 'Operator', 'Viewer')
    AND NOT EXISTS (
        SELECT 1 FROM role_permissions rp
        INNER JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = r.id AND p.resource = 'tenant'
    )
)
SELECT name as role_name, 'Missing tenant permissions' as issue
FROM roles_without_tenant;

SELECT '';
SELECT 'GROUP-BASED USERS AND THEIR PERMISSIONS' as validation;
SELECT '';

-- Check Alice and other group-based users
SELECT 
    u.email,
    STRING_AGG(DISTINCT tg.role_type, ', ') as group_roles,
    STRING_AGG(DISTINCT p.resource || ':' || p.action, ', ' ORDER BY p.resource || ':' || p.action) as permissions
FROM users u
INNER JOIN group_members gm ON u.id = gm.user_id
INNER JOIN tenant_groups tg ON gm.group_id = tg.id
INNER JOIN rbac_roles r ON LOWER(r.name) = tg.role_type AND r.tenant_id IS NULL
INNER JOIN role_permissions rp ON r.id = rp.role_id
INNER JOIN permissions p ON rp.permission_id = p.id
WHERE gm.removed_at IS NULL
AND tg.status = 'active'
GROUP BY u.id, u.email
ORDER BY u.email;

SELECT '';
SELECT 'PERMISSION CHECK FOR ALICE (OWNER ROLE)' as validation;
SELECT '';

-- Check if Alice has tenant:list permission
SELECT 
    u.email,
    p.resource,
    p.action,
    p.is_system_permission,
    EXISTS(
        SELECT 1 FROM role_permissions rp
        WHERE rp.role_id = (SELECT id FROM rbac_roles WHERE LOWER(name) = 'owner' AND tenant_id IS NULL)
        AND rp.permission_id = p.id
    ) as role_has_permission
FROM users u
CROSS JOIN permissions p
WHERE u.email = 'alice.johnson@imagefactory.local'
AND p.resource = 'tenant'
AND p.action IN ('list', 'read')
ORDER BY u.email, p.action;
