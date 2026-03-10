package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type ProjectSource struct {
	ID             uuid.UUID  `db:"id"`
	Name           string     `db:"name"`
	ProjectID      uuid.UUID  `db:"project_id"`
	TenantID       uuid.UUID  `db:"tenant_id"`
	Provider       string     `db:"provider"`
	RepositoryURL  string     `db:"repository_url"`
	DefaultBranch  string     `db:"default_branch"`
	RepositoryAuth *uuid.UUID `db:"repository_auth_id"`
	IsDefault      bool       `db:"is_default"`
	IsActive       bool       `db:"is_active"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

type ProjectSourceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

var ErrDuplicateProjectSource = errors.New("duplicate project source for provider/repository/default_branch")

func NewProjectSourceRepository(db *sqlx.DB, logger *zap.Logger) *ProjectSourceRepository {
	return &ProjectSourceRepository{db: db, logger: logger}
}

func (r *ProjectSourceRepository) FindDefaultByProjectID(ctx context.Context, projectID uuid.UUID) (*ProjectSource, error) {
	query := `
		SELECT id, name, project_id, tenant_id, provider, repository_url, default_branch, repository_auth_id, is_default, is_active, created_at, updated_at
		FROM project_sources
		WHERE project_id = $1 AND is_active = true
		ORDER BY is_default DESC, created_at ASC
		LIMIT 1`

	var source ProjectSource
	if err := r.db.GetContext(ctx, &source, query, projectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find default project source", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}
	return &source, nil
}

func (r *ProjectSourceRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]ProjectSource, error) {
	query := `
		SELECT id, name, project_id, tenant_id, provider, repository_url, default_branch, repository_auth_id, is_default, is_active, created_at, updated_at
		FROM project_sources
		WHERE project_id = $1
		ORDER BY is_default DESC, created_at ASC`
	var sources []ProjectSource
	if err := r.db.SelectContext(ctx, &sources, query, projectID); err != nil {
		r.logger.Error("Failed to list project sources", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}
	return sources, nil
}

func (r *ProjectSourceRepository) FindByID(ctx context.Context, projectID, sourceID uuid.UUID) (*ProjectSource, error) {
	query := `
		SELECT id, name, project_id, tenant_id, provider, repository_url, default_branch, repository_auth_id, is_default, is_active, created_at, updated_at
		FROM project_sources
		WHERE id = $1 AND project_id = $2`
	var source ProjectSource
	if err := r.db.GetContext(ctx, &source, query, sourceID, projectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find project source", zap.Error(err), zap.String("project_id", projectID.String()), zap.String("source_id", sourceID.String()))
		return nil, err
	}
	return &source, nil
}

func (r *ProjectSourceRepository) Create(ctx context.Context, source *ProjectSource) error {
	if source == nil {
		return fmt.Errorf("source is required")
	}
	source.Name = strings.TrimSpace(source.Name)
	source.Provider = strings.TrimSpace(source.Provider)
	source.RepositoryURL = strings.TrimSpace(source.RepositoryURL)
	source.DefaultBranch = strings.TrimSpace(source.DefaultBranch)
	if source.Name == "" || source.RepositoryURL == "" {
		return fmt.Errorf("name and repository_url are required")
	}
	if source.Provider == "" {
		source.Provider = "generic"
	}
	if source.DefaultBranch == "" {
		source.DefaultBranch = "main"
	}
	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}
	duplicate, err := r.hasDuplicateSource(ctx, source.ProjectID, source.Provider, source.RepositoryURL, source.DefaultBranch, nil)
	if err != nil {
		return err
	}
	if duplicate {
		return ErrDuplicateProjectSource
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if source.IsDefault {
		if _, err = tx.ExecContext(ctx, `UPDATE project_sources SET is_default = false WHERE project_id = $1`, source.ProjectID); err != nil {
			return err
		}
	}

	query := `
		INSERT INTO project_sources (
			id, name, project_id, tenant_id, provider, repository_url, default_branch, repository_auth_id, is_default, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
		)`
	if _, err = tx.ExecContext(ctx, query, source.ID, source.Name, source.ProjectID, source.TenantID, source.Provider, source.RepositoryURL, source.DefaultBranch, source.RepositoryAuth, source.IsDefault, source.IsActive); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ProjectSourceRepository) Update(ctx context.Context, source *ProjectSource) error {
	if source == nil || source.ID == uuid.Nil {
		return fmt.Errorf("valid source is required")
	}
	source.Name = strings.TrimSpace(source.Name)
	source.Provider = strings.TrimSpace(source.Provider)
	source.RepositoryURL = strings.TrimSpace(source.RepositoryURL)
	source.DefaultBranch = strings.TrimSpace(source.DefaultBranch)
	if source.Name == "" || source.RepositoryURL == "" {
		return fmt.Errorf("name and repository_url are required")
	}
	if source.Provider == "" {
		source.Provider = "generic"
	}
	if source.DefaultBranch == "" {
		source.DefaultBranch = "main"
	}
	duplicate, err := r.hasDuplicateSource(ctx, source.ProjectID, source.Provider, source.RepositoryURL, source.DefaultBranch, &source.ID)
	if err != nil {
		return err
	}
	if duplicate {
		return ErrDuplicateProjectSource
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if source.IsDefault {
		if _, err = tx.ExecContext(ctx, `UPDATE project_sources SET is_default = false WHERE project_id = $1`, source.ProjectID); err != nil {
			return err
		}
	}

	query := `
		UPDATE project_sources
		SET name = $1, provider = $2, repository_url = $3, default_branch = $4, repository_auth_id = $5, is_default = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8 AND project_id = $9`
	result, err := tx.ExecContext(ctx, query, source.Name, source.Provider, source.RepositoryURL, source.DefaultBranch, source.RepositoryAuth, source.IsDefault, source.IsActive, source.ID, source.ProjectID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ProjectSourceRepository) Delete(ctx context.Context, projectID, sourceID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM project_sources WHERE id = $1 AND project_id = $2`, sourceID, projectID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ProjectSourceRepository) hasDuplicateSource(ctx context.Context, projectID uuid.UUID, provider, repositoryURL, defaultBranch string, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(1)
		FROM project_sources
		WHERE project_id = $1
		  AND LOWER(TRIM(provider)) = LOWER(TRIM($2))
		  AND LOWER(TRIM(repository_url)) = LOWER(TRIM($3))
		  AND LOWER(TRIM(default_branch)) = LOWER(TRIM($4))
		  AND ($5::uuid IS NULL OR id <> $5::uuid)`

	var count int
	var exclude interface{}
	if excludeID != nil && *excludeID != uuid.Nil {
		exclude = *excludeID
	}
	if err := r.db.GetContext(ctx, &count, query, projectID, provider, repositoryURL, defaultBranch, exclude); err != nil {
		return false, err
	}
	return count > 0, nil
}
