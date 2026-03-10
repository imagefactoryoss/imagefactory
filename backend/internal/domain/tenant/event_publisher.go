package tenant

import (
	"context"

	"go.uber.org/zap"
)

// NoOpEventPublisher is a simple implementation that logs events but doesn't publish them
type NoOpEventPublisher struct {
	logger *zap.Logger
}

// NewNoOpEventPublisher creates a new no-op event publisher
func NewNoOpEventPublisher(logger *zap.Logger) *NoOpEventPublisher {
	return &NoOpEventPublisher{
		logger: logger,
	}
}

// PublishTenantCreated logs the tenant created event
func (p *NoOpEventPublisher) PublishTenantCreated(ctx context.Context, event *TenantCreated) error {
	p.logger.Info("Tenant created audit event",
		zap.String("event_type", "tenant_created"),
		zap.String("tenant_id", event.TenantID().String()),
		zap.String("tenant_name", event.TenantName()),
		zap.Time("occurred_at", event.OccurredAt()),
		zap.String("action", "create"),
		zap.String("resource_type", "tenant"),
		// TODO: Add user_id from context when auth is implemented
		// zap.String("user_id", getUserIDFromContext(ctx)),
	)

	// TODO: In production, this would:
	// 1. Store audit event in audit_logs table
	// 2. Send to audit event stream (Kafka, etc.)
	// 3. Trigger compliance notifications

	return nil
}

// PublishTenantActivated logs the tenant activated event
func (p *NoOpEventPublisher) PublishTenantActivated(ctx context.Context, event *TenantActivated) error {
	p.logger.Info("Tenant activated audit event",
		zap.String("event_type", "tenant_activated"),
		zap.String("tenant_id", event.TenantID().String()),
		zap.Time("occurred_at", event.OccurredAt()),
		zap.String("action", "activate"),
		zap.String("resource_type", "tenant"),
		zap.String("previous_status", "pending"),
		zap.String("new_status", "active"),
		// TODO: Add user_id from context when auth is implemented
	)

	// TODO: In production, this would:
	// 1. Store audit event in audit_logs table
	// 2. Send activation notifications
	// 3. Update tenant status in external systems

	return nil
}
