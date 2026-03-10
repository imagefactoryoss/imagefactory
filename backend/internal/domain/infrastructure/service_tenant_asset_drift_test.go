package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

func TestComputeTenantAssetDriftStatus(t *testing.T) {
	tests := []struct {
		name      string
		desired   *string
		installed *string
		want      TenantAssetDriftStatus
	}{
		{
			name:      "unknown when desired missing",
			desired:   nil,
			installed: strPtr("sha256:aaa"),
			want:      TenantAssetDriftStatusUnknown,
		},
		{
			name:      "unknown when installed missing",
			desired:   strPtr("sha256:aaa"),
			installed: nil,
			want:      TenantAssetDriftStatusUnknown,
		},
		{
			name:      "current when versions match",
			desired:   strPtr("sha256:aaa"),
			installed: strPtr("sha256:aaa"),
			want:      TenantAssetDriftStatusCurrent,
		},
		{
			name:      "stale when versions differ",
			desired:   strPtr("sha256:aaa"),
			installed: strPtr("sha256:bbb"),
			want:      TenantAssetDriftStatusStale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeTenantAssetDriftStatus(tt.desired, tt.installed)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCalculateTektonAssetsVersion_OrderStable(t *testing.T) {
	t.Helper()
	dir := t.TempDir()

	a := filepath.Join(dir, "01-task.yaml")
	b := filepath.Join(dir, "02-pipeline.yaml")
	if err := os.WriteFile(a, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write file a: %v", err)
	}
	if err := os.WriteFile(b, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: pipeline\n"), 0o644); err != nil {
		t.Fatalf("failed to write file b: %v", err)
	}

	v1, err := calculateTektonAssetsVersion([]string{a, b})
	if err != nil {
		t.Fatalf("calculateTektonAssetsVersion failed: %v", err)
	}
	v2, err := calculateTektonAssetsVersion([]string{b, a})
	if err != nil {
		t.Fatalf("calculateTektonAssetsVersion failed: %v", err)
	}
	if v1 == "" || v2 == "" {
		t.Fatalf("expected non-empty version signatures, got %q and %q", v1, v2)
	}
	if v1 != v2 {
		t.Fatalf("expected stable version signature across ordering, got %q vs %q", v1, v2)
	}
}

func TestCalculateTektonAssetsVersionWithOverrides_ChangesVersion(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	taskFile := filepath.Join(dir, "task.yaml")
	content := `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: git-clone
spec:
  steps:
    - name: clone
      image: docker.io/alpine/git:2.45.2
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	defaultVersion, err := calculateTektonAssetsVersion([]string{taskFile})
	if err != nil {
		t.Fatalf("failed to calculate default version: %v", err)
	}
	overrideVersion, err := calculateTektonAssetsVersionWithOverrides([]string{taskFile}, &systemconfig.TektonTaskImagesConfig{
		GitClone:       "registry.local/tools/alpine-git:2.45.2",
		KanikoExecutor: "gcr.io/kaniko-project/executor:v1.23.2",
		Buildkit:       "docker.io/moby/buildkit:v0.13.2",
		Skopeo:         "quay.io/skopeo/stable:v1.15.0",
		Trivy:          "docker.io/aquasec/trivy:0.57.1",
		Syft:           "docker.io/anchore/syft:v1.18.1",
		Cosign:         "gcr.io/projectsigstore/cosign:v2.4.1",
		Packer:         "docker.io/hashicorp/packer:1.10.2",
		PythonAlpine:   "docker.io/library/python:3.12-alpine",
		Alpine:         "docker.io/library/alpine:3.20",
		CleanupKubectl: "docker.io/bitnami/kubectl:latest",
	})
	if err != nil {
		t.Fatalf("failed to calculate override version: %v", err)
	}
	if defaultVersion == overrideVersion {
		t.Fatalf("expected version hash to change with image overrides, got %q", defaultVersion)
	}
}

func strPtr(v string) *string {
	return &v
}

type driftRefreshRepoStub struct {
	upsertCalls int
	lastPrepare *ProviderTenantNamespacePrepare
}

func (r *driftRefreshRepoStub) UpsertTenantNamespacePrepare(ctx context.Context, prepare *ProviderTenantNamespacePrepare) error {
	r.upsertCalls++
	cp := *prepare
	r.lastPrepare = &cp
	return nil
}

func (r *driftRefreshRepoStub) GetTenantNamespacePrepare(ctx context.Context, providerID, tenantID uuid.UUID) (*ProviderTenantNamespacePrepare, error) {
	return nil, nil
}

func TestRefreshTenantNamespacePrepareAssetDrift_SkipsUpsertWhenNoChange(t *testing.T) {
	dir := t.TempDir()
	kustomizationPath := filepath.Join(dir, "kustomization.yaml")
	resourcePath := filepath.Join(dir, "task.yaml")
	if err := os.WriteFile(kustomizationPath, []byte("resources:\n  - task.yaml\n"), 0o644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}
	if err := os.WriteFile(resourcePath, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write resource: %v", err)
	}
	t.Setenv("IF_TEKTON_ASSETS_DIR", dir)

	desired, err := calculateTektonAssetsVersion([]string{resourcePath})
	if err != nil {
		t.Fatalf("failed to compute desired version: %v", err)
	}

	repo := &driftRefreshRepoStub{}
	service := &Service{
		tenantNamespacePrepareRepo: repo,
		logger:                     zap.NewNop(),
	}
	prepare := &ProviderTenantNamespacePrepare{
		DesiredAssetVersion:   &desired,
		InstalledAssetVersion: &desired,
		AssetDriftStatus:      TenantAssetDriftStatusCurrent,
	}

	if err := service.refreshTenantNamespacePrepareAssetDrift(context.Background(), prepare); err != nil {
		t.Fatalf("refreshTenantNamespacePrepareAssetDrift failed: %v", err)
	}
	if repo.upsertCalls != 0 {
		t.Fatalf("expected no upsert when drift state unchanged, got %d", repo.upsertCalls)
	}
}

func TestRefreshTenantNamespacePrepareAssetDrift_UpsertsWhenChanged(t *testing.T) {
	dir := t.TempDir()
	kustomizationPath := filepath.Join(dir, "kustomization.yaml")
	resourcePath := filepath.Join(dir, "task.yaml")
	if err := os.WriteFile(kustomizationPath, []byte("resources:\n  - task.yaml\n"), 0o644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}
	if err := os.WriteFile(resourcePath, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: task\n"), 0o644); err != nil {
		t.Fatalf("failed to write resource: %v", err)
	}
	t.Setenv("IF_TEKTON_ASSETS_DIR", dir)

	desired, err := calculateTektonAssetsVersion([]string{resourcePath})
	if err != nil {
		t.Fatalf("failed to compute desired version: %v", err)
	}

	repo := &driftRefreshRepoStub{}
	service := &Service{
		tenantNamespacePrepareRepo: repo,
		logger:                     zap.NewNop(),
	}
	prepare := &ProviderTenantNamespacePrepare{
		InstalledAssetVersion: &desired,
		AssetDriftStatus:      TenantAssetDriftStatusUnknown,
	}

	if err := service.refreshTenantNamespacePrepareAssetDrift(context.Background(), prepare); err != nil {
		t.Fatalf("refreshTenantNamespacePrepareAssetDrift failed: %v", err)
	}
	if repo.upsertCalls != 1 {
		t.Fatalf("expected upsert when drift state changes, got %d", repo.upsertCalls)
	}
	if repo.lastPrepare == nil || repo.lastPrepare.DesiredAssetVersion == nil {
		t.Fatalf("expected desired asset version to be persisted")
	}
	if repo.lastPrepare.AssetDriftStatus != TenantAssetDriftStatusCurrent {
		t.Fatalf("expected asset drift status current, got %q", repo.lastPrepare.AssetDriftStatus)
	}
}
