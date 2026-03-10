package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/notification"
)

type mockQueue struct {
	task        *notification.EmailTask
	dequeueErr  error
	markStatus  notification.EmailStatus
	markErrMsg  string
	retriedID   uuid.UUID
	retryCalled bool
	retryWhen   time.Time
	queueDepth  int64
}

func (m *mockQueue) Enqueue(ctx context.Context, task *notification.EmailTask) error { return nil }
func (m *mockQueue) Dequeue(ctx context.Context) (*notification.EmailTask, error) {
	return m.task, m.dequeueErr
}
func (m *mockQueue) MarkProcessed(ctx context.Context, taskID uuid.UUID, status notification.EmailStatus, errorMsg string) error {
	m.markStatus = status
	m.markErrMsg = errorMsg
	return nil
}
func (m *mockQueue) Retry(ctx context.Context, taskID uuid.UUID, nextAttemptAt time.Time) error {
	m.retryCalled = true
	m.retriedID = taskID
	m.retryWhen = nextAttemptAt
	return nil
}
func (m *mockQueue) Cancel(ctx context.Context, taskID uuid.UUID) error { return nil }
func (m *mockQueue) GetQueueStats(ctx context.Context) (*notification.QueueStats, error) {
	return &notification.QueueStats{}, nil
}
func (m *mockQueue) GetQueueDepth(ctx context.Context) (int64, error) { return m.queueDepth, nil }

type mockSender struct {
	err error
}

func (m *mockSender) Send(ctx context.Context, task *notification.EmailTask) error { return m.err }

func TestEmailWorkerProcessTaskSuccess(t *testing.T) {
	q := &mockQueue{}
	m := NewWorkerMetrics(1)
	w := NewEmailWorker(0, q, &mockSender{}, m, DefaultWorkerConfig())
	taskID := uuid.New()
	task := &notification.EmailTask{ID: taskID, ToAddress: "a@example.com", Attempts: 0, MaxAttempts: 3}

	w.processTask(context.Background(), task)
	if q.markStatus != notification.EmailStatusSent {
		t.Fatalf("expected sent status, got %s", q.markStatus)
	}
	if m.GetEmailsSent() != 1 {
		t.Fatalf("expected sent metric increment, got %d", m.GetEmailsSent())
	}
}

func TestEmailWorkerProcessTaskRetryThenFail(t *testing.T) {
	m := NewWorkerMetrics(1)
	cfg := DefaultWorkerConfig()
	cfg.MaxRetries = 2

	qRetry := &mockQueue{}
	wRetry := NewEmailWorker(0, qRetry, &mockSender{err: errors.New("smtp down")}, m, cfg)
	taskRetry := &notification.EmailTask{ID: uuid.New(), ToAddress: "a@example.com", Attempts: 0, MaxAttempts: 2}
	wRetry.processTask(context.Background(), taskRetry)
	if !qRetry.retryCalled {
		t.Fatal("expected retry to be scheduled")
	}
	if m.GetEmailsRetried() != 1 {
		t.Fatalf("expected retry metric increment, got %d", m.GetEmailsRetried())
	}

	qFail := &mockQueue{}
	wFail := NewEmailWorker(0, qFail, &mockSender{err: errors.New("smtp down")}, m, cfg)
	taskFail := &notification.EmailTask{ID: uuid.New(), ToAddress: "a@example.com", Attempts: 2, MaxAttempts: 2}
	wFail.processTask(context.Background(), taskFail)
	if qFail.markStatus != notification.EmailStatusFailed {
		t.Fatalf("expected failed status, got %s", qFail.markStatus)
	}
	if m.GetEmailsFailed() != 1 {
		t.Fatalf("expected failed metric increment, got %d", m.GetEmailsFailed())
	}
}

func TestEmailWorkerPoolStartStopAndHealthCheck(t *testing.T) {
	q := &mockQueue{queueDepth: 9}
	cfg := DefaultWorkerConfig()
	cfg.WorkerCount = 1
	cfg.PollInterval = 100 * time.Millisecond
	cfg.HealthCheckPeriod = 100 * time.Millisecond
	cfg.ShutdownTimeout = 2 * time.Second

	pool := NewEmailWorkerPool(q, &mockSender{}, cfg)
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("expected start success, got %v", err)
	}
	if !pool.IsRunning() {
		t.Fatal("expected pool running after start")
	}
	if err := pool.Start(context.Background()); err == nil {
		t.Fatal("expected error when starting already-running pool")
	}

	time.Sleep(250 * time.Millisecond)
	if pool.GetMetrics().GetQueueDepth() != 9 {
		t.Fatalf("expected queue depth from health checks, got %d", pool.GetMetrics().GetQueueDepth())
	}

	if err := pool.Stop(); err != nil {
		t.Fatalf("expected stop success, got %v", err)
	}
	if pool.IsRunning() {
		t.Fatal("expected pool stopped after stop")
	}
	if err := pool.Stop(); err == nil {
		t.Fatal("expected error when stopping non-running pool")
	}
}
