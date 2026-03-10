package tenant

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/cache"
	"github.com/srikarm/image-factory/internal/infrastructure/metrics"
)

// CachedService wraps Tenant service with caching layer
type CachedService struct {
	service     *Service
	tenantCache cache.TenantCache
	slugCache   map[string]struct {
		tenantID  uuid.UUID
		expiresAt time.Time
	} // slug -> {tenantID, expiresAt} for quick lookup with TTL
	slugCacheMutex   sync.RWMutex // Synchronization for slugCache
	logger           *zap.Logger
	tenantCacheTTL   time.Duration
	slugCacheTTL     time.Duration
	cacheMetrics     *metrics.CacheMetrics     // Optional metrics tracking
	operationMetrics *metrics.OperationMetrics // Optional operation metrics
}

// NewCachedService creates a cached tenant service
func NewCachedService(
	service *Service,
	tenantCache cache.TenantCache,
	logger *zap.Logger,
) *CachedService {
	return &CachedService{
		service:     service,
		tenantCache: tenantCache,
		slugCache: make(map[string]struct {
			tenantID  uuid.UUID
			expiresAt time.Time
		}),
		logger:           logger,
		tenantCacheTTL:   30 * time.Minute, // Tenants change less frequently than roles
		slugCacheTTL:     30 * time.Minute,
		cacheMetrics:     &metrics.CacheMetrics{}, // Initialize metrics
		operationMetrics: &metrics.OperationMetrics{},
	}
}

// GetTenant retrieves a tenant with caching
func (cs *CachedService) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	m := metrics.NewMeasurement(cs.operationMetrics)
	defer m.Record()

	// Try cache first
	if cached, found, err := cs.tenantCache.Get(ctx, id); err == nil && found {
		if tenant, ok := cached.(*Tenant); ok {
			cs.logger.Debug("Tenant cache hit", zap.String("tenant_id", id.String()))
			cs.cacheMetrics.RecordHit() // Record metric
			return tenant, nil
		}
	}

	cs.cacheMetrics.RecordMiss() // Record metric

	// Cache miss or error - fetch from service
	tenant, err := cs.service.GetTenant(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	// Cache the result
	if err := cs.tenantCache.Set(ctx, id, tenant, cs.tenantCacheTTL); err != nil {
		cs.logger.Warn("Failed to cache tenant", zap.Error(err))
		// Continue - cache failure shouldn't fail the request
	}

	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug with caching
func (cs *CachedService) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	// Try slug cache first for quick ID lookup
	cs.slugCacheMutex.RLock()
	cachedEntry, exists := cs.slugCache[slug]
	cs.slugCacheMutex.RUnlock()

	if exists && time.Now().Before(cachedEntry.expiresAt) {
		// Redirect to GetTenant for caching consistency
		return cs.GetTenant(ctx, cachedEntry.tenantID)
	}

	// Not in slug cache or expired - fetch from service
	tenant, err := cs.service.GetTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	// Cache both the tenant and slug mapping
	tenantID := tenant.ID()
	if err := cs.tenantCache.Set(ctx, tenantID, tenant, cs.tenantCacheTTL); err != nil {
		cs.logger.Warn("Failed to cache tenant", zap.Error(err))
	}

	// Store slug mapping with TTL
	cs.slugCacheMutex.Lock()
	cs.slugCache[slug] = struct {
		tenantID  uuid.UUID
		expiresAt time.Time
	}{
		tenantID:  tenantID,
		expiresAt: time.Now().Add(cs.slugCacheTTL),
	}
	cs.slugCacheMutex.Unlock()

	cs.logger.Debug("Tenant loaded from service",
		zap.String("tenant_id", tenantID.String()),
		zap.String("slug", slug))

	return tenant, nil
}

// CreateTenant creates a new tenant and clears cache
func (cs *CachedService) CreateTenant(ctx context.Context, companyID uuid.UUID, tenantCode, name, slug, description string) (*Tenant, error) {
	tenant, err := cs.service.CreateTenant(ctx, companyID, tenantCode, name, slug, description)
	if err != nil {
		return nil, err
	}

	// Cache new tenant
	if err := cs.tenantCache.Set(ctx, tenant.ID(), tenant, cs.tenantCacheTTL); err != nil {
		cs.logger.Warn("Failed to cache new tenant", zap.Error(err))
	}

	// Cache slug mapping with TTL
	cs.slugCacheMutex.Lock()
	cs.slugCache[slug] = struct {
		tenantID  uuid.UUID
		expiresAt time.Time
	}{
		tenantID:  tenant.ID(),
		expiresAt: time.Now().Add(cs.slugCacheTTL),
	}
	cs.slugCacheMutex.Unlock()

	return tenant, nil
}

// ListTenants retrieves tenants with filter (not cached for now)
func (cs *CachedService) ListTenants(ctx context.Context, filter TenantFilter) ([]*Tenant, error) {
	// List results are not cached as they depend on filters
	// Implement caching if list results become a bottleneck
	return cs.service.ListTenants(ctx, filter)
}

// ActivateTenant activates a tenant and invalidates cache
func (cs *CachedService) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	err := cs.service.ActivateTenant(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache on state change
	if err := cs.tenantCache.Invalidate(ctx, id); err != nil {
		cs.logger.Warn("Failed to invalidate tenant cache", zap.Error(err))
	}

	return nil
}

// SuspendTenant suspends a tenant and invalidates cache
func (cs *CachedService) SuspendTenant(ctx context.Context, id uuid.UUID) error {
	err := cs.service.SuspendTenant(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache on state change
	if err := cs.tenantCache.Invalidate(ctx, id); err != nil {
		cs.logger.Warn("Failed to invalidate tenant cache", zap.Error(err))
	}

	return nil
}

// UpdateTenant updates a tenant and invalidates cache
func (cs *CachedService) UpdateTenantQuota(ctx context.Context, id uuid.UUID, quota ResourceQuota) error {
	err := cs.service.UpdateTenantQuota(ctx, id, quota)
	if err != nil {
		return err
	}

	// Invalidate cache on update
	if err := cs.tenantCache.Invalidate(ctx, id); err != nil {
		cs.logger.Warn("Failed to invalidate tenant cache", zap.Error(err))
	}

	// Clear slug cache on changes
	cs.slugCacheMutex.Lock()
	cs.slugCache = make(map[string]struct {
		tenantID  uuid.UUID
		expiresAt time.Time
	})
	cs.slugCacheMutex.Unlock()

	cs.logger.Debug("Tenant cache invalidated after quota update",
		zap.String("tenant_id", id.String()))

	return nil
}

// UpdateTenant updates tenant properties and invalidates cache
func (cs *CachedService) UpdateTenant(ctx context.Context, id uuid.UUID, name, slug, description string, status string, quota *ResourceQuota, config *TenantConfig) (*Tenant, error) {
	tenant, err := cs.service.UpdateTenant(ctx, id, name, slug, description, status, quota, config)
	if err != nil {
		return nil, err
	}

	// Invalidate cache on update
	if err := cs.tenantCache.Invalidate(ctx, id); err != nil {
		cs.logger.Warn("Failed to invalidate tenant cache", zap.Error(err))
	}

	// Clear slug cache on changes
	cs.slugCacheMutex.Lock()
	cs.slugCache = make(map[string]struct {
		tenantID  uuid.UUID
		expiresAt time.Time
	})
	cs.slugCacheMutex.Unlock()

	cs.logger.Debug("Tenant cache invalidated after tenant update",
		zap.String("tenant_id", id.String()))

	return tenant, nil
}

// CacheStats returns cache statistics
func (cs *CachedService) CacheStats() cache.CacheStats {
	return cs.tenantCache.Stats()
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
