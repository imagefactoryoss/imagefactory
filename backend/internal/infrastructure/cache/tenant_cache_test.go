package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestInMemoryTenantCacheBasic tests basic get/set operations
func TestInMemoryTenantCacheBasic(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryTenantCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	tenantID := uuid.New()
	tenantData := map[string]string{
		"name": "TestTenant",
		"code": "TEST001",
	}

	err := cache.Set(ctx, tenantID, tenantData, 1*time.Hour)
	assert.NoError(t, err)

	retrieved, found, err := cache.Get(ctx, tenantID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.NotNil(t, retrieved)
}

// TestInMemoryTenantCacheExpiration tests TTL expiration
func TestInMemoryTenantCacheExpiration(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryTenantCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	tenantID := uuid.New()
	tenantData := "test data"

	// Set with 50ms TTL
	err := cache.Set(ctx, tenantID, tenantData, 50*time.Millisecond)
	assert.NoError(t, err)

	// Should be available immediately
	_, found, err := cache.Get(ctx, tenantID)
	assert.NoError(t, err)
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found, err = cache.Get(ctx, tenantID)
	assert.NoError(t, err)
	assert.False(t, found)
}

// TestInMemoryTenantCacheInvalidation tests cache invalidation
func TestInMemoryTenantCacheInvalidation(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryTenantCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	tenantID := uuid.New()
	tenantData := "test data"

	cache.Set(ctx, tenantID, tenantData, 1*time.Hour)
	_, found, _ := cache.Get(ctx, tenantID)
	assert.True(t, found)

	// Invalidate
	cache.Invalidate(ctx, tenantID)
	_, found, _ = cache.Get(ctx, tenantID)
	assert.False(t, found)
}

// TestInMemoryTenantCacheStats tests statistics tracking
func TestInMemoryTenantCacheStats(t *testing.T) {
	logger := zap.NewNop()
	cache := NewInMemoryTenantCache(logger, 100)
	defer cache.Shutdown(context.Background())

	ctx := context.Background()
	tenantID := uuid.New()
	tenantData := "test data"

	cache.Set(ctx, tenantID, tenantData, 1*time.Hour)

	// Cache hit
	cache.Get(ctx, tenantID)

	// Cache miss (non-existent)
	cache.Get(ctx, uuid.New())

	stats := cache.Stats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 1, stats.Size)
}

// TestNoOpTenantCache tests no-op cache behavior
func TestNoOpTenantCache(t *testing.T) {
	logger := zap.NewNop()
	cache := NewNoOpTenantCache(logger)

	ctx := context.Background()
	tenantID := uuid.New()
	tenantData := "test data"

	// Set should be no-op
	err := cache.Set(ctx, tenantID, tenantData, 1*time.Hour)
	assert.NoError(t, err)

	// Get should always miss
	_, found, err := cache.Get(ctx, tenantID)
	assert.NoError(t, err)
	assert.False(t, found)

	// Stats should be empty
	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
}
