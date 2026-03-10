package build

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"
)

func executionArtifactsPayload(result *BuildResult) []byte {
	if result == nil {
		return nil
	}
	if result.ScanResults != nil {
		if raw, ok := result.ScanResults["method_artifacts_json"].(string); ok && strings.TrimSpace(raw) != "" {
			return []byte(raw)
		}
		if raw, ok := result.ScanResults["method_artifacts"]; ok {
			if payload, err := json.Marshal(raw); err == nil {
				return payload
			}
		}
	}
	if len(result.Artifacts) > 0 {
		if payload, err := json.Marshal(result.Artifacts); err == nil {
			return payload
		}
	}
	return nil
}

func (s *Service) recordRepoConfigFailure(ctx context.Context, b *Build, stage string, cause error) {
	if s == nil || b == nil || cause == nil {
		return
	}
	manifest := b.Manifest()
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}
	manifest.Metadata["repo_config_applied"] = false
	manifest.Metadata["repo_config_error"] = cause.Error()
	manifest.Metadata["repo_config_error_stage"] = strings.TrimSpace(stage)
	manifest.Metadata["repo_config_error_at"] = time.Now().UTC().Format(time.RFC3339)
	b.manifest = manifest

	if (b.Status() == BuildStatusQueued || b.Status() == BuildStatusPending) && b.ErrorMessage() == "" {
		_ = b.Fail(cause.Error())
	}
	if err := s.repository.Update(ctx, b); err != nil {
		s.logger.Warn("Failed to persist repository-managed build config failure diagnostics",
			zap.String("build_id", b.ID().String()),
			zap.String("stage", strings.TrimSpace(stage)),
			zap.Error(err))
	}
}

func (s *Service) recordRepoConfigFallback(ctx context.Context, b *Build, stage string, cause error) {
	if s == nil || b == nil || cause == nil {
		return
	}
	manifest := b.Manifest()
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}
	manifest.Metadata["repo_config_applied"] = false
	manifest.Metadata["repo_config_error"] = cause.Error()
	manifest.Metadata["repo_config_error_stage"] = strings.TrimSpace(stage)
	manifest.Metadata["repo_config_error_at"] = time.Now().UTC().Format(time.RFC3339)
	manifest.Metadata["repo_config_error_policy"] = "fallback_to_ui"
	b.manifest = manifest

	if err := s.repository.Update(ctx, b); err != nil {
		s.logger.Warn("Failed to persist repository-managed build config fallback diagnostics",
			zap.String("build_id", b.ID().String()),
			zap.String("stage", strings.TrimSpace(stage)),
			zap.Error(err))
	}
	s.logger.Warn("Falling back to saved build config after repository-managed config error",
		zap.String("build_id", b.ID().String()),
		zap.String("stage", strings.TrimSpace(stage)),
		zap.Error(cause))
}

func (s *Service) publishRepoConfigFailureStatus(ctx context.Context, b *Build, stage string, cause error, isRetry bool) {
	if s == nil || s.eventPublisher == nil || b == nil || cause == nil {
		return
	}
	status := "repo_config_failed"
	message := "Repository build config validation failed"
	if isRetry {
		status = "retry_repo_config_failed"
		message = "Build retry blocked by repository config validation failure"
	}
	statusEvent := NewBuildStatusUpdated(b.ID(), b.TenantID(), status, message, map[string]interface{}{
		"error":       cause.Error(),
		"stage":       strings.TrimSpace(stage),
		"build_id":    b.ID().String(),
		"repo_config": true,
	})
	if err := s.eventPublisher.PublishBuildStatusUpdated(ctx, statusEvent); err != nil {
		s.logger.Warn("Failed to publish repository config failure status event",
			zap.String("build_id", b.ID().String()),
			zap.String("stage", strings.TrimSpace(stage)),
			zap.Error(err))
	}
}
