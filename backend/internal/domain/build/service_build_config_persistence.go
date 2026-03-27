package build

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// persistBuildConfigFromManifest materializes and stores BuildConfigData from a build manifest.
// It returns true when caller should abort early with a successful create response
// (legacy behavior when config persistence fails after build creation).
func (s *Service) persistBuildConfigFromManifest(ctx context.Context, b *Build, manifest BuildManifest) bool {
	if manifest.BuildConfig == nil {
		return false
	}

	config := buildConfigFromManifest(b.ID(), manifest)

	// Validate the config before saving
	if err := config.Validate(); err != nil {
		s.logger.Error("Failed to validate build config", zap.Error(err), zap.String("build_id", b.ID().String()))
		// Don't fail the build creation, just log the error
		s.logger.Warn("Proceeding with build creation despite config validation warning", zap.String("build_id", b.ID().String()))
		return false
	}

	// Save the config
	if err := s.repository.SaveBuildConfig(ctx, config); err != nil {
		s.logger.Error("Failed to save build config", zap.Error(err), zap.String("build_id", b.ID().String()))
		// Log but don't fail - the build was already created
		return true
	}
	b.SetConfig(config)
	return false
}

func buildConfigFromManifest(buildID uuid.UUID, manifest BuildManifest) *BuildConfigData {
	buildMethod := string(manifest.Type)
	if manifest.Type == BuildTypeContainer {
		buildMethod = string(BuildMethodDocker)
	}

	config := &BuildConfigData{
		BuildID:           buildID,
		BuildMethod:       buildMethod,
		SBOMTool:          manifest.BuildConfig.SBOMTool,
		ScanTool:          manifest.BuildConfig.ScanTool,
		RegistryType:      manifest.BuildConfig.RegistryType,
		SecretManagerType: manifest.BuildConfig.SecretManagerType,
		BuildArgs:         manifest.BuildConfig.BuildArgs,
		Secrets:           manifest.BuildConfig.Secrets,
	}
	if sourceID := metadataString(manifest.Metadata, "source_id", "sourceId"); sourceID != "" {
		if parsed, parseErr := uuid.Parse(sourceID); parseErr == nil {
			config.SourceID = &parsed
		}
	}
	config.RefPolicy = metadataString(manifest.Metadata, "ref_policy", "refPolicy")
	if config.RefPolicy == "" {
		config.RefPolicy = "source_default"
	}
	config.FixedRef = metadataString(manifest.Metadata, "fixed_ref", "fixedRef")

	// Map method-specific fields based on build type
	switch manifest.Type {
	case BuildTypeKaniko:
		config.Dockerfile = manifest.BuildConfig.Dockerfile
		config.BuildContext = manifest.BuildConfig.BuildContext
		config.CacheEnabled = manifest.BuildConfig.Cache
		config.CacheRepo = manifest.BuildConfig.CacheRepo
		config.Metadata = map[string]interface{}{
			"registry_repo":      manifest.BuildConfig.RegistryRepo,
			"skip_unused_stages": manifest.BuildConfig.SkipUnusedStages,
			"registry_auth_id":   uuidString(manifest.BuildConfig.RegistryAuthID),
		}
		if gitURL := metadataString(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl"); gitURL != "" {
			config.Metadata["git_url"] = gitURL
		}
		if gitBranch := metadataString(manifest.Metadata, "git_branch", "gitBranch", "branch"); gitBranch != "" {
			config.Metadata["git_branch"] = gitBranch
		}

	case BuildTypeBuildx:
		config.Dockerfile = manifest.BuildConfig.Dockerfile
		config.BuildContext = manifest.BuildConfig.BuildContext
		config.Platforms = normalizeBuildxPlatforms(manifest.BuildConfig.Platforms)
		config.CacheFrom = manifest.BuildConfig.CacheFrom
		config.CacheTo = manifest.BuildConfig.CacheTo
		config.Metadata = map[string]interface{}{
			"registry_repo":    manifest.BuildConfig.RegistryRepo,
			"registry_auth_id": uuidString(manifest.BuildConfig.RegistryAuthID),
		}
		if gitURL := metadataString(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl"); gitURL != "" {
			config.Metadata["git_url"] = gitURL
		}
		if gitBranch := metadataString(manifest.Metadata, "git_branch", "gitBranch", "branch"); gitBranch != "" {
			config.Metadata["git_branch"] = gitBranch
		}

	case BuildTypeContainer:
		config.Dockerfile = manifest.BuildConfig.Dockerfile
		config.BuildContext = manifest.BuildConfig.BuildContext
		config.TargetStage = manifest.BuildConfig.Target
		config.Metadata = map[string]interface{}{
			"registry_repo":    manifest.BuildConfig.RegistryRepo,
			"registry_auth_id": uuidString(manifest.BuildConfig.RegistryAuthID),
		}
		if gitURL := metadataString(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl"); gitURL != "" {
			config.Metadata["git_url"] = gitURL
		}
		if gitBranch := metadataString(manifest.Metadata, "git_branch", "gitBranch", "branch"); gitBranch != "" {
			config.Metadata["git_branch"] = gitBranch
		}

	case BuildTypePaketo:
		if manifest.BuildConfig.PaketoConfig != nil {
			config.Builder = manifest.BuildConfig.PaketoConfig.Builder
			config.Buildpacks = manifest.BuildConfig.PaketoConfig.Buildpacks
			config.Metadata = map[string]interface{}{
				"env":              manifest.BuildConfig.PaketoConfig.Env,
				"build_args":       manifest.BuildConfig.PaketoConfig.BuildArgs,
				"registry_auth_id": uuidString(manifest.BuildConfig.RegistryAuthID),
			}
		}

	case BuildTypePacker:
		config.PackerTemplate = manifest.BuildConfig.PackerTemplate
		config.PackerTargetProfileID = strings.TrimSpace(manifest.BuildConfig.PackerTargetProfileID)
		config.Metadata = packerMetadataFromBuildConfig(manifest.BuildConfig)
		if config.PackerTargetProfileID == "" {
			config.PackerTargetProfileID = metadataString(config.Metadata, "packer_target_profile_id", "packerTargetProfileId")
		}
	case BuildTypeNix:
		config.Metadata = map[string]interface{}{
			"nix_expression": manifest.BuildConfig.NixExpression,
			"flake_uri":      manifest.BuildConfig.FlakeURI,
			"attributes":     manifest.BuildConfig.Attributes,
			"outputs":        manifest.BuildConfig.Outputs,
			"cache_dir":      manifest.BuildConfig.CacheDir,
			"pure":           manifest.BuildConfig.Pure,
			"show_trace":     manifest.BuildConfig.ShowTrace,
		}
	}

	return config
}
