package infrastructure

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type readinessReconcileRepoStub struct {
	provider           *Provider
	updatedProvider    *Provider
	updatedReadiness   string
	updatedMissingList []string
}

func (r *readinessReconcileRepoStub) SaveProvider(ctx context.Context, provider *Provider) error {
	return nil
}

func (r *readinessReconcileRepoStub) FindProviderByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	if r.provider == nil || r.provider.ID != id {
		return nil, nil
	}
	cp := *r.provider
	return &cp, nil
}

func (r *readinessReconcileRepoStub) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	return &ListProvidersResult{}, nil
}

func (r *readinessReconcileRepoStub) FindProvidersAll(ctx context.Context, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	return &ListProvidersResult{}, nil
}

func (r *readinessReconcileRepoStub) UpdateProvider(ctx context.Context, provider *Provider) error {
	cp := *provider
	r.updatedProvider = &cp
	return nil
}

func (r *readinessReconcileRepoStub) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *readinessReconcileRepoStub) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	return false, nil
}

func (r *readinessReconcileRepoStub) SavePermission(ctx context.Context, permission *ProviderPermission) error {
	return nil
}

func (r *readinessReconcileRepoStub) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}

func (r *readinessReconcileRepoStub) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}

func (r *readinessReconcileRepoStub) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	return nil
}

func (r *readinessReconcileRepoStub) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	return false, nil
}

func (r *readinessReconcileRepoStub) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *ProviderHealth) error {
	return nil
}

func (r *readinessReconcileRepoStub) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*ProviderHealth, error) {
	return nil, nil
}

func (r *readinessReconcileRepoStub) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	r.updatedReadiness = status
	r.updatedMissingList = append([]string{}, missingPrereqs...)
	return nil
}

func TestUpdateProviderReadiness_ReadyMarksProviderSchedulable(t *testing.T) {
	providerID := uuid.New()
	provider := &Provider{
		ID:           providerID,
		Status:       ProviderStatusOffline,
		ProviderType: ProviderTypeKubernetes,
	}
	repo := &readinessReconcileRepoStub{provider: provider}
	svc := NewService(repo, nil, zap.NewNop())

	err := svc.UpdateProviderReadiness(context.Background(), providerID, "ready", time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("UpdateProviderReadiness returned error: %v", err)
	}
	if repo.updatedProvider == nil {
		t.Fatalf("expected provider update to be persisted")
	}
	if repo.updatedProvider.Status != ProviderStatusOnline {
		t.Fatalf("expected provider status online, got %s", repo.updatedProvider.Status)
	}
	if !repo.updatedProvider.IsSchedulable {
		t.Fatalf("expected provider to be schedulable")
	}
	if repo.updatedProvider.SchedulableReason == nil || *repo.updatedProvider.SchedulableReason != "provider is ready for scheduling" {
		t.Fatalf("unexpected schedulable reason: %+v", repo.updatedProvider.SchedulableReason)
	}
	if len(repo.updatedProvider.BlockedBy) != 0 {
		t.Fatalf("expected empty blocked_by for schedulable provider, got %+v", repo.updatedProvider.BlockedBy)
	}
	if repo.updatedProvider.HealthStatus == nil || *repo.updatedProvider.HealthStatus != "healthy" {
		t.Fatalf("expected provider health_status=healthy, got %+v", repo.updatedProvider.HealthStatus)
	}
	if repo.updatedProvider.ReadinessStatus == nil || *repo.updatedProvider.ReadinessStatus != "ready" {
		t.Fatalf("expected provider readiness_status=ready, got %+v", repo.updatedProvider.ReadinessStatus)
	}
}

func TestUpdateProviderReadiness_ClusterCapacityMarksProviderUnschedulable(t *testing.T) {
	providerID := uuid.New()
	provider := &Provider{
		ID:           providerID,
		Status:       ProviderStatusOnline,
		ProviderType: ProviderTypeKubernetes,
	}
	repo := &readinessReconcileRepoStub{provider: provider}
	svc := NewService(repo, nil, zap.NewNop())

	err := svc.UpdateProviderReadiness(context.Background(), providerID, "ready", time.Now().UTC(), []string{"cluster_capacity: no ready nodes"})
	if err != nil {
		t.Fatalf("UpdateProviderReadiness returned error: %v", err)
	}
	if repo.updatedProvider == nil {
		t.Fatalf("expected provider update to be persisted")
	}
	if repo.updatedProvider.Status != ProviderStatusOnline {
		t.Fatalf("expected provider status online, got %s", repo.updatedProvider.Status)
	}
	if repo.updatedProvider.IsSchedulable {
		t.Fatalf("expected provider to be unschedulable")
	}
	if repo.updatedProvider.SchedulableReason == nil || *repo.updatedProvider.SchedulableReason != "cluster capacity is not ready" {
		t.Fatalf("unexpected schedulable reason: %+v", repo.updatedProvider.SchedulableReason)
	}
	if len(repo.updatedProvider.BlockedBy) != 1 || repo.updatedProvider.BlockedBy[0] != "cluster_capacity" {
		t.Fatalf("unexpected blocked_by: %+v", repo.updatedProvider.BlockedBy)
	}
	if repo.updatedProvider.HealthStatus == nil || *repo.updatedProvider.HealthStatus != "healthy" {
		t.Fatalf("expected provider health_status=healthy, got %+v", repo.updatedProvider.HealthStatus)
	}
	if repo.updatedProvider.ReadinessStatus == nil || *repo.updatedProvider.ReadinessStatus != "ready" {
		t.Fatalf("expected provider readiness_status=ready, got %+v", repo.updatedProvider.ReadinessStatus)
	}
}

func TestUpdateProviderReadiness_NotReadyMarksProviderUnschedulable(t *testing.T) {
	providerID := uuid.New()
	provider := &Provider{
		ID:           providerID,
		Status:       ProviderStatusOnline,
		ProviderType: ProviderTypeKubernetes,
	}
	repo := &readinessReconcileRepoStub{provider: provider}
	svc := NewService(repo, nil, zap.NewNop())

	err := svc.UpdateProviderReadiness(context.Background(), providerID, "not_ready", time.Now().UTC(), []string{"missing tekton task: git-clone"})
	if err != nil {
		t.Fatalf("UpdateProviderReadiness returned error: %v", err)
	}
	if repo.updatedProvider == nil {
		t.Fatalf("expected provider update to be persisted")
	}
	if repo.updatedProvider.Status != ProviderStatusOffline {
		t.Fatalf("expected provider status offline, got %s", repo.updatedProvider.Status)
	}
	if repo.updatedProvider.IsSchedulable {
		t.Fatalf("expected provider to be unschedulable")
	}
	if repo.updatedProvider.SchedulableReason == nil || *repo.updatedProvider.SchedulableReason != "provider status is offline" {
		t.Fatalf("unexpected schedulable reason: %+v", repo.updatedProvider.SchedulableReason)
	}
	if len(repo.updatedProvider.BlockedBy) != 2 {
		t.Fatalf("expected two blocked_by reasons, got %+v", repo.updatedProvider.BlockedBy)
	}
	if repo.updatedProvider.BlockedBy[0] != "provider_not_ready" || repo.updatedProvider.BlockedBy[1] != "provider_status_offline" {
		t.Fatalf("unexpected blocked_by ordering/content: %+v", repo.updatedProvider.BlockedBy)
	}
	if repo.updatedProvider.HealthStatus == nil || *repo.updatedProvider.HealthStatus != "warning" {
		t.Fatalf("expected provider health_status=warning, got %+v", repo.updatedProvider.HealthStatus)
	}
	if repo.updatedProvider.ReadinessStatus == nil || *repo.updatedProvider.ReadinessStatus != "not_ready" {
		t.Fatalf("expected provider readiness_status=not_ready, got %+v", repo.updatedProvider.ReadinessStatus)
	}
}

func TestUpdateProviderReadiness_NotReadyConnectivityFailureMarksProviderUnhealthy(t *testing.T) {
	providerID := uuid.New()
	provider := &Provider{
		ID:           providerID,
		Status:       ProviderStatusOnline,
		ProviderType: ProviderTypeKubernetes,
	}
	repo := &readinessReconcileRepoStub{provider: provider}
	svc := NewService(repo, nil, zap.NewNop())

	err := svc.UpdateProviderReadiness(
		context.Background(),
		providerID,
		"not_ready",
		time.Now().UTC(),
		[]string{"kubernetes api unreachable: dial tcp 127.0.0.1:6443: connect: connection refused"},
	)
	if err != nil {
		t.Fatalf("UpdateProviderReadiness returned error: %v", err)
	}
	if repo.updatedProvider == nil {
		t.Fatalf("expected provider update to be persisted")
	}
	if repo.updatedProvider.HealthStatus == nil || *repo.updatedProvider.HealthStatus != "unhealthy" {
		t.Fatalf("expected provider health_status=unhealthy, got %+v", repo.updatedProvider.HealthStatus)
	}
}

func TestBootstrapRemediationCommands_ShapeIncludesNamespaceCommands(t *testing.T) {
	namespace := "image-factory-demo"
	commands := bootstrapRemediationCommands(namespace)
	if len(commands) < 5 {
		t.Fatalf("expected at least 5 remediation commands, got %d", len(commands))
	}

	expectedSubstrings := []string{
		"kubectl auth can-i create namespaces",
		"kubectl auth can-i patch tasks.tekton.dev -n " + namespace,
		"kubectl auth can-i patch pipelines.tekton.dev -n " + namespace,
		"kubectl auth can-i create pipelineruns.tekton.dev -n " + namespace,
		"kubectl apply -k backend/tekton -n " + namespace,
	}
	for _, want := range expectedSubstrings {
		found := false
		for _, cmd := range commands {
			if strings.Contains(cmd, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected remediation commands to include %q, got %+v", want, commands)
		}
	}
}
