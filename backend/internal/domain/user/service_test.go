package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockRepository is a mock implementation of Repository for testing
type MockRepository struct {
	users                  map[uuid.UUID]*User
	usersByEmail           map[string]*User
	usersByTenantID        map[uuid.UUID][]*User
	emailExists            map[string]bool
	tenantUserCount        map[uuid.UUID]int
	saveCallCount          int
	findByIDCallCount      int
	findByEmailCallCount   int
	updateCallCount        int
	deleteCallCount        int
	shouldFailNextSave     bool
	shouldFailNextUpdate   bool
	shouldFailNextDelete   bool
	shouldFailNextFindByID bool
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:           make(map[uuid.UUID]*User),
		usersByEmail:    make(map[string]*User),
		usersByTenantID: make(map[uuid.UUID][]*User),
		emailExists:     make(map[string]bool),
		tenantUserCount: make(map[uuid.UUID]int),
	}
}

func (m *MockRepository) Save(ctx context.Context, user *User) error {
	if m.shouldFailNextSave {
		m.shouldFailNextSave = false
		return ErrUserNotFound
	}
	m.saveCallCount++
	m.users[user.ID()] = user
	m.usersByEmail[user.Email()] = user
	m.emailExists[user.Email()] = true
	return nil
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.shouldFailNextFindByID {
		m.shouldFailNextFindByID = false
		return nil, ErrUserNotFound
	}
	m.findByIDCallCount++
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (m *MockRepository) FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error) {
	result := make(map[uuid.UUID]*User)
	for _, id := range ids {
		if user, ok := m.users[id]; ok {
			result[id] = user
		}
	}
	return result, nil
}

func (m *MockRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	m.findByEmailCallCount++
	if user, ok := m.usersByEmail[email]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func (m *MockRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*User, error) {
	if users, ok := m.usersByTenantID[tenantID]; ok {
		return users, nil
	}
	return []*User{}, nil
}

func (m *MockRepository) Update(ctx context.Context, user *User) error {
	if m.shouldFailNextUpdate {
		m.shouldFailNextUpdate = false
		return ErrUserNotFound
	}
	m.updateCallCount++
	m.users[user.ID()] = user
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.shouldFailNextDelete {
		m.shouldFailNextDelete = false
		return ErrUserNotFound
	}
	m.deleteCallCount++
	if _, ok := m.users[id]; !ok {
		return ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *MockRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return m.emailExists[email], nil
}

func (m *MockRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	if count, ok := m.tenantUserCount[tenantID]; ok {
		return count, nil
	}
	return len(m.usersByTenantID[tenantID]), nil
}

func (m *MockRepository) GetTotalUserCount(ctx context.Context) (int, error) {
	return len(m.users), nil
}

func (m *MockRepository) GetActiveUserCount(ctx context.Context, days int) (int, error) {
	// Mock implementation - return half the users as "active"
	return len(m.users) / 2, nil
}

func (m *MockRepository) FindAll(ctx context.Context) ([]*User, error) {
	users := make([]*User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

// Tests for UserService.CreateUser
func TestServiceCreateUser_Success(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()
	email := "john@example.com"
	firstName := "John"
	lastName := "Doe"
	password := "securePassword123"

	user, err := service.CreateUser(ctx, tenantID, email, firstName, lastName, password)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email())
	assert.Equal(t, firstName, user.FirstName())
	assert.Equal(t, lastName, user.LastName())
	assert.Equal(t, UserStatusPending, user.Status())
	assert.True(t, user.VerifyPassword(password))
	assert.Equal(t, 1, repo.saveCallCount)
}

func TestServiceCreateUser_DuplicateEmail(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()
	email := "john@example.com"

	// Create first user
	_, err := service.CreateUser(ctx, tenantID, email, "John", "Doe", "password123")
	require.NoError(t, err)

	// Try to create second user with same email
	_, err = service.CreateUser(ctx, tenantID, email, "Jane", "Doe", "password123")
	assert.Error(t, err)
	assert.Equal(t, ErrUserExists, err)
}

func TestServiceCreateUser_InvalidEmail(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateUser(ctx, tenantID, "invalid-email", "John", "Doe", "password123")
	assert.Error(t, err)
}

func TestServiceCreateUser_WeakPassword(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateUser(ctx, tenantID, "john@example.com", "John", "Doe", "short")
	assert.Error(t, err)
}

func TestServiceCreateUser_RepositoryError(t *testing.T) {
	repo := NewMockRepository()
	repo.shouldFailNextSave = true
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateUser(ctx, tenantID, "john@example.com", "John", "Doe", "password123")
	assert.Error(t, err)
}

// Tests for UserService.GetUserByID
func TestServiceGetUserByID_Success(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, uuid.New(), "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// Now retrieve it
	retrieved, err := service.GetUserByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, user.ID(), retrieved.ID())
	assert.Equal(t, user.Email(), retrieved.Email())
	assert.Equal(t, 1, repo.findByIDCallCount)
}

func TestServiceGetUserByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	nonexistentID := uuid.New()

	_, err := service.GetUserByID(ctx, nonexistentID)
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

// Tests for UserService.UpdateUser
func TestServiceUpdateUser_Success(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, uuid.New(), "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// Update user first name
	user.UpdateFirstName("Jane")
	err = service.UpdateUser(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.updateCallCount)

	// Verify update
	updated, err := service.GetUserByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, "Jane", updated.FirstName())
}

func TestServiceUpdateUser_RepositoryError(t *testing.T) {
	repo := NewMockRepository()
	repo.shouldFailNextUpdate = true
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, uuid.New(), "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	user.UpdateFirstName("Jane")
	err = service.UpdateUser(ctx, user)
	assert.Error(t, err)
}

// Tests for UserService.DeleteUser
func TestServiceDeleteUser_Success(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, uuid.New(), "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// Delete user
	err = service.DeleteUser(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, 1, repo.deleteCallCount)

	// Verify deletion
	_, err = service.GetUserByID(ctx, user.ID())
	assert.Error(t, err)
}

func TestServiceDeleteUser_NotFound(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	err := service.DeleteUser(ctx, uuid.New())
	assert.Error(t, err)
}

// Tests for UserService.GetUsersByTenantID
func TestServiceGetUsersByTenantID_Success(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()

	// Create multiple users for same tenant
	u1, err := service.CreateUser(ctx, tenantID, "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)
	u2, err := service.CreateUser(ctx, tenantID, "jane@example.com", "Jane", "Doe", "password123")
	require.NoError(t, err)
	repo.usersByTenantID[tenantID] = []*User{u1, u2}
	repo.tenantUserCount[tenantID] = 2

	// Create user for different tenant
	otherTenantID := uuid.New()
	u3, err := service.CreateUser(ctx, otherTenantID, "bob@example.com", "Bob", "Smith", "password123")
	require.NoError(t, err)
	repo.usersByTenantID[otherTenantID] = []*User{u3}
	repo.tenantUserCount[otherTenantID] = 1

	// Retrieve users for first tenant
	users, err := service.GetUsersByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(users))

	// Verify results are what repository exposed for the tenant
	assert.Equal(t, u1.ID(), users[0].ID())
	assert.Equal(t, u2.ID(), users[1].ID())
}

func TestServiceGetUsersByTenantID_Empty(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()

	users, err := service.GetUsersByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, len(users))
}

// Tests for ChangePassword
func TestServiceChangePassword_Success(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, uuid.New(), "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	oldPassword := "password123"
	newPassword := "newPassword456"

	// Change password
	err = service.ChangePassword(ctx, user.ID(), newPassword)
	require.NoError(t, err)

	// Verify old password doesn't work
	user, err = service.GetUserByID(ctx, user.ID())
	require.NoError(t, err)
	assert.False(t, user.VerifyPassword(oldPassword))

	// Verify new password works
	assert.True(t, user.VerifyPassword(newPassword))
}

func TestServiceChangePassword_WeakPassword(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, uuid.New(), "john@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// Try weak password
	err = service.ChangePassword(ctx, user.ID(), "weak")
	assert.Error(t, err)
}

func TestServiceChangePassword_UserNotFound(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	err := service.ChangePassword(ctx, uuid.New(), "newPassword456")
	assert.Error(t, err)
}

// Benchmark tests for N+1 query detection
func TestServiceGetUsersByTenantID_NoNPlusOneQueries(t *testing.T) {
	repo := NewMockRepository()
	logger := zap.NewNop()
	service := NewService(repo, logger, "test-secret")

	ctx := context.Background()
	tenantID := uuid.New()

	// Create 10 users
	for i := 0; i < 10; i++ {
		email := "user" + string(rune('0'+i)) + "@example.com"
		usr, err := service.CreateUser(ctx, tenantID, email, "User", "Test", "password123")
		require.NoError(t, err)
		repo.usersByTenantID[tenantID] = append(repo.usersByTenantID[tenantID], usr)
	}
	repo.tenantUserCount[tenantID] = 10

	// Reset call counters
	repo.findByIDCallCount = 0

	// Get all users - should be 1 query, not 11
	users, err := service.GetUsersByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 10, len(users))

	// Verify we didn't do N+1 queries (each FindByID would be 1 call)
	// Repository should not have called FindByID multiple times
	assert.Equal(t, 0, repo.findByIDCallCount)
}
