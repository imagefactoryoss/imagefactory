package build

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// executeBuild runs the build execution in a goroutine.
func (s *Service) executeBuild(ctx context.Context, build *Build, isRetry bool) {
	buildID := build.ID()
	manifest := build.Manifest()

	s.logger.Info("Executing build",
		zap.String("build_id", buildID.String()),
		zap.String("build_type", string(manifest.Type)))

	var execution *BuildExecution
	if s.executionService != nil {
		config := build.Config()
		if config != nil && config.ID != uuid.Nil {
			createdBy := uuid.Nil
			if build.CreatedBy() != nil {
				createdBy = *build.CreatedBy()
			}
			if createdBy == uuid.Nil {
				s.logger.Warn("Skipping build execution record: missing created_by", zap.String("build_id", buildID.String()))
			} else {
				exec, err := s.executionService.StartBuild(ctx, config.ID, createdBy)
				if err != nil {
					if errors.Is(err, ErrBuildAlreadyExecuting) {
						s.logger.Warn("Build execution already running, skipping duplicate dispatch", zap.String("build_id", buildID.String()))
						return
					}
					s.logger.Warn("Failed to create build execution record", zap.Error(err), zap.String("build_id", buildID.String()))
				} else {
					execution = exec
					ctx = WithExecutionID(ctx, execution.ID.String())
					_ = s.executionService.UpdateExecutionStatus(ctx, execution.ID, ExecutionRunning)
				}
			}
		}
	}

	// Select appropriate executor based on infrastructure and build type
	var executor BuildExecutor
	if build.InfrastructureType() == "kubernetes" && s.tektonExecutorFactory != nil && build.Config() != nil {
		method := BuildMethod(build.Config().BuildMethod)
		if method == BuildMethod("container") {
			method = BuildMethodDocker
		}
		if method.IsValid() && supportsMethod(s.tektonExecutorFactory, method) {
			executor = NewMethodExecutorAdapter(s.tektonExecutorFactory, s.logger)
		} else if s.localExecutorFactory != nil {
			s.logger.Info("Falling back to local executor for unsupported Tekton method",
				zap.String("build_id", buildID.String()),
				zap.String("method", build.Config().BuildMethod))
			executor = NewMethodExecutorAdapter(s.localExecutorFactory, s.logger)
		}
	} else if s.localExecutorFactory != nil && build.Config() != nil {
		executor = NewMethodExecutorAdapter(s.localExecutorFactory, s.logger)
	} else {
		switch manifest.Type {
		case BuildTypeVM:
			executor = s.vmExecutor
		case BuildTypeContainer, BuildTypeCloud, BuildTypePacker, BuildTypePaketo, BuildTypeKaniko, BuildTypeBuildx, BuildTypeNix:
			executor = s.containerExecutor
		default:
			s.logger.Error("Unsupported build type", zap.String("build_type", string(manifest.Type)), zap.String("build_id", buildID.String()))
			return
		}
	}

	if executor == nil {
		s.logger.Error("No executor available for build", zap.String("build_id", buildID.String()), zap.String("build_type", string(manifest.Type)))
		return
	}

	// Execute the build
	result, err := executor.Execute(ctx, build)
	if errors.Is(err, ErrBuildExecutionInProgress) {
		s.logger.Info("Build execution is running asynchronously",
			zap.String("build_id", buildID.String()),
			zap.String("build_type", string(manifest.Type)))
		return
	}

	// Reload build from repository to get latest state
	currentBuild, repoErr := s.repository.FindByID(ctx, buildID)
	if repoErr != nil {
		s.logger.Error("Failed to reload build after execution", zap.Error(repoErr), zap.String("build_id", buildID.String()))
		return
	}

	if currentBuild == nil {
		s.logger.Error("Build not found after execution", zap.String("build_id", buildID.String()))
		return
	}

	// Complete or fail the build
	if err != nil {
		s.logger.Error("Build execution failed", zap.Error(err), zap.String("build_id", buildID.String()))
		status := "failed"
		message := "Build execution failed"
		if isRetry {
			status = "retry_failed"
			message = "Build retry failed"
		}
		if strings.Contains(strings.ToLower(err.Error()), "preflight") {
			status = "preflight_blocked"
			message = "Build preflight blocked execution"
		}
		statusEvent := NewBuildStatusUpdated(buildID, currentBuild.TenantID(), status, message, map[string]interface{}{
			"error": err.Error(),
		})
		if publishErr := s.eventPublisher.PublishBuildStatusUpdated(ctx, statusEvent); publishErr != nil {
			s.logger.Warn("Failed to publish build status update event", zap.Error(publishErr), zap.String("build_id", buildID.String()))
		}
		if failErr := currentBuild.Fail(err.Error()); failErr != nil {
			s.logger.Error("Failed to mark build as failed", zap.Error(failErr), zap.String("build_id", buildID.String()))
			return
		}
		if s.executionService != nil && execution != nil {
			_ = s.executionService.CompleteExecution(ctx, execution.ID, false, err.Error(), nil)
		}
	} else {
		s.logger.Info("Build execution completed successfully", zap.String("build_id", buildID.String()))
		if isRetry {
			statusEvent := NewBuildStatusUpdated(buildID, currentBuild.TenantID(), "retry_succeeded", "Build retry succeeded", map[string]interface{}{})
			if publishErr := s.eventPublisher.PublishBuildStatusUpdated(ctx, statusEvent); publishErr != nil {
				s.logger.Warn("Failed to publish build retry succeeded event", zap.Error(publishErr), zap.String("build_id", buildID.String()))
			}
		}
		if completeErr := currentBuild.Complete(*result); completeErr != nil {
			s.logger.Error("Failed to mark build as completed", zap.Error(completeErr), zap.String("build_id", buildID.String()))
			return
		}
		if s.executionService != nil && execution != nil {
			artifactsPayload := executionArtifactsPayload(result)
			s.persistPackerExecutionMetadata(ctx, execution.ID, currentBuild, artifactsPayload)
			_ = s.executionService.CompleteExecution(ctx, execution.ID, true, "", artifactsPayload)
		}
	}

	// Update in repository
	if updateErr := s.repository.Update(ctx, currentBuild); updateErr != nil {
		s.logger.Error("Failed to update build after execution", zap.Error(updateErr), zap.String("build_id", buildID.String()))
		return
	}

	// Publish completion event
	resultForEvent := result
	if resultForEvent == nil {
		if currentBuild.Result() != nil {
			cloned := *currentBuild.Result()
			resultForEvent = &cloned
		} else {
			fallback := BuildResult{}
			if currentBuild.ErrorMessage() != "" {
				fallback.Logs = []string{currentBuild.ErrorMessage()}
			}
			resultForEvent = &fallback
		}
	}
	event := NewBuildCompleted(buildID, currentBuild.TenantID(), *resultForEvent)
	if publishErr := s.eventPublisher.PublishBuildCompleted(ctx, event); publishErr != nil {
		s.logger.Error("Failed to publish build completed event", zap.Error(publishErr), zap.String("build_id", buildID.String()))
	}

	s.logger.Info("Build execution finished", zap.String("build_id", buildID.String()), zap.String("status", string(currentBuild.Status())))
}

func (s *Service) dispatchBuild(ctx context.Context, build *Build, isRetry bool) {
	// Build dispatch/execution is asynchronous and must survive request cancellation.
	dispatchCtx := context.WithoutCancel(ctx)

	// Publish started event
	event := NewBuildStarted(build.ID(), build.TenantID())
	if err := s.eventPublisher.PublishBuildStarted(dispatchCtx, event); err != nil {
		s.logger.Error("Failed to publish build started event", zap.Error(err), zap.String("build_id", build.ID().String()))
	}

	// Execute the build asynchronously
	go s.executeBuild(dispatchCtx, build, isRetry)
}
