package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"go.uber.org/zap"
)

type RegistryAuthRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewRegistryAuthRepository(db *sqlx.DB, logger *zap.Logger) *RegistryAuthRepository {
	return &RegistryAuthRepository{db: db, logger: logger}
}

type registryAuthModel struct {
	ID             uuid.UUID  `db:"id"`
	TenantID       uuid.UUID  `db:"tenant_id"`
	ProjectID      *uuid.UUID `db:"project_id"`
	Name           string     `db:"name"`
	Description    string     `db:"description"`
	RegistryType   string     `db:"registry_type"`
	AuthType       string     `db:"auth_type"`
	RegistryHost   string     `db:"registry_host"`
	CredentialData []byte     `db:"credential_data"`
	IsActive       bool       `db:"is_active"`
	IsDefault      bool       `db:"is_default"`
	CreatedBy      uuid.UUID  `db:"created_by"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

func (r *RegistryAuthRepository) Save(ctx context.Context, auth *registryauth.RegistryAuth) error {
	var query string
	args := []interface{}{
		auth.ID, auth.TenantID, auth.ProjectID, auth.Name, auth.Description, auth.RegistryType, string(auth.AuthType), auth.RegistryHost,
		auth.CredentialData(), auth.IsActive, auth.IsDefault, auth.CreatedBy, auth.CreatedAt, auth.UpdatedAt,
	}

	if auth.ProjectID == nil {
		query = `
			INSERT INTO registry_auth (
				id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host,
				credential_data, is_active, is_default, created_by, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
			)
			ON CONFLICT (tenant_id, name) WHERE project_id IS NULL
			DO UPDATE SET
				description = EXCLUDED.description,
				registry_type = EXCLUDED.registry_type,
				auth_type = EXCLUDED.auth_type,
				registry_host = EXCLUDED.registry_host,
				credential_data = EXCLUDED.credential_data,
				is_active = EXCLUDED.is_active,
				is_default = EXCLUDED.is_default,
				created_by = EXCLUDED.created_by,
				updated_at = EXCLUDED.updated_at`
	} else {
		query = `
			INSERT INTO registry_auth (
				id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host,
				credential_data, is_active, is_default, created_by, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
			)
			ON CONFLICT (project_id, name) WHERE project_id IS NOT NULL
			DO UPDATE SET
				description = EXCLUDED.description,
				registry_type = EXCLUDED.registry_type,
				auth_type = EXCLUDED.auth_type,
				registry_host = EXCLUDED.registry_host,
				credential_data = EXCLUDED.credential_data,
				is_active = EXCLUDED.is_active,
				is_default = EXCLUDED.is_default,
				created_by = EXCLUDED.created_by,
				updated_at = EXCLUDED.updated_at`
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *RegistryAuthRepository) Update(ctx context.Context, auth *registryauth.RegistryAuth) error {
	query := `
		UPDATE registry_auth
		SET
			name = $2,
			description = $3,
			registry_type = $4,
			auth_type = $5,
			registry_host = $6,
			credential_data = $7,
			is_default = $8,
			updated_at = $9
		WHERE id = $1`
	result, err := r.db.ExecContext(
		ctx,
		query,
		auth.ID,
		auth.Name,
		auth.Description,
		auth.RegistryType,
		string(auth.AuthType),
		auth.RegistryHost,
		auth.CredentialData(),
		auth.IsDefault,
		auth.UpdatedAt,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return registryauth.ErrRegistryAuthNotFound
	}
	return nil
}

func (r *RegistryAuthRepository) FindByID(ctx context.Context, id uuid.UUID) (*registryauth.RegistryAuth, error) {
	query := `SELECT id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host, credential_data, is_active, is_default, created_by, created_at, updated_at FROM registry_auth WHERE id = $1`
	var model registryAuthModel
	if err := r.db.GetContext(ctx, &model, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.modelToDomain(model), nil
}

func (r *RegistryAuthRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*registryauth.RegistryAuth, error) {
	var rows []registryAuthModel
	if includeTenant {
		var tenantID uuid.UUID
		if err := r.db.GetContext(ctx, &tenantID, `SELECT tenant_id FROM projects WHERE id = $1`, projectID); err != nil {
			return nil, err
		}
		query := `
			SELECT id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host, credential_data, is_active, is_default, created_by, created_at, updated_at
			FROM registry_auth
			WHERE is_active = true AND ((project_id = $1) OR (project_id IS NULL AND tenant_id = $2))
			ORDER BY created_at DESC`
		if err := r.db.SelectContext(ctx, &rows, query, projectID, tenantID); err != nil {
			return nil, err
		}
	} else {
		query := `
			SELECT id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host, credential_data, is_active, is_default, created_by, created_at, updated_at
			FROM registry_auth
			WHERE project_id = $1 AND is_active = true
			ORDER BY created_at DESC`
		if err := r.db.SelectContext(ctx, &rows, query, projectID); err != nil {
			return nil, err
		}
	}
	out := make([]*registryauth.RegistryAuth, 0, len(rows))
	for _, row := range rows {
		out = append(out, r.modelToDomain(row))
	}
	return out, nil
}

func (r *RegistryAuthRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*registryauth.RegistryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host, credential_data, is_active, is_default, created_by, created_at, updated_at
		FROM registry_auth
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC`
	var rows []registryAuthModel
	if err := r.db.SelectContext(ctx, &rows, query, tenantID); err != nil {
		return nil, err
	}
	out := make([]*registryauth.RegistryAuth, 0, len(rows))
	for _, row := range rows {
		out = append(out, r.modelToDomain(row))
	}
	return out, nil
}

func (r *RegistryAuthRepository) FindDefaultByProjectID(ctx context.Context, projectID uuid.UUID) (*registryauth.RegistryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host, credential_data, is_active, is_default, created_by, created_at, updated_at
		FROM registry_auth
		WHERE project_id = $1 AND is_default = true AND is_active = true
		LIMIT 1`
	var row registryAuthModel
	if err := r.db.GetContext(ctx, &row, query, projectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.modelToDomain(row), nil
}

func (r *RegistryAuthRepository) FindDefaultByTenantID(ctx context.Context, tenantID uuid.UUID) (*registryauth.RegistryAuth, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, registry_type, auth_type, registry_host, credential_data, is_active, is_default, created_by, created_at, updated_at
		FROM registry_auth
		WHERE tenant_id = $1 AND project_id IS NULL AND is_default = true AND is_active = true
		LIMIT 1`
	var row registryAuthModel
	if err := r.db.GetContext(ctx, &row, query, tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.modelToDomain(row), nil
}

func (r *RegistryAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `UPDATE registry_auth SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return registryauth.ErrRegistryAuthNotFound
	}
	return nil
}

func (r *RegistryAuthRepository) ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error) {
	var query string
	var args []interface{}
	if projectID == nil {
		query = `SELECT COUNT(*) > 0 FROM registry_auth WHERE tenant_id = $1 AND project_id IS NULL AND name = $2 AND is_active = true`
		args = []interface{}{tenantID, name}
	} else {
		query = `SELECT COUNT(*) > 0 FROM registry_auth WHERE project_id = $1 AND name = $2 AND is_active = true`
		args = []interface{}{*projectID, name}
	}
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, args...); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *RegistryAuthRepository) modelToDomain(m registryAuthModel) *registryauth.RegistryAuth {
	return registryauth.NewRegistryAuthFromExisting(
		m.ID, m.TenantID, m.ProjectID, m.Name, m.Description, m.RegistryType, registryauth.AuthType(m.AuthType),
		m.RegistryHost, m.CredentialData, m.IsActive, m.IsDefault, m.CreatedBy, m.CreatedAt, m.UpdatedAt,
	)
}
