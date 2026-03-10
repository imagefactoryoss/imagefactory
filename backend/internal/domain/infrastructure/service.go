package infrastructure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/metrics"
	"github.com/srikarm/image-factory/internal/infrastructure/tektoncompat"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

var (
	ErrProviderNotFound      = errors.New("infrastructure provider not found")
	ErrProviderExists        = errors.New("infrastructure provider already exists")
	ErrPermissionDenied      = errors.New("permission denied")
	ErrInvalidProviderType   = errors.New("invalid provider type")
	ErrInvalidProviderStatus = errors.New("invalid provider status")
	// Validation error returned when a provider's target namespace is invalid/missing
	ErrInvalidTargetNamespace         = errors.New("invalid target namespace")
	ErrTektonInstallerNotConfigured   = errors.New("tekton installer repository not configured")
	ErrTektonInstallerJobInProgress   = errors.New("tekton installer job already in progress for provider")
	ErrTektonInstallerJobNotFound     = errors.New("tekton installer job not found")
	ErrTektonInstallerJobNotRetryable = errors.New("tekton installer job is not retryable")
	ErrInvalidTektonInstallerRequest  = errors.New("invalid tekton installer request")
	ErrProviderPrepareNotConfigured   = errors.New("provider prepare repository not configured")
	ErrProviderPrepareRunInProgress   = errors.New("provider prepare run already in progress for provider")
	ErrProviderPrepareRunNotFound     = errors.New("provider prepare run not found")
	kubernetesNamespacePattern        = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

const tenantNamespaceTenantIDLabelKey = "imagefactory.io/tenant-id"

// Service defines the business logic for infrastructure provider management
type Service struct {
	repository                    Repository
	installerRepository           TektonInstallerJobRepository
	prepareRepository             ProviderPrepareRunRepository
	prepareSummaryRepository      ProviderPrepareSummaryRepository
	tenantNamespacePrepareRepo    ProviderTenantNamespacePrepareRepository
	eventPublisher                EventPublisher
	logger                        *zap.Logger
	connectorFactory              *connectors.ConnectorFactory
	tektonCoreConfigLookup        func(ctx context.Context) (*systemconfig.TektonCoreConfig, error)
	tektonTaskImagesConfigLookup  func(ctx context.Context) (*systemconfig.TektonTaskImagesConfig, error)
	runtimeServicesConfigLookup   func(ctx context.Context) (*systemconfig.RuntimeServicesConfig, error)
	prepareSummaryBatchDuration   metrics.OperationMetrics
	prepareSummaryBatchErrors     int64
	prepareSummaryProvidersTotal  int64
	prepareSummaryBatchesRepo     int64
	prepareSummaryBatchesFallback int64
	tenantPrepareAsyncTriggered   int64
	tenantPrepareAsyncSkipped     int64
	tenantPrepareAsyncFailures    int64
	tenantAssetDriftWatchDuration metrics.OperationMetrics
	tenantAssetDriftWatchTicks    int64
	tenantAssetDriftWatchFailures int64
	tenantAssetDriftCurrent       int64
	tenantAssetDriftStale         int64
	tenantAssetDriftUnknown       int64
	tenantAssetReconcileRequests  int64
	tenantAssetReconcileSuccess   int64
	tenantAssetReconcileFailures  int64
}

type providerTenantNamespacePrepareLister interface {
	ListTenantNamespacePreparesByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderTenantNamespacePrepare, error)
}

const (
	tenantAssetReconcilePolicyFullOnPrepare = "full_reconcile_on_prepare"
	tenantAssetReconcilePolicyAsyncTrigger  = "async_trigger_only"
	tenantAssetReconcilePolicyManualOnly    = "manual_only"
	tektonHistoryCleanupCronJobName         = "tekton-history-cleanup"
	trivyDBWarmupCronJobName                = "trivy-db-warmup"
	internalRegistryServiceName             = "image-factory-registry"
	internalRegistryPVCNameDefault          = "image-factory-registry-data"
	internalRegistryStorageHostPathDefault  = "/var/lib/image-factory/registry"
	internalRegistryHostPathTypeDefault     = "DirectoryOrCreate"
	internalRegistryPVCSizeDefault          = "20Gi"
)

type PrepareSummaryBatchMetricsSnapshot struct {
	BatchCount        int64   `json:"batch_count"`
	BatchTotalMs      int64   `json:"batch_total_ms"`
	BatchMinMs        int64   `json:"batch_min_ms"`
	BatchMaxMs        int64   `json:"batch_max_ms"`
	BatchAvgMs        float64 `json:"batch_avg_ms"`
	BatchErrors       int64   `json:"batch_errors"`
	ProvidersTotal    int64   `json:"providers_total"`
	RepositoryBatches int64   `json:"repository_batches"`
	FallbackBatches   int64   `json:"fallback_batches"`
}

type TenantPrepareAutomationMetricsSnapshot struct {
	AsyncTriggered int64 `json:"async_triggered"`
	AsyncSkipped   int64 `json:"async_skipped"`
	AsyncFailures  int64 `json:"async_failures"`
}

type TenantAssetDriftMetricsSnapshot struct {
	WatchTicksTotal           int64   `json:"watch_ticks_total"`
	WatchFailuresTotal        int64   `json:"watch_failures_total"`
	WatchCurrentNamespaces    int64   `json:"watch_current_namespaces"`
	WatchStaleNamespaces      int64   `json:"watch_stale_namespaces"`
	WatchUnknownNamespaces    int64   `json:"watch_unknown_namespaces"`
	WatchDurationCount        int64   `json:"watch_duration_count"`
	WatchDurationTotalMs      int64   `json:"watch_duration_total_ms"`
	WatchDurationMinMs        int64   `json:"watch_duration_min_ms"`
	WatchDurationMaxMs        int64   `json:"watch_duration_max_ms"`
	WatchDurationAvgMs        float64 `json:"watch_duration_avg_ms"`
	ReconcileRequestsTotal    int64   `json:"reconcile_requests_total"`
	ReconcileRequestsSuccess  int64   `json:"reconcile_requests_success_total"`
	ReconcileRequestsFailures int64   `json:"reconcile_requests_failures_total"`
}

type ProviderReadinessWatchTickResult struct {
	TotalProviders int `json:"total_providers"`
	Attempted      int `json:"attempted"`
	Succeeded      int `json:"succeeded"`
	Failed         int `json:"failed"`
	Skipped        int `json:"skipped"`
	Ready          int `json:"ready"`
	NotReady       int `json:"not_ready"`
}

type TenantAssetDriftWatchTickResult struct {
	TotalProviders  int `json:"total_providers"`
	Attempted       int `json:"attempted"`
	Succeeded       int `json:"succeeded"`
	Failed          int `json:"failed"`
	Skipped         int `json:"skipped"`
	TotalNamespaces int `json:"total_namespaces"`
	Current         int `json:"current"`
	Stale           int `json:"stale"`
	Unknown         int `json:"unknown"`
}

type TenantNamespaceReconcileResult struct {
	TenantID string `json:"tenant_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

type TenantNamespaceReconcileSummary struct {
	ProviderID      string                           `json:"provider_id"`
	Mode            string                           `json:"mode"`
	Targeted        int                              `json:"targeted"`
	Applied         int                              `json:"applied"`
	Failed          int                              `json:"failed"`
	Skipped         int                              `json:"skipped"`
	StaleOnlyFilter bool                             `json:"stale_only_filter"`
	Results         []TenantNamespaceReconcileResult `json:"results"`
}

// NewService creates a new infrastructure service
func NewService(repository Repository, eventPublisher EventPublisher, logger *zap.Logger) *Service {
	var installerRepository TektonInstallerJobRepository
	if repo, ok := repository.(TektonInstallerJobRepository); ok {
		installerRepository = repo
	}
	var prepareRepository ProviderPrepareRunRepository
	if repo, ok := repository.(ProviderPrepareRunRepository); ok {
		prepareRepository = repo
	}
	var prepareSummaryRepository ProviderPrepareSummaryRepository
	if repo, ok := repository.(ProviderPrepareSummaryRepository); ok {
		prepareSummaryRepository = repo
	}
	var tenantNamespacePrepareRepo ProviderTenantNamespacePrepareRepository
	if repo, ok := repository.(ProviderTenantNamespacePrepareRepository); ok {
		tenantNamespacePrepareRepo = repo
	}

	return &Service{
		repository:                 repository,
		installerRepository:        installerRepository,
		prepareRepository:          prepareRepository,
		prepareSummaryRepository:   prepareSummaryRepository,
		tenantNamespacePrepareRepo: tenantNamespacePrepareRepo,
		eventPublisher:             eventPublisher,
		logger:                     logger,
		connectorFactory:           connectors.NewConnectorFactory(logger),
	}
}

// SetTektonCoreConfigLookup provides a (typically global) system-config backed lookup for Tekton core defaults.
// This allows updating install source/URLs/assets path without changing provider configs or code.
func (s *Service) SetTektonCoreConfigLookup(lookup func(ctx context.Context) (*systemconfig.TektonCoreConfig, error)) {
	if s == nil {
		return
	}
	s.tektonCoreConfigLookup = lookup
}

// SetTektonTaskImagesConfigLookup provides a (typically global) system-config backed lookup for Tekton task image overrides.
func (s *Service) SetTektonTaskImagesConfigLookup(lookup func(ctx context.Context) (*systemconfig.TektonTaskImagesConfig, error)) {
	if s == nil {
		return
	}
	s.tektonTaskImagesConfigLookup = lookup
}

// SetRuntimeServicesConfigLookup provides a system-config backed lookup for runtime-services policy toggles.
func (s *Service) SetRuntimeServicesConfigLookup(lookup func(ctx context.Context) (*systemconfig.RuntimeServicesConfig, error)) {
	if s == nil {
		return
	}
	s.runtimeServicesConfigLookup = lookup
}

func (s *Service) GetPrepareSummaryBatchMetrics() PrepareSummaryBatchMetricsSnapshot {
	count, totalMs, minMs, maxMs, avgMs := s.prepareSummaryBatchDuration.GetStats()
	return PrepareSummaryBatchMetricsSnapshot{
		BatchCount:        count,
		BatchTotalMs:      totalMs,
		BatchMinMs:        minMs,
		BatchMaxMs:        maxMs,
		BatchAvgMs:        avgMs,
		BatchErrors:       atomic.LoadInt64(&s.prepareSummaryBatchErrors),
		ProvidersTotal:    atomic.LoadInt64(&s.prepareSummaryProvidersTotal),
		RepositoryBatches: atomic.LoadInt64(&s.prepareSummaryBatchesRepo),
		FallbackBatches:   atomic.LoadInt64(&s.prepareSummaryBatchesFallback),
	}
}

func (s *Service) GetTenantPrepareAutomationMetrics() TenantPrepareAutomationMetricsSnapshot {
	return TenantPrepareAutomationMetricsSnapshot{
		AsyncTriggered: atomic.LoadInt64(&s.tenantPrepareAsyncTriggered),
		AsyncSkipped:   atomic.LoadInt64(&s.tenantPrepareAsyncSkipped),
		AsyncFailures:  atomic.LoadInt64(&s.tenantPrepareAsyncFailures),
	}
}

func (s *Service) GetTenantAssetDriftMetrics() TenantAssetDriftMetricsSnapshot {
	count, totalMs, minMs, maxMs, avgMs := s.tenantAssetDriftWatchDuration.GetStats()
	return TenantAssetDriftMetricsSnapshot{
		WatchTicksTotal:           atomic.LoadInt64(&s.tenantAssetDriftWatchTicks),
		WatchFailuresTotal:        atomic.LoadInt64(&s.tenantAssetDriftWatchFailures),
		WatchCurrentNamespaces:    atomic.LoadInt64(&s.tenantAssetDriftCurrent),
		WatchStaleNamespaces:      atomic.LoadInt64(&s.tenantAssetDriftStale),
		WatchUnknownNamespaces:    atomic.LoadInt64(&s.tenantAssetDriftUnknown),
		WatchDurationCount:        count,
		WatchDurationTotalMs:      totalMs,
		WatchDurationMinMs:        minMs,
		WatchDurationMaxMs:        maxMs,
		WatchDurationAvgMs:        avgMs,
		ReconcileRequestsTotal:    atomic.LoadInt64(&s.tenantAssetReconcileRequests),
		ReconcileRequestsSuccess:  atomic.LoadInt64(&s.tenantAssetReconcileSuccess),
		ReconcileRequestsFailures: atomic.LoadInt64(&s.tenantAssetReconcileFailures),
	}
}

// EnsureTenantNamespaceReady provisions a per-tenant namespace for managed Kubernetes providers.
// It uses the provider bootstrap credentials (config.bootstrap_auth) to create namespace-scoped
// runtime identities and to install namespace-scoped Tekton assets (tasks/pipelines).
//
// This is intentionally separate from provider-level prepare because:
// - Tekton Tasks/Pipelines are namespace-scoped
// - Runtime RBAC must be namespace-scoped for least privilege
func (s *Service) EnsureTenantNamespaceReady(
	ctx context.Context,
	providerID uuid.UUID,
	tenantID uuid.UUID,
	requestedBy *uuid.UUID,
) (*ProviderTenantNamespacePrepare, error) {
	return s.ensureTenantNamespaceReady(ctx, providerID, tenantID, requestedBy, false)
}

func (s *Service) ensureTenantNamespaceReady(
	ctx context.Context,
	providerID uuid.UUID,
	tenantID uuid.UUID,
	requestedBy *uuid.UUID,
	forceReapply bool,
) (*ProviderTenantNamespacePrepare, error) {
	if s == nil {
		return nil, errors.New("infrastructure service is nil")
	}
	if s.tenantNamespacePrepareRepo == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	if providerID == uuid.Nil || tenantID == uuid.Nil {
		return nil, fmt.Errorf("provider_id and tenant_id are required")
	}

	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	if provider.ProviderType == "" {
		return nil, ErrInvalidProviderType
	}
	if normalizeBootstrapMode(provider.BootstrapMode) != "image_factory_managed" {
		return nil, fmt.Errorf("tenant namespace provisioning is only supported for bootstrap_mode=image_factory_managed")
	}
	if provider.Config == nil {
		return nil, fmt.Errorf("provider config is required")
	}

	namespace := fmt.Sprintf("image-factory-%s", tenantID.String()[:8])

	// Fast path: if we already have a succeeded row, don't re-apply assets/RBAC unless explicitly forced.
	existing, err := s.tenantNamespacePrepareRepo.GetTenantNamespacePrepare(ctx, providerID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant namespace prepare status: %w", err)
	}
	if existing != nil {
		switch existing.Status {
		case ProviderTenantNamespacePrepareSucceeded:
			if !forceReapply && hasUsableRuntimeAuthConfig(provider.Config) {
				return existing, nil
			}
		case ProviderTenantNamespacePrepareRunning, ProviderTenantNamespacePreparePending:
			// Another request kicked this off; caller should wait/poll.
			return existing, nil
		}
	}

	now := time.Now().UTC()
	prepare := &ProviderTenantNamespacePrepare{
		ID:               uuid.New(),
		ProviderID:       providerID,
		TenantID:         tenantID,
		Namespace:        namespace,
		RequestedBy:      requestedBy,
		Status:           ProviderTenantNamespacePrepareRunning,
		ResultSummary:    map[string]interface{}{},
		AssetDriftStatus: TenantAssetDriftStatusUnknown,
		StartedAt:        &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if existing != nil {
		prepare.ID = existing.ID
		prepare.CreatedAt = existing.CreatedAt
	}
	if err := s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare); err != nil {
		return nil, fmt.Errorf("failed to persist tenant namespace prepare status: %w", err)
	}

	fail := func(err error) (*ProviderTenantNamespacePrepare, error) {
		msg := err.Error()
		completedAt := time.Now().UTC()
		prepare.Status = ProviderTenantNamespacePrepareFailed
		prepare.ErrorMessage = &msg
		prepare.CompletedAt = &completedAt
		prepare.UpdatedAt = completedAt
		_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
		return prepare, err
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfigForContext(provider.Config, connectors.AuthContextBootstrap)
	if err != nil {
		return fail(fmt.Errorf("invalid provider bootstrap auth config: %w", err))
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fail(fmt.Errorf("failed to create kubernetes client: %w", err))
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fail(fmt.Errorf("failed to create dynamic client: %w", err))
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return fail(fmt.Errorf("failed to create discovery client: %w", err))
	}

	systemNamespace := resolveProviderSystemNamespace(provider)
	runtimeProvisionResult, err := s.ensureManagedRuntimeIdentityAndPersistConfig(
		ctx,
		provider,
		k8sClient,
		systemNamespace,
		restConfig.Host,
		restConfig.TLSClientConfig.CAData,
		false,
	)
	if err != nil {
		return fail(fmt.Errorf("failed to ensure managed runtime identity: %w", err))
	}
	prepare.ResultSummary["runtime_auth"] = runtimeProvisionResult
	_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)

	// Ensure Tekton APIs exist (and install core if configured to do so).
	presentResources, installDetails, err := s.ensureTektonAPIsOrInstall(ctx, provider, namespace, dynamicClient, discoveryClient)
	if err != nil {
		if installDetails != nil {
			return fail(fmt.Errorf("tekton API preflight failed after install attempt (%v): %w", installDetails, err))
		}
		return fail(fmt.Errorf("tekton API preflight failed: %w", err))
	}
	prepare.ResultSummary["tekton_resources"] = presentResources
	if installDetails != nil {
		prepare.ResultSummary["core_install"] = installDetails
		_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
		if waitErr := s.waitForTektonWebhookReady(ctx, k8sClient, provider, namespace, 90*time.Second); waitErr != nil {
			return fail(fmt.Errorf("tekton webhook readiness check failed: %w", waitErr))
		}
		prepare.ResultSummary["core_webhook_ready"] = true
		_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
	}

	// Create per-tenant namespace using bootstrap credentials.
	tenantNamespaceLabels := map[string]string{
		tenantNamespaceTenantIDLabelKey: tenantID.String(),
	}
	if err := ensureNamespaceWithLabels(ctx, k8sClient, namespace, tenantNamespaceLabels); err != nil {
		return fail(fmt.Errorf("failed to ensure tenant namespace %s: %w", namespace, err))
	}
	prepare.ResultSummary["namespace_labels"] = tenantNamespaceLabels
	_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)

	// Apply runtime RBAC inside tenant namespace (namespace-scoped), binding to the provider runtime ServiceAccount.
	// The runtime ServiceAccount is expected to live in the provider system namespace.
	rbacYAML := runtimeTenantRBACYAML(systemNamespace, namespace)
	rbacObjects, err := parseUnstructuredYAMLBytes([]byte(rbacYAML), "runtime_rbac")
	if err != nil {
		return fail(fmt.Errorf("failed to parse runtime RBAC manifests: %w", err))
	}
	appliedRBAC, err := applyUnstructuredObjects(ctx, dynamicClient, discoveryClient, rbacObjects, namespace, "image-factory-tenant-rbac")
	if err != nil {
		return fail(fmt.Errorf("failed to apply runtime RBAC manifests: %w", err))
	}
	prepare.ResultSummary["rbac_applied_objects"] = appliedRBAC
	_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)

	// Apply Tekton tasks/pipelines to the tenant namespace.
	assetRoot, err := s.resolveTektonAssetRootDir(ctx)
	if err != nil {
		return fail(err)
	}
	profileVersion := resolveTektonAssetVersion(provider, "")
	resourceFiles, err := loadKustomizationResourceFilesForProfile(assetRoot, profileVersion)
	if err != nil {
		return fail(err)
	}
	if len(resourceFiles) == 0 {
		return fail(fmt.Errorf("tekton kustomization has no resources under %s", assetRoot))
	}
	if desiredVersion, versionErr := calculateTektonAssetsVersionWithOverrides(resourceFiles, s.getTektonTaskImagesConfig(ctx)); versionErr != nil {
		s.logger.Warn("Failed to calculate tenant asset desired version",
			zap.Error(versionErr),
			zap.String("provider_id", providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
	} else if desiredVersion != "" {
		prepare.DesiredAssetVersion = &desiredVersion
	}

	appliedAssets := make([]string, 0, 64)
	cleanupPolicy := s.tektonHistoryCleanupConfig(ctx)
	tektonTaskImageConfig := s.getTektonTaskImagesConfig(ctx)
	internalRegistryStorageProfile := s.resolveInternalRegistryStorageProfile(ctx)
	for _, resourceFile := range resourceFiles {
		if tektonResourceScope(resourceFile) == tektonResourceScopeShared {
			continue
		}
		objects, parseErr := parseUnstructuredYAMLFile(resourceFile)
		if parseErr != nil {
			return fail(fmt.Errorf("failed to parse manifest %s: %w", resourceFile, parseErr))
		}
		applyTektonCleanupPolicyToObjects(objects, cleanupPolicy)
		applyTektonTaskImageOverrides(objects, tektonTaskImageConfig)
		objects = applyStorageProfilesToObjects(objects, internalRegistryStorageProfile)
		applied, applyErr := applyUnstructuredObjects(ctx, dynamicClient, discoveryClient, objects, namespace, "image-factory-tenant-tekton-assets")
		if applyErr != nil {
			return fail(fmt.Errorf("failed to apply manifests from %s: %w", resourceFile, applyErr))
		}
		appliedAssets = append(appliedAssets, applied...)
	}
	prepare.ResultSummary["asset_root"] = assetRoot
	prepare.ResultSummary["resource_files"] = resourceFiles
	prepare.ResultSummary["assets_applied_objects"] = appliedAssets
	if aliasResult, aliasErr := s.ensureTenantInternalRegistryAliasService(ctx, k8sClient, namespace, systemNamespace); aliasErr != nil {
		prepare.ResultSummary["registry_alias"] = map[string]interface{}{
			"status":  "error",
			"message": aliasErr.Error(),
		}
		s.logger.Warn("Failed to ensure tenant internal registry alias service",
			zap.String("provider_id", providerID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.String("tenant_namespace", namespace),
			zap.String("system_namespace", systemNamespace),
			zap.Error(aliasErr),
		)
	} else if aliasResult != nil {
		prepare.ResultSummary["registry_alias"] = aliasResult
	}

	if warmupResult, warmupErr := s.triggerTrivyDBWarmupIfNeeded(ctx, k8sClient, systemNamespace); warmupErr != nil {
		prepare.ResultSummary["trivy_db_warmup"] = map[string]interface{}{
			"status":  "error",
			"message": warmupErr.Error(),
		}
		s.logger.Warn("Failed to trigger trivy DB warmup after tenant namespace prepare",
			zap.String("provider_id", providerID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.String("namespace", systemNamespace),
			zap.Error(warmupErr),
		)
	} else if warmupResult != nil {
		prepare.ResultSummary["trivy_db_warmup"] = warmupResult
	}
	if prepare.DesiredAssetVersion != nil && strings.TrimSpace(*prepare.DesiredAssetVersion) != "" {
		installed := strings.TrimSpace(*prepare.DesiredAssetVersion)
		prepare.InstalledAssetVersion = &installed
	}
	prepare.AssetDriftStatus = computeTenantAssetDriftStatus(prepare.DesiredAssetVersion, prepare.InstalledAssetVersion)

	completedAt := time.Now().UTC()
	prepare.Status = ProviderTenantNamespacePrepareSucceeded
	prepare.CompletedAt = &completedAt
	prepare.UpdatedAt = completedAt
	if err := s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare); err != nil {
		return prepare, fmt.Errorf("failed to persist tenant namespace prepare status: %w", err)
	}
	return prepare, nil
}

// DeprovisionTenantNamespace tears down a managed tenant namespace provisioned by Image Factory.
// Safety guard: namespace deletion is allowed only when the namespace is labeled for the tenant
// (`imagefactory.io/tenant-id=<tenant-id>`) or is explicitly marked `managedBy=image-factory`.
func (s *Service) DeprovisionTenantNamespace(
	ctx context.Context,
	providerID uuid.UUID,
	tenantID uuid.UUID,
	requestedBy *uuid.UUID,
) (*ProviderTenantNamespacePrepare, error) {
	if s == nil {
		return nil, errors.New("infrastructure service is nil")
	}
	if s.tenantNamespacePrepareRepo == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	if providerID == uuid.Nil || tenantID == uuid.Nil {
		return nil, fmt.Errorf("provider_id and tenant_id are required")
	}

	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	if normalizeBootstrapMode(provider.BootstrapMode) != "image_factory_managed" {
		return nil, fmt.Errorf("tenant namespace deprovision is only supported for bootstrap_mode=image_factory_managed")
	}
	if provider.Config == nil {
		return nil, fmt.Errorf("provider config is required")
	}

	namespace := fmt.Sprintf("image-factory-%s", tenantID.String()[:8])
	existing, err := s.tenantNamespacePrepareRepo.GetTenantNamespacePrepare(ctx, providerID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant namespace prepare status: %w", err)
	}

	now := time.Now().UTC()
	prepare := &ProviderTenantNamespacePrepare{
		ID:               uuid.New(),
		ProviderID:       providerID,
		TenantID:         tenantID,
		Namespace:        namespace,
		RequestedBy:      requestedBy,
		Status:           ProviderTenantNamespacePrepareRunning,
		ResultSummary:    map[string]interface{}{},
		AssetDriftStatus: TenantAssetDriftStatusUnknown,
		StartedAt:        &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if existing != nil {
		prepare.ID = existing.ID
		prepare.CreatedAt = existing.CreatedAt
		prepare.DesiredAssetVersion = existing.DesiredAssetVersion
	}
	if err := s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare); err != nil {
		return nil, fmt.Errorf("failed to persist tenant namespace prepare status: %w", err)
	}

	fail := func(err error) (*ProviderTenantNamespacePrepare, error) {
		msg := err.Error()
		completedAt := time.Now().UTC()
		prepare.Status = ProviderTenantNamespacePrepareFailed
		prepare.ErrorMessage = &msg
		prepare.CompletedAt = &completedAt
		prepare.UpdatedAt = completedAt
		_ = s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
		return prepare, err
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfigForContext(provider.Config, connectors.AuthContextBootstrap)
	if err != nil {
		return fail(fmt.Errorf("invalid provider bootstrap auth config: %w", err))
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fail(fmt.Errorf("failed to create kubernetes client: %w", err))
	}

	namespaceObj, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			prepare.ResultSummary["deprovision"] = map[string]interface{}{
				"namespace":           namespace,
				"namespace_deleted":   false,
				"namespace_not_found": true,
				"reason":              "already absent",
			}
			completedAt := time.Now().UTC()
			prepare.Status = ProviderTenantNamespacePrepareSucceeded
			prepare.ErrorMessage = nil
			prepare.CompletedAt = &completedAt
			prepare.UpdatedAt = completedAt
			prepare.InstalledAssetVersion = nil
			prepare.AssetDriftStatus = TenantAssetDriftStatusUnknown
			if err := s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare); err != nil {
				return prepare, fmt.Errorf("failed to persist tenant namespace deprovision status: %w", err)
			}
			return prepare, nil
		}
		return fail(fmt.Errorf("failed to get tenant namespace %s: %w", namespace, err))
	}

	labels := namespaceObj.GetLabels()
	tenantLabel := strings.TrimSpace(labels[tenantNamespaceTenantIDLabelKey])
	managedBy := strings.TrimSpace(labels["managedBy"])
	if tenantLabel != "" && tenantLabel != tenantID.String() {
		return fail(fmt.Errorf("safety check failed: namespace %s is labeled for tenant %s", namespace, tenantLabel))
	}
	if tenantLabel == "" && managedBy != "image-factory" {
		return fail(fmt.Errorf("safety check failed: namespace %s is not managed by image-factory", namespace))
	}

	if err := k8sClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fail(fmt.Errorf("failed to delete tenant namespace %s: %w", namespace, err))
	}
	if waitErr := waitForNamespaceDeletion(ctx, k8sClient, namespace, 2*time.Minute); waitErr != nil {
		return fail(fmt.Errorf("failed waiting for namespace %s deletion: %w", namespace, waitErr))
	}

	prepare.ResultSummary["deprovision"] = map[string]interface{}{
		"namespace":             namespace,
		"namespace_deleted":     true,
		"tenant_label":          tenantLabel,
		"managed_by":            managedBy,
		"cleanup_mode":          "namespace_delete",
		"runtime_auth_retained": true,
	}
	completedAt := time.Now().UTC()
	prepare.Status = ProviderTenantNamespacePrepareSucceeded
	prepare.ErrorMessage = nil
	prepare.CompletedAt = &completedAt
	prepare.UpdatedAt = completedAt
	prepare.InstalledAssetVersion = nil
	prepare.AssetDriftStatus = TenantAssetDriftStatusUnknown
	if err := s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare); err != nil {
		return prepare, fmt.Errorf("failed to persist tenant namespace deprovision status: %w", err)
	}

	return prepare, nil
}

func (s *Service) GetTenantNamespacePrepareStatus(ctx context.Context, providerID, tenantID uuid.UUID) (*ProviderTenantNamespacePrepare, error) {
	if s == nil || s.tenantNamespacePrepareRepo == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	if providerID == uuid.Nil || tenantID == uuid.Nil {
		return nil, fmt.Errorf("provider_id and tenant_id are required")
	}
	prepare, err := s.tenantNamespacePrepareRepo.GetTenantNamespacePrepare(ctx, providerID, tenantID)
	if err != nil || prepare == nil {
		return prepare, err
	}
	if refreshErr := s.refreshTenantNamespacePrepareAssetDrift(ctx, prepare); refreshErr != nil {
		s.logger.Warn("Failed to refresh tenant namespace prepare asset drift status",
			zap.Error(refreshErr),
			zap.String("provider_id", providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
	}
	return prepare, nil
}

// TriggerTenantNamespacePrepareAsync starts managed tenant namespace provisioning in the background.
// It is intentionally non-blocking for caller paths like tenant create and provider-access grant.
func (s *Service) TriggerTenantNamespacePrepareAsync(ctx context.Context, providerID, tenantID uuid.UUID, requestedBy *uuid.UUID) error {
	if s == nil {
		return errors.New("infrastructure service is nil")
	}
	if providerID == uuid.Nil || tenantID == uuid.Nil {
		return fmt.Errorf("provider_id and tenant_id are required")
	}
	if s.tenantNamespacePrepareRepo == nil {
		atomic.AddInt64(&s.tenantPrepareAsyncFailures, 1)
		return ErrProviderPrepareNotConfigured
	}

	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		atomic.AddInt64(&s.tenantPrepareAsyncFailures, 1)
		return fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		atomic.AddInt64(&s.tenantPrepareAsyncFailures, 1)
		return ErrProviderNotFound
	}
	if !isK8sCapableProviderType(provider.ProviderType) {
		atomic.AddInt64(&s.tenantPrepareAsyncSkipped, 1)
		s.logger.Debug("Skipping async tenant namespace prepare for non-kubernetes provider",
			zap.String("provider_id", provider.ID.String()),
			zap.String("provider_type", string(provider.ProviderType)),
			zap.String("tenant_id", tenantID.String()),
		)
		return nil
	}
	if normalizeBootstrapMode(provider.BootstrapMode) != "image_factory_managed" {
		atomic.AddInt64(&s.tenantPrepareAsyncSkipped, 1)
		s.logger.Debug("Skipping async tenant namespace prepare for self-managed provider",
			zap.String("provider_id", provider.ID.String()),
			zap.String("bootstrap_mode", provider.BootstrapMode),
			zap.String("tenant_id", tenantID.String()),
		)
		return nil
	}

	atomic.AddInt64(&s.tenantPrepareAsyncTriggered, 1)
	s.logger.Info("Triggering async tenant namespace prepare",
		zap.String("provider_id", provider.ID.String()),
		zap.String("tenant_id", tenantID.String()),
	)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		if _, prepErr := s.ensureTenantNamespaceReady(bgCtx, providerID, tenantID, requestedBy, true); prepErr != nil {
			atomic.AddInt64(&s.tenantPrepareAsyncFailures, 1)
			s.logger.Warn("Async tenant namespace prepare failed",
				zap.String("provider_id", providerID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(prepErr),
			)
		}
	}()

	return nil
}

// TriggerTenantNamespacePrepareForNewTenantAsync schedules prepares for global managed Kubernetes providers.
func (s *Service) TriggerTenantNamespacePrepareForNewTenantAsync(ctx context.Context, tenantID uuid.UUID, requestedBy *uuid.UUID) (int, error) {
	if s == nil {
		return 0, errors.New("infrastructure service is nil")
	}
	if tenantID == uuid.Nil {
		return 0, fmt.Errorf("tenant_id is required")
	}

	result, err := s.repository.FindProvidersAll(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to list providers: %w", err)
	}

	triggered := 0
	for i := range result.Providers {
		provider := result.Providers[i]
		if !provider.IsGlobal {
			continue
		}
		if !isK8sCapableProviderType(provider.ProviderType) {
			atomic.AddInt64(&s.tenantPrepareAsyncSkipped, 1)
			continue
		}
		if normalizeBootstrapMode(provider.BootstrapMode) != "image_factory_managed" {
			atomic.AddInt64(&s.tenantPrepareAsyncSkipped, 1)
			continue
		}
		if err := s.TriggerTenantNamespacePrepareAsync(ctx, provider.ID, tenantID, requestedBy); err != nil {
			s.logger.Warn("Failed to trigger tenant namespace prepare for new tenant",
				zap.String("provider_id", provider.ID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			continue
		}
		triggered++
	}

	return triggered, nil
}

// RunProviderReadinessWatchTick reconciles readiness/health for Kubernetes-capable providers.
// It is intended for scheduled background execution.
func (s *Service) RunProviderReadinessWatchTick(ctx context.Context, pageSize int) (*ProviderReadinessWatchTickResult, error) {
	if s == nil {
		return nil, errors.New("infrastructure service is nil")
	}
	if pageSize <= 0 {
		pageSize = 200
	}

	out := &ProviderReadinessWatchTickResult{}
	page := 1
	for {
		result, err := s.repository.FindProvidersAll(ctx, &ListProvidersOptions{
			Page:  page,
			Limit: pageSize,
		})
		if err != nil {
			return out, fmt.Errorf("failed to list providers for readiness watch: %w", err)
		}
		if result == nil || len(result.Providers) == 0 {
			break
		}

		for i := range result.Providers {
			provider := result.Providers[i]
			out.TotalProviders++
			if !isK8sCapableProviderType(provider.ProviderType) {
				out.Skipped++
				continue
			}

			out.Attempted++
			readinessStatus, _, refreshErr := s.refreshProviderReadinessAfterInstall(ctx, &TektonInstallerJob{
				ProviderID: provider.ID,
				TenantID:   provider.TenantID,
			})
			if refreshErr != nil {
				out.Failed++
				if s.logger != nil {
					s.logger.Warn("Provider readiness watch tick failed for provider",
						zap.String("provider_id", provider.ID.String()),
						zap.String("provider_type", string(provider.ProviderType)),
						zap.Error(refreshErr),
					)
				}
				continue
			}
			out.Succeeded++
			switch readinessStatus {
			case "ready":
				out.Ready++
			default:
				out.NotReady++
			}
		}

		if result.TotalPages <= page || result.TotalPages == 0 {
			break
		}
		page++
	}

	return out, nil
}

// RunTenantAssetDriftWatchTick refreshes tenant namespace asset drift status for managed Kubernetes providers.
func (s *Service) RunTenantAssetDriftWatchTick(ctx context.Context, pageSize int) (*TenantAssetDriftWatchTickResult, error) {
	if s == nil {
		return nil, errors.New("infrastructure service is nil")
	}
	if s.tenantNamespacePrepareRepo == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	lister, ok := s.repository.(providerTenantNamespacePrepareLister)
	if !ok {
		return nil, ErrProviderPrepareNotConfigured
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	start := time.Now()
	atomic.AddInt64(&s.tenantAssetDriftWatchTicks, 1)
	defer func() {
		s.tenantAssetDriftWatchDuration.RecordOperation(time.Since(start).Milliseconds())
	}()

	out := &TenantAssetDriftWatchTickResult{}
	page := 1
	for {
		result, err := s.repository.FindProvidersAll(ctx, &ListProvidersOptions{
			Page:  page,
			Limit: pageSize,
		})
		if err != nil {
			return out, fmt.Errorf("failed to list providers for tenant asset drift watch: %w", err)
		}
		if result == nil || len(result.Providers) == 0 {
			break
		}

		for i := range result.Providers {
			provider := result.Providers[i]
			out.TotalProviders++
			if !isK8sCapableProviderType(provider.ProviderType) {
				out.Skipped++
				continue
			}
			if normalizeBootstrapMode(provider.BootstrapMode) != "image_factory_managed" {
				out.Skipped++
				continue
			}

			tenantPrepares, listErr := lister.ListTenantNamespacePreparesByProvider(ctx, provider.ID)
			if listErr != nil {
				out.Attempted++
				out.Failed++
				if s.logger != nil {
					s.logger.Warn("Tenant asset drift watch failed to list tenant namespace prepares",
						zap.String("provider_id", provider.ID.String()),
						zap.Error(listErr),
					)
				}
				continue
			}
			if len(tenantPrepares) == 0 {
				out.Skipped++
				continue
			}

			out.Attempted++
			for _, prepare := range tenantPrepares {
				if prepare == nil {
					continue
				}
				out.TotalNamespaces++
				if refreshErr := s.refreshTenantNamespacePrepareAssetDrift(ctx, prepare); refreshErr != nil {
					out.Failed++
					if s.logger != nil {
						s.logger.Warn("Tenant asset drift watch failed for tenant namespace",
							zap.String("provider_id", provider.ID.String()),
							zap.String("tenant_id", prepare.TenantID.String()),
							zap.String("namespace", prepare.Namespace),
							zap.Error(refreshErr),
						)
					}
					continue
				}

				out.Succeeded++
				switch prepare.AssetDriftStatus {
				case TenantAssetDriftStatusCurrent:
					out.Current++
				case TenantAssetDriftStatusStale:
					out.Stale++
				default:
					out.Unknown++
				}
			}
		}

		if result.TotalPages <= page || result.TotalPages == 0 {
			break
		}
		page++
	}
	atomic.StoreInt64(&s.tenantAssetDriftCurrent, int64(out.Current))
	atomic.StoreInt64(&s.tenantAssetDriftStale, int64(out.Stale))
	atomic.StoreInt64(&s.tenantAssetDriftUnknown, int64(out.Unknown))
	if out.Failed > 0 {
		atomic.AddInt64(&s.tenantAssetDriftWatchFailures, int64(out.Failed))
	}
	return out, nil
}

func (s *Service) ReconcileStaleTenantNamespaces(ctx context.Context, providerID uuid.UUID, requestedBy *uuid.UUID) (*TenantNamespaceReconcileSummary, error) {
	return s.reconcileTenantNamespaces(ctx, providerID, nil, requestedBy, true)
}

func (s *Service) ReconcileSelectedTenantNamespaces(ctx context.Context, providerID uuid.UUID, tenantIDs []uuid.UUID, requestedBy *uuid.UUID) (*TenantNamespaceReconcileSummary, error) {
	return s.reconcileTenantNamespaces(ctx, providerID, tenantIDs, requestedBy, false)
}

func (s *Service) reconcileTenantNamespaces(
	ctx context.Context,
	providerID uuid.UUID,
	tenantIDs []uuid.UUID,
	requestedBy *uuid.UUID,
	staleOnly bool,
) (*TenantNamespaceReconcileSummary, error) {
	if s == nil {
		return nil, errors.New("infrastructure service is nil")
	}
	if s.tenantNamespacePrepareRepo == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	atomic.AddInt64(&s.tenantAssetReconcileRequests, 1)
	if providerID == uuid.Nil {
		atomic.AddInt64(&s.tenantAssetReconcileFailures, 1)
		return nil, fmt.Errorf("provider_id is required")
	}
	lister, ok := s.repository.(providerTenantNamespacePrepareLister)
	if !ok {
		return nil, ErrProviderPrepareNotConfigured
	}

	prepareRows, err := lister.ListTenantNamespacePreparesByProvider(ctx, providerID)
	if err != nil {
		atomic.AddInt64(&s.tenantAssetReconcileFailures, 1)
		return nil, fmt.Errorf("failed to list tenant namespace prepares: %w", err)
	}

	mode := "selected"
	if staleOnly {
		mode = "stale_only"
	}
	summary := &TenantNamespaceReconcileSummary{
		ProviderID:      providerID.String(),
		Mode:            mode,
		StaleOnlyFilter: staleOnly,
		Results:         make([]TenantNamespaceReconcileResult, 0),
	}

	targetSet := make(map[uuid.UUID]struct{})
	if staleOnly {
		for _, prep := range prepareRows {
			if prep == nil || prep.TenantID == uuid.Nil {
				continue
			}
			_ = s.refreshTenantNamespacePrepareAssetDrift(ctx, prep)
			if prep.AssetDriftStatus != TenantAssetDriftStatusStale {
				continue
			}
			targetSet[prep.TenantID] = struct{}{}
		}
	} else {
		for _, tenantID := range tenantIDs {
			if tenantID != uuid.Nil {
				targetSet[tenantID] = struct{}{}
			}
		}
	}

	targetList := make([]uuid.UUID, 0, len(targetSet))
	for tenantID := range targetSet {
		targetList = append(targetList, tenantID)
	}
	sort.Slice(targetList, func(i, j int) bool {
		return targetList[i].String() < targetList[j].String()
	})
	summary.Targeted = len(targetList)

	for _, tenantID := range targetList {
		if _, prepErr := s.ensureTenantNamespaceReady(ctx, providerID, tenantID, requestedBy, true); prepErr != nil {
			summary.Failed++
			summary.Results = append(summary.Results, TenantNamespaceReconcileResult{
				TenantID: tenantID.String(),
				Status:   "failed",
				Message:  prepErr.Error(),
			})
			continue
		}
		summary.Applied++
		summary.Results = append(summary.Results, TenantNamespaceReconcileResult{
			TenantID: tenantID.String(),
			Status:   "applied",
		})
	}

	if summary.Targeted == 0 {
		atomic.AddInt64(&s.tenantAssetReconcileSuccess, 1)
		return summary, nil
	}
	if summary.Failed > 0 {
		atomic.AddInt64(&s.tenantAssetReconcileFailures, 1)
		return summary, fmt.Errorf("failed tenant namespace reconcile for %d tenant(s)", summary.Failed)
	}
	atomic.AddInt64(&s.tenantAssetReconcileSuccess, 1)
	return summary, nil
}

func (s *Service) triggerTenantNamespacePrepareForProviderTenants(
	ctx context.Context,
	providerID uuid.UUID,
	runTenantID uuid.UUID,
	requestedBy *uuid.UUID,
) (map[string]interface{}, error) {
	if providerID == uuid.Nil {
		return nil, fmt.Errorf("provider_id is required")
	}
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load provider for tenant namespace catch-up: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	targetTenants := make(map[uuid.UUID]struct{})
	if runTenantID != uuid.Nil {
		targetTenants[runTenantID] = struct{}{}
	}
	// Tenant-scoped providers implicitly belong to their owner tenant even when
	// no explicit provider permission row exists yet.
	if !provider.IsGlobal && provider.TenantID != uuid.Nil {
		targetTenants[provider.TenantID] = struct{}{}
	}

	perms, err := s.repository.FindPermissionsByProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider permissions: %w", err)
	}
	for _, perm := range perms {
		if perm == nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(perm.Permission)) != "infrastructure:select" {
			continue
		}
		if perm.TenantID == uuid.Nil {
			continue
		}
		targetTenants[perm.TenantID] = struct{}{}
	}

	if lister, ok := s.repository.(providerTenantNamespacePrepareLister); ok {
		existingPrepares, listErr := lister.ListTenantNamespacePreparesByProvider(ctx, providerID)
		if listErr != nil {
			return nil, fmt.Errorf("failed to list existing tenant namespace prepares: %w", listErr)
		}
		for _, prep := range existingPrepares {
			if prep == nil || prep.TenantID == uuid.Nil {
				continue
			}
			targetTenants[prep.TenantID] = struct{}{}
		}
	}

	tenantIDs := make([]uuid.UUID, 0, len(targetTenants))
	for tenantID := range targetTenants {
		tenantIDs = append(tenantIDs, tenantID)
	}
	sort.Slice(tenantIDs, func(i, j int) bool {
		return tenantIDs[i].String() < tenantIDs[j].String()
	})

	policy := s.tenantAssetReconcilePolicy(ctx)

	if policy == tenantAssetReconcilePolicyManualOnly {
		summary := map[string]interface{}{
			"policy":           policy,
			"tenants_targeted": len(tenantIDs),
			"tenants_applied":  0,
			"tenants_failed":   0,
			"tenant_ids":       tenantIDs,
			"action":           "manual_only",
		}
		return summary, nil
	}

	if policy == tenantAssetReconcilePolicyAsyncTrigger {
		queued := 0
		failures := make([]string, 0)
		for _, tenantID := range tenantIDs {
			if err := s.TriggerTenantNamespacePrepareAsync(ctx, providerID, tenantID, requestedBy); err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", tenantID.String(), err))
				continue
			}
			queued++
		}
		summary := map[string]interface{}{
			"policy":           policy,
			"tenants_targeted": len(tenantIDs),
			"tenants_queued":   queued,
			"tenants_failed":   len(failures),
			"tenant_ids":       tenantIDs,
		}
		if len(failures) > 0 {
			summary["failures"] = failures
			return summary, fmt.Errorf("failed to queue tenant namespace prepare for %d tenant(s)", len(failures))
		}
		return summary, nil
	}

	applied := 0
	failures := make([]string, 0)
	for _, tenantID := range tenantIDs {
		if _, prepErr := s.ensureTenantNamespaceReady(ctx, providerID, tenantID, requestedBy, true); prepErr != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", tenantID.String(), prepErr))
			continue
		}
		applied++
	}

	summary := map[string]interface{}{
		"policy":           policy,
		"tenants_targeted": len(tenantIDs),
		"tenants_applied":  applied,
		"tenants_failed":   len(failures),
		"tenant_ids":       tenantIDs,
	}
	if len(failures) > 0 {
		summary["failures"] = failures
		return summary, fmt.Errorf("failed tenant namespace prepare for %d tenant(s)", len(failures))
	}
	return summary, nil
}

func (s *Service) tenantAssetReconcilePolicy(ctx context.Context) string {
	defaultPolicy := strings.TrimSpace(strings.ToLower(os.Getenv("IF_TENANT_ASSET_RECONCILE_POLICY")))
	switch defaultPolicy {
	case tenantAssetReconcilePolicyManualOnly, tenantAssetReconcilePolicyAsyncTrigger, tenantAssetReconcilePolicyFullOnPrepare:
	default:
		defaultPolicy = tenantAssetReconcilePolicyFullOnPrepare
	}

	if s == nil || s.runtimeServicesConfigLookup == nil {
		return defaultPolicy
	}
	cfg, err := s.runtimeServicesConfigLookup(ctx)
	if err != nil || cfg == nil {
		return defaultPolicy
	}
	configPolicy := strings.TrimSpace(strings.ToLower(cfg.TenantAssetReconcilePolicy))
	switch configPolicy {
	case tenantAssetReconcilePolicyManualOnly, tenantAssetReconcilePolicyAsyncTrigger, tenantAssetReconcilePolicyFullOnPrepare:
		return configPolicy
	default:
		return defaultPolicy
	}
}

type tektonHistoryCleanupPolicy struct {
	Enabled          bool
	Schedule         string
	KeepPipelineRuns int
	KeepTaskRuns     int
	KeepPods         int
}

func (s *Service) tektonHistoryCleanupConfig(ctx context.Context) tektonHistoryCleanupPolicy {
	policy := tektonHistoryCleanupPolicy{
		Enabled:          true,
		Schedule:         "30 2 * * *",
		KeepPipelineRuns: 120,
		KeepTaskRuns:     240,
		KeepPods:         240,
	}
	if s == nil || s.runtimeServicesConfigLookup == nil {
		return policy
	}

	cfg, err := s.runtimeServicesConfigLookup(ctx)
	if err != nil || cfg == nil {
		return policy
	}
	if cfg.TektonHistoryCleanupEnabled != nil {
		policy.Enabled = *cfg.TektonHistoryCleanupEnabled
	}
	if schedule := strings.TrimSpace(cfg.TektonHistoryCleanupSchedule); schedule != "" {
		policy.Schedule = schedule
	}
	if cfg.TektonHistoryCleanupKeepPipelineRuns > 0 {
		policy.KeepPipelineRuns = cfg.TektonHistoryCleanupKeepPipelineRuns
	}
	if cfg.TektonHistoryCleanupKeepTaskRuns > 0 {
		policy.KeepTaskRuns = cfg.TektonHistoryCleanupKeepTaskRuns
	}
	if cfg.TektonHistoryCleanupKeepPods > 0 {
		policy.KeepPods = cfg.TektonHistoryCleanupKeepPods
	}
	return policy
}

type tektonAssetScope string

const (
	tektonResourceScopeTenant tektonAssetScope = "tenant"
	tektonResourceScopeShared tektonAssetScope = "shared"
)

func tektonResourceScope(resourceFile string) tektonAssetScope {
	file := filepath.ToSlash(strings.TrimSpace(resourceFile))
	if strings.HasSuffix(file, "/jobs/v1/internal-registry-pvc.yaml") ||
		strings.HasSuffix(file, "/jobs/v1/internal-registry-service.yaml") ||
		strings.HasSuffix(file, "/jobs/v1/internal-registry-deployment.yaml") ||
		strings.HasSuffix(file, "/jobs/v1/trivy-db-warmup-cronjob.yaml") {
		return tektonResourceScopeShared
	}
	return tektonResourceScopeTenant
}

func (s *Service) ensureTenantInternalRegistryAliasService(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	tenantNamespace string,
	systemNamespace string,
) (map[string]interface{}, error) {
	tenantNamespace = strings.TrimSpace(tenantNamespace)
	systemNamespace = strings.TrimSpace(systemNamespace)
	if k8sClient == nil || tenantNamespace == "" || systemNamespace == "" {
		return map[string]interface{}{
			"status": "skipped",
			"reason": "missing_inputs",
		}, nil
	}
	if tenantNamespace == systemNamespace {
		return map[string]interface{}{
			"status": "skipped",
			"reason": "same_namespace",
		}, nil
	}

	desiredExternalName := fmt.Sprintf("%s.%s.svc.cluster.local", internalRegistryServiceName, systemNamespace)
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      internalRegistryServiceName,
			Namespace: tenantNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":         internalRegistryServiceName,
				"app.kubernetes.io/part-of":      "image-factory",
				"imagefactory.io/registry-alias": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: desiredExternalName,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 5000,
				},
			},
		},
	}

	current, err := k8sClient.CoreV1().Services(tenantNamespace).Get(ctx, internalRegistryServiceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			created, createErr := k8sClient.CoreV1().Services(tenantNamespace).Create(ctx, desired, metav1.CreateOptions{})
			if createErr != nil {
				return nil, fmt.Errorf("failed to create registry alias service: %w", createErr)
			}
			return map[string]interface{}{
				"status":        "created",
				"service":       created.Name,
				"external_name": desiredExternalName,
			}, nil
		}
		return nil, fmt.Errorf("failed to load registry alias service: %w", err)
	}

	if current.Spec.Type == corev1.ServiceTypeExternalName &&
		strings.EqualFold(strings.TrimSpace(current.Spec.ExternalName), desiredExternalName) {
		return map[string]interface{}{
			"status":        "unchanged",
			"service":       current.Name,
			"external_name": desiredExternalName,
		}, nil
	}

	if delErr := k8sClient.CoreV1().Services(tenantNamespace).Delete(ctx, internalRegistryServiceName, metav1.DeleteOptions{}); delErr != nil {
		return nil, fmt.Errorf("failed to replace existing registry service with alias: %w", delErr)
	}
	created, createErr := k8sClient.CoreV1().Services(tenantNamespace).Create(ctx, desired, metav1.CreateOptions{})
	if createErr != nil {
		return nil, fmt.Errorf("failed to recreate registry alias service: %w", createErr)
	}
	return map[string]interface{}{
		"status":        "recreated",
		"service":       created.Name,
		"external_name": desiredExternalName,
	}, nil
}

func (s *Service) triggerTrivyDBWarmupIfNeeded(ctx context.Context, k8sClient kubernetes.Interface, namespace string) (map[string]interface{}, error) {
	if k8sClient == nil || strings.TrimSpace(namespace) == "" {
		return map[string]interface{}{
			"status": "skipped",
			"reason": "missing_client_or_namespace",
		}, nil
	}

	cron, err := k8sClient.BatchV1().CronJobs(namespace).Get(ctx, trivyDBWarmupCronJobName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return map[string]interface{}{
				"status":  "skipped",
				"reason":  "cronjob_not_found",
				"cronjob": trivyDBWarmupCronJobName,
			}, nil
		}
		return nil, fmt.Errorf("failed to load cronjob %s: %w", trivyDBWarmupCronJobName, err)
	}

	if cron.Spec.Suspend != nil && *cron.Spec.Suspend {
		return map[string]interface{}{
			"status":  "skipped",
			"reason":  "cronjob_suspended",
			"cronjob": cron.Name,
		}, nil
	}

	if cron.Status.LastScheduleTime != nil {
		return map[string]interface{}{
			"status":               "skipped",
			"reason":               "already_scheduled",
			"cronjob":              cron.Name,
			"last_schedule_time":   cron.Status.LastScheduleTime.Time.UTC().Format(time.RFC3339),
			"active_jobs_observed": len(cron.Status.Active),
		}, nil
	}

	if len(cron.Status.Active) > 0 {
		return map[string]interface{}{
			"status":               "skipped",
			"reason":               "active_jobs_present",
			"cronjob":              cron.Name,
			"active_jobs_observed": len(cron.Status.Active),
		}, nil
	}

	existingManualJobs, listErr := k8sClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "imagefactory.io/warmup=manual-bootstrap",
	})
	if listErr != nil {
		return nil, fmt.Errorf("failed to list manual trivy warmup jobs: %w", listErr)
	}
	cooldown := trivyWarmupManualCooldown()
	now := time.Now().UTC()
	for _, job := range existingManualJobs.Items {
		if job.Namespace != namespace {
			continue
		}
		// Do not fan out multiple warmup jobs concurrently.
		if job.Status.Active > 0 {
			return map[string]interface{}{
				"status":   "skipped",
				"reason":   "manual_job_active",
				"cronjob":  cron.Name,
				"job_name": job.Name,
			}, nil
		}
		// Also suppress if a manual warmup already completed or started recently.
		if isWarmupJobRecent(&job, now, cooldown) {
			return map[string]interface{}{
				"status":   "skipped",
				"reason":   "manual_job_recent",
				"cronjob":  cron.Name,
				"job_name": job.Name,
			}, nil
		}
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       "trivy-db-warmup",
		"app.kubernetes.io/managed-by": "image-factory",
		"imagefactory.io/warmup":       "manual-bootstrap",
	}
	if cron.Spec.JobTemplate.Labels != nil {
		for k, v := range cron.Spec.JobTemplate.Labels {
			labels[k] = v
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: trivyDBWarmupCronJobName + "-manual-",
			Labels:       labels,
		},
		Spec: *cron.Spec.JobTemplate.Spec.DeepCopy(),
	}

	created, err := k8sClient.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed creating manual trivy DB warmup job: %w", err)
	}

	return map[string]interface{}{
		"status":   "triggered",
		"cronjob":  cron.Name,
		"job_name": created.Name,
	}, nil
}

func trivyWarmupManualCooldown() time.Duration {
	raw := strings.TrimSpace(os.Getenv("IF_TRIVY_WARMUP_MANUAL_COOLDOWN_SECONDS"))
	if raw == "" {
		return 15 * time.Minute
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(seconds) * time.Second
}

func isWarmupJobRecent(job *batchv1.Job, now time.Time, cooldown time.Duration) bool {
	if job == nil {
		return false
	}
	if cooldown <= 0 {
		cooldown = 15 * time.Minute
	}
	if job.Status.CompletionTime != nil && now.Sub(job.Status.CompletionTime.Time) <= cooldown {
		return true
	}
	if job.Status.StartTime != nil && now.Sub(job.Status.StartTime.Time) <= cooldown {
		return true
	}
	if job.CreationTimestamp.Time.IsZero() {
		return false
	}
	return now.Sub(job.CreationTimestamp.Time) <= cooldown
}

func applyTektonCleanupPolicyToObjects(objects []*unstructured.Unstructured, policy tektonHistoryCleanupPolicy) {
	for _, obj := range objects {
		if obj == nil || obj.GetKind() != "CronJob" || obj.GetName() != tektonHistoryCleanupCronJobName {
			continue
		}
		_ = unstructured.SetNestedField(obj.Object, policy.Schedule, "spec", "schedule")
		_ = unstructured.SetNestedField(obj.Object, !policy.Enabled, "spec", "suspend")

		env := []interface{}{
			map[string]interface{}{"name": "KEEP_PIPELINERUNS", "value": strconv.Itoa(policy.KeepPipelineRuns)},
			map[string]interface{}{"name": "KEEP_TASKRUNS", "value": strconv.Itoa(policy.KeepTaskRuns)},
			map[string]interface{}{"name": "KEEP_PODS", "value": strconv.Itoa(policy.KeepPods)},
		}
		containers, found, err := unstructured.NestedSlice(
			obj.Object,
			"spec",
			"jobTemplate",
			"spec",
			"template",
			"spec",
			"containers",
		)
		if err != nil || !found || len(containers) == 0 {
			continue
		}
		first, ok := containers[0].(map[string]interface{})
		if !ok {
			continue
		}
		first["env"] = env
		containers[0] = first
		_ = unstructured.SetNestedSlice(
			obj.Object,
			containers,
			"spec",
			"jobTemplate",
			"spec",
			"template",
			"spec",
			"containers",
		)
	}
}

func (s *Service) resolveInternalRegistryStorageProfile(ctx context.Context) systemconfig.RuntimeAssetStorageProfile {
	profile := systemconfig.RuntimeAssetStorageProfile{
		Type:         "hostPath",
		HostPath:     internalRegistryStorageHostPathDefault,
		HostPathType: internalRegistryHostPathTypeDefault,
		PVCName:      internalRegistryPVCNameDefault,
		PVCSize:      internalRegistryPVCSizeDefault,
		PVCAccessModes: []string{
			"ReadWriteOnce",
		},
	}
	if s == nil || s.runtimeServicesConfigLookup == nil {
		return profile
	}
	cfg, err := s.runtimeServicesConfigLookup(ctx)
	if err != nil || cfg == nil {
		return profile
	}
	override := cfg.StorageProfiles.InternalRegistry
	if storageType := strings.TrimSpace(override.Type); storageType != "" {
		profile.Type = storageType
	}
	if hostPath := strings.TrimSpace(override.HostPath); hostPath != "" {
		profile.HostPath = hostPath
	}
	if hostPathType := strings.TrimSpace(override.HostPathType); hostPathType != "" {
		profile.HostPathType = hostPathType
	}
	if pvcName := strings.TrimSpace(override.PVCName); pvcName != "" {
		profile.PVCName = pvcName
	}
	if pvcSize := strings.TrimSpace(override.PVCSize); pvcSize != "" {
		profile.PVCSize = pvcSize
	}
	if storageClass := strings.TrimSpace(override.PVCStorageClass); storageClass != "" {
		profile.PVCStorageClass = storageClass
	}
	if len(override.PVCAccessModes) > 0 {
		modes := make([]string, 0, len(override.PVCAccessModes))
		for _, mode := range override.PVCAccessModes {
			trimmed := strings.TrimSpace(mode)
			if trimmed != "" {
				modes = append(modes, trimmed)
			}
		}
		if len(modes) > 0 {
			profile.PVCAccessModes = modes
		}
	}
	return profile
}

func applyStorageProfilesToObjects(
	objects []*unstructured.Unstructured,
	internalRegistryProfile systemconfig.RuntimeAssetStorageProfile,
) []*unstructured.Unstructured {
	if len(objects) == 0 {
		return objects
	}
	out := make([]*unstructured.Unstructured, 0, len(objects))
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		kind := strings.TrimSpace(obj.GetKind())
		name := strings.TrimSpace(obj.GetName())
		if kind == "PersistentVolumeClaim" && name == internalRegistryPVCNameDefault {
			if !isPVCStorageType(internalRegistryProfile.Type) {
				continue
			}
			applyInternalRegistryPVCProfile(obj, internalRegistryProfile)
		}
		if kind == "Deployment" && name == internalRegistryServiceName {
			applyInternalRegistryDeploymentStorageProfile(obj, internalRegistryProfile)
		}
		out = append(out, obj)
	}
	return out
}

func normalizeStorageType(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "hostpath":
		return "hostpath"
	case "emptydir":
		return "emptydir"
	case "pvc":
		return "pvc"
	default:
		return "hostpath"
	}
}

func isPVCStorageType(raw string) bool {
	return normalizeStorageType(raw) == "pvc"
}

func applyInternalRegistryPVCProfile(obj *unstructured.Unstructured, profile systemconfig.RuntimeAssetStorageProfile) {
	if obj == nil {
		return
	}
	claimName := strings.TrimSpace(profile.PVCName)
	if claimName == "" {
		claimName = internalRegistryPVCNameDefault
	}
	obj.SetName(claimName)

	modes := make([]interface{}, 0, len(profile.PVCAccessModes))
	for _, mode := range profile.PVCAccessModes {
		trimmed := strings.TrimSpace(mode)
		if trimmed != "" {
			modes = append(modes, trimmed)
		}
	}
	if len(modes) == 0 {
		modes = []interface{}{"ReadWriteOnce"}
	}
	_ = unstructured.SetNestedSlice(obj.Object, modes, "spec", "accessModes")

	storageSize := strings.TrimSpace(profile.PVCSize)
	if storageSize == "" {
		storageSize = internalRegistryPVCSizeDefault
	}
	_ = unstructured.SetNestedMap(obj.Object, map[string]interface{}{
		"requests": map[string]interface{}{
			"storage": storageSize,
		},
	}, "spec", "resources")

	storageClass := strings.TrimSpace(profile.PVCStorageClass)
	if storageClass == "" {
		unstructured.RemoveNestedField(obj.Object, "spec", "storageClassName")
	} else {
		_ = unstructured.SetNestedField(obj.Object, storageClass, "spec", "storageClassName")
	}
}

func applyInternalRegistryDeploymentStorageProfile(obj *unstructured.Unstructured, profile systemconfig.RuntimeAssetStorageProfile) {
	if obj == nil {
		return
	}
	storageType := normalizeStorageType(profile.Type)

	volumes, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "volumes")
	if err != nil || !found {
		volumes = []interface{}{}
	}

	registryVolume := map[string]interface{}{
		"name": "registry-data",
	}
	switch storageType {
	case "emptydir":
		registryVolume["emptyDir"] = map[string]interface{}{}
	case "pvc":
		claimName := strings.TrimSpace(profile.PVCName)
		if claimName == "" {
			claimName = internalRegistryPVCNameDefault
		}
		registryVolume["persistentVolumeClaim"] = map[string]interface{}{
			"claimName": claimName,
		}
	default:
		hostPath := strings.TrimSpace(profile.HostPath)
		if hostPath == "" {
			hostPath = internalRegistryStorageHostPathDefault
		}
		hostPathType := strings.TrimSpace(profile.HostPathType)
		if hostPathType == "" {
			hostPathType = internalRegistryHostPathTypeDefault
		}
		registryVolume["hostPath"] = map[string]interface{}{
			"path": hostPath,
			"type": hostPathType,
		}
	}

	replaced := false
	for i, existing := range volumes {
		asMap, ok := existing.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprintf("%v", asMap["name"])) != "registry-data" {
			continue
		}
		volumes[i] = registryVolume
		replaced = true
		break
	}
	if !replaced {
		volumes = append(volumes, registryVolume)
	}
	_ = unstructured.SetNestedSlice(obj.Object, volumes, "spec", "template", "spec", "volumes")
}

func ensureInternalRegistryDeploymentStorageMode(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	namespace string,
	profile systemconfig.RuntimeAssetStorageProfile,
) (bool, error) {
	if k8sClient == nil || strings.TrimSpace(namespace) == "" {
		return false, nil
	}
	deployments := k8sClient.AppsV1().Deployments(namespace)
	deployment, err := deployments.Get(ctx, internalRegistryServiceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	var targetVolume *corev1.Volume
	for i := range deployment.Spec.Template.Spec.Volumes {
		if deployment.Spec.Template.Spec.Volumes[i].Name == "registry-data" {
			targetVolume = &deployment.Spec.Template.Spec.Volumes[i]
			break
		}
	}
	if targetVolume == nil {
		return false, nil
	}

	count := 0
	currentType := ""
	if targetVolume.HostPath != nil {
		count++
		currentType = "hostpath"
	}
	if targetVolume.PersistentVolumeClaim != nil {
		count++
		currentType = "pvc"
	}
	if targetVolume.EmptyDir != nil {
		count++
		currentType = "emptydir"
	}
	desiredType := normalizeStorageType(profile.Type)
	if count == 1 && currentType == desiredType {
		return false, nil
	}

	if delErr := deployments.Delete(ctx, internalRegistryServiceName, metav1.DeleteOptions{}); delErr != nil && !apierrors.IsNotFound(delErr) {
		return false, delErr
	}
	waitUntil := time.Now().Add(45 * time.Second)
	for time.Now().Before(waitUntil) {
		_, getErr := deployments.Get(ctx, internalRegistryServiceName, metav1.GetOptions{})
		if apierrors.IsNotFound(getErr) {
			return true, nil
		}
		if getErr != nil {
			return false, getErr
		}
		time.Sleep(1500 * time.Millisecond)
	}
	return true, fmt.Errorf("timed out waiting for deployment %s/%s deletion after storage mode reconciliation", namespace, internalRegistryServiceName)
}

func normalizeBootstrapMode(raw string) string {
	switch strings.TrimSpace(raw) {
	case "self_managed":
		return "self_managed"
	default:
		return "image_factory_managed"
	}
}

func normalizeCredentialScope(raw string) string {
	switch strings.TrimSpace(raw) {
	case "cluster_admin":
		return "cluster_admin"
	case "namespace_admin":
		return "namespace_admin"
	case "read_only":
		return "read_only"
	default:
		return "unknown"
	}
}

func runtimeTenantRBACYAML(runtimeServiceAccountNamespace, tenantNamespace string) string {
	// Keep runtime RBAC namespace-scoped for least privilege. Cluster-scoped namespace creation must be done
	// by bootstrap credentials (managed mode) rather than runtime credentials.
	return fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: image-factory-runtime-role
  namespace: %s
rules:
- apiGroups: ["tekton.dev"]
  resources: ["tasks", "pipelines"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["tekton.dev"]
  resources: ["taskruns"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns"]
  verbs: ["create", "get", "list", "watch", "delete"]
- apiGroups: [""]
  resources: ["pods", "pods/log", "pods/exec", "secrets", "configmaps", "serviceaccounts", "events", "persistentvolumeclaims"]
  verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: image-factory-runtime-binding
  namespace: %s
subjects:
- kind: ServiceAccount
  name: image-factory-runtime-sa
  namespace: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: image-factory-runtime-role
`, tenantNamespace, tenantNamespace, runtimeServiceAccountNamespace)
}

func defaultProviderSchedulable(providerType ProviderType) bool {
	return !isK8sCapableProviderType(providerType)
}

func validateKubernetesProviderAuthConfig(providerType ProviderType, config map[string]interface{}, bootstrapMode string) error {
	if !isK8sCapableProviderType(providerType) {
		return nil
	}
	if normalizeBootstrapMode(bootstrapMode) == "image_factory_managed" {
		if _, err := connectors.BuildRESTConfigFromProviderConfigForContext(config, connectors.AuthContextBootstrap); err != nil {
			return fmt.Errorf("bootstrap auth config is invalid: %w", err)
		}
		// Runtime auth is optional at onboarding in managed mode; it will be generated during prepare.
		if hasUsableRuntimeAuthConfig(config) {
			if _, err := connectors.BuildRESTConfigFromProviderConfigForContext(config, connectors.AuthContextRuntime); err != nil {
				return fmt.Errorf("runtime auth config is invalid: %w", err)
			}
		}
		return nil
	}
	if _, err := connectors.BuildRESTConfigFromProviderConfigForContext(config, connectors.AuthContextRuntime); err != nil {
		return fmt.Errorf("runtime auth config is invalid: %w", err)
	}
	return nil
}

func hasUsableRuntimeAuthConfig(config map[string]interface{}) bool {
	if config == nil {
		return false
	}
	if _, err := connectors.BuildRESTConfigFromProviderConfigForContext(config, connectors.AuthContextRuntime); err != nil {
		return false
	}
	return true
}

func sanitizeLegacyKubernetesConfig(providerType ProviderType, config map[string]interface{}) map[string]interface{} {
	if config == nil || !isK8sCapableProviderType(providerType) {
		return config
	}
	// Legacy field not used by current runtime/prepare flows.
	delete(config, "namespace")
	return config
}

func shallowCloneMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func testConnectionConfigForProvider(provider *Provider) (map[string]interface{}, string, error) {
	if provider == nil {
		return nil, "unknown", fmt.Errorf("provider is required")
	}
	config := shallowCloneMap(provider.Config)
	if !isK8sCapableProviderType(provider.ProviderType) {
		return config, "runtime", nil
	}

	if normalizeBootstrapMode(provider.BootstrapMode) == "image_factory_managed" {
		if _, err := connectors.BuildRESTConfigFromProviderConfigForContext(config, connectors.AuthContextBootstrap); err != nil {
			return nil, "bootstrap", fmt.Errorf("managed provider requires valid bootstrap auth for connection test: %w", err)
		}
		bootstrapAuth, _ := config["bootstrap_auth"].(map[string]interface{})
		config["runtime_auth"] = shallowCloneMap(bootstrapAuth)
		return config, "bootstrap", nil
	}

	if _, err := connectors.BuildRESTConfigFromProviderConfigForContext(config, connectors.AuthContextRuntime); err != nil {
		return nil, "runtime", fmt.Errorf("self-managed provider requires valid runtime auth for connection test: %w", err)
	}

	return config, "runtime", nil
}

func resolveProviderSystemNamespace(provider *Provider) string {
	systemNamespace := resolveTektonTargetNamespace(provider, uuid.Nil)
	if provider != nil && provider.Config != nil {
		if ns, ok := provider.Config["system_namespace"].(string); ok && strings.TrimSpace(ns) != "" {
			systemNamespace = strings.TrimSpace(ns)
		}
	}
	return systemNamespace
}

func (s *Service) ensureManagedRuntimeIdentityAndPersistConfig(
	ctx context.Context,
	provider *Provider,
	k8sClient kubernetes.Interface,
	systemNamespace string,
	endpoint string,
	caData []byte,
	forceRegenerate bool,
) (map[string]interface{}, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}
	runtimeAudiences := resolveRuntimeTokenAudiences(provider, endpoint)
	hadUsableRuntimeAuth := hasUsableRuntimeAuthConfig(provider.Config)
	regenerationReason := ""
	if hadUsableRuntimeAuth && !forceRegenerate {
		needsRegenerate, regenReason := managedRuntimeAuthNeedsRegeneration(provider, endpoint, runtimeAudiences)
		if !needsRegenerate {
			return map[string]interface{}{
				"generated": false,
				"reason":    "runtime_auth already configured",
			}, nil
		}
		regenerationReason = regenReason
	} else if hadUsableRuntimeAuth && forceRegenerate {
		regenerationReason = "runtime auth regeneration requested by run action"
	}

	if err := ensureNamespace(ctx, k8sClient, systemNamespace); err != nil {
		return nil, fmt.Errorf("failed to ensure system namespace %s: %w", systemNamespace, err)
	}

	const runtimeServiceAccountName = "image-factory-runtime-sa"
	const runtimeSystemRoleName = "image-factory-runtime-system-role"
	const runtimeSystemRoleBindingName = "image-factory-runtime-system-binding"

	if _, err := k8sClient.CoreV1().ServiceAccounts(systemNamespace).Get(ctx, runtimeServiceAccountName, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get runtime service account: %w", err)
		}
		if _, err := k8sClient.CoreV1().ServiceAccounts(systemNamespace).Create(ctx, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: runtimeServiceAccountName,
			},
		}, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create runtime service account: %w", err)
		}
	}

	desiredRoleRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"tekton.dev"},
			Resources: []string{"tasks", "pipelines", "taskruns", "pipelineruns"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "secrets", "serviceaccounts"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}

	roleClient := k8sClient.RbacV1().Roles(systemNamespace)
	if role, err := roleClient.Get(ctx, runtimeSystemRoleName, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get runtime system role: %w", err)
		}
		if _, err := roleClient.Create(ctx, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name: runtimeSystemRoleName,
			},
			Rules: desiredRoleRules,
		}, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create runtime system role: %w", err)
		}
	} else {
		role.Rules = desiredRoleRules
		if _, err := roleClient.Update(ctx, role, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to update runtime system role: %w", err)
		}
	}

	roleBindingClient := k8sClient.RbacV1().RoleBindings(systemNamespace)
	desiredSubjects := []rbacv1.Subject{{
		Kind:      "ServiceAccount",
		Name:      runtimeServiceAccountName,
		Namespace: systemNamespace,
	}}
	desiredRoleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     runtimeSystemRoleName,
	}

	if rb, err := roleBindingClient.Get(ctx, runtimeSystemRoleBindingName, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get runtime system role binding: %w", err)
		}
		if _, err := roleBindingClient.Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: runtimeSystemRoleBindingName,
			},
			Subjects: desiredSubjects,
			RoleRef:  desiredRoleRef,
		}, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create runtime system role binding: %w", err)
		}
	} else {
		rb.Subjects = desiredSubjects
		rb.RoleRef = desiredRoleRef
		if _, err := roleBindingClient.Update(ctx, rb, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to update runtime system role binding: %w", err)
		}
	}

	expirationSeconds := int64(365 * 24 * 60 * 60) // 1 year (cluster may cap max)
	tokenResponse, err := k8sClient.CoreV1().ServiceAccounts(systemNamespace).CreateToken(ctx, runtimeServiceAccountName, &authnv1.TokenRequest{
		Spec: authnv1.TokenRequestSpec{
			Audiences:         runtimeAudiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate runtime service account token: %w", err)
	}
	if tokenResponse == nil || strings.TrimSpace(tokenResponse.Status.Token) == "" {
		return nil, fmt.Errorf("runtime service account token generation returned empty token")
	}

	runtimeEndpoint := strings.TrimSpace(endpoint)
	if runtimeEndpoint == "" {
		return nil, fmt.Errorf("runtime endpoint is empty; cannot persist runtime auth")
	}

	if provider.Config == nil {
		provider.Config = map[string]interface{}{}
	}
	runtimeAuth := map[string]interface{}{
		"auth_method": "token",
		"endpoint":    runtimeEndpoint,
		"token":       strings.TrimSpace(tokenResponse.Status.Token),
	}
	if len(caData) > 0 {
		runtimeAuth["ca_cert"] = string(caData)
	}
	provider.Config["runtime_auth"] = runtimeAuth
	provider.Config["system_namespace"] = systemNamespace
	provider.UpdatedAt = time.Now().UTC()

	if err := s.repository.UpdateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to persist generated runtime auth: %w", err)
	}

	return map[string]interface{}{
		"generated":        true,
		"regenerated":      hadUsableRuntimeAuth,
		"reason":           regenerationReason,
		"system_namespace": systemNamespace,
		"service_account":  runtimeServiceAccountName,
		"role":             runtimeSystemRoleName,
		"role_binding":     runtimeSystemRoleBindingName,
		"endpoint":         runtimeEndpoint,
		"token_audiences":  runtimeAudiences,
	}, nil
}

func managedRuntimeAuthNeedsRegeneration(provider *Provider, endpoint string, expectedAudiences []string) (bool, string) {
	if provider == nil || provider.Config == nil {
		return true, "runtime_auth missing"
	}
	runtimeAuth, _ := provider.Config["runtime_auth"].(map[string]interface{})
	if runtimeAuth == nil {
		return true, "runtime_auth missing"
	}

	runtimeEndpoint, _ := runtimeAuth["endpoint"].(string)
	if strings.TrimSpace(runtimeEndpoint) == "" {
		return true, "runtime_auth endpoint missing"
	}
	if strings.TrimSpace(endpoint) != "" && strings.TrimSpace(runtimeEndpoint) != strings.TrimSpace(endpoint) {
		return true, "runtime_auth endpoint differs from bootstrap endpoint"
	}

	runtimeToken, _ := runtimeAuth["token"].(string)
	if strings.TrimSpace(runtimeToken) == "" {
		return true, "runtime_auth token missing"
	}
	runtimeTokenAudiences, err := parseJWTAudienceClaim(runtimeToken)
	if err != nil {
		// Unknown token format; keep existing auth instead of forcing rotation.
		return false, ""
	}

	bootstrapAudiences := bootstrapTokenAudiences(provider)
	if len(bootstrapAudiences) > 0 && !hasAudienceIntersection(runtimeTokenAudiences, bootstrapAudiences) {
		return true, "runtime_auth token audience does not match bootstrap audience"
	}

	if len(expectedAudiences) > 0 && !hasAudienceIntersection(runtimeTokenAudiences, expectedAudiences) {
		return true, "runtime_auth token audience does not match expected provider audience"
	}

	return false, ""
}

func hasAudienceIntersection(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, candidate := range left {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	for _, candidate := range right {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			return true
		}
	}
	return false
}

func resolveRuntimeTokenAudiences(provider *Provider, endpoint string) []string {
	audiences := make([]string, 0, 8)
	seen := make(map[string]struct{})
	providerType := ProviderType("")
	if provider != nil {
		providerType = provider.ProviderType
	}
	addAudience := func(candidate string) {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		audiences = append(audiences, trimmed)
	}

	for _, aud := range bootstrapTokenAudiences(provider) {
		addAudience(aud)
	}

	switch providerType {
	case ProviderTypeRancher:
		addAudience("k3s")
	}

	if parsed, err := url.Parse(strings.TrimSpace(endpoint)); err == nil {
		if parsed.Scheme != "" && parsed.Host != "" {
			addAudience(parsed.Scheme + "://" + parsed.Host)
			if host := parsed.Hostname(); host != "" {
				addAudience(parsed.Scheme + "://" + host)
			}
		}
	}

	addAudience("https://kubernetes.default.svc")
	addAudience("https://kubernetes.default.svc.cluster.local")

	return audiences
}

func bootstrapTokenAudiences(provider *Provider) []string {
	if provider == nil || provider.Config == nil {
		return nil
	}
	bootstrapAuth, _ := provider.Config["bootstrap_auth"].(map[string]interface{})
	token, _ := bootstrapAuth["token"].(string)
	if strings.TrimSpace(token) == "" {
		return nil
	}
	audiences, err := parseJWTAudienceClaim(token)
	if err != nil {
		return nil
	}
	return audiences
}

func parseJWTAudienceClaim(token string) ([]string, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode jwt payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal jwt payload: %w", err)
	}

	audClaim, ok := claims["aud"]
	if !ok {
		return nil, nil
	}

	seen := make(map[string]struct{})
	audiences := make([]string, 0, 4)
	appendAudience := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		audiences = append(audiences, trimmed)
	}

	switch typed := audClaim.(type) {
	case string:
		appendAudience(typed)
	case []interface{}:
		for _, raw := range typed {
			if candidate, ok := raw.(string); ok {
				appendAudience(candidate)
			}
		}
	}

	return audiences, nil
}

func requestedActionEnabled(actions map[string]interface{}, key string) bool {
	if actions == nil {
		return false
	}
	raw, ok := actions[key]
	if !ok {
		return false
	}
	enabled, ok := raw.(bool)
	return ok && enabled
}

func validateTargetNamespace(providerType ProviderType, targetNamespace *string) error {
	if !isK8sCapableProviderType(providerType) {
		return nil
	}
	if targetNamespace == nil {
		return fmt.Errorf("%w: target namespace is required", ErrInvalidTargetNamespace)
	}
	ns := strings.TrimSpace(*targetNamespace)
	if ns == "" {
		return fmt.Errorf("%w: target namespace is required", ErrInvalidTargetNamespace)
	}
	if len(ns) > 63 {
		return fmt.Errorf("%w: target namespace must be 63 characters or fewer", ErrInvalidTargetNamespace)
	}
	if !kubernetesNamespacePattern.MatchString(ns) {
		return fmt.Errorf("%w: target namespace must match kubernetes namespace format (lowercase alphanumeric and '-')", ErrInvalidTargetNamespace)
	}
	return nil
}

// CreateProvider creates a new infrastructure provider
func (s *Service) CreateProvider(ctx context.Context, tenantID, userID uuid.UUID, req *CreateProviderRequest) (*Provider, error) {
	// Validate provider type
	if req.ProviderType != ProviderTypeKubernetes &&
		req.ProviderType != ProviderTypeAWSEKS &&
		req.ProviderType != ProviderTypeGCPGKE &&
		req.ProviderType != ProviderTypeAzureAKS &&
		req.ProviderType != ProviderTypeOCIOKE &&
		req.ProviderType != ProviderTypeVMwareVKS &&
		req.ProviderType != ProviderTypeOpenShift &&
		req.ProviderType != ProviderTypeRancher &&
		req.ProviderType != ProviderTypeBuildNodes {
		return nil, ErrInvalidProviderType
	}

	// Check if provider name already exists for this tenant
	exists, err := s.repository.ExistsProviderByName(ctx, tenantID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check provider existence: %w", err)
	}
	if exists {
		return nil, ErrProviderExists
	}

	// Create new provider
	now := time.Now()
	cleanConfig := sanitizeLegacyKubernetesConfig(req.ProviderType, req.Config)
	provider := &Provider{
		ID:              uuid.New(),
		TenantID:        tenantID,
		IsGlobal:        req.IsGlobal,
		ProviderType:    req.ProviderType,
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Config:          cleanConfig,
		Status:          ProviderStatusPending,
		Capabilities:    req.Capabilities,
		CreatedBy:       userID,
		CreatedAt:       now,
		UpdatedAt:       now,
		BootstrapMode:   normalizeBootstrapMode(req.BootstrapMode),
		CredentialScope: normalizeCredentialScope(req.CredentialScope),
		TargetNamespace: req.TargetNamespace,
		IsSchedulable:   defaultProviderSchedulable(req.ProviderType),
	}
	if err := validateTargetNamespace(provider.ProviderType, provider.TargetNamespace); err != nil {
		return nil, err
	}
	if err := validateKubernetesProviderAuthConfig(provider.ProviderType, provider.Config, provider.BootstrapMode); err != nil {
		return nil, err
	}

	// Save provider
	if err := s.repository.SaveProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}

	// Grant configure permission to the tenant
	permission := &ProviderPermission{
		ID:         uuid.New(),
		ProviderID: provider.ID,
		TenantID:   tenantID,
		Permission: "infrastructure:configure",
		GrantedBy:  userID,
		GrantedAt:  now,
	}
	if err := s.repository.SavePermission(ctx, permission); err != nil {
		s.logger.Error("Failed to grant configure permission to tenant",
			zap.String("provider_id", provider.ID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
	}

	// Publish domain event
	if s.eventPublisher != nil {
		event := &ProviderCreated{
			ProviderID:   provider.ID,
			TenantID:     tenantID,
			ProviderType: string(provider.ProviderType),
			Name:         provider.Name,
			CreatedBy:    userID,
		}
		if err := s.eventPublisher.PublishProviderCreated(ctx, event); err != nil {
			s.logger.Error("Failed to publish provider created event", zap.Error(err))
		}
	}

	s.logger.Info("Infrastructure provider created successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("provider_name", provider.Name),
		zap.String("tenant_id", tenantID.String()),
		zap.String("created_by", userID.String()))

	return provider, nil
}

// GetProvider retrieves a provider by ID
func (s *Service) GetProvider(ctx context.Context, providerID uuid.UUID) (*Provider, error) {
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	return provider, nil
}

// ListProviders lists providers for a tenant with optional filtering
func (s *Service) ListProviders(ctx context.Context, tenantID uuid.UUID, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	result, err := s.repository.FindProvidersByTenant(ctx, tenantID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	return result, nil
}

// ListProvidersAll lists providers across all tenants with optional filtering.
func (s *Service) ListProvidersAll(ctx context.Context, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	result, err := s.repository.FindProvidersAll(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	return result, nil
}

// UpdateProvider updates an existing provider
func (s *Service) UpdateProvider(ctx context.Context, providerID uuid.UUID, req *UpdateProviderRequest) (*Provider, error) {
	// Get existing provider
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	// Update fields
	if req.DisplayName != nil {
		provider.DisplayName = *req.DisplayName
	}
	if req.Config != nil {
		provider.Config = sanitizeLegacyKubernetesConfig(provider.ProviderType, *req.Config)
	}
	if req.Capabilities != nil {
		provider.Capabilities = *req.Capabilities
	}
	if req.Status != nil {
		// Validate status
		if *req.Status != ProviderStatusOnline && *req.Status != ProviderStatusOffline &&
			*req.Status != ProviderStatusMaintenance && *req.Status != ProviderStatusPending {
			return nil, ErrInvalidProviderStatus
		}
		provider.Status = *req.Status
	}
	if req.IsGlobal != nil {
		provider.IsGlobal = *req.IsGlobal
	}
	if req.BootstrapMode != nil {
		provider.BootstrapMode = normalizeBootstrapMode(*req.BootstrapMode)
	}
	if req.CredentialScope != nil {
		provider.CredentialScope = normalizeCredentialScope(*req.CredentialScope)
	}
	if req.TargetNamespace != nil {
		provider.TargetNamespace = req.TargetNamespace
	}
	if err := validateTargetNamespace(provider.ProviderType, provider.TargetNamespace); err != nil {
		return nil, err
	}
	if err := validateKubernetesProviderAuthConfig(provider.ProviderType, provider.Config, provider.BootstrapMode); err != nil {
		return nil, err
	}
	provider.UpdatedAt = time.Now()

	// Save updated provider
	if err := s.repository.UpdateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// Publish domain event
	if s.eventPublisher != nil {
		event := &ProviderUpdated{
			ProviderID: providerID,
			TenantID:   provider.TenantID,
			UpdatedBy:  provider.CreatedBy, // TODO: Pass actual user ID
		}
		if err := s.eventPublisher.PublishProviderUpdated(ctx, event); err != nil {
			s.logger.Error("Failed to publish provider updated event", zap.Error(err))
		}
	}

	s.logger.Info("Infrastructure provider updated successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("provider_name", provider.Name))

	return provider, nil
}

// DeleteProvider deletes a provider
func (s *Service) DeleteProvider(ctx context.Context, providerID uuid.UUID) error {
	// Get existing provider
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	// Delete provider (permissions will be cascade deleted)
	if err := s.repository.DeleteProvider(ctx, providerID); err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	// Publish domain event
	if s.eventPublisher != nil {
		event := &ProviderDeleted{
			ProviderID: providerID,
			TenantID:   provider.TenantID,
			DeletedBy:  provider.CreatedBy, // TODO: Pass actual user ID
		}
		if err := s.eventPublisher.PublishProviderDeleted(ctx, event); err != nil {
			s.logger.Error("Failed to publish provider deleted event", zap.Error(err))
		}
	}

	s.logger.Info("Infrastructure provider deleted successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("provider_name", provider.Name))

	return nil
}

// GrantPermission grants a permission to a tenant for a provider
func (s *Service) GrantPermission(ctx context.Context, providerID, tenantID, grantedBy uuid.UUID, permission string) error {
	// Check if provider exists
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	// Create permission
	perm := &ProviderPermission{
		ID:         uuid.New(),
		ProviderID: providerID,
		TenantID:   tenantID,
		Permission: permission,
		GrantedBy:  grantedBy,
		GrantedAt:  time.Now(),
	}

	// Save permission
	if err := s.repository.SavePermission(ctx, perm); err != nil {
		return fmt.Errorf("failed to save permission: %w", err)
	}

	s.logger.Info("Provider permission granted successfully",
		zap.String("provider_id", providerID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("permission", permission))

	return nil
}

// RevokePermission revokes a permission from a tenant for a provider
func (s *Service) RevokePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	if err := s.repository.DeletePermission(ctx, providerID, tenantID, permission); err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	s.logger.Info("Provider permission revoked successfully",
		zap.String("provider_id", providerID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("permission", permission))

	return nil
}

// ListProviderPermissions returns permissions for a provider.
func (s *Service) ListProviderPermissions(ctx context.Context, providerID uuid.UUID) ([]*ProviderPermission, error) {
	perms, err := s.repository.FindPermissionsByProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider permissions: %w", err)
	}
	return perms, nil
}

// HasPermission checks if a tenant has a specific permission for a provider
func (s *Service) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	return s.repository.HasPermission(ctx, providerID, tenantID, permission)
}

// GetAvailableProviders returns providers that a tenant can select from
func (s *Service) GetAvailableProviders(ctx context.Context, tenantID uuid.UUID) ([]*Provider, error) {
	// Load all online providers first. This avoids missing legacy/global providers that
	// may not be stored under the nil tenant sentinel.
	onlineStatus := ProviderStatusOnline
	result, err := s.repository.FindProvidersAll(ctx, &ListProvidersOptions{
		Status: &onlineStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	available := make([]*Provider, 0, len(result.Providers))
	seen := make(map[uuid.UUID]struct{}, len(result.Providers))

	for i := range result.Providers {
		provider := result.Providers[i]
		if !isProviderSchedulable(&provider) {
			continue
		}

		// Tenant-local providers are directly selectable.
		if provider.TenantID == tenantID {
			if _, exists := seen[provider.ID]; exists {
				continue
			}
			copyProvider := provider
			available = append(available, &copyProvider)
			seen[provider.ID] = struct{}{}
			continue
		}

		// Global providers are selectable by all tenants regardless of owning tenant ID.
		if provider.IsGlobal {
			if _, exists := seen[provider.ID]; exists {
				continue
			}
			copyProvider := provider
			available = append(available, &copyProvider)
			seen[provider.ID] = struct{}{}
			continue
		}

		// Explicit per-tenant select grants for non-global providers.
		hasPermission, permErr := s.repository.HasPermission(ctx, provider.ID, tenantID, "infrastructure:select")
		if permErr != nil {
			s.logger.Error("Failed to check permission",
				zap.String("provider_id", provider.ID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(permErr))
			continue
		}
		if hasPermission {
			if _, exists := seen[provider.ID]; exists {
				continue
			}
			copyProvider := provider
			available = append(available, &copyProvider)
			seen[provider.ID] = struct{}{}
		}
	}

	return available, nil
}

func isProviderSchedulable(provider *Provider) bool {
	if provider == nil {
		return false
	}
	if provider.Status != ProviderStatusOnline {
		return false
	}

	if !isK8sCapableProviderType(provider.ProviderType) {
		return provider.IsSchedulable || defaultProviderSchedulable(provider.ProviderType)
	}
	return provider.IsSchedulable
}

func isK8sCapableProviderType(providerType ProviderType) bool {
	switch providerType {
	case ProviderTypeKubernetes,
		ProviderTypeAWSEKS,
		ProviderTypeGCPGKE,
		ProviderTypeAzureAKS,
		ProviderTypeOCIOKE,
		ProviderTypeVMwareVKS,
		ProviderTypeOpenShift,
		ProviderTypeRancher:
		return true
	default:
		return false
	}
}

func hasReadinessPrereq(prereqs []string, needle string) bool {
	needleLower := strings.ToLower(needle)
	for _, prereq := range prereqs {
		if strings.Contains(strings.ToLower(prereq), needleLower) {
			return true
		}
	}
	return false
}

// TestProviderConnection tests the connection to a provider
func (s *Service) TestProviderConnection(ctx context.Context, providerID uuid.UUID) (*TestConnectionResponse, error) {
	// Get provider
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	testConfig, authContext, err := testConnectionConfigForProvider(provider)
	if err != nil {
		s.logger.Warn("Provider connection test auth selection failed",
			zap.String("provider_id", providerID.String()),
			zap.String("provider_type", string(provider.ProviderType)),
			zap.String("auth_context", authContext),
			zap.Error(err))
		return &TestConnectionResponse{
			Success: false,
			Message: err.Error(),
			Details: map[string]interface{}{
				"auth_context": authContext,
			},
		}, nil
	}

	// Create connector for the provider type
	connector, err := s.connectorFactory.CreateConnector(string(provider.ProviderType), testConfig)
	if err != nil {
		s.logger.Error("Failed to create connector",
			zap.String("provider_type", string(provider.ProviderType)),
			zap.Error(err))
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create connector: %v", err),
		}, nil
	}

	// Test the connection
	result, err := connector.TestConnection(ctx)
	if err != nil {
		s.logger.Error("Connection test failed",
			zap.String("provider_id", providerID.String()),
			zap.Error(err))
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Connection test failed: %v", err),
		}, nil
	}
	if result != nil {
		if result.Details == nil {
			result.Details = map[string]interface{}{}
		}
		result.Details["auth_context"] = authContext
	}

	// Update health status based on connection result
	healthStatus := "unhealthy"
	if result.Success {
		healthStatus = "healthy"
	}

	health := &ProviderHealth{
		ProviderID: providerID,
		Status:     healthStatus,
		LastCheck:  time.Now(),
		Details:    result.Details,
	}
	if err := s.repository.UpdateProviderHealth(ctx, providerID, health); err != nil {
		s.logger.Error("Failed to update provider health", zap.Error(err))
	}

	// Update provider status based on health check
	if result.Success {
		provider.Status = ProviderStatusOnline
	} else {
		provider.Status = ProviderStatusOffline
	}
	provider.LastHealthCheck = &health.LastCheck
	provider.HealthStatus = &healthStatus
	provider.UpdatedAt = time.Now()
	if err := s.repository.UpdateProvider(ctx, provider); err != nil {
		s.logger.Error("Failed to update provider status", zap.Error(err))
	}

	response := &TestConnectionResponse{
		Success: result.Success,
		Message: result.Message,
		Details: result.Details,
	}

	s.logger.Info("Provider connection test completed",
		zap.String("provider_id", providerID.String()),
		zap.Bool("success", result.Success),
		zap.String("message", result.Message),
		zap.String("health_status", healthStatus))

	return response, nil
}

// GetProviderHealth gets the health status of a provider
func (s *Service) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*ProviderHealth, error) {
	health, err := s.repository.GetProviderHealth(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider health: %w", err)
	}
	return health, nil
}

// UpdateProviderReadiness persists readiness status for a provider.
func (s *Service) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("failed to load provider for readiness reconciliation: %w", err)
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	healthStatus := "healthy"
	targetStatus := provider.Status
	isSchedulable := true
	blockedBy := make([]string, 0, 3)
	if status != "ready" {
		if hasConnectivityFailurePrereq(missingPrereqs) {
			healthStatus = "unhealthy"
		} else {
			healthStatus = "warning"
		}
		targetStatus = ProviderStatusOffline
		isSchedulable = false
		blockedBy = append(blockedBy, "provider_not_ready")
	} else if provider.Status != ProviderStatusMaintenance {
		targetStatus = ProviderStatusOnline
	}
	if hasReadinessPrereq(missingPrereqs, "cluster_capacity") {
		isSchedulable = false
		blockedBy = append(blockedBy, "cluster_capacity")
	}
	if targetStatus != ProviderStatusOnline {
		isSchedulable = false
		blockedBy = append(blockedBy, fmt.Sprintf("provider_status_%s", targetStatus))
	}
	schedulableReason := schedulableReasonFromBlockedBy(blockedBy)
	provider.ReadinessStatus = &status
	provider.ReadinessLastChecked = &checkedAt
	provider.ReadinessMissingPrereqs = append([]string(nil), missingPrereqs...)
	provider.HealthStatus = &healthStatus
	provider.LastHealthCheck = &checkedAt
	provider.Status = targetStatus
	provider.IsSchedulable = isSchedulable
	provider.SchedulableReason = &schedulableReason
	provider.BlockedBy = dedupeBlockedBy(blockedBy)
	provider.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateProvider(ctx, provider); err != nil {
		return fmt.Errorf("failed to persist provider readiness reconciliation: %w", err)
	}

	return nil
}

func hasConnectivityFailurePrereq(prereqs []string) bool {
	for _, prereq := range prereqs {
		normalized := strings.ToLower(strings.TrimSpace(prereq))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "unauthorized") || strings.Contains(normalized, "access denied") || strings.Contains(normalized, "forbidden") {
			continue
		}
		if strings.Contains(normalized, "invalid provider kubeconfig/config") ||
			strings.Contains(normalized, "failed to create kubernetes client") ||
			strings.Contains(normalized, "kubernetes api unreachable") ||
			strings.Contains(normalized, "dial tcp") ||
			strings.Contains(normalized, "i/o timeout") ||
			strings.Contains(normalized, "connection refused") ||
			strings.Contains(normalized, "no such host") ||
			strings.Contains(normalized, "tls handshake timeout") ||
			strings.Contains(normalized, "context deadline exceeded") {
			return true
		}
	}
	return false
}

func dedupeBlockedBy(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func schedulableReasonFromBlockedBy(blockedBy []string) string {
	if len(blockedBy) == 0 {
		return "provider is ready for scheduling"
	}
	for _, reason := range blockedBy {
		if reason == "cluster_capacity" {
			return "cluster capacity is not ready"
		}
	}
	for _, reason := range blockedBy {
		if strings.HasPrefix(reason, "provider_status_") {
			return fmt.Sprintf("provider status is %s", strings.TrimPrefix(reason, "provider_status_"))
		}
	}
	for _, reason := range blockedBy {
		if reason == "provider_not_ready" {
			return "provider is not ready"
		}
	}
	return "provider is blocked by readiness or policy gates"
}

func (s *Service) StartProviderPrepareRun(
	ctx context.Context,
	providerID, tenantID, requestedBy uuid.UUID,
	req StartProviderPrepareRunRequest,
) (*ProviderPrepareRun, error) {
	if s.prepareRepository == nil {
		return nil, ErrProviderPrepareNotConfigured
	}

	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	if tenantID != provider.TenantID {
		return nil, ErrPermissionDenied
	}

	active, err := s.prepareRepository.FindActiveProviderPrepareRunByProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active provider prepare run: %w", err)
	}
	if active != nil {
		return nil, ErrProviderPrepareRunInProgress
	}

	now := time.Now().UTC()
	actions := req.RequestedActions
	if actions == nil {
		actions = map[string]interface{}{
			"connectivity": true,
			"bootstrap":    true,
			"readiness":    true,
		}
	}
	run := &ProviderPrepareRun{
		ID:               uuid.New(),
		ProviderID:       providerID,
		TenantID:         tenantID,
		RequestedBy:      requestedBy,
		Status:           ProviderPrepareRunStatusPending,
		RequestedActions: actions,
		ResultSummary:    map[string]interface{}{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.prepareRepository.CreateProviderPrepareRun(ctx, run); err != nil {
		if strings.Contains(err.Error(), "uq_provider_prepare_runs_provider_active") {
			return nil, ErrProviderPrepareRunInProgress
		}
		return nil, fmt.Errorf("failed to create provider prepare run: %w", err)
	}

	// Run asynchronously and keep API fast/responsive.
	go s.executeProviderPrepareRun(context.Background(), run.ID)

	return run, nil
}

func (s *Service) GetProviderPrepareStatus(ctx context.Context, providerID uuid.UUID) (*ProviderPrepareStatus, error) {
	if s.prepareRepository == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	active, err := s.prepareRepository.FindActiveProviderPrepareRunByProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active provider prepare run: %w", err)
	}
	status := &ProviderPrepareStatus{ProviderID: providerID.String()}
	run := active
	if run == nil {
		// If there's no active run, return the latest run + its checks so the UI can
		// render a complete timeline rather than an all-pending/blocked view.
		runs, runsErr := s.prepareRepository.ListProviderPrepareRunsByProvider(ctx, providerID, 1, 0)
		if runsErr != nil {
			return nil, fmt.Errorf("failed to list provider prepare runs: %w", runsErr)
		}
		if len(runs) > 0 {
			run = runs[0]
		}
	}
	if run != nil {
		status.ActiveRun = run
		checks, checksErr := s.prepareRepository.ListProviderPrepareRunChecks(ctx, run.ID, 500, 0)
		if checksErr != nil {
			return nil, fmt.Errorf("failed to list provider prepare checks: %w", checksErr)
		}
		status.Checks = checks
	}
	return status, nil
}

func (s *Service) ListProviderPrepareRuns(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*ProviderPrepareRun, error) {
	if s.prepareRepository == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	runs, err := s.prepareRepository.ListProviderPrepareRunsByProvider(ctx, providerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider prepare runs: %w", err)
	}
	return runs, nil
}

func (s *Service) ListProviderPrepareRunChecks(ctx context.Context, runID uuid.UUID, limit, offset int) ([]*ProviderPrepareRunCheck, error) {
	if s.prepareRepository == nil {
		return nil, ErrProviderPrepareNotConfigured
	}
	run, err := s.prepareRepository.GetProviderPrepareRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider prepare run: %w", err)
	}
	if run == nil {
		return nil, ErrProviderPrepareRunNotFound
	}
	checks, err := s.prepareRepository.ListProviderPrepareRunChecks(ctx, runID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider prepare run checks: %w", err)
	}
	return checks, nil
}

func (s *Service) GetProviderPrepareRunWithChecks(
	ctx context.Context,
	providerID, runID uuid.UUID,
	limit, offset int,
) (*ProviderPrepareRun, []*ProviderPrepareRunCheck, error) {
	if s.prepareRepository == nil {
		return nil, nil, ErrProviderPrepareNotConfigured
	}
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, nil, ErrProviderNotFound
	}
	run, err := s.prepareRepository.GetProviderPrepareRun(ctx, runID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get provider prepare run: %w", err)
	}
	if run == nil || run.ProviderID != providerID {
		return nil, nil, ErrProviderPrepareRunNotFound
	}
	checks, err := s.prepareRepository.ListProviderPrepareRunChecks(ctx, runID, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list provider prepare run checks: %w", err)
	}
	return run, checks, nil
}

func (s *Service) ListLatestProviderPrepareSummaries(ctx context.Context, providerIDs []uuid.UUID) (map[uuid.UUID]*ProviderPrepareLatestSummary, error) {
	startedAt := time.Now()
	source := "none"
	summaries := make(map[uuid.UUID]*ProviderPrepareLatestSummary, len(providerIDs))
	if len(providerIDs) == 0 {
		s.logger.Debug("Listed latest provider prepare summaries", zap.Int("provider_count", 0), zap.String("source", source), zap.Duration("duration", time.Since(startedAt)))
		return summaries, nil
	}
	atomic.AddInt64(&s.prepareSummaryProvidersTotal, int64(len(providerIDs)))
	recordBatchMetrics := func(batchSource string, hasError bool) {
		duration := time.Since(startedAt)
		s.prepareSummaryBatchDuration.RecordOperation(duration.Milliseconds())
		switch batchSource {
		case "repository_batch":
			atomic.AddInt64(&s.prepareSummaryBatchesRepo, 1)
		case "fallback_iterative":
			atomic.AddInt64(&s.prepareSummaryBatchesFallback, 1)
		}
		if hasError {
			atomic.AddInt64(&s.prepareSummaryBatchErrors, 1)
		}
	}
	if s.prepareRepository == nil {
		recordBatchMetrics(source, true)
		return nil, ErrProviderPrepareNotConfigured
	}
	if s.prepareSummaryRepository != nil {
		source = "repository_batch"
		batched, err := s.prepareSummaryRepository.ListLatestProviderPrepareSummaries(ctx, providerIDs)
		recordBatchMetrics(source, err != nil)
		if err == nil {
			s.logger.Debug("Listed latest provider prepare summaries",
				zap.Int("provider_count", len(providerIDs)),
				zap.Int("summary_count", len(batched)),
				zap.String("source", source),
				zap.Duration("duration", time.Since(startedAt)),
			)
		}
		return batched, err
	}
	source = "fallback_iterative"

	for _, providerID := range providerIDs {
		summary := &ProviderPrepareLatestSummary{ProviderID: providerID}
		runs, runsErr := s.prepareRepository.ListProviderPrepareRunsByProvider(ctx, providerID, 1, 0)
		if runsErr != nil {
			recordBatchMetrics(source, true)
			return nil, fmt.Errorf("failed to list provider prepare runs for provider %s: %w", providerID.String(), runsErr)
		}
		if len(runs) == 0 {
			summaries[providerID] = summary
			continue
		}
		run := runs[0]
		runID := run.ID
		runStatus := run.Status
		runUpdatedAt := run.UpdatedAt
		summary.RunID = &runID
		summary.Status = &runStatus
		summary.UpdatedAt = &runUpdatedAt
		summary.ErrorMessage = run.ErrorMessage

		checks, checksErr := s.prepareRepository.ListProviderPrepareRunChecks(ctx, run.ID, 200, 0)
		if checksErr != nil {
			recordBatchMetrics(source, true)
			return nil, fmt.Errorf("failed to list provider prepare run checks for provider %s run %s: %w", providerID.String(), run.ID.String(), checksErr)
		}
		if len(checks) > 0 {
			lastCheck := checks[len(checks)-1]
			checkCategory := lastCheck.Category
			checkSeverity := lastCheck.Severity
			summary.CheckCategory = &checkCategory
			summary.CheckSeverity = &checkSeverity
			if remediation, ok := lastCheck.Details["remediation"].(string); ok && strings.TrimSpace(remediation) != "" {
				trimmed := strings.TrimSpace(remediation)
				if len(trimmed) > 180 {
					trimmed = trimmed[:180] + "..."
				}
				summary.RemediationHint = &trimmed
			}
		}

		summaries[providerID] = summary
	}
	recordBatchMetrics(source, false)

	s.logger.Debug("Listed latest provider prepare summaries",
		zap.Int("provider_count", len(providerIDs)),
		zap.Int("summary_count", len(summaries)),
		zap.String("source", source),
		zap.Duration("duration", time.Since(startedAt)),
	)
	return summaries, nil
}

func (s *Service) executeProviderPrepareRun(ctx context.Context, runID uuid.UUID) {
	if s.prepareRepository == nil {
		return
	}
	finalized := false
	failRun := func(message string, summary map[string]interface{}) {
		if finalized {
			return
		}
		completedAt := time.Now().UTC()
		msg := strings.TrimSpace(message)
		if msg == "" {
			msg = "provider prepare run terminated unexpectedly"
		}
		_ = s.prepareRepository.UpdateProviderPrepareRunStatus(ctx, runID, ProviderPrepareRunStatusFailed, nil, &completedAt, &msg, summary)
		finalized = true
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			failRun(fmt.Sprintf("provider prepare panic: %v", recovered), map[string]interface{}{
				"stage":  "panic",
				"status": "failed",
			})
		} else if !finalized {
			failRun("provider prepare exited without terminal status", map[string]interface{}{
				"stage":  "unexpected_exit",
				"status": "failed",
			})
		}
	}()

	now := time.Now().UTC()
	_ = s.prepareRepository.UpdateProviderPrepareRunStatus(ctx, runID, ProviderPrepareRunStatusRunning, &now, nil, nil, map[string]interface{}{
		"stage": "running",
	})

	run, err := s.prepareRepository.GetProviderPrepareRun(ctx, runID)
	if err != nil || run == nil {
		return
	}
	provider, err := s.repository.FindProviderByID(ctx, run.ProviderID)
	if err != nil || provider == nil {
		msg := "provider not found for prepare run"
		completedAt := time.Now().UTC()
		_ = s.prepareRepository.UpdateProviderPrepareRunStatus(ctx, runID, ProviderPrepareRunStatusFailed, nil, &completedAt, &msg, map[string]interface{}{
			"stage": "provider_lookup",
		})
		finalized = true
		return
	}

	addCheck := func(checkKey, category, severity string, ok bool, message string, details map[string]interface{}) {
		_ = s.prepareRepository.AddProviderPrepareRunCheck(ctx, &ProviderPrepareRunCheck{
			ID:        uuid.New(),
			RunID:     runID,
			CheckKey:  checkKey,
			Category:  category,
			Severity:  severity,
			OK:        ok,
			Message:   message,
			Details:   details,
			CreatedAt: time.Now().UTC(),
		})
	}

	resultSummary := map[string]interface{}{
		"connectivity":     "pending",
		"permission_audit": "pending",
		"bootstrap":        "pending",
		"readiness":        "pending",
	}
	forceRuntimeAuthRegeneration := requestedActionEnabled(run.RequestedActions, "runtime_auth_regeneration")
	if forceRuntimeAuthRegeneration {
		resultSummary["runtime_auth_regeneration"] = "requested"
	}
	failedChecks := []string{}

	var k8sClient kubernetes.Interface
	connectivityReady := false
	connectivityAuthContext := connectors.AuthContextRuntime
	if normalizeBootstrapMode(provider.BootstrapMode) == "image_factory_managed" {
		connectivityAuthContext = connectors.AuthContextBootstrap
	}
	restConfig, cfgErr := connectors.BuildRESTConfigFromProviderConfigForContext(provider.Config, connectivityAuthContext)
	if cfgErr != nil {
		msg := fmt.Sprintf("invalid provider config: %v", cfgErr)
		addCheck("provider_config", "connectivity", "error", false, msg, nil)
		failedChecks = append(failedChecks, msg)
		resultSummary["connectivity"] = "failed"
	} else {
		addCheck("provider_config", "connectivity", "info", true, "provider config resolved", map[string]interface{}{
			"auth_context": connectivityAuthContext,
		})
		client, clientErr := kubernetes.NewForConfig(restConfig)
		if clientErr != nil {
			msg := fmt.Sprintf("failed to create kubernetes client: %v", clientErr)
			addCheck("kubernetes_client", "connectivity", "error", false, msg, nil)
			failedChecks = append(failedChecks, msg)
			resultSummary["connectivity"] = "failed"
		} else {
			k8sClient = client
			start := time.Now()
			version, versionErr := k8sClient.Discovery().ServerVersion()
			if versionErr != nil {
				msg := fmt.Sprintf("kubernetes API unreachable: %v", versionErr)
				addCheck("kubernetes_api", "connectivity", "error", false, msg, nil)
				failedChecks = append(failedChecks, msg)
				resultSummary["connectivity"] = "failed"
			} else {
				addCheck("kubernetes_api", "connectivity", "info", true, "kubernetes API reachable", map[string]interface{}{
					"git_version": version.GitVersion,
					"latency_ms":  time.Since(start).Milliseconds(),
				})
				resultSummary["connectivity"] = "passed"
				connectivityReady = true
			}
		}
	}

	targetNamespace := resolveTektonTargetNamespace(provider, run.TenantID)
	if provider.TargetNamespace != nil && strings.TrimSpace(*provider.TargetNamespace) != "" {
		targetNamespace = strings.TrimSpace(*provider.TargetNamespace)
	}

	if connectivityReady && k8sClient != nil {
		permissionClient := k8sClient
		if provider.BootstrapMode == "image_factory_managed" {
			bootstrapConfig, bootstrapCfgErr := connectors.BuildRESTConfigFromProviderConfigForContext(provider.Config, connectors.AuthContextBootstrap)
			if bootstrapCfgErr != nil {
				msg := fmt.Sprintf("invalid bootstrap auth config: %v", bootstrapCfgErr)
				addCheck("bootstrap_auth_config", "bootstrap", "error", false, msg, nil)
				failedChecks = append(failedChecks, msg)
				resultSummary["permission_audit"] = "failed"
			} else {
				bootstrapClient, bootstrapClientErr := kubernetes.NewForConfig(bootstrapConfig)
				if bootstrapClientErr != nil {
					msg := fmt.Sprintf("failed to create bootstrap kubernetes client: %v", bootstrapClientErr)
					addCheck("bootstrap_kubernetes_client", "bootstrap", "error", false, msg, nil)
					failedChecks = append(failedChecks, msg)
					resultSummary["permission_audit"] = "failed"
				} else {
					permissionClient = bootstrapClient
					addCheck("bootstrap_auth_config", "bootstrap", "info", true, "bootstrap auth config resolved", nil)
				}
			}
		}

		permissionFailed := false
		specs := permissionAuditSpecs(provider.BootstrapMode, targetNamespace)
		for _, spec := range specs {
			allowed, reason, err := evaluatePermissionSpec(ctx, permissionClient, spec)
			details := map[string]interface{}{
				"verb":        spec.Verb,
				"group":       spec.Group,
				"resource":    spec.Resource,
				"namespace":   spec.Namespace,
				"remediation": spec.Remediation,
			}
			if err != nil {
				msg := fmt.Sprintf("%s: failed to evaluate permission (%v)", spec.Key, err)
				addCheck(spec.Key, "permission_audit", "error", false, msg, details)
				failedChecks = append(failedChecks, msg)
				permissionFailed = true
				continue
			}
			if !allowed {
				msg := fmt.Sprintf("%s: missing permission (%s)", spec.Key, reason)
				addCheck(spec.Key, "permission_audit", "error", false, msg, details)
				failedChecks = append(failedChecks, msg)
				permissionFailed = true
				continue
			}
			addCheck(spec.Key, "permission_audit", "info", true, "permission check passed", details)
		}
		if permissionFailed {
			resultSummary["permission_audit"] = "failed"
		} else {
			resultSummary["permission_audit"] = "passed"
		}
		if !permissionFailed && provider.BootstrapMode == "image_factory_managed" {
			bootstrapResult, bootstrapErr := s.runManagedBootstrapForPrepareWithOptions(
				ctx,
				provider,
				run.TenantID,
				forceRuntimeAuthRegeneration,
			)
			if bootstrapErr != nil {
				msg := fmt.Sprintf("managed bootstrap failed: %v", bootstrapErr)
				addCheck("bootstrap.apply", "bootstrap", "error", false, msg, map[string]interface{}{
					"target_namespace":     targetNamespace,
					"remediation_commands": bootstrapRemediationCommands(targetNamespace),
				})
				failedChecks = append(failedChecks, msg)
				resultSummary["bootstrap"] = "failed"
			} else {
				addCheck("bootstrap.apply", "bootstrap", "info", true, "managed bootstrap completed", bootstrapResult)
				resultSummary["bootstrap"] = "passed"
				if forceRuntimeAuthRegeneration {
					resultSummary["runtime_auth_regeneration"] = "applied"
				}
			}
		}
	} else {
		resultSummary["permission_audit"] = "failed"
		msg := "permission audit skipped because connectivity failed"
		addCheck("permission_audit", "permission_audit", "error", false, msg, nil)
		failedChecks = append(failedChecks, msg)
	}

	if provider.BootstrapMode == "image_factory_managed" && provider.CredentialScope == "read_only" {
		msg := "bootstrap mode image_factory_managed requires elevated credentials; credential_scope=read_only"
		addCheck("bootstrap_credentials", "bootstrap", "error", false, msg, nil)
		failedChecks = append(failedChecks, msg)
		resultSummary["bootstrap"] = "failed"
	} else {
		if _, ok := resultSummary["bootstrap"]; !ok {
			resultSummary["bootstrap"] = "pending"
		}
		addCheck("bootstrap_mode", "bootstrap", "info", true, fmt.Sprintf("bootstrap mode: %s", provider.BootstrapMode), map[string]interface{}{
			"credential_scope": provider.CredentialScope,
		})
		if provider.BootstrapMode == "self_managed" {
			resultSummary["bootstrap"] = "skipped"
		} else if resultSummary["bootstrap"] == "pending" {
			resultSummary["bootstrap"] = "assumed"
		}
	}

	if connectivityReady {
		readinessStatus, missingPrereqs, readinessErr := s.refreshProviderReadinessAfterInstall(ctx, &TektonInstallerJob{
			ProviderID: provider.ID,
			TenantID:   run.TenantID,
		})
		if readinessErr != nil {
			msg := fmt.Sprintf("readiness evaluation failed: %v", readinessErr)
			addCheck("readiness_eval", "readiness", "error", false, msg, map[string]interface{}{
				"target_namespace": targetNamespace,
			})
			failedChecks = append(failedChecks, msg)
			resultSummary["readiness"] = "failed"
		} else if readinessStatus != "ready" {
			msg := "provider readiness is not ready"
			addCheck("readiness_eval", "readiness", "error", false, msg, map[string]interface{}{
				"status":           readinessStatus,
				"missing_prereqs":  missingPrereqs,
				"target_namespace": targetNamespace,
			})
			for _, missing := range missingPrereqs {
				failedChecks = append(failedChecks, missing)
			}
			resultSummary["readiness"] = "failed"
		} else {
			addCheck("readiness_eval", "readiness", "info", true, "provider readiness is ready", map[string]interface{}{
				"target_namespace": targetNamespace,
			})
			resultSummary["readiness"] = "passed"
		}
	} else {
		resultSummary["readiness"] = "failed"
		msg := "readiness evaluation skipped because connectivity failed"
		addCheck("readiness_eval", "readiness", "error", false, msg, nil)
		failedChecks = append(failedChecks, msg)
	}

	finalStatus := ProviderPrepareRunStatusSucceeded
	var errorMessage *string
	if len(failedChecks) > 0 {
		finalStatus = ProviderPrepareRunStatusFailed
		msg := strings.Join(failedChecks, "; ")
		errorMessage = &msg
		resultSummary["status"] = "failed"
	} else {
		resultSummary["status"] = "succeeded"
	}

	if normalizeBootstrapMode(provider.BootstrapMode) == "image_factory_managed" {
		reconcileSummary, err := s.triggerTenantNamespacePrepareForProviderTenants(ctx, provider.ID, run.TenantID, &run.RequestedBy)
		if reconcileSummary != nil {
			resultSummary["tenant_namespace_catchup"] = reconcileSummary
		}
		if err != nil {
			finalStatus = ProviderPrepareRunStatusFailed
			msg := fmt.Sprintf("tenant namespace catch-up failed: %v", err)
			errorMessage = &msg
			resultSummary["status"] = "failed"
			failedChecks = append(failedChecks, msg)
			addCheck("tenant_namespace_catchup", "bootstrap", "error", false, msg, reconcileSummary)
		} else {
			addCheck("tenant_namespace_catchup", "bootstrap", "info", true, "tenant namespace catch-up completed", reconcileSummary)
		}
	}

	completedAt := time.Now().UTC()
	_ = s.prepareRepository.UpdateProviderPrepareRunStatus(ctx, runID, finalStatus, nil, &completedAt, errorMessage, resultSummary)
	finalized = true

	// On failed prepare, force provider non-ready/non-schedulable.
	if finalStatus == ProviderPrepareRunStatusFailed {
		_ = s.UpdateProviderReadiness(ctx, provider.ID, "not_ready", completedAt, failedChecks)
		return
	}
}

type permissionSpec struct {
	Key         string
	Group       string
	Resource    string
	Verb        string
	Namespace   string
	Remediation string
}

func permissionAuditSpecs(bootstrapMode, targetNamespace string) []permissionSpec {
	specs := []permissionSpec{
		{
			Key:         "perm.tasks.get",
			Group:       "tekton.dev",
			Resource:    "tasks",
			Verb:        "get",
			Namespace:   targetNamespace,
			Remediation: "grant get/list/watch on tekton tasks in target namespace",
		},
		{
			Key:         "perm.pipelines.get",
			Group:       "tekton.dev",
			Resource:    "pipelines",
			Verb:        "get",
			Namespace:   targetNamespace,
			Remediation: "grant get/list/watch on tekton pipelines in target namespace",
		},
		{
			Key:         "perm.pipelineruns.create",
			Group:       "tekton.dev",
			Resource:    "pipelineruns",
			Verb:        "create",
			Namespace:   targetNamespace,
			Remediation: "grant create/get/list/watch on tekton pipelineruns in target namespace",
		},
		{
			Key:         "perm.secrets.get",
			Group:       "",
			Resource:    "secrets",
			Verb:        "get",
			Namespace:   targetNamespace,
			Remediation: "grant get/list/watch on secrets in target namespace",
		},
		{
			Key:         "perm.pods.create",
			Group:       "",
			Resource:    "pods",
			Verb:        "create",
			Namespace:   targetNamespace,
			Remediation: "grant create/get/list/watch on pods in target namespace",
		},
		// Runtime resources commonly required by Tekton build executions
		{
			Key:         "perm.configmaps.get",
			Group:       "",
			Resource:    "configmaps",
			Verb:        "get",
			Namespace:   targetNamespace,
			Remediation: "grant get/list/watch on configmaps in target namespace",
		},
		{
			Key:         "perm.persistentvolumeclaims.create",
			Group:       "",
			Resource:    "persistentvolumeclaims",
			Verb:        "create",
			Namespace:   targetNamespace,
			Remediation: "grant create/get/list on persistentvolumeclaims in target namespace",
		},
	}

	// Additional write/create permissions required only for managed bootstrap
	if bootstrapMode == "image_factory_managed" {
		specs = append(specs,
			permissionSpec{
				Key:         "perm.namespaces.create",
				Group:       "",
				Resource:    "namespaces",
				Verb:        "create",
				Remediation: "grant create/get/list on namespaces for managed bootstrap",
			},
			permissionSpec{
				Key:         "perm.tasks.patch",
				Group:       "tekton.dev",
				Resource:    "tasks",
				Verb:        "patch",
				Namespace:   targetNamespace,
				Remediation: "grant patch/update on tekton tasks in target namespace for managed bootstrap",
			},
			permissionSpec{
				Key:         "perm.pipelines.patch",
				Group:       "tekton.dev",
				Resource:    "pipelines",
				Verb:        "patch",
				Namespace:   targetNamespace,
				Remediation: "grant patch/update on tekton pipelines in target namespace for managed bootstrap",
			},
			// RBAC and runtime identity provisioning
			permissionSpec{
				Key:         "perm.roles.create",
				Group:       "rbac.authorization.k8s.io",
				Resource:    "roles",
				Verb:        "create",
				Namespace:   targetNamespace,
				Remediation: "grant create/patch on Role resources in target namespace for managed bootstrap",
			},
			permissionSpec{
				Key:         "perm.rolebindings.create",
				Group:       "rbac.authorization.k8s.io",
				Resource:    "rolebindings",
				Verb:        "create",
				Namespace:   targetNamespace,
				Remediation: "grant create/patch on RoleBinding resources in target namespace for managed bootstrap",
			},
			permissionSpec{
				Key:         "perm.serviceaccounts.create",
				Group:       "",
				Resource:    "serviceaccounts",
				Verb:        "create",
				Namespace:   targetNamespace,
				Remediation: "grant create/get on ServiceAccount resources in target namespace for managed bootstrap",
			},
		)
	}

	return specs
}

func evaluatePermissionSpec(ctx context.Context, k8sClient kubernetes.Interface, spec permissionSpec) (bool, string, error) {
	if k8sClient == nil {
		return false, "kubernetes client unavailable", errors.New("kubernetes client unavailable")
	}
	review := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: spec.Namespace,
				Verb:      spec.Verb,
				Group:     spec.Group,
				Resource:  spec.Resource,
			},
		},
	}
	result, err := k8sClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return false, "", err
	}
	if result.Status.Allowed {
		return true, "allowed", nil
	}
	reason := result.Status.Reason
	if reason == "" {
		reason = "access denied"
	}
	return false, reason, nil
}

func (s *Service) runManagedBootstrapForPrepare(ctx context.Context, provider *Provider, tenantID uuid.UUID) (map[string]interface{}, error) {
	return s.runManagedBootstrapForPrepareWithOptions(ctx, provider, tenantID, false)
}

func (s *Service) runManagedBootstrapForPrepareWithOptions(
	ctx context.Context,
	provider *Provider,
	tenantID uuid.UUID,
	forceRuntimeAuthRegeneration bool,
) (map[string]interface{}, error) {
	restConfig, err := connectors.BuildRESTConfigFromProviderConfigForContext(provider.Config, connectors.AuthContextBootstrap)
	if err != nil {
		return nil, fmt.Errorf("invalid provider bootstrap auth config: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	targetNamespace := resolveTektonTargetNamespace(provider, tenantID)
	if provider.TargetNamespace != nil && strings.TrimSpace(*provider.TargetNamespace) != "" {
		targetNamespace = strings.TrimSpace(*provider.TargetNamespace)
	}
	presentResources, installDetails, err := s.ensureTektonAPIsOrInstall(ctx, provider, targetNamespace, dynamicClient, discoveryClient)
	if err != nil {
		if installDetails != nil {
			return nil, fmt.Errorf("tekton API preflight failed after install attempt (%v): %w", installDetails, err)
		}
		return nil, fmt.Errorf("tekton API preflight failed: %w", err)
	}
	if installDetails != nil {
		if waitErr := s.waitForTektonWebhookReady(ctx, k8sClient, provider, targetNamespace, 90*time.Second); waitErr != nil {
			return nil, fmt.Errorf("tekton webhook readiness check failed: %w", waitErr)
		}
	}

	if err := ensureNamespace(ctx, k8sClient, targetNamespace); err != nil {
		return nil, fmt.Errorf("failed to ensure target namespace %s: %w", targetNamespace, err)
	}

	runtimeProvisionResult, err := s.ensureManagedRuntimeIdentityAndPersistConfig(
		ctx,
		provider,
		k8sClient,
		targetNamespace,
		restConfig.Host,
		restConfig.TLSClientConfig.CAData,
		forceRuntimeAuthRegeneration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure managed runtime identity: %w", err)
	}

	assetRoot, err := s.resolveTektonAssetRootDir(ctx)
	if err != nil {
		return nil, err
	}
	profileVersion := resolveTektonAssetVersion(provider, "")
	resourceFiles, err := loadKustomizationResourceFilesForProfile(assetRoot, profileVersion)
	if err != nil {
		return nil, err
	}
	if len(resourceFiles) == 0 {
		return nil, fmt.Errorf("tekton kustomization has no resources under %s", assetRoot)
	}

	applied := make([]string, 0, len(resourceFiles))
	cleanupPolicy := s.tektonHistoryCleanupConfig(ctx)
	tektonTaskImageConfig := s.getTektonTaskImagesConfig(ctx)
	internalRegistryStorageProfile := s.resolveInternalRegistryStorageProfile(ctx)
	systemNamespace := resolveProviderSystemNamespace(provider)

	for _, resourceFile := range resourceFiles {
		objects, parseErr := parseUnstructuredYAMLFile(resourceFile)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse manifest %s: %w", resourceFile, parseErr)
		}
		applyTektonCleanupPolicyToObjects(objects, cleanupPolicy)
		applyTektonTaskImageOverrides(objects, tektonTaskImageConfig)
		objects = applyStorageProfilesToObjects(objects, internalRegistryStorageProfile)
		applyNamespace := targetNamespace
		if tektonResourceScope(resourceFile) == tektonResourceScopeShared {
			applyNamespace = systemNamespace
			if strings.TrimSpace(applyNamespace) == "" {
				applyNamespace = targetNamespace
			}
			if err := ensureNamespace(ctx, k8sClient, applyNamespace); err != nil {
				return nil, fmt.Errorf("failed to ensure shared asset namespace %s: %w", applyNamespace, err)
			}
		}
		if strings.HasSuffix(filepath.ToSlash(resourceFile), "/jobs/v1/internal-registry-deployment.yaml") {
			if _, reconcileErr := ensureInternalRegistryDeploymentStorageMode(ctx, k8sClient, applyNamespace, internalRegistryStorageProfile); reconcileErr != nil {
				return nil, fmt.Errorf("failed to reconcile internal registry deployment storage mode in namespace %s: %w", applyNamespace, reconcileErr)
			}
		}
		appliedObjects, applyErr := applyUnstructuredObjects(ctx, dynamicClient, discoveryClient, objects, applyNamespace, "image-factory-prepare")
		if applyErr != nil {
			return nil, fmt.Errorf("failed to apply manifests from %s: %w", resourceFile, applyErr)
		}
		applied = append(applied, appliedObjects...)
	}
	if _, aliasErr := s.ensureTenantInternalRegistryAliasService(ctx, k8sClient, targetNamespace, systemNamespace); aliasErr != nil {
		s.logger.Warn("Failed to ensure internal registry alias in target namespace",
			zap.String("provider_id", provider.ID.String()),
			zap.String("target_namespace", targetNamespace),
			zap.String("system_namespace", systemNamespace),
			zap.Error(aliasErr))
	}
	if _, warmupErr := s.triggerTrivyDBWarmupIfNeeded(ctx, k8sClient, systemNamespace); warmupErr != nil {
		s.logger.Warn("Failed to trigger trivy DB warmup after managed bootstrap",
			zap.String("provider_id", provider.ID.String()),
			zap.String("system_namespace", systemNamespace),
			zap.Error(warmupErr))
	}

	return map[string]interface{}{
		"target_namespace": targetNamespace,
		"runtime_auth":     runtimeProvisionResult,
		"asset_root":       assetRoot,
		"resource_files":   resourceFiles,
		"applied_objects":  applied,
		"tekton_resources": presentResources,
		"core_install":     installDetails,
	}, nil
}

func bootstrapRemediationCommands(targetNamespace string) []string {
	return []string{
		"kubectl auth can-i create namespaces",
		fmt.Sprintf("kubectl auth can-i patch tasks.tekton.dev -n %s", targetNamespace),
		fmt.Sprintf("kubectl auth can-i patch pipelines.tekton.dev -n %s", targetNamespace),
		fmt.Sprintf("kubectl auth can-i create pipelineruns.tekton.dev -n %s", targetNamespace),
		fmt.Sprintf("kubectl apply -k backend/tekton -n %s", targetNamespace),
	}
}

func (s *Service) StartTektonInstallerJob(
	ctx context.Context,
	providerID, tenantID, requestedBy uuid.UUID,
	req StartTektonInstallerJobRequest,
) (*TektonInstallerJob, error) {
	if s.installerRepository == nil {
		return nil, ErrTektonInstallerNotConfigured
	}
	if req.Operation == "" {
		return nil, ErrInvalidTektonInstallerRequest
	}
	switch req.Operation {
	case TektonInstallerOperationInstall, TektonInstallerOperationUpgrade, TektonInstallerOperationValidate:
	default:
		return nil, ErrInvalidTektonInstallerRequest
	}

	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	if tenantID != provider.TenantID {
		return nil, ErrPermissionDenied
	}

	if req.IdempotencyKey != "" {
		existingJob, err := s.installerRepository.FindInstallerJobByProviderAndIdempotencyKey(ctx, providerID, req.Operation, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing tekton installer job by idempotency key: %w", err)
		}
		if existingJob != nil {
			return existingJob, nil
		}
	}

	now := time.Now().UTC()
	job := &TektonInstallerJob{
		ID:           uuid.New(),
		ProviderID:   providerID,
		TenantID:     tenantID,
		RequestedBy:  requestedBy,
		Operation:    req.Operation,
		InstallMode:  resolveTektonInstallMode(provider, req.InstallMode),
		AssetVersion: resolveTektonAssetVersion(provider, req.AssetVersion),
		Status:       TektonInstallerJobStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.installerRepository.CreateInstallerJob(ctx, job); err != nil {
		if strings.Contains(err.Error(), "uq_tekton_installer_jobs_provider_active") {
			return nil, ErrTektonInstallerJobInProgress
		}
		return nil, fmt.Errorf("failed to create tekton installer job: %w", err)
	}

	details := map[string]interface{}{
		"operation": req.Operation,
	}
	if req.IdempotencyKey != "" {
		details["idempotency_key"] = req.IdempotencyKey
	}

	event := &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  string(req.Operation) + ".requested",
		Message:    fmt.Sprintf("Tekton %s requested for provider", req.Operation),
		Details:    details,
		CreatedBy:  &requestedBy,
		CreatedAt:  now,
	}
	if err := s.installerRepository.AddInstallerJobEvent(ctx, event); err != nil {
		s.logger.Warn("Failed to add tekton installer job event",
			zap.String("job_id", job.ID.String()),
			zap.Error(err))
	}

	return job, nil
}

func (s *Service) GetTektonInstallerStatus(ctx context.Context, providerID uuid.UUID, limit int) (*TektonInstallerStatus, error) {
	if s.installerRepository == nil {
		return nil, ErrTektonInstallerNotConfigured
	}
	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}
	if limit <= 0 {
		limit = 20
	}

	jobs, err := s.installerRepository.ListInstallerJobsByProvider(ctx, providerID, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list tekton installer jobs: %w", err)
	}

	status := &TektonInstallerStatus{
		ProviderID: providerID,
		RecentJobs: jobs,
	}

	for _, job := range jobs {
		if job == nil {
			continue
		}
		if job.Status == TektonInstallerJobStatusPending || job.Status == TektonInstallerJobStatusRunning {
			status.ActiveJob = job
			break
		}
	}

	if status.ActiveJob != nil {
		events, err := s.installerRepository.ListInstallerJobEvents(ctx, status.ActiveJob.ID, 200, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to list tekton installer job events: %w", err)
		}
		status.ActiveJobEvents = events
	}

	return status, nil
}

func (s *Service) RetryTektonInstallerJob(
	ctx context.Context,
	providerID, jobID, tenantID, requestedBy uuid.UUID,
) (*TektonInstallerJob, error) {
	if s.installerRepository == nil {
		return nil, ErrTektonInstallerNotConfigured
	}

	provider, err := s.repository.FindProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	if tenantID != provider.TenantID {
		return nil, ErrPermissionDenied
	}

	sourceJob, err := s.installerRepository.GetInstallerJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source installer job: %w", err)
	}
	if sourceJob == nil || sourceJob.ProviderID != providerID {
		return nil, ErrTektonInstallerJobNotFound
	}
	if sourceJob.Status != TektonInstallerJobStatusFailed {
		return nil, ErrTektonInstallerJobNotRetryable
	}

	op, err := s.getInstallerJobOperation(ctx, sourceJob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source job operation: %w", err)
	}

	retryJob, err := s.StartTektonInstallerJob(ctx, providerID, tenantID, requestedBy, StartTektonInstallerJobRequest{
		Operation:    op,
		InstallMode:  sourceJob.InstallMode,
		AssetVersion: sourceJob.AssetVersion,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      retryJob.ID,
		ProviderID: retryJob.ProviderID,
		TenantID:   retryJob.TenantID,
		EventType:  "job.retry.requested",
		Message:    "Installer job created from failed job retry",
		Details: map[string]interface{}{
			"source_job_id": sourceJob.ID.String(),
			"operation":     op,
		},
		CreatedBy: &requestedBy,
		CreatedAt: now,
	})

	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      sourceJob.ID,
		ProviderID: sourceJob.ProviderID,
		TenantID:   sourceJob.TenantID,
		EventType:  "job.retry.spawned",
		Message:    "Retry job created from this failed installer job",
		Details: map[string]interface{}{
			"retry_job_id": retryJob.ID.String(),
		},
		CreatedBy: &requestedBy,
		CreatedAt: now,
	})

	return retryJob, nil
}

func (s *Service) RunNextTektonInstallerJob(ctx context.Context) (bool, error) {
	if s.installerRepository == nil {
		return false, ErrTektonInstallerNotConfigured
	}

	job, err := s.installerRepository.ClaimNextPendingInstallerJob(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to claim installer job: %w", err)
	}
	if job == nil {
		return false, nil
	}

	now := time.Now().UTC()
	startedEvent := &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  "job.started",
		Message:    "Installer job execution started",
		Details: map[string]interface{}{
			"status": "running",
		},
		CreatedAt: now,
	}
	_ = s.installerRepository.AddInstallerJobEvent(ctx, startedEvent)

	op, opErr := s.getInstallerJobOperation(ctx, job.ID)
	if opErr != nil {
		message := opErr.Error()
		completedAt := time.Now().UTC()
		_ = s.installerRepository.UpdateInstallerJobStatus(ctx, job.ID, TektonInstallerJobStatusFailed, nil, &completedAt, &message)
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  "job.failed",
			Message:    "Installer job failed to resolve requested operation",
			Details: map[string]interface{}{
				"error": message,
			},
			CreatedAt: completedAt,
		})
		return true, nil
	}

	runErr := s.executeInstallerJob(ctx, job, op)
	if runErr != nil {
		message := runErr.Error()
		checkedAt := time.Now().UTC()
		failureMissingPrereqs := []string{fmt.Sprintf("tekton %s failed: %s", op, message)}
		if readinessErr := s.UpdateProviderReadiness(ctx, job.ProviderID, "not_ready", checkedAt, failureMissingPrereqs); readinessErr != nil {
			s.logger.Warn("Failed to persist provider readiness after installer failure",
				zap.String("provider_id", job.ProviderID.String()),
				zap.String("job_id", job.ID.String()),
				zap.Error(readinessErr))
		}
		completedAt := time.Now().UTC()
		_ = s.installerRepository.UpdateInstallerJobStatus(ctx, job.ID, TektonInstallerJobStatusFailed, nil, &completedAt, &message)
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  "job.failed",
			Message:    "Installer job failed",
			Details: map[string]interface{}{
				"operation": op,
				"error":     message,
			},
			CreatedAt: completedAt,
		})
		return true, nil
	}

	if op == TektonInstallerOperationInstall || op == TektonInstallerOperationUpgrade || op == TektonInstallerOperationValidate {
		readinessStatus, missingPrereqs, readinessErr := s.refreshProviderReadinessAfterInstall(ctx, job)
		eventType := "job.readiness.summary"
		message := "Post-operation readiness check completed"
		details := map[string]interface{}{
			"status":          readinessStatus,
			"missing_prereqs": missingPrereqs,
		}
		if readinessErr != nil {
			eventType = "job.readiness.failed"
			message = "Post-operation readiness check failed"
			details["error"] = readinessErr.Error()
		}
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  eventType,
			Message:    message,
			Details:    details,
			CreatedAt:  time.Now().UTC(),
		})
	}

	completedAt := time.Now().UTC()
	if err := s.installerRepository.UpdateInstallerJobStatus(ctx, job.ID, TektonInstallerJobStatusSucceeded, nil, &completedAt, nil); err != nil {
		return true, fmt.Errorf("failed to mark installer job succeeded: %w", err)
	}
	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  "job.succeeded",
		Message:    "Installer job completed successfully",
		Details: map[string]interface{}{
			"operation": op,
		},
		CreatedAt: completedAt,
	})
	return true, nil
}

func (s *Service) getInstallerJobOperation(ctx context.Context, jobID uuid.UUID) (TektonInstallerOperation, error) {
	events, err := s.installerRepository.ListInstallerJobEvents(ctx, jobID, 1, 0)
	if err != nil {
		return "", fmt.Errorf("failed to load installer job events: %w", err)
	}
	if len(events) == 0 || events[0] == nil {
		return "", errors.New("missing installer job request event")
	}

	event := events[0]
	if event.Details != nil {
		if raw, ok := event.Details["operation"].(string); ok {
			op := TektonInstallerOperation(strings.ToLower(raw))
			switch op {
			case TektonInstallerOperationInstall, TektonInstallerOperationUpgrade, TektonInstallerOperationValidate:
				return op, nil
			}
		}
	}

	if strings.HasSuffix(event.EventType, ".requested") {
		op := TektonInstallerOperation(strings.TrimSuffix(strings.ToLower(event.EventType), ".requested"))
		switch op {
		case TektonInstallerOperationInstall, TektonInstallerOperationUpgrade, TektonInstallerOperationValidate:
			return op, nil
		}
	}
	return "", fmt.Errorf("unsupported installer job operation from event type %q", event.EventType)
}

func (s *Service) executeInstallerJob(ctx context.Context, job *TektonInstallerJob, op TektonInstallerOperation) error {
	switch op {
	case TektonInstallerOperationValidate:
		return s.executeValidateInstallerJob(ctx, job)
	case TektonInstallerOperationInstall, TektonInstallerOperationUpgrade:
		return s.executeInstallOrUpgradeInstallerJob(ctx, job, op)
	default:
		return fmt.Errorf("unsupported installer operation %q", op)
	}
}

func (s *Service) executeValidateInstallerJob(ctx context.Context, job *TektonInstallerJob) error {
	provider, err := s.repository.FindProviderByID(ctx, job.ProviderID)
	if err != nil {
		return fmt.Errorf("failed to load provider: %w", err)
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	connector, err := s.connectorFactory.CreateConnector(string(provider.ProviderType), provider.Config)
	if err != nil {
		return fmt.Errorf("failed to create provider connector: %w", err)
	}

	result, err := connector.TestConnection(ctx)
	if err != nil {
		return fmt.Errorf("provider connection test failed: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("provider validation failed: %s", result.Message)
	}

	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  "validate.completed",
		Message:    "Provider connectivity validation succeeded",
		Details: map[string]interface{}{
			"message": result.Message,
			"details": result.Details,
		},
		CreatedAt: time.Now().UTC(),
	})
	return nil
}

func (s *Service) executeInstallOrUpgradeInstallerJob(ctx context.Context, job *TektonInstallerJob, op TektonInstallerOperation) error {
	mode := string(job.InstallMode)
	if mode == "" {
		mode = string(TektonInstallModeImageFactoryInstaller)
	}
	resolvedMode := TektonInstallMode(mode)

	switch resolvedMode {
	case TektonInstallModeGitOps:
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  string(op) + ".delegated",
			Message:    "GitOps mode selected; Image Factory does not apply Tekton assets directly",
			Details: map[string]interface{}{
				"install_mode":   resolvedMode,
				"asset_version":  job.AssetVersion,
				"manual_action":  "sync assets via GitOps controller",
				"operation_type": op,
			},
			CreatedAt: time.Now().UTC(),
		})
		return nil
	case TektonInstallModeImageFactoryInstaller:
		return s.applyTektonAssetsToProvider(ctx, job, op)
	default:
		return fmt.Errorf("%s operation is not implemented for install_mode=%s yet", op, mode)
	}
}

func (s *Service) applyTektonAssetsToProvider(ctx context.Context, job *TektonInstallerJob, op TektonInstallerOperation) error {
	provider, err := s.repository.FindProviderByID(ctx, job.ProviderID)
	if err != nil {
		return fmt.Errorf("failed to load provider: %w", err)
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfigForContext(provider.Config, connectors.AuthContextBootstrap)
	if err != nil {
		return fmt.Errorf("invalid provider bootstrap auth config: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  string(op) + ".preflight.started",
		Message:    "Running Tekton API preflight checks",
		Details: map[string]interface{}{
			"group_versions": []string{"tekton.dev/v1", "tekton.dev/v1beta1"},
		},
		CreatedAt: time.Now().UTC(),
	})
	targetNamespace := resolveTektonTargetNamespace(provider, job.TenantID)
	presentResources, installDetails, err := s.ensureTektonAPIsOrInstall(ctx, provider, targetNamespace, dynamicClient, discoveryClient)
	if err != nil {
		details := map[string]interface{}{
			"group_versions": []string{"tekton.dev/v1", "tekton.dev/v1beta1"},
			"error":          err.Error(),
		}
		if installDetails != nil {
			details["core_install"] = installDetails
		}
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  string(op) + ".preflight.failed",
			Message:    "Tekton API preflight failed",
			Details:    details,
			CreatedAt:  time.Now().UTC(),
		})
		return fmt.Errorf("tekton API preflight failed: %w", err)
	}
	if installDetails != nil {
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  string(op) + ".webhook.wait.started",
			Message:    "Waiting for Tekton webhook service endpoints",
			Details: map[string]interface{}{
				"service": "tekton-pipelines-webhook",
			},
			CreatedAt: time.Now().UTC(),
		})
		if waitErr := s.waitForTektonWebhookReady(ctx, k8sClient, provider, targetNamespace, 90*time.Second); waitErr != nil {
			_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
				ID:         uuid.New(),
				JobID:      job.ID,
				ProviderID: job.ProviderID,
				TenantID:   job.TenantID,
				EventType:  string(op) + ".webhook.wait.failed",
				Message:    "Tekton webhook readiness check failed",
				Details: map[string]interface{}{
					"error": waitErr.Error(),
				},
				CreatedAt: time.Now().UTC(),
			})
			return fmt.Errorf("tekton webhook readiness check failed: %w", waitErr)
		}
		_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
			ID:         uuid.New(),
			JobID:      job.ID,
			ProviderID: job.ProviderID,
			TenantID:   job.TenantID,
			EventType:  string(op) + ".webhook.wait.succeeded",
			Message:    "Tekton webhook service endpoints are ready",
			Details: map[string]interface{}{
				"service": "tekton-pipelines-webhook",
			},
			CreatedAt: time.Now().UTC(),
		})
	}
	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  string(op) + ".preflight.passed",
		Message:    "Tekton API preflight passed",
		Details: map[string]interface{}{
			"group_versions":     []string{"tekton.dev/v1", "tekton.dev/v1beta1"},
			"required_resources": requiredTektonResources(),
			"present_resources":  presentResources,
			"core_install":       installDetails,
		},
		CreatedAt: time.Now().UTC(),
	})
	if err := ensureNamespace(ctx, k8sClient, targetNamespace); err != nil {
		return fmt.Errorf("failed to ensure target namespace %s: %w", targetNamespace, err)
	}

	assetRoot, err := s.resolveTektonAssetRootDir(ctx)
	if err != nil {
		return err
	}
	assetVersion := resolveTektonAssetVersion(nil, job.AssetVersion)
	resourceFiles, err := loadKustomizationResourceFilesForProfile(assetRoot, assetVersion)
	if err != nil {
		return err
	}
	if len(resourceFiles) == 0 {
		return fmt.Errorf("tekton kustomization has no resources under %s", assetRoot)
	}

	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  string(op) + ".apply.started",
		Message:    "Applying Tekton asset manifests",
		Details: map[string]interface{}{
			"asset_root":       assetRoot,
			"target_namespace": targetNamespace,
			"resource_files":   resourceFiles,
		},
		CreatedAt: time.Now().UTC(),
	})

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	applied := make([]string, 0, len(resourceFiles))
	cleanupPolicy := s.tektonHistoryCleanupConfig(ctx)
	tektonTaskImageConfig := s.getTektonTaskImagesConfig(ctx)
	internalRegistryStorageProfile := s.resolveInternalRegistryStorageProfile(ctx)

	for _, resourceFile := range resourceFiles {
		objects, parseErr := parseUnstructuredYAMLFile(resourceFile)
		if parseErr != nil {
			return fmt.Errorf("failed to parse manifest %s: %w", resourceFile, parseErr)
		}
		applyTektonCleanupPolicyToObjects(objects, cleanupPolicy)
		applyTektonTaskImageOverrides(objects, tektonTaskImageConfig)
		objects = applyStorageProfilesToObjects(objects, internalRegistryStorageProfile)
		if strings.HasSuffix(filepath.ToSlash(resourceFile), "/jobs/v1/internal-registry-deployment.yaml") {
			if _, reconcileErr := ensureInternalRegistryDeploymentStorageMode(ctx, k8sClient, targetNamespace, internalRegistryStorageProfile); reconcileErr != nil {
				return fmt.Errorf("failed to reconcile internal registry deployment storage mode in namespace %s: %w", targetNamespace, reconcileErr)
			}
		}
		for _, obj := range objects {
			if obj == nil {
				continue
			}
			mapping, mapErr := mapper.RESTMapping(schema.GroupKind{
				Group: obj.GroupVersionKind().Group,
				Kind:  obj.GetKind(),
			}, obj.GroupVersionKind().Version)
			if mapErr != nil {
				_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
					ID:         uuid.New(),
					JobID:      job.ID,
					ProviderID: job.ProviderID,
					TenantID:   job.TenantID,
					EventType:  string(op) + ".apply.failed",
					Message:    "Failed to map manifest resource",
					Details: map[string]interface{}{
						"resource_file": resourceFile,
						"kind":          obj.GetKind(),
						"name":          obj.GetName(),
						"error":         mapErr.Error(),
					},
					CreatedAt: time.Now().UTC(),
				})
				return fmt.Errorf("failed to map resource %s/%s: %w", obj.GetKind(), obj.GetName(), mapErr)
			}

			namespaceable := dynamicClient.Resource(mapping.Resource)
			var resourceInterface dynamic.ResourceInterface
			namespaced := mapping.Scope.Name() == metaRESTScopeNameNamespace
			if namespaced {
				ns := obj.GetNamespace()
				if ns == "" {
					ns = targetNamespace
					obj.SetNamespace(ns)
				}
				resourceInterface = namespaceable.Namespace(ns)
			} else {
				resourceInterface = namespaceable
			}

			objJSON, marshalErr := json.Marshal(obj.Object)
			if marshalErr != nil {
				_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
					ID:         uuid.New(),
					JobID:      job.ID,
					ProviderID: job.ProviderID,
					TenantID:   job.TenantID,
					EventType:  string(op) + ".apply.failed",
					Message:    "Failed to marshal manifest object",
					Details: map[string]interface{}{
						"resource_file": resourceFile,
						"kind":          obj.GetKind(),
						"name":          obj.GetName(),
						"error":         marshalErr.Error(),
					},
					CreatedAt: time.Now().UTC(),
				})
				return fmt.Errorf("failed to marshal %s/%s: %w", obj.GetKind(), obj.GetName(), marshalErr)
			}
			force := true
			if _, patchErr := resourceInterface.Patch(
				ctx,
				obj.GetName(),
				types.ApplyPatchType,
				objJSON,
				metav1.PatchOptions{
					FieldManager: "image-factory-tekton-installer",
					Force:        &force,
				},
			); patchErr != nil {
				_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
					ID:         uuid.New(),
					JobID:      job.ID,
					ProviderID: job.ProviderID,
					TenantID:   job.TenantID,
					EventType:  string(op) + ".apply.failed",
					Message:    "Failed to apply manifest object",
					Details: map[string]interface{}{
						"resource_file": resourceFile,
						"kind":          obj.GetKind(),
						"name":          obj.GetName(),
						"namespace":     obj.GetNamespace(),
						"error":         patchErr.Error(),
					},
					CreatedAt: time.Now().UTC(),
				})
				return fmt.Errorf("failed to apply %s/%s from %s: %w", obj.GetKind(), obj.GetName(), resourceFile, patchErr)
			}
			applied = append(applied, fmt.Sprintf("%s/%s", obj.GetKind(), obj.GetName()))
			_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
				ID:         uuid.New(),
				JobID:      job.ID,
				ProviderID: job.ProviderID,
				TenantID:   job.TenantID,
				EventType:  string(op) + ".apply.object",
				Message:    "Applied manifest object",
				Details: map[string]interface{}{
					"resource_file": resourceFile,
					"kind":          obj.GetKind(),
					"name":          obj.GetName(),
					"namespace":     obj.GetNamespace(),
				},
				CreatedAt: time.Now().UTC(),
			})
		}
	}

	_ = s.installerRepository.AddInstallerJobEvent(ctx, &TektonInstallerJobEvent{
		ID:         uuid.New(),
		JobID:      job.ID,
		ProviderID: job.ProviderID,
		TenantID:   job.TenantID,
		EventType:  string(op) + ".apply.completed",
		Message:    "Applied Tekton asset manifests successfully",
		Details: map[string]interface{}{
			"asset_root":       assetRoot,
			"target_namespace": targetNamespace,
			"applied_objects":  applied,
		},
		CreatedAt: time.Now().UTC(),
	})

	return nil
}

const metaRESTScopeNameNamespace = "namespace"
const tektonCoreInstallSourceManifest = "manifest"
const tektonCoreInstallSourceHelm = "helm"
const tektonCoreInstallSourcePreinstalled = "preinstalled"

var defaultTektonCoreManifestURLs = []string{
	"https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml",
}

func requiredTektonResources() []string {
	return []string{"tasks", "pipelines", "pipelineruns"}
}

func requiredTektonTaskNames() []string {
	return []string{"git-clone", "docker-build", "buildx", "kaniko-no-push", "scan-image", "generate-sbom", "push-image", "packer"}
}

func requiredTektonPipelineNamesForProvider(provider *Provider) []string {
	version := resolveTektonAssetVersion(provider, "")
	return []string{
		fmt.Sprintf("image-factory-build-%s-docker", version),
		fmt.Sprintf("image-factory-build-%s-buildx", version),
		fmt.Sprintf("image-factory-build-%s-kaniko", version),
		fmt.Sprintf("image-factory-build-%s-packer", version),
	}
}

func ensureTektonAPIsAvailable(discoveryClient discovery.DiscoveryInterface) ([]string, error) {
	present, missing, err := ensureTektonAPIsAvailableDetailed(discoveryClient)
	if err != nil {
		return present, err
	}
	if len(missing) > 0 {
		return present, fmt.Errorf("missing required Tekton resources: %s", strings.Join(missing, ", "))
	}
	return present, nil
}

func ensureTektonAPIsAvailableDetailed(discoveryClient discovery.DiscoveryInterface) ([]string, []string, error) {
	if discoveryClient == nil {
		return nil, nil, errors.New("discovery client is required")
	}

	// Tekton API version differs across clusters (v1 vs v1beta1). Prefer v1 and fall back to v1beta1.
	groupVersions := []string{"tekton.dev/v1", "tekton.dev/v1beta1"}
	var lastErr error
	for _, gv := range groupVersions {
		resourceList, err := discoveryClient.ServerResourcesForGroupVersion(gv)
		if err != nil {
			lastErr = err
			continue
		}
		present, missing := summarizeTektonResources(resourceList, requiredTektonResources())
		if len(missing) == 0 {
			return present, missing, nil
		}
		// Found a Tekton groupVersion but missing expected resources: treat as success with missing list.
		return present, missing, nil
	}

	return nil, requiredTektonResources(), lastErr
}

func (s *Service) ensureTektonAPIsOrInstall(
	ctx context.Context,
	provider *Provider,
	targetNamespace string,
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
) ([]string, map[string]interface{}, error) {
	present, missing, err := ensureTektonAPIsAvailableDetailed(discoveryClient)
	if err == nil && len(missing) == 0 {
		return present, nil, nil
	}

	installSource := s.resolveTektonCoreInstallSource(ctx, provider)
	if installSource == tektonCoreInstallSourcePreinstalled {
		if err != nil {
			return present, nil, fmt.Errorf("tekton API preflight failed (source=%s): %w", installSource, err)
		}
		return present, nil, fmt.Errorf("tekton API preflight failed (source=%s): missing resources: %s", installSource, strings.Join(missing, ", "))
	}

	if installSource == tektonCoreInstallSourceHelm {
		sys := s.getTektonCoreConfig(ctx)
		repo := defaultIfEmpty(configString(provider.Config, "tekton_helm_repo_url"), sysString(sys, func(c *systemconfig.TektonCoreConfig) string { return c.HelmRepoURL }))
		chart := defaultIfEmpty(
			defaultIfEmpty(configString(provider.Config, "tekton_helm_chart"), sysString(sys, func(c *systemconfig.TektonCoreConfig) string { return c.HelmChart })),
			"tekton-pipeline",
		)
		release := defaultIfEmpty(
			defaultIfEmpty(configString(provider.Config, "tekton_helm_release_name"), sysString(sys, func(c *systemconfig.TektonCoreConfig) string { return c.HelmReleaseName })),
			"tekton-pipelines",
		)
		ns := defaultIfEmpty(
			defaultIfEmpty(configString(provider.Config, "tekton_helm_namespace"), sysString(sys, func(c *systemconfig.TektonCoreConfig) string { return c.HelmNamespace })),
			"tekton-pipelines",
		)
		helmHint := "configure tekton_helm_repo_url and install Tekton core with Helm, then retry prepare"
		if repo != "" {
			helmHint = fmt.Sprintf("helm repo add tekton %s && helm upgrade --install %s tekton/%s -n %s --create-namespace", repo, release, chart, ns)
		}
		if err != nil {
			return present, map[string]interface{}{
				"install_source": installSource,
				"helm_repo_url":  repo,
				"helm_chart":     chart,
				"helm_release":   release,
				"helm_namespace": ns,
				"hint":           helmHint,
			}, fmt.Errorf("tekton APIs unavailable and source=%s does not auto-install in-process: %w", installSource, err)
		}
		return present, map[string]interface{}{
			"install_source": installSource,
			"helm_repo_url":  repo,
			"helm_chart":     chart,
			"helm_release":   release,
			"helm_namespace": ns,
			"hint":           helmHint,
		}, fmt.Errorf("tekton APIs missing and source=%s does not auto-install in-process: missing resources: %s", installSource, strings.Join(missing, ", "))
	}

	manifestURLs := s.resolveTektonCoreManifestURLs(ctx, provider)
	if len(manifestURLs) == 0 {
		return present, nil, errors.New("tekton core install source=manifest but no manifest URLs configured")
	}

	appliedObjects := make([]string, 0, 64)
	for _, manifestURL := range manifestURLs {
		objects, fetchErr := fetchAndParseUnstructuredYAML(ctx, manifestURL)
		if fetchErr != nil {
			return present, map[string]interface{}{
				"install_source": installSource,
				"manifest_url":   manifestURL,
			}, fmt.Errorf("failed to fetch tekton core manifest %s: %w", manifestURL, fetchErr)
		}
		applied, applyErr := applyUnstructuredObjects(ctx, dynamicClient, discoveryClient, objects, targetNamespace, "image-factory-tekton-core-installer")
		if applyErr != nil {
			return present, map[string]interface{}{
				"install_source": installSource,
				"manifest_url":   manifestURL,
			}, fmt.Errorf("failed to apply tekton core manifest %s: %w", manifestURL, applyErr)
		}
		appliedObjects = append(appliedObjects, applied...)
	}

	var lastErr error
	var lastMissing []string
	for i := 0; i < 15; i++ {
		present, lastMissing, lastErr = ensureTektonAPIsAvailableDetailed(discoveryClient)
		if lastErr == nil && len(lastMissing) == 0 {
			return present, map[string]interface{}{
				"install_source":  installSource,
				"manifest_urls":   manifestURLs,
				"applied_objects": appliedObjects,
			}, nil
		}
		time.Sleep(2 * time.Second)
	}

	if lastErr != nil {
		return present, map[string]interface{}{
			"install_source":  installSource,
			"manifest_urls":   manifestURLs,
			"applied_objects": appliedObjects,
		}, fmt.Errorf("tekton core installation completed but API preflight still failing: %w", lastErr)
	}
	return present, map[string]interface{}{
		"install_source":  installSource,
		"manifest_urls":   manifestURLs,
		"applied_objects": appliedObjects,
	}, fmt.Errorf("tekton core installation completed but required resources are still missing: %s", strings.Join(lastMissing, ", "))
}

func summarizeTektonResources(resourceList *metav1.APIResourceList, required []string) (present []string, missing []string) {
	if resourceList == nil {
		return []string{}, required
	}
	available := make(map[string]bool, len(resourceList.APIResources))
	for _, resource := range resourceList.APIResources {
		name := strings.ToLower(strings.TrimSpace(resource.Name))
		if name != "" {
			available[name] = true
		}
	}
	present = make([]string, 0, len(required))
	missing = make([]string, 0, len(required))
	for _, requiredName := range required {
		if available[strings.ToLower(requiredName)] {
			present = append(present, requiredName)
		} else {
			missing = append(missing, requiredName)
		}
	}
	sort.Strings(present)
	sort.Strings(missing)
	return present, missing
}

func (s *Service) refreshProviderReadinessAfterInstall(ctx context.Context, job *TektonInstallerJob) (string, []string, error) {
	provider, err := s.repository.FindProviderByID(ctx, job.ProviderID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load provider: %w", err)
	}
	if provider == nil {
		return "", nil, ErrProviderNotFound
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config)
	if err != nil {
		return "", nil, fmt.Errorf("invalid provider kubeconfig/config: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	tektonClient, err := tektonclient.NewForConfig(restConfig)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create tekton client: %w", err)
	}

	namespace := resolveTektonTargetNamespace(provider, job.TenantID)
	version, versionErr := tektoncompat.DetectAPIVersion(ctx, tektonClient, namespace)
	if versionErr != nil {
		return "", nil, fmt.Errorf("failed to detect tekton api version: %w", versionErr)
	}
	compat := tektoncompat.New(tektonClient, version)
	missing := make([]string, 0, 16)

	// Avoid cluster-scoped namespace GET here: runtime credentials may be intentionally namespace-scoped.
	// Probe existence using a namespaced API call; if namespace doesn't exist, this returns NotFound.
	if _, err := k8sClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{Limit: 1}); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			missing = append(missing, fmt.Sprintf("missing namespace: %s", namespace))
		case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
			missing = append(missing, fmt.Sprintf("namespace access denied for %s (secrets list forbidden)", namespace))
		default:
			missing = append(missing, fmt.Sprintf("namespace probe failed for %s: %v", namespace, err))
		}
	}

	for _, taskName := range requiredTektonTaskNames() {
		if err := compat.GetTask(ctx, namespace, taskName); err != nil {
			switch {
			case apierrors.IsNotFound(err):
				missing = append(missing, fmt.Sprintf("missing tekton task: %s in namespace %s", taskName, namespace))
			case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
				missing = append(missing, fmt.Sprintf("tekton task access denied: %s in namespace %s (%v)", taskName, namespace, err))
			default:
				missing = append(missing, fmt.Sprintf("failed to check tekton task %s in namespace %s (%v)", taskName, namespace, err))
			}
		}
	}

	for _, pipelineName := range requiredTektonPipelineNamesForProvider(provider) {
		if err := compat.GetPipeline(ctx, namespace, pipelineName); err != nil {
			switch {
			case apierrors.IsNotFound(err):
				missing = append(missing, fmt.Sprintf("missing tekton pipeline: %s in namespace %s", pipelineName, namespace))
			case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
				missing = append(missing, fmt.Sprintf("tekton pipeline access denied: %s in namespace %s (%v)", pipelineName, namespace, err))
			default:
				missing = append(missing, fmt.Sprintf("failed to check tekton pipeline %s in namespace %s (%v)", pipelineName, namespace, err))
			}
		}
	}

	status := "ready"
	if len(missing) > 0 {
		status = "not_ready"
	}
	checkedAt := time.Now().UTC()
	if err := s.UpdateProviderReadiness(ctx, job.ProviderID, status, checkedAt, missing); err != nil {
		return status, missing, fmt.Errorf("failed to persist provider readiness: %w", err)
	}
	return status, missing, nil
}

func resolveTektonTargetNamespace(provider *Provider, tenantID uuid.UUID) string {
	if provider != nil && provider.TargetNamespace != nil {
		if ns := strings.TrimSpace(*provider.TargetNamespace); ns != "" {
			return ns
		}
	}
	if provider != nil && provider.Config != nil {
		if ns, ok := provider.Config["tekton_target_namespace"].(string); ok && ns != "" {
			return ns
		}
		if ns, ok := provider.Config["system_namespace"].(string); ok && strings.TrimSpace(ns) != "" {
			return strings.TrimSpace(ns)
		}
	}

	effectiveTenantID := tenantID
	if provider != nil && provider.TenantID != uuid.Nil {
		effectiveTenantID = provider.TenantID
	}
	if effectiveTenantID == uuid.Nil {
		return "image-factory-default"
	}
	return fmt.Sprintf("image-factory-%s", effectiveTenantID.String()[:8])
}

func ensureNamespace(ctx context.Context, k8sClient kubernetes.Interface, namespace string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}
	_, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	_, err = k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"app":       "image-factory",
				"managedBy": "image-factory",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func ensureNamespaceWithLabels(ctx context.Context, k8sClient kubernetes.Interface, namespace string, labels map[string]string) error {
	if err := ensureNamespace(ctx, k8sClient, namespace); err != nil {
		return err
	}
	if len(labels) == 0 {
		return nil
	}
	ns, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	changed := false
	for k, v := range labels {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		if ns.Labels[k] != v {
			ns.Labels[k] = v
			changed = true
		}
	}
	if !changed {
		return nil
	}
	_, err = k8sClient.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	return err
}

func waitForNamespaceDeletion(ctx context.Context, k8sClient kubernetes.Interface, namespace string, timeout time.Duration) error {
	if k8sClient == nil {
		return errors.New("kubernetes client is required")
	}
	if strings.TrimSpace(namespace) == "" {
		return errors.New("namespace is required")
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	deadline := time.Now().Add(timeout)
	for {
		_, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *Service) waitForTektonWebhookReady(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	provider *Provider,
	targetNamespace string,
	timeout time.Duration,
) error {
	if k8sClient == nil {
		return errors.New("kubernetes client is required")
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}

	const webhookServiceName = "tekton-pipelines-webhook"
	candidateNamespaces := s.resolveTektonWebhookCandidateNamespaces(ctx, provider, targetNamespace)
	if len(candidateNamespaces) == 0 {
		candidateNamespaces = []string{"tekton-pipelines"}
	}

	deadline := time.Now().Add(timeout)
	lastObserved := "tekton webhook service has no ready endpoints yet"
	for {
		foundService := false
		for _, namespace := range candidateNamespaces {
			if strings.TrimSpace(namespace) == "" {
				continue
			}
			if _, err := k8sClient.CoreV1().Services(namespace).Get(ctx, webhookServiceName, metav1.GetOptions{}); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("failed to query service %s/%s: %w", namespace, webhookServiceName, err)
			}
			foundService = true

			endpoints, err := k8sClient.CoreV1().Endpoints(namespace).Get(ctx, webhookServiceName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					lastObserved = fmt.Sprintf("service %s/%s exists but endpoints are not created yet", namespace, webhookServiceName)
					continue
				}
				return fmt.Errorf("failed to query endpoints %s/%s: %w", namespace, webhookServiceName, err)
			}
			if tektonWebhookEndpointsReady(endpoints) {
				return nil
			}
			lastObserved = fmt.Sprintf("service %s/%s has no ready endpoints yet", namespace, webhookServiceName)
		}
		if !foundService {
			lastObserved = fmt.Sprintf("service %s not found in candidate namespaces: %s", webhookServiceName, strings.Join(candidateNamespaces, ", "))
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s (waited %s)", lastObserved, timeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func tektonWebhookEndpointsReady(endpoints *corev1.Endpoints) bool {
	if endpoints == nil {
		return false
	}
	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 {
			return true
		}
	}
	return false
}

func (s *Service) resolveTektonWebhookCandidateNamespaces(ctx context.Context, provider *Provider, targetNamespace string) []string {
	candidates := make([]string, 0, 5)
	seen := map[string]struct{}{}
	add := func(namespace string) {
		ns := strings.TrimSpace(namespace)
		if ns == "" {
			return
		}
		if _, exists := seen[ns]; exists {
			return
		}
		seen[ns] = struct{}{}
		candidates = append(candidates, ns)
	}

	if provider != nil {
		add(configString(provider.Config, "tekton_webhook_namespace"))
		add(configString(provider.Config, "tekton_helm_namespace"))
	}
	if sys := s.getTektonCoreConfig(ctx); sys != nil {
		add(sys.HelmNamespace)
	}
	add("tekton-pipelines")
	add(targetNamespace)
	return candidates
}

func (s *Service) resolveTektonAssetRootDir(ctx context.Context) (string, error) {
	if sys := s.getTektonCoreConfig(ctx); sys != nil {
		if override := strings.TrimSpace(sys.AssetsDir); override != "" {
			if fileExists(filepath.Join(override, "kustomization.yaml")) {
				return override, nil
			}
			return "", fmt.Errorf("tekton core system config assets_dir is set but kustomization.yaml not found in %s", override)
		}
	}
	if override := strings.TrimSpace(os.Getenv("IF_TEKTON_ASSETS_DIR")); override != "" {
		if fileExists(filepath.Join(override, "kustomization.yaml")) {
			return override, nil
		}
		return "", fmt.Errorf("IF_TEKTON_ASSETS_DIR is set but kustomization.yaml not found in %s", override)
	}

	candidates := make([]string, 0, 8)
	seen := map[string]struct{}{}
	addCandidate := func(path string) {
		candidate := strings.TrimSpace(path)
		if candidate == "" {
			return
		}
		candidate = filepath.Clean(candidate)
		if _, exists := seen[candidate]; exists {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	wd, wdErr := os.Getwd()
	if wdErr == nil {
		addCandidate(filepath.Join(wd, "backend", "tekton"))
		addCandidate(filepath.Join(wd, "tekton"))
	}
	addCandidate(filepath.Join("backend", "tekton"))
	addCandidate("tekton")
	addCandidate(filepath.Join("/app", "tekton"))

	if exePath, exeErr := os.Executable(); exeErr == nil {
		exeDir := filepath.Dir(exePath)
		addCandidate(filepath.Join(exeDir, "tekton"))
		addCandidate(filepath.Join(exeDir, "..", "tekton"))
		addCandidate(filepath.Join(exeDir, "..", "backend", "tekton"))
	}

	for _, candidate := range candidates {
		if fileExists(filepath.Join(candidate, "kustomization.yaml")) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to resolve tekton asset root directory (checked %s)", strings.Join(candidates, ", "))
}

type tektonKustomization struct {
	Resources []string `yaml:"resources"`
}

func loadKustomizationResourceFiles(assetRoot string) ([]string, error) {
	kustomizationPath := filepath.Join(assetRoot, "kustomization.yaml")
	content, err := os.ReadFile(kustomizationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kustomization file: %w", err)
	}

	var k tektonKustomization
	if err := yaml.Unmarshal(content, &k); err != nil {
		return nil, fmt.Errorf("failed to parse kustomization file: %w", err)
	}

	files := make([]string, 0, len(k.Resources))
	for _, resource := range k.Resources {
		resource = strings.TrimSpace(resource)
		if resource == "" {
			continue
		}
		path := filepath.Join(assetRoot, resource)
		if !fileExists(path) {
			return nil, fmt.Errorf("kustomization resource does not exist: %s", path)
		}
		files = append(files, path)
	}
	return files, nil
}

func loadKustomizationResourceFilesForProfile(assetRoot, profileVersion string) ([]string, error) {
	resourceFiles, err := loadKustomizationResourceFiles(assetRoot)
	if err != nil {
		return nil, err
	}
	version := strings.ToLower(strings.TrimSpace(profileVersion))
	if version == "" || version == "v1" {
		return resourceFiles, nil
	}

	adjusted := make([]string, 0, len(resourceFiles))
	for _, resourceFile := range resourceFiles {
		rel, relErr := filepath.Rel(assetRoot, resourceFile)
		if relErr != nil {
			return nil, fmt.Errorf("failed to resolve tekton asset relative path for %s: %w", resourceFile, relErr)
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "tasks/v1/") || strings.HasPrefix(rel, "pipelines/v1/") {
			targetRel := strings.Replace(rel, "/v1/", "/"+version+"/", 1)
			targetPath := filepath.Join(assetRoot, filepath.FromSlash(targetRel))
			if !fileExists(targetPath) {
				return nil, fmt.Errorf("tekton profile version %s requested but resource does not exist: %s", version, targetPath)
			}
			adjusted = append(adjusted, targetPath)
			continue
		}
		adjusted = append(adjusted, resourceFile)
	}
	return adjusted, nil
}

func parseUnstructuredYAMLFile(path string) ([]*unstructured.Unstructured, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(string(content)), 4096)
	objects := make([]*unstructured.Unstructured, 0)

	for {
		raw := map[string]interface{}{}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(raw) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{Object: raw}
		if obj.GetAPIVersion() == "" || obj.GetKind() == "" || obj.GetName() == "" {
			return nil, fmt.Errorf("manifest object missing apiVersion/kind/name in %s", path)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (s *Service) refreshTenantNamespacePrepareAssetDrift(ctx context.Context, prepare *ProviderTenantNamespacePrepare) error {
	if s == nil || prepare == nil || s.tenantNamespacePrepareRepo == nil {
		return nil
	}

	originalDesired := strings.TrimSpace(stringValueOrEmpty(prepare.DesiredAssetVersion))
	originalStatus := prepare.AssetDriftStatus
	if originalStatus == "" {
		originalStatus = TenantAssetDriftStatusUnknown
	}

	assetRoot, err := s.resolveTektonAssetRootDir(ctx)
	if err != nil {
		nextStatus := computeTenantAssetDriftStatus(prepare.DesiredAssetVersion, prepare.InstalledAssetVersion)
		if originalDesired == "" && originalStatus == nextStatus {
			return nil
		}
		prepare.AssetDriftStatus = nextStatus
		prepare.UpdatedAt = time.Now().UTC()
		return s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
	}
	profileVersion := "v1"
	if s.repository != nil {
		provider, providerErr := s.repository.FindProviderByID(ctx, prepare.ProviderID)
		if providerErr == nil && provider != nil {
			profileVersion = resolveTektonAssetVersion(provider, "")
		}
	}
	resourceFiles, err := loadKustomizationResourceFilesForProfile(assetRoot, profileVersion)
	if err != nil || len(resourceFiles) == 0 {
		nextStatus := computeTenantAssetDriftStatus(prepare.DesiredAssetVersion, prepare.InstalledAssetVersion)
		if originalDesired == "" && originalStatus == nextStatus {
			return nil
		}
		prepare.AssetDriftStatus = nextStatus
		prepare.UpdatedAt = time.Now().UTC()
		return s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
	}

	desiredVersion, err := calculateTektonAssetsVersionWithOverrides(resourceFiles, s.getTektonTaskImagesConfig(ctx))
	if err != nil || desiredVersion == "" {
		nextStatus := computeTenantAssetDriftStatus(prepare.DesiredAssetVersion, prepare.InstalledAssetVersion)
		if originalDesired == "" && originalStatus == nextStatus {
			return nil
		}
		prepare.AssetDriftStatus = nextStatus
		prepare.UpdatedAt = time.Now().UTC()
		return s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
	}

	nextStatus := computeTenantAssetDriftStatus(&desiredVersion, prepare.InstalledAssetVersion)
	if originalDesired == strings.TrimSpace(desiredVersion) && originalStatus == nextStatus {
		return nil
	}
	prepare.DesiredAssetVersion = &desiredVersion
	prepare.AssetDriftStatus = nextStatus
	prepare.UpdatedAt = time.Now().UTC()
	return s.tenantNamespacePrepareRepo.UpsertTenantNamespacePrepare(ctx, prepare)
}

func computeTenantAssetDriftStatus(desiredVersion, installedVersion *string) TenantAssetDriftStatus {
	desired := ""
	if desiredVersion != nil {
		desired = strings.TrimSpace(*desiredVersion)
	}
	installed := ""
	if installedVersion != nil {
		installed = strings.TrimSpace(*installedVersion)
	}
	if desired == "" || installed == "" {
		return TenantAssetDriftStatusUnknown
	}
	if desired == installed {
		return TenantAssetDriftStatusCurrent
	}
	return TenantAssetDriftStatusStale
}

func calculateTektonAssetsVersion(resourceFiles []string) (string, error) {
	return calculateTektonAssetsVersionWithOverrides(resourceFiles, nil)
}

func calculateTektonAssetsVersionWithOverrides(resourceFiles []string, taskImageCfg *systemconfig.TektonTaskImagesConfig) (string, error) {
	if len(resourceFiles) == 0 {
		return "", nil
	}

	sortedFiles := append([]string{}, resourceFiles...)
	sort.Strings(sortedFiles)
	hasher := sha256.New()

	for _, resourceFile := range sortedFiles {
		normalizedPath := filepath.ToSlash(strings.TrimSpace(resourceFile))
		if normalizedPath == "" {
			continue
		}
		if _, err := hasher.Write([]byte(normalizedPath)); err != nil {
			return "", fmt.Errorf("failed to hash resource path: %w", err)
		}
		objects, err := parseUnstructuredYAMLFile(resourceFile)
		if err != nil {
			return "", fmt.Errorf("failed to parse resource file %s: %w", resourceFile, err)
		}
		applyTektonTaskImageOverrides(objects, taskImageCfg)
		for _, obj := range objects {
			if obj == nil {
				continue
			}
			content, marshalErr := json.Marshal(obj.Object)
			if marshalErr != nil {
				return "", fmt.Errorf("failed to marshal resource object %s/%s from %s: %w", obj.GetKind(), obj.GetName(), resourceFile, marshalErr)
			}
			if _, err := hasher.Write(content); err != nil {
				return "", fmt.Errorf("failed to hash resource content %s: %w", resourceFile, err)
			}
		}
	}

	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func applyTektonTaskImageOverrides(objects []*unstructured.Unstructured, config *systemconfig.TektonTaskImagesConfig) {
	if len(objects) == 0 {
		return
	}
	replacements := tektonTaskImageReplacementsBySource(config)
	if len(replacements) == 0 {
		return
	}
	for _, obj := range objects {
		if obj == nil || !isTektonTaskImageOverrideTarget(obj) {
			continue
		}
		rewriteImageFields(obj.Object, replacements)
	}
}

func isTektonTaskImageOverrideTarget(obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	if isTektonHistoryCleanupCronJob(obj) {
		return true
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(obj.GetAPIVersion())), "tekton.dev/") {
		return false
	}
	switch strings.TrimSpace(obj.GetKind()) {
	case "Task", "Pipeline":
		return true
	default:
		return false
	}
}

func isTektonHistoryCleanupCronJob(obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	if strings.TrimSpace(obj.GetAPIVersion()) != "batch/v1" {
		return false
	}
	if strings.TrimSpace(obj.GetKind()) != "CronJob" {
		return false
	}
	return strings.TrimSpace(obj.GetName()) == tektonHistoryCleanupCronJobName
}

func rewriteImageFields(node interface{}, replacements map[string]string) {
	switch typed := node.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			if key == "image" {
				if src, ok := value.(string); ok {
					if dst, found := replacements[strings.TrimSpace(src)]; found && strings.TrimSpace(dst) != "" {
						typed[key] = dst
						continue
					}
				}
			}
			rewriteImageFields(value, replacements)
		}
	case []interface{}:
		for _, item := range typed {
			rewriteImageFields(item, replacements)
		}
	}
}

func tektonTaskImageReplacementsBySource(config *systemconfig.TektonTaskImagesConfig) map[string]string {
	effective := defaultTektonTaskImageByLogicalKey()
	if config != nil {
		if v := strings.TrimSpace(config.GitClone); v != "" {
			effective["git_clone"] = v
		}
		if v := strings.TrimSpace(config.KanikoExecutor); v != "" {
			effective["kaniko_executor"] = v
		}
		if v := strings.TrimSpace(config.Buildkit); v != "" {
			effective["buildkit"] = v
		}
		if v := strings.TrimSpace(config.Skopeo); v != "" {
			effective["skopeo"] = v
		}
		if v := strings.TrimSpace(config.Trivy); v != "" {
			effective["trivy"] = v
		}
		if v := strings.TrimSpace(config.Syft); v != "" {
			effective["syft"] = v
		}
		if v := strings.TrimSpace(config.Cosign); v != "" {
			effective["cosign"] = v
		}
		if v := strings.TrimSpace(config.Packer); v != "" {
			effective["packer"] = v
		}
		if v := strings.TrimSpace(config.PythonAlpine); v != "" {
			effective["python_alpine"] = v
		}
		if v := strings.TrimSpace(config.Alpine); v != "" {
			effective["alpine"] = v
		}
		if v := strings.TrimSpace(config.CleanupKubectl); v != "" {
			effective["cleanup_kubectl"] = v
		}
	}

	replacements := map[string]string{}
	for logicalKey, src := range defaultTektonTaskImageByLogicalKey() {
		dst := strings.TrimSpace(effective[logicalKey])
		if dst == "" {
			continue
		}
		replacements[src] = dst
	}
	return replacements
}

func defaultTektonTaskImageByLogicalKey() map[string]string {
	return map[string]string{
		"git_clone":       "docker.io/alpine/git:2.45.2",
		"kaniko_executor": "gcr.io/kaniko-project/executor:v1.23.2",
		"buildkit":        "docker.io/moby/buildkit:v0.13.2",
		"skopeo":          "quay.io/skopeo/stable:v1.15.0",
		"trivy":           "docker.io/aquasec/trivy:0.57.1",
		"syft":            "docker.io/anchore/syft:v1.18.1",
		"cosign":          "gcr.io/projectsigstore/cosign:v2.4.1",
		"packer":          "docker.io/hashicorp/packer:1.10.2",
		"python_alpine":   "docker.io/library/python:3.12-alpine",
		"alpine":          "docker.io/library/alpine:3.20",
		"cleanup_kubectl": "docker.io/bitnami/kubectl:latest",
	}
}

func stringValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseUnstructuredYAMLBytes(content []byte, source string) ([]*unstructured.Unstructured, error) {
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
	objects := make([]*unstructured.Unstructured, 0)

	for {
		raw := map[string]interface{}{}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(raw) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{Object: raw}
		if obj.GetAPIVersion() == "" || obj.GetKind() == "" || obj.GetName() == "" {
			return nil, fmt.Errorf("manifest object missing apiVersion/kind/name in %s", source)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func fetchAndParseUnstructuredYAML(ctx context.Context, manifestURL string) ([]*unstructured.Unstructured, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseUnstructuredYAMLBytes(content, manifestURL)
}

func applyUnstructuredObjects(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	objects []*unstructured.Unstructured,
	defaultNamespace string,
	fieldManager string,
) ([]string, error) {
	if dynamicClient == nil {
		return nil, errors.New("dynamic client is required")
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	applied := make([]string, 0, len(objects))
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		mapping, mapErr := mapper.RESTMapping(schema.GroupKind{
			Group: obj.GroupVersionKind().Group,
			Kind:  obj.GetKind(),
		}, obj.GroupVersionKind().Version)
		if mapErr != nil {
			return applied, fmt.Errorf("failed to map resource %s/%s: %w", obj.GetKind(), obj.GetName(), mapErr)
		}

		resourceClient := dynamicClient.Resource(mapping.Resource)
		var resourceInterface dynamic.ResourceInterface
		namespaced := mapping.Scope.Name() == metaRESTScopeNameNamespace
		if namespaced {
			ns := strings.TrimSpace(obj.GetNamespace())
			if ns == "" {
				ns = defaultNamespace
			}
			if ns == "" {
				return applied, fmt.Errorf("resource %s/%s is namespaced but namespace is empty", obj.GetKind(), obj.GetName())
			}
			obj.SetNamespace(ns)
			resourceInterface = resourceClient.Namespace(ns)
		} else {
			resourceInterface = resourceClient
		}

		objJSON, marshalErr := json.Marshal(obj.Object)
		if marshalErr != nil {
			return applied, fmt.Errorf("failed to marshal %s/%s: %w", obj.GetKind(), obj.GetName(), marshalErr)
		}
		force := true
		if _, patchErr := resourceInterface.Patch(
			ctx,
			obj.GetName(),
			types.ApplyPatchType,
			objJSON,
			metav1.PatchOptions{
				FieldManager: fieldManager,
				Force:        &force,
			},
		); patchErr != nil {
			return applied, fmt.Errorf("failed to apply %s/%s: %w", obj.GetKind(), obj.GetName(), patchErr)
		}
		applied = append(applied, fmt.Sprintf("%s/%s", obj.GetKind(), obj.GetName()))
	}
	return applied, nil
}

func (s *Service) resolveTektonCoreInstallSource(ctx context.Context, provider *Provider) string {
	if provider != nil && provider.Config != nil {
		source := strings.ToLower(strings.TrimSpace(configString(provider.Config, "tekton_core_install_source")))
		switch source {
		case tektonCoreInstallSourceManifest, tektonCoreInstallSourceHelm, tektonCoreInstallSourcePreinstalled:
			return source
		}
	}

	if sys := s.getTektonCoreConfig(ctx); sys != nil {
		source := strings.ToLower(strings.TrimSpace(sys.InstallSource))
		switch source {
		case tektonCoreInstallSourceManifest, tektonCoreInstallSourceHelm, tektonCoreInstallSourcePreinstalled:
			return source
		}
	}

	return tektonCoreInstallSourceManifest
}

func sysString(cfg *systemconfig.TektonCoreConfig, get func(*systemconfig.TektonCoreConfig) string) string {
	if cfg == nil || get == nil {
		return ""
	}
	return strings.TrimSpace(get(cfg))
}

func (s *Service) getTektonCoreConfig(ctx context.Context) *systemconfig.TektonCoreConfig {
	if s == nil || s.tektonCoreConfigLookup == nil {
		return nil
	}
	cfg, err := s.tektonCoreConfigLookup(ctx)
	if err != nil {
		s.logger.Warn("Failed to load Tekton core system config; falling back to provider/env defaults", zap.Error(err))
		return nil
	}
	return cfg
}

func (s *Service) getTektonTaskImagesConfig(ctx context.Context) *systemconfig.TektonTaskImagesConfig {
	if s == nil || s.tektonTaskImagesConfigLookup == nil {
		return nil
	}
	cfg, err := s.tektonTaskImagesConfigLookup(ctx)
	if err != nil {
		s.logger.Warn("Failed to load Tekton task images system config; falling back to task defaults", zap.Error(err))
		return nil
	}
	return cfg
}

func (s *Service) resolveTektonCoreManifestURLs(ctx context.Context, provider *Provider) []string {
	if provider != nil && provider.Config != nil {
		if raw, ok := provider.Config["tekton_core_manifest_urls"]; ok {
			values := parseStringList(raw)
			if len(values) > 0 {
				return values
			}
		}
		if raw, ok := provider.Config["tekton_core_manifest_url"]; ok {
			values := parseStringList(raw)
			if len(values) > 0 {
				return values
			}
		}
	}

	if sys := s.getTektonCoreConfig(ctx); sys != nil {
		if len(sys.ManifestURLs) > 0 {
			out := make([]string, 0, len(sys.ManifestURLs))
			for _, u := range sys.ManifestURLs {
				u = strings.TrimSpace(u)
				if u != "" {
					out = append(out, u)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	return append([]string{}, defaultTektonCoreManifestURLs...)
}

func parseStringList(value interface{}) []string {
	out := make([]string, 0)
	switch v := value.(type) {
	case string:
		for _, part := range strings.FieldsFunc(v, func(r rune) bool {
			return r == '\n' || r == ',' || r == ';'
		}) {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				trimmed := strings.TrimSpace(str)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
	case []string:
		for _, item := range v {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func configString(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	raw, ok := config[key]
	if !ok || raw == nil {
		return ""
	}
	str, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func resolveTektonInstallMode(provider *Provider, requested TektonInstallMode) TektonInstallMode {
	if requested == TektonInstallModeGitOps || requested == TektonInstallModeImageFactoryInstaller {
		return requested
	}
	if provider != nil && provider.Config != nil {
		if rawMode, ok := provider.Config["tekton_install_mode"].(string); ok {
			mode := TektonInstallMode(rawMode)
			if mode == TektonInstallModeGitOps || mode == TektonInstallModeImageFactoryInstaller {
				return mode
			}
		}
	}
	return TektonInstallModeImageFactoryInstaller
}

func resolveTektonAssetVersion(provider *Provider, requested string) string {
	if requested != "" {
		return strings.ToLower(strings.TrimSpace(requested))
	}
	if provider != nil && provider.Config != nil {
		if v, ok := provider.Config["tekton_profile_version"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.ToLower(strings.TrimSpace(v))
		}
	}
	return "v1"
}
