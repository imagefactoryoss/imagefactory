package build

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"sigs.k8s.io/yaml"
)

var errRepoBuildConfigNotFound = errors.New("repo build config file not found")

type projectGitAuthLookup func(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error)
type projectSourceGitAuthLookup func(ctx context.Context, projectID, sourceID uuid.UUID) (map[string][]byte, error)
type projectBuildSettingsLookup func(ctx context.Context, projectID uuid.UUID) (*ProjectBuildSettings, error)

type ProjectBuildSettings struct {
	BuildConfigMode    string
	BuildConfigFile    string
	BuildConfigOnError string
}

type repoBuildConfigEnvelope struct {
	Version string      `json:"version"`
	Build   interface{} `json:"build"`
}

func (s *Service) SetProjectGitAuthLookup(lookup func(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error)) {
	if lookup == nil {
		s.projectGitAuthLookup = nil
		return
	}
	s.projectGitAuthLookup = projectGitAuthLookup(lookup)
}

func (s *Service) SetProjectSourceGitAuthLookup(lookup func(ctx context.Context, projectID, sourceID uuid.UUID) (map[string][]byte, error)) {
	if lookup == nil {
		s.projectSourceGitAuthLookup = nil
		return
	}
	s.projectSourceGitAuthLookup = projectSourceGitAuthLookup(lookup)
}

func (s *Service) applyRepoManagedBuildConfig(ctx context.Context, b *Build) error {
	if b == nil {
		return nil
	}

	manifest := b.Manifest()
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}

	var settings *ProjectBuildSettings
	if s.projectBuildSettingsLookup != nil {
		resolvedSettings, settingsErr := s.projectBuildSettingsLookup(ctx, b.ProjectID())
		if settingsErr != nil {
			return fmt.Errorf("failed to resolve project build settings: %w", settingsErr)
		}
		settings = resolvedSettings
	}

	mode := strings.ToLower(strings.TrimSpace(metadataString(manifest.Metadata, "build_config_mode", "buildConfigMode")))
	if mode == "" && settings != nil {
		mode = strings.ToLower(strings.TrimSpace(settings.BuildConfigMode))
	}
	if mode == "" {
		mode = "repo_managed"
	}
	if mode == "ui_managed" {
		return nil
	}

	repoURL := strings.TrimSpace(metadataString(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl"))
	if repoURL == "" {
		return nil
	}

	ref := strings.TrimSpace(metadataString(manifest.Metadata, "git_branch", "gitBranch", "branch", "trigger_ref", "triggerRef"))
	if ref == "" {
		ref = "main"
	}

	configPath := strings.TrimSpace(metadataString(manifest.Metadata, "build_config_file", "buildConfigFile"))
	if configPath == "" && settings != nil {
		configPath = strings.TrimSpace(settings.BuildConfigFile)
	}
	if configPath == "" {
		configPath = "image-factory.yaml"
	}

	gitAuthSecret, err := s.resolveGitAuthForManifest(ctx, b.ProjectID(), manifest)
	if err != nil {
		return err
	}
	raw, err := s.fetchRepositoryFile(ctx, repoURL, ref, configPath, gitAuthSecret)
	if err != nil {
		if errors.Is(err, errRepoBuildConfigNotFound) {
			return nil
		}
		return fmt.Errorf("failed to resolve repo build config: %w", err)
	}

	overrideManifest, err := parseRepoBuildManifest(raw)
	if err != nil {
		return fmt.Errorf("invalid repo build config %s: %w", configPath, err)
	}
	if err := validateRepoManagedManifestPolicy(overrideManifest); err != nil {
		return fmt.Errorf("repo build config policy violation: %w", err)
	}

	merged := mergeRepoBuildManifest(manifest, overrideManifest)
	if merged.Metadata == nil {
		merged.Metadata = map[string]interface{}{}
	}
	merged.Metadata["repo_config_path"] = configPath
	merged.Metadata["repo_config_ref"] = ref
	merged.Metadata["repo_config_applied"] = true
	delete(merged.Metadata, "repo_config_error")
	delete(merged.Metadata, "repo_config_error_stage")
	delete(merged.Metadata, "repo_config_error_at")

	if err := validateManifest(merged); err != nil {
		return fmt.Errorf("repo build config validation failed: %w", err)
	}
	if err := s.validateToolAvailability(ctx, b.TenantID(), merged); err != nil {
		return fmt.Errorf("repo build config violates tool availability: %w", err)
	}
	if err := s.validateBuildCapabilities(ctx, b.TenantID(), merged); err != nil {
		return fmt.Errorf("repo build config violates build capabilities: %w", err)
	}

	b.manifest = merged

	if merged.BuildConfig != nil {
		cfg := buildConfigDataFromManifest(merged, b.ID())
		if cfg != nil {
			if existing := b.Config(); existing != nil {
				if existing.ID != uuid.Nil {
					cfg.ID = existing.ID
				}
				if !existing.CreatedAt.IsZero() {
					cfg.CreatedAt = existing.CreatedAt
				}
			}
			if cfg.Validate() != nil {
				return fmt.Errorf("repo build config produced invalid build method config")
			}

			if cfg.ID != uuid.Nil {
				if err := s.repository.UpdateBuildConfig(ctx, cfg); err != nil {
					return fmt.Errorf("failed to update build config from repo file: %w", err)
				}
			} else {
				if err := s.repository.SaveBuildConfig(ctx, cfg); err != nil {
					return fmt.Errorf("failed to save build config from repo file: %w", err)
				}
			}
			b.SetConfig(cfg)
		}
	}

	return nil
}

func (s *Service) shouldFallbackToUIOnRepoConfigError(ctx context.Context, b *Build) bool {
	if b == nil {
		return false
	}
	manifest := b.Manifest()
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}

	var settings *ProjectBuildSettings
	if s.projectBuildSettingsLookup != nil {
		resolvedSettings, err := s.projectBuildSettingsLookup(ctx, b.ProjectID())
		if err != nil {
			return false
		}
		settings = resolvedSettings
	}

	mode := strings.ToLower(strings.TrimSpace(metadataString(manifest.Metadata, "build_config_mode", "buildConfigMode")))
	if mode == "" && settings != nil {
		mode = strings.ToLower(strings.TrimSpace(settings.BuildConfigMode))
	}
	if mode == "" {
		mode = "repo_managed"
	}
	if mode != "repo_managed" {
		return false
	}

	onError := strings.ToLower(strings.TrimSpace(metadataString(manifest.Metadata, "build_config_on_error", "buildConfigOnError")))
	if onError == "" && settings != nil {
		onError = strings.ToLower(strings.TrimSpace(settings.BuildConfigOnError))
	}
	if onError == "" {
		onError = "strict"
	}
	return onError == "fallback_to_ui"
}

func parseRepoBuildManifest(raw []byte) (BuildManifest, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return BuildManifest{}, errors.New("file is empty")
	}
	jsonBytes, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return BuildManifest{}, err
	}

	var envelope repoBuildConfigEnvelope
	if err := yaml.Unmarshal(jsonBytes, &envelope); err != nil {
		return BuildManifest{}, err
	}
	if envelope.Version != "" {
		v := strings.ToLower(strings.TrimSpace(envelope.Version))
		if v != "v1" && v != "1" {
			return BuildManifest{}, fmt.Errorf("unsupported version %q", envelope.Version)
		}
		if envelope.Build == nil {
			return BuildManifest{}, errors.New("versioned config requires top-level 'build' object")
		}
	}

	var manifest BuildManifest
	if envelope.Build != nil {
		b, err := yaml.Marshal(envelope.Build)
		if err != nil {
			return BuildManifest{}, err
		}
		if err := yaml.Unmarshal(b, &manifest); err != nil {
			return BuildManifest{}, err
		}
		return manifest, nil
	}

	if err := yaml.Unmarshal(jsonBytes, &manifest); err != nil {
		return BuildManifest{}, err
	}
	return manifest, nil
}

func mergeRepoBuildManifest(base, override BuildManifest) BuildManifest {
	merged := base

	if strings.TrimSpace(override.Name) != "" {
		merged.Name = strings.TrimSpace(override.Name)
	}
	if strings.TrimSpace(string(override.Type)) != "" {
		merged.Type = override.Type
	}
	if strings.TrimSpace(override.BaseImage) != "" {
		merged.BaseImage = strings.TrimSpace(override.BaseImage)
	}
	if len(override.Instructions) > 0 {
		merged.Instructions = override.Instructions
	}
	if len(override.Tags) > 0 {
		merged.Tags = override.Tags
	}
	if len(override.Environment) > 0 {
		merged.Environment = override.Environment
	}
	if override.BuildConfig != nil {
		merged.BuildConfig = override.BuildConfig
	}
	if override.Metadata != nil {
		if merged.Metadata == nil {
			merged.Metadata = map[string]interface{}{}
		}
		for k, v := range override.Metadata {
			key := strings.ToLower(strings.TrimSpace(k))
			switch key {
			case "git_url", "giturl", "repo_url", "repourl", "repository_url", "repositoryurl", "git_branch", "gitbranch", "branch", "git_commit", "gitcommit", "source_id", "sourceid", "trigger_ref", "triggerref", "trigger_type", "triggertype", "trigger_event", "triggerevent", "trigger_provider", "triggerprovider":
				continue
			default:
				merged.Metadata[k] = v
			}
		}
	}

	return merged
}

func validateRepoManagedManifestPolicy(manifest BuildManifest) error {
	if manifest.InfrastructureType != "" || manifest.InfrastructureProviderID != nil {
		return errors.New("infrastructure selection is managed outside repository config")
	}
	if manifest.BuildConfig != nil && len(manifest.BuildConfig.Secrets) > 0 {
		return errors.New("build_config.secrets is not allowed in repository config; use managed secret references")
	}
	return nil
}

func (s *Service) resolveGitAuthForManifest(ctx context.Context, projectID uuid.UUID, manifest BuildManifest) (map[string][]byte, error) {
	if s.projectSourceGitAuthLookup != nil {
		if sourceID := metadataString(manifest.Metadata, "source_id", "sourceId"); strings.TrimSpace(sourceID) != "" {
			parsedSourceID, err := uuid.Parse(strings.TrimSpace(sourceID))
			if err == nil {
				authData, lookupErr := s.projectSourceGitAuthLookup(ctx, projectID, parsedSourceID)
				if lookupErr != nil {
					return nil, fmt.Errorf("failed to resolve source repository auth: %w", lookupErr)
				}
				if len(authData) > 0 {
					return authData, nil
				}
			}
		}
	}
	if s.projectGitAuthLookup != nil {
		authData, authErr := s.projectGitAuthLookup(ctx, projectID)
		if authErr != nil {
			return nil, fmt.Errorf("failed to resolve project repository auth: %w", authErr)
		}
		if len(authData) > 0 {
			return authData, nil
		}
	}
	return nil, nil
}

func (s *Service) fetchRepositoryFile(ctx context.Context, repoURL, ref, filePath string, authData map[string][]byte) ([]byte, error) {
	if strings.TrimSpace(repoURL) == "" {
		return nil, errors.New("repository URL is required")
	}
	cleanPath := filepath.Clean(strings.TrimSpace(filePath))
	if cleanPath == "." || cleanPath == "" || strings.HasPrefix(cleanPath, "../") || filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("invalid config file path %q", filePath)
	}

	tmpDir, err := os.MkdirTemp("", "image-factory-repo-config-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	effectiveURL := repoURL
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	urlWithCreds, extraEnv, applyErr := applyRepositoryAuth(repoURL, authData)
	if applyErr != nil {
		return nil, applyErr
	}
	if urlWithCreds != "" {
		effectiveURL = urlWithCreds
	}
	if len(extraEnv) > 0 {
		env = append(env, extraEnv...)
	}

	if out, err := runGit(ctx, tmpDir, env, "init"); err != nil {
		return nil, fmt.Errorf("git init failed: %s", strings.TrimSpace(out))
	}
	if out, err := runGit(ctx, tmpDir, env, "remote", "add", "origin", effectiveURL); err != nil {
		return nil, fmt.Errorf("git remote add failed: %s", strings.TrimSpace(out))
	}
	if out, err := runGit(ctx, tmpDir, env, "fetch", "--depth", "1", "origin", ref); err != nil {
		trimmedRef := strings.TrimPrefix(ref, "refs/heads/")
		if trimmedRef == ref {
			return nil, fmt.Errorf("git fetch failed: %s", strings.TrimSpace(out))
		}
		if retryOut, retryErr := runGit(ctx, tmpDir, env, "fetch", "--depth", "1", "origin", trimmedRef); retryErr != nil {
			return nil, fmt.Errorf("git fetch failed: %s", strings.TrimSpace(retryOut))
		}
	}

	showArg := fmt.Sprintf("FETCH_HEAD:%s", filepath.ToSlash(cleanPath))
	content, err := runGit(ctx, tmpDir, env, "show", showArg)
	if err != nil {
		low := strings.ToLower(content)
		if strings.Contains(low, "path") && strings.Contains(low, "does not exist") {
			return nil, errRepoBuildConfigNotFound
		}
		return nil, fmt.Errorf("git show failed: %s", strings.TrimSpace(content))
	}
	return []byte(content), nil
}

func applyRepositoryAuth(repoURL string, authData map[string][]byte) (string, []string, error) {
	if len(authData) == 0 {
		return repoURL, nil, nil
	}
	authType := strings.ToLower(strings.TrimSpace(string(authData["auth_type"])))
	switch authType {
	case "token", "oauth":
		token := strings.TrimSpace(string(authData["token"]))
		if token == "" {
			return "", nil, errors.New("repository auth token is empty")
		}
		username := strings.TrimSpace(string(authData["username"]))
		if username == "" {
			username = "token"
		}
		withCreds, err := applyHTTPBasicToURL(repoURL, username, token)
		if err != nil {
			return "", nil, err
		}
		return withCreds, nil, nil
	case "basic":
		username := strings.TrimSpace(string(authData["username"]))
		password := strings.TrimSpace(string(authData["password"]))
		if username == "" || password == "" {
			return "", nil, errors.New("repository basic auth is missing username or password")
		}
		withCreds, err := applyHTTPBasicToURL(repoURL, username, password)
		if err != nil {
			return "", nil, err
		}
		return withCreds, nil, nil
	case "ssh":
		privateKey := authData["ssh-privatekey"]
		if len(privateKey) == 0 {
			return "", nil, errors.New("repository SSH auth is missing private key")
		}
		keyFile, err := os.CreateTemp("", "repo-auth-ssh-key-*")
		if err != nil {
			return "", nil, err
		}
		if _, err := keyFile.Write(privateKey); err != nil {
			keyFile.Close()
			os.Remove(keyFile.Name())
			return "", nil, err
		}
		if err := keyFile.Close(); err != nil {
			os.Remove(keyFile.Name())
			return "", nil, err
		}
		if err := os.Chmod(keyFile.Name(), 0600); err != nil {
			os.Remove(keyFile.Name())
			return "", nil, err
		}
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile.Name())
		return repoURL, []string{"GIT_SSH_COMMAND=" + sshCmd}, nil
	default:
		return repoURL, nil, nil
	}
}

func applyHTTPBasicToURL(repoURL, username, password string) (string, error) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("repository URL must be http(s) for token/basic auth")
	}
	parsed.User = url.UserPassword(username, password)
	return parsed.String(), nil
}

func runGit(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func buildConfigDataFromManifest(manifest BuildManifest, buildID uuid.UUID) *BuildConfigData {
	if manifest.BuildConfig == nil {
		return nil
	}

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
	case BuildTypeContainer:
		config.Dockerfile = manifest.BuildConfig.Dockerfile
		config.BuildContext = manifest.BuildConfig.BuildContext
		config.TargetStage = manifest.BuildConfig.Target
		config.Metadata = map[string]interface{}{
			"registry_repo":    manifest.BuildConfig.RegistryRepo,
			"registry_auth_id": uuidString(manifest.BuildConfig.RegistryAuthID),
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
		config.Metadata = packerMetadataFromBuildConfig(manifest.BuildConfig)
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

	if gitURL := metadataString(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl"); gitURL != "" {
		if config.Metadata == nil {
			config.Metadata = map[string]interface{}{}
		}
		config.Metadata["git_url"] = gitURL
	}
	if gitBranch := metadataString(manifest.Metadata, "git_branch", "gitBranch", "branch"); gitBranch != "" {
		if config.Metadata == nil {
			config.Metadata = map[string]interface{}{}
		}
		config.Metadata["git_branch"] = gitBranch
	}

	return config
}
