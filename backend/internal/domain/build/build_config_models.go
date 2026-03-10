package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildConfig represents the multi-tool build configuration
type BuildConfig struct {
	BuildType         BuildType         // "packer", "paketo", "kaniko", "buildx"
	SBOMTool          SBOMTool          // "syft", "grype", "trivy"
	ScanTool          ScanTool          // "trivy", "clair", "grype", "snyk"
	RegistryType      RegistryType      // "s3", "harbor", "quay", "artifactory"
	SecretManagerType SecretManagerType // "vault", "aws_secretsmanager", "azure_keyvault", "gcp_secretmanager"
	// Packer-specific fields
	PackerTemplate string // Packer HCL template (for packer builds)
	// Buildpack-specific fields
	PaketoConfig *PaketoConfig // Paketo buildpack configuration (for paketo builds)
	// Dockerfile-based fields
	Dockerfile       string            // Dockerfile content or path (for kaniko/buildx)
	BuildContext     string            // Build context directory (for kaniko/buildx)
	BuildArgs        map[string]string // Build arguments (for kaniko/buildx)
	Target           string            // Target build stage
	Cache            bool              // Enable build cache
	CacheRepo        string            // Cache repository for layers
	RegistryRepo     string            // Target image repository (required for kaniko)
	RegistryAuthID   *uuid.UUID        // Selected registry authentication reference
	SkipUnusedStages bool              // Skip unused stages during Kaniko builds
	// Nix-specific fields
	NixExpression string            // Nix expression content
	FlakeURI      string            // Flake URI
	Attributes    []string          // Nix attributes to build
	Outputs       map[string]string // Named output mapping
	CacheDir      string            // Nix cache directory
	Pure          bool              // Pure evaluation mode
	ShowTrace     bool              // Show evaluation traces
	// Buildx-specific fields
	Platforms []string          // Target platforms (e.g., "linux/amd64,linux/arm64")
	CacheTo   string            // Cache export location
	CacheFrom []string          // Cache import locations
	Secrets   map[string]string // Build secrets
	// Common fields
	Variables      map[string]interface{} // Template/build variables
	Builders       []PackerBuilder        // Packer builders (packer only)
	Provisioners   []VMProvisioner        // Packer provisioners (packer only)
	PostProcessors []VMPostProcessor      // Packer post-processors (packer only)
}

// UnmarshalJSON allows dockerfile to be provided as a string or a structured object.
func (c *BuildConfig) UnmarshalJSON(data []byte) error {
	type snakePayload struct {
		BuildType         BuildType              `json:"build_type"`
		SBOMTool          SBOMTool               `json:"sbom_tool"`
		ScanTool          ScanTool               `json:"scan_tool"`
		RegistryType      RegistryType           `json:"registry_type"`
		SecretManagerType SecretManagerType      `json:"secret_manager_type"`
		PackerTemplate    string                 `json:"packer_template"`
		PaketoConfig      *PaketoConfig          `json:"paketo_config"`
		Dockerfile        json.RawMessage        `json:"dockerfile"`
		BuildContext      string                 `json:"build_context"`
		BuildArgs         map[string]string      `json:"build_args"`
		Target            string                 `json:"target"`
		Cache             bool                   `json:"cache"`
		CacheRepo         string                 `json:"cache_repo"`
		RegistryRepo      string                 `json:"registry_repo"`
		RegistryAuthID    string                 `json:"registry_auth_id"`
		SkipUnusedStages  bool                   `json:"skip_unused_stages"`
		NixExpression     string                 `json:"nix_expression"`
		FlakeURI          string                 `json:"flake_uri"`
		Attributes        []string               `json:"attributes"`
		Outputs           map[string]string      `json:"outputs"`
		CacheDir          string                 `json:"cache_dir"`
		Pure              bool                   `json:"pure"`
		ShowTrace         bool                   `json:"show_trace"`
		Platforms         []string               `json:"platforms"`
		CacheTo           string                 `json:"cache_to"`
		CacheFrom         []string               `json:"cache_from"`
		Secrets           map[string]string      `json:"secrets"`
		Variables         map[string]interface{} `json:"variables"`
		Builders          []PackerBuilder        `json:"builders"`
		Provisioners      []VMProvisioner        `json:"provisioners"`
		PostProcessors    []VMPostProcessor      `json:"post_processors"`
	}

	type camelPayload struct {
		BuildType         BuildType              `json:"buildType"`
		SBOMTool          SBOMTool               `json:"sbomTool"`
		ScanTool          ScanTool               `json:"scanTool"`
		RegistryType      RegistryType           `json:"registryType"`
		SecretManagerType SecretManagerType      `json:"secretManagerType"`
		PackerTemplate    string                 `json:"packerTemplate"`
		PaketoConfig      *PaketoConfig          `json:"paketoConfig"`
		Dockerfile        json.RawMessage        `json:"dockerfile"`
		BuildContext      string                 `json:"buildContext"`
		BuildArgs         map[string]string      `json:"buildArgs"`
		Target            string                 `json:"target"`
		Cache             bool                   `json:"cache"`
		CacheRepo         string                 `json:"cacheRepo"`
		RegistryRepo      string                 `json:"registryRepo"`
		RegistryAuthID    string                 `json:"registryAuthId"`
		SkipUnusedStages  bool                   `json:"skipUnusedStages"`
		NixExpression     string                 `json:"nixExpression"`
		FlakeURI          string                 `json:"flakeUri"`
		Attributes        []string               `json:"attributes"`
		Outputs           map[string]string      `json:"outputs"`
		CacheDir          string                 `json:"cacheDir"`
		Pure              bool                   `json:"pure"`
		ShowTrace         bool                   `json:"showTrace"`
		Platforms         []string               `json:"platforms"`
		CacheTo           string                 `json:"cacheTo"`
		CacheFrom         []string               `json:"cacheFrom"`
		Secrets           map[string]string      `json:"secrets"`
		Variables         map[string]interface{} `json:"variables"`
		Builders          []PackerBuilder        `json:"builders"`
		Provisioners      []VMProvisioner        `json:"provisioners"`
		PostProcessors    []VMPostProcessor      `json:"postProcessors"`
	}

	parseDockerfile := func(raw json.RawMessage) (string, error) {
		if len(raw) == 0 || string(raw) == "null" {
			return "", nil
		}
		var str string
		if err := json.Unmarshal(raw, &str); err == nil {
			return str, nil
		}
		var obj struct {
			Source   string `json:"source"`
			Path     string `json:"path"`
			Content  string `json:"content"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", err
		}
		if obj.Source == "path" {
			return obj.Path, nil
		}
		return obj.Content, nil
	}

	var snake snakePayload
	if err := json.Unmarshal(data, &snake); err != nil {
		return err
	}
	var camel camelPayload
	if err := json.Unmarshal(data, &camel); err != nil {
		return err
	}

	payload := snake
	// Fill missing fields from camelCase payload
	if payload.BuildType == "" {
		payload.BuildType = camel.BuildType
	}
	if payload.SBOMTool == "" {
		payload.SBOMTool = camel.SBOMTool
	}
	if payload.ScanTool == "" {
		payload.ScanTool = camel.ScanTool
	}
	if payload.RegistryType == "" {
		payload.RegistryType = camel.RegistryType
	}
	if payload.SecretManagerType == "" {
		payload.SecretManagerType = camel.SecretManagerType
	}
	if payload.PackerTemplate == "" {
		payload.PackerTemplate = camel.PackerTemplate
	}
	if payload.PaketoConfig == nil {
		payload.PaketoConfig = camel.PaketoConfig
	}
	if payload.BuildContext == "" {
		payload.BuildContext = camel.BuildContext
	}
	if payload.BuildArgs == nil {
		payload.BuildArgs = camel.BuildArgs
	}
	if payload.Target == "" {
		payload.Target = camel.Target
	}
	if payload.CacheRepo == "" {
		payload.CacheRepo = camel.CacheRepo
	}
	if payload.RegistryRepo == "" {
		payload.RegistryRepo = camel.RegistryRepo
	}
	if payload.RegistryAuthID == "" {
		payload.RegistryAuthID = camel.RegistryAuthID
	}
	if len(payload.Platforms) == 0 {
		payload.Platforms = camel.Platforms
	}
	if payload.NixExpression == "" {
		payload.NixExpression = camel.NixExpression
	}
	if payload.FlakeURI == "" {
		payload.FlakeURI = camel.FlakeURI
	}
	if len(payload.Attributes) == 0 {
		payload.Attributes = camel.Attributes
	}
	if payload.Outputs == nil {
		payload.Outputs = camel.Outputs
	}
	if payload.CacheDir == "" {
		payload.CacheDir = camel.CacheDir
	}
	payload.Pure = payload.Pure || camel.Pure
	payload.ShowTrace = payload.ShowTrace || camel.ShowTrace
	if payload.CacheTo == "" {
		payload.CacheTo = camel.CacheTo
	}
	if len(payload.CacheFrom) == 0 {
		payload.CacheFrom = camel.CacheFrom
	}
	if payload.Secrets == nil {
		payload.Secrets = camel.Secrets
	}
	if payload.Variables == nil {
		payload.Variables = camel.Variables
	}
	if payload.Builders == nil {
		payload.Builders = camel.Builders
	}
	if payload.Provisioners == nil {
		payload.Provisioners = camel.Provisioners
	}
	if payload.PostProcessors == nil {
		payload.PostProcessors = camel.PostProcessors
	}
	if len(payload.Dockerfile) == 0 {
		payload.Dockerfile = camel.Dockerfile
	}

	dockerfile, err := parseDockerfile(payload.Dockerfile)
	if err != nil {
		return err
	}

	c.BuildType = payload.BuildType
	c.SBOMTool = payload.SBOMTool
	c.ScanTool = payload.ScanTool
	c.RegistryType = payload.RegistryType
	c.SecretManagerType = payload.SecretManagerType
	c.PackerTemplate = payload.PackerTemplate
	c.PaketoConfig = payload.PaketoConfig
	c.Dockerfile = dockerfile
	c.BuildContext = payload.BuildContext
	c.BuildArgs = payload.BuildArgs
	c.Target = payload.Target
	c.Cache = payload.Cache
	c.CacheRepo = payload.CacheRepo
	c.RegistryRepo = payload.RegistryRepo
	if payload.RegistryAuthID != "" {
		parsed, err := uuid.Parse(payload.RegistryAuthID)
		if err != nil {
			return fmt.Errorf("invalid registry_auth_id: %w", err)
		}
		c.RegistryAuthID = &parsed
	}
	c.SkipUnusedStages = payload.SkipUnusedStages || camel.SkipUnusedStages
	c.Platforms = normalizeBuildxPlatforms(payload.Platforms)
	c.CacheTo = payload.CacheTo
	c.CacheFrom = payload.CacheFrom
	c.Secrets = payload.Secrets
	c.NixExpression = payload.NixExpression
	c.FlakeURI = payload.FlakeURI
	c.Attributes = payload.Attributes
	c.Outputs = payload.Outputs
	c.CacheDir = payload.CacheDir
	c.Pure = payload.Pure
	c.ShowTrace = payload.ShowTrace
	c.Variables = payload.Variables
	c.Builders = payload.Builders
	c.Provisioners = payload.Provisioners
	c.PostProcessors = payload.PostProcessors

	return nil
}

// PackerBuilder represents a Packer builder configuration
type PackerBuilder struct {
	Type   string                 `json:"type"` // amazon-ebs, azure-arm, etc.
	Config map[string]interface{} `json:"config"`
}

// PaketoConfig represents Paketo buildpack configuration
type PaketoConfig struct {
	Builder    string            `json:"builder"`    // Paketo builder image
	Buildpacks []string          `json:"buildpacks"` // Additional buildpacks
	Env        map[string]string `json:"env"`        // Environment variables
	BuildArgs  map[string]string `json:"build_args"` // Build arguments
}

// BuildConfigData represents method-specific build configuration stored in build_configs table
type BuildConfigData struct {
	ID          uuid.UUID  `json:"id"`
	BuildID     uuid.UUID  `json:"build_id"`
	BuildMethod string     `json:"build_method"` // kaniko, buildx, container, paketo, packer
	SourceID    *uuid.UUID `json:"source_id,omitempty"`
	RefPolicy   string     `json:"ref_policy,omitempty"` // source_default, fixed, event_ref
	FixedRef    string     `json:"fixed_ref,omitempty"`

	// Shared fields
	SBOMTool          SBOMTool               `json:"sbom_tool,omitempty"`
	ScanTool          ScanTool               `json:"scan_tool,omitempty"`
	RegistryType      RegistryType           `json:"registry_type,omitempty"`
	SecretManagerType SecretManagerType      `json:"secret_manager_type,omitempty"`
	BuildArgs         map[string]string      `json:"build_args,omitempty"`
	Environment       map[string]string      `json:"environment,omitempty"`
	Secrets           map[string]string      `json:"secrets,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`

	// Kaniko-specific
	Dockerfile   string `json:"dockerfile,omitempty"`
	BuildContext string `json:"build_context,omitempty"`
	CacheEnabled bool   `json:"cache_enabled,omitempty"`
	CacheRepo    string `json:"cache_repo,omitempty"`

	// Buildx-specific
	Platforms []string `json:"platforms,omitempty"`
	CacheFrom []string `json:"cache_from,omitempty"`
	CacheTo   string   `json:"cache_to,omitempty"`

	// Container (Docker)-specific
	TargetStage string `json:"target_stage,omitempty"`

	// Paketo-specific
	Builder    string   `json:"builder,omitempty"`
	Buildpacks []string `json:"buildpacks,omitempty"`

	// Packer-specific
	PackerTemplate string `json:"packer_template,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidateKanikoConfig validates Kaniko-specific configuration
func (c *BuildConfigData) ValidateKanikoConfig() error {
	if c.Dockerfile == "" {
		return errors.New("dockerfile is required for kaniko builds")
	}
	if c.BuildContext == "" {
		return errors.New("build context is required for kaniko builds")
	}
	if c.Metadata == nil {
		return errors.New("registry_repo is required for kaniko builds")
	}
	registryRepo, ok := c.Metadata["registry_repo"].(string)
	if !ok || registryRepo == "" {
		return errors.New("registry_repo is required for kaniko builds")
	}
	return nil
}

// ValidateBuildxConfig validates Buildx-specific configuration
func (c *BuildConfigData) ValidateBuildxConfig() error {
	c.Platforms = normalizeBuildxPlatforms(c.Platforms)
	if c.Dockerfile == "" {
		return errors.New("dockerfile is required for buildx builds")
	}
	if c.BuildContext == "" {
		return errors.New("build context is required for buildx builds")
	}
	if c.Metadata == nil {
		return errors.New("registry_repo is required for buildx builds")
	}
	registryRepo, ok := c.Metadata["registry_repo"].(string)
	if !ok || registryRepo == "" {
		return errors.New("registry_repo is required for buildx builds")
	}
	if len(c.Platforms) == 0 {
		return errors.New("platforms are required for buildx builds")
	}
	return nil
}

// ValidateContainerConfig validates Container (Docker)-specific configuration
func (c *BuildConfigData) ValidateContainerConfig() error {
	if c.Dockerfile == "" {
		return errors.New("dockerfile is required for container builds")
	}
	if c.Metadata == nil {
		return errors.New("registry_repo is required for container builds")
	}
	registryRepo, ok := c.Metadata["registry_repo"].(string)
	if !ok || registryRepo == "" {
		return errors.New("registry_repo is required for container builds")
	}
	return nil
}

// ValidatePaketoConfig validates Paketo-specific configuration
func (c *BuildConfigData) ValidatePaketoConfig() error {
	if c.Builder == "" {
		return errors.New("builder is required for paketo builds")
	}
	return nil
}

// ValidatePackerConfig validates Packer-specific configuration
func (c *BuildConfigData) ValidatePackerConfig() error {
	if c.PackerTemplate == "" {
		return errors.New("packer template is required for packer builds")
	}
	return nil
}

// Validate validates the build config based on build method
func (c *BuildConfigData) Validate() error {
	if c.BuildID == uuid.Nil {
		return errors.New("build ID is required")
	}

	if c.BuildMethod == "" {
		return errors.New("build method is required")
	}
	if c.RefPolicy == "" {
		c.RefPolicy = "source_default"
	}
	switch c.RefPolicy {
	case "source_default", "fixed", "event_ref":
	default:
		return fmt.Errorf("invalid ref policy: %s", c.RefPolicy)
	}
	if c.RefPolicy == "fixed" && strings.TrimSpace(c.FixedRef) == "" {
		return errors.New("fixed_ref is required when ref_policy is fixed")
	}

	// Validate method-specific configuration
	switch c.BuildMethod {
	case "kaniko":
		return c.ValidateKanikoConfig()
	case "buildx":
		return c.ValidateBuildxConfig()
	case "container":
		return c.ValidateContainerConfig()
	case "docker":
		return c.ValidateContainerConfig()
	case "paketo":
		return c.ValidatePaketoConfig()
	case "packer":
		return c.ValidatePackerConfig()
	case "nix":
		return nil
	default:
		return fmt.Errorf("invalid build method: %s", c.BuildMethod)
	}
}
