package build

import (
	"errors"
	"fmt"
	"strings"
)

// DefaultConfigValidator provides built-in validation for all build method configs
type DefaultConfigValidator struct{}

// NewDefaultConfigValidator creates a new default validator
func NewDefaultConfigValidator() *DefaultConfigValidator {
	return &DefaultConfigValidator{}
}

// Validate validates any build method configuration
func (v *DefaultConfigValidator) Validate(config BuildMethodConfig) error {
	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	// Always call the config's own Validate method first
	if err := config.Validate(); err != nil {
		return err
	}

	// Then apply method-specific validation
	switch c := config.(type) {
	case *PackerConfig:
		return v.ValidatePacker(c)
	case *BuildxConfig:
		return v.ValidateBuildx(c)
	case *KanikoConfig:
		return v.ValidateKaniko(c)
	default:
		return fmt.Errorf("unknown configuration type: %T", config)
	}
}

// ============================================================================
// Packer Validation
// ============================================================================

// ValidatePacker validates Packer-specific configuration rules
func (v *DefaultConfigValidator) ValidatePacker(config *PackerConfig) error {
	if config == nil {
		return errors.New("packer configuration cannot be nil")
	}

	// Template validation
	if err := validatePackerTemplate(config.Template()); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Variables validation
	if err := validatePackerVariables(config.Variables()); err != nil {
		return fmt.Errorf("variables validation failed: %w", err)
	}

	// Build vars validation
	if err := validatePackerBuildVars(config.BuildVars()); err != nil {
		return fmt.Errorf("build vars validation failed: %w", err)
	}

	// OnError validation
	validOnErrors := map[string]bool{"ask": true, "cleanup": true, "abort": true}
	if !validOnErrors[config.OnError()] {
		return fmt.Errorf("invalid on_error value: %s", config.OnError())
	}

	return nil
}

// validatePackerTemplate validates Packer template format
func validatePackerTemplate(template string) error {
	if template == "" {
		return errors.New("template cannot be empty")
	}

	// Template should contain basic Packer JSON structure
	template = strings.TrimSpace(template)
	if !strings.HasPrefix(template, "{") || !strings.HasSuffix(template, "}") {
		return errors.New("template must be valid JSON (should start with { and end with })")
	}

	// Check for required top-level fields for a basic Packer config
	if !strings.Contains(template, "\"builders\"") && !strings.Contains(template, `"builders"`) {
		return errors.New("template must contain a 'builders' section")
	}

	return nil
}

// validatePackerVariables validates variable definitions
func validatePackerVariables(vars map[string]interface{}) error {
	for key, value := range vars {
		if key == "" {
			return errors.New("variable key cannot be empty")
		}
		if value == nil {
			return fmt.Errorf("variable '%s' has nil value", key)
		}
	}
	return nil
}

// validatePackerBuildVars validates build-time variables
func validatePackerBuildVars(vars map[string]string) error {
	for key, value := range vars {
		if key == "" {
			return errors.New("build var key cannot be empty")
		}
		if value == "" {
			return fmt.Errorf("build var '%s' cannot be empty", key)
		}
	}
	return nil
}

// ============================================================================
// Buildx Validation
// ============================================================================

// ValidateBuildx validates Docker Buildx-specific configuration rules
func (v *DefaultConfigValidator) ValidateBuildx(config *BuildxConfig) error {
	if config == nil {
		return errors.New("buildx configuration cannot be nil")
	}

	// Dockerfile validation
	if err := validateBuildxDockerfile(config.Dockerfile()); err != nil {
		return fmt.Errorf("dockerfile validation failed: %w", err)
	}

	// Build context validation
	if err := validateBuildxBuildContext(config.BuildContext()); err != nil {
		return fmt.Errorf("build context validation failed: %w", err)
	}

	// Platforms validation
	if err := validateBuildxPlatforms(config.Platforms()); err != nil {
		return fmt.Errorf("platforms validation failed: %w", err)
	}

	// Build args validation
	if err := validateBuildxBuildArgs(config.BuildArgs()); err != nil {
		return fmt.Errorf("build args validation failed: %w", err)
	}

	// Secrets validation
	if err := validateBuildxSecrets(config.Secrets()); err != nil {
		return fmt.Errorf("secrets validation failed: %w", err)
	}

	// Cache validation
	if config.CacheFrom() != "" && !isValidCacheReference(config.CacheFrom()) {
		return fmt.Errorf("invalid cache_from reference: %s", config.CacheFrom())
	}

	if config.CacheTo() != "" && !isValidCacheReference(config.CacheTo()) {
		return fmt.Errorf("invalid cache_to reference: %s", config.CacheTo())
	}

	return nil
}

// validateBuildxDockerfile validates Dockerfile path/content
func validateBuildxDockerfile(dockerfile string) error {
	if dockerfile == "" {
		return errors.New("dockerfile cannot be empty")
	}
	if !strings.Contains(dockerfile, "Dockerfile") && !strings.Contains(dockerfile, ".dockerfile") {
		return errors.New("dockerfile path should contain 'Dockerfile' or '.dockerfile'")
	}
	return nil
}

// validateBuildxBuildContext validates build context path
func validateBuildxBuildContext(context string) error {
	if context == "" {
		return errors.New("build context cannot be empty")
	}
	if context == "." || context == ".." || strings.HasPrefix(context, "/") || strings.HasPrefix(context, "http") {
		return nil
	}
	return nil // Allow relative and absolute paths
}

// validateBuildxPlatforms validates platform specifications
func validateBuildxPlatforms(platforms []string) error {
	if len(platforms) == 0 {
		return errors.New("at least one platform must be specified")
	}

	validPlatforms := map[string]bool{
		"linux/amd64":   true,
		"linux/arm64":   true,
		"linux/arm/v7":  true,
		"linux/386":     true,
		"linux/ppc64le": true,
		"linux/s390x":   true,
		"linux/riscv64": true,
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"windows/amd64": true,
	}

	for _, platform := range platforms {
		if !validPlatforms[platform] {
			// Don't fail - allow custom/new platforms
			if !strings.Contains(platform, "/") {
				return fmt.Errorf("invalid platform format: %s (must be os/arch)", platform)
			}
		}
	}
	return nil
}

// validateBuildxBuildArgs validates build arguments
func validateBuildxBuildArgs(args map[string]string) error {
	for key := range args {
		if key == "" {
			return errors.New("build arg key cannot be empty")
		}
	}
	return nil
}

// validateBuildxSecrets validates secret definitions
func validateBuildxSecrets(secrets map[string]string) error {
	for key := range secrets {
		if key == "" {
			return errors.New("secret key cannot be empty")
		}
	}
	return nil
}

// isValidCacheReference validates cache reference format
func isValidCacheReference(ref string) bool {
	// type=registry,ref=...
	// type=local,dest=...
	// type=s3,endpoint=...,bucket=...
	// etc.
	if strings.HasPrefix(ref, "type=") {
		return true
	}
	// Allow image references too
	if strings.Contains(ref, ":") || strings.Contains(ref, "/") {
		return true
	}
	return false
}

// ============================================================================
// Kaniko Validation
// ============================================================================

// ValidateKaniko validates Kaniko-specific configuration rules
func (v *DefaultConfigValidator) ValidateKaniko(config *KanikoConfig) error {
	if config == nil {
		return errors.New("kaniko configuration cannot be nil")
	}

	// Dockerfile validation
	if err := validateKanikoDockerfile(config.Dockerfile()); err != nil {
		return fmt.Errorf("dockerfile validation failed: %w", err)
	}

	// Build context validation
	if err := validateKanikoBuildContext(config.BuildContext()); err != nil {
		return fmt.Errorf("build context validation failed: %w", err)
	}

	// Registry repo validation
	if err := validateKanikoRegistryRepo(config.RegistryRepo()); err != nil {
		return fmt.Errorf("registry repo validation failed: %w", err)
	}

	// Cache repo validation (optional but should be valid if provided)
	if config.CacheRepo() != "" {
		if err := validateKanikoRegistryRepo(config.CacheRepo()); err != nil {
			return fmt.Errorf("cache repo validation failed: %w", err)
		}
	}

	// Build args validation
	if err := validateKanikoBuildArgs(config.BuildArgs()); err != nil {
		return fmt.Errorf("build args validation failed: %w", err)
	}

	return nil
}

// validateKanikoDockerfile validates Kaniko Dockerfile
func validateKanikoDockerfile(dockerfile string) error {
	if dockerfile == "" {
		return errors.New("dockerfile cannot be empty")
	}
	if !strings.Contains(dockerfile, "Dockerfile") && !strings.Contains(dockerfile, ".dockerfile") {
		return errors.New("dockerfile path should contain 'Dockerfile' or '.dockerfile'")
	}
	return nil
}

// validateKanikoBuildContext validates Kaniko build context
func validateKanikoBuildContext(context string) error {
	if context == "" {
		return errors.New("build context cannot be empty")
	}
	// Kaniko supports git, S3, and local paths
	if strings.HasPrefix(context, "git://") || strings.HasPrefix(context, "s3://") ||
		context == "." || strings.HasPrefix(context, "/") {
		return nil
	}
	return nil // Allow other formats
}

// validateKanikoRegistryRepo validates Docker registry repository URL
func validateKanikoRegistryRepo(repo string) error {
	if repo == "" {
		return errors.New("registry repo cannot be empty")
	}

	// Must contain at least a registry or image name
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return errors.New("invalid registry repo format")
	}

	// Check if looks like a valid registry reference
	if !strings.Contains(repo, ".") && !strings.Contains(repo, ":") && len(parts) < 2 {
		return fmt.Errorf("invalid registry repo: %s (should be registry/image or similar)", repo)
	}

	return nil
}

// validateKanikoBuildArgs validates Kaniko build arguments
func validateKanikoBuildArgs(args map[string]string) error {
	for key := range args {
		if key == "" {
			return errors.New("build arg key cannot be empty")
		}
	}
	return nil
}

// ============================================================================
// Common Validators
// ============================================================================

// ValidateTemplate validates a raw template string
func (v *DefaultConfigValidator) ValidateTemplate(method BuildMethod, templateStr string) error {
	switch method {
	case BuildMethodPacker:
		return validatePackerTemplate(templateStr)
	case BuildMethodBuildx, BuildMethodDocker:
		// For Buildx, template is dockerfile content
		if templateStr == "" {
			return errors.New("dockerfile content cannot be empty")
		}
		if !strings.Contains(strings.ToLower(templateStr), "from") {
			return errors.New("dockerfile must contain a FROM instruction")
		}
		return nil
	case BuildMethodKaniko:
		// For Kaniko, same as Buildx
		if templateStr == "" {
			return errors.New("dockerfile content cannot be empty")
		}
		if !strings.Contains(strings.ToLower(templateStr), "from") {
			return errors.New("dockerfile must contain a FROM instruction")
		}
		return nil
	case BuildMethodPaketo:
		// Paketo builds are builder/buildpack driven and don't require templates.
		return nil
	case BuildMethodNix:
		// Nix builds may be expression-based or flake-URI based.
		// Empty template is valid when using flake_uri.
		return nil
	default:
		return fmt.Errorf("unsupported build method for template validation: %s", method)
	}
}
