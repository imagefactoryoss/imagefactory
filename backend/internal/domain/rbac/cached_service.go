package rbac

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/cache"
	"github.com/srikarm/image-factory/internal/infrastructure/metrics"
)

// CachedService wraps RBAC service with caching layer for roles
type CachedService struct {
	service          *Service
	roleCache        cache.RoleCache
	logger           *zap.Logger
	roleCacheTTL     time.Duration
	cacheMetrics     *metrics.CacheMetrics     // Optional metrics tracking
	operationMetrics *metrics.OperationMetrics // Optional operation metrics
}

// NewCachedService creates a new cached RBAC service
func NewCachedService(service *Service, roleCache cache.RoleCache, logger *zap.Logger) *CachedService {
	return &CachedService{
		service:          service,
		roleCache:        roleCache,
		logger:           logger,
		roleCacheTTL:     1 * time.Hour,           // Default 1 hour TTL
		cacheMetrics:     &metrics.CacheMetrics{}, // Initialize metrics
		operationMetrics: &metrics.OperationMetrics{},
	}
}

// SetRoleCacheTTL sets the TTL for role cache entries
func (cs *CachedService) SetRoleCacheTTL(ttl time.Duration) {
	cs.roleCacheTTL = ttl
}

// GetUserRoles retrieves roles for a user with caching
func (cs *CachedService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error) {
	m := metrics.NewMeasurement(cs.operationMetrics)
	defer m.Record()

	// Try cache first
	cachedRoleIDs, found, err := cs.roleCache.Get(ctx, userID)
	if err != nil {
		cs.logger.Error("Cache get error", zap.Error(err))
		// Fall through to database
	} else if found {
		cs.logger.Debug("Cache hit for user roles",
			zap.String("user_id", userID.String()),
			zap.Int("role_count", len(cachedRoleIDs)))
		cs.cacheMetrics.RecordHit() // Record metric
		// Convert role IDs back to Role objects
		return cs.getRolesByIDs(ctx, cachedRoleIDs)
	}

	cs.cacheMetrics.RecordMiss() // Record metric

	// Cache miss - fetch from database
	roles, err := cs.service.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Store in cache for future access
	roleIDs := make([]uuid.UUID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID()
	}

	if err := cs.roleCache.Set(ctx, userID, roleIDs, cs.roleCacheTTL); err != nil {
		cs.logger.Warn("Failed to cache user roles", zap.Error(err))
		// Don't fail the request if caching fails
	}

	return roles, nil
}

// GetUserRolesBatch retrieves roles for multiple users with caching
func (cs *CachedService) GetUserRolesBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*Role, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID][]*Role), nil
	}

	// Try to get from cache
	cachedRoles, err := cs.roleCache.GetBatch(ctx, userIDs)
	if err != nil {
		cs.logger.Error("Cache batch get error", zap.Error(err))
		// Fall through to database
	}

	// Determine which users need to be fetched from database
	uncachedUserIDs := make([]uuid.UUID, 0)
	for _, userID := range userIDs {
		if _, exists := cachedRoles[userID]; !exists {
			uncachedUserIDs = append(uncachedUserIDs, userID)
		}
	}

	// Fetch uncached users from database
	var dbRoles map[uuid.UUID][]*Role
	if len(uncachedUserIDs) > 0 {
		var err error
		dbRoles, err = cs.service.GetUserRolesBatch(ctx, uncachedUserIDs)
		if err != nil {
			return nil, err
		}

		// Cache the newly fetched roles
		roleIDsToCache := make(map[uuid.UUID][]uuid.UUID)
		for userID, roles := range dbRoles {
			roleIDs := make([]uuid.UUID, len(roles))
			for i, role := range roles {
				roleIDs[i] = role.ID()
			}
			roleIDsToCache[userID] = roleIDs
		}

		if err := cs.roleCache.SetBatch(ctx, roleIDsToCache, cs.roleCacheTTL); err != nil {
			cs.logger.Warn("Failed to cache batch user roles", zap.Error(err))
		}
	} else {
		dbRoles = make(map[uuid.UUID][]*Role)
	}

	// Merge cached and newly fetched roles
	result := make(map[uuid.UUID][]*Role)

	// Add cached roles
	for userID, roleIDs := range cachedRoles {
		roles, err := cs.getRolesByIDs(ctx, roleIDs)
		if err != nil {
			cs.logger.Warn("Failed to convert cached role IDs to roles", zap.Error(err))
			roles = []*Role{} // Empty fallback
		}
		result[userID] = roles
	}

	// Add database roles
	for userID, roles := range dbRoles {
		result[userID] = roles
	}

	// Ensure all requested users are in result (even with empty roles)
	for _, userID := range userIDs {
		if _, exists := result[userID]; !exists {
			result[userID] = []*Role{}
		}
	}

	cs.logger.Debug("Batch role lookup completed",
		zap.Int("total_users", len(userIDs)),
		zap.Int("cached_hits", len(cachedRoles)),
		zap.Int("db_fetches", len(uncachedUserIDs)))

	return result, nil
}

// InvalidateUserRolesCache invalidates cached roles for a user
// This should be called when role assignments change
func (cs *CachedService) InvalidateUserRolesCache(ctx context.Context, userID uuid.UUID) error {
	return cs.roleCache.Invalidate(ctx, userID)
}

// InvalidateUserRolesCacheBatch invalidates cached roles for multiple users
func (cs *CachedService) InvalidateUserRolesCacheBatch(ctx context.Context, userIDs []uuid.UUID) error {
	return cs.roleCache.InvalidateBatch(ctx, userIDs)
}

// InvalidateAllRolesCache clears the entire role cache
// This should be called when role definitions change
func (cs *CachedService) InvalidateAllRolesCache(ctx context.Context) error {
	return cs.roleCache.InvalidateAll(ctx)
}

// GetCacheStats returns cache statistics
func (cs *CachedService) GetCacheStats() cache.CacheStats {
	return cs.roleCache.Stats()
}

// Delegate other methods to underlying service

// CreateRole delegates to underlying service
func (cs *CachedService) CreateRole(ctx context.Context, tenantID uuid.UUID, name, description string, permissions []Permission) (*Role, error) {
	return cs.service.CreateRole(ctx, tenantID, name, description, permissions)
}

// CreateSystemRole delegates to underlying service
func (cs *CachedService) CreateSystemRole(ctx context.Context, name, description string, permissions []Permission) (*Role, error) {
	return cs.service.CreateSystemRole(ctx, name, description, permissions)
}

// GetRoleByID delegates to underlying service
func (cs *CachedService) GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	return cs.service.GetRoleByID(ctx, id)
}

// GetRoleByName delegates to underlying service
func (cs *CachedService) GetRoleByName(ctx context.Context, tenantID uuid.UUID, name string) (*Role, error) {
	return cs.service.GetRoleByName(ctx, tenantID, name)
}

// GetRolesByTenantID delegates to underlying service
func (cs *CachedService) GetRolesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*Role, error) {
	return cs.service.GetRolesByTenantID(ctx, tenantID)
}

// GetAllSystemLevelRoles delegates to underlying service
func (cs *CachedService) GetAllSystemLevelRoles(ctx context.Context) ([]*Role, error) {
	return cs.service.GetAllSystemLevelRoles(ctx)
}

// AssignRoleToUser delegates to underlying service and invalidates cache
func (cs *CachedService) AssignRoleToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	err := cs.service.AssignRoleToUser(ctx, userID, roleID, assignedBy)
	if err == nil {
		// Invalidate cache for this user since roles changed
		_ = cs.roleCache.Invalidate(ctx, userID)
	}
	return err
}

// RemoveRoleFromUser delegates to underlying service and invalidates cache
func (cs *CachedService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	err := cs.service.RemoveRoleFromUser(ctx, userID, roleID)
	if err == nil {
		// Invalidate cache for this user since roles changed
		_ = cs.roleCache.Invalidate(ctx, userID)
	}
	return err
}

// UpdateRole delegates to underlying service and invalidates all caches
func (cs *CachedService) UpdateRole(ctx context.Context, role *Role) error {
	err := cs.service.UpdateRole(ctx, role)
	if err == nil {
		// Invalidate all caches since role definition changed
		_ = cs.roleCache.InvalidateAll(ctx)
	}
	return err
}

// DeleteRole delegates to underlying service and invalidates all caches
func (cs *CachedService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	err := cs.service.DeleteRole(ctx, id)
	if err == nil {
		// Invalidate all caches since role was deleted
		_ = cs.roleCache.InvalidateAll(ctx)
	}
	return err
}

// Helper method to convert role IDs back to Role objects
func (cs *CachedService) getRolesByIDs(ctx context.Context, roleIDs []uuid.UUID) ([]*Role, error) {
	// Check if we have a batch get method for roles
	if len(roleIDs) == 0 {
		return []*Role{}, nil
	}

	// Try to get roles in batch (optimization over individual calls)
	roles := make([]*Role, 0, len(roleIDs))
	// Create a map to track which roles we've already found
	foundRoles := make(map[uuid.UUID]bool)

	// TODO: Implement a batch get method on the service
	// For now, use individual calls but with optimization potential
	for _, roleID := range roleIDs {
		if foundRoles[roleID] {
			continue // Skip duplicates
		}

		role, err := cs.service.GetRoleByID(ctx, roleID)
		if err != nil {
			// Role not found or error - skip it
			cs.logger.Warn("Failed to get role for cache conversion",
				zap.String("role_id", roleID.String()),
				zap.Error(err))
			continue
		}
		if role != nil {
			roles = append(roles, role)
			foundRoles[roleID] = true
		}
	}
	return roles, nil
}

// GetCacheMetrics returns cache performance metrics
func (cs *CachedService) GetCacheMetrics() (hits, misses, invalidations int64, hitRate float64) {
	hits, misses, invalidations = cs.cacheMetrics.GetStats()
	hitRate = cs.cacheMetrics.HitRate()
	return
}

// GetOperationMetrics returns operation performance metrics
func (cs *CachedService) GetOperationMetrics() (count int64, avgMs float64) {
	count, _, _, _, avgMs = cs.operationMetrics.GetStats()
	return
}
