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

func TestValidatingBusAcceptsSREFindingObservedPayload(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type: EventTypeSREFindingObserved,
		Payload: map[string]interface{}{
			"incident_id":     "11111111-1111-1111-1111-111111111111",
			"correlation_key": "provider_readiness:global",
			"domain":          "infrastructure",
			"incident_type":   "provider_readiness_degraded",
			"status":          "observed",
			"severity":        "warning",
			"finding_id":      "22222222-2222-2222-2222-222222222222",
			"signal_type":     "provider_readiness_summary",
			"signal_key":      "global",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatingBusAcceptsSREActionProposedPayload(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type: EventTypeSREActionProposed,
		Payload: map[string]interface{}{
			"incident_id":       "11111111-1111-1111-1111-111111111111",
			"correlation_key":   "tenant_asset_drift:global",
			"domain":            "application_services",
			"incident_type":     "tenant_asset_drift_detected",
			"status":            "observed",
			"action_attempt_id": "33333333-3333-3333-3333-333333333333",
			"action_key":        "reconcile_tenant_assets",
			"action_class":      "recommendation",
			"target_kind":       "tenant_asset",
			"target_ref":        "global",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatingBusAcceptsSREDetectorFindingObservedPayload(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type: EventTypeSREDetectorFindingObserved,
		Payload: map[string]interface{}{
			"correlation_key": "logs:image_pull_rate_limit",
			"domain":          "runtime_services",
			"incident_type":   "registry_pull_failure",
			"summary":         "Repeated toomanyrequests responses were detected",
			"source":          "log_detector",
			"severity":        "warning",
			"confidence":      "high",
			"finding_title":   "Docker Hub rate limit signature matched",
			"finding_message": "Matched repeated toomanyrequests log lines",
			"signal_type":     "log_signature",
			"signal_key":      "toomanyrequests",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatingBusAcceptsSREDetectorFindingRecoveredPayload(t *testing.T) {
	inner := NewInProcessBus(nil)
	bus := NewValidatingBus(inner, ValidationConfig{
		SchemaVersion:  "v1",
		ValidateEvents: true,
	}, nil)

	err := bus.Publish(context.Background(), Event{
		Type: EventTypeSREDetectorFindingRecovered,
		Payload: map[string]interface{}{
			"correlation_key": "logs:image_pull_rate_limit",
			"domain":          "runtime_services",
			"incident_type":   "registry_pull_failure",
			"summary":         "Registry pull rate limit detected recovered",
			"source":          "log_detector",
			"resolved_at":     "2026-03-14T12:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
