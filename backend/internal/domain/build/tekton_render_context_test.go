package build

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateTektonRenderContext_KanikoAllowsInlineDockerfileWithBase64(t *testing.T) {
	ctx := TektonRenderContext{
		GitURL:                 "https://github.com/example/repo.git",
		ImageName:              "registry.example.com/team/app:latest",
		DockerfilePath:         "Dockerfile",
		DockerfileInlineBase64: "RlJPTSBhbHBpbmU6My4yMApSVU4gZWNobyBoaQo=",
	}

	if err := validateTektonRenderContext(ctx, BuildMethodKaniko); err != nil {
		t.Fatalf("expected inline dockerfile to be valid for kaniko, got: %v", err)
	}
}

func TestValidateTektonRenderContext_RejectsInlineDockerfileContentWithoutBase64(t *testing.T) {
	ctx := TektonRenderContext{
		GitURL:         "https://github.com/example/repo.git",
		ImageName:      "registry.example.com/team/app:latest",
		DockerfilePath: "FROM alpine:3.20\nRUN echo hi\n",
	}

	err := validateTektonRenderContext(ctx, BuildMethodKaniko)
	if err == nil {
		t.Fatalf("expected inline dockerfile content validation error")
	}
}

func TestNewTektonRenderContext_DefaultBuildContext(t *testing.T) {
	buildID := uuid.New()
	cfg, err := NewKanikoConfig(buildID, "Dockerfile", ".", "registry.example.com/team/app:latest")
	if err != nil {
		t.Fatalf("failed creating kaniko config: %v", err)
	}

	ctx := newTektonRenderContext(nil, cfg, BuildMethodKaniko)
	if ctx.BuildContext != "." {
		t.Fatalf("expected default build context '.', got %q", ctx.BuildContext)
	}
}

func TestNewTektonRenderContext_UsesConfiguredBuildContext(t *testing.T) {
	buildID := uuid.New()
	cfg, err := NewKanikoConfig(buildID, "Dockerfile", "backend", "registry.example.com/team/app:latest")
	if err != nil {
		t.Fatalf("failed creating kaniko config: %v", err)
	}

	ctx := newTektonRenderContext(nil, cfg, BuildMethodKaniko)
	if ctx.BuildContext != "backend" {
		t.Fatalf("expected build context 'backend', got %q", ctx.BuildContext)
	}
}

func TestNewTektonRenderContext_UsesMetadataFeatureFlags(t *testing.T) {
	build := &Build{
		id:       uuid.New(),
		tenantID: uuid.New(),
		manifest: BuildManifest{
			Metadata: map[string]interface{}{
				"git_url":                "https://github.com/example/repo.git",
				"enable_scan":            false,
				"enable_sbom":            "false",
				"enable_sign":            true,
				"sign_key_secret":        "cosign-key",
				"enable_temp_scan_stage": true,
			},
			BuildConfig: &BuildConfig{
				RegistryRepo: "registry.example.com/team/app:latest",
			},
		},
	}
	buildID := uuid.New()
	cfg, err := NewKanikoConfig(buildID, "Dockerfile", ".", "registry.example.com/team/app:latest")
	if err != nil {
		t.Fatalf("failed creating kaniko config: %v", err)
	}

	ctx := newTektonRenderContext(build, cfg, BuildMethodKaniko)
	if ctx.EnableScan != "false" {
		t.Fatalf("expected enable scan false, got %q", ctx.EnableScan)
	}
	if ctx.EnableSBOM != "false" {
		t.Fatalf("expected enable sbom false, got %q", ctx.EnableSBOM)
	}
	if ctx.EnableSign != "true" {
		t.Fatalf("expected enable sign true, got %q", ctx.EnableSign)
	}
	if !ctx.IncludeSignKey || ctx.SignKeySecretName != "cosign-key" {
		t.Fatalf("expected signing key secret metadata to be reflected in context")
	}
	if ctx.EnableTempScanStage != "true" {
		t.Fatalf("expected temp scan stage true, got %q", ctx.EnableTempScanStage)
	}
	if ctx.TempScanImageName == "" {
		t.Fatalf("expected derived temp scan image name")
	}
	if ctx.ScanSourceImageRef == "" {
		t.Fatalf("expected scan source image ref for temp scan stage")
	}
	if ctx.SBOMSource != "docker-archive:/workspace/source/.image/image.tar" {
		t.Fatalf("expected default tar sbom source for compatibility, got %q", ctx.SBOMSource)
	}
}

func TestNewTektonRenderContext_DefaultTempScanStageFromEnv(t *testing.T) {
	t.Setenv("IF_ENABLE_TEMP_SCAN_STAGE", "true")
	build := &Build{
		id:       uuid.New(),
		tenantID: uuid.New(),
		manifest: BuildManifest{
			Metadata: map[string]interface{}{
				"git_url": "https://github.com/example/repo.git",
			},
			BuildConfig: &BuildConfig{
				RegistryRepo: "registry.example.com/team/app:latest",
			},
		},
	}
	buildID := uuid.New()
	cfg, err := NewKanikoConfig(buildID, "Dockerfile", ".", "registry.example.com/team/app:latest")
	if err != nil {
		t.Fatalf("failed creating kaniko config: %v", err)
	}

	ctx := newTektonRenderContext(build, cfg, BuildMethodKaniko)
	if ctx.EnableTempScanStage != "true" {
		t.Fatalf("expected env-enabled temp scan stage, got %q", ctx.EnableTempScanStage)
	}
	if ctx.ScanSourceImageRef == "" || ctx.SBOMSource == "" {
		t.Fatalf("expected scan/sbom sources to be derived when env enables temp scan")
	}
}

func TestNewTektonRenderContext_BuildxUsesBuildConfigRegistryRepoAsImageName(t *testing.T) {
	build := &Build{
		id:       uuid.New(),
		tenantID: uuid.New(),
		manifest: BuildManifest{
			Metadata: map[string]interface{}{
				"git_url": "https://github.com/example/repo.git",
			},
			BuildConfig: &BuildConfig{
				RegistryRepo: "registry.example.com/team/buildx-app:latest",
			},
		},
	}
	cfg, err := NewBuildxConfig(uuid.New(), "Dockerfile", ".")
	if err != nil {
		t.Fatalf("failed creating buildx config: %v", err)
	}

	ctx := newTektonRenderContext(build, cfg, BuildMethodBuildx)
	if ctx.ImageName != "registry.example.com/team/buildx-app:latest" {
		t.Fatalf("expected image name from build config registry repo, got %q", ctx.ImageName)
	}
}

func TestNewTektonRenderContext_UsesMetadataRegistryRepoFallbackForImageName(t *testing.T) {
	build := &Build{
		id:       uuid.New(),
		tenantID: uuid.New(),
		manifest: BuildManifest{
			Metadata: map[string]interface{}{
				"git_url":       "https://github.com/example/repo.git",
				"registry_repo": "registry.example.com/team/fallback-app:latest",
			},
		},
	}
	cfg, err := NewBuildxConfig(uuid.New(), "Dockerfile", ".")
	if err != nil {
		t.Fatalf("failed creating buildx config: %v", err)
	}

	ctx := newTektonRenderContext(build, cfg, BuildMethodBuildx)
	if ctx.ImageName != "registry.example.com/team/fallback-app:latest" {
		t.Fatalf("expected image name from metadata fallback, got %q", ctx.ImageName)
	}
}

func TestNewTektonRenderContext_BuildxNormalizesDuplicatePlatforms(t *testing.T) {
	build := &Build{
		id:       uuid.New(),
		tenantID: uuid.New(),
		manifest: BuildManifest{
			Metadata: map[string]interface{}{
				"git_url": "https://github.com/example/repo.git",
			},
			BuildConfig: &BuildConfig{
				RegistryRepo: "registry.example.com/team/buildx-app:latest",
			},
		},
	}
	cfg := &BuildxConfig{
		buildID:      uuid.New(),
		dockerfile:   "Dockerfile",
		buildContext: ".",
		platforms:    []string{"linux/amd64", "linux/amd64", " linux/arm64 "},
	}

	ctx := newTektonRenderContext(build, cfg, BuildMethodBuildx)
	if ctx.Platforms != "linux/amd64,linux/arm64" {
		t.Fatalf("expected deduplicated platforms, got %q", ctx.Platforms)
	}
}

func TestNewTektonRenderContext_PackerMergesVariablesAndBuildVars(t *testing.T) {
	cfg, err := NewPackerConfig(uuid.New(), "templates/base.pkr.hcl")
	if err != nil {
		t.Fatalf("failed creating packer config: %v", err)
	}
	if err := cfg.SetVariable("region", "us-east-1"); err != nil {
		t.Fatalf("failed setting variable: %v", err)
	}
	if err := cfg.SetBuildVar("image_name", "base-ami"); err != nil {
		t.Fatalf("failed setting build var: %v", err)
	}

	ctx := newTektonRenderContext(nil, cfg, BuildMethodPacker)
	if len(ctx.PackerVars) != 2 {
		t.Fatalf("expected 2 packer vars, got %d (%v)", len(ctx.PackerVars), ctx.PackerVars)
	}
	if ctx.PackerVars[0] != "image_name=base-ami" || ctx.PackerVars[1] != "region=us-east-1" {
		t.Fatalf("expected deterministic sorted packer vars, got %v", ctx.PackerVars)
	}
}

func TestNewTektonRenderContext_PackerOnErrorDefaultsAndOverride(t *testing.T) {
	cfg, err := NewPackerConfig(uuid.New(), "templates/base.pkr.hcl")
	if err != nil {
		t.Fatalf("failed creating packer config: %v", err)
	}

	ctx := newTektonRenderContext(nil, cfg, BuildMethodPacker)
	if ctx.PackerOnError != "cleanup" {
		t.Fatalf("expected default on_error cleanup, got %q", ctx.PackerOnError)
	}

	if err := cfg.SetOnError("abort"); err != nil {
		t.Fatalf("failed setting on_error: %v", err)
	}
	ctx = newTektonRenderContext(nil, cfg, BuildMethodPacker)
	if ctx.PackerOnError != "abort" {
		t.Fatalf("expected on_error override abort, got %q", ctx.PackerOnError)
	}
}
