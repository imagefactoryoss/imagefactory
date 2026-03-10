-- Add release permission for quarantine workflow governance.
INSERT INTO permissions (resource, action, description, category, is_system_permission, created_at, updated_at)
VALUES (
	'quarantine',
	'release',
	'Release quarantine-approved artifacts for tenant consumption',
	'quarantine_management',
	true,
	CURRENT_TIMESTAMP,
	CURRENT_TIMESTAMP
)
ON CONFLICT (resource, action) DO NOTHING;

-- Grant release permission to system security reviewers.
INSERT INTO role_permissions (role_id, permission_id, granted_at)
SELECT r.id, p.id, CURRENT_TIMESTAMP
FROM rbac_roles r
JOIN permissions p ON p.resource = 'quarantine' AND p.action = 'release'
WHERE r.name = 'Security Reviewer'
  AND r.is_system = true
  AND r.tenant_id IS NULL
ON CONFLICT (role_id, permission_id) DO NOTHING;
