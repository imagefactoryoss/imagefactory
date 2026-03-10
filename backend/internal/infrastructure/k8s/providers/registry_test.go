package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Test registering a provider
	awsProvider := &AWSProvider{}
	registry.Register(awsProvider)

	// Test provider retrieval
	provider, err := registry.GetProvider("aws-eks")
	assert.NoError(t, err)
	assert.Equal(t, "Amazon EKS", provider.DisplayName())

	// Test auth methods
	authMethods := provider.AuthMethods()
	found := false
	for _, method := range authMethods {
		if method.Name == "iam" {
			found = true
			break
		}
	}
	assert.True(t, found, "iam auth method should be present")

	// Test listing all providers
	providers := registry.ListProviders()
	assert.Len(t, providers, 1)
	assert.Equal(t, "aws-eks", providers[0].Name())

	// Test getting non-existent provider
	_, err = registry.GetProvider("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestProviderRegistry_MultipleProviders(t *testing.T) {
	registry := NewProviderRegistry()

	// Register multiple providers
	providers := []Provider{
		&AWSProvider{},
		&GCPProvider{},
		&AzureProvider{},
		&OCIProvider{},
		&VMwareProvider{},
		&OpenShiftProvider{},
		&RancherProvider{},
		&StandardK8sProvider{},
	}

	for _, provider := range providers {
		registry.Register(provider)
	}

	// Test all providers are registered
	allProviders := registry.ListProviders()
	assert.Len(t, allProviders, 8)

	// Test specific provider retrieval
	gcpProvider, err := registry.GetProvider("gcp-gke")
	assert.NoError(t, err)
	assert.Equal(t, "Google Kubernetes Engine", gcpProvider.DisplayName())
}

func TestProviderCapabilities(t *testing.T) {
	awsProvider := &AWSProvider{}

	capabilities := awsProvider.Capabilities()

	// Check that expected capabilities are present
	capabilityNames := make([]string, len(capabilities))
	for i, cap := range capabilities {
		capabilityNames[i] = cap.Name
	}

	assert.Contains(t, capabilityNames, "gpu-support")
	assert.Contains(t, capabilityNames, "auto-scaling")
	assert.Contains(t, capabilityNames, "load-balancer")
}

func TestProviderValidation(t *testing.T) {
	awsProvider := &AWSProvider{}

	// Test valid config
	validConfig := ProviderConfig{
		AuthMethod: "iam",
		Region:     "us-east-1",
		ClusterName: "test-cluster",
	}
	err := awsProvider.ValidateConfig(validConfig)
	assert.NoError(t, err)

	// Test invalid config
	invalidConfig := ProviderConfig{
		AuthMethod: "invalid",
	}
	err = awsProvider.ValidateConfig(invalidConfig)
	assert.Error(t, err)
}

func TestProviderDiscovery(t *testing.T) {
	awsProvider := &AWSProvider{}

	// Mock config for testing
	config := ProviderConfig{
		AuthMethod: "iam",
		Region:     "us-east-1",
	}

	// This would normally connect to AWS, but for now returns empty slice
	clusters, err := awsProvider.DiscoverClusters(context.Background(), config)
	assert.NoError(t, err)
	assert.Empty(t, clusters)
}