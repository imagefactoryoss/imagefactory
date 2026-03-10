package rest

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
)

// PasswordResetHandler handles password reset HTTP requests
type PasswordResetHandler struct {
	userService  user.PasswordResetService
	auditService *audit.Service
	logger       *zap.Logger
}

// NewPasswordResetHandler creates a new password reset handler
func NewPasswordResetHandler(userService user.PasswordResetService, auditService *audit.Service, logger *zap.Logger) *PasswordResetHandler {
	return &PasswordResetHandler{
		userService:  userService,
		auditService: auditService,
		logger:       logger,
	}
}

// RequestPasswordResetRequest represents a password reset request
type RequestPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// RequestPasswordResetResponse represents a password reset response
type RequestPasswordResetResponse struct {
	Message string `json:"message"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ResetPasswordResponse represents a password reset response
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ValidateResetTokenRequest represents a token validation request
type ValidateResetTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// ValidateResetTokenResponse represents a token validation response
type ValidateResetTokenResponse struct {
	Valid     bool   `json:"valid"`
	Email     string `json:"email,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func clientIPFromRequest(r *http.Request) string {
	xForwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xRealIP != "" {
		return xRealIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}

	return strings.Trim(strings.TrimSpace(r.RemoteAddr), "[]")
}

// RequestPasswordReset handles POST /auth/password-reset/request
func (h *PasswordResetHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	var req RequestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode password reset request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Get client information for security tracking
	ipAddress := clientIPFromRequest(r)
	userAgent := r.Header.Get("User-Agent")

	// Request password reset
	err := h.userService.RequestPasswordReset(r.Context(), req.Email, ipAddress, userAgent)
	if err != nil {
		h.logger.Error("Failed to request password reset",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		// Check for specific password reset not allowed error
		if err == user.ErrPasswordResetNotAllowed {
			WriteError(w, r.Context(), BadRequest("This account is managed by your corporate directory (LDAP). Reset your password in your corporate identity system."))
			return
		}
		WriteError(w, r.Context(), InternalServer("Failed to process password reset request").WithCause(err))
		return
	}

	// Always return success for security (don't reveal if email exists)
	response := RequestPasswordResetResponse{
		Message: "If we find your account in the system, you'll receive a password reset link.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Password reset requested",
		zap.String("email", req.Email),
		zap.String("ip_address", ipAddress),
	)

	// Audit password reset request
	if h.auditService != nil {
		h.auditService.LogSystemAction(r.Context(), uuid.Nil, audit.AuditEventPasswordReset, "auth", "request_password_reset",
			"Password reset requested",
			map[string]interface{}{
				"email":      req.Email,
				"ip_address": ipAddress,
				"user_agent": userAgent,
			})
	}
}

// ResetPassword handles POST /auth/password-reset/reset
func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode password reset request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Get client information for security tracking
	ipAddress := clientIPFromRequest(r)
	userAgent := r.Header.Get("User-Agent")

	// Reset password
	err := h.userService.ResetPassword(r.Context(), req.Token, req.NewPassword, ipAddress, userAgent)
	if err != nil {
		h.logger.Error("Failed to reset password", zap.Error(err))

		// Provide generic error messages for security
		switch err {
		case user.ErrTokenNotFound, user.ErrTokenExpired, user.ErrTokenAlreadyUsed, user.ErrInvalidToken:
			WriteError(w, r.Context(), BadRequest("Invalid or expired reset token"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to reset password").WithCause(err))
		}
		return
	}

	response := ResetPasswordResponse{
		Message: "Password has been successfully reset.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Password reset successful",
		zap.String("ip_address", ipAddress),
	)
}

// ValidateResetToken handles POST /auth/validate-reset-token
func (h *PasswordResetHandler) ValidateResetToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	var req ValidateResetTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode token validation request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Validate the token
	validation, err := h.userService.ValidateResetToken(r.Context(), req.Token)
	if err != nil {
		h.logger.Error("Failed to validate reset token", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to validate token").WithCause(err))
		return
	}

	response := ValidateResetTokenResponse{
		Valid:     validation.Valid,
		Email:     validation.Email,
		ExpiresAt: validation.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
