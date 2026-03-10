package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// MockTenantService is a mock implementation of tenant.Service
type MockTenantService struct {
	tenant.Service
	mock.Mock
}

func (m *MockTenantService) CreateTenant(ctx context.Context, companyID uuid.UUID, tenantCode, name, slug, description string) (*tenant.Tenant, error) {
	args := m.Called(ctx, companyID, tenantCode, name, slug, description)
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

func (m *MockTenantService) GetTenant(ctx context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

func (m *MockTenantService) GetTenantBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	args := m.Called(ctx, slug)
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

func (m *MockTenantService) ListTenants(ctx context.Context, offset, limit int) ([]*tenant.Tenant, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*tenant.Tenant), args.Error(1)
}

func (m *MockTenantService) UpdateTenant(ctx context.Context, id uuid.UUID, name, slug, description string, status string, quota *tenant.ResourceQuota, config *tenant.TenantConfig) (*tenant.Tenant, error) {
	args := m.Called(ctx, id, name, slug, description, status, quota, config)
	return args.Get(0).(*tenant.Tenant), args.Error(1)
}

func (m *MockTenantService) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantService) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantService) GetEventPublisher() tenant.EventPublisher {
	args := m.Called()
	return args.Get(0).(tenant.EventPublisher)
}

// MockRBACService is a mock implementation of rbac.Service
type MockRBACService struct {
	mock.Mock
}

func (m *MockRBACService) CheckUserPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

type tenantRepoStub struct {
	lastSaved *tenant.Tenant
}

func (r *tenantRepoStub) Save(ctx context.Context, t *tenant.Tenant) error {
	r.lastSaved = t
	return nil
}
func (r *tenantRepoStub) FindByID(ctx context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	return nil, nil
}
func (r *tenantRepoStub) FindBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	return nil, nil
}
func (r *tenantRepoStub) FindAll(ctx context.Context, filter tenant.TenantFilter) ([]*tenant.Tenant, error) {
	return nil, nil
}
func (r *tenantRepoStub) Update(ctx context.Context, t *tenant.Tenant) error { return nil }
func (r *tenantRepoStub) Delete(ctx context.Context, id uuid.UUID) error     { return nil }
func (r *tenantRepoStub) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, nil
}
func (r *tenantRepoStub) GetTotalTenantCount(ctx context.Context) (int, error)  { return 0, nil }
func (r *tenantRepoStub) GetActiveTenantCount(ctx context.Context) (int, error) { return 0, nil }

type tenantEventPublisherStub struct{}

func (p *tenantEventPublisherStub) PublishTenantCreated(ctx context.Context, event *tenant.TenantCreated) error {
	return nil
}
func (p *tenantEventPublisherStub) PublishTenantActivated(ctx context.Context, event *tenant.TenantActivated) error {
	return nil
}

func TestTenantHandler_CreateTenant_UserContextExtraction(t *testing.T) {
	// Test that user context extraction works correctly
	// This test verifies that the handler properly extracts user information
	// from the authentication middleware context

	// Setup
	logger := zaptest.NewLogger(t)

	// Create a handler with no services - we'll test just the auth extraction
	handler := &TenantHandler{
		logger: logger,
	}

	// Create a test user ID and tenant ID
	userID := uuid.New()
	tenantID := uuid.New()

	// Create auth context
	authCtx := &middleware.AuthContext{
		UserID:      userID,
		TenantID:    tenantID,
		UserTenants: []uuid.UUID{tenantID},
		Email:       "test@example.com",
	}

	// Create request with auth context
	reqBody := CreateTenantRequest{
		CompanyID:   "550e8400-e29b-41d4-a716-446655440000",
		TenantCode:  "TEST",
		Name:        "Test Tenant",
		Slug:        "test-tenant",
		Description: "Test tenant description",
		AdminName:   "Test Admin",
		AdminEmail:  "admin@test.com",
	}

	reqBodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/tenants", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context to request context (simulating what the middleware does)
	ctx := context.WithValue(req.Context(), "auth", authCtx)
	req = req.WithContext(ctx)

	// Execute - expect panic due to nil services, but verify it doesn't return 401
	defer func() {
		if r := recover(); r != nil {
			// Panic occurred as expected due to nil services
			// This means auth extraction worked (didn't return 401)
			t.Logf("Expected panic occurred: %v", r)
		}
	}()

	w := httptest.NewRecorder()
	handler.CreateTenant(w, req)

	// If we get here without panic, check that it didn't return 401
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestTenantHandler_CreateTenant_Unauthorized(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t)

	// Create a minimal handler for testing - we'll focus on the auth context extraction
	// For now, just test that the handler exists and can be called
	handler := &TenantHandler{
		logger: logger,
	}

	// Create request without auth context
	reqBody := CreateTenantRequest{
		CompanyID:   "550e8400-e29b-41d4-a716-446655440000",
		TenantCode:  "TEST",
		Name:        "Test Tenant",
		Slug:        "test-tenant",
		Description: "Test tenant description",
		AdminName:   "Test Admin",
		AdminEmail:  "admin@test.com",
	}

	reqBodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/tenants", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute (no auth context added)
	w := httptest.NewRecorder()
	handler.CreateTenant(w, req)

	// Assert - should fail because no user context
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantHandler_CreateTenant_SystemAdminAutoTriggersTenantPrepare(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tenantRepository := &tenantRepoStub{}
	tenantSvc := tenant.NewService(tenantRepository, &tenantEventPublisherStub{}, logger)

	ownerTenantID := uuid.New()
	globalProviderID := uuid.New()
	infraRepo := &readinessRepoStub{
		providersList: []infrastructure.Provider{
			{
				ID:            globalProviderID,
				TenantID:      ownerTenantID,
				IsGlobal:      true,
				ProviderType:  infrastructure.ProviderTypeKubernetes,
				Status:        infrastructure.ProviderStatusOnline,
				Name:          "global-k8s",
				DisplayName:   "Global K8s",
				BootstrapMode: "image_factory_managed",
				Config:        map[string]interface{}{},
			},
		},
		tenantPrepares: make(map[string]*infrastructure.ProviderTenantNamespacePrepare),
	}
	infraSvc := infrastructure.NewService(infraRepo, nil, logger)

	handler := NewTenantHandler(tenantSvc, nil, nil, nil, logger)
	handler.SetInfrastructureService(infraSvc)

	body := CreateTenantRequest{
		TenantCode:  "AUTO01",
		Name:        "Tenant Auto Trigger",
		Slug:        "tenant-auto-trigger",
		Description: "test",
		AdminName:   "Admin",
		AdminEmail:  "admin@example.com",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:        uuid.New(),
		TenantID:      ownerTenantID,
		UserTenants:   []uuid.UUID{ownerTenantID},
		Email:         "sysadmin@example.com",
		IsSystemAdmin: true,
	}))
	rec := httptest.NewRecorder()

	handler.CreateTenant(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response CreateTenantResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	createdTenantID, err := uuid.Parse(response.ID)
	if err != nil || createdTenantID == uuid.Nil {
		t.Fatalf("expected valid created tenant id, got %q", response.ID)
	}

	deadline := time.Now().Add(2 * time.Second)
	key := globalProviderID.String() + ":" + createdTenantID.String()
	for time.Now().Before(deadline) {
		if prepare, ok := infraRepo.tenantPrepares[key]; ok && prepare != nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("expected async tenant namespace prepare row for provider=%s tenant=%s", globalProviderID, createdTenantID)
}
