package build

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const defaultScheduledTriggerBatchSize = 10

// ProcessScheduledTriggers processes due active schedule triggers and queues builds.
func (s *Service) ProcessScheduledTriggers(ctx context.Context, limit int) (int, error) {
	if s == nil || s.triggerRepository == nil || s.repository == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = defaultScheduledTriggerBatchSize
	}

	triggers, err := s.triggerRepository.GetActiveScheduledTriggers(ctx, uuid.Nil)
	if err != nil {
		return 0, fmt.Errorf("failed to load active scheduled triggers: %w", err)
	}
	if len(triggers) == 0 {
		return 0, nil
	}

	processed := 0
	for _, trigger := range triggers {
		if processed >= limit {
			break
		}
		if trigger == nil || !trigger.IsActive || trigger.Type != TriggerTypeSchedule {
			continue
		}
		now := time.Now().UTC()
		nextTrigger, nextErr := nextTriggerTimeFromCron(trigger.CronExpr, trigger.Timezone, now)
		if nextErr != nil {
			s.logger.Warn("Scheduled trigger has invalid cron expression; skipping",
				zap.String("trigger_id", trigger.ID.String()),
				zap.String("cron_expr", trigger.CronExpr),
				zap.Error(nextErr))
			s.publishScheduledTriggerStatus(ctx, trigger, "scheduled_failed", "Scheduled trigger failed due to invalid cron expression", map[string]interface{}{
				"reason":    "invalid_cron_expression",
				"cron_expr": trigger.CronExpr,
			})
			continue
		}

		sourceBuild, findErr := s.repository.FindByID(ctx, trigger.BuildID)
		if findErr != nil || sourceBuild == nil {
			s.logger.Warn("Scheduled trigger source build is missing; advancing trigger",
				zap.String("trigger_id", trigger.ID.String()),
				zap.String("build_id", trigger.BuildID.String()),
				zap.Error(findErr))
			trigger.RecordTrigger(nextTrigger)
			_ = s.triggerRepository.UpdateTrigger(ctx, trigger)
			s.publishScheduledTriggerStatus(ctx, trigger, "scheduled_failed", "Scheduled trigger failed because source build was not found", map[string]interface{}{
				"reason":   "source_build_missing",
				"build_id": trigger.BuildID.String(),
			})
			continue
		}

		if sourceBuild.Manifest().Type != BuildTypePacker {
			// PR7 scope: schedule automation is restricted to packer builds.
			trigger.RecordTrigger(nextTrigger)
			_ = s.triggerRepository.UpdateTrigger(ctx, trigger)
			continue
		}

		if s.shouldSkipForScheduledForbidPolicy(ctx, sourceBuild) {
			s.logger.Info("Skipping scheduled trigger due to active packer build in project",
				zap.String("trigger_id", trigger.ID.String()),
				zap.String("project_id", sourceBuild.ProjectID().String()),
				zap.String("concurrency_policy", "forbid"))
			trigger.RecordTrigger(nextTrigger)
			_ = s.triggerRepository.UpdateTrigger(ctx, trigger)
			s.publishScheduledTriggerStatus(ctx, trigger, "scheduled_noop", "Scheduled trigger skipped due to forbid concurrency policy", map[string]interface{}{
				"reason":             "forbid_concurrency_policy",
				"source_project_id":  sourceBuild.ProjectID().String(),
				"source_build_id":    sourceBuild.ID().String(),
				"concurrency_policy": "forbid",
			})
			processed++
			continue
		}

		manifest := cloneBuildManifest(sourceBuild.Manifest())
		if manifest.Metadata == nil {
			manifest.Metadata = map[string]interface{}{}
		}
		manifest.Metadata["trigger_type"] = string(TriggerTypeSchedule)
		manifest.Metadata["trigger_mode"] = "scheduled"
		manifest.Metadata["schedule_trigger_id"] = trigger.ID.String()
		manifest.Metadata["schedule_fire_timestamp"] = now.Format(time.RFC3339)
		manifest.Metadata["schedule_concurrency_policy"] = "forbid"

		newBuild, createErr := s.CreateBuild(ctx, trigger.TenantID, trigger.ProjectID, manifest, &trigger.CreatedBy)
		if createErr != nil {
			s.logger.Warn("Failed to queue build for scheduled trigger",
				zap.String("trigger_id", trigger.ID.String()),
				zap.String("build_id", trigger.BuildID.String()),
				zap.Error(createErr))
			s.publishScheduledTriggerStatus(ctx, trigger, "scheduled_failed", "Scheduled trigger failed to queue a build", map[string]interface{}{
				"reason":          "create_build_failed",
				"source_build_id": trigger.BuildID.String(),
				"error":           createErr.Error(),
			})
			continue
		}

		s.logger.Info("Scheduled trigger queued build",
			zap.String("trigger_id", trigger.ID.String()),
			zap.String("source_build_id", trigger.BuildID.String()),
			zap.String("queued_build_id", newBuild.ID().String()),
			zap.String("concurrency_policy", "forbid"))
		s.publishScheduledTriggerStatus(ctx, trigger, "scheduled_queued", "Scheduled trigger queued a build", map[string]interface{}{
			"source_build_id": trigger.BuildID.String(),
			"queued_build_id": newBuild.ID().String(),
		})

		trigger.RecordTrigger(nextTrigger)
		if updateErr := s.triggerRepository.UpdateTrigger(ctx, trigger); updateErr != nil {
			s.logger.Warn("Failed to persist trigger schedule timestamps",
				zap.String("trigger_id", trigger.ID.String()),
				zap.Error(updateErr))
		}
		processed++
	}
	return processed, nil
}

func (s *Service) publishScheduledTriggerStatus(ctx context.Context, trigger *BuildTrigger, status, message string, metadata map[string]interface{}) {
	if s == nil || s.eventPublisher == nil || trigger == nil {
		return
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadata["trigger_id"] = trigger.ID.String()
	metadata["trigger_type"] = string(trigger.Type)
	if _, exists := metadata["build_id"]; !exists {
		metadata["build_id"] = trigger.BuildID.String()
	}
	event := NewBuildStatusUpdated(trigger.BuildID, trigger.TenantID, status, message, metadata)
	if err := s.eventPublisher.PublishBuildStatusUpdated(ctx, event); err != nil {
		s.logger.Warn("Failed to publish scheduled trigger status event",
			zap.String("trigger_id", trigger.ID.String()),
			zap.String("status", status),
			zap.Error(err))
	}
}

func (s *Service) shouldSkipForScheduledForbidPolicy(ctx context.Context, sourceBuild *Build) bool {
	if s == nil || s.repository == nil || sourceBuild == nil {
		return false
	}
	candidates, err := s.repository.FindByProjectID(ctx, sourceBuild.ProjectID(), 50, 0)
	if err != nil {
		s.logger.Warn("Failed to evaluate scheduled trigger concurrency policy", zap.Error(err), zap.String("project_id", sourceBuild.ProjectID().String()))
		return false
	}

	for _, candidate := range candidates {
		if candidate == nil || candidate.ID() == sourceBuild.ID() {
			continue
		}
		if candidate.Status() != BuildStatusQueued && candidate.Status() != BuildStatusRunning {
			continue
		}
		if candidate.Manifest().Type != BuildTypePacker {
			continue
		}
		return true
	}
	return false
}

func cloneBuildManifest(manifest BuildManifest) BuildManifest {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return manifest
	}
	var cloned BuildManifest
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return manifest
	}
	return cloned
}
