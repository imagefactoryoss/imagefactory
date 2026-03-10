package build

import "errors"

// Domain errors
var (
	ErrBuildNotFound               = errors.New("build not found")
	ErrInvalidManifest             = errors.New("invalid build manifest")
	ErrBuildAlreadyStarted         = errors.New("build already started")
	ErrCannotCancelBuild           = errors.New("cannot cancel build in current state")
	ErrConfigVersionNotFound       = errors.New("config version not found")
	ErrNoVersionsFound             = errors.New("no versions found for this build")
	ErrInvalidConfigSnapshot       = errors.New("invalid config snapshot")
	ErrInvalidVersionDiff          = errors.New("invalid version diff")
	ErrVersionsFromDifferentBuilds = errors.New("versions must be from the same build")
	ErrInvalidBuildID              = errors.New("invalid build ID")
	ErrInvalidConfigID             = errors.New("invalid config ID")
	ErrInvalidBuildMethod          = errors.New("invalid build method")
	ErrBuildCapabilityNotEntitled  = errors.New("build capability not entitled")
	ErrDiffNotFound                = errors.New("diff not found")
	ErrTemplateNotFound            = errors.New("template not found")
	ErrTemplateNameRequired        = errors.New("template name is required")
	ErrDuplicateTemplateName       = errors.New("template name already exists in this project")
	ErrUnauthorizedTemplate        = errors.New("unauthorized to access this template")
	ErrShareNotFound               = errors.New("share not found")
	ErrInvalidTemplateMethod       = errors.New("invalid template method")
)
