package providers

import (
	"context"
	"fmt"
)

// ProviderRegistry manages all Kubernetes providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// GetProvider retrieves a provider by name
func (r *ProviderRegistry) GetProvider(name string) (Provider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// ListProviders returns all registered providers
func (r *ProviderRegistry) ListProviders() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// Provider defines the interface for Kubernetes providers
type Provider interface {
	Name() string
	DisplayName() string
	AuthMethods() []AuthMethod
	Capabilities() []Capability
	ValidateConfig(config ProviderConfig) error
	CreateConnector(config ProviderConfig) (ClusterConnector, error)
	DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error)
}

// AuthMethod represents an authentication method for a provider
type AuthMethod struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	RequiredFields []string `json:"required_fields"`
}

// Capability represents a capability of a provider
type Capability struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// ProviderConfig holds configuration for connecting to a provider
type ProviderConfig struct {
	AuthMethod   string            `json:"auth_method"`
	Region       string            `json:"region,omitempty"`
	ClusterName  string            `json:"cluster_name,omitempty"`
	Endpoint     string            `json:"endpoint,omitempty"`
	Credentials  map[string]string `json:"credentials"`
	ExtraConfig  map[string]interface{} `json:"extra_config,omitempty"`
}

// ClusterInfo represents information about a discovered cluster
type ClusterInfo struct {
	Name         string            `json:"name"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	NodeCount    int               `json:"node_count"`
	Endpoint     string            `json:"endpoint"`
	Labels       map[string]string `json:"labels"`
	Capabilities []string          `json:"capabilities"`
}

// ClusterConnector defines the interface for connecting to clusters
type ClusterConnector interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	GetClusterInfo(ctx context.Context) (*ClusterInfo, error)
	ExecuteBuild(ctx context.Context, build *BuildRequest) (*BuildResult, error)
	IsHealthy(ctx context.Context) (bool, error)
}

// BuildRequest represents a build execution request
type BuildRequest struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	Config      map[string]interface{} `json:"config"`
	Resources   ResourceRequirements `json:"resources"`
	Timeout     int                 `json:"timeout_seconds"`
	Environment map[string]string   `json:"environment"`
}

// ResourceRequirements defines resource requirements for a build
type ResourceRequirements struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
}

// BuildResult represents the result of a build execution
type BuildResult struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	Duration  int    `json:"duration_seconds"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
}

// Artifact represents a build artifact
type Artifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}