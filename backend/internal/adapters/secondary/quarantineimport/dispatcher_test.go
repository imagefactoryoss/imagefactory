package quarantineimport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	domainimageimport "github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pipelineManagerStub struct {
	namespace string
	yaml      string
}

func (s *pipelineManagerStub) CreatePipelineRun(ctx context.Context, namespace, yamlContent string) (*tektonv1.PipelineRun, error) {
	s.namespace = namespace
	s.yaml = yamlContent
	return &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "quarantine-import-abc123",
		},
	}, nil
}

func (s *pipelineManagerStub) GetPipelineRun(ctx context.Context, namespace, name string) (*tektonv1.PipelineRun, error) {
	return nil, nil
}

func (s *pipelineManagerStub) ListPipelineRuns(ctx context.Context, namespace string, limit int) ([]*tektonv1.PipelineRun, error) {
	return nil, nil
}

func (s *pipelineManagerStub) DeletePipelineRun(ctx context.Context, namespace, name string) error {
	return nil
}

func (s *pipelineManagerStub) GetLogs(ctx context.Context, namespace, pipelineRunName string) (map[string]string, error) {
	return nil, nil
}

func TestDispatcher_DispatchTektonRendersPipelineRun(t *testing.T) {
	mgr := &pipelineManagerStub{}
	dispatcher := NewTektonDispatcher(mgr, "quarantine-ns", "image-factory-build-v1-quarantine-import", "registry.local", "regcred", zap.NewNop())
	req, err := domainimageimport.NewImportRequest(uuid.New(), uuid.New(), domainimageimport.RequestTypeQuarantine, "APP-100", "ghcr.io", "ghcr.io/acme/api:1.2.3", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	result, err := dispatcher.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mgr.namespace != "quarantine-ns" {
		t.Fatalf("unexpected namespace %s", mgr.namespace)
	}
	if !strings.Contains(mgr.yaml, "pipelineRef:") || !strings.Contains(mgr.yaml, "name: image-factory-build-v1-quarantine-import") {
		t.Fatalf("expected pipelineRef in rendered YAML")
	}
	if !strings.Contains(mgr.yaml, "secretName: regcred") {
		t.Fatalf("expected dockerconfig secret workspace binding")
	}
	if !strings.Contains(mgr.yaml, "policy-snapshot-json") {
		t.Fatalf("expected policy snapshot param in rendered YAML")
	}
	if strings.TrimSpace(result.PipelineRunName) == "" {
		t.Fatalf("expected pipeline run name in dispatch result")
	}
	if strings.TrimSpace(result.InternalImageRef) == "" {
		t.Fatalf("expected internal image ref in dispatch result")
	}
}

func TestDispatcher_DispatchTektonUsesDefaultDockerConfigSecret(t *testing.T) {
	mgr := &pipelineManagerStub{}
	dispatcher := NewTektonDispatcher(
		mgr,
		"quarantine-ns",
		"image-factory-build-v1-quarantine-import",
		"registry.local",
		"",
		zap.NewNop(),
	)
	req, err := domainimageimport.NewImportRequest(
		uuid.New(),
		uuid.New(),
		domainimageimport.RequestTypeQuarantine,
		"APP-100",
		"docker.io",
		"library/nginx:latest",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if _, err := dispatcher.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(mgr.yaml, "secretName: docker-config") {
		t.Fatalf("expected default docker-config secret workspace binding")
	}
}

func TestDispatcher_DispatchTektonUsesRequestPolicySnapshot(t *testing.T) {
	mgr := &pipelineManagerStub{}
	dispatcher := NewTektonDispatcher(mgr, "quarantine-ns", "image-factory-build-v1-quarantine-import", "registry.local", "regcred", zap.NewNop())
	req, err := domainimageimport.NewImportRequest(uuid.New(), uuid.New(), domainimageimport.RequestTypeQuarantine, "APP-200", "ghcr.io", "ghcr.io/acme/api:3.0.0", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.PolicySnapshotJSON = `{"mode":"enforce","max_critical":1,"max_p2":2,"max_p3":3,"max_cvss":7.5}`
	if _, err := dispatcher.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(mgr.yaml, req.PolicySnapshotJSON) {
		t.Fatalf("expected custom policy snapshot to be rendered in pipeline YAML")
	}
}

func TestDispatcher_DispatchFailsWhenPipelineManagerMissing(t *testing.T) {
	dispatcher := NewTektonDispatcher(nil, "quarantine-ns", "image-factory-build-v1-quarantine-import", "registry.local", "regcred", zap.NewNop())
	req, err := domainimageimport.NewImportRequest(uuid.New(), uuid.New(), domainimageimport.RequestTypeQuarantine, "APP-101", "ghcr.io", "ghcr.io/acme/api:2.0.0", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if _, err := dispatcher.Dispatch(context.Background(), req); err == nil {
		t.Fatalf("expected dispatch to fail when pipeline manager is missing")
	}
}

func TestSelectQuarantineDispatchProvider_RequiresTektonAndQuarantineEnabled(t *testing.T) {
	now := time.Now().UTC()
	tenantID := uuid.New()
	validID := uuid.New()
	invalidID := uuid.New()

	selected := selectQuarantineDispatchProvider([]*infrastructure.Provider{
		{
			ID:            invalidID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			IsSchedulable: true,
			Status:        infrastructure.ProviderStatusOnline,
			CreatedAt:     now.Add(-1 * time.Minute),
			Config: map[string]interface{}{
				"tekton_enabled": true,
			},
		},
		{
			ID:            validID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			IsSchedulable: true,
			Status:        infrastructure.ProviderStatusOnline,
			CreatedAt:     now,
			Config: map[string]interface{}{
				"tekton_enabled":              true,
				"quarantine_dispatch_enabled": true,
			},
		},
	})

	if selected == nil {
		t.Fatalf("expected provider to be selected")
	}
	if selected.ID != validID {
		t.Fatalf("expected provider %s, got %s", validID, selected.ID)
	}
}

func TestSelectQuarantineDispatchProvider_UsesPriorityDescending(t *testing.T) {
	now := time.Now().UTC()
	tenantID := uuid.New()
	lowID := uuid.New()
	highID := uuid.New()

	selected := selectQuarantineDispatchProvider([]*infrastructure.Provider{
		{
			ID:            lowID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			IsSchedulable: true,
			Status:        infrastructure.ProviderStatusOnline,
			CreatedAt:     now.Add(-2 * time.Minute),
			Config: map[string]interface{}{
				"tekton_enabled":               true,
				"quarantine_dispatch_enabled":  true,
				"quarantine_dispatch_priority": 1,
			},
		},
		{
			ID:            highID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			IsSchedulable: true,
			Status:        infrastructure.ProviderStatusOnline,
			CreatedAt:     now,
			Config: map[string]interface{}{
				"tekton_enabled":               true,
				"quarantine_dispatch_enabled":  true,
				"quarantine_dispatch_priority": 10,
			},
		},
	})

	if selected == nil {
		t.Fatalf("expected provider to be selected")
	}
	if selected.ID != highID {
		t.Fatalf("expected highest-priority provider %s, got %s", highID, selected.ID)
	}
}

func TestResolveQuarantineDispatchNamespace_PrefersExplicitOverride(t *testing.T) {
	tenantID := uuid.New()
	ns := resolveQuarantineDispatchNamespace(
		&infrastructure.Provider{},
		tenantID,
		"custom-quarantine-ns",
		"default",
	)
	if ns != "custom-quarantine-ns" {
		t.Fatalf("expected explicit namespace override, got %q", ns)
	}
}

func TestResolveQuarantineDispatchNamespace_DefaultsToTenantNamespace(t *testing.T) {
	tenantID := uuid.New()
	provider := &infrastructure.Provider{
		Config: map[string]interface{}{
			"tekton_namespace": "default",
		},
	}
	ns := resolveQuarantineDispatchNamespace(provider, tenantID, "", "default")
	expected := "image-factory-" + tenantID.String()[:8]
	if ns != expected {
		t.Fatalf("expected tenant namespace %q, got %q", expected, ns)
	}
}

func TestResolveQuarantineDispatchNamespace_FallsBackWhenTenantMissing(t *testing.T) {
	ns := resolveQuarantineDispatchNamespace(
		&infrastructure.Provider{
			Config: map[string]interface{}{
				"tekton_target_namespace": "provider-target",
			},
		},
		uuid.Nil,
		"",
		"default",
	)
	if ns != "provider-target" {
		t.Fatalf("expected provider target namespace fallback, got %q", ns)
	}
}
