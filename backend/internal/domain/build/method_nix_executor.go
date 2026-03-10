package build

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MethodNixExecutor implements BuildMethodExecutor for Nix builds
type MethodNixExecutor struct {
	service BuildExecutionService
	running map[string]*exec.Cmd
	mu      sync.Mutex
}

// NewMethodNixExecutor creates a new Nix executor
func NewMethodNixExecutor(service BuildExecutionService) BuildMethodExecutor {
	return &MethodNixExecutor{
		service: service,
		running: make(map[string]*exec.Cmd),
	}
}

// Supports checks if this executor supports Nix builds
func (e *MethodNixExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodNix
}

// Execute runs a Nix build
func (e *MethodNixExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Check if nix is available
	if err := e.validateNixAvailable(); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Nix validation failed: %v", err), nil)
		return nil, err
	}

	output := &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		StartTime:   time.Now().Unix(),
		Artifacts:   []MethodArtifact{},
	}

	e.service.AddLog(ctx, executionID, LogInfo, "Starting Nix build", nil)

	config, err := configForMethod(ctx, BuildMethodNix)
	if err != nil {
		return nil, err
	}
	args := []string{"--json"}
	if metadataBool(config.Metadata, "show_trace", "showTrace") {
		args = append(args, "--show-trace")
	}
	attributes := metadataStringSlice(config.Metadata, "attributes")
	flakeURI := metadataString(config.Metadata, "flake_uri", "flakeUri", "flakeURI")
	nixExpr := metadataString(config.Metadata, "nix_expression", "nixExpression")
	var tempExprFile string
	if flakeURI != "" {
		if len(attributes) > 0 {
			args = append(args, fmt.Sprintf("%s#%s", flakeURI, attributes[0]))
		} else {
			args = append(args, flakeURI)
		}
		cmd := exec.CommandContext(ctx, "nix", append([]string{"build"}, args...)...)
		output.Output, err = e.runAndStream(ctx, executionID, cmd)
		if err != nil {
			output.Status = ExecutionFailed
			output.ErrorMessage = err.Error()
			e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Nix build failed: %v", err), nil)
		} else {
			output.Status = ExecutionSuccess
			e.service.AddLog(ctx, executionID, LogInfo, "Nix build completed successfully", nil)
			output.Artifacts = e.parseNixArtifacts(output.Output)
		}
		output.EndTime = time.Now().Unix()
		output.Duration = int(output.EndTime - output.StartTime)
		return output, nil
	}
	if nixExpr == "" {
		return nil, fmt.Errorf("either nix_expression or flake_uri is required")
	}
	tempDir, err := os.MkdirTemp("", "image-factory-nix-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary nix directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	tempExprFile = filepath.Join(tempDir, "default.nix")
	if err := os.WriteFile(tempExprFile, []byte(nixExpr), 0600); err != nil {
		return nil, fmt.Errorf("failed to write nix expression: %w", err)
	}
	for _, attr := range attributes {
		args = append(args, "-A", attr)
	}
	args = append(args, tempExprFile)
	cmd := exec.CommandContext(ctx, "nix-build", args...)

	// Stream output
	output.Output, err = e.runAndStream(ctx, executionID, cmd)
	if err != nil {
		output.Status = ExecutionFailed
		output.ErrorMessage = err.Error()
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Nix build failed: %v", err), nil)
	} else {
		output.Status = ExecutionSuccess
		e.service.AddLog(ctx, executionID, LogInfo, "Nix build completed successfully", nil)

		// Parse artifacts from output
		artifacts := e.parseNixArtifacts(output.Output)
		output.Artifacts = artifacts
	}

	output.EndTime = time.Now().Unix()
	output.Duration = int(output.EndTime - output.StartTime)

	return output, nil
}

func (e *MethodNixExecutor) runAndStream(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	e.mu.Lock()
	e.running[executionID.String()] = cmd
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.running, executionID.String())
		e.mu.Unlock()
	}()
	return e.streamNixBuild(ctx, executionID, cmd)
}

// Cancel stops a running Nix build
func (e *MethodNixExecutor) Cancel(ctx context.Context, executionID string) error {
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

// GetStatus retrieves the current status of a Nix execution
func (e *MethodNixExecutor) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
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

// streamNixBuild streams output from a Nix build command
func (e *MethodNixExecutor) streamNixBuild(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start nix-build: %w", err)
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

// parseNixArtifacts extracts build artifacts from Nix output
func (e *MethodNixExecutor) parseNixArtifacts(output string) []MethodArtifact {
	artifacts := []MethodArtifact{}

	// Parse nix-build JSON output
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "/nix/store") {
			continue
		}

		// Extract nix store path as artifact
		storePath := strings.TrimSpace(line)
		artifact := MethodArtifact{
			Name:        storePath,
			Type:        "derivation",
			Size:        0,
			Path:        storePath,
			Checksum:    "",
			ContentType: "application/octet-stream",
			CreatedAt:   time.Now().Unix(),
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts
}

// validateNixAvailable checks if Nix is installed
func (e *MethodNixExecutor) validateNixAvailable() error {
	cmd := exec.Command("nix-build", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nix-build not installed or not in PATH: %w", err)
	}
	return nil
}
