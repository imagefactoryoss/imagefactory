ALTER TABLE external_image_imports
    DROP COLUMN IF EXISTS pipeline_namespace,
    DROP COLUMN IF EXISTS pipeline_run_name;
