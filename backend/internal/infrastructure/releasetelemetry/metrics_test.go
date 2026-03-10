package releasetelemetry

import (
	"testing"

	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
)

func TestMetrics_RecordAndSnapshot(t *testing.T) {
	metrics := NewMetrics()
	metrics.Record(messaging.EventTypeQuarantineReleaseRequested)
	metrics.Record(messaging.EventTypeQuarantineReleased)
	metrics.Record(messaging.EventTypeQuarantineReleaseFailed)
	metrics.Record(messaging.EventTypeQuarantineReleaseFailed)

	snapshot := metrics.Snapshot()
	if snapshot.Requested != 1 {
		t.Fatalf("expected requested=1, got %d", snapshot.Requested)
	}
	if snapshot.Released != 1 {
		t.Fatalf("expected released=1, got %d", snapshot.Released)
	}
	if snapshot.Failed != 2 {
		t.Fatalf("expected failed=2, got %d", snapshot.Failed)
	}
	if snapshot.Total != 4 {
		t.Fatalf("expected total=4, got %d", snapshot.Total)
	}
}

func TestMetrics_RecordIgnoresEmptyEventType(t *testing.T) {
	metrics := NewMetrics()
	metrics.Record("")
	metrics.Record("  ")

	snapshot := metrics.Snapshot()
	if snapshot.Total != 0 {
		t.Fatalf("expected total=0, got %d", snapshot.Total)
	}
}
