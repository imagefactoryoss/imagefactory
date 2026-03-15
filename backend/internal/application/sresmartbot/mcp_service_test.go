package sresmartbot

import (
	"testing"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
)

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
