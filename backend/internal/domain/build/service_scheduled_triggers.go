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
			continue
		}

		s.logger.Info("Scheduled trigger queued build",
			zap.String("trigger_id", trigger.ID.String()),
			zap.String("source_build_id", trigger.BuildID.String()),
			zap.String("queued_build_id", newBuild.ID().String()),
			zap.String("concurrency_policy", "forbid"))

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
