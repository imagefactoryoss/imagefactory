package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/mfa"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// MFAHandler handles MFA HTTP requests
type MFAHandler struct {
	mfaService   *mfa.Service
	auditService *audit.Service
	logger       *zap.Logger
}

// NewMFAHandler creates a new MFA handler
func NewMFAHandler(mfaService *mfa.Service, auditService *audit.Service, logger *zap.Logger) *MFAHandler {
	return &MFAHandler{
		mfaService:   mfaService,
		auditService: auditService,
		logger:       logger,
	}
}

// SetupTOTPSecretRequest represents a request to setup TOTP
type SetupTOTPSecretRequest struct {
	AccountName string `json:"account_name" validate:"required"`
	Issuer      string `json:"issuer" validate:"required"`
}

// SetupTOTPSecretResponse represents the response for TOTP setup
type SetupTOTPSecretResponse struct {
	Secret      string `json:"secret"`
	QRCodeURL   string `json:"qr_code_url"`
	AccountName string `json:"account_name"`
	Issuer      string `json:"issuer"`
	Algorithm   string `json:"algorithm"`
	Digits      int    `json:"digits"`
	Period      int    `json:"period"`
}

// SetupTOTPConfirmRequest represents a request to confirm TOTP setup
type SetupTOTPConfirmRequest struct {
	Secret string `json:"secret" validate:"required"`
	Code   string `json:"code" validate:"required"`
}

// MFAChallengeResponse represents the response for MFA challenge
type MFAChallengeResponse struct {
	ID        string `json:"id"`
	Method    string `json:"method"`
	ExpiresAt string `json:"expires_at"`
	Status    string `json:"status"`
	Attempts  int    `json:"attempts"`
}

// SetupTOTPSecret handles POST /mfa/totp/setup/secret
func (h *MFAHandler) SetupTOTPSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (assumed to be set by auth middleware)
	userID, err := h.getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SetupTOTPSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode TOTP setup request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate TOTP secret
	totpSecret, err := h.mfaService.SetupTOTPSecret(r.Context(), userID, req.AccountName, req.Issuer)
	if err != nil {
		h.logger.Error("Failed to setup TOTP secret",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		http.Error(w, "Failed to setup TOTP secret", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := SetupTOTPSecretResponse{
		Secret:      totpSecret.Secret,
		QRCodeURL:   totpSecret.QRCodeURL,
		AccountName: totpSecret.AccountName,
		Issuer:      totpSecret.Issuer,
		Algorithm:   totpSecret.Algorithm,
		Digits:      totpSecret.Digits,
		Period:      totpSecret.Period,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("TOTP secret generated",
		zap.String("user_id", userID.String()),
		zap.String("account_name", req.AccountName))
}

// ConfirmTOTPSetup handles POST /mfa/totp/setup/confirm
func (h *MFAHandler) ConfirmTOTPSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID, err := h.getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SetupTOTPConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode TOTP confirm request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Confirm TOTP setup
	if err := h.mfaService.ConfirmTOTPSetup(r.Context(), userID, req.Secret, req.Code); err != nil {
		h.logger.Error("Failed to confirm TOTP setup",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		http.Error(w, "Failed to confirm TOTP setup: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "TOTP setup completed successfully",
	})

	h.logger.Info("TOTP setup confirmed", zap.String("user_id", userID.String()))

	// Audit MFA enable
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, userID,
				audit.AuditEventMFAEnable, "mfa", "enable_totp",
				"TOTP MFA enabled successfully",
				map[string]interface{}{
					"user_id": userID.String(),
				})
		}
	}
}

// StartMFAChallengeRequest represents a request to start an MFA challenge
type StartMFAChallengeRequest struct {
	Method mfa.MFAMethod `json:"method" validate:"required"`
}

// StartMFAChallenge handles POST /mfa/challenge
func (h *MFAHandler) StartMFAChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID, err := h.getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req StartMFAChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode MFA challenge request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Start MFA challenge
	challengeReq := mfa.MFAChallengeRequest{
		UserID: userID,
		Method: req.Method,
	}

	challenge, err := h.mfaService.StartMFAChallenge(r.Context(), challengeReq)
	if err != nil {
		h.logger.Error("Failed to start MFA challenge",
			zap.String("user_id", userID.String()),
			zap.String("method", string(req.Method)),
			zap.Error(err))
		http.Error(w, "Failed to start MFA challenge: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to response format
	response := MFAChallengeResponse{
		ID:        challenge.ID.String(),
		Method:    string(challenge.Method),
		ExpiresAt: challenge.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:    string(challenge.Status),
		Attempts:  challenge.Attempts,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("MFA challenge started",
		zap.String("user_id", userID.String()),
		zap.String("method", string(req.Method)))
}

// VerifyMFAChallengeRequest represents a request to verify an MFA challenge
type VerifyMFAChallengeRequest struct {
	ChallengeID uuid.UUID `json:"challenge_id" validate:"required"`
	Code        string    `json:"code" validate:"required"`
}

// VerifyMFAChallenge handles POST /mfa/verify
func (h *MFAHandler) VerifyMFAChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyMFAChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode MFA verify request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify MFA challenge
	verifyReq := mfa.MFAVerificationRequest{
		ChallengeID: req.ChallengeID,
		Code:        req.Code,
	}

	if err := h.mfaService.VerifyMFAChallenge(r.Context(), verifyReq); err != nil {
		h.logger.Error("MFA verification failed",
			zap.String("challenge_id", req.ChallengeID.String()),
			zap.Error(err))
		http.Error(w, "Invalid code or expired challenge", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "MFA verification successful",
	})

	h.logger.Info("MFA verification successful",
		zap.String("challenge_id", req.ChallengeID.String()))
}

// GetMFAStatus handles GET /mfa/status
func (h *MFAHandler) GetMFAStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID, err := h.getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's MFA methods (would need to implement this in the service)
	// For now, return a simple status
	response := map[string]interface{}{
		"user_id": userID.String(),
		"enabled": false, // TODO: Check if user has any MFA methods configured
		"methods": []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getUserIDFromContext extracts user ID from request context
// This is a placeholder implementation - in a real app, this would extract
// the user ID from JWT token or session data set by authentication middleware
func (h *MFAHandler) getUserIDFromContext(r *http.Request) (uuid.UUID, error) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("auth context missing user id")
	}
	return authCtx.UserID, nil
}
