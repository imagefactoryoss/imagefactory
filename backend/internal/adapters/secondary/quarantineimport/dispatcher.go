package quarantineimport

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	imageimportsteps "github.com/srikarm/image-factory/internal/application/imageimport/steps"
	domainbuild "github.com/srikarm/image-factory/internal/domain/build"
	domainimageimport "github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	k8sinfra "github.com/srikarm/image-factory/internal/infrastructure/kubernetes"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultQuarantinePipelineName = "image-factory-build-v1-quarantine-import"
	defaultQuarantineNamespace    = "default"
	defaultDockerConfigSecretName = "docker-config"
	defaultTargetRegistry         = "image-factory-registry:5000"
)

var ErrDispatcherUnavailable = errors.New("waiting_for_dispatch: no tekton-enabled quarantine dispatcher available")

type Dispatcher struct {
	mu                     sync.RWMutex
	pipelineManager        domainbuild.PipelineManager
	infraService           *infrastructure.Service
	namespace              string
	pipelineName           string
	targetRegistry         string
	dockerConfigSecretName string
	logger                 *zap.Logger
}

func NewTektonDispatcher(
	pipelineManager domainbuild.PipelineManager,
	namespace string,
	pipelineName string,
	targetRegistry string,
	dockerConfigSecretName string,
	logger *zap.Logger,
) *Dispatcher {
	return &Dispatcher{
		pipelineManager:        pipelineManager,
		namespace:              valueOrDefault(namespace, defaultQuarantineNamespace),
		pipelineName:           valueOrDefault(pipelineName, defaultQuarantinePipelineName),
		targetRegistry:         strings.Trim(strings.TrimSpace(valueOrDefault(targetRegistry, defaultTargetRegistry)), "/"),
		dockerConfigSecretName: valueOrDefault(dockerConfigSecretName, defaultDockerConfigSecretName),
		logger:                 logger,
	}
}

func (d *Dispatcher) SetPipelineManager(pipelineManager domainbuild.PipelineManager) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.pipelineManager = pipelineManager
	d.mu.Unlock()
}

func (d *Dispatcher) SetInfrastructureService(infraService *infrastructure.Service) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.infraService = infraService
	d.mu.Unlock()
}

func (d *Dispatcher) HasPipelineManager() bool {
	if d == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.pipelineManager != nil
}

func (d *Dispatcher) Dispatch(ctx context.Context, req *domainimageimport.ImportRequest) (imageimportsteps.DispatchResult, error) {
	if req == nil {
		return imageimportsteps.DispatchResult{}, fmt.Errorf("import request is required")
	}
	pipelineManager, namespace, pipelineName, targetRegistry, dockerConfigSecretName, err := d.resolveDispatchContext(ctx, req)
	if err != nil {
		if errors.Is(err, ErrDispatcherUnavailable) {
			return imageimportsteps.DispatchResult{}, ErrDispatcherUnavailable
		}
		return imageimportsteps.DispatchResult{}, err
	}
	if pipelineManager == nil {
		return imageimportsteps.DispatchResult{}, ErrDispatcherUnavailable
	}

	pipelineRunYAML := d.renderPipelineRunYAML(req, pipelineName, targetRegistry, dockerConfigSecretName)
	created, err := pipelineManager.CreatePipelineRun(ctx, namespace, pipelineRunYAML)
	if err != nil {
		return imageimportsteps.DispatchResult{}, fmt.Errorf("failed to create quarantine pipeline run: %w", err)
	}
	result := imageimportsteps.DispatchResult{
		PipelineRunName:  created.Name,
		Namespace:        namespace,
		InternalImageRef: buildInternalPublishedRef(targetRegistry, req),
	}

	if d.logger != nil {
		d.logger.Info("Dispatched external image import via Tekton",
			zap.String("import_request_id", req.ID.String()),
			zap.String("tenant_id", req.TenantID.String()),
			zap.String("pipeline_name", d.pipelineName),
			zap.String("pipeline_run_name", result.PipelineRunName),
			zap.String("namespace", result.Namespace),
		)
	}

	return result, nil
}

func (d *Dispatcher) GetPipelineRun(ctx context.Context, req *domainimageimport.ImportRequest) (*tektonv1.PipelineRun, error) {
	if req == nil {
		return nil, fmt.Errorf("import request is required")
	}
	if strings.TrimSpace(req.PipelineNamespace) == "" || strings.TrimSpace(req.PipelineRunName) == "" {
		return nil, fmt.Errorf("pipeline namespace and name are required")
	}
	pipelineManager, _, _, _, _, err := d.resolveDispatchContext(ctx, req)
	if err != nil {
		if errors.Is(err, ErrDispatcherUnavailable) {
			return nil, ErrDispatcherUnavailable
		}
		return nil, err
	}
	if pipelineManager == nil {
		return nil, ErrDispatcherUnavailable
	}
	return pipelineManager.GetPipelineRun(ctx, req.PipelineNamespace, req.PipelineRunName)
}

func (d *Dispatcher) resolveDispatchContext(ctx context.Context, req *domainimageimport.ImportRequest) (domainbuild.PipelineManager, string, string, string, string, error) {
	d.mu.RLock()
	fallbackPipelineManager := d.pipelineManager
	infraService := d.infraService
	namespace := d.namespace
	pipelineName := d.pipelineName
	targetRegistry := d.targetRegistry
	dockerConfigSecretName := d.dockerConfigSecretName
	d.mu.RUnlock()

	if infraService == nil {
		if fallbackPipelineManager == nil {
			return nil, "", "", "", "", ErrDispatcherUnavailable
		}
		return fallbackPipelineManager, namespace, pipelineName, targetRegistry, dockerConfigSecretName, nil
	}

	providers, err := infraService.GetAvailableProviders(ctx, req.TenantID)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("failed to list available infrastructure providers: %w", err)
	}
	selected := selectQuarantineDispatchProvider(providers)
	if selected == nil {
		return nil, "", "", "", "", ErrDispatcherUnavailable
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfig(selected.Config)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("selected provider %s has invalid kubeconfig/config: %w", selected.ID.String(), err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("failed to create Kubernetes client from provider %s: %w", selected.ID.String(), err)
	}
	tektonClient, err := tektonclient.NewForConfig(restConfig)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("failed to create Tekton client from provider %s: %w", selected.ID.String(), err)
	}
	providerNamespace := strings.TrimSpace(configString(selected.Config, "quarantine_import_namespace"))
	providerNamespace = resolveQuarantineDispatchNamespace(selected, req.TenantID, providerNamespace, namespace)
	return k8sinfra.NewKubernetesPipelineManager(k8sClient, tektonClient, d.logger), providerNamespace, pipelineName, targetRegistry, dockerConfigSecretName, nil
}

func selectQuarantineDispatchProvider(providers []*infrastructure.Provider) *infrastructure.Provider {
	if len(providers) == 0 {
		return nil
	}
	candidates := make([]*infrastructure.Provider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if !isKubernetesProvider(provider.ProviderType) {
			continue
		}
		if !configBool(provider.Config, "tekton_enabled", false) {
			continue
		}
		if !configBool(provider.Config, "quarantine_dispatch_enabled", false) {
			continue
		}
		candidates = append(candidates, provider)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		lp := configInt(left.Config, "quarantine_dispatch_priority", 0)
		rp := configInt(right.Config, "quarantine_dispatch_priority", 0)
		if lp != rp {
			return lp > rp
		}
		if left.IsGlobal != right.IsGlobal {
			return left.IsGlobal
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})
	return candidates[0]
}

func isKubernetesProvider(providerType infrastructure.ProviderType) bool {
	switch providerType {
	case infrastructure.ProviderTypeKubernetes,
		infrastructure.ProviderTypeAWSEKS,
		infrastructure.ProviderTypeGCPGKE,
		infrastructure.ProviderTypeAzureAKS,
		infrastructure.ProviderTypeOCIOKE,
		infrastructure.ProviderTypeVMwareVKS,
		infrastructure.ProviderTypeOpenShift,
		infrastructure.ProviderTypeRancher:
		return true
	default:
		return false
	}
}

func resolveQuarantineDispatchNamespace(provider *infrastructure.Provider, tenantID uuid.UUID, explicitNamespace, fallbackNamespace string) string {
	if ns := strings.TrimSpace(explicitNamespace); ns != "" {
		return ns
	}
	if tenantID != uuid.Nil {
		return fmt.Sprintf("image-factory-%s", tenantID.String()[:8])
	}
	if provider != nil {
		if provider.TargetNamespace != nil {
			if ns := strings.TrimSpace(*provider.TargetNamespace); ns != "" {
				return ns
			}
		}
		if provider.Config != nil {
			if ns := strings.TrimSpace(configString(provider.Config, "tekton_target_namespace")); ns != "" {
				return ns
			}
			if ns := strings.TrimSpace(configString(provider.Config, "tekton_namespace")); ns != "" {
				return ns
			}
			if ns := strings.TrimSpace(configString(provider.Config, "system_namespace")); ns != "" {
				return ns
			}
		}
	}
	return strings.TrimSpace(fallbackNamespace)
}

func (d *Dispatcher) renderPipelineRunYAML(req *domainimageimport.ImportRequest, pipelineName, targetRegistry, dockerConfigSecretName string) string {
	internalPublished := buildInternalPublishedRef(targetRegistry, req)
	internalQuarantine := buildInternalQuarantineRef(targetRegistry, req)
	policySnapshot := strings.TrimSpace(req.PolicySnapshotJSON)
	if policySnapshot == "" {
		policySnapshot = `{"mode":"enforce","thresholds":{"max_critical":0,"max_p2":0,"max_p3":999999}}`
	}
	sourceRef := strings.TrimSpace(req.SourceImageRef)
	if sourceRef == "" {
		sourceRef = strings.TrimSpace(req.SourceRegistry)
	}
	return fmt.Sprintf(`apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: quarantine-import-
  labels:
    app.kubernetes.io/part-of: image-factory
    image-factory.io/workflow-subject: external-image-import
    image-factory.io/external-image-import-id: "%s"
spec:
  pipelineRef:
    name: %s
  params:
    - name: external-image-import-id
      value: "%s"
    - name: tenant-id
      value: "%s"
    - name: source-image-ref
      value: "%s"
    - name: target-published-image
      value: "%s"
    - name: target-quarantine-image
      value: "%s"
    - name: policy-snapshot-json
      value: '%s'
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 2Gi
    - name: dockerconfig
      secret:
        secretName: %s
`, req.ID.String(), pipelineName, req.ID.String(), req.TenantID.String(), sourceRef, internalPublished, internalQuarantine, policySnapshot, dockerConfigSecretName)
}

func buildInternalPublishedRef(targetRegistry string, req *domainimageimport.ImportRequest) string {
	return fmt.Sprintf("%s/published/%s/%s:latest", targetRegistry, shortID(req.TenantID.String(), 12), shortID(req.ID.String(), 12))
}

func buildInternalQuarantineRef(targetRegistry string, req *domainimageimport.ImportRequest) string {
	return fmt.Sprintf("%s/quarantine/%s/%s:latest", targetRegistry, shortID(req.TenantID.String(), 12), shortID(req.ID.String(), 12))
}

func shortID(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func configString(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	raw, ok := config[key]
	if !ok || raw == nil {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func configBool(config map[string]interface{}, key string, fallback bool) bool {
	if config == nil {
		return fallback
	}
	raw, ok := config[key]
	if !ok || raw == nil {
		return fallback
	}
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	case float64:
		return typed != 0
	case int:
		return typed != 0
	}
	return fallback
}

func configInt(config map[string]interface{}, key string, fallback int) int {
	if config == nil {
		return fallback
	}
	raw, ok := config[key]
	if !ok || raw == nil {
		return fallback
	}
	switch typed := raw.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
