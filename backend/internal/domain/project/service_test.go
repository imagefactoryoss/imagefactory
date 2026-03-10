package project

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

// MockMemberRepository is a mock implementation of the MemberRepository interface
type MockMemberRepository struct {
	mock.Mock
}

func (m *MockMemberRepository) CreateMember(ctx context.Context, member *Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*Member, error) {
	args := m.Called(ctx, projectID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Member), args.Error(1)
}

func (m *MockMemberRepository) ListMembers(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Member, int, error) {
	args := m.Called(ctx, projectID, limit, offset)
	return args.Get(0).([]*Member), args.Int(1), args.Error(2)
}

func (m *MockMemberRepository) ListUserProjects(ctx context.Context, userID, tenantID uuid.UUID, limit, offset int) ([]*Project, int, error) {
	args := m.Called(ctx, userID, tenantID, limit, offset)
	return args.Get(0).([]*Project), args.Int(1), args.Error(2)
}

func (m *MockMemberRepository) UpdateMember(ctx context.Context, member *Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) DeleteMember(ctx context.Context, projectID, userID uuid.UUID) error {
	args := m.Called(ctx, projectID, userID)
	return args.Error(0)
}

func (m *MockMemberRepository) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMemberRepository) CountMembers(ctx context.Context, projectID uuid.UUID) (int, error) {
	args := m.Called(ctx, projectID)
	return args.Int(0), args.Error(1)
}

func (m *MockMemberRepository) DeleteProjectMembers(ctx context.Context, projectID uuid.UUID) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockRepository) Save(ctx context.Context, project *Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Project), args.Error(1)
}

func (m *MockRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID, limit, offset int) ([]*Project, error) {
	args := m.Called(ctx, tenantID, viewerID, limit, offset)
	return args.Get(0).([]*Project), args.Error(1)
}

func (m *MockRepository) FindByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (*Project, error) {
	args := m.Called(ctx, name, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Project), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, project *Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID, viewerID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) ExistsByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (bool, error) {
	args := m.Called(ctx, name, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) ExistsBySlugAndTenantID(ctx context.Context, slug string, tenantID uuid.UUID) (bool, error) {
	args := m.Called(ctx, slug, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) PurgeDeletedBefore(ctx context.Context, cutoff time.Time) (int, error) {
	args := m.Called(ctx, cutoff)
	return args.Int(0), args.Error(1)
}

func TestServiceCreateProject(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	name := "test-project"
	slug := "test-project"
	description := "Test description"
	gitRepo := "https://github.com/example/repo.git"

	// Mock the repository calls
	mockRepo.On("ExistsByNameAndTenantID", ctx, name, tenantID).Return(false, nil)
	mockRepo.On("ExistsBySlugAndTenantID", ctx, slug, tenantID).Return(false, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*project.Project")).Return(nil).Run(func(args mock.Arguments) {
		project := args.Get(1).(*Project)
		// Simulate setting ID in repository
		project.id = uuid.New()
		project.createdAt = time.Now()
		project.updatedAt = time.Now()
	})

	project, err := service.CreateProject(ctx, tenantID, name, slug, description, gitRepo, "main", "private", "generic", nil, nil, false)

	require.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, tenantID, project.TenantID())
	assert.Equal(t, name, project.Name())
	assert.Equal(t, description, project.Description())
	assert.Equal(t, gitRepo, project.GitRepo())

	mockRepo.AssertExpectations(t)
}

func TestServiceCreateProjectValidationError(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	// Mock exists check (this happens before validation)
	mockRepo.On("ExistsByNameAndTenantID", ctx, "", tenantID).Return(false, nil)
	mockRepo.On("ExistsBySlugAndTenantID", ctx, "", tenantID).Return(false, nil)

	// Invalid project name (empty)
	_, err := service.CreateProject(ctx, tenantID, "", "", "desc", "repo", "main", "private", "generic", nil, nil, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project name is required")

	// Save should not be called due to validation failure
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestServiceCreateProjectRepositoryError(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	name := "test-project"
	slug := "test-project"
	description := "Test description"
	gitRepo := "https://github.com/example/repo.git"

	// Mock repository error on exists check
	mockRepo.On("ExistsByNameAndTenantID", ctx, name, tenantID).Return(false, errors.New("database error"))
	mockRepo.On("ExistsBySlugAndTenantID", ctx, slug, tenantID).Return(false, nil).Maybe()

	_, err := service.CreateProject(ctx, tenantID, name, slug, description, gitRepo, "main", "private", "generic", nil, nil, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

func TestServiceGetProject(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()
	tenantID := uuid.New()

	expectedProject := NewProjectFromExisting(
		projectID, tenantID, "test", "test", "desc", "repo", "main", "generic",
		ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), nil, 1,
	)

	mockRepo.On("FindByID", ctx, projectID).Return(expectedProject, nil)

	project, err := service.GetProject(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, expectedProject, project)

	mockRepo.AssertExpectations(t)
}

func TestServiceGetProjectNotFound(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()

	mockRepo.On("FindByID", ctx, projectID).Return(nil, ErrProjectNotFound)

	_, err := service.GetProject(ctx, projectID)

	assert.Error(t, err)
	assert.Equal(t, ErrProjectNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestServiceListProjects(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	limit := 100
	offset := 0

	projects := []*Project{
		NewProjectFromExisting(uuid.New(), tenantID, "project1", "project1", "desc1", "repo1", "main", "generic", ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), nil, 1),
		NewProjectFromExisting(uuid.New(), tenantID, "project2", "project2", "desc2", "repo2", "main", "generic", ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), nil, 1),
	}
	totalCount := 2

	mockRepo.On("FindByTenantID", ctx, tenantID, (*uuid.UUID)(nil), limit, offset).Return(projects, nil)
	mockRepo.On("CountByTenantID", ctx, tenantID, (*uuid.UUID)(nil)).Return(totalCount, nil)

	result, total, err := service.ListProjects(ctx, tenantID, nil, limit, offset)

	require.NoError(t, err)
	assert.Equal(t, projects, result)
	assert.Equal(t, totalCount, total)

	mockRepo.AssertExpectations(t)
}

func TestServiceUpdateProject(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()
	tenantID := uuid.New()

	existingProject := NewProjectFromExisting(
		projectID, tenantID, "old-name", "old-name", "old desc", "old repo", "main", "generic",
		ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), nil, 1,
	)

	mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
	mockRepo.On("ExistsByNameAndTenantID", ctx, "new-name", tenantID).Return(false, nil)
	mockRepo.On("ExistsBySlugAndTenantID", ctx, "new-name", tenantID).Return(false, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*project.Project")).Return(nil).Run(func(args mock.Arguments) {
		project := args.Get(1).(*Project)
		project.updatedAt = time.Now()
	})

	updatedProject, err := service.UpdateProject(ctx, projectID, "new-name", "new-name", "new desc", "new repo", "develop", "github", nil, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "new-name", updatedProject.Name())
	assert.Equal(t, "new desc", updatedProject.Description())
	assert.Equal(t, "new repo", updatedProject.GitRepo())

	mockRepo.AssertExpectations(t)
}

func TestServiceUpdateProjectNotFound(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()

	mockRepo.On("FindByID", ctx, projectID).Return(nil, ErrProjectNotFound)

	_, err := service.UpdateProject(ctx, projectID, "new-name", "new-name", "new desc", "new repo", "main", "generic", nil, nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrProjectNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestServiceUpdateProjectValidationError(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()
	tenantID := uuid.New()

	existingProject := NewProjectFromExisting(
		projectID, tenantID, "old-name", "old-name", "old desc", "old repo", "main", "generic",
		ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), nil, 1,
	)

	mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
	mockRepo.On("ExistsByNameAndTenantID", ctx, "", tenantID).Return(false, nil)
	mockRepo.On("ExistsBySlugAndTenantID", ctx, "", tenantID).Return(false, nil)

	// Invalid update (empty name)
	_, err := service.UpdateProject(ctx, projectID, "", "", "new desc", "new repo", "main", "generic", nil, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project name is required")

	// Update should not be called
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestServiceDeleteProject(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()
	tenantID := uuid.New()

	existingProject := NewProjectFromExisting(
		projectID, tenantID, "test", "test", "desc", "repo", "main", "generic",
		ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), nil, 1,
	)

	mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*project.Project")).Return(nil)

	err := service.DeleteProject(ctx, projectID, nil)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestServiceDeleteProjectNotFound(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()

	mockRepo.On("FindByID", ctx, projectID).Return(nil, ErrProjectNotFound)

	err := service.DeleteProject(ctx, projectID, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrProjectNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestServiceDeleteProjectAlreadyDeleted(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	mockMemberRepo := &MockMemberRepository{}
	service := NewService(mockRepo, mockMemberRepo, nil, logger)

	ctx := context.Background()
	projectID := uuid.New()
	tenantID := uuid.New()
	deletedAt := time.Now()

	existingProject := NewProjectFromExisting(
		projectID, tenantID, "test", "test", "desc", "repo", "main", "generic",
		ProjectStatusActive, "private", nil, nil, false, 0, time.Now(), time.Now(), &deletedAt, 1,
	)

	mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)

	err := service.DeleteProject(ctx, projectID, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already deleted")

	// Update should not be called since Delete() fails
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}
