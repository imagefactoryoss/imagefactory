package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

const (
	ProjectBuildConfigModeRepoManaged = "repo_managed"
	ProjectBuildConfigModeUIManaged   = "ui_managed"
	ProjectBuildConfigOnErrorStrict   = "strict"
	ProjectBuildConfigOnErrorFallback = "fallback_to_ui"
)

type ProjectBuildSettings struct {
	ProjectID          uuid.UUID `db:"project_id"`
	BuildConfigMode    string    `db:"build_config_mode"`
	BuildConfigFile    string    `db:"build_config_file"`
	BuildConfigOnError string    `db:"build_config_on_error"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

type ProjectBuildSettingsRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewProjectBuildSettingsRepository(db *sqlx.DB, logger *zap.Logger) *ProjectBuildSettingsRepository {
	return &ProjectBuildSettingsRepository{db: db, logger: logger}
}

func (r *ProjectBuildSettingsRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*ProjectBuildSettings, error) {
	query := `
		SELECT project_id, build_config_mode, build_config_file, build_config_on_error, created_at, updated_at
		FROM project_build_settings
		WHERE project_id = $1
	`
	var settings ProjectBuildSettings
	if err := r.db.GetContext(ctx, &settings, query, projectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to fetch project build settings", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}
	return r.normalizeSettings(&settings), nil
}

func (r *ProjectBuildSettingsRepository) Upsert(ctx context.Context, settings ProjectBuildSettings) (*ProjectBuildSettings, error) {
	mode := normalizeBuildConfigMode(settings.BuildConfigMode)
	file := normalizeBuildConfigFile(settings.BuildConfigFile)

	query := `
		INSERT INTO project_build_settings (project_id, build_config_mode, build_config_file, build_config_on_error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (project_id)
		DO UPDATE SET build_config_mode = EXCLUDED.build_config_mode,
		              build_config_file = EXCLUDED.build_config_file,
		              build_config_on_error = EXCLUDED.build_config_on_error,
		              updated_at = NOW()
		RETURNING project_id, build_config_mode, build_config_file, build_config_on_error, created_at, updated_at
	`

	var stored ProjectBuildSettings
	if err := r.db.GetContext(ctx, &stored, query, settings.ProjectID, mode, file, normalizeBuildConfigOnError(settings.BuildConfigOnError)); err != nil {
		r.logger.Error("Failed to upsert project build settings", zap.Error(err), zap.String("project_id", settings.ProjectID.String()))
		return nil, err
	}

	return r.normalizeSettings(&stored), nil
}

func normalizeBuildConfigMode(mode string) string {
	candidate := strings.ToLower(strings.TrimSpace(mode))
	switch candidate {
	case ProjectBuildConfigModeUIManaged:
		return ProjectBuildConfigModeUIManaged
	case ProjectBuildConfigModeRepoManaged:
		fallthrough
	default:
		return ProjectBuildConfigModeRepoManaged
	}
}

func normalizeBuildConfigFile(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "image-factory.yaml"
	}
	return trimmed
}

func normalizeBuildConfigOnError(policy string) string {
	candidate := strings.ToLower(strings.TrimSpace(policy))
	switch candidate {
	case ProjectBuildConfigOnErrorFallback:
		return ProjectBuildConfigOnErrorFallback
	case ProjectBuildConfigOnErrorStrict:
		fallthrough
	default:
		return ProjectBuildConfigOnErrorStrict
	}
}

func (r *ProjectBuildSettingsRepository) normalizeSettings(settings *ProjectBuildSettings) *ProjectBuildSettings {
	if settings == nil {
		return nil
	}
	settings.BuildConfigMode = normalizeBuildConfigMode(settings.BuildConfigMode)
	settings.BuildConfigFile = normalizeBuildConfigFile(settings.BuildConfigFile)
	settings.BuildConfigOnError = normalizeBuildConfigOnError(settings.BuildConfigOnError)
	return settings
}
