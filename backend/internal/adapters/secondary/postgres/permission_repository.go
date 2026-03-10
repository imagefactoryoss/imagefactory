package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/rbac"
)

// PermissionRepository handles permission-related database operations
type PermissionRepository struct {
	db *sql.DB
}

// NewPermissionRepository creates a new permission repository
func NewPermissionRepository(db *sql.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// FindByResourceAction retrieves a permission by resource and action
func (pr *PermissionRepository) FindByResourceAction(ctx context.Context, resource, action string) (*rbac.PermissionRecord, error) {
	query := `
		SELECT id, resource, action, description, category, is_system_permission, created_at, updated_at
		FROM permissions
		WHERE resource = $1 AND action = $2
		LIMIT 1
	`

	var perm rbac.PermissionRecord
	err := pr.db.QueryRowContext(ctx, query, resource, action).Scan(
		&perm.ID,
		&perm.Resource,
		&perm.Action,
		&perm.Description,
		&perm.Category,
		&perm.IsSystemPermission,
		&perm.CreatedAt,
		&perm.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query permission: %w", err)
	}

	return &perm, nil
}

// FindAll retrieves all permissions from the database
func (pr *PermissionRepository) FindAll(ctx context.Context) ([]*rbac.PermissionRecord, error) {
	query := `
		SELECT id, resource, action, description, category, is_system_permission, created_at, updated_at
		FROM permissions
		ORDER BY resource, action
	`

	rows, err := pr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*rbac.PermissionRecord
	for rows.Next() {
		var perm rbac.PermissionRecord
		err := rows.Scan(
			&perm.ID,
			&perm.Resource,
			&perm.Action,
			&perm.Description,
			&perm.Category,
			&perm.IsSystemPermission,
			&perm.CreatedAt,
			&perm.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &perm)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permissions: %w", err)
	}

	return permissions, nil
}

// FindByResource retrieves all permissions for a specific resource
func (pr *PermissionRepository) FindByResource(ctx context.Context, resource string) ([]*rbac.PermissionRecord, error) {
	query := `
		SELECT id, resource, action, description, category, is_system_permission, created_at, updated_at
		FROM permissions
		WHERE resource = $1
		ORDER BY action
	`

	rows, err := pr.db.QueryContext(ctx, query, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions for resource '%s': %w", resource, err)
	}
	defer rows.Close()

	var permissions []*rbac.PermissionRecord
	for rows.Next() {
		var perm rbac.PermissionRecord
		err := rows.Scan(
			&perm.ID,
			&perm.Resource,
			&perm.Action,
			&perm.Description,
			&perm.Category,
			&perm.IsSystemPermission,
			&perm.CreatedAt,
			&perm.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &perm)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permissions: %w", err)
	}

	return permissions, nil
}

// GetResourceList returns all unique resources from the permissions table
func (pr *PermissionRepository) GetResourceList(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT resource
		FROM permissions
		ORDER BY resource
	`

	rows, err := pr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query resources: %w", err)
	}
	defer rows.Close()

	var resources []string
	for rows.Next() {
		var resource string
		if err := rows.Scan(&resource); err != nil {
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}
		resources = append(resources, resource)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating resources: %w", err)
	}

	return resources, nil
}

// CountByResource returns the count of permissions for a specific resource
func (pr *PermissionRepository) CountByResource(ctx context.Context, resource string) (int, error) {
	query := `SELECT COUNT(*) FROM permissions WHERE resource = $1`

	var count int
	err := pr.db.QueryRowContext(ctx, query, resource).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count permissions for resource '%s': %w", resource, err)
	}

	return count, nil
}

// Create creates a new permission in the database
func (pr *PermissionRepository) Create(ctx context.Context, perm *rbac.PermissionRecord) (*rbac.PermissionRecord, error) {
	query := `
		INSERT INTO permissions (id, resource, action, description, category, is_system_permission, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, resource, action, description, category, is_system_permission, created_at, updated_at
	`

	var created rbac.PermissionRecord
	err := pr.db.QueryRowContext(ctx, query,
		perm.ID,
		perm.Resource,
		perm.Action,
		perm.Description,
		perm.Category,
		perm.IsSystemPermission,
		perm.CreatedAt,
		perm.UpdatedAt,
	).Scan(
		&created.ID,
		&created.Resource,
		&created.Action,
		&created.Description,
		&created.Category,
		&created.IsSystemPermission,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return &created, nil
}

// Update updates an existing permission in the database
func (pr *PermissionRepository) Update(ctx context.Context, perm *rbac.PermissionRecord) (*rbac.PermissionRecord, error) {
	query := `
		UPDATE permissions
		SET description = $1, category = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, resource, action, description, category, is_system_permission, created_at, updated_at
	`

	var updated rbac.PermissionRecord
	err := pr.db.QueryRowContext(ctx, query,
		perm.Description,
		perm.Category,
		perm.UpdatedAt,
		perm.ID,
	).Scan(
		&updated.ID,
		&updated.Resource,
		&updated.Action,
		&updated.Description,
		&updated.Category,
		&updated.IsSystemPermission,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	return &updated, nil
}

// Delete deletes a permission from the database
func (pr *PermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM permissions WHERE id = $1`

	result, err := pr.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("permission with ID %s not found", id)
	}

	return nil
}
