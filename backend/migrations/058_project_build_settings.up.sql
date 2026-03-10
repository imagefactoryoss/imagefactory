CREATE TABLE IF NOT EXISTS project_build_settings (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    build_config_mode TEXT NOT NULL DEFAULT 'repo_managed',
    build_config_file TEXT NOT NULL DEFAULT 'image-factory.yaml',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_project_build_settings_mode
        CHECK (build_config_mode IN ('ui_managed', 'repo_managed')),
    CONSTRAINT chk_project_build_settings_file_nonempty
        CHECK (length(trim(build_config_file)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_project_build_settings_updated_at
    ON project_build_settings(updated_at DESC);
