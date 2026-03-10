package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type ProjectNotificationTriggerHandler struct {
	service        *buildnotification.Service
	projectService *project.Service
	logger         *zap.Logger
}

func NewProjectNotificationTriggerHandler(service *buildnotification.Service, projectService *project.Service, logger *zap.Logger) *ProjectNotificationTriggerHandler {
	return &ProjectNotificationTriggerHandler{service: service, projectService: projectService, logger: logger}
}

type updateProjectTriggerPrefsRequest struct {
	Preferences []projectTriggerPreferencePayload `json:"preferences"`
}

type projectTriggerPreferencePayload struct {
	TriggerID            string   `json:"trigger_id"`
	Enabled              bool     `json:"enabled"`
	Channels             []string `json:"channels"`
	RecipientPolicy      string   `json:"recipient_policy"`
	CustomRecipientUsers []string `json:"custom_recipient_user_ids,omitempty"`
	SeverityOverride     *string  `json:"severity_override,omitempty"`
}

func (h *ProjectNotificationTriggerHandler) GetProjectNotificationTriggers(w http.ResponseWriter, r *http.Request) {
	authCtx, projectID, tenantID, ok := h.resolveProjectContext(w, r)
	if !ok {
		return
	}

	prefs, err := h.service.ListProjectTriggerPreferences(r.Context(), tenantID, projectID)
	if err != nil {
		h.logger.Error("Failed to list project notification trigger preferences", zap.Error(err), zap.String("project_id", projectID.String()), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to load project notification triggers")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":  projectID,
		"preferences": prefs,
	})
}

func (h *ProjectNotificationTriggerHandler) UpdateProjectNotificationTriggers(w http.ResponseWriter, r *http.Request) {
	authCtx, projectID, tenantID, ok := h.resolveProjectContext(w, r)
	if !ok {
		return
	}

	var req updateProjectTriggerPrefsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	prefs := make([]buildnotification.ProjectTriggerPreference, 0, len(req.Preferences))
	for _, item := range req.Preferences {
		pref, err := payloadToDomain(item)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		prefs = append(prefs, pref)
	}

	if err := h.service.UpsertProjectTriggerPreferences(r.Context(), tenantID, projectID, authCtx.UserID, prefs); err != nil {
		h.logger.Error("Failed to upsert project notification trigger preferences", zap.Error(err), zap.String("project_id", projectID.String()), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.service.ListProjectTriggerPreferences(r.Context(), tenantID, projectID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load updated project notification triggers")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":  projectID,
		"preferences": updated,
	})
}

func (h *ProjectNotificationTriggerHandler) DeleteProjectNotificationTrigger(w http.ResponseWriter, r *http.Request) {
	authCtx, projectID, tenantID, ok := h.resolveProjectContext(w, r)
	if !ok {
		return
	}

	triggerID := buildnotification.TriggerID(chi.URLParam(r, "trigger_id"))
	if triggerID == "" {
		h.respondError(w, http.StatusBadRequest, "trigger_id is required")
		return
	}

	if err := h.service.DeleteProjectTriggerPreference(r.Context(), tenantID, projectID, triggerID); err != nil {
		h.logger.Error("Failed to delete project notification trigger preference", zap.Error(err), zap.String("project_id", projectID.String()), zap.String("trigger_id", string(triggerID)), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	prefs, err := h.service.ListProjectTriggerPreferences(r.Context(), tenantID, projectID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load project notification triggers")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":  projectID,
		"preferences": prefs,
	})
}

func (h *ProjectNotificationTriggerHandler) GetTenantNotificationTriggers(w http.ResponseWriter, r *http.Request) {
	authCtx, tenantID, ok := h.resolveTenantContext(w, r)
	if !ok {
		return
	}

	prefs, err := h.service.ListTenantTriggerPreferences(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to list tenant notification trigger preferences", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to load tenant notification triggers")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id":   tenantID,
		"preferences": prefs,
	})
}

func (h *ProjectNotificationTriggerHandler) UpdateTenantNotificationTriggers(w http.ResponseWriter, r *http.Request) {
	authCtx, tenantID, ok := h.resolveTenantContext(w, r)
	if !ok {
		return
	}

	var req updateProjectTriggerPrefsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	prefs := make([]buildnotification.TenantTriggerPreference, 0, len(req.Preferences))
	for _, item := range req.Preferences {
		pref, err := payloadToTenantDomain(item)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		prefs = append(prefs, pref)
	}

	if err := h.service.UpsertTenantTriggerPreferences(r.Context(), tenantID, authCtx.UserID, prefs); err != nil {
		h.logger.Error("Failed to upsert tenant notification trigger preferences", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.service.ListTenantTriggerPreferences(r.Context(), tenantID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to load updated tenant notification triggers")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id":   tenantID,
		"preferences": updated,
	})
}

func (h *ProjectNotificationTriggerHandler) resolveProjectContext(w http.ResponseWriter, r *http.Request) (*middleware.AuthContext, uuid.UUID, uuid.UUID, bool) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return nil, uuid.Nil, uuid.Nil, false
	}

	projectIDStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID")
		return nil, uuid.Nil, uuid.Nil, false
	}

	p, err := h.projectService.GetProject(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			h.respondError(w, http.StatusNotFound, "Project not found")
			return nil, uuid.Nil, uuid.Nil, false
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to resolve project")
		return nil, uuid.Nil, uuid.Nil, false
	}
	if p == nil {
		h.respondError(w, http.StatusNotFound, "Project not found")
		return nil, uuid.Nil, uuid.Nil, false
	}

	tenantID := p.TenantID()
	if authCtx.IsSystemAdmin {
		return authCtx, projectID, tenantID, true
	}
	if authCtx.TenantID != tenantID && !authCtx.HasTenant(tenantID) {
		h.respondError(w, http.StatusForbidden, "Project belongs to a different tenant")
		return nil, uuid.Nil, uuid.Nil, false
	}

	return authCtx, projectID, tenantID, true
}

func payloadToDomain(item projectTriggerPreferencePayload) (buildnotification.ProjectTriggerPreference, error) {
	pref := buildnotification.ProjectTriggerPreference{
		TriggerID:       buildnotification.TriggerID(item.TriggerID),
		Enabled:         item.Enabled,
		RecipientPolicy: buildnotification.RecipientPolicy(item.RecipientPolicy),
	}

	pref.Channels = make([]buildnotification.Channel, 0, len(item.Channels))
	for _, c := range item.Channels {
		pref.Channels = append(pref.Channels, buildnotification.Channel(c))
	}

	pref.CustomRecipientUsers = make([]uuid.UUID, 0, len(item.CustomRecipientUsers))
	for _, userID := range item.CustomRecipientUsers {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return pref, errors.New("invalid custom_recipient_user_ids value")
		}
		pref.CustomRecipientUsers = append(pref.CustomRecipientUsers, parsed)
	}

	if item.SeverityOverride != nil && *item.SeverityOverride != "" {
		severity := buildnotification.Severity(*item.SeverityOverride)
		pref.SeverityOverride = &severity
	}

	return pref, nil
}

func payloadToTenantDomain(item projectTriggerPreferencePayload) (buildnotification.TenantTriggerPreference, error) {
	pref := buildnotification.TenantTriggerPreference{
		TriggerID:       buildnotification.TriggerID(item.TriggerID),
		Enabled:         item.Enabled,
		RecipientPolicy: buildnotification.RecipientPolicy(item.RecipientPolicy),
	}

	pref.Channels = make([]buildnotification.Channel, 0, len(item.Channels))
	for _, c := range item.Channels {
		pref.Channels = append(pref.Channels, buildnotification.Channel(c))
	}

	pref.CustomRecipientUsers = make([]uuid.UUID, 0, len(item.CustomRecipientUsers))
	for _, userID := range item.CustomRecipientUsers {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return pref, errors.New("invalid custom_recipient_user_ids value")
		}
		pref.CustomRecipientUsers = append(pref.CustomRecipientUsers, parsed)
	}

	if item.SeverityOverride != nil && *item.SeverityOverride != "" {
		severity := buildnotification.Severity(*item.SeverityOverride)
		pref.SeverityOverride = &severity
	}

	return pref, nil
}

func (h *ProjectNotificationTriggerHandler) resolveTenantContext(w http.ResponseWriter, r *http.Request) (*middleware.AuthContext, uuid.UUID, bool) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return nil, uuid.Nil, false
	}

	tenantIDStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return nil, uuid.Nil, false
	}

	if authCtx.IsSystemAdmin || authCtx.TenantID == tenantID || authCtx.HasTenant(tenantID) {
		return authCtx, tenantID, true
	}
	h.respondError(w, http.StatusForbidden, "Tenant scope mismatch")
	return nil, uuid.Nil, false
}

func (h *ProjectNotificationTriggerHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *ProjectNotificationTriggerHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
