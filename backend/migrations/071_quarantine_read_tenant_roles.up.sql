-- Grant tenant roles read access to quarantine request state so
-- tenant dashboards can list import request history.
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r
JOIN permissions p ON p.resource = 'quarantine' AND p.action = 'read'
WHERE r.tenant_id IS NULL
  AND r.is_system = true
  AND r.name IN ('Owner', 'Developer', 'Viewer')
ON CONFLICT (role_id, permission_id) DO NOTHING;
