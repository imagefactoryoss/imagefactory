package bootstrap

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	quarantineimportadapter "github.com/srikarm/image-factory/internal/adapters/secondary/quarantineimport"
	buildsteps "github.com/srikarm/image-factory/internal/application/build/steps"
	imageimportsteps "github.com/srikarm/image-factory/internal/application/imageimport/steps"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	appworkflow "github.com/srikarm/image-factory/internal/application/workflow"
	domainbuild "github.com/srikarm/image-factory/internal/domain/build"
	domainimageimport "github.com/srikarm/image-factory/internal/domain/imageimport"
	domaininfrastructure "github.com/srikarm/image-factory/internal/domain/infrastructure"
	domainsystemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type WorkflowRunnerConfig struct {
	Enabled         bool
	PollInterval    time.Duration
	MaxStepsPerTick int
	BuildTektonMode bool
}

type WorkflowQuarantinePolicyConfigReader interface {
	GetQuarantinePolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*domainsystemconfig.QuarantinePolicyConfig, error)
}

type WorkflowRunnerDeps struct {
	ProcessHealthStore  *runtimehealth.Store
	BuildService        buildsteps.BuildControlPlaneService
	Preflight           buildsteps.InfrastructurePreflight
	WorkflowRepo        domainworkflow.Repository
	ImageImportRepo     domainimageimport.Repository
	PipelineManager     domainbuild.PipelineManager
	Infrastructure      *domaininfrastructure.Service
	SystemConfigService WorkflowQuarantinePolicyConfigReader
	EventBus            messaging.EventBus
	Logger              *zap.Logger
}

func StartWorkflowRunner(deps WorkflowRunnerDeps, cfg WorkflowRunnerConfig) *appworkflow.Controller {
	if deps.ProcessHealthStore == nil {
		return nil
	}
	if !cfg.Enabled {
		deps.ProcessHealthStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "workflow orchestrator disabled",
		})
		if deps.Logger != nil {
			deps.Logger.Info("Background process disabled",
				zap.String("component", "workflow_orchestrator"),
			)
		}
		return nil
	}

	deps.ProcessHealthStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      false,
		LastActivity: time.Now().UTC(),
		Message:      "workflow orchestrator is starting",
	})

	handlers := buildsteps.NewPhase2ControlPlaneHandlers(
		deps.BuildService,
		deps.Preflight,
		deps.Logger,
	)

	imageImportDispatcher := quarantineimportadapter.NewTektonDispatcher(
		deps.PipelineManager,
		os.Getenv("IF_QUARANTINE_IMPORT_NAMESPACE"),
		os.Getenv("IF_QUARANTINE_IMPORT_PIPELINE_NAME"),
		os.Getenv("IF_QUARANTINE_IMPORT_TARGET_REGISTRY"),
		os.Getenv("IF_QUARANTINE_IMPORT_DOCKERCONFIG_SECRET"),
		deps.Logger,
	)
	imageImportDispatcher.SetInfrastructureService(deps.Infrastructure)

	if deps.Logger != nil {
		deps.Logger.Info("Quarantine dispatcher selection mode",
			zap.String("mode", "provider_detect"),
			zap.Bool("tekton_config_enabled", cfg.BuildTektonMode),
			zap.Bool("startup_pipeline_manager_configured", deps.PipelineManager != nil),
			zap.String("required_provider_flags", "tekton_enabled=true, quarantine_dispatch_enabled=true"),
		)
	}

	importPolicyProvider := imageimportsteps.QuarantinePolicyProviderFunc(func(ctx context.Context, tenantID uuid.UUID) (*imageimportsteps.QuarantinePolicy, error) {
		if deps.SystemConfigService == nil {
			return nil, nil
		}
		policyCfg, err := deps.SystemConfigService.GetQuarantinePolicyConfig(ctx, &tenantID)
		if err != nil {
			return nil, err
		}
		if policyCfg == nil {
			return nil, nil
		}
		return &imageimportsteps.QuarantinePolicy{
			Mode:        policyCfg.Mode,
			MaxCritical: policyCfg.MaxCritical,
			MaxP2:       policyCfg.MaxP2,
			MaxP3:       policyCfg.MaxP3,
			MaxCVSS:     policyCfg.MaxCVSS,
			Thresholds: map[string]int{
				"max_critical": policyCfg.MaxCritical,
				"max_p2":       policyCfg.MaxP2,
				"max_p3":       policyCfg.MaxP3,
			},
			Metadata: map[string]string{
				"source": "system_config.quarantine_policy",
			},
		}, nil
	})

	handlers = append(handlers, imageimportsteps.NewExternalImageImportWorkflowHandlersWithPolicyAndEvents(
		deps.ImageImportRepo,
		deps.WorkflowRepo,
		imageImportDispatcher,
		imageImportDispatcher,
		importPolicyProvider,
		deps.EventBus,
		deps.Logger,
	)...)

	if deps.Logger != nil {
		deps.Logger.Info("Background process starting",
			zap.String("component", "workflow_orchestrator"),
			zap.Duration("poll_interval", cfg.PollInterval),
			zap.Int("max_steps_per_tick", cfg.MaxStepsPerTick),
			zap.String("mode", "phase2_build_control_plane"),
		)
	}

	orchestrator := appworkflow.NewOrchestrator(deps.WorkflowRepo, handlers, deps.Logger)
	controller := appworkflow.NewController(
		orchestrator,
		appworkflow.ControllerConfig{
			PollInterval:    cfg.PollInterval,
			MaxStepsPerTick: cfg.MaxStepsPerTick,
		},
		appworkflow.ControllerHooks{
			OnStart: func() {
				deps.ProcessHealthStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: time.Now().UTC(),
					Message:      "workflow orchestrator running",
				})
			},
			OnTick: func() {
				deps.ProcessHealthStore.Touch("workflow_orchestrator")
			},
			OnStop: func() {
				deps.ProcessHealthStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      false,
					LastActivity: time.Now().UTC(),
					Message:      "workflow orchestrator stopped",
				})
			},
		},
		deps.Logger,
	)
	controller.Start(context.Background())
	return controller
}
