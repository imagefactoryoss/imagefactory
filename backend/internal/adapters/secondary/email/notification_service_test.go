package email

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type captureBus struct {
	lastEvent messaging.Event
}

func (b *captureBus) Publish(_ context.Context, event messaging.Event) error {
	b.lastEvent = event
	return nil
}

func (b *captureBus) Subscribe(string, messaging.Handler) (unsubscribe func()) {
	return func() {}
}

func TestRequestNotificationPublishesEvent(t *testing.T) {
	bus := &captureBus{}
	tenantID := uuid.New()
	svc := NewNotificationService(nil, bus, true, zap.NewNop(), "noreply@example.com", uuid.New(), nil)

	metadata, _ := json.Marshal(map[string]string{"k": "v"})
	req := domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   "user@example.com",
		FromEmail: "noreply@example.com",
		Subject:   "subject",
		BodyText:  "body",
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  3,
		Metadata:  metadata,
	}

	if err := svc.requestNotification(context.Background(), "user_added_to_tenant", req); err != nil {
		t.Fatalf("requestNotification returned error: %v", err)
	}

	if bus.lastEvent.Type != "notification.requested" {
		t.Fatalf("unexpected event type: %s", bus.lastEvent.Type)
	}
	if bus.lastEvent.TenantID != tenantID.String() {
		t.Fatalf("unexpected tenant id: %s", bus.lastEvent.TenantID)
	}
	if got, _ := bus.lastEvent.Payload["notification_type"].(string); got != "user_added_to_tenant" {
		t.Fatalf("unexpected notification type: %v", bus.lastEvent.Payload["notification_type"])
	}
}

func TestSMTPEnabledDefaultsToTrueWhenConfigServiceMissing(t *testing.T) {
	svc := NewNotificationService(nil, nil, false, zap.NewNop(), "noreply@example.com", uuid.New(), nil)
	if !svc.smtpEnabled(context.Background(), uuid.New()) {
		t.Fatal("expected smtpEnabled=true when systemConfigService is nil")
	}
}

func TestRenderTemplate(t *testing.T) {
	out, err := renderTemplate("Hello {{.Name}}", map[string]string{"Name": "Team"})
	if err != nil {
		t.Fatalf("renderTemplate returned error: %v", err)
	}
	if out != "Hello Team" {
		t.Fatalf("unexpected template output: %q", out)
	}

	_, err = renderTemplate("{{", map[string]string{})
	if err == nil {
		t.Fatal("expected parse error for invalid template")
	}
}
