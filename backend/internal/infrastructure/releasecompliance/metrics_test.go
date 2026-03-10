package releasecompliance

import "testing"

func TestMetrics_RecordAndSnapshot(t *testing.T) {
	metrics := NewMetrics()
	metrics.RecordTick(2, 9)
	metrics.RecordFailure()
	metrics.AddDetected(3)
	metrics.AddRecovered(1)

	s := metrics.Snapshot()
	if s.WatchTicksTotal != 1 {
		t.Fatalf("expected watch ticks=1, got %d", s.WatchTicksTotal)
	}
	if s.WatchFailuresTotal != 1 {
		t.Fatalf("expected watch failures=1, got %d", s.WatchFailuresTotal)
	}
	if s.DriftDetectedTotal != 3 {
		t.Fatalf("expected detected=3, got %d", s.DriftDetectedTotal)
	}
	if s.DriftRecoveredTotal != 1 {
		t.Fatalf("expected recovered=1, got %d", s.DriftRecoveredTotal)
	}
	if s.ActiveDriftCount != 2 || s.ReleasedCount != 9 {
		t.Fatalf("unexpected active/released counts: %+v", s)
	}
	if s.LastTickUnix == 0 {
		t.Fatalf("expected last tick timestamp")
	}
}
