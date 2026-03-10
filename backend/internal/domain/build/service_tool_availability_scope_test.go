package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

type tenantScopedToolAvailabilityConfigStub struct {
	receivedTenantID *uuid.UUID
	configByTenant   map[uuid.UUID]*systemconfig.ToolAvailabilityConfig
}

func (s *tenantScopedToolAvailabilityConfigStub) GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.BuildConfig, error) {
	return &systemconfig.BuildConfig{}, nil
}

func (s *tenantScopedToolAvailabilityConfigStub) GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.ToolAvailabilityConfig, error) {
	s.receivedTenantID = tenantID
	if tenantID == nil {
		return nil, fmt.Errorf("expected tenant-scoped tool availability lookup")
	}

	if cfg, ok := s.configByTenant[*tenantID]; ok {
		return cfg, nil
	}

	return &systemconfig.ToolAvailabilityConfig{
		BuildMethods:   systemconfig.BuildMethodAvailability{Kaniko: true, Container: true, Nix: true},
		SBOMTools:      systemconfig.SBOMToolAvailability{Syft: true},
		ScanTools:      systemconfig.ScanToolAvailability{Trivy: true},
		RegistryTypes:  systemconfig.RegistryTypeAvailability{S3: true},
		SecretManagers: systemconfig.SecretManagerAvailability{Vault: true},
	}, nil
}

func TestValidateToolAvailability_UsesTenantScopedConfig(t *testing.T) {
	tenantID := uuid.New()
	systemConfigStub := &tenantScopedToolAvailabilityConfigStub{}
	service := &Service{
		systemConfigService: systemConfigStub,
		logger:              zap.NewNop(),
	}

	manifest := BuildManifest{
		Type: BuildTypeKaniko,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeS3,
			SecretManagerType: SecretManagerVault,
		},
	}

	if err := service.validateToolAvailability(context.Background(), tenantID, manifest); err != nil {
		t.Fatalf("expected tenant-scoped validation to succeed, got error: %v", err)
	}

	if systemConfigStub.receivedTenantID == nil {
		t.Fatalf("expected tenant ID to be passed to tool availability lookup")
	}
	if *systemConfigStub.receivedTenantID != tenantID {
		t.Fatalf("expected tenant ID %s, got %s", tenantID.String(), systemConfigStub.receivedTenantID.String())
	}
}

func TestValidateToolAvailability_DifferentTenantsCanHaveDifferentMethodPolicies(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	systemConfigStub := &tenantScopedToolAvailabilityConfigStub{
		configByTenant: map[uuid.UUID]*systemconfig.ToolAvailabilityConfig{
			tenantA: {
				BuildMethods:   systemconfig.BuildMethodAvailability{Kaniko: true, Buildx: false, Container: false, Nix: false},
				SBOMTools:      systemconfig.SBOMToolAvailability{Syft: true},
				ScanTools:      systemconfig.ScanToolAvailability{Trivy: true},
				RegistryTypes:  systemconfig.RegistryTypeAvailability{S3: true},
				SecretManagers: systemconfig.SecretManagerAvailability{Vault: true},
			},
			tenantB: {
				BuildMethods:   systemconfig.BuildMethodAvailability{Kaniko: false, Buildx: true, Container: false, Nix: false},
				SBOMTools:      systemconfig.SBOMToolAvailability{Syft: true},
				ScanTools:      systemconfig.ScanToolAvailability{Trivy: true},
				RegistryTypes:  systemconfig.RegistryTypeAvailability{S3: true},
				SecretManagers: systemconfig.SecretManagerAvailability{Vault: true},
			},
		},
	}
	service := &Service{
		systemConfigService: systemConfigStub,
		logger:              zap.NewNop(),
	}

	kanikoManifest := BuildManifest{
		Type: BuildTypeKaniko,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeS3,
			SecretManagerType: SecretManagerVault,
		},
	}

	// Tenant A allows Kaniko
	if err := service.validateToolAvailability(context.Background(), tenantA, kanikoManifest); err != nil {
		t.Fatalf("expected tenant A kaniko validation to succeed, got: %v", err)
	}
	// Tenant B blocks Kaniko
	err := service.validateToolAvailability(context.Background(), tenantB, kanikoManifest)
	if err == nil {
		t.Fatalf("expected tenant B kaniko validation to fail")
	}
	if !strings.Contains(err.Error(), "kaniko build method is not available") {
		t.Fatalf("unexpected tenant B error: %v", err)
	}

	buildxManifest := BuildManifest{
		Type: BuildTypeBuildx,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeBuildx,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeS3,
			SecretManagerType: SecretManagerVault,
		},
	}

	// Tenant A blocks Buildx
	err = service.validateToolAvailability(context.Background(), tenantA, buildxManifest)
	if err == nil {
		t.Fatalf("expected tenant A buildx validation to fail")
	}
	if !strings.Contains(err.Error(), "buildx build method is not available") {
		t.Fatalf("unexpected tenant A error: %v", err)
	}
	// Tenant B allows Buildx
	if err := service.validateToolAvailability(context.Background(), tenantB, buildxManifest); err != nil {
		t.Fatalf("expected tenant B buildx validation to succeed, got: %v", err)
	}
}
