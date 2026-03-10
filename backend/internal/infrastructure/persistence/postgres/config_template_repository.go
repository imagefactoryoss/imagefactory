package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/domain/build"
)

// ConfigTemplateRepository PostgreSQL implementation
type ConfigTemplateRepository struct {
	db *sqlx.DB
}

// NewConfigTemplateRepository creates new template repository
func NewConfigTemplateRepository(db *sqlx.DB) *ConfigTemplateRepository {
	return &ConfigTemplateRepository{db: db}
}

// SaveTemplate creates new template
func (r *ConfigTemplateRepository) SaveTemplate(ctx context.Context, template *build.ConfigTemplate) error {
	if template == nil {
		return build.ErrInvalidManifest
	}

	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}

	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}

	if template.UpdatedAt.IsZero() {
		template.UpdatedAt = time.Now()
	}

	templateDataJSON, err := json.Marshal(template.TemplateData)
	if err != nil {
		return fmt.Errorf("failed to marshal template data: %w", err)
	}

	query := `
		INSERT INTO config_templates (id, project_id, created_by_user_id, name, description, method, template_data, is_shared, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (project_id, name) DO NOTHING
	`

	result, err := r.db.ExecContext(ctx, query,
		template.ID,
		template.ProjectID,
		template.CreatedByUserID,
		template.Name,
		template.Description,
		template.Method,
		templateDataJSON,
		template.IsShared,
		template.IsPublic,
		template.CreatedAt,
		template.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique violation
				return build.ErrDuplicateTemplateName
			}
		}
		return fmt.Errorf("failed to save template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return build.ErrDuplicateTemplateName
	}

	return nil
}

// GetTemplate retrieves template by ID
func (r *ConfigTemplateRepository) GetTemplate(ctx context.Context, id uuid.UUID) (*build.ConfigTemplate, error) {
	if id == uuid.Nil {
		return nil, build.ErrInvalidConfigID
	}

	query := `
		SELECT id, project_id, created_by_user_id, name, description, method, template_data, is_shared, is_public, created_at, updated_at
		FROM config_templates
		WHERE id = $1
	`

	var template build.ConfigTemplate
	var templateDataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.ProjectID,
		&template.CreatedByUserID,
		&template.Name,
		&template.Description,
		&template.Method,
		&templateDataJSON,
		&template.IsShared,
		&template.IsPublic,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, build.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Unmarshal JSONB data
	templateData := make(map[string]interface{})
	if err := json.Unmarshal(templateDataJSON, &templateData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template data: %w", err)
	}
	template.TemplateData = templateData

	return &template, nil
}

// ListTemplatesByProject lists templates for a project with pagination
func (r *ConfigTemplateRepository) ListTemplatesByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*build.ConfigTemplate, int, error) {
	if projectID == uuid.Nil {
		return nil, 0, build.ErrInvalidBuildID
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM config_templates WHERE project_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, projectID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, project_id, created_by_user_id, name, description, method, template_data, is_shared, is_public, created_at, updated_at
		FROM config_templates
		WHERE project_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, projectID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*build.ConfigTemplate
	for rows.Next() {
		var template build.ConfigTemplate
		var templateDataJSON []byte

		err := rows.Scan(
			&template.ID,
			&template.ProjectID,
			&template.CreatedByUserID,
			&template.Name,
			&template.Description,
			&template.Method,
			&templateDataJSON,
			&template.IsShared,
			&template.IsPublic,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan template: %w", err)
		}

		// Unmarshal JSONB data
		templateData := make(map[string]interface{})
		if err := json.Unmarshal(templateDataJSON, &templateData); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal template data: %w", err)
		}
		template.TemplateData = templateData

		templates = append(templates, &template)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating templates: %w", err)
	}

	return templates, total, nil
}

// UpdateTemplate updates existing template
func (r *ConfigTemplateRepository) UpdateTemplate(ctx context.Context, template *build.ConfigTemplate) error {
	if template == nil || template.ID == uuid.Nil {
		return build.ErrInvalidConfigID
	}

	template.UpdatedAt = time.Now()

	templateDataJSON, err := json.Marshal(template.TemplateData)
	if err != nil {
		return fmt.Errorf("failed to marshal template data: %w", err)
	}

	query := `
		UPDATE config_templates
		SET name = $1, description = $2, method = $3, template_data = $4, is_shared = $5, is_public = $6, updated_at = $7
		WHERE id = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		template.Name,
		template.Description,
		template.Method,
		templateDataJSON,
		template.IsShared,
		template.IsPublic,
		template.UpdatedAt,
		template.ID,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique violation
				return build.ErrDuplicateTemplateName
			}
		}
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return build.ErrTemplateNotFound
	}

	return nil
}

// DeleteTemplate removes template and cascades to shares
func (r *ConfigTemplateRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return build.ErrInvalidConfigID
	}

	query := `DELETE FROM config_templates WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return build.ErrTemplateNotFound
	}

	return nil
}

// ShareTemplate creates share record
func (r *ConfigTemplateRepository) ShareTemplate(ctx context.Context, share *build.ConfigTemplateShare) error {
	if share == nil {
		return build.ErrInvalidManifest
	}

	if share.ID == uuid.Nil {
		share.ID = uuid.New()
	}

	if share.CreatedAt.IsZero() {
		share.CreatedAt = time.Now()
	}

	if share.UpdatedAt.IsZero() {
		share.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO config_template_shares (id, template_id, shared_with_user_id, can_use, can_edit, can_delete, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (template_id, shared_with_user_id) DO NOTHING
	`

	result, err := r.db.ExecContext(ctx, query,
		share.ID,
		share.TemplateID,
		share.SharedWithUserID,
		share.CanUse,
		share.CanEdit,
		share.CanDelete,
		share.CreatedAt,
		share.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" { // foreign key violation
				return build.ErrTemplateNotFound
			}
		}
		return fmt.Errorf("failed to share template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Success even if duplicate (idempotent)
	_ = rowsAffected

	return nil
}

// GetSharesByTemplate retrieves all shares for a template
func (r *ConfigTemplateRepository) GetSharesByTemplate(ctx context.Context, templateID uuid.UUID) ([]*build.ConfigTemplateShare, error) {
	if templateID == uuid.Nil {
		return nil, build.ErrInvalidConfigID
	}

	query := `
		SELECT id, template_id, shared_with_user_id, can_use, can_edit, can_delete, created_at, updated_at
		FROM config_template_shares
		WHERE template_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares: %w", err)
	}
	defer rows.Close()

	var shares []*build.ConfigTemplateShare
	for rows.Next() {
		var share build.ConfigTemplateShare
		err := rows.Scan(
			&share.ID,
			&share.TemplateID,
			&share.SharedWithUserID,
			&share.CanUse,
			&share.CanEdit,
			&share.CanDelete,
			&share.CreatedAt,
			&share.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, &share)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shares: %w", err)
	}

	return shares, nil
}

// GetSharesByUser retrieves templates shared with a user
func (r *ConfigTemplateRepository) GetSharesByUser(ctx context.Context, userID uuid.UUID) ([]*build.ConfigTemplateShare, error) {
	if userID == uuid.Nil {
		return nil, build.ErrInvalidManifest
	}

	query := `
		SELECT id, template_id, shared_with_user_id, can_use, can_edit, can_delete, created_at, updated_at
		FROM config_template_shares
		WHERE shared_with_user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user shares: %w", err)
	}
	defer rows.Close()

	var shares []*build.ConfigTemplateShare
	for rows.Next() {
		var share build.ConfigTemplateShare
		err := rows.Scan(
			&share.ID,
			&share.TemplateID,
			&share.SharedWithUserID,
			&share.CanUse,
			&share.CanEdit,
			&share.CanDelete,
			&share.CreatedAt,
			&share.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, &share)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user shares: %w", err)
	}

	return shares, nil
}

// DeleteShare removes a share record
func (r *ConfigTemplateRepository) DeleteShare(ctx context.Context, templateID, userID uuid.UUID) error {
	if templateID == uuid.Nil || userID == uuid.Nil {
		return build.ErrInvalidManifest
	}

	query := `DELETE FROM config_template_shares WHERE template_id = $1 AND shared_with_user_id = $2`

	result, err := r.db.ExecContext(ctx, query, templateID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete share: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return build.ErrShareNotFound
	}

	return nil
}
