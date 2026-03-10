package imageimportnotifications

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	buildnotifications "github.com/srikarm/image-factory/internal/application/buildnotifications"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type deliveryRepoStub struct {
	adminUserIDs       []uuid.UUID
	systemAdminUserIDs []uuid.UUID
	reviewerIDs        []uuid.UUID
	userEmails         []string
	userEmailByID      map[uuid.UUID]string
	rows               []buildnotifications.InAppNotificationRow
	claims             map[string]struct{}
}

func (s *deliveryRepoStub) ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID{}, s.adminUserIDs...), nil
}

func (s *deliveryRepoStub) ListSystemAdminUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	return append([]uuid.UUID{}, s.systemAdminUserIDs...), nil
}

func (s *deliveryRepoStub) ListSecurityReviewerUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	return append([]uuid.UUID{}, s.reviewerIDs...), nil
}

func (s *deliveryRepoStub) ListUserEmailsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]string, error) {
	if len(s.userEmailByID) > 0 {
		out := make([]string, 0, len(userIDs))
		for _, id := range userIDs {
			if email, ok := s.userEmailByID[id]; ok {
				out = append(out, email)
			}
		}
		return out, nil
	}
	return append([]string{}, s.userEmails...), nil
}

func (s *deliveryRepoStub) InsertInAppNotifications(ctx context.Context, rows []buildnotifications.InAppNotificationRow) error {
	s.rows = append(s.rows, rows...)
	return nil
}

func (s *deliveryRepoStub) TryClaimImageImportNotification(ctx context.Context, tenantID, userID uuid.UUID, eventType, idempotencyKey string) (bool, error) {
	if s.claims == nil {
		s.claims = make(map[string]struct{})
	}
	key := tenantID.String() + "|" + userID.String() + "|" + eventType + "|" + idempotencyKey
	if _, exists := s.claims[key]; exists {
		return false, nil
	}
	s.claims[key] = struct{}{}
	return true, nil
}

func TestEventSubscriber_HandlesTerminalFailedEvent(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	adminID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{adminUserIDs: []uuid.UUID{adminID}}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"message":                  "quarantine pipeline failed",
			"failure_code":             "runtime_failed",
			"idempotency_key":          importID.String() + ":failed",
		},
	})

	if len(repo.rows) != 2 {
		t.Fatalf("expected 2 notifications (requester + admin), got %d", len(repo.rows))
	}
	seenRequester := false
	seenAdmin := false
	for _, row := range repo.rows {
		if row.TenantID != tenantID {
			t.Fatalf("unexpected tenant id %s", row.TenantID)
		}
		if row.RelatedResourceID == nil || *row.RelatedResourceID != importID {
			t.Fatalf("expected related resource id %s", importID)
		}
		if row.NotificationType != "external_image_import_failed" {
			t.Fatalf("expected failure notification type, got %s", row.NotificationType)
		}
		if row.Message != "quarantine pipeline failed (runtime failure)" {
			t.Fatalf("expected actionable failure message, got %q", row.Message)
		}
		if row.UserID == requesterID {
			seenRequester = true
		}
		if row.UserID == adminID {
			seenAdmin = true
		}
	}
	if !seenRequester || !seenAdmin {
		t.Fatalf("expected requester and admin recipients, got requester=%v admin=%v", seenRequester, seenAdmin)
	}
}

func TestRegisterEventSubscriber_SubscribesEventTypes(t *testing.T) {
	bus := messaging.NewInProcessBus(zap.NewNop())
	repo := &deliveryRepoStub{reviewerIDs: []uuid.UUID{uuid.New()}}
	sub := NewEventSubscriber(repo, zap.NewNop())
	unsub := RegisterEventSubscriber(bus, sub)
	defer unsub()

	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()
	bus.Publish(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportApprovalRequested,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "pending",
			"idempotency_key":          importID.String() + ":approval_requested",
		},
	})

	// In-process bus handlers execute asynchronously.
	for i := 0; i < 50 && len(repo.rows) == 0; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	if len(repo.rows) == 0 {
		t.Fatalf("expected at least one notification row from subscribed event")
	}
}

func TestEventSubscriber_HandlesDispatchFailedEvent(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportDispatchFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"message":                  "dispatch timed out",
			"failure_code":             "dispatch_timeout",
			"dispatch_attempt":         2,
			"idempotency_key":          importID.String() + ":dispatch_failed:2",
		},
	})
	if len(repo.rows) != 1 {
		t.Fatalf("expected one notification row for requester, got %d", len(repo.rows))
	}
	if repo.rows[0].NotificationType != "external_image_import_dispatch_failed" {
		t.Fatalf("expected dispatch failed notification type, got %s", repo.rows[0].NotificationType)
	}
	if repo.rows[0].Message != "dispatch timed out (dispatcher timeout)" {
		t.Fatalf("expected actionable dispatch failure message, got %q", repo.rows[0].Message)
	}
}

func TestEventSubscriber_CompletedQuarantineRoutesToSecurityReviewers(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	reviewerID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{reviewerIDs: []uuid.UUID{reviewerID}}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportCompleted,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"request_type":             "quarantine",
			"status":                   "success",
			"idempotency_key":          importID.String() + ":completed",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one reviewer notification row, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != reviewerID {
		t.Fatalf("expected reviewer recipient %s, got %s", reviewerID, repo.rows[0].UserID)
	}
	if repo.rows[0].NotificationType != "external_image_import_completed" {
		t.Fatalf("expected completed notification type, got %s", repo.rows[0].NotificationType)
	}
}

func TestEventSubscriber_DispatchFailedEscalatesToSystemAdmins(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	adminID := uuid.New()
	systemAdminID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{
		adminUserIDs:       []uuid.UUID{adminID},
		systemAdminUserIDs: []uuid.UUID{systemAdminID},
	}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportDispatchFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"failure_class":            "dispatch",
			"failure_code":             "dispatch_timeout",
			"message":                  "dispatch timed out",
			"dispatch_attempt":         2,
			"idempotency_key":          importID.String() + ":dispatch_failed:2",
		},
	})

	if len(repo.rows) != 3 {
		t.Fatalf("expected three notifications (requester + tenant admin + system admin), got %d", len(repo.rows))
	}
	seenRequester := false
	seenAdmin := false
	seenSystemAdmin := false
	for _, row := range repo.rows {
		if row.UserID == requesterID {
			seenRequester = true
		}
		if row.UserID == adminID {
			seenAdmin = true
		}
		if row.UserID == systemAdminID {
			seenSystemAdmin = true
		}
	}
	if !seenRequester || !seenAdmin || !seenSystemAdmin {
		t.Fatalf("expected requester/admin/system-admin recipients, got requester=%v admin=%v system_admin=%v", seenRequester, seenAdmin, seenSystemAdmin)
	}
}

func TestEventSubscriber_FailedPolicyDoesNotEscalateToSystemAdmins(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	adminID := uuid.New()
	systemAdminID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{
		adminUserIDs:       []uuid.UUID{adminID},
		systemAdminUserIDs: []uuid.UUID{systemAdminID},
	}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"failure_class":            "policy",
			"failure_code":             "policy_blocked",
			"message":                  "policy denied import",
			"idempotency_key":          importID.String() + ":failed",
		},
	})

	if len(repo.rows) != 2 {
		t.Fatalf("expected two notifications (requester + tenant admin), got %d", len(repo.rows))
	}
	seenRequester := false
	seenAdmin := false
	seenSystemAdmin := false
	for _, row := range repo.rows {
		if row.UserID == requesterID {
			seenRequester = true
		}
		if row.UserID == adminID {
			seenAdmin = true
		}
		if row.UserID == systemAdminID {
			seenSystemAdmin = true
		}
	}
	if !seenRequester || !seenAdmin || seenSystemAdmin {
		t.Fatalf("expected requester/admin recipients without system-admin escalation, got requester=%v admin=%v system_admin=%v", seenRequester, seenAdmin, seenSystemAdmin)
	}
}

func TestEventSubscriber_DuplicateIdempotencyKey_NoDuplicateNotifications(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	event := messaging.Event{
		Type:     messaging.EventTypeExternalImageImportDispatchFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"message":                  "dispatch timed out",
			"dispatch_attempt":         2,
			"idempotency_key":          importID.String() + ":dispatch_failed:2",
		},
	}
	sub.HandleImportEvent(context.Background(), event)
	sub.HandleImportEvent(context.Background(), event)

	if len(repo.rows) != 1 {
		t.Fatalf("expected one notification row after duplicate replay, got %d", len(repo.rows))
	}
}

func TestEventSubscriber_DuplicateReplayAcrossSubscriberRestart_NoDuplicateNotifications(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{}
	event := messaging.Event{
		Type:     messaging.EventTypeExternalImageImportDispatchFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"message":                  "dispatch timed out",
			"dispatch_attempt":         2,
			"idempotency_key":          importID.String() + ":dispatch_failed:2",
		},
	}

	subFirst := NewEventSubscriber(repo, zap.NewNop())
	subFirst.HandleImportEvent(context.Background(), event)

	subAfterRestart := NewEventSubscriber(repo, zap.NewNop())
	subAfterRestart.HandleImportEvent(context.Background(), event)

	if len(repo.rows) != 1 {
		t.Fatalf("expected one notification row after replay across restart, got %d", len(repo.rows))
	}
}

type emailSenderStub struct {
	calls          int
	ccCalls        []string
	lastTemplate   map[string]interface{}
	templateByCall []map[string]interface{}
}

func (s *emailSenderStub) SendBuildNotificationEmailWithCC(
	ctx context.Context,
	tenantID uuid.UUID,
	toEmail string,
	ccEmail string,
	templateType string,
	templateData map[string]interface{},
	fallbackSubject string,
	fallbackBody string,
) error {
	s.calls++
	s.ccCalls = append(s.ccCalls, ccEmail)
	cloned := make(map[string]interface{}, len(templateData))
	for k, v := range templateData {
		cloned[k] = v
	}
	s.lastTemplate = cloned
	s.templateByCall = append(s.templateByCall, cloned)
	return nil
}

func TestEventSubscriber_QueuesEmailNotificationsWhenEmailSenderConfigured(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{
		userEmails: []string{"requester@example.com"},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportApproved,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"source_image_ref":         "ghcr.io/acme/app:1.0.0",
			"status":                   "approved",
			"idempotency_key":          importID.String() + ":approved",
		},
	})

	if emailSender.calls != 1 {
		t.Fatalf("expected 1 queued email, got %d", emailSender.calls)
	}
}

func TestEventSubscriber_EPRRequestedTargetsSecurityReviewersAndCCsRequester(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	reviewerID := uuid.New()
	requestID := uuid.New()

	repo := &deliveryRepoStub{
		reviewerIDs: []uuid.UUID{reviewerID},
		userEmailByID: map[uuid.UUID]string{
			reviewerID:  "reviewer@imagefactory.local",
			requesterID: "requester@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeEPRRegistrationRequested,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"epr_registration_request_id": requestID.String(),
			"requested_by_user_id":        requesterID.String(),
			"epr_record_id":               "EPR-20260303-ABCD1234",
			"status":                      "pending",
			"idempotency_key":             requestID.String() + ":pending",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one in-app notification for reviewer, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != reviewerID {
		t.Fatalf("expected reviewer recipient %s, got %s", reviewerID, repo.rows[0].UserID)
	}
	if emailSender.calls != 1 {
		t.Fatalf("expected one email notification, got %d", emailSender.calls)
	}
	if len(emailSender.ccCalls) != 1 || emailSender.ccCalls[0] != "requester@imagefactory.local" {
		t.Fatalf("expected requester cc, got %#v", emailSender.ccCalls)
	}
}

func TestEventSubscriber_EPRRequestedFallsBackToRequesterWhenNoSecurityReviewersConfigured(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	requestID := uuid.New()

	repo := &deliveryRepoStub{
		userEmailByID: map[uuid.UUID]string{
			requesterID: "requester@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeEPRRegistrationRequested,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"epr_registration_request_id": requestID.String(),
			"requested_by_user_id":        requesterID.String(),
			"epr_record_id":               "EPR-20260303-ABCD1234",
			"status":                      "pending",
			"idempotency_key":             requestID.String() + ":pending",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one fallback in-app notification for requester, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != requesterID {
		t.Fatalf("expected fallback requester recipient %s, got %s", requesterID, repo.rows[0].UserID)
	}
	if emailSender.calls != 1 {
		t.Fatalf("expected one fallback email notification, got %d", emailSender.calls)
	}
	if len(emailSender.ccCalls) != 1 || emailSender.ccCalls[0] != "" {
		t.Fatalf("expected no cc for fallback requester notification, got %#v", emailSender.ccCalls)
	}
}

func TestEventSubscriber_EPRRequestedFallsBackToSystemAdminsWhenNoSecurityReviewersConfigured(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	systemAdminID := uuid.New()
	requestID := uuid.New()

	repo := &deliveryRepoStub{
		systemAdminUserIDs: []uuid.UUID{systemAdminID},
		userEmailByID: map[uuid.UUID]string{
			systemAdminID: "admin@imagefactory.local",
			requesterID:   "requester@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeEPRRegistrationRequested,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"epr_registration_request_id": requestID.String(),
			"requested_by_user_id":        requesterID.String(),
			"epr_record_id":               "EPR-20260303-ABCD1234",
			"status":                      "pending",
			"idempotency_key":             requestID.String() + ":pending",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one in-app notification for system admin fallback, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != systemAdminID {
		t.Fatalf("expected system admin fallback recipient %s, got %s", systemAdminID, repo.rows[0].UserID)
	}
	if emailSender.calls != 1 {
		t.Fatalf("expected one fallback email notification, got %d", emailSender.calls)
	}
	if len(emailSender.ccCalls) != 1 || emailSender.ccCalls[0] != "requester@imagefactory.local" {
		t.Fatalf("expected requester cc for system admin fallback, got %#v", emailSender.ccCalls)
	}
}

func TestEventSubscriber_ApprovalRequestedFallsBackToSystemAdminsWhenNoSecurityReviewersConfigured(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	systemAdminID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{
		systemAdminUserIDs: []uuid.UUID{systemAdminID},
		userEmailByID: map[uuid.UUID]string{
			systemAdminID: "admin@imagefactory.local",
			requesterID:   "requester@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportApprovalRequested,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"source_image_ref":         "docker.io/library/nginx:1.27",
			"status":                   "pending",
			"idempotency_key":          importID.String() + ":approval_requested",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one in-app notification for system admin fallback, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != systemAdminID {
		t.Fatalf("expected system admin fallback recipient %s, got %s", systemAdminID, repo.rows[0].UserID)
	}
	if repo.rows[0].NotificationType != "external_image_import_approval_requested" {
		t.Fatalf("expected approval requested notification type, got %s", repo.rows[0].NotificationType)
	}
	if emailSender.calls != 1 {
		t.Fatalf("expected one fallback email notification, got %d", emailSender.calls)
	}
	if len(emailSender.ccCalls) != 1 || emailSender.ccCalls[0] != "requester@imagefactory.local" {
		t.Fatalf("expected requester cc for system admin fallback, got %#v", emailSender.ccCalls)
	}
}

func TestEventSubscriber_ApprovalRequestedFallsBackToRequesterWhenNoSecurityReviewersOrSystemAdmins(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{
		userEmailByID: map[uuid.UUID]string{
			requesterID: "requester@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportApprovalRequested,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"source_image_ref":         "docker.io/library/nginx:1.27",
			"status":                   "pending",
			"idempotency_key":          importID.String() + ":approval_requested",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one in-app notification for requester fallback, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != requesterID {
		t.Fatalf("expected requester fallback recipient %s, got %s", requesterID, repo.rows[0].UserID)
	}
	if emailSender.calls != 1 {
		t.Fatalf("expected one fallback email notification, got %d", emailSender.calls)
	}
	if len(emailSender.ccCalls) != 1 || emailSender.ccCalls[0] != "" {
		t.Fatalf("expected no cc for requester fallback, got %#v", emailSender.ccCalls)
	}
}

func TestEventSubscriber_EPRLifecycleExpiringTargetsSecurityReviewers(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	reviewerID := uuid.New()
	requestID := uuid.New()

	repo := &deliveryRepoStub{
		reviewerIDs: []uuid.UUID{reviewerID},
		userEmailByID: map[uuid.UUID]string{
			reviewerID:  "reviewer@imagefactory.local",
			requesterID: "requester@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeEPRLifecycleExpiring,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"epr_registration_request_id": requestID.String(),
			"requested_by_user_id":        requesterID.String(),
			"epr_record_id":               "EPR-TEST-1",
			"status":                      "approved",
			"idempotency_key":             requestID.String() + ":expiring",
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("expected one in-app notification for reviewer, got %d", len(repo.rows))
	}
	if repo.rows[0].UserID != reviewerID {
		t.Fatalf("expected reviewer recipient %s, got %s", reviewerID, repo.rows[0].UserID)
	}
	if emailSender.calls != 1 {
		t.Fatalf("expected one email notification, got %d", emailSender.calls)
	}
	if len(emailSender.ccCalls) != 1 || emailSender.ccCalls[0] != "requester@imagefactory.local" {
		t.Fatalf("expected requester cc for expiring notification, got %#v", emailSender.ccCalls)
	}
}

func TestEventSubscriber_EPRRegistrationSuspendedTargetsTenantAdminsAndRequester(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	adminID := uuid.New()
	requestID := uuid.New()

	repo := &deliveryRepoStub{
		adminUserIDs: []uuid.UUID{adminID},
		userEmailByID: map[uuid.UUID]string{
			adminID:     "admin@imagefactory.local",
			requesterID: "requester@imagefactory.local",
		},
	}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeEPRRegistrationSuspended,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"epr_registration_request_id": requestID.String(),
			"requested_by_user_id":        requesterID.String(),
			"epr_record_id":               "EPR-TEST-2",
			"status":                      "approved",
			"idempotency_key":             requestID.String() + ":suspended",
		},
	})

	if len(repo.rows) != 2 {
		t.Fatalf("expected two in-app notifications (admin + requester), got %d", len(repo.rows))
	}
	seenAdmin := false
	seenRequester := false
	for _, row := range repo.rows {
		if row.UserID == adminID {
			seenAdmin = true
		}
		if row.UserID == requesterID {
			seenRequester = true
		}
		if row.NotificationType != "epr_registration_suspended" {
			t.Fatalf("expected suspended notification type, got %s", row.NotificationType)
		}
	}
	if !seenAdmin || !seenRequester {
		t.Fatalf("expected admin and requester recipients, got admin=%v requester=%v", seenAdmin, seenRequester)
	}
}

func TestEventSubscriber_DispatchFailedEmailTemplateIncludesFailureMetadata(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	systemAdminID := uuid.New()
	importID := uuid.New()

	repo := &deliveryRepoStub{
		systemAdminUserIDs: []uuid.UUID{systemAdminID},
		userEmailByID: map[uuid.UUID]string{
			systemAdminID: "admin@imagefactory.local",
		},
	}
	emailSender := &emailSenderStub{}
	sub := NewEventSubscriber(repo, zap.NewNop())
	sub.SetEmailSender(emailSender)
	sub.HandleImportEvent(context.Background(), messaging.Event{
		Type:     messaging.EventTypeExternalImageImportDispatchFailed,
		TenantID: tenantID.String(),
		Payload: map[string]interface{}{
			"external_image_import_id": importID.String(),
			"requested_by_user_id":     requesterID.String(),
			"status":                   "failed",
			"failure_class":            "dispatch",
			"failure_code":             "dispatch_timeout",
			"dispatch_attempt":         4,
			"message":                  "dispatch timed out",
			"idempotency_key":          importID.String() + ":dispatch_failed:4",
		},
	})

	if emailSender.calls != 1 {
		t.Fatalf("expected one queued email, got %d", emailSender.calls)
	}
	if emailSender.lastTemplate == nil {
		t.Fatalf("expected template payload to be captured")
	}
	if got := emailSender.lastTemplate["FailureClass"]; got != "dispatch" {
		t.Fatalf("expected FailureClass=dispatch, got %#v", got)
	}
	if got := emailSender.lastTemplate["FailureCode"]; got != "dispatch_timeout" {
		t.Fatalf("expected FailureCode=dispatch_timeout, got %#v", got)
	}
	if got := emailSender.lastTemplate["FailureHint"]; got != "dispatcher timeout" {
		t.Fatalf("expected FailureHint=dispatcher timeout, got %#v", got)
	}
	if got := emailSender.lastTemplate["DispatchAttempt"]; got != "4" {
		t.Fatalf("expected DispatchAttempt=4, got %#v", got)
	}
}
