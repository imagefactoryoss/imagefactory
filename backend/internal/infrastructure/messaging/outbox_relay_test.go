package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type relayStoreStub struct {
	mu             sync.Mutex
	messages       []OutboxMessage
	published      []uuid.UUID
	failed         []uuid.UUID
	lastNextRetry  map[uuid.UUID]time.Time
	lastClaimOwner string
	lastClaimLease time.Duration
}

func (s *relayStoreStub) Enqueue(ctx context.Context, event Event) error { return nil }
func (s *relayStoreStub) ClaimDue(ctx context.Context, limit int, claimOwner string, claimLease time.Duration) ([]OutboxMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastClaimOwner = claimOwner
	s.lastClaimLease = claimLease
	out := make([]OutboxMessage, len(s.messages))
	copy(out, s.messages)
	return out, nil
}
func (s *relayStoreStub) MarkPublished(ctx context.Context, id uuid.UUID, claimOwner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published = append(s.published, id)
	return nil
}
func (s *relayStoreStub) MarkFailed(ctx context.Context, id uuid.UUID, claimOwner string, lastError string, nextAttemptAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed = append(s.failed, id)
	if s.lastNextRetry == nil {
		s.lastNextRetry = make(map[uuid.UUID]time.Time)
	}
	s.lastNextRetry[id] = nextAttemptAt
	return nil
}
func (s *relayStoreStub) PendingCount(ctx context.Context) (int64, error) {
	return int64(len(s.messages)), nil
}

type relayBusStub struct {
	pubErr error
}

func (b *relayBusStub) Publish(ctx context.Context, event Event) error {
	return b.pubErr
}
func (b *relayBusStub) Subscribe(eventType string, handler Handler) (unsubscribe func()) {
	return func() {}
}

func TestOutboxRelayReplayOnceMarksPublished(t *testing.T) {
	id := uuid.New()
	store := &relayStoreStub{
		messages: []OutboxMessage{
			{ID: id, Event: Event{ID: id.String(), Type: "build.execution.failed"}},
		},
	}
	relay := NewOutboxRelay(store, &relayBusStub{}, OutboxRelayConfig{BatchSize: 10}, nil)

	processed, err := relay.ReplayOnce(context.Background())
	if err != nil {
		t.Fatalf("ReplayOnce returned error: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed=1, got %d", processed)
	}
	if len(store.published) != 1 || store.published[0] != id {
		t.Fatalf("expected published id %s, got %+v", id.String(), store.published)
	}
	if store.lastClaimOwner == "" {
		t.Fatal("expected claim owner to be set")
	}
	if store.lastClaimLease <= 0 {
		t.Fatalf("expected positive claim lease, got %s", store.lastClaimLease)
	}
}

func TestOutboxRelayReplayOnceMarksFailedWithBackoff(t *testing.T) {
	id := uuid.New()
	store := &relayStoreStub{
		messages: []OutboxMessage{
			{ID: id, Event: Event{ID: id.String(), Type: "build.execution.failed"}, PublishAttempts: 2},
		},
	}
	relay := NewOutboxRelay(store, &relayBusStub{pubErr: errors.New("nats down")}, OutboxRelayConfig{
		BatchSize:   10,
		BaseBackoff: 2 * time.Second,
		MaxBackoff:  20 * time.Second,
	}, nil)

	processed, err := relay.ReplayOnce(context.Background())
	if err != nil {
		t.Fatalf("ReplayOnce returned error: %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected processed=0 on publish failure, got %d", processed)
	}
	if len(store.failed) != 1 || store.failed[0] != id {
		t.Fatalf("expected failed id %s, got %+v", id.String(), store.failed)
	}
	if next := store.lastNextRetry[id]; next.IsZero() {
		t.Fatal("expected next retry time to be set")
	}
	snapshot := relay.Snapshot()
	if snapshot.ReplayFailureTotal != 1 {
		t.Fatalf("expected replay failure total 1, got %d", snapshot.ReplayFailureTotal)
	}
}
