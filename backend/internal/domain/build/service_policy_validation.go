package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// validateToolAvailability validates that requested tools are available.
// It prefers tenant-scoped configuration and relies on system-config fallback
// behavior when tenant-specific config has not been materialized.
func (s *Service) validateToolAvailability(ctx context.Context, tenantID uuid.UUID, manifest BuildManifest) error {
	// Get tenant-scoped tool availability configuration (with global fallback).
	var toolScope *uuid.UUID
	if tenantID != uuid.Nil {
		toolScope = &tenantID
	}
	toolConfig, err := s.systemConfigService.GetToolAvailabilityConfig(ctx, toolScope)
	if err != nil {
		s.logger.Warn("Failed to get tool availability config, allowing build",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()))
		return nil // Allow build on config error
	}

	if manifest.BuildConfig == nil {
		// Legacy builds without BuildConfig are allowed
		return nil
	}

	// Validate BuildType availability
	buildType := manifest.BuildConfig.BuildType
	if buildType == "" {
		buildType = manifest.Type
	}

	switch buildType {
	case BuildTypeContainer:
		if !toolConfig.BuildMethods.Container {
			return fmt.Errorf("container build method is not available for this tenant")
		}
	case BuildTypePacker:
		if !toolConfig.BuildMethods.Packer {
			return fmt.Errorf("packer build method is not available for this tenant")
		}
	case BuildTypePaketo:
		if !toolConfig.BuildMethods.Paketo {
			return fmt.Errorf("paketo build method is not available for this tenant")
		}
	case BuildTypeKaniko:
		if !toolConfig.BuildMethods.Kaniko {
			return fmt.Errorf("kaniko build method is not available for this tenant")
		}
	case BuildTypeBuildx:
		if !toolConfig.BuildMethods.Buildx {
			return fmt.Errorf("buildx build method is not available for this tenant")
		}
	case BuildTypeNix:
		if !toolConfig.BuildMethods.Nix {
			return fmt.Errorf("nix build method is not available for this tenant")
		}
	default:
		return fmt.Errorf("unsupported build type: %s", buildType)
	}

	// Validate SBOM tool availability
	switch manifest.BuildConfig.SBOMTool {
	case SBOMToolSyft:
		if !toolConfig.SBOMTools.Syft {
			return fmt.Errorf("syft SBOM tool is not available for this tenant")
		}
	case SBOMToolGrype:
		if !toolConfig.SBOMTools.Grype {
			return fmt.Errorf("grype SBOM tool is not available for this tenant")
		}
	case SBOMToolTrivy:
		if !toolConfig.SBOMTools.Trivy {
			return fmt.Errorf("trivy SBOM tool is not available for this tenant")
		}
	default:
		return fmt.Errorf("unsupported SBOM tool: %s", manifest.BuildConfig.SBOMTool)
	}

	// Validate scan tool availability
	switch manifest.BuildConfig.ScanTool {
	case ScanToolTrivy:
		if !toolConfig.ScanTools.Trivy {
			return fmt.Errorf("trivy scan tool is not available for this tenant")
		}
	case ScanToolClair:
		if !toolConfig.ScanTools.Clair {
			return fmt.Errorf("clair scan tool is not available for this tenant")
		}
	case ScanToolGrype:
		if !toolConfig.ScanTools.Grype {
			return fmt.Errorf("grype scan tool is not available for this tenant")
		}
	case ScanToolSnyk:
		if !toolConfig.ScanTools.Snyk {
			return fmt.Errorf("snyk scan tool is not available for this tenant")
		}
	default:
		return fmt.Errorf("unsupported scan tool: %s", manifest.BuildConfig.ScanTool)
	}

	// Validate registry type availability
	switch manifest.BuildConfig.RegistryType {
	case RegistryTypeS3:
		if !toolConfig.RegistryTypes.S3 {
			return fmt.Errorf("S3 registry type is not available for this tenant")
		}
	case RegistryTypeHarbor:
		if !toolConfig.RegistryTypes.Harbor {
			return fmt.Errorf("harbor registry type is not available for this tenant")
		}
	case RegistryTypeQuay:
		if !toolConfig.RegistryTypes.Quay {
			return fmt.Errorf("quay registry type is not available for this tenant")
		}
	case RegistryTypeArtifactory:
		if !toolConfig.RegistryTypes.Artifactory {
			return fmt.Errorf("artifactory registry type is not available for this tenant")
		}
	default:
		return fmt.Errorf("unsupported registry type: %s", manifest.BuildConfig.RegistryType)
	}

	// Validate secret manager type availability
	switch manifest.BuildConfig.SecretManagerType {
	case SecretManagerVault:
		if !toolConfig.SecretManagers.Vault {
			return fmt.Errorf("vault secret manager is not available for this tenant")
		}
	case SecretManagerAWSSM:
		if !toolConfig.SecretManagers.AWSSM {
			return fmt.Errorf("AWS secrets manager is not available for this tenant")
		}
	case SecretManagerAzureKV:
		if !toolConfig.SecretManagers.AzureKV {
			return fmt.Errorf("Azure key vault is not available for this tenant")
		}
	case SecretManagerGCP:
		if !toolConfig.SecretManagers.GCP {
			return fmt.Errorf("GCP secret manager is not available for this tenant")
		}
	default:
		return fmt.Errorf("unsupported secret manager type: %s", manifest.BuildConfig.SecretManagerType)
	}

	return nil
}

func (s *Service) validateBuildCapabilities(ctx context.Context, tenantID uuid.UUID, manifest BuildManifest) error {
	provider, ok := s.systemConfigService.(BuildCapabilitiesConfigProvider)
	if !ok || provider == nil {
		return nil
	}

	var scope *uuid.UUID
	if tenantID != uuid.Nil {
		scope = &tenantID
	}
	cfg, err := provider.GetBuildCapabilitiesConfig(ctx, scope)
	if err != nil {
		s.logger.Warn("Failed to get build capabilities config, allowing build",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()))
		return nil
	}
	if cfg == nil {
		return nil
	}

	requiredCapabilities, err := collectRequiredCapabilities(manifest)
	if err != nil {
		return err
	}

	for _, capability := range requiredCapabilities {
		if isBuildCapabilityEntitled(*cfg, capability) {
			continue
		}
		return fmt.Errorf("%w: %s build capability is not entitled for this tenant", ErrBuildCapabilityNotEntitled, capability)
	}

	return nil
}

func collectRequiredCapabilities(manifest BuildManifest) ([]string, error) {
	required := map[string]struct{}{}

	buildType := manifest.Type
	if manifest.BuildConfig != nil && manifest.BuildConfig.BuildType != "" {
		buildType = manifest.BuildConfig.BuildType
	}
	switch buildType {
	case BuildTypePacker:
		required["privileged"] = struct{}{}
	case BuildTypeBuildx:
		if manifest.BuildConfig != nil && len(manifest.BuildConfig.Platforms) > 1 {
			required["multi_arch"] = struct{}{}
		}
	}

	if metadataTruthy(manifest.Metadata, "requires_gpu", "requiresGpu", "gpu") {
		required["gpu"] = struct{}{}
	}
	if metadataTruthy(manifest.Metadata, "requires_privileged", "requiresPrivileged", "privileged") {
		required["privileged"] = struct{}{}
	}
	if metadataTruthy(manifest.Metadata, "requires_high_memory", "requiresHighMemory", "high_memory", "highMemory") {
		required["high_memory"] = struct{}{}
	}
	if metadataTruthy(manifest.Metadata, "requires_host_networking", "requiresHostNetworking", "host_networking", "hostNetworking") {
		required["host_networking"] = struct{}{}
	}
	if metadataTruthy(manifest.Metadata, "premium", "requires_premium", "requiresPremium") {
		required["premium"] = struct{}{}
	}

	if err := mergeExplicitRequiredCapabilities(required, manifest.Metadata); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(required))
	for key := range required {
		keys = append(keys, key)
	}
	return keys, nil
}

func applyTrivyRuntimeDefaults(manifest *BuildManifest, toolConfig *systemconfig.ToolAvailabilityConfig) {
	if manifest == nil || toolConfig == nil {
		return
	}
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}

	if !hasManifestStringMetadata(manifest.Metadata, "trivy_cache_mode", "trivyCacheMode") {
		if value := strings.TrimSpace(toolConfig.TrivyRuntime.CacheMode); value != "" {
			manifest.Metadata["trivy_cache_mode"] = value
		}
	}
	if !hasManifestStringMetadata(manifest.Metadata, "trivy_db_repository", "trivyDbRepository") {
		if value := strings.TrimSpace(toolConfig.TrivyRuntime.DBRepository); value != "" {
			manifest.Metadata["trivy_db_repository"] = value
		}
	}
	if !hasManifestStringMetadata(manifest.Metadata, "trivy_java_db_repository", "trivyJavaDbRepository") {
		if value := strings.TrimSpace(toolConfig.TrivyRuntime.JavaDBRepository); value != "" {
			manifest.Metadata["trivy_java_db_repository"] = value
		}
	}
}

func applyBuildRuntimeDefaults(manifest *BuildManifest, buildConfig *systemconfig.BuildConfig) {
	if manifest == nil || buildConfig == nil {
		return
	}
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}
	if _, ok := manifest.Metadata["enable_temp_scan_stage"]; !ok {
		if _, ok := manifest.Metadata["enableTempScanStage"]; !ok {
			manifest.Metadata["enable_temp_scan_stage"] = buildConfig.EnableTempScanStage
		}
	}
}

func mergeExplicitRequiredCapabilities(required map[string]struct{}, metadata map[string]interface{}) error {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata["required_capabilities"]
	if !ok || raw == nil {
		return nil
	}

	switch values := raw.(type) {
	case []string:
		for _, capability := range values {
			capability = strings.TrimSpace(strings.ToLower(capability))
			if capability != "" {
				required[capability] = struct{}{}
			}
		}
	case []interface{}:
		for _, value := range values {
			name, ok := value.(string)
			if !ok {
				return fmt.Errorf("required_capabilities must contain string values")
			}
			name = strings.TrimSpace(strings.ToLower(name))
			if name != "" {
				required[name] = struct{}{}
			}
		}
	case string:
		for _, part := range strings.Split(values, ",") {
			name := strings.TrimSpace(strings.ToLower(part))
			if name != "" {
				required[name] = struct{}{}
			}
		}
	default:
		return fmt.Errorf("required_capabilities must be an array or comma-separated string")
	}

	return nil
}

func metadataTruthy(metadata map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			if typed {
				return true
			}
		case string:
			if strings.EqualFold(strings.TrimSpace(typed), "true") {
				return true
			}
		}
	}
	return false
}

func hasManifestStringMetadata(metadata map[string]interface{}, keys ...string) bool {
	if len(metadata) == 0 {
		return false
	}
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		if typed, ok := value.(string); ok && strings.TrimSpace(typed) != "" {
			return true
		}
	}
	return false
}

func isBuildCapabilityEntitled(cfg systemconfig.BuildCapabilitiesConfig, capability string) bool {
	switch strings.ToLower(strings.TrimSpace(capability)) {
	case "gpu":
		return cfg.GPU
	case "privileged":
		return cfg.Privileged
	case "multi_arch", "multi-arch":
		return cfg.MultiArch
	case "high_memory", "high-memory":
		return cfg.HighMemory
	case "host_networking", "host-networking":
		return cfg.HostNetworking
	case "premium":
		return cfg.Premium
	default:
		return false
	}
}
