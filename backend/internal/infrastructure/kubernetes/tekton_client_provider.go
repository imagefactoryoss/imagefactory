package kubernetes

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
)

// TektonClientProvider resolves Tekton clients based on infrastructure provider configuration.
type TektonClientProvider struct {
	infraService *infrastructure.Service
	logger       *zap.Logger
}

// NewTektonClientProvider creates a new Tekton client provider.
func NewTektonClientProvider(infraService *infrastructure.Service, logger *zap.Logger) *TektonClientProvider {
	return &TektonClientProvider{
		infraService: infraService,
		logger:       logger,
	}
}

// ClientsForBuild resolves clients for a build based on available providers.
func (p *TektonClientProvider) ClientsForBuild(ctx context.Context, b *build.Build) (*build.TektonClients, error) {
	if b == nil {
		return nil, fmt.Errorf("build is required")
	}
	if p.infraService == nil {
		return nil, fmt.Errorf("infrastructure service is not configured")
	}
	if b.InfrastructureType() != "kubernetes" {
		return nil, fmt.Errorf("build infrastructure type is not kubernetes")
	}

	providers, err := p.infraService.GetAvailableProviders(ctx, b.TenantID())
	if err != nil {
		return nil, fmt.Errorf("failed to load infrastructure providers: %w", err)
	}

	provider, err := selectKubernetesProvider(providers, b.InfrastructureProviderID(), requiredFallbackCapabilities(b))
	if err != nil {
		return nil, err
	}

	if enabled, ok := provider.Config["tekton_enabled"].(bool); !ok || !enabled {
		return nil, fmt.Errorf("selected infrastructure provider %s has tekton_enabled=false", provider.ID.String())
	}

	// Managed providers must provision per-tenant namespaces using bootstrap credentials.
	// Build execution uses runtime credentials and must not rely on cluster-wide namespace create.
	if isImageFactoryManagedBootstrapMode(provider.BootstrapMode) {
		status, statusErr := p.infraService.GetTenantNamespacePrepareStatus(ctx, provider.ID, b.TenantID())
		if statusErr != nil || status == nil || status.Status != infrastructure.ProviderTenantNamespacePrepareSucceeded {
			if _, err := p.infraService.EnsureTenantNamespaceReady(ctx, provider.ID, b.TenantID(), b.CreatedBy()); err != nil {
				return nil, fmt.Errorf("tenant namespace provisioning failed for provider %s tenant %s: %w", provider.ID.String(), b.TenantID().String(), err)
			}
		}
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to build kube config from provider %s: %w", provider.ID.String(), err)
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	tektonClient, err := tektonclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tekton client: %w", err)
	}

	p.logger.Info("Resolved Tekton client from infrastructure provider",
		zap.String("provider_id", provider.ID.String()),
		zap.String("provider_type", string(provider.ProviderType)),
		zap.String("provider_name", provider.Name),
		zap.String("tenant_id", b.TenantID().String()),
	)

	return &build.TektonClients{
		K8sClient:    k8sClient,
		TektonClient: tektonClient,
		NamespaceMgr: NewKubernetesNamespaceManager(k8sClient, p.logger),
		PipelineMgr:  NewKubernetesPipelineManager(k8sClient, tektonClient, p.logger),
	}, nil
}

func isImageFactoryManagedBootstrapMode(raw string) bool {
	// Default to managed to keep behavior deterministic when data is missing.
	switch strings.TrimSpace(raw) {
	case "self_managed":
		return false
	default:
		return true
	}
}

func selectKubernetesProvider(providers []*infrastructure.Provider, providerID *uuid.UUID, requiredCapabilities []string) (*infrastructure.Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no infrastructure providers available")
	}

	k8sProviders := make([]*infrastructure.Provider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if isKubernetesProvider(provider.ProviderType) {
			k8sProviders = append(k8sProviders, provider)
		}
	}
	if len(k8sProviders) == 0 {
		return nil, fmt.Errorf("no kubernetes infrastructure providers available")
	}

	if providerID != nil {
		for _, provider := range k8sProviders {
			if provider != nil && provider.ID == *providerID {
				if _, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config); err != nil {
					return nil, fmt.Errorf("selected provider is not usable: %w", err)
				}
				return provider, nil
			}
		}
		return nil, fmt.Errorf("selected infrastructure provider not available")
	}

	// If no provider was explicitly selected on the build, only allow fallback to a
	// global/shared provider.
	globalProviders := make([]*infrastructure.Provider, 0, len(k8sProviders))
	for _, provider := range k8sProviders {
		if provider != nil && provider.IsGlobal {
			globalProviders = append(globalProviders, provider)
		}
	}
	if len(globalProviders) == 0 {
		return nil, fmt.Errorf("infrastructure_provider_id is required for kubernetes builds (no global provider available)")
	}
	if len(requiredCapabilities) > 0 {
		matchingCapabilities := make([]*infrastructure.Provider, 0, len(globalProviders))
		for _, provider := range globalProviders {
			if providerSupportsCapabilities(provider, requiredCapabilities) {
				matchingCapabilities = append(matchingCapabilities, provider)
			}
		}
		if len(matchingCapabilities) == 0 {
			return nil, fmt.Errorf("no global kubernetes provider matches required capabilities (%s)", strings.Join(requiredCapabilities, ","))
		}
		globalProviders = matchingCapabilities
	}

	sort.SliceStable(globalProviders, func(i, j int) bool {
		left := globalProviders[i]
		right := globalProviders[j]
		leftLoadScore := providerLoadScore(left)
		rightLoadScore := providerLoadScore(right)
		if !floatEqual(leftLoadScore, rightLoadScore) {
			return leftLoadScore < rightLoadScore
		}
		leftPriority := providerPriority(left.ProviderType)
		rightPriority := providerPriority(right.ProviderType)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	for _, provider := range globalProviders {
		if provider == nil {
			continue
		}
		if _, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config); err == nil {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no global kubernetes provider has a usable configuration")
}

func requiredFallbackCapabilities(b *build.Build) []string {
	if b == nil {
		return nil
	}
	manifest := b.Manifest()
	if manifest.BuildConfig == nil {
		return nil
	}
	required := make([]string, 0, 2)
	seen := map[string]struct{}{}
	addRequired := func(capability string) {
		if capability == "" {
			return
		}
		normalized := strings.ToLower(strings.TrimSpace(capability))
		if normalized == "" {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		required = append(required, normalized)
	}

	for _, capability := range extractRequestedProviderCapabilities(manifest) {
		addRequired(capability)
	}

	for _, platform := range manifest.BuildConfig.Platforms {
		normalized := strings.ToLower(strings.TrimSpace(platform))
		if strings.Contains(normalized, "arm64") {
			addRequired("arm64")
		}
	}

	for key, value := range manifest.BuildConfig.BuildArgs {
		combined := strings.ToLower(strings.TrimSpace(key + "=" + value))
		if containsGPUKeyword(combined) {
			addRequired("gpu")
		}
	}

	sort.Strings(required)
	return required
}

func extractRequestedProviderCapabilities(manifest build.BuildManifest) []string {
	if manifest.Metadata == nil {
		return nil
	}
	for _, key := range []string{"required_provider_capabilities", "requiredProviderCapabilities", "required_capabilities", "requiredCapabilities"} {
		raw, ok := manifest.Metadata[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case []string:
			return typed
		case []interface{}:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if value, ok := item.(string); ok {
					out = append(out, value)
				}
			}
			return out
		case string:
			if strings.TrimSpace(typed) == "" {
				return nil
			}
			parts := strings.Split(typed, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				out = append(out, strings.TrimSpace(part))
			}
			return out
		}
	}
	return nil
}

func providerSupportsCapabilities(provider *infrastructure.Provider, required []string) bool {
	if provider == nil {
		return false
	}
	if len(required) == 0 {
		return true
	}
	if len(provider.Capabilities) == 0 {
		return false
	}

	available := make(map[string]struct{}, len(provider.Capabilities)*2)
	for _, capability := range provider.Capabilities {
		normalized := strings.ToLower(strings.TrimSpace(capability))
		if normalized == "" {
			continue
		}
		available[normalized] = struct{}{}
		switch normalized {
		case "nvidia", "cuda":
			available["gpu"] = struct{}{}
		case "aarch64":
			available["arm64"] = struct{}{}
		}
	}

	for _, requirement := range required {
		if _, ok := available[requirement]; !ok {
			return false
		}
	}
	return true
}

func containsGPUKeyword(value string) bool {
	for _, keyword := range []string{"gpu", "cuda", "nvidia", "amd"} {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func providerLoadScore(provider *infrastructure.Provider) float64 {
	if provider == nil || provider.Config == nil {
		return math.MaxFloat64
	}

	for _, key := range []string{"routing_load_score", "cluster_load_score", "queue_depth"} {
		if value, ok := extractNumericConfigValue(provider.Config[key]); ok {
			return value
		}
	}

	if routingRaw, ok := provider.Config["routing"].(map[string]interface{}); ok {
		for _, key := range []string{"load_score", "cluster_load_score", "queue_depth"} {
			if value, ok := extractNumericConfigValue(routingRaw[key]); ok {
				return value
			}
		}
	}

	return math.MaxFloat64
}

func extractNumericConfigValue(raw interface{}) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func floatEqual(left, right float64) bool {
	return math.Abs(left-right) <= 0.000001
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

func providerPriority(providerType infrastructure.ProviderType) int {
	priorities := map[infrastructure.ProviderType]int{
		infrastructure.ProviderTypeKubernetes: 0,
		infrastructure.ProviderTypeOpenShift:  1,
		infrastructure.ProviderTypeRancher:    2,
		infrastructure.ProviderTypeAWSEKS:     3,
		infrastructure.ProviderTypeGCPGKE:     4,
		infrastructure.ProviderTypeAzureAKS:   5,
		infrastructure.ProviderTypeOCIOKE:     6,
		infrastructure.ProviderTypeVMwareVKS:  7,
	}
	if priority, ok := priorities[providerType]; ok {
		return priority
	}
	return 100
}

var _ build.TektonClientProvider = (*TektonClientProvider)(nil)
