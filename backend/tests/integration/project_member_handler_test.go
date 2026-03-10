package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/srikarm/image-factory/internal/domain/project"
)

// TestProjectMemberLifecycle tests the complete member management flow
func TestProjectMemberLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup with in-memory repositories
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	memberRepo := newInMemoryMemberRepository()
	projectRepo := newInMemoryProjectRepository()

	svc := project.NewService(projectRepo, memberRepo, nil, logger)

	// Create test data
	tenantID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	assignedByUserID := uuid.New()

	// Create a project
	proj, err := svc.CreateProject(ctx, tenantID, "test-project", "", "test description", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	// Test 1: Add member
	member1, err := svc.AddMember(ctx, proj.ID(), userID1, &assignedByUserID)
	require.NoError(t, err)
	assert.Equal(t, proj.ID(), member1.ProjectID())
	assert.Equal(t, userID1, member1.UserID())

	// Test 2: Add second member
	member2, err := svc.AddMember(ctx, proj.ID(), userID2, &assignedByUserID)
	require.NoError(t, err)
	assert.Equal(t, userID2, member2.UserID())

	// Test 3: List members
	members, totalCount, err := svc.ListMembers(ctx, proj.ID(), 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, totalCount)
	assert.Len(t, members, 2)

	// Test 4: Update member role
	roleID := uuid.New()
	updated, err := svc.UpdateMemberRole(ctx, proj.ID(), userID1, &roleID)
	require.NoError(t, err)
	assert.NotNil(t, updated.RoleID())
	assert.Equal(t, roleID, *updated.RoleID())

	// Test 5: Clear member role
	updated, err = svc.UpdateMemberRole(ctx, proj.ID(), userID1, nil)
	require.NoError(t, err)
	assert.Nil(t, updated.RoleID())

	// Test 6: Remove member
	err = svc.RemoveMember(ctx, proj.ID(), userID1)
	require.NoError(t, err)

	// Test 7: Verify member was removed
	members, totalCount, err = svc.ListMembers(ctx, proj.ID(), 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, totalCount)
	assert.Len(t, members, 1)
	assert.Equal(t, userID2, members[0].UserID())
}

// TestAddMemberDuplicate tests adding a member that already exists
func TestAddMemberDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	memberRepo := newInMemoryMemberRepository()
	projectRepo := newInMemoryProjectRepository()

	svc := project.NewService(projectRepo, memberRepo, nil, logger)

	tenantID := uuid.New()
	userID := uuid.New()
	assignedByUserID := uuid.New()

	// Create project and add member
	proj, err := svc.CreateProject(ctx, tenantID, "test-project", "", "", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	_, err = svc.AddMember(ctx, proj.ID(), userID, &assignedByUserID)
	require.NoError(t, err)

	// Try to add same member again
	_, err = svc.AddMember(ctx, proj.ID(), userID, &assignedByUserID)
	require.Error(t, err)
	assert.Equal(t, project.ErrMemberAlreadyExists, err)
}

// TestRemoveNonexistentMember tests removing a member that doesn't exist
func TestRemoveNonexistentMember(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	memberRepo := newInMemoryMemberRepository()
	projectRepo := newInMemoryProjectRepository()

	svc := project.NewService(projectRepo, memberRepo, nil, logger)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create project
	proj, err := svc.CreateProject(ctx, tenantID, "test-project", "", "", "", "main", "private", "generic", nil, nil, false)
	require.NoError(t, err)

	// Try to remove non-existent member
	err = svc.RemoveMember(ctx, proj.ID(), userID)
	require.Error(t, err)
	assert.Equal(t, project.ErrMemberNotFound, err)
}

// In-memory test repositories
type inMemoryProjectRepository struct {
	projects map[uuid.UUID]*project.Project
}

func newInMemoryProjectRepository() *inMemoryProjectRepository {
	return &inMemoryProjectRepository{
		projects: make(map[uuid.UUID]*project.Project),
	}
}

func (r *inMemoryProjectRepository) Save(ctx context.Context, p *project.Project) error {
	r.projects[p.ID()] = p
	return nil
}

func (r *inMemoryProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*project.Project, error) {
	if proj, exists := r.projects[id]; exists {
		return proj, nil
	}
	return nil, project.ErrProjectNotFound
}

func (r *inMemoryProjectRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID, limit, offset int) ([]*project.Project, error) {
	var projects []*project.Project
	for _, p := range r.projects {
		if p.TenantID() == tenantID {
			if viewerID != nil && p.IsDraft() && (p.CreatedBy() == nil || *p.CreatedBy() != *viewerID) {
				continue
			}
			projects = append(projects, p)
		}
	}
	return projects, nil
}

func (r *inMemoryProjectRepository) FindByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (*project.Project, error) {
	for _, p := range r.projects {
		if p.Name() == name && p.TenantID() == tenantID {
			return p, nil
		}
	}
	return nil, project.ErrProjectNotFound
}

func (r *inMemoryProjectRepository) Update(ctx context.Context, p *project.Project) error {
	if _, exists := r.projects[p.ID()]; !exists {
		return project.ErrProjectNotFound
	}
	r.projects[p.ID()] = p
	return nil
}

func (r *inMemoryProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, exists := r.projects[id]; !exists {
		return project.ErrProjectNotFound
	}
	delete(r.projects, id)
	return nil
}

func (r *inMemoryProjectRepository) PurgeDeletedBefore(ctx context.Context, cutoff time.Time) (int, error) {
	return 0, nil
}

func (r *inMemoryProjectRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID) (int, error) {
	count := 0
	for _, p := range r.projects {
		if p.TenantID() == tenantID {
			if viewerID != nil && p.IsDraft() && (p.CreatedBy() == nil || *p.CreatedBy() != *viewerID) {
				continue
			}
			count++
		}
	}
	return count, nil
}

func (r *inMemoryProjectRepository) ExistsByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (bool, error) {
	for _, p := range r.projects {
		if p.Name() == name && p.TenantID() == tenantID {
			return true, nil
		}
	}
	return false, nil
}

func (r *inMemoryProjectRepository) ExistsBySlugAndTenantID(ctx context.Context, slug string, tenantID uuid.UUID) (bool, error) {
	for _, p := range r.projects {
		if p.Slug() == slug && p.TenantID() == tenantID {
			return true, nil
		}
	}
	return false, nil
}

type inMemoryMemberRepository struct {
	members map[uuid.UUID]*project.Member
}

func newInMemoryMemberRepository() *inMemoryMemberRepository {
	return &inMemoryMemberRepository{
		members: make(map[uuid.UUID]*project.Member),
	}
}

func (r *inMemoryMemberRepository) CreateMember(ctx context.Context, m *project.Member) error {
	r.members[m.ID()] = m
	return nil
}

func (r *inMemoryMemberRepository) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*project.Member, error) {
	for _, m := range r.members {
		if m.ProjectID() == projectID && m.UserID() == userID {
			return m, nil
		}
	}
	return nil, project.ErrMemberNotFound
}

func (r *inMemoryMemberRepository) ListMembers(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*project.Member, int, error) {
	var members []*project.Member
	for _, m := range r.members {
		if m.ProjectID() == projectID {
			members = append(members, m)
		}
	}

	total := len(members)
	if offset >= total {
		return []*project.Member{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return members[offset:end], total, nil
}

func (r *inMemoryMemberRepository) ListUserProjects(ctx context.Context, userID, tenantID uuid.UUID, limit, offset int) ([]*project.Project, int, error) {
	return []*project.Project{}, 0, nil
}

func (r *inMemoryMemberRepository) UpdateMember(ctx context.Context, m *project.Member) error {
	if _, exists := r.members[m.ID()]; !exists {
		return project.ErrMemberNotFound
	}
	r.members[m.ID()] = m
	return nil
}

func (r *inMemoryMemberRepository) DeleteMember(ctx context.Context, projectID, userID uuid.UUID) error {
	for id, m := range r.members {
		if m.ProjectID() == projectID && m.UserID() == userID {
			delete(r.members, id)
			return nil
		}
	}
	return project.ErrMemberNotFound
}

func (r *inMemoryMemberRepository) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	for _, m := range r.members {
		if m.ProjectID() == projectID && m.UserID() == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *inMemoryMemberRepository) CountMembers(ctx context.Context, projectID uuid.UUID) (int, error) {
	count := 0
	for _, m := range r.members {
		if m.ProjectID() == projectID {
			count++
		}
	}
	return count, nil
}

func (r *inMemoryMemberRepository) DeleteProjectMembers(ctx context.Context, projectID uuid.UUID) error {
	for id, m := range r.members {
		if m.ProjectID() == projectID {
			delete(r.members, id)
		}
	}
	return nil
}
