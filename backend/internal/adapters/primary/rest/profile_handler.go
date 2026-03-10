package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

// ProfileHandler handles user profile HTTP requests
type ProfileHandler struct {
	db           *sqlx.DB
	logger       *zap.Logger
	auditService *audit.Service
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(db *sqlx.DB, auditService *audit.Service, logger *zap.Logger) *ProfileHandler {
	return &ProfileHandler{
		db:           db,
		logger:       logger,
		auditService: auditService,
	}
}

// ProfileResponse represents the user profile response
type ProfileResponse struct {
	ID             string                    `json:"id"`
	Email          string                    `json:"email"`
	FirstName      string                    `json:"first_name"`
	LastName       string                    `json:"last_name"`
	Status         string                    `json:"status"`
	IsActive       bool                      `json:"is_active"`
	IsSystemAdmin  bool                      `json:"is_system_admin"`
	CanAccessAdmin bool                      `json:"can_access_admin"`
	DefaultLanding string                    `json:"default_landing_route"`
	Groups         []GroupProfileResponse    `json:"groups"`
	RolesByTenant  map[string][]RoleResponse `json:"roles_by_tenant,omitempty"`
	TenantNames    map[string]string         `json:"tenant_names,omitempty"`
	Preferences    map[string]interface{}    `json:"preferences,omitempty"`
	Avatar         *string                   `json:"avatar,omitempty"`
	CreatedAt      string                    `json:"created_at,omitempty"`
	HasMultiTenant bool                      `json:"has_multi_tenant"`
}

// GroupProfileResponse represents a group in profile response
type GroupProfileResponse struct {
	ID       string `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	RoleType string `json:"role_type" db:"role_type"`
	TenantID string `json:"tenant_id,omitempty" db:"tenant_id"`
	IsAdmin  bool   `json:"is_admin" db:"is_group_admin"`
}

// GetProfile handles GET /api/v1/profile
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get auth context from middleware
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Auth context not found")
		return
	}

	// Fetch user with roles
	query := `
		SELECT 
			u.id, 
			u.email, 
			u.first_name, 
			u.last_name, 
			u.status, 
			CASE WHEN u.status = 'active' THEN true ELSE false END as is_active,
			u.created_at,
			COALESCE(upp.preferences, '{}'::jsonb) AS profile_preferences
		FROM users u
		LEFT JOIN user_profile_preferences upp ON upp.user_id = u.id
		WHERE u.id = $1
	`

	var user struct {
		ID                 string          `db:"id"`
		Email              string          `db:"email"`
		FirstName          string          `db:"first_name"`
		LastName           string          `db:"last_name"`
		Status             string          `db:"status"`
		IsActive           bool            `db:"is_active"`
		CreatedAt          string          `db:"created_at"`
		ProfilePreferences json.RawMessage `db:"profile_preferences"`
	}

	if err := h.db.QueryRowx(query, authCtx.UserID).StructScan(&user); err != nil {
		h.logger.Error("Failed to fetch user profile", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch profile")
		return
	}

	// Fetch user groups (which define user roles in the system)
	groupsQuery := `
		SELECT 
			tg.id,
			tg.name,
			tg.role_type,
			tg.tenant_id,
			gm.is_group_admin
		FROM tenant_groups tg
		INNER JOIN group_members gm ON tg.id = gm.group_id
		WHERE gm.user_id = $1 AND gm.removed_at IS NULL AND tg.status = 'active'
		ORDER BY tg.name ASC
	`

	groups := []GroupProfileResponse{}
	if err := h.db.Select(&groups, groupsQuery, authCtx.UserID); err != nil {
		h.logger.Error("Failed to fetch user groups", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		// Continue without groups - not a fatal error
		groups = []GroupProfileResponse{}
	}

	// Initialize tenant names map early
	tenantNames := make(map[string]string)

	// Build tenant names map from groups - ensure all group tenants have names
	for _, group := range groups {
		if group.TenantID != "" && tenantNames[group.TenantID] == "" {
			// Need to fetch tenant name for this group
			var tenantName string
			tenantQuery := `SELECT name FROM tenants WHERE id = $1 AND status IN ('active', 'pending')`
			if err := h.db.QueryRow(tenantQuery, group.TenantID).Scan(&tenantName); err != nil {
				tenantName = "Tenant " + group.TenantID[:8] + "..."
			}
			tenantNames[group.TenantID] = tenantName
		}
	}

	// Fetch per-tenant role assignments with tenant names (rolesByTenant)
	rolesByTenantQuery := `
		SELECT tenant_id, tenant_name, id, name, description, is_system
		FROM (
			-- Direct tenant-scoped role assignments
			SELECT DISTINCT
				ura.tenant_id::text as tenant_id,
				CASE
					WHEN t.name IS NOT NULL THEN t.name
					ELSE 'Tenant ' || substring(ura.tenant_id::text, 1, 8) || '...'
				END as tenant_name,
				r.id,
				r.name,
				r.description,
				COALESCE(r.is_system, false) as is_system
			FROM user_role_assignments ura
			INNER JOIN rbac_roles r ON ura.role_id = r.id
			LEFT JOIN tenants t ON ura.tenant_id = t.id AND t.status IN ('active', 'pending')
			WHERE ura.user_id = $1
			  AND ura.tenant_id IS NOT NULL

			UNION

			-- Group-based tenant roles (map role_type to system-level role)
			SELECT DISTINCT
				tg.tenant_id::text as tenant_id,
				CASE
					WHEN t.name IS NOT NULL THEN t.name
					ELSE 'Tenant ' || substring(tg.tenant_id::text, 1, 8) || '...'
				END as tenant_name,
				r.id,
				r.name,
				r.description,
				COALESCE(r.is_system, false) as is_system
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			INNER JOIN rbac_roles r ON REPLACE(LOWER(r.name), ' ', '_') = tg.role_type AND r.tenant_id IS NULL
			LEFT JOIN tenants t ON tg.tenant_id = t.id AND t.status IN ('active', 'pending')
			WHERE gm.user_id = $1
			  AND gm.removed_at IS NULL
			  AND tg.status = 'active'
			  AND tg.tenant_id IS NOT NULL
		) tenant_roles
		ORDER BY tenant_id, name ASC
	`

	type RoleByTenantRow struct {
		TenantID    string `db:"tenant_id"`
		TenantName  string `db:"tenant_name"`
		ID          string `db:"id"`
		Name        string `db:"name"`
		Description string `db:"description"`
		IsSystem    bool   `db:"is_system"`
	}

	rolesByTenantRows := []RoleByTenantRow{}
	rolesByTenant := make(map[string][]RoleResponse)

	if err := h.db.Select(&rolesByTenantRows, rolesByTenantQuery, authCtx.UserID); err != nil {
		h.logger.Warn("Failed to fetch user role assignments", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		// Continue without role assignments - not a fatal error
	} else {
		for _, row := range rolesByTenantRows {
			rolesByTenant[row.TenantID] = append(rolesByTenant[row.TenantID], RoleResponse{
				ID:          row.ID,
				Name:        row.Name,
				Description: row.Description,
				IsSystem:    row.IsSystem,
			})
			// Store tenant name mapping (query provides proper defaults)
			tenantNames[row.TenantID] = row.TenantName
		}
		h.logger.Info("Built tenant names map",
			zap.String("user_id", authCtx.UserID.String()),
			zap.Int("tenant_count", len(tenantNames)),
			zap.Any("tenant_ids", mapKeys(tenantNames)),
		)
	}

	// Calculate has_multi_tenant based on actual roles_by_tenant data
	hasMultiTenant := len(rolesByTenant) > 1

	preferences := map[string]interface{}{}
	if len(user.ProfilePreferences) > 0 {
		_ = json.Unmarshal(user.ProfilePreferences, &preferences)
	}

	profile := ProfileResponse{
		ID:             user.ID,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Status:         user.Status,
		IsActive:       user.IsActive,
		IsSystemAdmin:  authCtx.IsSystemAdmin,
		CanAccessAdmin: canAccessAdminConsole(authCtx.IsSystemAdmin, groups),
		DefaultLanding: deriveDefaultLandingRoute(
			canAccessAdminConsole(authCtx.IsSystemAdmin, groups),
			hasSecurityReviewerRole(groups, rolesByTenant),
			len(rolesByTenant) > 0 || len(groups) > 0,
		),
		Groups:         groups,
		RolesByTenant:  rolesByTenant,
		TenantNames:    tenantNames,
		Preferences:    preferences,
		Avatar:         nil,
		CreatedAt:      user.CreatedAt,
		HasMultiTenant: hasMultiTenant,
	}

	h.logger.Info("Returning profile response",
		zap.String("user_id", authCtx.UserID.String()),
		zap.Int("role_by_tenant_count", len(rolesByTenant)),
		zap.Int("tenant_names_count", len(tenantNames)),
		zap.Any("tenant_names", tenantNames),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(profile)
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName   string                 `json:"first_name,omitempty"`
	LastName    string                 `json:"last_name,omitempty"`
	Avatar      string                 `json:"avatar,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

// UpdateProfile handles PUT /api/v1/profile
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get auth context from middleware
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Auth context not found")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update profile request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build update query for user-core fields (preferences are stored separately).
	updateQuery := `UPDATE users SET `
	args := []interface{}{}
	argIndex := 1
	hasUserFieldUpdate := false

	if req.FirstName != "" {
		if argIndex > 1 {
			updateQuery += ", "
		}
		updateQuery += "first_name = $" + strconv.Itoa(argIndex)
		args = append(args, req.FirstName)
		argIndex++
		hasUserFieldUpdate = true
	}

	if req.LastName != "" {
		if argIndex > 1 {
			updateQuery += ", "
		}
		updateQuery += "last_name = $" + strconv.Itoa(argIndex)
		args = append(args, req.LastName)
		argIndex++
		hasUserFieldUpdate = true
	}

	if req.Avatar != "" {
		if argIndex > 1 {
			updateQuery += ", "
		}
		updateQuery += "avatar = $" + strconv.Itoa(argIndex)
		args = append(args, req.Avatar)
		argIndex++
		hasUserFieldUpdate = true
	}

	if !hasUserFieldUpdate && req.Preferences == nil {
		h.respondError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	if hasUserFieldUpdate {
		updateQuery += ", updated_at = CURRENT_TIMESTAMP WHERE id = $" + strconv.Itoa(argIndex)
		args = append(args, authCtx.UserID)

		if _, err := h.db.Exec(updateQuery, args...); err != nil {
			h.logger.Error("Failed to update profile", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to update profile")
			return
		}
	}

	if req.Preferences != nil {
		preferencesJSON, marshalErr := json.Marshal(req.Preferences)
		if marshalErr != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid preferences payload")
			return
		}
		prefQuery := `
			INSERT INTO user_profile_preferences (user_id, preferences, created_at, updated_at)
			VALUES ($1, $2::jsonb, NOW(), NOW())
			ON CONFLICT (user_id)
			DO UPDATE SET preferences = EXCLUDED.preferences, updated_at = NOW()
		`
		if _, err := h.db.Exec(prefQuery, authCtx.UserID, preferencesJSON); err != nil {
			h.logger.Error("Failed to update profile preferences", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to update profile")
			return
		}
	}

	// Fetch and return updated profile
	query := `
		SELECT 
			u.id, 
			u.email, 
			u.first_name, 
			u.last_name, 
			u.status, 
			CASE WHEN u.status = 'active' THEN true ELSE false END as is_active,
			u.created_at,
			COALESCE(upp.preferences, '{}'::jsonb) AS profile_preferences
		FROM users u
		LEFT JOIN user_profile_preferences upp ON upp.user_id = u.id
		WHERE u.id = $1
	`

	var user struct {
		ID                 string          `db:"id"`
		Email              string          `db:"email"`
		FirstName          string          `db:"first_name"`
		LastName           string          `db:"last_name"`
		Status             string          `db:"status"`
		IsActive           bool            `db:"is_active"`
		CreatedAt          string          `db:"created_at"`
		ProfilePreferences json.RawMessage `db:"profile_preferences"`
	}

	if err := h.db.QueryRowx(query, authCtx.UserID).StructScan(&user); err != nil {
		h.logger.Error("Failed to fetch user profile", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch profile")
		return
	}

	// Fetch user groups (which define user roles in the system)
	groupsQuery := `
		SELECT 
			tg.id,
			tg.name,
			tg.role_type,
			tg.tenant_id,
			gm.is_group_admin
		FROM tenant_groups tg
		INNER JOIN group_members gm ON tg.id = gm.group_id
		WHERE gm.user_id = $1 AND gm.removed_at IS NULL AND tg.status = 'active'
		ORDER BY tg.name ASC
	`

	groupsUpdate := []GroupProfileResponse{}
	if err := h.db.Select(&groupsUpdate, groupsQuery, authCtx.UserID); err != nil {
		h.logger.Error("Failed to fetch user groups", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		groupsUpdate = []GroupProfileResponse{}
	}

	preferencesUpdate := map[string]interface{}{}
	if len(user.ProfilePreferences) > 0 {
		_ = json.Unmarshal(user.ProfilePreferences, &preferencesUpdate)
	}

	profile := ProfileResponse{
		ID:             user.ID,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Status:         user.Status,
		IsActive:       user.IsActive,
		IsSystemAdmin:  authCtx.IsSystemAdmin,
		CanAccessAdmin: canAccessAdminConsole(authCtx.IsSystemAdmin, groupsUpdate),
		DefaultLanding: deriveDefaultLandingRoute(
			canAccessAdminConsole(authCtx.IsSystemAdmin, groupsUpdate),
			hasSecurityReviewerRole(groupsUpdate, nil),
			len(groupsUpdate) > 0,
		),
		Groups:         groupsUpdate,
		Preferences:    preferencesUpdate,
		CreatedAt:      user.CreatedAt,
		HasMultiTenant: authCtx.HasMultiTenant,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(profile)

	// Audit profile update
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID,
			audit.AuditEventProfileUpdate, "profile", "update",
			"User profile updated",
			map[string]interface{}{
				"user_id": authCtx.UserID.String(),
				"fields_updated": map[string]interface{}{
					"first_name":  req.FirstName != "",
					"last_name":   req.LastName != "",
					"avatar":      req.Avatar != "",
					"preferences": req.Preferences != nil,
				},
			})
	}
}

// Helper function to get keys from string map for logging
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func canAccessAdminConsole(isSystemAdmin bool, groups []GroupProfileResponse) bool {
	if isSystemAdmin {
		return true
	}
	for _, group := range groups {
		if group.RoleType == "system_administrator_viewer" {
			return true
		}
	}
	return false
}

func hasSecurityReviewerRole(groups []GroupProfileResponse, rolesByTenant map[string][]RoleResponse) bool {
	for _, group := range groups {
		if normalizeRoleKey(group.RoleType) == "security_reviewer" {
			return true
		}
	}

	for _, roles := range rolesByTenant {
		for _, role := range roles {
			if normalizeRoleKey(role.Name) == "security_reviewer" {
				return true
			}
		}
	}

	return false
}

func deriveDefaultLandingRoute(canAccessAdmin bool, hasSecurityReviewer bool, hasAnyTenantAccess bool) string {
	if canAccessAdmin {
		return "/admin/dashboard"
	}
	if hasSecurityReviewer {
		return "/reviewer/dashboard"
	}
	if hasAnyTenantAccess {
		return "/dashboard"
	}
	return "/no-access"
}

func normalizeRoleKey(raw string) string {
	if raw == "" {
		return ""
	}
	key := make([]rune, 0, len(raw))
	previousUnderscore := false
	for _, ch := range raw {
		switch {
		case ch >= 'A' && ch <= 'Z':
			ch = ch + 32
			key = append(key, ch)
			previousUnderscore = false
		case (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9'):
			key = append(key, ch)
			previousUnderscore = false
		case ch == ' ' || ch == '-' || ch == '_':
			if !previousUnderscore {
				key = append(key, '_')
				previousUnderscore = true
			}
		}
	}
	// trim leading/trailing underscore
	for len(key) > 0 && key[0] == '_' {
		key = key[1:]
	}
	for len(key) > 0 && key[len(key)-1] == '_' {
		key = key[:len(key)-1]
	}
	return string(key)
}

// respondError sends a JSON error response
func (h *ProfileHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
