package rest

import (
	"encoding/json"
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

// nilToEmpty converts a nil string pointer to empty string
func nilToEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PermissionHandler handles permission-related HTTP requests
type PermissionHandler struct {
	permissionService *rbac.PermissionService
	rbacService       *rbac.Service
	auditService      *audit.Service
	logger            *zap.Logger
}

// NewPermissionHandler creates a new permission handler
func NewPermissionHandler(
	permissionService *rbac.PermissionService,
	rbacService *rbac.Service,
	auditService *audit.Service,
	logger *zap.Logger,
) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		rbacService:       rbacService,
		auditService:      auditService,
		logger:            logger,
	}
}

// PermissionResponse represents a permission in API responses
type PermissionResponse struct {
	ID                 uuid.UUID `json:"id"`
	Resource           string    `json:"resource"`
	Action             string    `json:"action"`
	Description        string    `json:"description,omitempty"`
	Category           string    `json:"category,omitempty"`
	IsSystemPermission bool      `json:"is_system_permission"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// CreatePermissionRequest represents the request to create a new permission
type CreatePermissionRequest struct {
	Resource    string  `json:"resource"`
	Action      string  `json:"action"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
}

// UpdatePermissionRequest represents the request to update a permission
type UpdatePermissionRequest struct {
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
}

// ListPermissionsResponse represents paginated permission list
type ListPermissionsResponse struct {
	Data       []PermissionResponse `json:"data"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

// GetPermissions handles GET /api/v1/permissions - List all permissions with pagination
// PermissionsGroupedResponse represents permissions grouped by resource
type PermissionsGroupedResponse struct {
	Resources map[string][]PermissionResponse `json:"resources"`
}

func (h *PermissionHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	h.logger.Info("Getting permissions", zap.String("path", r.URL.Path))

	// Parse query parameters
	page := 1
	pageSize := 50
	resourceFilter := r.URL.Query().Get("resource") // Support filtering by resource
	grouped := r.URL.Query().Get("grouped") == "true"
	
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if sizeStr := r.URL.Query().Get("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	// Fetch all permissions from service
	allPermissions, err := h.permissionService.GetAllPermissions(r.Context())
	if err != nil {
		h.logger.Error("Failed to fetch permissions", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch permissions"})
		return
	}

	// Filter by resource if specified
	permissions := allPermissions
	if resourceFilter != "" {
		filtered := make([]*rbac.PermissionRecord, 0, len(allPermissions))
		for _, perm := range allPermissions {
			if perm.Resource == resourceFilter {
				filtered = append(filtered, perm)
			}
		}
		permissions = filtered
	}

	// If grouped response requested, return grouped by resource
	if grouped {
		groupedPerms := make(map[string][]PermissionResponse)
		for _, perm := range permissions {
			if _, exists := groupedPerms[perm.Resource]; !exists {
				groupedPerms[perm.Resource] = []PermissionResponse{}
			}
			groupedPerms[perm.Resource] = append(groupedPerms[perm.Resource], PermissionResponse{
				ID:                 perm.ID,
				Resource:           perm.Resource,
				Action:             perm.Action,
				Description:        nilToEmpty(perm.Description),
				Category:           nilToEmpty(perm.Category),
				IsSystemPermission: perm.IsSystemPermission,
				CreatedAt:          perm.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:          perm.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(PermissionsGroupedResponse{Resources: groupedPerms})
		h.logger.Info("Grouped permissions retrieved successfully", zap.Int("resource_count", len(groupedPerms)))
		return
	}

	// Apply pagination for non-grouped response
	total := len(permissions)
	totalPages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// Convert to response format
	response := make([]PermissionResponse, 0, end-start)
	for _, perm := range permissions[start:end] {
		response = append(response, PermissionResponse{
			ID:                 perm.ID,
			Resource:           perm.Resource,
			Action:             perm.Action,
			Description:        nilToEmpty(perm.Description),
			Category:           nilToEmpty(perm.Category),
			IsSystemPermission: perm.IsSystemPermission,
			CreatedAt:          perm.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:          perm.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	result := ListPermissionsResponse{
		Data:       response,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)

	h.logger.Info("Permissions retrieved successfully",
		zap.Int("count", len(response)),
		zap.Int("total", total),
		zap.String("resource_filter", resourceFilter))
}

// GetPermissionByID handles GET /api/v1/permissions/:id - Get a specific permission
func (h *PermissionHandler) GetPermissionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract permission ID from path
	permIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/permissions/")
	if permIDStr == r.URL.Path || permIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID"})
		return
	}

	h.logger.Info("Getting permission by ID", zap.String("permission_id", permIDStr))

	// Parse UUID
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID format"})
		return
	}

	// Get from service
	permissions, err := h.permissionService.GetAllPermissions(r.Context())
	if err != nil {
		h.logger.Error("Failed to fetch permissions", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch permissions"})
		return
	}

	// Find by ID
	var found *rbac.PermissionRecord
	for _, perm := range permissions {
		if perm.ID == permID {
			found = perm
			break
		}
	}

	if found == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Permission not found"})
		return
	}

	response := PermissionResponse{
		ID:                 found.ID,
		Resource:           found.Resource,
		Action:             found.Action,
		Description:        nilToEmpty(found.Description),
		Category:           nilToEmpty(found.Category),
		IsSystemPermission: found.IsSystemPermission,
		CreatedAt:          found.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          found.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Permission retrieved successfully",
		zap.String("permission_id", found.ID.String()))
}

// GetRolePermissions handles GET /api/v1/roles/:id/permissions - Get permissions for a role
// GetRolePermissions handles GET /api/v1/roles/{id}/permissions
func (h *PermissionHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	// Note: This is a read-only endpoint that doesn't require authentication
	// It's used by the permissions management UI to display role permissions
	// Auth is not strictly required for reading role metadata

	roleIDStr := chi.URLParam(r, "id")

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID"})
		return
	}

	h.logger.Info("Getting role permissions", zap.String("role_id", roleID.String()))

	// Get role from service
	role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
	if err != nil {
		h.logger.Error("Failed to fetch role", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Role not found"})
		return
	}

	// Convert role permissions to API response format
	// The role's Permissions() returns []rbac.Permission (just resource:action)
	// We need to convert these back to detailed PermissionResponse objects using the permission service
	allPermissions, err := h.permissionService.GetAllPermissions(r.Context())
	if err != nil {
		h.logger.Error("Failed to fetch full permission details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch permissions"})
		return
	}

	// Build response with full permission details
	response := make([]PermissionResponse, 0)
	rolePerms := role.Permissions()
	h.logger.Debug("GetRolePermissions: Matching role permissions with full details",
		zap.Int("role_perm_count", len(rolePerms)),
		zap.Int("all_perm_count", len(allPermissions)),
	)
	for _, rolePerm := range rolePerms {
		// Find the detailed permission info
		found := false
		for _, fullPerm := range allPermissions {
			if fullPerm.Resource == rolePerm.Resource && fullPerm.Action == rolePerm.Action {
				response = append(response, PermissionResponse{
					ID:          fullPerm.ID,
					Resource:    fullPerm.Resource,
					Action:      fullPerm.Action,
					Description: nilToEmpty(fullPerm.Description),
				})
				found = true
				break
			}
		}
		if !found {
			h.logger.Warn("Permission in role not found in full permissions list",
				zap.String("resource", rolePerm.Resource),
				zap.String("action", rolePerm.Action),
			)
		}
	}

	result := ListPermissionsResponse{
		Data:       response,
		Total:      len(response),
		Page:       1,
		PageSize:   len(response),
		TotalPages: 1,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)

	h.logger.Info("Role permissions retrieved successfully",
		zap.String("role_id", roleID.String()),
		zap.Int("count", len(response)))
}

// AddPermissionToRole handles POST /api/v1/roles/:id/permissions/:perm-id - Add permission to role
func (h *PermissionHandler) AddPermissionToRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract role ID and permission ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	if path == r.URL.Path || path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID or permission ID"})
		return
	}

	// Split path to get role ID and permission ID
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" || parts[2] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID or permission ID"})
		return
	}

	roleIDStr := parts[0]
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID format"})
		return
	}

	permIDStr := parts[2]
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID format"})
		return
	}

	h.logger.Info("Adding permission to role",
		zap.String("role_id", roleID.String()),
		zap.String("permission_id", permID.String()))

	// Get the permission to add
	allPerms, err := h.permissionService.GetAllPermissions(r.Context())
	if err != nil {
		h.logger.Error("Failed to verify permission", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to verify permission"})
		return
	}

	var permToAdd *rbac.PermissionRecord
	for _, p := range allPerms {
		if p.ID == permID {
			permToAdd = p
			break
		}
	}

	if permToAdd == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Permission not found"})
		return
	}

	// Get the role
	role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
	if err != nil {
		h.logger.Error("Failed to fetch role", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Role not found"})
		return
	}

	// Check if permission already exists in role
	if role.HasPermission(permToAdd.Resource, permToAdd.Action) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "Permission already assigned to role"})
		return
	}

	// Add permission to role using domain method
	role.AddPermission(permToAdd.Resource, permToAdd.Action)

	// Update role in database
	err = h.rbacService.UpdateRole(r.Context(), role)
	if err != nil {
		h.logger.Error("Failed to update role", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update role"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Permission added to role successfully",
		"role_id": roleID.String(),
	})

	h.logger.Info("Permission added to role successfully",
		zap.String("role_id", roleID.String()),
		zap.String("permission_id", permID.String()))
}

// RemovePermissionFromRole handles DELETE /api/v1/roles/:id/permissions/:perm-id - Remove permission from role
func (h *PermissionHandler) RemovePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract role ID and permission ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/roles/")
	if path == r.URL.Path || path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID or permission ID"})
		return
	}

	// Split path to get role ID and permission ID
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" || parts[2] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID or permission ID"})
		return
	}

	roleIDStr := parts[0]
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role ID format"})
		return
	}

	permIDStr := parts[2]
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID format"})
		return
	}

	h.logger.Info("Removing permission from role",
		zap.String("role_id", roleID.String()),
		zap.String("permission_id", permID.String()))

	// Get the permission to remove
	allPerms, err := h.permissionService.GetAllPermissions(r.Context())
	if err != nil {
		h.logger.Error("Failed to verify permission", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to verify permission"})
		return
	}

	var permToRemove *rbac.PermissionRecord
	for _, p := range allPerms {
		if p.ID == permID {
			permToRemove = p
			break
		}
	}

	if permToRemove == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Permission not found"})
		return
	}

	// Get the role
	role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
	if err != nil {
		h.logger.Error("Failed to fetch role", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Role not found"})
		return
	}

	// Check if permission is actually assigned to this role
	if !role.HasPermission(permToRemove.Resource, permToRemove.Action) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Permission not assigned to this role"})
		return
	}

	// Remove permission from role using domain method
	role.RemovePermission(permToRemove.Resource, permToRemove.Action)

	// Update role in database
	err = h.rbacService.UpdateRole(r.Context(), role)
	if err != nil {
		h.logger.Error("Failed to update role", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update role"})
		return
	}

	w.WriteHeader(http.StatusNoContent)

	h.logger.Info("Permission removed from role successfully",
		zap.String("role_id", roleID.String()),
		zap.String("permission_id", permID.String()))
}

// BulkAssignPermissionRequest represents the request body for bulk permission assignment
type BulkAssignPermissionRequest struct {
	RoleIDs []string `json:"roleIds"`
}

// BulkAssignPermissionResponse represents the response for bulk permission assignment
type BulkAssignPermissionResponse struct {
	PermissionID string                       `json:"permissionId"`
	Assigned     []BulkAssignedRole           `json:"assigned"`
	Failed       []BulkAssignmentFailure      `json:"failed"`
	Total        int                          `json:"total"`
	Success      int                          `json:"success"`
	FailureCount int                          `json:"failureCount"`
}

// BulkAssignedRole represents a successfully assigned role
type BulkAssignedRole struct {
	RoleID string `json:"roleId"`
	Status string `json:"status"` // "newly_assigned", "already_assigned"
}

// BulkAssignmentFailure represents a failed assignment
type BulkAssignmentFailure struct {
	RoleID string `json:"roleId"`
	Error  string `json:"error"`
}

// AssignPermissionToMultipleRoles handles POST /api/v1/permissions/:id/roles - Assign permission to multiple roles in a single request
func (h *PermissionHandler) AssignPermissionToMultipleRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract permission ID from URL params
	permIDStr := chi.URLParam(r, "id")
	if permIDStr == "" {
		// Fallback to path parsing if URL param not available
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/permissions/")
		if path == r.URL.Path || path == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID"})
			return
		}

		// Split path to get permission ID and ensure /roles is next
		parts := strings.Split(path, "/")
		if len(parts) < 2 || parts[0] == "" || parts[1] != "roles" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request path"})
			return
		}
		permIDStr = parts[0]
	}

	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID format"})
		return
	}

	// Parse request body
	var req BulkAssignPermissionRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if len(req.RoleIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "roleIds array cannot be empty"})
		return
	}

	h.logger.Info("Assigning permission to multiple roles",
		zap.String("permission_id", permID.String()),
		zap.Int("role_count", len(req.RoleIDs)))

	// Get the permission
	allPerms, err := h.permissionService.GetAllPermissions(r.Context())
	if err != nil {
		h.logger.Error("Failed to verify permission", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to verify permission"})
		return
	}

	var permToAdd *rbac.PermissionRecord
	for _, p := range allPerms {
		if p.ID == permID {
			permToAdd = p
			break
		}
	}

	if permToAdd == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Permission not found"})
		return
	}

	// Process each role - assign permission and track results
	response := BulkAssignPermissionResponse{
		PermissionID: permID.String(),
		Assigned:     []BulkAssignedRole{},
		Failed:       []BulkAssignmentFailure{},
		Total:        len(req.RoleIDs),
		Success:      0,
		FailureCount: 0,
	}

	for _, roleIDStr := range req.RoleIDs {
		roleID, err := uuid.Parse(roleIDStr)
		if err != nil {
			response.Failed = append(response.Failed, BulkAssignmentFailure{
				RoleID: roleIDStr,
				Error:  "Invalid role ID format",
			})
			response.FailureCount++
			continue
		}

		// Get the role
		role, err := h.rbacService.GetRoleByID(r.Context(), roleID)
		if err != nil {
			response.Failed = append(response.Failed, BulkAssignmentFailure{
				RoleID: roleID.String(),
				Error:  "Role not found",
			})
			response.FailureCount++
			continue
		}

		// Check if permission already exists
		if role.HasPermission(permToAdd.Resource, permToAdd.Action) {
			response.Assigned = append(response.Assigned, BulkAssignedRole{
				RoleID: roleID.String(),
				Status: "already_assigned",
			})
			response.Success++
			continue
		}

		// Add permission to role
		role.AddPermission(permToAdd.Resource, permToAdd.Action)

		// Update role in database
		err = h.rbacService.UpdateRole(r.Context(), role)
		if err != nil {
			h.logger.Error("Failed to update role", zap.Error(err))
			response.Failed = append(response.Failed, BulkAssignmentFailure{
				RoleID: roleID.String(),
				Error:  "Failed to update role: " + err.Error(),
			})
			response.FailureCount++
			continue
		}

		response.Assigned = append(response.Assigned, BulkAssignedRole{
			RoleID: roleID.String(),
			Status: "newly_assigned",
		})
		response.Success++
	}

	h.logger.Info("Permission assigned to multiple roles completed",
		zap.String("permission_id", permID.String()),
		zap.Int("success_count", response.Success),
		zap.Int("failure_count", response.FailureCount))

	w.Header().Set("Content-Type", "application/json")
	if response.FailureCount > 0 && response.Success == 0 {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
}

// CreatePermission handles POST /api/v1/permissions - Create a new permission
func (h *PermissionHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	h.logger.Info("Creating new permission", zap.String("path", r.URL.Path))

	var req CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Resource == "" || req.Action == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Resource and action are required"})
		return
	}

	// Create permission
	created, err := h.permissionService.CreatePermission(r.Context(), req.Resource, req.Action, req.Description, req.Category)
	if err != nil {
		h.logger.Error("Failed to create permission", zap.Error(err))
		if strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := PermissionResponse{
		ID:                 created.ID,
		Resource:           created.Resource,
		Action:             created.Action,
		Description:        nilToEmpty(created.Description),
		Category:           nilToEmpty(created.Category),
		IsSystemPermission: created.IsSystemPermission,
		CreatedAt:          created.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          created.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	// Audit permission creation
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionCheck, "permissions", "create",
				"Created permission",
				map[string]interface{}{
					"permission_id": created.ID.String(),
					"resource":      created.Resource,
					"action":        created.Action,
					"description":   nilToEmpty(created.Description),
					"category":      nilToEmpty(created.Category),
				})
		}
	}

	h.logger.Info("Permission created successfully",
		zap.String("permission_id", created.ID.String()),
		zap.String("resource", created.Resource),
		zap.String("action", created.Action))
}

// UpdatePermission handles PUT /api/v1/permissions/:id - Update a permission
func (h *PermissionHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract permission ID from path
	permIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/permissions/")
	if permIDStr == r.URL.Path || permIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID"})
		return
	}

	h.logger.Info("Updating permission", zap.String("permission_id", permIDStr))

	// Parse UUID
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID format"})
		return
	}

	var req UpdatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Update permission
	updated, err := h.permissionService.UpdatePermission(r.Context(), permID, req.Description, req.Category)
	if err != nil {
		h.logger.Error("Failed to update permission", zap.Error(err))
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
		} else if strings.Contains(err.Error(), "system permission") {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := PermissionResponse{
		ID:                 updated.ID,
		Resource:           updated.Resource,
		Action:             updated.Action,
		Description:        nilToEmpty(updated.Description),
		Category:           nilToEmpty(updated.Category),
		IsSystemPermission: updated.IsSystemPermission,
		CreatedAt:          updated.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          updated.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	// Audit permission update
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionCheck, "permissions", "update",
				"Updated permission",
				map[string]interface{}{
					"permission_id": updated.ID.String(),
					"resource":      updated.Resource,
					"action":        updated.Action,
					"description":   nilToEmpty(updated.Description),
					"category":      nilToEmpty(updated.Category),
				})
		}
	}

	h.logger.Info("Permission updated successfully",
		zap.String("permission_id", updated.ID.String()))
}

// DeletePermission handles DELETE /api/v1/permissions/:id - Delete a permission
func (h *PermissionHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract permission ID from path
	permIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/permissions/")
	if permIDStr == r.URL.Path || permIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID"})
		return
	}

	h.logger.Info("Deleting permission", zap.String("permission_id", permIDStr))

	// Parse UUID
	permID, err := uuid.Parse(permIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid permission ID format"})
		return
	}

	// Delete permission
	err = h.permissionService.DeletePermission(r.Context(), permID)
	if err != nil {
		h.logger.Error("Failed to delete permission", zap.Error(err))
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
		} else if strings.Contains(err.Error(), "system permission") {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Audit permission deletion
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionCheck, "permissions", "delete",
				"Deleted permission",
				map[string]interface{}{
					"permission_id": permID.String(),
				})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Permission deleted successfully"})

	h.logger.Info("Permission deleted successfully",
		zap.String("permission_id", permIDStr))
}
