package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type EPRRegistrationHandler struct {
	service  *eprregistration.Service
	logger   *zap.Logger
	eventBus messaging.EventBus
}

func NewEPRRegistrationHandler(service *eprregistration.Service, logger *zap.Logger) *EPRRegistrationHandler {
	return &EPRRegistrationHandler{service: service, logger: logger}
}

func (h *EPRRegistrationHandler) SetEventBus(eventBus messaging.EventBus) {
	h.eventBus = eventBus
}

type createEPRRegistrationRequest struct {
	EPRRecordID           string `json:"epr_record_id"`
	ProductName           string `json:"product_name"`
	TechnologyName        string `json:"technology_name"`
	BusinessJustification string `json:"business_justification"`
}

type decideEPRRegistrationRequest struct {
	Reason string `json:"reason"`
}

type bulkLifecycleRequest struct {
	RequestIDs []string `json:"request_ids"`
	Reason     string   `json:"reason"`
}

type eprRegistrationResponse struct {
	ID                    string  `json:"id"`
	TenantID              string  `json:"tenant_id"`
	EPRRecordID           string  `json:"epr_record_id"`
	ProductName           string  `json:"product_name"`
	TechnologyName        string  `json:"technology_name"`
	BusinessJustification string  `json:"business_justification,omitempty"`
	RequestedByUserID     string  `json:"requested_by_user_id"`
	Status                string  `json:"status"`
	LifecycleStatus       string  `json:"lifecycle_status"`
	ApprovedAt            string  `json:"approved_at,omitempty"`
	ExpiresAt             string  `json:"expires_at,omitempty"`
	SuspensionReason      string  `json:"suspension_reason,omitempty"`
	LastReviewedAt        string  `json:"last_reviewed_at,omitempty"`
	DecidedByUserID       *string `json:"decided_by_user_id,omitempty"`
	DecisionReason        string  `json:"decision_reason,omitempty"`
	DecidedAt             string  `json:"decided_at,omitempty"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
}

func (h *EPRRegistrationHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	var req createEPRRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", "invalid request body", nil)
		return
	}

	created, err := h.service.CreateRequest(r.Context(), eprregistration.CreateInput{
		TenantID:              authCtx.TenantID,
		RequestedByUserID:     authCtx.UserID,
		EPRRecordID:           req.EPRRecordID,
		ProductName:           req.ProductName,
		TechnologyName:        req.TechnologyName,
		BusinessJustification: req.BusinessJustification,
	})
	if err != nil {
		switch {
		case errors.Is(err, eprregistration.ErrInvalidEPRRecord),
			errors.Is(err, eprregistration.ErrInvalidProduct),
			errors.Is(err, eprregistration.ErrInvalidTechnology):
			writeImageImportError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		default:
			h.logger.Error("Failed to create EPR registration request", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to create EPR registration request", nil)
		}
		return
	}

	h.publishTransitionEvent(r, authCtx.UserID, created, messaging.EventTypeEPRRegistrationRequested)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": mapEPRRegistrationResponse(created)})
}

func (h *EPRRegistrationHandler) ListTenantRequests(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := (page - 1) * limit
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	items, err := h.service.ListByTenant(r.Context(), authCtx.TenantID, status, limit, offset)
	if err != nil {
		if errors.Is(err, eprregistration.ErrInvalidStatus) {
			writeImageImportError(w, http.StatusBadRequest, "validation_failed", "invalid status filter", nil)
			return
		}
		h.logger.Error("Failed to list tenant EPR registration requests", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to list EPR registration requests", nil)
		return
	}

	rows := make([]eprRegistrationResponse, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapEPRRegistrationResponse(item))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": rows,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *EPRRegistrationHandler) ListAllRequests(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := (page - 1) * limit
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	items, err := h.service.ListAll(r.Context(), status, limit, offset)
	if err != nil {
		if errors.Is(err, eprregistration.ErrInvalidStatus) {
			writeImageImportError(w, http.StatusBadRequest, "validation_failed", "invalid status filter", nil)
			return
		}
		h.logger.Error("Failed to list EPR registration requests", zap.Error(err))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to list EPR registration requests", nil)
		return
	}

	rows := make([]eprRegistrationResponse, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapEPRRegistrationResponse(item))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": rows,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *EPRRegistrationHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	h.decideRequest(w, r, true)
}

func (h *EPRRegistrationHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	h.decideRequest(w, r, false)
}

func (h *EPRRegistrationHandler) WithdrawTenantRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "request id must be a valid uuid", nil)
		return
	}

	var payload decideEPRRegistrationRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	updated, err := h.service.Withdraw(r.Context(), authCtx.TenantID, id, authCtx.UserID, strings.TrimSpace(payload.Reason))
	if err != nil {
		switch {
		case errors.Is(err, eprregistration.ErrNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "EPR registration request not found", nil)
		case errors.Is(err, eprregistration.ErrAlreadyDecided):
			writeImageImportError(w, http.StatusConflict, "approval_already_decided", "EPR registration request is already decided", nil)
		default:
			h.logger.Error("Failed to withdraw EPR registration request", zap.Error(err), zap.String("request_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to withdraw EPR registration request", nil)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": mapEPRRegistrationResponse(updated)})
}

func (h *EPRRegistrationHandler) SuspendRequest(w http.ResponseWriter, r *http.Request) {
	h.lifecycleAction(w, r, "suspend")
}

func (h *EPRRegistrationHandler) ReactivateRequest(w http.ResponseWriter, r *http.Request) {
	h.lifecycleAction(w, r, "reactivate")
}

func (h *EPRRegistrationHandler) RevalidateRequest(w http.ResponseWriter, r *http.Request) {
	h.lifecycleAction(w, r, "revalidate")
}

func (h *EPRRegistrationHandler) BulkSuspendRequests(w http.ResponseWriter, r *http.Request) {
	h.bulkLifecycleAction(w, r, "suspend")
}

func (h *EPRRegistrationHandler) BulkReactivateRequests(w http.ResponseWriter, r *http.Request) {
	h.bulkLifecycleAction(w, r, "reactivate")
}

func (h *EPRRegistrationHandler) BulkRevalidateRequests(w http.ResponseWriter, r *http.Request) {
	h.bulkLifecycleAction(w, r, "revalidate")
}

func (h *EPRRegistrationHandler) decideRequest(w http.ResponseWriter, r *http.Request, approve bool) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}

	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "request id must be a valid uuid", nil)
		return
	}

	var payload decideEPRRegistrationRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	reason := strings.TrimSpace(payload.Reason)

	var updated *eprregistration.Request
	if approve {
		updated, err = h.service.Approve(r.Context(), id, authCtx.UserID, reason)
	} else {
		updated, err = h.service.Reject(r.Context(), id, authCtx.UserID, reason)
	}
	if err != nil {
		switch {
		case errors.Is(err, eprregistration.ErrNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "EPR registration request not found", nil)
		case errors.Is(err, eprregistration.ErrAlreadyDecided):
			writeImageImportError(w, http.StatusConflict, "approval_already_decided", "EPR registration request is already decided", nil)
		default:
			h.logger.Error("Failed to decide EPR registration request", zap.Error(err), zap.String("request_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to decide EPR registration request", nil)
		}
		return
	}

	if approve {
		h.publishTransitionEvent(r, authCtx.UserID, updated, messaging.EventTypeEPRRegistrationApproved)
	} else {
		h.publishTransitionEvent(r, authCtx.UserID, updated, messaging.EventTypeEPRRegistrationRejected)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": mapEPRRegistrationResponse(updated)})
}

func (h *EPRRegistrationHandler) lifecycleAction(w http.ResponseWriter, r *http.Request, action string) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}

	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "request id must be a valid uuid", nil)
		return
	}

	var payload decideEPRRegistrationRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	reason := strings.TrimSpace(payload.Reason)

	var updated *eprregistration.Request
	eventType := ""
	switch action {
	case "suspend":
		updated, err = h.service.Suspend(r.Context(), id, authCtx.UserID, reason)
		eventType = messaging.EventTypeEPRRegistrationSuspended
	case "reactivate":
		updated, err = h.service.Reactivate(r.Context(), id, authCtx.UserID, reason)
		eventType = messaging.EventTypeEPRRegistrationReactivated
	case "revalidate":
		updated, err = h.service.Revalidate(r.Context(), id, authCtx.UserID, reason)
		eventType = messaging.EventTypeEPRRegistrationRevalidated
	default:
		writeImageImportError(w, http.StatusBadRequest, "invalid_action", "invalid lifecycle action", nil)
		return
	}
	if err != nil {
		switch {
		case errors.Is(err, eprregistration.ErrNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "EPR registration request not found", nil)
		case errors.Is(err, eprregistration.ErrLifecycleAction):
			writeImageImportError(w, http.StatusConflict, "invalid_lifecycle_action", "EPR registration lifecycle action not allowed", nil)
		default:
			h.logger.Error("Failed to execute EPR lifecycle action", zap.String("action", action), zap.Error(err), zap.String("request_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to execute EPR lifecycle action", nil)
		}
		return
	}
	h.publishTransitionEvent(r, authCtx.UserID, updated, eventType)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": mapEPRRegistrationResponse(updated)})
}

func (h *EPRRegistrationHandler) bulkLifecycleAction(w http.ResponseWriter, r *http.Request, action string) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}

	var payload bulkLifecycleRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", "invalid request body", nil)
		return
	}
	if len(payload.RequestIDs) == 0 {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", "request_ids must not be empty", nil)
		return
	}
	if len(payload.RequestIDs) > 200 {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", "request_ids must not exceed 200 entries", nil)
		return
	}
	reason := strings.TrimSpace(payload.Reason)

	updatedRows := make([]eprRegistrationResponse, 0, len(payload.RequestIDs))
	for _, rawID := range payload.RequestIDs {
		id, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			writeImageImportError(w, http.StatusBadRequest, "invalid_id", "request_ids must contain valid uuids", nil)
			return
		}
		var updated *eprregistration.Request
		eventType := ""
		switch action {
		case "suspend":
			updated, err = h.service.Suspend(r.Context(), id, authCtx.UserID, reason)
			eventType = messaging.EventTypeEPRRegistrationSuspended
		case "reactivate":
			updated, err = h.service.Reactivate(r.Context(), id, authCtx.UserID, reason)
			eventType = messaging.EventTypeEPRRegistrationReactivated
		case "revalidate":
			updated, err = h.service.Revalidate(r.Context(), id, authCtx.UserID, reason)
			eventType = messaging.EventTypeEPRRegistrationRevalidated
		default:
			writeImageImportError(w, http.StatusBadRequest, "invalid_action", "invalid lifecycle action", nil)
			return
		}
		if err != nil {
			switch {
			case errors.Is(err, eprregistration.ErrNotFound):
				writeImageImportError(w, http.StatusNotFound, "not_found", "EPR registration request not found", map[string]interface{}{"request_id": id.String()})
			case errors.Is(err, eprregistration.ErrLifecycleAction):
				writeImageImportError(w, http.StatusConflict, "invalid_lifecycle_action", "EPR registration lifecycle action not allowed", map[string]interface{}{"request_id": id.String()})
			default:
				h.logger.Error("Failed to execute bulk EPR lifecycle action", zap.String("action", action), zap.Error(err), zap.String("request_id", id.String()))
				writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to execute EPR lifecycle action", map[string]interface{}{"request_id": id.String()})
			}
			return
		}
		h.publishTransitionEvent(r, authCtx.UserID, updated, eventType)
		updatedRows = append(updatedRows, mapEPRRegistrationResponse(updated))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  updatedRows,
		"count": len(updatedRows),
	})
}

func mapEPRRegistrationResponse(req *eprregistration.Request) eprRegistrationResponse {
	var decidedBy *string
	if req.DecidedByUserID != nil {
		v := req.DecidedByUserID.String()
		decidedBy = &v
	}
	decidedAt := ""
	if req.DecidedAt != nil {
		decidedAt = req.DecidedAt.UTC().Format(time.RFC3339)
	}
	approvedAt := ""
	if req.ApprovedAt != nil {
		approvedAt = req.ApprovedAt.UTC().Format(time.RFC3339)
	}
	expiresAt := ""
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt.UTC().Format(time.RFC3339)
	}
	lastReviewedAt := ""
	if req.LastReviewedAt != nil {
		lastReviewedAt = req.LastReviewedAt.UTC().Format(time.RFC3339)
	}
	return eprRegistrationResponse{
		ID:                    req.ID.String(),
		TenantID:              req.TenantID.String(),
		EPRRecordID:           req.EPRRecordID,
		ProductName:           req.ProductName,
		TechnologyName:        req.TechnologyName,
		BusinessJustification: req.BusinessJustification,
		RequestedByUserID:     req.RequestedByUserID.String(),
		Status:                string(req.Status),
		LifecycleStatus:       string(req.LifecycleStatus),
		ApprovedAt:            approvedAt,
		ExpiresAt:             expiresAt,
		SuspensionReason:      req.SuspensionReason,
		LastReviewedAt:        lastReviewedAt,
		DecidedByUserID:       decidedBy,
		DecisionReason:        req.DecisionReason,
		DecidedAt:             decidedAt,
		CreatedAt:             req.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             req.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *EPRRegistrationHandler) publishTransitionEvent(r *http.Request, actorUserID uuid.UUID, req *eprregistration.Request, eventType string) {
	if h == nil || h.eventBus == nil || req == nil {
		return
	}
	authCtx, _ := middleware.GetAuthContext(r)
	payload := map[string]interface{}{
		"epr_registration_request_id": req.ID.String(),
		"tenant_id":                   req.TenantID.String(),
		"requested_by_user_id":        req.RequestedByUserID.String(),
		"epr_record_id":               req.EPRRecordID,
		"product_name":                req.ProductName,
		"technology_name":             req.TechnologyName,
		"status":                      string(req.Status),
		"lifecycle_status":            string(req.LifecycleStatus),
		"idempotency_key":             req.ID.String() + ":" + eventType + ":" + string(req.LifecycleStatus),
	}
	if req.DecisionReason != "" {
		payload["message"] = req.DecisionReason
	}
	event := messaging.Event{
		Type:       eventType,
		TenantID:   req.TenantID.String(),
		ActorID:    actorUserID.String(),
		Source:     "api.epr_registration",
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
	}
	if authCtx != nil {
		event.RequestID = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	// Publish on a detached context: subscribers run asynchronously and should not be canceled
	// when the HTTP request context is closed immediately after response write.
	publishCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := h.eventBus.Publish(publishCtx, event); err != nil {
		h.logger.Warn("Failed to publish EPR registration transition event", zap.String("event_type", eventType), zap.Error(err))
	}
}
