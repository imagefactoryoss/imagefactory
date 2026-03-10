package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrBuildExecutionInProgress = errors.New("build execution is still running")

// MethodExecutorAdapter bridges BuildMethodExecutorFactory into BuildExecutor.
type MethodExecutorAdapter struct {
	factory BuildMethodExecutorFactory
	logger  *zap.Logger
}

// NewMethodExecutorAdapter creates a new adapter.
func NewMethodExecutorAdapter(factory BuildMethodExecutorFactory, logger *zap.Logger) *MethodExecutorAdapter {
	return &MethodExecutorAdapter{
		factory: factory,
		logger:  logger,
	}
}

// Execute runs a build using the configured build method executor.
func (e *MethodExecutorAdapter) Execute(ctx context.Context, b *Build) (*BuildResult, error) {
	if b == nil {
		return nil, fmt.Errorf("build is required")
	}
	ctx = WithBuild(ctx, b)

	config := b.Config()
	if config == nil {
		return nil, fmt.Errorf("build config is required for method execution")
	}
	if config.ID == uuid.Nil {
		return nil, fmt.Errorf("build config id is required for method execution")
	}

	method := BuildMethod(config.BuildMethod)
	if method == BuildMethod("container") {
		method = BuildMethodDocker
	}
	if !method.IsValid() {
		return nil, fmt.Errorf("invalid build method: %s", config.BuildMethod)
	}

	executor, err := e.factory.CreateExecutor(method)
	if err != nil {
		return nil, err
	}

	output, err := executor.Execute(ctx, config.ID.String(), method)
	if err != nil {
		return nil, err
	}
	if output.Status == ExecutionFailed {
		if output.ErrorMessage != "" {
			return nil, fmt.Errorf("build execution failed: %s", output.ErrorMessage)
		}
		return nil, fmt.Errorf("build execution failed")
	}
	if output.Status == ExecutionCancelled {
		return nil, fmt.Errorf("build execution cancelled")
	}
	if output.Status == ExecutionRunning || output.Status == ExecutionPending {
		return nil, ErrBuildExecutionInProgress
	}

	result := &BuildResult{
		Logs:        []string{output.Output},
		Duration:    time.Duration(output.Duration) * time.Second,
		Artifacts:   artifactNames(output.Artifacts),
		ScanResults: map[string]interface{}{"method_artifacts": output.Artifacts},
	}
	enrichBuildResultFromArtifacts(result, output.Artifacts)

	return result, nil
}

// Cancel is a no-op for method executors without execution ID context.
func (e *MethodExecutorAdapter) Cancel(ctx context.Context, buildID uuid.UUID) error {
	e.logger.Debug("Method executor cancel requested", zap.String("build_id", buildID.String()))
	return nil
}

func artifactNames(artifacts []MethodArtifact) []string {
	if len(artifacts) == 0 {
		return nil
	}
	names := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		switch {
		case strings.TrimSpace(artifact.Name) != "":
			names = append(names, strings.TrimSpace(artifact.Name))
		case strings.TrimSpace(artifact.Path) != "":
			names = append(names, strings.TrimSpace(artifact.Path))
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

func enrichBuildResultFromArtifacts(result *BuildResult, artifacts []MethodArtifact) {
	if result == nil || len(artifacts) == 0 {
		return
	}
	var totalSize int64
	for _, artifact := range artifacts {
		totalSize += artifact.Size
		digest := strings.TrimSpace(artifact.Checksum)
		if result.ImageDigest == "" && strings.HasPrefix(digest, "sha256:") {
			result.ImageDigest = digest
		}
		if result.ImageID == "" {
			path := strings.TrimSpace(artifact.Path)
			name := strings.TrimSpace(artifact.Name)
			switch {
			case strings.Contains(path, "/") || strings.Contains(path, ":"):
				result.ImageID = path
			case strings.Contains(name, "/") || strings.Contains(name, ":"):
				result.ImageID = name
			}
		}
	}
	if totalSize > 0 {
		result.Size = totalSize
	}

	// Preserve raw method artifacts as JSON-safe fallback payload for downstream parsers.
	if result.ScanResults == nil {
		result.ScanResults = map[string]interface{}{}
	}
	if _, exists := result.ScanResults["method_artifacts_json"]; !exists {
		if payload, err := json.Marshal(artifacts); err == nil {
			result.ScanResults["method_artifacts_json"] = string(payload)
		}
	}
}
