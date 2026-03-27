package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/packertarget"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type PackerTargetProfileHandler struct {
	service *packertarget.Service
	logger  *zap.Logger
}

func NewPackerTargetProfileHandler(service *packertarget.Service, logger *zap.Logger) *PackerTargetProfileHandler {
	return &PackerTargetProfileHandler{service: service, logger: logger}
}

type createPackerTargetProfileRequest struct {
	TenantID    string                 `json:"tenant_id,omitempty"`
	IsGlobal    bool                   `json:"is_global"`
	Name        string                 `json:"name"`
	Provider    string                 `json:"provider"`
	Description string                 `json:"description,omitempty"`
	SecretRef   string                 `json:"secret_ref"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

type updatePackerTargetProfileRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	SecretRef   *string                 `json:"secret_ref,omitempty"`
	Options     *map[string]interface{} `json:"options,omitempty"`
	IsGlobal    *bool                   `json:"is_global,omitempty"`
}

type listPackerTargetProfileResponse struct {
	Profiles []*packertarget.Profile `json:"profiles"`
	Total    int                     `json:"total"`
}

func (h *PackerTargetProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, true)
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	profiles, err := h.service.List(r.Context(), tenantID, allTenants, r.URL.Query().Get("provider"))
	if err != nil {
		h.logger.Error("Failed to list packer target profiles", zap.Error(err))
		h.writeDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, listPackerTargetProfileResponse{Profiles: profiles, Total: len(profiles)})
}

func (h *PackerTargetProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	baseTenantID, _, status, message := resolveTenantScopeFromRequest(r, authCtx, false)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	var req createPackerTargetProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tenantID := baseTenantID
	if strings.TrimSpace(req.TenantID) != "" {
		parsedTenantID, err := uuid.Parse(strings.TrimSpace(req.TenantID))
		if err != nil || parsedTenantID == uuid.Nil {
			http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
			return
		}
		if !authCtx.IsSystemAdmin && parsedTenantID != authCtx.TenantID {
			http.Error(w, "Access denied to this tenant", http.StatusForbidden)
			return
		}
		tenantID = parsedTenantID
	}
	if tenantID == uuid.Nil {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	profile, err := h.service.Create(r.Context(), packertarget.CreateRequest{
		TenantID:    tenantID,
		IsGlobal:    req.IsGlobal,
		Name:        req.Name,
		Provider:    req.Provider,
		Description: req.Description,
		SecretRef:   req.SecretRef,
		Options:     req.Options,
		CreatedBy:   authCtx.UserID,
	})
	if err != nil {
		h.logger.Error("Failed to create packer target profile", zap.Error(err))
		h.writeDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, profile)
}

func (h *PackerTargetProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := h.resolveScopeAndID(w, r)
	if !ok {
		return
	}
	profile, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, profile)
}

func (h *PackerTargetProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := h.resolveScopeAndID(w, r)
	if !ok {
		return
	}
	var req updatePackerTargetProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	profile, err := h.service.Update(r.Context(), tenantID, id, packertarget.UpdateRequest{
		Name:        req.Name,
		Description: req.Description,
		SecretRef:   req.SecretRef,
		Options:     req.Options,
		IsGlobal:    req.IsGlobal,
	})
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, profile)
}

func (h *PackerTargetProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := h.resolveScopeAndID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		h.writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PackerTargetProfileHandler) ValidateProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := h.resolveScopeAndID(w, r)
	if !ok {
		return
	}
	result, err := h.service.Validate(r.Context(), tenantID, id)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *PackerTargetProfileHandler) resolveScopeAndID(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return uuid.Nil, uuid.Nil, false
	}
	tenantID, _, status, message := resolveTenantScopeFromRequest(r, authCtx, false)
	if status != 0 {
		http.Error(w, message, status)
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return uuid.Nil, uuid.Nil, false
	}
	if tenantID == uuid.Nil {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, id, true
}

func (h *PackerTargetProfileHandler) writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, packertarget.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, packertarget.ErrInvalidTenant),
		errors.Is(err, packertarget.ErrInvalidName),
		errors.Is(err, packertarget.ErrInvalidProvider),
		errors.Is(err, packertarget.ErrInvalidSecretRef):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *PackerTargetProfileHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}
