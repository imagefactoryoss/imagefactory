ALTER TABLE external_image_imports
    ADD COLUMN IF NOT EXISTS request_type VARCHAR(32) NOT NULL DEFAULT 'quarantine';

UPDATE external_image_imports
SET request_type = 'quarantine'
WHERE request_type IS NULL OR request_type = '';

ALTER TABLE external_image_imports
    DROP CONSTRAINT IF EXISTS chk_external_image_import_request_type;

ALTER TABLE external_image_imports
    ADD CONSTRAINT chk_external_image_import_request_type
        CHECK (request_type IN ('quarantine', 'scan'));

CREATE INDEX IF NOT EXISTS idx_external_image_imports_tenant_request_type_created_at
    ON external_image_imports (tenant_id, request_type, created_at DESC);
