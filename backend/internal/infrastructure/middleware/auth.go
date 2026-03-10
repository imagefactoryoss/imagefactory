package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
)

// cacheEntry stores cache data with expiration time
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// authCache caches authentication-related data
type authCache struct {
	mu          sync.RWMutex
	adminCache  map[uuid.UUID]cacheEntry
	multiTenant map[uuid.UUID]cacheEntry
	ttl         time.Duration
}

func newAuthCache(ttl time.Duration) *authCache {
	return &authCache{
		adminCache:  make(map[uuid.UUID]cacheEntry),
		multiTenant: make(map[uuid.UUID]cacheEntry),
		ttl:         ttl,
	}
}

func (c *authCache) getAdminStatus(userID uuid.UUID) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.adminCache[userID]
	if found && time.Now().Before(entry.expiresAt) {
		return entry.value.(bool), true
	}
	return false, false
}

func (c *authCache) setAdminStatus(userID uuid.UUID, isAdmin bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.adminCache[userID] = cacheEntry{
		value:     isAdmin,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *authCache) getMultiTenantStatus(userID uuid.UUID) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.multiTenant[userID]
	if found && time.Now().Before(entry.expiresAt) {
		return entry.value.(bool), true
	}
	return false, false
}

func (c *authCache) setMultiTenantStatus(userID uuid.UUID, hasMultiTenant bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.multiTenant[userID] = cacheEntry{
		value:     hasMultiTenant,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// AuthContext represents the authentication context
type AuthContext struct {
	UserID         uuid.UUID   // User ID from JWT
	TenantID       uuid.UUID   // Selected tenant for this request (from header or default)
	UserTenants    []uuid.UUID // All tenants the user has roles in (authoritative source via RBAC)
	Email          string      // User email from JWT
	HasMultiTenant bool        // Whether user has access to multiple tenants
	IsSystemAdmin  bool        // Whether user has system admin role
}

// PrimaryTenant returns a sensible single-tenant context for callers that expect a single tenant.
// Logic: prefer the request-selected TenantID if present and the user has access to it; otherwise
// return the single tenant from UserTenants when only one is available; otherwise return uuid.Nil.
func (a *AuthContext) PrimaryTenant() uuid.UUID {
	if a == nil {
		return uuid.Nil
	}
	if a.TenantID != uuid.Nil {
		for _, t := range a.UserTenants {
			if t == a.TenantID {
				return a.TenantID
			}
		}
	}
	if len(a.UserTenants) == 1 {
		return a.UserTenants[0]
	}
	return uuid.Nil
}

// HasTenant reports whether the user has an assignment for the given tenant.
func (a *AuthContext) HasTenant(tenant uuid.UUID) bool {
	if a == nil || tenant == uuid.Nil {
		return false
	}
	for _, t := range a.UserTenants {
		if t == tenant {
			return true
		}
	}
	return false
}

func (m *AuthMiddleware) getUserTenants(ctx context.Context, userID uuid.UUID) []uuid.UUID {
	userTenants := []uuid.UUID{}
	rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository)
	if !ok {
		return userTenants
	}

	assignments, err := rbacRepo.FindUserRoleAssignmentsByUser(ctx, userID)
	if err != nil {
		return userTenants
	}

	for tid := range assignments {
		if tid != uuid.Nil {
			userTenants = append(userTenants, tid)
		}
	}
	return userTenants
}

func (m *AuthMiddleware) getUserPrimaryTenant(ctx context.Context, userID uuid.UUID) uuid.UUID {
	userTenants := m.getUserTenants(ctx, userID)
	if len(userTenants) == 1 {
		return userTenants[0]
	}
	return uuid.Nil
}

// userTokenValidator is a minimal interface for validating access tokens.
// Using an interface here makes the middleware easier to unit-test with mocks.
type userTokenValidator interface {
	ValidateToken(ctx context.Context, accessToken string) (*user.User, error)
}

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	authService         userTokenValidator
	rbacRepo            interface{} // RBAC repository for tenant access validation
	auditService        *audit.Service
	logger              *zap.Logger
	systemConfigService *systemconfig.Service
	bootstrapService    BootstrapStatusService
	cache               *authCache // Cache for authentication-related data
}

// BootstrapStatusService provides setup-required status checks for first-run gating.
type BootstrapStatusService interface {
	IsSetupRequired(ctx context.Context) (bool, error)
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService userTokenValidator, rbacRepo interface{}, auditService *audit.Service, logger *zap.Logger, systemConfigService *systemconfig.Service) *AuthMiddleware {
	return &AuthMiddleware{
		authService:         authService,
		rbacRepo:            rbacRepo,
		auditService:        auditService,
		logger:              logger,
		systemConfigService: systemConfigService,
		cache:               newAuthCache(5 * time.Minute), // Cache for 5 minutes
	}
}

// SetBootstrapService sets first-run bootstrap status service for request gating.
func (m *AuthMiddleware) SetBootstrapService(service BootstrapStatusService) {
	m.bootstrapService = service
}

// Authenticate is middleware that validates JWT tokens and adds user context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isWSUpgrade := isWebSocketUpgradeRequest(r)
		// Extract token from Authorization header (standard) or query parameter for WebSocket upgrades.
		authHeader := r.Header.Get("Authorization")
		token := ""
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				m.logger.Debug("Invalid authorization header format")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token = parts[1]
		} else if isWebSocketUpgradeRequest(r) {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if token == "" {
			m.logger.Debug("No authentication token provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token
		if !isWSUpgrade {
			m.logger.Info("Validating token for request", zap.String("path", r.URL.Path), zap.String("method", r.Method))
		}
		usr, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			if isExpectedTokenValidationFailure(err) {
				m.logger.Info("Token rejected", zap.Error(err), zap.String("path", r.URL.Path), zap.String("method", r.Method))
			} else {
				m.logger.Warn("Token validation failed", zap.Error(err), zap.String("path", r.URL.Path), zap.String("method", r.Method))
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !isWSUpgrade {
			m.logger.Info("Token validation successful", zap.String("user_id", usr.ID().String()), zap.String("email", usr.Email()), zap.String("path", r.URL.Path))
		}

		if m.isPasswordChangeGateBlocked(r, usr) {
			http.Error(w, "Password change required before accessing this endpoint", http.StatusPreconditionRequired)
			return
		}

		// Extract tenant ID from header (required for HTTP; query parameter allowed for WebSocket upgrades)
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" && isWebSocketUpgradeRequest(r) {
			tenantIDStr = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		}
		var selectedTenantID uuid.UUID
		var hasMultiTenant bool

		// Check if user is a system admin (member of system_administrator group)
		isSystemAdmin, isCached := m.cache.getAdminStatus(usr.ID())
		if !isCached {
			if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
				isAdmin, err := rbacRepo.IsUserSystemAdmin(r.Context(), usr.ID())
				if err != nil {
					m.logger.Warn("Failed to check system admin status",
						zap.String("user_id", usr.ID().String()),
						zap.Error(err),
					)
				} else {
					isSystemAdmin = isAdmin
					m.cache.setAdminStatus(usr.ID(), isAdmin)
				}
			}
		}

		// Tenant ID is always required for authenticated HTTP requests. Treat
		// missing or nil tenant IDs as a client error regardless of role. This
		// avoids implicit system-scoped behavior that can be abused.
		if tenantIDStr == "" {
			m.logger.Warn("Missing required X-Tenant-ID header",
				zap.String("user_id", usr.ID().String()),
				zap.String("email", usr.Email()),
			)
			http.Error(w, "Tenant context required (set X-Tenant-ID)", http.StatusBadRequest)
			return
		}

		// Parse tenant ID from header
		selectedTenantIDParsed, parseErr := uuid.Parse(tenantIDStr)
		if parseErr != nil {
			m.logger.Warn("Invalid X-Tenant-ID header format",
				zap.String("tenant_id", tenantIDStr),
				zap.String("user_id", usr.ID().String()),
			)
			http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
			return
		}
		selectedTenantID = selectedTenantIDParsed

		if selectedTenantID == uuid.Nil {
			m.logger.Warn("Nil tenant UUID is not allowed",
				zap.String("user_id", usr.ID().String()),
				zap.String("email", usr.Email()),
			)
			http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
			return
		}

		// Regular tenant - validate user has access (system admins can access any tenant explicitly)
		if !isSystemAdmin {
			if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
				hasAccess, err := rbacRepo.UserHasTenantAccess(r.Context(), usr.ID(), selectedTenantID)
				if err != nil {
					m.logger.Error("Failed to validate tenant access",
						zap.String("user_id", usr.ID().String()),
						zap.String("tenant_id", selectedTenantID.String()),
						zap.Error(err),
					)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				if !hasAccess {
					m.logger.Warn("User attempted to access unauthorized tenant",
						zap.String("user_id", usr.ID().String()),
						zap.String("tenant_id", selectedTenantID.String()),
					)
					http.Error(w, "Forbidden: No access to tenant", http.StatusForbidden)
					return
				}
			}
		}

		// Tenant context must always be explicit and non-nil for authenticated requests.
		// Treat nil tenant for any authenticated request as invalid.
		if selectedTenantID == uuid.Nil {
			m.logger.Warn("Authenticated request missing valid tenant context",
				zap.String("user_id", usr.ID().String()),
				zap.String("email", usr.Email()),
				zap.Bool("is_system_admin", isSystemAdmin),
			)
			http.Error(w, "Tenant context required (set X-Tenant-ID)", http.StatusBadRequest)
			return
		}

		// Check if user has multi-tenant access (has roles in multiple tenants)
		cachedMultiTenant, isCached := m.cache.getMultiTenantStatus(usr.ID())
		if isCached {
			hasMultiTenant = cachedMultiTenant
		} else {
			if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
				userRoles, err := rbacRepo.FindUserRoles(r.Context(), usr.ID())
				if err == nil {
					// Count unique tenants from roles
					tenantSet := make(map[uuid.UUID]bool)
					for _, role := range userRoles {
						if role.TenantID() != uuid.Nil {
							tenantSet[role.TenantID()] = true
						}
					}
					hasMultiTenant = len(tenantSet) > 1
					m.cache.setMultiTenantStatus(usr.ID(), hasMultiTenant)
				}
			}
		}

		// Build authoritative user tenant list from RBAC (user_role_assignments)
		userTenants := m.getUserTenants(r.Context(), usr.ID())

		// Create auth context (use RBAC-derived tenant list instead of legacy user.tenant_id)
		authCtx := &AuthContext{
			UserID:         usr.ID(),
			TenantID:       selectedTenantID,
			UserTenants:    userTenants,
			Email:          usr.Email(),
			HasMultiTenant: hasMultiTenant,
			IsSystemAdmin:  isSystemAdmin,
		}

		// Add auth context to request context
		ctx := context.WithValue(r.Context(), "auth", authCtx)
		r = r.WithContext(ctx)

		if m.isSetupGateBlocked(r, authCtx) {
			http.Error(w, "Initial system setup is required before accessing this endpoint", http.StatusConflict)
			return
		}

		// Log successful authentication
		logTenantIDStr := "none"
		if selectedTenantID != uuid.Nil {
			logTenantIDStr = selectedTenantID.String()
		}
		if !isWSUpgrade {
			m.logger.Info("User authenticated successfully",
				zap.String("user_id", usr.ID().String()),
				zap.String("tenant_id", logTenantIDStr),
				zap.String("email", usr.Email()))
		}

		// Continue with next handler
		next.ServeHTTP(w, r)
	})
}

// RequirePermission is middleware that checks if the authenticated user has required permissions
// It validates the permission definition and then enforces role/group-based permissions.
func (m *AuthMiddleware) RequirePermission(rbacService interface{}, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Chain with Authenticate middleware to ensure token is validated first
		return m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get auth context
			authCtx, ok := r.Context().Value("auth").(*AuthContext)
			if !ok {
				m.logger.Debug("No auth context found after authentication")
				// Audit permission denial due to missing auth context
				if m.auditService != nil {
					m.auditService.LogSystemAction(r.Context(), uuid.Nil, audit.AuditEventPermissionDenied, resource, action,
						"Permission denied: missing authentication context",
						map[string]interface{}{
							"resource":   resource,
							"action":     action,
							"path":       r.URL.Path,
							"method":     r.Method,
							"reason":     "missing_auth_context",
							"client_ip":  r.RemoteAddr,
							"user_agent": r.UserAgent(),
						})
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if m.isWriteBlockedByMaintenance(r, authCtx) {
				http.Error(w, "Service temporarily read-only for maintenance", http.StatusServiceUnavailable)
				return
			}

			// Type assert to get PermissionService
			permService, ok := rbacService.(*rbac.PermissionService)
			if !ok {
				m.logger.Error("Invalid RBAC service type",
					zap.String("user_id", authCtx.UserID.String()),
					zap.String("resource", resource),
					zap.String("action", action))
				// Audit permission denial due to invalid RBAC service
				if m.auditService != nil {
					m.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionDenied, resource, action,
						"Permission denied: invalid RBAC service configuration",
						map[string]interface{}{
							"resource":   resource,
							"action":     action,
							"path":       r.URL.Path,
							"method":     r.Method,
							"reason":     "invalid_rbac_service",
							"user_id":    authCtx.UserID.String(),
							"client_ip":  r.RemoteAddr,
							"user_agent": r.UserAgent(),
						})
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Validate that the permission exists in the system
			perm, err := permService.FindPermissionByResourceAction(r.Context(), resource, action)
			if err != nil {
				m.logger.Error("Failed to validate permission",
					zap.String("user_id", authCtx.UserID.String()),
					zap.String("resource", resource),
					zap.String("action", action),
					zap.Error(err))
				// Audit permission denial due to validation error
				if m.auditService != nil {
					m.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionDenied, resource, action,
						"Permission denied: validation error",
						map[string]interface{}{
							"resource":   resource,
							"action":     action,
							"path":       r.URL.Path,
							"method":     r.Method,
							"reason":     "validation_error",
							"error":      err.Error(),
							"user_id":    authCtx.UserID.String(),
							"client_ip":  r.RemoteAddr,
							"user_agent": r.UserAgent(),
						})
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Check if permission exists
			if perm == nil {
				m.logger.Warn("Permission not found",
					zap.String("user_id", authCtx.UserID.String()),
					zap.String("resource", resource),
					zap.String("action", action))
				// Audit permission denial due to non-existent permission
				if m.auditService != nil {
					m.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionDenied, resource, action,
						"Permission denied: permission does not exist",
						map[string]interface{}{
							"resource":   resource,
							"action":     action,
							"path":       r.URL.Path,
							"method":     r.Method,
							"reason":     "permission_not_found",
							"user_id":    authCtx.UserID.String(),
							"client_ip":  r.RemoteAddr,
							"user_agent": r.UserAgent(),
						})
				}
				http.Error(w, fmt.Sprintf("Invalid permission: %s:%s", resource, action), http.StatusForbidden)
				return
			}

			// Log successful permission validation (skip noisy websocket upgrades)
			if !isWebSocketUpgradeRequest(r) {
				m.logger.Info("Permission validated",
					zap.String("user_id", authCtx.UserID.String()),
					zap.String("resource", resource),
					zap.String("action", action),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method))
			}

			// Check if user has required permission via role/group assignments.
			// This applies to both system-level and tenant-level permissions.
			if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
				hasPermission, err := rbacRepo.UserHasPermission(r.Context(), authCtx.UserID, resource, action)
				if err != nil {
					m.logger.Error("Failed to check user permission",
						zap.String("user_id", authCtx.UserID.String()),
						zap.String("resource", resource),
						zap.String("action", action),
						zap.Error(err))
					// Audit permission denial due to check error
					if m.auditService != nil {
						m.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionDenied, resource, action,
							"Permission denied: permission check error",
							map[string]interface{}{
								"resource":   resource,
								"action":     action,
								"path":       r.URL.Path,
								"method":     r.Method,
								"reason":     "permission_check_error",
								"error":      err.Error(),
								"user_id":    authCtx.UserID.String(),
								"client_ip":  r.RemoteAddr,
								"user_agent": r.UserAgent(),
							})
					}
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}

				if !hasPermission {
					m.logger.Warn("User lacks required permission",
						zap.String("user_id", authCtx.UserID.String()),
						zap.String("email", authCtx.Email),
						zap.String("resource", resource),
						zap.String("action", action))
					// Audit permission denial
					if m.auditService != nil {
						m.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventPermissionDenied, resource, action,
							"Permission denied: insufficient role permissions",
							map[string]interface{}{
								"resource":   resource,
								"action":     action,
								"path":       r.URL.Path,
								"method":     r.Method,
								"reason":     "insufficient_role_permissions",
								"user_id":    authCtx.UserID.String(),
								"client_ip":  r.RemoteAddr,
								"user_agent": r.UserAgent(),
							})
					}
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			// Add resource and action to context for handler access
			ctx := context.WithValue(r.Context(), "permission_resource", resource)
			ctx = context.WithValue(ctx, "permission_action", action)
			r = r.WithContext(ctx)

			// Continue with next handler
			next.ServeHTTP(w, r)
		}))
	}
}

func (m *AuthMiddleware) isWriteBlockedByMaintenance(r *http.Request, authCtx *AuthContext) bool {
	if m.systemConfigService == nil {
		return false
	}
	if authCtx == nil {
		return false
	}
	if authCtx.IsSystemAdmin {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	config, err := m.systemConfigService.GetConfigByTypeAndKey(r.Context(), nil, systemconfig.ConfigTypeGeneral, "general")
	if err != nil {
		return false
	}

	generalConfig, err := config.GetGeneralConfig()
	if err != nil {
		return false
	}

	return generalConfig.MaintenanceMode
}

// OptionalAuth is middleware that adds auth context if token is present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No token provided, continue without auth context
			next.ServeHTTP(w, r)
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue without auth context
			next.ServeHTTP(w, r)
			return
		}

		token := parts[1]

		// Validate token
		usr, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			// Token invalid, continue without auth context
			m.logger.Debug("Optional token validation failed", zap.Error(err))
			next.ServeHTTP(w, r)
			return
		}

		if m.isPasswordChangeGateBlocked(r, usr) {
			http.Error(w, "Password change required before accessing this endpoint", http.StatusPreconditionRequired)
			return
		}

		// Check if user is a system admin (member of system_administrator group)
		isSystemAdmin := false
		if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
			isAdmin, err := rbacRepo.IsUserSystemAdmin(r.Context(), usr.ID())
			if err == nil {
				isSystemAdmin = isAdmin
			}
		}

		// Derive a default tenant from RBAC assignments (single-tenant users only).
		primaryTenant := m.getUserPrimaryTenant(r.Context(), usr.ID())

		// Create auth context (tenant list populated elsewhere during Authenticate)
		authCtx := &AuthContext{
			UserID:        usr.ID(),
			TenantID:      primaryTenant,
			Email:         usr.Email(),
			IsSystemAdmin: isSystemAdmin,
		}

		// Add auth context to request context
		ctx := context.WithValue(r.Context(), "auth", authCtx)
		r = r.WithContext(ctx)

		if m.isSetupGateBlocked(r, authCtx) {
			http.Error(w, "Initial system setup is required before accessing this endpoint", http.StatusConflict)
			return
		}

		// Continue with next handler
		next.ServeHTTP(w, r)
	})
}

func isWebSocketUpgradeRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	upgrade := strings.ToLower(strings.TrimSpace(r.Header.Get("Upgrade")))
	connection := strings.ToLower(r.Header.Get("Connection"))
	return upgrade == "websocket" && strings.Contains(connection, "upgrade")
}

func isExpectedTokenValidationFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case msg == "invalid access token":
		return true
	case msg == "token has expired, please login again":
		return true
	case msg == "invalid token claims":
		return true
	case msg == "invalid user id in token":
		return true
	case msg == "invalid user id format":
		return true
	case msg == "user not found":
		return true
	case msg == "sql: no rows in result set":
		return true
	case strings.Contains(msg, "unexpected signing method"):
		return true
	case strings.Contains(msg, "token has expired"):
		return true
	case strings.Contains(msg, "account disabled"):
		return true
	default:
		return false
	}
}

// AuthenticateWithoutTenant validates JWT token without requiring valid tenant context
// Useful for user profile and other personal endpoints
func (m *AuthMiddleware) AuthenticateWithoutTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.logger.Debug("No authorization header provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.logger.Debug("Invalid authorization header format")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		usr, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			if isExpectedTokenValidationFailure(err) {
				m.logger.Debug("Token rejected", zap.Error(err))
			} else {
				m.logger.Warn("Token validation failed", zap.Error(err))
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Tenant header is optional for this middleware. If absent, fall back to
		// a RBAC-derived tenant only for single-tenant users.
		selectedTenantID := m.getUserPrimaryTenant(r.Context(), usr.ID())
		tenantIDStr := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if tenantIDStr != "" {
			parsedTenantID, parseErr := uuid.Parse(tenantIDStr)
			if parseErr != nil {
				m.logger.Warn("Invalid X-Tenant-ID header format",
					zap.String("tenant_id", tenantIDStr),
					zap.String("user_id", usr.ID().String()),
				)
				http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
				return
			}
			if parsedTenantID == uuid.Nil {
				m.logger.Warn("Nil tenant UUID is not allowed",
					zap.String("user_id", usr.ID().String()),
					zap.String("email", usr.Email()),
				)
				http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
				return
			}
			selectedTenantID = parsedTenantID
		}

		// Check if user has multi-tenant access (has roles in multiple tenants)
		var hasMultiTenant bool
		if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
			// Validate tenant access for non-system-admin users when tenant context is present.
			isAdmin, err := rbacRepo.IsUserSystemAdmin(r.Context(), usr.ID())
			if err == nil && !isAdmin && selectedTenantID != uuid.Nil {
				hasAccess, accessErr := rbacRepo.UserHasTenantAccess(r.Context(), usr.ID(), selectedTenantID)
				if accessErr != nil {
					m.logger.Error("Failed to validate tenant access",
						zap.String("user_id", usr.ID().String()),
						zap.String("tenant_id", selectedTenantID.String()),
						zap.Error(accessErr),
					)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				if !hasAccess {
					m.logger.Warn("User attempted to access unauthorized tenant",
						zap.String("user_id", usr.ID().String()),
						zap.String("tenant_id", selectedTenantID.String()),
					)
					http.Error(w, "Forbidden: No access to tenant", http.StatusForbidden)
					return
				}
			}

			userRoles, err := rbacRepo.FindUserRoles(r.Context(), usr.ID())
			if err == nil {
				// Count unique tenants from roles
				tenantSet := make(map[uuid.UUID]bool)
				for _, role := range userRoles {
					if role.TenantID() != uuid.Nil {
						tenantSet[role.TenantID()] = true
					}
				}
				hasMultiTenant = len(tenantSet) > 1
			} else {
				// Log the error but don't fail authentication
				m.logger.Debug("Failed to check multi-tenant access, assuming single tenant", zap.Error(err), zap.String("user_id", usr.ID().String()))
			}
		}

		// Check if user is a system admin (member of system_administrator group)
		isSystemAdmin := false
		if rbacRepo, ok := m.rbacRepo.(*postgres.RBACRepository); ok {
			isAdmin, err := rbacRepo.IsUserSystemAdmin(r.Context(), usr.ID())
			if err == nil {
				isSystemAdmin = isAdmin
			}
		}

		// Build authoritative user tenant list from RBAC (user_role_assignments)
		userTenants := m.getUserTenants(r.Context(), usr.ID())

		// Create auth context (use RBAC-derived tenant list instead of legacy user.tenant_id)
		authCtx := &AuthContext{
			UserID:         usr.ID(),
			TenantID:       selectedTenantID,
			UserTenants:    userTenants,
			Email:          usr.Email(),
			HasMultiTenant: hasMultiTenant,
			IsSystemAdmin:  isSystemAdmin,
		}

		// Add auth context to request context
		ctx := context.WithValue(r.Context(), "auth", authCtx)
		r = r.WithContext(ctx)

		// Log successful authentication
		m.logger.Debug("User authenticated without tenant validation",
			zap.String("user_id", usr.ID().String()),
			zap.String("email", usr.Email()))

		// Continue with next handler
		next.ServeHTTP(w, r)
	})
}

// GetAuthContext extracts authentication context from request
func GetAuthContext(r *http.Request) (*AuthContext, bool) {
	authCtx, ok := r.Context().Value("auth").(*AuthContext)
	return authCtx, ok
}

// RequireAuth is a convenience function that applies authentication middleware
func RequireAuth(authService userTokenValidator, rbacRepo interface{}, auditService *audit.Service, logger *zap.Logger) func(http.Handler) http.Handler {
	middleware := NewAuthMiddleware(authService, rbacRepo, auditService, logger, nil)
	return middleware.Authenticate
}

// RequirePermission is a convenience function that applies permission middleware
func RequirePermission(authService userTokenValidator, rbacRepo interface{}, auditService *audit.Service, rbacService interface{}, resource, action string, logger *zap.Logger) func(http.Handler) http.Handler {
	middleware := NewAuthMiddleware(authService, rbacRepo, auditService, logger, nil)
	return middleware.RequirePermission(rbacService, resource, action)
}

func (m *AuthMiddleware) isSetupGateBlocked(r *http.Request, authCtx *AuthContext) bool {
	if m.bootstrapService == nil || authCtx == nil || !authCtx.IsSystemAdmin {
		return false
	}

	setupRequired, err := m.bootstrapService.IsSetupRequired(r.Context())
	if err != nil {
		m.logger.Warn("Failed to evaluate setup-required status", zap.Error(err))
		return false
	}
	if !setupRequired {
		return false
	}

	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/bootstrap/") {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/auth/") {
		switch path {
		case "/api/v1/auth/me", "/api/v1/auth/logout", "/api/v1/auth/change-password", "/api/v1/auth/refresh":
			return false
		}
	}

	allowedPrefixes := []string{
		"/api/v1/profile",
		"/api/v1/system-configs",
		"/api/v1/admin/ldap",
		"/api/v1/admin/external-services",
		"/api/v1/sso/",
		"/api/v1/admin/system/reboot",
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}

	m.logger.Info("Blocking request until initial setup completes",
		zap.String("user_id", authCtx.UserID.String()),
		zap.String("path", path),
	)
	return true
}

func (m *AuthMiddleware) isPasswordChangeGateBlocked(r *http.Request, usr *user.User) bool {
	if usr == nil || !usr.MustChangePassword() {
		return false
	}

	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/api/v1/auth/change-password"),
		strings.HasPrefix(path, "/api/v1/auth/logout"),
		strings.HasPrefix(path, "/api/v1/auth/me"),
		strings.HasPrefix(path, "/api/v1/profile"),
		strings.HasPrefix(path, "/api/v1/bootstrap/status"):
		return false
	default:
		m.logger.Info("Blocking request until user changes password",
			zap.String("user_id", usr.ID().String()),
			zap.String("path", path),
		)
		return true
	}
}
