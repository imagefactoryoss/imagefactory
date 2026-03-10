package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/project"
)

// ProjectAuthMiddleware provides project-level authorization checks
type ProjectAuthMiddleware struct {
	projectService *project.Service
	logger         *zap.Logger
}

// NewProjectAuthMiddleware creates a new project authorization middleware
func NewProjectAuthMiddleware(projectService *project.Service, logger *zap.Logger) *ProjectAuthMiddleware {
	return &ProjectAuthMiddleware{
		projectService: projectService,
		logger:         logger,
	}
}

// ProjectContextKey is the key for storing project ID in request context
const ProjectContextKey = "project_id"

// ValidateProjectAccess is middleware that validates user has access to the requested project
// It extracts the project ID from the URL parameter and checks if the user is a member of that project
func (m *ProjectAuthMiddleware) ValidateProjectAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get auth context from request
		authCtx, ok := GetAuthContext(r)
		if !ok {
			m.logger.Warn("No auth context found in request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract project ID from URL query parameter or path
		// This assumes the project ID is passed as a query parameter or path variable
		projectIDStr := r.URL.Query().Get("project_id")
		if projectIDStr == "" {
			// Try to get from URL path (e.g., /api/projects/{projectID}/builds)
			projectIDStr = r.PathValue("projectID")
			if projectIDStr == "" {
				projectIDStr = r.PathValue("project_id")
			}
		}

		if projectIDStr == "" {
			m.logger.Warn("No project ID provided in request")
			http.Error(w, "Bad Request: missing project ID", http.StatusBadRequest)
			return
		}

		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			m.logger.Warn("Invalid project ID format", zap.String("project_id", projectIDStr), zap.Error(err))
			http.Error(w, "Bad Request: invalid project ID", http.StatusBadRequest)
			return
		}

		// Check if user has access to this project
		hasAccess, err := m.projectService.UserHasProjectAccess(r.Context(), projectID, authCtx.UserID)
		if err != nil {
			m.logger.Error("Failed to check project access", zap.Error(err), zap.String("project_id", projectID.String()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !hasAccess {
			m.logger.Warn("User denied access to project",
				zap.String("user_id", authCtx.UserID.String()),
				zap.String("project_id", projectID.String()))
			http.Error(w, "Forbidden: Access denied to this project", http.StatusForbidden)
			return
		}

		// Add project ID to request context
		ctx := context.WithValue(r.Context(), ProjectContextKey, projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetProjectIDFromContext extracts the project ID from request context
func GetProjectIDFromContext(r *http.Request) (uuid.UUID, error) {
	projectID, ok := r.Context().Value(ProjectContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("project_id not found in request context")
	}
	return projectID, nil
}

// RequireProjectAccess is a helper function that checks if user has project access
// It can be used directly in handlers without middleware wrapping
func (m *ProjectAuthMiddleware) RequireProjectAccess(projectID uuid.UUID, authCtx *AuthContext) bool {
	if authCtx == nil {
		return false
	}

	hasAccess, err := m.projectService.UserHasProjectAccess(context.Background(), projectID, authCtx.UserID)
	if err != nil {
		m.logger.Error("Failed to check project access", zap.Error(err))
		return false
	}

	return hasAccess
}
