package health

import "testing"

func TestGetEnvString(t *testing.T) {
	t.Setenv("HC_STR", "")
	if got := getEnvString("HC_STR", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	t.Setenv("HC_STR", "value")
	if got := getEnvString("HC_STR", "fallback"); got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("HC_INT", "")
	if got := getEnvInt("HC_INT", 7); got != 7 {
		t.Fatalf("expected fallback 7, got %d", got)
	}

	t.Setenv("HC_INT", "42")
	if got := getEnvInt("HC_INT", 7); got != 42 {
		t.Fatalf("expected parsed int 42, got %d", got)
	}

	t.Setenv("HC_INT", "not-an-int")
	if got := getEnvInt("HC_INT", 7); got != 7 {
		t.Fatalf("expected fallback on parse error, got %d", got)
	}
}

func TestComponentMetadataFromEnv(t *testing.T) {
	t.Setenv("HC_REPO", "registry.example.com/image-factory/backend")
	t.Setenv("HC_TAG", "v1.2.3")

	meta := componentMetadataFromEnv("HC_REPO", "HC_TAG")
	if meta.Repository != "registry.example.com/image-factory/backend" {
		t.Fatalf("unexpected repository: %s", meta.Repository)
	}
	if meta.Tag != "v1.2.3" {
		t.Fatalf("unexpected tag: %s", meta.Tag)
	}
	if meta.Image != "registry.example.com/image-factory/backend:v1.2.3" {
		t.Fatalf("unexpected image: %s", meta.Image)
	}
}

func TestReadDeploymentInfo(t *testing.T) {
	t.Setenv("IF_SERVER_ENVIRONMENT", "production")
	t.Setenv("IF_HELM_RELEASE_NAME", "image-factory")
	t.Setenv("IF_HELM_RELEASE_NAMESPACE", "image-factory")
	t.Setenv("IF_BACKEND_IMAGE_REPOSITORY", "registry.example.com/image-factory/backend")
	t.Setenv("IF_BACKEND_IMAGE_TAG", "v0.1.0-abc123")
	t.Setenv("IF_FRONTEND_IMAGE_REPOSITORY", "registry.example.com/image-factory/frontend")
	t.Setenv("IF_FRONTEND_IMAGE_TAG", "v0.1.0-abc123")
	t.Setenv("IF_DISPATCHER_URL", "http://image-factory-dispatcher")

	build := BuildInfo{Revision: "abc123def456"}
	info := readDeploymentInfo(build)

	if info.Environment != "production" {
		t.Fatalf("unexpected environment: %s", info.Environment)
	}
	if info.ReleaseName != "image-factory" {
		t.Fatalf("unexpected release name: %s", info.ReleaseName)
	}
	if info.ReleaseNamespace != "image-factory" {
		t.Fatalf("unexpected release namespace: %s", info.ReleaseNamespace)
	}

	backend, ok := info.Components["backend"]
	if !ok {
		t.Fatalf("expected backend component metadata")
	}
	if backend.Image != "registry.example.com/image-factory/backend:v0.1.0-abc123" {
		t.Fatalf("unexpected backend image: %s", backend.Image)
	}
	if backend.Revision != "abc123def456" {
		t.Fatalf("expected backend revision from build info, got: %s", backend.Revision)
	}

	if info.RuntimeEndpoints["dispatcher_url"] != "http://image-factory-dispatcher" {
		t.Fatalf("unexpected dispatcher url: %s", info.RuntimeEndpoints["dispatcher_url"])
	}
}

func TestReadBuildInfo_UsesEnvFallback(t *testing.T) {
	t.Setenv("IF_BUILD_COMMIT", "abc1234")
	t.Setenv("IF_BUILD_TIME", "2026-03-02T10:00:00Z")
	t.Setenv("IF_BUILD_DIRTY", "false")
	t.Setenv("IF_BACKEND_IMAGE_TAG", "v0.1.0-abc1234")

	info := readBuildInfo()
	if info.Revision != "abc1234" {
		t.Fatalf("expected env commit fallback, got %s", info.Revision)
	}
	if info.Time != "2026-03-02T10:00:00Z" {
		t.Fatalf("expected env build time fallback, got %s", info.Time)
	}
	if info.Modified != "false" {
		t.Fatalf("expected env dirty fallback, got %s", info.Modified)
	}
}

func TestParseCommitFromImageTag(t *testing.T) {
	if got := parseCommitFromImageTag("v0.1.0-6c42ac0-hz20260302074157"); got != "6c42ac0" {
		t.Fatalf("expected 6c42ac0, got %s", got)
	}
	if got := parseCommitFromImageTag("latest"); got != "" {
		t.Fatalf("expected empty commit for non-sha tag, got %s", got)
	}
}
