package build

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MarkBuildFailed marks a non-terminal build as failed and completes non-terminal executions.
func (s *Service) MarkBuildFailed(ctx context.Context, buildID uuid.UUID, reason string) error {
	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		return fmt.Errorf("failed to find build: %w", err)
	}
	if build == nil {
		return ErrBuildNotFound
	}
	if build.IsTerminal() {
		return nil
	}
	if reason == "" {
		reason = "build failed by control plane"
	}

	if err := build.Fail(reason); err != nil {
		return fmt.Errorf("failed to transition build to failed: %w", err)
	}
	if err := s.repository.Update(ctx, build); err != nil {
		return fmt.Errorf("failed to persist failed build state: %w", err)
	}

	executions, _, execErr := s.executionService.GetBuildExecutions(ctx, buildID, 25, 0)
	if execErr != nil {
		s.logger.Warn("Failed to list build executions while marking build failed", zap.String("build_id", buildID.String()), zap.Error(execErr))
		return nil
	}
	for _, execution := range executions {
		if execution.Status.IsTerminalStatus() {
			continue
		}
		if completeErr := s.executionService.CompleteExecution(ctx, execution.ID, false, reason, nil); completeErr != nil {
			s.logger.Warn("Failed to complete execution while marking build failed",
				zap.String("build_id", buildID.String()),
				zap.String("execution_id", execution.ID.String()),
				zap.Error(completeErr))
		}
	}
	return nil
}

// StartBuild starts execution of a build.
func (s *Service) StartBuild(ctx context.Context, buildID uuid.UUID) error {
	s.logger.Info("Starting build", zap.String("build_id", buildID.String()))

	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to find build for starting", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to find build: %w", err)
	}

	if build == nil {
		s.logger.Warn("Build not found for starting", zap.String("build_id", buildID.String()))
		return ErrBuildNotFound
	}

	if err := s.applyRepoManagedBuildConfig(ctx, build); err != nil {
		if s.shouldFallbackToUIOnRepoConfigError(ctx, build) {
			s.recordRepoConfigFallback(ctx, build, "start", err)
		} else {
			s.recordRepoConfigFailure(ctx, build, "start", err)
			s.publishRepoConfigFailureStatus(ctx, build, "start", err, false)
			s.logger.Warn("Repository-managed build config validation failed",
				zap.Error(err),
				zap.String("build_id", buildID.String()),
				zap.String("stage", "start"))
			return err
		}
	}
	if err := s.validatePackerTargetProfileForBuildConfig(ctx, build); err != nil {
		s.logger.Warn("Packer target profile preflight failed",
			zap.Error(err),
			zap.String("build_id", buildID.String()))
		return err
	}

	if build.Status() == BuildStatusRunning {
		s.logger.Info("Build already running; skipping start", zap.String("build_id", buildID.String()))
		return nil
	}

	if err := build.Start(); err != nil {
		s.logger.Error("Failed to start build", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to start build: %w", err)
	}

	if err := s.repository.Update(ctx, build); err != nil {
		s.logger.Error("Failed to update build after starting", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to update build: %w", err)
	}

	s.dispatchBuild(ctx, build, false)

	s.logger.Info("Build started successfully", zap.String("build_id", buildID.String()))
	return nil
}

// DispatchBuild starts execution for a build already marked as running.
// Used by the dispatcher after claiming a queued build.
func (s *Service) DispatchBuild(ctx context.Context, build *Build) error {
	if build == nil {
		return errors.New("build is required")
	}
	if build.Status() != BuildStatusRunning {
		return fmt.Errorf("build must be running to dispatch (status=%s)", build.Status())
	}
	if err := s.validateBuildCapabilities(ctx, build.TenantID(), build.Manifest()); err != nil {
		return err
	}
	if err := s.validatePackerTargetProfileForBuildConfig(ctx, build); err != nil {
		return err
	}

	s.dispatchBuild(ctx, build, false)
	return nil
}

// CancelBuild cancels a running build.
func (s *Service) CancelBuild(ctx context.Context, buildID uuid.UUID) error {
	s.logger.Info("Cancelling build", zap.String("build_id", buildID.String()))

	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to find build for cancelling", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to find build: %w", err)
	}

	if build == nil {
		s.logger.Warn("Build not found for cancelling", zap.String("build_id", buildID.String()))
		return ErrBuildNotFound
	}

	if err := build.Cancel(); err != nil {
		s.logger.Error("Failed to cancel build", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to cancel build: %w", err)
	}

	if err := s.repository.Update(ctx, build); err != nil {
		s.logger.Error("Failed to update build after cancelling", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to update build: %w", err)
	}

	manifest := build.Manifest()
	var executor BuildExecutor
	switch manifest.Type {
	case BuildTypeVM:
		executor = s.vmExecutor
	case BuildTypeContainer, BuildTypeCloud, BuildTypePacker, BuildTypePaketo, BuildTypeKaniko, BuildTypeBuildx, BuildTypeNix:
		executor = s.containerExecutor
	default:
		s.logger.Warn("Unsupported build type for cancellation", zap.String("build_type", string(manifest.Type)), zap.String("build_id", buildID.String()))
		executor = s.containerExecutor
	}

	if err := executor.Cancel(ctx, buildID); err != nil {
		s.logger.Warn("Failed to cancel build execution", zap.Error(err), zap.String("build_id", buildID.String()))
	}

	cancelledResult := BuildResult{Logs: []string{"Build cancelled by user"}}
	event := NewBuildCompleted(build.ID(), build.TenantID(), cancelledResult)
	if err := s.eventPublisher.PublishBuildCompleted(ctx, event); err != nil {
		s.logger.Error("Failed to publish build cancelled event", zap.Error(err), zap.String("build_id", buildID.String()))
	}

	s.logger.Info("Build cancelled successfully", zap.String("build_id", buildID.String()))
	return nil
}

// RetryBuild restarts a failed/cancelled build under the same build ID.
// Each retry produces a new execution attempt in build_executions.
func (s *Service) RetryBuild(ctx context.Context, buildID uuid.UUID) error {
	s.logger.Info("Retrying build", zap.String("build_id", buildID.String()))

	b, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to find build for retry", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to find build: %w", err)
	}
	if b == nil {
		s.logger.Warn("Build not found for retry", zap.String("build_id", buildID.String()))
		return ErrBuildNotFound
	}
	if err := s.validateBuildCapabilities(ctx, b.TenantID(), b.Manifest()); err != nil {
		s.logger.Warn("Build capability entitlement validation failed on retry",
			zap.Error(err),
			zap.String("build_id", buildID.String()),
			zap.String("tenant_id", b.TenantID().String()))
		return err
	}
	if err := s.applyRepoManagedBuildConfig(ctx, b); err != nil {
		if s.shouldFallbackToUIOnRepoConfigError(ctx, b) {
			s.recordRepoConfigFallback(ctx, b, "retry", err)
		} else {
			s.recordRepoConfigFailure(ctx, b, "retry", err)
			s.publishRepoConfigFailureStatus(ctx, b, "retry", err, true)
			s.logger.Warn("Repository-managed build config validation failed on retry",
				zap.Error(err),
				zap.String("build_id", buildID.String()),
				zap.String("stage", "retry"))
			return err
		}
	}
	if err := s.validatePackerTargetProfileForBuildConfig(ctx, b); err != nil {
		s.logger.Warn("Packer target profile preflight failed on retry",
			zap.Error(err),
			zap.String("build_id", buildID.String()))
		return err
	}

	if err := b.RetryStart(); err != nil {
		return err
	}

	if err := s.repository.Update(ctx, b); err != nil {
		s.logger.Error("Failed to update build after retry start", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to update build: %w", err)
	}

	statusEvent := NewBuildStatusUpdated(b.ID(), b.TenantID(), "retry_started", "Build retry started", map[string]interface{}{
		"build_id": b.ID().String(),
	})
	if err := s.eventPublisher.PublishBuildStatusUpdated(ctx, statusEvent); err != nil {
		s.logger.Warn("Failed to publish build retry started event", zap.Error(err), zap.String("build_id", b.ID().String()))
	}

	s.dispatchBuild(ctx, b, true)
	s.logger.Info("Build retry started", zap.String("build_id", buildID.String()))
	return nil
}

// DeleteBuild removes a build and all cascaded child records.
// Running or queued builds cannot be deleted.
func (s *Service) DeleteBuild(ctx context.Context, buildID uuid.UUID) error {
	s.logger.Info("Deleting build", zap.String("build_id", buildID.String()))

	b, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to find build for deletion", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to find build: %w", err)
	}
	if b == nil {
		s.logger.Warn("Build not found for deletion", zap.String("build_id", buildID.String()))
		return ErrBuildNotFound
	}

	switch b.Status() {
	case BuildStatusRunning, BuildStatusQueued:
		return fmt.Errorf("cannot delete build while execution is running or queued")
	}

	if err := s.repository.Delete(ctx, buildID); err != nil {
		s.logger.Error("Failed to delete build", zap.Error(err), zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to delete build: %w", err)
	}

	s.logger.Info("Build deleted successfully", zap.String("build_id", buildID.String()))
	return nil
}
