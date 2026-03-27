package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appbootstrap "github.com/srikarm/image-factory/internal/application/bootstrap"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService  *user.Service
	ldapService  *user.LDAPService // LDAP-enabled service
	rbacService  *rbac.Service
	auditService *audit.Service
	bootstrap    *appbootstrap.Service
	logger       *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *user.Service, ldapService *user.LDAPService, rbacService *rbac.Service, auditService *audit.Service, bootstrap *appbootstrap.Service, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		ldapService:  ldapService,
		rbacService:  rbacService,
		auditService: auditService,
		bootstrap:    bootstrap,
		logger:       logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	MFAToken string `json:"mfa_token,omitempty"`
	UseLDAP  bool   `json:"use_ldap,omitempty"` // Flag to use LDAP authentication
}

// LoginResponse represents a login response
type LoginResponse struct {
	User                   UserResponse `json:"user"`
	AccessToken            string       `json:"access_token"`
	RefreshToken           string       `json:"refresh_token"`
	AccessTokenTTL         int          `json:"access_token_ttl"`    // TTL in seconds
	RefreshTokenTTL        int          `json:"refresh_token_ttl"`   // TTL in seconds
	AccessTokenExpiry      int64        `json:"access_token_expiry"` // Unix timestamp
	RequiresMFA            bool         `json:"requires_mfa,omitempty"`
	SetupRequired          bool         `json:"setup_required,omitempty"`
	RequiresPasswordChange bool         `json:"requires_password_change,omitempty"`
}

// UserResponse represents user information in responses
type UserResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Status     string `json:"status"`
	IsActive   bool   `json:"is_active"`
	AuthMethod string `json:"auth_method"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshTokenResponse represents a token refresh response
type RefreshTokenResponse struct {
	AccessToken       string       `json:"access_token"`
	RefreshToken      string       `json:"refresh_token"`
	AccessTokenTTL    int          `json:"access_token_ttl"`    // TTL in seconds
	RefreshTokenTTL   int          `json:"refresh_token_ttl"`   // TTL in seconds
	AccessTokenExpiry int64        `json:"access_token_expiry"` // Unix timestamp
	User              UserResponse `json:"user"`
}

// LogoutResponse represents a logout response
type LogoutResponse struct {
	Message string `json:"message"`
}

// LoginOptionsResponse represents public login capability flags.
type LoginOptionsResponse struct {
	LDAPEnabled bool `json:"ldap_enabled"`
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		h.logger.Error("Failed to read request body", zap.Error(readErr))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if decodeErr := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&req); decodeErr != nil {
		h.logger.Error("Failed to decode login request", zap.Error(decodeErr), zap.Int("body_bytes", len(bodyBytes)))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Email == "" {
		h.logger.Warn("Login attempt with empty email")
		WriteError(w, r.Context(), BadRequest("Email is required"))
		return
	}
	if req.Password == "" {
		h.logger.Warn("Login attempt with empty password", zap.String("email", req.Email))
		WriteError(w, r.Context(), BadRequest("Password is required"))
		return
	}

	// Basic email format validation
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		h.logger.Warn("Login attempt with invalid email format", zap.String("email", req.Email))
		WriteError(w, r.Context(), BadRequest("Invalid email format"))
		return
	}

	var result *user.AuthResult
	var err error

	// Check if LDAP login is requested
	if req.UseLDAP {
		// Use LDAP authentication (users may be provisioned without tenant assignment)
		h.logger.Info("Attempting LDAP login", zap.String("email", req.Email))
		result, err = h.ldapService.LDAPLogin(r.Context(), user.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
			MFAToken: req.MFAToken,
		})
	} else {
		// Use standard authentication
		h.logger.Info("Attempting standard login", zap.String("email", req.Email))
		result, err = h.authService.Login(r.Context(), user.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
			MFAToken: req.MFAToken,
		})
	}

	if err != nil {
		h.logger.Warn("Login failed", zap.String("email", req.Email), zap.Error(err))

		// Audit login failure
		if h.auditService != nil {
			h.auditService.LogSystemAction(r.Context(), uuid.Nil, audit.AuditEventLoginFailure, "auth", "login",
				"Failed login attempt", map[string]interface{}{
					"email":  req.Email,
					"reason": err.Error(),
				})
		}

		// Return specific error message instead of generic message
		WriteError(w, r.Context(), Unauthorized(err.Error()))
		return
	}

	// Convert user to response format
	userResp := UserResponse{
		ID:        result.User.ID().String(),
		Email:     result.User.Email(),
		FirstName: result.User.FirstName(),
		LastName:  result.User.LastName(),
		Status:    string(result.User.Status()),
		IsActive:  result.User.IsActive(),
	}

	// Calculate token expiration times
	accessTokenExpiry := time.Now().Add(15 * time.Minute).Unix()

	response := LoginResponse{
		User:                   userResp,
		AccessToken:            result.AccessToken,
		RefreshToken:           result.RefreshToken,
		AccessTokenTTL:         900,    // 15 minutes in seconds
		RefreshTokenTTL:        604800, // 7 days in seconds
		AccessTokenExpiry:      accessTokenExpiry,
		RequiresMFA:            result.RequiresMFA,
		RequiresPasswordChange: result.RequiresPasswordChange,
	}
	if h.bootstrap != nil {
		setupRequired, setupErr := h.bootstrap.IsSetupRequired(r.Context())
		if setupErr != nil {
			h.logger.Warn("Failed to resolve bootstrap setup status during login", zap.Error(setupErr))
		} else {
			response.SetupRequired = setupRequired
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User logged in successfully", zap.String("user_id", result.User.ID().String()))

	// Audit successful login
	if h.auditService != nil {
		// Get tenant ID from auth context
		authCtx, ok := middleware.GetAuthContext(r)
		tenantID := uuid.Nil
		if ok {
			tenantID = authCtx.TenantID
		}
		h.auditService.LogUserAction(r.Context(), tenantID, result.User.ID(),
			audit.AuditEventLoginSuccess, "auth", "login", "User logged in successfully", nil)
	}
}

// LoginOptions handles GET /auth/login-options
func (h *AuthHandler) LoginOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ldapEnabled := false
	if h.ldapService != nil {
		enabled, err := h.ldapService.IsLDAPLoginEnabled(r.Context())
		if err != nil {
			h.logger.Warn("Failed to resolve LDAP login availability", zap.Error(err))
		} else {
			ldapEnabled = enabled
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(LoginOptionsResponse{LDAPEnabled: ldapEnabled})
}

// RefreshToken handles POST /auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode refresh token request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	if req.RefreshToken == "" {
		h.logger.Warn("Token refresh attempt with empty refresh token")
		WriteError(w, r.Context(), BadRequest("Refresh token is required"))
		return
	}

	// Refresh token
	result, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Error("Token refresh failed", zap.Error(err))
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Calculate token expiration times
	accessTokenExpiry := time.Now().Add(15 * time.Minute).Unix()

	// Convert user to response format
	userResp := UserResponse{
		ID:        result.User.ID().String(),
		Email:     result.User.Email(),
		FirstName: result.User.FirstName(),
		LastName:  result.User.LastName(),
		Status:    string(result.User.Status()),
		IsActive:  result.User.IsActive(),
	}

	response := RefreshTokenResponse{
		AccessToken:       result.AccessToken,
		RefreshToken:      result.RefreshToken,
		AccessTokenTTL:    900,    // 15 minutes in seconds
		RefreshTokenTTL:   604800, // 7 days in seconds
		AccessTokenExpiry: accessTokenExpiry,
		User:              userResp,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Token refreshed successfully", zap.String("user_id", result.User.ID().String()))
}

// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user info from context if authenticated
	var userID string
	var tenantID uuid.UUID

	if authCtx, ok := middleware.GetAuthContext(r); ok {
		userID = authCtx.UserID.String()
		tenantID = authCtx.TenantID
	}

	// For stateless JWT, logout is handled client-side by removing tokens
	// In a future implementation, we could implement token blacklisting for additional security

	response := LogoutResponse{
		Message: "Logged out successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	// Log logout if user was authenticated
	if userID != "" {
		h.logger.Info("User logged out", zap.String("user_id", userID))

		// Audit logout action
		if h.auditService != nil {
			h.auditService.LogUserAction(r.Context(), tenantID, uuid.MustParse(userID),
				audit.AuditEventLogout, "auth", "logout", "User logged out successfully", nil)
		}
	}
}

// Me handles GET /auth/me - returns current user information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user from context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user details and roles in optimized query to avoid N+1 problem
	// This uses a single database query instead of multiple separate queries
	userWithRoles, err := h.rbacService.GetUserWithRoles(r.Context(), authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to get user with roles", zap.String("user_id", authCtx.UserID.String()), zap.Error(err))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Convert roles to response format - optimized batch conversion
	roleResponses := make([]RoleResponse, 0, len(userWithRoles.Roles))
	for _, roleData := range userWithRoles.Roles {
		var permissions []rbac.Permission
		if err := json.Unmarshal([]byte(roleData.Permissions), &permissions); err != nil {
			h.logger.Warn("Failed to parse role permissions",
				zap.String("role_id", roleData.ID.String()),
				zap.Error(err))
			continue
		}

		roleResponses = append(roleResponses, RoleResponse{
			ID:          roleData.ID.String(),
			Name:        roleData.Name,
			Description: roleData.Description,
			IsSystem:    roleData.IsSystem,
		})
	}

	response := MeResponse{
		User: UserResponse{
			ID:         userWithRoles.ID.String(),
			Email:      userWithRoles.Email,
			FirstName:  userWithRoles.FirstName,
			LastName:   userWithRoles.LastName,
			Status:     userWithRoles.Status,
			IsActive:   userWithRoles.IsActive,
			AuthMethod: userWithRoles.AuthMethod,
		},
		Roles: roleResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RoleResponse represents role information in responses
type RoleResponse struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	IsSystem    bool   `json:"is_system" db:"is_system"`
}

// MeResponse represents the response for /auth/me
type MeResponse struct {
	User  UserResponse   `json:"user"`
	Roles []RoleResponse `json:"roles"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ChangePassword handles POST /auth/change-password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user from context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode change password request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CurrentPassword == "" {
		h.logger.Warn("Password change attempt with empty current password", zap.String("user_id", authCtx.UserID.String()))
		WriteError(w, r.Context(), BadRequest("Current password is required"))
		return
	}
	if req.NewPassword == "" {
		h.logger.Warn("Password change attempt with empty new password", zap.String("user_id", authCtx.UserID.String()))
		WriteError(w, r.Context(), BadRequest("New password is required"))
		return
	}

	// Basic password strength validation
	if len(req.NewPassword) < 8 {
		h.logger.Warn("Password change attempt with weak password", zap.String("user_id", authCtx.UserID.String()))
		WriteError(w, r.Context(), BadRequest("Password must be at least 8 characters long"))
		return
	}

	// Verify current password first
	user, err := h.authService.GetUserByID(r.Context(), authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to get user for password change", zap.String("user_id", authCtx.UserID.String()), zap.Error(err))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if !user.VerifyPassword(req.CurrentPassword) {
		http.Error(w, "Current password is incorrect", http.StatusBadRequest)
		return
	}

	// Change password
	if err := h.authService.ChangePassword(r.Context(), authCtx.UserID, req.NewPassword); err != nil {
		h.logger.Error("Failed to change password", zap.String("user_id", authCtx.UserID.String()), zap.Error(err))

		// Audit password change failure
		if h.auditService != nil {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID,
				audit.AuditEventPasswordChange, "auth", "change_password", "Password change failed", map[string]interface{}{
					"reason": err.Error(),
				})
		}

		WriteError(w, r.Context(), InternalServer("Failed to change password").WithCause(err))
		return
	}

	response := map[string]string{
		"message": "Password changed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Password changed successfully", zap.String("user_id", authCtx.UserID.String()))

	// Audit successful password change
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID,
			audit.AuditEventPasswordChange, "auth", "change_password", "Password changed successfully", nil)
	}
}

// SearchLDAPUsersRequest represents a request to search LDAP users
type SearchLDAPUsersRequest struct {
	Query string `json:"query" validate:"required,min=2"`
	Limit int    `json:"limit,omitempty"`
}

// LDAPUserResult represents a simplified LDAP user for search results
type LDAPUserResult struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// SearchLDAPUsersResponse represents the response for LDAP user search
type SearchLDAPUsersResponse struct {
	Users []LDAPUserResult `json:"users"`
}

// SearchLDAPUsers handles POST /auth/ldap/search-users
func (h *AuthHandler) SearchLDAPUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchLDAPUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode LDAP search request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate query
	if req.Query == "" || len(req.Query) < 2 {
		WriteError(w, r.Context(), BadRequest("Search query must be at least 2 characters"))
		return
	}

	// Set default limit
	if req.Limit <= 0 || req.Limit > 20 {
		req.Limit = 10
	}

	// Search LDAP users
	ldapUsers, err := h.ldapService.SearchLDAPUsers(r.Context(), req.Query, req.Limit)
	if err != nil {
		h.logger.Error("Failed to search LDAP users",
			zap.String("query", req.Query),
			zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to search LDAP users").WithCause(err))
		return
	}

	// Convert to response format
	results := make([]LDAPUserResult, len(ldapUsers))
	for i, user := range ldapUsers {
		results[i] = LDAPUserResult{
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FirstName + " " + user.LastName,
		}
	}

	response := SearchLDAPUsersResponse{
		Users: results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
