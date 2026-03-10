package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/tenant"
)

// TenantRepository implements the tenant.Repository interface
type TenantRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
	domainColumn sync.Once
	hasDomainCol bool
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sqlx.DB, logger *zap.Logger) *TenantRepository {
	return &TenantRepository{
		db:     db,
		logger: logger,
	}
}

// tenantModel represents the database model for tenant
type tenantModel struct {
	ID         uuid.UUID `db:"id"`
	TenantCode string    `db:"tenant_code"`
	Name       string    `db:"name"`
	Slug       string    `db:"slug"`
	Description string   `db:"description"`
	Status     string    `db:"status"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// Save persists a tenant
func (r *TenantRepository) Save(ctx context.Context, t *tenant.Tenant) error {
	var err error
	if r.hasTenantDomainColumn(ctx) {
		query := `
			INSERT INTO tenants (id, tenant_code, name, slug, domain, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err = r.db.ExecContext(ctx, query,
			t.ID(),
			t.TenantCode(),
			t.Name(),
			t.Slug(),
			r.tenantDomainFromSlug(t.Slug()),
			t.Description(),
			string(t.Status()),
			t.CreatedAt(),
			t.UpdatedAt(),
		)
	} else {
		query := `
			INSERT INTO tenants (id, tenant_code, name, slug, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = r.db.ExecContext(ctx, query,
			t.ID(),
			t.TenantCode(),
			t.Name(),
			t.Slug(),
			t.Description(),
			string(t.Status()),
			t.CreatedAt(),
			t.UpdatedAt(),
		)
	}

	if err != nil {
		r.logger.Error("Failed to save tenant",
			zap.String("tenant_id", t.ID().String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	r.logger.Info("Tenant saved successfully",
		zap.String("tenant_id", t.ID().String()),
		zap.String("tenant_name", t.Name()),
	)

	return nil
}

func (r *TenantRepository) hasTenantDomainColumn(ctx context.Context) bool {
	r.domainColumn.Do(func() {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = 'tenants'
				  AND column_name = 'domain'
			)
		`
		if err := r.db.GetContext(ctx, &exists, query); err != nil {
			r.logger.Warn("Failed to detect tenants.domain column; defaulting to legacy schema", zap.Error(err))
			r.hasDomainCol = false
			return
		}
		r.hasDomainCol = exists
	})
	return r.hasDomainCol
}

func (r *TenantRepository) tenantDomainFromSlug(slug string) string {
	normalized := strings.TrimSpace(strings.ToLower(slug))
	if normalized == "" {
		return "tenant.local"
	}
	return normalized + ".local"
}

// FindByID retrieves a tenant by ID
func (r *TenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	var model tenantModel

	query := `
		SELECT id, tenant_code, name, slug, description, status, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find tenant by ID",
			zap.String("tenant_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	return r.modelToDomain(&model)
}

// FindBySlug retrieves a tenant by slug
func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	var model tenantModel

	query := `
		SELECT id, tenant_code, name, slug, description, status, created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`

	err := r.db.GetContext(ctx, &model, query, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find tenant by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	return r.modelToDomain(&model)
}

// FindAll retrieves all tenants with optional filtering
func (r *TenantRepository) FindAll(ctx context.Context, filter tenant.TenantFilter) ([]*tenant.Tenant, error) {
	var models []tenantModel

	query := `
		SELECT DISTINCT t.id, t.tenant_code, t.name, t.slug, t.description, t.status, t.created_at, t.updated_at
		FROM tenants t
	`

	args := []interface{}{}
	whereClause := ""
	argCount := 0

	// Add user ownership filter if specified
	if filter.UserID != nil {
		query += `
			LEFT JOIN user_role_assignments ura ON ura.tenant_id = t.id AND ura.user_id = $1
			LEFT JOIN rbac_roles r ON r.id = ura.role_id
			LEFT JOIN group_members gm ON gm.user_id = $1 AND gm.removed_at IS NULL
			LEFT JOIN tenant_groups tg ON tg.id = gm.group_id AND tg.tenant_id = t.id AND tg.role_type IN ('owner', 'developer', 'operator')
		`
		args = append(args, filter.UserID.String())
		argCount++

		// User must have owner/administrator role OR be in an owner/developer/operator group for the tenant
		if whereClause == "" {
			whereClause += " WHERE"
		} else {
			whereClause += " AND"
		}
		whereClause += ` (
			r.name IN ('Owner', 'Administrator')
			OR tg.id IS NOT NULL
			OR t.id IN (
				SELECT ura2.tenant_id FROM user_role_assignments ura2
				JOIN rbac_roles r2 ON r2.id = ura2.role_id
				WHERE ura2.user_id = $` + strconv.Itoa(argCount) + ` AND r2.name IN ('Owner', 'Administrator')
			)
		)`
	}

	// Add filtering conditions
	if filter.Status != nil {
		if whereClause == "" {
			whereClause += " WHERE"
		} else {
			whereClause += " AND"
		}
		whereClause += " t.status = $" + strconv.Itoa(argCount+1)
		args = append(args, string(*filter.Status))
		argCount++
	}

	query += whereClause

	// Add ordering
	query += " ORDER BY t.created_at DESC"

	// Add pagination
	if filter.Limit > 0 {
		query += " LIMIT $" + strconv.Itoa(argCount+1)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += " OFFSET $" + strconv.Itoa(argCount+1)
		args = append(args, filter.Offset)
	}

	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		r.logger.Error("Failed to find all tenants",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find tenants: %w", err)
	}

	tenants := make([]*tenant.Tenant, len(models))
	for i, model := range models {
		t, err := r.modelToDomain(&model)
		if err != nil {
			return nil, err
		}
		tenants[i] = t
	}

	return tenants, nil
}

// Update updates an existing tenant
func (r *TenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, slug = $3, description = $4, status = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		t.ID(),
		t.Name(),
		t.Slug(),
		t.Description(),
		string(t.Status()),
		t.UpdatedAt(),
	)

	if err != nil {
		r.logger.Error("Failed to update tenant",
			zap.String("tenant_id", t.ID().String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found or version conflict")
	}

	r.logger.Info("Tenant updated successfully",
		zap.String("tenant_id", t.ID().String()),
	)

	return nil
}

// Delete removes a tenant
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tenants WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete tenant",
			zap.String("tenant_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	r.logger.Info("Tenant deleted successfully",
		zap.String("tenant_id", id.String()),
	)

	return nil
}

// ExistsBySlug checks if a tenant exists by slug
func (r *TenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int

	query := `SELECT COUNT(*) FROM tenants WHERE slug = $1`

	err := r.db.GetContext(ctx, &count, query, slug)
	if err != nil {
		r.logger.Error("Failed to check tenant existence",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check tenant existence: %w", err)
	}

	return count > 0, nil
}

// modelToDomain converts database model to domain entity
func (r *TenantRepository) modelToDomain(model *tenantModel) (*tenant.Tenant, error) {
	// Create default values for fields that don't exist in the database schema
	defaultQuota := tenant.ResourceQuota{
		MaxBuilds:         100,
		MaxImages:         500,
		MaxStorageGB:      100.0,
		MaxConcurrentJobs: 5,
	}

	defaultConfig := tenant.TenantConfig{
		BuildTimeout:         30 * time.Minute,
		AllowedImageTypes:    []string{"container", "vm"},
		SecurityPolicies:     make(map[string]interface{}),
		NotificationSettings: make(map[string]interface{}),
	}

	status := tenant.TenantStatus(model.Status)

	return tenant.NewTenantFromExisting(
		model.ID,
		0, // numericID not used
		uuid.Nil, // companyID not in schema
		model.TenantCode,
		model.Name,
		model.Slug,
		model.Description,
		status,
		defaultQuota,
		defaultConfig,
		model.CreatedAt,
		model.UpdatedAt,
		1, // version always 1 for now
	)
}

// GetTotalTenantCount returns the total number of tenants in the system
func (r *TenantRepository) GetTotalTenantCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tenants`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		r.logger.Error("Failed to get total tenant count", zap.Error(err))
		return 0, fmt.Errorf("failed to get total tenant count: %w", err)
	}

	return count, nil
}

// GetActiveTenantCount returns the number of active tenants in the system
func (r *TenantRepository) GetActiveTenantCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tenants WHERE status = 'active'`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		r.logger.Error("Failed to get active tenant count", zap.Error(err))
		return 0, fmt.Errorf("failed to get active tenant count: %w", err)
	}

	return count, nil
}
