package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type stubAuditService struct {
	logEventCalled      bool
	logUserActionCalled bool
	logSystemCalled     bool
	lastEvent           *AuditEvent
	lastTenantID        uuid.UUID
	lastUserID          uuid.UUID
	lastDetails         map[string]interface{}
}

func (s *stubAuditService) LogEvent(_ context.Context, event *AuditEvent) error {
	s.logEventCalled = true
	s.lastEvent = event
	s.lastDetails = event.Details
	return nil
}

func (s *stubAuditService) LogUserAction(_ context.Context, tenantID, userID uuid.UUID, _ AuditEventType, _, _, _ string, details map[string]interface{}) error {
	s.logUserActionCalled = true
	s.lastTenantID = tenantID
	s.lastUserID = userID
	s.lastDetails = details
	return nil
}

func (s *stubAuditService) LogSystemAction(_ context.Context, tenantID uuid.UUID, _ AuditEventType, _, _, _ string, details map[string]interface{}) error {
	s.logSystemCalled = true
	s.lastTenantID = tenantID
	s.lastDetails = details
	return nil
}

func (s *stubAuditService) QueryEvents(context.Context, *uuid.UUID, AuditEventFilter, int, int) ([]*AuditEvent, error) {
	return nil, nil
}

func (s *stubAuditService) CountEvents(context.Context, *uuid.UUID, AuditEventFilter) (int, error) {
	return 0, nil
}

func TestRegisterEventBusSubscriberUsesUserActionWhenActorPresent(t *testing.T) {
	bus := messaging.NewInProcessBus(zap.NewNop())
	service := &stubAuditService{}
	unsubscribe := RegisterEventBusSubscriber(bus, service, zap.NewNop())
	defer unsubscribe()

	tenantID := uuid.New()
	actorID := uuid.New()

	err := bus.Publish(context.Background(), messaging.Event{
		Type:     messaging.EventTypeBuildCreated,
		TenantID: tenantID.String(),
		ActorID:  actorID.String(),
		Payload:  map[string]interface{}{"build_id": "b1"},
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for !service.logUserActionCalled && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !service.logUserActionCalled {
		t.Fatal("expected LogUserAction to be called")
	}
	if service.lastTenantID != tenantID || service.lastUserID != actorID {
		t.Fatal("expected tenant and actor IDs to be forwarded")
	}
}

func TestRegisterEventBusSubscriberUsesSystemActionWhenNoActor(t *testing.T) {
	bus := messaging.NewInProcessBus(zap.NewNop())
	service := &stubAuditService{}
	unsubscribe := RegisterEventBusSubscriber(bus, service, zap.NewNop())
	defer unsubscribe()

	tenantID := uuid.New()
	err := bus.Publish(context.Background(), messaging.Event{
		Type:     messaging.EventTypeTenantActivated,
		TenantID: tenantID.String(),
		Payload:  map[string]interface{}{"tenant_code": "acme"},
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for !service.logSystemCalled && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !service.logSystemCalled {
		t.Fatal("expected LogSystemAction to be called")
	}
	if service.lastTenantID != tenantID {
		t.Fatal("expected tenant ID to be forwarded")
	}
}

func TestRegisterEventBusSubscriberFallsBackToLogEventAndMetadata(t *testing.T) {
	bus := messaging.NewInProcessBus(zap.NewNop())
	service := &stubAuditService{}
	unsubscribe := RegisterEventBusSubscriber(bus, service, zap.NewNop())
	defer unsubscribe()

	err := bus.Publish(context.Background(), messaging.Event{
		Type:          messaging.EventTypeProjectCreated,
		CorrelationID: "corr-1",
		RequestID:     "req-1",
		TraceID:       "trace-1",
		Source:        "test",
		SchemaVersion: "v1",
		Payload:       map[string]interface{}{"project": "demo"},
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for !service.logEventCalled && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !service.logEventCalled || service.lastEvent == nil {
		t.Fatal("expected LogEvent to be called")
	}
	if service.lastDetails["project"] != "demo" {
		t.Fatal("expected payload details to be preserved")
	}
	if service.lastDetails["correlation_id"] != "corr-1" {
		t.Fatal("expected correlation metadata in details")
	}
}

func TestParseUUIDAndMetadataHelpers(t *testing.T) {
	if got := parseUUID("not-a-uuid"); got != uuid.Nil {
		t.Fatal("expected invalid UUID to map to Nil")
	}

	id := uuid.New()
	if got := parseUUID(id.String()); got != id {
		t.Fatal("expected valid UUID to parse")
	}

	details := withEventMetadata(messaging.Event{
		Payload:       map[string]interface{}{"k": "v"},
		RequestID:     "r1",
		CorrelationID: "c1",
	})
	if details["k"] != "v" || details["request_id"] != "r1" || details["correlation_id"] != "c1" {
		t.Fatal("unexpected metadata mapping")
	}
}
