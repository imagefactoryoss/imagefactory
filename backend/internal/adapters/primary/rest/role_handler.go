package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// RoleHandler handles role management HTTP requests
type RoleHandler struct {
	rbacService       *rbac.Service
	permissionService *rbac.PermissionService
	auditService      *audit.Service
	logger            *zap.Logger
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(rbacService *rbac.Service, auditService *audit.Service, logger *zap.Logger) *RoleHandler {
	return &RoleHandler{
		rbacService:  rbacService,
		auditService: auditService,
		logger:       logger,
	}
}

// SetPermissionService sets the permission service on the role handler
func (h *RoleHandler) SetPermissionService(permissionService *rbac.PermissionService) {
	h.permissionService = permissionService
}

// CreateRoleRequest represents a role creation request
type CreateRoleRequest struct {
	Name        string   `json:"name" validate:"required,min=3,max=100"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions" validate:"required,min=1"`
	TenantID    string   `json:"tenant_id,omitempty"`
	IsSystem    bool     `json:"is_system,omitempty"`
}

// UpdateRoleRequest represents a role update request
type UpdateRoleRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// RolePermissionResponse represents a permission with details
type RolePermissionResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// RoleDetailResponse represents detailed role information
type RoleDetailResponse struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	IsSystem    bool                     `json:"is_system"`
	Permissions []RolePermissionResponse `json:"permissions"`
	TenantID    *string                  `json:"tenant_id,omitempty"`
	CreatedAt   string                   `json:"created_at"`
	UpdatedAt   string                   `json:"updated_at"`
}

// ListRolesResponse represents a list of roles with pagination
type ListRolesResponse struct {
	Data       []RoleDetailResponse `json:"data"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

func normalizeRoleTypeFromName(roleName string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(roleName)))
	return strings.Join(parts, "_")
}

func shouldExposeRoleForTenant(roleName string, allowedRoleTypes map[string]struct{}) bool {
	lowerRole := strings.ToLower(strings.TrimSpace(roleName))
	if lowerRole == "system administrator" || lowerRole == "system administrator viewer" {
		return false
	}
	if len(allowedRoleTypes) == 0 {
		return false
	}
	_, ok := allowedRoleTypes[normalizeRoleTypeFromName(roleName)]
	return ok
}

// GetRoles handles GET /roles - List all roles (system and tenant-specific)
func (h *RoleHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Get pagination parameters
	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	// Check if a tenant_id is provided - if so, filter out system-only roles
	tenantID := r.URL.Query().Get("tenant_id")

	// Get all system-level roles (both system and tenant-level roles where tenant_id is NULL)
	allRoles, err := h.rbacService.GetAllSystemLevelRoles(r.Context())
	if err != nil {
		h.logger.Error("Failed to fetch roles", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to fetch roles").WithCause(err))
		return
	}

	// Convert to response format
	roles := make([]RoleDetailResponse, 0)
	for _, role := range allRoles {
		// If tenant_id is provided, filter out privileged system admin roles
		// (they should not be assignable to tenant members).
		lowerRole := strings.ToLower(role.Name())
		if tenantID != "" && (lowerRole == "system administrator" || lowerRole == "system administrator viewer") {
			h.logger.Debug("Skipping system administrator role for tenant",
				zap.String("tenant_id", tenantID),
				zap.String("role_name", role.Name()))
			continue
		}
		roleResp := h.roleToDetailResponse(role)
		roles = append(roles, roleResp)
	}

	// Apply pagination
	total := len(roles)
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if startIdx > total {
		startIdx = total
	}
	if endIdx > total {
		endIdx = total
	}

	pagedRoles := roles[startIdx:endIdx]
	totalPages := (total + pageSize - 1) / pageSize

	response := ListRolesResponse{
		Data:       pagedRoles,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := encodeJSON(w, response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// GetRolesByTenant handles GET /tenants/{tenantId}/roles - List roles for a specific tenant
func (h *RoleHandler) GetRolesByTenant(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("GetRolesByTenant called", zap.String("method", r.Method), zap.String("url", r.URL.String()))

	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Extract tenant ID from URL parameter
	tenantIDStr := chi.URLParam(r, "tenantId")
	if tenantIDStr == "" {
		h.logger.Error("Tenant ID parameter is empty")
		WriteError(w, r.Context(), BadRequest("Tenant ID is required"))
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid tenant ID"))
		return
	}

	assignableRoleTypes, err := h.rbacService.GetAssignableRoleTypesByTenant(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to fetch assignable role types for tenant",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to fetch roles").WithCause(err))
		return
	}
	allowedRoleTypes := make(map[string]struct{}, len(assignableRoleTypes))
	for _, roleType := range assignableRoleTypes {
		normalized := strings.ToLower(strings.TrimSpace(roleType))
		if normalized == "" {
			continue
		}
		allowedRoleTypes[normalized] = struct{}{}
	}

	// Get all system-level roles (available for assignment in any tenant)
	roles, err := h.rbacService.GetAllSystemLevelRoles(r.Context())
	if err != nil {
		h.logger.Error("Failed to fetch system-level roles",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to fetch roles").WithCause(err))
		return
	}

	// Convert to response format, excluding privileged system admin roles
	roleResponses := make([]RoleDetailResponse, 0, len(roles))
	for _, role := range roles {
		if !shouldExposeRoleForTenant(role.Name(), allowedRoleTypes) {
			continue
		}
		roleResp := h.roleToDetailResponse(role)
		roleResponses = append(roleResponses, roleResp)
	}

	response := ListRolesResponse{
		Data:       roleResponses,
		Total:      len(roleResponses),
		Page:       1,
		PageSize:   len(roleResponses),
		TotalPages: 1,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := encodeJSON(w, response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// GetRoleByID handles GET /roles/:id - Get role details
func (h *RoleHandler) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	roleID := chi.URLParam(r, "id")
	if roleID == "" {
		WriteError(w, r.Context(), BadRequest("Role ID is required"))
		return
	}

	// Parse role ID
	id, err := uuid.Parse(roleID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid role ID").WithCause(err))
		return
	}

	// Get role
	role, err := h.rbacService.GetRoleByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to fetch role", zap.String("roleID", roleID), zap.Error(err))
		WriteError(w, r.Context(), NotFound("Role not found").WithCause(err))
		return
	}

	response := h.roleToDetailResponse(role)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := encodeJSON(w, response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// CreateRole handles POST /roles - Create a new role
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create role request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Validate required fields
	if req.Name == "" {
		WriteError(w, r.Context(), BadRequest("Role name is required"))
		return
	}
	if len(req.Permissions) == 0 {
		WriteError(w, r.Context(), BadRequest("At least one permission is required"))
		return
	}

	// Validate all permissions exist and have correct format
	if h.permissionService != nil {
		for _, permStr := range req.Permissions {
			parts := strings.Split(permStr, ":")
			if len(parts) != 2 {
				WriteError(w, r.Context(), BadRequest(fmt.Sprintf("Invalid permission format: %s (expected 'resource:action')", permStr)))
				return
			}

			// Validate permission exists in database
			exists, err := h.permissionService.ValidatePermission(r.Context(), parts[0], parts[1])
			if err != nil {
				h.logger.Error("Failed to validate permission", zap.String("permission", permStr), zap.Error(err))
				WriteError(w, r.Context(), InternalServer("Failed to validate permissions").WithCause(err))
				return
			}
			if !exists {
				WriteError(w, r.Context(), BadRequest(fmt.Sprintf("Permission does not exist: %s", permStr)))
				return
			}
		}
	}

	// Convert permission strings to Permission objects
	permissions := make([]rbac.Permission, 0)
	for _, permStr := range req.Permissions {
		// Parse permission format: "resource:action"
		parts := strings.Split(permStr, ":")
		if len(parts) == 2 {
			perm := rbac.NewPermission(parts[0], parts[1])
			permissions = append(permissions, perm)
		}
	}

	if len(permissions) == 0 {
		WriteError(w, r.Context(), BadRequest("Invalid permission format. Expected 'resource:action'"))
		return
	}

	// Create role
	var role *rbac.Role
	var err error

	if req.IsSystem {
		// Create system role
		role, err = h.rbacService.CreateSystemRole(r.Context(), req.Name, req.Description, permissions)
	} else {
		// Create tenant-specific role
		if req.TenantID == "" {
			WriteError(w, r.Context(), BadRequest("Tenant ID is required for tenant-specific roles"))
			return
		}

		tenantID, err := uuid.Parse(req.TenantID)
		if err != nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant ID").WithCause(err))
			return
		}

		role, err = h.rbacService.CreateRole(r.Context(), tenantID, req.Name, req.Description, permissions)
	}

	if err != nil {
		h.logger.Error("Failed to create role", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to create role").WithCause(err))
		return
	}

	// Audit role creation
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventRoleAssign, "roles", "create",
				fmt.Sprintf("Created role '%s' with %d permissions", role.Name(), len(role.Permissions())),
				map[string]interface{}{
					"role_id":     role.ID().String(),
					"role_name":   role.Name(),
					"permissions": req.Permissions,
					"is_system":   req.IsSystem,
					"tenant_id":   req.TenantID,
				})
		}
	}

	response := h.roleToDetailResponse(role)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := encodeJSON(w, response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// UpdateRole handles PUT /roles/:id - Update a role
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	roleID := chi.URLParam(r, "id")
	if roleID == "" {
		WriteError(w, r.Context(), BadRequest("Role ID is required"))
		return
	}

	// Parse role ID
	id, err := uuid.Parse(roleID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid role ID").WithCause(err))
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update role request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Get existing role
	role, err := h.rbacService.GetRoleByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to fetch role", zap.String("roleID", roleID), zap.Error(err))
		WriteError(w, r.Context(), NotFound("Role not found").WithCause(err))
		return
	}

	// Update role fields
	if req.Name != "" || req.Description != "" {
		name := role.Name()
		description := role.Description()

		if req.Name != "" {
			name = req.Name
		}
		if req.Description != "" {
			description = req.Description
		}

		if err := role.UpdateDetails(name, description); err != nil {
			h.logger.Error("Failed to update role details", zap.String("roleID", roleID), zap.Error(err))
			WriteError(w, r.Context(), InternalServer("Failed to update role").WithCause(err))
			return
		}
	}

	// Update permissions if provided
	if len(req.Permissions) > 0 {
		// Validate all permissions exist and have correct format before making changes
		if h.permissionService != nil {
			for _, permStr := range req.Permissions {
				parts := strings.Split(permStr, ":")
				if len(parts) != 2 {
					h.logger.Warn("Invalid permission format", zap.String("permission", permStr))
					WriteError(w, r.Context(), BadRequest(fmt.Sprintf("Invalid permission format: %s (expected 'resource:action')", permStr)))
					return
				}

				// Validate permission exists in database
				exists, err := h.permissionService.ValidatePermission(r.Context(), parts[0], parts[1])
				if err != nil {
					h.logger.Error("Failed to validate permission", zap.String("permission", permStr), zap.String("resource", parts[0]), zap.String("action", parts[1]), zap.Error(err))
					WriteError(w, r.Context(), InternalServer("Failed to validate permissions").WithCause(err))
					return
				}
				if !exists {
					h.logger.Warn("Permission does not exist", zap.String("permission", permStr), zap.String("resource", parts[0]), zap.String("action", parts[1]))
					WriteError(w, r.Context(), BadRequest(fmt.Sprintf("Permission does not exist: %s", permStr)))
					return
				}
			}
		}

		// All permissions validated, now update the role
		// Remove all existing permissions
		for _, perm := range role.Permissions() {
			role.RemovePermission(perm.Resource, perm.Action)
		}

		// Add new permissions
		for _, permStr := range req.Permissions {
			parts := strings.Split(permStr, ":")
			if len(parts) == 2 {
				role.AddPermission(parts[0], parts[1])
			}
		}
	}

	// Save updated role
	if err := h.rbacService.UpdateRole(r.Context(), role); err != nil {
		h.logger.Error("Failed to update role", zap.String("roleID", roleID), zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to update role").WithCause(err))
		return
	}

	// Audit role update
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventRoleAssign, "roles", "update",
				fmt.Sprintf("Updated role '%s'", role.Name()),
				map[string]interface{}{
					"role_id":     role.ID().String(),
					"role_name":   role.Name(),
					"permissions": req.Permissions,
				})
		}
	}

	response := h.roleToDetailResponse(role)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := encodeJSON(w, response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// DeleteRole handles DELETE /roles/:id - Delete a role
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	roleID := chi.URLParam(r, "id")
	if roleID == "" {
		WriteError(w, r.Context(), BadRequest("Role ID is required"))
		return
	}

	// Parse role ID
	id, err := uuid.Parse(roleID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid role ID").WithCause(err))
		return
	}

	// Check if role exists
	role, err := h.rbacService.GetRoleByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to fetch role", zap.String("roleID", roleID), zap.Error(err))
		WriteError(w, r.Context(), NotFound("Role not found").WithCause(err))
		return
	}

	// Prevent deletion of system roles
	if role.IsSystem() {
		WriteError(w, r.Context(), Forbidden("Cannot delete system roles"))
		return
	}

	// Delete role
	if err := h.rbacService.DeleteRole(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete role", zap.String("roleID", roleID), zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to delete role").WithCause(err))
		return
	}

	// Audit role deletion
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventRoleRemove, "roles", "delete",
				fmt.Sprintf("Deleted role '%s'", role.Name()),
				map[string]interface{}{
					"role_id":   role.ID().String(),
					"role_name": role.Name(),
				})
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper function to convert Role to RoleDetailResponse
func (h *RoleHandler) roleToDetailResponse(role *rbac.Role) RoleDetailResponse {
	permissions := make([]RolePermissionResponse, 0)
	for _, perm := range role.Permissions() {
		permissions = append(permissions, RolePermissionResponse{
			ID:          perm.ID,
			Name:        perm.String(),
			Description: perm.String(),
			Resource:    perm.Resource,
			Action:      perm.Action,
		})
	}

	var tenantID *string
	if role.TenantID() != uuid.Nil {
		id := role.TenantID().String()
		tenantID = &id
	}

	return RoleDetailResponse{
		ID:          role.ID().String(),
		Name:        role.Name(),
		Description: role.Description(),
		IsSystem:    role.IsSystem(),
		Permissions: permissions,
		TenantID:    tenantID,
		CreatedAt:   role.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   role.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
