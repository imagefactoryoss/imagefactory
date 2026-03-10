ALTER TABLE external_image_imports
    ADD COLUMN IF NOT EXISTS pipeline_run_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS pipeline_namespace VARCHAR(255);
