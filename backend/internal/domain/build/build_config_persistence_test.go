package build

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuildConfigFromManifest_BuildxPersistsRegistryRepoMetadata(t *testing.T) {
	registryAuthID := uuid.New()
	manifest := BuildManifest{
		Type: BuildTypeBuildx,
		BuildConfig: &BuildConfig{
			BuildType:      BuildTypeBuildx,
			Dockerfile:     "Dockerfile",
			BuildContext:   ".",
			Platforms:      []string{"linux/amd64"},
			RegistryRepo:   "registry.example.com/team/app:latest",
			RegistryAuthID: &registryAuthID,
		},
	}

	config := buildConfigFromManifest(uuid.New(), manifest)
	if config == nil {
		t.Fatalf("expected config to be created")
	}
	if got, _ := config.Metadata["registry_repo"].(string); got != "registry.example.com/team/app:latest" {
		t.Fatalf("expected registry_repo metadata to be persisted for buildx, got %q", got)
	}
}

func TestBuildConfigDataFromManifest_BuildxPersistsRegistryRepoMetadata(t *testing.T) {
	registryAuthID := uuid.New()
	manifest := BuildManifest{
		Type: BuildTypeBuildx,
		BuildConfig: &BuildConfig{
			BuildType:      BuildTypeBuildx,
			Dockerfile:     "Dockerfile",
			BuildContext:   ".",
			Platforms:      []string{"linux/amd64"},
			RegistryRepo:   "registry.example.com/team/app:latest",
			RegistryAuthID: &registryAuthID,
		},
	}

	config := buildConfigDataFromManifest(manifest, uuid.New())
	if config == nil {
		t.Fatalf("expected config to be created")
	}
	if got, _ := config.Metadata["registry_repo"].(string); got != "registry.example.com/team/app:latest" {
		t.Fatalf("expected registry_repo metadata to be persisted for buildx, got %q", got)
	}
}

func TestBuildConfigFromManifest_BuildxNormalizesDuplicatePlatforms(t *testing.T) {
	manifest := BuildManifest{
		Type: BuildTypeBuildx,
		BuildConfig: &BuildConfig{
			BuildType:    BuildTypeBuildx,
			Dockerfile:   "Dockerfile",
			BuildContext: ".",
			Platforms:    []string{"linux/amd64", " linux/amd64 ", "linux/arm64", ""},
			RegistryRepo: "registry.example.com/team/app:latest",
		},
	}

	config := buildConfigFromManifest(uuid.New(), manifest)
	if config == nil {
		t.Fatalf("expected config to be created")
	}
	if got, want := len(config.Platforms), 2; got != want {
		t.Fatalf("expected %d normalized platforms, got %d (%v)", want, got, config.Platforms)
	}
	if config.Platforms[0] != "linux/amd64" || config.Platforms[1] != "linux/arm64" {
		t.Fatalf("unexpected normalized platforms order/value: %v", config.Platforms)
	}
}

func TestBuildConfigDataFromManifest_BuildxNormalizesDuplicatePlatforms(t *testing.T) {
	manifest := BuildManifest{
		Type: BuildTypeBuildx,
		BuildConfig: &BuildConfig{
			BuildType:    BuildTypeBuildx,
			Dockerfile:   "Dockerfile",
			BuildContext: ".",
			Platforms:    []string{"linux/amd64", "linux/amd64", "linux/arm64"},
			RegistryRepo: "registry.example.com/team/app:latest",
		},
	}

	config := buildConfigDataFromManifest(manifest, uuid.New())
	if config == nil {
		t.Fatalf("expected config to be created")
	}
	if got, want := len(config.Platforms), 2; got != want {
		t.Fatalf("expected %d normalized platforms, got %d (%v)", want, got, config.Platforms)
	}
	if config.Platforms[0] != "linux/amd64" || config.Platforms[1] != "linux/arm64" {
		t.Fatalf("unexpected normalized platforms order/value: %v", config.Platforms)
	}
}

func TestBuildConfigFromManifest_ContainerPersistsRegistryRepoMetadata(t *testing.T) {
	registryAuthID := uuid.New()
	manifest := BuildManifest{
		Type: BuildTypeContainer,
		BuildConfig: &BuildConfig{
			BuildType:      BuildTypeContainer,
			Dockerfile:     "Dockerfile",
			BuildContext:   ".",
			RegistryRepo:   "registry.example.com/team/docker-app:latest",
			RegistryAuthID: &registryAuthID,
		},
	}

	config := buildConfigFromManifest(uuid.New(), manifest)
	if config == nil {
		t.Fatalf("expected config to be created")
	}
	if got, _ := config.Metadata["registry_repo"].(string); got != "registry.example.com/team/docker-app:latest" {
		t.Fatalf("expected registry_repo metadata to be persisted for container builds, got %q", got)
	}
}

func TestBuildConfigDataFromManifest_ContainerPersistsRegistryRepoMetadata(t *testing.T) {
	registryAuthID := uuid.New()
	manifest := BuildManifest{
		Type: BuildTypeContainer,
		BuildConfig: &BuildConfig{
			BuildType:      BuildTypeContainer,
			Dockerfile:     "Dockerfile",
			BuildContext:   ".",
			RegistryRepo:   "registry.example.com/team/docker-app:latest",
			RegistryAuthID: &registryAuthID,
		},
	}

	config := buildConfigDataFromManifest(manifest, uuid.New())
	if config == nil {
		t.Fatalf("expected config to be created")
	}
	if got, _ := config.Metadata["registry_repo"].(string); got != "registry.example.com/team/docker-app:latest" {
		t.Fatalf("expected registry_repo metadata to be persisted for container builds, got %q", got)
	}
}
