package sresmartbot

import (
	"context"
	"testing"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
)

type consumerLagProviderStub struct {
	snapshots []messaging.NATSConsumerLagSnapshot
	err       error
}

func (s consumerLagProviderStub) ConsumerLagSnapshots(ctx context.Context) ([]messaging.NATSConsumerLagSnapshot, error) {
	return s.snapshots, s.err
}

func TestInvokeMessagingTransport(t *testing.T) {
	processHealth := runtimehealth.NewStore()
	processHealth.Upsert("nats_transport_signal_runner", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: time.Now().UTC(),
		Message:      "nats transport status=connected reconnects=4 disconnects=2",
		Metrics: map[string]int64{
			"nats_reconnect_threshold":   3,
			"nats_transport_reconnects":  4,
			"nats_transport_disconnects": 2,
		},
	})

	service := &MCPService{processHealth: processHealth}
	payload, err := service.invokeMessagingTransport()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := payload["reconnects"]; got != int64(4) {
		t.Fatalf("expected reconnects=4, got %#v", got)
	}
	if got := payload["disconnects"]; got != int64(2) {
		t.Fatalf("expected disconnects=2, got %#v", got)
	}
	if got := payload["reconnect_threshold"]; got != int64(3) {
		t.Fatalf("expected reconnect_threshold=3, got %#v", got)
	}
}

func TestInvokeMessagingConsumers(t *testing.T) {
	service := &MCPService{
		consumerLagProvider: consumerLagProviderStub{
			snapshots: []messaging.NATSConsumerLagSnapshot{
				{
					Stream:          "build-events",
					Consumer:        "dispatcher",
					PendingCount:    12,
					AckPendingCount: 4,
					WaitingCount:    1,
				},
				{
					Stream:       "notifications",
					Consumer:     "email-worker",
					PendingCount: 0,
				},
			},
		},
	}

	payload, err := service.invokeMessagingConsumers(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := payload["provider"]; got != "nats" {
		t.Fatalf("expected provider=nats, got %#v", got)
	}
	if got := payload["count"]; got != 2 {
		t.Fatalf("expected count=2, got %#v", got)
	}
	if got := payload["lagging_count"]; got != 1 {
		t.Fatalf("expected lagging_count=1, got %#v", got)
	}
	if got := payload["max_pending_count"]; got != uint64(12) {
		t.Fatalf("expected max_pending_count=12, got %#v", got)
	}
	consumers, ok := payload["consumers"].([]map[string]any)
	if !ok {
		t.Fatalf("expected consumers slice, got %#v", payload["consumers"])
	}
	if len(consumers) != 2 {
		t.Fatalf("expected 2 consumers, got %d", len(consumers))
	}
	if consumers[0]["stream"] != "build-events" || consumers[0]["consumer"] != "dispatcher" {
		t.Fatalf("unexpected first consumer payload: %#v", consumers[0])
	}
}
