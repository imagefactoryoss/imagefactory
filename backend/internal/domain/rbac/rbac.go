package rbac

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrRoleNotFound     = errors.New("role not found")
	ErrRoleExists       = errors.New("role already exists")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInvalidRoleID    = errors.New("invalid role ID")
	ErrInvalidRoleName  = errors.New("invalid role name")
)

// Permission represents a system permission
type Permission struct {
	ID       string
	Resource string
	Action   string
}

// NewPermission creates a new permission
func NewPermission(resource, action string) Permission {
	return Permission{
		ID:       resource + ":" + action,
		Resource: resource,
		Action:   action,
	}
}

// NewPermissionWithID creates a new permission with explicit ID from database
func NewPermissionWithID(id, resource, action string) Permission {
	return Permission{
		ID:       id,
		Resource: resource,
		Action:   action,
	}
}

// String returns the string representation of a permission
func (p Permission) String() string {
	return p.Resource + ":" + p.Action
}

// Role represents a role with associated permissions
type Role struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	name        string
	description string
	permissions []Permission
	isSystem    bool
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewRole creates a new role
func NewRole(tenantID uuid.UUID, name, description string, permissions []Permission) (*Role, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if name == "" {
		return nil, ErrInvalidRoleName
	}

	now := time.Now().UTC()

	return &Role{
		id:          uuid.New(),
		tenantID:    tenantID,
		name:        name,
		description: description,
		permissions: permissions,
		isSystem:    false,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, nil
}

// NewSystemRole creates a new system role (not tenant-specific)
func NewSystemRole(name, description string, permissions []Permission) (*Role, error) {
	if name == "" {
		return nil, ErrInvalidRoleName
	}

	now := time.Now().UTC()

	return &Role{
		id:          uuid.New(),
		tenantID:    uuid.Nil, // System roles don't belong to a tenant
		name:        name,
		description: description,
		permissions: permissions,
		isSystem:    true,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, nil
}

// NewRoleFromExisting creates a role from existing data
func NewRoleFromExisting(
	id, tenantID uuid.UUID,
	name, description string,
	permissions []Permission,
	isSystem bool,
	createdAt, updatedAt time.Time,
	version int,
) (*Role, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidRoleID
	}
	if name == "" {
		return nil, ErrInvalidRoleName
	}

	return &Role{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		permissions: permissions,
		isSystem:    isSystem,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ID returns the role ID
func (r *Role) ID() uuid.UUID {
	return r.id
}

// TenantID returns the tenant ID
func (r *Role) TenantID() uuid.UUID {
	return r.tenantID
}

// Name returns the role name
func (r *Role) Name() string {
	return r.name
}

// Description returns the role description
func (r *Role) Description() string {
	return r.description
}

// Permissions returns the role permissions
func (r *Role) Permissions() []Permission {
	// Return a copy to prevent external modification
	perms := make([]Permission, len(r.permissions))
	copy(perms, r.permissions)
	return perms
}

// IsSystem returns true if this is a system role
func (r *Role) IsSystem() bool {
	return r.isSystem
}

// CreatedAt returns the creation timestamp
func (r *Role) CreatedAt() time.Time {
	return r.createdAt
}

// UpdatedAt returns the last update timestamp
func (r *Role) UpdatedAt() time.Time {
	return r.updatedAt
}

// Version returns the aggregate version for optimistic concurrency
func (r *Role) Version() int {
	return r.version
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(resource, action string) bool {
	for _, perm := range r.permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
		// Check for wildcard permissions
		if perm.Resource == "*" && perm.Action == "*" {
			return true
		}
		if perm.Resource == "*" && perm.Action == action {
			return true
		}
		if perm.Resource == resource && perm.Action == "*" {
			return true
		}
	}
	return false
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(resource, action string) {
	perm := NewPermission(resource, action)

	// Check if permission already exists
	for _, existing := range r.permissions {
		if existing.Resource == resource && existing.Action == action {
			return // Already exists
		}
	}

	r.permissions = append(r.permissions, perm)
	r.updatedAt = time.Now().UTC()
	r.version++
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(resource, action string) {
	for i, perm := range r.permissions {
		if perm.Resource == resource && perm.Action == action {
			r.permissions = append(r.permissions[:i], r.permissions[i+1:]...)
			r.updatedAt = time.Now().UTC()
			r.version++
			return
		}
	}
}

// UpdateDetails updates the role name and description
func (r *Role) UpdateDetails(name, description string) error {
	if name == "" {
		return ErrInvalidRoleName
	}

	r.name = name
	r.description = description
	r.updatedAt = time.Now().UTC()
	r.version++
	return nil
}

// UserRoleAssignment represents the assignment of a role to a user, optionally scoped to a tenant
// When TenantID is nil, the role is global/system-wide
// When TenantID is set, the role is scoped to that specific tenant
type UserRoleAssignment struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	RoleID     uuid.UUID
	TenantID   *uuid.UUID // Pointer to allow NULL values for system-wide roles
	AssignedAt time.Time
	AssignedBy uuid.UUID
	ExpiresAt  *time.Time // Pointer to allow NULL values for non-expiring assignments
}

// NewUserRoleAssignment creates a new user-role assignment with optional tenant scoping
func NewUserRoleAssignment(userID, roleID, assignedBy uuid.UUID, tenantID *uuid.UUID) *UserRoleAssignment {
	return &UserRoleAssignment{
		ID:         uuid.New(),
		UserID:     userID,
		RoleID:     roleID,
		TenantID:   tenantID,
		AssignedAt: time.Now().UTC(),
		AssignedBy: assignedBy,
	}
}
