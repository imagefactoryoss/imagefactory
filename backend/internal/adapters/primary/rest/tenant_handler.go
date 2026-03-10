package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// TenantHandler handles tenant-related HTTP requests
type TenantHandler struct {
	tenantService       *tenant.Service
	rbacService         *rbac.Service
	ldapService         *user.LDAPService
	userService         *user.Service
	infrastructureSvc   *infrastructure.Service
	auditService        interface{} // Will be audit.Service when implemented
	logger              *zap.Logger
	notificationService *email.NotificationService
	db                  interface{} // Database connection (*sqlx.DB)
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantService *tenant.Service, rbacService *rbac.Service, ldapService *user.LDAPService, auditService interface{}, logger *zap.Logger) *TenantHandler {
	return &TenantHandler{
		tenantService:       tenantService,
		rbacService:         rbacService,
		ldapService:         ldapService,
		userService:         nil, // Will be set via SetUserService if needed
		infrastructureSvc:   nil,
		auditService:        auditService,
		logger:              logger,
		notificationService: nil, // Will be set separately if needed
		db:                  nil, // Will be set via SetDB if needed
	}
}

// SetUserService sets the user service
func (h *TenantHandler) SetUserService(userService *user.Service) {
	h.userService = userService
}

// SetInfrastructureService sets the infrastructure service used for tenant onboarding automation.
func (h *TenantHandler) SetInfrastructureService(infrastructureSvc *infrastructure.Service) {
	h.infrastructureSvc = infrastructureSvc
}

// SetDB sets the database connection
func (h *TenantHandler) SetDB(db interface{}) {
	h.db = db
}

// SetNotificationService sets the notification service
func (h *TenantHandler) SetNotificationService(notificationService *email.NotificationService) {
	h.notificationService = notificationService
}

// createDefaultGroups creates the default groups for a new tenant
func (h *TenantHandler) createDefaultGroups(tenantID uuid.UUID) (map[string]uuid.UUID, error) {
	if h.db == nil {
		h.logger.Warn("Database not available for group creation")
		return make(map[string]uuid.UUID), nil // Return empty map, don't fail
	}

	sqlxDB, ok := h.db.(*sqlx.DB)
	if !ok {
		h.logger.Warn("Database is not *sqlx.DB")
		return make(map[string]uuid.UUID), nil
	}

	groupMap := make(map[string]uuid.UUID)
	defaultGroups := []struct {
		name        string
		slug        string
		description string
		roleType    string
	}{
		{"Owners", "owners", "Tenant owners with full access", "owner"},
		{"Developers", "developers", "Developers with build and deployment permissions", "developer"},
		{"Operators", "operators", "Operators with read and operational permissions", "operator"},
		{"Viewers", "viewers", "Viewers with read-only access", "viewer"},
	}

	for _, groupDef := range defaultGroups {
		groupID := uuid.New()
		query := `
			INSERT INTO tenant_groups (id, tenant_id, name, slug, description, role_type, is_system_group, status)
			VALUES ($1, $2, $3, $4, $5, $6, true, 'active')
			ON CONFLICT (tenant_id, slug) DO NOTHING
		`

		if _, err := sqlxDB.Exec(query, groupID, tenantID, groupDef.name, groupDef.slug, groupDef.description, groupDef.roleType); err != nil {
			h.logger.Warn("Failed to create default group",
				zap.String("tenant_id", tenantID.String()),
				zap.String("group_name", groupDef.name),
				zap.Error(err))
			continue
		}

		// Fetch the actual group ID (in case of conflict, it was already created)
		var createdID uuid.UUID
		err := sqlxDB.QueryRow(
			"SELECT id FROM tenant_groups WHERE tenant_id = $1 AND slug = $2",
			tenantID, groupDef.slug,
		).Scan(&createdID)
		if err != nil {
			h.logger.Warn("Failed to fetch created group ID",
				zap.String("tenant_id", tenantID.String()),
				zap.String("group_slug", groupDef.slug),
				zap.Error(err))
			continue
		}

		groupMap[groupDef.roleType] = createdID
		h.logger.Info("Created default group",
			zap.String("tenant_id", tenantID.String()),
			zap.String("group_name", groupDef.name),
			zap.String("group_id", createdID.String()),
			zap.String("role_type", groupDef.roleType))
	}

	return groupMap, nil
}

// createDefaultRoles creates the default RBAC roles for a new tenant
// NOTE: createDefaultRoles is no longer needed - all roles are system-level
// and created in migration 022_complete_rbac_system.up.sql
// Keeping function signature for now in case it's called elsewhere, but it's a no-op

// addUserToGroup adds a user to a group
func (h *TenantHandler) addUserToGroup(groupID, userID uuid.UUID, isAdmin bool) error {
	if h.db == nil {
		h.logger.Warn("Database not available for adding user to group")
		return nil // Don't fail if DB is not available
	}

	sqlxDB, ok := h.db.(*sqlx.DB)
	if !ok {
		h.logger.Warn("Database is not *sqlx.DB")
		return nil
	}

	query := `
		INSERT INTO group_members (id, group_id, user_id, is_group_admin, added_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (group_id, user_id) DO UPDATE
		SET removed_at = NULL
	`

	memberID := uuid.New()
	if _, err := sqlxDB.Exec(query, memberID, groupID, userID, isAdmin); err != nil {
		h.logger.Error("Failed to add user to group",
			zap.String("group_id", groupID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return err
	}

	h.logger.Info("Added user to group",
		zap.String("group_id", groupID.String()),
		zap.String("user_id", userID.String()),
		zap.Bool("is_admin", isAdmin))
	return nil
}

// assignUserToTenantRole is DEPRECATED - no longer used
// Role assignment is now done via group membership only
// System-level roles are matched to groups via role_type field
// Keeping function for backward compatibility, but it's a no-op

// NewTenantHandlerWithNotification creates a new tenant handler with notification service
func NewTenantHandlerWithNotification(tenantService *tenant.Service, rbacService *rbac.Service, ldapService *user.LDAPService, auditService interface{}, logger *zap.Logger, notificationService *email.NotificationService) *TenantHandler {
	return &TenantHandler{
		tenantService:       tenantService,
		rbacService:         rbacService,
		ldapService:         ldapService,
		auditService:        auditService,
		logger:              logger,
		notificationService: notificationService,
	}
}

// respondError sends a JSON error response
func (h *TenantHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// CreateTenantRequest represents the request payload for creating a tenant
// Can be used with external tenant selection or manual entry
type CreateTenantRequest struct {
	CompanyID        string `json:"company_id" validate:"omitempty,uuid"`
	ExternalTenantID string `json:"external_tenant_id"` // UUID from external tenant service
	TenantCode       string `json:"tenant_code" validate:"required,max=8"`
	Name             string `json:"name" validate:"required"`
	Slug             string `json:"slug" validate:"required"`
	Description      string `json:"description,omitempty"`
	AdminName        string `json:"admin_name" validate:"required"`        // Tenant administrator name
	AdminEmail       string `json:"admin_email" validate:"required,email"` // Tenant administrator email
	ContactEmail     string `json:"contact_email"`                         // From external tenant
	Industry         string `json:"industry"`                              // From external tenant
	Country          string `json:"country"`                               // From external tenant
	APIRateLimit     int    `json:"api_rate_limit"`                        // Editable quota
	StorageLimit     int    `json:"storage_limit"`                         // Editable quota in GB
	MaxUsers         int    `json:"max_users"`                             // Editable quota
}

// CreateTenantResponse represents the response for tenant creation
type CreateTenantResponse struct {
	ID           string    `json:"id"`
	NumericID    string    `json:"numericId"`
	TenantCode   string    `json:"tenantCode"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description,omitempty"`
	ContactEmail string    `json:"contactEmail,omitempty"`
	Industry     string    `json:"industry,omitempty"`
	Country      string    `json:"country,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    string    `json:"createdAt"`
	UpdatedAt    string    `json:"updatedAt"`
	Version      int       `json:"version"`
	Quota        QuotaInfo `json:"quota,omitempty"`
}

type externalTenantMetadata struct {
	ContactEmail string
	Industry     string
	Country      string
}

func (h *TenantHandler) getExternalTenantMetadataByCode(tenantID uuid.UUID, tenantCode string) *externalTenantMetadata {
	if tenantCode == "" || h.db == nil {
		return nil
	}

	sqlxDB, ok := h.db.(*sqlx.DB)
	if !ok {
		return nil
	}

	var contactEmail sql.NullString
	var industry sql.NullString
	var country sql.NullString

	err := sqlxDB.QueryRow(
		`SELECT contact_email, industry, country FROM external_tenants WHERE tenant_id = $1`,
		tenantCode,
	).Scan(&contactEmail, &industry, &country)
	if err != nil {
		if err != sql.ErrNoRows {
			h.logger.Warn("Failed to fetch external tenant metadata",
				zap.String("tenant_code", tenantCode),
				zap.Error(err))
		}
	}

	// Fallback for prototype: derive contact from first active tenant user if external contact is unavailable.
	if !contactEmail.Valid && tenantID != uuid.Nil {
		var userEmail sql.NullString
		userErr := sqlxDB.QueryRow(
			`SELECT u.email
			   FROM users u
			  WHERE u.status = 'active'
			    AND (
			      EXISTS (
			        SELECT 1
			          FROM user_role_assignments ura
			         WHERE ura.user_id = u.id
			           AND ura.tenant_id = $1
			      )
			      OR EXISTS (
			        SELECT 1
			          FROM group_members gm
			          JOIN tenant_groups tg ON tg.id = gm.group_id
			         WHERE gm.user_id = u.id
			           AND gm.removed_at IS NULL
			           AND tg.status = 'active'
			           AND tg.tenant_id = $1
			      )
			    )
			  ORDER BY u.created_at ASC
			  LIMIT 1`,
			tenantID,
		).Scan(&userEmail)
		if userErr != nil && userErr != sql.ErrNoRows {
			h.logger.Warn("Failed to fetch tenant fallback contact email",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(userErr))
		} else if userEmail.Valid {
			contactEmail = userEmail
		}
	}

	// Nothing useful found.
	if !contactEmail.Valid && !industry.Valid && !country.Valid {
		return nil
	}

	return &externalTenantMetadata{
		ContactEmail: contactEmail.String,
		Industry:     industry.String,
		Country:      country.String,
	}
}

// UpdateTenantRequest represents the request payload for updating a tenant
type UpdateTenantRequest struct {
	Name        string     `json:"name,omitempty"`
	Slug        string     `json:"slug,omitempty"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status,omitempty"`
	Quota       *QuotaInfo `json:"quota,omitempty"`
}

type QuotaInfo struct {
	MaxBuilds         int     `json:"maxBuilds"`
	MaxImages         int     `json:"maxImages"`
	MaxStorageGB      float64 `json:"maxStorageGB"`
	MaxConcurrentJobs int     `json:"maxConcurrentJobs"`
}

// materializeTenantToolAvailabilityConfig creates tenant-scoped tool_availability
// from the global default when it does not yet exist.
func (h *TenantHandler) materializeTenantToolAvailabilityConfig(ctx context.Context, tenantID, actorID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return nil
	}
	if h.db == nil {
		return nil
	}

	sqlxDB, ok := h.db.(*sqlx.DB)
	if !ok {
		return nil
	}

	var exists bool
	err := sqlxDB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM system_configs
			WHERE tenant_id = $1
			  AND config_type = 'tool_settings'
			  AND config_key = 'tool_availability'
		)
	`, tenantID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	var globalConfigValue []byte
	var globalDescription string
	err = sqlxDB.QueryRowContext(ctx, `
		SELECT config_value, COALESCE(description, '')
		FROM system_configs
		WHERE tenant_id IS NULL
		  AND config_type = 'tool_settings'
		  AND config_key = 'tool_availability'
		  AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&globalConfigValue, &globalDescription)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	if actorID == uuid.Nil {
		_ = sqlxDB.QueryRowContext(ctx, `SELECT id FROM users ORDER BY created_at ASC LIMIT 1`).Scan(&actorID)
	}

	_, err = sqlxDB.ExecContext(ctx, `
		INSERT INTO system_configs (
			id, tenant_id, config_type, config_key, config_value,
			status, description, is_default, created_by, updated_by,
			created_at, updated_at, version
		) VALUES (
			$1, $2, 'tool_settings', 'tool_availability', $3,
			'active', $4, false, $5, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1
		)
	`, uuid.New(), tenantID, globalConfigValue, globalDescription, actorID)
	return err
}

// CreateTenant handles POST /tenants
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	// Extract user from context (JWT middleware)
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("No authentication context found in request")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := authCtx.UserID
	if userID == uuid.Nil {
		h.logger.Warn("Invalid user ID in authentication context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check RBAC permissions for tenant creation
	if h.rbacService != nil {
		hasPermission, err := h.rbacService.CheckUserPermission(r.Context(), userID, "tenant", "create")
		if err != nil {
			h.logger.Error("Failed to check tenant creation permission",
				zap.String("user_id", userID.String()),
				zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !hasPermission {
			h.logger.Warn("User does not have permission to create tenants",
				zap.String("user_id", userID.String()))
			http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
			return
		}
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create tenant request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.TenantCode == "" {
		h.logger.Warn("Tenant creation attempt with empty tenant code")
		h.respondError(w, http.StatusBadRequest, "Tenant code is required")
		return
	}
	if req.Name == "" {
		h.logger.Warn("Tenant creation attempt with empty name")
		h.respondError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.Slug == "" {
		h.logger.Warn("Tenant creation attempt with empty slug")
		h.respondError(w, http.StatusBadRequest, "Slug is required")
		return
	}
	if req.AdminName == "" {
		h.logger.Warn("Tenant creation attempt with empty admin name")
		h.respondError(w, http.StatusBadRequest, "Admin name is required")
		return
	}
	if req.AdminEmail == "" {
		h.logger.Warn("Tenant creation attempt with empty admin email")
		h.respondError(w, http.StatusBadRequest, "Admin email is required")
		return
	}

	// Basic tenant code validation (1-8 characters, alphanumeric)
	if len(req.TenantCode) < 1 || len(req.TenantCode) > 8 {
		h.logger.Warn("Tenant creation attempt with invalid tenant code length", zap.String("tenant_code", req.TenantCode))
		h.respondError(w, http.StatusBadRequest, "Tenant code must be between 1 and 8 characters")
		return
	}
	for _, char := range req.TenantCode {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			h.logger.Warn("Tenant creation attempt with invalid tenant code characters", zap.String("tenant_code", req.TenantCode))
			h.respondError(w, http.StatusBadRequest, "Tenant code must contain only alphanumeric characters")
			return
		}
	}

	// Basic slug validation (lowercase, alphanumeric, hyphens)
	for _, char := range req.Slug {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			h.logger.Warn("Tenant creation attempt with invalid slug format", zap.String("slug", req.Slug))
			h.respondError(w, http.StatusBadRequest, "Slug must contain only lowercase letters, numbers, and hyphens")
			return
		}
	}

	companyID := authCtx.TenantID
	if req.CompanyID != "" {
		parsedCompanyID, err := uuid.Parse(req.CompanyID)
		if err != nil {
			h.logger.Error("Invalid company ID format", zap.String("company_id", req.CompanyID), zap.Error(err))
			h.respondError(w, http.StatusBadRequest, "Invalid company ID format")
			return
		}
		companyID = parsedCompanyID
	} else {
		h.logger.Info("company_id not provided; defaulting to auth tenant",
			zap.String("company_id", companyID.String()),
			zap.String("user_id", authCtx.UserID.String()))
	}

	createdTenant, err := h.tenantService.CreateTenant(r.Context(), companyID, req.TenantCode, req.Name, req.Slug, req.Description)
	if err != nil {
		h.logger.Error("Failed to create tenant", zap.Error(err))

		// Audit tenant creation failure
		if auditSvc, ok := h.auditService.(*audit.Service); ok {
			auditSvc.LogSystemAction(r.Context(), companyID, audit.AuditEventTenantCreate, "tenants", "create",
				"Tenant creation failed", map[string]interface{}{
					"tenant_code": req.TenantCode,
					"name":        req.Name,
					"slug":        req.Slug,
					"reason":      err.Error(),
				})
		}

		if err == tenant.ErrTenantExists {
			h.respondError(w, http.StatusConflict, "Tenant already exists")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to create tenant")
		return
	}

	// Create default groups for the tenant
	groupMap, err := h.createDefaultGroups(createdTenant.ID())
	if err != nil {
		h.logger.Error("Failed to create default groups",
			zap.String("tenant_id", createdTenant.ID().String()),
			zap.Error(err))
		// Don't fail the entire request if group creation fails
	}

	if err := h.materializeTenantToolAvailabilityConfig(r.Context(), createdTenant.ID(), authCtx.UserID); err != nil {
		h.logger.Warn("Failed to materialize tenant tool availability config",
			zap.String("tenant_id", createdTenant.ID().String()),
			zap.Error(err))
	}

	if authCtx.IsSystemAdmin && h.infrastructureSvc != nil {
		triggered, asyncErr := h.infrastructureSvc.TriggerTenantNamespacePrepareForNewTenantAsync(r.Context(), createdTenant.ID(), &authCtx.UserID)
		if asyncErr != nil && asyncErr != infrastructure.ErrProviderPrepareNotConfigured {
			h.logger.Warn("Failed to auto-trigger tenant namespace prepare jobs for new tenant",
				zap.String("tenant_id", createdTenant.ID().String()),
				zap.Error(asyncErr))
		} else if triggered > 0 {
			h.logger.Info("Auto-triggered tenant namespace prepare jobs for new tenant",
				zap.String("tenant_id", createdTenant.ID().String()),
				zap.Int("provider_count", triggered))
		}
	}

	// NOTE: Default roles are now created at the system level (not per-tenant)
	// This prevents duplicate roles with the same name in different tenants
	// The system roles (Owner, Developer, Operator, Viewer) are used for all tenants
	// through the group membership mechanism via the GetUserRoles query
	/*
		// Create default roles for the tenant
		if err := h.createDefaultRoles(createdTenant.ID()); err != nil {
			h.logger.Error("Failed to create default roles",
				zap.String("tenant_id", createdTenant.ID().String()),
				zap.Error(err))
			// Don't fail the entire request if role creation fails
		}
	*/

	var adminUserID uuid.UUID
	var adminEmail string

	// Set admin email for CC (always use the requested admin email)
	adminEmail = req.AdminEmail

	// Search LDAP for the tenant admin by email
	if h.ldapService != nil && req.AdminEmail != "" {
		h.logger.Info("Searching LDAP for tenant admin", zap.String("email", req.AdminEmail))

		ldapUsers, err := h.ldapService.SearchLDAPUsers(r.Context(), req.AdminEmail, 1)
		if err != nil {
			h.logger.Warn("Failed to search LDAP for tenant admin",
				zap.String("email", req.AdminEmail),
				zap.Error(err))
		} else if len(ldapUsers) > 0 {
			ldapUser := ldapUsers[0]
			h.logger.Info("Found LDAP user for tenant admin",
				zap.String("email", req.AdminEmail),
				zap.String("username", ldapUser.Username),
				zap.String("displayName", ldapUser.DisplayName))

			// Create or update user in the system
			if h.userService != nil && h.db != nil {
				sqlxDB, ok := h.db.(*sqlx.DB)
				if ok {
					// Check if user exists, if not create them
					query := `SELECT id FROM users WHERE email = $1`
					err := sqlxDB.QueryRow(query, ldapUser.Email).Scan(&adminUserID)
					if err != nil {
						if err.Error() != "sql: no rows in result set" {
							h.logger.Error("Failed to query user by email",
								zap.String("email", ldapUser.Email),
								zap.Error(err))
						}
						// User doesn't exist, create them
						adminUserID = uuid.New()
						insertQuery := `
						INSERT INTO users (id, email, first_name, last_name, status, created_at, updated_at, version)
						VALUES ($1, $2, $3, $4, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
							ON CONFLICT (email) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
							RETURNING id
						`
						err = sqlxDB.QueryRow(insertQuery, adminUserID, ldapUser.Email, ldapUser.FirstName, ldapUser.LastName).Scan(&adminUserID)
						if err != nil {
							h.logger.Error("Failed to create user for tenant admin",
								zap.String("email", ldapUser.Email),
								zap.Error(err))
						} else {
							h.logger.Info("Created user for tenant admin",
								zap.String("user_id", adminUserID.String()),
								zap.String("email", ldapUser.Email))
						}
					} else {
						h.logger.Info("Found existing user for tenant admin",
							zap.String("user_id", adminUserID.String()),
							zap.String("email", ldapUser.Email))
					}
				}
			}

			// Assign user to the Owners group of the new tenant
			// NOTE: Group membership IS the role assignment - no separate role assignment needed
			if adminUserID != uuid.Nil {
				if adminGroupID, exists := groupMap["owner"]; exists {
					if err := h.addUserToGroup(adminGroupID, adminUserID, true); err != nil {
						h.logger.Error("Failed to add user to owners group",
							zap.String("user_id", adminUserID.String()),
							zap.String("group_id", adminGroupID.String()),
							zap.Error(err))
					}
				}
			}
		} else {
			h.logger.Warn("No LDAP user found for tenant admin email", zap.String("email", req.AdminEmail))
		}
	}

	response := CreateTenantResponse{
		ID:          createdTenant.ID().String(),
		NumericID:   strconv.Itoa(createdTenant.NumericID()),
		TenantCode:  createdTenant.TenantCode(),
		Name:        createdTenant.Name(),
		Slug:        createdTenant.Slug(),
		Description: createdTenant.Description(),
		Status:      string(createdTenant.Status()),
		CreatedAt:   createdTenant.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   createdTenant.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Version:     createdTenant.Version(),
		Quota: QuotaInfo{
			MaxBuilds:         createdTenant.Quota().MaxBuilds,
			MaxImages:         createdTenant.Quota().MaxImages,
			MaxStorageGB:      createdTenant.Quota().MaxStorageGB,
			MaxConcurrentJobs: createdTenant.Quota().MaxConcurrentJobs,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	// Send email notifications if notification service is available
	if h.notificationService != nil {
		// Send to contact email with CC to tenant admin (if admin email exists and is different)
		if req.ContactEmail != "" {
			notificationData := &email.TenantOnboardingData{
				ContactName:  req.Name,
				TenantName:   req.Name,
				TenantID:     req.TenantCode,
				Industry:     req.Industry,
				Country:      req.Country,
				ContactEmail: req.ContactEmail,
				AdminEmail:   adminEmail, // CC the admin if they have an email
				APIRateLimit: req.APIRateLimit,
				StorageLimit: req.StorageLimit,
				MaxUsers:     req.MaxUsers,
				DashboardURL: "https://app.imagefactory.local/dashboard",
			}

			// Send email with CC to tenant admin (if different from contact)
			// Email worker service on port 8081 will process every 30 seconds
			if err := h.notificationService.SendTenantOnboardingEmail(r.Context(), notificationData, createdTenant.ID()); err != nil {
				h.logger.Warn("Failed to enqueue tenant onboarding email",
					zap.Error(err),
					zap.String("tenant_id", createdTenant.ID().String()),
					zap.String("to_email", req.ContactEmail),
					zap.String("cc_email", adminEmail))
				// Don't fail the entire request if email fails - just log it
				// Email will be retried by email worker if it fails
			} else {
				h.logger.Info("Enqueued tenant onboarding email with CC",
					zap.String("tenant_id", createdTenant.ID().String()),
					zap.String("to_email", req.ContactEmail),
					zap.String("cc_email", adminEmail))
			}
		}
	}

	// Audit successful tenant creation
	if auditSvc, ok := h.auditService.(*audit.Service); ok {
		auditSvc.LogSystemAction(r.Context(), createdTenant.ID(), audit.AuditEventTenantCreate, "tenants", "create",
			"Tenant created successfully", map[string]interface{}{
				"tenant_id":   createdTenant.ID().String(),
				"tenant_code": createdTenant.TenantCode(),
				"name":        createdTenant.Name(),
				"slug":        createdTenant.Slug(),
			})
	}
}

// GetTenant handles GET /tenants/{id}
func (h *TenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	// Extract user from context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("No authentication context found in request")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := authCtx.UserID
	if userID == uuid.Nil {
		h.logger.Warn("Invalid user ID in authentication context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from URL path
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		h.logger.Error("Tenant ID not found in URL")
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Error("Invalid tenant ID", zap.String("tenant_id", tenantID), zap.Error(err))
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	tenant, err := h.tenantService.GetTenant(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get tenant", zap.String("tenant_id", tenantID), zap.Error(err))
		http.Error(w, "Failed to get tenant", http.StatusInternalServerError)
		return
	}

	// Check RBAC permissions for tenant viewing
	if h.rbacService != nil {
		hasPermission, err := h.rbacService.CheckUserPermission(r.Context(), userID, "tenant", "read")
		if err != nil {
			h.logger.Error("Failed to check tenant read permission",
				zap.String("user_id", userID.String()),
				zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !hasPermission {
			h.logger.Warn("User does not have permission to view tenants",
				zap.String("user_id", userID.String()))
			http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
			return
		}
	}

	response := CreateTenantResponse{
		ID:          tenant.ID().String(),
		NumericID:   strconv.Itoa(tenant.NumericID()),
		TenantCode:  tenant.TenantCode(),
		Name:        tenant.Name(),
		Slug:        tenant.Slug(),
		Description: tenant.Description(),
		Status:      string(tenant.Status()),
		CreatedAt:   tenant.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   tenant.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Version:     tenant.Version(),
		Quota: QuotaInfo{
			MaxBuilds:         tenant.Quota().MaxBuilds,
			MaxImages:         tenant.Quota().MaxImages,
			MaxStorageGB:      tenant.Quota().MaxStorageGB,
			MaxConcurrentJobs: tenant.Quota().MaxConcurrentJobs,
		},
	}
	if externalMeta := h.getExternalTenantMetadataByCode(tenant.ID(), tenant.TenantCode()); externalMeta != nil {
		response.ContactEmail = externalMeta.ContactEmail
		response.Industry = externalMeta.Industry
		response.Country = externalMeta.Country
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTenantBySlug handles GET /tenants/slug/{slug}
func (h *TenantHandler) GetTenantBySlug(w http.ResponseWriter, r *http.Request) {
	// Extract slug from URL path
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "Slug is required", http.StatusBadRequest)
		return
	}

	tenant, err := h.tenantService.GetTenantBySlug(r.Context(), slug)
	if err != nil {
		h.logger.Error("Failed to get tenant by slug", zap.String("slug", slug), zap.Error(err))
		http.Error(w, "Failed to get tenant", http.StatusInternalServerError)
		return
	}

	response := CreateTenantResponse{
		ID:         tenant.ID().String(),
		NumericID:  strconv.Itoa(tenant.NumericID()),
		TenantCode: tenant.TenantCode(),
		Name:       tenant.Name(),
		Slug:       tenant.Slug(),
		Status:     string(tenant.Status()),
		CreatedAt:  tenant.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  tenant.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Version:    tenant.Version(),
		Quota: QuotaInfo{
			MaxBuilds:         tenant.Quota().MaxBuilds,
			MaxImages:         tenant.Quota().MaxImages,
			MaxStorageGB:      tenant.Quota().MaxStorageGB,
			MaxConcurrentJobs: tenant.Quota().MaxConcurrentJobs,
		},
	}
	if externalMeta := h.getExternalTenantMetadataByCode(tenant.ID(), tenant.TenantCode()); externalMeta != nil {
		response.ContactEmail = externalMeta.ContactEmail
		response.Industry = externalMeta.Industry
		response.Country = externalMeta.Country
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateTenant handles PUT /tenants/{id}
func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	// Extract user from context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil {
		h.logger.Warn("Update tenant attempt without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from URL path
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		h.logger.Error("Tenant ID not found in URL")
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Error("Invalid tenant ID", zap.String("tenant_id", tenantID), zap.Error(err))
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update tenant request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert quota if provided
	var quota *tenant.ResourceQuota
	if req.Quota != nil {
		quota = &tenant.ResourceQuota{
			MaxBuilds:         req.Quota.MaxBuilds,
			MaxImages:         req.Quota.MaxImages,
			MaxStorageGB:      req.Quota.MaxStorageGB,
			MaxConcurrentJobs: req.Quota.MaxConcurrentJobs,
		}
	}

	// Update tenant
	updatedTenant, err := h.tenantService.UpdateTenant(r.Context(), id, req.Name, req.Slug, req.Description, req.Status, quota, nil)
	if err != nil {
		h.logger.Error("Failed to update tenant", zap.String("tenant_id", tenantID), zap.Error(err))

		// Audit tenant update failure
		if auditSvc, ok := h.auditService.(*audit.Service); ok {
			auditSvc.LogUserAction(r.Context(), id, authCtx.UserID, audit.AuditEventTenantUpdate, "tenants", "update",
				"Tenant update failed", map[string]interface{}{
					"tenant_id": id.String(),
					"reason":    err.Error(),
				})
		}

		h.respondError(w, http.StatusInternalServerError, "Failed to update tenant")
		return
	}

	// Audit successful tenant update
	if auditSvc, ok := h.auditService.(*audit.Service); ok {
		auditSvc.LogUserAction(r.Context(), updatedTenant.ID(), authCtx.UserID, audit.AuditEventTenantUpdate, "tenants", "update",
			"Tenant updated successfully", map[string]interface{}{
				"tenant_id":   updatedTenant.ID().String(),
				"name":        updatedTenant.Name(),
				"slug":        updatedTenant.Slug(),
				"description": updatedTenant.Description(),
			})
	}

	response := CreateTenantResponse{
		ID:          updatedTenant.ID().String(),
		NumericID:   strconv.Itoa(updatedTenant.NumericID()),
		TenantCode:  updatedTenant.TenantCode(),
		Name:        updatedTenant.Name(),
		Slug:        updatedTenant.Slug(),
		Description: updatedTenant.Description(),
		Status:      string(updatedTenant.Status()),
		CreatedAt:   updatedTenant.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   updatedTenant.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Version:     updatedTenant.Version(),
		Quota: QuotaInfo{
			MaxBuilds:         updatedTenant.Quota().MaxBuilds,
			MaxImages:         updatedTenant.Quota().MaxImages,
			MaxStorageGB:      updatedTenant.Quota().MaxStorageGB,
			MaxConcurrentJobs: updatedTenant.Quota().MaxConcurrentJobs,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ActivateTenant handles POST /tenants/{id}/activate
func (h *TenantHandler) ActivateTenant(w http.ResponseWriter, r *http.Request) {
	// Extract user from context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("No authentication context found in request")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := authCtx.UserID
	if userID == uuid.Nil {
		h.logger.Warn("Invalid user ID in authentication context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check RBAC permissions for tenant activation
	if h.rbacService != nil {
		hasPermission, err := h.rbacService.CheckUserPermission(r.Context(), userID, "tenant", "activate")
		if err != nil {
			h.logger.Error("Failed to check tenant activation permission",
				zap.String("user_id", userID.String()),
				zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !hasPermission {
			h.logger.Warn("User does not have permission to activate tenants",
				zap.String("user_id", userID.String()))
			http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
			return
		}
	}

	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	parts := strings.Split(path, "/")
	idStr := parts[0]

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid tenant ID", zap.String("tenant_id", idStr), zap.Error(err))
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	err = h.tenantService.ActivateTenant(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to activate tenant", zap.String("tenant_id", idStr), zap.Error(err))

		// Audit tenant activation failure
		if auditSvc, ok := h.auditService.(*audit.Service); ok {
			auditSvc.LogSystemAction(r.Context(), id, audit.AuditEventTenantActivate, "tenants", "activate",
				"Tenant activation failed", map[string]interface{}{
					"tenant_id": id.String(),
					"reason":    err.Error(),
				})
		}

		h.respondError(w, http.StatusInternalServerError, "Failed to activate tenant")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Tenant activated successfully"})

	// Audit successful tenant activation
	if auditSvc, ok := h.auditService.(*audit.Service); ok {
		auditSvc.LogSystemAction(r.Context(), id, audit.AuditEventTenantActivate, "tenants", "activate",
			"Tenant activated successfully", map[string]interface{}{
				"tenant_id": id.String(),
			})
	}
}

// DeleteTenant handles DELETE /tenants/{id}
func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	// Extract user from context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("No authentication context found in request")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := authCtx.UserID
	if userID == uuid.Nil {
		h.logger.Warn("Invalid user ID in authentication context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check RBAC permissions for tenant deletion
	if h.rbacService != nil {
		hasPermission, err := h.rbacService.CheckUserPermission(r.Context(), userID, "tenant", "delete")
		if err != nil {
			h.logger.Error("Failed to check tenant deletion permission",
				zap.String("user_id", userID.String()),
				zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !hasPermission {
			h.logger.Warn("User does not have permission to delete tenants",
				zap.String("user_id", userID.String()))
			http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
			return
		}
	}

	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	idStr := strings.Split(path, "/")[0]

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("Invalid tenant ID", zap.String("tenant_id", idStr), zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	// Get the tenant first to verify it exists
	tenantToDelete, err := h.tenantService.GetTenant(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get tenant for deletion", zap.String("tenant_id", idStr), zap.Error(err))
		h.respondError(w, http.StatusNotFound, "Tenant not found")
		return
	}

	// Soft delete the tenant (mark as deleted, preserves data for audit trail)
	err = h.tenantService.DeleteTenant(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete tenant", zap.String("tenant_id", idStr), zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete tenant")
		return
	}

	h.logger.Info("Tenant deleted successfully",
		zap.String("tenant_id", id.String()),
		zap.String("tenant_code", tenantToDelete.TenantCode()),
		zap.String("tenant_name", tenantToDelete.Name()))

	// Audit tenant deletion
	if auditSvc, ok := h.auditService.(*audit.Service); ok {
		auditSvc.LogSystemAction(r.Context(), tenantToDelete.ID(), audit.AuditEventTenantDelete, "tenants", "delete",
			"Tenant marked as deleted (soft delete)", map[string]interface{}{
				"tenant_id":   id.String(),
				"tenant_code": tenantToDelete.TenantCode(),
				"tenant_name": tenantToDelete.Name(),
			})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Tenant deleted successfully",
		"tenant_id":   id.String(),
		"tenant_name": tenantToDelete.Name(),
		"note":        "Tenant data is preserved and marked as deleted (soft delete)",
	})
}

// ListTenants handles GET /tenants
func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	// Get user from auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Warn("No auth context found in ListTenants")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	companyIDStr := r.URL.Query().Get("company_id")
	statusStr := r.URL.Query().Get("status")

	filter := tenant.TenantFilter{}

	// Parse limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	} else {
		filter.Limit = 50 // default
	}

	// Parse offset
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	// Parse company_id filter
	if companyIDStr != "" {
		if companyID, err := uuid.Parse(companyIDStr); err == nil {
			filter.CompanyID = &companyID
		}
	}

	// Parse status filter
	if statusStr != "" {
		status := tenant.TenantStatus(statusStr)
		// Validate status is one of the allowed values
		validStatuses := []tenant.TenantStatus{
			tenant.TenantStatusActive,
			tenant.TenantStatusSuspended,
			tenant.TenantStatusPending,
			tenant.TenantStatusDeleted,
		}
		for _, validStatus := range validStatuses {
			if status == validStatus {
				filter.Status = &status
				break
			}
		}
	}

	// Set user ID filter to get only tenants where user is an owner/administrator.
	// System admins must explicitly request all tenants with all_tenants=true.
	isSystemAdmin, err := h.rbacService.IsUserSystemAdmin(r.Context(), authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to check system admin status", zap.Error(err))
		// Don't fail the request, just proceed without the check
		filter.UserID = &authCtx.UserID
	} else if !isSystemAdmin {
		// Only filter by user ownership if NOT a system admin
		filter.UserID = &authCtx.UserID
	} else if !isAllTenantsScopeRequested(r, authCtx) {
		filter.UserID = &authCtx.UserID
	}

	tenants, err := h.tenantService.ListTenants(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list tenants", zap.Error(err))
		http.Error(w, "Failed to list tenants", http.StatusInternalServerError)
		return
	}

	var response []CreateTenantResponse
	for _, t := range tenants {
		tenantResp := CreateTenantResponse{
			ID:          t.ID().String(),
			NumericID:   strconv.Itoa(t.NumericID()),
			TenantCode:  t.TenantCode(),
			Name:        t.Name(),
			Slug:        t.Slug(),
			Description: t.Description(),
			Status:      string(t.Status()),
			CreatedAt:   t.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   t.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
			Version:     t.Version(),
			Quota: QuotaInfo{
				MaxBuilds:         t.Quota().MaxBuilds,
				MaxImages:         t.Quota().MaxImages,
				MaxStorageGB:      t.Quota().MaxStorageGB,
				MaxConcurrentJobs: t.Quota().MaxConcurrentJobs,
			},
		}
		if externalMeta := h.getExternalTenantMetadataByCode(t.ID(), t.TenantCode()); externalMeta != nil {
			tenantResp.ContactEmail = externalMeta.ContactEmail
			tenantResp.Industry = externalMeta.Industry
			tenantResp.Country = externalMeta.Country
		}

		response = append(response, tenantResp)
	}

	// Return paginated response structure
	paginatedResp := map[string]interface{}{
		"data": response,
		"pagination": map[string]interface{}{
			"page":       1,
			"limit":      filter.Limit,
			"total":      len(tenants),
			"totalPages": 1,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedResp)
}
