package rest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	appbootstrap "github.com/srikarm/image-factory/internal/application/bootstrap"
	imagecatalog "github.com/srikarm/image-factory/internal/application/imagecatalog"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/domain/build"
	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"github.com/srikarm/image-factory/internal/domain/gitprovider"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/notification"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
	"github.com/srikarm/image-factory/internal/domain/repositorybranch"
	"github.com/srikarm/image-factory/internal/domain/sso"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type platformWiring struct {
	systemConfigService        *systemconfig.Service
	bootstrapService           *appbootstrap.Service
	emailDomainService         *domainEmail.Service
	notificationService        *email.NotificationService
	infrastructureService      *infrastructure.Service
	infrastructureProviderHand *InfrastructureProviderHandler
}

func initializePlatformWiring(
	sqlxDB *sqlx.DB,
	cfg *config.Config,
	systemConfigRepo systemconfig.Repository,
	notificationTemplateRepo notification.Repository,
	eventBus messaging.EventBus,
	infrastructureEventPublisher infrastructure.EventPublisher,
	logger *zap.Logger,
) platformWiring {
	systemConfigService := systemconfig.NewService(systemConfigRepo, logger)
	bootstrapService := appbootstrap.NewService(sqlxDB, logger)
	notificationDomainService := notification.NewService(notificationTemplateRepo, logger)
	emailRepo := postgres.NewEmailRepository(sqlxDB, logger)
	emailDomainService := domainEmail.NewServiceWithNotification(
		emailRepo,
		logger,
		os.Getenv("IF_SMTP_HOST"),
		getIntEnv("IF_SMTP_PORT", 1025),
		os.Getenv("IF_SMTP_FROM_EMAIL"),
		notificationDomainService,
	)
	notificationService := email.NewNotificationService(
		emailDomainService,
		eventBus,
		cfg.Messaging.EnableNATS,
		logger,
		os.Getenv("IF_SMTP_FROM_EMAIL"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		systemConfigService,
	)
	infrastructureRepo := postgres.NewInfrastructureRepository(sqlxDB, logger)
	infrastructureService := infrastructure.NewService(infrastructureRepo, infrastructureEventPublisher, logger)
	infrastructureProviderHandler := NewInfrastructureProviderHandler(infrastructureService, logger)

	return platformWiring{
		systemConfigService:        systemConfigService,
		bootstrapService:           bootstrapService,
		emailDomainService:         emailDomainService,
		notificationService:        notificationService,
		infrastructureService:      infrastructureService,
		infrastructureProviderHand: infrastructureProviderHandler,
	}
}

type ssoWiring struct {
	ssoHandler       *SSOHandler
	bootstrapHandler *BootstrapHandler
}

func initializeSSOWiring(
	bootstrapService *appbootstrap.Service,
	systemConfigService *systemconfig.Service,
	auditService *audit.Service,
	logger *zap.Logger,
) ssoWiring {
	defaultOIDCID := uuid.New()
	samlRepo := &mockSAMLRepository{
		providers: make(map[uuid.UUID]*sso.SAMLProvider),
		logger:    logger,
	}
	oidcRepo := &mockOIDCRepository{
		providers: map[uuid.UUID]*sso.OpenIDConnectProvider{
			defaultOIDCID: {
				ID:               defaultOIDCID,
				Name:             "PingFed",
				Issuer:           "https://pingfed.local",
				ClientID:         "image-factory-client",
				ClientSecret:     "",
				AuthorizationURL: "https://pingfed.local/as/authorization.oauth2",
				TokenURL:         "https://pingfed.local/as/token.oauth2",
				UserInfoURL:      "https://pingfed.local/idp/userinfo.openid",
				JWKSURL:          "https://pingfed.local/pf/JWKS",
				RedirectURIs:     []string{"http://localhost:3000/auth/callback"},
				Scopes:           []string{"openid", "profile", "email"},
				ResponseTypes:    []string{"code"},
				GrantTypes:       []string{"authorization_code"},
				Attributes:       map[string]interface{}{"provider": "pingfed"},
				Enabled:          true,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
			},
		},
		logger: logger,
	}
	ssoService := sso.NewService(samlRepo, oidcRepo, logger)
	return ssoWiring{
		ssoHandler:       NewSSOHandler(ssoService, auditService, logger),
		bootstrapHandler: NewBootstrapHandler(bootstrapService, systemConfigService, ssoService, logger),
	}
}

type repoAuthWiring struct {
	repoAuthService     *repositoryauth.Service
	registryAuthService *registryauth.Service
	gitProviderHandler  *GitProviderHandler
	repoBranchHandler   *RepositoryBranchHandler
	registryAuthHandler *RegistryAuthHandler
}

func initializeRepositoryAuthWiring(
	sqlxDB *sqlx.DB,
	logger *zap.Logger,
	projectService *project.Service,
	buildService *build.Service,
	buildHandler *BuildHandler,
	projectBuildSettingsRepo *postgres.ProjectBuildSettingsRepository,
	projectSourceRepo *postgres.ProjectSourceRepository,
) repoAuthWiring {
	repoAuthRepo := postgres.NewRepositoryAuthRepository(sqlxDB, logger)
	encryptor, err := crypto.NewAESGCMEncryptorFromEnv()
	if err != nil {
		logger.Fatal("Failed to create AES-GCM encryptor", zap.Error(err))
	}
	repoAuthService := repositoryauth.NewService(repoAuthRepo, encryptor)
	registryAuthRepo := postgres.NewRegistryAuthRepository(sqlxDB, logger)
	registryAuthService := registryauth.NewService(registryAuthRepo, encryptor)

	configureBuildAuthResolvers(buildService, buildHandler, repoAuthService, registryAuthService, projectBuildSettingsRepo, projectSourceRepo)

	gitProviderRepo := postgres.NewGitProviderRepository(sqlxDB, logger)
	gitProviderService := gitprovider.NewService(gitProviderRepo)
	gitProviderHandler := NewGitProviderHandler(gitProviderService, logger)
	repoBranchService := repositorybranch.NewService(
		repoAuthService,
		gitProviderService,
		http.DefaultClient,
		repositorybranch.NewExecGitRunner(),
		logger,
	)
	repoBranchHandler := NewRepositoryBranchHandler(repoBranchService, projectService, logger)
	registryAuthHandler := NewRegistryAuthHandler(registryAuthService, projectService, logger)

	return repoAuthWiring{
		repoAuthService:     repoAuthService,
		registryAuthService: registryAuthService,
		gitProviderHandler:  gitProviderHandler,
		repoBranchHandler:   repoBranchHandler,
		registryAuthHandler: registryAuthHandler,
	}
}

func configureBuildAuthResolvers(
	buildService *build.Service,
	buildHandler *BuildHandler,
	repoAuthService *repositoryauth.Service,
	registryAuthService *registryauth.Service,
	projectBuildSettingsRepo *postgres.ProjectBuildSettingsRepository,
	projectSourceRepo *postgres.ProjectSourceRepository,
) {
	if buildService != nil {
		buildService.SetRegistryAuthResolver(registryAuthService)
		buildService.SetProjectBuildSettingsLookup(func(ctx context.Context, projectID uuid.UUID) (*build.ProjectBuildSettings, error) {
			settings, err := projectBuildSettingsRepo.GetByProjectID(ctx, projectID)
			if err != nil {
				return nil, err
			}
			if settings == nil {
				return nil, nil
			}
			return &build.ProjectBuildSettings{
				BuildConfigMode:    settings.BuildConfigMode,
				BuildConfigFile:    settings.BuildConfigFile,
				BuildConfigOnError: settings.BuildConfigOnError,
			}, nil
		})
		buildService.SetProjectGitAuthLookup(func(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error) {
			return repoAuthService.ResolveGitAuthSecretData(ctx, projectID)
		})
		buildService.SetProjectSourceGitAuthLookup(func(ctx context.Context, projectID, sourceID uuid.UUID) (map[string][]byte, error) {
			source, err := projectSourceRepo.FindByID(ctx, projectID, sourceID)
			if err != nil || source == nil || !source.IsActive || source.RepositoryAuth == nil {
				return nil, err
			}
			auth, err := repoAuthService.GetRepositoryAuth(ctx, *source.RepositoryAuth)
			if err != nil {
				return nil, err
			}
			if auth == nil || !auth.GetIsActive() {
				return nil, nil
			}
			if auth.GetTenantID() != source.TenantID {
				return nil, fmt.Errorf("source repository auth belongs to a different tenant")
			}
			if scopedProjectID := auth.GetProjectID(); scopedProjectID != nil && *scopedProjectID != projectID {
				return nil, fmt.Errorf("source repository auth belongs to a different project")
			}
			return repoAuthService.ResolveGitAuthSecretDataByAuthID(ctx, auth.GetID())
		})
	}
	if buildHandler != nil {
		buildHandler.SetProjectRepoAuthLookup(func(ctx context.Context, projectID uuid.UUID) (bool, error) {
			auth, err := repoAuthService.GetActiveRepositoryAuth(ctx, projectID)
			if err != nil {
				return false, err
			}
			return auth != nil && auth.GetIsActive(), nil
		})
		buildHandler.SetProjectGitAuthLookup(func(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error) {
			return repoAuthService.ResolveGitAuthSecretData(ctx, projectID)
		})
		buildHandler.SetPublicRepoProbe(defaultPublicRepoProbe)
	}
}

func configureImageCatalogSubscriber(
	sqlxDB *sqlx.DB,
	systemConfigService *systemconfig.Service,
	buildRepo build.Repository,
	buildExecutionService build.BuildExecutionService,
	imageRepo *postgres.ImageRepository,
	versionRepo *postgres.ImageVersionRepository,
	wsHub *WebSocketHub,
	processStatusProvider runtimehealth.Provider,
	eventBus messaging.EventBus,
	logger *zap.Logger,
) {
	processStatusStore, hasProcessStatusStore := processStatusProvider.(*runtimehealth.Store)
	if hasProcessStatusStore {
		processStatusStore.Upsert("image_catalog_event_subscriber", runtimehealth.ProcessStatus{
			Enabled:      eventBus != nil,
			Running:      eventBus != nil,
			LastActivity: time.Now().UTC(),
			Message:      "image catalog event subscriber pending initialization",
			Metrics:      imageCatalogZeroMetrics(),
		})
	}

	if eventBus != nil {
		imageCatalogStrictExecutionID := resolveImageCatalogStrictExecutionID(context.Background(), systemConfigService, logger)
		imageCatalogSubscriber := imagecatalog.NewEventSubscriber(
			buildRepo,
			buildExecutionService,
			imageRepo,
			versionRepo,
			logger,
		)
		if imageCatalogStrictExecutionID {
			imageCatalogSubscriber.SetStrictExecutionResolution(true)
			logger.Info("Image catalog subscriber strict execution-id resolution enabled")
		} else {
			logger.Warn("Image catalog subscriber strict execution-id resolution disabled")
		}
		imageCatalogSubscriber.SetAlertNotifier(&imageCatalogAlertNotifier{
			deliveryRepo: postgres.NewBuildNotificationDeliveryRepository(sqlxDB, logger),
			wsHub:        wsHub,
			logger:       logger,
		})
		imageCatalogSubscriber.SetBuildEvidenceRepository(postgres.NewBuildEvidenceRepository(sqlxDB, logger))
		imagecatalog.RegisterEventSubscriber(eventBus, imageCatalogSubscriber)
		if hasProcessStatusStore {
			processStatusStore.Upsert("image_catalog_event_subscriber", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: time.Now().UTC(),
				Message:      "image catalog event subscriber started",
				Metrics:      imageCatalogZeroMetrics(),
			})
			go func() {
				ticker := time.NewTicker(15 * time.Second)
				defer ticker.Stop()
				for {
					snapshot := imageCatalogSubscriber.Snapshot()
					processStatusStore.Upsert("image_catalog_event_subscriber", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      true,
						LastActivity: time.Now().UTC(),
						Message: fmt.Sprintf(
							"events=%d missing_execution_id=%d explicit_lookup_failures=%d missing_evidence=%d alerts=%d",
							snapshot.EventsReceived,
							snapshot.MissingExecutionID,
							snapshot.ExplicitExecutionLookupFailures,
							snapshot.IngestsMissingEvidence,
							snapshot.AlertsEmitted,
						),
						Metrics: map[string]int64{
							"image_catalog_events_received_total":          snapshot.EventsReceived,
							"image_catalog_missing_execution_id_total":     snapshot.MissingExecutionID,
							"image_catalog_explicit_lookup_failures_total": snapshot.ExplicitExecutionLookupFailures,
							"image_catalog_fallback_attempts_total":        snapshot.FallbackExecutionLookupAttempts,
							"image_catalog_fallback_success_total":         snapshot.FallbackExecutionLookupSuccess,
							"image_catalog_fallback_skipped_total":         snapshot.FallbackExecutionLookupSkipped,
							"image_catalog_missing_evidence_total":         snapshot.IngestsMissingEvidence,
							"image_catalog_alerts_emitted_total":           snapshot.AlertsEmitted,
							"image_catalog_alert_delivery_failures_total":  snapshot.AlertDeliveryFailures,
						},
					})
					<-ticker.C
				}
			}()
		}
		return
	}

	if hasProcessStatusStore {
		processStatusStore.Upsert("image_catalog_event_subscriber", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "image catalog event subscriber disabled",
			Metrics:      imageCatalogZeroMetrics(),
		})
	}
}

func imageCatalogZeroMetrics() map[string]int64 {
	return map[string]int64{
		"image_catalog_events_received_total":          0,
		"image_catalog_missing_execution_id_total":     0,
		"image_catalog_explicit_lookup_failures_total": 0,
		"image_catalog_fallback_attempts_total":        0,
		"image_catalog_fallback_success_total":         0,
		"image_catalog_fallback_skipped_total":         0,
		"image_catalog_missing_evidence_total":         0,
		"image_catalog_alerts_emitted_total":           0,
		"image_catalog_alert_delivery_failures_total":  0,
	}
}

func configureEPRRegistrationLifecycleTransitionRunner(
	service *eprregistration.Service,
	processStatusProvider runtimehealth.Provider,
	eventBus messaging.EventBus,
	logger *zap.Logger,
	interval time.Duration,
	expiringWindow time.Duration,
) {
	if interval <= 0 {
		interval = time.Hour
	}
	if expiringWindow <= 0 {
		expiringWindow = 30 * 24 * time.Hour
	}
	processStatusStore, hasProcessStatusStore := processStatusProvider.(*runtimehealth.Store)
	if hasProcessStatusStore {
		processStatusStore.Upsert("epr_lifecycle_transition_runner", runtimehealth.ProcessStatus{
			Enabled:      service != nil,
			Running:      service != nil,
			LastActivity: time.Now().UTC(),
			Message:      "epr lifecycle transition runner initialized",
			Metrics: map[string]int64{
				"epr_lifecycle_transition_runs_total":       0,
				"epr_lifecycle_transition_failures_total":   0,
				"epr_lifecycle_transition_expiring_total":   0,
				"epr_lifecycle_transition_expired_total":    0,
				"epr_lifecycle_transition_window_hours":     int64(expiringWindow / time.Hour),
				"epr_lifecycle_transition_interval_seconds": int64(interval / time.Second),
			},
		})
	}

	if service == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var runsTotal int64
		var failuresTotal int64
		var expiringTotal int64
		var expiredTotal int64

		run := func() {
			now := time.Now().UTC()
			runsTotal++
			result, err := service.RunLifecycleTransitions(context.Background(), now, expiringWindow)
			if err != nil {
				failuresTotal++
				if logger != nil {
					logger.Warn("EPR lifecycle transition run failed", zap.Error(err))
				}
				if hasProcessStatusStore {
					processStatusStore.Upsert("epr_lifecycle_transition_runner", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      true,
						LastActivity: now,
						Message:      fmt.Sprintf("transition run failed: %v", err),
						Metrics: map[string]int64{
							"epr_lifecycle_transition_runs_total":       runsTotal,
							"epr_lifecycle_transition_failures_total":   failuresTotal,
							"epr_lifecycle_transition_expiring_total":   expiringTotal,
							"epr_lifecycle_transition_expired_total":    expiredTotal,
							"epr_lifecycle_transition_window_hours":     int64(expiringWindow / time.Hour),
							"epr_lifecycle_transition_interval_seconds": int64(interval / time.Second),
						},
					})
				}
				return
			}
			if result != nil {
				expiringTotal += int64(result.ExpiringCount)
				expiredTotal += int64(result.ExpiredCount)
				if eventBus != nil {
					for _, row := range result.ExpiringRecords {
						eventBus.Publish(context.Background(), messaging.Event{
							Type:       messaging.EventTypeEPRLifecycleExpiring,
							TenantID:   row.TenantID.String(),
							Source:     "system.epr_lifecycle_transition_runner",
							OccurredAt: now,
							Payload: map[string]interface{}{
								"epr_registration_request_id": row.RequestID.String(),
								"tenant_id":                   row.TenantID.String(),
								"requested_by_user_id":        row.RequestedByUserID.String(),
								"epr_record_id":               row.EPRRecordID,
								"lifecycle_status":            string(row.LifecycleStatus),
								"idempotency_key":             row.RequestID.String() + ":" + messaging.EventTypeEPRLifecycleExpiring + ":" + now.Format("20060102"),
							},
						})
					}
					for _, row := range result.ExpiredRecords {
						eventBus.Publish(context.Background(), messaging.Event{
							Type:       messaging.EventTypeEPRLifecycleExpired,
							TenantID:   row.TenantID.String(),
							Source:     "system.epr_lifecycle_transition_runner",
							OccurredAt: now,
							Payload: map[string]interface{}{
								"epr_registration_request_id": row.RequestID.String(),
								"tenant_id":                   row.TenantID.String(),
								"requested_by_user_id":        row.RequestedByUserID.String(),
								"epr_record_id":               row.EPRRecordID,
								"lifecycle_status":            string(row.LifecycleStatus),
								"idempotency_key":             row.RequestID.String() + ":" + messaging.EventTypeEPRLifecycleExpired + ":" + now.Format("20060102"),
							},
						})
					}
				}
			}
			if hasProcessStatusStore {
				expiring := int64(0)
				expired := int64(0)
				if result != nil {
					expiring = int64(result.ExpiringCount)
					expired = int64(result.ExpiredCount)
				}
				processStatusStore.Upsert("epr_lifecycle_transition_runner", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: now,
					Message:      fmt.Sprintf("transition run completed expiring=%d expired=%d", expiring, expired),
					Metrics: map[string]int64{
						"epr_lifecycle_transition_runs_total":       runsTotal,
						"epr_lifecycle_transition_failures_total":   failuresTotal,
						"epr_lifecycle_transition_expiring_total":   expiringTotal,
						"epr_lifecycle_transition_expired_total":    expiredTotal,
						"epr_lifecycle_transition_last_expiring":    expiring,
						"epr_lifecycle_transition_last_expired":     expired,
						"epr_lifecycle_transition_window_hours":     int64(expiringWindow / time.Hour),
						"epr_lifecycle_transition_interval_seconds": int64(interval / time.Second),
					},
				})
			}
		}

		run()
		for {
			<-ticker.C
			run()
		}
	}()
}
