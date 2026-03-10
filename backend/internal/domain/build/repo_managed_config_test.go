package build

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseRepoBuildManifest_VersionedEnvelope(t *testing.T) {
	registryAuthID := uuid.New()
	raw := []byte(`
version: v1
build:
  name: repo-build
  type: buildx
  build_config:
    build_type: buildx
    sbom_tool: syft
    scan_tool: trivy
    registry_type: harbor
    secret_manager_type: vault
    registry_auth_id: "` + registryAuthID.String() + `"
    dockerfile: Dockerfile
    build_context: .
    platforms:
      - linux/amd64
      - linux/arm64
`)

	manifest, err := parseRepoBuildManifest(raw)
	if err != nil {
		t.Fatalf("expected versioned envelope to parse: %v", err)
	}
	if manifest.Name != "repo-build" {
		t.Fatalf("expected name repo-build, got %q", manifest.Name)
	}
	if manifest.Type != BuildTypeBuildx {
		t.Fatalf("expected type buildx, got %q", manifest.Type)
	}
	if manifest.BuildConfig == nil {
		t.Fatalf("expected build_config to parse")
	}
	if manifest.BuildConfig.RegistryAuthID == nil {
		t.Fatalf("expected registry_auth_id to parse")
	}
	if *manifest.BuildConfig.RegistryAuthID != registryAuthID {
		t.Fatalf("expected registry_auth_id %s, got %s", registryAuthID.String(), manifest.BuildConfig.RegistryAuthID.String())
	}
	if len(manifest.BuildConfig.Platforms) != 2 {
		t.Fatalf("expected buildx platforms to parse, got %v", manifest.BuildConfig.Platforms)
	}
}

func TestParseRepoBuildManifest_RejectsUnsupportedVersion(t *testing.T) {
	raw := []byte(`
version: v2
build:
  type: container
  name: test
`)

	_, err := parseRepoBuildManifest(raw)
	if err == nil {
		t.Fatalf("expected unsupported version to fail")
	}
	if !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRepoBuildManifest_VersionedEnvelopeRequiresBuildKey(t *testing.T) {
	raw := []byte(`
version: v1
builder:
  name: wrong-key
  type: kaniko
`)

	_, err := parseRepoBuildManifest(raw)
	if err == nil {
		t.Fatalf("expected missing build key to fail")
	}
	if !strings.Contains(err.Error(), "requires top-level 'build'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeRepoBuildManifest_DoesNotOverrideSourceMetadata(t *testing.T) {
	base := BuildManifest{
		Name: "base",
		Type: BuildTypeKaniko,
		Metadata: map[string]interface{}{
			"git_url": "https://example.com/base.git",
			"custom":  "base",
		},
	}
	override := BuildManifest{
		Metadata: map[string]interface{}{
			"git_url": "https://example.com/repo.git",
			"custom":  "repo",
		},
	}

	merged := mergeRepoBuildManifest(base, override)
	if got := metadataString(merged.Metadata, "git_url"); got != "https://example.com/base.git" {
		t.Fatalf("expected git_url to stay from base manifest, got %q", got)
	}
	if got := metadataString(merged.Metadata, "custom"); got != "repo" {
		t.Fatalf("expected custom metadata to merge, got %q", got)
	}
}

func TestValidateRepoManagedManifestPolicy_RejectsInlineSecrets(t *testing.T) {
	err := validateRepoManagedManifestPolicy(BuildManifest{
		Type: BuildTypeKaniko,
		BuildConfig: &BuildConfig{
			Secrets: map[string]string{"TOKEN": "plain-text"},
		},
	})
	if err == nil {
		t.Fatalf("expected inline secrets policy violation")
	}
	if !strings.Contains(err.Error(), "build_config.secrets") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRepoManagedManifestPolicy_RejectsInfrastructureOverride(t *testing.T) {
	err := validateRepoManagedManifestPolicy(BuildManifest{InfrastructureType: "kubernetes"})
	if err == nil {
		t.Fatalf("expected infrastructure override policy violation")
	}
	if !strings.Contains(err.Error(), "infrastructure selection") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveGitAuthForManifest_PrefersSourceAuthThenProjectAuth(t *testing.T) {
	projectID := uuid.New()
	sourceID := uuid.New()

	service := &Service{
		projectSourceGitAuthLookup: func(ctx context.Context, pid, sid uuid.UUID) (map[string][]byte, error) {
			if pid != projectID || sid != sourceID {
				t.Fatalf("unexpected source lookup ids pid=%s sid=%s", pid, sid)
			}
			return map[string][]byte{"auth_type": []byte("token"), "token": []byte("source-token")}, nil
		},
		projectGitAuthLookup: func(ctx context.Context, pid uuid.UUID) (map[string][]byte, error) {
			t.Fatalf("project fallback should not be used when source auth exists")
			return nil, nil
		},
	}

	got, err := service.resolveGitAuthForManifest(context.Background(), projectID, BuildManifest{
		Metadata: map[string]interface{}{"source_id": sourceID.String()},
	})
	if err != nil {
		t.Fatalf("expected source auth resolution success: %v", err)
	}
	if string(got["token"]) != "source-token" {
		t.Fatalf("expected source token, got %q", string(got["token"]))
	}
}

func TestResolveGitAuthForManifest_FallsBackToProjectAuth(t *testing.T) {
	projectID := uuid.New()
	sourceID := uuid.New()
	projectCalled := false

	service := &Service{
		projectSourceGitAuthLookup: func(ctx context.Context, pid, sid uuid.UUID) (map[string][]byte, error) {
			return nil, nil
		},
		projectGitAuthLookup: func(ctx context.Context, pid uuid.UUID) (map[string][]byte, error) {
			projectCalled = true
			if pid != projectID {
				t.Fatalf("unexpected project id %s", pid)
			}
			return map[string][]byte{"auth_type": []byte("token"), "token": []byte("project-token")}, nil
		},
	}

	got, err := service.resolveGitAuthForManifest(context.Background(), projectID, BuildManifest{
		Metadata: map[string]interface{}{"source_id": sourceID.String()},
	})
	if err != nil {
		t.Fatalf("expected project fallback resolution success: %v", err)
	}
	if !projectCalled {
		t.Fatalf("expected project auth fallback to be used")
	}
	if string(got["token"]) != "project-token" {
		t.Fatalf("expected project token, got %q", string(got["token"]))
	}
}

func TestResolveGitAuthForManifest_InvalidSourceIDUsesProjectAuth(t *testing.T) {
	projectID := uuid.New()
	projectCalled := false
	service := &Service{
		projectSourceGitAuthLookup: func(ctx context.Context, pid, sid uuid.UUID) (map[string][]byte, error) {
			t.Fatalf("source lookup should be skipped for invalid source_id")
			return nil, nil
		},
		projectGitAuthLookup: func(ctx context.Context, pid uuid.UUID) (map[string][]byte, error) {
			projectCalled = true
			return map[string][]byte{"auth_type": []byte("token"), "token": []byte("project-token")}, nil
		},
	}

	got, err := service.resolveGitAuthForManifest(context.Background(), projectID, BuildManifest{
		Metadata: map[string]interface{}{"source_id": "not-a-uuid"},
	})
	if err != nil {
		t.Fatalf("expected fallback success: %v", err)
	}
	if !projectCalled {
		t.Fatalf("expected project auth lookup to run")
	}
	if string(got["token"]) != "project-token" {
		t.Fatalf("expected project token, got %q", string(got["token"]))
	}
}

func TestApplyRepoManagedBuildConfig_ProjectSettingsCanDisableRepoConfig(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	buildObj, err := NewBuild(tenantID, projectID, BuildManifest{
		Name:      "settings-mode-test",
		Type:      BuildTypeContainer,
		BaseImage: "alpine:3.19",
		Instructions: []string{
			"RUN echo ready",
		},
		Metadata: map[string]interface{}{
			"git_url": "https://example.com/private.git",
		},
	}, nil)
	if err != nil {
		t.Fatalf("expected build creation to succeed: %v", err)
	}

	service := &Service{
		projectBuildSettingsLookup: func(ctx context.Context, pid uuid.UUID) (*ProjectBuildSettings, error) {
			if pid != projectID {
				t.Fatalf("unexpected project id: %s", pid)
			}
			return &ProjectBuildSettings{BuildConfigMode: "ui_managed"}, nil
		},
	}

	if err := service.applyRepoManagedBuildConfig(context.Background(), buildObj); err != nil {
		t.Fatalf("expected ui_managed project setting to skip repo config: %v", err)
	}
}
