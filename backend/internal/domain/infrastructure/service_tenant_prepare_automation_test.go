package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

type automationRepoStub struct {
	providers        map[uuid.UUID]*Provider
	allProviders     []Provider
	tenantPrepares   map[string]*ProviderTenantNamespacePrepare
	savedPermissions []*ProviderPermission
}

func (r *automationRepoStub) SaveProvider(ctx context.Context, provider *Provider) error { return nil }
func (r *automationRepoStub) FindProviderByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	if p, ok := r.providers[id]; ok && p != nil {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}
func (r *automationRepoStub) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	return &ListProvidersResult{}, nil
}
func (r *automationRepoStub) FindProvidersAll(ctx context.Context, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	return &ListProvidersResult{Providers: append([]Provider{}, r.allProviders...)}, nil
}
func (r *automationRepoStub) UpdateProvider(ctx context.Context, provider *Provider) error {
	return nil
}
func (r *automationRepoStub) DeleteProvider(ctx context.Context, id uuid.UUID) error { return nil }
func (r *automationRepoStub) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	return false, nil
}
func (r *automationRepoStub) SavePermission(ctx context.Context, permission *ProviderPermission) error {
	r.savedPermissions = append(r.savedPermissions, permission)
	return nil
}
func (r *automationRepoStub) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}
func (r *automationRepoStub) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}
func (r *automationRepoStub) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	return nil
}
func (r *automationRepoStub) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	return false, nil
}
func (r *automationRepoStub) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *ProviderHealth) error {
	return nil
}
func (r *automationRepoStub) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*ProviderHealth, error) {
	return nil, nil
}
func (r *automationRepoStub) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	return nil
}

func (r *automationRepoStub) UpsertTenantNamespacePrepare(ctx context.Context, prepare *ProviderTenantNamespacePrepare) error {
	if r.tenantPrepares == nil {
		r.tenantPrepares = make(map[string]*ProviderTenantNamespacePrepare)
	}
	key := prepare.ProviderID.String() + ":" + prepare.TenantID.String()
	cp := *prepare
	r.tenantPrepares[key] = &cp
	return nil
}

func (r *automationRepoStub) GetTenantNamespacePrepare(ctx context.Context, providerID, tenantID uuid.UUID) (*ProviderTenantNamespacePrepare, error) {
	if r.tenantPrepares == nil {
		return nil, nil
	}
	key := providerID.String() + ":" + tenantID.String()
	if existing, ok := r.tenantPrepares[key]; ok && existing != nil {
		cp := *existing
		return &cp, nil
	}
	return nil, nil
}

func (r *automationRepoStub) ListTenantNamespacePreparesByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderTenantNamespacePrepare, error) {
	if r.tenantPrepares == nil {
		return nil, nil
	}
	out := make([]*ProviderTenantNamespacePrepare, 0, len(r.tenantPrepares))
	for _, prep := range r.tenantPrepares {
		if prep == nil || prep.ProviderID != providerID {
			continue
		}
		cp := *prep
		out = append(out, &cp)
	}
	return out, nil
}

func TestTriggerTenantNamespacePrepareAsync_SkipsNonManagedOrNonKubernetesAndTracksMetrics(t *testing.T) {
	tenantID := uuid.New()
	nonK8sID := uuid.New()
	selfManagedID := uuid.New()
	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			nonK8sID: {
				ID:            nonK8sID,
				ProviderType:  ProviderTypeBuildNodes,
				BootstrapMode: "image_factory_managed",
			},
			selfManagedID: {
				ID:            selfManagedID,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "self_managed",
			},
		},
	}

	service := NewService(repo, nil, zap.NewNop())

	if err := service.TriggerTenantNamespacePrepareAsync(context.Background(), nonK8sID, tenantID, nil); err != nil {
		t.Fatalf("unexpected error for non-k8s provider: %v", err)
	}
	if err := service.TriggerTenantNamespacePrepareAsync(context.Background(), selfManagedID, tenantID, nil); err != nil {
		t.Fatalf("unexpected error for self-managed provider: %v", err)
	}

	metrics := service.GetTenantPrepareAutomationMetrics()
	if metrics.AsyncTriggered != 0 {
		t.Fatalf("expected 0 triggers, got %d", metrics.AsyncTriggered)
	}
	if metrics.AsyncSkipped < 2 {
		t.Fatalf("expected at least 2 skipped triggers, got %d", metrics.AsyncSkipped)
	}
}

func TestTriggerTenantNamespacePrepareAsync_ManagedProviderTracksTriggerAndFailure(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:            providerID,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
				// Missing bootstrap_auth intentionally to fail quickly.
				Config: map[string]interface{}{
					"tekton_enabled": true,
				},
			},
		},
		tenantPrepares: make(map[string]*ProviderTenantNamespacePrepare),
	}

	service := NewService(repo, nil, zap.NewNop())
	if err := service.TriggerTenantNamespacePrepareAsync(context.Background(), providerID, tenantID, nil); err != nil {
		t.Fatalf("unexpected error when triggering managed provider: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		metrics := service.GetTenantPrepareAutomationMetrics()
		if metrics.AsyncFailures >= 1 {
			if metrics.AsyncTriggered < 1 {
				t.Fatalf("expected at least one trigger, got %d", metrics.AsyncTriggered)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected async failure metric to increment, got %+v", metrics)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestTriggerTenantNamespacePrepareForNewTenantAsync_TriggersOnlyGlobalManagedKubernetesProviders(t *testing.T) {
	tenantID := uuid.New()
	globalManagedID := uuid.New()
	globalSelfManagedID := uuid.New()
	globalNonK8sID := uuid.New()
	tenantLocalManagedID := uuid.New()

	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			globalManagedID: {
				ID:            globalManagedID,
				IsGlobal:      true,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
				Config: map[string]interface{}{
					"tekton_enabled": true,
				},
			},
			globalSelfManagedID: {
				ID:            globalSelfManagedID,
				IsGlobal:      true,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "self_managed",
			},
			globalNonK8sID: {
				ID:            globalNonK8sID,
				IsGlobal:      true,
				ProviderType:  ProviderTypeBuildNodes,
				BootstrapMode: "image_factory_managed",
			},
			tenantLocalManagedID: {
				ID:            tenantLocalManagedID,
				IsGlobal:      false,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
			},
		},
		allProviders: []Provider{
			{ID: globalManagedID, IsGlobal: true, ProviderType: ProviderTypeKubernetes, BootstrapMode: "image_factory_managed", Config: map[string]interface{}{"tekton_enabled": true}},
			{ID: globalSelfManagedID, IsGlobal: true, ProviderType: ProviderTypeKubernetes, BootstrapMode: "self_managed"},
			{ID: globalNonK8sID, IsGlobal: true, ProviderType: ProviderTypeBuildNodes, BootstrapMode: "image_factory_managed"},
			{ID: tenantLocalManagedID, IsGlobal: false, ProviderType: ProviderTypeKubernetes, BootstrapMode: "image_factory_managed"},
		},
		tenantPrepares: make(map[string]*ProviderTenantNamespacePrepare),
	}

	service := NewService(repo, nil, zap.NewNop())
	triggered, err := service.TriggerTenantNamespacePrepareForNewTenantAsync(context.Background(), tenantID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if triggered != 1 {
		t.Fatalf("expected exactly one provider trigger, got %d", triggered)
	}

	metrics := service.GetTenantPrepareAutomationMetrics()
	if metrics.AsyncTriggered < 1 {
		t.Fatalf("expected async trigger metric >= 1, got %d", metrics.AsyncTriggered)
	}
	if metrics.AsyncSkipped < 2 {
		t.Fatalf("expected async skip metric >= 2 for filtered global providers, got %d", metrics.AsyncSkipped)
	}
}

func TestTriggerTenantNamespacePrepareForProviderTenants_IncludesOwnerTenantForTenantScopedProvider(t *testing.T) {
	ownerTenantID := uuid.New()
	providerID := uuid.New()
	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:            providerID,
				TenantID:      ownerTenantID,
				IsGlobal:      false,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
			},
		},
	}

	service := NewService(repo, nil, zap.NewNop())
	service.SetRuntimeServicesConfigLookup(func(ctx context.Context) (*systemconfig.RuntimeServicesConfig, error) {
		return &systemconfig.RuntimeServicesConfig{TenantAssetReconcilePolicy: tenantAssetReconcilePolicyManualOnly}, nil
	})

	summary, err := service.triggerTenantNamespacePrepareForProviderTenants(context.Background(), providerID, uuid.Nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary["policy"] != tenantAssetReconcilePolicyManualOnly {
		t.Fatalf("expected policy %q, got %v", tenantAssetReconcilePolicyManualOnly, summary["policy"])
	}
	if summary["tenants_targeted"] != 1 {
		t.Fatalf("expected tenants_targeted=1, got %v", summary["tenants_targeted"])
	}
	tenantIDs, ok := summary["tenant_ids"].([]uuid.UUID)
	if !ok || len(tenantIDs) != 1 || tenantIDs[0] != ownerTenantID {
		t.Fatalf("expected owner tenant in tenant_ids, got %#v", summary["tenant_ids"])
	}
}

func TestRunTenantAssetDriftWatchTick_UpdatesAndCountsStatuses(t *testing.T) {
	providerID := uuid.New()
	tenantCurrent := uuid.New()
	tenantStale := uuid.New()

	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "kustomization.yaml"), []byte("resources:\n  - task.yaml\n"), 0o644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "task.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	t.Setenv("IF_TEKTON_ASSETS_DIR", assetsDir)

	desired, err := calculateTektonAssetsVersion([]string{filepath.Join(assetsDir, "task.yaml")})
	if err != nil {
		t.Fatalf("failed to compute desired version: %v", err)
	}

	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:            providerID,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
			},
		},
		allProviders: []Provider{
			{ID: providerID, ProviderType: ProviderTypeKubernetes, BootstrapMode: "image_factory_managed"},
		},
		tenantPrepares: map[string]*ProviderTenantNamespacePrepare{
			providerID.String() + ":" + tenantCurrent.String(): {
				ID:                    uuid.New(),
				ProviderID:            providerID,
				TenantID:              tenantCurrent,
				Namespace:             "image-factory-" + tenantCurrent.String()[:8],
				Status:                ProviderTenantNamespacePrepareSucceeded,
				InstalledAssetVersion: &desired,
				AssetDriftStatus:      TenantAssetDriftStatusUnknown,
				CreatedAt:             time.Now().UTC(),
				UpdatedAt:             time.Now().UTC(),
			},
			providerID.String() + ":" + tenantStale.String(): {
				ID:                    uuid.New(),
				ProviderID:            providerID,
				TenantID:              tenantStale,
				Namespace:             "image-factory-" + tenantStale.String()[:8],
				Status:                ProviderTenantNamespacePrepareSucceeded,
				InstalledAssetVersion: strPtr("sha256:old"),
				AssetDriftStatus:      TenantAssetDriftStatusUnknown,
				CreatedAt:             time.Now().UTC(),
				UpdatedAt:             time.Now().UTC(),
			},
		},
	}

	service := NewService(repo, nil, zap.NewNop())
	result, err := service.RunTenantAssetDriftWatchTick(context.Background(), 100)
	if err != nil {
		t.Fatalf("RunTenantAssetDriftWatchTick failed: %v", err)
	}
	if result.TotalNamespaces != 2 {
		t.Fatalf("expected total_namespaces=2 got %d", result.TotalNamespaces)
	}
	if result.Current != 1 {
		t.Fatalf("expected current=1 got %d", result.Current)
	}
	if result.Stale != 1 {
		t.Fatalf("expected stale=1 got %d", result.Stale)
	}
}

func TestReconcileStaleTenantNamespaces_AppliesRunningTenantRows(t *testing.T) {
	providerID := uuid.New()
	tenantID := uuid.New()

	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "kustomization.yaml"), []byte("resources:\n  - task.yaml\n"), 0o644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "task.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	t.Setenv("IF_TEKTON_ASSETS_DIR", assetsDir)

	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:            providerID,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
				Config:        map[string]interface{}{},
			},
		},
		tenantPrepares: map[string]*ProviderTenantNamespacePrepare{
			providerID.String() + ":" + tenantID.String(): {
				ID:                    uuid.New(),
				ProviderID:            providerID,
				TenantID:              tenantID,
				Namespace:             "image-factory-" + tenantID.String()[:8],
				Status:                ProviderTenantNamespacePrepareRunning,
				InstalledAssetVersion: strPtr("sha256:old"),
				AssetDriftStatus:      TenantAssetDriftStatusStale,
				CreatedAt:             time.Now().UTC(),
				UpdatedAt:             time.Now().UTC(),
			},
		},
	}

	service := NewService(repo, nil, zap.NewNop())
	summary, err := service.ReconcileStaleTenantNamespaces(context.Background(), providerID, nil)
	if err != nil {
		t.Fatalf("ReconcileStaleTenantNamespaces failed: %v", err)
	}
	if summary.Targeted != 1 {
		t.Fatalf("expected targeted=1 got %d", summary.Targeted)
	}
	if summary.Applied != 1 {
		t.Fatalf("expected applied=1 got %d", summary.Applied)
	}
	if summary.Failed != 0 {
		t.Fatalf("expected failed=0 got %d", summary.Failed)
	}
}

func TestRunTenantAssetDriftWatchTick_LongRunningStaleSet(t *testing.T) {
	providerID := uuid.New()
	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "kustomization.yaml"), []byte("resources:\n  - task.yaml\n"), 0o644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "task.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	t.Setenv("IF_TEKTON_ASSETS_DIR", assetsDir)

	desired, err := calculateTektonAssetsVersion([]string{filepath.Join(assetsDir, "task.yaml")})
	if err != nil {
		t.Fatalf("failed to compute desired version: %v", err)
	}

	tenantPrepares := make(map[string]*ProviderTenantNamespacePrepare)
	for i := 0; i < 75; i++ {
		tenantID := uuid.New()
		key := providerID.String() + ":" + tenantID.String()
		tenantPrepares[key] = &ProviderTenantNamespacePrepare{
			ID:                    uuid.New(),
			ProviderID:            providerID,
			TenantID:              tenantID,
			Namespace:             "image-factory-" + tenantID.String()[:8],
			Status:                ProviderTenantNamespacePrepareSucceeded,
			InstalledAssetVersion: strPtr("sha256:old"),
			AssetDriftStatus:      TenantAssetDriftStatusUnknown,
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}
	}
	for i := 0; i < 10; i++ {
		tenantID := uuid.New()
		key := providerID.String() + ":" + tenantID.String()
		tenantPrepares[key] = &ProviderTenantNamespacePrepare{
			ID:                    uuid.New(),
			ProviderID:            providerID,
			TenantID:              tenantID,
			Namespace:             "image-factory-" + tenantID.String()[:8],
			Status:                ProviderTenantNamespacePrepareSucceeded,
			InstalledAssetVersion: &desired,
			AssetDriftStatus:      TenantAssetDriftStatusUnknown,
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}
	}
	for i := 0; i < 5; i++ {
		tenantID := uuid.New()
		key := providerID.String() + ":" + tenantID.String()
		tenantPrepares[key] = &ProviderTenantNamespacePrepare{
			ID:               uuid.New(),
			ProviderID:       providerID,
			TenantID:         tenantID,
			Namespace:        "image-factory-" + tenantID.String()[:8],
			Status:           ProviderTenantNamespacePrepareSucceeded,
			AssetDriftStatus: TenantAssetDriftStatusUnknown,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
	}

	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:            providerID,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
			},
		},
		allProviders: []Provider{
			{ID: providerID, ProviderType: ProviderTypeKubernetes, BootstrapMode: "image_factory_managed"},
		},
		tenantPrepares: tenantPrepares,
	}

	service := NewService(repo, nil, zap.NewNop())
	result, err := service.RunTenantAssetDriftWatchTick(context.Background(), 200)
	if err != nil {
		t.Fatalf("RunTenantAssetDriftWatchTick failed: %v", err)
	}
	if result.TotalNamespaces != 90 {
		t.Fatalf("expected total_namespaces=90 got %d", result.TotalNamespaces)
	}
	if result.Stale != 75 {
		t.Fatalf("expected stale=75 got %d", result.Stale)
	}
	if result.Current != 10 {
		t.Fatalf("expected current=10 got %d", result.Current)
	}
	if result.Unknown != 5 {
		t.Fatalf("expected unknown=5 got %d", result.Unknown)
	}

	metrics := service.GetTenantAssetDriftMetrics()
	if metrics.WatchTicksTotal != 1 {
		t.Fatalf("expected watch_ticks_total=1 got %d", metrics.WatchTicksTotal)
	}
	if metrics.WatchStaleNamespaces != 75 {
		t.Fatalf("expected watch_stale_namespaces=75 got %d", metrics.WatchStaleNamespaces)
	}
}

func TestReconcileSelectedTenantNamespaces_UnderLoad(t *testing.T) {
	providerID := uuid.New()
	assetsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsDir, "kustomization.yaml"), []byte("resources:\n  - task.yaml\n"), 0o644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "task.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	t.Setenv("IF_TEKTON_ASSETS_DIR", assetsDir)

	tenantPrepares := make(map[string]*ProviderTenantNamespacePrepare)
	targetTenantIDs := make([]uuid.UUID, 0, 12)
	for i := 0; i < 60; i++ {
		tenantID := uuid.New()
		if i < 12 {
			targetTenantIDs = append(targetTenantIDs, tenantID)
		}
		key := providerID.String() + ":" + tenantID.String()
		tenantPrepares[key] = &ProviderTenantNamespacePrepare{
			ID:                    uuid.New(),
			ProviderID:            providerID,
			TenantID:              tenantID,
			Namespace:             "image-factory-" + tenantID.String()[:8],
			Status:                ProviderTenantNamespacePrepareRunning,
			InstalledAssetVersion: strPtr("sha256:old"),
			AssetDriftStatus:      TenantAssetDriftStatusStale,
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}
	}

	repo := &automationRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:            providerID,
				ProviderType:  ProviderTypeKubernetes,
				BootstrapMode: "image_factory_managed",
				Config:        map[string]interface{}{},
			},
		},
		tenantPrepares: tenantPrepares,
	}

	service := NewService(repo, nil, zap.NewNop())
	summary, err := service.ReconcileSelectedTenantNamespaces(context.Background(), providerID, targetTenantIDs, nil)
	if err != nil {
		t.Fatalf("ReconcileSelectedTenantNamespaces failed: %v", err)
	}
	if summary.Targeted != 12 {
		t.Fatalf("expected targeted=12 got %d", summary.Targeted)
	}
	if summary.Applied != 12 {
		t.Fatalf("expected applied=12 got %d", summary.Applied)
	}
	if summary.Failed != 0 {
		t.Fatalf("expected failed=0 got %d", summary.Failed)
	}

	metrics := service.GetTenantAssetDriftMetrics()
	if metrics.ReconcileRequestsTotal != 1 {
		t.Fatalf("expected reconcile_requests_total=1 got %d", metrics.ReconcileRequestsTotal)
	}
	if metrics.ReconcileRequestsSuccess != 1 {
		t.Fatalf("expected reconcile_requests_success_total=1 got %d", metrics.ReconcileRequestsSuccess)
	}
}
