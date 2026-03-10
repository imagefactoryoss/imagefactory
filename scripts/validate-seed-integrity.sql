-- ============================================================================
-- Seed Integrity Validation
-- ============================================================================
-- Fails fast if essential seeded relationships are broken.
-- Run after:
--   1) schema migrations
--   2) seed-essential-data.sql
--   3) essential-config-seeder (system bootstrap)
-- ============================================================================

DO $$
DECLARE
    v_admin_user_id UUID;
    v_sysadmin_tenant_id UUID;
    v_sysadmin_group_id UUID;
    v_missing_required_admin_permissions INTEGER;
BEGIN
    -- ------------------------------------------------------------------------
    -- Core principal checks
    -- ------------------------------------------------------------------------
    SELECT id
      INTO v_admin_user_id
      FROM users
     WHERE email = 'admin@imagefactory.local'
     LIMIT 1;

    IF v_admin_user_id IS NULL THEN
        RAISE EXCEPTION 'Seed integrity failure: admin user missing (admin@imagefactory.local)';
    END IF;

    SELECT id
      INTO v_sysadmin_tenant_id
      FROM tenants
     WHERE tenant_code = 'sysadmin'
     LIMIT 1;

    IF v_sysadmin_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Seed integrity failure: sysadmin tenant missing (tenant_code=sysadmin)';
    END IF;

    SELECT tg.id
      INTO v_sysadmin_group_id
      FROM tenant_groups tg
     WHERE tg.tenant_id = v_sysadmin_tenant_id
       AND tg.role_type = 'system_administrator'
     LIMIT 1;

    IF v_sysadmin_group_id IS NULL THEN
        RAISE EXCEPTION 'Seed integrity failure: system_administrator tenant group missing for sysadmin tenant';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM group_members gm
         WHERE gm.group_id = v_sysadmin_group_id
           AND gm.user_id = v_admin_user_id
    ) THEN
        RAISE EXCEPTION 'Seed integrity failure: admin user is not member of sysadmin system administrator group';
    END IF;

    -- ------------------------------------------------------------------------
    -- Required RBAC roles and permissions
    -- ------------------------------------------------------------------------
    IF NOT EXISTS (SELECT 1 FROM rbac_roles WHERE tenant_id IS NULL AND is_system = true AND name = 'System Administrator') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing system role System Administrator';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM rbac_roles WHERE tenant_id IS NULL AND is_system = true AND name = 'Owner') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing system role Owner';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM rbac_roles WHERE tenant_id IS NULL AND is_system = true AND name = 'Developer') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing system role Developer';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM rbac_roles WHERE tenant_id IS NULL AND is_system = true AND name = 'Viewer') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing system role Viewer';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'tenant' AND action = 'list') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing permission tenant:list';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'user' AND action = 'list') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing permission user:list';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'image' AND action = 'list') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing permission image:list';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'build' AND action = 'create') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing permission build:create';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'system' AND action = 'manage_config') THEN
        RAISE EXCEPTION 'Seed integrity failure: missing permission system:manage_config';
    END IF;

    -- System Administrator role must include all required core admin permissions.
    SELECT COUNT(*)
      INTO v_missing_required_admin_permissions
      FROM (
            VALUES
                ('system', 'read'),
                ('system', 'manage_config'),
                ('tenant', 'create'),
                ('tenant', 'list'),
                ('user', 'create'),
                ('user', 'list'),
                ('role', 'manage_permissions'),
                ('permissions', 'manage'),
                ('build', 'create'),
                ('image', 'update'),
                ('infrastructure', 'select')
      ) AS required(resource, action)
     WHERE NOT EXISTS (
            SELECT 1
              FROM permissions p
              JOIN role_permissions rp ON rp.permission_id = p.id
              JOIN rbac_roles r ON r.id = rp.role_id
             WHERE p.resource = required.resource
               AND p.action = required.action
               AND r.tenant_id IS NULL
               AND r.is_system = true
               AND r.name = 'System Administrator'
      );

    IF v_missing_required_admin_permissions > 0 THEN
        RAISE EXCEPTION 'Seed integrity failure: System Administrator role missing % required core permissions',
            v_missing_required_admin_permissions;
    END IF;

    -- Other core roles must have at least one permission.
    IF NOT EXISTS (
        SELECT 1
          FROM role_permissions rp
          JOIN rbac_roles r ON r.id = rp.role_id
         WHERE r.tenant_id IS NULL
           AND r.is_system = true
           AND r.name = 'Owner'
    ) THEN
        RAISE EXCEPTION 'Seed integrity failure: Owner role has no role_permissions';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM role_permissions rp
          JOIN rbac_roles r ON r.id = rp.role_id
         WHERE r.tenant_id IS NULL
           AND r.is_system = true
           AND r.name = 'Developer'
    ) THEN
        RAISE EXCEPTION 'Seed integrity failure: Developer role has no role_permissions';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM role_permissions rp
          JOIN rbac_roles r ON r.id = rp.role_id
         WHERE r.tenant_id IS NULL
           AND r.is_system = true
           AND r.name = 'Viewer'
    ) THEN
        RAISE EXCEPTION 'Seed integrity failure: Viewer role has no role_permissions';
    END IF;

    -- Admin user must be assigned System Administrator role.
    IF NOT EXISTS (
        SELECT 1
          FROM user_role_assignments ura
          JOIN rbac_roles r ON r.id = ura.role_id
         WHERE ura.user_id = v_admin_user_id
           AND r.tenant_id IS NULL
           AND r.is_system = true
           AND r.name = 'System Administrator'
    ) THEN
        RAISE EXCEPTION 'Seed integrity failure: admin user missing System Administrator role assignment';
    END IF;

    RAISE NOTICE 'Seed integrity validation passed.';
END $$;
