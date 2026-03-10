DROP INDEX IF EXISTS idx_external_image_imports_tenant_request_type_created_at;

ALTER TABLE external_image_imports
    DROP CONSTRAINT IF EXISTS chk_external_image_import_request_type;

ALTER TABLE external_image_imports
    DROP COLUMN IF EXISTS request_type;
