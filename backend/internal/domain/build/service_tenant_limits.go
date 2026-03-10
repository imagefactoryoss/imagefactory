package build

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) validateTektonAllowed(ctx context.Context, tenantID uuid.UUID) error {
	if s.systemConfigService == nil {
		return fmt.Errorf("system configuration service unavailable")
	}

	cfg, err := s.systemConfigService.GetBuildConfig(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to read build configuration: %w", err)
	}
	if cfg == nil || !cfg.TektonEnabled {
		return fmt.Errorf("tekton is disabled for this tenant")
	}
	return nil
}

func (s *Service) validateTektonExecutable(ctx context.Context, tenantID uuid.UUID) error {
	if err := s.validateTektonAllowed(ctx, tenantID); err != nil {
		return err
	}
	if s.tektonExecutorFactory == nil {
		return fmt.Errorf("tekton executor is not configured on this server")
	}
	return nil
}

// validateBuildLimits validates build creation against tenant-specific limits.
func (s *Service) validateBuildLimits(ctx context.Context, tenantID uuid.UUID) error {
	if s.systemConfigService == nil {
		// If no system config service, allow build (backward compatibility)
		return nil
	}

	buildConfig, err := s.systemConfigService.GetBuildConfig(ctx, tenantID)
	if err != nil {
		s.logger.Warn("Failed to get build config, allowing build", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil // Allow build on config error
	}

	// Check concurrent builds limit
	runningCount, err := s.repository.CountByStatus(ctx, tenantID, BuildStatusRunning)
	if err != nil {
		s.logger.Error("Failed to count running builds", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return fmt.Errorf("failed to validate concurrent build limit: %w", err)
	}

	if runningCount >= buildConfig.MaxConcurrentJobs {
		return fmt.Errorf("maximum concurrent builds limit reached (%d/%d)", runningCount, buildConfig.MaxConcurrentJobs)
	}

	// Check queued builds limit
	queuedCount, err := s.repository.CountByStatus(ctx, tenantID, BuildStatusQueued)
	if err != nil {
		s.logger.Error("Failed to count queued builds", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return fmt.Errorf("failed to validate queue limit: %w", err)
	}

	if queuedCount >= buildConfig.MaxQueueSize {
		return fmt.Errorf("build queue limit reached (%d/%d)", queuedCount, buildConfig.MaxQueueSize)
	}

	return nil
}
