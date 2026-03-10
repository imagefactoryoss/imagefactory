package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/project"
)

// ProjectRepository implements the project.Repository interface
type ProjectRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *sqlx.DB, logger *zap.Logger) *ProjectRepository {
	return &ProjectRepository{
		db:     db,
		logger: logger,
	}
}

// projectModel represents the database model for project
type projectModel struct {
	ID          uuid.UUID  `db:"id"`
	TenantID    uuid.UUID  `db:"tenant_id"`
	Name        string     `db:"name"`
	Slug        string     `db:"slug"`
	Description string     `db:"description"`
	Status      string     `db:"status"`
	Visibility  string     `db:"visibility"`
	GitRepo     string     `db:"git_repository_url"`
	GitBranch   string     `db:"git_branch"`
	GitProvider string     `db:"git_provider_key"`
	RepoAuthID  *uuid.UUID `db:"repository_auth_id"`
	CreatedBy   *uuid.UUID `db:"created_by"`
	IsDraft     bool       `db:"is_draft"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

// Save persists a project
func (r *ProjectRepository) Save(ctx context.Context, p *project.Project) error {
	query := `
		INSERT INTO projects (id, tenant_id, name, slug, description, status, visibility, git_provider_key, repository_auth_id, created_by, is_draft, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		p.ID(),
		p.TenantID(),
		p.Name(),
		p.Slug(),
		p.Description(),
		string(p.Status()),
		string(p.Visibility()),
		p.GitProvider(),
		p.RepositoryAuthID(),
		p.CreatedBy(),
		p.IsDraft(),
		p.CreatedAt(),
		p.UpdatedAt(),
		p.DeletedAt(),
	)

	if err != nil {
		r.logger.Error("Failed to save project", zap.Error(err), zap.String("project_id", p.ID().String()))
		return err
	}
	if strings.TrimSpace(p.GitRepo()) != "" {
		if err := r.upsertDefaultProjectSource(ctx, p); err != nil {
			return err
		}
	}

	r.logger.Debug("Project saved successfully", zap.String("project_id", p.ID().String()))
	return nil
}

// FindByID retrieves a project by ID
func (r *ProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*project.Project, error) {
	query := `
		SELECT p.id, p.tenant_id, p.name, p.slug, p.description, p.status, p.visibility,
		       COALESCE(ps.repository_url, '') AS git_repository_url,
		       COALESCE(ps.default_branch, 'main') AS git_branch,
		       p.git_provider_key, p.repository_auth_id, p.created_by, p.is_draft, p.created_at, p.updated_at, p.deleted_at
		FROM projects p
		LEFT JOIN LATERAL (
		    SELECT repository_url, default_branch
		    FROM project_sources
		    WHERE project_id = p.id AND is_active = true
		    ORDER BY is_default DESC, created_at ASC
		    LIMIT 1
		) ps ON true
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`

	var model projectModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find project by ID", zap.Error(err), zap.String("project_id", id.String()))
		return nil, err
	}

	return r.modelToProject(model), nil
}

// FindByTenantID retrieves projects for a tenant
func (r *ProjectRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID, limit, offset int) ([]*project.Project, error) {
	query := `
		SELECT p.id, p.tenant_id, p.name, p.slug, p.description, p.status, p.visibility,
		       COALESCE(ps.repository_url, '') AS git_repository_url,
		       COALESCE(ps.default_branch, 'main') AS git_branch,
		       p.git_provider_key, p.repository_auth_id, p.created_by, p.is_draft, p.created_at, p.updated_at, p.deleted_at
		FROM projects p
		LEFT JOIN LATERAL (
		    SELECT repository_url, default_branch
		    FROM project_sources
		    WHERE project_id = p.id AND is_active = true
		    ORDER BY is_default DESC, created_at ASC
		    LIMIT 1
		) ps ON true
		WHERE p.tenant_id = $1 AND p.deleted_at IS NULL
	`
	args := []interface{}{tenantID}
	argIndex := 2
	if viewerID != nil {
		query += " AND (is_draft = false OR created_by = $2)"
		args = append(args, *viewerID)
		argIndex++
	}
	query += `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argIndex) + ` OFFSET $` + fmt.Sprint(argIndex+1)
	args = append(args, limit, offset)

	var models []projectModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		r.logger.Error("Failed to find projects by tenant ID", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, err
	}

	projects := make([]*project.Project, len(models))
	for i, model := range models {
		projects[i] = r.modelToProject(model)
	}

	return projects, nil
}

// FindByNameAndTenantID retrieves a project by name and tenant ID
func (r *ProjectRepository) FindByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (*project.Project, error) {
	query := `
		SELECT p.id, p.tenant_id, p.name, p.slug, p.description, p.status, p.visibility,
		       COALESCE(ps.repository_url, '') AS git_repository_url,
		       COALESCE(ps.default_branch, 'main') AS git_branch,
		       p.git_provider_key, p.repository_auth_id, p.created_by, p.is_draft, p.created_at, p.updated_at, p.deleted_at
		FROM projects p
		LEFT JOIN LATERAL (
		    SELECT repository_url, default_branch
		    FROM project_sources
		    WHERE project_id = p.id AND is_active = true
		    ORDER BY is_default DESC, created_at ASC
		    LIMIT 1
		) ps ON true
		WHERE p.name = $1 AND p.tenant_id = $2 AND p.deleted_at IS NULL
	`

	var model projectModel
	err := r.db.GetContext(ctx, &model, query, name, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find project by name and tenant ID", zap.Error(err), zap.String("name", name), zap.String("tenant_id", tenantID.String()))
		return nil, err
	}

	return r.modelToProject(model), nil
}

// Update updates an existing project
func (r *ProjectRepository) Update(ctx context.Context, p *project.Project) error {
	query := `
		UPDATE projects
		SET name = $1, slug = $2, description = $3, status = $4, git_provider_key = $5, repository_auth_id = $6, is_draft = $7, updated_at = $8, deleted_at = $9
		WHERE id = $10
	`

	result, err := r.db.ExecContext(ctx, query,
		p.Name(),
		p.Slug(),
		p.Description(),
		string(p.Status()),
		p.GitProvider(),
		p.RepositoryAuthID(),
		p.IsDraft(),
		p.UpdatedAt(),
		p.DeletedAt(),
		p.ID(),
	)

	if err != nil {
		r.logger.Error("Failed to update project", zap.Error(err), zap.String("project_id", p.ID().String()))
		return err
	}
	if strings.TrimSpace(p.GitRepo()) != "" {
		if err := r.upsertDefaultProjectSource(ctx, p); err != nil {
			return err
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn("No project updated", zap.String("project_id", p.ID().String()))
		return sql.ErrNoRows
	}

	r.logger.Debug("Project updated successfully", zap.String("project_id", p.ID().String()))
	return nil
}

// Delete performs soft delete
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to start transaction for delete", zap.Error(err))
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	projectQuery := `
		UPDATE projects
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := tx.ExecContext(ctx, projectQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete project", zap.Error(err), zap.String("project_id", id.String()))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn("No project deleted", zap.String("project_id", id.String()))
		return sql.ErrNoRows
	}

	authQuery := `
		UPDATE repository_auth
		SET is_active = false, updated_at = NOW()
		WHERE project_id = $1 AND is_active = true
	`
	if _, err = tx.ExecContext(ctx, authQuery, id); err != nil {
		r.logger.Error("Failed to deactivate repository auths", zap.Error(err), zap.String("project_id", id.String()))
		return err
	}

	if err = tx.Commit(); err != nil {
		r.logger.Error("Failed to commit delete transaction", zap.Error(err))
		return err
	}

	r.logger.Debug("Project deleted successfully", zap.String("project_id", id.String()))
	return nil
}

// PurgeDeletedBefore permanently deletes projects deleted before cutoff.
func (r *ProjectRepository) PurgeDeletedBefore(ctx context.Context, cutoff time.Time) (int, error) {
	query := `
		DELETE FROM projects
		WHERE deleted_at IS NOT NULL AND deleted_at < $1
	`

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		r.logger.Error("Failed to purge deleted projects", zap.Error(err))
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected for purge", zap.Error(err))
		return 0, err
	}

	return int(rowsAffected), nil
}

// CountByTenantID counts projects for a tenant
func (r *ProjectRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM projects
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{tenantID}
	if viewerID != nil {
		query += " AND (is_draft = false OR created_by = $2)"
		args = append(args, *viewerID)
	}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		r.logger.Error("Failed to count projects by tenant ID", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return 0, err
	}

	return count, nil
}

// ExistsByNameAndTenantID checks if a project name exists for a tenant
func (r *ProjectRepository) ExistsByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM projects
			WHERE name = $1 AND tenant_id = $2 AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, name, tenantID)
	if err != nil {
		r.logger.Error("Failed to check project existence", zap.Error(err), zap.String("name", name), zap.String("tenant_id", tenantID.String()))
		return false, err
	}

	return exists, nil
}

// ExistsBySlugAndTenantID checks if a project slug exists for a tenant
func (r *ProjectRepository) ExistsBySlugAndTenantID(ctx context.Context, slug string, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM projects
			WHERE slug = $1 AND tenant_id = $2 AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, slug, tenantID)
	if err != nil {
		r.logger.Error("Failed to check project slug existence", zap.Error(err), zap.String("slug", slug), zap.String("tenant_id", tenantID.String()))
		return false, err
	}

	return exists, nil
}

// modelToProject converts a database model to a domain project
func (r *ProjectRepository) modelToProject(model projectModel) *project.Project {
	status := project.ProjectStatus(model.Status)
	return project.NewProjectFromExisting(
		model.ID,
		model.TenantID,
		model.Name,
		model.Slug,
		model.Description,
		model.GitRepo,
		model.GitBranch,
		model.GitProvider,
		status,
		model.Visibility,
		model.RepoAuthID,
		model.CreatedBy,
		model.IsDraft,
		0, // buildCount - calculated dynamically
		model.CreatedAt,
		model.UpdatedAt,
		model.DeletedAt,
		1, // version - simplified for now
	)
}

func (r *ProjectRepository) upsertDefaultProjectSource(ctx context.Context, p *project.Project) error {
	branch := strings.TrimSpace(p.GitBranch())
	if branch == "" {
		branch = "main"
	}
	provider := strings.TrimSpace(p.GitProvider())
	if provider == "" {
		provider = "generic"
	}

	query := `
		INSERT INTO project_sources (
			id, tenant_id, project_id, name, provider, repository_url, default_branch, repository_auth_id, is_default, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'primary', $4, $5, $6, $7, true, true, NOW(), NOW()
		)
		ON CONFLICT (project_id, name) DO UPDATE SET
			provider = EXCLUDED.provider,
			repository_url = EXCLUDED.repository_url,
			default_branch = EXCLUDED.default_branch,
			repository_auth_id = EXCLUDED.repository_auth_id,
			is_default = true,
			is_active = true,
			updated_at = NOW()`

	if _, err := r.db.ExecContext(ctx, query,
		uuid.New(),
		p.TenantID(),
		p.ID(),
		provider,
		p.GitRepo(),
		branch,
		p.RepositoryAuthID(),
	); err != nil {
		r.logger.Error("Failed to upsert default project source", zap.Error(err), zap.String("project_id", p.ID().String()))
		return err
	}
	return nil
}
