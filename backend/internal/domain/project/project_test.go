package project

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject(t *testing.T) {
	tenantID := uuid.New()
	name := "test-project"
	description := "Test project description"
	gitRepo := "https://github.com/example/repo.git"

	project, err := NewProject(tenantID, name, "", description, gitRepo, "main", "private", "generic", nil, nil, false)

	require.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, tenantID, project.TenantID())
	assert.Equal(t, name, project.Name())
	assert.Equal(t, description, project.Description())
	assert.Equal(t, gitRepo, project.GitRepo())
	assert.Equal(t, "main", project.GitBranch())
	assert.Equal(t, "generic", project.GitProvider())
	assert.Equal(t, ProjectStatusActive, project.Status())
	assert.Equal(t, 0, project.BuildCount())
	assert.False(t, project.IsDeleted())
	assert.True(t, project.IsActive())
}

func TestNewProjectValidation(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name        string
		projectName string
		description string
		gitRepo     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid project",
			projectName: "valid-project",
			description: "Valid description",
			gitRepo:     "https://github.com/example/repo.git",
			expectError: false,
		},
		{
			name:        "empty name",
			projectName: "",
			expectError: true,
			errorMsg:    "project name is required",
		},
		{
			name:        "name too short",
			projectName: "ab",
			expectError: true,
			errorMsg:    "project name must be between 3 and 100 characters",
		},
		{
			name:        "name too long",
			projectName: string(make([]byte, 101)),
			expectError: true,
			errorMsg:    "project name must be between 3 and 100 characters",
		},
		{
			name:        "description too long",
			projectName: "valid-name",
			description: string(make([]byte, 1001)),
			expectError: true,
			errorMsg:    "project description must not exceed 1000 characters",
		},
		{
			name:        "git repo too long",
			projectName: "valid-name",
			gitRepo:     string(make([]byte, 501)),
			expectError: true,
			errorMsg:    "git repository URL must not exceed 500 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProject(tenantID, tt.projectName, "", tt.description, tt.gitRepo, "main", "private", "generic", nil, nil, false)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewProjectNilTenantID(t *testing.T) {
	_, err := NewProject(uuid.Nil, "test", "", "desc", "", "main", "private", "generic", nil, nil, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestProjectUpdate(t *testing.T) {
	project, err := NewProject(uuid.New(), "original", "", "original desc", "original repo", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	originalUpdatedAt := project.UpdatedAt()

	// Wait a bit to ensure updated_at changes
	time.Sleep(time.Millisecond)

	err = project.Update("updated", "updated", "updated desc", "updated repo", "develop", "github", nil)
	require.NoError(t, err)

	assert.Equal(t, "updated", project.Name())
	assert.Equal(t, "updated desc", project.Description())
	assert.Equal(t, "updated repo", project.GitRepo())
	assert.True(t, project.UpdatedAt().After(originalUpdatedAt))
}

func TestProjectUpdateValidation(t *testing.T) {
	project, err := NewProject(uuid.New(), "original", "", "desc", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	// Try to update with invalid name
	err = project.Update("", "", "desc", "", "main", "generic", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project name is required")

	// Name should not have changed
	assert.Equal(t, "original", project.Name())
}

func TestProjectStatusChanges(t *testing.T) {
	project, err := NewProject(uuid.New(), "test", "", "desc", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	// Initially active
	assert.Equal(t, ProjectStatusActive, project.Status())
	assert.True(t, project.IsActive())

	// Archive
	err = project.Archive()
	require.NoError(t, err)
	assert.Equal(t, ProjectStatusArchived, project.Status())
	assert.False(t, project.IsActive())

	// Try to archive again
	err = project.Archive()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already archived")

	// Activate
	err = project.Activate()
	require.NoError(t, err)
	assert.Equal(t, ProjectStatusActive, project.Status())
	assert.True(t, project.IsActive())

	// Suspend
	err = project.Suspend()
	require.NoError(t, err)
	assert.Equal(t, ProjectStatusSuspended, project.Status())
	assert.False(t, project.IsActive())

	// Try to suspend again
	err = project.Suspend()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already suspended")
}

func TestProjectSoftDelete(t *testing.T) {
	project, err := NewProject(uuid.New(), "test", "", "desc", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	assert.False(t, project.IsDeleted())
	assert.Nil(t, project.DeletedAt())

	err = project.Delete()
	require.NoError(t, err)

	assert.True(t, project.IsDeleted())
	assert.NotNil(t, project.DeletedAt())
	assert.False(t, project.IsActive())
}

func TestProjectIncrementBuildCount(t *testing.T) {
	project, err := NewProject(uuid.New(), "test", "", "desc", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	assert.Equal(t, 0, project.BuildCount())

	project.IncrementBuildCount()
	assert.Equal(t, 1, project.BuildCount())

	project.IncrementBuildCount()
	assert.Equal(t, 2, project.BuildCount())
}

func TestNewProjectFromExisting(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	name := "test"
	description := "desc"
	gitRepo := "repo"
	status := ProjectStatusActive
	buildCount := 5
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()
	deletedAt := (*time.Time)(nil)
	version := 2

	project := NewProjectFromExisting(id, tenantID, name, "test", description, gitRepo, "main", "generic", status, "private", nil, nil, false, buildCount, createdAt, updatedAt, deletedAt, version)

	assert.Equal(t, id, project.ID())
	assert.Equal(t, tenantID, project.TenantID())
	assert.Equal(t, name, project.Name())
	assert.Equal(t, description, project.Description())
	assert.Equal(t, gitRepo, project.GitRepo())
	assert.Equal(t, status, project.Status())
	assert.Equal(t, buildCount, project.BuildCount())
	assert.Equal(t, createdAt, project.CreatedAt())
	assert.Equal(t, updatedAt, project.UpdatedAt())
	assert.Equal(t, deletedAt, project.DeletedAt())
	assert.Equal(t, version, project.Version())
}
