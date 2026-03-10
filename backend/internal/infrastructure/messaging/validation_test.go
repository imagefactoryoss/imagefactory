package messaging

import (
	"context"
	"testing"
)

func TestValidatingBusRejectsMissingFields(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type:    EventTypeBuildCreated,
		Payload: map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected validation error for missing required fields")
	}
}

func TestValidatingBusAppliesSchemaVersion(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type: EventTypeTenantCreated,
		Payload: map[string]interface{}{
			"tenant_id":   "tenant",
			"tenant_name": "Acme",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatingBusAcceptsQuarantineReleaseDriftDetectedPayload(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type: EventTypeQuarantineReleaseDriftDetected,
		Payload: map[string]interface{}{
			"external_image_import_id": "11111111-1111-1111-1111-111111111111",
			"tenant_id":                "22222222-2222-2222-2222-222222222222",
			"release_state":            "withdrawn",
			"source_image_digest":      "sha256:abc",
			"internal_image_ref":       "registry.local/q/app@sha256:abc",
			"released_at":              "2026-03-05T00:00:00Z",
			"idempotency_key":          "idempotency",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
