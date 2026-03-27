package rest

import (
	"net/http"

	"github.com/srikarm/image-factory/internal/adapters/primary/http/handlers"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

func registerCoreAPIRoutes(
	router *Router,
	userService *user.Service,
	rbacRepo rbac.Repository,
	auditService *audit.Service,
	logger *zap.Logger,
	permissionService *rbac.PermissionService,
	authMiddleware *middleware.AuthMiddleware,
	tenantStatusMiddleware *middleware.TenantStatusMiddleware,
	tenantHandler *TenantHandler,
	roleHandler *RoleHandler,
	groupHandler *GroupHandler,
	workerHandler *WorkerHandler,
	analyticsHandler *AnalyticsHandler,
	configHandler *ConfigHandler,
	registryAuthHandler *RegistryAuthHandler,
	projectHandler *ProjectHandler,
	buildHandler *BuildHandler,
	buildService *build.Service,
	wsHub *WebSocketHub,
	buildTriggerHandler *handlers.BuildTriggerHandler,
	buildWebhookReceiverHandler *handlers.BuildWebhookReceiverHandler,
	projectNotificationTriggerHandler *ProjectNotificationTriggerHandler,
	notificationCenterHandler *NotificationCenterHandler,
	tenantDashboardHandler *TenantDashboardHandler,
	vmImageHandler *VMImageHandler,
	buildNotificationReplayHandler *BuildNotificationReplayHandler,
	infrastructureHandler *handlers.InfrastructureHandler,
	infrastructureProviderHandler *InfrastructureProviderHandler,
	packerTargetProfileHandler *PackerTargetProfileHandler,
	authHandler *AuthHandler,
	passwordResetHandler *PasswordResetHandler,
	bootstrapHandler *BootstrapHandler,
	onDemandScanRequestHandler *ImageImportHandler,
) {
	// Specific route for tenant roles - must come before the catch-all tenants/ route
	router.Get("/api/v1/tenants/{tenantId}/roles", authMiddleware.RequirePermission(permissionService, "role", "read")(http.HandlerFunc(roleHandler.GetRolesByTenant)).ServeHTTP)

	router.Get("/api/v1/slug/{slug}", tenantHandler.GetTenantBySlug)

	// Specific handler for /api/v1/tenants/{id}/groups
	router.Get("/api/v1/tenants/{id}/groups", authMiddleware.RequirePermission(permissionService, "tenant", "read")(http.HandlerFunc(groupHandler.ListGroupsByTenant)).ServeHTTP)

	// Register worker routes with API versioning
	router.Post("/api/v1/workers/register", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "workers", "register")(http.HandlerFunc(workerHandler.Register))).ServeHTTP)
	router.Delete("/api/v1/workers/unregister", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "workers", "unregister")(http.HandlerFunc(workerHandler.Unregister))).ServeHTTP)
	router.Post("/api/v1/workers/heartbeat", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "workers", "heartbeat")(http.HandlerFunc(workerHandler.Heartbeat))).ServeHTTP)
	router.Get("/api/v1/workers/stats", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "workers", "read")(http.HandlerFunc(workerHandler.GetStats))).ServeHTTP)

	// Register analytics routes with API versioning
	router.Get("/api/v1/analytics/eta", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "analytics", "read")(http.HandlerFunc(analyticsHandler.PredictETA))).ServeHTTP)
	router.Get("/api/v1/analytics/performance", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "analytics", "read")(http.HandlerFunc(analyticsHandler.GetPerformanceMetrics))).ServeHTTP)
	router.Get("/api/v1/analytics/health", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "analytics", "read")(http.HandlerFunc(analyticsHandler.GetHealthScore))).ServeHTTP)

	// Register build configuration routes (build_configs)
	router.Post("/api/v1/config/packer", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.CreatePackerConfig))).ServeHTTP)
	router.Post("/api/v1/config/buildx", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.CreateBuildxConfig))).ServeHTTP)
	router.Post("/api/v1/config/kaniko", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.CreateKanikoConfig))).ServeHTTP)
	router.Post("/api/v1/config/docker", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.CreateDockerConfig))).ServeHTTP)
	router.Post("/api/v1/config/paketo", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.CreatePaketoConfig))).ServeHTTP)
	router.Post("/api/v1/config/nix", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.CreateNixConfig))).ServeHTTP)
	router.Get("/api/v1/config/{buildId}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "read")(http.HandlerFunc(configHandler.GetConfig))).ServeHTTP)
	router.Delete("/api/v1/config/{buildId}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(configHandler.DeleteConfig))).ServeHTTP)
	router.Get("/api/v1/config", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "read")(http.HandlerFunc(configHandler.ListConfigsByMethod))).ServeHTTP)
	router.Get("/api/v1/config/presets", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "read")(http.HandlerFunc(configHandler.GetPresets))).ServeHTTP)

	// Registry authentication routes
	router.Post("/api/v1/registry-auth", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(registryAuthHandler.Create))).ServeHTTP)
	router.Get("/api/v1/registry-auth", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "read")(http.HandlerFunc(registryAuthHandler.List))).ServeHTTP)
	router.Post("/api/v1/registry-auth/{id}/test-permissions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "read")(http.HandlerFunc(registryAuthHandler.TestPermissions))).ServeHTTP)
	router.Put("/api/v1/registry-auth/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(registryAuthHandler.Update))).ServeHTTP)
	router.Delete("/api/v1/registry-auth/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "config", "write")(http.HandlerFunc(registryAuthHandler.Delete))).ServeHTTP)

	// Admin maintenance routes
	router.Post("/api/v1/admin/projects/purge-deleted", authMiddleware.RequirePermission(permissionService, "system", "manage_config")(http.HandlerFunc(projectHandler.PurgeDeletedProjects)).ServeHTTP)

	// Register build routes with API versioning
	router.Post("/api/v1/builds", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(buildHandler.CreateBuild))).ServeHTTP)
	router.Get("/api/v1/builds", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "list")(http.HandlerFunc(buildHandler.ListBuilds))).ServeHTTP)

	// Build-specific routes with {id} parameter
	router.Get("/api/v1/builds/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetBuild))).ServeHTTP)
	router.Delete("/api/v1/builds/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "delete")(http.HandlerFunc(buildHandler.DeleteBuild))).ServeHTTP)

	// Build action routes
	router.Post("/api/v1/builds/{id}/start", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(buildHandler.StartBuild))).ServeHTTP)
	router.Post("/api/v1/builds/{id}/cancel", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "cancel")(http.HandlerFunc(buildHandler.CancelBuild))).ServeHTTP)
	router.Post("/api/v1/builds/{id}/retry", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(buildHandler.RetryBuild))).ServeHTTP)
	router.Post("/api/v1/builds/{id}/clone", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(buildHandler.CloneBuild))).ServeHTTP)

	// Build status and logs routes
	router.Get("/api/v1/builds/{id}/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetStatus))).ServeHTTP)
	router.Get("/api/v1/builds/{id}/executions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetExecutions))).ServeHTTP)
	router.Get("/api/v1/builds/{id}/trace", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetTrace))).ServeHTTP)
	router.Get("/api/v1/builds/{id}/trace/export", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetTraceExport))).ServeHTTP)
	router.Get("/api/v1/builds/{id}/logs", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetLogs))).ServeHTTP)
	router.Get("/api/v1/builds/{id}/workflow", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildHandler.GetWorkflow))).ServeHTTP)

	// WebSocket handler for real-time build logs
	buildWSHandler := NewBuildWSHandler(wsHub, buildService, logger)

	router.Get("/api/v1/builds/{id}/logs/stream", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildWSHandler.HandleBuildLogs))).ServeHTTP)

	// WebSocket handler for general build events
	// Preferred endpoint (legacy + stable in current clients).
	router.Get("/api/builds/events", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildWSHandler.HandleBuildEvents))).ServeHTTP)
	// Backward/forward-compatible alias for clients using versioned API prefix.
	router.Get("/api/v1/builds/events", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildWSHandler.HandleBuildEvents))).ServeHTTP)
	router.Get("/api/notifications/events", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(buildWSHandler.HandleNotificationEvents))).ServeHTTP)

	// Register project routes with API versioning
	router.Post("/api/v1/projects", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "create")(http.HandlerFunc(projectHandler.CreateProject))).ServeHTTP)
	router.Get("/api/v1/projects", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(projectHandler.ListProjects))).ServeHTTP)

	router.Get("/api/v1/projects/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(projectHandler.GetProject))).ServeHTTP)
	router.Put("/api/v1/projects/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectHandler.UpdateProject))).ServeHTTP)
	router.Delete("/api/v1/projects/{id}", tenantStatusMiddleware.EnforceTenantStatus(middleware.RequireAuth(userService, rbacRepo, auditService, logger)(http.HandlerFunc(projectHandler.DeleteProject))).ServeHTTP)
	router.Get("/api/v1/projects/{id}/build-settings", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(projectHandler.GetProjectBuildSettings))).ServeHTTP)
	router.Put("/api/v1/projects/{id}/build-settings", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectHandler.UpdateProjectBuildSettings))).ServeHTTP)
	router.Get("/api/v1/projects/{id}/sources", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(projectHandler.ListProjectSources))).ServeHTTP)
	router.Post("/api/v1/projects/{id}/sources", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectHandler.CreateProjectSource))).ServeHTTP)
	router.Patch("/api/v1/projects/{id}/sources/{sourceID}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectHandler.UpdateProjectSource))).ServeHTTP)
	router.Delete("/api/v1/projects/{id}/sources/{sourceID}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectHandler.DeleteProjectSource))).ServeHTTP)
	router.Get("/api/v1/projects/{id}/webhook-receipts", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(projectHandler.ListProjectWebhookReceipts))).ServeHTTP)
	router.Get("/api/v1/projects/{id}/notification-triggers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(projectNotificationTriggerHandler.GetProjectNotificationTriggers))).ServeHTTP)
	router.Put("/api/v1/projects/{id}/notification-triggers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectNotificationTriggerHandler.UpdateProjectNotificationTriggers))).ServeHTTP)
	router.Delete("/api/v1/projects/{id}/notification-triggers/{trigger_id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update")(http.HandlerFunc(projectNotificationTriggerHandler.DeleteProjectNotificationTrigger))).ServeHTTP)
	router.Get("/api/v1/notifications", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.ListNotifications))).ServeHTTP)
	router.Get("/api/v1/notifications/unread-count", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.GetUnreadCount))).ServeHTTP)
	router.Post("/api/v1/notifications/{id}/read", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.MarkAsRead))).ServeHTTP)
	router.Delete("/api/v1/notifications/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.DeleteOne))).ServeHTTP)
	router.Post("/api/v1/notifications/delete-bulk", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.DeleteBulk))).ServeHTTP)
	router.Post("/api/v1/notifications/read-all", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.MarkAllAsRead))).ServeHTTP)
	router.Delete("/api/v1/notifications/read", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(notificationCenterHandler.DeleteRead))).ServeHTTP)
	router.Get("/api/v1/dashboard/tenant/summary", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(tenantDashboardHandler.GetTenantSummary))).ServeHTTP)
	router.Get("/api/v1/dashboard/tenant/activity", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(tenantDashboardHandler.GetTenantActivity))).ServeHTTP)
	router.Get("/api/v1/images/vm", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(vmImageHandler.ListTenantVMImages))).ServeHTTP)
	router.Get("/api/v1/images/vm/{executionId}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "image", "read")(http.HandlerFunc(vmImageHandler.GetTenantVMImage))).ServeHTTP)
	router.Get("/api/v1/admin/tenants/{tenant_id}/notification-triggers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "read")(http.HandlerFunc(projectNotificationTriggerHandler.GetTenantNotificationTriggers))).ServeHTTP)
	router.Put("/api/v1/admin/tenants/{tenant_id}/notification-triggers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "update")(http.HandlerFunc(projectNotificationTriggerHandler.UpdateTenantNotificationTriggers))).ServeHTTP)
	router.Get("/api/v1/admin/tenants/{tenant_id}/notification-replay/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "read")(http.HandlerFunc(buildNotificationReplayHandler.GetTenantBuildNotificationReplayStatus))).ServeHTTP)
	router.Post("/api/v1/admin/tenants/{tenant_id}/notification-replay", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "update")(http.HandlerFunc(buildNotificationReplayHandler.ReplayTenantBuildNotificationFailures))).ServeHTTP)
	router.Get("/api/v1/projects/{projectId}/build-context-suggestions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "read")(http.HandlerFunc(buildHandler.GetBuildContextSuggestions))).ServeHTTP)

	// Register project member routes
	router.Post("/api/v1/projects/{projectID}/members", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "add_member")(http.HandlerFunc(projectHandler.AddMember))).ServeHTTP)
	router.Get("/api/v1/projects/{projectID}/members", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "view_members")(http.HandlerFunc(projectHandler.ListMembers))).ServeHTTP)

	router.Delete("/api/v1/projects/{projectID}/members/{userID}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "remove_member")(http.HandlerFunc(projectHandler.RemoveMember))).ServeHTTP)
	router.Patch("/api/v1/projects/{projectID}/members/{userID}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "projects", "update_member_role")(http.HandlerFunc(projectHandler.UpdateMemberRole))).ServeHTTP)

	// Register build trigger routes
	router.Post("/api/v1/projects/{projectID}/triggers/webhook", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "manage_triggers")(http.HandlerFunc(buildTriggerHandler.CreateProjectWebhookTrigger))).ServeHTTP)
	router.Post("/api/v1/projects/{projectID}/builds/{buildID}/triggers/webhook", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "manage_triggers")(http.HandlerFunc(buildTriggerHandler.CreateWebhookTrigger))).ServeHTTP)
	router.Post("/api/v1/projects/{projectID}/builds/{buildID}/triggers/schedule", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "manage_triggers")(http.HandlerFunc(buildTriggerHandler.CreateScheduleTrigger))).ServeHTTP)
	router.Post("/api/v1/projects/{projectID}/builds/{buildID}/triggers/git-event", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "manage_triggers")(http.HandlerFunc(buildTriggerHandler.CreateGitEventTrigger))).ServeHTTP)
	router.Get("/api/v1/projects/{projectID}/builds/{buildID}/triggers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildTriggerHandler.GetBuildTriggers))).ServeHTTP)
	router.Get("/api/v1/projects/{projectID}/triggers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "read")(http.HandlerFunc(buildTriggerHandler.GetProjectTriggers))).ServeHTTP)
	router.Patch("/api/v1/projects/{projectID}/triggers/{triggerID}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "manage_triggers")(http.HandlerFunc(buildTriggerHandler.UpdateProjectWebhookTrigger))).ServeHTTP)
	router.Delete("/api/v1/projects/{projectID}/triggers/{triggerID}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "manage_triggers")(http.HandlerFunc(buildTriggerHandler.DeleteTrigger))).ServeHTTP)
	router.Post("/api/v1/webhooks/{provider}/{projectID}", http.HandlerFunc(buildWebhookReceiverHandler.ReceiveProjectWebhook))

	// Register infrastructure routes
	router.Post("/api/v1/builds/infrastructure-recommendation", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "build", "create")(http.HandlerFunc(infrastructureHandler.GetInfrastructureRecommendation))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/usage", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "system", "read")(http.HandlerFunc(infrastructureHandler.GetInfrastructureUsage))).ServeHTTP)

	// Register infrastructure provider routes
	router.Get("/api/v1/admin/infrastructure/providers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.ListProviders))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "create")(http.HandlerFunc(infrastructureProviderHandler.CreateProvider))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/prepare/summary", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProviderPrepareSummaries))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProvider))).ServeHTTP)
	router.Put("/api/v1/admin/infrastructure/providers/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.UpdateProvider))).ServeHTTP)
	router.Delete("/api/v1/admin/infrastructure/providers/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "delete")(http.HandlerFunc(infrastructureProviderHandler.DeleteProvider))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/test-connection", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.TestProviderConnection))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/health", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProviderHealth))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/readiness", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProviderReadiness))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/quarantine-dispatch-readiness", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProviderQuarantineDispatchReadiness))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/prepare", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.PrepareProvider))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/prepare/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProviderPrepareStatus))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/prepare/runs", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.ListProviderPrepareRuns))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/prepare/runs/{run_id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetProviderPrepareRun))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/prepare/stream", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.StreamProviderPrepareStatus))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/provision-namespace", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.ProvisionTenantNamespace))).ServeHTTP)
	// Backward-compatible alias; prefer /provision-namespace.
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.PrepareTenantNamespace))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/deprovision-namespace", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.DeprovisionTenantNamespace))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/provision-namespace/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetTenantNamespacePrepareStatus))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/provision-namespace/stream", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.StreamTenantNamespacePrepareStatus))).ServeHTTP)
	// Backward-compatible aliases; prefer /provision-namespace/*.
	router.Get("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetTenantNamespacePrepareStatus))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace/stream", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.StreamTenantNamespacePrepareStatus))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tenants/reconcile-stale", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.ReconcileStaleTenantNamespaces))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tenants/reconcile-selected", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.ReconcileSelectedTenantNamespaces))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tekton/install", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.InstallTekton))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tekton/upgrade", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.UpgradeTekton))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tekton/retry", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.RetryTektonJob))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/tekton/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.GetTektonStatus))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/tekton/validate", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.ValidateTekton))).ServeHTTP)
	router.Patch("/api/v1/admin/infrastructure/providers/{id}/status", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.ToggleProviderStatus))).ServeHTTP)
	router.Get("/api/v1/admin/infrastructure/providers/{id}/permissions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(infrastructureProviderHandler.ListProviderPermissions))).ServeHTTP)
	router.Post("/api/v1/admin/infrastructure/providers/{id}/permissions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.GrantProviderPermission))).ServeHTTP)
	router.Delete("/api/v1/admin/infrastructure/providers/{id}/permissions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(infrastructureProviderHandler.RevokeProviderPermission))).ServeHTTP)
	router.Get("/api/v1/admin/packer-target-profiles", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(packerTargetProfileHandler.ListProfiles))).ServeHTTP)
	router.Post("/api/v1/admin/packer-target-profiles", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "create")(http.HandlerFunc(packerTargetProfileHandler.CreateProfile))).ServeHTTP)
	router.Get("/api/v1/admin/packer-target-profiles/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "read")(http.HandlerFunc(packerTargetProfileHandler.GetProfile))).ServeHTTP)
	router.Put("/api/v1/admin/packer-target-profiles/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(packerTargetProfileHandler.UpdateProfile))).ServeHTTP)
	router.Delete("/api/v1/admin/packer-target-profiles/{id}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "delete")(http.HandlerFunc(packerTargetProfileHandler.DeleteProfile))).ServeHTTP)
	router.Post("/api/v1/admin/packer-target-profiles/{id}/validate", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "update")(http.HandlerFunc(packerTargetProfileHandler.ValidateProfile))).ServeHTTP)

	// User-facing infrastructure provider routes
	router.Get("/api/v1/infrastructure/providers/available", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "infrastructure", "select")(http.HandlerFunc(infrastructureProviderHandler.GetAvailableProviders))).ServeHTTP)

	// Register authentication routes
	router.Get("/api/v1/auth/login-options", authHandler.LoginOptions)
	router.Post("/api/v1/auth/login", authHandler.Login)
	router.Post("/api/v1/auth/refresh", authHandler.RefreshToken)
	router.Post("/api/v1/auth/logout", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(authHandler.Logout)).ServeHTTP)
	router.Get("/api/v1/auth/me", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(authHandler.Me)).ServeHTTP)
	router.Post("/api/v1/auth/change-password", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(authHandler.ChangePassword)).ServeHTTP)
	router.Post("/api/v1/auth/ldap/search-users", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(authHandler.SearchLDAPUsers)).ServeHTTP)
	router.Post("/api/v1/auth/forgot-password", passwordResetHandler.RequestPasswordReset)
	router.Post("/api/v1/auth/reset-password", passwordResetHandler.ResetPassword)
	router.Post("/api/v1/auth/validate-reset-token", passwordResetHandler.ValidateResetToken)

	// Register bootstrap/setup routes
	router.Get("/api/v1/bootstrap/status", bootstrapHandler.GetStatus)
	router.Get("/api/v1/bootstrap/defaults", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(bootstrapHandler.GetDefaults)).ServeHTTP)
	router.Post("/api/v1/bootstrap/start", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(bootstrapHandler.StartSetup)).ServeHTTP)
	router.Post("/api/v1/bootstrap/steps/{step}/save", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(bootstrapHandler.SaveStep)).ServeHTTP)
	router.Post("/api/v1/bootstrap/save-all", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(bootstrapHandler.SaveAll)).ServeHTTP)
	router.Post("/api/v1/bootstrap/complete", authMiddleware.AuthenticateWithoutTenant(http.HandlerFunc(bootstrapHandler.CompleteSetup)).ServeHTTP)
}
