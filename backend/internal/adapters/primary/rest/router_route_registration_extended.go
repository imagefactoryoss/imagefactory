package rest

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/build"
	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
	"github.com/srikarm/image-factory/internal/domain/mfa"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

func registerIdentitySystemAdminRoutes(
	router *Router,
	sqlxDB *sqlx.DB,
	db interface{},
	buildPolicyService *build.BuildPolicyService,
	auditService *audit.Service,
	logger *zap.Logger,
	userService *user.Service,
	rbacRepo rbac.Repository,
	permissionService *rbac.PermissionService,
	authMiddleware *middleware.AuthMiddleware,
	tenantStatusMiddleware *middleware.TenantStatusMiddleware,
	userHandler *UserHandler,
	userInvitationHandler *UserInvitationHandler,
	roleHandler *RoleHandler,
	permissionHandler *PermissionHandler,
	systemConfigHandler *SystemConfigHandler,
	systemStatsHandler *SystemStatsHandler,
	systemComponentsStatusHandler *SystemComponentsStatusHandler,
	dispatcherMetricsHandler *DispatcherMetricsHandler,
	dispatcherControlHandler *DispatcherControlHandler,
	orchestratorControlHandler *OrchestratorControlHandler,
	executionPipelineHealthHandler *ExecutionPipelineHealthHandler,
	executionPipelineMetricsHandler *ExecutionPipelineMetricsHandler,
	workflowHandler *WorkflowHandler,
	auditHandler *AuditHandler,
	onboardingHandler *OnboardingHandler,
) {
	// Register profile routes
	profileHandler := NewProfileHandler(sqlxDB, auditService, logger)
	router.Get("/api/v1/profile", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(profileHandler.GetProfile)).ServeHTTP)
	router.Put("/api/v1/profile", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(profileHandler.UpdateProfile)).ServeHTTP)

	// Register user management routes (basic CRUD now handled by routes.UserRoutes)
	// Additional user routes not covered by UserRoutes:
	// User roles management
	router.Post("/api/v1/users/roles", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "manage_roles")(http.HandlerFunc(userHandler.AssignRoleToUser))).ServeHTTP)
	router.Post("/api/v1/users/roles/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "manage_roles")(http.HandlerFunc(userHandler.AssignRoleToUser))).ServeHTTP)
	router.Delete("/api/v1/users/roles/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "manage_roles")(http.HandlerFunc(userHandler.RemoveRoleFromUser))).ServeHTTP)

	// Register user invitation routes
	router.Post("/api/v1/invitations", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "invitation", "create")(http.HandlerFunc(userInvitationHandler.CreateInvitation))).ServeHTTP)
	router.Get("/api/v1/invitations", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "invitation", "list")(http.HandlerFunc(userInvitationHandler.ListInvitations))).ServeHTTP)

	// Accept invitation (public): POST /api/v1/invitations/accept
	// Register this BEFORE the parameterized routes to avoid matching as {invitationId}
	router.Post("/api/v1/invitations/accept", userInvitationHandler.AcceptInvitation)

	// Resend invitation: POST /api/v1/invitations/{invitationId}/resend
	router.Post("/api/v1/invitations/{invitationId}/resend", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "invitation", "create")(http.HandlerFunc(userInvitationHandler.ResendInvitation))).ServeHTTP)

	// Other invitation routes (by invitationId)
	router.Get("/api/v1/invitations/{invitationId}", userInvitationHandler.GetInvitation)
	router.Delete("/api/v1/invitations/{invitationId}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "invitation", "delete")(http.HandlerFunc(userInvitationHandler.CancelInvitation))).ServeHTTP)

	// Register role management routes
	router.Post("/api/v1/roles", authMiddleware.RequirePermission(permissionService, "role", "create")(http.HandlerFunc(roleHandler.CreateRole)).ServeHTTP)
	router.Get("/api/v1/roles", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(roleHandler.GetRoles)).ServeHTTP)

	// Consolidated handler for /api/v1/roles/{id} covering both role and permission endpoints
	router.Get("/api/v1/roles/{id}", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(roleHandler.GetRoleByID)).ServeHTTP)
	router.Put("/api/v1/roles/{id}", authMiddleware.RequirePermission(permissionService, "role", "update")(http.HandlerFunc(roleHandler.UpdateRole)).ServeHTTP)
	router.Delete("/api/v1/roles/{id}", authMiddleware.RequirePermission(permissionService, "role", "delete")(http.HandlerFunc(roleHandler.DeleteRole)).ServeHTTP)

	// Specific handler for /api/v1/roles/{id}/permissions
	router.Get("/api/v1/roles/{id}/permissions", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(permissionHandler.GetRolePermissions)).ServeHTTP)
	router.Post("/api/v1/roles/{id}/permissions", authMiddleware.RequirePermission(permissionService, "role", "manage_permissions")(http.HandlerFunc(permissionHandler.AddPermissionToRole)).ServeHTTP)
	router.Delete("/api/v1/roles/{id}/permissions", authMiddleware.RequirePermission(permissionService, "role", "manage_permissions")(http.HandlerFunc(permissionHandler.RemovePermissionFromRole)).ServeHTTP)

	// Handler for /api/v1/roles/{id}/permissions/{permissionId} - Add/remove permission to/from role
	router.Post("/api/v1/roles/{id}/permissions/{permissionId}", authMiddleware.RequirePermission(permissionService, "role", "manage_permissions")(http.HandlerFunc(permissionHandler.AddPermissionToRole)).ServeHTTP)
	router.Delete("/api/v1/roles/{id}/permissions/{permissionId}", authMiddleware.RequirePermission(permissionService, "role", "manage_permissions")(http.HandlerFunc(permissionHandler.RemovePermissionFromRole)).ServeHTTP)

	// Register permission management routes
	router.Get("/api/v1/permissions", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.GetPermissions)).ServeHTTP)
	router.Post("/api/v1/permissions", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.CreatePermission)).ServeHTTP)

	// Handler for /api/v1/permissions/{id}/roles - bulk assign permission to multiple roles
	router.Post("/api/v1/permissions/{id}/roles", authMiddleware.RequirePermission(permissionService, "role", "manage_permissions")(http.HandlerFunc(permissionHandler.AssignPermissionToMultipleRoles)).ServeHTTP)

	// Consolidated handler for /api/v1/permissions/ covering single permission operations
	router.Get("/api/v1/permissions/{id}", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.GetPermissionByID)).ServeHTTP)
	router.Put("/api/v1/permissions/{id}", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.UpdatePermission)).ServeHTTP)
	router.Delete("/api/v1/permissions/{id}", authMiddleware.RequirePermission(permissionService, "permissions", "manage")(http.HandlerFunc(permissionHandler.DeletePermission)).ServeHTTP)

	// Register system configuration routes
	router.Post("/api/v1/system-configs", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.CreateConfig)).ServeHTTP)
	router.Get("/api/v1/system-configs", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.ListConfigs)).ServeHTTP)

	// Register system configuration category routes
	router.Post("/api/v1/system-configs/category", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateCategoryConfig)).ServeHTTP)

	// System actions (non-production only)
	router.Post("/api/v1/admin/system/reboot", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.RebootServer)).ServeHTTP)

	router.Get("/api/v1/system-configs/{id}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetConfig)).ServeHTTP)
	router.Put("/api/v1/system-configs/{id}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateConfig)).ServeHTTP)
	router.Delete("/api/v1/system-configs/{id}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.DeleteConfig)).ServeHTTP)

	router.Post("/api/v1/system-configs/activate/{id}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.ActivateConfig)).ServeHTTP)
	router.Post("/api/v1/system-configs/deactivate/{id}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.DeactivateConfig)).ServeHTTP)

	router.Post("/api/v1/system-configs/test-connection", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.TestConnection)).ServeHTTP)

	// Register tool availability routes
	router.Get("/api/v1/settings/tools", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(systemConfigHandler.GetToolAvailability))).ServeHTTP)
	router.Get("/api/v1/admin/settings/tools", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetToolAvailability)).ServeHTTP)
	router.Put("/api/v1/admin/settings/tools", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateToolAvailability)).ServeHTTP)
	router.Get("/api/v1/admin/settings/tekton-task-images", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetTektonTaskImages)).ServeHTTP)
	router.Put("/api/v1/admin/settings/tekton-task-images", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateTektonTaskImages)).ServeHTTP)
	router.Get("/api/v1/settings/build-capabilities", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(systemConfigHandler.GetBuildCapabilities))).ServeHTTP)
	router.Get("/api/v1/admin/settings/build-capabilities", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetBuildCapabilities)).ServeHTTP)
	router.Put("/api/v1/admin/settings/build-capabilities", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateBuildCapabilities)).ServeHTTP)
	router.Get("/api/v1/settings/operation-capabilities", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetOperationCapabilities))).ServeHTTP)
	router.Get("/api/v1/admin/settings/operation-capabilities", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetOperationCapabilities)).ServeHTTP)
	router.Put("/api/v1/admin/settings/operation-capabilities", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateOperationCapabilities)).ServeHTTP)
	router.Get("/api/v1/settings/capability-surfaces", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetCapabilitySurfaces))).ServeHTTP)
	router.Get("/api/v1/admin/settings/capability-surfaces", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetCapabilitySurfaces)).ServeHTTP)
	router.Get("/api/v1/settings/quarantine-policy", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetQuarantinePolicy))).ServeHTTP)
	router.Get("/api/v1/admin/settings/quarantine-policy", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetQuarantinePolicy)).ServeHTTP)
	router.Put("/api/v1/admin/settings/quarantine-policy", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateQuarantinePolicy)).ServeHTTP)
	router.Post("/api/v1/admin/settings/quarantine-policy/validate", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.ValidateQuarantinePolicy)).ServeHTTP)
	router.Post("/api/v1/admin/settings/quarantine-policy/simulate", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.SimulateQuarantinePolicy)).ServeHTTP)
	router.Get("/api/v1/settings/epr-registration", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetSORRegistration))).ServeHTTP)
	router.Get("/api/v1/admin/settings/epr-registration", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetSORRegistration)).ServeHTTP)
	router.Put("/api/v1/admin/settings/epr-registration", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateSORRegistration)).ServeHTTP)
	router.Get("/api/v1/settings/release-governance-policy", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.Authenticate(http.HandlerFunc(systemConfigHandler.GetReleaseGovernancePolicy))).ServeHTTP)
	router.Get("/api/v1/admin/settings/release-governance-policy", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetReleaseGovernancePolicy)).ServeHTTP)
	router.Put("/api/v1/admin/settings/release-governance-policy", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateReleaseGovernancePolicy)).ServeHTTP)

	// Register external services routes
	router.Get("/api/v1/admin/external-services/{name}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetExternalService)).ServeHTTP)
	router.Put("/api/v1/admin/external-services/{name}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateExternalService)).ServeHTTP)
	router.Delete("/api/v1/admin/external-services/{name}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.DeleteExternalService)).ServeHTTP)
	router.Get("/api/v1/admin/external-services", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetExternalServices)).ServeHTTP)
	router.Post("/api/v1/admin/external-services", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.CreateExternalService)).ServeHTTP)

	// Register LDAP configuration routes
	router.Get("/api/v1/admin/ldap", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetLDAPConfigs)).ServeHTTP)
	router.Post("/api/v1/admin/ldap", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.CreateLDAPConfig)).ServeHTTP)
	router.Get("/api/v1/admin/ldap/{id}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemConfigHandler.GetLDAPConfig)).ServeHTTP)
	router.Put("/api/v1/admin/ldap/{id}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.UpdateLDAPConfig)).ServeHTTP)
	router.Delete("/api/v1/admin/ldap/{id}", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(systemConfigHandler.DeleteLDAPConfig)).ServeHTTP)

	// Register system stats routes
	router.Get("/api/v1/admin/stats", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemStatsHandler.GetSystemStats)).ServeHTTP)
	router.Get("/api/v1/admin/system/components-status", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(systemComponentsStatusHandler.GetStatus)).ServeHTTP)
	router.Get("/api/v1/admin/dispatcher/metrics", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(dispatcherMetricsHandler.GetMetrics)).ServeHTTP)
	router.Get("/api/v1/admin/dispatcher/status", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(dispatcherControlHandler.GetDispatcherStatus)).ServeHTTP)
	router.Get("/api/v1/admin/execution-pipeline/health", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(executionPipelineHealthHandler.GetHealth)).ServeHTTP)
	router.Get("/api/v1/admin/execution-pipeline/metrics", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(executionPipelineMetricsHandler.GetMetrics)).ServeHTTP)
	router.Post("/api/v1/admin/dispatcher/start", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(dispatcherControlHandler.StartDispatcher)).ServeHTTP)
	router.Post("/api/v1/admin/orchestrator/start", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(orchestratorControlHandler.StartOrchestrator)).ServeHTTP)
	router.Post("/api/v1/admin/orchestrator/stop", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(orchestratorControlHandler.StopOrchestrator)).ServeHTTP)
	router.Post("/api/v1/admin/dispatcher/stop", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(dispatcherControlHandler.StopDispatcher)).ServeHTTP)
	router.Post("/api/v1/workflows", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(workflowHandler.CreateWorkflow)).ServeHTTP)
	router.Get("/api/v1/workflows/{id}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(workflowHandler.GetWorkflow)).ServeHTTP)

	// Register build analytics routes
	var sqlDB *sql.DB
	if typedSQLX, ok := db.(*sqlx.DB); ok {
		sqlDB = typedSQLX.DB
	} else if plainDB, ok := db.(*sql.DB); ok {
		sqlDB = plainDB
	}
	if sqlDB != nil {
		buildAnalyticsHandler := NewBuildAnalyticsHandler(sqlDB)

		router.Get("/api/v1/admin/builds/analytics", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(buildAnalyticsHandler.GetAnalytics)).ServeHTTP)
		router.Get("/api/v1/admin/builds/performance", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(buildAnalyticsHandler.GetPerformance)).ServeHTTP)
		router.Get("/api/v1/admin/builds/failures", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(buildAnalyticsHandler.GetFailures)).ServeHTTP)

		// Register build policy routes
		buildPolicyHandler := NewBuildPolicyHandler(buildPolicyService, auditService, logger)
		router.Get("/api/v1/admin/builds/policies", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(buildPolicyHandler.GetPolicies)).ServeHTTP)
		router.Post("/api/v1/admin/builds/policies", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(buildPolicyHandler.CreatePolicy)).ServeHTTP)
		router.Get("/api/v1/admin/builds/policies/{id}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(buildPolicyHandler.GetPolicy)).ServeHTTP)
		router.Put("/api/v1/admin/builds/policies/{id}", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(buildPolicyHandler.UpdatePolicy)).ServeHTTP)
		router.Delete("/api/v1/admin/builds/policies/{id}", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(buildPolicyHandler.DeletePolicy)).ServeHTTP)

		// Register infrastructure management routes
		infraAdminHandler := NewInfrastructureHandler(sqlDB, logger)
		router.Get("/api/v1/admin/infrastructure/nodes", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(infraAdminHandler.GetNodes)).ServeHTTP)
		router.Post("/api/v1/admin/infrastructure/nodes", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(infraAdminHandler.CreateNode)).ServeHTTP)
		router.Get("/api/v1/admin/infrastructure/nodes/{id}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(infraAdminHandler.GetNode)).ServeHTTP)
		router.Put("/api/v1/admin/infrastructure/nodes/{id}", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(infraAdminHandler.UpdateNode)).ServeHTTP)
		router.Delete("/api/v1/admin/infrastructure/nodes/{id}", authMiddleware.RequirePermission(permissionService, "system", "write")(http.HandlerFunc(infraAdminHandler.DeleteNode)).ServeHTTP)
		router.Get("/api/v1/admin/infrastructure/health", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(infraAdminHandler.GetInfrastructureHealth)).ServeHTTP)
	}

	// Check user email endpoint
	router.Get("/api/v1/admin/users/check-email", authMiddleware.RequirePermission(permissionService, "user", "read")(http.HandlerFunc(userHandler.CheckUserEmail)).ServeHTTP)

	// Register audit routes
	router.Get("/api/v1/audit-events", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(auditHandler.ListAuditEvents)).ServeHTTP)
	router.Get("/api/v1/audit-events/{id}", authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(auditHandler.GetAuditEvent)).ServeHTTP)

	// Register onboarding routes (start onboarding doesn't require auth, status/resume require auth)
	router.Post("/api/v1/onboarding/start", onboardingHandler.StartOnboarding)
	router.Get("/api/v1/onboarding/status/{id}", middleware.RequireAuth(userService, rbacRepo, auditService, logger)(http.HandlerFunc(onboardingHandler.GetOnboardingStatus)).ServeHTTP)
	router.Post("/api/v1/onboarding/resume/{id}", middleware.RequireAuth(userService, rbacRepo, auditService, logger)(http.HandlerFunc(onboardingHandler.ResumeOnboarding)).ServeHTTP)
}

func registerSecurityCatalogRepoRoutes(
	router *Router,
	logger *zap.Logger,
	systemConfigRepo systemconfig.Repository,
	emailDomainService *domainEmail.Service,
	auditService *audit.Service,
	permissionService *rbac.PermissionService,
	authMiddleware *middleware.AuthMiddleware,
	tenantStatusMiddleware *middleware.TenantStatusMiddleware,
	ssoHandler *SSOHandler,
	groupHandler *GroupHandler,
	imageHandler *ImageHandler,
	imageImportHandler *ImageImportHandler,
	onDemandScanRequestHandler *ImageImportHandler,
	eprRegistrationHandler *EPRRegistrationHandler,
	repoAuthService *repositoryauth.Service,
	projectService *project.Service,
	gitProviderHandler *GitProviderHandler,
	repoBranchHandler *RepositoryBranchHandler,
) {
	// Register SSO routes
	router.Post("/api/v1/sso/saml/providers", authMiddleware.RequirePermission(permissionService, "sso", "write")(http.HandlerFunc(ssoHandler.CreateSAMLProvider)).ServeHTTP)
	router.Get("/api/v1/sso/saml/providers", authMiddleware.RequirePermission(permissionService, "sso", "read")(http.HandlerFunc(ssoHandler.GetSAMLProviders)).ServeHTTP)
	router.Patch("/api/v1/sso/saml/providers/{provider_id}/status", authMiddleware.RequirePermission(permissionService, "sso", "write")(http.HandlerFunc(ssoHandler.ToggleSAMLProviderStatus)).ServeHTTP)
	router.Post("/api/v1/sso/oidc/providers", authMiddleware.RequirePermission(permissionService, "sso", "write")(http.HandlerFunc(ssoHandler.CreateOIDCProvider)).ServeHTTP)
	router.Get("/api/v1/sso/oidc/providers", authMiddleware.RequirePermission(permissionService, "sso", "read")(http.HandlerFunc(ssoHandler.GetOIDCProviders)).ServeHTTP)
	router.Patch("/api/v1/sso/oidc/providers/{provider_id}/status", authMiddleware.RequirePermission(permissionService, "sso", "write")(http.HandlerFunc(ssoHandler.ToggleOIDCProviderStatus)).ServeHTTP)
	router.Get("/api/v1/sso/configuration", ssoHandler.GetSSOConfiguration)
	router.Get("/api/v1/sso/saml/metadata/{id}", ssoHandler.ValidateSAMLMetadata)

	// Initialize MFA repository and service
	mfaRepo := postgres.NewMFARepository(logger)
	mfaService := mfa.NewServiceWithEmail(mfaRepo, emailDomainService, os.Getenv("IF_SMTP_FROM_EMAIL"), logger)
	mfaHandler := NewMFAHandler(mfaService, auditService, logger)

	// Register MFA routes
	router.Post("/api/v1/mfa/totp/setup/secret", authMiddleware.RequirePermission(permissionService, "mfa", "write")(http.HandlerFunc(mfaHandler.SetupTOTPSecret)).ServeHTTP)
	router.Post("/api/v1/mfa/totp/setup/confirm", authMiddleware.RequirePermission(permissionService, "mfa", "write")(http.HandlerFunc(mfaHandler.ConfirmTOTPSetup)).ServeHTTP)
	router.Post("/api/v1/mfa/challenge", authMiddleware.RequirePermission(permissionService, "mfa", "write")(http.HandlerFunc(mfaHandler.StartMFAChallenge)).ServeHTTP)
	router.Post("/api/v1/mfa/verify", authMiddleware.RequirePermission(permissionService, "mfa", "write")(http.HandlerFunc(mfaHandler.VerifyMFAChallenge)).ServeHTTP)
	router.Get("/api/v1/mfa/status", authMiddleware.RequirePermission(permissionService, "mfa", "read")(http.HandlerFunc(mfaHandler.GetMFAStatus)).ServeHTTP)

	// List group members: GET /api/v1/groups/{groupId}/members
	router.Get("/api/v1/groups/{groupId}/members", authMiddleware.RequirePermission(permissionService, "groups", "read")(http.HandlerFunc(groupHandler.ListGroupMembers)).ServeHTTP)
	router.Post("/api/v1/groups/{groupId}/members", authMiddleware.RequirePermission(permissionService, "groups", "write")(http.HandlerFunc(groupHandler.AddGroupMember)).ServeHTTP)
	router.Delete("/api/v1/groups/{groupId}/members", authMiddleware.RequirePermission(permissionService, "groups", "write")(http.HandlerFunc(groupHandler.RemoveGroupMember)).ServeHTTP)

	// Initialize external tenant handler
	externalTenantHTTPClient := &http.Client{Timeout: 10 * time.Second}
	externalTenantHandler := NewExternalTenantHandler(
		logger,
		systemconfig.NewService(systemConfigRepo, logger),
		externalTenantHTTPClient,
	)

	// Register external tenant routes (for tenant onboarding lookup)
	// Protected with JWT authentication (not API key)
	router.Get("/api/v1/external-tenants", authMiddleware.RequirePermission(permissionService, "tenant", "create")(http.HandlerFunc(externalTenantHandler.ListExternalTenants)).ServeHTTP)
	router.Get("/api/v1/external-tenants/{id}", authMiddleware.RequirePermission(permissionService, "tenant", "create")(http.HandlerFunc(externalTenantHandler.GetExternalTenant)).ServeHTTP)

	// Register image routes
	router.Post("/api/v1/images", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(imageHandler.CreateImage))).ServeHTTP)
	router.Get("/api/v1/images/search", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.SearchImages))).ServeHTTP)
	router.Get("/api/v1/images/popular", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetPopularImages))).ServeHTTP)
	router.Get("/api/v1/images/recent", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetRecentImages))).ServeHTTP)
	router.Get("/api/v1/images/{id}/versions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetImageVersions))).ServeHTTP)
	router.Get("/api/v1/images/{id}/details", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetImageDetails))).ServeHTTP)
	router.Get("/api/v1/images/{id}/tags", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetImageTags))).ServeHTTP)
	router.Post("/api/v1/images/{id}/tags", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "update")(http.HandlerFunc(imageHandler.AddImageTags))).ServeHTTP)
	router.Delete("/api/v1/images/{id}/tags", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "update")(http.HandlerFunc(imageHandler.RemoveImageTags))).ServeHTTP)
	router.Get("/api/v1/images/{id}/stats", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetImageStats))).ServeHTTP)
	router.Post("/api/v1/images/{id}/scan", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "update")(http.HandlerFunc(imageHandler.TriggerOnDemandScan))).ServeHTTP)
	router.Get("/api/v1/images/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageHandler.GetImage))).ServeHTTP)
	router.Put("/api/v1/images/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "update")(http.HandlerFunc(imageHandler.UpdateImage))).ServeHTTP)
	router.Delete("/api/v1/images/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "delete")(http.HandlerFunc(imageHandler.DeleteImage))).ServeHTTP)
	router.Post("/api/v1/images/import-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(imageImportHandler.CreateImportRequest))).ServeHTTP)
	router.Get("/api/v1/images/released-artifacts", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageImportHandler.ListReleasedArtifacts))).ServeHTTP)
	router.Post("/api/v1/images/released-artifacts/{id}/consume", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(imageImportHandler.ConsumeReleasedArtifact))).ServeHTTP)
	router.Get("/api/v1/images/import-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.ListImportRequests))).ServeHTTP)
	router.Get("/api/v1/admin/images/import-requests", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.ListAllImportRequests)).ServeHTTP)
	router.Get("/api/v1/admin/images/import-requests/{id}/workflow", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.GetImportRequestWorkflowAdmin)).ServeHTTP)
	router.Get("/api/v1/admin/images/import-requests/{id}/logs", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.GetImportRequestLogsAdmin)).ServeHTTP)
	router.Get("/api/v1/admin/images/import-requests/{id}/logs/stream", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.StreamImportRequestLogsAdmin)).ServeHTTP)
	router.Post("/api/v1/admin/images/import-requests/{id}/approve", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.ApproveImportRequestAdmin)).ServeHTTP)
	router.Post("/api/v1/admin/images/import-requests/{id}/reject", authMiddleware.RequirePermission(permissionService, "quarantine", "reject")(http.HandlerFunc(imageImportHandler.RejectImportRequestAdmin)).ServeHTTP)
	router.Get("/api/v1/images/import-requests/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.GetImportRequest))).ServeHTTP)
	router.Get("/api/v1/images/import-requests/{id}/workflow", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.GetImportRequestWorkflow))).ServeHTTP)
	router.Get("/api/v1/images/import-requests/{id}/logs", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.GetImportRequestLogs))).ServeHTTP)
	router.Get("/api/v1/images/import-requests/{id}/logs/stream", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "read")(http.HandlerFunc(imageImportHandler.StreamImportRequestLogs))).ServeHTTP)
	router.Post("/api/v1/images/import-requests/{id}/approve", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(imageImportHandler.ApproveImportRequest))).ServeHTTP)
	router.Post("/api/v1/images/import-requests/{id}/reject", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "reject")(http.HandlerFunc(imageImportHandler.RejectImportRequest))).ServeHTTP)
	router.Post("/api/v1/images/import-requests/{id}/release", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "quarantine", "release")(http.HandlerFunc(imageImportHandler.ReleaseImportRequest))).ServeHTTP)
	router.Post("/api/v1/images/import-requests/{id}/retry", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(imageImportHandler.RetryImportRequest))).ServeHTTP)
	router.Post("/api/v1/images/import-requests/{id}/withdraw", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(imageImportHandler.WithdrawImportRequest))).ServeHTTP)
	router.Post("/api/v1/images/scan-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(onDemandScanRequestHandler.CreateImportRequest))).ServeHTTP)
	router.Get("/api/v1/images/scan-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(onDemandScanRequestHandler.ListImportRequests))).ServeHTTP)
	router.Get("/api/v1/images/scan-requests/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(onDemandScanRequestHandler.GetImportRequest))).ServeHTTP)
	router.Post("/api/v1/images/scan-requests/{id}/retry", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(onDemandScanRequestHandler.RetryImportRequest))).ServeHTTP)
	router.Post("/api/v1/epr/registration-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(eprRegistrationHandler.CreateRequest))).ServeHTTP)
	router.Get("/api/v1/epr/registration-requests", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(eprRegistrationHandler.ListTenantRequests))).ServeHTTP)
	router.Post("/api/v1/epr/registration-requests/{id}/withdraw", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "create")(http.HandlerFunc(eprRegistrationHandler.WithdrawTenantRequest))).ServeHTTP)
	router.Get("/api/v1/admin/epr/registration-requests", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.ListAllRequests)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/{id}/approve", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.ApproveRequest)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/{id}/reject", authMiddleware.RequirePermission(permissionService, "quarantine", "reject")(http.HandlerFunc(eprRegistrationHandler.RejectRequest)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/{id}/suspend", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.SuspendRequest)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/{id}/reactivate", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.ReactivateRequest)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/{id}/revalidate", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.RevalidateRequest)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/bulk/suspend", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.BulkSuspendRequests)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/bulk/reactivate", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.BulkReactivateRequests)).ServeHTTP)
	router.Post("/api/v1/admin/epr/registration-requests/bulk/revalidate", authMiddleware.RequirePermission(permissionService, "quarantine", "approve")(http.HandlerFunc(eprRegistrationHandler.BulkRevalidateRequests)).ServeHTTP)

	// Register repository authentication routes
	repoAuthHandler := NewRepositoryAuthHandler(repoAuthService, projectService, logger)
	router.Post("/api/v1/repository-auth", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.CreateScopedRepositoryAuth))).ServeHTTP)
	router.Get("/api/v1/repository-auth", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(repoAuthHandler.ListScopedRepositoryAuth))).ServeHTTP)
	router.Put("/api/v1/repository-auth/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.UpdateRepositoryAuth))).ServeHTTP)
	router.Delete("/api/v1/repository-auth/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.DeleteRepositoryAuth))).ServeHTTP)
	router.Route("/api/v1/projects/{projectId}/repository-auth", func(r chi.Router) {
		r.Post("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.CreateRepositoryAuth))).ServeHTTP)
		r.Get("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(repoAuthHandler.GetRepositoryAuths))).ServeHTTP)
		r.Get("/available", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(repoAuthHandler.ListAvailableRepositoryAuths))).ServeHTTP)
		r.Post("/clone", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.CloneRepositoryAuth))).ServeHTTP)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(repoAuthHandler.GetRepositoryAuth))).ServeHTTP)
			r.Put("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.UpdateRepositoryAuth))).ServeHTTP)
			r.Delete("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.DeleteRepositoryAuth))).ServeHTTP)
			r.Post("/test-connection", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "manage_repository_auth")(http.HandlerFunc(repoAuthHandler.TestRepositoryAuth))).ServeHTTP)
		})
	})

	router.Get("/api/v1/git-providers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(gitProviderHandler.ListProviders))).ServeHTTP)
	router.Post("/api/v1/projects/{projectId}/repository-branches", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(repoBranchHandler.ListBranches))).ServeHTTP)
}
