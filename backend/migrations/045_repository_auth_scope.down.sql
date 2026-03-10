DROP INDEX IF EXISTS idx_repository_auth_tenant_id;
DROP INDEX IF EXISTS uq_repository_auth_project_name;
DROP INDEX IF EXISTS uq_repository_auth_tenant_name;

DELETE FROM repository_auth WHERE project_id IS NULL;

ALTER TABLE repository_auth
    DROP CONSTRAINT IF EXISTS fk_repository_auth_tenant_id,
    DROP COLUMN IF EXISTS tenant_id,
    ALTER COLUMN project_id SET NOT NULL;

ALTER TABLE repository_auth
    ADD CONSTRAINT repository_auth_project_id_name_key UNIQUE (project_id, name);
