DELETE FROM role_permissions rp
USING rbac_roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.is_system = true
  AND r.name IN ('Owner', 'Developer', 'Viewer')
  AND p.resource = 'quarantine'
  AND p.action = 'read';
