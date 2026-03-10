package tenant_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
)

// MockEventPublisher implements tenant.EventPublisher for testing
type MockEventPublisher struct{}

func (m *MockEventPublisher) PublishTenantCreated(ctx context.Context, event *tenant.TenantCreated) error {
	return nil
}

func (m *MockEventPublisher) PublishTenantActivated(ctx context.Context, event *tenant.TenantActivated) error {
	return nil
}

// MockTenantRepository implements tenant.Repository for testing
type MockTenantRepository struct {
	tenants map[uuid.UUID]*tenant.Tenant
}

func NewMockTenantRepository() *MockTenantRepository {
	return &MockTenantRepository{
		tenants: make(map[uuid.UUID]*tenant.Tenant),
	}
}

func (r *MockTenantRepository) Save(ctx context.Context, t *tenant.Tenant) error {
	r.tenants[t.ID()] = t
	return nil
}

func (r *MockTenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	r.tenants[t.ID()] = t
	return nil
}

func (r *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.tenants, id)
	return nil
}

func (r *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	t, exists := r.tenants[id]
	if !exists {
		return nil, tenant.ErrTenantNotFound
	}
	return t, nil
}

func (r *MockTenantRepository) FindBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	for _, t := range r.tenants {
		if t.Slug() == slug {
			return t, nil
		}
	}
	return nil, tenant.ErrTenantNotFound
}

func (r *MockTenantRepository) FindAll(ctx context.Context, filter tenant.TenantFilter) ([]*tenant.Tenant, error) {
	var result []*tenant.Tenant
	for _, t := range r.tenants {
		if filter.CompanyID != nil && *filter.CompanyID != t.CompanyID() {
			continue
		}
		if filter.Status != nil && *filter.Status != t.Status() {
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

func (r *MockTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	for _, t := range r.tenants {
		if t.Slug() == slug {
			return true, nil
		}
	}
	return false, nil
}

func (r *MockTenantRepository) GetTotalTenantCount(ctx context.Context) (int, error) {
	return len(r.tenants), nil
}

func (r *MockTenantRepository) GetActiveTenantCount(ctx context.Context) (int, error) {
	count := 0
	for _, t := range r.tenants {
		if t.IsActive() {
			count++
		}
	}
	return count, nil
}

// MockUserService implements user.Service interface for testing
type MockUserService struct{}

func (s *MockUserService) CreateUser(ctx context.Context, tenantID uuid.UUID, email, firstName, lastName, password string, roleIDs []uuid.UUID) (*user.User, error) {
	// Mock implementation - in real implementation would hash password, etc.
	return &user.User{}, nil
}

func (s *MockUserService) ValidatePassword(ctx context.Context, email, password string) (*user.User, error) {
	return nil, nil
}

func (s *MockUserService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return "mock-token", nil
}

func (s *MockUserService) ValidateToken(ctx context.Context, token string) (*user.User, error) {
	return nil, nil
}

func (s *MockUserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	return nil
}

// MockRbacService implements rbac.Service interface for testing
type MockRbacService struct{}

func (s *MockRbacService) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return nil
}

func (s *MockRbacService) CheckUserPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	return true, nil
}

// MockSystemConfigRepository implements systemconfig.Repository for testing
type MockSystemConfigRepository struct{}

func (r *MockSystemConfigRepository) Save(ctx context.Context, config *systemconfig.SystemConfig) error {
	return nil
}

func (r *MockSystemConfigRepository) Update(ctx context.Context, config *systemconfig.SystemConfig) error {
	return nil
}

func (r *MockSystemConfigRepository) FindByID(ctx context.Context, id uuid.UUID) (*systemconfig.SystemConfig, error) {
	return &systemconfig.SystemConfig{}, nil
}

func (r *MockSystemConfigRepository) FindByKey(ctx context.Context, key string) (*systemconfig.SystemConfig, error) {
	return &systemconfig.SystemConfig{}, nil
}

func (r *MockSystemConfigRepository) FindByScope(ctx context.Context, scope interface{}) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}

func (r *MockSystemConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestOnboardingService_StartOnboarding(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	ctx := context.Background()
	
	tenantRepo := NewMockTenantRepository()
	eventPublisher := &MockEventPublisher{}
	onboardingService := tenant.NewOnboardingService(tenantRepo, eventPublisher, logger)
	
	// Create test data
	tenantID := uuid.New()
	companyID := uuid.New()
	
	request := tenant.OnboardingRequest{
		TenantID:      tenantID,
		CompanyID:     companyID,
		TenantCode:    "TEST",
		TenantName:    "Test Tenant",
		TenantSlug:    "test-tenant",
		AdminEmail:    "admin@test.com",
		AdminFirstName: "Admin",
		AdminLastName:  "User",
		AdminPassword:  "password123",
		Template:      "standard",
		CustomData:    map[string]interface{}{},
	}
	
	// Test - Start onboarding
	workflow, err := onboardingService.StartOnboarding(ctx, request)
	if err != nil {
		t.Fatalf("StartOnboarding failed: %v", err)
	}
	
	// Validate workflow
	if workflow == nil {
		t.Fatal("Workflow should not be nil")
	}
	
	t.Logf("Workflow started successfully with ID: %s", workflow.ID.String())
}

func TestOnboardingService_GetOnboardingStatus(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	ctx := context.Background()
	
	tenantRepo := NewMockTenantRepository()
	eventPublisher := &MockEventPublisher{}
	onboardingService := tenant.NewOnboardingService(tenantRepo, eventPublisher, logger)
	
	// Create test data
	tenantID := uuid.New()
	companyID := uuid.New()
	
	request := tenant.OnboardingRequest{
		TenantID:      tenantID,
		CompanyID:     companyID,
		TenantCode:    "TEST",
		TenantName:    "Test Tenant",
		TenantSlug:    "test-tenant-status",
		AdminEmail:    "admin@test.com",
		AdminFirstName: "Admin",
		AdminLastName:  "User",
		AdminPassword:  "password123",
		Template:      "standard",
		CustomData:    map[string]interface{}{},
	}
	
	// Start onboarding
	_, err := onboardingService.StartOnboarding(ctx, request)
	if err != nil {
		t.Fatalf("StartOnboarding failed: %v", err)
	}
	
	// Test - Get onboarding status
	retrievedWorkflow, err := onboardingService.GetOnboardingStatus(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOnboardingStatus failed: %v", err)
	}
	
	// Check that we got a workflow with the correct tenant ID
	if retrievedWorkflow.TenantID != tenantID {
		t.Errorf("Expected tenant ID %v, got %v", tenantID, retrievedWorkflow.TenantID)
	}
	if retrievedWorkflow.Status != tenant.StatusCompleted {
		t.Errorf("Expected status %v, got %v", tenant.StatusCompleted, retrievedWorkflow.Status)
	}
}

func TestOnboardingService_ResumeOnboarding(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	ctx := context.Background()
	
	tenantRepo := NewMockTenantRepository()
	eventPublisher := &MockEventPublisher{}
	onboardingService := tenant.NewOnboardingService(tenantRepo, eventPublisher, logger)
	
	// Create test data
	tenantID := uuid.New()
	companyID := uuid.New()
	
	request := tenant.OnboardingRequest{
		TenantID:      tenantID,
		CompanyID:     companyID,
		TenantCode:    "TEST",
		TenantName:    "Test Tenant",
		TenantSlug:    "test-tenant-resume",
		AdminEmail:    "admin@test.com",
		AdminFirstName: "Admin",
		AdminLastName:  "User",
		AdminPassword:  "password123",
		Template:      "standard",
		CustomData:    map[string]interface{}{},
	}
	
	// Start onboarding
	_, err := onboardingService.StartOnboarding(ctx, request)
	if err != nil {
		t.Fatalf("StartOnboarding failed: %v", err)
	}
	
	// Test - Resume onboarding (without completing steps)
	err = onboardingService.ResumeOnboarding(ctx, tenantID)
	if err != nil {
		t.Fatalf("ResumeOnboarding failed: %v", err)
	}
	
	// Get updated status
	updatedWorkflow, err := onboardingService.GetOnboardingStatus(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOnboardingStatus failed: %v", err)
	}
	
	t.Logf("Updated workflow status: %s", updatedWorkflow.Status)
}

func TestOnboardingService_EnterpriseTemplate(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	ctx := context.Background()
	
	tenantRepo := NewMockTenantRepository()
	eventPublisher := &MockEventPublisher{}
	onboardingService := tenant.NewOnboardingService(tenantRepo, eventPublisher, logger)
	
	// Create test data with enterprise template
	tenantID := uuid.New()
	companyID := uuid.New()
	
	request := tenant.OnboardingRequest{
		TenantID:      tenantID,
		CompanyID:     companyID,
		TenantCode:    "ENT",
		TenantName:    "Enterprise Tenant",
		TenantSlug:    "enterprise-tenant",
		AdminEmail:    "admin@enterprise.com",
		AdminFirstName: "Enterprise",
		AdminLastName:  "Admin",
		AdminPassword:  "password123",
		Template:      "enterprise",
		CustomData:    map[string]interface{}{
			"ldap_enabled":     true,
			"sso_enabled":      true,
			"advanced_monitoring": true,
		},
	}
	
	// Test - Start enterprise onboarding
	workflow, err := onboardingService.StartOnboarding(ctx, request)
	if err != nil {
		t.Fatalf("Enterprise StartOnboarding failed: %v", err)
	}
	
	// Verify enterprise-specific steps
	hasLDAPStep := false
	
	for _, step := range workflow.Steps {
		if step.Name == "ldap_configuration" {
			hasLDAPStep = true
			break
		}
	}
	
	if !hasLDAPStep {
		t.Error("Enterprise template should include LDAP configuration step")
	}
	
	t.Logf("Enterprise template includes %d steps", len(workflow.Steps))
}

func TestOnboardingService_ValidationErrors(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	ctx := context.Background()
	
	tenantRepo := NewMockTenantRepository()
	eventPublisher := &MockEventPublisher{}
	onboardingService := tenant.NewOnboardingService(tenantRepo, eventPublisher, logger)
	
	// Test cases with missing required fields
	testCases := []struct {
		name    string
		request tenant.OnboardingRequest
		hasError bool
	}{
		{
			name: "Missing tenant ID",
			request: tenant.OnboardingRequest{
				TenantID:      uuid.Nil,
				CompanyID:     uuid.New(),
				TenantCode:    "TEST",
				TenantName:    "Test Tenant",
				TenantSlug:    "test-tenant",
				AdminEmail:    "admin@test.com",
				AdminFirstName: "Admin",
				AdminLastName:  "User",
				AdminPassword:  "password123",
			},
			hasError: true,
		},
		{
			name: "Missing admin email",
			request: tenant.OnboardingRequest{
				TenantID:      uuid.New(),
				CompanyID:     uuid.New(),
				TenantCode:    "TEST",
				TenantName:    "Test Tenant",
				TenantSlug:    "test-tenant-2",
				AdminEmail:    "",
				AdminFirstName: "Admin",
				AdminLastName:  "User",
				AdminPassword:  "password123",
			},
			hasError: true,
		},
		{
			name: "Valid request",
			request: tenant.OnboardingRequest{
				TenantID:      uuid.New(),
				CompanyID:     uuid.New(),
				TenantCode:    "TEST",
				TenantName:    "Test Tenant",
				TenantSlug:    "test-tenant-3",
				AdminEmail:    "admin@test.com",
				AdminFirstName: "Admin",
				AdminLastName:  "User",
				AdminPassword:  "password123",
			},
			hasError: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := onboardingService.StartOnboarding(ctx, tc.request)
			
			if tc.hasError && err == nil {
				t.Errorf("Expected error for test case: %s", tc.name)
			} else if !tc.hasError && err != nil {
				t.Errorf("Unexpected error for test case %s: %v", tc.name, err)
			}
		})
	}
}