ALTER TABLE repository_auth
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE repository_auth ra
SET tenant_id = p.tenant_id
FROM projects p
WHERE p.id = ra.project_id;

ALTER TABLE repository_auth
    ALTER COLUMN tenant_id SET NOT NULL,
    ALTER COLUMN project_id DROP NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE table_schema = current_schema()
          AND table_name = 'repository_auth'
          AND constraint_name = 'fk_repository_auth_tenant_id'
    ) THEN
        ALTER TABLE repository_auth
            ADD CONSTRAINT fk_repository_auth_tenant_id
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

ALTER TABLE repository_auth
    DROP CONSTRAINT IF EXISTS repository_auth_project_id_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS uq_repository_auth_tenant_name
    ON repository_auth (tenant_id, name)
    WHERE project_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_repository_auth_project_name
    ON repository_auth (project_id, name)
    WHERE project_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_repository_auth_tenant_id ON repository_auth(tenant_id);
