package rest

import (
	"strings"
	"testing"
)

// TestRouter_AdminRoutePermissionContracts ensures critical admin/system routes
// remain protected by route-level permission middleware.
func TestRouter_AdminRoutePermissionContracts(t *testing.T) {
	content := readRouterRegistrationSources(t)

	requiredSnippets := []string{
		`router.Get("/api/v1/roles", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(roleHandler.GetRoles)).ServeHTTP)`,
		`router.Get("/api/v1/roles/{id}", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(roleHandler.GetRoleByID)).ServeHTTP)`,
		`router.Get("/api/v1/roles/{id}/permissions", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(permissionHandler.GetRolePermissions)).ServeHTTP)`,
		`router.Get("/api/v1/permissions", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.GetPermissions)).ServeHTTP)`,
		`router.Get("/api/v1/permissions/{id}", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.GetPermissionByID)).ServeHTTP)`,
		`router.Get("/api/v1/admin/settings/tools", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetToolAvailability)).ServeHTTP)`,
		`router.Get("/api/v1/admin/users/check-email", authMiddleware.RequirePermission(permissionService, "user", "read")(http.HandlerFunc(userHandler.CheckUserEmail)).ServeHTTP)`,
		`router.Get("/api/v1/images/released-artifacts", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageImportHandler.ListReleasedArtifacts))).ServeHTTP)`,
		`router.Post("/api/v1/images/released-artifacts/{id}/consume", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageImportHandler.ConsumeReleasedArtifact))).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.ListImportRequests))).ServeHTTP)`,
		`router.Get("/api/v1/admin/images/import-requests", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.ListAllImportRequests)).ServeHTTP)`,
		`router.Get("/api/v1/admin/images/import-requests/{id}/workflow", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.GetImportRequestWorkflowAdmin)).ServeHTTP)`,
		`router.Get("/api/v1/admin/images/import-requests/{id}/logs", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.GetImportRequestLogsAdmin)).ServeHTTP)`,
		`router.Get("/api/v1/admin/images/import-requests/{id}/logs/stream", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.StreamImportRequestLogsAdmin)).ServeHTTP)`,
		`router.Post("/api/v1/admin/images/import-requests/{id}/approve", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.ApproveImportRequestAdmin)).ServeHTTP)`,
		`router.Post("/api/v1/admin/images/import-requests/{id}/reject", authMiddleware.RequirePermission(permissionService, "quarantine", "reject")(http.HandlerFunc(imageImportHandler.RejectImportRequestAdmin)).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.GetImportRequest))).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests/{id}/workflow", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.GetImportRequestWorkflow))).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests/{id}/logs", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.GetImportRequestLogs))).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests/{id}/logs/stream", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.StreamImportRequestLogs))).ServeHTTP)`,
		`router.Post("/api/v1/images/import-requests/{id}/approve", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.ApproveImportRequest))).ServeHTTP)`,
		`router.Post("/api/v1/images/import-requests/{id}/reject", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "reject")(http.HandlerFunc(imageImportHandler.RejectImportRequest))).ServeHTTP)`,
		`router.Post("/api/v1/images/import-requests/{id}/release", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "release")(http.HandlerFunc(imageImportHandler.ReleaseImportRequest))).ServeHTTP)`,
		`router.Post("/api/v1/images/import-requests/{id}/withdraw", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(imageImportHandler.WithdrawImportRequest))).ServeHTTP)`,
		`router.Post("/api/v1/epr/registration-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(eprRegistrationHandler.CreateRequest))).ServeHTTP)`,
		`router.Get("/api/v1/epr/registration-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(eprRegistrationHandler.ListTenantRequests))).ServeHTTP)`,
		`router.Post("/api/v1/epr/registration-requests/{id}/withdraw", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(eprRegistrationHandler.WithdrawTenantRequest))).ServeHTTP)`,
		`router.Get("/api/v1/admin/epr/registration-requests", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.ListAllRequests)).ServeHTTP)`,
		`router.Post("/api/v1/admin/epr/registration-requests/{id}/approve", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.ApproveRequest)).ServeHTTP)`,
		`router.Post("/api/v1/admin/epr/registration-requests/{id}/reject", authMiddleware.RequirePermission(permissionService, "quarantine", "reject")(http.HandlerFunc(eprRegistrationHandler.RejectRequest)).ServeHTTP)`,
		`router.Get("/api/v1/admin/settings/release-governance-policy", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetReleaseGovernancePolicy)).ServeHTTP)`,
		`router.Put("/api/v1/admin/settings/release-governance-policy", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateReleaseGovernancePolicy)).ServeHTTP)`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected secured route snippet not found:\n%s", snippet)
		}
	}

	forbiddenSnippets := []string{
		`router.Get("/api/v1/admin/settings/tools", authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetToolAvailability)).ServeHTTP)`,
		`router.Get("/api/v1/admin/users/check-email", authMiddleware.Authenticate(http.HandlerFunc(userHandler.CheckUserEmail)).ServeHTTP)`,
		`router.Get("/api/v1/admin/settings/release-governance-policy", authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetReleaseGovernancePolicy)).ServeHTTP)`,
		`router.Get("/api/v1/images/released-artifacts", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.ListReleasedArtifacts))).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageImportHandler.ListImportRequests))).ServeHTTP)`,
		`router.Get("/api/v1/images/import-requests/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageImportHandler.GetImportRequest))).ServeHTTP)`,
	}

	for _, snippet := range forbiddenSnippets {
		if strings.Contains(content, snippet) {
			t.Fatalf("found legacy unsecured route snippet:\n%s", snippet)
		}
	}
}
