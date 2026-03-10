package imageimportnotifications

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	buildnotifications "github.com/srikarm/image-factory/internal/application/buildnotifications"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type DeliveryRepository interface {
	ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error)
	ListSystemAdminUserIDs(ctx context.Context) ([]uuid.UUID, error)
	ListSecurityReviewerUserIDs(ctx context.Context) ([]uuid.UUID, error)
	ListUserEmailsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]string, error)
	InsertInAppNotifications(ctx context.Context, rows []buildnotifications.InAppNotificationRow) error
	TryClaimImageImportNotification(ctx context.Context, tenantID, userID uuid.UUID, eventType, idempotencyKey string) (bool, error)
}

type EmailSender interface {
	SendBuildNotificationEmailWithCC(
		ctx context.Context,
		tenantID uuid.UUID,
		toEmail string,
		ccEmail string,
		templateType string,
		templateData map[string]interface{},
		fallbackSubject string,
		fallbackBody string,
	) error
}

type NotificationRealtimePublisher interface {
	BroadcastNotificationEvent(tenantID, userID uuid.UUID, eventType string, notificationID *uuid.UUID, metadata map[string]interface{})
}

type EventSubscriber struct {
	deliveryRepo DeliveryRepository
	emailSender  EmailSender
	logger       *zap.Logger
	realtime     NotificationRealtimePublisher
	config       Config
}

type Config struct {
	AllowMissingIdempotencyKey bool
}

func NewEventSubscriber(deliveryRepo DeliveryRepository, logger *zap.Logger) *EventSubscriber {
	return NewEventSubscriberWithConfig(deliveryRepo, logger, Config{})
}

func NewEventSubscriberWithConfig(deliveryRepo DeliveryRepository, logger *zap.Logger, cfg Config) *EventSubscriber {
	return &EventSubscriber{
		deliveryRepo: deliveryRepo,
		logger:       logger,
		config:       cfg,
	}
}

func (s *EventSubscriber) SetEmailSender(sender EmailSender) {
	if s == nil {
		return
	}
	s.emailSender = sender
}

func (s *EventSubscriber) SetRealtimePublisher(publisher NotificationRealtimePublisher) {
	if s == nil {
		return
	}
	s.realtime = publisher
}

func (s *EventSubscriber) HandleImportEvent(ctx context.Context, event messaging.Event) {
	if s == nil || s.deliveryRepo == nil {
		return
	}
	// Event handlers run asynchronously; decouple downstream IO from caller cancellation.
	workCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tenantID, err := parseUUID(event.TenantID)
	if err != nil {
		s.logWarn("image import notification skipped due to invalid tenant_id", zap.String("tenant_id", event.TenantID), zap.Error(err))
		return
	}
	resourceType, resourceIDKey := eventResourceTypeAndIDKey(event.Type)
	resourceID, err := parseUUID(stringPayload(event.Payload, resourceIDKey))
	if err != nil {
		s.logWarn("notification skipped due to invalid resource id", zap.String("event_type", event.Type), zap.String("resource_id_key", resourceIDKey), zap.Error(err))
		return
	}
	requesterID, requesterErr := parseUUID(stringPayload(event.Payload, "requested_by_user_id"))
	if requesterErr != nil {
		requesterID = uuid.Nil
	}
	failureClass := normalizeFailureClass(stringPayload(event.Payload, "failure_class"))
	securityReviewEvent := isSecurityReviewEvent(event)

	recipientSet := make(map[uuid.UUID]struct{})
	primaryRecipients := make([]uuid.UUID, 0)
	if securityReviewEvent {
		reviewerIDs, reviewerErr := s.deliveryRepo.ListSecurityReviewerUserIDs(workCtx)
		if reviewerErr != nil {
			s.logWarn("failed to resolve security reviewer recipients for image import notification",
				zap.String("tenant_id", tenantID.String()),
				zap.String("event_type", event.Type),
				zap.Error(reviewerErr))
			return
		}
		for _, id := range reviewerIDs {
			if id == uuid.Nil {
				continue
			}
			if _, exists := recipientSet[id]; exists {
				continue
			}
			recipientSet[id] = struct{}{}
			primaryRecipients = append(primaryRecipients, id)
		}
		if len(primaryRecipients) == 0 {
			systemAdminUserIDs, adminErr := s.deliveryRepo.ListSystemAdminUserIDs(workCtx)
			if adminErr != nil {
				s.logWarn("failed to resolve system admin fallback recipients for security review notification",
					zap.String("event_type", event.Type),
					zap.Error(adminErr))
				return
			}
			for _, id := range systemAdminUserIDs {
				if id == uuid.Nil {
					continue
				}
				if _, exists := recipientSet[id]; exists {
					continue
				}
				recipientSet[id] = struct{}{}
				primaryRecipients = append(primaryRecipients, id)
			}
		}
		if len(primaryRecipients) == 0 && requesterID != uuid.Nil {
			recipientSet[requesterID] = struct{}{}
			primaryRecipients = append(primaryRecipients, requesterID)
			s.logWarn("security reviewer recipients not configured; falling back to requester only",
				zap.String("tenant_id", tenantID.String()),
				zap.String("event_type", event.Type),
				zap.String("requester_id", requesterID.String()))
		} else if len(primaryRecipients) > 0 && len(reviewerIDs) == 0 {
			s.logWarn("security reviewer recipients not configured; falling back to system admins",
				zap.String("tenant_id", tenantID.String()),
				zap.String("event_type", event.Type),
				zap.Int("fallback_system_admin_count", len(primaryRecipients)))
		}
	} else {
		adminUserIDs, err := s.deliveryRepo.ListTenantAdminUserIDs(workCtx, tenantID)
		if err != nil {
			s.logWarn("failed to resolve tenant admin recipients for image import notification", zap.String("tenant_id", tenantID.String()), zap.Error(err))
			return
		}
		for _, id := range adminUserIDs {
			if id == uuid.Nil {
				continue
			}
			if _, exists := recipientSet[id]; exists {
				continue
			}
			recipientSet[id] = struct{}{}
			primaryRecipients = append(primaryRecipients, id)
		}
		if requesterID != uuid.Nil {
			recipientSet[requesterID] = struct{}{}
		}
		if shouldEscalateFailureToSystemAdmins(event.Type, failureClass) {
			systemAdminUserIDs, adminErr := s.deliveryRepo.ListSystemAdminUserIDs(workCtx)
			if adminErr != nil {
				s.logWarn("failed to resolve system admin escalation recipients for image import failure",
					zap.String("tenant_id", tenantID.String()),
					zap.String("event_type", event.Type),
					zap.String("failure_class", failureClass),
					zap.Error(adminErr))
				return
			}
			for _, id := range systemAdminUserIDs {
				if id == uuid.Nil {
					continue
				}
				if _, exists := recipientSet[id]; exists {
					continue
				}
				recipientSet[id] = struct{}{}
				primaryRecipients = append(primaryRecipients, id)
			}
		}
	}
	if !securityReviewEvent && requesterID != uuid.Nil {
		recipientSet[requesterID] = struct{}{}
	}
	if len(recipientSet) == 0 {
		return
	}
	idempotencyKey := strings.TrimSpace(stringPayload(event.Payload, "idempotency_key"))
	if idempotencyKey == "" {
		if !s.config.AllowMissingIdempotencyKey {
			s.logWarn("image import notification skipped due to missing idempotency_key",
				zap.String("event_type", event.Type),
				zap.String("resource_id", resourceID.String()),
			)
			return
		}
		idempotencyKey = strings.TrimSpace(event.ID)
	}
	if idempotencyKey == "" {
		s.logWarn("image import notification skipped due to empty fallback idempotency_key",
			zap.String("event_type", event.Type),
			zap.String("resource_id", resourceID.String()),
		)
		return
	}

	title, message, notificationType, ok := mapEventToNotification(event)
	if !ok {
		return
	}
	recipients := make([]uuid.UUID, 0, len(recipientSet))
	for id := range recipientSet {
		recipients = append(recipients, id)
	}
	sort.Slice(recipients, func(i, j int) bool {
		return recipients[i].String() < recipients[j].String()
	})

	rows := make([]buildnotifications.InAppNotificationRow, 0, len(recipients))
	for _, userID := range recipients {
		claimed, claimErr := s.deliveryRepo.TryClaimImageImportNotification(workCtx, tenantID, userID, event.Type, idempotencyKey)
		if claimErr != nil {
			s.logWarn("failed to claim image import notification idempotency key",
				zap.String("event_type", event.Type),
				zap.String("tenant_id", tenantID.String()),
				zap.String("user_id", userID.String()),
				zap.String("idempotency_key", idempotencyKey),
				zap.Error(claimErr),
			)
			return
		}
		if !claimed {
			continue
		}
		rows = append(rows, buildnotifications.InAppNotificationRow{
			ID:                  uuid.New(),
			UserID:              userID,
			TenantID:            tenantID,
			Title:               title,
			Message:             message,
			NotificationType:    notificationType,
			RelatedResourceType: resourceType,
			RelatedResourceID:   &resourceID,
			Channel:             string(buildnotification.ChannelInApp),
		})
	}
	if len(rows) == 0 {
		return
	}
	if err := s.deliveryRepo.InsertInAppNotifications(workCtx, rows); err != nil {
		s.logWarn("failed to persist image import in-app notifications", zap.String("event_type", event.Type), zap.Error(err))
		return
	}
	if s.realtime != nil {
		for _, row := range rows {
			notificationID := row.ID
			s.realtime.BroadcastNotificationEvent(
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
	if s.emailSender != nil {
		emailRecipientUserIDs := recipients
		if securityReviewEvent {
			emailRecipientUserIDs = primaryRecipients
		}
		emails, err := s.deliveryRepo.ListUserEmailsByIDs(workCtx, emailRecipientUserIDs)
		if err != nil {
			s.logWarn("failed to resolve recipient emails for image import notification",
				zap.String("event_type", event.Type),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			return
		}
		requesterEmail := ""
		if securityReviewEvent && requesterID != uuid.Nil {
			requesterEmails, reqErr := s.deliveryRepo.ListUserEmailsByIDs(workCtx, []uuid.UUID{requesterID})
			if reqErr != nil {
				s.logWarn("failed to resolve requester email for cc",
					zap.String("event_type", event.Type),
					zap.String("tenant_id", tenantID.String()),
					zap.String("requester_id", requesterID.String()),
					zap.Error(reqErr))
			} else if len(requesterEmails) > 0 {
				requesterEmail = strings.TrimSpace(requesterEmails[0])
			}
		}
		baseTemplateData := buildTemplateDataForEmail(event, title, message)
		for _, email := range emails {
			templateData := make(map[string]interface{}, len(baseTemplateData)+2)
			for k, v := range baseTemplateData {
				templateData[k] = v
			}
			templateData["UserEmail"] = email
			templateData["UserName"] = userNameFromEmail(email)
			ccEmail := ""
			if securityReviewEvent && !strings.EqualFold(strings.TrimSpace(email), requesterEmail) {
				ccEmail = requesterEmail
			}
			if sendErr := s.emailSender.SendBuildNotificationEmailWithCC(
				workCtx,
				tenantID,
				email,
				ccEmail,
				notificationType,
				templateData,
				title,
				message,
			); sendErr != nil {
				s.logWarn("failed to enqueue image import notification email",
					zap.String("event_type", event.Type),
					zap.String("tenant_id", tenantID.String()),
					zap.String("recipient_email", email),
					zap.Error(sendErr))
			}
		}
	}
}

func RegisterEventSubscriber(bus messaging.EventBus, subscriber *EventSubscriber) func() {
	if bus == nil || subscriber == nil {
		return func() {}
	}
	unsubscribes := []func(){
		bus.Subscribe(messaging.EventTypeExternalImageImportApprovalRequested, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeExternalImageImportApproved, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeExternalImageImportRejected, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeExternalImageImportDispatchFailed, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeExternalImageImportCompleted, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeExternalImageImportQuarantined, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeExternalImageImportFailed, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRRegistrationRequested, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRRegistrationApproved, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRRegistrationRejected, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRRegistrationSuspended, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRRegistrationReactivated, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRRegistrationRevalidated, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRLifecycleExpiring, subscriber.HandleImportEvent),
		bus.Subscribe(messaging.EventTypeEPRLifecycleExpired, subscriber.HandleImportEvent),
	}
	return func() {
		for _, unsub := range unsubscribes {
			if unsub != nil {
				unsub()
			}
		}
	}
}

func mapEventToNotification(event messaging.Event) (title string, message string, notificationType string, ok bool) {
	status := strings.TrimSpace(stringPayload(event.Payload, "status"))
	reference := strings.TrimSpace(stringPayload(event.Payload, "source_image_ref"))
	requestType := strings.TrimSpace(strings.ToLower(stringPayload(event.Payload, "request_type")))
	if reference == "" {
		reference = strings.TrimSpace(stringPayload(event.Payload, "internal_image_ref"))
	}
	if reference == "" {
		reference = "image"
	}
	requestNoun := "Import request"
	if requestType == "scan" {
		requestNoun = "On-demand scan request"
	}
	baseMsg := fmt.Sprintf("%s for %s", requestNoun, reference)
	errorMessage := strings.TrimSpace(stringPayload(event.Payload, "message"))
	failureCode := strings.TrimSpace(strings.ToLower(stringPayload(event.Payload, "failure_code")))

	switch event.Type {
	case messaging.EventTypeExternalImageImportApprovalRequested:
		if requestType == "scan" {
			return "On-Demand Scan Queued", baseMsg + " has been queued for processing.", "external_image_import_approval_requested", true
		}
		return "Quarantine Approval Requested", baseMsg + " is waiting for approval.", "external_image_import_approval_requested", true
	case messaging.EventTypeExternalImageImportApproved:
		if requestType == "scan" {
			return "On-Demand Scan Dispatching", baseMsg + " was accepted and will be dispatched.", "external_image_import_approved", true
		}
		return "Quarantine Request Approved", baseMsg + " was approved and will be dispatched.", "external_image_import_approved", true
	case messaging.EventTypeExternalImageImportRejected:
		if errorMessage == "" {
			errorMessage = "Request was rejected by approver"
		}
		if requestType == "scan" {
			return "On-Demand Scan Rejected", errorMessage, "external_image_import_rejected", true
		}
		return "Quarantine Request Rejected", errorMessage, "external_image_import_rejected", true
	case messaging.EventTypeExternalImageImportDispatchFailed:
		if errorMessage == "" {
			errorMessage = "Quarantine import dispatch failed"
		}
		if hint := failureCodeHint(failureCode); hint != "" {
			errorMessage = errorMessage + " (" + hint + ")"
		}
		if requestType == "scan" {
			if errorMessage == "Quarantine import dispatch failed" {
				errorMessage = "On-demand scan dispatch failed"
			}
			return "On-Demand Scan Dispatch Failed", errorMessage, "external_image_import_dispatch_failed", true
		}
		return "Quarantine Import Dispatch Failed", errorMessage, "external_image_import_dispatch_failed", true
	case messaging.EventTypeExternalImageImportCompleted:
		if status == "" {
			status = "success"
		}
		if requestType == "scan" {
			return "On-Demand Scan Completed", baseMsg + " completed with status " + status + ".", "external_image_import_completed", true
		}
		return "Quarantine Import Completed", baseMsg + " completed with status " + status + ".", "external_image_import_completed", true
	case messaging.EventTypeExternalImageImportQuarantined:
		if requestType == "scan" {
			return "On-Demand Scan Completed", baseMsg + " completed with policy decision quarantine.", "external_image_import_quarantined", true
		}
		return "Image Quarantined", baseMsg + " completed and was quarantined by policy.", "external_image_import_quarantined", true
	case messaging.EventTypeExternalImageImportFailed:
		if errorMessage == "" {
			errorMessage = "Quarantine import failed"
		}
		if hint := failureCodeHint(failureCode); hint != "" {
			errorMessage = errorMessage + " (" + hint + ")"
		}
		if requestType == "scan" {
			if errorMessage == "Quarantine import failed" {
				errorMessage = "On-demand scan failed"
			}
			return "On-Demand Scan Failed", errorMessage, "external_image_import_failed", true
		}
		return "Quarantine Import Failed", errorMessage, "external_image_import_failed", true
	case messaging.EventTypeEPRRegistrationRequested:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Requested", "EPR registration request for " + eprRecordID + " is waiting for security review.", "epr_registration_requested", true
	case messaging.EventTypeEPRRegistrationApproved:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Approved", "EPR registration request for " + eprRecordID + " has been approved. You can now submit quarantine requests.", "epr_registration_approved", true
	case messaging.EventTypeEPRRegistrationRejected:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		if errorMessage == "" {
			errorMessage = "EPR registration request was rejected by security reviewer"
		}
		return "EPR Registration Rejected", eprRecordID + ": " + errorMessage, "epr_registration_rejected", true
	case messaging.EventTypeEPRRegistrationSuspended:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Suspended", eprRecordID + " has been suspended and can no longer be used for quarantine intake.", "epr_registration_suspended", true
	case messaging.EventTypeEPRRegistrationReactivated:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Reactivated", eprRecordID + " has been reactivated and is eligible for quarantine intake.", "epr_registration_reactivated", true
	case messaging.EventTypeEPRRegistrationRevalidated:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Revalidated", eprRecordID + " has been revalidated and remains eligible for quarantine intake.", "epr_registration_revalidated", true
	case messaging.EventTypeEPRLifecycleExpiring:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Expiring", eprRecordID + " is entering expiring state and requires reviewer attention.", "epr_registration_expiring", true
	case messaging.EventTypeEPRLifecycleExpired:
		eprRecordID := strings.TrimSpace(stringPayload(event.Payload, "epr_record_id"))
		if eprRecordID == "" {
			eprRecordID = "EPR record"
		}
		return "EPR Registration Expired", eprRecordID + " has expired and now blocks quarantine intake until reactivated.", "epr_registration_expired", true
	default:
		return "", "", "", false
	}
}

func failureCodeHint(code string) string {
	switch strings.TrimSpace(strings.ToLower(code)) {
	case "dispatcher_unavailable":
		return "no eligible dispatcher available"
	case "dispatch_timeout":
		return "dispatcher timeout"
	case "dispatch_error":
		return "dispatcher execution error"
	case "auth_error":
		return "authentication failure"
	case "connectivity_error":
		return "connectivity issue"
	case "policy_blocked", "quarantined_by_policy":
		return "policy blocked"
	case "runtime_failed":
		return "runtime failure"
	default:
		return ""
	}
}

func normalizeFailureClass(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

func shouldEscalateFailureToSystemAdmins(eventType, failureClass string) bool {
	switch eventType {
	case messaging.EventTypeExternalImageImportDispatchFailed, messaging.EventTypeExternalImageImportFailed:
		switch normalizeFailureClass(failureClass) {
		case "", "dispatch", "auth", "connectivity", "runtime":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func eventResourceTypeAndIDKey(eventType string) (resourceType string, idKey string) {
	switch eventType {
	case messaging.EventTypeEPRRegistrationRequested,
		messaging.EventTypeEPRRegistrationApproved,
		messaging.EventTypeEPRRegistrationRejected,
		messaging.EventTypeEPRRegistrationSuspended,
		messaging.EventTypeEPRRegistrationReactivated,
		messaging.EventTypeEPRRegistrationRevalidated,
		messaging.EventTypeEPRLifecycleExpiring,
		messaging.EventTypeEPRLifecycleExpired:
		return "epr_registration_request", "epr_registration_request_id"
	default:
		return "external_image_import", "external_image_import_id"
	}
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, err
	}
	return parsed, nil
}

func stringPayload(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func payloadAsString(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int32:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func buildTemplateDataForEmail(event messaging.Event, title, message string) map[string]interface{} {
	payload := event.Payload
	reference := strings.TrimSpace(stringPayload(payload, "source_image_ref"))
	if reference == "" {
		reference = strings.TrimSpace(stringPayload(payload, "internal_image_ref"))
	}
	if reference == "" {
		reference = "image"
	}
	eprRecordID := strings.TrimSpace(stringPayload(payload, "epr_record_id"))
	if eprRecordID == "" {
		eprRecordID = strings.TrimSpace(stringPayload(payload, "sor_record_id"))
	}
	failureCode := strings.TrimSpace(strings.ToLower(stringPayload(payload, "failure_code")))
	failureHint := failureCodeHint(failureCode)
	return map[string]interface{}{
		"NotificationTitle": title,
		"Message":           message,
		"Status":            strings.TrimSpace(stringPayload(payload, "status")),
		"RequestType":       strings.TrimSpace(stringPayload(payload, "request_type")),
		"SourceImageRef":    reference,
		"EPRRecordID":       eprRecordID,
		"ProductName":       strings.TrimSpace(stringPayload(payload, "product_name")),
		"TechnologyName":    strings.TrimSpace(stringPayload(payload, "technology_name")),
		"DashboardURL":      strings.TrimSpace(stringPayload(payload, "dashboard_url")),
		"FailureClass":      normalizeFailureClass(stringPayload(payload, "failure_class")),
		"FailureCode":       failureCode,
		"FailureHint":       failureHint,
		"DispatchAttempt":   payloadAsString(payload, "dispatch_attempt"),
	}
}

func userNameFromEmail(email string) string {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return "User"
	}
	if at := strings.Index(trimmed, "@"); at > 0 {
		return trimmed[:at]
	}
	return trimmed
}

func isSecurityReviewEvent(event messaging.Event) bool {
	switch event.Type {
	case messaging.EventTypeEPRRegistrationRequested,
		messaging.EventTypeEPRLifecycleExpiring,
		messaging.EventTypeEPRLifecycleExpired,
		messaging.EventTypeExternalImageImportApprovalRequested:
		return true
	case messaging.EventTypeExternalImageImportCompleted:
		requestType := strings.TrimSpace(strings.ToLower(stringPayload(event.Payload, "request_type")))
		status := strings.TrimSpace(strings.ToLower(stringPayload(event.Payload, "status")))
		return requestType == "quarantine" && (status == "" || status == "success")
	default:
		return false
	}
}

func (s *EventSubscriber) logWarn(msg string, fields ...zap.Field) {
	if s != nil && s.logger != nil {
		s.logger.Warn(msg, fields...)
	}
}
