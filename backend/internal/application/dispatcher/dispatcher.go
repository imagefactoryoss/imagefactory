package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/infrastructure/k8s"
)

// BuildExecutor defines the interface for executing builds
type BuildExecutor interface {
	Execute(ctx context.Context, build *build.Build) (any, error)
	Cancel(ctx context.Context, buildID uuid.UUID) error
}

// QueueManager defines the interface for queuing builds for execution
type QueueManager interface {
	QueueBuild(ctx context.Context, build *build.Build, executor BuildExecutor) error
}

// MetricsCollector defines the interface for collecting infrastructure metrics
type MetricsCollector interface {
	RecordInfrastructureUsage(ctx context.Context, build *build.Build, decision *k8s.InfrastructureDecision, duration time.Duration, err error)
}

// BuildRepository defines the interface for build persistence
type BuildRepository interface {
	UpdateInfrastructureSelection(ctx context.Context, build *build.Build) error
}

// SmartBuildDispatcher handles intelligent routing of builds to appropriate infrastructure
type SmartBuildDispatcher struct {
	selector       *k8s.InfrastructureSelector
	k8sExecutor    BuildExecutor
	nodeExecutor   BuildExecutor
	queueManager   QueueManager
	metrics        MetricsCollector
	repo           BuildRepository
}

// NewSmartBuildDispatcher creates a new smart build dispatcher
func NewSmartBuildDispatcher(
	selector *k8s.InfrastructureSelector,
	k8sExecutor, nodeExecutor BuildExecutor,
	queueManager QueueManager,
	metrics MetricsCollector,
	repo BuildRepository,
) *SmartBuildDispatcher {
	return &SmartBuildDispatcher{
		selector:     selector,
		k8sExecutor:  k8sExecutor,
		nodeExecutor: nodeExecutor,
		queueManager: queueManager,
		metrics:      metrics,
		repo:         repo,
	}
}

// Dispatch selects appropriate infrastructure and routes the build for execution
func (d *SmartBuildDispatcher) Dispatch(ctx context.Context, build *build.Build) error {
	startTime := time.Now()

	// Convert domain build to k8s build for selection
	k8sBuild := d.convertToK8sBuild(build)

	// Select infrastructure
	if d.selector == nil {
		d.metrics.RecordInfrastructureUsage(ctx, build, nil, time.Since(startTime), fmt.Errorf("infrastructure selector not configured"))
		return fmt.Errorf("infrastructure selector not configured")
	}
	decision, err := d.selector.SelectInfrastructure(ctx, k8sBuild)
	if err != nil {
		d.metrics.RecordInfrastructureUsage(ctx, build, nil, time.Since(startTime), err)
		return fmt.Errorf("infrastructure selection failed: %w", err)
	}

	// Update build with selection
	build.SetInfrastructureSelection(string(decision.Type), decision.Reason)

	// Save infrastructure selection to database
	if err := d.repo.UpdateInfrastructureSelection(ctx, build); err != nil {
		d.metrics.RecordInfrastructureUsage(ctx, build, decision, time.Since(startTime), err)
		return fmt.Errorf("failed to save infrastructure selection: %w", err)
	}

	// Route to appropriate executor
	var executor BuildExecutor
	switch decision.Type {
	case k8s.InfrastructureKubernetes:
		executor = d.k8sExecutor
	case k8s.InfrastructureBuildNodes:
		executor = d.nodeExecutor
	default:
		d.metrics.RecordInfrastructureUsage(ctx, build, decision, time.Since(startTime), fmt.Errorf("unsupported infrastructure type: %s", decision.Type))
		return fmt.Errorf("unsupported infrastructure type: %s", decision.Type)
	}

	// Queue build for execution
	if err := d.queueManager.QueueBuild(ctx, build, executor); err != nil {
		d.metrics.RecordInfrastructureUsage(ctx, build, decision, time.Since(startTime), err)
		return fmt.Errorf("failed to queue build: %w", err)
	}

	// Record successful dispatch
	d.metrics.RecordInfrastructureUsage(ctx, build, decision, time.Since(startTime), nil)

	return nil
}

// GetInfrastructureRecommendation returns infrastructure recommendations for a build
func (d *SmartBuildDispatcher) GetInfrastructureRecommendation(ctx context.Context, build *build.Build) (*k8s.InfrastructureDecision, error) {
	k8sBuild := d.convertToK8sBuild(build)
	return d.selector.SelectInfrastructure(ctx, k8sBuild)
}

// convertToK8sBuild converts a domain build to a k8s build for selection
func (d *SmartBuildDispatcher) convertToK8sBuild(build *build.Build) *k8s.Build {
	manifest := build.Manifest()
	config := build.Config()

	// Determine build method
	method := "docker" // default
	if config != nil {
		method = config.BuildMethod
	}

	// Extract resources (simplified)
	resources := k8s.BuildResources{
		CPU:     2.0, // default
		MemoryGB: 4.0, // default
		DiskGB:   20.0, // default
	}

	// Check for GPU requirements (simplified check)
	requiresGPU := false
	if config != nil {
		// Check if any environment variables or build args indicate GPU usage
		for _, env := range config.Environment {
			if hasGPUKeywords(env) {
				requiresGPU = true
				break
			}
		}
		for _, arg := range config.BuildArgs {
			if hasGPUKeywords(arg) {
				requiresGPU = true
				break
			}
		}
	}

	return &k8s.Build{
		ID:          build.ID().String(),
		Method:      method,
		Resources:   resources,
		Timeout:     30 * time.Minute, // default timeout
		Environment: manifest.Environment,
		RequiresGPU: requiresGPU,
	}
}

// hasGPUKeywords checks if a string contains GPU-related keywords
func hasGPUKeywords(s string) bool {
	gpuKeywords := []string{"gpu", "cuda", "nvidia", "amd", "radeon"}
	for _, keyword := range gpuKeywords {
		if strings.Contains(strings.ToLower(s), keyword) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// InfrastructureStats represents infrastructure usage statistics
type InfrastructureStats struct {
	TenantID             uuid.UUID     `json:"tenant_id"`
	Period               time.Duration `json:"period"`
	TotalBuilds          int           `json:"total_builds"`
	KubernetesBuilds     int           `json:"kubernetes_builds"`
	BuildNodeBuilds      int           `json:"build_node_builds"`
	AvgSelectionTime     time.Duration `json:"avg_selection_time"`
	SelectionSuccessRate float64       `json:"selection_success_rate"`
}

// GetInfrastructureStats returns infrastructure usage statistics
func (d *SmartBuildDispatcher) GetInfrastructureStats(ctx context.Context, tenantID uuid.UUID, period time.Duration) (*InfrastructureStats, error) {
	// For now, return mock stats
	// This would typically query the metrics/usage data
	return &InfrastructureStats{
		TenantID:            tenantID,
		Period:              period,
		TotalBuilds:         100,
		KubernetesBuilds:    95,
		BuildNodeBuilds:     5,
		AvgSelectionTime:    50 * time.Millisecond,
		SelectionSuccessRate: 0.99,
	}, nil
}