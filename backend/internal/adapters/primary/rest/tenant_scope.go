package rest

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// isAllTenantsScopeRequested returns true only when a system admin explicitly
// requests all-tenant scope via query or header.
func isAllTenantsScopeRequested(r *http.Request, authCtx *middleware.AuthContext) bool {
	if authCtx == nil || !authCtx.IsSystemAdmin {
		return false
	}

	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Tenant-Scope")), "all") {
		return true
	}

	switch strings.TrimSpace(strings.ToLower(r.URL.Query().Get("all_tenants"))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// resolveTenantScopeFromRequest resolves tenant scope for handlers that support:
// - default tenant scope (current auth tenant),
// - optional explicit all-tenant scope for system admins,
// - optional explicit tenant_id overrides.
func resolveTenantScopeFromRequest(r *http.Request, authCtx *middleware.AuthContext, allowAllTenants bool) (tenantID uuid.UUID, allTenants bool, status int, message string) {
	if authCtx == nil {
		return uuid.Nil, false, http.StatusUnauthorized, "Unauthorized"
	}

	tenantID = authCtx.TenantID
	if allowAllTenants {
		allTenants = isAllTenantsScopeRequested(r, authCtx)
	}

	if tenantIDRaw := strings.TrimSpace(r.URL.Query().Get("tenant_id")); tenantIDRaw != "" {
		parsedTenantID, err := uuid.Parse(tenantIDRaw)
		if err != nil || parsedTenantID == uuid.Nil {
			return uuid.Nil, false, http.StatusBadRequest, "Invalid tenant_id"
		}
		if !authCtx.IsSystemAdmin && parsedTenantID != authCtx.TenantID {
			return uuid.Nil, false, http.StatusForbidden, "Access denied to this tenant"
		}
		return parsedTenantID, false, 0, ""
	}

	return tenantID, allTenants, 0, ""
}
