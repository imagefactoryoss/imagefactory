package template

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PostgresTemplateRepository implements TemplateRepository with PostgreSQL and caching
type PostgresTemplateRepository struct {
	db       *sql.DB
	cache    *templateCache
	cacheTTL time.Duration
}

// templateCache is a simple in-memory cache with TTL
type templateCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	template  *EmailTemplate
	expiresAt time.Time
}

// NewPostgresTemplateRepository creates a new PostgreSQL template repository
func NewPostgresTemplateRepository(db *sql.DB, cacheTTL time.Duration) *PostgresTemplateRepository {
	return &PostgresTemplateRepository{
		db:       db,
		cacheTTL: cacheTTL,
		cache: &templateCache{
			items: make(map[string]*cacheItem),
		},
	}
}

// GetTemplate retrieves a template by name, company, tenant, and locale
func (r *PostgresTemplateRepository) GetTemplate(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID, templateName, locale string) (*EmailTemplate, error) {
	// Check cache first
	cacheKey := r.buildCacheKey(companyID, tenantID, templateName, locale)
	if cached := r.cache.get(cacheKey); cached != nil {
		return cached, nil
	}

	// Query database
	query := `
		SELECT 
			id, company_id, tenant_id, template_name, notification_type,
			subject_template, body_template, channel_type, html_template,
			available_variables, is_active, is_default, locale, metadata,
			created_at, updated_at
		FROM notification_templates
		WHERE company_id = $1 
			AND (tenant_id = $2 OR tenant_id IS NULL)
			AND template_name = $3
			AND locale = $4
			AND is_active = true
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	var tmpl EmailTemplate
	var tenantIDPtr *uuid.UUID
	var availableVarsJSON []byte
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, companyID, tenantID, templateName, locale).Scan(
		&tmpl.ID,
		&tmpl.CompanyID,
		&tenantIDPtr,
		&tmpl.TemplateName,
		&tmpl.NotificationType,
		&tmpl.SubjectTemplate,
		&tmpl.BodyTemplate,
		&tmpl.ChannelType,
		&tmpl.HTMLTemplate,
		&availableVarsJSON,
		&tmpl.IsActive,
		&tmpl.IsDefault,
		&tmpl.Locale,
		&metadataJSON,
		&tmpl.CreatedAt,
		&tmpl.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &TemplateNotFoundError{
			CompanyID:    companyID,
			TenantID:     tenantID,
			TemplateName: templateName,
			Locale:       locale,
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	tmpl.TenantID = tenantIDPtr

	// Parse JSON fields
	if availableVarsJSON != nil {
		if err := json.Unmarshal(availableVarsJSON, &tmpl.AvailableVars); err != nil {
			return nil, fmt.Errorf("failed to parse available_variables: %w", err)
		}
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &tmpl.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// Cache the result
	r.cache.set(cacheKey, &tmpl, r.cacheTTL)

	return &tmpl, nil
}

// LoadTemplates loads all templates for a company/tenant combination
func (r *PostgresTemplateRepository) LoadTemplates(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) ([]*EmailTemplate, error) {
	query := `
		SELECT 
			id, company_id, tenant_id, template_name, notification_type,
			subject_template, body_template, channel_type, html_template,
			available_variables, is_active, is_default, locale, metadata,
			created_at, updated_at
		FROM notification_templates
		WHERE company_id = $1 
			AND (tenant_id = $2 OR tenant_id IS NULL)
			AND is_active = true
		ORDER BY template_name, locale
	`

	rows, err := r.db.QueryContext(ctx, query, companyID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []*EmailTemplate

	for rows.Next() {
		var tmpl EmailTemplate
		var tenantIDPtr *uuid.UUID
		var availableVarsJSON []byte
		var metadataJSON []byte

		err := rows.Scan(
			&tmpl.ID,
			&tmpl.CompanyID,
			&tenantIDPtr,
			&tmpl.TemplateName,
			&tmpl.NotificationType,
			&tmpl.SubjectTemplate,
			&tmpl.BodyTemplate,
			&tmpl.ChannelType,
			&tmpl.HTMLTemplate,
			&availableVarsJSON,
			&tmpl.IsActive,
			&tmpl.IsDefault,
			&tmpl.Locale,
			&metadataJSON,
			&tmpl.CreatedAt,
			&tmpl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		tmpl.TenantID = tenantIDPtr

		// Parse JSON fields
		if availableVarsJSON != nil {
			if err := json.Unmarshal(availableVarsJSON, &tmpl.AvailableVars); err != nil {
				return nil, fmt.Errorf("failed to parse available_variables: %w", err)
			}
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &tmpl.Metadata); err != nil {
				return nil, fmt.Errorf("failed to parse metadata: %w", err)
			}
		}

		templates = append(templates, &tmpl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating templates: %w", err)
	}

	return templates, nil
}

// CreateTemplate creates a new template
func (r *PostgresTemplateRepository) CreateTemplate(ctx context.Context, template *EmailTemplate) error {
	availableVarsJSON, err := json.Marshal(template.AvailableVars)
	if err != nil {
		return fmt.Errorf("failed to marshal available_variables: %w", err)
	}

	metadataJSON, err := json.Marshal(template.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO notification_templates (
			company_id, tenant_id, template_name, notification_type,
			subject_template, body_template, channel_type, html_template,
			available_variables, is_active, is_default, locale, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		template.CompanyID,
		template.TenantID,
		template.TemplateName,
		template.NotificationType,
		template.SubjectTemplate,
		template.BodyTemplate,
		template.ChannelType,
		template.HTMLTemplate,
		availableVarsJSON,
		template.IsActive,
		template.IsDefault,
		template.Locale,
		metadataJSON,
	).Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// UpdateTemplate updates an existing template
func (r *PostgresTemplateRepository) UpdateTemplate(ctx context.Context, template *EmailTemplate) error {
	availableVarsJSON, err := json.Marshal(template.AvailableVars)
	if err != nil {
		return fmt.Errorf("failed to marshal available_variables: %w", err)
	}

	metadataJSON, err := json.Marshal(template.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE notification_templates SET
			subject_template = $1,
			body_template = $2,
			channel_type = $3,
			html_template = $4,
			available_variables = $5,
			is_active = $6,
			is_default = $7,
			locale = $8,
			metadata = $9,
			updated_at = NOW()
		WHERE id = $10
		RETURNING updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		template.SubjectTemplate,
		template.BodyTemplate,
		template.ChannelType,
		template.HTMLTemplate,
		availableVarsJSON,
		template.IsActive,
		template.IsDefault,
		template.Locale,
		metadataJSON,
		template.ID,
	).Scan(&template.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("template not found: %s", template.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// Invalidate cache
	if template.TenantID != nil {
		r.InvalidateCache(template.CompanyID, *template.TenantID, template.TemplateName)
	}

	return nil
}

// DeleteTemplate deletes a template
func (r *PostgresTemplateRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notification_templates WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("template not found: %s", id)
	}

	return nil
}

// InvalidateCache manually invalidates cached templates
func (r *PostgresTemplateRepository) InvalidateCache(companyID uuid.UUID, tenantID uuid.UUID, templateName string) {
	// Invalidate all locales for this template
	r.cache.mu.Lock()
	defer r.cache.mu.Unlock()

	prefix := fmt.Sprintf("%s:%s:%s:", companyID, tenantID, templateName)
	for key := range r.cache.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(r.cache.items, key)
		}
	}
}

// buildCacheKey creates a cache key from template identifiers
func (r *PostgresTemplateRepository) buildCacheKey(companyID uuid.UUID, tenantID uuid.UUID, templateName, locale string) string {
	return fmt.Sprintf("%s:%s:%s:%s", companyID, tenantID, templateName, locale)
}

// Cache methods

func (c *templateCache) get(key string) *EmailTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		return nil
	}

	return item.template
}

func (c *templateCache) set(key string, template *EmailTemplate, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		template:  template,
		expiresAt: time.Now().Add(ttl),
	}
}

// Background cleanup goroutine (optional, can be started separately)
func (c *templateCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}
