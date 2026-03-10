package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type recordingBus struct {
	mu       sync.Mutex
	events   []Event
	handlers map[string][]Handler
	pubErr   error
}

func newRecordingBus() *recordingBus {
	return &recordingBus{
		handlers: make(map[string][]Handler),
	}
}

func (b *recordingBus) Publish(ctx context.Context, event Event) error {
	if b.pubErr != nil {
		return b.pubErr
	}
	b.mu.Lock()
	b.events = append(b.events, event)
	handlers := append([]Handler{}, b.handlers[event.Type]...)
	handlers = append(handlers, b.handlers["*"]...)
	b.mu.Unlock()

	for _, handler := range handlers {
		handler(ctx, event)
	}
	return nil
}

type outboxRecorder struct {
	mu       sync.Mutex
	enqueued []Event
}

func (o *outboxRecorder) Enqueue(ctx context.Context, event Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enqueued = append(o.enqueued, event)
	return nil
}
func (o *outboxRecorder) ClaimDue(ctx context.Context, limit int, claimOwner string, claimLease time.Duration) ([]OutboxMessage, error) {
	return nil, nil
}
func (o *outboxRecorder) MarkPublished(ctx context.Context, id uuid.UUID, claimOwner string) error {
	return nil
}
func (o *outboxRecorder) MarkFailed(ctx context.Context, id uuid.UUID, claimOwner string, lastError string, nextAttemptAt time.Time) error {
	return nil
}
func (o *outboxRecorder) PendingCount(ctx context.Context) (int64, error) { return 0, nil }

func (b *recordingBus) Subscribe(eventType string, handler Handler) (unsubscribe func()) {
	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.mu.Unlock()
	return func() {}
}

func (b *recordingBus) Events() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	events := make([]Event, len(b.events))
	copy(events, b.events)
	return events
}

func TestHybridBusPublishesToBothBuses(t *testing.T) {
	local := newRecordingBus()
	external := newRecordingBus()
	bus := NewHybridBus(local, external, "local-1", nil)
	t.Cleanup(bus.Close)

	if err := bus.Publish(context.Background(), Event{Type: "project.created"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if got := len(local.Events()); got != 1 {
		t.Fatalf("expected local publish count 1, got %d", got)
	}
	if got := len(external.Events()); got != 1 {
		t.Fatalf("expected external publish count 1, got %d", got)
	}
}

func TestHybridBusBridgesExternalEventsToLocal(t *testing.T) {
	local := newRecordingBus()
	external := newRecordingBus()
	bus := NewHybridBus(local, external, "local-1", nil)
	t.Cleanup(bus.Close)

	var received int
	local.Subscribe("*", func(ctx context.Context, event Event) {
		received++
	})

	if err := external.Publish(context.Background(), Event{Type: "project.updated", Source: "remote-1"}); err != nil {
		t.Fatalf("external publish failed: %v", err)
	}

	if received != 1 {
		t.Fatalf("expected local to receive 1 bridged event, got %d", received)
	}

	if err := external.Publish(context.Background(), Event{Type: "project.updated", Source: "local-1"}); err != nil {
		t.Fatalf("external publish failed: %v", err)
	}

	if received != 1 {
		t.Fatalf("expected local to ignore local source event, got %d", received)
	}
}

func TestHybridBusExternalFailureEnqueuesOutboxAndSucceeds(t *testing.T) {
	local := newRecordingBus()
	external := newRecordingBus()
	external.pubErr = errors.New("nats unavailable")
	outbox := &outboxRecorder{}

	bus := NewHybridBusWithOutbox(local, external, outbox, "local-1", nil)
	t.Cleanup(bus.Close)

	err := bus.Publish(context.Background(), Event{Type: "build.execution.completed"})
	if err != nil {
		t.Fatalf("expected no error when outbox is configured, got %v", err)
	}

	if got := len(local.Events()); got != 1 {
		t.Fatalf("expected local publish count 1, got %d", got)
	}
	outbox.mu.Lock()
	defer outbox.mu.Unlock()
	if len(outbox.enqueued) != 1 {
		t.Fatalf("expected 1 outbox enqueue, got %d", len(outbox.enqueued))
	}
}
