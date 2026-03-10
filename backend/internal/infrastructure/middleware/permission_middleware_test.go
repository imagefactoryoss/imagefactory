package middleware_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// Test: Auth context creation and retrieval
func TestAuthContextCreation(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	email := "test@example.com"

	authCtx := &middleware.AuthContext{
		UserID:         userID,
		TenantID:       tenantID,
		UserTenants:    []uuid.UUID{tenantID},
		Email:          email,
		HasMultiTenant: false,
	}

	if authCtx.UserID != userID {
		t.Errorf("UserID mismatch: expected %s, got %s", userID, authCtx.UserID)
	}

	if authCtx.TenantID != tenantID {
		t.Errorf("TenantID mismatch: expected %s, got %s", tenantID, authCtx.TenantID)
	}

	if authCtx.Email != email {
		t.Errorf("Email mismatch: expected %s, got %s", email, authCtx.Email)
	}
}

// Test: System admin auth context (nil tenant)
func TestSystemAdminAuthContext(t *testing.T) {
	authCtx := &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.Nil,
		Email:    "admin@system.com",
	}

	if authCtx.TenantID != uuid.Nil {
		t.Error("System admin TenantID should be nil")
	}
}

// Test: Tenant user auth context (valid tenant)
func TestTenantUserAuthContext(t *testing.T) {
	tenantID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
		Email:    "user@tenant.com",
	}

	if authCtx.TenantID == uuid.Nil {
		t.Error("Tenant user TenantID should not be nil")
	}

	if authCtx.TenantID != tenantID {
		t.Errorf("TenantID mismatch: expected %s, got %s", tenantID, authCtx.TenantID)
	}
}

// Test: Auth context in request context
func TestGetAuthContext(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()

	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
		Email:    "test@example.com",
	}

	// Create request with auth context
	ctx := context.Background()
	ctx = context.WithValue(ctx, "auth", authCtx)

	// Create a minimal HTTP request with the context
	req := &http.Request{}
	req = req.WithContext(ctx)

	// Retrieve using GetAuthContext function
	retrieved, ok := middleware.GetAuthContext(req)
	if !ok {
		t.Fatal("Failed to retrieve auth context")
	}

	if retrieved.UserID != userID {
		t.Errorf("UserID mismatch: expected %s, got %s", userID, retrieved.UserID)
	}

	if retrieved.TenantID != tenantID {
		t.Errorf("TenantID mismatch: expected %s, got %s", tenantID, retrieved.TenantID)
	}
}

// Test: Missing auth context
func TestGetAuthContextMissing(t *testing.T) {
	ctx := context.Background()
	req := &http.Request{}
	req = req.WithContext(ctx)

	retrieved, ok := middleware.GetAuthContext(req)

	if ok {
		t.Error("Expected GetAuthContext to return false for missing context")
	}

	if retrieved != nil {
		t.Error("Expected retrieved auth context to be nil")
	}
}

// Test: Multi-tenant user attributes
func TestMultiTenantUserContext(t *testing.T) {
	userID := uuid.New()
	userTenantID := uuid.New()
	selectedTenantID := uuid.New()

	authCtx := &middleware.AuthContext{
		UserID:         userID,
		TenantID:       selectedTenantID, // Currently selected tenant
		UserTenants:    []uuid.UUID{userTenantID},     // User's default tenant
		Email:          "multiuser@example.com",
		HasMultiTenant: true, // User has access to multiple tenants
	}

	if !authCtx.HasMultiTenant {
		t.Error("HasMultiTenant should be true")
	}

	if authCtx.TenantID == authCtx.PrimaryTenant() {
		t.Error("Selected tenant should differ from default tenant in this test")
	}

	if authCtx.TenantID == uuid.Nil {
		t.Error("Selected TenantID should not be nil")
	}
}

// Test: Auth context email field
func TestAuthContextEmail(t *testing.T) {
	emails := []string{
		"user@example.com",
		"admin@company.org",
		"system@internal",
	}

	for _, email := range emails {
		authCtx := &middleware.AuthContext{
			UserID: uuid.New(),
			Email:  email,
		}

		if authCtx.Email != email {
			t.Errorf("Email mismatch: expected %s, got %s", email, authCtx.Email)
		}
	}
}

// Test: Auth context with same user and default tenant IDs
func TestAuthContextSameIDs(t *testing.T) {
	id := uuid.New()

	authCtx := &middleware.AuthContext{
		UserID:      id,
		TenantID:    id, // Same as UserID (valid for single-tenant scenario)
		UserTenants: []uuid.UUID{id},
		Email:        "user@example.com",
	}

	if authCtx.UserID != id {
		t.Error("UserID should match")
	}

	if authCtx.TenantID != id {
		t.Error("TenantID should match UserID")
	}

	if authCtx.PrimaryTenant() != id {
		t.Error("PrimaryTenant should match")
	}
}

// FakeRequest wraps a context for testing GetAuthContext
type FakeRequest struct {
	ctx context.Context
}

func (r *FakeRequest) Context() context.Context {
	return r.ctx
}
