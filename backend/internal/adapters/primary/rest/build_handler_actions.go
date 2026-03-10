package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	appbuild "github.com/srikarm/image-factory/internal/application/build"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

func (h *BuildHandler) CreateBuild(w http.ResponseWriter, r *http.Request) {
	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized build creation attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create build request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.TenantID == "" {
		h.respondError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	if req.ProjectID == "" {
		h.respondError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id format")
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	h.logger.Info("CreateBuild payload received",
		zap.String("tenant_id", req.TenantID),
		zap.String("project_id", req.ProjectID),
		zap.String("manifest_type", string(req.Manifest.Type)),
		zap.Any("build_config", req.Manifest.BuildConfig),
	)

	// Create build with project scope
	var actorID *uuid.UUID
	if authCtx, ok := middleware.GetAuthContext(r); ok {
		actorID = &authCtx.UserID
	}
	b, err := h.buildAppService.CreateBuild(r.Context(), tenantID, projectID, req.Manifest, actorID)
	if err != nil {
		h.logger.Error("Failed to create build", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("project_id", projectID.String()))

		var preflightErr *buildHTTPError
		if errors.As(err, &preflightErr) {
			h.respondBuildHTTPError(w, preflightErr)
			return
		}

		if errors.Is(err, appbuild.ErrProjectTenantMismatch) {
			h.respondError(w, http.StatusForbidden, "Permission denied: project belongs to a different tenant")
			return
		}

		var validationErr *appbuild.ValidationError
		if errors.As(err, &validationErr) {
			h.respondError(w, http.StatusBadRequest, validationErr.Error())
			return
		}

		h.respondError(w, http.StatusInternalServerError, "Failed to create build")
		return
	}

	// Audit build creation
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), tenantID, authCtx.UserID, audit.AuditEventBuildCreate, "builds", "create",
				"Build created successfully",
				map[string]interface{}{
					"build_id":   b.ID().String(),
					"tenant_id":  tenantID.String(),
					"project_id": projectID.String(),
					"build_name": b.Manifest().Name,
					"build_type": string(b.Manifest().Type),
				})
		}
	}

	response := CreateBuildResponse{
		ID:        b.ID().String(),
		TenantID:  b.TenantID().String(),
		Name:      b.Manifest().Name,
		Type:      string(b.Manifest().Type),
		Status:    string(b.Status()),
		CreatedAt: b.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	h.respondJSON(w, http.StatusCreated, response)
}

func (h *BuildHandler) StartBuild(w http.ResponseWriter, r *http.Request) {
	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized build start attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	err = h.buildService.StartBuild(r.Context(), buildID)
	if err != nil {
		if errors.Is(err, build.ErrBuildNotFound) {
			h.respondError(w, http.StatusNotFound, "Build not found")
			return
		}
		if classified := classifyBuildLifecycleError(err); classified != nil {
			h.respondBuildHTTPError(w, classified)
			return
		}
		h.logger.Error("Failed to start build", zap.Error(err), zap.String("build_id", buildID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to start build")
		return
	}

	// Audit build start
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventBuildStart, "builds", "start",
				"Build started successfully",
				map[string]interface{}{
					"build_id": buildID.String(),
				})
		}
	}

	h.respondJSON(w, http.StatusAccepted, map[string]string{"status": "build_started"})
}

// CancelBuild handles POST /builds/{id}/cancel
func (h *BuildHandler) CancelBuild(w http.ResponseWriter, r *http.Request) {
	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized build cancel attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	err = h.buildService.CancelBuild(r.Context(), buildID)
	if err != nil {
		switch {
		case errors.Is(err, build.ErrBuildNotFound):
			h.respondError(w, http.StatusNotFound, "Build not found")
			return
		case errors.Is(err, build.ErrCannotCancelBuild):
			h.respondError(w, http.StatusConflict, "Cannot cancel build in current state")
			return
		default:
			h.logger.Error("Failed to cancel build", zap.Error(err), zap.String("build_id", buildID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to cancel build")
			return
		}
	}

	// Audit build cancellation
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventBuildCancel, "builds", "cancel",
				"Build cancelled successfully",
				map[string]interface{}{
					"build_id": buildID.String(),
				})
		}
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "build_cancelled"})
}

func (h *BuildHandler) RetryBuild(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	// Retry by restarting the same build ID (new execution attempt is created under build_executions).
	err = h.buildAppService.RetryBuild(r.Context(), buildID)
	if err != nil {
		if errors.Is(err, build.ErrBuildNotFound) {
			h.respondError(w, http.StatusNotFound, "Build not found")
			return
		}
		var notRetriable appbuild.ErrBuildNotRetriable
		if errors.As(err, &notRetriable) {
			h.respondError(w, http.StatusBadRequest, notRetriable.Error())
			return
		}
		var preflightErr *buildHTTPError
		if errors.As(err, &preflightErr) {
			h.respondBuildHTTPError(w, preflightErr)
			return
		}
		if classified := classifyBuildLifecycleError(err); classified != nil {
			h.respondBuildHTTPError(w, classified)
			return
		}
		h.logger.Error("Failed to retry build", zap.Error(err), zap.String("build_id", buildID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to retry build")
		return
	}

	// Audit build retry
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventBuildStart, "builds", "retry",
				"Build retried successfully",
				map[string]interface{}{
					"build_id": buildID.String(),
				})
		}
	}

	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"status":   "build_retrying",
		"build_id": buildID.String(),
	})
}

// CloneBuild handles POST /api/v1/builds/{id}/clone
// It creates a new build using the same manifest and project/tenant scope as the source build.
func (h *BuildHandler) CloneBuild(w http.ResponseWriter, r *http.Request) {
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	sourceBuildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	sourceBuild, err := h.buildService.GetBuild(r.Context(), sourceBuildID)
	if err != nil {
		if errors.Is(err, build.ErrBuildNotFound) {
			h.respondError(w, http.StatusNotFound, "Build not found")
			return
		}
		h.logger.Error("Failed to load source build for clone", zap.Error(err), zap.String("build_id", sourceBuildID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to clone build")
		return
	}
	if sourceBuild == nil {
		h.respondError(w, http.StatusNotFound, "Build not found")
		return
	}

	if sourceBuild.TenantID() != authCtx.TenantID {
		h.respondError(w, http.StatusForbidden, "Build not found in selected tenant context")
		return
	}

	manifest := sourceBuild.Manifest()
	if infraErr := h.validateQuarantineArtifactAdmission(r.Context(), sourceBuild.TenantID(), &manifest); infraErr != nil {
		h.respondBuildHTTPError(w, &buildHTTPError{
			status:  infraErr.status,
			message: infraErr.message,
			code:    infraErr.code,
			details: infraErr.details,
		})
		return
	}

	clonedBuild, err := h.buildService.CreateBuildDraft(r.Context(), sourceBuild.TenantID(), sourceBuild.ProjectID(), manifest, &authCtx.UserID)
	if err != nil {
		var validationErr *appbuild.ValidationError
		if errors.As(err, &validationErr) {
			h.respondError(w, http.StatusBadRequest, validationErr.Error())
			return
		}
		h.logger.Error("Failed to clone build", zap.Error(err), zap.String("source_build_id", sourceBuildID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to clone build")
		return
	}

	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventBuildCreate, "builds", "clone",
			"Build cloned successfully",
			map[string]interface{}{
				"source_build_id": sourceBuildID.String(),
				"new_build_id":    clonedBuild.ID().String(),
			})
	}

	h.respondJSON(w, http.StatusCreated, map[string]string{
		"status":          "build_cloned_draft",
		"source_build_id": sourceBuildID.String(),
		"new_build_id":    clonedBuild.ID().String(),
	})
}

// GetBuildContextSuggestions handles GET /api/v1/projects/{projectId}/build-context-suggestions
// and returns likely build contexts and Dockerfile paths discovered from repository structure.
func (h *BuildHandler) DeleteBuild(w http.ResponseWriter, r *http.Request) {
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	b, err := h.buildService.GetBuild(r.Context(), buildID)
	if err != nil || b == nil {
		h.respondError(w, http.StatusNotFound, "Build not found")
		return
	}

	// Enforce selected tenant context ownership for this delete operation.
	if b.TenantID() != authCtx.TenantID {
		h.respondError(w, http.StatusForbidden, "Build not found in selected tenant context")
		return
	}

	if err := h.buildService.DeleteBuild(r.Context(), buildID); err != nil {
		errMsg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errMsg, "not found"):
			h.respondError(w, http.StatusNotFound, "Build not found")
		case strings.Contains(errMsg, "cannot delete build while execution is running or queued"):
			h.respondError(w, http.StatusBadRequest, "Cannot delete build while execution is running or queued")
		default:
			h.logger.Error("Failed to delete build", zap.Error(err), zap.String("build_id", buildID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to delete build")
		}
		return
	}

	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventBuildCancel, "builds", "delete",
			"Build deleted",
			map[string]interface{}{
				"build_id": buildID.String(),
			},
		)
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":   "deleted",
		"build_id": buildID.String(),
	})
}
