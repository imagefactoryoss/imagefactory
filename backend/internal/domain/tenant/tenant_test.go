package tenant

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenant(t *testing.T) {
	companyID := uuid.New()
	tenantCode := "TEST123"
	name := "Test Tenant"
	slug := "test-tenant"

	t.Run("success", func(t *testing.T) {
		tenant, err := NewTenant(uuid.New(), companyID, tenantCode, name, slug, "Test description")

		require.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.Equal(t, companyID, tenant.CompanyID())
		assert.Equal(t, tenantCode, tenant.TenantCode())
		assert.Equal(t, name, tenant.Name())
		assert.Equal(t, slug, tenant.Slug())
		assert.Equal(t, TenantStatusPending, tenant.Status())
		assert.NotEqual(t, uuid.Nil, tenant.ID())
		assert.True(t, tenant.CreatedAt().After(time.Now().Add(-time.Second)))
		assert.True(t, tenant.UpdatedAt().After(time.Now().Add(-time.Second)))
		assert.Equal(t, 1, tenant.Version())
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := NewTenant(uuid.New(), companyID, tenantCode, "", slug, "Test description")
		assert.Equal(t, ErrInvalidTenantName, err)
	})

	t.Run("empty tenant code", func(t *testing.T) {
		_, err := NewTenant(uuid.New(), companyID, "", name, slug, "Test description")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant code cannot be empty")
	})

	t.Run("tenant code too long", func(t *testing.T) {
		longCode := "TOOLONG123"
		_, err := NewTenant(uuid.New(), companyID, longCode, name, slug, "Test description")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant code cannot exceed 8 characters")
	})
}

func TestNewTenantFromExisting(t *testing.T) {
	id := uuid.New()
	companyID := uuid.New()
	tenantCode := "EXIST123"
	name := "Existing Tenant"
	slug := "existing-tenant"
	status := TenantStatusActive
	quota := ResourceQuota{MaxBuilds: 200}
	config := TenantConfig{BuildTimeout: time.Hour}
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()
	version := 5

	t.Run("success", func(t *testing.T) {
		tenant, err := NewTenantFromExisting(id, 123456, companyID, tenantCode, name, slug, "Test description", status, quota, config, createdAt, updatedAt, version)

		require.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.Equal(t, id, tenant.ID())
		assert.Equal(t, companyID, tenant.CompanyID())
		assert.Equal(t, tenantCode, tenant.TenantCode())
		assert.Equal(t, name, tenant.Name())
		assert.Equal(t, slug, tenant.Slug())
		assert.Equal(t, status, tenant.Status())
		assert.Equal(t, quota, tenant.Quota())
		assert.Equal(t, config, tenant.Config())
		assert.Equal(t, createdAt, tenant.CreatedAt())
		assert.Equal(t, updatedAt, tenant.UpdatedAt())
		assert.Equal(t, version, tenant.Version())
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := NewTenantFromExisting(id, 123456, companyID, tenantCode, "", slug, "Test description", status, quota, config, createdAt, updatedAt, version)
		assert.Equal(t, ErrInvalidTenantName, err)
	})

	t.Run("empty tenant code", func(t *testing.T) {
		_, err := NewTenantFromExisting(id, 123456, companyID, "", name, slug, "Test description", status, quota, config, createdAt, updatedAt, version)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant code cannot be empty")
	})
}

func TestTenant_Activate(t *testing.T) {
	companyID := uuid.New()
	tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")

	t.Run("activate pending tenant", func(t *testing.T) {
		err := tenant.Activate()
		assert.NoError(t, err)
		assert.Equal(t, TenantStatusActive, tenant.Status())
		assert.True(t, tenant.UpdatedAt().After(tenant.CreatedAt()))
		assert.Equal(t, 2, tenant.Version())
	})

	t.Run("activate already active tenant", func(t *testing.T) {
		err := tenant.Activate()
		assert.NoError(t, err)
		assert.Equal(t, TenantStatusActive, tenant.Status())
	})

	t.Run("cannot activate deleted tenant", func(t *testing.T) {
		// Create a deleted tenant for testing
		deletedTenant, _ := NewTenantFromExisting(
			uuid.New(), 234567, companyID, "DEL123", "Deleted Tenant", "deleted-tenant", "Test description",
			TenantStatusDeleted, ResourceQuota{}, TenantConfig{},
			time.Now(), time.Now(), 1,
		)

		err := deletedTenant.Activate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot activate deleted tenant")
	})
}

func TestTenant_Suspend(t *testing.T) {
	companyID := uuid.New()
	tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")

	t.Run("suspend active tenant", func(t *testing.T) {
		// First activate it
		_ = tenant.Activate()

		err := tenant.Suspend()
		assert.NoError(t, err)
		assert.Equal(t, TenantStatusSuspended, tenant.Status())
		assert.Equal(t, 3, tenant.Version())
	})

	t.Run("cannot suspend deleted tenant", func(t *testing.T) {
		deletedTenant, _ := NewTenantFromExisting(
			uuid.New(), 345678, companyID, "DEL123", "Deleted Tenant", "deleted-tenant", "Test description",
			TenantStatusDeleted, ResourceQuota{}, TenantConfig{},
			time.Now(), time.Now(), 1,
		)

		err := deletedTenant.Suspend()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot suspend deleted tenant")
	})
}

func TestTenant_UpdateQuota(t *testing.T) {
	companyID := uuid.New()
	tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")

	newQuota := ResourceQuota{
		MaxBuilds:         500,
		MaxImages:         1000,
		MaxStorageGB:      200.0,
		MaxConcurrentJobs: 10,
	}

	tenant.UpdateQuota(newQuota)

	assert.Equal(t, newQuota, tenant.Quota())
	assert.Equal(t, 2, tenant.Version())
}

func TestTenant_UpdateConfig(t *testing.T) {
	companyID := uuid.New()
	tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")

	newConfig := TenantConfig{
		BuildTimeout:         time.Hour,
		AllowedImageTypes:    []string{"container", "vm", "iso"},
		SecurityPolicies:     map[string]interface{}{"scan_images": true},
		NotificationSettings: map[string]interface{}{"email_enabled": true},
	}

	tenant.UpdateConfig(newConfig)

	assert.Equal(t, newConfig, tenant.Config())
	// Timestamps can have coarse resolution on some platforms; ensure UpdatedAt is not earlier than CreatedAt.
	assert.False(t, tenant.UpdatedAt().Before(tenant.CreatedAt()))
	assert.Equal(t, 2, tenant.Version())
}

func TestTenant_IsActive(t *testing.T) {
	companyID := uuid.New()

	t.Run("pending tenant", func(t *testing.T) {
		tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")
		assert.False(t, tenant.IsActive())
	})

	t.Run("active tenant", func(t *testing.T) {
		tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")
		_ = tenant.Activate()
		assert.True(t, tenant.IsActive())
	})

	t.Run("suspended tenant", func(t *testing.T) {
		tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")
		_ = tenant.Activate()
		_ = tenant.Suspend()
		assert.False(t, tenant.IsActive())
	})
}

func TestTenant_CanPerformBuild(t *testing.T) {
	companyID := uuid.New()

	t.Run("active tenant", func(t *testing.T) {
		tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")
		_ = tenant.Activate()
		assert.True(t, tenant.CanPerformBuild())
	})

	t.Run("pending tenant", func(t *testing.T) {
		tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")
		assert.False(t, tenant.CanPerformBuild())
	})

	t.Run("suspended tenant", func(t *testing.T) {
		tenant, _ := NewTenant(uuid.New(), companyID, "TEST123", "Test Tenant", "test-tenant", "Test description")
		_ = tenant.Activate()
		_ = tenant.Suspend()
		assert.False(t, tenant.CanPerformBuild())
	})
}
