package buildnotifications

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type stubBuildRepo struct {
	build *build.Build
	err   error
}

func (s *stubBuildRepo) FindByID(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	return s.build, s.err
}

type stubTriggerService struct {
	prefs []buildnotification.ProjectTriggerPreference
	err   error
}

func (s *stubTriggerService) ListProjectTriggerPreferences(ctx context.Context, tenantID, projectID uuid.UUID) ([]buildnotification.ProjectTriggerPreference, error) {
	return s.prefs, s.err
}

type stubDeliveryRepo struct {
	projectMembers []uuid.UUID
	tenantAdmins   []uuid.UUID
	userEmails     []string
	inserted       []InAppNotificationRow
	err            error
}

func (s *stubDeliveryRepo) ListProjectMemberUserIDs(ctx context.Context, projectID uuid.UUID) ([]uuid.UUID, error) {
	return s.projectMembers, nil
}

func (s *stubDeliveryRepo) ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return s.tenantAdmins, nil
}

func (s *stubDeliveryRepo) ListUserEmailsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]string, error) {
	return s.userEmails, nil
}

func (s *stubDeliveryRepo) InsertInAppNotifications(ctx context.Context, rows []InAppNotificationRow) error {
	if s.err != nil {
		return s.err
	}
	s.inserted = append(s.inserted, rows...)
	return nil
}

type sentEmail struct {
	tenantID uuid.UUID
	to       string
	template string
	subject  string
	bodyText string
}

type stubEmailSender struct {
	sent []sentEmail
	err  error
}

func (s *stubEmailSender) SendBuildNotificationEmail(ctx context.Context, tenantID uuid.UUID, toEmail, templateType string, templateData map[string]interface{}, fallbackSubject, fallbackBody string) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, sentEmail{
		tenantID: tenantID,
		to:       toEmail,
		template: templateType,
		subject:  fallbackSubject,
		bodyText: fallbackBody,
	})
	return nil
}

func TestEventSubscriber_InitiatorNotification(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	initiatorID := uuid.New()

	b := build.NewBuildFromDB(
		buildID,
		tenantID,
		projectID,
		build.BuildManifest{Name: "test", Type: build.BuildTypeKaniko},
		build.BuildStatusQueued,
		testNow(),
		testNow(),
		&initiatorID,
	)

	sub := NewEventSubscriber(
		&stubBuildRepo{build: b},
		&stubTriggerService{
			prefs: []buildnotification.ProjectTriggerPreference{
				{
					TriggerID:       buildnotification.TriggerBuildStarted,
					Enabled:         true,
					Channels:        []buildnotification.Channel{buildnotification.ChannelInApp},
					RecipientPolicy: buildnotification.RecipientInitiator,
				},
			},
		},
		&stubDeliveryRepo{},
		nil,
		zap.NewNop(),
	)

	delivery := sub.deliveryRepo.(*stubDeliveryRepo)
	sub.HandleBuildEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildStarted,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
		},
	})

	if len(delivery.inserted) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(delivery.inserted))
	}
	if delivery.inserted[0].UserID != initiatorID {
		t.Fatalf("expected initiator recipient %s, got %s", initiatorID, delivery.inserted[0].UserID)
	}
	if delivery.inserted[0].NotificationType != "build_started" {
		t.Fatalf("expected notification type build_started, got %s", delivery.inserted[0].NotificationType)
	}
}

func TestEventSubscriber_IgnoresCancelledTriggerWhenDisabled(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()

	b := build.NewBuildFromDB(
		buildID,
		tenantID,
		projectID,
		build.BuildManifest{Name: "test", Type: build.BuildTypeKaniko},
		build.BuildStatusRunning,
		testNow(),
		testNow(),
		nil,
	)

	sub := NewEventSubscriber(
		&stubBuildRepo{build: b},
		&stubTriggerService{
			prefs: []buildnotification.ProjectTriggerPreference{
				{
					TriggerID:       buildnotification.TriggerBuildCancelled,
					Enabled:         false,
					Channels:        []buildnotification.Channel{buildnotification.ChannelInApp},
					RecipientPolicy: buildnotification.RecipientProjectMember,
				},
			},
		},
		&stubDeliveryRepo{projectMembers: []uuid.UUID{uuid.New()}},
		nil,
		zap.NewNop(),
	)

	delivery := sub.deliveryRepo.(*stubDeliveryRepo)
	sub.HandleBuildEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionStatusUpdate,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
			"status":   "cancelled",
		},
	})

	if len(delivery.inserted) != 0 {
		t.Fatalf("expected 0 notifications when trigger disabled, got %d", len(delivery.inserted))
	}
}

func testNow() (tVal time.Time) {
	return time.Unix(1, 0).UTC()
}

func TestEventSubscriber_EmailOnlyNotification(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	initiatorID := uuid.New()

	b := build.NewBuildFromDB(
		buildID,
		tenantID,
		projectID,
		build.BuildManifest{Name: "test", Type: build.BuildTypeKaniko},
		build.BuildStatusQueued,
		testNow(),
		testNow(),
		&initiatorID,
	)

	delivery := &stubDeliveryRepo{
		userEmails: []string{"builder@example.com"},
	}
	emailSender := &stubEmailSender{}

	sub := NewEventSubscriber(
		&stubBuildRepo{build: b},
		&stubTriggerService{
			prefs: []buildnotification.ProjectTriggerPreference{
				{
					TriggerID:       buildnotification.TriggerBuildStarted,
					Enabled:         true,
					Channels:        []buildnotification.Channel{buildnotification.ChannelEmail},
					RecipientPolicy: buildnotification.RecipientInitiator,
				},
			},
		},
		delivery,
		emailSender,
		zap.NewNop(),
	)

	sub.HandleBuildEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildStarted,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
		},
	})

	if len(delivery.inserted) != 0 {
		t.Fatalf("expected no in-app notifications for email-only channel, got %d", len(delivery.inserted))
	}
	if len(emailSender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(emailSender.sent))
	}
	if emailSender.sent[0].to != "builder@example.com" {
		t.Fatalf("expected recipient builder@example.com, got %s", emailSender.sent[0].to)
	}
	if emailSender.sent[0].tenantID != tenantID {
		t.Fatalf("expected tenant_id %s, got %s", tenantID, emailSender.sent[0].tenantID)
	}
	if emailSender.sent[0].template != "build_started" {
		t.Fatalf("expected template build_started, got %s", emailSender.sent[0].template)
	}
}

func TestMapEventToTrigger_PreflightBlockedFromFailureMessage(t *testing.T) {
	trigger, notificationType, title, ok := mapEventToTrigger(messaging.Event{
		Type: messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{
			"build_id": "00000000-0000-0000-0000-000000000001",
			"message":  "Tekton preflight failed: missing task",
		},
	})
	if !ok {
		t.Fatal("expected preflight blocked trigger to be detected")
	}
	if trigger != buildnotification.TriggerPreflightBlocked {
		t.Fatalf("expected trigger %s, got %s", buildnotification.TriggerPreflightBlocked, trigger)
	}
	if notificationType != "preflight_blocked" {
		t.Fatalf("expected notification type preflight_blocked, got %s", notificationType)
	}
	if title != "Build Preflight Blocked" {
		t.Fatalf("unexpected title: %s", title)
	}
}

func TestMapEventToTrigger_RetryStatusUpdate(t *testing.T) {
	trigger, notificationType, title, ok := mapEventToTrigger(messaging.Event{
		Type: messaging.EventTypeBuildExecutionStatusUpdate,
		Payload: map[string]interface{}{
			"build_id": "00000000-0000-0000-0000-000000000001",
			"status":   "retry_started",
		},
	})
	if !ok {
		t.Fatal("expected retry_started trigger")
	}
	if trigger != buildnotification.TriggerBuildRetryStarted {
		t.Fatalf("expected trigger %s, got %s", buildnotification.TriggerBuildRetryStarted, trigger)
	}
	if notificationType != "build_retry_started" {
		t.Fatalf("expected notification type build_retry_started, got %s", notificationType)
	}
	if title != "Build Retry Started" {
		t.Fatalf("unexpected title: %s", title)
	}
}

func TestMapEventToTrigger_ScheduledStatusUpdates(t *testing.T) {
	cases := []struct {
		status           string
		trigger          buildnotification.TriggerID
		notificationType string
		title            string
	}{
		{
			status:           "scheduled_queued",
			trigger:          buildnotification.TriggerBuildScheduledQueued,
			notificationType: "build_scheduled_queued",
			title:            "Scheduled Build Queued",
		},
		{
			status:           "scheduled_failed",
			trigger:          buildnotification.TriggerBuildScheduledFailed,
			notificationType: "build_scheduled_failed",
			title:            "Scheduled Build Failed",
		},
		{
			status:           "scheduled_noop",
			trigger:          buildnotification.TriggerBuildScheduledNoOp,
			notificationType: "build_scheduled_noop",
			title:            "Scheduled Build Skipped",
		},
	}

	for _, tc := range cases {
		trigger, notificationType, title, ok := mapEventToTrigger(messaging.Event{
			Type: messaging.EventTypeBuildExecutionStatusUpdate,
			Payload: map[string]interface{}{
				"build_id": "00000000-0000-0000-0000-000000000001",
				"status":   tc.status,
			},
		})
		if !ok {
			t.Fatalf("expected status %s to map to a trigger", tc.status)
		}
		if trigger != tc.trigger {
			t.Fatalf("status %s expected trigger %s, got %s", tc.status, tc.trigger, trigger)
		}
		if notificationType != tc.notificationType {
			t.Fatalf("status %s expected notification type %s, got %s", tc.status, tc.notificationType, notificationType)
		}
		if title != tc.title {
			t.Fatalf("status %s expected title %s, got %s", tc.status, tc.title, title)
		}
	}
}

func TestEventSubscriber_TriggerMatrix_CustomUsers_DeliversInAppAndEmail(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	customUserA := uuid.New()
	customUserB := uuid.New()

	b := build.NewBuildFromDB(
		buildID,
		tenantID,
		projectID,
		build.BuildManifest{Name: "matrix-test", Type: build.BuildTypeKaniko},
		build.BuildStatusFailed,
		testNow(),
		testNow(),
		nil,
	)

	delivery := &stubDeliveryRepo{
		userEmails: []string{"alpha@example.com", "beta@example.com"},
	}
	emailSender := &stubEmailSender{}

	sub := NewEventSubscriber(
		&stubBuildRepo{build: b},
		&stubTriggerService{
			prefs: []buildnotification.ProjectTriggerPreference{
				{
					TriggerID:            buildnotification.TriggerBuildFailed,
					Enabled:              true,
					Channels:             []buildnotification.Channel{buildnotification.ChannelInApp, buildnotification.ChannelEmail},
					RecipientPolicy:      buildnotification.RecipientCustomUsers,
					CustomRecipientUsers: []uuid.UUID{customUserA, customUserB},
				},
			},
		},
		delivery,
		emailSender,
		zap.NewNop(),
	)

	sub.HandleBuildEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
			"message":  "Build failed in matrix test",
		},
	})

	if len(delivery.inserted) != 2 {
		t.Fatalf("expected 2 in-app notifications, got %d", len(delivery.inserted))
	}
	if len(emailSender.sent) != 2 {
		t.Fatalf("expected 2 email handoff records, got %d", len(emailSender.sent))
	}
	snapshot := sub.Snapshot()
	if snapshot.EventsReceived != 1 {
		t.Fatalf("expected events_received=1, got %d", snapshot.EventsReceived)
	}
	if snapshot.MappedEvents != 1 {
		t.Fatalf("expected mapped_events=1, got %d", snapshot.MappedEvents)
	}
	if snapshot.InAppDelivered != 2 {
		t.Fatalf("expected in_app_delivered=2, got %d", snapshot.InAppDelivered)
	}
	if snapshot.EmailQueued != 2 {
		t.Fatalf("expected email_queued=2, got %d", snapshot.EmailQueued)
	}
	if snapshot.Failures != 0 {
		t.Fatalf("expected failures=0, got %d", snapshot.Failures)
	}
}
