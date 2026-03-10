package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RoleCache defines interface for role caching
type RoleCache interface {
	Get(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error)
	GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
	Set(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, ttl time.Duration) error
	SetBatch(ctx context.Context, roles map[uuid.UUID][]uuid.UUID, ttl time.Duration) error
	Invalidate(ctx context.Context, userID uuid.UUID) error
	InvalidateBatch(ctx context.Context, userIDs []uuid.UUID) error
	InvalidateAll(ctx context.Context) error
	Stats() CacheStats
}

// CacheStats contains cache statistics
type CacheStats struct {
	Hits   int64
	Misses int64
	Size   int
}

// cacheEntry stores role IDs with expiration
type cacheEntry struct {
	roleIDs   []uuid.UUID
	expiresAt time.Time
}

// InMemoryRoleCache implements RoleCache using in-memory storage
type InMemoryRoleCache struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
	hits  int64
	misses int64
	logger *zap.Logger
}

// NewInMemoryRoleCache creates a new in-memory role cache
func NewInMemoryRoleCache(logger *zap.Logger, maxSize int) *InMemoryRoleCache {
	return &InMemoryRoleCache{
		cache:  make(map[string]cacheEntry),
		logger: logger,
	}
}

// Get retrieves cached roles for a user
func (c *InMemoryRoleCache) Get(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := userID.String()
	entry, exists := c.cache[key]

	if !exists {
		c.misses++
		return nil, false, nil
	}

	if time.Now().After(entry.expiresAt) {
		c.misses++
		return nil, false, nil
	}

	c.hits++
	return entry.roleIDs, true, nil
}

// GetBatch retrieves cached roles for multiple users
func (c *InMemoryRoleCache) GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[uuid.UUID][]uuid.UUID)
	now := time.Now()

	for _, userID := range userIDs {
		key := userID.String()
		entry, exists := c.cache[key]

		if exists && now.Before(entry.expiresAt) {
			result[userID] = entry.roleIDs
			c.hits++
		} else {
			c.misses++
		}
	}

	return result, nil
}

// Set stores roles for a user
func (c *InMemoryRoleCache) Set(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := userID.String()
	c.cache[key] = cacheEntry{
		roleIDs:   roleIDs,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// SetBatch stores roles for multiple users
func (c *InMemoryRoleCache) SetBatch(ctx context.Context, roles map[uuid.UUID][]uuid.UUID, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Now().Add(ttl)

	for userID, roleIDs := range roles {
		key := userID.String()
		c.cache[key] = cacheEntry{
			roleIDs:   roleIDs,
			expiresAt: expiresAt,
		}
	}

	return nil
}

// Invalidate removes cached roles for a user
func (c *InMemoryRoleCache) Invalidate(ctx context.Context, userID uuid.UUID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := userID.String()
	delete(c.cache, key)

	return nil
}

// InvalidateBatch removes cached roles for multiple users
func (c *InMemoryRoleCache) InvalidateBatch(ctx context.Context, userIDs []uuid.UUID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, userID := range userIDs {
		key := userID.String()
		delete(c.cache, key)
	}

	return nil
}

// InvalidateAll clears entire role cache
func (c *InMemoryRoleCache) InvalidateAll(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]cacheEntry)

	return nil
}

// Stats returns cache statistics
func (c *InMemoryRoleCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:   c.hits,
		Misses: c.misses,
		Size:   len(c.cache),
	}
}

// Shutdown stops the cache (cleanup goroutines if any)
func (c *InMemoryRoleCache) Shutdown(ctx context.Context) error {
	return nil
}

// NoOpRoleCache implements RoleCache but doesn't cache
type NoOpRoleCache struct {
	logger *zap.Logger
}

// NewNoOpRoleCache creates a cache that doesn't cache anything
func NewNoOpRoleCache(logger *zap.Logger) *NoOpRoleCache {
	return &NoOpRoleCache{logger: logger}
}

// Get always returns cache miss
func (c *NoOpRoleCache) Get(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error) {
	return nil, false, nil
}

// GetBatch always returns empty result
func (c *NoOpRoleCache) GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	return make(map[uuid.UUID][]uuid.UUID), nil
}

// Set does nothing
func (c *NoOpRoleCache) Set(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID, ttl time.Duration) error {
	return nil
}

// SetBatch does nothing
func (c *NoOpRoleCache) SetBatch(ctx context.Context, roles map[uuid.UUID][]uuid.UUID, ttl time.Duration) error {
	return nil
}

// Invalidate does nothing
func (c *NoOpRoleCache) Invalidate(ctx context.Context, userID uuid.UUID) error {
	return nil
}

// InvalidateBatch does nothing
func (c *NoOpRoleCache) InvalidateBatch(ctx context.Context, userIDs []uuid.UUID) error {
	return nil
}

// InvalidateAll does nothing
func (c *NoOpRoleCache) InvalidateAll(ctx context.Context) error {
	return nil
}

// Stats returns empty stats
func (c *NoOpRoleCache) Stats() CacheStats {
	return CacheStats{}
}

// Shutdown does nothing
func (c *NoOpRoleCache) Shutdown(ctx context.Context) error {
	return nil
}
