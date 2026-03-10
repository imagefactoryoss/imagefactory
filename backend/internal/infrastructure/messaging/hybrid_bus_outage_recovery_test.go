package messaging

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memoryOutboxStore struct {
	mu        sync.Mutex
	messages  []OutboxMessage
	published map[uuid.UUID]bool
	failed    map[uuid.UUID]bool
}

func newMemoryOutboxStore() *memoryOutboxStore {
	return &memoryOutboxStore{
		published: map[uuid.UUID]bool{},
		failed:    map[uuid.UUID]bool{},
	}
}

func (s *memoryOutboxStore) Enqueue(ctx context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New()
	if event.ID != "" {
		if parsed, err := uuid.Parse(event.ID); err == nil {
			id = parsed
		}
	}
	event.ID = id.String()
	s.messages = append(s.messages, OutboxMessage{
		ID:              id,
		Event:           event,
		PublishAttempts: 0,
	})
	return nil
}

func (s *memoryOutboxStore) ClaimDue(ctx context.Context, limit int, claimOwner string, claimLease time.Duration) ([]OutboxMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.messages) {
		limit = len(s.messages)
	}
	out := make([]OutboxMessage, 0, limit)
	for _, msg := range s.messages {
		if s.published[msg.ID] || s.failed[msg.ID] {
			continue
		}
		out = append(out, msg)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *memoryOutboxStore) MarkPublished(ctx context.Context, id uuid.UUID, claimOwner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published[id] = true
	return nil
}

func (s *memoryOutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, claimOwner string, lastError string, nextAttemptAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed[id] = true
	return nil
}

func (s *memoryOutboxStore) PendingCount(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pending int64
	for _, msg := range s.messages {
		if s.published[msg.ID] || s.failed[msg.ID] {
			continue
		}
		pending++
	}
	return pending, nil
}

func TestHybridBusLocalDeliveryAndOutboxRecoveryAfterExternalOutage(t *testing.T) {
	local := newRecordingBus()
	external := newRecordingBus()
	external.pubErr = assertErr("nats unavailable")
	outbox := newMemoryOutboxStore()
	bus := NewHybridBusWithOutbox(local, external, outbox, "local-1", nil)
	t.Cleanup(bus.Close)

	var localReceived int
	bus.Subscribe("build.execution.completed", func(ctx context.Context, event Event) {
		localReceived++
	})

	ev := Event{
		ID:   uuid.New().String(),
		Type: "build.execution.completed",
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
		},
	}
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("expected publish success with outbox fallback, got error: %v", err)
	}
	if localReceived != 1 {
		t.Fatalf("expected local delivery during external outage, got %d", localReceived)
	}
	if pending, _ := outbox.PendingCount(context.Background()); pending != 1 {
		t.Fatalf("expected one pending outbox message during outage, got %d", pending)
	}

	external.pubErr = nil
	relay := NewOutboxRelay(outbox, external, OutboxRelayConfig{BatchSize: 10}, nil)
	processed, err := relay.ReplayOnce(context.Background())
	if err != nil {
		t.Fatalf("ReplayOnce returned error: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected replay to process 1 message, got %d", processed)
	}

	if pending, _ := outbox.PendingCount(context.Background()); pending != 0 {
		t.Fatalf("expected outbox to drain after external recovery, got pending=%d", pending)
	}
	if got := len(external.Events()); got != 1 {
		t.Fatalf("expected one external event after recovery replay, got %d", got)
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
