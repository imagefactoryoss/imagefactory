package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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

// roleExists checks if a role already exists in a list
func roleExists(roles []RoleResponse, roleID uuid.UUID) bool {
	for _, r := range roles {
		rid, err := uuid.Parse(r.ID)
		if err == nil && rid == roleID {
			return true
		}
	}
	return false
}

func roleAssignedTenantID(roleID uuid.UUID, assignmentsByTenant map[uuid.UUID][]uuid.UUID) (uuid.UUID, bool) {
	for tenantID, roleIDs := range assignmentsByTenant {
		for _, assignedRoleID := range roleIDs {
			if assignedRoleID == roleID {
				return tenantID, true
			}
		}
	}
	return uuid.Nil, false
}

func resolveRoleTenantID(role *rbac.Role, assignmentsByTenant map[uuid.UUID][]uuid.UUID) uuid.UUID {
	if role == nil {
		return uuid.Nil
	}
	if role.TenantID() != uuid.Nil {
		return role.TenantID()
	}
	if tenantID, ok := roleAssignedTenantID(role.ID(), assignmentsByTenant); ok {
		return tenantID
	}
	return uuid.Nil
}

func roleTypeSet(roleTypes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(roleTypes))
	for _, roleType := range roleTypes {
		key := normalizeRoleKey(roleType)
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return set
}

func isTenantRoleAssignable(roleName string, allowedRoleTypes map[string]struct{}) bool {
	if len(allowedRoleTypes) == 0 {
		return false
	}
	_, ok := allowedRoleTypes[normalizeRoleKey(roleName)]
	return ok
}

func (h *UserHandler) ensureRoleAssignableToTenant(ctx context.Context, tenantID, roleID uuid.UUID) error {
	assignableRoleTypes, err := h.rbacService.GetAssignableRoleTypesByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to resolve assignable role types: %w", err)
	}

	allowed := roleTypeSet(assignableRoleTypes)
	role, err := h.rbacService.GetRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to resolve role: %w", err)
	}

	if !isTenantRoleAssignable(role.Name(), allowed) {
		return fmt.Errorf("role %q is not assignable for tenant", role.Name())
	}
	return nil
}

// respondError sends a JSON error response
func (h *UserHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// UserHandler handles user management HTTP requests
type UserHandler struct {
	userService         *user.Service
	rbacService         *rbac.Service
	auditService        *audit.Service
	notificationService interface{} // Notification service for sending emails
	db                  *sqlx.DB    // Database connection for group operations
	config              *config.Config
	logger              *zap.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *user.Service, rbacService *rbac.Service, auditService *audit.Service, config *config.Config, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService:  userService,
		rbacService:  rbacService,
		auditService: auditService,
		config:       config,
		logger:       logger,
	}
}

// SetNotificationService sets the notification service for sending emails
func (h *UserHandler) SetNotificationService(notificationService interface{}) {
	h.notificationService = notificationService
}

// SetDatabase sets the database connection for group operations
func (h *UserHandler) SetDatabase(db *sqlx.DB) {
	h.db = db
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	TenantID        string                  `json:"tenant_id" validate:"omitempty,uuid"` // Deprecated, kept for backward compatibility
	TenantIDs       []string                `json:"tenant_ids,omitempty"`
	Email           string                  `json:"email" validate:"required,email"`
	FirstName       string                  `json:"first_name" validate:"required"`
	LastName        string                  `json:"last_name" validate:"required"`
	Password        string                  `json:"password" validate:"required,min=8"`
	RoleIDs         []string                `json:"role_ids,omitempty"` // Deprecated, kept for backward compatibility
	RoleAssignments []RoleAssignmentRequest `json:"role_assignments,omitempty"`
	Status          string                  `json:"status,omitempty"`
}

// CreateUserResponse represents a user creation response
type CreateUserResponse struct {
	User  UserResponse   `json:"user"`
	Roles []RoleResponse `json:"roles,omitempty"`
}

// ListUsersResponse represents a list of users response
type ListUsersResponse struct {
	Users []UserWithRolesResponse `json:"users"`
	Total int                     `json:"total"`
}

// UserWithRolesResponse represents user information with roles grouped by tenant
type UserWithRolesResponse struct {
	User          UserResponse              `json:"user"`
	Roles         []RoleResponse            `json:"roles"`                     // Backward compatibility: all roles across all tenants
	RolesByTenant map[string][]RoleResponse `json:"roles_by_tenant,omitempty"` // New: roles grouped by tenant ID
}

// RoleAssignmentRequest represents a role assignment for a specific tenant
type RoleAssignmentRequest struct {
	TenantID string `json:"tenant_id" validate:"required,uuid"`
	RoleID   string `json:"role_id" validate:"required,uuid"`
	// Also accept camelCase for frontend compatibility
	TenantIDCamel string `json:"tenantId"`
	RoleIDCamel   string `json:"roleId"`
}

// UnmarshalJSON allows the struct to accept both snake_case and camelCase
func (r *RoleAssignmentRequest) UnmarshalJSON(data []byte) error {
	type Alias RoleAssignmentRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Prefer snake_case, fall back to camelCase
	if r.TenantID == "" && r.TenantIDCamel != "" {
		r.TenantID = r.TenantIDCamel
	}
	if r.RoleID == "" && r.RoleIDCamel != "" {
		r.RoleID = r.RoleIDCamel
	}

	return nil
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	FirstName       string                  `json:"first_name,omitempty"`
	LastName        string                  `json:"last_name,omitempty"`
	Status          string                  `json:"status,omitempty"`
	TenantIDs       []string                `json:"tenant_ids,omitempty"`
	RoleAssignments []RoleAssignmentRequest `json:"role_assignments,omitempty"` // New: per-tenant roles
	RoleIDs         []string                `json:"role_ids,omitempty"`         // Deprecated: kept for backward compatibility
}

// AssignRoleRequest represents a role assignment request
type AssignRoleRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	RoleID string `json:"role_id" validate:"required,uuid"`
}

// CreateUser handles POST /users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create user request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Validate required fields
	// For backward compatibility: accept either TenantID (single) or TenantIDs (multiple)
	var tenantIDs []uuid.UUID
	if len(req.TenantIDs) > 0 {
		// New approach: multiple tenants
		for _, tidStr := range req.TenantIDs {
			tid, err := uuid.Parse(tidStr)
			if err != nil {
				h.logger.Warn("Invalid tenant ID in array", zap.String("tenant_id", tidStr))
				continue
			}
			tenantIDs = append(tenantIDs, tid)
		}
		if len(tenantIDs) == 0 {
			WriteError(w, r.Context(), BadRequest("At least one valid tenant ID is required"))
			return
		}
	} else if req.TenantID != "" {
		// Backward compatibility: single tenant ID
		tid, err := uuid.Parse(req.TenantID)
		if err != nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant ID").WithCause(err))
			return
		}
		tenantIDs = []uuid.UUID{tid}
	} else {
		h.logger.Warn("User creation attempt with no tenant ID")
		WriteError(w, r.Context(), BadRequest("Tenant ID is required"))
		return
	}

	if req.Email == "" {
		h.logger.Warn("User creation attempt with empty email")
		WriteError(w, r.Context(), BadRequest("Email is required"))
		return
	}
	if req.FirstName == "" {
		h.logger.Warn("User creation attempt with empty first name")
		WriteError(w, r.Context(), BadRequest("First name is required"))
		return
	}
	if req.LastName == "" {
		h.logger.Warn("User creation attempt with empty last name")
		WriteError(w, r.Context(), BadRequest("Last name is required"))
		return
	}
	if req.Password == "" {
		h.logger.Warn("User creation attempt with empty password")
		WriteError(w, r.Context(), BadRequest("Password is required"))
		return
	}

	// Basic email format validation
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		h.logger.Warn("User creation attempt with invalid email format", zap.String("email", req.Email))
		WriteError(w, r.Context(), ValidationErrorHTTP("Invalid email format"))
		return
	}

	// Password strength validation
	if len(req.Password) < 8 {
		h.logger.Warn("User creation attempt with weak password")
		WriteError(w, r.Context(), ValidationErrorHTTP("Password must be at least 8 characters long"))
		return
	}

	// Create user in the primary (first) tenant
	primaryTenantID := tenantIDs[0]
	usr, err := h.userService.CreateUser(r.Context(), primaryTenantID, req.Email, req.FirstName, req.LastName, req.Password)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))

		// Audit user creation failure
		if h.auditService != nil {
			h.auditService.LogSystemAction(r.Context(), primaryTenantID, audit.AuditEventUserCreate, "users", "create",
				"User creation failed", map[string]interface{}{
					"email":  req.Email,
					"reason": err.Error(),
				})
		}

		WriteError(w, r.Context(), InternalServer("Failed to create user").WithCause(err))
		return
	}

	// Assign roles if provided
	authCtx, _ := middleware.GetAuthContext(r)
	assignedBy := authCtx.UserID // Use authenticated user as assigner
	var assignedRoles []*rbac.Role

	// Handle per-tenant role assignments (new approach).
	// Note: an explicit empty array means "remove all assignments", so we must
	// branch on nil vs non-nil instead of len > 0.
	if req.RoleAssignments != nil {
		for _, assignment := range req.RoleAssignments {
			tenantID, err := uuid.Parse(assignment.TenantID)
			if err != nil {
				h.logger.Warn("Invalid tenant ID in role assignment", zap.String("tenant_id", assignment.TenantID))
				continue
			}

			roleID, err := uuid.Parse(assignment.RoleID)
			if err != nil {
				h.logger.Warn("Invalid role ID in role assignment", zap.String("role_id", assignment.RoleID))
				continue
			}

			if err := h.rbacService.AssignRoleToUserForTenant(r.Context(), usr.ID(), roleID, tenantID, assignedBy); err != nil {
				h.logger.Error("Failed to assign role to user for tenant",
					zap.String("user_id", usr.ID().String()),
					zap.String("role_id", roleID.String()),
					zap.String("tenant_id", tenantID.String()),
					zap.Error(err))
			}

			// Also add user to group for this role
			if err := h.addUserToGroupByRole(r.Context(), usr.ID(), roleID, tenantID); err != nil {
				h.logger.Warn("Failed to add user to group during creation",
					zap.String("user_id", usr.ID().String()),
					zap.String("role_id", roleID.String()),
					zap.Error(err))
				// Don't fail if group assignment fails
			}
		}
		// Get assigned roles
		assignedRoles, _ = h.rbacService.GetUserRoles(r.Context(), usr.ID())
	} else if len(req.RoleIDs) > 0 {
		// Backward compatibility: old approach with global roles
		for _, roleIDStr := range req.RoleIDs {
			roleID, err := uuid.Parse(roleIDStr)
			if err != nil {
				h.logger.Warn("Invalid role ID in request", zap.String("role_id", roleIDStr))
				continue
			}

			if err := h.rbacService.AssignRoleToUser(r.Context(), usr.ID(), roleID, assignedBy); err != nil {
				h.logger.Error("Failed to assign role to user",
					zap.String("user_id", usr.ID().String()),
					zap.String("role_id", roleID.String()),
					zap.Error(err))
			}

			// Check if this is a tenant-scoped role and add to group
			role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
			if err == nil && role.TenantID() != uuid.Nil {
				if err := h.addUserToGroupByRole(r.Context(), usr.ID(), roleID, role.TenantID()); err != nil {
					h.logger.Warn("Failed to add user to group during creation",
						zap.String("user_id", usr.ID().String()),
						zap.String("role_id", roleID.String()),
						zap.Error(err))
				}
			}
		}
		// Get assigned roles
		assignedRoles, _ = h.rbacService.GetUserRoles(r.Context(), usr.ID())
	}

	// Convert to response format
	userResp := UserResponse{
		ID:         usr.ID().String(),
		Email:      usr.Email(),
		FirstName:  usr.FirstName(),
		LastName:   usr.LastName(),
		Status:     string(usr.Status()),
		IsActive:   usr.IsActive(),
		AuthMethod: string(usr.AuthMethod()),
	}
	roleResponses := make([]RoleResponse, len(assignedRoles))
	for i, role := range assignedRoles {
		roleResponses[i] = RoleResponse{
			ID:          role.ID().String(),
			Name:        role.Name(),
			Description: role.Description(),
			IsSystem:    role.IsSystem(),
		}
	}

	response := CreateUserResponse{
		User:  userResp,
		Roles: roleResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User created successfully", zap.String("user_id", usr.ID().String()))

	// Audit successful user creation
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), primaryTenantID, usr.ID(),
			audit.AuditEventUserCreate, "users", "create", "User created successfully", map[string]interface{}{
				"email": req.Email,
			})
	}
}

// ListUsers handles GET /users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("ListUsers called", zap.String("path", r.URL.Path), zap.String("method", r.Method))
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Get authenticated user context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in ListUsers")
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	h.logger.Info("Auth context found", zap.String("user_id", authCtx.UserID.String()), zap.String("tenant_id", authCtx.TenantID.String()))

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	pageStr := r.URL.Query().Get("page")
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	tenantIDStr := r.URL.Query().Get("tenantId")
	roleStr := r.URL.Query().Get("role")

	limit := 20 // default
	offset := 0 // default
	page := 1

	// Parse pagination parameters
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
		offset = (page - 1) * limit
	} else if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	// Get users based on explicit tenant context.
	// Cross-tenant reads are only allowed when a system admin explicitly requests tenantId.
	var users []*user.User
	var err error
	var contextTenantID uuid.UUID // Track which tenant we're querying for

	if tenantIDStr != "" {
		// User is explicitly requesting users from a specific tenant
		// Validate they have access to that tenant
		filterTenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			h.logger.Error("Invalid tenant ID in query", zap.String("tenant_id", tenantIDStr))
			WriteError(w, r.Context(), BadRequest("Invalid tenant ID format"))
			return
		}

		// Check if user has access to this tenant
		// System admins can access any tenant
		// Tenant users can only access their own tenant
		if !authCtx.IsSystemAdmin && authCtx.TenantID != filterTenantID {
			h.logger.Warn("User attempted to access users from different tenant",
				zap.String("user_id", authCtx.UserID.String()),
				zap.String("user_tenant_id", authCtx.TenantID.String()),
				zap.String("requested_tenant_id", filterTenantID.String()))
			WriteError(w, r.Context(), Forbidden("Access denied to this tenant"))
			return
		}

		// Get users for the specified tenant
		h.logger.Info("Listing users for specified tenant", zap.String("tenant_id", filterTenantID.String()))
		users, err = h.userService.GetUsersByTenantID(r.Context(), filterTenantID)
		contextTenantID = filterTenantID // Track the tenant context for role filtering
	} else {
		if isAllTenantsScopeRequested(r, authCtx) {
			h.logger.Info("Listing users for all tenants (explicit system-admin scope)")
			users, err = h.userService.GetAllUsers(r.Context())
			contextTenantID = uuid.Nil
		} else {
			// Default to the authenticated tenant context for all users, including system admins.
			h.logger.Info("Listing users for tenant", zap.String("tenant_id", authCtx.TenantID.String()))
			users, err = h.userService.GetUsersByTenantID(r.Context(), authCtx.TenantID)
			contextTenantID = authCtx.TenantID // Track the tenant context for role filtering
		}
	}

	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to list users").WithCause(err))
		return
	}

	// Apply filters
	filteredUsers := users

	// Filter by search (email or name)
	if search != "" {
		searchLower := strings.ToLower(search)
		filtered := make([]*user.User, 0)
		for _, usr := range filteredUsers {
			if strings.Contains(strings.ToLower(usr.Email()), searchLower) ||
				strings.Contains(strings.ToLower(usr.FirstName()), searchLower) ||
				strings.Contains(strings.ToLower(usr.LastName()), searchLower) {
				filtered = append(filtered, usr)
			}
		}
		filteredUsers = filtered
	}

	// Filter by status
	if status != "" {
		filtered := make([]*user.User, 0)
		for _, usr := range filteredUsers {
			if string(usr.Status()) == status {
				filtered = append(filtered, usr)
			}
		}
		filteredUsers = filtered
	}

	// Filter by tenant (for non-system admins)
	if tenantIDStr != "" && !authCtx.IsSystemAdmin {
		// Only filter if the user is requesting a specific tenant and they're a tenant admin
		filterTenantID, err := uuid.Parse(tenantIDStr)
		if err == nil && filterTenantID == authCtx.TenantID {
			filtered := make([]*user.User, 0)
			for _, usr := range filteredUsers {
				// Check if user has any roles in the requested tenant
				allRoles, err := h.rbacService.GetUserRolesForTenant(r.Context(), usr.ID(), filterTenantID)
				if err == nil && len(allRoles) > 0 {
					filtered = append(filtered, usr)
				}
			}
			filteredUsers = filtered
		}
	}

	// Filter by role
	if roleStr != "" {
		filtered := make([]*user.User, 0)
		for _, usr := range filteredUsers {
			var allRoles []*rbac.Role
			var err error

			if contextTenantID == uuid.Nil {
				allRoles, err = h.rbacService.GetUserRoles(r.Context(), usr.ID())
			} else {
				allRoles, err = h.rbacService.GetUserRolesForTenant(r.Context(), usr.ID(), contextTenantID)
			}

			if err == nil {
				hasRole := false
				for _, role := range allRoles {
					if role.Name() == roleStr {
						hasRole = true
						break
					}
				}
				if hasRole {
					filtered = append(filtered, usr)
				}
			}
		}
		filteredUsers = filtered
	}

	// Get total count after filtering
	totalCount := len(filteredUsers)

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(filteredUsers) {
		start = len(filteredUsers)
	}
	if end > len(filteredUsers) {
		end = len(filteredUsers)
	}
	paginatedUsers := filteredUsers[start:end]

	// Extract user IDs for batch loading
	userIDs := make([]uuid.UUID, len(paginatedUsers))
	for i, usr := range paginatedUsers {
		userIDs[i] = usr.ID()
	}

	// Batch load all roles for all users (single query instead of N+1)
	userRolesMap, err := h.rbacService.GetUserRolesBatch(r.Context(), userIDs)
	if err != nil {
		h.logger.Error("Failed to batch load user roles", zap.Error(err))
		// Fallback to empty roles for all users
		userRolesMap = make(map[uuid.UUID][]*rbac.Role)
	}

	// Build response with user roles (both global and per-tenant)
	userResponses := make([]UserWithRolesResponse, len(paginatedUsers))
	for i, usr := range paginatedUsers {
		// Get roles from batch result
		allRoles := userRolesMap[usr.ID()]
		if allRoles == nil {
			allRoles = []*rbac.Role{} // Fallback to empty roles
		}
		assignmentsByTenant, assignmentErr := h.rbacService.GetUserRoleAssignmentsByTenant(r.Context(), usr.ID())
		if assignmentErr != nil {
			assignmentsByTenant = map[uuid.UUID][]uuid.UUID{}
			h.logger.Warn("Failed to load user role assignments by tenant for role grouping",
				zap.String("user_id", usr.ID().String()),
				zap.Error(assignmentErr))
		}

		// When listing a specific tenant, filter roles to only those for that tenant
		if contextTenantID != uuid.Nil {
			filteredRoles := make([]*rbac.Role, 0)
			for _, role := range allRoles {
				if role.TenantID() == contextTenantID {
					filteredRoles = append(filteredRoles, role)
				}
			}
			allRoles = filteredRoles
		}

		// Convert all roles to response format (for backward compatibility)
		roleResponses := make([]RoleResponse, len(allRoles))
		for j, role := range allRoles {
			roleResponses[j] = RoleResponse{
				ID:          role.ID().String(),
				Name:        role.Name(),
				Description: role.Description(),
				IsSystem:    role.IsSystem(),
			}
		}

		// Build roles grouped by tenant
		rolesByTenant := make(map[string][]RoleResponse)

		// Group roles by tenant (reuse the already converted roleResponses)
		for j, role := range allRoles {
			// Add to per-tenant list
			tenantIDStr := resolveRoleTenantID(role, assignmentsByTenant).String()
			if !roleExists(rolesByTenant[tenantIDStr], role.ID()) {
				rolesByTenant[tenantIDStr] = append(rolesByTenant[tenantIDStr], roleResponses[j])
			}
		}

		userResponses[i] = UserWithRolesResponse{
			User: UserResponse{
				ID:         usr.ID().String(),
				Email:      usr.Email(),
				FirstName:  usr.FirstName(),
				LastName:   usr.LastName(),
				Status:     string(usr.Status()),
				IsActive:   usr.IsActive(),
				AuthMethod: string(usr.AuthMethod()),
			},
			Roles:         roleResponses,
			RolesByTenant: rolesByTenant,
		}
	}

	response := ListUsersResponse{
		Users: userResponses,
		Total: totalCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CheckUserEmail handles GET /admin/users/check-email?email=...
func (h *UserHandler) CheckUserEmail(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, true)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		WriteError(w, r.Context(), BadRequest("Email parameter is required"))
		return
	}

	// Check if user exists by email
	user, err := h.userService.GetUserByEmail(r.Context(), email)
	if err != nil {
		// User not found is expected for available emails
		if err.Error() == "user not found" || err.Error() == "sql: no rows in result set" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"exists": false})
			return
		}

		h.logger.Error("Failed to check user email",
			zap.String("email", email),
			zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to check user"))
		return
	}

	// Prevent cross-tenant user enumeration unless explicit all-tenant scope is requested.
	if !allTenants {
		rolesForTenant, roleErr := h.rbacService.GetUserRolesForTenant(r.Context(), user.ID(), scopeTenantID)
		if roleErr != nil || len(rolesForTenant) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"exists": false})
			return
		}
	}

	// User exists
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists": true,
		"id":     user.ID().String(),
	})
}

// GetUser handles GET /users/{id}
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	_, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in GetUser")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userIDStr := chi.URLParam(r, "id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get user
	usr, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user", zap.String("user_id", userID.String()), zap.Error(err))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get all roles for this user (including group-based roles)
	allRoles, err := h.rbacService.GetUserRoles(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user roles", zap.String("user_id", userID.String()), zap.Error(err))
		allRoles = []*rbac.Role{}
	}

	// Convert to response format
	roleResponses := make([]RoleResponse, 0)
	rolesByTenant := make(map[string][]RoleResponse)
	assignmentsByTenant, assignmentErr := h.rbacService.GetUserRoleAssignmentsByTenant(r.Context(), userID)
	if assignmentErr != nil {
		assignmentsByTenant = map[uuid.UUID][]uuid.UUID{}
		h.logger.Warn("Failed to load user role assignments by tenant for role grouping",
			zap.String("user_id", userID.String()),
			zap.Error(assignmentErr))
	}

	// Group roles by tenant
	for _, role := range allRoles {
		roleResp := RoleResponse{
			ID:          role.ID().String(),
			Name:        role.Name(),
			Description: role.Description(),
			IsSystem:    role.IsSystem(),
		}

		// Add to global roles list (for backward compatibility)
		if !roleExists(roleResponses, role.ID()) {
			roleResponses = append(roleResponses, roleResp)
		}

		// Add to per-tenant list
		tenantIDStr := resolveRoleTenantID(role, assignmentsByTenant).String()
		if !roleExists(rolesByTenant[tenantIDStr], role.ID()) {
			rolesByTenant[tenantIDStr] = append(rolesByTenant[tenantIDStr], roleResp)
		}
	}

	response := UserWithRolesResponse{
		User: UserResponse{
			ID:         usr.ID().String(),
			Email:      usr.Email(),
			FirstName:  usr.FirstName(),
			LastName:   usr.LastName(),
			Status:     string(usr.Status()),
			IsActive:   usr.IsActive(),
			AuthMethod: string(usr.AuthMethod()),
		},
		Roles:         roleResponses,
		RolesByTenant: rolesByTenant,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateUser handles PUT /users/{id}
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in UpdateUser")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userIDStr := chi.URLParam(r, "id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update user request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user
	currentUser, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user for update", zap.String("user_id", userID.String()), zap.Error(err))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if user is LDAP-enabled - LDAP users cannot have certain fields modified
	isLDAPUser := currentUser.AuthMethod() == user.AuthMethodLDAP

	// Update user fields (with LDAP restrictions)
	if req.FirstName != "" {
		if isLDAPUser {
			h.logger.Warn("Cannot update first name for LDAP user - field is managed by LDAP directory",
				zap.String("user_id", userID.String()))
		} else {
			if err := currentUser.UpdateFirstName(req.FirstName); err != nil {
				h.logger.Warn("Failed to update first name",
					zap.String("user_id", userID.String()),
					zap.Error(err))
			}
		}
	}
	if req.LastName != "" {
		if isLDAPUser {
			h.logger.Warn("Cannot update last name for LDAP user - field is managed by LDAP directory",
				zap.String("user_id", userID.String()))
		} else {
			if err := currentUser.UpdateLastName(req.LastName); err != nil {
				h.logger.Warn("Failed to update last name",
					zap.String("user_id", userID.String()),
					zap.Error(err))
			}
		}
	}

	// Update user status if provided
	if req.Status != "" {
		// Validate the status is valid
		validStatuses := map[string]user.UserStatus{
			"active":    user.UserStatusActive,
			"pending":   user.UserStatusPending,
			"suspended": user.UserStatusSuspended,
			"disabled":  user.UserStatusDisabled,
			"locked":    user.UserStatusLocked,
		}
		if status, ok := validStatuses[strings.ToLower(req.Status)]; ok {
			if err := currentUser.UpdateStatus(status); err != nil {
				h.logger.Warn("Failed to update user status",
					zap.String("user_id", userID.String()),
					zap.Error(err))
			}
		} else {
			h.logger.Warn("Invalid status provided",
				zap.String("user_id", userID.String()),
				zap.String("status", req.Status))
		}
	}

	// Persist user changes (first name, last name, status) if any were made
	if req.FirstName != "" || req.LastName != "" || req.Status != "" {
		if err := h.userService.UpdateUser(r.Context(), currentUser); err != nil {
			h.logger.Error("Failed to persist user changes",
				zap.String("user_id", userID.String()),
				zap.Error(err))
			http.Error(w, "Failed to update user details", http.StatusInternalServerError)
			return
		}
	}

	authCtx, ok = middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle per-tenant role assignments (new approach).
	// Explicit empty array means clear assignments.
	if req.RoleAssignments != nil {
		h.logger.Info("Processing role assignments for user",
			zap.String("user_id", userID.String()),
			zap.Int("assignment_count", len(req.RoleAssignments)))

		// Compare current vs requested assignments to determine what actually changed
		// Get current assignments
		currentAssignments, err := h.rbacService.GetUserRoleAssignmentsByTenant(r.Context(), userID)
		if err != nil {
			h.logger.Error("Failed to get current role assignments", zap.String("user_id", userID.String()), zap.Error(err))
			currentAssignments = make(map[uuid.UUID][]uuid.UUID) // Empty map as fallback
		}

		// Build requested assignments map
		requestedAssignments := make(map[uuid.UUID]uuid.UUID) // tenantID -> roleID
		for _, assignment := range req.RoleAssignments {
			tenantID, err := uuid.Parse(assignment.TenantID)
			if err != nil {
				continue
			}
			roleID, err := uuid.Parse(assignment.RoleID)
			if err != nil {
				continue
			}
			requestedAssignments[tenantID] = roleID
		}

		// Determine what to remove, change, and add
		toRemove := make(map[uuid.UUID][]uuid.UUID) // tenantID -> []roleID
		toAssign := make(map[uuid.UUID]uuid.UUID)   // tenantID -> roleID
		notifications := make([]struct {
			tenantID    uuid.UUID
			oldRoleID   uuid.UUID
			newRoleID   uuid.UUID
			oldRoleName string
			newRoleName string
		}, 0)

		// Check current assignments
		for tenantID, currentRoleIDs := range currentAssignments {
			if len(currentRoleIDs) == 0 {
				continue
			}
			currentRoleID := currentRoleIDs[0] // Assume one role per tenant for now

			if requestedRoleID, exists := requestedAssignments[tenantID]; exists {
				// Tenant exists in both current and requested
				if requestedRoleID != currentRoleID {
					// Role changed - remove old, assign new
					toRemove[tenantID] = []uuid.UUID{currentRoleID}
					toAssign[tenantID] = requestedRoleID

					// Get role names for notification
					oldRoleName := "Unknown"
					newRoleName := "Unknown"
					if oldRole, err := h.rbacService.GetRoleByID(r.Context(), currentRoleID); err == nil {
						oldRoleName = oldRole.Name()
					}
					if newRole, err := h.rbacService.GetRoleByID(r.Context(), requestedRoleID); err == nil {
						newRoleName = newRole.Name()
					}

					notifications = append(notifications, struct {
						tenantID    uuid.UUID
						oldRoleID   uuid.UUID
						newRoleID   uuid.UUID
						oldRoleName string
						newRoleName string
					}{tenantID, currentRoleID, requestedRoleID, oldRoleName, newRoleName})
				}
				// If roles are the same, do nothing
			} else {
				// Tenant exists in current but not in requested - remove it
				toRemove[tenantID] = []uuid.UUID{currentRoleID}

				// Get role name for notification
				oldRoleName := "Unknown"
				if oldRole, err := h.rbacService.GetRoleByID(r.Context(), currentRoleID); err == nil {
					oldRoleName = oldRole.Name()
				}

				notifications = append(notifications, struct {
					tenantID    uuid.UUID
					oldRoleID   uuid.UUID
					newRoleID   uuid.UUID
					oldRoleName string
					newRoleName string
				}{tenantID, currentRoleID, uuid.Nil, oldRoleName, "None"})
			}
		}

		// Check for new assignments (tenants in requested but not in current)
		for tenantID, requestedRoleID := range requestedAssignments {
			if _, exists := currentAssignments[tenantID]; !exists {
				// New assignment
				toAssign[tenantID] = requestedRoleID

				// Get role name for notification
				newRoleName := "Unknown"
				if newRole, err := h.rbacService.GetRoleByID(r.Context(), requestedRoleID); err == nil {
					newRoleName = newRole.Name()
				}

				notifications = append(notifications, struct {
					tenantID    uuid.UUID
					oldRoleID   uuid.UUID
					newRoleID   uuid.UUID
					oldRoleName string
					newRoleName string
				}{tenantID, uuid.Nil, requestedRoleID, "None", newRoleName})
			}
		}

		// Execute removals
		for tenantID, roleIDs := range toRemove {
			for _, roleID := range roleIDs {
				if tenantID == uuid.Nil {
					if err := h.rbacService.RemoveRoleFromUser(r.Context(), userID, roleID); err != nil {
						h.logger.Error("Failed to remove system-wide role from user",
							zap.String("user_id", userID.String()),
							zap.String("role_id", roleID.String()),
							zap.Error(err))
					}
				} else {
					if err := h.rbacService.RemoveRoleFromUserForTenant(r.Context(), userID, roleID, tenantID); err != nil {
						h.logger.Error("Failed to remove tenant-specific role from user",
							zap.String("user_id", userID.String()),
							zap.String("role_id", roleID.String()),
							zap.String("tenant_id", tenantID.String()),
							zap.Error(err))
					}
					// Keep group-based role view in sync with RBAC assignment removals.
					if err := h.removeUserFromGroupByRole(r.Context(), userID, roleID, tenantID); err != nil {
						h.logger.Warn("Failed to remove user from group when removing tenant role assignment",
							zap.String("user_id", userID.String()),
							zap.String("role_id", roleID.String()),
							zap.String("tenant_id", tenantID.String()),
							zap.Error(err))
					}
				}
			}
		}

		// Execute assignments
		for tenantID, roleID := range toAssign {
			if err := h.rbacService.AssignRoleToUserForTenant(r.Context(), userID, roleID, tenantID, authCtx.UserID); err != nil {
				h.logger.Error("Failed to assign role to user for tenant",
					zap.String("user_id", userID.String()),
					zap.String("role_id", roleID.String()),
					zap.String("tenant_id", tenantID.String()),
					zap.Error(err))
			}

			// Also add user to group for this role
			if err := h.addUserToGroupByRole(r.Context(), userID, roleID, tenantID); err != nil {
				h.logger.Warn("Failed to add user to group when updating role assignment",
					zap.String("user_id", userID.String()),
					zap.String("role_id", roleID.String()),
					zap.Error(err))
			}
		}

		// Send notifications for actual changes
		for _, notif := range notifications {
			if notif.newRoleID == uuid.Nil {
				// Role removal
				h.sendUserRoleRemovedNotification(r.Context(), userID, notif.tenantID, notif.oldRoleName)
			} else {
				// Role change or assignment
				h.sendUserRoleChangedNotification(r.Context(), userID, notif.newRoleID, notif.tenantID, notif.oldRoleName)
			}
		}

		// Clean up group memberships for tenants that no longer have roles
		// This prevents stale group-based roles from appearing in FindUserRoles query
		tenantsWithRoles := make(map[uuid.UUID]bool)
		for tenantID := range requestedAssignments {
			tenantsWithRoles[tenantID] = true
		}
		if err := h.removeUserFromAllGroupsInExcludedTenants(r.Context(), userID, tenantsWithRoles); err != nil {
			h.logger.Warn("Failed to clean up group memberships for excluded tenants",
				zap.String("user_id", userID.String()),
				zap.Error(err))
		}
	} else if len(req.RoleIDs) > 0 {
		// Backward compatibility: old approach with global roles
		// Remove existing direct role assignments
		currentAssignments, err := h.rbacService.GetUserRoleAssignmentsByTenant(r.Context(), userID)
		if err != nil {
			h.logger.Error("Failed to get current role assignments", zap.String("user_id", userID.String()), zap.Error(err))
		} else {
			// Remove all direct role assignments (only system-wide ones for backward compatibility)
			for tenantID, roleIDs := range currentAssignments {
				for _, roleID := range roleIDs {
					if tenantID == uuid.Nil {
						// System-wide role assignment
						if err := h.rbacService.RemoveRoleFromUser(r.Context(), userID, roleID); err != nil {
							h.logger.Error("Failed to remove system-wide role from user",
								zap.String("user_id", userID.String()),
								zap.String("role_id", roleID.String()),
								zap.Error(err))
						}
					}
					// Note: For backward compatibility, we don't remove tenant-specific assignments here
					// The old API only dealt with system-wide roles
				}
			}
		}

		// Assign new roles (global, no tenant scoping)
		for _, roleIDStr := range req.RoleIDs {
			roleID, err := uuid.Parse(roleIDStr)
			if err != nil {
				h.logger.Warn("Invalid role ID in update request", zap.String("role_id", roleIDStr))
				continue
			}

			// Get the old role name before assignment
			oldRoleName := "None"
			// Note: For backward compatibility, we don't track old roles in the old API

			if err := h.rbacService.AssignRoleToUser(r.Context(), userID, roleID, authCtx.UserID); err != nil {
				h.logger.Error("Failed to assign role to user",
					zap.String("user_id", userID.String()),
					zap.String("role_id", roleID.String()),
					zap.Error(err))
			}

			// Send notification about role change (system-wide roles don't have tenant)
			h.sendUserRoleChangedNotification(r.Context(), userID, roleID, uuid.Nil, oldRoleName)
		}
	}

	// Get updated user and roles
	updatedUser, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get updated user", zap.String("user_id", userID.String()), zap.Error(err))
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	roles, err := h.rbacService.GetUserRoles(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get updated user roles", zap.String("user_id", userID.String()), zap.Error(err))
		roles = []*rbac.Role{}
	}

	roleResponses := make([]RoleResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = RoleResponse{
			ID:          role.ID().String(),
			Name:        role.Name(),
			Description: role.Description(),
			IsSystem:    role.IsSystem(),
		}
	}

	response := UserWithRolesResponse{
		User: UserResponse{
			ID:         updatedUser.ID().String(),
			Email:      updatedUser.Email(),
			FirstName:  updatedUser.FirstName(),
			LastName:   updatedUser.LastName(),
			Status:     string(updatedUser.Status()),
			IsActive:   updatedUser.IsActive(),
			AuthMethod: string(updatedUser.AuthMethod()),
		},
		Roles:         roleResponses,
		RolesByTenant: make(map[string][]RoleResponse),
	}

	// Build RolesByTenant map for per-tenant role assignments
	for _, assignment := range req.RoleAssignments {
		tenantID, err := uuid.Parse(assignment.TenantID)
		if err != nil {
			continue
		}
		roleID, err := uuid.Parse(assignment.RoleID)
		if err != nil {
			continue
		}

		// Find the role in our roles list
		for _, role := range roles {
			if role.ID() == roleID {
				response.RolesByTenant[tenantID.String()] = append(response.RolesByTenant[tenantID.String()], RoleResponse{
					ID:          role.ID().String(),
					Name:        role.Name(),
					Description: role.Description(),
					IsSystem:    role.IsSystem(),
				})
				break
			}
		}
	}

	// Send notification emails for role assignments if notification service is available
	if h.notificationService != nil && len(req.RoleAssignments) > 0 {
		h.logger.Info("Notification service available, sending role assignment notifications",
			zap.String("user_id", userID.String()),
			zap.Int("assignment_count", len(req.RoleAssignments)))
		go func(ns interface{}, uid uuid.UUID, assignments []RoleAssignmentRequest, logger *zap.Logger, db *sqlx.DB, userSvc *user.Service, rbacSvc *rbac.Service) {
			// Create a background context for the notification since the request context will be canceled
			ctx := context.Background()

			logger.Info("Starting role assignment notification goroutine",
				zap.String("user_id", uid.String()),
				zap.Int("assignment_count", len(assignments)))

			// Fetch user details for notification
			user, err := userSvc.GetUserByID(ctx, uid)
			if err != nil {
				logger.Warn("Failed to fetch user for notification", zap.Error(err))
				return
			}

			// Send notification for each role assignment
			for _, assignment := range assignments {
				tenantID, err := uuid.Parse(assignment.TenantID)
				if err != nil {
					logger.Warn("Invalid tenant ID in notification", zap.String("tenant_id", assignment.TenantID))
					continue
				}

				roleID, err := uuid.Parse(assignment.RoleID)
				if err != nil {
					logger.Warn("Invalid role ID in notification", zap.String("role_id", assignment.RoleID))
					continue
				}

				// Fetch tenant name
				tenantName := "Tenant" // fallback
				if db != nil {
					var name string
					err := db.GetContext(ctx, &name, "SELECT name FROM tenants WHERE id = $1", tenantID.String())
					if err == nil {
						tenantName = name
					} else {
						logger.Warn("Failed to fetch tenant name for notification", zap.Error(err))
					}
				}

				// Fetch role name
				roleName := "Role" // fallback
				if role, err := rbacSvc.GetRoleByID(ctx, roleID); err == nil {
					roleName = role.Name()
				}

				// Create notification data for role assignment
				notifData := &email.UserAddedToTenantData{
					UserEmail:    user.Email(),
					UserName:     user.FullName(),
					TenantName:   tenantName,
					TenantID:     tenantID,
					Role:         roleName,
					DashboardURL: h.config.Frontend.DashboardURL,
				}

				// Send the notification email
				if notifSvc, ok := ns.(*email.NotificationService); ok {
					if err := notifSvc.SendUserAddedToTenantEmail(ctx, notifData); err != nil {
						logger.Error("Failed to send user role assignment notification",
							zap.String("user_email", user.Email()),
							zap.String("tenant_id", tenantID.String()),
							zap.String("role", roleName),
							zap.Error(err))
					} else {
						logger.Info("User role assignment notification sent",
							zap.String("user_id", uid.String()),
							zap.String("user_email", user.Email()),
							zap.String("tenant_id", tenantID.String()),
							zap.String("role", roleName))
					}
				}
			}
		}(h.notificationService, userID, req.RoleAssignments, h.logger, h.db, h.userService, h.rbacService)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User updated successfully", zap.String("user_id", userID.String()))
}

// DeleteUser handles DELETE /users/{id}
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	_, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in DeleteUser")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userIDStr := chi.URLParam(r, "id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Delete user
	if err := h.userService.DeleteUser(r.Context(), userID); err != nil {
		h.logger.Error("Failed to delete user", zap.String("user_id", userID.String()), zap.Error(err))
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "User deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User deleted successfully", zap.String("user_id", userID.String()))
}

// AssignRoleToUser handles POST /users/{id}/roles
func (h *UserHandler) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	userIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	// Remove /roles suffix if present
	userIDStr = strings.TrimSuffix(userIDStr, "/roles")
	if userIDStr == "" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode assign role request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Override user ID from URL
	req.UserID = userIDStr

	// Parse role ID
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	// Get authenticated user as assigner
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Assign role
	if err := h.rbacService.AssignRoleToUser(r.Context(), userID, roleID, authCtx.UserID); err != nil {
		h.logger.Error("Failed to assign role to user",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.Error(err))
		http.Error(w, "Failed to assign role", http.StatusInternalServerError)
		return
	}

	// Get role details to check if it's tenant-scoped and add to group
	role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
	if err == nil && role.TenantID() != uuid.Nil {
		// This is a tenant-scoped role, add user to corresponding group
		if err := h.addUserToGroupByRole(r.Context(), userID, roleID, role.TenantID()); err != nil {
			h.logger.Warn("Failed to add user to group when assigning role",
				zap.String("user_id", userID.String()),
				zap.String("role_id", roleID.String()),
				zap.Error(err))
			// Don't fail the entire operation if group assignment fails
		}
	}

	response := map[string]string{
		"message": "Role assigned successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Role assigned to user successfully",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()))
}

// RemoveRoleFromUser handles DELETE /users/{id}/roles/{roleId}
func (h *UserHandler) RemoveRoleFromUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID and role ID from URL
	// URL format: /api/v1/users/{userId}/roles/{roleId}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	parts := strings.Split(path, "/roles/")
	if len(parts) != 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	userIDStr := parts[0]
	roleIDStr := parts[1]

	if userIDStr == "" || roleIDStr == "" {
		http.Error(w, "User ID and Role ID are required", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	// Get role details to check if it's tenant-scoped
	role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
	var tenantID uuid.UUID
	isTenantScoped := false
	if err == nil && role.TenantID() != uuid.Nil {
		tenantID = role.TenantID()
		isTenantScoped = true
	}

	// Remove role
	if err := h.rbacService.RemoveRoleFromUser(r.Context(), userID, roleID); err != nil {
		h.logger.Error("Failed to remove role from user",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.Error(err))
		http.Error(w, "Failed to remove role", http.StatusInternalServerError)
		return
	}

	// If this was a tenant-scoped role, also remove user from group
	if isTenantScoped {
		if err := h.removeUserFromGroupByRole(r.Context(), userID, roleID, tenantID); err != nil {
			h.logger.Warn("Failed to remove user from group when removing role",
				zap.String("user_id", userID.String()),
				zap.String("role_id", roleID.String()),
				zap.Error(err))
			// Don't fail the entire operation if group removal fails
		}
	}

	response := map[string]string{
		"message": "Role removed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Role removed from user successfully",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()))
}

// SuspendUser suspends a user account
func (h *UserHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in SuspendUser")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Use explicit tenant context; fall back to user's primary tenant when needed.
	tenantID := authCtx.TenantID
	if tenantID == uuid.Nil {
		tenantID = authCtx.PrimaryTenant()
	}
	if tenantID == uuid.Nil {
		h.respondError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	err = h.userService.SuspendUser(r.Context(), userID, tenantID)
	if err != nil {
		h.logger.Error("Failed to suspend user", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to suspend user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User suspended successfully"})

	h.logger.Info("User suspended successfully", zap.String("user_id", userID.String()))
}

// ActivateUser activates a suspended user account
func (h *UserHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in ActivateUser")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Use explicit tenant context; fall back to user's primary tenant when needed.
	tenantID := authCtx.TenantID
	if tenantID == uuid.Nil {
		tenantID = authCtx.PrimaryTenant()
	}
	if tenantID == uuid.Nil {
		h.respondError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	h.logger.Info("Activating user with tenant ID", zap.String("user_id", userID.String()), zap.String("tenant_id", tenantID.String()))

	err = h.userService.ActivateUser(r.Context(), userID, tenantID)
	if err != nil {
		h.logger.Error("Failed to activate user", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to activate user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User activated successfully"})

	h.logger.Info("User activated successfully", zap.String("user_id", userID.String()))
}

// GetUserActivity retrieves user activity/audit logs
func (h *UserHandler) GetUserActivity(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	activities, err := h.auditService.GetUserActivity(r.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user activity", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get user activity")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(activities)
}

// GetLoginHistory retrieves user login history
func (h *UserHandler) GetLoginHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	history, err := h.auditService.GetLoginHistory(r.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get login history", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get login history")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(history)
}

// GetUserSessions retrieves active user sessions
func (h *UserHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	sessions, err := h.auditService.GetUserSessions(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user sessions", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get user sessions")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sessions)
}

// UpdateUserRoles updates all roles for a user (bulk operation)
func (h *UserHandler) UpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		RoleIDs []string `json:"roleIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.RoleIDs) == 0 {
		h.respondError(w, http.StatusBadRequest, "At least one role ID is required")
		return
	}

	// Get authenticated user
	authCtx, _ := middleware.GetAuthContext(r)

	// Get current roles
	currentRoles, err := h.rbacService.GetUserRoles(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get current roles", zap.String("user_id", userID.String()), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get current roles")
		return
	}

	// Remove old roles and their group memberships
	for _, role := range currentRoles {
		if err := h.rbacService.RemoveRoleFromUser(r.Context(), userID, role.ID()); err != nil {
			h.logger.Error("Failed to remove role from user",
				zap.String("user_id", userID.String()),
				zap.String("role_id", role.ID().String()),
				zap.Error(err))
		}

		// If this was a tenant-scoped role, also remove from group
		if role.TenantID() != uuid.Nil {
			if err := h.removeUserFromGroupByRole(r.Context(), userID, role.ID(), role.TenantID()); err != nil {
				h.logger.Warn("Failed to remove user from group",
					zap.String("user_id", userID.String()),
					zap.String("role_id", role.ID().String()),
					zap.Error(err))
			}
		}
	}

	// Assign new roles and add to groups
	newRoleIDs := make([]uuid.UUID, 0, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		if id, err := uuid.Parse(roleID); err == nil {
			newRoleIDs = append(newRoleIDs, id)
		}
	}

	for _, roleID := range newRoleIDs {
		if err := h.rbacService.AssignRoleToUser(r.Context(), userID, roleID, authCtx.UserID); err != nil {
			h.logger.Error("Failed to assign role to user",
				zap.String("user_id", userID.String()),
				zap.String("role_id", roleID.String()),
				zap.Error(err))
		}

		// Check if this is a tenant-scoped role and add to group
		role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
		if err == nil && role.TenantID() != uuid.Nil {
			if err := h.addUserToGroupByRole(r.Context(), userID, roleID, role.TenantID()); err != nil {
				h.logger.Warn("Failed to add user to group",
					zap.String("user_id", userID.String()),
					zap.String("role_id", roleID.String()),
					zap.Error(err))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User roles updated successfully"})

	h.logger.Info("User roles updated successfully",
		zap.String("user_id", userID.String()),
		zap.Int("role_count", len(newRoleIDs)))
}

// GetTenantUsers retrieves all users in a tenant
func (h *UserHandler) GetTenantUsers(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, err := h.userService.GetTenantUsers(r.Context(), tenantID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get tenant users", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get tenant users")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)
}

// AddUserToTenant adds a user to a tenant
func (h *UserHandler) AddUserToTenant(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req struct {
		UserID  string   `json:"userId"`
		RoleIDs []string `json:"roleIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	h.logger.Info("AddUserToTenant request",
		zap.String("tenant_id", tenantID.String()),
		zap.String("user_id_raw", req.UserID),
		zap.Strings("role_ids_raw", req.RoleIDs))

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.String("user_id", req.UserID), zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	roleIDs := make([]uuid.UUID, 0, len(req.RoleIDs))
	var roleName string

	// Get all available roles for role name lookups
	var allRoles []*rbac.Role
	systemRoles, err := h.rbacService.GetAllSystemLevelRoles(r.Context())
	if err != nil {
		h.logger.Warn("Failed to get system roles for lookup", zap.Error(err))
		// Continue anyway, we can still process UUIDs
	} else {
		allRoles = systemRoles
	}

	for _, roleIDStr := range req.RoleIDs {
		h.logger.Info("Processing role", zap.String("role_id_str", roleIDStr))
		// Try to parse as UUID first
		if id, err := uuid.Parse(roleIDStr); err == nil {
			roleIDs = append(roleIDs, id)
			h.logger.Info("Parsed role as UUID", zap.String("role_id", id.String()))
			// Get the first role name for notification
			if roleName == "" {
				if role, err := h.rbacService.GetRoleByID(r.Context(), id); err == nil {
					roleName = role.Name()
				}
			}
		} else {
			// Not a UUID, try to look up by name
			h.logger.Info("Role ID is not a UUID, trying to lookup by name",
				zap.String("role_name", roleIDStr),
				zap.String("tenant_id", tenantID.String()))

			// Find role by name (case-insensitive)
			var foundRole *rbac.Role
			for _, role := range allRoles {
				h.logger.Info("Checking role", zap.String("role_name", role.Name()))
				if strings.EqualFold(role.Name(), roleIDStr) {
					foundRole = role
					break
				}
			}

			if foundRole != nil {
				roleIDs = append(roleIDs, foundRole.ID())
				h.logger.Info("Found role by name", zap.String("role_name", foundRole.Name()), zap.String("role_id", foundRole.ID().String()))
				if roleName == "" {
					roleName = foundRole.Name()
				}
			} else {
				h.logger.Warn("Role not found by name or ID",
					zap.String("role_id", roleIDStr),
					zap.String("tenant_id", tenantID.String()))
				// Continue without this role
			}
		}
	}

	h.logger.Info("Final role processing complete", zap.Int("role_count", len(roleIDs)))

	for _, roleID := range roleIDs {
		if err := h.ensureRoleAssignableToTenant(r.Context(), tenantID, roleID); err != nil {
			h.logger.Warn("Rejected non-assignable role for tenant user add",
				zap.String("tenant_id", tenantID.String()),
				zap.String("role_id", roleID.String()),
				zap.Error(err))
			h.respondError(w, http.StatusBadRequest, "Selected role is not assignable for this tenant")
			return
		}
	}

	// First check if user can be added (not suspended, exists, etc.)
	err = h.userService.AddUserToTenant(r.Context(), userID, tenantID, roleIDs)
	if err != nil {
		h.logger.Error("Failed to validate user for tenant", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to add user to tenant")
		return
	}

	// Get the current user making the request (for audit purposes)
	authCtx, _ := middleware.GetAuthContext(r)
	assignedBy := authCtx.UserID

	// Assign roles to the user in the tenant
	for _, roleID := range roleIDs {
		if err := h.rbacService.AssignRoleToUserForTenant(r.Context(), userID, roleID, tenantID, assignedBy); err != nil {
			h.logger.Error("Failed to assign role to user",
				zap.Error(err),
				zap.String("user_id", userID.String()),
				zap.String("role_id", roleID.String()),
				zap.String("tenant_id", tenantID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to assign role to user")
			return
		}

		// Also add user to the corresponding group for this role
		if err := h.addUserToGroupByRole(r.Context(), userID, roleID, tenantID); err != nil {
			h.logger.Warn("Failed to add user to group",
				zap.Error(err),
				zap.String("user_id", userID.String()),
				zap.String("role_id", roleID.String()))
			// Don't fail the whole operation if group assignment fails
		}
	}

	// Send notification email if notification service is available
	if h.notificationService != nil {
		go func(ns interface{}, uid uuid.UUID, tid uuid.UUID, role string, logger *zap.Logger, db *sqlx.DB) {
			// Create a background context for the notification since the request context will be canceled
			ctx := context.Background()

			// Fetch user details for notification
			user, err := h.userService.GetUserByID(ctx, uid)
			if err != nil {
				logger.Warn("Failed to fetch user for notification", zap.Error(err))
				return
			}

			// Fetch tenant name
			tenantName := "Tenant" // fallback
			if db != nil {
				var name string
				err := db.GetContext(ctx, &name, "SELECT name FROM tenants WHERE id = $1", tid.String())
				if err == nil {
					tenantName = name
				} else {
					logger.Warn("Failed to fetch tenant name for notification", zap.Error(err))
				}
			}

			// Create notification data with basic info
			notifData := &email.UserAddedToTenantData{
				UserEmail:    user.Email(),
				UserName:     user.FullName(),
				TenantName:   tenantName,
				TenantID:     tid,
				Role:         role,
				DashboardURL: h.config.Frontend.DashboardURL,
			}

			// Send the notification email
			if notifSvc, ok := ns.(*email.NotificationService); ok {
				if err := notifSvc.SendUserAddedToTenantEmail(ctx, notifData); err != nil {
					logger.Error("Failed to send user added to tenant notification",
						zap.String("user_email", user.Email()),
						zap.String("tenant_id", tid.String()),
						zap.Error(err))
				} else {
					logger.Info("User added to tenant notification sent",
						zap.String("user_id", uid.String()),
						zap.String("user_email", user.Email()),
						zap.String("tenant_id", tid.String()))
				}
			}
		}(h.notificationService, userID, tenantID, roleName, h.logger, h.db)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User added to tenant successfully"})

	h.logger.Info("User added to tenant successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.String("user_id", userID.String()))
}

// RemoveUserFromTenant removes a user from a tenant
func (h *UserHandler) RemoveUserFromTenant(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Fetch user details before removal for notification
	user, err := h.userService.GetUserByID(r.Context(), userID)
	var userName string
	if err == nil {
		userName = user.FirstName() + " " + user.LastName()
	}

	err = h.userService.RemoveUserFromTenant(r.Context(), userID, tenantID)
	if err != nil {
		h.logger.Error("Failed to remove user from tenant", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to remove user from tenant")
		return
	}

	// Remove all role assignments for this user in the tenant
	if h.rbacService != nil {
		userRoles, err := h.rbacService.GetUserRolesForTenant(r.Context(), userID, tenantID)
		if err != nil {
			h.logger.Warn("Failed to get user roles for tenant",
				zap.String("user_id", userID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
		} else {
			// Remove each role assignment and clean up group membership
			for _, role := range userRoles {
				if err := h.rbacService.RemoveRoleFromUserForTenant(r.Context(), userID, role.ID(), tenantID); err != nil {
					h.logger.Warn("Failed to remove role from user",
						zap.String("user_id", userID.String()),
						zap.String("role_id", role.ID().String()),
						zap.String("tenant_id", tenantID.String()),
						zap.Error(err))
				} else {
					h.logger.Info("Role removed from user",
						zap.String("user_id", userID.String()),
						zap.String("role_id", role.ID().String()),
						zap.String("tenant_id", tenantID.String()))
				}

				// Also remove from group
				if err := h.removeUserFromGroupByRole(r.Context(), userID, role.ID(), tenantID); err != nil {
					h.logger.Warn("Failed to remove user from group when removing from tenant",
						zap.String("user_id", userID.String()),
						zap.String("role_id", role.ID().String()),
						zap.String("tenant_id", tenantID.String()),
						zap.Error(err))
				}
			}
		}
	}

	// Send notification email if notification service is available
	if h.notificationService != nil && user != nil {
		go func(ns interface{}, uid uuid.UUID, tid uuid.UUID, uname string, uemail string, logger *zap.Logger) {
			// Create a background context for the notification since the request context will be canceled
			ctx := context.Background()

			// Create notification data
			notifData := &email.UserRemovedFromTenantData{
				UserEmail:  uemail,
				UserName:   uname,
				TenantName: "Tenant", // This will be resolved by the email service from templates
				TenantID:   tid,
			}

			// Send the notification email
			if notifSvc, ok := ns.(*email.NotificationService); ok {
				if err := notifSvc.SendUserRemovedFromTenantEmail(ctx, notifData); err != nil {
					logger.Warn("Failed to send user removed from tenant notification",
						zap.String("user_email", uemail),
						zap.String("tenant_id", tid.String()),
						zap.Error(err))
				} else {
					logger.Info("User removed from tenant notification sent",
						zap.String("user_id", uid.String()),
						zap.String("user_email", uemail),
						zap.String("tenant_id", tid.String()))
				}
			}
		}(h.notificationService, userID, tenantID, userName, user.Email(), h.logger)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User removed from tenant successfully"})

	h.logger.Info("User removed from tenant successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.String("user_id", userID.String()))
}

// addUserToGroupByRole adds a user to the group corresponding to a role in a tenant
func (h *UserHandler) addUserToGroupByRole(ctx context.Context, userID, roleID, tenantID uuid.UUID) error {
	if h.db == nil {
		h.logger.Warn("Database not available for adding user to group")
		return nil // Don't fail if DB is not available
	}

	// Get the role to determine its name
	role, err := h.rbacService.GetRoleByID(ctx, roleID)
	if err != nil {
		h.logger.Warn("Failed to get role for group assignment",
			zap.String("role_id", roleID.String()),
			zap.Error(err))
		return nil // Don't fail if we can't get the role
	}

	// Query for the group with matching role_type and tenant_id
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
			return nil // Group doesn't exist, skip adding to group
		}
		h.logger.Warn("Failed to find group for role",
			zap.String("role_name", role.Name()),
			zap.Error(err))
		return nil
	}

	// Add user to the group
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
		return nil // Don't fail if we can't add to group
	}

	h.logger.Info("Added user to group",
		zap.String("group_id", groupID.String()),
		zap.String("user_id", userID.String()),
		zap.String("role_name", role.Name()))
	return nil
}

// removeUserFromGroupByRole removes a user from the group corresponding to a role in a tenant
func (h *UserHandler) removeUserFromGroupByRole(ctx context.Context, userID, roleID, tenantID uuid.UUID) error {
	if h.db == nil {
		h.logger.Warn("Database not available for removing user from group")
		return nil // Don't fail if DB is not available
	}

	// Get the role to determine its name
	role, err := h.rbacService.GetRoleByID(ctx, roleID)
	if err != nil {
		h.logger.Warn("Failed to get role for group removal",
			zap.String("role_id", roleID.String()),
			zap.Error(err))
		return nil // Don't fail if we can't get the role
	}

	// Query for the group with matching role_type and tenant_id
	query := `
		SELECT id FROM tenant_groups 
		WHERE tenant_id = $1 AND role_type = $2 AND status = 'active'
		LIMIT 1
	`

	var groupID uuid.UUID
	err = h.db.QueryRowContext(ctx, query, tenantID, strings.ToLower(strings.ReplaceAll(role.Name(), " ", "_"))).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("Group not found for role during removal",
				zap.String("tenant_id", tenantID.String()),
				zap.String("role_name", role.Name()))
			return nil // Group doesn't exist, skip removal
		}
		h.logger.Warn("Failed to find group for role during removal",
			zap.String("role_name", role.Name()),
			zap.Error(err))
		return nil
	}

	// Soft delete: set removed_at timestamp
	removeQuery := `
		UPDATE group_members 
		SET removed_at = CURRENT_TIMESTAMP 
		WHERE group_id = $1 AND user_id = $2 AND removed_at IS NULL
	`

	if _, err := h.db.ExecContext(ctx, removeQuery, groupID, userID); err != nil {
		h.logger.Warn("Failed to remove user from group",
			zap.String("group_id", groupID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil // Don't fail if we can't remove from group
	}

	h.logger.Info("Removed user from group",
		zap.String("group_id", groupID.String()),
		zap.String("user_id", userID.String()),
		zap.String("role_name", role.Name()))
	return nil
}

// removeUserFromAllGroupsInExcludedTenants removes user from all groups in tenants NOT in the assigned list
// This prevents stale group-based roles from appearing when roles are removed
func (h *UserHandler) removeUserFromAllGroupsInExcludedTenants(ctx context.Context, userID uuid.UUID, assignedTenants map[uuid.UUID]bool) error {
	// Get all group memberships for this user
	query := `
		SELECT gm.id, gm.group_id, tg.tenant_id
		FROM group_members gm
		INNER JOIN tenant_groups tg ON gm.group_id = tg.id
		WHERE gm.user_id = $1 AND gm.removed_at IS NULL
	`

	rows, err := h.db.QueryContext(ctx, query, userID)
	if err != nil {
		h.logger.Warn("Failed to find group memberships for cleanup",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil // Don't fail on this, it's just cleanup
	}
	defer rows.Close()

	var toRemove []struct {
		groupMemberId string
		groupId       string
		tenantId      uuid.UUID
	}

	for rows.Next() {
		var groupMemberId, groupId string
		var tenantId uuid.UUID
		if err := rows.Scan(&groupMemberId, &groupId, &tenantId); err != nil {
			continue
		}
		// Only mark for removal if tenant is NOT in assigned list
		if !assignedTenants[tenantId] {
			toRemove = append(toRemove, struct {
				groupMemberId string
				groupId       string
				tenantId      uuid.UUID
			}{groupMemberId, groupId, tenantId})
		}
	}

	// Remove from groups in excluded tenants
	for _, item := range toRemove {
		removeQuery := `
			UPDATE group_members 
			SET removed_at = CURRENT_TIMESTAMP 
			WHERE id = $1 AND user_id = $2 AND removed_at IS NULL
		`
		if _, err := h.db.ExecContext(ctx, removeQuery, item.groupMemberId, userID); err != nil {
			h.logger.Warn("Failed to remove user from group in excluded tenant",
				zap.String("user_id", userID.String()),
				zap.String("group_id", item.groupId),
				zap.String("tenant_id", item.tenantId.String()),
				zap.Error(err))
			continue // Keep trying to remove from other groups
		}
		h.logger.Info("Cleaned up group membership in excluded tenant",
			zap.String("user_id", userID.String()),
			zap.String("group_id", item.groupId),
			zap.String("tenant_id", item.tenantId.String()))
	}

	return nil
}

// UpdateTenantUserRole updates a user's role assignment in a tenant
func (h *UserHandler) UpdateTenantUserRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		RoleID string `json:"roleId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RoleID == "" {
		h.logger.Error("Empty role ID in request",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()))
		h.respondError(w, http.StatusBadRequest, "Role ID is required")
		return
	}

	// Try to parse as UUID first
	var roleID uuid.UUID
	parsedID, err := uuid.Parse(req.RoleID)
	if err != nil {
		// Not a UUID, try to look up by role name (case-insensitive)
		h.logger.Info("Role ID is not a UUID, trying to lookup by name",
			zap.String("role_name", req.RoleID),
			zap.String("tenant_id", tenantID.String()))

		// Get all system-level roles (both system roles and tenant-level roles)
		// This matches what the roles API endpoint does for availability
		roles, err := h.rbacService.GetAllSystemLevelRoles(r.Context())
		if err != nil {
			h.logger.Error("Failed to get system-level roles",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			h.respondError(w, http.StatusBadRequest, "Failed to get available roles")
			return
		}

		h.logger.Info("Found roles",
			zap.String("tenant_id", tenantID.String()),
			zap.Int("role_count", len(roles)))

		// Find role by name (case-insensitive)
		var foundRole *rbac.Role
		for _, role := range roles {
			h.logger.Info("Checking role",
				zap.String("role_name", role.Name()),
				zap.String("looking_for", req.RoleID),
				zap.Bool("match", strings.EqualFold(role.Name(), req.RoleID)))

			if strings.EqualFold(role.Name(), req.RoleID) {
				foundRole = role
				break
			}
		}

		if foundRole == nil {
			h.logger.Error("Role not found by name",
				zap.String("role_name", req.RoleID),
				zap.String("tenant_id", tenantID.String()),
				zap.Int("available_roles", len(roles)))
			h.respondError(w, http.StatusBadRequest, "Invalid role name or ID")
			return
		}
		roleID = foundRole.ID()
	} else {
		roleID = parsedID
	}

	// Get the current user making the request (for audit purposes)
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Prevent tenant owners from editing their own role unless there's another owner
	if authCtx.UserID == userID {
		// Check if the current user is a tenant owner
		currentUserRoles, err := h.rbacService.GetUserRolesForTenant(r.Context(), authCtx.UserID, tenantID)
		if err != nil && err.Error() != "no roles found" {
			h.logger.Error("Failed to get current user roles for tenant", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, "Failed to validate permissions")
			return
		}

		isOwner := false
		for _, role := range currentUserRoles {
			if role.Name() == "owner" {
				isOwner = true
				break
			}
		}

		if isOwner {
			// Check if there are other owners in the tenant
			tenantUsers, err := h.userService.GetTenantUsers(r.Context(), tenantID, 1000, 0) // Get all users
			if err != nil {
				h.logger.Error("Failed to get tenant users", zap.Error(err))
				h.respondError(w, http.StatusInternalServerError, "Failed to validate tenant ownership")
				return
			}

			otherOwners := 0
			for _, tenantUser := range tenantUsers {
				if tenantUser.ID() != authCtx.UserID {
					// Check if this user has owner role
					userRoles, err := h.rbacService.GetUserRolesForTenant(r.Context(), tenantUser.ID(), tenantID)
					if err == nil {
						for _, role := range userRoles {
							if role.Name() == "owner" {
								otherOwners++
								break
							}
						}
					}
				}
			}

			if otherOwners == 0 {
				h.respondError(w, http.StatusForbidden, "Cannot modify your own role as the only tenant owner. Please assign another user as owner first.")
				return
			}
		}
	}

	if err := h.ensureRoleAssignableToTenant(r.Context(), tenantID, roleID); err != nil {
		h.logger.Warn("Rejected non-assignable role update for tenant",
			zap.String("tenant_id", tenantID.String()),
			zap.String("role_id", roleID.String()),
			zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Selected role is not assignable for this tenant")
		return
	}

	// Get all current roles for the user in this tenant
	currentRoles, err := h.rbacService.GetUserRolesForTenant(r.Context(), userID, tenantID)
	if err != nil && err.Error() != "no roles found" {
		h.logger.Error("Failed to get user roles for tenant", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get user roles")
		return
	}

	// Remove existing roles (only if there are any)
	for _, role := range currentRoles {
		if err := h.rbacService.RemoveRoleFromUserForTenant(r.Context(), userID, role.ID(), tenantID); err != nil {
			// Log the error but don't fail - the role may have already been removed
			h.logger.Warn("Could not remove existing role (may not exist or already removed)",
				zap.String("user_id", userID.String()),
				zap.String("role_id", role.ID().String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			// Continue - we still want to assign the new role
		}

		// Also remove user from the group for this role
		if err := h.removeUserFromGroupByRole(r.Context(), userID, role.ID(), tenantID); err != nil {
			h.logger.Warn("Could not remove user from group",
				zap.String("user_id", userID.String()),
				zap.String("role_id", role.ID().String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
		}
	}

	// Assign new role
	if err := h.rbacService.AssignRoleToUserForTenant(r.Context(), userID, roleID, tenantID, authCtx.UserID); err != nil {
		h.logger.Error("Failed to assign role to user",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to assign role")
		return
	}

	// Also add user to the corresponding group for this role
	if err := h.addUserToGroupByRole(r.Context(), userID, roleID, tenantID); err != nil {
		h.logger.Warn("Failed to add user to group",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		// Don't fail the whole operation if group assignment fails
	}

	h.logger.Info("User role updated in tenant",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()),
		zap.String("tenant_id", tenantID.String()))

	// Send notification email if notification service is available
	if h.notificationService != nil {
		h.logger.Info("Notification service available, sending role changed notification",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()))
		go func(ns interface{}, uid uuid.UUID, tid uuid.UUID, rid uuid.UUID, oldRoles []*rbac.Role, logger *zap.Logger, db *sqlx.DB, userSvc *user.Service, rbacSvc *rbac.Service) {
			// Create a background context for the notification since the request context will be canceled
			ctx := context.Background()

			logger.Info("Starting role changed notification goroutine",
				zap.String("user_id", uid.String()),
				zap.String("tenant_id", tid.String()))

			// Fetch user details for notification
			user, err := userSvc.GetUserByID(ctx, uid)
			if err != nil {
				logger.Warn("Failed to fetch user for notification", zap.Error(err))
				return
			}

			// Fetch tenant name
			tenantName := "Tenant" // fallback
			if db != nil {
				var name string
				err := db.GetContext(ctx, &name, "SELECT name FROM tenants WHERE id = $1", tid.String())
				if err == nil {
					tenantName = name
				} else {
					logger.Warn("Failed to fetch tenant name for notification", zap.Error(err))
				}
			}

			// Fetch old role name
			oldRoleName := "No role" // fallback
			if len(oldRoles) > 0 {
				oldRoleName = oldRoles[0].Name()
			}

			// Fetch new role name
			newRoleName := "Role" // fallback
			if role, err := rbacSvc.GetRoleByID(ctx, rid); err == nil {
				newRoleName = role.Name()
			}

			// Create notification data for role change
			notifData := &email.UserRoleChangedData{
				UserEmail:    user.Email(),
				UserName:     user.FullName(),
				TenantName:   tenantName,
				TenantID:     tid,
				OldRole:      oldRoleName,
				NewRole:      newRoleName,
				DashboardURL: h.config.Frontend.DashboardURL,
			}

			// Send the notification email
			if notifSvc, ok := ns.(*email.NotificationService); ok {
				if err := notifSvc.SendUserRoleChangedEmail(ctx, notifData); err != nil {
					logger.Error("Failed to send user role changed notification",
						zap.String("user_email", user.Email()),
						zap.String("tenant_id", tid.String()),
						zap.String("old_role", oldRoleName),
						zap.String("new_role", newRoleName),
						zap.Error(err))
				} else {
					logger.Info("User role changed notification sent",
						zap.String("user_id", uid.String()),
						zap.String("user_email", user.Email()),
						zap.String("tenant_id", tid.String()),
						zap.String("old_role", oldRoleName),
						zap.String("new_role", newRoleName))
				}
			}
		}(h.notificationService, userID, tenantID, roleID, currentRoles, h.logger, h.db, h.userService, h.rbacService)
	} else {
		h.logger.Warn("Notification service not available, skipping role changed notification",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "User role updated successfully",
		"user_id":   userID.String(),
		"role_id":   roleID.String(),
		"tenant_id": tenantID.String(),
	})
}

// sendUserRoleChangedNotification sends an email notification when a user's role is changed
func (h *UserHandler) sendUserRoleChangedNotification(ctx context.Context, userID, roleID, tenantID uuid.UUID, oldRoleName string) {
	if h.notificationService == nil {
		h.logger.Debug("Notification service not available for role change notification",
			zap.String("user_id", userID.String()),
			zap.String("role_id", roleID.String()))
		return
	}

	h.logger.Info("Sending role change notification",
		zap.String("user_id", userID.String()),
		zap.String("role_id", roleID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("old_role", oldRoleName))

	// Run notification in background goroutine to avoid blocking the request
	go func(ns interface{}, uid uuid.UUID, rid uuid.UUID, tid uuid.UUID, oldRole string, logger *zap.Logger, db *sqlx.DB, userSvc *user.Service, rbacSvc *rbac.Service) {
		// Create a background context for the notification since the request context will be canceled
		bgCtx := context.Background()

		// Fetch user details for notification
		user, err := userSvc.GetUserByID(bgCtx, uid)
		if err != nil {
			logger.Warn("Failed to fetch user for role change notification",
				zap.String("user_id", uid.String()),
				zap.Error(err))
			return
		}

		// Fetch tenant name if tenantID is not nil
		tenantName := "System"
		if tid != uuid.Nil {
			if db != nil {
				var name string
				err := db.GetContext(bgCtx, &name, "SELECT name FROM tenants WHERE id = $1", tid.String())
				if err == nil {
					tenantName = name
				} else {
					logger.Warn("Failed to fetch tenant name for notification",
						zap.String("tenant_id", tid.String()),
						zap.Error(err))
				}
			}
		}

		// Fetch role name
		newRoleName := "Role"
		if role, err := rbacSvc.GetRoleByID(bgCtx, rid); err == nil {
			newRoleName = role.Name()
		} else {
			logger.Warn("Failed to fetch role name for notification",
				zap.String("role_id", rid.String()),
				zap.Error(err))
		}

		// Create notification data for role change
		notifData := &email.UserRoleChangedData{
			UserEmail:    user.Email(),
			UserName:     user.FirstName() + " " + user.LastName(),
			TenantName:   tenantName,
			TenantID:     tid,
			OldRole:      oldRole,
			NewRole:      newRoleName,
			DashboardURL: "http://localhost:3000",
		}

		// Send notification
		notifSvc := ns.(*email.NotificationService)
		if err := notifSvc.SendUserRoleChangedEmail(bgCtx, notifData); err != nil {
			logger.Warn("Failed to send user role changed notification",
				zap.String("user_id", uid.String()),
				zap.String("tenant_id", tid.String()),
				zap.Error(err))
		} else {
			logger.Info("User role changed notification sent successfully",
				zap.String("user_id", uid.String()),
				zap.String("tenant_id", tid.String()))
		}
	}(h.notificationService, userID, roleID, tenantID, oldRoleName, h.logger, h.db, h.userService, h.rbacService)
}

// sendUserRoleRemovedNotification sends an email notification when a user's role is removed
func (h *UserHandler) sendUserRoleRemovedNotification(ctx context.Context, userID, tenantID uuid.UUID, oldRoleName string) {
	if h.notificationService == nil {
		h.logger.Debug("Notification service not available for role removal notification",
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()))
		return
	}

	h.logger.Info("Sending role removal notification",
		zap.String("user_id", userID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("removed_role", oldRoleName))

	// Run notification in background goroutine to avoid blocking the request
	go func(ns interface{}, uid uuid.UUID, tid uuid.UUID, oldRole string, logger *zap.Logger, db *sqlx.DB, userSvc *user.Service) {
		// Create a background context for the notification since the request context will be canceled
		bgCtx := context.Background()

		// Fetch user details for notification
		user, err := userSvc.GetUserByID(bgCtx, uid)
		if err != nil {
			logger.Warn("Failed to fetch user for role removal notification",
				zap.String("user_id", uid.String()),
				zap.Error(err))
			return
		}

		// Fetch tenant name if tenantID is not nil
		tenantName := "System"
		if tid != uuid.Nil {
			if db != nil {
				var name string
				err := db.GetContext(bgCtx, &name, "SELECT name FROM tenants WHERE id = $1", tid.String())
				if err == nil {
					tenantName = name
				} else {
					logger.Warn("Failed to fetch tenant name for notification",
						zap.String("tenant_id", tid.String()),
						zap.Error(err))
				}
			}
		}

		// Create notification data for role removal
		notifData := &email.UserRoleChangedData{
			UserEmail:    user.Email(),
			UserName:     user.FirstName() + " " + user.LastName(),
			TenantName:   tenantName,
			TenantID:     tid,
			OldRole:      oldRole,
			NewRole:      "None",
			DashboardURL: "http://localhost:3000",
		}

		// Send notification
		notifSvc := ns.(*email.NotificationService)
		if err := notifSvc.SendUserRoleChangedEmail(bgCtx, notifData); err != nil {
			logger.Warn("Failed to send user role removal notification",
				zap.String("user_id", uid.String()),
				zap.String("tenant_id", tid.String()),
				zap.Error(err))
		} else {
			logger.Info("User role removal notification sent successfully",
				zap.String("user_id", uid.String()),
				zap.String("tenant_id", tid.String()))
		}
	}(h.notificationService, userID, tenantID, oldRoleName, h.logger, h.db, h.userService)
}
