package rbac

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPermission(t *testing.T) {
	perm := NewPermission("users", "read")
	assert.Equal(t, "users", perm.Resource)
	assert.Equal(t, "read", perm.Action)
	assert.Equal(t, "users:read", perm.String())
}

func TestNewRole(t *testing.T) {
	tests := []struct {
		testName    string
		tenantID    uuid.UUID
		roleName    string
		description string
		permissions []Permission
		expectError bool
	}{
		{
			testName:    "success",
			tenantID:    uuid.New(),
			roleName:    "admin",
			description: "Administrator role",
			permissions: []Permission{
				NewPermission("users", "read"),
				NewPermission("users", "write"),
			},
		},
		{
			testName:    "empty tenant ID",
			tenantID:    uuid.Nil,
			roleName:    "admin",
			description: "Administrator role",
			permissions: []Permission{},
			expectError: true,
		},
		{
			testName:    "empty name",
			tenantID:    uuid.New(),
			roleName:    "",
			description: "Administrator role",
			permissions: []Permission{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			role, err := NewRole(tt.tenantID, tt.roleName, tt.description, tt.permissions)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, role)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, tt.tenantID, role.TenantID())
				assert.Equal(t, tt.roleName, role.Name())
				assert.Equal(t, tt.description, role.Description())
				assert.Equal(t, tt.permissions, role.Permissions())
				assert.False(t, role.IsSystem())
				assert.Equal(t, 1, role.Version())
			}
		})
	}
}

func TestNewSystemRole(t *testing.T) {
	tests := []struct {
		testName    string
		roleName    string
		description string
		permissions []Permission
		expectError bool
	}{
		{
			testName:    "success",
			roleName:    "system_admin",
			description: "System administrator",
			permissions: []Permission{
				NewPermission("*", "*"),
			},
		},
		{
			testName:    "empty name",
			roleName:    "",
			description: "System administrator",
			permissions: []Permission{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			role, err := NewSystemRole(tt.roleName, tt.description, tt.permissions)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, role)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, uuid.Nil, role.TenantID())
				assert.Equal(t, tt.roleName, role.Name())
				assert.Equal(t, tt.description, role.Description())
				assert.Equal(t, tt.permissions, role.Permissions())
				assert.True(t, role.IsSystem())
				assert.Equal(t, 1, role.Version())
			}
		})
	}
}

func TestNewRoleFromExisting(t *testing.T) {
	tests := []struct {
		testName    string
		id          uuid.UUID
		tenantID    uuid.UUID
		roleName    string
		isSystem    bool
		expectError bool
	}{
		{
			testName: "success tenant role",
			id:       uuid.New(),
			tenantID: uuid.New(),
			roleName: "admin",
			isSystem: false,
		},
		{
			testName: "success system role",
			id:       uuid.New(),
			tenantID: uuid.Nil,
			roleName: "system_admin",
			isSystem: true,
		},
		{
			testName:    "empty ID",
			id:          uuid.Nil,
			tenantID:    uuid.New(),
			roleName:    "admin",
			isSystem:    false,
			expectError: true,
		},
		{
			testName:    "empty name",
			id:          uuid.New(),
			tenantID:    uuid.New(),
			roleName:    "",
			isSystem:    false,
			expectError: true,
		},
		{
			testName:    "system role with tenant ID",
			id:          uuid.New(),
			tenantID:    uuid.New(),
			roleName:    "system_admin",
			isSystem:    true,
			expectError: false, // This should work as per current implementation
		},
		{
			testName:    "non-system role with nil tenant ID",
			id:          uuid.New(),
			tenantID:    uuid.Nil,
			roleName:    "template_role",
			isSystem:    false,
			expectError: false, // Allow template roles with nil tenant ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			permissions := []Permission{NewPermission("users", "read")}
			now := time.Now()

			role, err := NewRoleFromExisting(
				tt.id,
				tt.tenantID,
				tt.roleName,
				"Description",
				permissions,
				tt.isSystem,
				now,
				now,
				1,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, role)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, tt.id, role.ID())
				assert.Equal(t, tt.tenantID, role.TenantID())
				assert.Equal(t, tt.roleName, role.Name())
				assert.Equal(t, tt.isSystem, role.IsSystem())
			}
		})
	}
}

func TestRole_HasPermission(t *testing.T) {
	permissions := []Permission{
		NewPermission("users", "read"),
		NewPermission("users", "write"),
		NewPermission("*", "read"),
	}

	role, err := NewRole(uuid.New(), "admin", "Administrator", permissions)
	require.NoError(t, err)

	// Test exact permissions
	assert.True(t, role.HasPermission("users", "read"))
	assert.True(t, role.HasPermission("users", "write"))

	// Test wildcard permissions
	assert.True(t, role.HasPermission("posts", "read")) // * resource matches

	// Test non-matching permissions
	assert.False(t, role.HasPermission("posts", "write"))
	assert.False(t, role.HasPermission("posts", "delete"))
	assert.False(t, role.HasPermission("users", "delete"))
}

func TestRole_AddPermission(t *testing.T) {
	role, err := NewRole(uuid.New(), "admin", "Administrator", []Permission{})
	require.NoError(t, err)

	initialVersion := role.Version()

	// Add new permission
	role.AddPermission("users", "read")
	assert.True(t, role.HasPermission("users", "read"))
	assert.Equal(t, initialVersion+1, role.Version())

	// Try to add duplicate permission (should not add again)
	role.AddPermission("users", "read")
	permissions := role.Permissions()
	assert.Len(t, permissions, 1) // Should still be 1
}

func TestRole_RemovePermission(t *testing.T) {
	permissions := []Permission{
		NewPermission("users", "read"),
		NewPermission("users", "write"),
	}

	role, err := NewRole(uuid.New(), "admin", "Administrator", permissions)
	require.NoError(t, err)

	initialVersion := role.Version()

	// Remove existing permission
	role.RemovePermission("users", "read")
	assert.False(t, role.HasPermission("users", "read"))
	assert.True(t, role.HasPermission("users", "write"))
	assert.Equal(t, initialVersion+1, role.Version())

	// Try to remove non-existing permission (should not error)
	role.RemovePermission("posts", "read")
	assert.Len(t, role.Permissions(), 1) // Should still have 1 permission
}

func TestRole_UpdateDetails(t *testing.T) {
	role, err := NewRole(uuid.New(), "admin", "Administrator", []Permission{})
	require.NoError(t, err)

	initialVersion := role.Version()

	// Update details
	err = role.UpdateDetails("super_admin", "Super Administrator")
	assert.NoError(t, err)
	assert.Equal(t, "super_admin", role.Name())
	assert.Equal(t, "Super Administrator", role.Description())
	assert.Equal(t, initialVersion+1, role.Version())

	// Try to update with empty name
	err = role.UpdateDetails("", "Empty Name")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidRoleName, err)
}

func TestNewUserRoleAssignment(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	assignedBy := uuid.New()

	assignment := NewUserRoleAssignment(userID, roleID, assignedBy, nil)

	assert.Equal(t, userID, assignment.UserID)
	assert.Equal(t, roleID, assignment.RoleID)
	assert.Equal(t, assignedBy, assignment.AssignedBy)
	assert.NotZero(t, assignment.AssignedAt)
}
