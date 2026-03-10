package worker

import "testing"

func TestWorkerMetricsCountersAndStatuses(t *testing.T) {
	m := NewWorkerMetrics(2)

	m.IncrementEmailsSent()
	m.IncrementEmailsFailed()
	m.IncrementEmailsRetried()
	m.UpdateQueueDepth(7)
	m.UpdateLastHealthCheck()

	if m.GetEmailsSent() != 1 || m.GetEmailsFailed() != 1 || m.GetEmailsRetried() != 1 {
		t.Fatalf("unexpected counters: sent=%d failed=%d retried=%d", m.GetEmailsSent(), m.GetEmailsFailed(), m.GetEmailsRetried())
	}
	if m.GetQueueDepth() != 7 {
		t.Fatalf("expected queue depth 7, got %d", m.GetQueueDepth())
	}
	if m.GetLastHealthCheck().IsZero() {
		t.Fatal("expected last health check timestamp")
	}

	if m.GetWorkerStatus(0) != WorkerStatusIdle {
		t.Fatalf("expected default worker 0 idle, got %s", m.GetWorkerStatus(0))
	}
	m.SetWorkerStatus(1, WorkerStatusProcessing)
	if m.GetWorkerStatus(1) != WorkerStatusProcessing {
		t.Fatalf("expected worker 1 processing, got %s", m.GetWorkerStatus(1))
	}

	s := m.Snapshot()
	if s.EmailsSent != 1 || s.QueueDepth != 7 {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
}
