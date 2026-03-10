package messaging

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestInProcessBusPublishValidatesType(t *testing.T) {
	bus := NewInProcessBus(zap.NewNop())
	err := bus.Publish(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error when event type is empty")
	}
}

func TestInProcessBusDeliversEvent(t *testing.T) {
	bus := NewInProcessBus(zap.NewNop())
	received := make(chan Event, 1)

	bus.Subscribe("build.created", func(ctx context.Context, event Event) {
		received <- event
	})

	err := bus.Publish(context.Background(), Event{Type: "build.created"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-received:
		if event.Type != "build.created" {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if event.ID == "" {
			t.Fatalf("expected event ID to be set")
		}
		if event.OccurredAt.IsZero() {
			t.Fatalf("expected event OccurredAt to be set")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestInProcessBusWildcardSubscriberReceivesAll(t *testing.T) {
	bus := NewInProcessBus(zap.NewNop())
	received := make(chan string, 2)

	bus.Subscribe("*", func(ctx context.Context, event Event) {
		received <- event.Type
	})

	_ = bus.Publish(context.Background(), Event{Type: "tenant.created"})
	_ = bus.Publish(context.Background(), Event{Type: "build.started"})

	want := map[string]bool{
		"tenant.created": true,
		"build.started":  true,
	}

	deadline := time.After(2 * time.Second)
	for len(want) > 0 {
		select {
		case eventType := <-received:
			delete(want, eventType)
		case <-deadline:
			t.Fatal("timed out waiting for wildcard events")
		}
	}
}

func TestInProcessBusUnsubscribeStopsDelivery(t *testing.T) {
	bus := NewInProcessBus(zap.NewNop())
	received := make(chan Event, 1)

	unsubscribe := bus.Subscribe("build.completed", func(ctx context.Context, event Event) {
		received <- event
	})
	unsubscribe()

	_ = bus.Publish(context.Background(), Event{Type: "build.completed"})

	select {
	case <-received:
		t.Fatal("did not expect to receive event after unsubscribe")
	case <-time.After(500 * time.Millisecond):
	}
}

func TestInProcessBusHandlerPanicDoesNotBlockOthers(t *testing.T) {
	bus := NewInProcessBus(zap.NewNop())
	received := make(chan Event, 1)

	bus.Subscribe("build.started", func(ctx context.Context, event Event) {
		panic("boom")
	})
	bus.Subscribe("build.started", func(ctx context.Context, event Event) {
		received <- event
	})

	_ = bus.Publish(context.Background(), Event{Type: "build.started"})

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("expected non-panicking handler to receive event")
	}
}
