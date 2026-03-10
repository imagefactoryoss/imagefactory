ALTER TABLE project_build_settings
    ADD COLUMN IF NOT EXISTS build_config_on_error TEXT NOT NULL DEFAULT 'strict';

ALTER TABLE project_build_settings
    DROP CONSTRAINT IF EXISTS chk_project_build_settings_on_error;

ALTER TABLE project_build_settings
    ADD CONSTRAINT chk_project_build_settings_on_error
        CHECK (build_config_on_error IN ('strict', 'fallback_to_ui'));
