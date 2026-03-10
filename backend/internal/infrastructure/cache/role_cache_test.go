package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestInMemoryRoleCacheBasic tests basic get/set operations
func TestInMemoryRoleCacheBasic(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryRoleCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	userID := uuid.New()
	roleIDs := []uuid.UUID{uuid.New(), uuid.New()}
	ttl := 1 * time.Hour

	err := cache.Set(ctx, userID, roleIDs, ttl)
	assert.NoError(t, err)

	retrieved, found, err := cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, len(roleIDs), len(retrieved))
}

// TestInMemoryRoleCacheBatch tests batch operations
func TestInMemoryRoleCacheBatch(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryRoleCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	ttl := 1 * time.Hour

	users := make(map[uuid.UUID][]uuid.UUID)
	userIDs := make([]uuid.UUID, 5)

	for i := 0; i < 5; i++ {
		userID := uuid.New()
		userIDs[i] = userID
		roleID := uuid.New()
		users[userID] = []uuid.UUID{roleID}
	}

	err := cache.SetBatch(ctx, users, ttl)
	assert.NoError(t, err)

	retrieved, err := cache.GetBatch(ctx, userIDs)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(retrieved))
}

// TestInMemoryRoleCacheExpiration tests TTL and expiration
func TestInMemoryRoleCacheExpiration(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryRoleCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	userID := uuid.New()
	roleIDs := []uuid.UUID{uuid.New()}

	shortTTL := 100 * time.Millisecond
	err := cache.Set(ctx, userID, roleIDs, shortTTL)
	assert.NoError(t, err)

	// Immediately available
	_, found, err := cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Now expired
	_, found, err = cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, found)
}

// TestInMemoryRoleCacheStats tests statistics tracking
func TestInMemoryRoleCacheStats(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryRoleCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	userID := uuid.New()
	roleIDs := []uuid.UUID{uuid.New()}
	ttl := 1 * time.Hour

	cache.Set(ctx, userID, roleIDs, ttl)

	// Cache hits
	cache.Get(ctx, userID)
	cache.Get(ctx, userID)

	// Cache miss
	cache.Get(ctx, uuid.New())

	stats := cache.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 1, stats.Size)
}

// TestInMemoryRoleCacheConcurrency tests concurrent access
func TestInMemoryRoleCacheConcurrency(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryRoleCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	ttl := 1 * time.Hour

	userID := uuid.New()
	roleIDs := []uuid.UUID{uuid.New()}
	cache.Set(ctx, userID, roleIDs, ttl)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			cache.Get(ctx, userID)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := cache.Stats()
	assert.Equal(t, int64(10), stats.Hits)
}

// TestNoOpRoleCache tests no-op cache behavior
func TestNoOpRoleCache(t *testing.T) {
	logger := zap.NewNop()
	cache := NewNoOpRoleCache(logger)

	ctx := context.Background()
	userID := uuid.New()
	roleIDs := []uuid.UUID{uuid.New()}

	_, found, err := cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, found)

	err = cache.Set(ctx, userID, roleIDs, 1*time.Hour)
	assert.NoError(t, err)

	// Still a miss
	_, found, err = cache.Get(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, found)
}
