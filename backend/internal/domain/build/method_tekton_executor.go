package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var tektonRecoveryOnce sync.Once
var (
	tektonMonitorModeMu      sync.RWMutex
	tektonMonitorEventDriven *bool
)

// MethodTektonExecutor executes builds using Tekton pipelines on Kubernetes
type MethodTektonExecutor struct {
	k8sClient       kubernetes.Interface
	tektonClient    tektonclient.Interface
	logger          *zap.Logger
	namespaceMgr    NamespaceManager
	pipelineMgr     PipelineManager
	templateEngine  TemplateEngine
	service         BuildExecutionService
	configRepo      BuildMethodConfigRepository
	buildRepo       Repository
	clientProvider  TektonClientProvider
	registryAuth    RegistryDockerConfigResolver
	repositoryAuth  RepositoryGitAuthResolver
	instanceID      string
	monitorLeaseTTl time.Duration
	leaseRenewEvery time.Duration
	running         map[string]tektonExecutionRuntime
	mu              sync.Mutex
}

type tektonExecutionRuntime struct {
	cancel          context.CancelFunc
	namespace       string
	pipelineRunName string
	pipelineMgr     PipelineManager
}

// SetTektonMonitorEventDrivenOverride applies a runtime hint used to tune Tekton monitor polling cadence.
// Pass nil to clear and fallback to environment defaults.
func SetTektonMonitorEventDrivenOverride(value *bool) {
	tektonMonitorModeMu.Lock()
	defer tektonMonitorModeMu.Unlock()
	if value == nil {
		tektonMonitorEventDriven = nil
		return
	}
	v := *value
	tektonMonitorEventDriven = &v
}

// RegistryDockerConfigResolver resolves registry auth into docker config JSON content.
type RegistryDockerConfigResolver interface {
	ResolveDockerConfigJSON(ctx context.Context, registryAuthID uuid.UUID) ([]byte, error)
}

// RepositoryGitAuthResolver resolves project repository auth into git-auth secret data.
type RepositoryGitAuthResolver interface {
	ResolveGitAuthSecretData(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error)
}

// NewMethodTektonExecutor creates a new Tekton executor
func NewMethodTektonExecutor(
	k8sClient kubernetes.Interface,
	tektonClient tektonclient.Interface,
	logger *zap.Logger,
	namespaceMgr NamespaceManager,
	pipelineMgr PipelineManager,
	templateEngine TemplateEngine,
	service BuildExecutionService,
	configRepo BuildMethodConfigRepository,
	buildRepo Repository,
	clientProvider TektonClientProvider,
	registryAuth RegistryDockerConfigResolver,
	repositoryAuth RepositoryGitAuthResolver,
) BuildMethodExecutor {
	executor := &MethodTektonExecutor{
		k8sClient:       k8sClient,
		tektonClient:    tektonClient,
		logger:          logger,
		namespaceMgr:    namespaceMgr,
		pipelineMgr:     pipelineMgr,
		templateEngine:  templateEngine,
		service:         service,
		configRepo:      configRepo,
		buildRepo:       buildRepo,
		clientProvider:  clientProvider,
		registryAuth:    registryAuth,
		repositoryAuth:  repositoryAuth,
		instanceID:      uuid.NewString(),
		monitorLeaseTTl: 45 * time.Second,
		leaseRenewEvery: 15 * time.Second,
		running:         make(map[string]tektonExecutionRuntime),
	}
	executor.startRecoveryLoopOnce()
	return executor
}

func (e *MethodTektonExecutor) startRecoveryLoopOnce() {
	tektonRecoveryOnce.Do(func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := e.reconcileRunningExecutions(ctx); err != nil {
				e.logger.Warn("Tekton execution reconciliation completed with errors", zap.Error(err))
			}
		}()
	})
}

// Supports checks if this executor supports the given method
func (e *MethodTektonExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodDocker ||
		method == BuildMethodBuildx ||
		method == BuildMethodKaniko ||
		method == BuildMethodPacker
}

// executeWithConfig executes with a BuildConfig (internal method)
func (e *MethodTektonExecutor) executeWithConfig(ctx context.Context, executionID uuid.UUID, build *Build, config BuildMethodConfig, method BuildMethod) (*MethodExecutionOutput, error) {
	if build == nil {
		return nil, fmt.Errorf("build is required for Tekton execution")
	}

	k8sClient, tektonClient, namespaceMgr, pipelineMgr, err := e.resolveClients(ctx, build)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			e.service.AddLog(ctx, executionID, LogWarn, "Build execution canceled before Tekton client resolution", nil)
		} else {
			e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Failed to resolve Tekton clients: %v", err), nil)
		}
		return nil, err
	}

	if namespaceMgr == nil || pipelineMgr == nil || k8sClient == nil || tektonClient == nil {
		e.service.AddLog(ctx, executionID, LogError, "Tekton clients are not configured", nil)
		return nil, fmt.Errorf("tekton clients are not configured")
	}

	namespace := namespaceMgr.GetNamespace(build.TenantID())
	if _, err := k8sClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{Limit: 1}); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Tenant namespace probe failed: %v", err), nil)
		switch {
		case apierrors.IsNotFound(err):
			return nil, fmt.Errorf("tenant namespace %q is not prepared (missing). Run provider tenant-namespace prepare and retry", namespace)
		case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
			return nil, fmt.Errorf("tenant namespace %q access denied for runtime credentials: %w", namespace, err)
		default:
			return nil, fmt.Errorf("failed to probe tenant namespace %q: %w", namespace, err)
		}
	}

	e.service.AddLog(ctx, executionID, LogInfo, fmt.Sprintf("Using Kubernetes namespace: %s", namespace), nil)

	var registryAuthID *uuid.UUID
	var targetImageRef string
	if manifest := build.Manifest(); manifest.BuildConfig != nil {
		registryAuthID = manifest.BuildConfig.RegistryAuthID
		targetImageRef = manifest.BuildConfig.RegistryRepo
	}
	if err := e.reconcileDockerConfigSecret(ctx, executionID, namespace, method, registryAuthID, targetImageRef, k8sClient); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Failed to reconcile docker-config secret: %v", err), nil)
		return nil, fmt.Errorf("failed to reconcile docker-config secret: %w", err)
	}
	includeGitAuth, err := e.reconcileGitAuthSecret(ctx, executionID, namespace, build.ProjectID(), k8sClient)
	if err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Failed to reconcile git-auth secret: %v", err), nil)
		return nil, fmt.Errorf("failed to reconcile git-auth secret: %w", err)
	}

	template, err := e.selectPipelineTemplate(method)
	if err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("No pipeline template for method %s: %v", method, err), nil)
		return nil, fmt.Errorf("no pipeline template for method %s: %w", method, err)
	}

	renderCtx := newTektonRenderContext(build, config, method)
	renderCtx.IncludeGitAuth = includeGitAuth
	if method == BuildMethodKaniko && renderCtx.DockerfileInlineBase64 != "" {
		e.service.AddLog(ctx, executionID, LogInfo, fmt.Sprintf("Inline Dockerfile provided; overriding repository Dockerfile path %q", renderCtx.DockerfilePath), nil)
	}
	if err := validateTektonRenderContext(renderCtx, method); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Invalid Tekton render context: %v", err), nil)
		return nil, err
	}
	pipelineRunYAML, err := e.templateEngine.Render(template, renderCtx)
	if err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Failed to render pipeline: %v", err), nil)
		return nil, fmt.Errorf("failed to render pipeline: %w", err)
	}

	if err := e.preflightPipelineRun(ctx, namespace, pipelineRunYAML, k8sClient, tektonClient); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Tekton preflight failed: %v", err), nil)
		return nil, fmt.Errorf("tekton preflight failed: %w", err)
	}

	pipelineRun, err := pipelineMgr.CreatePipelineRun(ctx, namespace, pipelineRunYAML)
	if err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Failed to create PipelineRun: %v", err), nil)
		return nil, fmt.Errorf("failed to create PipelineRun: %w", err)
	}

	e.service.AddLog(ctx, executionID, LogInfo, fmt.Sprintf("Created PipelineRun: %s", pipelineRun.Name), nil)
	e.logPipelineRunScheduled(ctx, executionID, build, method, pipelineRun)
	e.persistPipelineRunMetadata(ctx, executionID, build, method, pipelineRun)

	if started := e.tryStartMonitoring(ctx, executionID, pipelineRun, pipelineMgr, k8sClient, tektonClient, "scheduled"); !started {
		e.service.AddLog(ctx, executionID, LogWarn, "PipelineRun scheduled but monitoring lease is owned by another instance", nil)
	}

	return &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		Output:      fmt.Sprintf("PipelineRun %s started successfully", pipelineRun.Name),
		StartTime:   time.Now().Unix(),
	}, nil
}

func (e *MethodTektonExecutor) selectPipelineTemplate(method BuildMethod) (string, error) {
	switch method {
	case BuildMethodDocker:
		return e.getDockerPipelineTemplate(), nil
	case BuildMethodBuildx:
		return e.getBuildxPipelineTemplate(), nil
	case BuildMethodKaniko:
		return e.getKanikoPipelineTemplate(), nil
	case BuildMethodPacker:
		return e.getPackerPipelineTemplate(), nil
	default:
		return "", fmt.Errorf("unsupported build method for Tekton: %s", method)
	}
}

func (e *MethodTektonExecutor) Cancel(ctx context.Context, executionIDStr string) error {
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return fmt.Errorf("invalid execution ID: %w", err)
	}

	e.mu.Lock()
	runtimeState, exists := e.running[executionIDStr]
	e.mu.Unlock()

	if exists {
		runtimeState.cancel()
		if err := e.deletePipelineRun(ctx, executionID, runtimeState.namespace, runtimeState.pipelineRunName, runtimeState.pipelineMgr); err != nil {
			return err
		}
		_ = e.service.ReleaseMonitoringLease(context.Background(), executionID, e.instanceID)
		e.service.AddLog(ctx, executionID, LogInfo, "Pipeline execution cancelled by user", nil)
		return nil
	}

	execution, err := e.service.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to load execution for cancellation: %w", err)
	}
	namespace, pipelineRunName, ok := extractTektonExecutionRef(execution.Metadata)
	if !ok {
		return fmt.Errorf("execution not found or missing tekton reference: %s", executionIDStr)
	}

	build, err := e.getBuild(ctx, execution.BuildID)
	if err != nil {
		return fmt.Errorf("failed to load build for cancellation: %w", err)
	}
	_, _, _, pipelineMgr, err := e.resolveClients(ctx, build)
	if err != nil {
		return fmt.Errorf("failed to resolve clients for cancellation: %w", err)
	}

	if err := e.deletePipelineRun(ctx, executionID, namespace, pipelineRunName, pipelineMgr); err != nil {
		return err
	}
	_ = e.service.ReleaseMonitoringLease(context.Background(), executionID, e.instanceID)
	e.service.AddLog(ctx, executionID, LogInfo, "Pipeline execution cancelled by user", nil)
	return nil
}

func (e *MethodTektonExecutor) GetStatus(ctx context.Context, executionIDStr string) (ExecutionStatus, error) {
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return ExecutionPending, fmt.Errorf("invalid execution ID: %w", err)
	}

	execution, err := e.service.GetExecution(ctx, executionID)
	if err != nil {
		return ExecutionPending, err
	}

	return execution.Status, nil
}

func (e *MethodTektonExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	e.logger.Info("Starting Tekton pipeline execution",
		zap.String("execution_id", executionID.String()),
		zap.String("method", string(method)))

	execution, err := e.service.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	config, err := e.getBuildConfig(ctx, execution.BuildID, method)
	if err != nil {
		return nil, fmt.Errorf("failed to get build config: %w", err)
	}

	build, err := e.getBuild(ctx, execution.BuildID)
	if err != nil {
		return nil, err
	}

	return e.executeWithConfig(ctx, executionID, build, config, method)
}

func (e *MethodTektonExecutor) getBuild(ctx context.Context, buildID uuid.UUID) (*Build, error) {
	if e.buildRepo == nil {
		return nil, fmt.Errorf("build repository is not configured")
	}
	build, err := e.buildRepo.FindByID(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to load build: %w", err)
	}
	if build == nil {
		return nil, fmt.Errorf("build not found: %s", buildID.String())
	}
	return build, nil
}

func (e *MethodTektonExecutor) resolveClients(ctx context.Context, build *Build) (kubernetes.Interface, tektonclient.Interface, NamespaceManager, PipelineManager, error) {
	if e.clientProvider != nil {
		clients, err := e.clientProvider.ClientsForBuild(ctx, build)
		if err == nil && clients != nil {
			return clients.K8sClient, clients.TektonClient, clients.NamespaceMgr, clients.PipelineMgr, nil
		}
		if err != nil {
			e.logger.Warn("Failed to resolve Tekton clients from infrastructure provider", zap.Error(err))
		}
	}

	if e.k8sClient == nil || e.tektonClient == nil || e.namespaceMgr == nil || e.pipelineMgr == nil {
		return nil, nil, nil, nil, fmt.Errorf("tekton clients are not available")
	}

	return e.k8sClient, e.tektonClient, e.namespaceMgr, e.pipelineMgr, nil
}

func (e *MethodTektonExecutor) getBuildConfig(ctx context.Context, buildID uuid.UUID, method BuildMethod) (BuildMethodConfig, error) {
	return e.configRepo.FindByBuildIDAndMethod(ctx, buildID, method)
}

func (e *MethodTektonExecutor) logPipelineRunScheduled(ctx context.Context, executionID uuid.UUID, build *Build, method BuildMethod, pipelineRun *tektonv1.PipelineRun) {
	metadata := map[string]interface{}{
		"phase":     "scheduled",
		"method":    method,
		"namespace": pipelineRun.Namespace,
		"name":      pipelineRun.Name,
	}
	if pipelineRun.Spec.PipelineRef != nil {
		metadata["pipeline"] = pipelineRun.Spec.PipelineRef.Name
	}
	if build != nil && build.InfrastructureProviderID() != nil {
		metadata["infrastructure_provider_id"] = build.InfrastructureProviderID().String()
	}

	raw, err := json.Marshal(metadata)
	if err != nil {
		e.logger.Warn("Failed to marshal PipelineRun schedule metadata", zap.Error(err))
		return
	}
	_ = e.service.AddLog(ctx, executionID, LogInfo, "PipelineRun scheduled", raw)
}

func (e *MethodTektonExecutor) persistPipelineRunMetadata(ctx context.Context, executionID uuid.UUID, build *Build, method BuildMethod, pipelineRun *tektonv1.PipelineRun) {
	execution, err := e.service.GetExecution(ctx, executionID)
	if err != nil {
		e.logger.Warn("Failed to load execution for metadata update", zap.String("execution_id", executionID.String()), zap.Error(err))
		return
	}

	metadata := map[string]interface{}{}
	if len(execution.Metadata) > 0 {
		if err := json.Unmarshal(execution.Metadata, &metadata); err != nil {
			e.logger.Warn("Failed to parse existing execution metadata, resetting", zap.String("execution_id", executionID.String()), zap.Error(err))
			metadata = map[string]interface{}{}
		}
		if metadata == nil {
			metadata = map[string]interface{}{}
		}
	}

	tektonMetadata := map[string]interface{}{
		"namespace":    pipelineRun.Namespace,
		"pipeline_run": pipelineRun.Name,
		"method":       method,
		"scheduled_at": time.Now().UTC().Format(time.RFC3339),
	}
	if pipelineRun.Spec.PipelineRef != nil {
		tektonMetadata["pipeline"] = pipelineRun.Spec.PipelineRef.Name
	}
	if build != nil && build.InfrastructureProviderID() != nil {
		tektonMetadata["provider_id"] = build.InfrastructureProviderID().String()
	}
	metadata["tekton"] = tektonMetadata

	payload, err := json.Marshal(metadata)
	if err != nil {
		e.logger.Warn("Failed to marshal execution metadata", zap.String("execution_id", executionID.String()), zap.Error(err))
		return
	}

	if err := e.service.UpdateExecutionMetadata(ctx, executionID, payload); err != nil {
		e.logger.Warn("Failed to persist execution metadata", zap.String("execution_id", executionID.String()), zap.Error(err))
		return
	}
}

func (e *MethodTektonExecutor) reconcileRunningExecutions(ctx context.Context) error {
	executions, err := e.service.ListRunningExecutions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list running executions: %w", err)
	}

	for _, execution := range executions {
		namespace, pipelineRunName, ok := extractTektonExecutionRef(execution.Metadata)
		if !ok {
			continue
		}

		e.mu.Lock()
		_, alreadyRunning := e.running[execution.ID.String()]
		e.mu.Unlock()
		if alreadyRunning {
			continue
		}

		build, err := e.getBuild(ctx, execution.BuildID)
		if err != nil {
			e.logger.Warn("Skipping execution recovery: failed to load build",
				zap.String("execution_id", execution.ID.String()),
				zap.String("build_id", execution.BuildID.String()),
				zap.Error(err))
			continue
		}

		k8sClient, tektonClient, _, pipelineMgr, err := e.resolveClients(ctx, build)
		if err != nil {
			e.logger.Warn("Skipping execution recovery: failed to resolve Tekton clients",
				zap.String("execution_id", execution.ID.String()),
				zap.String("build_id", execution.BuildID.String()),
				zap.Error(err))
			continue
		}

		acquired, err := e.service.TryAcquireMonitoringLease(ctx, execution.ID, e.instanceID, e.monitorLeaseTTl)
		if err != nil {
			e.logger.Warn("Skipping execution recovery: failed to acquire monitoring lease",
				zap.String("execution_id", execution.ID.String()),
				zap.Error(err))
			continue
		}
		if !acquired {
			continue
		}

		pipelineRun, err := pipelineMgr.GetPipelineRun(ctx, namespace, pipelineRunName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				e.logger.Warn("Recovered execution references missing PipelineRun; marking execution failed",
					zap.String("execution_id", execution.ID.String()),
					zap.String("namespace", namespace),
					zap.String("pipeline_run", pipelineRunName))
				e.finalizeExecution(ctx, execution.ID, ExecutionFailed, fmt.Sprintf("PipelineRun %s/%s not found during recovery", namespace, pipelineRunName), nil)
				_ = e.service.ReleaseMonitoringLease(context.Background(), execution.ID, e.instanceID)
				continue
			}
			e.logger.Warn("Skipping execution recovery: failed to fetch PipelineRun",
				zap.String("execution_id", execution.ID.String()),
				zap.String("namespace", namespace),
				zap.String("pipeline_run", pipelineRunName),
				zap.Error(err))
			_ = e.service.ReleaseMonitoringLease(context.Background(), execution.ID, e.instanceID)
			continue
		}

		if pipelineRun.Status.CompletionTime != nil {
			e.finalizeExecutionFromPipelineRun(ctx, execution.ID, pipelineRun, "", tektonClient)
			_ = e.service.ReleaseMonitoringLease(context.Background(), execution.ID, e.instanceID)
			continue
		}

		monitorCtx, cancel := context.WithCancel(context.Background())
		e.mu.Lock()
		if e.running == nil {
			e.running = make(map[string]tektonExecutionRuntime)
		}
		e.running[execution.ID.String()] = tektonExecutionRuntime{
			cancel:          cancel,
			namespace:       namespace,
			pipelineRunName: pipelineRunName,
			pipelineMgr:     pipelineMgr,
		}
		e.mu.Unlock()

		e.service.AddLog(ctx, execution.ID, LogInfo, fmt.Sprintf("Recovered Tekton PipelineRun monitoring for %s/%s", namespace, pipelineRunName), nil)
		if emitter, ok := e.service.(BuildStatusUpdateEmitter); ok {
			emitter.EmitBuildStatusUpdate(ctx, execution.BuildID, "recovered", "Build monitoring recovered", map[string]interface{}{
				"execution_id": execution.ID.String(),
				"namespace":    namespace,
				"pipeline_run": pipelineRunName,
				"source":       "tekton_recovery",
			})
		}
		go e.monitorPipelineRun(monitorCtx, execution.ID, pipelineRun, k8sClient, tektonClient)
	}

	return nil
}

func (e *MethodTektonExecutor) tryStartMonitoring(
	ctx context.Context,
	executionID uuid.UUID,
	pipelineRun *tektonv1.PipelineRun,
	pipelineMgr PipelineManager,
	k8sClient kubernetes.Interface,
	tektonClient tektonclient.Interface,
	source string,
) bool {
	acquired, err := e.service.TryAcquireMonitoringLease(ctx, executionID, e.instanceID, e.monitorLeaseTTl)
	if err != nil {
		e.logger.Warn("Failed to acquire monitoring lease",
			zap.String("execution_id", executionID.String()),
			zap.String("source", source),
			zap.Error(err))
		return false
	}
	if !acquired {
		e.logger.Info("Monitoring lease already owned by another instance",
			zap.String("execution_id", executionID.String()),
			zap.String("source", source))
		return false
	}

	monitorCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	if e.running == nil {
		e.running = make(map[string]tektonExecutionRuntime)
	}
	e.running[executionID.String()] = tektonExecutionRuntime{
		cancel:          cancel,
		namespace:       pipelineRun.Namespace,
		pipelineRunName: pipelineRun.Name,
		pipelineMgr:     pipelineMgr,
	}
	e.mu.Unlock()

	go e.monitorPipelineRun(monitorCtx, executionID, pipelineRun, k8sClient, tektonClient)
	return true
}

func (e *MethodTektonExecutor) deletePipelineRun(
	ctx context.Context,
	executionID uuid.UUID,
	namespace string,
	pipelineRunName string,
	pipelineMgr PipelineManager,
) error {
	if pipelineMgr == nil || namespace == "" || pipelineRunName == "" {
		return fmt.Errorf("missing PipelineRun reference for cancellation")
	}
	if deleteErr := pipelineMgr.DeletePipelineRun(ctx, namespace, pipelineRunName); deleteErr != nil {
		if apierrors.IsNotFound(deleteErr) {
			e.service.AddLog(ctx, executionID, LogInfo, fmt.Sprintf("PipelineRun %s/%s already deleted", namespace, pipelineRunName), nil)
			return nil
		}
		e.logger.Warn("Failed to delete PipelineRun during cancellation",
			zap.String("execution_id", executionID.String()),
			zap.String("namespace", namespace),
			zap.String("pipeline_run", pipelineRunName),
			zap.Error(deleteErr))
		e.service.AddLog(ctx, executionID, LogWarn, fmt.Sprintf("Failed to delete PipelineRun %s/%s: %v", namespace, pipelineRunName, deleteErr), nil)
		return fmt.Errorf("failed to delete PipelineRun %s/%s: %w", namespace, pipelineRunName, deleteErr)
	}
	e.service.AddLog(ctx, executionID, LogInfo, fmt.Sprintf("Deleted PipelineRun %s/%s", namespace, pipelineRunName), nil)
	return nil
}

func extractTektonExecutionRef(raw json.RawMessage) (string, string, bool) {
	if len(raw) == 0 {
		return "", "", false
	}

	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return "", "", false
	}

	tektonRaw, ok := metadata["tekton"]
	if !ok || len(tektonRaw) == 0 {
		return "", "", false
	}

	var tekton struct {
		Namespace   string `json:"namespace"`
		PipelineRun string `json:"pipeline_run"`
	}
	if err := json.Unmarshal(tektonRaw, &tekton); err != nil {
		return "", "", false
	}
	if tekton.Namespace == "" || tekton.PipelineRun == "" {
		return "", "", false
	}
	return tekton.Namespace, tekton.PipelineRun, true
}
