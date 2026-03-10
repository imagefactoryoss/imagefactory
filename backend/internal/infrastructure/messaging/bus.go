package messaging

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrEmptyEventType = errors.New("event type is required")

// Event represents a domain event envelope.
type Event struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	ActorID       string                 `json:"actor_id,omitempty"`
	Source        string                 `json:"source,omitempty"`
	OccurredAt    time.Time              `json:"occurred_at"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SchemaVersion string                 `json:"schema_version,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

// Handler handles events published on the bus.
type Handler func(context.Context, Event)

// EventBus provides publish/subscribe for domain events.
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType string, handler Handler) (unsubscribe func())
}

type handlerEntry struct {
	id string
	fn Handler
}

// InProcessBus is an in-memory event bus using Go channels and goroutines.
type InProcessBus struct {
	mu       sync.RWMutex
	handlers map[string][]handlerEntry
	logger   *zap.Logger
}

// NewInProcessBus creates an in-process event bus.
func NewInProcessBus(logger *zap.Logger) *InProcessBus {
	return &InProcessBus{
		handlers: make(map[string][]handlerEntry),
		logger:   logger,
	}
}

// Publish emits an event to all subscribers of the given type and wildcard subscribers.
func (b *InProcessBus) Publish(ctx context.Context, event Event) error {
	if event.Type == "" {
		return ErrEmptyEventType
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	b.mu.RLock()
	typedHandlers := append([]handlerEntry{}, b.handlers[event.Type]...)
	wildcardHandlers := append([]handlerEntry{}, b.handlers["*"]...)
	b.mu.RUnlock()

	dispatch := func(entry handlerEntry) {
		defer func() {
			if recoverErr := recover(); recoverErr != nil && b.logger != nil {
				b.logger.Warn("Recovered from event handler panic",
					zap.String("event_type", event.Type),
					zap.String("handler_id", entry.id),
					zap.Any("panic", recoverErr),
				)
			}
		}()
		entry.fn(ctx, event)
	}

	for _, handler := range append(typedHandlers, wildcardHandlers...) {
		go dispatch(handler)
	}

	return nil
}

// Subscribe registers a handler for an event type. Use "*" to receive all events.
func (b *InProcessBus) Subscribe(eventType string, handler Handler) (unsubscribe func()) {
	entry := handlerEntry{
		id: uuid.NewString(),
		fn: handler,
	}

	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], entry)
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		handlers := b.handlers[eventType]
		for i := range handlers {
			if handlers[i].id == entry.id {
				b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
	}
}
