package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/packertarget"
	"go.uber.org/zap"
)

type PackerTargetProfileRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewPackerTargetProfileRepository(db *sqlx.DB, logger *zap.Logger) packertarget.Repository {
	return &PackerTargetProfileRepository{db: db, logger: logger}
}

type dbPackerTargetProfile struct {
	ID                    uuid.UUID      `db:"id"`
	TenantID              uuid.UUID      `db:"tenant_id"`
	IsGlobal              bool           `db:"is_global"`
	Name                  string         `db:"name"`
	Provider              string         `db:"provider"`
	Description           sql.NullString `db:"description"`
	SecretRef             string         `db:"secret_ref"`
	Options               []byte         `db:"options"`
	ValidationStatus      string         `db:"validation_status"`
	LastValidatedAt       sql.NullTime   `db:"last_validated_at"`
	LastValidationMessage sql.NullString `db:"last_validation_message"`
	LastRemediationHints  []byte         `db:"last_remediation_hints"`
	CreatedBy             uuid.UUID      `db:"created_by"`
	CreatedAt             time.Time      `db:"created_at"`
	UpdatedAt             time.Time      `db:"updated_at"`
}

func (r *PackerTargetProfileRepository) Create(ctx context.Context, profile *packertarget.Profile) error {
	optionsJSON, err := json.Marshal(profile.Options)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	hintsJSON, err := json.Marshal(profile.LastRemediationHints)
	if err != nil {
		return fmt.Errorf("marshal remediation hints: %w", err)
	}

	query := `
		INSERT INTO packer_target_profiles (
			id, tenant_id, is_global, name, provider, description, secret_ref, options,
			validation_status, last_validated_at, last_validation_message, last_remediation_hints,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8,
			$9, $10, NULLIF($11, ''), $12,
			$13, $14, $15
		)`

	_, err = r.db.ExecContext(ctx, query,
		profile.ID,
		profile.TenantID,
		profile.IsGlobal,
		profile.Name,
		profile.Provider,
		strings.TrimSpace(profile.Description),
		profile.SecretRef,
		optionsJSON,
		profile.ValidationStatus,
		profile.LastValidatedAt,
		nilString(profile.LastValidationMessage),
		hintsJSON,
		profile.CreatedBy,
		profile.CreatedAt,
		profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create packer target profile: %w", err)
	}
	return nil
}

func (r *PackerTargetProfileRepository) Update(ctx context.Context, profile *packertarget.Profile) error {
	optionsJSON, err := json.Marshal(profile.Options)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	hintsJSON, err := json.Marshal(profile.LastRemediationHints)
	if err != nil {
		return fmt.Errorf("marshal remediation hints: %w", err)
	}

	query := `
		UPDATE packer_target_profiles
		SET is_global = $1,
			name = $2,
			description = NULLIF($3, ''),
			secret_ref = $4,
			options = $5,
			validation_status = $6,
			last_validated_at = $7,
			last_validation_message = NULLIF($8, ''),
			last_remediation_hints = $9,
			updated_at = $10
		WHERE id = $11 AND tenant_id = $12`
	result, err := r.db.ExecContext(ctx, query,
		profile.IsGlobal,
		profile.Name,
		strings.TrimSpace(profile.Description),
		profile.SecretRef,
		optionsJSON,
		profile.ValidationStatus,
		profile.LastValidatedAt,
		nilString(profile.LastValidationMessage),
		hintsJSON,
		profile.UpdatedAt,
		profile.ID,
		profile.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update packer target profile: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return packertarget.ErrNotFound
	}
	return nil
}

func (r *PackerTargetProfileRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM packer_target_profiles WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete packer target profile: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return packertarget.ErrNotFound
	}
	return nil
}

func (r *PackerTargetProfileRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*packertarget.Profile, error) {
	var row dbPackerTargetProfile
	err := r.db.GetContext(ctx, &row, `
		SELECT id, tenant_id, is_global, name, provider, description, secret_ref, options,
		       validation_status, last_validated_at, last_validation_message, last_remediation_hints,
		       created_by, created_at, updated_at
		FROM packer_target_profiles
		WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, packertarget.ErrNotFound
		}
		return nil, fmt.Errorf("get packer target profile: %w", err)
	}
	return mapPackerTargetProfile(row)
}

func (r *PackerTargetProfileRepository) List(ctx context.Context, tenantID uuid.UUID, allTenants bool, provider string) ([]*packertarget.Profile, error) {
	base := `
		SELECT id, tenant_id, is_global, name, provider, description, secret_ref, options,
		       validation_status, last_validated_at, last_validation_message, last_remediation_hints,
		       created_by, created_at, updated_at
		FROM packer_target_profiles`
	args := make([]interface{}, 0, 2)
	where := make([]string, 0, 2)

	if !allTenants {
		args = append(args, tenantID)
		where = append(where, fmt.Sprintf("tenant_id = $%d", len(args)))
	}
	if strings.TrimSpace(provider) != "" {
		args = append(args, strings.TrimSpace(provider))
		where = append(where, fmt.Sprintf("provider = $%d", len(args)))
	}
	query := base
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows := make([]dbPackerTargetProfile, 0)
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list packer target profiles: %w", err)
	}

	profiles := make([]*packertarget.Profile, 0, len(rows))
	for _, row := range rows {
		profile, err := mapPackerTargetProfile(row)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func (r *PackerTargetProfileRepository) UpdateValidation(ctx context.Context, tenantID, id uuid.UUID, result packertarget.ValidationResult) error {
	hintsJSON, err := json.Marshal(result.RemediationHints)
	if err != nil {
		return fmt.Errorf("marshal remediation hints: %w", err)
	}

	query := `
		UPDATE packer_target_profiles
		SET validation_status = $1,
			last_validated_at = $2,
			last_validation_message = NULLIF($3, ''),
			last_remediation_hints = $4,
			updated_at = $5
		WHERE id = $6 AND tenant_id = $7`
	resultExec, err := r.db.ExecContext(ctx, query,
		result.Status,
		result.CheckedAt,
		result.Message,
		hintsJSON,
		time.Now().UTC(),
		id,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("update packer target profile validation: %w", err)
	}
	rows, _ := resultExec.RowsAffected()
	if rows == 0 {
		return packertarget.ErrNotFound
	}
	return nil
}

func mapPackerTargetProfile(row dbPackerTargetProfile) (*packertarget.Profile, error) {
	options := map[string]interface{}{}
	if len(row.Options) > 0 {
		if err := json.Unmarshal(row.Options, &options); err != nil {
			return nil, fmt.Errorf("unmarshal options: %w", err)
		}
	}
	hints := make([]string, 0)
	if len(row.LastRemediationHints) > 0 {
		if err := json.Unmarshal(row.LastRemediationHints, &hints); err != nil {
			return nil, fmt.Errorf("unmarshal remediation hints: %w", err)
		}
	}

	profile := &packertarget.Profile{
		ID:               row.ID,
		TenantID:         row.TenantID,
		IsGlobal:         row.IsGlobal,
		Name:             row.Name,
		Provider:         row.Provider,
		SecretRef:        row.SecretRef,
		Options:          options,
		ValidationStatus: row.ValidationStatus,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if row.Description.Valid {
		profile.Description = row.Description.String
	}
	if row.LastValidatedAt.Valid {
		validatedAt := row.LastValidatedAt.Time
		profile.LastValidatedAt = &validatedAt
	}
	if row.LastValidationMessage.Valid {
		msg := row.LastValidationMessage.String
		profile.LastValidationMessage = &msg
	}
	profile.LastRemediationHints = hints
	return profile, nil
}

func nilString(value *string) interface{} {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
