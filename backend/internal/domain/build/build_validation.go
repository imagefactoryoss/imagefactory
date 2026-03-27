package build

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// validateManifest validates the build manifest.
func validateManifest(manifest BuildManifest) error {
	if manifest.Name == "" {
		return errors.New("build name is required")
	}
	if manifest.Type == "" {
		return errors.New("build type is required")
	}
	if manifest.InfrastructureProviderID != nil {
		if manifest.InfrastructureType == "" {
			return errors.New("infrastructure_type is required when infrastructure_provider_id is set")
		}
		if manifest.InfrastructureType == "auto" {
			return errors.New("infrastructure_provider_id cannot be set when infrastructure_type is auto")
		}
	}
	if manifest.InfrastructureType != "" && manifest.InfrastructureType != "auto" && manifest.InfrastructureProviderID == nil {
		return errors.New("infrastructure_provider_id is required when infrastructure_type is set")
	}

	switch manifest.Type {
	case BuildTypeContainer:
		if manifest.BuildConfig != nil && manifest.BuildConfig.Dockerfile != "" {
			if manifest.BuildConfig.BuildContext == "" {
				return errors.New("build context is required for container builds")
			}
			if manifest.BuildConfig.RegistryRepo == "" {
				return errors.New("registry_repo is required for container builds")
			}
		} else {
			if manifest.BaseImage == "" {
				return errors.New("base image is required for container builds")
			}
			if len(manifest.Instructions) == 0 {
				return errors.New("build instructions are required for container builds")
			}
		}
	case BuildTypeVM:
		if manifest.VMConfig == nil {
			return errors.New("VM configuration is required for VM builds")
		}
		if manifest.VMConfig.CloudProvider == "" {
			return errors.New("cloud provider is required for VM builds")
		}
		if manifest.VMConfig.PackerTemplate == "" && manifest.BaseImage == "" {
			return errors.New("either packer template or base image is required for VM builds")
		}
		if manifest.VMConfig.OutputFormat == "" {
			return errors.New("output format is required for VM builds")
		}
	case BuildTypeCloud:
		if manifest.BaseImage == "" {
			return errors.New("base image is required for cloud builds")
		}
	case BuildTypePacker:
		if manifest.BuildConfig == nil {
			return errors.New("build config is required for packer builds")
		}
		if manifest.BuildConfig.PackerTemplate == "" {
			return errors.New("packer template is required for packer builds")
		}
		if strings.TrimSpace(manifest.BuildConfig.PackerTargetProfileID) == "" {
			return errors.New("packer_target_profile_id is required for packer builds")
		}
		if _, err := uuid.Parse(strings.TrimSpace(manifest.BuildConfig.PackerTargetProfileID)); err != nil {
			return errors.New("packer_target_profile_id must be a valid UUID")
		}
	case BuildTypePaketo:
		if manifest.BuildConfig == nil {
			return errors.New("build config is required for paketo builds")
		}
		if manifest.BuildConfig.PaketoConfig == nil {
			return errors.New("paketo config is required for paketo builds")
		}
		if manifest.BuildConfig.PaketoConfig.Builder == "" {
			return errors.New("paketo builder is required")
		}
	case BuildTypeKaniko:
		if manifest.BuildConfig == nil {
			return errors.New("build config is required for kaniko builds")
		}
		if manifest.BuildConfig.Dockerfile == "" {
			return errors.New("dockerfile is required for kaniko builds")
		}
		if manifest.BuildConfig.BuildContext == "" {
			return errors.New("build context is required for kaniko builds")
		}
		if manifest.BuildConfig.RegistryRepo == "" {
			return errors.New("registry_repo is required for kaniko builds")
		}
	case BuildTypeBuildx:
		if manifest.BuildConfig == nil {
			return errors.New("build config is required for buildx builds")
		}
		if manifest.BuildConfig.Dockerfile == "" {
			return errors.New("dockerfile is required for buildx builds")
		}
		if manifest.BuildConfig.BuildContext == "" {
			return errors.New("build context is required for buildx builds")
		}
		if manifest.BuildConfig.RegistryRepo == "" {
			return errors.New("registry_repo is required for buildx builds")
		}
		if len(manifest.BuildConfig.Platforms) == 0 {
			return errors.New("platforms are required for buildx builds")
		}
	case BuildTypeNix:
		if manifest.BuildConfig == nil {
			return errors.New("build config is required for nix builds")
		}
		if manifest.BuildConfig.NixExpression == "" && manifest.BuildConfig.FlakeURI == "" {
			return errors.New("either nix_expression or flake_uri is required for nix builds")
		}
	default:
		return fmt.Errorf("unsupported build type: %s", manifest.Type)
	}

	if manifest.BuildConfig != nil {
		if err := validateBuildConfig(*manifest.BuildConfig); err != nil {
			return fmt.Errorf("invalid build config: %w", err)
		}
	}

	return nil
}

// validateBuildConfig validates the build configuration.
func validateBuildConfig(config BuildConfig) error {
	switch config.BuildType {
	case BuildTypeContainer, BuildTypePacker, BuildTypePaketo, BuildTypeKaniko, BuildTypeBuildx, BuildTypeNix:
	default:
		return fmt.Errorf("invalid build type in config: %s", config.BuildType)
	}

	switch config.SBOMTool {
	case SBOMToolSyft, SBOMToolGrype, SBOMToolTrivy:
	default:
		return fmt.Errorf("invalid SBOM tool: %s", config.SBOMTool)
	}

	switch config.ScanTool {
	case ScanToolTrivy, ScanToolClair, ScanToolGrype, ScanToolSnyk:
	default:
		return fmt.Errorf("invalid scan tool: %s", config.ScanTool)
	}

	switch config.RegistryType {
	case RegistryTypeS3, RegistryTypeHarbor, RegistryTypeQuay, RegistryTypeArtifactory:
	default:
		return fmt.Errorf("invalid registry type: %s", config.RegistryType)
	}

	switch config.SecretManagerType {
	case SecretManagerVault, SecretManagerAWSSM, SecretManagerAzureKV, SecretManagerGCP:
	default:
		return fmt.Errorf("invalid secret manager type: %s", config.SecretManagerType)
	}

	return nil
}
