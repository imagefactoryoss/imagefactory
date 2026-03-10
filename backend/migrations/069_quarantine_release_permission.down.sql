DELETE FROM role_permissions rp
USING permissions p
WHERE rp.permission_id = p.id
  AND p.resource = 'quarantine'
  AND p.action = 'release';

DELETE FROM permissions
WHERE resource = 'quarantine'
  AND action = 'release';
