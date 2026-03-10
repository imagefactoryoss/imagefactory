ALTER TABLE project_build_settings
    DROP CONSTRAINT IF EXISTS chk_project_build_settings_on_error;

ALTER TABLE project_build_settings
    DROP COLUMN IF EXISTS build_config_on_error;
