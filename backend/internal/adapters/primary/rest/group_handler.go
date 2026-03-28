package rest

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

// GroupHandler handles group-related HTTP requests
type GroupHandler struct {
	db                   *sqlx.DB
	logger               *zap.Logger
	notificationService  *email.NotificationService
	auditService         *audit.Service
}

// NewGroupHandler creates a new group handler
func NewGroupHandler(db *sqlx.DB, auditService *audit.Service, logger *zap.Logger) *GroupHandler {
	return &GroupHandler{
		db:                   db,
		logger:               logger,
		notificationService:  nil,
		auditService:         auditService,
	}
}

// SetNotificationService sets the notification service
func (h *GroupHandler) SetNotificationService(notificationService *email.NotificationService) {
	h.notificationService = notificationService
}

// GroupDTO represents a group data transfer object
type GroupDTO struct {
	ID            string `db:"id" json:"id"`
	TenantID      string `db:"tenant_id" json:"tenant_id"`
	Name          string `db:"name" json:"name"`
	Slug          string `db:"slug" json:"slug"`
	Description   string `db:"description" json:"description"`
	RoleType      string `db:"role_type" json:"role_type"`
	IsSystemGroup bool   `db:"is_system_group" json:"is_system_group"`
	Status        string `db:"status" json:"status"`
	MemberCount   int    `db:"member_count" json:"member_count"`
	CreatedAt     string `db:"created_at" json:"created_at"`
	UpdatedAt     string `db:"updated_at" json:"updated_at"`
}

// ListGroupsByTenant lists all groups for a tenant
func (h *GroupHandler) ListGroupsByTenant(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from URL path: /api/v1/tenants/{tenantId}/groups
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/"), "/")
	if len(parts) < 2 {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	tenantID := parts[0]
	if tenantID == "" {
		h.respondError(w, http.StatusBadRequest, "Tenant ID is required")
		return
	}

	// Validate tenant ID is a valid UUID
	if _, err := uuid.Parse(tenantID); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID format")
		return
	}

	// Query groups with member counts
	query := `
		SELECT 
			tg.id,
			tg.tenant_id,
			tg.name,
			tg.slug,
			tg.description,
			tg.role_type,
			tg.is_system_group,
			tg.status,
			COALESCE(COUNT(gm.id), 0) as member_count,
			tg.created_at,
			tg.updated_at
		FROM tenant_groups tg
		LEFT JOIN group_members gm ON tg.id = gm.group_id AND gm.removed_at IS NULL
		WHERE tg.tenant_id = $1 AND tg.status = 'active'
		GROUP BY tg.id, tg.tenant_id, tg.name, tg.slug, tg.description, tg.role_type, tg.is_system_group, tg.status, tg.created_at, tg.updated_at
		ORDER BY tg.role_type ASC, tg.name ASC
	`

	rows, err := h.db.Queryx(query, tenantID)
	if err != nil {
		h.logger.Error("Failed to query groups", zap.Error(err), zap.String("tenant_id", tenantID))
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch groups")
		return
	}
	defer rows.Close()

	groups := []GroupDTO{}
	for rows.Next() {
		var group GroupDTO
		if err := rows.StructScan(&group); err != nil {
			h.logger.Error("Failed to scan group", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, "Failed to parse group data")
			return
		}
		groups = append(groups, group)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": groups,
	})
}

// ListGroupMembers lists all members in a group
func (h *GroupHandler) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	// Extract group ID from URL path: /api/v1/groups/{groupId}/members
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/groups/"), "/")
	if len(parts) < 2 {
		h.respondError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	groupID := parts[0]
	if groupID == "" {
		h.respondError(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	// Validate group ID is a valid UUID
	if _, err := uuid.Parse(groupID); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid group ID format")
		return
	}

	type GroupMember struct {
		ID          string `db:"id" json:"id"`
		UserID      string `db:"user_id" json:"user_id"`
		Email       string `db:"email" json:"email"`
		FirstName   string `db:"first_name" json:"first_name"`
		LastName    string `db:"last_name" json:"last_name"`
		IsGroupAdmin bool   `db:"is_group_admin" json:"is_group_admin"`
		AddedAt     string `db:"added_at" json:"added_at"`
	}

	// Query group members
	query := `
		SELECT 
			gm.id,
			gm.user_id,
			u.email,
			u.first_name,
			u.last_name,
			gm.is_group_admin,
			gm.added_at
		FROM group_members gm
		INNER JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1 AND gm.removed_at IS NULL
		ORDER BY u.first_name ASC, u.last_name ASC
	`

	rows, err := h.db.Queryx(query, groupID)
	if err != nil {
		h.logger.Error("Failed to query group members", zap.Error(err), zap.String("group_id", groupID))
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch group members")
		return
	}
	defer rows.Close()

	members := []GroupMember{}
	for rows.Next() {
		var member GroupMember
		if err := rows.StructScan(&member); err != nil {
			h.logger.Error("Failed to scan group member", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, "Failed to parse member data")
			return
		}
		members = append(members, member)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": members,
	})
}

// AddGroupMember adds a user to a group
func (h *GroupHandler) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Extract group ID from URL path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/groups/"), "/")
	if len(parts) < 2 {
		h.respondError(w, http.StatusBadRequest, "Invalid group ID")
		return
	}

	groupID := parts[0]
	if _, err := uuid.Parse(groupID); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid group ID format")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	if _, err := uuid.Parse(req.UserID); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	// Check if user is already in group
	var existingID string
	err := h.db.Get(&existingID, `
		SELECT id FROM group_members 
		WHERE group_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, groupID, req.UserID)

	if err == nil {
		h.respondError(w, http.StatusConflict, "User is already a member of this group")
		return
	} else if err != sql.ErrNoRows {
		h.logger.Error("Failed to check existing membership", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}

	// Add user to group
	memberID := uuid.New().String()
	_, err = h.db.Exec(`
		INSERT INTO group_members (id, group_id, user_id, is_group_admin, added_at)
		VALUES ($1, $2, $3, false, CURRENT_TIMESTAMP)
	`, memberID, groupID, req.UserID)

	if err != nil {
		h.logger.Error("Failed to add group member", zap.Error(err), zap.String("group_id", groupID), zap.String("user_id", req.UserID))
		h.respondError(w, http.StatusInternalServerError, "Failed to add user to group")
		return
	}

	// Fetch user and group details for notification
	var user struct {
		Email     string `db:"email"`
		FirstName string `db:"first_name"`
		LastName  string `db:"last_name"`
	}
	userErr := h.db.Get(&user, `
		SELECT email, first_name, last_name FROM users WHERE id = $1
	`, req.UserID)

	var group struct {
		Name     string    `db:"name"`
		TenantID uuid.UUID `db:"tenant_id"`
	}
	groupErr := h.db.Get(&group, `
		SELECT name, tenant_id FROM tenant_groups WHERE id = $1
	`, groupID)

	var tenant struct {
		Name string `db:"name"`
	}
	tenantErr := h.db.Get(&tenant, `
		SELECT name FROM tenants WHERE id = $1
	`, group.TenantID)

	// Send notification if all details were fetched successfully and notification service is available
	if h.notificationService != nil && userErr == nil && groupErr == nil && tenantErr == nil {
		notificationData := &email.UserAddedToGroupData{
			UserEmail:    user.Email,
			UserName:     user.FirstName + " " + user.LastName,
			GroupName:    group.Name,
			TenantName:   tenant.Name,
			TenantID:     group.TenantID,
			DashboardURL: "https://app.imgfactory.com/dashboard",
		}

		if err := h.notificationService.SendUserAddedToGroupEmail(r.Context(), notificationData); err != nil {
			h.logger.Warn("Failed to send user added to group notification",
				zap.Error(err),
				zap.String("user_id", req.UserID),
				zap.String("group_id", groupID),
				zap.String("user_email", user.Email))
			// Don't fail the request if email fails
		} else {
			h.logger.Info("Sent user added to group notification",
				zap.String("user_id", req.UserID),
				zap.String("group_id", groupID),
				zap.String("user_email", user.Email))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": memberID,
		"message": "User added to group successfully",
	})

	// Audit group member addition
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), userCtx.TenantID, userCtx.UserID,
			audit.AuditEventGroupMemberAdd, "groups", "add_member",
			"User added to group",
			map[string]interface{}{
				"group_id":     groupID,
				"user_id":      req.UserID,
				"member_id":    memberID,
			})
	}
}

// RemoveGroupMember removes a user from a group
func (h *GroupHandler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Extract group ID and member ID from URL path: /api/v1/groups/{groupId}/members/{memberId}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/groups/"), "/")
	if len(parts) < 4 {
		h.respondError(w, http.StatusBadRequest, "Invalid request path")
		return
	}

	groupID := parts[0]
	memberID := parts[2]

	if _, err := uuid.Parse(groupID); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid group ID format")
		return
	}

	if _, err := uuid.Parse(memberID); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid member ID format")
		return
	}

	// Check if the member being removed is a group admin (owner)
	var isGroupAdmin bool
	err := h.db.QueryRow(`
		SELECT is_group_admin 
		FROM group_members 
		WHERE id = $1 AND group_id = $2 AND removed_at IS NULL
	`, memberID, groupID).Scan(&isGroupAdmin)

	if err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Group member not found")
		} else {
			h.logger.Error("Failed to check group member status", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, "Failed to check member status")
		}
		return
	}

	// If the member is a group owner, check that there are other owners in the group
	if isGroupAdmin {
		var ownerCount int
		err := h.db.QueryRow(`
			SELECT COUNT(*) 
			FROM group_members 
			WHERE group_id = $1 AND is_group_admin = true AND removed_at IS NULL
		`, groupID).Scan(&ownerCount)

		if err != nil {
			h.logger.Error("Failed to count group owners", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, "Failed to validate group owners")
			return
		}

		// Prevent removal if this is the only owner
		if ownerCount <= 1 {
			h.respondError(w, http.StatusBadRequest, "Cannot remove the last owner from a group. Assign another owner first.")
			return
		}
	}

	// Soft delete: set removed_at timestamp
	result, err := h.db.Exec(`
		UPDATE group_members 
		SET removed_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND group_id = $2 AND removed_at IS NULL
	`, memberID, groupID)

	if err != nil {
		h.logger.Error("Failed to remove group member", zap.Error(err), zap.String("group_id", groupID), zap.String("member_id", memberID))
		h.respondError(w, http.StatusInternalServerError, "Failed to remove user from group")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		h.respondError(w, http.StatusNotFound, "Group member not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User removed from group successfully",
	})

	// Audit group member removal
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), userCtx.TenantID, userCtx.UserID,
			audit.AuditEventGroupMemberRemove, "groups", "remove_member",
			"User removed from group",
			map[string]interface{}{
				"group_id":  groupID,
				"member_id": memberID,
			})
	}
}

// respondError sends an error response
func (h *GroupHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}
