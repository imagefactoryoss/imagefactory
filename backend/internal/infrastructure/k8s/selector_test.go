package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/srikarm/image-factory/internal/infrastructure/k8s/providers"
	"github.com/stretchr/testify/assert"
)

func TestInfrastructureSelector_K8sUnavailable(t *testing.T) {
	selector := NewInfrastructureSelector(false)

	build := &Build{
		ID:     "test-build",
		Method: "docker",
		Resources: BuildResources{
			CPU:      1,
			MemoryGB: 2,
			DiskGB:   10,
		},
		Timeout: 30 * time.Minute,
	}

	decision, err := selector.SelectInfrastructure(context.Background(), build)

	assert.NoError(t, err)
	assert.Equal(t, InfrastructureBuildNodes, decision.Type)
	assert.Contains(t, decision.Reason, "unavailable")
	assert.Equal(t, 1.0, decision.Confidence)
}

func TestInfrastructureSelector_K8sAvailable(t *testing.T) {
	selector := NewInfrastructureSelector(true)

	build := &Build{
		ID:     "test-build",
		Method: "docker",
		Resources: BuildResources{
			CPU:      1,
			MemoryGB: 2,
			DiskGB:   10,
		},
		Timeout: 30 * time.Minute,
	}

	decision, err := selector.SelectInfrastructure(context.Background(), build)

	assert.NoError(t, err)
	assert.Equal(t, InfrastructureKubernetes, decision.Type)
	assert.Contains(t, decision.Reason, "Selected")
	assert.Equal(t, 0.95, decision.Confidence)
	assert.NotEmpty(t, decision.Provider)
	assert.NotEmpty(t, decision.Cluster)
}

func TestInfrastructureSelector_BuildRequirements(t *testing.T) {
	selector := NewInfrastructureSelector(true)

	tests := []struct {
		name       string
		build      *Build
		expectCaps []string
	}{
		{
			name: "Docker build",
			build: &Build{Method: "docker"},
			expectCaps: []string{"container-runtime"},
		},
		{
			name: "Kaniko build",
			build: &Build{Method: "kaniko"},
			expectCaps: []string{"container-runtime", "security-context"},
		},
		{
			name: "GPU required",
			build: &Build{Method: "docker", RequiresGPU: true},
			expectCaps: []string{"container-runtime", "gpu-support"},
		},
		{
			name: "High memory",
			build: &Build{
				Method: "docker",
				Resources: BuildResources{MemoryGB: 32},
			},
			expectCaps: []string{"container-runtime", "high-memory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := selector.AnalyzeBuildRequirements(tt.build)
			assert.Equal(t, tt.build.Method, requirements.Method)
			for _, cap := range tt.expectCaps {
				assert.Contains(t, requirements.Capabilities, cap)
			}
		})
	}
}

func TestCostEstimator(t *testing.T) {
	estimator := NewCostEstimator()

	cluster := &providers.ClusterInfo{
		Name:     "test-cluster",
		Provider: "aws-eks",
	}

	requirements := &BuildRequirements{
		Resources: BuildResources{
			CPU:      4,
			MemoryGB: 16,
		},
	}

	cost, err := estimator.EstimateCost(cluster, requirements)
	assert.NoError(t, err)
	assert.Greater(t, cost, 0.0)
}