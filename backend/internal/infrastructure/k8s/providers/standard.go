package providers

import (
	"context"
	"fmt"
)

// StandardK8sProvider implements the Provider interface for standard Kubernetes
type StandardK8sProvider struct{}

func (p *StandardK8sProvider) Name() string { return "standard-k8s" }
func (p *StandardK8sProvider) DisplayName() string { return "Standard Kubernetes" }

func (p *StandardK8sProvider) AuthMethods() []AuthMethod {
	return []AuthMethod{
		{Name: "kubeconfig", DisplayName: "Kubeconfig", Description: "Use kubeconfig file", RequiredFields: []string{"kubeconfig_path"}},
		{Name: "token", DisplayName: "Token", Description: "Use service account token", RequiredFields: []string{"endpoint", "token", "ca_cert"}},
		{Name: "client-cert", DisplayName: "Client Certificate", Description: "Use client certificate", RequiredFields: []string{"endpoint", "client_cert", "client_key", "ca_cert"}},
	}
}

func (p *StandardK8sProvider) Capabilities() []Capability {
	return []Capability{
		{Name: "vanilla-k8s", DisplayName: "Vanilla Kubernetes", Description: "Standard Kubernetes API"},
		{Name: "customizable", DisplayName: "Customizable", Description: "Highly customizable deployment"},
		{Name: "extensible", DisplayName: "Extensible", Description: "Supports all Kubernetes extensions"},
	}
}

func (p *StandardK8sProvider) ValidateConfig(config ProviderConfig) error {
	if config.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}
	return nil
}

func (p *StandardK8sProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
	return &BaseClusterConnector{config: config}, nil
}

func (p *StandardK8sProvider) DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error) {
	return []ClusterInfo{}, nil
}