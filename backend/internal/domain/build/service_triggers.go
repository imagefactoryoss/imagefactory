package build

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CreateWebhookTrigger creates and saves a webhook trigger for a build.
func (s *Service) CreateWebhookTrigger(
	ctx context.Context,
	tenantID, projectID, buildID, createdBy uuid.UUID,
	name, description, webhookURL, webhookSecret string,
	events []string,
) (*BuildTrigger, error) {
	s.logger.Info("Creating webhook trigger",
		zap.String("build_id", buildID.String()),
		zap.String("webhook_url", webhookURL))

	// Verify build exists
	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to verify build exists", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to verify build: %w", err)
	}
	if build == nil {
		return nil, fmt.Errorf("build not found: %s", buildID.String())
	}

	// Create trigger domain object
	trigger, err := NewWebhookTrigger(tenantID, projectID, buildID, createdBy, name, description, webhookURL, webhookSecret, events)
	if err != nil {
		s.logger.Error("Failed to create webhook trigger", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, err
	}

	// Save to repository
	if err := s.triggerRepository.SaveTrigger(ctx, trigger); err != nil {
		s.logger.Error("Failed to save webhook trigger", zap.Error(err), zap.String("trigger_id", trigger.ID.String()))
		return nil, fmt.Errorf("failed to save trigger: %w", err)
	}

	s.logger.Info("Webhook trigger created successfully", zap.String("trigger_id", trigger.ID.String()))
	return trigger, nil
}

// CreateProjectWebhookTrigger creates a webhook trigger at project scope.
// If buildID is nil, the latest project build is used as the trigger template.
func (s *Service) CreateProjectWebhookTrigger(
	ctx context.Context,
	tenantID, projectID uuid.UUID,
	buildID *uuid.UUID,
	createdBy uuid.UUID,
	name, description, webhookURL, webhookSecret string,
	events []string,
) (*BuildTrigger, error) {
	resolvedBuildID := uuid.Nil
	if buildID != nil {
		resolvedBuildID = *buildID
	}

	if resolvedBuildID == uuid.Nil {
		projectBuilds, err := s.repository.FindByProjectID(ctx, projectID, 1, 0)
		if err != nil {
			s.logger.Error("Failed to resolve latest build for project trigger", zap.Error(err), zap.String("project_id", projectID.String()))
			return nil, fmt.Errorf("failed to resolve project build template: %w", err)
		}
		if len(projectBuilds) == 0 {
			return nil, errors.New("no project builds found; create at least one build before adding webhook triggers")
		}
		resolvedBuildID = projectBuilds[0].ID()
	}

	return s.CreateWebhookTrigger(
		ctx,
		tenantID,
		projectID,
		resolvedBuildID,
		createdBy,
		name,
		description,
		webhookURL,
		webhookSecret,
		events,
	)
}

// CreateScheduledTrigger creates and saves a scheduled trigger for a build.
func (s *Service) CreateScheduledTrigger(
	ctx context.Context,
	tenantID, projectID, buildID, createdBy uuid.UUID,
	name, description, cronExpr, timezone string,
) (*BuildTrigger, error) {
	s.logger.Info("Creating scheduled trigger",
		zap.String("build_id", buildID.String()),
		zap.String("cron_expr", cronExpr))

	// Verify build exists
	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to verify build exists", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to verify build: %w", err)
	}
	if build == nil {
		return nil, fmt.Errorf("build not found: %s", buildID.String())
	}

	// Create trigger domain object
	trigger, err := NewScheduledTrigger(tenantID, projectID, buildID, createdBy, name, description, cronExpr, timezone)
	if err != nil {
		s.logger.Error("Failed to create scheduled trigger", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, err
	}
	if next, nextErr := nextTriggerTimeFromCron(cronExpr, timezone, time.Now().UTC()); nextErr != nil {
		s.logger.Warn("Failed to compute next trigger timestamp; schedule will be immediately eligible",
			zap.String("build_id", buildID.String()),
			zap.String("cron_expr", cronExpr),
			zap.Error(nextErr))
	} else {
		trigger.NextTrigger = next
	}

	// Save to repository
	if err := s.triggerRepository.SaveTrigger(ctx, trigger); err != nil {
		s.logger.Error("Failed to save scheduled trigger", zap.Error(err), zap.String("trigger_id", trigger.ID.String()))
		return nil, fmt.Errorf("failed to save trigger: %w", err)
	}

	s.logger.Info("Scheduled trigger created successfully", zap.String("trigger_id", trigger.ID.String()))
	return trigger, nil
}

// CreateGitEventTrigger creates and saves a Git event trigger for a build.
func (s *Service) CreateGitEventTrigger(
	ctx context.Context,
	tenantID, projectID, buildID, createdBy uuid.UUID,
	name, description string,
	provider GitProvider,
	repoURL, branchPattern string,
) (*BuildTrigger, error) {
	s.logger.Info("Creating Git event trigger",
		zap.String("build_id", buildID.String()),
		zap.String("git_provider", string(provider)))

	// Verify build exists
	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to verify build exists", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to verify build: %w", err)
	}
	if build == nil {
		return nil, fmt.Errorf("build not found: %s", buildID.String())
	}

	// Create trigger domain object
	trigger, err := NewGitEventTrigger(tenantID, projectID, buildID, createdBy, name, description, provider, repoURL, branchPattern)
	if err != nil {
		s.logger.Error("Failed to create Git event trigger", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, err
	}

	// Save to repository
	if err := s.triggerRepository.SaveTrigger(ctx, trigger); err != nil {
		s.logger.Error("Failed to save Git event trigger", zap.Error(err), zap.String("trigger_id", trigger.ID.String()))
		return nil, fmt.Errorf("failed to save trigger: %w", err)
	}

	s.logger.Info("Git event trigger created successfully", zap.String("trigger_id", trigger.ID.String()))
	return trigger, nil
}

// GetBuildTriggers retrieves all triggers for a build.
func (s *Service) GetBuildTriggers(ctx context.Context, buildID uuid.UUID) ([]*BuildTrigger, error) {
	triggers, err := s.triggerRepository.GetTriggersByBuild(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to get triggers for build", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}
	return triggers, nil
}

// GetProjectTriggers retrieves all triggers for a project.
func (s *Service) GetProjectTriggers(ctx context.Context, projectID uuid.UUID) ([]*BuildTrigger, error) {
	triggers, err := s.triggerRepository.GetTriggersByProject(ctx, projectID)
	if err != nil {
		s.logger.Error("Failed to get triggers for project", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}
	return triggers, nil
}

// UpdateProjectWebhookTrigger updates editable fields for a project webhook trigger.
func (s *Service) UpdateProjectWebhookTrigger(
	ctx context.Context,
	projectID, triggerID uuid.UUID,
	name, description, webhookURL, webhookSecret *string,
	events []string,
	isActive *bool,
) (*BuildTrigger, error) {
	trigger, err := s.triggerRepository.GetTrigger(ctx, triggerID)
	if err != nil {
		s.logger.Error("Failed to get trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return nil, fmt.Errorf("failed to get trigger: %w", err)
	}
	if trigger == nil {
		return nil, fmt.Errorf("trigger not found: %s", triggerID.String())
	}
	if trigger.ProjectID != projectID {
		return nil, errors.New("trigger does not belong to this project")
	}
	if trigger.Type != TriggerTypeWebhook {
		return nil, errors.New("only webhook triggers are editable via this endpoint")
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, errors.New("trigger name is required")
		}
		trigger.Name = trimmed
	}
	if description != nil {
		trigger.Description = strings.TrimSpace(*description)
	}
	if webhookURL != nil {
		trimmed := strings.TrimSpace(*webhookURL)
		if trimmed == "" {
			return nil, errors.New("webhook URL is required")
		}
		trigger.WebhookURL = trimmed
	}
	if webhookSecret != nil {
		trigger.WebhookSecret = strings.TrimSpace(*webhookSecret)
	}
	if events != nil {
		filteredEvents := make([]string, 0, len(events))
		for _, event := range events {
			trimmed := strings.TrimSpace(event)
			if trimmed == "" {
				continue
			}
			filteredEvents = append(filteredEvents, trimmed)
		}
		if len(filteredEvents) == 0 {
			return nil, errors.New("at least one webhook event is required")
		}
		trigger.WebhookEvents = filteredEvents
	}
	if isActive != nil {
		trigger.IsActive = *isActive
	}
	trigger.UpdatedAt = time.Now().UTC()

	if err := s.triggerRepository.UpdateTrigger(ctx, trigger); err != nil {
		s.logger.Error("Failed to update trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return nil, fmt.Errorf("failed to update trigger: %w", err)
	}
	return trigger, nil
}

// DeactivateTrigger deactivates a trigger.
func (s *Service) DeactivateTrigger(ctx context.Context, triggerID uuid.UUID) error {
	s.logger.Info("Deactivating trigger", zap.String("trigger_id", triggerID.String()))

	// Get the trigger
	trigger, err := s.triggerRepository.GetTrigger(ctx, triggerID)
	if err != nil {
		s.logger.Error("Failed to get trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return fmt.Errorf("failed to get trigger: %w", err)
	}
	if trigger == nil {
		return fmt.Errorf("trigger not found: %s", triggerID.String())
	}

	// Deactivate
	trigger.Deactivate()

	// Update in repository
	if err := s.triggerRepository.UpdateTrigger(ctx, trigger); err != nil {
		s.logger.Error("Failed to update trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return fmt.Errorf("failed to deactivate trigger: %w", err)
	}

	s.logger.Info("Trigger deactivated successfully", zap.String("trigger_id", triggerID.String()))
	return nil
}

// DeleteTrigger deletes a trigger.
func (s *Service) DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error {
	s.logger.Info("Deleting trigger", zap.String("trigger_id", triggerID.String()))

	if err := s.triggerRepository.DeleteTrigger(ctx, triggerID); err != nil {
		s.logger.Error("Failed to delete trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return fmt.Errorf("failed to delete trigger: %w", err)
	}

	s.logger.Info("Trigger deleted successfully", zap.String("trigger_id", triggerID.String()))
	return nil
}
