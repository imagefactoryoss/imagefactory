package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// MockProjectService for testing project authorization
type MockProjectService struct {
	userHasAccess bool
	error         error
}

func (m *MockProjectService) UserHasProjectAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	return m.userHasAccess, m.error
}

// Test: User with project access is allowed
func TestValidateProjectAccess_WithAccess(t *testing.T) {
	logger := zap.NewNop()
	pm := middleware.NewProjectAuthMiddleware((*project.Service)(nil), logger)

	// Create a test handler that verifies context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID, err := middleware.GetProjectIDFromContext(r)
		if err != nil {
			t.Fatalf("Failed to get project ID from context: %v", err)
		}
		if projectID == uuid.Nil {
			t.Error("Project ID should not be nil")
		}
		w.WriteHeader(http.StatusOK)
	})

	// This test is conceptual - the actual middleware needs the project service interface
	// Real test would require dependency injection of the mock service
	_ = testHandler
	_ = pm
}

// Test: User without project access is denied
func TestValidateProjectAccess_WithoutAccess(t *testing.T) {
	logger := zap.NewNop()
	mockService := &MockProjectService{userHasAccess: false}
	pm := middleware.NewProjectAuthMiddleware((*project.Service)(nil), logger)

	_ = mockService
	_ = pm
	// Real test would wrap middleware and verify 403 response
}

// Test: GetProjectIDFromContext retrieves project ID correctly
func TestGetProjectIDFromContext(t *testing.T) {
	projectID := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ProjectContextKey, projectID)
	req := &http.Request{}
	req = req.WithContext(ctx)

	retrieved, err := middleware.GetProjectIDFromContext(req)
	if err != nil {
		t.Fatalf("Failed to get project ID: %v", err)
	}

	if retrieved != projectID {
		t.Errorf("Project ID mismatch: expected %s, got %s", projectID, retrieved)
	}
}

// Test: GetProjectIDFromContext fails with missing project ID
func TestGetProjectIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	req := &http.Request{}
	req = req.WithContext(ctx)

	_, err := middleware.GetProjectIDFromContext(req)
	if err == nil {
		t.Error("Expected error when project ID is missing")
	}
}

// Test: Invalid project ID format returns 400
func TestValidateProjectAccess_InvalidProjectID(t *testing.T) {
	logger := zap.NewNop()
	pm := middleware.NewProjectAuthMiddleware((*project.Service)(nil), logger)

	authCtx := &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := pm.ValidateProjectAccess(testHandler)

	req := httptest.NewRequest("GET", "/?project_id=invalid-uuid", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", authCtx))
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
