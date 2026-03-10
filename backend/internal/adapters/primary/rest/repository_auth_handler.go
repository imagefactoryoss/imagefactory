package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// RepositoryAuthHandler handles repository authentication HTTP requests
type RepositoryAuthHandler struct {
	service        repositoryauth.ServiceInterface
	projectService *project.Service
	logger         *zap.Logger
}

// NewRepositoryAuthHandler creates a new repository authentication handler
func NewRepositoryAuthHandler(service repositoryauth.ServiceInterface, projectService *project.Service, logger *zap.Logger) *RepositoryAuthHandler {
	return &RepositoryAuthHandler{
		service:        service,
		projectService: projectService,
		logger:         logger,
	}
}

type CloneRepositoryAuthRequest struct {
	SourceAuthID string `json:"source_auth_id"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
}

type CreateScopedRepositoryAuthRequest struct {
	ProjectID   string                  `json:"project_id,omitempty"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	AuthType    repositoryauth.AuthType `json:"auth_type"`
	Username    string                  `json:"username,omitempty"`
	SSHKey      string                  `json:"ssh_key,omitempty"`
	Token       string                  `json:"token,omitempty"`
	Password    string                  `json:"password,omitempty"`
}

func (h *RepositoryAuthHandler) requireAuthContext(w http.ResponseWriter, r *http.Request) (*middleware.AuthContext, bool) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("Auth context not found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return authCtx, true
}

func (h *RepositoryAuthHandler) parseAndValidateProjectScope(r *http.Request, authCtx *middleware.AuthContext, projectIDStr string) (*uuid.UUID, bool) {
	if projectIDStr == "" {
		return nil, true
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return nil, false
	}
	projectModel, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil || projectModel == nil {
		return nil, false
	}
	if projectModel.TenantID() != authCtx.TenantID {
		return nil, false
	}
	return &projectID, true
}

func credentialsFromCreate(req CreateScopedRepositoryAuthRequest) map[string]interface{} {
	credentials := make(map[string]interface{})
	switch req.AuthType {
	case repositoryauth.AuthTypeSSH:
		credentials["ssh_key"] = req.SSHKey
	case repositoryauth.AuthTypeToken:
		credentials["token"] = req.Token
	case repositoryauth.AuthTypeBasic:
		credentials["username"] = req.Username
		credentials["password"] = req.Password
	case repositoryauth.AuthTypeOAuth:
		// OAuth credentials would go here
	}
	return credentials
}

// CreateRepositoryAuth creates a project-scoped repository authentication.
func (h *RepositoryAuthHandler) CreateRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	projectIDStr := chi.URLParam(r, "projectId")
	projectID, valid := h.parseAndValidateProjectScope(r, authCtx, projectIDStr)
	if !valid || projectID == nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var req CreateScopedRepositoryAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.ProjectID = projectID.String()
	createReq := repositoryauth.RepositoryAuthCreate{
		TenantID:    authCtx.TenantID,
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		AuthType:    req.AuthType,
		Username:    req.Username,
		SSHKey:      req.SSHKey,
		Token:       req.Token,
		Password:    req.Password,
	}

	if err := createReq.Validate(); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	auth, err := h.service.CreateRepositoryAuth(r.Context(), authCtx.TenantID, projectID, req.Name, req.Description, req.AuthType, credentialsFromCreate(req), authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to create repository auth", zap.Error(err))
		if err == repositoryauth.ErrDuplicateRepositoryAuthName {
			http.Error(w, "Repository authentication name already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(auth)
}

// CreateScopedRepositoryAuth creates tenant/project scoped repository authentication.
func (h *RepositoryAuthHandler) CreateScopedRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	var req CreateScopedRepositoryAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	projectID, valid := h.parseAndValidateProjectScope(r, authCtx, req.ProjectID)
	if !valid {
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	createReq := repositoryauth.RepositoryAuthCreate{
		TenantID:    authCtx.TenantID,
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		AuthType:    req.AuthType,
		Username:    req.Username,
		SSHKey:      req.SSHKey,
		Token:       req.Token,
		Password:    req.Password,
	}
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	auth, err := h.service.CreateRepositoryAuth(r.Context(), authCtx.TenantID, projectID, req.Name, req.Description, req.AuthType, credentialsFromCreate(req), authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to create scoped repository auth", zap.Error(err))
		if err == repositoryauth.ErrDuplicateRepositoryAuthName {
			http.Error(w, "Repository authentication name already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(auth)
}

// GetRepositoryAuths gets all repository authentications for a project
func (h *RepositoryAuthHandler) GetRepositoryAuths(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	projectIDStr := chi.URLParam(r, "projectId")
	projectID, valid := h.parseAndValidateProjectScope(r, authCtx, projectIDStr)
	if !valid || projectID == nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	includeTenant := r.URL.Query().Get("include_tenant") == "true"
	auths, err := h.service.GetRepositoryAuthsForProject(r.Context(), *projectID, includeTenant)
	if err != nil {
		h.logger.Error("Failed to get repository auths", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(auths)
}

// ListScopedRepositoryAuth lists repository auth by scope.
func (h *RepositoryAuthHandler) ListScopedRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	includeTenant := r.URL.Query().Get("include_tenant") == "true"

	if projectIDStr == "" {
		auths, err := h.service.GetRepositoryAuthsByTenant(r.Context(), authCtx.TenantID)
		if err != nil {
			h.logger.Error("Failed to list tenant repository auth", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"repository_auths": auths,
			"total_count":      len(auths),
		})
		return
	}

	projectID, valid := h.parseAndValidateProjectScope(r, authCtx, projectIDStr)
	if !valid || projectID == nil {
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	auths, err := h.service.GetRepositoryAuthsForProject(r.Context(), *projectID, includeTenant)
	if err != nil {
		h.logger.Error("Failed to list project repository auth", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"repository_auths": auths,
		"total_count":      len(auths),
	})
}

// ListAvailableRepositoryAuths lists repository auths available to clone within the tenant.
func (h *RepositoryAuthHandler) ListAvailableRepositoryAuths(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	projectIDStr := chi.URLParam(r, "projectId")
	projectID, valid := h.parseAndValidateProjectScope(r, authCtx, projectIDStr)
	if !valid || projectID == nil {
		h.logger.Error("Invalid project ID", zap.String("projectId", projectIDStr))
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	summaries, err := h.service.GetRepositoryAuthSummariesByTenant(r.Context(), authCtx.TenantID)
	if err != nil {
		h.logger.Error("Failed to list repository auths by tenant", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	filtered := make([]repositoryauth.RepositoryAuthSummary, 0, len(summaries))
	for _, summary := range summaries {
		if summary.ProjectID == *projectID {
			continue
		}
		if !summary.IsActive {
			continue
		}
		filtered = append(filtered, summary)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"repository_auths": filtered,
		"total_count":      len(filtered),
	})
}

// CloneRepositoryAuth clones a repository authentication into the target project.
func (h *RepositoryAuthHandler) CloneRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	projectIDStr := chi.URLParam(r, "projectId")
	projectID, valid := h.parseAndValidateProjectScope(r, authCtx, projectIDStr)
	if !valid || projectID == nil {
		h.logger.Error("Invalid project ID", zap.String("projectId", projectIDStr))
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var req CloneRepositoryAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceAuthID == "" {
		http.Error(w, "source_auth_id is required", http.StatusBadRequest)
		return
	}

	sourceID, err := uuid.Parse(req.SourceAuthID)
	if err != nil {
		http.Error(w, "Invalid source auth ID", http.StatusBadRequest)
		return
	}

	targetProject, err := h.projectService.GetProject(r.Context(), *projectID)
	if err != nil || targetProject == nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	sourceAuth, err := h.service.GetRepositoryAuth(r.Context(), sourceID)
	if err != nil {
		h.logger.Error("Failed to get repository auth", zap.Error(err))
		http.Error(w, "Repository authentication not found", http.StatusNotFound)
		return
	}

	if sourceAuth.GetTenantID() != targetProject.TenantID() {
		http.Error(w, "Repository authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	name := req.Name
	if name == "" {
		name = sourceAuth.GetName() + " (Copy)"
	}

	createdAuth, err := h.service.CloneRepositoryAuth(r.Context(), sourceID, *projectID, name, req.Description, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to clone repository auth", zap.Error(err))
		if err == repositoryauth.ErrDuplicateRepositoryAuthName {
			http.Error(w, "Repository authentication name already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createdAuth)
}

// GetRepositoryAuth gets a specific repository authentication
func (h *RepositoryAuthHandler) GetRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid repository auth ID", zap.Error(err), zap.String("id", idStr))
		http.Error(w, "Invalid repository auth ID", http.StatusBadRequest)
		return
	}

	auth, err := h.service.GetRepositoryAuth(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get repository auth", zap.Error(err))
		if err == repositoryauth.ErrRepositoryAuthNotFound {
			http.Error(w, "Repository authentication not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if auth.GetTenantID() != authCtx.TenantID {
		http.Error(w, "Repository authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(auth)
}

// UpdateRepositoryAuth updates a repository authentication
func (h *RepositoryAuthHandler) UpdateRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid repository auth ID", zap.Error(err), zap.String("id", idStr))
		http.Error(w, "Invalid repository auth ID", http.StatusBadRequest)
		return
	}

	var req repositoryauth.RepositoryAuthUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	auth, err := h.service.GetRepositoryAuth(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get repository auth for update", zap.Error(err))
		if err == repositoryauth.ErrRepositoryAuthNotFound {
			http.Error(w, "Repository authentication not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if auth.GetTenantID() != authCtx.TenantID {
		http.Error(w, "Repository authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	updateName := auth.GetName()
	updateDescription := auth.GetDescription()
	if req.Name != nil {
		updateName = *req.Name
	}
	if req.Description != nil {
		updateDescription = *req.Description
	}

	credentials := make(map[string]interface{})
	if req.AuthType != nil {
		switch *req.AuthType {
		case repositoryauth.AuthTypeSSH:
			if req.SSHKey != nil {
				credentials["ssh_key"] = *req.SSHKey
			}
		case repositoryauth.AuthTypeToken:
			if req.Token != nil {
				credentials["token"] = *req.Token
			}
		case repositoryauth.AuthTypeBasic:
			if req.Username != nil {
				credentials["username"] = *req.Username
			}
			if req.Password != nil {
				credentials["password"] = *req.Password
			}
		case repositoryauth.AuthTypeOAuth:
		}
	}
	if len(credentials) == 0 {
		credentials = nil
	}

	updatedAuth, err := h.service.UpdateRepositoryAuth(r.Context(), id, updateName, updateDescription, credentials)
	if err != nil {
		h.logger.Error("Failed to update repository auth", zap.Error(err))
		if err == repositoryauth.ErrRepositoryAuthNotFound {
			http.Error(w, "Repository authentication not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updatedAuth)
}

// DeleteRepositoryAuth deletes a repository authentication
func (h *RepositoryAuthHandler) DeleteRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid repository auth ID", zap.Error(err), zap.String("id", idStr))
		http.Error(w, "Invalid repository auth ID", http.StatusBadRequest)
		return
	}

	auth, err := h.service.GetRepositoryAuth(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load repository auth", zap.Error(err))
		if err == repositoryauth.ErrRepositoryAuthNotFound {
			http.Error(w, "Repository authentication not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if auth.GetTenantID() != authCtx.TenantID {
		http.Error(w, "Repository authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	err = h.service.DeleteRepositoryAuth(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete repository auth", zap.Error(err))
		var inUseErr *repositoryauth.RepositoryAuthInUseError
		if errors.As(err, &inUseErr) {
			projectNames := make([]string, 0, len(inUseErr.Projects))
			for _, p := range inUseErr.Projects {
				projectNames = append(projectNames, p.ProjectName)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "repository authentication is currently used by active projects",
				"message": fmt.Sprintf("Tenant-scoped repository authentication %q is assigned to active projects. Remove it from those projects first.", inUseErr.AuthName),
				"details": map[string]interface{}{
					"active_projects":      inUseErr.Projects,
					"active_project_names": projectNames,
					"active_project_count": len(inUseErr.Projects),
				},
			})
			return
		}
		if err == repositoryauth.ErrRepositoryAuthNotFound {
			http.Error(w, "Repository authentication not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestRepositoryAuth tests the connection using repository authentication
func (h *RepositoryAuthHandler) TestRepositoryAuth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid repository auth ID", zap.Error(err), zap.String("id", idStr))
		http.Error(w, "Invalid repository auth ID", http.StatusBadRequest)
		return
	}

	auth, err := h.service.GetRepositoryAuth(r.Context(), id)
	if err == nil && auth.GetTenantID() != authCtx.TenantID {
		http.Error(w, "Repository authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	var payload struct {
		RepoURL  string `json:"repo_url"`
		FullTest bool   `json:"full_test"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	result, err := h.service.TestRepositoryAuth(r.Context(), id, repositoryauth.TestOptions{
		FullTest: payload.FullTest,
		RepoURL:  payload.RepoURL,
	})
	if err != nil {
		h.logger.Error("Failed to test repository auth", zap.Error(err))
		if err == repositoryauth.ErrRepositoryAuthNotFound {
			http.Error(w, "Repository authentication not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Repository auth test completed",
		zap.String("auth_id", id.String()),
		zap.Bool("success", result.Success),
		zap.Bool("full_test", payload.FullTest),
		zap.Any("details", result.Details),
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
