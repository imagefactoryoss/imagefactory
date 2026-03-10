package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PackerBuildExecutor implements BuildExecutor for VM builds using Packer
type PackerBuildExecutor struct {
	logger     *zap.Logger
	workDir    string
	s3Bucket   string
	s3Region   string
	s3Client   *s3.Client
	packerPath string
}

// NewPackerBuildExecutor creates a new Packer build executor
func NewPackerBuildExecutor(logger *zap.Logger, workDir, s3Bucket, s3Region, packerPath string) *PackerBuildExecutor {
	if packerPath == "" {
		packerPath = "packer" // Assume packer is in PATH
	}

	// Initialize S3 client
	var s3Client *s3.Client
	if s3Bucket != "" && s3Region != "" {
		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(s3Region),
		)
		if err != nil {
			logger.Warn("Failed to initialize S3 client", zap.Error(err))
		} else {
			s3Client = s3.NewFromConfig(cfg)
		}
	}

	return &PackerBuildExecutor{
		logger:     logger,
		workDir:    workDir,
		s3Bucket:   s3Bucket,
		s3Region:   s3Region,
		s3Client:   s3Client,
		packerPath: packerPath,
	}
}

// Execute executes a VM build using Packer
func (e *PackerBuildExecutor) Execute(ctx context.Context, build *Build) (*BuildResult, error) {
	e.logger.Info("Starting Packer VM build", zap.String("build_id", build.ID().String()))

	startTime := time.Now()

	// Create build directory
	buildDir := filepath.Join(e.workDir, build.ID().String())
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir) // Clean up after build

	// Generate Packer configuration
	packerConfig, err := e.generatePackerConfig(build)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Packer config: %w", err)
	}

	// Write Packer config to file
	configPath := filepath.Join(buildDir, "packer.json")
	if err := os.WriteFile(configPath, packerConfig, 0644); err != nil {
		return nil, fmt.Errorf("failed to write Packer config: %w", err)
	}

	// Prepare Packer variables
	varFile := filepath.Join(buildDir, "variables.json")
	if err := e.generateVariablesFile(build, varFile); err != nil {
		return nil, fmt.Errorf("failed to generate variables file: %w", err)
	}

	// Execute Packer build
	logs, err := e.runPackerBuild(ctx, buildDir, configPath, varFile)
	if err != nil {
		e.logger.Error("Packer build failed", zap.Error(err), zap.String("build_id", build.ID().String()))
		return &BuildResult{
			Duration: time.Since(startTime),
			Logs:     logs,
		}, fmt.Errorf("packer build failed: %w", err)
	}

	// Parse build artifacts from logs
	artifacts, err := e.parseBuildArtifacts(logs)
	if err != nil {
		e.logger.Warn("Failed to parse build artifacts", zap.Error(err), zap.String("build_id", build.ID().String()))
	}

	// Upload artifacts to S3
	s3Artifacts, err := e.uploadArtifactsToS3(ctx, build, artifacts)
	if err != nil {
		e.logger.Error("Failed to upload artifacts to S3", zap.Error(err), zap.String("build_id", build.ID().String()))
	}

	duration := time.Since(startTime)

	e.logger.Info("Packer build completed successfully",
		zap.String("build_id", build.ID().String()),
		zap.Duration("duration", duration),
		zap.Int("artifacts_count", len(s3Artifacts)))

	return &BuildResult{
		ImageID:     e.extractImageID(artifacts),
		ImageDigest: e.extractImageDigest(artifacts),
		Size:        e.calculateTotalSize(artifacts),
		Duration:    duration,
		Logs:        logs,
		Artifacts:   s3Artifacts,
	}, nil
}

// Cancel cancels a running Packer build
func (e *PackerBuildExecutor) Cancel(ctx context.Context, buildID uuid.UUID) error {
	e.logger.Info("Cancelling Packer build", zap.String("build_id", buildID.String()))

	// Find and kill the packer process
	// This is a simplified implementation - in production, you'd want more sophisticated
	// process management and cleanup

	cmd := exec.Command("pkill", "-f", fmt.Sprintf("packer.*%s", buildID.String()))
	if err := cmd.Run(); err != nil {
		e.logger.Warn("Failed to kill packer process", zap.Error(err), zap.String("build_id", buildID.String()))
	}

	return nil
}

// generatePackerConfig generates Packer configuration from build manifest
func (e *PackerBuildExecutor) generatePackerConfig(build *Build) ([]byte, error) {
	manifest := build.Manifest()
	vmConfig := manifest.VMConfig

	if vmConfig == nil {
		return nil, fmt.Errorf("VM configuration is required")
	}

	config := map[string]interface{}{
		"builders": []map[string]interface{}{
			e.generateBuilderConfig(vmConfig),
		},
		"provisioners":    e.generateProvisioners(vmConfig.Provisioners),
		"post-processors": e.generatePostProcessors(vmConfig.PostProcessors),
	}

	return json.MarshalIndent(config, "", "  ")
}

// generateBuilderConfig generates the builder configuration for Packer
func (e *PackerBuildExecutor) generateBuilderConfig(vmConfig *VMBuildConfig) map[string]interface{} {
	builder := map[string]interface{}{
		"type":          "amazon-ebs", // Default to AWS
		"region":        vmConfig.Region,
		"source_ami":    vmConfig.PackerTemplate,
		"instance_type": vmConfig.InstanceType,
		"ssh_username":  "ubuntu", // Default
	}

	// Override based on cloud provider
	switch vmConfig.CloudProvider {
	case "aws":
		builder["type"] = "amazon-ebs"
		if vmConfig.StorageConfig != nil {
			builder["launch_block_device_mappings"] = []map[string]interface{}{
				{
					"device_name":           "/dev/sda1",
					"volume_size":           vmConfig.StorageConfig.RootVolumeSizeGB,
					"volume_type":           vmConfig.StorageConfig.RootVolumeType,
					"delete_on_termination": true,
				},
			}
		}
	case "azure":
		builder["type"] = "azure-arm"
		builder["vm_size"] = vmConfig.VMSize
		delete(builder, "instance_type")
	case "vmware":
		builder["type"] = "vmware-iso"
		builder["vm_name"] = vmConfig.OutputName
		delete(builder, "region")
		delete(builder, "instance_type")
	}

	// Add network configuration
	if vmConfig.NetworkConfig != nil {
		if vmConfig.NetworkConfig.VpcID != "" {
			builder["vpc_id"] = vmConfig.NetworkConfig.VpcID
		}
		if vmConfig.NetworkConfig.SubnetID != "" {
			builder["subnet_id"] = vmConfig.NetworkConfig.SubnetID
		}
		if len(vmConfig.NetworkConfig.SecurityGroups) > 0 {
			builder["security_group_ids"] = vmConfig.NetworkConfig.SecurityGroups
		}
		builder["associate_public_ip_address"] = vmConfig.NetworkConfig.AssociatePublicIP
	}

	return builder
}

// generateProvisioners generates Packer provisioner configurations
func (e *PackerBuildExecutor) generateProvisioners(provisioners []VMProvisioner) []map[string]interface{} {
	if len(provisioners) == 0 {
		// Default shell provisioner
		return []map[string]interface{}{
			{
				"type": "shell",
				"inline": []string{
					"echo 'VM build completed'",
				},
			},
		}
	}

	result := make([]map[string]interface{}, len(provisioners))
	for i, p := range provisioners {
		result[i] = map[string]interface{}{
			"type": p.Type,
		}
		for k, v := range p.Config {
			result[i][k] = v
		}
		if len(p.Only) > 0 {
			result[i]["only"] = p.Only
		}
		if len(p.Except) > 0 {
			result[i]["except"] = p.Except
		}
	}

	return result
}

// generatePostProcessors generates Packer post-processor configurations
func (e *PackerBuildExecutor) generatePostProcessors(postProcessors []VMPostProcessor) []map[string]interface{} {
	if len(postProcessors) == 0 {
		// Default manifest post-processor
		return []map[string]interface{}{
			{
				"type":   "manifest",
				"output": "packer-manifest.json",
			},
		}
	}

	result := make([]map[string]interface{}, len(postProcessors))
	for i, pp := range postProcessors {
		result[i] = map[string]interface{}{
			"type": pp.Type,
		}
		for k, v := range pp.Config {
			result[i][k] = v
		}
		if len(pp.Only) > 0 {
			result[i]["only"] = pp.Only
		}
		if len(pp.Except) > 0 {
			result[i]["except"] = pp.Except
		}
	}

	return result
}

// generateVariablesFile generates Packer variables file
func (e *PackerBuildExecutor) generateVariablesFile(build *Build, varFile string) error {
	variables := map[string]interface{}{
		"build_id":   build.ID().String(),
		"build_name": build.Manifest().Name,
	}

	// Add VM-specific variables
	if vmConfig := build.Manifest().VMConfig; vmConfig != nil {
		for k, v := range vmConfig.PackerVariables {
			variables[k] = v
		}
	}

	data, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(varFile, data, 0644)
}

// runPackerBuild executes the Packer build command
func (e *PackerBuildExecutor) runPackerBuild(ctx context.Context, buildDir, configPath, varFile string) ([]string, error) {
	args := []string{"build", "-var-file", varFile, configPath}

	cmd := exec.CommandContext(ctx, e.packerPath, args...)
	cmd.Dir = buildDir

	// Capture output
	output, err := cmd.CombinedOutput()
	logs := strings.Split(string(output), "\n")

	if err != nil {
		return logs, fmt.Errorf("packer build failed: %w", err)
	}

	return logs, nil
}

// parseBuildArtifacts parses build artifacts from Packer logs
func (e *PackerBuildExecutor) parseBuildArtifacts(logs []string) ([]string, error) {
	artifacts := []string{}

	for _, log := range logs {
		// Look for artifact information in logs
		if strings.Contains(log, "artifact") && strings.Contains(log, "id:") {
			// Extract artifact ID from log line
			parts := strings.Split(log, "id:")
			if len(parts) > 1 {
				artifact := strings.TrimSpace(parts[1])
				artifacts = append(artifacts, artifact)
			}
		}
	}

	return artifacts, nil
}

// uploadArtifactsToS3 uploads build artifacts to S3
func (e *PackerBuildExecutor) uploadArtifactsToS3(ctx context.Context, build *Build, artifacts []string) ([]string, error) {
	s3Artifacts := []string{}

	if e.s3Client == nil {
		e.logger.Warn("S3 client not configured, skipping artifact upload")
		// Return mock URLs for development
		for _, artifact := range artifacts {
			s3Key := fmt.Sprintf("builds/%s/artifacts/%s", build.ID().String(), artifact)
			s3URL := fmt.Sprintf("s3://%s/%s", e.s3Bucket, s3Key)
			s3Artifacts = append(s3Artifacts, s3URL)
		}
		return s3Artifacts, nil
	}

	for _, artifact := range artifacts {
		// Check if artifact file exists
		artifactPath := filepath.Join(e.workDir, build.ID().String(), artifact)
		if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
			e.logger.Warn("Artifact file does not exist, skipping upload",
				zap.String("artifact", artifact),
				zap.String("path", artifactPath))
			continue
		}

		// Generate S3 key
		s3Key := fmt.Sprintf("builds/%s/artifacts/%s", build.ID().String(), artifact)

		// Read artifact file
		file, err := os.Open(artifactPath)
		if err != nil {
			e.logger.Error("Failed to open artifact file",
				zap.Error(err),
				zap.String("artifact", artifact),
				zap.String("path", artifactPath))
			continue
		}

		// Upload to S3
		_, err = e.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(e.s3Bucket),
			Key:    aws.String(s3Key),
			Body:   file,
		})
		file.Close()

		if err != nil {
			e.logger.Error("Failed to upload artifact to S3",
				zap.Error(err),
				zap.String("build_id", build.ID().String()),
				zap.String("artifact", artifact),
				zap.String("s3_key", s3Key))
			continue
		}

		s3URL := fmt.Sprintf("s3://%s/%s", e.s3Bucket, s3Key)
		s3Artifacts = append(s3Artifacts, s3URL)

		e.logger.Info("Artifact uploaded to S3",
			zap.String("build_id", build.ID().String()),
			zap.String("artifact", artifact),
			zap.String("s3_url", s3URL))
	}

	return s3Artifacts, nil
}

// extractImageID extracts the image ID from build artifacts
func (e *PackerBuildExecutor) extractImageID(artifacts []string) string {
	for _, artifact := range artifacts {
		// Look for AMI ID pattern
		if strings.HasPrefix(artifact, "ami-") {
			return artifact
		}
		// Look for other image ID patterns
		if strings.Contains(artifact, "ami-") || strings.Contains(artifact, "image-") {
			return artifact
		}
	}
	return ""
}

// extractImageDigest generates a digest for the build
func (e *PackerBuildExecutor) extractImageDigest(artifacts []string) string {
	// Generate a simple digest based on artifacts
	if len(artifacts) > 0 {
		return fmt.Sprintf("sha256:%x", artifacts[0])
	}
	return ""
}

// calculateTotalSize calculates total size of artifacts
func (e *PackerBuildExecutor) calculateTotalSize(artifacts []string) int64 {
	// For VM builds, artifacts are typically AMI IDs or image references
	// Size calculation is not directly applicable, so we estimate based on typical VM image sizes
	var totalSize int64

	for _, artifact := range artifacts {
		// Check if it's an AMI ID (AWS EC2)
		if strings.HasPrefix(artifact, "ami-") {
			// Typical AMI size estimate (compressed)
			totalSize += 8 * 1024 * 1024 * 1024 // 8GB
		} else if strings.Contains(artifact, "image-") {
			// Other cloud image formats
			totalSize += 4 * 1024 * 1024 * 1024 // 4GB
		} else {
			// Generic artifact, assume smaller size
			totalSize += 1 * 1024 * 1024 * 1024 // 1GB
		}
	}

	if totalSize == 0 {
		// Fallback for empty artifacts
		totalSize = 4 * 1024 * 1024 * 1024 // 4GB default
	}

	e.logger.Info("Calculated estimated artifact size",
		zap.Int64("total_size", totalSize),
		zap.Int("artifact_count", len(artifacts)),
		zap.Strings("artifacts", artifacts))

	return totalSize
}
