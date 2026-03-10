package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/rbac"
)

// RBACRepository implements the rbac.Repository interface
type RBACRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewRBACRepository creates a new RBAC repository
func NewRBACRepository(db *sqlx.DB, logger *zap.Logger) *RBACRepository {
	return &RBACRepository{
		db:     db,
		logger: logger,
	}
}

// roleModel represents the database model for role
type roleModel struct {
	ID           uuid.UUID `db:"id"`
	CompanyID    uuid.UUID `db:"company_id"`
	Name         string    `db:"name"`
	Description  string    `db:"description"`
	Scope        string    `db:"scope"`
	IsSystemRole bool      `db:"is_system_role"`
	Status       string    `db:"status"`
	CreatedAt    string    `db:"created_at"`
	UpdatedAt    string    `db:"updated_at"`
}

// userRoleModel represents the database model for user-role assignments
type userRoleModel struct {
	UserID     uuid.UUID `db:"user_id"`
	RoleID     uuid.UUID `db:"role_id"`
	AssignedAt string    `db:"assigned_at"`
	AssignedBy uuid.UUID `db:"assigned_by"`
}

// UserWithRoles represents a user with their roles for optimized queries
type UserWithRoles struct {
	ID         uuid.UUID      `db:"id"`
	TenantID   uuid.UUID      `db:"tenant_id"`
	Email      string         `db:"email"`
	FirstName  string         `db:"first_name"`
	LastName   string         `db:"last_name"`
	Status     string         `db:"status"`
	IsActive   bool           `db:"is_active"`
	AuthMethod string         `db:"auth_method"`
	Roles      []RoleWithMeta `db:"roles"`
}

// RoleWithMeta represents role metadata for user queries
type RoleWithMeta struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions string    `json:"permissions"`
	IsSystem    bool      `json:"is_system"`
}

// SaveRole persists a role
func (r *RBACRepository) SaveRole(ctx context.Context, role *rbac.Role) error {
	// Insert or update the role itself (without permissions JSON)
	query := `
		INSERT INTO rbac_roles (id, tenant_id, name, description, is_system, created_at, updated_at, version)
		VALUES ($1, NULLIF($2::uuid, $9::uuid), $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, name) DO UPDATE SET 
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at,
			version = EXCLUDED.version + 1
		RETURNING id
	`

	var roleID uuid.UUID
	err := r.db.QueryRowContext(ctx, query,
		role.ID(),
		role.TenantID(),
		role.Name(),
		role.Description(),
		role.IsSystem(),
		role.CreatedAt(),
		role.UpdatedAt(),
		role.Version(),
		uuid.Nil,
	).Scan(&roleID)

	if err != nil {
		r.logger.Error("Failed to save role",
			zap.String("role_id", role.ID().String()),
			zap.String("role_name", role.Name()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save role: %w", err)
	}

	// Delete old permission assignments
	_, err = r.db.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		r.logger.Error("Failed to delete old role permissions",
			zap.String("role_id", roleID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete old role permissions: %w", err)
	}

	// Insert new permission assignments via junction table
	for _, perm := range role.Permissions() {
		insertPermQuery := `
			INSERT INTO role_permissions (role_id, permission_id, granted_at)
			VALUES ($1, (SELECT id FROM permissions WHERE resource = $2 AND action = $3), $4)
		`
		_, err = r.db.ExecContext(ctx, insertPermQuery, roleID, perm.Resource, perm.Action, time.Now())
		if err != nil {
			r.logger.Error("Failed to assign permission to role",
				zap.String("role_id", roleID.String()),
				zap.String("resource", perm.Resource),
				zap.String("action", perm.Action),
				zap.Error(err),
			)
			return fmt.Errorf("failed to assign permission to role: %w", err)
		}
	}

	r.logger.Info("Role saved successfully with permissions",
		zap.String("role_id", roleID.String()),
		zap.String("role_name", role.Name()),
		zap.Int("permission_count", len(role.Permissions())),
	)

	return nil
}

// FindRoleByID retrieves a role by ID
func (r *RBACRepository) FindRoleByID(ctx context.Context, id uuid.UUID) (*rbac.Role, error) {
	var model roleModel

	query := `
		SELECT id, COALESCE(tenant_id, $2::uuid) as company_id, 
		       name, description, is_system, created_at, updated_at
		FROM rbac_roles
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id, uuid.Nil).Scan(
		&model.ID, &model.CompanyID, &model.Name, &model.Description,
		&model.IsSystemRole, &model.CreatedAt, &model.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, rbac.ErrRoleNotFound
		}
		r.logger.Error("Failed to find role by ID",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find role by ID: %w", err)
	}

	// Load permissions from junction table
	permissions, err := r.getRolePermissionsFromJunction(ctx, id)
	if err != nil {
		r.logger.Error("Failed to load permissions for role",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		// Return error instead of proceeding with empty permissions
		return nil, fmt.Errorf("failed to load role permissions: %w", err)
	}

	r.logger.Debug("FindRoleByID: Retrieved role from DB",
		zap.String("role_id", id.String()),
		zap.Int("permission_count", len(permissions)),
	)

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, model.CreatedAt)
	if err != nil {
		createdAt = time.Now().UTC()
	}
	updatedAt, err := time.Parse(time.RFC3339, model.UpdatedAt)
	if err != nil {
		updatedAt = time.Now().UTC()
	}

	return rbac.NewRoleFromExisting(
		model.ID,
		model.CompanyID,
		model.Name,
		model.Description,
		permissions,
		model.IsSystemRole,
		createdAt,
		updatedAt,
		1,
	)
}

// FindRoleByName retrieves a role by name and tenant
func (r *RBACRepository) FindRoleByName(ctx context.Context, tenantID uuid.UUID, name string) (*rbac.Role, error) {
	var id uuid.UUID
	var companyID uuid.UUID
	var description string
	var isSystem bool
	var createdAt, updatedAt string

	query := `
		SELECT id, COALESCE(tenant_id, $3::uuid),
		       name, description, is_system, created_at, updated_at
		FROM rbac_roles
		WHERE (tenant_id = $1 OR (tenant_id IS NULL AND $1::uuid IS NULL)) AND name = $2
	`

	err := r.db.QueryRowContext(ctx, query, tenantID, name, uuid.Nil).Scan(
		&id, &companyID, &name, &description, &isSystem, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, rbac.ErrRoleNotFound
		}
		r.logger.Error("Failed to find role by name",
			zap.String("tenant_id", tenantID.String()),
			zap.String("role_name", name),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find role by name: %w", err)
	}

	// Load permissions from junction table
	permissions, err := r.getRolePermissionsFromJunction(ctx, id)
	if err != nil {
		r.logger.Error("Failed to load permissions for role",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to load role permissions: %w", err)
	}

	// Parse timestamps
	createdTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		createdTime = time.Now().UTC()
	}
	updatedTime, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		updatedTime = time.Now().UTC()
	}

	return rbac.NewRoleFromExisting(id, companyID, name, description, permissions, isSystem, createdTime, updatedTime, 1)
}

// FindRolesByTenantID retrieves all roles for a tenant
func (r *RBACRepository) FindRolesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*rbac.Role, error) {
	query := `
		SELECT id, COALESCE(tenant_id, $2::uuid),
		       name, description, is_system, created_at, updated_at
		FROM rbac_roles
		WHERE tenant_id = $1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, uuid.Nil)
	if err != nil {
		r.logger.Error("Failed to find roles by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find roles by tenant ID: %w", err)
	}
	defer rows.Close()

	var roles []*rbac.Role
	for rows.Next() {
		var id uuid.UUID
		var companyID uuid.UUID
		var name, description string
		var isSystem bool
		var createdAt, updatedAt string

		err := rows.Scan(&id, &companyID, &name, &description, &isSystem, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		// Load permissions from junction table
		permissions, err := r.getRolePermissionsFromJunction(ctx, id)
		if err != nil {
			r.logger.Warn("Failed to load permissions for role",
				zap.String("role_id", id.String()),
				zap.Error(err),
			)
			continue
		}

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			createdTime = time.Now().UTC()
		}
		updatedTime, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			updatedTime = time.Now().UTC()
		}

		role, err := rbac.NewRoleFromExisting(id, companyID, name, description, permissions, isSystem, createdTime, updatedTime, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create role: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate roles: %w", err)
	}

	return roles, nil
}

// FindAssignableRoleTypesByTenant retrieves active tenant group role_type values used to scope assignable roles.
func (r *RBACRepository) FindAssignableRoleTypesByTenant(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT LOWER(role_type) AS role_type
		FROM tenant_groups
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND role_type IS NOT NULL
		  AND role_type <> ''
		ORDER BY role_type
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to find assignable role types by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find assignable role types by tenant ID: %w", err)
	}
	defer rows.Close()

	roleTypes := make([]string, 0)
	for rows.Next() {
		var roleType string
		if err := rows.Scan(&roleType); err != nil {
			return nil, fmt.Errorf("failed to scan role type: %w", err)
		}
		roleTypes = append(roleTypes, roleType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate role types: %w", err)
	}

	return roleTypes, nil
}

// FindSystemRoles retrieves all system roles
func (r *RBACRepository) FindSystemRoles(ctx context.Context) ([]*rbac.Role, error) {
	query := `
		SELECT id, COALESCE(tenant_id, $1::uuid),
		       name, description, is_system, created_at, updated_at
		FROM rbac_roles
		WHERE is_system = true
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, uuid.Nil)
	if err != nil {
		r.logger.Error("Failed to find system roles", zap.Error(err))
		return nil, fmt.Errorf("failed to find system roles: %w", err)
	}
	defer rows.Close()

	var roles []*rbac.Role
	for rows.Next() {
		var id uuid.UUID
		var companyID uuid.UUID
		var name, description string
		var isSystem bool
		var createdAt, updatedAt string

		err := rows.Scan(&id, &companyID, &name, &description, &isSystem, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		// Load permissions from junction table
		permissions, err := r.getRolePermissionsFromJunction(ctx, id)
		if err != nil {
			r.logger.Warn("Failed to load permissions for role",
				zap.String("role_id", id.String()),
				zap.Error(err),
			)
			continue
		}

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			createdTime = time.Now().UTC()
		}
		updatedTime, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			updatedTime = time.Now().UTC()
		}

		role, err := rbac.NewRoleFromExisting(id, companyID, name, description, permissions, isSystem, createdTime, updatedTime, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create role: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate system roles: %w", err)
	}

	return roles, nil
}

// FindAllSystemLevelRoles retrieves all roles at the system level (where tenant_id IS NULL)
// This includes both system roles (is_system=true) and tenant-level roles configured at the system level
// Optimized: Uses batch query to load permissions for all roles in a single database call (eliminates N+1)
func (r *RBACRepository) FindAllSystemLevelRoles(ctx context.Context) ([]*rbac.Role, error) {
	query := `
		SELECT id, COALESCE(tenant_id, $1::uuid),
		       name, description, is_system, created_at, updated_at
		FROM rbac_roles
		WHERE tenant_id IS NULL
		ORDER BY CASE WHEN is_system THEN 0 ELSE 1 END, name
	`

	rows, err := r.db.QueryContext(ctx, query, uuid.Nil)
	if err != nil {
		r.logger.Error("Failed to find system-level roles", zap.Error(err))
		return nil, fmt.Errorf("failed to find system-level roles: %w", err)
	}
	defer rows.Close()

	var roles []*rbac.Role
	var roleIDs []uuid.UUID                   // Collect role IDs for batch permission loading
	roleMap := make(map[uuid.UUID]*rbac.Role) // Map to maintain order while assigning permissions

	for rows.Next() {
		var id uuid.UUID
		var companyID uuid.UUID
		var name, description string
		var isSystem bool
		var createdAt, updatedAt string

		err := rows.Scan(&id, &companyID, &name, &description, &isSystem, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		roleIDs = append(roleIDs, id)

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			createdTime = time.Now().UTC()
		}
		updatedTime, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			updatedTime = time.Now().UTC()
		}

		// Create role with empty permissions initially
		role, err := rbac.NewRoleFromExisting(id, companyID, name, description, []rbac.Permission{}, isSystem, createdTime, updatedTime, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to create role: %w", err)
		}

		roles = append(roles, role)
		roleMap[id] = role
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate system-level roles: %w", err)
	}

	// Batch load permissions for all roles in ONE query (eliminates N+1 problem)
	if len(roleIDs) > 0 {
		permissionsMap, err := r.getRolePermissionsFromJunctionBatch(ctx, roleIDs)
		if err != nil {
			r.logger.Warn("Failed to load permissions for system-level roles",
				zap.Int("role_count", len(roleIDs)),
				zap.Error(err),
			)
			// Continue without permissions rather than failing completely
		} else {
			// Assign permissions to each role by reassigning with permissions
			for i, role := range roles {
				if perms, exists := permissionsMap[role.ID()]; exists && len(perms) > 0 {
					// Recreate the role with the loaded permissions
					rolesWithPerms, err := rbac.NewRoleFromExisting(
						role.ID(),
						role.TenantID(),
						role.Name(),
						role.Description(),
						perms,
						role.IsSystem(),
						role.CreatedAt(),
						role.UpdatedAt(),
						role.Version(),
					)
					if err == nil {
						roles[i] = rolesWithPerms
					}
				}
			}
		}
	}

	return roles, nil
}

// UpdateRole updates an existing role
func (r *RBACRepository) UpdateRole(ctx context.Context, role *rbac.Role) error {
	// Update the role itself (without permissions JSON)
	query := `
		UPDATE rbac_roles SET
			name = $2, description = $3, updated_at = $4, version = version + 1
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		role.ID(),
		role.Name(),
		role.Description(),
		role.UpdatedAt(),
	)

	if err != nil {
		r.logger.Error("Failed to update role",
			zap.String("role_id", role.ID().String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role not found")
	}

	// Delete old permission assignments
	_, err = r.db.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id = $1", role.ID())
	if err != nil {
		r.logger.Error("Failed to delete old role permissions",
			zap.String("role_id", role.ID().String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete old role permissions: %w", err)
	}

	// Insert new permission assignments via junction table
	for _, perm := range role.Permissions() {
		insertPermQuery := `
			INSERT INTO role_permissions (role_id, permission_id, granted_at)
			VALUES ($1, (SELECT id FROM permissions WHERE resource = $2 AND action = $3), $4)
		`
		_, err = r.db.ExecContext(ctx, insertPermQuery, role.ID(), perm.Resource, perm.Action, time.Now())
		if err != nil {
			r.logger.Error("Failed to assign permission to role",
				zap.String("role_id", role.ID().String()),
				zap.String("resource", perm.Resource),
				zap.String("action", perm.Action),
				zap.Error(err),
			)
			return fmt.Errorf("failed to assign permission to role: %w", err)
		}
	}

	r.logger.Info("Role updated successfully with permissions",
		zap.String("role_id", role.ID().String()),
		zap.String("role_name", role.Name()),
		zap.Int("permission_count", len(role.Permissions())),
	)

	return nil
}

// DeleteRole removes a role and cascades cleanup of user_role_assignments
func (r *RBACRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	// Start transaction to ensure atomic cleanup
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to start transaction for role deletion",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, remove all user role assignments for this role
	deleteAssignmentsQuery := `DELETE FROM user_role_assignments WHERE role_id = $1`
	result, err := tx.ExecContext(ctx, deleteAssignmentsQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete user role assignments",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete user role assignments: %w", err)
	}

	assignmentsDeleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for assignments: %w", err)
	}

	if assignmentsDeleted > 0 {
		r.logger.Info("Cleaned up user role assignments during role deletion",
			zap.String("role_id", id.String()),
			zap.Int64("assignments_deleted", assignmentsDeleted),
		)
	}

	// Then delete the role itself
	deleteRoleQuery := `DELETE FROM rbac_roles WHERE id = $1`
	result, err = tx.ExecContext(ctx, deleteRoleQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete role",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return rbac.ErrRoleNotFound
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit role deletion transaction",
			zap.String("role_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Role deleted successfully with cascading cleanup",
		zap.String("role_id", id.String()),
		zap.Int64("user_assignments_removed", assignmentsDeleted),
	)

	return nil
}

// AssignRoleToUser assigns a role to a user
func (r *RBACRepository) AssignRoleToUser(ctx context.Context, assignment *rbac.UserRoleAssignment) error {
	query := `
		INSERT INTO user_role_assignments (user_id, role_id, tenant_id, assigned_at, assigned_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, role_id, tenant_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.UserID,
		assignment.RoleID,
		assignment.TenantID,
		assignment.AssignedAt,
		assignment.AssignedBy,
	)

	if err != nil {
		r.logger.Error("Failed to assign role to user",
			zap.String("user_id", assignment.UserID.String()),
			zap.String("role_id", assignment.RoleID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	r.logger.Info("Role assigned to user successfully",
		zap.String("user_id", assignment.UserID.String()),
		zap.String("role_id", assignment.RoleID.String()),
	)

	return nil
}

// RemoveRoleFromUser removes a role assignment from a user (all tenants)
func (r *RBACRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_role_assignments WHERE user_id = $1 AND role_id = $2`

	result, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		r.logger.Error("Failed to remove role from user",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role assignment not found")
	}

	r.logger.Info("Role removed from user successfully",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()),
	)

	return nil
}

// RemoveRoleFromUserForTenant removes a role assignment from a user for a specific tenant
func (r *RBACRepository) RemoveRoleFromUserForTenant(ctx context.Context, userID, roleID, tenantID uuid.UUID) error {
	query := `DELETE FROM user_role_assignments WHERE user_id = $1 AND role_id = $2 AND tenant_id = $3`

	result, err := r.db.ExecContext(ctx, query, userID, roleID, tenantID)
	if err != nil {
		r.logger.Error("Failed to remove role from user for tenant",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to remove role from user for tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role assignment not found for this tenant")
	}

	r.logger.Info("Role removed from user for tenant successfully",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}

// FindUserRoles retrieves all roles assigned to a user with optimized query
// Includes both direct role assignments and roles derived from group memberships
func (r *RBACRepository) FindUserRoles(ctx context.Context, userID uuid.UUID) ([]*rbac.Role, error) {
	query := `
		-- Roles from group memberships (map group role_type to system-level role)
		-- All roles are now system-level (tenant_id IS NULL)
		-- Groups are tenant-scoped, so users get tenant context via group membership
		SELECT DISTINCT rr.id, tg.tenant_id,  -- Use group's tenant_id for tenant context
		       rr.name, rr.description, rr.is_system, rr.created_at, rr.updated_at
		FROM rbac_roles rr
		INNER JOIN tenant_groups tg ON REPLACE(LOWER(rr.name), ' ', '_') = tg.role_type AND rr.tenant_id IS NULL
		INNER JOIN group_members gm ON tg.id = gm.group_id
		WHERE gm.user_id = $1 
		  AND gm.removed_at IS NULL
		  AND tg.status = 'active'

		UNION

		-- Legacy: Direct role assignments (for backward compatibility)
		SELECT DISTINCT r.id, COALESCE(ura.tenant_id, $2::uuid),
		       r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM rbac_roles r
		INNER JOIN user_role_assignments ura ON r.id = ura.role_id
		WHERE ura.user_id = $1

		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, userID, uuid.Nil)
	if err != nil {
		r.logger.Error("Failed to find user roles",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user roles: %w", err)
	}
	defer rows.Close()

	// First pass: collect role data and IDs
	type roleData struct {
		id        uuid.UUID
		companyID uuid.UUID
		name      string
		desc      string
		isSystem  bool
		createdAt string
		updatedAt string
	}
	var roleDatas []roleData
	roleIDs := make([]uuid.UUID, 0)

	for rows.Next() {
		var rd roleData
		err := rows.Scan(&rd.id, &rd.companyID, &rd.name, &rd.desc, &rd.isSystem, &rd.createdAt, &rd.updatedAt)
		if err != nil {
			r.logger.Warn("Failed to scan user role",
				zap.String("user_id", userID.String()),
				zap.Error(err),
			)
			continue
		}
		roleDatas = append(roleDatas, rd)
		roleIDs = append(roleIDs, rd.id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate user roles: %w", err)
	}

	// Batch load permissions for all roles
	permissionsMap, err := r.getRolePermissionsFromJunctionBatch(ctx, roleIDs)
	if err != nil {
		r.logger.Warn("Failed to batch load permissions for roles",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		// Continue with empty permissions rather than failing
		permissionsMap = make(map[uuid.UUID][]rbac.Permission)
	}

	// Second pass: create roles with permissions
	var roles []*rbac.Role
	for _, rd := range roleDatas {
		permissions := permissionsMap[rd.id]

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, rd.createdAt)
		if err != nil {
			createdTime = time.Now().UTC()
		}
		updatedTime, err := time.Parse(time.RFC3339, rd.updatedAt)
		if err != nil {
			updatedTime = time.Now().UTC()
		}

		role, err := rbac.NewRoleFromExisting(rd.id, rd.companyID, rd.name, rd.desc, permissions, rd.isSystem, createdTime, updatedTime, 1)
		if err != nil {
			r.logger.Warn("Failed to create role",
				zap.String("role_id", rd.id.String()),
				zap.Error(err),
			)
			continue
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate user roles: %w", err)
	}

	return roles, nil
}

// FindUserRolesForTenant retrieves all roles assigned to a user for a specific tenant
// Includes both direct role assignments and group-based roles
func (r *RBACRepository) FindUserRolesForTenant(ctx context.Context, userID, tenantID uuid.UUID) ([]*rbac.Role, error) {
	query := `
		-- Direct role assignments
		SELECT DISTINCT r.id, COALESCE(r.tenant_id, $2::uuid),
		       r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM rbac_roles r
		INNER JOIN user_role_assignments ura ON r.id = ura.role_id
		WHERE ura.user_id = $1 AND ura.tenant_id = $2

		UNION

		-- Group-based roles (map group role_type to system-level role)
		SELECT DISTINCT rr.id, $2::uuid,
		       rr.name, rr.description, rr.is_system, rr.created_at, rr.updated_at
		FROM rbac_roles rr
		INNER JOIN tenant_groups tg ON REPLACE(LOWER(rr.name), ' ', '_') = tg.role_type AND rr.tenant_id IS NULL
		INNER JOIN group_members gm ON tg.id = gm.group_id
		WHERE gm.user_id = $1 
		  AND gm.removed_at IS NULL
		  AND tg.tenant_id = $2
		  AND tg.status = 'active'

		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, userID, tenantID)
	if err != nil {
		r.logger.Error("Failed to find user roles for tenant",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user roles for tenant: %w", err)
	}
	defer rows.Close()

	// First pass: collect role data and IDs
	type roleData struct {
		id        uuid.UUID
		companyID uuid.UUID
		name      string
		desc      string
		isSystem  bool
		createdAt string
		updatedAt string
	}
	var roleDatas []roleData
	roleIDs := make([]uuid.UUID, 0)

	for rows.Next() {
		var rd roleData
		err := rows.Scan(&rd.id, &rd.companyID, &rd.name, &rd.desc, &rd.isSystem, &rd.createdAt, &rd.updatedAt)
		if err != nil {
			r.logger.Warn("Failed to scan user role for tenant",
				zap.String("user_id", userID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			continue
		}
		roleDatas = append(roleDatas, rd)
		roleIDs = append(roleIDs, rd.id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate user roles for tenant: %w", err)
	}

	// Batch load permissions for all roles
	permissionsMap, err := r.getRolePermissionsFromJunctionBatch(ctx, roleIDs)
	if err != nil {
		r.logger.Warn("Failed to batch load permissions for roles",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		// Continue with empty permissions rather than failing
		permissionsMap = make(map[uuid.UUID][]rbac.Permission)
	}

	// Second pass: create roles with permissions
	var roles []*rbac.Role
	for _, rd := range roleDatas {
		permissions := permissionsMap[rd.id]

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, rd.createdAt)
		if err != nil {
			createdTime = time.Now().UTC()
		}
		updatedTime, err := time.Parse(time.RFC3339, rd.updatedAt)
		if err != nil {
			updatedTime = time.Now().UTC()
		}

		role, err := rbac.NewRoleFromExisting(rd.id, rd.companyID, rd.name, rd.desc, permissions, rd.isSystem, createdTime, updatedTime, 1)
		if err != nil {
			r.logger.Warn("Failed to create role",
				zap.String("role_id", rd.id.String()),
				zap.Error(err),
			)
			continue
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate user roles for tenant: %w", err)
	}

	return roles, nil
}

// FindUserRolesWithUserInfo retrieves user with roles in a single query to avoid N+1
func (r *RBACRepository) FindUserRolesWithUserInfo(ctx context.Context, userID uuid.UUID) (*rbac.UserWithRoles, error) {
	query := `
		SELECT
			u.id, u.tenant_id, u.email, u.first_name, u.last_name, u.status, u.is_active, u.auth_method,
			ARRAY_AGG(
				json_build_object(
					'id', rr.id,
					'name', rr.name,
					'description', rr.description,
					'permissions', rr.permissions,
					'is_system', rr.is_system
				)
			) FILTER (WHERE rr.id IS NOT NULL) as roles
		FROM users u
		LEFT JOIN user_role_assignments ura ON u.id = ura.user_id
		LEFT JOIN rbac_roles rr ON ura.role_id = rr.id
		WHERE u.id = $1
		GROUP BY u.id, u.tenant_id, u.email, u.first_name, u.last_name, u.status, u.is_active, u.auth_method
	`

	var result UserWithRoles
	err := r.db.GetContext(ctx, &result, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, rbac.ErrRoleNotFound
		}
		r.logger.Error("Failed to find user with roles",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user with roles: %w", err)
	}

	// Convert to domain type
	domainResult := &rbac.UserWithRoles{
		ID:         result.ID,
		TenantID:   result.TenantID,
		Email:      result.Email,
		FirstName:  result.FirstName,
		LastName:   result.LastName,
		Status:     result.Status,
		IsActive:   result.IsActive,
		AuthMethod: result.AuthMethod,
		Roles:      make([]rbac.RoleWithMeta, len(result.Roles)),
	}

	for i, role := range result.Roles {
		domainResult.Roles[i] = rbac.RoleWithMeta{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			Permissions: role.Permissions,
			IsSystem:    role.IsSystem,
		}
	}

	return domainResult, nil
}

// FindUsersByRole retrieves all users assigned to a role
func (r *RBACRepository) FindUsersByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID

	query := `SELECT user_id FROM user_role_assignments WHERE role_id = $1 ORDER BY assigned_at`

	err := r.db.SelectContext(ctx, &userIDs, query, roleID)
	if err != nil {
		r.logger.Error("Failed to find users by role",
			zap.String("role_id", roleID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find users by role: %w", err)
	}

	return userIDs, nil
}

// IsUserAssignedRole checks if a user is assigned a specific role (any tenant)
func (r *RBACRepository) IsUserAssignedRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM user_role_assignments WHERE user_id = $1 AND role_id = $2)`

	err := r.db.GetContext(ctx, &exists, query, userID, roleID)
	if err != nil {
		r.logger.Error("Failed to check user role assignment",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check user role assignment: %w", err)
	}

	return exists, nil
}

// IsUserAssignedRoleForTenant checks if a user is assigned a specific role for a specific tenant
func (r *RBACRepository) IsUserAssignedRoleForTenant(ctx context.Context, userID, roleID, tenantID uuid.UUID) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM user_role_assignments WHERE user_id = $1 AND role_id = $2 AND tenant_id = $3)`

	err := r.db.GetContext(ctx, &exists, query, userID, roleID, tenantID)
	if err != nil {
		r.logger.Error("Failed to check user role assignment for tenant",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check user role assignment for tenant: %w", err)
	}

	return exists, nil
}

// UserHasPermission checks if a user has a specific permission through their roles (both RBAC and group-based)
func (r *RBACRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	var hasPermission bool

	// Check both direct RBAC role assignments AND group-based role assignments
	query := `
		SELECT EXISTS(
			-- Check direct RBAC role assignments
			SELECT 1
			FROM user_role_assignments ura
			INNER JOIN rbac_roles role ON ura.role_id = role.id
			INNER JOIN role_permissions rp ON role.id = rp.role_id
			INNER JOIN permissions p ON rp.permission_id = p.id
			WHERE ura.user_id = $1
			AND (
				(p.resource = $2 AND p.action = $3)
				OR (p.resource = '*' AND p.action = $3)
				OR (p.resource = $2 AND p.action = '*')
				OR (p.resource = '*' AND p.action = '*')
			)
			
			UNION ALL
			
			-- Check group-based role assignments
			SELECT 1
			FROM group_members gm
			INNER JOIN tenant_groups tg ON gm.group_id = tg.id
			INNER JOIN rbac_roles role ON REPLACE(LOWER(role.name), ' ', '_') = tg.role_type AND role.tenant_id IS NULL
			INNER JOIN role_permissions rp ON role.id = rp.role_id
			INNER JOIN permissions p ON rp.permission_id = p.id
			WHERE gm.user_id = $1
			AND gm.removed_at IS NULL
			AND tg.status = 'active'
			AND (
				(p.resource = $2 AND p.action = $3)
				OR (p.resource = '*' AND p.action = $3)
				OR (p.resource = $2 AND p.action = '*')
				OR (p.resource = '*' AND p.action = '*')
			)
		)
	`

	err := r.db.GetContext(ctx, &hasPermission, query, userID, resource, action)
	if err != nil {
		r.logger.Error("Failed to check user permission",
			zap.String("user_id", userID.String()),
			zap.String("resource", resource),
			zap.String("action", action),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check user permission: %w", err)
	}

	return hasPermission, nil
}

// modelToRole converts a database model to a domain role
func (r *RBACRepository) modelToRole(model *roleModel, permissions []rbac.Permission) (*rbac.Role, error) {
	createdAt, err := time.Parse(time.RFC3339, model.CreatedAt)
	if err != nil {
		createdAt = time.Now().UTC()
	}

	updatedAt, err := time.Parse(time.RFC3339, model.UpdatedAt)
	if err != nil {
		updatedAt = time.Now().UTC()
	}

	return rbac.NewRoleFromExisting(
		model.ID,
		model.CompanyID,
		model.Name,
		model.Description,
		permissions,
		model.IsSystemRole,
		createdAt,
		updatedAt,
		1,
	)
}

// rowToRole converts a single database row to a domain role
func (r *RBACRepository) rowToRole(id uuid.UUID, tenantID uuid.UUID, name, description string, permissionsJSON []byte, isSystem bool, createdAt, updatedAt string) (*rbac.Role, error) {
	// Parse permissions from JSON
	permissions := make([]rbac.Permission, 0)
	if len(permissionsJSON) > 0 && string(permissionsJSON) != "[]" {
		// First, try to unmarshal as array of Permission objects
		err := json.Unmarshal(permissionsJSON, &permissions)
		if err != nil {
			// If that fails, try to handle legacy string array format
			var stringPermissions []string
			err2 := json.Unmarshal(permissionsJSON, &stringPermissions)
			if err2 != nil {
				// Log both errors - the one we tried and the fallback
				r.logger.Error("Failed to unmarshal permissions",
					zap.String("role_id", id.String()),
					zap.String("raw_data", string(permissionsJSON)),
					zap.Error(err),
				)
			} else {
				// Convert legacy string format (e.g., "resource:action") to Permission objects
				for _, perm := range stringPermissions {
					parts := strings.Split(perm, ":")
					if len(parts) >= 2 {
						permissions = append(permissions, rbac.NewPermission(parts[0], parts[1]))
					}
				}
			}
		}
	}

	// Parse timestamps
	createdTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		createdTime = time.Now().UTC()
	}
	updatedTime, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		updatedTime = time.Now().UTC()
	}

	return rbac.NewRoleFromExisting(
		id,
		tenantID,
		name,
		description,
		permissions,
		isSystem,
		createdTime,
		updatedTime,
		1,
	)
}

// FindUserRolesBatch retrieves roles for multiple users in a single optimized query
// This eliminates N+1 queries by loading all user-role assignments at once
// Includes both direct role assignments and group-based roles
func (r *RBACRepository) FindUserRolesBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*rbac.Role, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID][]*rbac.Role), nil
	}

	// Build query with IN clause for batch loading - includes both direct and group-based roles
	query := `
		-- Direct role assignments
		SELECT r.id, COALESCE(r.tenant_id, $2::uuid),
		       r.name, r.description, r.is_system,
		       r.created_at, r.updated_at,
		       ura.user_id
		FROM rbac_roles r
		INNER JOIN user_role_assignments ura ON r.id = ura.role_id
		WHERE ura.user_id = ANY($1)

		UNION

		-- Group-based roles (map group role_type to system-level role)
		SELECT rr.id, tg.tenant_id,
		       rr.name, rr.description, rr.is_system,
		       rr.created_at, rr.updated_at,
		       gm.user_id
		FROM rbac_roles rr
		INNER JOIN tenant_groups tg ON REPLACE(LOWER(rr.name), ' ', '_') = tg.role_type AND rr.tenant_id IS NULL
		INNER JOIN group_members gm ON tg.id = gm.group_id
		WHERE gm.user_id = ANY($1)
		  AND gm.removed_at IS NULL
		  AND tg.status = 'active'

		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(userIDs), uuid.Nil)
	if err != nil {
		r.logger.Error("Failed to find user roles in batch",
			zap.Int("user_count", len(userIDs)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user roles in batch: %w", err)
	}
	defer rows.Close()

	// First pass: collect role data and IDs
	type roleData struct {
		id        uuid.UUID
		companyID uuid.UUID
		name      string
		desc      string
		isSystem  bool
		createdAt string
		updatedAt string
		userID    uuid.UUID
	}
	var roleDatas []roleData
	roleIDs := make([]uuid.UUID, 0)

	for rows.Next() {
		var rd roleData
		err := rows.Scan(
			&rd.id, &rd.companyID, &rd.name, &rd.desc,
			&rd.isSystem, &rd.createdAt, &rd.updatedAt, &rd.userID,
		)
		if err != nil {
			r.logger.Error("Failed to scan role in batch query", zap.Error(err))
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roleDatas = append(roleDatas, rd)
		roleIDs = append(roleIDs, rd.id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	// Batch load permissions for all roles
	permissionsMap, err := r.getRolePermissionsFromJunctionBatch(ctx, roleIDs)
	if err != nil {
		r.logger.Warn("Failed to batch load permissions for roles in batch",
			zap.Int("user_count", len(userIDs)),
			zap.Error(err),
		)
		// Continue with empty permissions rather than failing
		permissionsMap = make(map[uuid.UUID][]rbac.Permission)
	}

	// Map user IDs to their roles
	result := make(map[uuid.UUID][]*rbac.Role)
	for _, userID := range userIDs {
		result[userID] = []*rbac.Role{}
	}

	// Second pass: create roles with permissions
	for _, rd := range roleDatas {
		permissions := permissionsMap[rd.id]

		// Parse timestamps
		createdTime, err := time.Parse(time.RFC3339, rd.createdAt)
		if err != nil {
			createdTime = time.Now().UTC()
		}
		updatedTime, err := time.Parse(time.RFC3339, rd.updatedAt)
		if err != nil {
			updatedTime = time.Now().UTC()
		}

		role, err := rbac.NewRoleFromExisting(rd.id, rd.companyID, rd.name, rd.desc, permissions, rd.isSystem, createdTime, updatedTime, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to convert role model: %w", err)
		}

		result[rd.userID] = append(result[rd.userID], role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return result, nil
}

// UserHasTenantAccess checks if a user has access to a specific tenant
// Returns true if user has roles in the tenant (via RBAC or groups) or has system-wide roles
func (r *RBACRepository) UserHasTenantAccess(ctx context.Context, userID, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			-- Check direct RBAC role assignments
			SELECT 1 FROM user_role_assignments ura
			WHERE ura.user_id = $1 AND ura.tenant_id = $2
		) OR EXISTS(
			-- Check group-based access to the tenant
			SELECT 1 FROM group_members gm
			INNER JOIN tenant_groups tg ON gm.group_id = tg.id
			WHERE gm.user_id = $1 
			AND tg.tenant_id = $2
			AND gm.removed_at IS NULL
			AND tg.status = 'active'
		) OR EXISTS(
			-- Check system-wide roles
			SELECT 1 FROM user_role_assignments ura
			INNER JOIN rbac_roles r ON ura.role_id = r.id
			WHERE ura.user_id = $1 AND r.is_system = true
		) OR EXISTS(
			-- Allow central security reviewers to operate across tenants.
			SELECT 1 FROM group_members gm
			INNER JOIN tenant_groups tg ON gm.group_id = tg.id
			WHERE gm.user_id = $1
			AND tg.role_type = 'security_reviewer'
			AND gm.removed_at IS NULL
			AND tg.status = 'active'
		)
	`

	var hasAccess bool
	err := r.db.QueryRowContext(ctx, query, userID, tenantID).Scan(&hasAccess)
	if err != nil {
		r.logger.Error("Failed to check user tenant access",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check user tenant access: %w", err)
	}

	return hasAccess, nil
}

// IsUserSystemAdmin checks if a user is a system administrator
// System admins are determined by group membership with role_type = 'system_administrator'
func (r *RBACRepository) IsUserSystemAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM group_members gm
			INNER JOIN tenant_groups tg ON gm.group_id = tg.id
			WHERE gm.user_id = $1 
			AND tg.role_type = 'system_administrator'
			AND gm.removed_at IS NULL
			AND tg.status = 'active'
		)
	`

	var isSystemAdmin bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&isSystemAdmin)
	if err != nil {
		r.logger.Error("Failed to check user system admin status",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check user system admin status: %w", err)
	}

	return isSystemAdmin, nil
}

// FindUserRoleAssignmentsByUser retrieves all role assignments for a user grouped by tenant
// Returns a map of tenantID -> roleIDs
func (r *RBACRepository) FindUserRoleAssignmentsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	query := `
		SELECT tenant_id, role_id
		FROM (
			-- Direct role assignments (tenant-scoped and global)
			SELECT ura.tenant_id, ura.role_id
			FROM user_role_assignments ura
			WHERE ura.user_id = $1

			UNION

			-- Group-based role assignments (mapped from tenant group role_type)
			SELECT tg.tenant_id, rr.id AS role_id
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			INNER JOIN rbac_roles rr ON REPLACE(LOWER(rr.name), ' ', '_') = tg.role_type AND rr.tenant_id IS NULL
			WHERE gm.user_id = $1
			  AND gm.removed_at IS NULL
			  AND tg.status = 'active'
		) assignments
		ORDER BY tenant_id, role_id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to find user role assignments",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user role assignments: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]uuid.UUID)
	for rows.Next() {
		var tenantID *uuid.UUID
		var roleID uuid.UUID

		if err := rows.Scan(&tenantID, &roleID); err != nil {
			r.logger.Warn("Failed to scan role assignment",
				zap.String("user_id", userID.String()),
				zap.Error(err),
			)
			continue
		}

		// Convert NULL tenant_id to zero UUID (system-wide role)
		tid := uuid.Nil
		if tenantID != nil {
			tid = *tenantID
		}

		result[tid] = append(result[tid], roleID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate role assignments: %w", err)
	}

	return result, nil
}

// getRolePermissionsFromJunction fetches permissions for a role from the role_permissions junction table
func (r *RBACRepository) getRolePermissionsFromJunction(ctx context.Context, roleID uuid.UUID) ([]rbac.Permission, error) {
	query := `
		SELECT p.id, p.resource, p.action, p.description, p.category, p.is_system_permission, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action
	`

	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		r.logger.Error("Failed to fetch role permissions from junction table",
			zap.String("role_id", roleID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch role permissions: %w", err)
	}
	defer rows.Close()

	permissions := make([]rbac.Permission, 0)
	for rows.Next() {
		var permID uuid.UUID
		var resource, action string
		var description sql.NullString
		var category sql.NullString
		var isSystemPerm bool
		var createdAt, updatedAt string

		err := rows.Scan(
			&permID,
			&resource,
			&action,
			&description,
			&category,
			&isSystemPerm,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Warn("Failed to scan permission",
				zap.String("role_id", roleID.String()),
				zap.Error(err),
			)
			continue
		}

		perm := rbac.NewPermissionWithID(permID.String(), resource, action)
		permissions = append(permissions, perm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate permissions: %w", err)
	}

	r.logger.Debug("Fetched role permissions from junction table",
		zap.String("role_id", roleID.String()),
		zap.Int("permission_count", len(permissions)),
	)

	return permissions, nil
}

// getRolePermissionsFromJunctionBatch fetches permissions for multiple roles in a single query
// This eliminates N+1 queries when loading permissions for multiple roles
func (r *RBACRepository) getRolePermissionsFromJunctionBatch(ctx context.Context, roleIDs []uuid.UUID) (map[uuid.UUID][]rbac.Permission, error) {
	if len(roleIDs) == 0 {
		return make(map[uuid.UUID][]rbac.Permission), nil
	}

	// Build IN clause for batch query
	placeholders := make([]string, len(roleIDs))
	args := make([]interface{}, len(roleIDs))
	for i, id := range roleIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT rp.role_id, p.id, p.resource, p.action, p.description, p.category, p.is_system_permission, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id IN (%s)
		ORDER BY rp.role_id, p.resource, p.action
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to fetch batch role permissions from junction table",
			zap.Int("role_count", len(roleIDs)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch batch role permissions: %w", err)
	}
	defer rows.Close()

	// Map of role_id -> permissions
	result := make(map[uuid.UUID][]rbac.Permission)

	for rows.Next() {
		var roleID uuid.UUID
		var permID uuid.UUID
		var resource, action string
		var description, category sql.NullString
		var isSystemPerm bool
		var createdAt, updatedAt string

		err := rows.Scan(
			&roleID,
			&permID,
			&resource,
			&action,
			&description,
			&category,
			&isSystemPerm,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Warn("Failed to scan permission in batch",
				zap.Error(err),
			)
			continue
		}

		perm := rbac.NewPermissionWithID(permID.String(), resource, action)
		result[roleID] = append(result[roleID], perm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate batch permissions: %w", err)
	}

	r.logger.Debug("Fetched batch role permissions from junction table",
		zap.Int("role_count", len(roleIDs)),
		zap.Int("unique_roles_with_permissions", len(result)),
	)

	return result, nil
}
