package build

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// BuildMethod represents the type of build method used
type BuildMethod string

const (
	BuildMethodPacker BuildMethod = "packer"
	BuildMethodBuildx BuildMethod = "buildx"
	BuildMethodKaniko BuildMethod = "kaniko"
	BuildMethodDocker BuildMethod = "docker"
	BuildMethodPaketo BuildMethod = "paketo"
	BuildMethodNix    BuildMethod = "nix"
)

// IsValid checks if the build method is valid
func (b BuildMethod) IsValid() bool {
	switch b {
	case BuildMethodPacker, BuildMethodBuildx, BuildMethodKaniko, BuildMethodDocker, BuildMethodPaketo, BuildMethodNix:
		return true
	default:
		return false
	}
}

// BuildMethodConfig is the base interface for all build method configurations
type BuildMethodConfig interface {
	// Identification
	ID() uuid.UUID
	BuildID() uuid.UUID
	Method() BuildMethod

	// Validation
	Validate() error

	// Serialization
	ToMap() map[string]interface{}
}

// ============================================================================
// Packer Configuration
// ============================================================================

// PackerConfig represents configuration for Packer builds
type PackerConfig struct {
	id        uuid.UUID
	buildID   uuid.UUID
	template  string
	variables map[string]interface{}
	buildVars map[string]string
	onError   string // "ask", "cleanup", "abort"
	parallel  bool
}

// NewPackerConfig creates a new Packer configuration
func NewPackerConfig(buildID uuid.UUID, template string) (*PackerConfig, error) {
	if buildID == uuid.Nil {
		return nil, errors.New("build id cannot be nil")
	}
	if template == "" {
		return nil, errors.New("template cannot be empty")
	}

	return &PackerConfig{
		id:        uuid.New(),
		buildID:   buildID,
		template:  template,
		variables: make(map[string]interface{}),
		buildVars: make(map[string]string),
		onError:   "cleanup",
		parallel:  false,
	}, nil
}

// Accessors
func (p *PackerConfig) ID() uuid.UUID                     { return p.id }
func (p *PackerConfig) BuildID() uuid.UUID                { return p.buildID }
func (p *PackerConfig) Method() BuildMethod               { return BuildMethodPacker }
func (p *PackerConfig) Template() string                  { return p.template }
func (p *PackerConfig) Variables() map[string]interface{} { return p.variables }
func (p *PackerConfig) BuildVars() map[string]string      { return p.buildVars }
func (p *PackerConfig) OnError() string                   { return p.onError }
func (p *PackerConfig) Parallel() bool                    { return p.parallel }

// Setters
func (p *PackerConfig) SetTemplate(template string) error {
	if template == "" {
		return errors.New("template cannot be empty")
	}
	p.template = template
	return nil
}

func (p *PackerConfig) SetVariable(key string, value interface{}) error {
	if key == "" {
		return errors.New("variable key cannot be empty")
	}
	p.variables[key] = value
	return nil
}

func (p *PackerConfig) SetBuildVar(key, value string) error {
	if key == "" {
		return errors.New("build var key cannot be empty")
	}
	p.buildVars[key] = value
	return nil
}

func (p *PackerConfig) SetOnError(onError string) error {
	switch onError {
	case "ask", "cleanup", "abort":
		p.onError = onError
		return nil
	default:
		return errors.New("invalid on_error value: must be 'ask', 'cleanup', or 'abort'")
	}
}

func (p *PackerConfig) SetParallel(parallel bool) {
	p.parallel = parallel
}

// Validate checks if the Packer configuration is valid
func (p *PackerConfig) Validate() error {
	if p.buildID == uuid.Nil {
		return errors.New("build id cannot be nil")
	}
	if p.template == "" {
		return errors.New("template cannot be empty")
	}
	if !p.Method().IsValid() {
		return errors.New("invalid build method")
	}
	return nil
}

// ToMap converts the configuration to a map
func (p *PackerConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         p.id.String(),
		"build_id":   p.buildID.String(),
		"method":     string(p.Method()),
		"template":   p.template,
		"variables":  p.variables,
		"build_vars": p.buildVars,
		"on_error":   p.onError,
		"parallel":   p.parallel,
	}
}

// ============================================================================
// Buildx Configuration
// ============================================================================

// BuildxConfig represents configuration for Docker Buildx builds
type BuildxConfig struct {
	id           uuid.UUID
	buildID      uuid.UUID
	dockerfile   string
	buildContext string
	platforms    []string
	buildArgs    map[string]string
	secrets      map[string]string
	cacheFrom    string
	cacheTo      string
	outputs      []string
	noCache      bool
}

// NewBuildxConfig creates a new Buildx configuration
func NewBuildxConfig(buildID uuid.UUID, dockerfile, buildContext string) (*BuildxConfig, error) {
	if buildID == uuid.Nil {
		return nil, errors.New("build id cannot be nil")
	}
	if dockerfile == "" {
		return nil, errors.New("dockerfile cannot be empty")
	}
	if buildContext == "" {
		return nil, errors.New("build context cannot be empty")
	}

	return &BuildxConfig{
		id:           uuid.New(),
		buildID:      buildID,
		dockerfile:   dockerfile,
		buildContext: buildContext,
		platforms:    []string{"linux/amd64"},
		buildArgs:    make(map[string]string),
		secrets:      make(map[string]string),
		outputs:      []string{},
		noCache:      false,
	}, nil
}

// Accessors
func (b *BuildxConfig) ID() uuid.UUID                { return b.id }
func (b *BuildxConfig) BuildID() uuid.UUID           { return b.buildID }
func (b *BuildxConfig) Method() BuildMethod          { return BuildMethodBuildx }
func (b *BuildxConfig) Dockerfile() string           { return b.dockerfile }
func (b *BuildxConfig) BuildContext() string         { return b.buildContext }
func (b *BuildxConfig) Platforms() []string          { return b.platforms }
func (b *BuildxConfig) BuildArgs() map[string]string { return b.buildArgs }
func (b *BuildxConfig) Secrets() map[string]string   { return b.secrets }
func (b *BuildxConfig) CacheFrom() string            { return b.cacheFrom }
func (b *BuildxConfig) CacheTo() string              { return b.cacheTo }
func (b *BuildxConfig) Outputs() []string            { return b.outputs }
func (b *BuildxConfig) NoCache() bool                { return b.noCache }

// Setters
func (b *BuildxConfig) SetDockerfile(dockerfile string) error {
	if dockerfile == "" {
		return errors.New("dockerfile cannot be empty")
	}
	b.dockerfile = dockerfile
	return nil
}

func (b *BuildxConfig) AddPlatform(platform string) error {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return errors.New("platform cannot be empty")
	}
	for _, existing := range b.platforms {
		if existing == platform {
			return nil
		}
	}
	b.platforms = append(b.platforms, platform)
	return nil
}

func (b *BuildxConfig) SetBuildArg(key, value string) error {
	if key == "" {
		return errors.New("build arg key cannot be empty")
	}
	b.buildArgs[key] = value
	return nil
}

func (b *BuildxConfig) SetSecret(key, value string) error {
	if key == "" {
		return errors.New("secret key cannot be empty")
	}
	b.secrets[key] = value
	return nil
}

func (b *BuildxConfig) SetCache(from, to string) {
	b.cacheFrom = from
	b.cacheTo = to
}

func (b *BuildxConfig) SetNoCache(noCache bool) {
	b.noCache = noCache
}

// Validate checks if the Buildx configuration is valid
func (b *BuildxConfig) Validate() error {
	if b.buildID == uuid.Nil {
		return errors.New("build id cannot be nil")
	}
	if b.dockerfile == "" {
		return errors.New("dockerfile cannot be empty")
	}
	if b.buildContext == "" {
		return errors.New("build context cannot be empty")
	}
	b.platforms = normalizeBuildxPlatforms(b.platforms)
	if len(b.platforms) == 0 {
		return errors.New("at least one platform must be specified")
	}
	return nil
}

// ToMap converts the configuration to a map
func (b *BuildxConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":            b.id.String(),
		"build_id":      b.buildID.String(),
		"method":        string(b.Method()),
		"dockerfile":    b.dockerfile,
		"build_context": b.buildContext,
		"platforms":     b.platforms,
		"build_args":    b.buildArgs,
		"secrets":       b.secrets,
		"cache_from":    b.cacheFrom,
		"cache_to":      b.cacheTo,
		"outputs":       b.outputs,
		"no_cache":      b.noCache,
	}
}

// ============================================================================
// Kaniko Configuration
// ============================================================================

// KanikoConfig represents configuration for Kaniko container builds
type KanikoConfig struct {
	id               uuid.UUID
	buildID          uuid.UUID
	dockerfile       string
	buildContext     string
	cacheRepo        string
	registryRepo     string
	buildArgs        map[string]string
	skipUnusedStages bool
}

// NewKanikoConfig creates a new Kaniko configuration
func NewKanikoConfig(buildID uuid.UUID, dockerfile, buildContext, registryRepo string) (*KanikoConfig, error) {
	if buildID == uuid.Nil {
		return nil, errors.New("build id cannot be nil")
	}
	if dockerfile == "" {
		return nil, errors.New("dockerfile cannot be empty")
	}
	if buildContext == "" {
		return nil, errors.New("build context cannot be empty")
	}
	if registryRepo == "" {
		return nil, errors.New("registry repo cannot be empty")
	}

	return &KanikoConfig{
		id:               uuid.New(),
		buildID:          buildID,
		dockerfile:       dockerfile,
		buildContext:     buildContext,
		registryRepo:     registryRepo,
		buildArgs:        make(map[string]string),
		skipUnusedStages: true,
	}, nil
}

// Accessors
func (k *KanikoConfig) ID() uuid.UUID                { return k.id }
func (k *KanikoConfig) BuildID() uuid.UUID           { return k.buildID }
func (k *KanikoConfig) Method() BuildMethod          { return BuildMethodKaniko }
func (k *KanikoConfig) Dockerfile() string           { return k.dockerfile }
func (k *KanikoConfig) BuildContext() string         { return k.buildContext }
func (k *KanikoConfig) CacheRepo() string            { return k.cacheRepo }
func (k *KanikoConfig) RegistryRepo() string         { return k.registryRepo }
func (k *KanikoConfig) BuildArgs() map[string]string { return k.buildArgs }
func (k *KanikoConfig) SkipUnusedStages() bool       { return k.skipUnusedStages }

// Setters
func (k *KanikoConfig) SetCacheRepo(cacheRepo string) {
	k.cacheRepo = cacheRepo
}

func (k *KanikoConfig) SetBuildArg(key, value string) error {
	if key == "" {
		return errors.New("build arg key cannot be empty")
	}
	k.buildArgs[key] = value
	return nil
}

func (k *KanikoConfig) SetSkipUnusedStages(skip bool) {
	k.skipUnusedStages = skip
}

// Validate checks if the Kaniko configuration is valid
func (k *KanikoConfig) Validate() error {
	if k.buildID == uuid.Nil {
		return errors.New("build id cannot be nil")
	}
	if k.dockerfile == "" {
		return errors.New("dockerfile cannot be empty")
	}
	if k.buildContext == "" {
		return errors.New("build context cannot be empty")
	}
	if k.registryRepo == "" {
		return errors.New("registry repo cannot be empty")
	}
	return nil
}

// ToMap converts the configuration to a map
func (k *KanikoConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":                 k.id.String(),
		"build_id":           k.buildID.String(),
		"method":             string(k.Method()),
		"dockerfile":         k.dockerfile,
		"build_context":      k.buildContext,
		"cache_repo":         k.cacheRepo,
		"registry_repo":      k.registryRepo,
		"build_args":         k.buildArgs,
		"skip_unused_stages": k.skipUnusedStages,
	}
}

// ============================================================================
// Paketo Configuration
// ============================================================================

// PaketoMethodConfig represents configuration for Paketo builds
type PaketoMethodConfig struct {
	id         uuid.UUID
	buildID    uuid.UUID
	builder    string
	buildpacks []string
	env        map[string]string
	buildArgs  map[string]string
}

// NewPaketoMethodConfig creates a new Paketo configuration
func NewPaketoMethodConfig(buildID uuid.UUID, builder string) (*PaketoMethodConfig, error) {
	if buildID == uuid.Nil {
		return nil, errors.New("build id cannot be nil")
	}
	if builder == "" {
		return nil, errors.New("builder cannot be empty")
	}

	return &PaketoMethodConfig{
		id:         uuid.New(),
		buildID:    buildID,
		builder:    builder,
		buildpacks: []string{},
		env:        make(map[string]string),
		buildArgs:  make(map[string]string),
	}, nil
}

func (p *PaketoMethodConfig) ID() uuid.UUID                { return p.id }
func (p *PaketoMethodConfig) BuildID() uuid.UUID           { return p.buildID }
func (p *PaketoMethodConfig) Method() BuildMethod          { return BuildMethodPaketo }
func (p *PaketoMethodConfig) Builder() string              { return p.builder }
func (p *PaketoMethodConfig) Buildpacks() []string         { return p.buildpacks }
func (p *PaketoMethodConfig) Env() map[string]string       { return p.env }
func (p *PaketoMethodConfig) BuildArgs() map[string]string { return p.buildArgs }

func (p *PaketoMethodConfig) AddBuildpack(buildpack string) error {
	if buildpack == "" {
		return errors.New("buildpack cannot be empty")
	}
	p.buildpacks = append(p.buildpacks, buildpack)
	return nil
}

func (p *PaketoMethodConfig) SetEnv(key, value string) error {
	if key == "" {
		return errors.New("env key cannot be empty")
	}
	p.env[key] = value
	return nil
}

func (p *PaketoMethodConfig) SetBuildArg(key, value string) error {
	if key == "" {
		return errors.New("build arg key cannot be empty")
	}
	p.buildArgs[key] = value
	return nil
}

func (p *PaketoMethodConfig) Validate() error {
	if p.buildID == uuid.Nil {
		return errors.New("build id cannot be nil")
	}
	if p.builder == "" {
		return errors.New("builder cannot be empty")
	}
	return nil
}

func (p *PaketoMethodConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         p.id.String(),
		"build_id":   p.buildID.String(),
		"method":     string(p.Method()),
		"builder":    p.builder,
		"buildpacks": p.buildpacks,
		"env":        p.env,
		"build_args": p.buildArgs,
	}
}

// ============================================================================
// Nix Configuration
// ============================================================================

// NixMethodConfig represents configuration for Nix builds
type NixMethodConfig struct {
	id            uuid.UUID
	buildID       uuid.UUID
	nixExpression string
	flakeURI      string
	attributes    []string
	outputs       map[string]string
	cacheDir      string
	pure          bool
	showTrace     bool
}

// NewNixMethodConfig creates a new Nix configuration
func NewNixMethodConfig(buildID uuid.UUID) (*NixMethodConfig, error) {
	if buildID == uuid.Nil {
		return nil, errors.New("build id cannot be nil")
	}

	return &NixMethodConfig{
		id:         uuid.New(),
		buildID:    buildID,
		attributes: []string{},
		outputs:    make(map[string]string),
		pure:       true,
		showTrace:  false,
	}, nil
}

func (n *NixMethodConfig) ID() uuid.UUID              { return n.id }
func (n *NixMethodConfig) BuildID() uuid.UUID         { return n.buildID }
func (n *NixMethodConfig) Method() BuildMethod        { return BuildMethodNix }
func (n *NixMethodConfig) NixExpression() string      { return n.nixExpression }
func (n *NixMethodConfig) FlakeURI() string           { return n.flakeURI }
func (n *NixMethodConfig) Attributes() []string       { return n.attributes }
func (n *NixMethodConfig) Outputs() map[string]string { return n.outputs }
func (n *NixMethodConfig) CacheDir() string           { return n.cacheDir }
func (n *NixMethodConfig) Pure() bool                 { return n.pure }
func (n *NixMethodConfig) ShowTrace() bool            { return n.showTrace }

func (n *NixMethodConfig) SetNixExpression(expr string) {
	n.nixExpression = expr
}

func (n *NixMethodConfig) SetFlakeURI(uri string) {
	n.flakeURI = uri
}

func (n *NixMethodConfig) AddAttribute(attribute string) error {
	if attribute == "" {
		return errors.New("attribute cannot be empty")
	}
	n.attributes = append(n.attributes, attribute)
	return nil
}

func (n *NixMethodConfig) SetOutput(name, path string) error {
	if name == "" {
		return errors.New("output name cannot be empty")
	}
	n.outputs[name] = path
	return nil
}

func (n *NixMethodConfig) SetCacheDir(cacheDir string) {
	n.cacheDir = cacheDir
}

func (n *NixMethodConfig) SetPure(pure bool) {
	n.pure = pure
}

func (n *NixMethodConfig) SetShowTrace(showTrace bool) {
	n.showTrace = showTrace
}

func (n *NixMethodConfig) Validate() error {
	if n.buildID == uuid.Nil {
		return errors.New("build id cannot be nil")
	}
	if n.nixExpression == "" && n.flakeURI == "" {
		return errors.New("either nix expression or flake uri is required")
	}
	return nil
}

func (n *NixMethodConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             n.id.String(),
		"build_id":       n.buildID.String(),
		"method":         string(n.Method()),
		"nix_expression": n.nixExpression,
		"flake_uri":      n.flakeURI,
		"attributes":     n.attributes,
		"outputs":        n.outputs,
		"cache_dir":      n.cacheDir,
		"pure":           n.pure,
		"show_trace":     n.showTrace,
	}
}

// ============================================================================
// Config Factory
// ============================================================================

// ConfigFactory creates the appropriate config type based on build method
type ConfigFactory struct{}

// NewConfigFactory creates a new config factory
func NewConfigFactory() *ConfigFactory {
	return &ConfigFactory{}
}

// Create creates a new config of the specified type
func (f *ConfigFactory) Create(buildID uuid.UUID, method BuildMethod, params map[string]interface{}) (BuildMethodConfig, error) {
	if !method.IsValid() {
		return nil, fmt.Errorf("invalid build method: %s", method)
	}

	switch method {
	case BuildMethodPacker:
		template, ok := params["template"].(string)
		if !ok {
			return nil, errors.New("packer: template parameter required")
		}
		return NewPackerConfig(buildID, template)

	case BuildMethodBuildx:
		dockerfile, ok := params["dockerfile"].(string)
		if !ok {
			return nil, errors.New("buildx: dockerfile parameter required")
		}
		buildContext, ok := params["build_context"].(string)
		if !ok {
			return nil, errors.New("buildx: build_context parameter required")
		}
		return NewBuildxConfig(buildID, dockerfile, buildContext)

	case BuildMethodKaniko:
		dockerfile, ok := params["dockerfile"].(string)
		if !ok {
			return nil, errors.New("kaniko: dockerfile parameter required")
		}
		buildContext, ok := params["build_context"].(string)
		if !ok {
			return nil, errors.New("kaniko: build_context parameter required")
		}
		registryRepo, ok := params["registry_repo"].(string)
		if !ok {
			return nil, errors.New("kaniko: registry_repo parameter required")
		}
		return NewKanikoConfig(buildID, dockerfile, buildContext, registryRepo)

	case BuildMethodPaketo:
		builder, ok := params["builder"].(string)
		if !ok {
			return nil, errors.New("paketo: builder parameter required")
		}
		return NewPaketoMethodConfig(buildID, builder)

	case BuildMethodNix:
		cfg, err := NewNixMethodConfig(buildID)
		if err != nil {
			return nil, err
		}
		if nixExpression, ok := params["nix_expression"].(string); ok {
			cfg.SetNixExpression(nixExpression)
		}
		if flakeURI, ok := params["flake_uri"].(string); ok {
			cfg.SetFlakeURI(flakeURI)
		}
		return cfg, nil

	default:
		return nil, fmt.Errorf("unsupported build method: %s", method)
	}
}
