package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/adapters/primary/http/handlers"
	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	epradapter "github.com/srikarm/image-factory/internal/adapters/secondary/epr"
	policyadapter "github.com/srikarm/image-factory/internal/adapters/secondary/policy"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	workflowapprovaladapter "github.com/srikarm/image-factory/internal/adapters/secondary/workflowapproval"
	appbuild "github.com/srikarm/image-factory/internal/application/build"
	appbuildnotifications "github.com/srikarm/image-factory/internal/application/buildnotifications"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"github.com/srikarm/image-factory/internal/domain/health"
	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/notification"
	"github.com/srikarm/image-factory/internal/domain/packertarget"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/domain/worker"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/denialtelemetry"
	"github.com/srikarm/image-factory/internal/infrastructure/k8s"
	"github.com/srikarm/image-factory/internal/infrastructure/logdetector"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	persistencePg "github.com/srikarm/image-factory/internal/infrastructure/persistence/postgres"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
	"go.uber.org/zap"
)

// Helper function to get integer from environment variable
func getIntEnv(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func resolveImageCatalogStrictExecutionID(ctx context.Context, systemConfigService *systemconfig.Service, logger *zap.Logger) bool {
	strict := true // secure default: do not silently mask missing execution contract

	if systemConfigService != nil {
		cfg, err := systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeBuild, "build")
		if err == nil && cfg != nil {
			payload := map[string]interface{}{}
			if unmarshalErr := json.Unmarshal(cfg.ConfigValue(), &payload); unmarshalErr != nil {
				if logger != nil {
					logger.Warn("Failed to parse system build config for image catalog strict execution-id setting", zap.Error(unmarshalErr))
				}
			} else if raw, ok := payload["image_catalog_strict_execution_id"]; ok {
				if value, ok := raw.(bool); ok {
					strict = value
				}
			}
		} else if err != nil && !errors.Is(err, systemconfig.ErrConfigNotFound) {
			if logger != nil {
				logger.Warn("Failed to load system build config for image catalog strict execution-id setting", zap.Error(err))
			}
		}
	}

	if raw := strings.TrimSpace(os.Getenv("IF_IMAGE_CATALOG_STRICT_EXECUTION_ID")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			if logger != nil {
				logger.Warn("Invalid IF_IMAGE_CATALOG_STRICT_EXECUTION_ID value; using system-config/default value", zap.String("value", raw), zap.Error(err))
			}
		} else {
			strict = parsed
		}
	}
	return strict
}

type imageCatalogAlertNotifier struct {
	deliveryRepo *postgres.BuildNotificationDeliveryRepository
	wsHub        *WebSocketHub
	logger       *zap.Logger
}

func (n *imageCatalogAlertNotifier) NotifyCatalogIngestIssue(
	ctx context.Context,
	tenantID uuid.UUID,
	buildID uuid.UUID,
	imageID uuid.UUID,
	title string,
	message string,
	metadata map[string]interface{},
) error {
	if n == nil || n.deliveryRepo == nil {
		return nil
	}
	if tenantID == uuid.Nil {
		return nil
	}
	recipients, err := n.deliveryRepo.ListTenantAdminUserIDs(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}
	metadataJSON, _ := json.Marshal(metadata)
	fullMessage := strings.TrimSpace(message)
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" && string(metadataJSON) != "{}" {
		fullMessage = strings.TrimSpace(fullMessage + " details=" + string(metadataJSON))
	}
	rows := make([]appbuildnotifications.InAppNotificationRow, 0, len(recipients))
	for _, userID := range recipients {
		related := buildID
		if imageID != uuid.Nil {
			related = imageID
		}
		rows = append(rows, appbuildnotifications.InAppNotificationRow{
			ID:                  uuid.New(),
			UserID:              userID,
			TenantID:            tenantID,
			Title:               title,
			Message:             fullMessage,
			NotificationType:    "system_alert",
			RelatedResourceType: "build",
			RelatedResourceID:   &related,
			Channel:             string(buildnotification.ChannelInApp),
		})
	}
	if err := n.deliveryRepo.InsertInAppNotifications(ctx, rows); err != nil {
		return err
	}
	if n.wsHub != nil {
		for _, row := range rows {
			notificationID := row.ID
			n.wsHub.BroadcastNotificationEvent(
				row.TenantID,
				row.UserID,
				"notification.created",
				&notificationID,
				map[string]interface{}{
					"notification_type":     row.NotificationType,
					"related_resource_type": row.RelatedResourceType,
				},
			)
		}
	}
	if n.logger != nil {
		n.logger.Warn("Image catalog ingest alert emitted",
			zap.String("tenant_id", tenantID.String()),
			zap.String("build_id", buildID.String()),
			zap.Int("recipients", len(rows)))
	}
	return nil
}

// Router represents the HTTP router
type Router struct {
	chi.Router
	logger             *zap.Logger
	healthService      *health.Service
	healthHandler      *HealthHandler
	auditMiddleware    *middleware.AuditMiddleware
	httpRequestSignals *middleware.HTTPRequestSignalStore
}

// setupTenantRoutes sets up routes for tenant operations
func setupTenantRoutes(r chi.Router, tenantHandler *TenantHandler, userHandler *UserHandler, authMiddleware *middleware.AuthMiddleware, tenantStatusMiddleware *middleware.TenantStatusMiddleware, permissionService *rbac.PermissionService) {
	r.Route("/api/v1/tenants", func(r chi.Router) {
		// Tenant CRUD operations
		r.Get("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "list")(http.HandlerFunc(tenantHandler.ListTenants))).ServeHTTP)
		r.Post("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "create")(http.HandlerFunc(tenantHandler.CreateTenant))).ServeHTTP)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "read")(http.HandlerFunc(tenantHandler.GetTenant))).ServeHTTP)
			r.Put("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "update")(http.HandlerFunc(tenantHandler.UpdateTenant))).ServeHTTP)
			r.Patch("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "update")(http.HandlerFunc(tenantHandler.UpdateTenant))).ServeHTTP)
			r.Delete("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "delete")(http.HandlerFunc(tenantHandler.DeleteTenant))).ServeHTTP)

			// Tenant-specific operations
			r.Post("/activate", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "activate")(http.HandlerFunc(tenantHandler.ActivateTenant))).ServeHTTP)

			// Tenant user management routes
			r.Get("/users", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "read")(http.HandlerFunc(userHandler.GetTenantUsers))).ServeHTTP)
			r.Post("/users", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "manage_users")(http.HandlerFunc(userHandler.AddUserToTenant))).ServeHTTP)

			r.Route("/users/{userId}", func(r chi.Router) {
				r.Delete("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "manage_users")(http.HandlerFunc(userHandler.RemoveUserFromTenant))).ServeHTTP)
				r.Patch("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "manage_users")(http.HandlerFunc(userHandler.UpdateTenantUserRole))).ServeHTTP)
			})
		})
	})

	// Tenant slug-based routes
	r.Get("/api/v1/tenants/by-slug/{slug}", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "tenant", "read")(http.HandlerFunc(tenantHandler.GetTenantBySlug))).ServeHTTP)
}

// setupUserRoutes sets up routes for user operations
func setupUserRoutes(r chi.Router, userHandler *UserHandler, authMiddleware *middleware.AuthMiddleware, tenantStatusMiddleware *middleware.TenantStatusMiddleware, permissionService *rbac.PermissionService) {
	r.Route("/api/v1/users", func(r chi.Router) {
		// User CRUD operations
		r.Get("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "list")(http.HandlerFunc(userHandler.ListUsers))).ServeHTTP)
		r.Post("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "create")(http.HandlerFunc(userHandler.CreateUser))).ServeHTTP)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "read")(http.HandlerFunc(userHandler.GetUser))).ServeHTTP)
			r.Put("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "update")(http.HandlerFunc(userHandler.UpdateUser))).ServeHTTP)
			r.Delete("/", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "delete")(http.HandlerFunc(userHandler.DeleteUser))).ServeHTTP)

			// User action endpoints: suspend and activate
			r.Post("/suspend", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "update")(http.HandlerFunc(userHandler.SuspendUser))).ServeHTTP)
			r.Post("/activate", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "update")(http.HandlerFunc(userHandler.ActivateUser))).ServeHTTP)

			// User activity and history endpoints
			r.Get("/activity", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "read")(http.HandlerFunc(userHandler.GetUserActivity))).ServeHTTP)
			r.Get("/login-history", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "read")(http.HandlerFunc(userHandler.GetLoginHistory))).ServeHTTP)
			r.Get("/sessions", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "read")(http.HandlerFunc(userHandler.GetUserSessions))).ServeHTTP)

			// Bulk user role update endpoint
			r.Patch("/roles", tenantStatusMiddleware.EnforceTenantStatus(authMiddleware.RequirePermission(permissionService, "user", "manage_roles")(http.HandlerFunc(userHandler.UpdateUserRoles))).ServeHTTP)
		})
	})
}

// NewRouter creates a new HTTP router with health service
func NewRouter(cfg *config.Config, logger *zap.Logger, db interface{}, auditService *audit.Service) *Router {
	router := chi.NewRouter()

	// Add default middleware
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.RealIP)
	router.Use(chiMiddleware.Maybe(chiMiddleware.Logger, func(r *http.Request) bool {
		// Skip noisy request logging for websocket endpoints.
		// Websocket auth/permission errors are still logged by auth middleware.
		return r.URL.Path != "/api/builds/events" &&
			r.URL.Path != "/api/notifications/events" &&
			r.URL.Path != "/health" &&
			r.URL.Path != "/ready" &&
			r.URL.Path != "/alive" &&
			!strings.HasSuffix(r.URL.Path, "/logs/stream")
	}))
	router.Use(chiMiddleware.Recoverer)

	// Create health service from database connection
	sqlxDB := db.(*sqlx.DB)
	healthService := health.NewService(sqlxDB, cfg, logger)
	healthHandler := NewHealthHandler(healthService, logger)
	httpRequestSignals := middleware.NewHTTPRequestSignalStore()
	router.Use(httpRequestSignals.Middleware)

	r := &Router{
		Router:             router,
		logger:             logger,
		healthService:      healthService,
		healthHandler:      healthHandler,
		httpRequestSignals: httpRequestSignals,
	}

	// Initialize audit middleware if provided
	if auditService != nil {
		r.auditMiddleware = middleware.NewAuditMiddleware(auditService, logger)
		router.Use(r.auditMiddleware.Wrap)
	}

	// Register health check endpoints
	router.Get("/health", healthHandler.HandleCheck)
	router.Get("/healthz", healthHandler.HandleCheck)
	router.Get("/ready", healthHandler.HandleReady)
	router.Get("/alive", healthHandler.HandleAlive)
	router.Get("/", r.defaultHandler)

	return r
}

func (r *Router) HTTPRequestSignalStore() *middleware.HTTPRequestSignalStore {
	if r == nil {
		return nil
	}
	return r.httpRequestSignals
}

// SetAuditMiddleware sets the audit middleware for the router
func (r *Router) SetAuditMiddleware(auditMiddleware *middleware.AuditMiddleware) {
	r.auditMiddleware = auditMiddleware
	r.Use(auditMiddleware.Wrap)
}

// defaultHandler handles all other requests
func (r *Router) defaultHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Image Factory API","version":"1.0.0"}`))
}

// SetupRoutes configures all HTTP routes for the application
func SetupRoutes(
	router *Router,
	tenantRepo tenant.Repository,
	tenantService *tenant.Service,
	buildRepo build.Repository,
	buildService *build.Service,
	buildExecutionService build.BuildExecutionService,
	projectRepo project.Repository,
	projectService *project.Service,
	userRepo user.Repository,
	userService *user.Service,
	rbacRepo rbac.Repository,
	rbacService *rbac.Service,
	systemConfigRepo systemconfig.Repository,
	logger *zap.Logger,
	ldapService *user.LDAPService,
	auditService *audit.Service,
	notificationTemplateRepo notification.Repository,
	wsHub *WebSocketHub,
	db interface{},
	cfg *config.Config,
	buildPolicyService *build.BuildPolicyService,
	dispatcherMetricsProvider DispatcherMetricsProvider,
	dispatcherController DispatcherController,
	orchestratorController WorkflowOrchestratorController,
	dispatcherRuntimeReader DispatcherRuntimeReader,
	processStatusProvider runtimehealth.Provider,
	dispatcherEnabled bool,
	infrastructureEventPublisher infrastructure.EventPublisher,
	releaseComplianceMetrics *releasecompliance.Metrics,
	eventBus messaging.EventBus,
) {
	// Get the database connection early
	sqlxDB := db.(*sqlx.DB)

	// Initialize handlers
	tenantHandler := NewTenantHandler(tenantService, rbacService, ldapService, auditService, logger)
	// Set optional services on tenant handler
	tenantHandler.SetUserService(userService)
	tenantHandler.SetDB(db)
	platform := initializePlatformWiring(
		sqlxDB,
		cfg,
		systemConfigRepo,
		notificationTemplateRepo,
		eventBus,
		infrastructureEventPublisher,
		logger,
	)
	systemConfigService := platform.systemConfigService
	bootstrapService := platform.bootstrapService
	emailDomainService := platform.emailDomainService
	notificationService := platform.notificationService
	infrastructureService := platform.infrastructureService
	infrastructureProviderHandler := platform.infrastructureProviderHand
	packerTargetProfileRepo := postgres.NewPackerTargetProfileRepository(sqlxDB, logger)
	packerTargetProfileService := packertarget.NewService(packerTargetProfileRepo)
	if buildService != nil {
		buildService.SetPackerTargetProfileLookup(packerTargetProfileService)
	}
	packerTargetProfileHandler := NewPackerTargetProfileHandler(packerTargetProfileService, logger)
	tenantHandler.SetNotificationService(notificationService)
	tenantHandler.SetInfrastructureService(infrastructureService)

	workflowRepo := postgres.NewWorkflowRepository(sqlxDB, logger)
	imageImportNotificationRepo := postgres.NewBuildNotificationDeliveryRepository(sqlxDB, logger)
	projectNotificationTriggerRepo := postgres.NewProjectNotificationTriggerRepository(sqlxDB, logger)
	projectSourceRepo := postgres.NewProjectSourceRepository(sqlxDB, logger)
	webhookReceiptRepo := postgres.NewWebhookReceiptRepository(sqlxDB, logger)
	projectBuildSettingsRepo := postgres.NewProjectBuildSettingsRepository(sqlxDB, logger)
	projectNotificationTriggerService := buildnotification.NewService(projectNotificationTriggerRepo)
	buildAppService := appbuild.NewService(buildService, logger)
	buildAppService.SetWorkflowWriter(workflowRepo)
	buildAppService.SetProjectTenantLookup(func(ctx context.Context, projectID uuid.UUID) (uuid.UUID, bool, error) {
		if projectService == nil {
			return uuid.Nil, false, nil
		}
		project, err := projectService.GetProject(ctx, projectID)
		if err != nil || project == nil {
			return uuid.Nil, false, nil
		}
		return project.TenantID(), true, nil
	})
	buildHandler := NewBuildHandler(buildService, buildAppService, buildExecutionService, workflowRepo, auditService, infrastructureService, logger)
	if projectService != nil {
		buildHandler.SetProjectContextLookup(func(ctx context.Context, projectID uuid.UUID) (uuid.UUID, string, string, error) {
			project, err := projectService.GetProject(ctx, projectID)
			if err != nil {
				return uuid.Nil, "", "", err
			}
			if project == nil {
				return uuid.Nil, "", "", fmt.Errorf("project not found")
			}
			source, err := projectSourceRepo.FindDefaultByProjectID(ctx, projectID)
			if err != nil {
				return uuid.Nil, "", "", err
			}
			if source == nil {
				return project.TenantID(), "", "", nil
			}
			return project.TenantID(), source.RepositoryURL, source.DefaultBranch, nil
		})
	}
	buildHandler.SetProcessStatusProvider(processStatusProvider)
	buildTriggerHandler := handlers.NewBuildTriggerHandler(buildService, logger)
	buildWebhookReceiverHandler := handlers.NewBuildWebhookReceiverHandler(buildService, projectService, projectSourceRepo, webhookReceiptRepo, logger)

	// Initialize infrastructure selector and handler
	infrastructureSelector := k8s.NewInfrastructureSelector(true) // TODO: Make this configurable
	infrastructureHandler := handlers.NewInfrastructureHandler(buildService, infrastructureService, infrastructureSelector, logger)

	projectHandler := NewProjectHandlerWithConfig(projectService, projectSourceRepo, webhookReceiptRepo, projectBuildSettingsRepo, systemConfigService, auditService, rbacService, logger)
	projectNotificationTriggerHandler := NewProjectNotificationTriggerHandler(projectNotificationTriggerService, projectService, logger)
	notificationCenterHandler := NewNotificationCenterHandler(sqlxDB, logger, wsHub)
	buildNotificationReplayHandler := NewBuildNotificationReplayHandler(sqlxDB, logger)
	tenantDashboardHandler := NewTenantDashboardHandler(sqlxDB, logger)
	vmImageHandler := NewVMImageHandler(sqlxDB, logger)
	authHandler := NewAuthHandler(userService, ldapService, rbacService, auditService, bootstrapService, logger)
	userHandler := NewUserHandler(userService, rbacService, auditService, cfg, logger)
	// Set notification service on user handler for member addition/removal notifications
	userHandler.SetNotificationService(notificationService)
	// Set database on user handler for group operations
	userHandler.SetDatabase(sqlxDB)
	roleHandler := NewRoleHandler(rbacService, auditService, logger)
	permissionRepo := postgres.NewPermissionRepository(sqlxDB.DB)
	permissionService := rbac.NewPermissionService(permissionRepo)
	roleHandler.SetPermissionService(permissionService) // Set permission service on role handler
	permissionHandler := NewPermissionHandler(permissionService, rbacService, auditService, logger)
	denialMetrics := denialtelemetry.NewMetrics()
	releaseMetrics := releasetelemetry.NewMetrics()
	systemConfigHandler := NewSystemConfigHandler(systemConfigService, auditService, logger, cfg.Server.Environment)
	systemComponentsStatusHandler := NewSystemComponentsStatusHandler(systemConfigService, logger)
	systemStatsHandler := NewSystemStatsHandler(userService, tenantService, denialMetrics, releaseMetrics, releaseComplianceMetrics, logger)
	dispatcherMetricsHandler := NewDispatcherMetricsHandler(dispatcherMetricsProvider, dispatcherRuntimeReader, logger)
	dispatcherControlHandler := NewDispatcherControlHandler(dispatcherController, dispatcherRuntimeReader, wsHub)
	orchestratorControlHandler := NewOrchestratorControlHandler(orchestratorController, processStatusProvider, wsHub)
	executionPipelineHealthHandler := NewExecutionPipelineHealthHandler(dispatcherController, dispatcherRuntimeReader, processStatusProvider, dispatcherEnabled, workflowRepo)
	executionPipelineMetricsHandler := NewExecutionPipelineMetricsHandler(processStatusProvider)
	auditHandler := NewAuditHandler(auditService, logger)
	workflowHandler := NewWorkflowHandler(sqlxDB, logger)
	// Initialize password reset handler
	passwordResetHandler := NewPasswordResetHandler(userService, auditService, logger)
	// Initialize group handler and set notification service
	groupHandler := NewGroupHandler(sqlxDB, auditService, logger)
	groupHandler.SetNotificationService(notificationService)
	// Initialize onboarding service and handler
	onboardingService := tenant.NewOnboardingService(tenantRepo, tenantService.GetEventPublisher(), logger)
	onboardingHandler := NewOnboardingHandler(onboardingService, userService, db.(*sqlx.DB), logger)

	// Initialize user invitation handler
	userInvitationHandler := NewUserInvitationHandler(userService, rbacService, auditService, logger, ldapService, cfg)
	userInvitationHandler.SetNotificationService(notificationService)
	userInvitationHandler.SetDatabase(sqlxDB)

	// Initialize image repository and service
	imageRepo := postgres.NewImageRepository(sqlxDB, logger)

	// Create permission checker with tenant validation
	permissionChecker := &rbacPermissionChecker{
		rbacService: rbacService,
		tenantRepo:  tenantRepo,
	}
	auditLogger := &auditLoggerAdapter{auditService: auditService}
	versionRepo := postgres.NewImageVersionRepository(sqlxDB, logger)
	tagRepo := postgres.NewImageTagRepository(sqlxDB, logger)

	imageService := image.NewService(imageRepo, versionRepo, tagRepo, permissionChecker, auditLogger, logger)
	imageHandler := NewImageHandler(imageService, tenantRepo, auditService, logger)
	imageHandler.SetVersionRepository(versionRepo)
	imageHandler.SetTagRepository(tagRepo)
	imageHandler.SetDB(sqlxDB)
	imageHandler.SetDenialMetrics(denialMetrics)
	eprValidator := epradapter.NewExternalValidator(logger, systemConfigService, &http.Client{Timeout: 10 * time.Second})
	eprRegistrationRepo := postgres.NewEPRRegistrationRequestRepository(sqlxDB, logger)
	systemStatsHandler.SetEPRStatsReader(eprRegistrationRepo)
	eprRegistrationService := eprregistration.NewService(eprRegistrationRepo, logger)
	eprLifecycleTransitionIntervalSeconds := getIntEnv("IF_EPR_LIFECYCLE_TRANSITION_INTERVAL_SECONDS", 3600)
	if eprLifecycleTransitionIntervalSeconds < 60 {
		eprLifecycleTransitionIntervalSeconds = 60
	}
	eprLifecycleExpiringWindowHours := getIntEnv("IF_EPR_LIFECYCLE_EXPIRING_WINDOW_HOURS", 24*30)
	if eprLifecycleExpiringWindowHours < 24 {
		eprLifecycleExpiringWindowHours = 24
	}
	configureEPRRegistrationLifecycleTransitionRunner(
		eprRegistrationService,
		processStatusProvider,
		eventBus,
		logger,
		time.Duration(eprLifecycleTransitionIntervalSeconds)*time.Second,
		time.Duration(eprLifecycleExpiringWindowHours)*time.Hour,
	)
	eprRegistrationHandler := NewEPRRegistrationHandler(eprRegistrationService, logger)
	eprRegistrationHandler.SetEventBus(eventBus)
	eprValidator.SetApprovedRegistrationChecker(eprRegistrationService)
	operationCapabilityChecker := policyadapter.NewOperationCapabilityChecker(systemConfigService)
	imageHandler.SetOperationCapabilityChecker(operationCapabilityChecker)
	imageImportApprovalRequester := workflowapprovaladapter.NewImageImportApprovalRequester(workflowRepo)
	imageImportRepo := postgres.NewImageImportRepository(sqlxDB, logger)
	buildHandler.SetQuarantineArtifactAdmissionChecker(imageImportRepo)
	imageImportService := imageimport.NewService(imageImportRepo, eprValidator, operationCapabilityChecker, imageImportApprovalRequester, logger)
	imageImportHandler := NewImageImportHandler(imageImportService, logger)
	onDemandScanCapabilityChecker := policyadapter.NewOnDemandScanImportCapabilityChecker(operationCapabilityChecker)
	onDemandScanRequestService := imageimport.NewService(imageImportRepo, nil, onDemandScanCapabilityChecker, imageImportApprovalRequester, logger)
	onDemandScanRequestHandler := NewOnDemandScanRequestHandler(onDemandScanRequestService, logger)
	imageImportHandler.SetWorkflowRepository(workflowRepo)
	onDemandScanRequestHandler.SetWorkflowRepository(workflowRepo)
	imageImportHandler.SetNotificationReconciliationRepository(imageImportNotificationRepo)
	onDemandScanRequestHandler.SetNotificationReconciliationRepository(imageImportNotificationRepo)
	imageImportHandler.SetAuditService(auditService)
	imageImportHandler.SetDenialMetrics(denialMetrics)
	imageImportHandler.SetReleaseMetrics(releaseMetrics)
	imageImportHandler.SetSystemConfigService(systemConfigService)
	imageImportHandler.SetReleaseCapabilityChecker(operationCapabilityChecker)
	imageImportHandler.SetEventBus(eventBus)
	imageImportHandler.SetInfrastructureService(infrastructureService)
	onDemandScanRequestHandler.SetAuditService(auditService)
	onDemandScanRequestHandler.SetDenialMetrics(denialMetrics)
	onDemandScanRequestHandler.SetSystemConfigService(systemConfigService)
	sreSmartBotRepo := postgres.NewSRESmartBotRepository(sqlxDB, logger)
	appSignalsRepo := postgres.NewAppSignalsRepository(sqlxDB, logger)
	buildNotificationDeliveryRepo := postgres.NewBuildNotificationDeliveryRepository(sqlxDB, logger)
	sreSmartBotActionService := appsresmartbot.NewActionService(sreSmartBotRepo, infrastructureService, systemConfigService, buildNotificationDeliveryRepo, notificationService, logger)
	sreSmartBotDemoService := appsresmartbot.NewDemoService(appsresmartbot.NewService(sreSmartBotRepo, nil, logger), sreSmartBotRepo, logger)
	sreSmartBotRemediationPackService := appsresmartbot.NewRemediationPackService(systemConfigService)
	sreSmartBotDetectorSuggestionService := appsresmartbot.NewDetectorRuleSuggestionService(sreSmartBotRepo, systemConfigService, logger)
	sreSmartBotWorkspaceService := appsresmartbot.NewWorkspaceService(sreSmartBotRepo, systemConfigService)
	sreLogDetectorBaseURL := strings.TrimSpace(os.Getenv("IF_SRE_LOG_DETECTOR_LOKI_BASE_URL"))
	sreLogDetectorTimeout := time.Duration(getIntEnv("IF_SRE_LOG_DETECTOR_TIMEOUT_SECONDS", 15)) * time.Second
	sreSmartBotLokiClient := logdetector.NewLokiClient(sreLogDetectorBaseURL, &http.Client{Timeout: sreLogDetectorTimeout})
	var consumerLagProvider interface {
		ConsumerLagSnapshots(ctx context.Context) ([]messaging.NATSConsumerLagSnapshot, error)
	}
	if provider, ok := eventBus.(interface {
		ConsumerLagSnapshots(ctx context.Context) ([]messaging.NATSConsumerLagSnapshot, error)
	}); ok {
		consumerLagProvider = provider
	}
	sreSmartBotMCPService := appsresmartbot.NewMCPService(sreSmartBotRepo, systemConfigService, processStatusProvider, appSignalsRepo, releaseComplianceMetrics, sreSmartBotLokiClient, consumerLagProvider, sqlxDB)
	sreSmartBotAgentService := appsresmartbot.NewAgentService(sreSmartBotWorkspaceService, sreSmartBotMCPService)
	sreSmartBotProbeService := appsresmartbot.NewAgentRuntimeProbeService()
	sreSmartBotInterpretationService := appsresmartbot.NewInterpretationService(sreSmartBotAgentService, sreSmartBotWorkspaceService)
	sreSmartBotHandler := NewSRESmartBotHandler(sreSmartBotRepo, sreSmartBotActionService, sreSmartBotDemoService, sreSmartBotRemediationPackService, sreSmartBotDetectorSuggestionService, sreSmartBotWorkspaceService, sreSmartBotMCPService, sreSmartBotAgentService, sreSmartBotProbeService, sreSmartBotInterpretationService, logger)
	configureImageCatalogSubscriber(
		sqlxDB,
		systemConfigService,
		buildRepo,
		buildExecutionService,
		imageRepo,
		versionRepo,
		wsHub,
		processStatusProvider,
		eventBus,
		logger,
	)

	// Initialize worker repositories and services
	workerRepo := persistencePg.NewWorkerRepository(sqlxDB, logger)
	workerPoolService := worker.NewPoolService(workerRepo, logger)
	workerHandler := NewWorkerHandler(workerPoolService, logger)

	buildHistoryRepo := persistencePg.NewBuildHistoryRepository(sqlxDB, logger)
	etaService := build.NewETAService(buildHistoryRepo, logger)
	analyticsHandler := NewAnalyticsHandler(etaService, buildHistoryRepo, logger)
	configHandler := NewConfigHandler(buildRepo, sqlxDB, logger)
	configHandler.SetPackerTargetProfileLookup(packerTargetProfileService)

	ssoWiring := initializeSSOWiring(bootstrapService, systemConfigService, auditService, logger)
	ssoHandler := ssoWiring.ssoHandler
	bootstrapHandler := ssoWiring.bootstrapHandler

	repoAuthWiring := initializeRepositoryAuthWiring(
		sqlxDB,
		logger,
		projectService,
		buildService,
		buildHandler,
		projectBuildSettingsRepo,
		projectSourceRepo,
	)
	repoAuthService := repoAuthWiring.repoAuthService
	registryAuthService := repoAuthWiring.registryAuthService
	gitProviderHandler := repoAuthWiring.gitProviderHandler
	repoBranchHandler := repoAuthWiring.repoBranchHandler
	registryAuthHandler := repoAuthWiring.registryAuthHandler
	imageImportHandler.SetRegistryAuthService(registryAuthService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(userService, rbacRepo, auditService, logger, systemConfigService)
	authMiddleware.SetBootstrapService(bootstrapService)
	tenantStatusMiddleware := middleware.NewTenantStatusMiddleware(sqlxDB, logger)

	// Register routes using organized route functions
	setupTenantRoutes(router, tenantHandler, userHandler, authMiddleware, tenantStatusMiddleware, permissionService)
	setupUserRoutes(router, userHandler, authMiddleware, tenantStatusMiddleware, permissionService)
	registerCoreAPIRoutes(
		router,
		userService,
		rbacRepo,
		auditService,
		logger,
		permissionService,
		authMiddleware,
		tenantStatusMiddleware,
		tenantHandler,
		roleHandler,
		groupHandler,
		workerHandler,
		analyticsHandler,
		configHandler,
		registryAuthHandler,
		projectHandler,
		buildHandler,
		buildService,
		wsHub,
		buildTriggerHandler,
		buildWebhookReceiverHandler,
		projectNotificationTriggerHandler,
		notificationCenterHandler,
		tenantDashboardHandler,
		vmImageHandler,
		buildNotificationReplayHandler,
		infrastructureHandler,
		infrastructureProviderHandler,
		packerTargetProfileHandler,
		authHandler,
		passwordResetHandler,
		bootstrapHandler,
		onDemandScanRequestHandler,
	)

	registerIdentitySystemAdminRoutes(
		router,
		sqlxDB,
		db,
		buildPolicyService,
		auditService,
		logger,
		userService,
		rbacRepo,
		permissionService,
		authMiddleware,
		tenantStatusMiddleware,
		userHandler,
		userInvitationHandler,
		roleHandler,
		permissionHandler,
		systemConfigHandler,
		systemStatsHandler,
		systemComponentsStatusHandler,
		dispatcherMetricsHandler,
		dispatcherControlHandler,
		orchestratorControlHandler,
		executionPipelineHealthHandler,
		executionPipelineMetricsHandler,
		workflowHandler,
		auditHandler,
		onboardingHandler,
		sreSmartBotHandler,
	)

	registerSecurityCatalogRepoRoutes(
		router,
		logger,
		systemConfigRepo,
		emailDomainService,
		auditService,
		permissionService,
		authMiddleware,
		tenantStatusMiddleware,
		ssoHandler,
		groupHandler,
		imageHandler,
		imageImportHandler,
		onDemandScanRequestHandler,
		eprRegistrationHandler,
		repoAuthService,
		projectService,
		gitProviderHandler,
		repoBranchHandler,
	)

	logger.Info("Routes setup completed")
}

// SetupRoutesWithNotification is a wrapper around SetupRoutes for backward compatibility
// It accepts a notification service parameter but doesn't use it yet
func SetupRoutesWithNotification(
	router *Router,
	tenantRepo tenant.Repository,
	tenantService *tenant.Service,
	buildRepo build.Repository,
	buildService *build.Service,
	buildExecutionService build.BuildExecutionService,
	projectRepo project.Repository,
	projectService *project.Service,
	userRepo user.Repository,
	userService *user.Service,
	rbacRepo rbac.Repository,
	rbacService *rbac.Service,
	systemConfigRepo systemconfig.Repository,
	logger *zap.Logger,
	ldapService *user.LDAPService,
	auditService *audit.Service,
	notificationTemplateRepo notification.Repository,
	wsHub *WebSocketHub,
	db interface{},
	notificationService *email.NotificationService,
	cfg *config.Config,
	buildPolicyService *build.BuildPolicyService,
	eventBus messaging.EventBus,
) {
	// TODO: Pass notification service to tenant handler when integrated
	SetupRoutes(router, tenantRepo, tenantService, buildRepo, buildService, buildExecutionService, projectRepo, projectService, userRepo, userService, rbacRepo, rbacService, systemConfigRepo, logger, ldapService, auditService, notificationTemplateRepo, wsHub, db, cfg, buildPolicyService, nil, nil, nil, nil, nil, cfg.Dispatcher.Enabled, nil, nil, eventBus)
}

// encodeJSON encodes data as JSON and writes it to the response writer
func encodeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}
