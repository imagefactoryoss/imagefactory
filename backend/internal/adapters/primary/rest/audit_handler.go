package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	auditService *audit.Service
	logger       *zap.Logger
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService *audit.Service, logger *zap.Logger) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
		logger:       logger,
	}
}

// AuditEventResponse represents an audit event in API responses
type AuditEventResponse struct {
	ID        string                 `json:"id"`
	TenantID  *string                `json:"tenant_id,omitempty"`
	UserID    *string                `json:"user_id,omitempty"`
	UserName  string                 `json:"user_name,omitempty"`
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Message   string                 `json:"message"`
	Timestamp string                 `json:"timestamp"`
}

// ListAuditEventsResponse represents a list of audit events response
type ListAuditEventsResponse struct {
	Events []AuditEventResponse `json:"events"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// ListAuditEvents handles GET /api/v1/audit-events
func (h *AuditHandler) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	// Get authenticated user context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	userIDStr := r.URL.Query().Get("user_id")
	eventTypeStr := r.URL.Query().Get("event_type")
	severityStr := r.URL.Query().Get("severity")
	resourceStr := r.URL.Query().Get("resource")
	actionStr := r.URL.Query().Get("action")
	searchStr := r.URL.Query().Get("search")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	tenantIDStr := r.URL.Query().Get("tenant_id")

	limit := 50 // default
	offset := 0 // default

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	// Build filter
	filter := audit.AuditEventFilter{}

	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if eventTypeStr != "" {
		eventType := audit.AuditEventType(eventTypeStr)
		filter.EventType = &eventType
	}

	if severityStr != "" {
		severity := audit.AuditEventSeverity(severityStr)
		filter.Severity = &severity
	}

	if resourceStr != "" {
		filter.Resource = &resourceStr
	}

	if actionStr != "" {
		filter.Action = &actionStr
	}

	if searchStr != "" {
		filter.Search = &searchStr
	}

	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	// Determine tenant ID for querying.
	var queryTenantID *uuid.UUID
	if isAllTenantsScopeRequested(r, authCtx) {
		queryTenantID = nil
	} else {
		queryTenantID = &authCtx.TenantID
	}
	if tenantIDStr != "" {
		parsedTenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant_id"))
			return
		}
		if parsedTenantID == uuid.Nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant_id"))
			return
		}
		if !authCtx.IsSystemAdmin && parsedTenantID != authCtx.TenantID {
			WriteError(w, r.Context(), Forbidden("Access denied to this tenant"))
			return
		}
		queryTenantID = &parsedTenantID
	}

	// Query audit events
	events, err := h.auditService.QueryEvents(r.Context(), queryTenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error("Failed to query audit events", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to query audit events").WithCause(err))
		return
	}

	// Get total count for pagination
	total, err := h.auditService.CountEvents(r.Context(), queryTenantID, filter)
	if err != nil {
		h.logger.Warn("Failed to count audit events", zap.Error(err))
		total = len(events) // fallback
	}

	// Convert to response format
	eventResponses := make([]AuditEventResponse, len(events))
	for i, event := range events {
		var userIDPtr *string
		if event.UserID != nil {
			userIDStr := event.UserID.String()
			userIDPtr = &userIDStr
		}

		var tenantIDPtr *string
		if event.TenantID != nil {
			tenantIDStr := event.TenantID.String()
			tenantIDPtr = &tenantIDStr
		}

		eventResponses[i] = AuditEventResponse{
			ID:        event.ID.String(),
			TenantID:  tenantIDPtr,
			UserID:    userIDPtr,
			UserName:  event.UserName,
			EventType: string(event.EventType),
			Severity:  string(event.Severity),
			Resource:  event.Resource,
			Action:    event.Action,
			IPAddress: event.IPAddress,
			UserAgent: event.UserAgent,
			Details:   event.Details,
			Message:   event.Message,
			Timestamp: event.Timestamp.Format(time.RFC3339),
		}
	}

	response := ListAuditEventsResponse{
		Events: eventResponses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetAuditEvent handles GET /api/v1/audit-events/{id}
func (h *AuditHandler) GetAuditEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	// Get authenticated user context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	// Extract event ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		WriteError(w, r.Context(), BadRequest("Invalid URL format"))
		return
	}

	eventIDStr := pathParts[len(pathParts)-1]
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid audit event ID"))
		return
	}

	// Determine tenant ID for querying.
	var queryTenantID *uuid.UUID
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if isAllTenantsScopeRequested(r, authCtx) {
		queryTenantID = nil
	} else {
		queryTenantID = &authCtx.TenantID
	}
	if tenantIDStr != "" {
		parsedTenantID, err := uuid.Parse(tenantIDStr)
		if err != nil || parsedTenantID == uuid.Nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant_id"))
			return
		}
		if !authCtx.IsSystemAdmin && parsedTenantID != authCtx.TenantID {
			WriteError(w, r.Context(), Forbidden("Access denied to this tenant"))
			return
		}
		queryTenantID = &parsedTenantID
	}
	events, err := h.auditService.QueryEvents(r.Context(), queryTenantID, audit.AuditEventFilter{}, 1, 0)
	if err != nil {
		h.logger.Error("Failed to query audit event", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to get audit event").WithCause(err))
		return
	}

	// Find the specific event
	var foundEvent *audit.AuditEvent
	for _, event := range events {
		if event.ID == eventID {
			foundEvent = event
			break
		}
	}

	if foundEvent == nil {
		WriteError(w, r.Context(), NotFound("Audit event not found"))
		return
	}

	// Convert to response format
	var userIDPtr *string
	if foundEvent.UserID != nil {
		userIDStr := foundEvent.UserID.String()
		userIDPtr = &userIDStr
	}

	var tenantIDPtr *string
	if foundEvent.TenantID != nil {
		tenantIDStr := foundEvent.TenantID.String()
		tenantIDPtr = &tenantIDStr
	}

	response := AuditEventResponse{
		ID:        foundEvent.ID.String(),
		TenantID:  tenantIDPtr,
		UserID:    userIDPtr,
		EventType: string(foundEvent.EventType),
		Severity:  string(foundEvent.Severity),
		Resource:  foundEvent.Resource,
		Action:    foundEvent.Action,
		IPAddress: foundEvent.IPAddress,
		UserAgent: foundEvent.UserAgent,
		Details:   foundEvent.Details,
		Message:   foundEvent.Message,
		Timestamp: foundEvent.Timestamp.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
