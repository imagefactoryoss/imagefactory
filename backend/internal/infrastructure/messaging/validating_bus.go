package messaging

import (
	"context"

	"go.uber.org/zap"
)

// ValidatingBus wraps another bus to apply schema defaults and validation.
type ValidatingBus struct {
	inner          EventBus
	schemaVersion  string
	validateEvents bool
	logger         *zap.Logger
}

func NewValidatingBus(inner EventBus, config ValidationConfig, logger *zap.Logger) *ValidatingBus {
	return &ValidatingBus{
		inner:          inner,
		schemaVersion:  config.SchemaVersion,
		validateEvents: config.ValidateEvents,
		logger:         logger,
	}
}

func (b *ValidatingBus) Publish(ctx context.Context, event Event) error {
	if event.SchemaVersion == "" && b.schemaVersion != "" {
		event.SchemaVersion = b.schemaVersion
	}
	if b.validateEvents {
		if err := validateEvent(event); err != nil {
			if b.logger != nil {
				b.logger.Warn("Event validation failed", zap.String("event_type", event.Type), zap.Error(err))
			}
			return err
		}
	}
	return b.inner.Publish(ctx, event)
}

func (b *ValidatingBus) Subscribe(eventType string, handler Handler) (unsubscribe func()) {
	return b.inner.Subscribe(eventType, handler)
}

func (b *ValidatingBus) Close() {
	if closable, ok := b.inner.(interface{ Close() }); ok {
		closable.Close()
	}
}
