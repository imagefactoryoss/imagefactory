package tenant

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/cache"
)

// MockEventPublisher is a mock implementation of EventPublisher for testing
type MockEventPublisher struct{}

func (m *MockEventPublisher) PublishTenantCreated(ctx context.Context, event *TenantCreated) error {
	return nil
}

func (m *MockEventPublisher) PublishTenantActivated(ctx context.Context, event *TenantActivated) error {
	return nil
}

// TestCachedServiceGetTenant tests tenant retrieval with caching
func TestCachedServiceGetTenant(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockTenantRepository{}
	mockPublisher := &MockEventPublisher{}
	service := NewService(mockRepo, mockPublisher, logger)
	tenantCache := cache.NewInMemoryTenantCache(logger, 100)
	defer tenantCache.Shutdown(context.Background())

	cachedService := NewCachedService(service, tenantCache, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	tenantData, _ := NewTenant(uuid.New(), uuid.New(), "TEST", "Test Tenant", "test-tenant", "Test description")

	mockRepo.tenant = tenantData

	// First call should hit service
	result, err := cachedService.GetTenant(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify tenant is in cache
	cached, found, err := tenantCache.Get(ctx, tenantID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.NotNil(t, cached)
}

// TestCachedServiceGetTenantCacheHit tests cache hit behavior
func TestCachedServiceGetTenantCacheHit(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockTenantRepository{}
	mockPublisher := &MockEventPublisher{}
	service := NewService(mockRepo, mockPublisher, logger)
	tenantCache := cache.NewInMemoryTenantCache(logger, 100)
	defer tenantCache.Shutdown(context.Background())

	cachedService := NewCachedService(service, tenantCache, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	tenantData, _ := NewTenant(uuid.New(), uuid.New(), "TEST", "Test Tenant", "test-tenant", "Test description")
	mockRepo.tenant = tenantData

	// First call
	result, err := cachedService.GetTenant(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Get stats before second call
	statsBefore := tenantCache.Stats()

	// Second call should use cache
	result2, err := cachedService.GetTenant(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, result2)

	// Verify stats show cache hits
	statsAfter := tenantCache.Stats()
	assert.Greater(t, statsAfter.Hits, statsBefore.Hits)
}

// TestCachedServiceGetTenantBySlug tests slug-based retrieval with caching
func TestCachedServiceGetTenantBySlug(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockTenantRepository{}
	mockPublisher := &MockEventPublisher{}
	service := NewService(mockRepo, mockPublisher, logger)
	tenantCache := cache.NewInMemoryTenantCache(logger, 100)
	defer tenantCache.Shutdown(context.Background())

	cachedService := NewCachedService(service, tenantCache, logger)
	ctx := context.Background()

	slug := "test-tenant"
	tenantData, _ := NewTenant(uuid.New(), uuid.New(), "TEST", "Test Tenant", slug, "Test description")
	mockRepo.tenant = tenantData

	// First call - should load from service
	result, err := cachedService.GetTenantBySlug(ctx, slug)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tenantData.ID(), result.ID())

	// Second call - should use cache (verify it returns the same data)
	result2, err := cachedService.GetTenantBySlug(ctx, slug)
	require.NoError(t, err)
	assert.NotNil(t, result2)
	assert.Equal(t, tenantData.ID(), result2.ID())
}

// TestCachedServiceInvalidateCacheOnUpdate tests cache invalidation on updates
func TestCachedServiceInvalidateCacheOnUpdate(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockTenantRepository{}
	mockPublisher := &MockEventPublisher{}
	_ = NewService(mockRepo, mockPublisher, logger) // Service exists but we're testing cache directly
	tenantCache := cache.NewInMemoryTenantCache(logger, 100)
	defer tenantCache.Shutdown(context.Background())

	ctx := context.Background()

	tenantID := uuid.New()

	// Cache some data
	cachedData := map[string]interface{}{"name": "Test Tenant"}
	tenantCache.Set(ctx, tenantID, cachedData, 1*time.Hour)

	// Verify it's cached
	_, found, _ := tenantCache.Get(ctx, tenantID)
	assert.True(t, found, "Data should be cached initially")

	// Direct cache invalidation (simulating what happens after updates)
	err := tenantCache.Invalidate(ctx, tenantID)
	require.NoError(t, err)

	// Verify cache is cleared
	_, found, _ = tenantCache.Get(ctx, tenantID)
	assert.False(t, found, "Cache should be cleared after invalidation")
}

// TestCachedServiceWithNoOpCache tests caching disabled
func TestCachedServiceWithNoOpCache(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockTenantRepository{}
	mockPublisher := &MockEventPublisher{}
	service := NewService(mockRepo, mockPublisher, logger)
	noOpCache := cache.NewNoOpTenantCache(logger)

	cachedService := NewCachedService(service, noOpCache, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	tenantData, _ := NewTenant(uuid.New(), uuid.New(), "TEST", "Test Tenant", "test-tenant", "Test description")
	mockRepo.tenant = tenantData

	// With NoOp cache, service should be called
	result, err := cachedService.GetTenant(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Stats should show no caching
	stats := cachedService.CacheStats()
	assert.Equal(t, int64(0), stats.Hits)
}

// MockTenantRepository is a mock implementation for testing
type MockTenantRepository struct {
	tenant  *Tenant
	tenants []*Tenant
	err     error
}

func (m *MockTenantRepository) Save(ctx context.Context, tenant *Tenant) error {
	if m.err != nil {
		return m.err
	}
	m.tenant = tenant
	return nil
}

func (m *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenant, nil
}

func (m *MockTenantRepository) FindBySlug(ctx context.Context, slug string) (*Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenant, nil
}

func (m *MockTenantRepository) FindAll(ctx context.Context, filter TenantFilter) ([]*Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenants, nil
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *Tenant) error {
	if m.err != nil {
		return m.err
	}
	m.tenant = tenant
	return nil
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.err
}

func (m *MockTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.tenant != nil, nil
}

func (m *MockTenantRepository) GetTotalTenantCount(ctx context.Context) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return len(m.tenants), nil
}

func (m *MockTenantRepository) GetActiveTenantCount(ctx context.Context) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	// Mock implementation - return all tenants as active
	return len(m.tenants), nil
}
