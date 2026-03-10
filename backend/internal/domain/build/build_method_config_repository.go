package build

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// BuildMethodConfigRepository handles persistence of build method configurations
type BuildMethodConfigRepository interface {
	// SavePacker saves a Packer configuration
	SavePacker(ctx context.Context, config *PackerConfig) error

	// SaveBuildx saves a Buildx configuration
	SaveBuildx(ctx context.Context, config *BuildxConfig) error

	// SaveKaniko saves a Kaniko configuration
	SaveKaniko(ctx context.Context, config *KanikoConfig) error

	// FindByBuildID retrieves the config for a specific build
	FindByBuildID(ctx context.Context, buildID uuid.UUID) (BuildMethodConfig, error)

	// FindByBuildIDAndMethod retrieves a config by build ID and method type
	FindByBuildIDAndMethod(ctx context.Context, buildID uuid.UUID, method BuildMethod) (BuildMethodConfig, error)

	// DeleteByBuildID deletes the config for a build
	DeleteByBuildID(ctx context.Context, buildID uuid.UUID) error

	// ListByMethod lists all configs of a specific build method
	ListByMethod(ctx context.Context, projectID uuid.UUID, method BuildMethod) ([]BuildMethodConfig, error)
}

// ConfigValidator validates build method configurations
type ConfigValidator interface {
	// Validate validates a configuration
	Validate(config BuildMethodConfig) error

	// ValidatePacker validates Packer-specific rules
	ValidatePacker(config *PackerConfig) error

	// ValidateBuildx validates Buildx-specific rules
	ValidateBuildx(config *BuildxConfig) error

	// ValidateKaniko validates Kaniko-specific rules
	ValidateKaniko(config *KanikoConfig) error
}

// ConfigResolver resolves configurations from different sources
type ConfigResolver interface {
	// ResolveVariables resolves template variables in configuration
	ResolveVariables(config BuildMethodConfig, vars map[string]string) (map[string]interface{}, error)

	// ResolveBuildContext resolves build context paths
	ResolveBuildContext(buildContext string) (string, error)

	// ResolveSecrets resolves secret references
	ResolveSecrets(secretRefs map[string]string) (map[string]string, error)

	// GetDefaultPresets returns default preset configurations
	GetDefaultPresets() map[BuildMethod][]PresetConfig
}

// ErrInvalidBuildMethodConfig errors for build method config operations
var (
	ErrConfigNotFound      = errors.New("build method configuration not found")
	ErrInvalidConfigMethod = errors.New("invalid build method for configuration")
	ErrConfigValidation    = errors.New("configuration validation failed")
	ErrConfigConflict      = errors.New("configuration conflict detected")
)
