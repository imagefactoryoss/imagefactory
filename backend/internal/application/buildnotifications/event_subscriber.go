package buildnotifications

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type BuildRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*build.Build, error)
}

type TriggerService interface {
	ListProjectTriggerPreferences(ctx context.Context, tenantID, projectID uuid.UUID) ([]buildnotification.ProjectTriggerPreference, error)
}

type DeliveryRepository interface {
	ListProjectMemberUserIDs(ctx context.Context, projectID uuid.UUID) ([]uuid.UUID, error)
	ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error)
	ListUserEmailsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]string, error)
	InsertInAppNotifications(ctx context.Context, rows []InAppNotificationRow) error
}

type EmailSender interface {
	SendBuildNotificationEmail(
		ctx context.Context,
		tenantID uuid.UUID,
		toEmail string,
		templateType string,
		templateData map[string]interface{},
		fallbackSubject string,
		fallbackBody string,
	) error
}

type InAppNotificationRow struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	TenantID            uuid.UUID
	Title               string
	Message             string
	NotificationType    string
	RelatedResourceType string
	RelatedResourceID   *uuid.UUID
	Channel             string
}

type EventSubscriber struct {
	buildRepo         BuildRepository
	triggerService    TriggerService
	deliveryRepo      DeliveryRepository
	emailSender       EmailSender
	realtimePublisher NotificationRealtimePublisher
	logger            *zap.Logger
	eventsReceived    atomic.Int64
	mappedEvents      atomic.Int64
	inAppDelivered    atomic.Int64
	emailQueued       atomic.Int64
	failures          atomic.Int64
}

type NotificationRealtimePublisher interface {
	BroadcastNotificationEvent(tenantID, userID uuid.UUID, eventType string, notificationID *uuid.UUID, metadata map[string]interface{})
}

type Snapshot struct {
	EventsReceived int64
	MappedEvents   int64
	InAppDelivered int64
	EmailQueued    int64
	Failures       int64
}

func NewEventSubscriber(buildRepo BuildRepository, triggerService TriggerService, deliveryRepo DeliveryRepository, emailSender EmailSender, logger *zap.Logger) *EventSubscriber {
	return &EventSubscriber{
		buildRepo:      buildRepo,
		triggerService: triggerService,
		deliveryRepo:   deliveryRepo,
		emailSender:    emailSender,
		logger:         logger,
	}
}

func (s *EventSubscriber) SetRealtimePublisher(publisher NotificationRealtimePublisher) {
	if s == nil {
		return
	}
	s.realtimePublisher = publisher
}

func (s *EventSubscriber) HandleBuildEvent(ctx context.Context, event messaging.Event) {
	if s == nil || s.buildRepo == nil || s.triggerService == nil || s.deliveryRepo == nil {
		return
	}
	s.eventsReceived.Add(1)

	triggerID, notificationType, defaultTitle, ok := mapEventToTrigger(event)
	if !ok {
		return
	}
	s.mappedEvents.Add(1)

	buildID, err := parseBuildID(event)
	if err != nil {
		s.failures.Add(1)
		s.logDebug("build notification subscriber ignored event: invalid build_id",
			zap.String("event_type", event.Type),
			zap.Error(err))
		return
	}

	b, err := s.buildRepo.FindByID(ctx, buildID)
	if err != nil {
		s.failures.Add(1)
		s.logWarn("build notification subscriber failed to load build",
			zap.String("event_type", event.Type),
			zap.String("build_id", buildID.String()),
			zap.Error(err))
		return
	}
	if b == nil {
		s.logDebug("build notification subscriber found no build for event",
			zap.String("event_type", event.Type),
			zap.String("build_id", buildID.String()))
		return
	}

	prefs, err := s.triggerService.ListProjectTriggerPreferences(ctx, b.TenantID(), b.ProjectID())
	if err != nil {
		s.failures.Add(1)
		s.logWarn("build notification subscriber failed to load trigger preferences",
			zap.String("build_id", b.ID().String()),
			zap.String("tenant_id", b.TenantID().String()),
			zap.String("project_id", b.ProjectID().String()),
			zap.Error(err))
		return
	}

	pref, found := findPreferenceForTrigger(prefs, triggerID)
	if !found || !pref.Enabled {
		return
	}
	deliverInApp := containsInApp(pref.Channels)
	deliverEmail := containsEmail(pref.Channels)
	if !deliverInApp && !deliverEmail {
		return
	}

	recipientIDs, err := s.resolveRecipients(ctx, b, pref)
	if err != nil {
		s.failures.Add(1)
		s.logWarn("build notification subscriber failed to resolve recipients",
			zap.String("build_id", b.ID().String()),
			zap.String("trigger_id", string(triggerID)),
			zap.Error(err))
		return
	}
	if len(recipientIDs) == 0 {
		return
	}

	message := strings.TrimSpace(stringPayload(event.Payload, "message"))
	if message == "" {
		message = defaultMessageForTrigger(triggerID, b.ID())
	}

	if deliverInApp {
		rows := make([]InAppNotificationRow, 0, len(recipientIDs))
		buildRef := b.ID()
		for _, recipientID := range recipientIDs {
			rows = append(rows, InAppNotificationRow{
				ID:                  uuid.New(),
				UserID:              recipientID,
				TenantID:            b.TenantID(),
				Title:               defaultTitle,
				Message:             message,
				NotificationType:    notificationType,
				RelatedResourceType: "build",
				RelatedResourceID:   &buildRef,
				Channel:             string(buildnotification.ChannelInApp),
			})
		}
		if err := s.deliveryRepo.InsertInAppNotifications(ctx, rows); err != nil {
			s.failures.Add(1)
			s.logWarn("build notification subscriber failed to persist in-app notifications",
				zap.String("build_id", b.ID().String()),
				zap.String("trigger_id", string(triggerID)),
				zap.Error(err))
			return
		}
		s.inAppDelivered.Add(int64(len(rows)))
		if s.realtimePublisher != nil {
			for _, row := range rows {
				notificationID := row.ID
				s.realtimePublisher.BroadcastNotificationEvent(
					row.TenantID,
					row.UserID,
					"notification.created",
					&notificationID,
					map[string]interface{}{
						"notification_type":     row.NotificationType,
						"related_resource_type": row.RelatedResourceType,
					},
				)
			}
		}
	}

	if deliverEmail && s.emailSender != nil {
		emails, err := s.deliveryRepo.ListUserEmailsByIDs(ctx, recipientIDs)
		if err != nil {
			s.failures.Add(1)
			s.logWarn("build notification subscriber failed to resolve recipient emails",
				zap.String("build_id", b.ID().String()),
				zap.String("trigger_id", string(triggerID)),
				zap.Error(err))
			return
		}
		templateType := templateTypeForTrigger(triggerID)
		for _, email := range emails {
			templateData := buildTemplateDataForEmail(b, event, email, message)
			if sendErr := s.emailSender.SendBuildNotificationEmail(
				ctx,
				b.TenantID(),
				email,
				templateType,
				templateData,
				defaultTitle,
				message,
			); sendErr != nil {
				s.failures.Add(1)
				s.logWarn("build notification subscriber failed to enqueue email notification",
					zap.String("build_id", b.ID().String()),
					zap.String("trigger_id", string(triggerID)),
					zap.String("recipient_email", email),
					zap.Error(sendErr))
				continue
			}
			s.emailQueued.Add(1)
		}
	}
}

func (s *EventSubscriber) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	return Snapshot{
		EventsReceived: s.eventsReceived.Load(),
		MappedEvents:   s.mappedEvents.Load(),
		InAppDelivered: s.inAppDelivered.Load(),
		EmailQueued:    s.emailQueued.Load(),
		Failures:       s.failures.Load(),
	}
}

func templateTypeForTrigger(triggerID buildnotification.TriggerID) string {
	switch triggerID {
	case buildnotification.TriggerBuildStarted:
		return "build_started"
	case buildnotification.TriggerBuildCompleted:
		return "build_completed"
	case buildnotification.TriggerBuildFailed:
		return "build_failed"
	case buildnotification.TriggerBuildCancelled:
		return "build_cancelled"
	case buildnotification.TriggerBuildRetryStarted:
		return "build_started"
	case buildnotification.TriggerBuildRetryFailed:
		return "build_failed"
	case buildnotification.TriggerBuildRetrySucceeded:
		return "build_completed"
	case buildnotification.TriggerBuildRecovered:
		return "build_completed"
	case buildnotification.TriggerPreflightBlocked:
		return "build_failed"
	case buildnotification.TriggerBuildScheduledQueued:
		return "build_started"
	case buildnotification.TriggerBuildScheduledFailed:
		return "build_failed"
	case buildnotification.TriggerBuildScheduledNoOp:
		return "build_completed"
	default:
		return "build_failed"
	}
}

func buildTemplateDataForEmail(b *build.Build, event messaging.Event, recipientEmail, message string) map[string]interface{} {
	manifest := b.Manifest()
	now := time.Now().UTC()
	startTime := now
	if b.StartedAt() != nil {
		startTime = *b.StartedAt()
	}
	failureTime := now
	if b.CompletedAt() != nil {
		failureTime = *b.CompletedAt()
	}

	userName := recipientEmail
	if at := strings.Index(userName, "@"); at > 0 {
		userName = userName[:at]
	}
	branch := strings.TrimSpace(stringPayload(event.Payload, "branch"))
	commitHash := strings.TrimSpace(stringPayload(event.Payload, "commit"))
	if commitHash == "" {
		commitHash = strings.TrimSpace(stringPayload(event.Payload, "commit_hash"))
	}
	duration := strings.TrimSpace(stringPayload(event.Payload, "duration"))
	if duration == "" {
		duration = "N/A"
	}
	registryURL := ""
	imageName := ""
	imageTag := "latest"
	if manifest.BuildConfig != nil {
		registryURL = manifest.BuildConfig.RegistryRepo
		imageName = manifest.BuildConfig.RegistryRepo
	}

	return map[string]interface{}{
		"UserName":        userName,
		"ProjectName":     manifest.Name,
		"BuildID":         b.ID().String(),
		"Branch":          branch,
		"CommitHash":      commitHash,
		"StartTime":       startTime.Format(time.RFC3339),
		"FailureTime":     failureTime.Format(time.RFC3339),
		"TriggeredBy":     userName,
		"Duration":        duration,
		"ImageName":       imageName,
		"ImageTag":        imageTag,
		"ImageSize":       "N/A",
		"LayerCount":      "N/A",
		"RegistryURL":     registryURL,
		"DashboardURL":    "",
		"BuildLogsURL":    "",
		"ImageDetailsURL": "",
		"ErrorMessage":    message,
	}
}

func (s *EventSubscriber) resolveRecipients(ctx context.Context, b *build.Build, pref buildnotification.ProjectTriggerPreference) ([]uuid.UUID, error) {
	switch pref.RecipientPolicy {
	case buildnotification.RecipientInitiator:
		if b.CreatedBy() == nil || *b.CreatedBy() == uuid.Nil {
			return nil, nil
		}
		return []uuid.UUID{*b.CreatedBy()}, nil
	case buildnotification.RecipientProjectMember:
		return s.deliveryRepo.ListProjectMemberUserIDs(ctx, b.ProjectID())
	case buildnotification.RecipientTenantAdmins:
		return s.deliveryRepo.ListTenantAdminUserIDs(ctx, b.TenantID())
	case buildnotification.RecipientCustomUsers:
		if len(pref.CustomRecipientUsers) == 0 {
			return nil, nil
		}
		return dedupeUserIDs(pref.CustomRecipientUsers), nil
	default:
		return nil, nil
	}
}

func mapEventToTrigger(event messaging.Event) (buildnotification.TriggerID, string, string, bool) {
	switch event.Type {
	case messaging.EventTypeBuildStarted:
		return buildnotification.TriggerBuildStarted, "build_started", "Build Started", true
	case messaging.EventTypeBuildExecutionCompleted:
		return buildnotification.TriggerBuildCompleted, "build_completed", "Build Completed", true
	case messaging.EventTypeBuildExecutionFailed:
		message := strings.ToLower(strings.TrimSpace(stringPayload(event.Payload, "message")))
		if strings.Contains(message, "preflight") {
			return buildnotification.TriggerPreflightBlocked, "preflight_blocked", "Build Preflight Blocked", true
		}
		return buildnotification.TriggerBuildFailed, "build_failed", "Build Failed", true
	case messaging.EventTypeBuildExecutionStatusUpdate:
		status := strings.ToLower(strings.TrimSpace(stringPayload(event.Payload, "status")))
		switch status {
		case "retry_started":
			return buildnotification.TriggerBuildRetryStarted, "build_retry_started", "Build Retry Started", true
		case "retry_failed":
			return buildnotification.TriggerBuildRetryFailed, "build_retry_failed", "Build Retry Failed", true
		case "retry_succeeded":
			return buildnotification.TriggerBuildRetrySucceeded, "build_retry_succeeded", "Build Retry Succeeded", true
		case "recovered":
			return buildnotification.TriggerBuildRecovered, "build_recovered", "Build Recovered", true
		case "preflight_blocked":
			return buildnotification.TriggerPreflightBlocked, "preflight_blocked", "Build Preflight Blocked", true
		case "scheduled_queued":
			return buildnotification.TriggerBuildScheduledQueued, "build_scheduled_queued", "Scheduled Build Queued", true
		case "scheduled_failed":
			return buildnotification.TriggerBuildScheduledFailed, "build_scheduled_failed", "Scheduled Build Failed", true
		case "scheduled_noop":
			return buildnotification.TriggerBuildScheduledNoOp, "build_scheduled_noop", "Scheduled Build Skipped", true
		}
		if status == "cancelled" || status == "canceled" {
			return buildnotification.TriggerBuildCancelled, "build_cancelled", "Build Cancelled", true
		}
		return "", "", "", false
	default:
		return "", "", "", false
	}
}

func findPreferenceForTrigger(prefs []buildnotification.ProjectTriggerPreference, triggerID buildnotification.TriggerID) (buildnotification.ProjectTriggerPreference, bool) {
	for _, pref := range prefs {
		if pref.TriggerID == triggerID {
			return pref, true
		}
	}
	return buildnotification.ProjectTriggerPreference{}, false
}

func parseBuildID(event messaging.Event) (uuid.UUID, error) {
	buildID := strings.TrimSpace(stringPayload(event.Payload, "build_id"))
	if buildID == "" {
		return uuid.Nil, fmt.Errorf("missing build_id")
	}
	id, err := uuid.Parse(buildID)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func containsInApp(channels []buildnotification.Channel) bool {
	for _, channel := range channels {
		if channel == buildnotification.ChannelInApp {
			return true
		}
	}
	return false
}

func containsEmail(channels []buildnotification.Channel) bool {
	for _, channel := range channels {
		if channel == buildnotification.ChannelEmail {
			return true
		}
	}
	return false
}

func dedupeUserIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func stringPayload(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return s
}

func defaultMessageForTrigger(triggerID buildnotification.TriggerID, buildID uuid.UUID) string {
	switch triggerID {
	case buildnotification.TriggerBuildStarted:
		return fmt.Sprintf("Build %s has started.", buildID.String())
	case buildnotification.TriggerBuildCompleted:
		return fmt.Sprintf("Build %s completed successfully.", buildID.String())
	case buildnotification.TriggerBuildFailed:
		return fmt.Sprintf("Build %s failed.", buildID.String())
	case buildnotification.TriggerBuildCancelled:
		return fmt.Sprintf("Build %s was cancelled.", buildID.String())
	case buildnotification.TriggerBuildRetryStarted:
		return fmt.Sprintf("Build %s retry started.", buildID.String())
	case buildnotification.TriggerBuildRetryFailed:
		return fmt.Sprintf("Build %s retry failed.", buildID.String())
	case buildnotification.TriggerBuildRetrySucceeded:
		return fmt.Sprintf("Build %s retry succeeded.", buildID.String())
	case buildnotification.TriggerBuildRecovered:
		return fmt.Sprintf("Build %s recovered and resumed.", buildID.String())
	case buildnotification.TriggerPreflightBlocked:
		return fmt.Sprintf("Build %s was blocked by preflight checks.", buildID.String())
	case buildnotification.TriggerBuildScheduledQueued:
		return fmt.Sprintf("Scheduled run queued for build %s.", buildID.String())
	case buildnotification.TriggerBuildScheduledFailed:
		return fmt.Sprintf("Scheduled run failed for build %s.", buildID.String())
	case buildnotification.TriggerBuildScheduledNoOp:
		return fmt.Sprintf("Scheduled run skipped for build %s.", buildID.String())
	default:
		return fmt.Sprintf("Build %s status changed.", buildID.String())
	}
}

func (s *EventSubscriber) logWarn(msg string, fields ...zap.Field) {
	if s.logger != nil {
		s.logger.Warn(msg, fields...)
	}
}

func (s *EventSubscriber) logDebug(msg string, fields ...zap.Field) {
	if s.logger != nil {
		s.logger.Debug(msg, fields...)
	}
}

func RegisterEventSubscriber(bus messaging.EventBus, subscriber *EventSubscriber) func() {
	if bus == nil || subscriber == nil {
		return func() {}
	}
	unsubscribers := []func(){
		bus.Subscribe(messaging.EventTypeBuildStarted, subscriber.HandleBuildEvent),
		bus.Subscribe(messaging.EventTypeBuildExecutionCompleted, subscriber.HandleBuildEvent),
		bus.Subscribe(messaging.EventTypeBuildExecutionFailed, subscriber.HandleBuildEvent),
		bus.Subscribe(messaging.EventTypeBuildExecutionStatusUpdate, subscriber.HandleBuildEvent),
	}
	return func() {
		for _, unsubscribe := range unsubscribers {
			unsubscribe()
		}
	}
}
