package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/infrastructure/k8s/providers"
)

// InfrastructureType represents the type of infrastructure to use
type InfrastructureType string

const (
	InfrastructureKubernetes InfrastructureType = "kubernetes"
	InfrastructureBuildNodes InfrastructureType = "build_nodes"
)

// InfrastructureDecision represents the decision about which infrastructure to use
type InfrastructureDecision struct {
	Type        InfrastructureType `json:"type"`
	Provider    string             `json:"provider,omitempty"`
	Cluster     string             `json:"cluster,omitempty"`
	Reason      string             `json:"reason"`
	Confidence  float64            `json:"confidence"`
	EstimatedCost float64          `json:"estimated_cost,omitempty"`
}

// InfrastructureSelector handles intelligent infrastructure selection
type InfrastructureSelector struct {
	registry      *providers.ProviderRegistry
	k8sAvailable  bool
	costEstimator *CostEstimator
}

// NewInfrastructureSelector creates a new infrastructure selector
func NewInfrastructureSelector(k8sAvailable bool) *InfrastructureSelector {
	registry := providers.NewProviderRegistry()

	// Register all providers
	registry.Register(&providers.AWSProvider{})
	registry.Register(&providers.GCPProvider{})
	registry.Register(&providers.AzureProvider{})
	registry.Register(&providers.OCIProvider{})
	registry.Register(&providers.VMwareProvider{})
	registry.Register(&providers.OpenShiftProvider{})
	registry.Register(&providers.RancherProvider{})
	registry.Register(&providers.StandardK8sProvider{})

	return &InfrastructureSelector{
		registry:      registry,
		k8sAvailable:  k8sAvailable,
		costEstimator: NewCostEstimator(),
	}
}

// SelectInfrastructure selects the appropriate infrastructure for a build
func (s *InfrastructureSelector) SelectInfrastructure(ctx context.Context, build *Build) (*InfrastructureDecision, error) {
	if !s.k8sAvailable {
		return &InfrastructureDecision{
			Type:       InfrastructureBuildNodes,
			Reason:     "Kubernetes infrastructure unavailable, using build nodes",
			Confidence: 1.0,
		}, nil
	}

	// Analyze build requirements
	requirements := s.analyzeBuildRequirements(build)

	// Find suitable Kubernetes clusters
	suitableClusters, err := s.findSuitableClusters(ctx, requirements)
	if err != nil {
		return &InfrastructureDecision{
			Type:       InfrastructureBuildNodes,
			Reason:     fmt.Sprintf("Failed to find suitable Kubernetes clusters: %v", err),
			Confidence: 0.8,
		}, nil
	}

	if len(suitableClusters) == 0 {
		return &InfrastructureDecision{
			Type:       InfrastructureBuildNodes,
			Reason:     "No suitable Kubernetes clusters available for build requirements",
			Confidence: 0.9,
		}, nil
	}

	// Select the best cluster
	bestCluster := s.selectBestCluster(suitableClusters, requirements)

	estimatedCost, _ := s.costEstimator.EstimateCost(bestCluster, requirements)

	return &InfrastructureDecision{
		Type:          InfrastructureKubernetes,
		Provider:      bestCluster.Provider,
		Cluster:       bestCluster.Name,
		Reason:        fmt.Sprintf("Selected %s cluster %s based on capability matching", bestCluster.Provider, bestCluster.Name),
		Confidence:    0.95,
		EstimatedCost: estimatedCost,
	}, nil
}

// AnalyzeBuildRequirements analyzes the build requirements (public for testing)
func (s *InfrastructureSelector) AnalyzeBuildRequirements(build *Build) *BuildRequirements {
	return s.analyzeBuildRequirements(build)
}

// analyzeBuildRequirements analyzes the build requirements
func (s *InfrastructureSelector) analyzeBuildRequirements(build *Build) *BuildRequirements {
	requirements := &BuildRequirements{
		Method:      build.Method,
		Resources:   build.Resources,
		Timeout:     build.Timeout,
		Environment: build.Environment,
	}

	// Determine required capabilities based on build method
	switch build.Method {
	case "docker":
		requirements.Capabilities = []string{"container-runtime"}
	case "kaniko":
		requirements.Capabilities = []string{"container-runtime", "security-context"}
	case "buildx":
		requirements.Capabilities = []string{"container-runtime", "multi-arch"}
	case "packer":
		requirements.Capabilities = []string{"persistent-storage", "privileged"}
	case "nix":
		requirements.Capabilities = []string{"persistent-storage"}
	case "paketo":
		requirements.Capabilities = []string{"container-runtime"}
	}

	// Check for GPU requirements
	if build.RequiresGPU {
		requirements.Capabilities = append(requirements.Capabilities, "gpu-support")
	}

	// Check for high memory requirements
	if build.Resources.MemoryGB > 16 {
		requirements.Capabilities = append(requirements.Capabilities, "high-memory")
	}

	return requirements
}

// findSuitableClusters finds clusters that can handle the build requirements
func (s *InfrastructureSelector) findSuitableClusters(ctx context.Context, requirements *BuildRequirements) ([]*providers.ClusterInfo, error) {
	var suitableClusters []*providers.ClusterInfo

	// Get all providers
	allProviders := s.registry.ListProviders()

	for _, provider := range allProviders {
		// For now, we'll simulate cluster discovery
		// In a real implementation, this would query the database for configured clusters
		clusters := []*providers.ClusterInfo{
			{
				Name:         fmt.Sprintf("%s-cluster-1", provider.Name()),
				Provider:     provider.Name(),
				Status:       "ready",
				NodeCount:    3,
				Capabilities: []string{"container-runtime", "gpu-support", "auto-scaling", "load-balancer"}, // Mock capabilities for testing
			},
		}

		for _, cluster := range clusters {
			if s.clusterMeetsRequirements(cluster, requirements) {
				suitableClusters = append(suitableClusters, cluster)
			}
		}
	}

	return suitableClusters, nil
}

// clusterMeetsRequirements checks if a cluster meets the build requirements
func (s *InfrastructureSelector) clusterMeetsRequirements(cluster *providers.ClusterInfo, requirements *BuildRequirements) bool {
	// Check if cluster is ready
	if cluster.Status != "ready" {
		return false
	}

	// Check required capabilities
	for _, requiredCap := range requirements.Capabilities {
		found := false
		for _, clusterCap := range cluster.Capabilities {
			if clusterCap == requiredCap {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check resource availability (simplified)
	if cluster.NodeCount == 0 {
		return false
	}

	return true
}

// selectBestCluster selects the best cluster from suitable options
func (s *InfrastructureSelector) selectBestCluster(clusters []*providers.ClusterInfo, requirements *BuildRequirements) *providers.ClusterInfo {
	if len(clusters) == 0 {
		return nil
	}

	// For now, select the first cluster
	// In a real implementation, this would consider factors like:
	// - Cost
	// - Performance
	// - Current load
	// - Geographic location
	// - Provider preferences
	return clusters[0]
}

// getProviderCapabilities returns the capabilities of a provider
func (s *InfrastructureSelector) getProviderCapabilities(provider providers.Provider) []string {
	capabilities := provider.Capabilities()
	capabilityNames := make([]string, len(capabilities))
	for i, cap := range capabilities {
		capabilityNames[i] = cap.Name
	}
	return capabilityNames
}

// Build represents a build request
type Build struct {
	ID          string
	Method      string
	Resources   BuildResources
	Timeout     time.Duration
	Environment map[string]string
	RequiresGPU bool
}

// BuildResources represents the resource requirements for a build
type BuildResources struct {
	CPU     float64
	MemoryGB float64
	DiskGB   float64
}

// BuildRequirements represents the analyzed requirements for a build
type BuildRequirements struct {
	Method       string
	Resources    BuildResources
	Timeout      time.Duration
	Environment  map[string]string
	Capabilities []string
}

// CostEstimator estimates the cost of running a build on a cluster
type CostEstimator struct{}

// NewCostEstimator creates a new cost estimator
func NewCostEstimator() *CostEstimator {
	return &CostEstimator{}
}

// EstimateCost estimates the cost of running a build on a cluster
func (c *CostEstimator) EstimateCost(cluster *providers.ClusterInfo, requirements *BuildRequirements) (float64, error) {
	// Simplified cost estimation
	// In a real implementation, this would consider:
	// - Provider pricing
	// - Instance types
	// - Regional pricing
	// - Reserved instances, etc.

	baseCost := 0.1 // Base cost per minute

	// Adjust based on resources
	if requirements.Resources.CPU > 2 {
		baseCost *= 1.5
	}
	if requirements.Resources.MemoryGB > 8 {
		baseCost *= 1.3
	}

	// Estimate duration (simplified)
	estimatedDuration := 10.0 // minutes

	return baseCost * estimatedDuration, nil
}