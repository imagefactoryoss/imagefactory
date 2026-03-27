package build

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MethodPackerExecutor implements BuildMethodExecutor for Packer builds
type MethodPackerExecutor struct {
	service BuildExecutionService
	running map[string]*exec.Cmd
	mu      sync.Mutex
}

// NewMethodPackerExecutor creates a new Packer executor for build methods
func NewMethodPackerExecutor(service BuildExecutionService) BuildMethodExecutor {
	return &MethodPackerExecutor{
		service: service,
		running: make(map[string]*exec.Cmd),
	}
}

// Supports checks if this executor supports Packer builds
func (e *MethodPackerExecutor) Supports(method BuildMethod) bool {
	return method == BuildMethodPacker
}

// Execute runs a Packer build
func (e *MethodPackerExecutor) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	if !e.Supports(method) {
		return nil, fmt.Errorf("executor does not support method: %s", method)
	}

	// Parse execution ID from context or config
	executionID, err := ResolveExecutionID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Check if packer is installed
	if err := e.validatePackerInstalled(); err != nil {
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Packer validation failed: %v", err), nil)
		return nil, err
	}

	// Build output container
	output := &MethodExecutionOutput{
		ExecutionID: executionID.String(),
		Status:      ExecutionRunning,
		StartTime:   time.Now().Unix(),
		Artifacts:   []MethodArtifact{},
	}

	// Log start
	e.service.AddLog(ctx, executionID, LogInfo, "Starting Packer build", nil)

	config, err := configForMethod(ctx, BuildMethodPacker)
	if err != nil {
		return nil, err
	}
	template := config.PackerTemplate
	if template == "" {
		return nil, fmt.Errorf("packer template is required")
	}
	templatePath := template
	cleanupDir := ""
	if _, statErr := os.Stat(template); statErr != nil {
		tempDir, mkErr := os.MkdirTemp("", "image-factory-packer-*")
		if mkErr != nil {
			return nil, fmt.Errorf("failed to create temporary packer directory: %w", mkErr)
		}
		cleanupDir = tempDir
		templatePath = filepath.Join(tempDir, "template.pkr.hcl")
		if writeErr := os.WriteFile(templatePath, []byte(template), 0600); writeErr != nil {
			_ = os.RemoveAll(tempDir)
			return nil, fmt.Errorf("failed to write temporary packer template: %w", writeErr)
		}
	}
	if cleanupDir != "" {
		defer os.RemoveAll(cleanupDir)
	}

	args := []string{"build", "-json"}
	if metadataBool(config.Metadata, "parallel") {
		return nil, fmt.Errorf("parallel=true is not supported yet for packer builds")
	}
	if onError := metadataString(config.Metadata, "on_error", "onError"); onError != "" {
		args = append(args, "-on-error="+onError)
	}
	args = append(args, packerVarArgs(config.Metadata)...)
	args = append(args, templatePath)
	cmd := exec.CommandContext(ctx, "packer", args...)
	cmd.Dir = filepath.Dir(templatePath)

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
	output.Output, err = e.streamPackerBuild(ctx, executionID, cmd)
	if err != nil {
		output.Status = ExecutionFailed
		output.ErrorMessage = err.Error()
		e.service.AddLog(ctx, executionID, LogError, fmt.Sprintf("Packer build failed: %v", err), nil)
	} else {
		output.Status = ExecutionSuccess
		e.service.AddLog(ctx, executionID, LogInfo, "Packer build completed successfully", nil)

		// Parse artifacts from output
		artifacts, err := e.parsePackerArtifacts(output.Output)
		if err == nil {
			output.Artifacts = artifacts
		}
	}

	output.EndTime = time.Now().Unix()
	output.Duration = int(output.EndTime - output.StartTime)

	return output, nil
}

func packerVarArgs(metadata map[string]interface{}) []string {
	if metadata == nil {
		return nil
	}

	values := map[string]string{}
	if rawVars, ok := metadata["variables"].(map[string]interface{}); ok {
		for key, value := range rawVars {
			if strings.TrimSpace(key) == "" {
				continue
			}
			values[key] = fmt.Sprintf("%v", value)
		}
	}
	for key, value := range metadataStringMap(metadata, "build_vars") {
		if strings.TrimSpace(key) == "" {
			continue
		}
		values[key] = value
	}
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		args = append(args, "-var", fmt.Sprintf("%s=%s", key, values[key]))
	}
	return args
}

// Cancel stops a running Packer build
func (e *MethodPackerExecutor) Cancel(ctx context.Context, executionID string) error {
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

// GetStatus retrieves the current status of a Packer execution
func (e *MethodPackerExecutor) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
	e.mu.Lock()
	cmd, ok := e.running[executionID]
	e.mu.Unlock()

	if !ok {
		return "", ErrExecutionNotFound
	}

	// If command still in running map, it's running
	if cmd.ProcessState == nil {
		return ExecutionRunning, nil
	}

	if cmd.ProcessState.Success() {
		return ExecutionSuccess, nil
	}

	return ExecutionFailed, nil
}

// streamPackerBuild streams output from a Packer build command
func (e *MethodPackerExecutor) streamPackerBuild(ctx context.Context, executionID uuid.UUID, cmd *exec.Cmd) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start packer: %w", err)
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

			// Log to service
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

			// Log as error level
			e.service.AddLog(ctx, executionID, LogError, line, nil)
		}
	}()

	// Wait for completion
	err = cmd.Wait()
	wg.Wait()

	return output.String(), err
}

// parsePackerArtifacts extracts artifacts from Packer output
func (e *MethodPackerExecutor) parsePackerArtifacts(output string) ([]MethodArtifact, error) {
	artifacts := []MethodArtifact{}

	// Parse JSON lines from packer build -json output
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "artifact") {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		// Extract artifact information from packer JSON output
		if artifact, ok := data["artifact"].(map[string]interface{}); ok {
			artifactID := ""
			if id, ok := artifact["id"].(string); ok {
				artifactID = id
			}

			if artifactType, ok := artifact["type"].(string); ok {
				artifact := MethodArtifact{
					Name:        artifactID,
					Type:        artifactType,
					Size:        0,
					Path:        "",
					Checksum:    "",
					ContentType: "application/octet-stream",
					CreatedAt:   time.Now().Unix(),
				}

				// Try to determine size from artifact
				if artifactID != "" {
					if info, err := os.Stat(artifactID); err == nil {
						artifact.Size = info.Size()
					}
				}

				artifacts = append(artifacts, artifact)
			}
		}
	}

	return artifacts, nil
}

// validatePackerInstalled checks if Packer is installed and accessible
func (e *MethodPackerExecutor) validatePackerInstalled() error {
	cmd := exec.Command("packer", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("packer not installed or not in PATH: %w", err)
	}
	return nil
}
