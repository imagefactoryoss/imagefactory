package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
)

// RepositoryAuthRepository implements the repositoryauth.Repository interface
type RepositoryAuthRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewRepositoryAuthRepository creates a new repository authentication repository
func NewRepositoryAuthRepository(db *sqlx.DB, logger *zap.Logger) *RepositoryAuthRepository {
	return &RepositoryAuthRepository{
		db:     db,
		logger: logger,
	}
}

// repositoryAuthModel represents the database model for repository authentication
type repositoryAuthModel struct {
	ID             uuid.UUID  `db:"id"`
	TenantID       uuid.UUID  `db:"tenant_id"`
	ProjectID      *uuid.UUID `db:"project_id"`
	Name           string     `db:"name"`
	Description    string     `db:"description"`
	AuthType       string     `db:"auth_type"`
	CredentialData []byte     `db:"credential_data"`
	IsActive       bool       `db:"is_active"`
	CreatedBy      uuid.UUID  `db:"created_by"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

type repositoryAuthSummaryModel struct {
	ID             uuid.UUID `db:"id"`
	ProjectID      uuid.UUID `db:"project_id"`
	ProjectName    string    `db:"project_name"`
	GitProviderKey string    `db:"git_provider_key"`
	Name           string    `db:"name"`
	Description    string    `db:"description"`
	AuthType       string    `db:"auth_type"`
	IsActive       bool      `db:"is_active"`
	CreatedBy      uuid.UUID `db:"created_by"`
	CreatedByEmail string    `db:"created_by_email"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Save persists a repository authentication configuration
func (r *RepositoryAuthRepository) Save(ctx context.Context, auth *repositoryauth.RepositoryAuth) error {
	query := `
		INSERT INTO repository_auth (id, tenant_id, project_id, name, description, auth_type, credential_data, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	model := repositoryAuthModel{
		ID:             auth.GetID(),
		TenantID:       auth.GetTenantID(),
		ProjectID:      auth.GetProjectID(),
		Name:           auth.GetName(),
		Description:    auth.GetDescription(),
		AuthType:       string(auth.GetAuthType()),
		CredentialData: auth.CredentialData(),
		IsActive:       auth.GetIsActive(),
		CreatedBy:      auth.GetCreatedBy(),
		CreatedAt:      auth.GetCreatedAt(),
		UpdatedAt:      auth.GetUpdatedAt(),
	}

	_, err := r.db.ExecContext(ctx, query,
		model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
		model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to save repository auth", zap.Error(err), zap.String("id", auth.GetID().String()))
		return err
	}

	r.logger.Info("Repository auth saved", zap.String("id", auth.GetID().String()), zap.String("tenantId", auth.GetTenantID().String()))
	return nil
}

// FindByID retrieves a repository authentication by ID
func (r *RepositoryAuthRepository) FindByID(ctx context.Context, id uuid.UUID) (*repositoryauth.RepositoryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, auth_type, credential_data, is_active, created_by, created_at, updated_at
		FROM repository_auth
		WHERE id = $1
	`

	var model repositoryAuthModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find repository auth by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	auth := repositoryauth.NewRepositoryAuthFromExisting(
		model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
		model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt, 1,
	)

	return auth, nil
}

// FindByProjectID retrieves all project-scoped repository authentications for a project
func (r *RepositoryAuthRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*repositoryauth.RepositoryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, auth_type, credential_data, is_active, created_by, created_at, updated_at
		FROM repository_auth
		WHERE project_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	var models []repositoryAuthModel
	err := r.db.SelectContext(ctx, &models, query, projectID)
	if err != nil {
		r.logger.Error("Failed to find repository auths by project ID", zap.Error(err), zap.String("projectId", projectID.String()))
		return nil, err
	}

	auths := make([]*repositoryauth.RepositoryAuth, len(models))
	for i, model := range models {
		auths[i] = repositoryauth.NewRepositoryAuthFromExisting(
			model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
			model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt, 1,
		)
	}

	return auths, nil
}

func (r *RepositoryAuthRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*repositoryauth.RepositoryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, auth_type, credential_data, is_active, created_by, created_at, updated_at
		FROM repository_auth
		WHERE tenant_id = $1 AND project_id IS NULL AND is_active = true
		ORDER BY created_at DESC
	`

	var models []repositoryAuthModel
	err := r.db.SelectContext(ctx, &models, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to find repository auths by tenant ID", zap.Error(err), zap.String("tenantId", tenantID.String()))
		return nil, err
	}

	auths := make([]*repositoryauth.RepositoryAuth, len(models))
	for i, model := range models {
		auths[i] = repositoryauth.NewRepositoryAuthFromExisting(
			model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
			model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt, 1,
		)
	}
	return auths, nil
}

func (r *RepositoryAuthRepository) FindByProjectIDWithTenant(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*repositoryauth.RepositoryAuth, error) {
	query := `
		SELECT ra.id, ra.tenant_id, ra.project_id, ra.name, ra.description, ra.auth_type, ra.credential_data, ra.is_active, ra.created_by, ra.created_at, ra.updated_at
		FROM repository_auth ra
		JOIN projects p ON p.id = $1
		WHERE (ra.project_id = $1 OR ($2 = true AND ra.tenant_id = p.tenant_id AND ra.project_id IS NULL))
		  AND ra.is_active = true
		ORDER BY ra.created_at DESC
	`

	var models []repositoryAuthModel
	if err := r.db.SelectContext(ctx, &models, query, projectID, includeTenant); err != nil {
		r.logger.Error("Failed to find repository auths by project with tenant fallback", zap.Error(err), zap.String("projectId", projectID.String()))
		return nil, err
	}

	auths := make([]*repositoryauth.RepositoryAuth, len(models))
	for i, model := range models {
		auths[i] = repositoryauth.NewRepositoryAuthFromExisting(
			model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
			model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt, 1,
		)
	}
	return auths, nil
}

// FindSummariesByTenantID retrieves project-scoped repository auths for a tenant with project context.
func (r *RepositoryAuthRepository) FindSummariesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]repositoryauth.RepositoryAuthSummary, error) {
	query := `
		SELECT ra.id, ra.project_id, p.name AS project_name, COALESCE(p.git_provider_key, '') AS git_provider_key,
		       ra.name, ra.description, ra.auth_type, ra.is_active, ra.created_by,
		       COALESCE(u.email, '') AS created_by_email,
		       ra.created_at, ra.updated_at
		FROM repository_auth ra
		JOIN projects p ON p.id = ra.project_id
		LEFT JOIN users u ON u.id = ra.created_by
		WHERE ra.tenant_id = $1 AND ra.project_id IS NOT NULL AND ra.is_active = true
		ORDER BY ra.created_at DESC
	`

	var models []repositoryAuthSummaryModel
	if err := r.db.SelectContext(ctx, &models, query, tenantID); err != nil {
		r.logger.Error("Failed to find repository auths by tenant ID", zap.Error(err), zap.String("tenantId", tenantID.String()))
		return nil, err
	}

	auths := make([]repositoryauth.RepositoryAuthSummary, len(models))
	for i, model := range models {
		auths[i] = repositoryauth.RepositoryAuthSummary{
			ID:             model.ID,
			ProjectID:      model.ProjectID,
			ProjectName:    model.ProjectName,
			GitProviderKey: model.GitProviderKey,
			Name:           model.Name,
			Description:    model.Description,
			AuthType:       repositoryauth.AuthType(model.AuthType),
			IsActive:       model.IsActive,
			CreatedBy:      model.CreatedBy,
			CreatedByEmail: model.CreatedByEmail,
			CreatedAt:      model.CreatedAt,
			UpdatedAt:      model.UpdatedAt,
		}
	}

	return auths, nil
}

// FindActiveByProjectID retrieves the active repository authentication for a project
func (r *RepositoryAuthRepository) FindActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*repositoryauth.RepositoryAuth, error) {
	query := `
		SELECT ra.id, ra.tenant_id, ra.project_id, ra.name, ra.description, ra.auth_type, ra.credential_data, ra.is_active, ra.created_by, ra.created_at, ra.updated_at
		FROM repository_auth ra
		JOIN projects p ON p.id = $1
		WHERE ra.is_active = true
		  AND (ra.project_id = $1 OR (ra.tenant_id = p.tenant_id AND ra.project_id IS NULL))
		ORDER BY CASE WHEN ra.project_id = $1 THEN 0 ELSE 1 END, ra.created_at DESC
		LIMIT 1
	`

	var model repositoryAuthModel
	err := r.db.GetContext(ctx, &model, query, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find active repository auth by project ID", zap.Error(err), zap.String("projectId", projectID.String()))
		return nil, err
	}

	auth := repositoryauth.NewRepositoryAuthFromExisting(
		model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
		model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt, 1,
	)

	return auth, nil
}

// FindByNameAndProjectID retrieves a repository authentication by name and project ID
func (r *RepositoryAuthRepository) FindByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (*repositoryauth.RepositoryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, auth_type, credential_data, is_active, created_by, created_at, updated_at
		FROM repository_auth
		WHERE name = $1 AND project_id = $2
	`

	var model repositoryAuthModel
	err := r.db.GetContext(ctx, &model, query, name, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find repository auth by name and project ID", zap.Error(err), zap.String("name", name), zap.String("projectId", projectID.String()))
		return nil, err
	}

	auth := repositoryauth.NewRepositoryAuthFromExisting(
		model.ID, model.TenantID, model.ProjectID, model.Name, model.Description, model.AuthType,
		model.CredentialData, model.IsActive, model.CreatedBy, model.CreatedAt, model.UpdatedAt, 1,
	)

	return auth, nil
}

// Update updates an existing repository authentication
func (r *RepositoryAuthRepository) Update(ctx context.Context, auth *repositoryauth.RepositoryAuth) error {
	query := `
		UPDATE repository_auth
		SET name = $1, description = $2, credential_data = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		auth.GetName(), auth.GetDescription(), auth.CredentialData(), auth.GetUpdatedAt(), auth.GetID())

	if err != nil {
		r.logger.Error("Failed to update repository auth", zap.Error(err), zap.String("id", auth.GetID().String()))
		return err
	}

	r.logger.Info("Repository auth updated", zap.String("id", auth.GetID().String()))
	return nil
}

// Delete performs soft delete (deactivate) of a repository authentication
func (r *RepositoryAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE repository_auth
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete repository auth", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		return repositoryauth.ErrRepositoryAuthNotFound
	}

	r.logger.Info("Repository auth deleted (deactivated)", zap.String("id", id.String()))
	return nil
}

// ExistsByNameAndProjectID checks if a repository authentication name exists for a project
func (r *RepositoryAuthRepository) ExistsByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM repository_auth
		WHERE name = $1 AND project_id = $2 AND is_active = true
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, name, projectID)
	if err != nil {
		r.logger.Error("Failed to check repository auth existence", zap.Error(err), zap.String("name", name), zap.String("projectId", projectID.String()))
		return false, err
	}

	return exists, nil
}

func (r *RepositoryAuthRepository) ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error) {
	var (
		query  string
		exists bool
		err    error
	)

	if projectID == nil {
		query = `
			SELECT COUNT(*) > 0
			FROM repository_auth
			WHERE tenant_id = $1 AND project_id IS NULL AND name = $2 AND is_active = true
		`
		err = r.db.GetContext(ctx, &exists, query, tenantID, name)
	} else {
		query = `
			SELECT COUNT(*) > 0
			FROM repository_auth
			WHERE project_id = $1 AND name = $2 AND is_active = true
		`
		err = r.db.GetContext(ctx, &exists, query, *projectID, name)
	}

	if err != nil {
		r.logger.Error("Failed to check repository auth existence in scope", zap.Error(err), zap.String("name", name), zap.String("tenantId", tenantID.String()))
		return false, err
	}

	return exists, nil
}

// FindActiveProjectUsages returns active (non-deleted) projects currently using the auth.
func (r *RepositoryAuthRepository) FindActiveProjectUsages(ctx context.Context, authID uuid.UUID) ([]repositoryauth.ProjectUsage, error) {
	query := `
		SELECT p.id AS project_id, p.name AS project_name
		FROM projects p
		WHERE p.repository_auth_id = $1
		  AND p.deleted_at IS NULL
		ORDER BY p.name ASC
	`

	type usageRow struct {
		ProjectID   uuid.UUID `db:"project_id"`
		ProjectName string    `db:"project_name"`
	}

	var rows []usageRow
	if err := r.db.SelectContext(ctx, &rows, query, authID); err != nil {
		r.logger.Error("Failed to find active project usages for repository auth", zap.Error(err), zap.String("auth_id", authID.String()))
		return nil, err
	}

	out := make([]repositoryauth.ProjectUsage, 0, len(rows))
	for _, row := range rows {
		out = append(out, repositoryauth.ProjectUsage{
			ProjectID:   row.ProjectID,
			ProjectName: row.ProjectName,
		})
	}
	return out, nil
}
