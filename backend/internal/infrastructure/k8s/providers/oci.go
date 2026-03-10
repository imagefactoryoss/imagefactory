package providers

import (
	"context"
	"fmt"
)

// OCIProvider implements the Provider interface for Oracle OKE
type OCIProvider struct{}

func (p *OCIProvider) Name() string { return "oci-oke" }
func (p *OCIProvider) DisplayName() string { return "Oracle Container Engine for Kubernetes" }

func (p *OCIProvider) AuthMethods() []AuthMethod {
	return []AuthMethod{
		{Name: "instance-principal", DisplayName: "Instance Principal", Description: "Use OCI instance principal", RequiredFields: []string{"tenancy_ocid", "region"}},
		{Name: "api-key", DisplayName: "API Key", Description: "Use OCI API key authentication", RequiredFields: []string{"tenancy_ocid", "user_ocid", "fingerprint", "private_key", "region"}},
	}
}

func (p *OCIProvider) Capabilities() []Capability {
	return []Capability{
		{Name: "gpu-support", DisplayName: "GPU Support", Description: "Supports GPU-enabled node pools"},
		{Name: "auto-scaling", DisplayName: "Auto Scaling", Description: "Supports cluster auto-scaling"},
		{Name: "load-balancer", DisplayName: "Load Balancer", Description: "Supports OCI Load Balancer"},
		{Name: "persistent-storage", DisplayName: "Persistent Storage", Description: "Supports OCI Block Storage"},
	}
}

func (p *OCIProvider) ValidateConfig(config ProviderConfig) error {
	if config.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}
	return nil
}

func (p *OCIProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
	return &BaseClusterConnector{config: config}, nil
}

func (p *OCIProvider) DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error) {
	return []ClusterInfo{}, nil
}