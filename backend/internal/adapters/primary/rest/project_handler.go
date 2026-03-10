package rest

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	projectService *project.Service
	sourceRepo     *postgres.ProjectSourceRepository
	receiptRepo    *postgres.WebhookReceiptRepository
	buildSettings  *postgres.ProjectBuildSettingsRepository
	systemConfig   *systemconfig.Service
	auditService   *audit.Service
	rbacService    *rbac.Service
	logger         *zap.Logger
}

// checkPermission verifies that the request has the required permission
// Returns the auth context if user is authorized, nil otherwise
func (h *ProjectHandler) checkPermission(r *http.Request) *middleware.AuthContext {
	// Extract auth context (set by auth middleware)
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok {
		h.logger.Debug("no auth context in request")
		return nil
	}

	if authCtx == nil || authCtx.UserID.String() == "" {
		h.logger.Debug("invalid auth context")
		return nil
	}

	return authCtx
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService *project.Service, sourceRepo *postgres.ProjectSourceRepository, receiptRepo *postgres.WebhookReceiptRepository, buildSettings *postgres.ProjectBuildSettingsRepository, auditService *audit.Service, rbacService *rbac.Service, logger *zap.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		sourceRepo:     sourceRepo,
		receiptRepo:    receiptRepo,
		buildSettings:  buildSettings,
		systemConfig:   nil,
		auditService:   auditService,
		rbacService:    rbacService,
		logger:         logger,
	}
}

// NewProjectHandlerWithConfig creates a new project handler with system config access
func NewProjectHandlerWithConfig(projectService *project.Service, sourceRepo *postgres.ProjectSourceRepository, receiptRepo *postgres.WebhookReceiptRepository, buildSettings *postgres.ProjectBuildSettingsRepository, systemConfig *systemconfig.Service, auditService *audit.Service, rbacService *rbac.Service, logger *zap.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		sourceRepo:     sourceRepo,
		receiptRepo:    receiptRepo,
		buildSettings:  buildSettings,
		systemConfig:   systemConfig,
		auditService:   auditService,
		rbacService:    rbacService,
		logger:         logger,
	}
}

// CreateProjectRequest represents the request payload for creating a project
type CreateProjectRequest struct {
	TenantID    string `json:"tenant_id" validate:"required,uuid"`
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Slug        string `json:"slug,omitempty" validate:"omitempty,min=3,max=100"`
	Description string `json:"description,omitempty" validate:"max=1000"`
	GitRepo     string `json:"git_repo,omitempty" validate:"max=500"`
	GitBranch   string `json:"git_branch,omitempty" validate:"max=200"`
	GitProvider string `json:"git_provider,omitempty" validate:"max=100"`
	RepoAuthID  string `json:"repository_auth_id,omitempty" validate:"omitempty,uuid"`
	Visibility  string `json:"visibility,omitempty" validate:"omitempty,oneof=private internal public"`
	IsDraft     bool   `json:"is_draft,omitempty"`
}

// UpdateProjectRequest represents the request payload for updating a project
type UpdateProjectRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Slug        string `json:"slug,omitempty" validate:"omitempty,min=3,max=100"`
	Description string `json:"description,omitempty" validate:"max=1000"`
	GitRepo     string `json:"git_repo,omitempty" validate:"max=500"`
	GitBranch   string `json:"git_branch,omitempty" validate:"max=200"`
	GitProvider string `json:"git_provider,omitempty" validate:"max=100"`
	RepoAuthID  string `json:"repository_auth_id,omitempty" validate:"omitempty,uuid"`
	IsDraft     *bool  `json:"is_draft,omitempty"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Visibility  string `json:"visibility"`
	GitRepo     string `json:"git_repo"`
	GitBranch   string `json:"git_branch"`
	GitProvider string `json:"git_provider"`
	RepoAuthID  string `json:"repository_auth_id,omitempty"`
	IsDraft     bool   `json:"is_draft"`
	BuildCount  int    `json:"build_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProjectSourceRequest struct {
	Name             string `json:"name"`
	Provider         string `json:"provider"`
	RepositoryURL    string `json:"repository_url"`
	DefaultBranch    string `json:"default_branch"`
	RepositoryAuthID string `json:"repository_auth_id,omitempty"`
	IsDefault        bool   `json:"is_default"`
	IsActive         *bool  `json:"is_active,omitempty"`
}

type ProjectSourceResponse struct {
	ID               string `json:"id"`
	ProjectID        string `json:"project_id"`
	TenantID         string `json:"tenant_id"`
	Name             string `json:"name"`
	Provider         string `json:"provider"`
	RepositoryURL    string `json:"repository_url"`
	DefaultBranch    string `json:"default_branch"`
	RepositoryAuthID string `json:"repository_auth_id,omitempty"`
	IsDefault        bool   `json:"is_default"`
	IsActive         bool   `json:"is_active"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type ProjectBuildSettingsRequest struct {
	BuildConfigMode    string `json:"build_config_mode"`
	BuildConfigFile    string `json:"build_config_file,omitempty"`
	BuildConfigOnError string `json:"build_config_on_error,omitempty"`
}

type ProjectBuildSettingsResponse struct {
	ProjectID          string `json:"project_id"`
	BuildConfigMode    string `json:"build_config_mode"`
	BuildConfigFile    string `json:"build_config_file"`
	BuildConfigOnError string `json:"build_config_on_error"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

type WebhookReceiptResponse struct {
	ID                  string   `json:"id"`
	Provider            string   `json:"provider"`
	DeliveryID          string   `json:"delivery_id,omitempty"`
	EventType           string   `json:"event_type"`
	EventRef            string   `json:"event_ref,omitempty"`
	EventBranch         string   `json:"event_branch,omitempty"`
	EventCommitSHA      string   `json:"event_commit_sha,omitempty"`
	RepoURL             string   `json:"repo_url,omitempty"`
	EventSHA            string   `json:"event_sha,omitempty"`
	SignatureValid      bool     `json:"signature_valid"`
	Status              string   `json:"status"`
	Reason              string   `json:"reason,omitempty"`
	MatchedTriggerCount int      `json:"matched_trigger_count"`
	TriggeredBuildIDs   []string `json:"triggered_build_ids"`
	ReceivedAt          string   `json:"received_at"`
}

// ProjectListResponse represents a paginated list of projects
type ProjectListResponse struct {
	Projects   []ProjectResponse `json:"projects"`
	TotalCount int               `json:"total_count"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// CreateProject handles POST /api/v1/projects
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create project request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.TenantID == "" {
		h.respondError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id format")
		return
	}

	var repoAuthID *uuid.UUID
	if req.RepoAuthID != "" {
		parsedID, err := uuid.Parse(req.RepoAuthID)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid repository_auth_id format")
			return
		}
		repoAuthID = &parsedID
	}

	var actorID *uuid.UUID
	if authCtx, ok := middleware.GetAuthContext(r); ok {
		actorID = &authCtx.UserID
	}

	// Create project
	p, err := h.projectService.CreateProject(
		r.Context(),
		tenantID,
		req.Name,
		req.Slug,
		req.Description,
		req.GitRepo,
		req.GitBranch,
		req.Visibility,
		req.GitProvider,
		repoAuthID,
		actorID,
		req.IsDraft,
	)
	if err != nil {
		h.logger.Error("Failed to create project", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		if err == project.ErrDuplicateProjectName {
			h.respondError(w, http.StatusConflict, "Project name already exists in tenant")
			return
		}
		if err == project.ErrDuplicateProjectSlug {
			h.respondError(w, http.StatusConflict, "Project slug already exists in tenant")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to create project")
		return
	}

	response := h.projectToResponse(p)
	h.respondJSON(w, http.StatusCreated, map[string]interface{}{"data": response})
}

// GetProject handles GET /api/v1/projects/{id}
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Project ID is required")
		return
	}

	projectID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}

	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil {
		h.logger.Error("Failed to get project", zap.Error(err), zap.String("project_id", projectID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to get project")
		return
	}

	if p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	response := h.projectToResponse(p)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"data": response})
}

// ListProjects handles GET /api/v1/projects
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	// Get tenant ID from query parameter or auth context
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		// Also check for camelCase variant for frontend compatibility
		tenantIDStr = r.URL.Query().Get("tenantId")
	}
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized project list attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var tenantID uuid.UUID
	if tenantIDStr != "" {
		var err error
		tenantID, err = uuid.Parse(tenantIDStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid tenant_id format")
			return
		}
	} else {
		// Use tenant ID from auth context
		tenantID = authCtx.TenantID
	}

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var viewerID *uuid.UUID
	if authCtx.IsSystemAdmin {
		viewerID = nil
	} else {
		viewerID = &authCtx.UserID
	}

	projects, totalCount, err := h.projectService.ListProjects(r.Context(), tenantID, viewerID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list projects", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to list projects")
		return
	}

	response := ProjectListResponse{
		Projects:   make([]ProjectResponse, len(projects)),
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	for i, p := range projects {
		response.Projects[i] = *h.projectToResponse(p)
	}

	h.respondJSON(w, http.StatusOK, response)
}

// UpdateProject handles PUT /api/v1/projects/{id}
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Project ID is required")
		return
	}

	projectID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update project request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Update project
	var repoAuthID *uuid.UUID
	if req.RepoAuthID != "" {
		parsedID, err := uuid.Parse(req.RepoAuthID)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid repository_auth_id format")
			return
		}
		repoAuthID = &parsedID
	}

	var actorID *uuid.UUID
	if authCtx, ok := middleware.GetAuthContext(r); ok {
		actorID = &authCtx.UserID
	}

	p, err := h.projectService.UpdateProject(
		r.Context(),
		projectID,
		req.Name,
		req.Slug,
		req.Description,
		req.GitRepo,
		req.GitBranch,
		req.GitProvider,
		repoAuthID,
		actorID,
		req.IsDraft,
	)
	if err != nil {
		h.logger.Error("Failed to update project", zap.Error(err), zap.String("project_id", projectID.String()))
		if err == project.ErrDuplicateProjectName {
			h.respondError(w, http.StatusConflict, "Project name already exists in tenant")
			return
		}
		if err == project.ErrDuplicateProjectSlug {
			h.respondError(w, http.StatusConflict, "Project slug already exists in tenant")
			return
		}
		if err == project.ErrProjectNotFound {
			h.respondError(w, http.StatusNotFound, "Project not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to update project")
		return
	}

	response := h.projectToResponse(p)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"data": response})
}

// DeleteProject handles DELETE /api/v1/projects/{id}
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Project ID is required")
		return
	}

	projectID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var actorID *uuid.UUID
	if authCtx.UserID != uuid.Nil {
		actorID = &authCtx.UserID
	}

	hasDeletePermission := false
	draftDelete := false
	var draftProject *project.Project
	if h.rbacService != nil {
		allowed, permErr := h.rbacService.CheckUserPermission(r.Context(), authCtx.UserID, "projects", "delete")
		if permErr != nil {
			h.logger.Error("Failed to check project delete permission", zap.Error(permErr))
			h.respondError(w, http.StatusInternalServerError, "Failed to validate permissions")
			return
		}
		hasDeletePermission = allowed
	}

	if !hasDeletePermission {
		p, getErr := h.projectService.GetProject(r.Context(), projectID)
		if getErr != nil {
			if getErr == project.ErrProjectNotFound {
				h.respondError(w, http.StatusNotFound, "Project not found")
				return
			}
			h.logger.Error("Failed to load project for delete validation", zap.Error(getErr))
			h.respondError(w, http.StatusInternalServerError, "Failed to validate project delete")
			return
		}

		if p.TenantID() != authCtx.TenantID || !p.IsDraft() || p.CreatedBy() == nil || *p.CreatedBy() != authCtx.UserID {
			h.respondError(w, http.StatusForbidden, "You do not have permission to delete this project")
			return
		}
		draftDelete = true
		draftProject = p
	}

	err = h.projectService.DeleteProject(r.Context(), projectID, actorID)
	if err != nil {
		h.logger.Error("Failed to delete project", zap.Error(err), zap.String("project_id", projectID.String()))
		if err == project.ErrProjectNotFound {
			h.respondError(w, http.StatusNotFound, "Project not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to delete project")
		return
	}

	if draftDelete && draftProject != nil && h.auditService != nil {
		source := r.URL.Query().Get("source")
		details := map[string]interface{}{
			"project_id": projectID.String(),
			"draft":      true,
		}
		if source != "" {
			details["source"] = source
		}
		_ = h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventProjectDelete, "projects", "delete", "Draft project discarded", details)
	}

	h.respondJSON(w, http.StatusNoContent, nil)
}

// PurgeDeletedProjects handles POST /api/v1/admin/projects/purge-deleted
func (h *ProjectHandler) PurgeDeletedProjects(w http.ResponseWriter, r *http.Request) {
	if h.systemConfig == nil {
		h.respondError(w, http.StatusInternalServerError, "System config service not configured")
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	retentionDays := 30
	if param := r.URL.Query().Get("retention_days"); param != "" {
		if parsed, err := strconv.Atoi(param); err == nil {
			retentionDays = parsed
		}
	} else {
		config, err := h.systemConfig.GetConfigByTypeAndKey(r.Context(), &authCtx.TenantID, systemconfig.ConfigTypeGeneral, "general")
		if err == nil && config != nil {
			generalConfig, cfgErr := config.GetGeneralConfig()
			if cfgErr == nil && generalConfig.ProjectRetentionDays >= 0 {
				retentionDays = generalConfig.ProjectRetentionDays
			}
		}
	}

	deletedCount, err := h.projectService.PurgeDeletedProjects(r.Context(), retentionDays)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to purge deleted projects")
		return
	}

	// Update last purge metadata (best-effort)
	if retentionDays > 0 {
		config, err := h.systemConfig.GetConfigByTypeAndKey(r.Context(), &authCtx.TenantID, systemconfig.ConfigTypeGeneral, "general")
		if err == nil && config != nil {
			if generalConfig, cfgErr := config.GetGeneralConfig(); cfgErr == nil {
				generalConfig.ProjectLastPurgeAt = time.Now().UTC().Format(time.RFC3339)
				generalConfig.ProjectLastPurgeCount = deletedCount
				_, _ = h.systemConfig.CreateOrUpdateCategoryConfig(
					r.Context(),
					&authCtx.TenantID,
					systemconfig.ConfigTypeGeneral,
					"general",
					generalConfig,
					authCtx.UserID,
				)
			}
		}
	}

	if h.auditService != nil {
		_ = h.auditService.LogUserAction(
			r.Context(),
			authCtx.TenantID,
			authCtx.UserID,
			audit.AuditEventProjectPurge,
			"projects",
			"purge_deleted",
			"Purged deleted projects",
			map[string]interface{}{
				"retention_days": retentionDays,
				"deleted_count":  deletedCount,
			},
		)
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"deleted_count":  deletedCount,
		"retention_days": retentionDays,
	})
}

// ============================================================================
// PROJECT MEMBER MANAGEMENT
// ============================================================================

// AddMemberRequest represents the request payload for adding a member to a project
type AddMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// MemberResponse represents a project member in API responses
type MemberResponse struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"project_id"`
	UserID           string  `json:"user_id"`
	RoleID           *string `json:"role_id,omitempty"`
	AssignedByUserID *string `json:"assigned_by_user_id,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// MembersListResponse represents a paginated list of project members
type MembersListResponse struct {
	Members    []MemberResponse `json:"members"`
	TotalCount int              `json:"total_count"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
}

// UpdateMemberRoleRequest represents the request payload for updating a member's role
type UpdateMemberRoleRequest struct {
	RoleID *string `json:"role_id"` // null to clear override
}

// AddMember handles POST /api/v1/projects/{projectID}/members
func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	if projectIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "project_id is required in path")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode add member request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	// Get auth context for audit trail
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Add member
	member, err := h.projectService.AddMember(r.Context(), projectID, userID, &authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to add project member", zap.Error(err), zap.String("project_id", projectID.String()))

		// Check for specific error types
		if err == project.ErrMemberAlreadyExists {
			h.respondError(w, http.StatusConflict, "User is already a member of this project")
			return
		}
		if err == project.ErrProjectNotFound {
			h.respondError(w, http.StatusNotFound, "Project not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "Failed to add member")
		return
	}

	// Audit
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventMemberAdded, "projects", "add_member",
			"Member added to project",
			map[string]interface{}{
				"project_id": projectID.String(),
				"user_id":    userID.String(),
				"member_id":  member.ID().String(),
			})
	}

	h.respondJSON(w, http.StatusCreated, h.memberToResponse(member))
}

// RemoveMember handles DELETE /api/v1/projects/{projectID}/members/{userID}
func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	userIDStr := chi.URLParam(r, "userID")

	if projectIDStr == "" || userIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "project_id and user_id are required in path")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Remove member
	err = h.projectService.RemoveMember(r.Context(), projectID, userID)
	if err != nil {
		h.logger.Error("Failed to remove project member", zap.Error(err), zap.String("project_id", projectID.String()))

		if err == project.ErrMemberNotFound {
			h.respondError(w, http.StatusNotFound, "Member not found")
			return
		}
		if err == project.ErrProjectNotFound {
			h.respondError(w, http.StatusNotFound, "Project not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "Failed to remove member")
		return
	}

	// Audit
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventMemberRemoved, "projects", "remove_member",
			"Member removed from project",
			map[string]interface{}{
				"project_id": projectID.String(),
				"user_id":    userID.String(),
			})
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/projects/{projectID}/members
func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	if projectIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "project_id is required in path")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	// Parse pagination
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// List members
	members, totalCount, err := h.projectService.ListMembers(r.Context(), projectID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list project members", zap.Error(err), zap.String("project_id", projectID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to list members")
		return
	}

	// Convert to response
	memberResponses := make([]MemberResponse, len(members))
	for i, m := range members {
		memberResponses[i] = h.memberToResponse(m)
	}

	response := MembersListResponse{
		Members:    memberResponses,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// UpdateMemberRole handles PATCH /api/v1/projects/{projectID}/members/{userID}
func (h *ProjectHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	userIDStr := chi.URLParam(r, "userID")

	if projectIDStr == "" || userIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "project_id and user_id are required in path")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update member role request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse role ID if provided
	var roleID *uuid.UUID
	if req.RoleID != nil {
		parsed, err := uuid.Parse(*req.RoleID)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid role_id format")
			return
		}
		roleID = &parsed
	}

	// Update member role
	member, err := h.projectService.UpdateMemberRole(r.Context(), projectID, userID, roleID)
	if err != nil {
		h.logger.Error("Failed to update project member role", zap.Error(err), zap.String("project_id", projectID.String()))

		if err == project.ErrMemberNotFound {
			h.respondError(w, http.StatusNotFound, "Member not found")
			return
		}
		if err == project.ErrProjectNotFound {
			h.respondError(w, http.StatusNotFound, "Project not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "Failed to update member")
		return
	}

	// Audit
	if h.auditService != nil {
		roleIDStr := "cleared"
		if roleID != nil {
			roleIDStr = roleID.String()
		}

		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventMemberUpdated, "projects", "update_member_role",
			"Member role updated",
			map[string]interface{}{
				"project_id": projectID.String(),
				"user_id":    userID.String(),
				"role_id":    roleIDStr,
			})
	}

	h.respondJSON(w, http.StatusOK, h.memberToResponse(member))
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// memberToResponse converts a domain member to a response DTO
func (h *ProjectHandler) memberToResponse(m *project.Member) MemberResponse {
	response := MemberResponse{
		ID:        m.ID().String(),
		ProjectID: m.ProjectID().String(),
		UserID:    m.UserID().String(),
		CreatedAt: m.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if roleID := m.RoleID(); roleID != nil {
		roleIDStr := roleID.String()
		response.RoleID = &roleIDStr
	}

	if assignedByUserID := m.AssignedByUserID(); assignedByUserID != nil {
		assignedByUserIDStr := assignedByUserID.String()
		response.AssignedByUserID = &assignedByUserIDStr
	}

	return response
}

// projectToResponse converts a project domain object to API response
func (h *ProjectHandler) projectToResponse(p *project.Project) *ProjectResponse {
	response := &ProjectResponse{
		ID:          p.ID().String(),
		TenantID:    p.TenantID().String(),
		Name:        p.Name(),
		Slug:        p.Slug(),
		Description: p.Description(),
		Status:      string(p.Status()),
		Visibility:  string(p.Visibility()),
		GitRepo:     p.GitRepo(),
		GitBranch:   p.GitBranch(),
		GitProvider: p.GitProvider(),
		IsDraft:     p.IsDraft(),
		BuildCount:  p.BuildCount(),
		CreatedAt:   p.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if repoAuthID := p.RepositoryAuthID(); repoAuthID != nil {
		response.RepoAuthID = repoAuthID.String()
	}

	return response
}

// respondJSON sends a JSON response
func (h *ProjectHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *ProjectHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func (h *ProjectHandler) GetProjectBuildSettings(w http.ResponseWriter, r *http.Request) {
	if h.buildSettings == nil {
		h.respondError(w, http.StatusNotImplemented, "project build settings repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil || p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	settings, err := h.buildSettings.GetByProjectID(r.Context(), projectID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load project build settings")
		return
	}
	if settings == nil {
		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"data": ProjectBuildSettingsResponse{
				ProjectID:          projectID.String(),
				BuildConfigMode:    postgres.ProjectBuildConfigModeRepoManaged,
				BuildConfigFile:    "image-factory.yaml",
				BuildConfigOnError: postgres.ProjectBuildConfigOnErrorStrict,
			},
		})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": ProjectBuildSettingsResponse{
			ProjectID:          settings.ProjectID.String(),
			BuildConfigMode:    settings.BuildConfigMode,
			BuildConfigFile:    settings.BuildConfigFile,
			BuildConfigOnError: settings.BuildConfigOnError,
			UpdatedAt:          settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

func (h *ProjectHandler) UpdateProjectBuildSettings(w http.ResponseWriter, r *http.Request) {
	if h.buildSettings == nil {
		h.respondError(w, http.StatusNotImplemented, "project build settings repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil || p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	var req ProjectBuildSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.BuildConfigMode))
	switch mode {
	case "", postgres.ProjectBuildConfigModeRepoManaged:
		mode = postgres.ProjectBuildConfigModeRepoManaged
	case postgres.ProjectBuildConfigModeUIManaged:
	default:
		h.respondError(w, http.StatusBadRequest, "Invalid build_config_mode. Supported values: ui_managed, repo_managed")
		return
	}

	file := strings.TrimSpace(req.BuildConfigFile)
	if file == "" {
		file = "image-factory.yaml"
	}
	onError := strings.ToLower(strings.TrimSpace(req.BuildConfigOnError))
	switch onError {
	case "", postgres.ProjectBuildConfigOnErrorStrict:
		onError = postgres.ProjectBuildConfigOnErrorStrict
	case postgres.ProjectBuildConfigOnErrorFallback:
	default:
		h.respondError(w, http.StatusBadRequest, "Invalid build_config_on_error. Supported values: strict, fallback_to_ui")
		return
	}

	settings, err := h.buildSettings.Upsert(r.Context(), postgres.ProjectBuildSettings{
		ProjectID:          projectID,
		BuildConfigMode:    mode,
		BuildConfigFile:    file,
		BuildConfigOnError: onError,
	})
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to save project build settings")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": ProjectBuildSettingsResponse{
			ProjectID:          settings.ProjectID.String(),
			BuildConfigMode:    settings.BuildConfigMode,
			BuildConfigFile:    settings.BuildConfigFile,
			BuildConfigOnError: settings.BuildConfigOnError,
			UpdatedAt:          settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

func (h *ProjectHandler) ListProjectSources(w http.ResponseWriter, r *http.Request) {
	if h.sourceRepo == nil {
		h.respondError(w, http.StatusNotImplemented, "project source repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil || p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	sources, err := h.sourceRepo.ListByProjectID(r.Context(), projectID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to list project sources")
		return
	}
	out := make([]ProjectSourceResponse, 0, len(sources))
	for _, s := range sources {
		out = append(out, h.projectSourceToResponse(s))
	}
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"data": out})
}

func (h *ProjectHandler) CreateProjectSource(w http.ResponseWriter, r *http.Request) {
	if h.sourceRepo == nil {
		h.respondError(w, http.StatusNotImplemented, "project source repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil || p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	var req ProjectSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	repoAuth, err := parseOptionalUUID(req.RepositoryAuthID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid repository_auth_id format")
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	source := &postgres.ProjectSource{
		ID:             uuid.New(),
		Name:           strings.TrimSpace(req.Name),
		ProjectID:      projectID,
		TenantID:       p.TenantID(),
		Provider:       strings.TrimSpace(req.Provider),
		RepositoryURL:  strings.TrimSpace(req.RepositoryURL),
		DefaultBranch:  strings.TrimSpace(req.DefaultBranch),
		RepositoryAuth: repoAuth,
		IsDefault:      req.IsDefault,
		IsActive:       isActive,
	}
	if err := h.sourceRepo.Create(r.Context(), source); err != nil {
		if errors.Is(err, postgres.ErrDuplicateProjectSource) {
			h.respondError(w, http.StatusConflict, "A source with the same provider, repository, and default branch already exists for this project")
			return
		}
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.sourceRepo.FindByID(r.Context(), projectID, source.ID)
	if err != nil || created == nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load created source")
		return
	}
	h.respondJSON(w, http.StatusCreated, map[string]interface{}{"data": h.projectSourceToResponse(*created)})
}

func (h *ProjectHandler) UpdateProjectSource(w http.ResponseWriter, r *http.Request) {
	if h.sourceRepo == nil {
		h.respondError(w, http.StatusNotImplemented, "project source repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid source ID format")
		return
	}
	current, err := h.sourceRepo.FindByID(r.Context(), projectID, sourceID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load source")
		return
	}
	if current == nil {
		h.respondError(w, http.StatusNotFound, "Project source not found")
		return
	}
	var req ProjectSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	repoAuth, err := parseOptionalUUID(req.RepositoryAuthID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid repository_auth_id format")
		return
	}
	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	current.Name = strings.TrimSpace(req.Name)
	current.Provider = strings.TrimSpace(req.Provider)
	current.RepositoryURL = strings.TrimSpace(req.RepositoryURL)
	current.DefaultBranch = strings.TrimSpace(req.DefaultBranch)
	current.RepositoryAuth = repoAuth
	current.IsDefault = req.IsDefault
	current.IsActive = isActive

	if err := h.sourceRepo.Update(r.Context(), current); err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Project source not found")
			return
		}
		if errors.Is(err, postgres.ErrDuplicateProjectSource) {
			h.respondError(w, http.StatusConflict, "A source with the same provider, repository, and default branch already exists for this project")
			return
		}
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.sourceRepo.FindByID(r.Context(), projectID, sourceID)
	if err != nil || updated == nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load updated source")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"data": h.projectSourceToResponse(*updated)})
}

func (h *ProjectHandler) DeleteProjectSource(w http.ResponseWriter, r *http.Request) {
	if h.sourceRepo == nil {
		h.respondError(w, http.StatusNotImplemented, "project source repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid source ID format")
		return
	}
	if err := h.sourceRepo.Delete(r.Context(), projectID, sourceID); err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Project source not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to delete project source")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
}

func (h *ProjectHandler) ListProjectWebhookReceipts(w http.ResponseWriter, r *http.Request) {
	if h.receiptRepo == nil {
		h.respondError(w, http.StatusNotImplemented, "webhook receipt repository not configured")
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil || p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	_ = p
	limit := 50
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed >= 0 {
			offset = parsed
		}
	}

	receipts, err := h.receiptRepo.ListByProject(r.Context(), projectID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to list webhook receipts")
		return
	}
	out := make([]WebhookReceiptResponse, 0, len(receipts))
	for _, receipt := range receipts {
		out = append(out, webhookReceiptToResponse(receipt))
	}
	h.respondJSON(w, http.StatusOK, map[string]interface{}{"data": out})
}

func (h *ProjectHandler) projectSourceToResponse(source postgres.ProjectSource) ProjectSourceResponse {
	resp := ProjectSourceResponse{
		ID:            source.ID.String(),
		ProjectID:     source.ProjectID.String(),
		TenantID:      source.TenantID.String(),
		Name:          source.Name,
		Provider:      source.Provider,
		RepositoryURL: source.RepositoryURL,
		DefaultBranch: source.DefaultBranch,
		IsDefault:     source.IsDefault,
		IsActive:      source.IsActive,
		CreatedAt:     source.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     source.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if source.RepositoryAuth != nil {
		resp.RepositoryAuthID = source.RepositoryAuth.String()
	}
	return resp
}

func webhookReceiptToResponse(receipt postgres.WebhookReceipt) WebhookReceiptResponse {
	resp := WebhookReceiptResponse{
		ID:                  receipt.ID.String(),
		Provider:            receipt.Provider,
		EventType:           receipt.EventType,
		SignatureValid:      receipt.SignatureValid,
		Status:              receipt.Status,
		MatchedTriggerCount: receipt.MatchedTriggerCount,
		TriggeredBuildIDs:   receipt.TriggeredBuildIDs,
		ReceivedAt:          receipt.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if receipt.DeliveryID != nil {
		resp.DeliveryID = *receipt.DeliveryID
	}
	if receipt.EventRef != nil {
		resp.EventRef = *receipt.EventRef
	}
	if receipt.EventBranch != nil {
		resp.EventBranch = *receipt.EventBranch
	}
	if receipt.EventCommitSHA != nil {
		resp.EventCommitSHA = *receipt.EventCommitSHA
	}
	if receipt.RepoURL != nil {
		resp.RepoURL = *receipt.RepoURL
	}
	if receipt.EventSHA != nil {
		resp.EventSHA = *receipt.EventSHA
	}
	if receipt.Reason != nil {
		resp.Reason = *receipt.Reason
	}
	return resp
}

func parseOptionalUUID(value string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
