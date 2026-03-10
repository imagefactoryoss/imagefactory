-- Database Schema Audit - Comprehensive Integrity Check
-- This script checks for missing foreign keys, orphaned records, and cascade delete issues

-- 1. Check for orphaned user_role_assignments
SELECT 'ORPHAN: user_role_assignments -> users' AS issue, 
       COUNT(*) as orphaned_count
FROM user_role_assignments ura
WHERE ura.user_id NOT IN (SELECT id FROM users);

-- 2. Check role_permissions orphans
SELECT 'ORPHAN: role_permissions -> rbac_roles' AS issue, 
       COUNT(*) as orphaned_count
FROM role_permissions rp
WHERE rp.role_id NOT IN (SELECT id FROM rbac_roles);

-- 3. Check group_members orphans by user
SELECT 'ORPHAN: group_members -> users' AS issue,
       COUNT(*) as orphaned_count
FROM group_members gm
WHERE gm.user_id NOT IN (SELECT id FROM users);

-- 4. Check group_members orphans by group
SELECT 'ORPHAN: group_members -> tenant_groups' AS issue,
       COUNT(*) as orphaned_count
FROM group_members gm
WHERE gm.group_id NOT IN (SELECT id FROM tenant_groups);

-- 5. Check project_members orphans
SELECT 'ORPHAN: project_members -> users' AS issue,
       COUNT(*) as orphaned_count
FROM project_members pm
WHERE pm.user_id NOT IN (SELECT id FROM users);

-- 6. Detailed cascade rules analysis
SELECT 
    constraint_name,
    table_name,
    column_name,
    foreign_table_name,
    foreign_column_name,
    delete_rule,
    update_rule
FROM (
    SELECT
        rc.constraint_name,
        kcu.table_name,
        kcu.column_name,
        ccu.table_name AS foreign_table_name,
        ccu.column_name AS foreign_column_name,
        rc.delete_rule,
        rc.update_rule
    FROM 
        information_schema.referential_constraints rc
    JOIN 
        information_schema.key_column_usage kcu 
        ON rc.constraint_name = kcu.constraint_name 
        AND kcu.table_schema = 'public'
    JOIN 
        information_schema.constraint_column_usage ccu 
        ON rc.unique_constraint_name = ccu.constraint_name
        AND ccu.table_schema = 'public'
) AS fks
WHERE 
    table_name IN (
        'user_role_assignments', 'group_members', 'project_members', 
        'role_permissions', 'user_invitations', 'builds', 'deployments',
        'audit_events', 'api_keys', 'email_templates', 'container_registries'
    )
ORDER BY 
    table_name, delete_rule DESC;

-- 7. Check for tables with potential missing cascade deletes
SELECT 
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS referenced_table,
    rc.delete_rule
FROM 
    information_schema.table_constraints tc
JOIN 
    information_schema.key_column_usage kcu 
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = 'public'
JOIN 
    information_schema.constraint_column_usage ccu 
    ON ccu.constraint_name = tc.constraint_name
    AND ccu.table_schema = 'public'
JOIN 
    information_schema.referential_constraints rc
    ON rc.constraint_name = tc.constraint_name
WHERE 
    tc.constraint_type = 'FOREIGN KEY'
    AND rc.delete_rule != 'CASCADE'
    AND tc.table_name IN (
        'user_role_assignments', 'group_members', 'project_members', 
        'user_invitations', 'audit_events', 'builds'
    )
ORDER BY tc.table_name;
