package providers

import (
	"context"
	"fmt"
)

// RancherProvider implements the Provider interface for Rancher
type RancherProvider struct{}

func (p *RancherProvider) Name() string { return "rancher" }
func (p *RancherProvider) DisplayName() string { return "Rancher Kubernetes" }

func (p *RancherProvider) AuthMethods() []AuthMethod {
	return []AuthMethod{
		{Name: "api-key", DisplayName: "API Key", Description: "Use Rancher API key", RequiredFields: []string{"endpoint", "api_key"}},
		{Name: "kubeconfig", DisplayName: "Kubeconfig", Description: "Use kubeconfig file", RequiredFields: []string{"kubeconfig_path"}},
	}
}

func (p *RancherProvider) Capabilities() []Capability {
	return []Capability{
		{Name: "multi-cluster", DisplayName: "Multi-Cluster", Description: "Multi-cluster management"},
		{Name: "cattle", DisplayName: "Cattle", Description: "Rancher cattle orchestration"},
		{Name: "fleet", DisplayName: "Fleet", Description: "GitOps with Fleet"},
		{Name: "monitoring", DisplayName: "Monitoring", Description: "Integrated monitoring stack"},
	}
}

func (p *RancherProvider) ValidateConfig(config ProviderConfig) error {
	if config.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}
	return nil
}

func (p *RancherProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
	return &BaseClusterConnector{config: config}, nil
}

func (p *RancherProvider) DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error) {
	return []ClusterInfo{}, nil
}