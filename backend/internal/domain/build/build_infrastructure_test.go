package build

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewBuild_InfrastructureSelectionRequiresProvider(t *testing.T) {
	manifest := BuildManifest{
		Name:               "Build",
		Type:               BuildTypeKaniko,
		InfrastructureType: "kubernetes",
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeHarbor,
			SecretManagerType: SecretManagerVault,
			Dockerfile:        "FROM alpine",
			BuildContext:      ".",
			RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
		},
	}

	_, err := NewBuild(uuid.New(), uuid.New(), manifest, nil)
	if err == nil {
		t.Fatalf("expected error when infrastructure_type set without provider_id")
	}
}

func TestNewBuild_InfrastructureProviderWithAutoRejected(t *testing.T) {
	providerID := uuid.New()
	manifest := BuildManifest{
		Name:                     "Build",
		Type:                     BuildTypeKaniko,
		InfrastructureType:       "auto",
		InfrastructureProviderID: &providerID,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeHarbor,
			SecretManagerType: SecretManagerVault,
			Dockerfile:        "FROM alpine",
			BuildContext:      ".",
			RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
		},
	}

	_, err := NewBuild(uuid.New(), uuid.New(), manifest, nil)
	if err == nil {
		t.Fatalf("expected error when provider_id set with auto infrastructure")
	}
}

func TestNewBuild_InfrastructureProviderWithoutTypeRejected(t *testing.T) {
	providerID := uuid.New()
	manifest := BuildManifest{
		Name:                     "Build",
		Type:                     BuildTypeKaniko,
		InfrastructureProviderID: &providerID,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeHarbor,
			SecretManagerType: SecretManagerVault,
			Dockerfile:        "FROM alpine",
			BuildContext:      ".",
			RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
		},
	}

	_, err := NewBuild(uuid.New(), uuid.New(), manifest, nil)
	if err == nil {
		t.Fatalf("expected error when provider_id set without infrastructure_type")
	}
}

func TestNewBuild_InfrastructureSelectionWithProviderAllowed(t *testing.T) {
	providerID := uuid.New()
	manifest := BuildManifest{
		Name:                     "Build",
		Type:                     BuildTypeKaniko,
		InfrastructureType:       "kubernetes",
		InfrastructureProviderID: &providerID,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeHarbor,
			SecretManagerType: SecretManagerVault,
			Dockerfile:        "FROM alpine",
			BuildContext:      ".",
			RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
		},
	}

	build, err := NewBuild(uuid.New(), uuid.New(), manifest, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if build.InfrastructureProviderID() == nil || *build.InfrastructureProviderID() != providerID {
		t.Fatalf("expected provider ID to be set")
	}
}
