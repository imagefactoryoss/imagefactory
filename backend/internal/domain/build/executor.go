package build

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NoOpBuildExecutor is a simple implementation that simulates build execution
type NoOpBuildExecutor struct {
	logger *zap.Logger
}

// NewNoOpBuildExecutor creates a new no-op build executor
func NewNoOpBuildExecutor(logger *zap.Logger) *NoOpBuildExecutor {
	return &NoOpBuildExecutor{
		logger: logger,
	}
}

// Execute simulates build execution
func (e *NoOpBuildExecutor) Execute(ctx context.Context, build *Build) (*BuildResult, error) {
	buildID := build.ID()
	manifest := build.Manifest()

	e.logger.Info("Starting build execution",
		zap.String("build_id", buildID.String()),
		zap.String("build_type", string(manifest.Type)),
		zap.String("build_name", manifest.Name),
	)

	// Simulate build execution time based on build type
	var executionTime time.Duration
	switch manifest.Type {
	case BuildTypeContainer:
		executionTime = 30 * time.Second // Container builds are faster
	case BuildTypeVM:
		executionTime = 5 * time.Minute // VM builds take longer
	case BuildTypeCloud:
		executionTime = 3 * time.Minute // Cloud builds are medium
	default:
		executionTime = 1 * time.Minute
	}

	// Simulate execution
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(executionTime):
		// Build completed
	}

	// Generate mock result based on build type
	result := e.generateMockResult(build)
	e.logger.Info("Build execution completed",
		zap.String("build_id", buildID.String()),
		zap.String("image_id", result.ImageID),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// Cancel simulates build cancellation
func (e *NoOpBuildExecutor) Cancel(ctx context.Context, buildID uuid.UUID) error {
	e.logger.Info("Cancelling build execution", zap.String("build_id", buildID.String()))

	// In a real implementation, this would signal the build process to stop
	// For now, just log the cancellation

	return nil
}

// generateMockResult creates a mock build result based on the build type
func (e *NoOpBuildExecutor) generateMockResult(build *Build) *BuildResult {
	buildID := build.ID()
	manifest := build.Manifest()

	baseResult := BuildResult{
		ImageID:   fmt.Sprintf("%s-%s", manifest.Type, buildID.String()[:8]),
		ImageDigest: fmt.Sprintf("sha256:%s", buildID.String()),
		Duration:  2 * time.Minute, // Mock duration
		Logs:      []string{"Build started", "Processing manifest", "Build completed successfully"},
		Artifacts: []string{fmt.Sprintf("%s.tar.gz", manifest.Name)},
		SBOM:      map[string]interface{}{"format": "spdx", "version": "2.3"},
		ScanResults: map[string]interface{}{
			"vulnerabilities": map[string]int{"critical": 0, "high": 1, "medium": 3, "low": 5},
			"passed": true,
		},
	}

	// Customize result based on build type
	switch manifest.Type {
	case BuildTypeVM:
		baseResult.Size = 5 * 1024 * 1024 * 1024 // 5GB for VM images
		baseResult.Artifacts = []string{
			fmt.Sprintf("%s.ova", manifest.Name),
			fmt.Sprintf("%s.vmdk", manifest.Name),
			fmt.Sprintf("%s.vhd", manifest.Name),
		}
		baseResult.Duration = 4 * time.Minute

	case BuildTypeContainer:
		baseResult.Size = 500 * 1024 * 1024 // 500MB for container images
		baseResult.Artifacts = []string{
			fmt.Sprintf("%s.tar.gz", manifest.Name),
			fmt.Sprintf("%s.sbom.json", manifest.Name),
		}
		baseResult.Duration = 30 * time.Second

	case BuildTypeCloud:
		baseResult.Size = 2 * 1024 * 1024 * 1024 // 2GB for cloud images
		baseResult.Artifacts = []string{
			fmt.Sprintf("%s-ami.json", manifest.Name),
			fmt.Sprintf("%s-azure.json", manifest.Name),
		}
		baseResult.Duration = 2 * time.Minute
	}

	return &baseResult
}