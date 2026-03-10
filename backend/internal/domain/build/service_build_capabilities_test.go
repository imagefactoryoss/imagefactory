package build

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

type buildCapabilitiesConfigStub struct {
	config *systemconfig.BuildCapabilitiesConfig
}

func (s *buildCapabilitiesConfigStub) GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.BuildConfig, error) {
	return &systemconfig.BuildConfig{}, nil
}

func (s *buildCapabilitiesConfigStub) GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.ToolAvailabilityConfig, error) {
	return &systemconfig.ToolAvailabilityConfig{
		BuildMethods:   systemconfig.BuildMethodAvailability{Packer: true, Buildx: true, Kaniko: true, Container: true, Nix: true},
		SBOMTools:      systemconfig.SBOMToolAvailability{Syft: true},
		ScanTools:      systemconfig.ScanToolAvailability{Trivy: true},
		RegistryTypes:  systemconfig.RegistryTypeAvailability{S3: true},
		SecretManagers: systemconfig.SecretManagerAvailability{Vault: true},
	}, nil
}

func (s *buildCapabilitiesConfigStub) GetBuildCapabilitiesConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.BuildCapabilitiesConfig, error) {
	if s.config == nil {
		return &systemconfig.BuildCapabilitiesConfig{}, nil
	}
	return s.config, nil
}

func TestValidateBuildCapabilities_DeniesDisabledCapability(t *testing.T) {
	service := &Service{
		systemConfigService: &buildCapabilitiesConfigStub{
			config: &systemconfig.BuildCapabilitiesConfig{
				GPU:            false,
				Privileged:     true,
				MultiArch:      true,
				HighMemory:     true,
				HostNetworking: true,
				Premium:        true,
			},
		},
		logger: zap.NewNop(),
	}

	manifest := BuildManifest{
		Type: BuildTypeKaniko,
		Metadata: map[string]interface{}{
			"requires_gpu": true,
		},
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeS3,
			SecretManagerType: SecretManagerVault,
		},
	}

	err := service.validateBuildCapabilities(context.Background(), uuid.New(), manifest)
	if err == nil {
		t.Fatalf("expected gpu capability validation error")
	}
	if !strings.Contains(err.Error(), "gpu build capability is not entitled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBuildCapabilities_AllowsEntitledCapabilities(t *testing.T) {
	service := &Service{
		systemConfigService: &buildCapabilitiesConfigStub{
			config: &systemconfig.BuildCapabilitiesConfig{
				GPU:            true,
				Privileged:     true,
				MultiArch:      true,
				HighMemory:     false,
				HostNetworking: false,
				Premium:        false,
			},
		},
		logger: zap.NewNop(),
	}

	manifest := BuildManifest{
		Type: BuildTypeBuildx,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeBuildx,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeS3,
			SecretManagerType: SecretManagerVault,
			Platforms:         []string{"linux/amd64", "linux/arm64"},
		},
	}

	if err := service.validateBuildCapabilities(context.Background(), uuid.New(), manifest); err != nil {
		t.Fatalf("expected capability validation success, got error: %v", err)
	}
}
