package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// UserInvitationHandler handles user invitation HTTP requests
type UserInvitationHandler struct {
	userService         user.UserInvitationService
	rbacService         interface{} // RBAC service for role lookups
	auditService        *audit.Service
	logger              *zap.Logger
	ldapService         *user.LDAPService
	notificationService *email.NotificationService
	db                  *sqlx.DB
	config              *config.Config
}

// NewUserInvitationHandler creates a new user invitation handler
func NewUserInvitationHandler(
	userService user.UserInvitationService,
	rbacService interface{},
	auditService *audit.Service,
	logger *zap.Logger,
	ldapService *user.LDAPService,
	cfg *config.Config,
) *UserInvitationHandler {
	return &UserInvitationHandler{
		userService:  userService,
		rbacService:  rbacService,
		auditService: auditService,
		logger:       logger,
		ldapService:  ldapService,
		config:       cfg,
	}
}

// SetNotificationService sets the notification service for sending emails
func (h *UserInvitationHandler) SetNotificationService(notificationService *email.NotificationService) {
	h.notificationService = notificationService
}

// SetDatabase sets the database connection for group operations
func (h *UserInvitationHandler) SetDatabase(db *sqlx.DB) {
	h.db = db
}

// CreateInvitationRequest represents a user invitation creation request
type CreateInvitationRequest struct {
	TenantID string `json:"tenant_id" validate:"required,uuid"`
	Email    string `json:"email" validate:"required,email"`
	RoleID   string `json:"role_id" validate:"required,uuid"`
	Message  string `json:"message,omitempty"`
	IsLDAP   bool   `json:"is_ldap,omitempty"`
}

// CreateInvitationResponse represents a user invitation creation response
type CreateInvitationResponse struct {
	Invitation UserInvitationResponse `json:"invitation"`
	Token      string                 `json:"token,omitempty"` // Only included for development/testing
}

// UserInvitationResponse represents a user invitation response
type UserInvitationResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	RoleID    string `json:"role_id"`
	RoleName  string `json:"role_name,omitempty"`
	InvitedBy string `json:"invited_by"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// AcceptInvitationRequest represents an invitation acceptance request
// For LDAP users: password is optional (auto-generated dummy password will be used)
// For regular users: password is required
type AcceptInvitationRequest struct {
	Token     string `json:"token" validate:"required"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Password  string `json:"password" validate:"omitempty,min=8"` // Optional - only for non-LDAP users
	IsLDAP    bool   `json:"is_ldap,omitempty"`                   // Indicates if this is an LDAP user
}

// AcceptInvitationResponse represents an invitation acceptance response
type AcceptInvitationResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// ListInvitationsResponse represents a list of invitations response
type ListInvitationsResponse struct {
	Invitations []UserInvitationResponse `json:"invitations"`
	Total       int                      `json:"total"`
}

// CreateInvitation handles POST /invitations
func (h *UserInvitationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("CreateInvitation called", zap.String("method", r.Method), zap.String("url", r.URL.String()))

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

	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create invitation request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Parse tenant ID
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid tenant ID"))
		return
	}

	// LDAP users: add immediately to tenant without invitation email or acceptance flow
	if req.IsLDAP {
		h.handleLDAPUserInvite(w, r, userCtx, tenantID, req)
		return
	}

	// Fallback: auto-detect LDAP users by email and add directly.
	// This keeps tenant-owner invite flow deterministic even when UI does not explicitly set is_ldap.
	if h.ldapService != nil {
		if ldapUsers, ldapErr := h.ldapService.SearchLDAPUsers(r.Context(), req.Email, 1); ldapErr == nil && len(ldapUsers) > 0 {
			if strings.EqualFold(strings.TrimSpace(ldapUsers[0].Email), strings.TrimSpace(req.Email)) {
				h.logger.Info("Auto-detected LDAP user during invitation flow; adding directly",
					zap.String("email", req.Email),
					zap.String("tenant_id", tenantID.String()))
				req.IsLDAP = true
				h.handleLDAPUserInvite(w, r, userCtx, tenantID, req)
				return
			}
		}
	}

	// Create invitation
	invitation, token, err := h.userService.CreateInvitation(
		r.Context(),
		tenantID,
		userCtx.UserID,
		req.Email,
		req.RoleID,
		req.Message,
	)
	if err != nil {
		h.logger.Error("Failed to create invitation",
			zap.String("email", req.Email),
			zap.String("tenant_id", req.TenantID),
			zap.Error(err),
		)

		switch err {
		case user.ErrInvitationAlreadyExists:
			WriteError(w, r.Context(), Conflict("Invitation already exists for this email and tenant"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to create invitation").WithCause(err))
		}
		return
	}

	response := CreateInvitationResponse{
		Invitation: UserInvitationResponse{
			ID:        invitation.ID().String(),
			TenantID:  invitation.TenantID().String(),
			Email:     invitation.Email(),
			RoleID:    invitation.RoleID().String(),
			InvitedBy: invitation.InvitedBy().String(),
			Status:    string(invitation.Status()),
			ExpiresAt: invitation.ExpiresAt().Format("2006-01-02T15:04:05Z"),
			Message:   invitation.Message(),
			CreatedAt: invitation.CreatedAt().Format("2006-01-02T15:04:05Z"),
			UpdatedAt: invitation.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		},
		// Token only included in development mode
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User invitation created",
		zap.String("invitation_id", invitation.ID().String()),
		zap.String("email", req.Email),
		zap.String("invited_by", userCtx.UserID.String()),
	)

	// Audit invitation creation
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), tenantID, userCtx.UserID,
			audit.AuditEventUserInviteCreate, "invitations", "create",
			"User invitation created",
			map[string]interface{}{
				"invitation_id": invitation.ID().String(),
				"email":         req.Email,
				"role_id":       req.RoleID,
				"tenant_id":     tenantID.String(),
			})
	}
}

func (h *UserInvitationHandler) handleLDAPUserInvite(
	w http.ResponseWriter,
	r *http.Request,
	userCtx *middleware.AuthContext,
	tenantID uuid.UUID,
	req CreateInvitationRequest,
) {
	if h.ldapService == nil {
		WriteError(w, r.Context(), BadRequest("LDAP is not configured"))
		return
	}

	roleID, roleName, err := h.resolveRoleID(r.Context(), req.RoleID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid role ID"))
		return
	}

	ldapUser, err := h.ldapService.EnsureLocalUser(r.Context(), req.Email)
	if err != nil {
		h.logger.Error("Failed to ensure LDAP user", zap.Error(err), zap.String("email", req.Email))
		WriteError(w, r.Context(), BadRequest("LDAP user not found or not eligible"))
		return
	}

	if ldapUser.Status() == user.UserStatusSuspended {
		WriteError(w, r.Context(), BadRequest("Cannot add suspended user to tenant"))
		return
	}

	rbacSvc, ok := h.rbacService.(*rbac.Service)
	if !ok {
		WriteError(w, r.Context(), InternalServer("RBAC service not available"))
		return
	}

	if err := rbacSvc.AssignRoleToUserForTenant(r.Context(), ldapUser.ID(), roleID, tenantID, userCtx.UserID); err != nil {
		h.logger.Error("Failed to assign role to LDAP user",
			zap.Error(err),
			zap.String("user_id", ldapUser.ID().String()),
			zap.String("role_id", roleID.String()),
			zap.String("tenant_id", tenantID.String()))
		WriteError(w, r.Context(), InternalServer("Failed to assign role to user"))
		return
	}

	if err := h.addUserToGroupByRole(r.Context(), ldapUser.ID(), roleID, tenantID); err != nil {
		h.logger.Warn("Failed to add LDAP user to group", zap.Error(err))
	}

	h.sendUserAddedNotification(r.Context(), ldapUser, tenantID, roleName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "LDAP user added to tenant successfully",
	})

	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), tenantID, userCtx.UserID,
			audit.AuditEventUserInviteCreate, "invitations", "create",
			"LDAP user added to tenant",
			map[string]interface{}{
				"email":     req.Email,
				"role_id":   roleID.String(),
				"tenant_id": tenantID.String(),
				"is_ldap":   true,
			})
	}
}

func (h *UserInvitationHandler) resolveRoleID(ctx context.Context, roleIDRaw string) (uuid.UUID, string, error) {
	if id, err := uuid.Parse(roleIDRaw); err == nil {
		roleName := ""
		if rbacSvc, ok := h.rbacService.(*rbac.Service); ok {
			if role, err := rbacSvc.GetRoleByID(ctx, id); err == nil && role != nil {
				roleName = role.Name()
			}
		}
		return id, roleName, nil
	}

	rbacSvc, ok := h.rbacService.(*rbac.Service)
	if !ok {
		return uuid.Nil, "", errors.New("rbac service not available")
	}

	roles, err := rbacSvc.GetAllSystemLevelRoles(ctx)
	if err != nil {
		return uuid.Nil, "", err
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name(), roleIDRaw) {
			return role.ID(), role.Name(), nil
		}
	}

	return uuid.Nil, "", errors.New("role not found")
}

func (h *UserInvitationHandler) addUserToGroupByRole(ctx context.Context, userID, roleID, tenantID uuid.UUID) error {
	if h.db == nil {
		h.logger.Warn("Database not available for adding user to group")
		return nil
	}

	rbacSvc, ok := h.rbacService.(*rbac.Service)
	if !ok {
		return nil
	}

	role, err := rbacSvc.GetRoleByID(ctx, roleID)
	if err != nil {
		h.logger.Warn("Failed to get role for group assignment",
			zap.String("role_id", roleID.String()),
			zap.Error(err))
		return nil
	}

	query := `
		SELECT id FROM tenant_groups 
		WHERE tenant_id = $1 AND role_type = $2 AND status = 'active'
		LIMIT 1
	`

	var groupID uuid.UUID
	err = h.db.QueryRowContext(ctx, query, tenantID, strings.ToLower(strings.ReplaceAll(role.Name(), " ", "_"))).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("Group not found for role",
				zap.String("tenant_id", tenantID.String()),
				zap.String("role_name", role.Name()))
			return nil
		}
		h.logger.Warn("Failed to find group for role",
			zap.String("role_name", role.Name()),
			zap.Error(err))
		return nil
	}

	insertQuery := `
		INSERT INTO group_members (id, group_id, user_id, is_group_admin, added_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (group_id, user_id) DO UPDATE
		SET removed_at = NULL
	`

	memberID := uuid.New()
	if _, err := h.db.ExecContext(ctx, insertQuery, memberID, groupID, userID, false); err != nil {
		h.logger.Warn("Failed to add user to group",
			zap.String("group_id", groupID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil
	}

	h.logger.Info("Added user to group",
		zap.String("group_id", groupID.String()),
		zap.String("user_id", userID.String()),
		zap.String("role_name", role.Name()))
	return nil
}

func (h *UserInvitationHandler) sendUserAddedNotification(ctx context.Context, usr *user.User, tenantID uuid.UUID, roleName string) {
	if h.notificationService == nil {
		return
	}

	tenantName := "Tenant"
	if h.db != nil {
		var name string
		if err := h.db.GetContext(ctx, &name, "SELECT name FROM tenants WHERE id = $1", tenantID.String()); err == nil {
			tenantName = name
		}
	}

	notifData := &email.UserAddedToTenantData{
		UserEmail:    usr.Email(),
		UserName:     usr.FullName(),
		TenantName:   tenantName,
		TenantID:     tenantID,
		Role:         roleName,
		DashboardURL: "",
	}

	if h.config != nil {
		notifData.DashboardURL = h.config.Frontend.DashboardURL
	}

	go func() {
		if err := h.notificationService.SendUserAddedToTenantEmail(context.Background(), notifData); err != nil {
			h.logger.Error("Failed to send user added to tenant notification",
				zap.String("user_email", usr.Email()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
		}
	}()
}

// AcceptInvitation handles POST /invitations/accept
func (h *UserInvitationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode accept invitation request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Get client information
	ipAddress := r.RemoteAddr
	userAgent := r.Header.Get("User-Agent")

	// Validate password requirement
	if !req.IsLDAP && req.Password == "" {
		WriteError(w, r.Context(), BadRequest("Password is required for non-LDAP users"))
		return
	}

	// Accept invitation
	user, err := h.userService.AcceptInvitation(
		r.Context(),
		req.Token,
		req.FirstName,
		req.LastName,
		req.Password,
		req.IsLDAP,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.logger.Error("Failed to accept invitation", zap.Error(err))

		// Generic error handling for now
		WriteError(w, r.Context(), BadRequest("Invalid or expired invitation token"))
		return
	}

	// Get the invitation by token to retrieve role assignment info
	invitation, err := h.userService.GetInvitationByToken(r.Context(), req.Token)
	if err != nil {
		h.logger.Warn("Failed to retrieve invitation details for role assignment", zap.Error(err))
	} else if invitation != nil && invitation.RoleID() != uuid.Nil {
		// Assign the role to the user for the tenant if RBAC service is available
		if rbacSvc, ok := h.rbacService.(*rbac.Service); ok {
			tenantID := invitation.TenantID()
			roleID := invitation.RoleID()

			// Use user's own ID as the assigner since they just accepted
			if err := rbacSvc.AssignRoleToUserForTenant(r.Context(), user.ID(), roleID, tenantID, user.ID()); err != nil {
				h.logger.Warn("Failed to assign role to user after invitation acceptance",
					zap.String("user_id", user.ID().String()),
					zap.String("role_id", roleID.String()),
					zap.String("tenant_id", tenantID.String()),
					zap.Error(err))
				// Don't fail the entire operation if role assignment fails
			} else {
				h.logger.Info("Role assigned to user after invitation acceptance",
					zap.String("user_id", user.ID().String()),
					zap.String("role_id", roleID.String()),
					zap.String("tenant_id", tenantID.String()))
			}
		}
	}

	// Generate access token for the new user
	accessToken, err := h.userService.GenerateAccessToken(r.Context(), user)
	if err != nil {
		h.logger.Error("Failed to generate access token for new user",
			zap.String("user_id", user.ID().String()),
			zap.Error(err))
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := AcceptInvitationResponse{
		User: UserResponse{
			ID:        user.ID().String(),
			Email:     user.Email(),
			FirstName: user.FirstName(),
			LastName:  user.LastName(),
			Status:    string(user.Status()),
			IsActive:  user.IsActive(),
		},
		Token: accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User invitation accepted",
		zap.String("user_id", user.ID().String()),
		zap.String("email", user.Email()),
	)

	// Audit invitation acceptance (system action since user isn't authenticated yet)
	if h.auditService != nil {
		h.auditService.LogSystemAction(r.Context(), uuid.Nil, audit.AuditEventUserInviteAccept, "invitations", "accept",
			"User invitation accepted and account created",
			map[string]interface{}{
				"user_id":    user.ID().String(),
				"email":      user.Email(),
				"ip_address": ipAddress,
				"user_agent": userAgent,
			})
	}
}

// GetInvitation handles GET /invitations/{token}
func (h *UserInvitationHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Extract token from URL path
	// URL format: /api/v1/invitations/{token}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		WriteError(w, r.Context(), BadRequest("Invalid URL format"))
		return
	}
	token := pathParts[len(pathParts)-1] // The token is the last part

	invitation, err := h.userService.GetInvitationByToken(r.Context(), token)
	if err != nil {
		h.logger.Error("Failed to get invitation", zap.Error(err))

		switch err {
		case user.ErrInvitationNotFound:
			WriteError(w, r.Context(), NotFound("Invitation not found"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to get invitation").WithCause(err))
		}
		return
	}

	response := UserInvitationResponse{
		ID:        invitation.ID().String(),
		TenantID:  invitation.TenantID().String(),
		Email:     invitation.Email(),
		RoleID:    invitation.RoleID().String(),
		InvitedBy: invitation.InvitedBy().String(),
		Status:    string(invitation.Status()),
		ExpiresAt: invitation.ExpiresAt().Format("2006-01-02T15:04:05Z"),
		Message:   invitation.Message(),
		CreatedAt: invitation.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: invitation.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListInvitations handles GET /invitations
func (h *UserInvitationHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("ListInvitations called", zap.String("method", r.Method), zap.String("url", r.URL.String()))

	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	// Use tenant ID from auth context
	tenantID := userCtx.TenantID

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	invitations, err := h.userService.ListInvitationsByTenant(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to list invitations",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		WriteError(w, r.Context(), InternalServer("Failed to list invitations").WithCause(err))
		return
	}

	// Apply pagination
	total := len(invitations)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedInvitations := invitations[start:end]

	response := ListInvitationsResponse{
		Invitations: make([]UserInvitationResponse, len(paginatedInvitations)),
		Total:       total,
	}

	for i, inv := range paginatedInvitations {
		roleName := ""
		if h.rbacService != nil {
			if rbacSvc, ok := h.rbacService.(interface {
				GetRoleByID(ctx context.Context, id uuid.UUID) (*rbac.Role, error)
			}); ok {
				if role, err := rbacSvc.GetRoleByID(r.Context(), inv.RoleID()); err == nil && role != nil {
					roleName = role.Name()
				}
			}
		}

		response.Invitations[i] = UserInvitationResponse{
			ID:        inv.ID().String(),
			TenantID:  inv.TenantID().String(),
			Email:     inv.Email(),
			RoleID:    inv.RoleID().String(),
			RoleName:  roleName,
			InvitedBy: inv.InvitedBy().String(),
			Status:    string(inv.Status()),
			ExpiresAt: inv.ExpiresAt().Format("2006-01-02T15:04:05Z"),
			Message:   inv.Message(),
			CreatedAt: inv.CreatedAt().Format("2006-01-02T15:04:05Z"),
			UpdatedAt: inv.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CancelInvitation handles DELETE /invitations/{id}
func (h *UserInvitationHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	// Extract invitation ID from URL path
	// URL format: /api/v1/invitations/{id}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		WriteError(w, r.Context(), BadRequest("Invalid URL format"))
		return
	}
	invitationIDStr := pathParts[len(pathParts)-1] // The ID is the last part

	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid invitation ID"))
		return
	}

	err = h.userService.CancelInvitation(r.Context(), invitationID)
	if err != nil {
		h.logger.Error("Failed to cancel invitation",
			zap.String("invitation_id", invitationID.String()),
			zap.Error(err),
		)

		switch err {
		case user.ErrInvitationNotFound:
			WriteError(w, r.Context(), NotFound("Invitation not found"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to cancel invitation").WithCause(err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invitation cancelled successfully"})

	h.logger.Info("Invitation cancelled",
		zap.String("invitation_id", invitationID.String()),
	)

	// Audit invitation cancellation
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), userCtx.TenantID, userCtx.UserID,
			audit.AuditEventUserInviteCancel, "invitations", "cancel",
			"User invitation cancelled",
			map[string]interface{}{
				"invitation_id": invitationID.String(),
			})
	}
}

// ResendInvitation handles POST /invitations/{id}/resend
func (h *UserInvitationHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	// Extract invitation ID from URL path
	// URL format: /api/v1/invitations/{id}/resend
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 || pathParts[len(pathParts)-1] != "resend" {
		WriteError(w, r.Context(), BadRequest("Invalid URL format"))
		return
	}
	invitationIDStr := pathParts[len(pathParts)-2] // The ID is the second-to-last part

	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid invitation ID"))
		return
	}

	err = h.userService.ResendInvitation(r.Context(), invitationID)
	if err != nil {
		h.logger.Error("Failed to resend invitation",
			zap.String("invitation_id", invitationID.String()),
			zap.Error(err),
		)

		switch err {
		case user.ErrInvitationNotFound:
			WriteError(w, r.Context(), NotFound("Invitation not found"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to resend invitation").WithCause(err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invitation resent successfully"})

	h.logger.Info("Invitation resent",
		zap.String("invitation_id", invitationID.String()),
	)

	// Audit invitation resend
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), userCtx.TenantID, userCtx.UserID,
			audit.AuditEventUserInviteResend, "invitations", "resend",
			"User invitation resent",
			map[string]interface{}{
				"invitation_id": invitationID.String(),
			})
	}
}
