package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// BuildTriggerHandler handles HTTP requests for build triggers
type BuildTriggerHandler struct {
	service *build.Service
	logger  *zap.Logger
}

// NewBuildTriggerHandler creates a new build trigger handler
func NewBuildTriggerHandler(service *build.Service, logger *zap.Logger) *BuildTriggerHandler {
	return &BuildTriggerHandler{
		service: service,
		logger:  logger,
	}
}

// Request DTOs

// CreateWebhookTriggerRequest represents a webhook trigger creation request
type CreateWebhookTriggerRequest struct {
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description,omitempty"`
	WebhookURL    string   `json:"webhook_url" validate:"required,url"`
	WebhookSecret string   `json:"webhook_secret,omitempty"`
	WebhookEvents []string `json:"webhook_events,omitempty"`
}

// CreateProjectWebhookTriggerRequest represents a project-level webhook trigger creation request.
// build_id is optional; when omitted the backend resolves the latest build in the project.
type CreateProjectWebhookTriggerRequest struct {
	BuildID       string   `json:"build_id,omitempty"`
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description,omitempty"`
	WebhookURL    string   `json:"webhook_url" validate:"required,url"`
	WebhookSecret string   `json:"webhook_secret,omitempty"`
	WebhookEvents []string `json:"webhook_events,omitempty"`
}

// CreateScheduleTriggerRequest represents a scheduled trigger creation request
type CreateScheduleTriggerRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
	CronExpr    string `json:"cron_expression" validate:"required"`
	Timezone    string `json:"timezone,omitempty"`
}

// CreateGitEventTriggerRequest represents a Git event trigger creation request
type CreateGitEventTriggerRequest struct {
	Name             string `json:"name" validate:"required"`
	Description      string `json:"description,omitempty"`
	GitProvider      string `json:"git_provider" validate:"required,oneof=github gitlab gitea bitbucket"`
	GitRepoURL       string `json:"git_repository_url" validate:"required,url"`
	GitBranchPattern string `json:"git_branch_pattern,omitempty"`
}

// UpdateProjectWebhookTriggerRequest represents partial update payload for project webhook triggers.
type UpdateProjectWebhookTriggerRequest struct {
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	WebhookURL    *string  `json:"webhook_url,omitempty"`
	WebhookSecret *string  `json:"webhook_secret,omitempty"`
	WebhookEvents []string `json:"webhook_events,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
}

// Response DTOs

// BuildTriggerResponse represents a trigger in API responses
type BuildTriggerResponse struct {
	ID                 uuid.UUID `json:"id"`
	BuildID            uuid.UUID `json:"build_id"`
	ProjectID          uuid.UUID `json:"project_id"`
	TriggerType        string    `json:"trigger_type"`
	TriggerName        string    `json:"name"`
	TriggerDescription string    `json:"description,omitempty"`

	// Webhook fields
	WebhookURL    *string  `json:"webhook_url,omitempty"`
	WebhookSecret *string  `json:"webhook_secret,omitempty"`
	WebhookEvents []string `json:"webhook_events,omitempty"`

	// Schedule fields
	CronExpression  *string `json:"cron_expression,omitempty"`
	Timezone        *string `json:"timezone,omitempty"`
	LastTriggeredAt *string `json:"last_triggered_at,omitempty"`
	NextTriggerAt   *string `json:"next_trigger_at,omitempty"`

	// Git event fields
	GitProvider      *string `json:"git_provider,omitempty"`
	GitRepoURL       *string `json:"git_repository_url,omitempty"`
	GitBranchPattern *string `json:"git_branch_pattern,omitempty"`

	// Common fields
	IsActive  bool      `json:"is_active"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// BuildTriggersListResponse represents a list of triggers
type BuildTriggersListResponse struct {
	Triggers []BuildTriggerResponse `json:"triggers"`
	Total    int                    `json:"total"`
}

// Helper function to convert domain trigger to response
func (h *BuildTriggerHandler) triggerToResponse(trigger *build.BuildTrigger) *BuildTriggerResponse {
	resp := &BuildTriggerResponse{
		ID:                 trigger.ID,
		BuildID:            trigger.BuildID,
		ProjectID:          trigger.ProjectID,
		TriggerType:        string(trigger.Type),
		TriggerName:        trigger.Name,
		TriggerDescription: trigger.Description,
		IsActive:           trigger.IsActive,
		CreatedBy:          trigger.CreatedBy,
		CreatedAt:          trigger.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          trigger.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add type-specific fields
	switch trigger.Type {
	case build.TriggerTypeWebhook:
		resp.WebhookURL = &trigger.WebhookURL
		resp.WebhookSecret = &trigger.WebhookSecret
		resp.WebhookEvents = trigger.WebhookEvents

	case build.TriggerTypeSchedule:
		resp.CronExpression = &trigger.CronExpr
		resp.Timezone = &trigger.Timezone
		if trigger.LastTriggered != nil {
			lastTriggered := trigger.LastTriggered.Format("2006-01-02T15:04:05Z07:00")
			resp.LastTriggeredAt = &lastTriggered
		}
		if trigger.NextTrigger != nil {
			nextTrigger := trigger.NextTrigger.Format("2006-01-02T15:04:05Z07:00")
			resp.NextTriggerAt = &nextTrigger
		}

	case build.TriggerTypeGitEvent:
		gitProvider := string(trigger.GitProvider)
		resp.GitProvider = &gitProvider
		resp.GitRepoURL = &trigger.GitRepoURL
		resp.GitBranchPattern = &trigger.GitBranchPattern
	}

	return resp
}

// HTTP Handlers

// CreateWebhookTrigger creates a new webhook trigger
// POST /api/v1/projects/{projectID}/builds/{buildID}/triggers/webhook
func (h *BuildTriggerHandler) CreateWebhookTrigger(w http.ResponseWriter, r *http.Request) {
	buildIDStr := chi.URLParam(r, "buildID")
	projectIDStr := chi.URLParam(r, "projectID")

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.logger.Error("Invalid build ID", zap.Error(err), zap.String("build_id", buildIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid build ID")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("Invalid project ID", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	// Get tenant and user from context
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req CreateWebhookTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create trigger via service
	trigger, err := h.service.CreateWebhookTrigger(
		r.Context(),
		tenantID,
		projectID,
		buildID,
		userID,
		req.Name,
		req.Description,
		req.WebhookURL,
		req.WebhookSecret,
		req.WebhookEvents,
	)
	if err != nil {
		h.logger.Error("Failed to create webhook trigger", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create trigger: %v", err))
		return
	}

	h.respondJSON(w, http.StatusCreated, h.triggerToResponse(trigger))
}

// CreateProjectWebhookTrigger creates a webhook trigger at project scope.
// POST /api/v1/projects/{projectID}/triggers/webhook
func (h *BuildTriggerHandler) CreateProjectWebhookTrigger(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("Invalid project ID", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req CreateProjectWebhookTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var buildID *uuid.UUID
	if req.BuildID != "" {
		parsedBuildID, parseErr := uuid.Parse(req.BuildID)
		if parseErr != nil {
			h.respondError(w, http.StatusBadRequest, "invalid build ID")
			return
		}
		buildID = &parsedBuildID
	}

	trigger, err := h.service.CreateProjectWebhookTrigger(
		r.Context(),
		tenantID,
		projectID,
		buildID,
		userID,
		req.Name,
		req.Description,
		req.WebhookURL,
		req.WebhookSecret,
		req.WebhookEvents,
	)
	if err != nil {
		h.logger.Error("Failed to create project webhook trigger", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to create trigger: %v", err))
		return
	}

	h.respondJSON(w, http.StatusCreated, h.triggerToResponse(trigger))
}

// CreateScheduleTrigger creates a new scheduled trigger
// POST /api/v1/projects/{projectID}/builds/{buildID}/triggers/schedule
func (h *BuildTriggerHandler) CreateScheduleTrigger(w http.ResponseWriter, r *http.Request) {
	buildIDStr := chi.URLParam(r, "buildID")
	projectIDStr := chi.URLParam(r, "projectID")

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.logger.Error("Invalid build ID", zap.Error(err), zap.String("build_id", buildIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid build ID")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("Invalid project ID", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	// Get tenant and user from context
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req CreateScheduleTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create trigger via service
	trigger, err := h.service.CreateScheduledTrigger(
		r.Context(),
		tenantID,
		projectID,
		buildID,
		userID,
		req.Name,
		req.Description,
		req.CronExpr,
		req.Timezone,
	)
	if err != nil {
		h.logger.Error("Failed to create scheduled trigger", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create trigger: %v", err))
		return
	}

	h.respondJSON(w, http.StatusCreated, h.triggerToResponse(trigger))
}

// CreateGitEventTrigger creates a new Git event trigger
// POST /api/v1/projects/{projectID}/builds/{buildID}/triggers/git-event
func (h *BuildTriggerHandler) CreateGitEventTrigger(w http.ResponseWriter, r *http.Request) {
	buildIDStr := chi.URLParam(r, "buildID")
	projectIDStr := chi.URLParam(r, "projectID")

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.logger.Error("Invalid build ID", zap.Error(err), zap.String("build_id", buildIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid build ID")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("Invalid project ID", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	// Get tenant and user from context
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	userID := r.Context().Value("user_id").(uuid.UUID)

	var req CreateGitEventTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create trigger via service
	trigger, err := h.service.CreateGitEventTrigger(
		r.Context(),
		tenantID,
		projectID,
		buildID,
		userID,
		req.Name,
		req.Description,
		build.GitProvider(req.GitProvider),
		req.GitRepoURL,
		req.GitBranchPattern,
	)
	if err != nil {
		h.logger.Error("Failed to create Git event trigger", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create trigger: %v", err))
		return
	}

	h.respondJSON(w, http.StatusCreated, h.triggerToResponse(trigger))
}

// GetBuildTriggers lists all triggers for a build
// GET /api/v1/projects/{projectID}/builds/{buildID}/triggers
func (h *BuildTriggerHandler) GetBuildTriggers(w http.ResponseWriter, r *http.Request) {
	buildIDStr := chi.URLParam(r, "buildID")
	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.logger.Error("Invalid build ID", zap.Error(err), zap.String("build_id", buildIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid build ID")
		return
	}

	triggers, err := h.service.GetBuildTriggers(r.Context(), buildID)
	if err != nil {
		h.logger.Error("Failed to get triggers", zap.Error(err), zap.String("build_id", buildIDStr))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get triggers: %v", err))
		return
	}

	// Convert to response
	triggerResponses := make([]BuildTriggerResponse, len(triggers))
	for i, trigger := range triggers {
		triggerResponses[i] = *h.triggerToResponse(trigger)
	}

	h.respondJSON(w, http.StatusOK, BuildTriggersListResponse{
		Triggers: triggerResponses,
		Total:    len(triggerResponses),
	})
}

// GetProjectTriggers lists all triggers for a project.
// GET /api/v1/projects/{projectID}/triggers
func (h *BuildTriggerHandler) GetProjectTriggers(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("Invalid project ID", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	triggers, err := h.service.GetProjectTriggers(r.Context(), projectID)
	if err != nil {
		h.logger.Error("Failed to get triggers", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get triggers: %v", err))
		return
	}

	triggerResponses := make([]BuildTriggerResponse, len(triggers))
	for i, trigger := range triggers {
		triggerResponses[i] = *h.triggerToResponse(trigger)
	}

	h.respondJSON(w, http.StatusOK, BuildTriggersListResponse{
		Triggers: triggerResponses,
		Total:    len(triggerResponses),
	})
}

// UpdateProjectWebhookTrigger updates a project webhook trigger.
// PATCH /api/v1/projects/{projectID}/triggers/{triggerID}
func (h *BuildTriggerHandler) UpdateProjectWebhookTrigger(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectID")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("Invalid project ID", zap.Error(err), zap.String("project_id", projectIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}
	triggerIDStr := chi.URLParam(r, "triggerID")
	triggerID, err := uuid.Parse(triggerIDStr)
	if err != nil {
		h.logger.Error("Invalid trigger ID", zap.Error(err), zap.String("trigger_id", triggerIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	var req UpdateProjectWebhookTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trigger, err := h.service.UpdateProjectWebhookTrigger(
		r.Context(),
		projectID,
		triggerID,
		req.Name,
		req.Description,
		req.WebhookURL,
		req.WebhookSecret,
		req.WebhookEvents,
		req.IsActive,
	)
	if err != nil {
		h.logger.Error("Failed to update trigger", zap.Error(err), zap.String("trigger_id", triggerIDStr))
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to update trigger: %v", err))
		return
	}

	h.respondJSON(w, http.StatusOK, h.triggerToResponse(trigger))
}

// DeleteTrigger deletes a trigger
// DELETE /api/v1/projects/{projectID}/triggers/{triggerID}
func (h *BuildTriggerHandler) DeleteTrigger(w http.ResponseWriter, r *http.Request) {
	triggerIDStr := chi.URLParam(r, "triggerID")
	triggerID, err := uuid.Parse(triggerIDStr)
	if err != nil {
		h.logger.Error("Invalid trigger ID", zap.Error(err), zap.String("trigger_id", triggerIDStr))
		h.respondError(w, http.StatusBadRequest, "invalid trigger ID")
		return
	}

	if err := h.service.DeleteTrigger(r.Context(), triggerID); err != nil {
		if errors.Is(err, fmt.Errorf("trigger not found: %s", triggerIDStr)) {
			h.respondError(w, http.StatusNotFound, "trigger not found")
			return
		}
		h.logger.Error("Failed to delete trigger", zap.Error(err), zap.String("trigger_id", triggerIDStr))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete trigger: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondJSON sends a JSON response with the specified status code
func (h *BuildTriggerHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

// respondError sends an error response in JSON format
func (h *BuildTriggerHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"success": false,
		"message": message,
	})
}
