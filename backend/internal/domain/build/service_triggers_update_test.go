package build

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubTriggerRepositoryForUpdate struct {
	trigger *BuildTrigger
}

func (s *stubTriggerRepositoryForUpdate) SaveTrigger(ctx context.Context, trigger *BuildTrigger) error {
	s.trigger = trigger
	return nil
}
func (s *stubTriggerRepositoryForUpdate) GetTrigger(ctx context.Context, triggerID uuid.UUID) (*BuildTrigger, error) {
	if s.trigger != nil && s.trigger.ID == triggerID {
		return s.trigger, nil
	}
	return nil, nil
}
func (s *stubTriggerRepositoryForUpdate) GetTriggersByBuild(ctx context.Context, buildID uuid.UUID) ([]*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepositoryForUpdate) GetTriggersByProject(ctx context.Context, projectID uuid.UUID) ([]*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepositoryForUpdate) GetActiveScheduledTriggers(ctx context.Context, tenantID uuid.UUID) ([]*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepositoryForUpdate) UpdateTrigger(ctx context.Context, trigger *BuildTrigger) error {
	s.trigger = trigger
	return nil
}
func (s *stubTriggerRepositoryForUpdate) DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error {
	return nil
}

type stubEventPublisherForTriggerUpdate struct {
	statuses []string
}

func (s *stubEventPublisherForTriggerUpdate) PublishBuildCreated(ctx context.Context, event *BuildCreated) error {
	return nil
}
func (s *stubEventPublisherForTriggerUpdate) PublishBuildStarted(ctx context.Context, event *BuildStarted) error {
	return nil
}
func (s *stubEventPublisherForTriggerUpdate) PublishBuildCompleted(ctx context.Context, event *BuildCompleted) error {
	return nil
}
func (s *stubEventPublisherForTriggerUpdate) PublishBuildStatusUpdated(ctx context.Context, event *BuildStatusUpdated) error {
	s.statuses = append(s.statuses, event.Status())
	return nil
}

func TestUpdateProjectWebhookTrigger_SchedulePauseAndResume(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	triggerID := uuid.New()
	repo := &stubTriggerRepositoryForUpdate{
		trigger: &BuildTrigger{
			ID:        triggerID,
			TenantID:  tenantID,
			ProjectID: projectID,
			BuildID:   buildID,
			Type:      TriggerTypeSchedule,
			Name:      "Nightly",
			CronExpr:  "0 0 * * *",
			Timezone:  "UTC",
			IsActive:  true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
	events := &stubEventPublisherForTriggerUpdate{}
	service := &Service{
		triggerRepository: repo,
		eventPublisher:    events,
		logger:            zap.NewNop(),
	}

	paused := false
	updated, err := service.UpdateProjectWebhookTrigger(
		context.Background(),
		projectID,
		triggerID,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&paused,
	)
	if err != nil {
		t.Fatalf("expected no error pausing schedule trigger, got %v", err)
	}
	if updated.IsActive {
		t.Fatal("expected schedule trigger to be paused")
	}
	if len(events.statuses) == 0 || events.statuses[len(events.statuses)-1] != "trigger.schedule.paused" {
		t.Fatalf("expected paused status event, got %+v", events.statuses)
	}

	resumed := true
	cron := "*/15 * * * *"
	updated, err = service.UpdateProjectWebhookTrigger(
		context.Background(),
		projectID,
		triggerID,
		nil,
		nil,
		nil,
		nil,
		&cron,
		nil,
		nil,
		&resumed,
	)
	if err != nil {
		t.Fatalf("expected no error resuming schedule trigger, got %v", err)
	}
	if !updated.IsActive {
		t.Fatal("expected schedule trigger to be resumed")
	}
	if updated.NextTrigger == nil {
		t.Fatal("expected next trigger to be calculated")
	}
	if len(events.statuses) == 0 || events.statuses[len(events.statuses)-1] != "trigger.schedule.resumed" {
		t.Fatalf("expected resumed status event, got %+v", events.statuses)
	}
}
