package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TenantCache defines interface for tenant caching
type TenantCache interface {
	Get(ctx context.Context, tenantID uuid.UUID) (interface{}, bool, error)
	Set(ctx context.Context, tenantID uuid.UUID, data interface{}, ttl time.Duration) error
	Invalidate(ctx context.Context, tenantID uuid.UUID) error
	InvalidateAll(ctx context.Context) error
	Stats() CacheStats
	Shutdown(ctx context.Context) error
}

// InMemoryTenantCache implements TenantCache using in-memory storage
type InMemoryTenantCache struct {
	mu     sync.RWMutex
	cache  map[string]tenantCacheEntry
	hits   int64
	misses int64
	logger *zap.Logger
	maxSize int
}

// tenantCacheEntry stores tenant data with expiration
type tenantCacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewInMemoryTenantCache creates a new in-memory tenant cache
func NewInMemoryTenantCache(logger *zap.Logger, maxSize int) *InMemoryTenantCache {
	return &InMemoryTenantCache{
		cache:   make(map[string]tenantCacheEntry),
		logger:  logger,
		maxSize: maxSize,
	}
}

// Get retrieves cached tenant data
func (c *InMemoryTenantCache) Get(ctx context.Context, tenantID uuid.UUID) (interface{}, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := tenantID.String()
	entry, exists := c.cache[key]
	
	if !exists {
		c.misses++
		return nil, false, nil
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		// Defer deletion to avoid holding write lock
		go func() {
			c.mu.Lock()
			delete(c.cache, key)
			c.mu.Unlock()
		}()
		c.misses++
		return nil, false, nil
	}

	c.hits++
	return entry.data, true, nil
}

// Set stores tenant data in cache
func (c *InMemoryTenantCache) Set(ctx context.Context, tenantID uuid.UUID, data interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := tenantID.String()
	c.cache[key] = tenantCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Invalidate removes tenant from cache
func (c *InMemoryTenantCache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := tenantID.String()
	delete(c.cache, key)

	return nil
}

// InvalidateAll clears entire cache
func (c *InMemoryTenantCache) InvalidateAll(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]tenantCacheEntry)
	return nil
}

// Stats returns cache statistics
func (c *InMemoryTenantCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:   c.hits,
		Misses: c.misses,
		Size:   len(c.cache),
	}
}

// Shutdown gracefully closes the cache
func (c *InMemoryTenantCache) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]tenantCacheEntry)
	c.logger.Info("Tenant cache shutdown",
		zap.Int64("total_hits", c.hits),
		zap.Int64("total_misses", c.misses))

	return nil
}

// NoOpTenantCache is a cache that doesn't cache (always cache miss)
type NoOpTenantCache struct {
	logger *zap.Logger
}

// NewNoOpTenantCache creates a no-op cache for testing
func NewNoOpTenantCache(logger *zap.Logger) *NoOpTenantCache {
	return &NoOpTenantCache{
		logger: logger,
	}
}

// Get always returns cache miss
func (c *NoOpTenantCache) Get(ctx context.Context, tenantID uuid.UUID) (interface{}, bool, error) {
	return nil, false, nil
}

// Set is a no-op
func (c *NoOpTenantCache) Set(ctx context.Context, tenantID uuid.UUID, data interface{}, ttl time.Duration) error {
	return nil
}

// Invalidate is a no-op
func (c *NoOpTenantCache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

// InvalidateAll is a no-op
func (c *NoOpTenantCache) InvalidateAll(ctx context.Context) error {
	return nil
}

// Stats returns empty stats
func (c *NoOpTenantCache) Stats() CacheStats {
	return CacheStats{}
}

// Shutdown is a no-op
func (c *NoOpTenantCache) Shutdown(ctx context.Context) error {
	return nil
}
