package rbac

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PermissionRecord represents a system permission as stored in the database
// This extends the basic Permission type with database metadata
type PermissionRecord struct {
	ID                 uuid.UUID
	Resource           string
	Action             string
	Description        *string
	Category           *string
	IsSystemPermission bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// GetIsSystemPermission returns whether this permission is a system-level permission
// System permissions can only be granted to system administrators
func (pr *PermissionRecord) GetIsSystemPermission() bool {
	return pr.IsSystemPermission
}

// PermissionRepository defines methods to interact with permissions
type PermissionRepository interface {
	FindByResourceAction(ctx context.Context, resource, action string) (*PermissionRecord, error)
	FindAll(ctx context.Context) ([]*PermissionRecord, error)
	FindByResource(ctx context.Context, resource string) ([]*PermissionRecord, error)
	GetResourceList(ctx context.Context) ([]string, error)
	CountByResource(ctx context.Context, resource string) (int, error)
	Create(ctx context.Context, perm *PermissionRecord) (*PermissionRecord, error)
	Update(ctx context.Context, perm *PermissionRecord) (*PermissionRecord, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PermissionService provides permission-related business logic with caching
type PermissionService struct {
	repo              PermissionRepository
	cache             map[string][]*PermissionRecord // resource -> []*PermissionRecord
	cacheMutex        sync.RWMutex
	cacheExpiry       map[string]time.Time
	cacheTTL          time.Duration
	allPermissions    []*PermissionRecord
	allPermissionsExp time.Time
	allPermissionsMux sync.RWMutex
}

// NewPermissionService creates a new instance of PermissionService
func NewPermissionService(repo PermissionRepository) *PermissionService {
	return &PermissionService{
		repo:        repo,
		cache:       make(map[string][]*PermissionRecord),
		cacheExpiry: make(map[string]time.Time),
		cacheTTL:    time.Hour, // 1 hour cache TTL
	}
}

// GetAllPermissions returns all permissions from cache or database
func (ps *PermissionService) GetAllPermissions(ctx context.Context) ([]*PermissionRecord, error) {
	ps.allPermissionsMux.RLock()
	if len(ps.allPermissions) > 0 && time.Now().Before(ps.allPermissionsExp) {
		defer ps.allPermissionsMux.RUnlock()
		return ps.allPermissions, nil
	}
	ps.allPermissionsMux.RUnlock()

	// Fetch from database
	perms, err := ps.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all permissions: %w", err)
	}

	// Update cache
	ps.allPermissionsMux.Lock()
	ps.allPermissions = perms
	ps.allPermissionsExp = time.Now().Add(ps.cacheTTL)
	ps.allPermissionsMux.Unlock()

	return perms, nil
}

// GetPermissionsByResource returns all permissions for a specific resource from cache or database
func (ps *PermissionService) GetPermissionsByResource(ctx context.Context, resource string) ([]*PermissionRecord, error) {
	ps.cacheMutex.RLock()
	if perms, exists := ps.cache[resource]; exists && time.Now().Before(ps.cacheExpiry[resource]) {
		defer ps.cacheMutex.RUnlock()
		return perms, nil
	}
	ps.cacheMutex.RUnlock()

	// Fetch from database
	perms, err := ps.repo.FindByResource(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch permissions for resource '%s': %w", resource, err)
	}

	// Update cache
	ps.cacheMutex.Lock()
	ps.cache[resource] = perms
	ps.cacheExpiry[resource] = time.Now().Add(ps.cacheTTL)
	ps.cacheMutex.Unlock()

	return perms, nil
}

// ValidatePermission checks if a specific permission (resource:action) exists
func (ps *PermissionService) ValidatePermission(ctx context.Context, resource, action string) (bool, error) {
	perm, err := ps.FindPermissionByResourceAction(ctx, resource, action)
	if err != nil {
		return false, err
	}
	return perm != nil, nil
}

// FindPermissionByResourceAction finds a specific permission by resource and action
func (ps *PermissionService) FindPermissionByResourceAction(ctx context.Context, resource, action string) (*PermissionRecord, error) {
	perm, err := ps.repo.FindByResourceAction(ctx, resource, action)
	if err != nil {
		return nil, fmt.Errorf("failed to find permission %s:%s: %w", resource, action, err)
	}
	return perm, nil
}

// ClearCache clears all cached permissions (useful for testing and manual cache invalidation)
func (ps *PermissionService) ClearCache() {
	ps.cacheMutex.Lock()
	defer ps.cacheMutex.Unlock()

	ps.cache = make(map[string][]*PermissionRecord)
	ps.cacheExpiry = make(map[string]time.Time)

	ps.allPermissionsMux.Lock()
	defer ps.allPermissionsMux.Unlock()
	ps.allPermissions = nil
	ps.allPermissionsExp = time.Time{}
}

// InvalidateResourceCache invalidates cache for a specific resource
func (ps *PermissionService) InvalidateResourceCache(resource string) {
	ps.cacheMutex.Lock()
	defer ps.cacheMutex.Unlock()
	delete(ps.cache, resource)
	delete(ps.cacheExpiry, resource)
}

// InvalidateAllCache invalidates all permissions cache
func (ps *PermissionService) InvalidateAllCache() {
	ps.allPermissionsMux.Lock()
	defer ps.allPermissionsMux.Unlock()
	ps.allPermissions = nil
	ps.allPermissionsExp = time.Time{}
}

// SetCacheTTL updates the cache time-to-live duration
func (ps *PermissionService) SetCacheTTL(ttl time.Duration) {
	ps.cacheTTL = ttl
}

// CreatePermission creates a new permission
func (ps *PermissionService) CreatePermission(ctx context.Context, resource, action string, description, category *string) (*PermissionRecord, error) {
	// Check if permission already exists
	existing, err := ps.FindPermissionByResourceAction(ctx, resource, action)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing permission: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("permission %s:%s already exists", resource, action)
	}

	// Create new permission record
	now := time.Now()
	perm := &PermissionRecord{
		ID:                 uuid.New(),
		Resource:           resource,
		Action:             action,
		Description:        description,
		Category:           category,
		IsSystemPermission: false, // User-created permissions are not system permissions
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Save to database
	created, err := ps.repo.Create(ctx, perm)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Invalidate cache
	ps.InvalidateAllCache()
	ps.InvalidateResourceCache(resource)

	return created, nil
}

// UpdatePermission updates an existing permission (only non-system permissions)
func (ps *PermissionService) UpdatePermission(ctx context.Context, id uuid.UUID, description, category *string) (*PermissionRecord, error) {
	// First get the existing permission
	allPerms, err := ps.GetAllPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	var existing *PermissionRecord
	for _, p := range allPerms {
		if p.ID == id {
			existing = p
			break
		}
	}

	if existing == nil {
		return nil, fmt.Errorf("permission with ID %s not found", id)
	}

	// Check if it's a system permission
	if existing.IsSystemPermission {
		return nil, fmt.Errorf("cannot update system permission")
	}

	// Update the permission
	existing.Description = description
	existing.Category = category
	existing.UpdatedAt = time.Now()

	updated, err := ps.repo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	// Invalidate cache
	ps.InvalidateAllCache()
	ps.InvalidateResourceCache(existing.Resource)

	return updated, nil
}

// DeletePermission deletes a permission (only non-system permissions)
func (ps *PermissionService) DeletePermission(ctx context.Context, id uuid.UUID) error {
	// First get the existing permission
	allPerms, err := ps.GetAllPermissions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}

	var existing *PermissionRecord
	for _, p := range allPerms {
		if p.ID == id {
			existing = p
			break
		}
	}

	if existing == nil {
		return fmt.Errorf("permission with ID %s not found", id)
	}

	// Check if it's a system permission
	if existing.IsSystemPermission {
		return fmt.Errorf("cannot delete system permission")
	}

	// Delete from database
	err = ps.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	// Invalidate cache
	ps.InvalidateAllCache()
	ps.InvalidateResourceCache(existing.Resource)

	return nil
}
