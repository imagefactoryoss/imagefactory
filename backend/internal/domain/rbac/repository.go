package rbac

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for RBAC persistence
type Repository interface {
	// Role operations
	SaveRole(ctx context.Context, role *Role) error
	FindRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)
	FindRoleByName(ctx context.Context, tenantID uuid.UUID, name string) (*Role, error)
	FindRolesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*Role, error)
	FindAssignableRoleTypesByTenant(ctx context.Context, tenantID uuid.UUID) ([]string, error)
	FindSystemRoles(ctx context.Context) ([]*Role, error)
	FindAllSystemLevelRoles(ctx context.Context) ([]*Role, error) // Returns both system roles and tenant-level roles where tenant_id is NULL
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error

	// User-Role assignment operations
	AssignRoleToUser(ctx context.Context, assignment *UserRoleAssignment) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRoleFromUserForTenant(ctx context.Context, userID, roleID, tenantID uuid.UUID) error
	FindUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)
	FindUserRolesForTenant(ctx context.Context, userID, tenantID uuid.UUID) ([]*Role, error)
	FindUsersByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	IsUserAssignedRole(ctx context.Context, userID, roleID uuid.UUID) (bool, error)
	IsUserAssignedRoleForTenant(ctx context.Context, userID, roleID, tenantID uuid.UUID) (bool, error)

	// Permission checking
	UserHasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)

	// Optimized queries to avoid N+1 problems
	FindUserRolesWithUserInfo(ctx context.Context, userID uuid.UUID) (*UserWithRoles, error)
	FindUserRolesBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*Role, error)
	FindUserRoleAssignmentsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) // Returns map of tenantID -> roleIDs
}

// UserWithRoles represents a user with their assigned roles for optimized queries
type UserWithRoles struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Email     string         `json:"email"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Status    string         `json:"status"`
	IsActive  bool           `json:"is_active"`
	AuthMethod string        `json:"auth_method"`
	Roles     []RoleWithMeta `json:"roles"`
}

// RoleWithMeta represents role metadata for user queries
type RoleWithMeta struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions string    `json:"permissions"`
	IsSystem    bool      `json:"is_system"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty"` // Tenant this role is assigned to (per-tenant assignments)
}
