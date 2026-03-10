package rbac

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service defines the business logic for RBAC management
type Service struct {
	repository Repository
	logger     *zap.Logger
}

// NewService creates a new RBAC service
func NewService(repository Repository, logger *zap.Logger) *Service {
	return &Service{
		repository: repository,
		logger:     logger,
	}
}

// CreateRole creates a new role
func (s *Service) CreateRole(ctx context.Context, tenantID uuid.UUID, name, description string, permissions []Permission) (*Role, error) {
	// Check if role already exists
	_, err := s.repository.FindRoleByName(ctx, tenantID, name)
	if err == nil {
		return nil, ErrRoleExists
	}
	if err != ErrRoleNotFound {
		return nil, err
	}

	// Create new role
	role, err := NewRole(tenantID, name, description, permissions)
	if err != nil {
		return nil, err
	}

	// Save role
	if err := s.repository.SaveRole(ctx, role); err != nil {
		return nil, err
	}

	s.logger.Info("Role created successfully",
		zap.String("role_id", role.ID().String()),
		zap.String("role_name", role.Name()),
		zap.String("tenant_id", tenantID.String()),
	)

	return role, nil
}

// CreateSystemRole creates a new system role
func (s *Service) CreateSystemRole(ctx context.Context, name, description string, permissions []Permission) (*Role, error) {
	// Check if system role already exists
	systemRoles, err := s.repository.FindSystemRoles(ctx)
	if err != nil {
		return nil, err
	}

	for _, existing := range systemRoles {
		if existing.Name() == name {
			return nil, ErrRoleExists
		}
	}

	// Create new system role
	role, err := NewSystemRole(name, description, permissions)
	if err != nil {
		return nil, err
	}

	// Save role
	if err := s.repository.SaveRole(ctx, role); err != nil {
		return nil, err
	}

	s.logger.Info("System role created successfully",
		zap.String("role_id", role.ID().String()),
		zap.String("role_name", role.Name()),
	)

	return role, nil
}

// GetRoleByID retrieves a role by ID
func (s *Service) GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	return s.repository.FindRoleByID(ctx, id)
}

// GetRoleByName retrieves a role by name and tenant
func (s *Service) GetRoleByName(ctx context.Context, tenantID uuid.UUID, name string) (*Role, error) {
	return s.repository.FindRoleByName(ctx, tenantID, name)
}

// GetRolesByTenantID retrieves all roles for a tenant
func (s *Service) GetRolesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*Role, error) {
	return s.repository.FindRolesByTenantID(ctx, tenantID)
}

// GetAssignableRoleTypesByTenant retrieves active role_type values configured for a tenant.
func (s *Service) GetAssignableRoleTypesByTenant(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return s.repository.FindAssignableRoleTypesByTenant(ctx, tenantID)
}

// GetSystemRoles retrieves all system roles
func (s *Service) GetSystemRoles(ctx context.Context) ([]*Role, error) {
	return s.repository.FindSystemRoles(ctx)
}

// GetAllSystemLevelRoles retrieves all system-level roles (both system roles and tenant-level roles at the system level where tenant_id is NULL)
func (s *Service) GetAllSystemLevelRoles(ctx context.Context) ([]*Role, error) {
	return s.repository.FindAllSystemLevelRoles(ctx)
}

// UpdateRole updates a role's details and permissions
func (s *Service) UpdateRole(ctx context.Context, role *Role) error {
	return s.repository.UpdateRole(ctx, role)
}

// DeleteRole deletes a role
func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.repository.DeleteRole(ctx, id)
}

// AssignRoleToUser assigns a role to a user (globally, without tenant scoping)
func (s *Service) AssignRoleToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	// Verify role exists
	_, err := s.repository.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}

	// Check if assignment already exists
	exists, err := s.repository.IsUserAssignedRole(ctx, userID, roleID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("user is already assigned this role")
	}

	// Create assignment with no tenant scoping (system-wide)
	assignment := NewUserRoleAssignment(userID, roleID, assignedBy, nil)

	// Save assignment
	if err := s.repository.AssignRoleToUser(ctx, assignment); err != nil {
		return err
	}

	s.logger.Info("Role assigned to user successfully",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()),
		zap.String("assigned_by", assignedBy.String()),
	)

	return nil
}

// AssignRoleToUserForTenant assigns a role to a user for a specific tenant
// This allows per-tenant role assignment (e.g., Developer for Tenant 1, Operator for Tenant 2)
func (s *Service) AssignRoleToUserForTenant(ctx context.Context, userID, roleID, tenantID, assignedBy uuid.UUID) error {
	// Verify role exists
	_, err := s.repository.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}

	// Check if assignment already exists for this tenant
	exists, err := s.repository.IsUserAssignedRoleForTenant(ctx, userID, roleID, tenantID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("user is already assigned this role for this tenant")
	}

	// Create tenant-scoped assignment
	assignment := NewUserRoleAssignment(userID, roleID, assignedBy, &tenantID)

	// Save assignment
	if err := s.repository.AssignRoleToUser(ctx, assignment); err != nil {
		return err
	}

	s.logger.Info("Role assigned to user for tenant successfully",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("assigned_by", assignedBy.String()),
	)

	return nil
}

// RemoveRoleFromUser removes a role assignment from a user (all tenants)
func (s *Service) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.repository.RemoveRoleFromUser(ctx, userID, roleID)
}

// RemoveRoleFromUserForTenant removes a role assignment from a user for a specific tenant
func (s *Service) RemoveRoleFromUserForTenant(ctx context.Context, userID, roleID, tenantID uuid.UUID) error {
	return s.repository.RemoveRoleFromUserForTenant(ctx, userID, roleID, tenantID)
}

// GetUserRoles retrieves all roles assigned to a user across all tenants
func (s *Service) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error) {
	return s.repository.FindUserRoles(ctx, userID)
}

// GetUserRolesForTenant retrieves all roles assigned to a user for a specific tenant
func (s *Service) GetUserRolesForTenant(ctx context.Context, userID, tenantID uuid.UUID) ([]*Role, error) {
	return s.repository.FindUserRolesForTenant(ctx, userID, tenantID)
}

// GetUsersByRole retrieves all users assigned to a role
func (s *Service) GetUsersByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return s.repository.FindUsersByRole(ctx, roleID)
}

// CheckUserPermission checks if a user has a specific permission
func (s *Service) CheckUserPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	return s.repository.UserHasPermission(ctx, userID, resource, action)
}

// RequirePermission checks if a user has a specific permission and returns an error if not
func (s *Service) RequirePermission(ctx context.Context, userID uuid.UUID, resource, action string) error {
	hasPermission, err := s.CheckUserPermission(ctx, userID, resource, action)
	if err != nil {
		return err
	}
	if !hasPermission {
		return ErrPermissionDenied
	}
	return nil
}

// InitializeSystemRoles creates default system roles if they don't exist
func (s *Service) InitializeSystemRoles(ctx context.Context) error {
	systemRoles := []struct {
		name        string
		description string
		permissions []Permission
	}{
		{
			name:        "admin",
			description: "Full system administrator with all permissions",
			permissions: []Permission{
				{Resource: "*", Action: "*"},
			},
		},
		{
			name:        "user_manager",
			description: "Can manage users and their roles",
			permissions: []Permission{
				{Resource: "users", Action: "read"},
				{Resource: "users", Action: "write"},
				{Resource: "users", Action: "delete"},
				{Resource: "roles", Action: "read"},
				{Resource: "roles", Action: "assign"},
			},
		},
		{
			name:        "viewer",
			description: "Read-only access to most resources",
			permissions: []Permission{
				{Resource: "*", Action: "read"},
			},
		},
	}

	for _, roleDef := range systemRoles {
		_, err := s.CreateSystemRole(ctx, roleDef.name, roleDef.description, roleDef.permissions)
		if err != nil && err != ErrRoleExists {
			return err
		}
	}

	s.logger.Info("System roles initialized successfully")
	return nil
}

// GetUserWithRoles retrieves a user with their roles in a single optimized query to avoid N+1
func (s *Service) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*UserWithRoles, error) {
	return s.repository.FindUserRolesWithUserInfo(ctx, userID)
}
// GetUserRolesBatch retrieves roles for multiple users in a single batch query
// This eliminates N+1 queries by loading all user-role assignments at once
func (s *Service) GetUserRolesBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*Role, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID][]*Role), nil
	}

	return s.repository.FindUserRolesBatch(ctx, userIDs)
}

// GetUserRoleAssignmentsByTenant retrieves all role assignments for a user grouped by tenant
func (s *Service) GetUserRoleAssignmentsByTenant(ctx context.Context, userID uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	return s.repository.FindUserRoleAssignmentsByUser(ctx, userID)
}

// IsUserSystemAdmin checks if a user is a system administrator
func (s *Service) IsUserSystemAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	// Try to access the repository's IsUserSystemAdmin method if it exists
	// This delegates to the postgres repository implementation
	if repo, ok := s.repository.(interface{ IsUserSystemAdmin(context.Context, uuid.UUID) (bool, error) }); ok {
		return repo.IsUserSystemAdmin(ctx, userID)
	}
	// Fallback: check if user has system administrator role
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.Name() == "system_administrator" {
			return true, nil
		}
	}
	return false, nil
}
