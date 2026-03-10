package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// UserDeactivationHandler handles user deactivation HTTP requests
type UserDeactivationHandler struct {
	userService  user.UserDeactivationService
	auditService interface{} // Will be audit.Service when implemented
	logger       *zap.Logger
}

// NewUserDeactivationHandler creates a new user deactivation handler
func NewUserDeactivationHandler(userService user.UserDeactivationService, auditService interface{}, logger *zap.Logger) *UserDeactivationHandler {
	return &UserDeactivationHandler{
		userService:  userService,
		auditService: auditService,
		logger:       logger,
	}
}

// DeactivateUserRequest represents a user deactivation request
type DeactivateUserRequest struct {
	UserID  string `json:"user_id" validate:"required,uuid"`
	Reason  string `json:"reason" validate:"required,oneof=voluntary admin_action security policy_violation inactivity account_merge"`
	Notes   string `json:"notes,omitempty"`
}

// DeactivateUserResponse represents a user deactivation response
type DeactivateUserResponse struct {
	Message string `json:"message"`
}

// ReactivateUserRequest represents a user reactivation request
type ReactivateUserRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// ReactivateUserResponse represents a user reactivation response
type ReactivateUserResponse struct {
	Message string `json:"message"`
}

// UserDeactivationResponse represents a user deactivation response
type UserDeactivationResponse struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	TenantID      string `json:"tenant_id"`
	DeactivatedBy string `json:"deactivated_by"`
	Reason        string `json:"reason"`
	Notes         string `json:"notes,omitempty"`
	DeactivatedAt string `json:"deactivated_at"`
	ReactivatedAt *string `json:"reactivated_at,omitempty"`
	ReactivatedBy *string `json:"reactivated_by,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// ListDeactivationsResponse represents a list of user deactivations response
type ListDeactivationsResponse struct {
	Deactivations []UserDeactivationResponse `json:"deactivations"`
	Total         int                        `json:"total"`
}

// DeactivateUser handles POST /users/{id}/deactivate
func (h *UserDeactivationHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		WriteError(w, r.Context(), BadRequest("User ID is required"))
		return
	}

	// Parse user ID from URL
	_, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid user ID"))
		return
	}

	var req DeactivateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode deactivate user request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Override userID from URL parameter
	req.UserID = userIDStr

	// Parse user ID from request
	requestUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid user ID in request"))
		return
	}

	// Validate reason
	var reason user.UserDeactivationReason
	switch req.Reason {
	case "voluntary":
		reason = user.UserDeactivationReasonVoluntary
	case "admin_action":
		reason = user.UserDeactivationReasonAdminAction
	case "security":
		reason = user.UserDeactivationReasonSecurity
	case "policy_violation":
		reason = user.UserDeactivationReasonPolicy
	case "inactivity":
		reason = user.UserDeactivationReasonInactivity
	case "account_merge":
		reason = user.UserDeactivationReasonAccountMerge
	default:
		WriteError(w, r.Context(), BadRequest("Invalid deactivation reason"))
		return
	}

	// Get client information
	ipAddress := r.RemoteAddr
	userAgent := r.Header.Get("User-Agent")

	// Deactivate user
	err = h.userService.DeactivateUser(
		r.Context(),
		requestUserID,
		userCtx.UserID,
		reason,
		req.Notes,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.logger.Error("Failed to deactivate user",
			zap.String("user_id", requestUserID.String()),
			zap.String("deactivated_by", userCtx.UserID.String()),
			zap.Error(err),
		)
		WriteError(w, r.Context(), InternalServer("Failed to deactivate user").WithCause(err))
		return
	}

	response := DeactivateUserResponse{
		Message: "User deactivated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User deactivated",
		zap.String("user_id", requestUserID.String()),
		zap.String("deactivated_by", userCtx.UserID.String()),
		zap.String("reason", string(reason)),
	)
}

// ReactivateUser handles POST /users/{id}/reactivate
func (h *UserDeactivationHandler) ReactivateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		WriteError(w, r.Context(), BadRequest("User ID is required"))
		return
	}

	// Parse user ID from URL
	_, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid user ID"))
		return
	}

	var req ReactivateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode reactivate user request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Override userID from URL parameter
	req.UserID = userIDStr

	// Parse user ID from request
	requestUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid user ID in request"))
		return
	}

	// Get client information
	ipAddress := r.RemoteAddr
	userAgent := r.Header.Get("User-Agent")

	// Reactivate user
	err = h.userService.ReactivateUser(
		r.Context(),
		requestUserID,
		userCtx.UserID,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.logger.Error("Failed to reactivate user",
			zap.String("user_id", requestUserID.String()),
			zap.String("reactivated_by", userCtx.UserID.String()),
			zap.Error(err),
		)
		WriteError(w, r.Context(), InternalServer("Failed to reactivate user").WithCause(err))
		return
	}

	response := ReactivateUserResponse{
		Message: "User reactivated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User reactivated",
		zap.String("user_id", requestUserID.String()),
		zap.String("reactivated_by", userCtx.UserID.String()),
	)
}

// GetUserDeactivationHistory handles GET /users/{id}/deactivation-history
func (h *UserDeactivationHandler) GetUserDeactivationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		WriteError(w, r.Context(), BadRequest("User ID is required"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid user ID"))
		return
	}

	deactivations, err := h.userService.GetUserDeactivationHistory(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user deactivation history",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		WriteError(w, r.Context(), InternalServer("Failed to get deactivation history").WithCause(err))
		return
	}

	response := ListDeactivationsResponse{
		Deactivations: make([]UserDeactivationResponse, len(deactivations)),
		Total:         len(deactivations),
	}

	for i, deactivation := range deactivations {
		resp := UserDeactivationResponse{
			ID:            deactivation.ID().String(),
			UserID:        deactivation.UserID().String(),
			TenantID:      deactivation.TenantID().String(),
			DeactivatedBy: deactivation.DeactivatedBy().String(),
			Reason:        string(deactivation.Reason()),
			Notes:         deactivation.Notes(),
			DeactivatedAt: deactivation.DeactivatedAt().Format("2006-01-02T15:04:05Z"),
			CreatedAt:     deactivation.CreatedAt().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     deactivation.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		}

		if reactivatedAt := deactivation.ReactivatedAt(); reactivatedAt != nil {
			reactivatedAtStr := reactivatedAt.Format("2006-01-02T15:04:05Z")
			resp.ReactivatedAt = &reactivatedAtStr
		}

		if reactivatedBy := deactivation.ReactivatedBy(); reactivatedBy != nil {
			reactivatedByStr := reactivatedBy.String()
			resp.ReactivatedBy = &reactivatedByStr
		}

		response.Deactivations[i] = resp
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}