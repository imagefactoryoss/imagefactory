package build

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MethodDockerExecutor implements BuildMethodExecutor for simple Docker builds
type MethodDockerExecutor struct {
	service BuildExecutionService
	running map[string]*exec.Cmd
	mu      sync.Mutex
}

// NewMethodDockerExecutor creates a new Docker executor
func NewMethodDockerExecutor(service BuildExecutionService) BuildMethodExecutor {
	return &MethodDockerExecutor{
		service: service,
		running: make(map[string]*exec.Cmd),
	}
}

// Supports checks if this executor supports Docker builds
func (e *MethodDockerExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodDocker
}

// Execute runs a Docker build
func (e *MethodDockerExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Check if docker is available
	if err := e.validateDockerAvailable(); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Docker validation failed: %v", err), nil)
		return nil, err
	}

	output := &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		StartTime:   time.Now().Unix(),
		Artifacts:   []MethodArtifact{},
	}

	e.service.AddLog(ctx, executionID, LogInfo, "Starting Docker build", nil)

	config, err := configForMethod(ctx, BuildMethodDocker)
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

	// Create docker build command
	args := []string{"build", "--tag", executionID.String()}
	if config.Dockerfile != "" {
		dockerfile := config.Dockerfile
		if !filepath.IsAbs(dockerfile) {
			dockerfile = filepath.Join(buildContext, dockerfile)
		}
		args = append(args, "--file", dockerfile)
	}
	for key, value := range config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
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

	// Stream output and handle completion
	output.Output, err = e.streamDockerBuild(ctx, executionID, cmd)
	if err != nil {
		output.Status = ExecutionFailed
		output.ErrorMessage = err.Error()
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Docker build failed: %v", err), nil)
	} else {
		output.Status = ExecutionSuccess
		e.service.AddLog(ctx, executionID, LogInfo, "Docker build completed successfully", nil)

		// Create artifact for built image
		artifacts := []MethodArtifact{
			{
				Name:        executionID.String(),
				Type:        "image",
				Size:        0,
				Path:        executionID.String(),
				Checksum:    "",
				ContentType: "application/vnd.docker.container.image.v1+json",
				CreatedAt:   time.Now().Unix(),
			},
		}
		output.Artifacts = artifacts
	}

	output.EndTime = time.Now().Unix()
	output.Duration = int(output.EndTime - output.StartTime)

	return output, nil
}

// Cancel stops a running Docker build
func (e *MethodDockerExecutor) Cancel(ctx context.Context, executionID string) error {
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

// GetStatus retrieves the current status of a Docker execution
func (e *MethodDockerExecutor) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
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

// streamDockerBuild streams output from a Docker build command
func (e *MethodDockerExecutor) streamDockerBuild(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start docker build: %w", err)
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

// validateDockerAvailable checks if Docker is installed
func (e *MethodDockerExecutor) validateDockerAvailable() error {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not installed or not in PATH: %w", err)
	}
	return nil
}
