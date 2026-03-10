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

// MethodPaketoExecutor executes Paketo builds via the `pack` CLI.
type MethodPaketoExecutor struct {
	service BuildExecutionService
	running map[string]*exec.Cmd
	mu      sync.Mutex
}

// NewMethodPaketoExecutor creates a new Paketo executor.
func NewMethodPaketoExecutor(service BuildExecutionService) BuildMethodExecutor {
	return &MethodPaketoExecutor{
		service: service,
		running: make(map[string]*exec.Cmd),
	}
}

// Supports checks if this executor supports Paketo builds.
func (e *MethodPaketoExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodPaketo
}

// Execute runs a Paketo build.
func (e *MethodPaketoExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	if err := e.validatePackAvailable(); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Paketo validation failed: %v", err), nil)
		return nil, err
	}

	output := &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		StartTime:   time.Now().Unix(),
		Artifacts:   []MethodArtifact{},
	}

	e.service.AddLog(ctx, executionID, LogInfo, "Starting Paketo build", nil)

	config, err := configForMethod(ctx, BuildMethodPaketo)
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
	if config.Builder == "" {
		return nil, fmt.Errorf("builder is required for paketo builds")
	}
	imageName := executionID.String()
	args := []string{"build", imageName, "--path", buildContext, "--builder", config.Builder}
	for _, buildpack := range config.Buildpacks {
		args = append(args, "--buildpack", buildpack)
	}
	if env := metadataStringMap(config.Metadata, "env"); env != nil {
		for key, value := range env {
			args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
		}
	}
	if buildArgs := metadataStringMap(config.Metadata, "build_args"); buildArgs != nil {
		for key, value := range buildArgs {
			args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
		}
	}
	cmd := exec.CommandContext(ctx, "pack", args...)

	e.mu.Lock()
	e.running[executionID.String()] = cmd
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.running, executionID.String())
		e.mu.Unlock()
	}()

	output.Output, err = e.streamPackBuild(ctx, executionID, cmd)
	if err != nil {
		output.Status = ExecutionFailed
		output.ErrorMessage = err.Error()
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Paketo build failed: %v", err), nil)
	} else {
		output.Status = ExecutionSuccess
		e.service.AddLog(ctx, executionID, LogInfo, "Paketo build completed successfully", nil)
		output.Artifacts = []MethodArtifact{
			{
				Name:        imageName,
				Type:        "image",
				Path:        imageName,
				ContentType: "application/vnd.docker.container.image.v1+json",
				CreatedAt:   time.Now().Unix(),
			},
		}
	}

	output.EndTime = time.Now().Unix()
	output.Duration = int(output.EndTime - output.StartTime)

	return output, nil
}

// Cancel stops a running Paketo build.
func (e *MethodPaketoExecutor) Cancel(ctx context.Context, executionID string) error {
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

// GetStatus retrieves the current status of a Paketo execution.
func (e *MethodPaketoExecutor) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
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

func (e *MethodPaketoExecutor) streamPackBuild(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start pack build: %w", err)
	}

	var output strings.Builder
	var wg sync.WaitGroup

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

	err = cmd.Wait()
	wg.Wait()
	return output.String(), err
}

func (e *MethodPaketoExecutor) validatePackAvailable() error {
	cmd := exec.Command("pack", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pack CLI not installed or not in PATH: %w", err)
	}
	return nil
}
