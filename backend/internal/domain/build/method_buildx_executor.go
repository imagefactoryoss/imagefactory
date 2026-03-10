package build

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MethodBuildxExecutor implements BuildMethodExecutor for Docker Buildx builds
type MethodBuildxExecutor struct {
	service BuildExecutionService
	running map[string]*exec.Cmd
	mu      sync.Mutex
}

// NewMethodBuildxExecutor creates a new Docker Buildx executor
func NewMethodBuildxExecutor(service BuildExecutionService) BuildMethodExecutor {
	return &MethodBuildxExecutor{
		service: service,
		running: make(map[string]*exec.Cmd),
	}
}

// Supports checks if this executor supports Buildx builds
func (e *MethodBuildxExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodBuildx
}

// Execute runs a Docker Buildx build
func (e *MethodBuildxExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Check if docker and buildx are available
	if err := e.validateBuildxInstalled(); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Buildx validation failed: %v", err), nil)
		return nil, err
	}

	output := &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		StartTime:   time.Now().Unix(),
		Artifacts:   []MethodArtifact{},
	}

	e.service.AddLog(ctx, executionID, LogInfo, "Starting Docker Buildx build", nil)

	config, err := configForMethod(ctx, BuildMethodBuildx)
	if err != nil {
		return nil, err
	}
	buildContext := config.BuildContext
	if buildContext == "" {
		buildContext = "."
	}
	buildContext, err = filepath.Abs(buildContext)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build context: %w", err)
	}

	args := []string{"buildx", "build", "--progress", "plain", "--tag", executionID.String()}
	if config.Dockerfile != "" {
		dockerfile := config.Dockerfile
		if !filepath.IsAbs(dockerfile) {
			dockerfile = filepath.Join(buildContext, dockerfile)
		}
		args = append(args, "--file", dockerfile)
	}
	if len(config.Platforms) > 0 {
		args = append(args, "--platform", strings.Join(config.Platforms, ","))
	}
	for key, value := range config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}
	if config.CacheTo != "" {
		args = append(args, "--cache-to", config.CacheTo)
	}
	if len(config.CacheFrom) > 0 {
		args = append(args, "--cache-from", config.CacheFrom[0])
	}
	args = append(args, buildContext)
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Register running command
	e.mu.Lock()
	e.running[executionID.String()] = cmd
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.running, executionID.String())
		e.mu.Unlock()
	}()

	// Stream output
	output.Output, err = e.streamBuildxBuild(ctx, executionID, cmd)
	if err != nil {
		output.Status = ExecutionFailed
		output.ErrorMessage = err.Error()
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Buildx build failed: %v", err), nil)
	} else {
		output.Status = ExecutionSuccess
		e.service.AddLog(ctx, executionID, LogInfo, "Buildx build completed successfully", nil)

		// Parse artifacts from output
		artifacts, err := e.parseBuildxArtifacts(output.Output)
		if err == nil {
			output.Artifacts = artifacts
		}
	}

	output.EndTime = time.Now().Unix()
	output.Duration = int(output.EndTime - output.StartTime)

	return output, nil
}

// Cancel stops a running Buildx build
func (e *MethodBuildxExecutor) Cancel(ctx context.Context, executionID string) error {
	e.mu.Lock()
	cmd, ok := e.running[executionID]
	e.mu.Unlock()

	if !ok {
		return ErrExecutionNotRunning
	}

	if cmd.Process != nil {
		return cmd.Process.Kill()
	}

	return nil
}

// GetStatus retrieves the current status of a Buildx execution
func (e *MethodBuildxExecutor) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
	e.mu.Lock()
	cmd, ok := e.running[executionID]
	e.mu.Unlock()

	if !ok {
		return "", ErrExecutionNotFound
	}

	if cmd.ProcessState == nil {
		return ExecutionRunning, nil
	}

	if cmd.ProcessState.Success() {
		return ExecutionSuccess, nil
	}

	return ExecutionFailed, nil
}

// streamBuildxBuild streams output from a Docker Buildx build command
func (e *MethodBuildxExecutor) streamBuildxBuild(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start buildx: %w", err)
	}

	var output strings.Builder
	var wg sync.WaitGroup

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			e.service.AddLog(ctx, executionID, LogInfo, line, nil)
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			e.service.AddLog(ctx, executionID, LogError, line, nil)
		}
	}()

	// Wait for completion
	err = cmd.Wait()
	wg.Wait()

	return output.String(), err
}

// parseBuildxArtifacts extracts container image artifacts from Buildx output
func (e *MethodBuildxExecutor) parseBuildxArtifacts(output string) ([]MethodArtifact, error) {
	artifacts := []MethodArtifact{}

	// Parse buildx JSON output
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "digest") && !strings.Contains(line, "image") {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		// Extract image digest as artifact
		if digest, ok := data["digest"].(string); ok && digest != "" {
			artifact := MethodArtifact{
				Name:        digest,
				Type:        "image",
				Size:        0,
				Path:        digest,
				Checksum:    digest,
				ContentType: "application/vnd.docker.container.image.v1+json",
				CreatedAt:   time.Now().Unix(),
			}
			artifacts = append(artifacts, artifact)
		}

		// Extract image reference
		if imageRef, ok := data["imageName"].(string); ok && imageRef != "" {
			artifact := MethodArtifact{
				Name:        imageRef,
				Type:        "image",
				Size:        0,
				Path:        imageRef,
				Checksum:    "",
				ContentType: "application/vnd.docker.container.image.v1+json",
				CreatedAt:   time.Now().Unix(),
			}
			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts, nil
}

// validateBuildxInstalled checks if Docker and Buildx are installed
func (e *MethodBuildxExecutor) validateBuildxInstalled() error {
	// Check docker
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not installed: %w", err)
	}

	// Check buildx plugin
	cmd = exec.Command("docker", "buildx", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker buildx plugin not available: %w", err)
	}

	return nil
}
