package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type testInfrastructureRepo struct {
	systemProviders []Provider
	tenantProviders map[uuid.UUID][]Provider
	permissions     map[uuid.UUID]map[uuid.UUID]bool
}

func (r *testInfrastructureRepo) SaveProvider(ctx context.Context, provider *Provider) error {
	return nil
}
func (r *testInfrastructureRepo) FindProviderByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	return nil, nil
}
func (r *testInfrastructureRepo) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	systemTenantID := uuid.Nil
	if tenantID == systemTenantID {
		return &ListProvidersResult{Providers: r.systemProviders}, nil
	}
	return &ListProvidersResult{Providers: r.tenantProviders[tenantID]}, nil
}
func (r *testInfrastructureRepo) FindProvidersAll(ctx context.Context, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	combined := make([]Provider, 0, len(r.systemProviders))
	combined = append(combined, r.systemProviders...)
	for _, providers := range r.tenantProviders {
		combined = append(combined, providers...)
	}
	return &ListProvidersResult{Providers: combined}, nil
}
func (r *testInfrastructureRepo) UpdateProvider(ctx context.Context, provider *Provider) error {
	return nil
}
func (r *testInfrastructureRepo) DeleteProvider(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testInfrastructureRepo) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	return false, nil
}
func (r *testInfrastructureRepo) SavePermission(ctx context.Context, permission *ProviderPermission) error {
	return nil
}
func (r *testInfrastructureRepo) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}
func (r *testInfrastructureRepo) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}
func (r *testInfrastructureRepo) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	return nil
}
func (r *testInfrastructureRepo) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	if providerPerms, ok := r.permissions[tenantID]; ok {
		return providerPerms[providerID], nil
	}
	return false, nil
}
func (r *testInfrastructureRepo) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *ProviderHealth) error {
	return nil
}
func (r *testInfrastructureRepo) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*ProviderHealth, error) {
	return nil, nil
}
func (r *testInfrastructureRepo) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	return nil
}

func TestGetAvailableProviders_FiltersUnschedulableK8sProviders(t *testing.T) {
	systemTenantID := uuid.Nil
	tenantID := uuid.New()

	ready := "ready"
	notReady := "not_ready"

	globalReadyK8s := Provider{ID: uuid.New(), TenantID: systemTenantID, IsGlobal: true, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, ReadinessStatus: &ready, IsSchedulable: true}
	globalNotReadyK8s := Provider{ID: uuid.New(), TenantID: systemTenantID, IsGlobal: true, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, ReadinessStatus: &notReady, IsSchedulable: false}
	permittedReadyK8s := Provider{ID: uuid.New(), TenantID: systemTenantID, IsGlobal: false, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, ReadinessStatus: &ready, IsSchedulable: true}
	permittedCapacityBlockedK8s := Provider{
		ID:                      uuid.New(),
		TenantID:                systemTenantID,
		IsGlobal:                false,
		Status:                  ProviderStatusOnline,
		ProviderType:            ProviderTypeKubernetes,
		ReadinessStatus:         &ready,
		ReadinessMissingPrereqs: []string{"cluster_capacity: no ready nodes"},
		IsSchedulable:           false,
	}
	globalBuildNodes := Provider{ID: uuid.New(), TenantID: systemTenantID, IsGlobal: true, Status: ProviderStatusOnline, ProviderType: ProviderTypeBuildNodes, IsSchedulable: true}

	tenantNotReadyK8s := Provider{ID: uuid.New(), TenantID: tenantID, IsGlobal: false, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, ReadinessStatus: &notReady, IsSchedulable: false}
	tenantUnknownReadinessK8s := Provider{ID: uuid.New(), TenantID: tenantID, IsGlobal: false, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, IsSchedulable: true}
	tenantBuildNodes := Provider{ID: uuid.New(), TenantID: tenantID, IsGlobal: false, Status: ProviderStatusOnline, ProviderType: ProviderTypeBuildNodes, IsSchedulable: true}

	repo := &testInfrastructureRepo{
		systemProviders: []Provider{
			globalReadyK8s,
			globalNotReadyK8s,
			permittedReadyK8s,
			permittedCapacityBlockedK8s,
			globalBuildNodes,
		},
		tenantProviders: map[uuid.UUID][]Provider{
			tenantID: {
				tenantNotReadyK8s,
				tenantUnknownReadinessK8s,
				tenantBuildNodes,
			},
		},
		permissions: map[uuid.UUID]map[uuid.UUID]bool{
			tenantID: {
				permittedReadyK8s.ID:           true,
				permittedCapacityBlockedK8s.ID: true,
			},
		},
	}

	service := NewService(repo, nil, nil)
	available, err := service.GetAvailableProviders(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetAvailableProviders returned error: %v", err)
	}

	got := make(map[uuid.UUID]bool, len(available))
	for _, provider := range available {
		got[provider.ID] = true
	}

	if !got[globalReadyK8s.ID] {
		t.Fatalf("expected global ready k8s provider to be available")
	}
	if !got[permittedReadyK8s.ID] {
		t.Fatalf("expected permitted ready k8s provider to be available")
	}
	if !got[globalBuildNodes.ID] {
		t.Fatalf("expected global build-nodes provider to be available")
	}
	if !got[tenantUnknownReadinessK8s.ID] {
		t.Fatalf("expected tenant k8s provider with unknown readiness to remain available")
	}
	if !got[tenantBuildNodes.ID] {
		t.Fatalf("expected tenant build-nodes provider to be available")
	}
	if got[globalNotReadyK8s.ID] {
		t.Fatalf("expected global not-ready k8s provider to be filtered out")
	}
	if got[permittedCapacityBlockedK8s.ID] {
		t.Fatalf("expected capacity-blocked k8s provider to be filtered out")
	}
	if got[tenantNotReadyK8s.ID] {
		t.Fatalf("expected tenant not-ready k8s provider to be filtered out")
	}
}

func TestGetAvailableProviders_SystemTenantStillFiltersUnschedulableK8s(t *testing.T) {
	systemTenantID := uuid.Nil
	ready := "ready"
	notReady := "not_ready"

	readyProvider := Provider{ID: uuid.New(), TenantID: systemTenantID, IsGlobal: true, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, ReadinessStatus: &ready, IsSchedulable: true}
	notReadyProvider := Provider{ID: uuid.New(), TenantID: systemTenantID, IsGlobal: true, Status: ProviderStatusOnline, ProviderType: ProviderTypeKubernetes, ReadinessStatus: &notReady, IsSchedulable: false}

	repo := &testInfrastructureRepo{
		systemProviders: []Provider{readyProvider, notReadyProvider},
	}

	service := NewService(repo, nil, nil)
	available, err := service.GetAvailableProviders(context.Background(), systemTenantID)
	if err != nil {
		t.Fatalf("GetAvailableProviders returned error: %v", err)
	}
	if len(available) != 1 {
		t.Fatalf("expected exactly 1 provider, got %d", len(available))
	}
	if available[0].ID != readyProvider.ID {
		t.Fatalf("expected ready provider %s, got %s", readyProvider.ID, available[0].ID)
	}
}
