package build

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// DefaultConfigResolver resolves configuration references and variables
type DefaultConfigResolver struct {
	secretResolver SecretResolver
}

// SecretResolver defines interface for resolving secrets
type SecretResolver interface {
	ResolveSecret(key string) (string, error)
}

// NewDefaultConfigResolver creates a new config resolver
func NewDefaultConfigResolver(secretResolver SecretResolver) *DefaultConfigResolver {
	return &DefaultConfigResolver{
		secretResolver: secretResolver,
	}
}

// ============================================================================
// Variable Resolution
// ============================================================================

// ResolveVariables resolves template variables in configuration
func (r *DefaultConfigResolver) ResolveVariables(config BuildMethodConfig, vars map[string]string) (map[string]interface{}, error) {
	if config == nil {
		return nil, errors.New("configuration cannot be nil")
	}

	switch c := config.(type) {
	case *PackerConfig:
		return r.resolvePackerVariables(c, vars)
	case *BuildxConfig:
		return r.resolveBuildxVariables(c, vars)
	case *KanikoConfig:
		return r.resolveKanikoVariables(c, vars)
	default:
		return nil, fmt.Errorf("unsupported configuration type: %T", config)
	}
}

// resolvePackerVariables resolves Packer-specific variables
func (r *DefaultConfigResolver) resolvePackerVariables(config *PackerConfig, vars map[string]string) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	// Start with existing template variables
	for k, v := range config.Variables() {
		resolved[k] = v
	}

	// Merge and resolve with provided variables
	for key, value := range vars {
		resolved[key] = r.interpolateString(value)
	}

	// Resolve build variables
	for key, value := range config.BuildVars() {
		resolved[fmt.Sprintf("var.%s", key)] = r.interpolateString(value)
	}

	return resolved, nil
}

// resolveBuildxVariables resolves Buildx build arguments
func (r *DefaultConfigResolver) resolveBuildxVariables(config *BuildxConfig, vars map[string]string) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	// Buildx build args
	for k, v := range config.BuildArgs() {
		resolved[fmt.Sprintf("arg.%s", k)] = r.interpolateString(v)
	}

	// Merge with provided variables
	for key, value := range vars {
		resolved[key] = r.interpolateString(value)
	}

	return resolved, nil
}

// resolveKanikoVariables resolves Kaniko build arguments
func (r *DefaultConfigResolver) resolveKanikoVariables(config *KanikoConfig, vars map[string]string) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	// Kaniko build args
	for k, v := range config.BuildArgs() {
		resolved[fmt.Sprintf("arg.%s", k)] = r.interpolateString(v)
	}

	// Merge with provided variables
	for key, value := range vars {
		resolved[key] = r.interpolateString(value)
	}

	return resolved, nil
}

// interpolateString performs variable interpolation using ${VAR} syntax
func (r *DefaultConfigResolver) interpolateString(s string) string {
	pattern := regexp.MustCompile(`\$\{([^}]+)\}`)
	return pattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR}
		varName := match[2 : len(match)-1]

		// Try to resolve the variable
		if value, err := r.resolveVariable(varName); err == nil {
			return value
		}

		// Return original if not found
		return match
	})
}

// resolveVariable resolves a single variable reference
func (r *DefaultConfigResolver) resolveVariable(varName string) (string, error) {
	// Support common prefixes
	if strings.HasPrefix(varName, "secret.") {
		secretKey := strings.TrimPrefix(varName, "secret.")
		if r.secretResolver != nil {
			return r.secretResolver.ResolveSecret(secretKey)
		}
		return "", fmt.Errorf("secret resolver not configured for: %s", varName)
	}

	// For now, return error for unresolved variables
	return "", fmt.Errorf("unresolved variable: %s", varName)
}

// ============================================================================
// Build Context Resolution
// ============================================================================

// ResolveBuildContext resolves and validates build context paths
func (r *DefaultConfigResolver) ResolveBuildContext(buildContext string) (string, error) {
	if buildContext == "" {
		return "", errors.New("build context cannot be empty")
	}

	// Current working directory
	if buildContext == "." {
		return ".", nil
	}

	// Absolute paths
	if strings.HasPrefix(buildContext, "/") {
		return buildContext, nil
	}

	// Git URLs
	if strings.HasPrefix(buildContext, "git://") || strings.HasPrefix(buildContext, "https://") ||
		strings.HasPrefix(buildContext, "http://") || strings.HasSuffix(buildContext, ".git") {
		return buildContext, nil
	}

	// S3 paths (for Kaniko)
	if strings.HasPrefix(buildContext, "s3://") {
		return buildContext, nil
	}

	// Relative paths
	if !strings.Contains(buildContext, "://") {
		return buildContext, nil
	}

	return "", fmt.Errorf("invalid build context format: %s", buildContext)
}

// ============================================================================
// Secret Resolution
// ============================================================================

// ResolveSecrets resolves secret references to actual values
func (r *DefaultConfigResolver) ResolveSecrets(secretRefs map[string]string) (map[string]string, error) {
	if secretRefs == nil {
		return make(map[string]string), nil
	}

	resolved := make(map[string]string)

	for key, ref := range secretRefs {
		if key == "" {
			return nil, errors.New("secret key cannot be empty")
		}

		value, err := r.resolveSecretReference(ref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve secret '%s': %w", key, err)
		}

		resolved[key] = value
	}

	return resolved, nil
}

// resolveSecretReference resolves a single secret reference
func (r *DefaultConfigResolver) resolveSecretReference(ref string) (string, error) {
	if ref == "" {
		return "", errors.New("secret reference cannot be empty")
	}

	// Support explicit secret:// protocol
	if strings.HasPrefix(ref, "secret://") {
		secretKey := strings.TrimPrefix(ref, "secret://")
		if r.secretResolver != nil {
			return r.secretResolver.ResolveSecret(secretKey)
		}
		return "", errors.New("secret resolver not configured")
	}

	// Assume direct secret key
	if r.secretResolver != nil {
		return r.secretResolver.ResolveSecret(ref)
	}

	return "", errors.New("secret resolver not configured")
}

// ============================================================================
// Registry Resolution
// ============================================================================

// ResolveRegistry resolves registry references and validates connectivity
type RegistryResolver interface {
	ValidateRegistry(registry string) error
	GetRegistryCredentials(registry string) (username, password string, err error)
}

// ResolveRegistryRepo validates and resolves registry repository references
func (r *DefaultConfigResolver) ResolveRegistryRepo(registryRepo string, registryResolver RegistryResolver) error {
	if registryRepo == "" {
		return errors.New("registry repo cannot be empty")
	}

	// Extract registry from repository reference
	parts := strings.Split(registryRepo, "/")
	if len(parts) == 0 {
		return errors.New("invalid registry repo format")
	}

	registry := parts[0]

	// Validate registry if resolver provided
	if registryResolver != nil {
		if err := registryResolver.ValidateRegistry(registry); err != nil {
			return fmt.Errorf("invalid registry '%s': %w", registry, err)
		}
	}

	return nil
}

// ============================================================================
// Dockerfile Resolution
// ============================================================================

// ResolveDockerfile validates and resolves Dockerfile content
func (r *DefaultConfigResolver) ResolveDockerfile(dockerfile string) (string, error) {
	if dockerfile == "" {
		return "", errors.New("dockerfile cannot be empty")
	}

	// Validate basic structure
	lines := strings.Split(dockerfile, "\n")
	hasFrom := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "FROM") {
			hasFrom = true
			break
		}
	}

	if !hasFrom {
		return "", errors.New("dockerfile must contain a FROM instruction")
	}

	return dockerfile, nil
}

// ============================================================================
// Preset Resolution
// ============================================================================

// PresetConfig represents a preset configuration template
type PresetConfig struct {
	Name        string                 `json:"name"`
	Method      BuildMethod            `json:"method"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GetDefaultPresets returns default preset configurations
func (r *DefaultConfigResolver) GetDefaultPresets() map[BuildMethod][]PresetConfig {
	return map[BuildMethod][]PresetConfig{
		BuildMethodPacker: {
			{
				Name:        "basic",
				Method:      BuildMethodPacker,
				Description: "Basic Packer AMI builder for AWS",
				Parameters: map[string]interface{}{
					"template": `{
  "builders": [{
    "type": "amazon-ebs",
    "region": "us-east-1",
    "source_ami_filter": {
      "filters": {
        "name": "amzn2-ami-hvm-*",
        "root-device-type": "ebs"
      },
      "owners": ["amazon"],
      "most_recent": true
    },
    "instance_type": "t2.small",
    "ssh_username": "ec2-user"
  }],
  "provisioners": [{
    "type": "shell",
    "inline": ["echo 'Building AMI'"]
  }]
}`,
				},
			},
		},
		BuildMethodBuildx: {
			{
				Name:        "amd64-arm64",
				Method:      BuildMethodBuildx,
				Description: "Multi-platform build for AMD64 and ARM64",
				Parameters: map[string]interface{}{
					"platforms": []string{"linux/amd64", "linux/arm64"},
					"cache_mode": "max",
				},
			},
		},
		BuildMethodKaniko: {
			{
				Name:        "ecr-cache",
				Method:      BuildMethodKaniko,
				Description: "Kaniko build with ECR caching",
				Parameters: map[string]interface{}{
					"skip_unused_stages": true,
				},
			},
		},
	}
}
