package messaging

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// HybridBus publishes to both local and external buses while keeping subscriptions local.
// External events are bridged back into the local bus with source-based deduplication.
type HybridBus struct {
	local       EventBus
	external    EventBus
	outbox      OutboxStore
	localSource string
	logger      *zap.Logger
	unsubscribe func()
}

func NewHybridBus(local EventBus, external EventBus, localSource string, logger *zap.Logger) *HybridBus {
	return NewHybridBusWithOutbox(local, external, nil, localSource, logger)
}

func NewHybridBusWithOutbox(local EventBus, external EventBus, outbox OutboxStore, localSource string, logger *zap.Logger) *HybridBus {
	bus := &HybridBus{
		local:       local,
		external:    external,
		outbox:      outbox,
		localSource: localSource,
		logger:      logger,
	}

	if external != nil {
		bus.unsubscribe = external.Subscribe("*", func(ctx context.Context, event Event) {
			if localSource != "" && event.Source == localSource {
				return
			}
			if err := local.Publish(ctx, event); err != nil && logger != nil {
				logger.Warn("Failed to bridge external event into local bus",
					zap.String("event_type", event.Type),
					zap.Error(err),
				)
			}
		})
	}

	return bus
}

func (b *HybridBus) Publish(ctx context.Context, event Event) error {
	if event.Source == "" && b.localSource != "" {
		event.Source = b.localSource
	}
	if err := b.local.Publish(ctx, event); err != nil {
		return err
	}
	if b.external != nil {
		if err := b.external.Publish(ctx, event); err != nil {
			if b.outbox != nil {
				if enqueueErr := b.outbox.Enqueue(ctx, event); enqueueErr != nil {
					return fmt.Errorf("external publish failed: %v; outbox enqueue failed: %w", err, enqueueErr)
				}
				if b.logger != nil {
					b.logger.Warn("External publish failed; queued event in outbox",
						zap.String("event_type", event.Type),
						zap.String("event_id", event.ID),
						zap.Error(err))
				}
				return nil
			}
			return err
		}
	}
	return nil
}

func (b *HybridBus) Subscribe(eventType string, handler Handler) (unsubscribe func()) {
	return b.local.Subscribe(eventType, handler)
}

func (b *HybridBus) Close() {
	if b.unsubscribe != nil {
		b.unsubscribe()
	}
	if closable, ok := b.external.(interface{ Close() }); ok {
		closable.Close()
	}
	if closable, ok := b.local.(interface{ Close() }); ok {
		closable.Close()
	}
}
