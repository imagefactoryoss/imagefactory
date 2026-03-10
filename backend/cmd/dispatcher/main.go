package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	appinfrastructure "github.com/srikarm/image-factory/internal/application/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
	"github.com/srikarm/image-factory/internal/infrastructure/database"
	k8sinfra "github.com/srikarm/image-factory/internal/infrastructure/kubernetes"
	"github.com/srikarm/image-factory/internal/infrastructure/logger"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/subosito/gotenv"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sync/atomic"
)

type dispatcherHealthState struct {
	instanceID string
	startedAt  time.Time
	running    atomic.Bool
}

func newDispatcherHealthServer(addr string, state *dispatcherHealthState) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respondDispatcherHealth(w, state, false)
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		respondDispatcherHealth(w, state, true)
	})
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func respondDispatcherHealth(w http.ResponseWriter, state *dispatcherHealthState, readiness bool) {
	running := state != nil && state.running.Load()
	status := "healthy"
	if readiness && !running {
		status = "not_ready"
	}

	code := http.StatusOK
	if readiness && !running {
		code = http.StatusServiceUnavailable
	}

	uptimeSeconds := int64(0)
	instanceID := ""
	if state != nil && !state.startedAt.IsZero() {
		uptimeSeconds = int64(time.Since(state.startedAt).Seconds())
		if uptimeSeconds < 0 {
			uptimeSeconds = 0
		}
		instanceID = state.instanceID
	}

	response := map[string]interface{}{
		"status":         status,
		"service":        "dispatcher",
		"mode":           appdispatcher.DispatcherModeExternal,
		"instance_id":    instanceID,
		"running":        running,
		"uptime_seconds": uptimeSeconds,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(response)
}

func envOrInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func shouldAttemptInClusterOrKubeconfig(kubeconfigPath string) bool {
	if kubeconfigPath != "" {
		return true
	}
	// Avoid noisy warnings when running locally; only attempt in-cluster config when
	// Kubernetes service env vars are present.
	return os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != ""
}

func buildKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}

func main() {
	var envFile = flag.String("env", "", "Path to environment file (e.g., .env.development, .env.production)")
	var runOnce = flag.Bool("once", false, "Run one dispatcher cycle and exit")
	var healthPort = flag.Int("health-port", envOrInt("IF_DISPATCHER_PORT", 8084), "Port for dispatcher health endpoints")
	flag.Parse()

	if *envFile != "" {
		if !filepath.IsAbs(*envFile) {
			absPath, err := filepath.Abs(*envFile)
			if err == nil {
				*envFile = absPath
			}
		}
		if err := gotenv.Load(*envFile); err != nil {
			fmt.Printf("Failed to load env file: %v\n", err)
		}
	} else {
		_ = gotenv.Load()
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	if readiness, readinessErr := database.CheckSchemaReadiness(context.Background(), db); readinessErr != nil {
		log.Warn("Database schema readiness check failed", zap.Error(readinessErr))
	} else {
		if readiness.CurrentSchema != cfg.Database.Schema {
			log.Warn("Database schema mismatch detected",
				zap.String("configured_schema", cfg.Database.Schema),
				zap.String("current_schema", readiness.CurrentSchema),
			)
		}
		log.Info("Database schema readiness check completed",
			zap.String("current_schema", readiness.CurrentSchema),
			zap.Bool("schema_migrations_present", readiness.SchemaMigrations),
			zap.Bool("tenants_domain_column", readiness.TenantsDomainColumn),
			zap.Bool("tenants_domain_not_null", readiness.TenantsDomainNotNull),
		)
	}
	runtimeStore := postgres.NewDispatcherRuntimeRepository(db, log)
	instanceID := "external"
	if hostname, hostErr := os.Hostname(); hostErr == nil && hostname != "" {
		instanceID = "external-" + hostname + "-" + strconv.Itoa(os.Getpid())
	}

	systemConfigRepo := postgres.NewSystemConfigRepository(db, log)
	systemConfigService := systemconfig.NewService(systemConfigRepo, log)

	messagingEnableNATS := cfg.Messaging.EnableNATS
	eventSource := cfg.Server.Environment
	baseBus := messaging.NewInProcessBus(log)
	var bus messaging.EventBus = baseBus
	if messagingEnableNATS {
		natsBus, err := messaging.NewNATSBus(cfg.NATS, log)
		if err != nil {
			log.Fatal("Failed to initialize NATS event bus", zap.Error(err))
		}
		hybridBus := messaging.NewHybridBus(baseBus, natsBus, eventSource, log)
		bus = hybridBus
		defer hybridBus.Close()
	}

	eventBus := messaging.NewValidatingBus(bus, messaging.ValidationConfig{
		SchemaVersion:  cfg.Messaging.SchemaVersion,
		ValidateEvents: cfg.Messaging.ValidateEvents,
	}, log)
	buildEventPublisher := messaging.NewBuildEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	buildStatusBroadcaster := messaging.NewBuildStatusBroadcaster(eventBus, eventSource, cfg.Messaging.SchemaVersion)

	buildRepo := postgres.NewBuildRepository(db, log)
	triggerRepo := postgres.NewTriggerRepository(db, log)
	infrastructureRepo := postgres.NewInfrastructureRepository(db, log)
	buildExecutionRepo := postgres.NewBuildExecutionRepository(db, log)
	buildExecutionService := build.NewBuildExecutionServiceWithWebSocket(buildExecutionRepo, buildStatusBroadcaster)

	containerExecutor := build.NewNoOpBuildExecutor(log)
	packerExecutor := build.NewPackerBuildExecutor(log, "/tmp/packer-work", "image-factory-builds", "us-east-1", "")
	localExecutorFactory := build.NewBuildMethodExecutorFactory(buildExecutionService)
	var repositoryAuthService *repositoryauth.Service
	var registryAuthService *registryauth.Service
	if encryptor, encErr := crypto.NewAESGCMEncryptorFromEnv(); encErr != nil {
		log.Warn("Credential encryptor unavailable; docker-config/git-auth reconciliation in Tekton executor will be disabled", zap.Error(encErr))
	} else {
		repositoryAuthRepo := postgres.NewRepositoryAuthRepository(db, log)
		repositoryAuthService = repositoryauth.NewService(repositoryAuthRepo, encryptor)
		registryAuthRepo := postgres.NewRegistryAuthRepository(db, log)
		registryAuthService = registryauth.NewService(registryAuthRepo, encryptor)
	}
	var tektonExecutorFactory build.BuildMethodExecutorFactory
	infrastructureService := infrastructure.NewService(infrastructureRepo, messaging.NewInfrastructureEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion), log)
	tektonClientProvider := k8sinfra.NewTektonClientProvider(infrastructureService, log)
	if cfg.Build.TektonEnabled {
		templateEngine := k8sinfra.NewGoTemplateEngine()
		buildMethodConfigRepo := postgres.NewBuildMethodConfigRepository(db, log)
		if shouldAttemptInClusterOrKubeconfig(cfg.Build.TektonKubeconfig) {
			kubeConfig, err := buildKubeConfig(cfg.Build.TektonKubeconfig)
			if err != nil {
				log.Warn("Failed to load Kubernetes config for Tekton", zap.Error(err))
			} else {
				k8sClient, err := kubernetes.NewForConfig(kubeConfig)
				if err != nil {
					log.Warn("Failed to initialize Kubernetes client", zap.Error(err))
				}
				tektonClient, err := tektonclient.NewForConfig(kubeConfig)
				if err != nil {
					log.Warn("Failed to initialize Tekton client", zap.Error(err))
				}
				if k8sClient != nil && tektonClient != nil {
					namespaceMgr := k8sinfra.NewKubernetesNamespaceManager(k8sClient, log)
					pipelineMgr := k8sinfra.NewKubernetesPipelineManager(k8sClient, tektonClient, log)
					tektonExecutorFactory = build.NewTektonExecutorFactory(
						k8sClient,
						tektonClient,
						log,
						namespaceMgr,
						pipelineMgr,
						templateEngine,
						buildExecutionService,
						buildMethodConfigRepo,
						buildRepo,
						tektonClientProvider,
						registryAuthService,
						repositoryAuthService,
					)
				}
			}
		}
		if tektonExecutorFactory == nil && tektonClientProvider != nil {
			tektonExecutorFactory = build.NewTektonExecutorFactory(
				nil,
				nil,
				log,
				nil,
				nil,
				templateEngine,
				buildExecutionService,
				buildMethodConfigRepo,
				buildRepo,
				tektonClientProvider,
				registryAuthService,
				repositoryAuthService,
			)
		}
	}
	buildService := build.NewService(buildRepo, triggerRepo, buildEventPublisher, containerExecutor, packerExecutor, buildExecutionService, localExecutorFactory, tektonExecutorFactory, systemConfigService, nil, log)

	dispatcherConfig := appdispatcher.QueueDispatcherConfig{
		PollInterval:       cfg.Dispatcher.PollInterval,
		MaxDispatchPerTick: cfg.Dispatcher.MaxDispatchPerTick,
		MaxRetries:         cfg.Dispatcher.MaxRetries,
		RetryBackoff:       cfg.Dispatcher.RetryBackoff,
		RetryBackoffMax:    cfg.Dispatcher.RetryBackoffMax,
	}

	queuedDispatcher := appdispatcher.NewQueuedBuildDispatcher(buildRepo, buildService, systemConfigService, log, dispatcherConfig)
	tektonInstallerDispatcher := appinfrastructure.NewTektonInstallerDispatcher(
		infrastructureService,
		log,
		appinfrastructure.TektonInstallerDispatcherConfig{
			PollInterval:   cfg.Dispatcher.PollInterval,
			MaxJobsPerTick: cfg.Dispatcher.MaxDispatchPerTick,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	healthState := &dispatcherHealthState{
		instanceID: instanceID,
		startedAt:  time.Now().UTC(),
	}
	healthState.running.Store(false)

	if *runOnce {
		count, err := queuedDispatcher.RunOnce(ctx)
		if err != nil {
			log.Error("Dispatcher run once failed", zap.Error(err))
			os.Exit(1)
		}
		processedInstallerJobs, installerErr := tektonInstallerDispatcher.RunOnce(ctx)
		if installerErr != nil {
			log.Error("Tekton installer dispatcher run once failed", zap.Error(installerErr))
			os.Exit(1)
		}
		_ = runtimeStore.UpsertRuntimeStatus(ctx, appdispatcher.RuntimeStatus{
			InstanceID:    instanceID,
			Mode:          appdispatcher.DispatcherModeExternal,
			Running:       true,
			LastHeartbeat: time.Now().UTC(),
			Metrics:       queuedDispatcher.DispatcherMetrics(),
		})
		log.Info("Dispatcher run once complete", zap.Int("dispatched", count), zap.Int("tekton_installer_jobs_processed", processedInstallerJobs))
		return
	}

	healthServer := newDispatcherHealthServer(fmt.Sprintf(":%d", *healthPort), healthState)
	go func() {
		log.Info("Starting dispatcher health server",
			zap.String("addr", healthServer.Addr),
			zap.String("health_path", "/health"),
			zap.String("ready_path", "/ready"))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Dispatcher health server failed", zap.Error(err))
		}
	}()

	go queuedDispatcher.Run(ctx)
	go tektonInstallerDispatcher.Run(ctx)
	healthState.running.Store(true)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := runtimeStore.UpsertRuntimeStatus(context.Background(), appdispatcher.RuntimeStatus{
				InstanceID:    instanceID,
				Mode:          appdispatcher.DispatcherModeExternal,
				Running:       true,
				LastHeartbeat: time.Now().UTC(),
				Metrics:       queuedDispatcher.DispatcherMetrics(),
			}); err != nil {
				log.Warn("Failed to persist external dispatcher runtime status", zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Dispatcher shutting down")
	healthState.running.Store(false)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = healthServer.Shutdown(shutdownCtx)
	_ = runtimeStore.UpsertRuntimeStatus(context.Background(), appdispatcher.RuntimeStatus{
		InstanceID:    instanceID,
		Mode:          appdispatcher.DispatcherModeExternal,
		Running:       false,
		LastHeartbeat: time.Now().UTC(),
		Metrics:       queuedDispatcher.DispatcherMetrics(),
	})
}
