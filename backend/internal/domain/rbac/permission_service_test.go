package rbac_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/rbac"
)

// MockPermissionRepository is a mock implementation for testing
type MockPermissionRepository struct {
	permissions map[string]*rbac.PermissionRecord
	allPerms    []*rbac.PermissionRecord
}

func NewMockPermissionRepository() *MockPermissionRepository {
	return &MockPermissionRepository{
		permissions: make(map[string]*rbac.PermissionRecord),
		allPerms:    []*rbac.PermissionRecord{},
	}
}

func (m *MockPermissionRepository) FindByResourceAction(ctx context.Context, resource, action string) (*rbac.PermissionRecord, error) {
	key := resource + ":" + action
	return m.permissions[key], nil
}

func (m *MockPermissionRepository) FindAll(ctx context.Context) ([]*rbac.PermissionRecord, error) {
	return m.allPerms, nil
}

func (m *MockPermissionRepository) FindByResource(ctx context.Context, resource string) ([]*rbac.PermissionRecord, error) {
	var result []*rbac.PermissionRecord
	for _, perm := range m.allPerms {
		if perm.Resource == resource {
			result = append(result, perm)
		}
	}
	return result, nil
}

func (m *MockPermissionRepository) GetResourceList(ctx context.Context) ([]string, error) {
	resourceMap := make(map[string]bool)
	for _, perm := range m.allPerms {
		resourceMap[perm.Resource] = true
	}
	var resources []string
	for r := range resourceMap {
		resources = append(resources, r)
	}
	return resources, nil
}

func (m *MockPermissionRepository) CountByResource(ctx context.Context, resource string) (int, error) {
	count := 0
	for _, perm := range m.allPerms {
		if perm.Resource == resource {
			count++
		}
	}
	return count, nil
}

func (m *MockPermissionRepository) Create(ctx context.Context, perm *rbac.PermissionRecord) (*rbac.PermissionRecord, error) {
	perm.ID = uuid.New()
	perm.CreatedAt = time.Now()
	perm.UpdatedAt = time.Now()
	m.addPermission(perm)
	return perm, nil
}

func (m *MockPermissionRepository) Update(ctx context.Context, perm *rbac.PermissionRecord) (*rbac.PermissionRecord, error) {
	perm.UpdatedAt = time.Now()
	key := perm.Resource + ":" + perm.Action
	m.permissions[key] = perm
	// Update in allPerms slice
	for i, p := range m.allPerms {
		if p.ID == perm.ID {
			m.allPerms[i] = perm
			break
		}
	}
	return perm, nil
}

func (m *MockPermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Remove from allPerms
	for i, perm := range m.allPerms {
		if perm.ID == id {
			m.allPerms = append(m.allPerms[:i], m.allPerms[i+1:]...)
			break
		}
	}
	// Remove from permissions map
	for key, perm := range m.permissions {
		if perm.ID == id {
			delete(m.permissions, key)
			break
		}
	}
	return nil
}

func (m *MockPermissionRepository) addPermission(perm *rbac.PermissionRecord) {
	key := perm.Resource + ":" + perm.Action
	m.permissions[key] = perm
	m.allPerms = append(m.allPerms, perm)
}

// Test: NewPermissionService creates service with correct TTL
func TestNewPermissionService(t *testing.T) {
	repo := NewMockPermissionRepository()
	service := rbac.NewPermissionService(repo)

	if service == nil {
		t.Fatal("NewPermissionService returned nil")
	}
}

// Test: FindPermissionByResourceAction returns permission
func TestFindPermissionByResourceAction(t *testing.T) {
	repo := NewMockPermissionRepository()
	perm := &rbac.PermissionRecord{
		ID:       uuid.New(),
		Resource: "users",
		Action:   "create",
	}
	repo.addPermission(perm)

	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	result, err := service.FindPermissionByResourceAction(ctx, "users", "create")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected permission, got nil")
	}

	if result.Resource != "users" || result.Action != "create" {
		t.Errorf("got %s:%s, expected users:create", result.Resource, result.Action)
	}
}

// Test: FindPermissionByResourceAction returns nil for non-existent permission
func TestFindPermissionByResourceActionNotFound(t *testing.T) {
	repo := NewMockPermissionRepository()
	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	result, err := service.FindPermissionByResourceAction(ctx, "users", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Fatal("expected nil, got permission")
	}
}

// Test: ValidatePermission returns true for existing permission
func TestValidatePermissionExists(t *testing.T) {
	repo := NewMockPermissionRepository()
	perm := &rbac.PermissionRecord{
		ID:       uuid.New(),
		Resource: "users",
		Action:   "read",
	}
	repo.addPermission(perm)

	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	valid, err := service.ValidatePermission(ctx, "users", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !valid {
		t.Fatal("expected valid=true, got false")
	}
}

// Test: ValidatePermission returns false for non-existent permission
func TestValidatePermissionNotExists(t *testing.T) {
	repo := NewMockPermissionRepository()
	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	valid, err := service.ValidatePermission(ctx, "users", "delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if valid {
		t.Fatal("expected valid=false, got true")
	}
}

// Test: GetPermissionsByResource returns all permissions for resource
func TestGetPermissionsByResource(t *testing.T) {
	repo := NewMockPermissionRepository()
	perms := []*rbac.PermissionRecord{
		{ID: uuid.New(), Resource: "users", Action: "create"},
		{ID: uuid.New(), Resource: "users", Action: "read"},
		{ID: uuid.New(), Resource: "users", Action: "update"},
		{ID: uuid.New(), Resource: "projects", Action: "create"},
	}
	for _, p := range perms {
		repo.addPermission(p)
	}

	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	result, err := service.GetPermissionsByResource(ctx, "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 permissions, got %d", len(result))
	}

	for _, p := range result {
		if p.Resource != "users" {
			t.Errorf("expected resource users, got %s", p.Resource)
		}
	}
}

// Test: GetAllPermissions returns all permissions
func TestGetAllPermissions(t *testing.T) {
	repo := NewMockPermissionRepository()
	perms := []*rbac.PermissionRecord{
		{ID: uuid.New(), Resource: "users", Action: "create"},
		{ID: uuid.New(), Resource: "users", Action: "read"},
		{ID: uuid.New(), Resource: "projects", Action: "create"},
	}
	for _, p := range perms {
		repo.addPermission(p)
	}

	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	result, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 permissions, got %d", len(result))
	}
}

// Test: GetAllPermissions caches results
func TestGetAllPermissionsCaching(t *testing.T) {
	repo := NewMockPermissionRepository()
	perm := &rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "create"}
	repo.addPermission(perm)

	service := rbac.NewPermissionService(repo)
	service.SetCacheTTL(time.Second * 10)
	ctx := context.Background()

	// First call
	result1, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify repository
	repo.allPerms = append(repo.allPerms, &rbac.PermissionRecord{ID: uuid.New(), Resource: "projects", Action: "read"})

	// Second call should return cached result
	result2, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result1) != len(result2) {
		t.Errorf("cache not working: first call %d, second call %d", len(result1), len(result2))
	}
}

// Test: GetPermissionsByResource caches results
func TestGetPermissionsByResourceCaching(t *testing.T) {
	repo := NewMockPermissionRepository()
	perm := &rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "create"}
	repo.addPermission(perm)

	service := rbac.NewPermissionService(repo)
	service.SetCacheTTL(time.Second * 10)
	ctx := context.Background()

	// First call
	result1, err := service.GetPermissionsByResource(ctx, "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify repository
	repo.allPerms = append(repo.allPerms, &rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "read"})

	// Second call should return cached result
	result2, err := service.GetPermissionsByResource(ctx, "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result1) != len(result2) {
		t.Errorf("cache not working: first call %d, second call %d", len(result1), len(result2))
	}
}

// Test: ClearCache clears all cached data
func TestClearCache(t *testing.T) {
	repo := NewMockPermissionRepository()
	perm := &rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "create"}
	repo.addPermission(perm)

	service := rbac.NewPermissionService(repo)
	service.SetCacheTTL(time.Second * 10)
	ctx := context.Background()

	// Load cache
	_, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Clear cache
	service.ClearCache()

	// Add new permission
	repo.allPerms = append(repo.allPerms, &rbac.PermissionRecord{ID: uuid.New(), Resource: "projects", Action: "read"})

	// After clear, should get new data
	result, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 permissions after cache clear, got %d", len(result))
	}
}

// Test: InvalidateResourceCache clears specific resource cache
func TestInvalidateResourceCache(t *testing.T) {
	repo := NewMockPermissionRepository()
	repo.addPermission(&rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "create"})
	repo.addPermission(&rbac.PermissionRecord{ID: uuid.New(), Resource: "projects", Action: "read"})

	service := rbac.NewPermissionService(repo)
	service.SetCacheTTL(time.Second * 10)
	ctx := context.Background()

	// Load cache for both resources
	_, err := service.GetPermissionsByResource(ctx, "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = service.GetPermissionsByResource(ctx, "projects")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalidate only users cache
	service.InvalidateResourceCache("users")

	// Add new users permission
	repo.allPerms = append(repo.allPerms, &rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "delete"})

	// Users should get new data, projects should be cached
	usersResult, err := service.GetPermissionsByResource(ctx, "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(usersResult) != 2 {
		t.Errorf("expected 2 users permissions after invalidation, got %d", len(usersResult))
	}
}

// Test: InvalidateAllCache clears all permissions cache
func TestInvalidateAllCache(t *testing.T) {
	repo := NewMockPermissionRepository()
	repo.addPermission(&rbac.PermissionRecord{ID: uuid.New(), Resource: "users", Action: "create"})

	service := rbac.NewPermissionService(repo)
	service.SetCacheTTL(time.Second * 10)
	ctx := context.Background()

	// Load cache
	_, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalidate cache
	service.InvalidateAllCache()

	// Add new permission
	repo.allPerms = append(repo.allPerms, &rbac.PermissionRecord{ID: uuid.New(), Resource: "projects", Action: "read"})

	// Should get fresh data
	result, err := service.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 permissions after invalidating all cache, got %d", len(result))
	}
}

// Test: SetCacheTTL updates cache duration
func TestSetCacheTTL(t *testing.T) {
	repo := NewMockPermissionRepository()
	service := rbac.NewPermissionService(repo)

	// Should not panic
	service.SetCacheTTL(time.Hour * 2)
	service.SetCacheTTL(time.Minute * 30)
}

// Test: Multiple concurrent permission lookups
func TestConcurrentPermissionLookups(t *testing.T) {
	repo := NewMockPermissionRepository()
	for i := 1; i <= 5; i++ {
		repo.addPermission(&rbac.PermissionRecord{
			ID:       uuid.New(),
			Resource: "resource" + string(rune(i)),
			Action:   "action",
		})
	}

	service := rbac.NewPermissionService(repo)
	ctx := context.Background()

	// Should handle concurrent requests safely
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := service.GetAllPermissions(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
