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

// MethodKanikoExecutor implements BuildMethodExecutor for Kaniko container builds
type MethodKanikoExecutor struct {
	service BuildExecutionService
	running map[string]*exec.Cmd
	mu      sync.Mutex
}

// NewMethodKanikoExecutor creates a new Kaniko executor
func NewMethodKanikoExecutor(service BuildExecutionService) BuildMethodExecutor {
	return &MethodKanikoExecutor{
		service: service,
		running: make(map[string]*exec.Cmd),
	}
}

// Supports checks if this executor supports Kaniko builds
func (e *MethodKanikoExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodKaniko
}

// Execute runs a Kaniko container build
func (e *MethodKanikoExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Check if kaniko or docker is available
	if err := e.validateKanikoAvailable(); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Kaniko validation failed: %v", err), nil)
		return nil, err
	}

	output := &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		StartTime:   time.Now().Unix(),
		Artifacts:   []MethodArtifact{},
	}

	e.service.AddLog(ctx, executionID, LogInfo, "Starting Kaniko container build", nil)

	config, err := configForMethod(ctx, BuildMethodKaniko)
	if err != nil {
		return nil, err
	}
	registryRepo := metadataString(config.Metadata, "registry_repo", "registryRepo")
	if registryRepo == "" {
		return nil, fmt.Errorf("registry_repo is required for kaniko builds")
	}
	buildContext := config.BuildContext
	if buildContext == "" {
		buildContext = "."
	}
	buildContext, err = filepath.Abs(buildContext)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve build context: %w", err)
	}
	dockerfile := config.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	if filepath.IsAbs(dockerfile) {
		if rel, relErr := filepath.Rel(buildContext, dockerfile); relErr == nil && !strings.HasPrefix(rel, "..") {
			dockerfile = rel
		} else {
			dockerfile = filepath.Base(dockerfile)
		}
	}

	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/workspace", buildContext),
		"gcr.io/kaniko-project/executor:latest",
		"--dockerfile=" + dockerfile,
		"--context=dir:///workspace",
		"--destination=" + registryRepo,
	}
	if config.CacheRepo != "" {
		args = append(args, "--cache=true", "--cache-repo="+config.CacheRepo)
	}
	if metadataBool(config.Metadata, "skip_unused_stages", "skipUnusedStages") {
		args = append(args, "--skip-unused-stages")
	}
	for key, value := range config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}
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
	output.Output, err = e.streamKanikoBuild(ctx, executionID, cmd)
	if err != nil {
		output.Status = ExecutionFailed
		output.ErrorMessage = err.Error()
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Kaniko build failed: %v", err), nil)
	} else {
		output.Status = ExecutionSuccess
		e.service.AddLog(ctx, executionID, LogInfo, "Kaniko build completed successfully", nil)

		// Parse artifacts
		artifacts := e.parseKanikoArtifacts(output.Output)
		output.Artifacts = artifacts
	}

	output.EndTime = time.Now().Unix()
	output.Duration = int(output.EndTime - output.StartTime)

	return output, nil
}

// Cancel stops a running Kaniko build
func (e *MethodKanikoExecutor) Cancel(ctx context.Context, executionID string) error {
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

// GetStatus retrieves the current status of a Kaniko execution
func (e *MethodKanikoExecutor) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
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

// streamKanikoBuild streams output from a Kaniko build command
func (e *MethodKanikoExecutor) streamKanikoBuild(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start kaniko: %w", err)
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

// parseKanikoArtifacts extracts container artifacts from Kaniko output
func (e *MethodKanikoExecutor) parseKanikoArtifacts(output string) []MethodArtifact {
	artifacts := []MethodArtifact{}

	// Parse kaniko output for image references
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "Pushing image to") && !strings.Contains(line, "digest") {
			continue
		}

		// Extract image reference from output
		if strings.Contains(line, "Pushing image to") {
			parts := strings.Fields(line)
			if len(parts) > 3 {
				imageRef := strings.TrimSpace(strings.Join(parts[3:], " "))
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
	}

	return artifacts
}

// validateKanikoAvailable checks if Kaniko is available
func (e *MethodKanikoExecutor) validateKanikoAvailable() error {
	// Check if docker is available (for running kaniko container)
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not installed (required for kaniko): %w", err)
	}
	return nil
}
