package providers

import (
	"context"
	"fmt"
)

// OpenShiftProvider implements the Provider interface for OpenShift
type OpenShiftProvider struct{}

func (p *OpenShiftProvider) Name() string { return "openshift" }
func (p *OpenShiftProvider) DisplayName() string { return "Red Hat OpenShift" }

func (p *OpenShiftProvider) AuthMethods() []AuthMethod {
	return []AuthMethod{
		{Name: "kubeconfig", DisplayName: "Kubeconfig", Description: "Use kubeconfig file", RequiredFields: []string{"kubeconfig_path"}},
		{Name: "token", DisplayName: "Token", Description: "Use service account token", RequiredFields: []string{"endpoint", "token"}},
	}
}

func (p *OpenShiftProvider) Capabilities() []Capability {
	return []Capability{
		{Name: "security-context", DisplayName: "Security Context", Description: "Advanced security contexts"},
		{Name: "operators", DisplayName: "Operators", Description: "Operator framework support"},
		{Name: "routes", DisplayName: "Routes", Description: "OpenShift routes for ingress"},
		{Name: "builds", DisplayName: "Builds", Description: "Integrated build system"},
	}
}

func (p *OpenShiftProvider) ValidateConfig(config ProviderConfig) error {
	if config.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}
	return nil
}

func (p *OpenShiftProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
	return &BaseClusterConnector{config: config}, nil
}

func (p *OpenShiftProvider) DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error) {
	return []ClusterInfo{}, nil
}