package bootstrap

import (
	"context"
	"os"
	"time"

	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	appinfrastructure "github.com/srikarm/image-factory/internal/application/infrastructure"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/domain/build"
	domaininfrastructure "github.com/srikarm/image-factory/internal/domain/infrastructure"
	"go.uber.org/zap"
)

type DispatcherRunnerConfig struct {
	Enabled            bool
	PollInterval       time.Duration
	MaxDispatchPerTick int
	MaxRetries         int
	RetryBackoff       time.Duration
	RetryBackoffMax    time.Duration
}

type DispatcherRunnerDeps struct {
	ProcessHealthStore  *runtimehealth.Store
	BuildRepo           build.Repository
	BuildService        appdispatcher.BuildDispatchService
	SystemConfigService build.SystemConfigService
	DispatcherRuntime   appdispatcher.RuntimeStatusStore
	Infrastructure      *domaininfrastructure.Service
	Logger              *zap.Logger
}

func StartDispatcherRunner(deps DispatcherRunnerDeps, cfg DispatcherRunnerConfig) (*appdispatcher.Controller, *appdispatcher.QueuedBuildDispatcher) {
	if deps.ProcessHealthStore == nil {
		return nil, nil
	}
	if !cfg.Enabled {
		deps.ProcessHealthStore.Upsert("dispatcher", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "dispatcher disabled",
		})
		if deps.Logger != nil {
			deps.Logger.Warn("Dispatcher is disabled; queued builds will not execute unless an external dispatcher is running",
				zap.Bool("dispatcher_enabled", cfg.Enabled),
				zap.Duration("poll_interval", cfg.PollInterval),
			)
		}
		return nil, nil
	}

	deps.ProcessHealthStore.Upsert("dispatcher", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      false,
		LastActivity: time.Now().UTC(),
		Message:      "dispatcher is starting",
	})
	if deps.Logger != nil {
		deps.Logger.Info("Background process starting",
			zap.String("component", "queued_build_dispatcher"),
			zap.Duration("poll_interval", cfg.PollInterval),
			zap.Int("max_dispatch_per_tick", cfg.MaxDispatchPerTick),
			zap.Int("max_retries", cfg.MaxRetries),
		)
	}

	dispatcherConfig := appdispatcher.QueueDispatcherConfig{
		PollInterval:       cfg.PollInterval,
		MaxDispatchPerTick: cfg.MaxDispatchPerTick,
		MaxRetries:         cfg.MaxRetries,
		RetryBackoff:       cfg.RetryBackoff,
		RetryBackoffMax:    cfg.RetryBackoffMax,
	}
	queuedDispatcher := appdispatcher.NewQueuedBuildDispatcher(
		deps.BuildRepo,
		deps.BuildService,
		deps.SystemConfigService,
		deps.Logger,
		dispatcherConfig,
	)
	controller := appdispatcher.NewController(queuedDispatcher)
	controller.Start(context.Background())

	deps.ProcessHealthStore.Upsert("dispatcher", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      controller.Status(),
		LastActivity: time.Now().UTC(),
		Message:      "embedded dispatcher running",
	})

	instanceID := "embedded"
	if hostname, hostErr := os.Hostname(); hostErr == nil && hostname != "" {
		instanceID = "embedded-" + hostname
	}
	if deps.Logger != nil {
		deps.Logger.Info("Background process starting",
			zap.String("component", "dispatcher_runtime_heartbeat"),
			zap.Duration("interval", 10*time.Second),
			zap.String("instance_id", instanceID),
		)
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			snapshot := appdispatcher.RuntimeStatus{
				InstanceID:    instanceID,
				Mode:          appdispatcher.DispatcherModeEmbedded,
				Running:       controller.Status(),
				LastHeartbeat: time.Now().UTC(),
				Metrics:       queuedDispatcher.DispatcherMetrics(),
			}
			if deps.DispatcherRuntime != nil {
				if err := deps.DispatcherRuntime.UpsertRuntimeStatus(context.Background(), snapshot); err != nil && deps.Logger != nil {
					deps.Logger.Warn("Failed to persist embedded dispatcher runtime status", zap.Error(err))
				}
			}
			deps.ProcessHealthStore.Upsert("dispatcher", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      snapshot.Running,
				LastActivity: snapshot.LastHeartbeat,
				Message:      "embedded dispatcher heartbeat",
			})
			<-ticker.C
		}
	}()

	if deps.Infrastructure != nil {
		tektonInstallerDispatcher := appinfrastructure.NewTektonInstallerDispatcher(
			deps.Infrastructure,
			deps.Logger,
			appinfrastructure.TektonInstallerDispatcherConfig{
				PollInterval:   cfg.PollInterval,
				MaxJobsPerTick: cfg.MaxDispatchPerTick,
			},
		)
		if deps.Logger != nil {
			deps.Logger.Info("Background process starting",
				zap.String("component", "tekton_installer_dispatcher"),
				zap.Duration("poll_interval", cfg.PollInterval),
				zap.Int("max_jobs_per_tick", cfg.MaxDispatchPerTick),
			)
		}
		go tektonInstallerDispatcher.Run(context.Background())
	}

	return controller, queuedDispatcher
}
