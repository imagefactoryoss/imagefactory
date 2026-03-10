package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type RegistryAuthHandler struct {
	service        registryauth.ServiceInterface
	projectService *project.Service
	logger         *zap.Logger
}

func NewRegistryAuthHandler(service registryauth.ServiceInterface, projectService *project.Service, logger *zap.Logger) *RegistryAuthHandler {
	return &RegistryAuthHandler{
		service:        service,
		projectService: projectService,
		logger:         logger,
	}
}

type CreateRegistryAuthRequest struct {
	ProjectID    string                 `json:"project_id,omitempty"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	RegistryType string                 `json:"registry_type"`
	AuthType     registryauth.AuthType  `json:"auth_type"`
	RegistryHost string                 `json:"registry_host"`
	IsDefault    bool                   `json:"is_default"`
	Credentials  map[string]interface{} `json:"credentials"`
}

type UpdateRegistryAuthRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	RegistryType string                 `json:"registry_type"`
	AuthType     registryauth.AuthType  `json:"auth_type"`
	RegistryHost string                 `json:"registry_host"`
	IsDefault    bool                   `json:"is_default"`
	Credentials  map[string]interface{} `json:"credentials,omitempty"`
}

func (h *RegistryAuthHandler) Create(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateRegistryAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var projectID *uuid.UUID
	if req.ProjectID != "" {
		parsed, err := uuid.Parse(req.ProjectID)
		if err != nil {
			http.Error(w, "Invalid project_id", http.StatusBadRequest)
			return
		}
		projectModel, err := h.projectService.GetProject(r.Context(), parsed)
		if err != nil || projectModel == nil {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		if projectModel.TenantID() != authCtx.TenantID {
			http.Error(w, "Project belongs to a different tenant", http.StatusForbidden)
			return
		}
		projectID = &parsed
	}

	created, err := h.service.Create(r.Context(), registryauth.RegistryAuthCreate{
		TenantID:     authCtx.TenantID,
		ProjectID:    projectID,
		Name:         req.Name,
		Description:  req.Description,
		RegistryType: req.RegistryType,
		AuthType:     req.AuthType,
		RegistryHost: req.RegistryHost,
		IsDefault:    req.IsDefault,
	}, req.Credentials, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to create registry auth", zap.Error(err))
		if err == registryauth.ErrDuplicateName {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *RegistryAuthHandler) List(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	includeTenant := r.URL.Query().Get("include_tenant") == "true"

	var auths []*registryauth.RegistryAuth
	var err error
	if projectIDStr != "" {
		projectID, parseErr := uuid.Parse(projectIDStr)
		if parseErr != nil {
			http.Error(w, "Invalid project_id", http.StatusBadRequest)
			return
		}
		projectModel, pErr := h.projectService.GetProject(r.Context(), projectID)
		if pErr != nil || projectModel == nil {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		if projectModel.TenantID() != authCtx.TenantID {
			http.Error(w, "Project belongs to a different tenant", http.StatusForbidden)
			return
		}
		auths, err = h.service.ListByProject(r.Context(), projectID, includeTenant)
	} else {
		auths, err = h.service.ListByTenant(r.Context(), authCtx.TenantID)
	}
	if err != nil {
		h.logger.Error("Failed to list registry auth", zap.Error(err))
		http.Error(w, "Failed to list registry authentication", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"registry_auth": auths,
		"total_count":   len(auths),
	})
}

func (h *RegistryAuthHandler) Delete(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	existing, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == registryauth.ErrRegistryAuthNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load registry authentication", http.StatusInternalServerError)
		return
	}
	if existing.TenantID != authCtx.TenantID {
		http.Error(w, "Registry authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if err == registryauth.ErrRegistryAuthNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete registry authentication", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RegistryAuthHandler) Update(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	existing, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == registryauth.ErrRegistryAuthNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load registry authentication", http.StatusInternalServerError)
		return
	}
	if existing.TenantID != authCtx.TenantID {
		http.Error(w, "Registry authentication belongs to a different tenant", http.StatusForbidden)
		return
	}

	var req UpdateRegistryAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated, err := h.service.Update(r.Context(), id, registryauth.RegistryAuthCreate{
		TenantID:     existing.TenantID,
		ProjectID:    existing.ProjectID,
		Name:         req.Name,
		Description:  req.Description,
		RegistryType: req.RegistryType,
		AuthType:     req.AuthType,
		RegistryHost: req.RegistryHost,
		IsDefault:    req.IsDefault,
	}, req.Credentials)
	if err != nil {
		h.logger.Error("Failed to update registry auth", zap.Error(err))
		if err == registryauth.ErrDuplicateName {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err == registryauth.ErrRegistryAuthNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}
