package build

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

// PublishBuildCreated logs the build created event
func (p *NoOpEventPublisher) PublishBuildCreated(ctx context.Context, event *BuildCreated) error {
	p.logger.Info("Build created audit event",
		zap.String("event_type", "build_created"),
		zap.String("build_id", event.BuildID().String()),
		zap.String("build_name", event.Manifest().Name),
		zap.String("build_type", string(event.Manifest().Type)),
		zap.String("tenant_id", event.TenantID().String()),
		zap.Time("occurred_at", event.OccurredAt()),
		zap.String("action", "create"),
		zap.String("resource_type", "build"),
	)

	// TODO: In production, this would:
	// 1. Store audit event in audit_logs table
	// 2. Send to build event stream (NATS, Kafka, etc.)
	// 3. Trigger build pipeline notifications

	return nil
}

// PublishBuildStarted logs the build started event
func (p *NoOpEventPublisher) PublishBuildStarted(ctx context.Context, event *BuildStarted) error {
	p.logger.Info("Build started audit event",
		zap.String("event_type", "build_started"),
		zap.String("build_id", event.BuildID().String()),
		zap.String("tenant_id", event.TenantID().String()),
		zap.Time("occurred_at", event.OccurredAt()),
		zap.String("action", "start"),
		zap.String("resource_type", "build"),
	)

	// TODO: In production, this would:
	// 1. Update build status in monitoring systems
	// 2. Send notifications to build subscribers
	// 3. Initialize build metrics collection

	return nil
}

// PublishBuildCompleted logs the build completed event
func (p *NoOpEventPublisher) PublishBuildCompleted(ctx context.Context, event *BuildCompleted) error {
	result := event.Result()
	p.logger.Info("Build completed audit event",
		zap.String("event_type", "build_completed"),
		zap.String("build_id", event.BuildID().String()),
		zap.String("tenant_id", event.TenantID().String()),
		zap.Time("occurred_at", event.OccurredAt()),
		zap.String("action", "complete"),
		zap.String("resource_type", "build"),
		zap.String("image_id", result.ImageID),
		zap.Int64("image_size", result.Size),
		zap.Duration("duration", result.Duration),
		zap.Int("artifacts_count", len(result.Artifacts)),
	)

	// TODO: In production, this would:
	// 1. Store final build results and artifacts
	// 2. Send completion notifications
	// 3. Update build metrics and analytics
	// 4. Trigger post-build processes (scanning, deployment, etc.)

	return nil
}

// PublishBuildStatusUpdated logs the build status update event.
func (p *NoOpEventPublisher) PublishBuildStatusUpdated(ctx context.Context, event *BuildStatusUpdated) error {
	p.logger.Info("Build status updated audit event",
		zap.String("event_type", "build_status_updated"),
		zap.String("build_id", event.BuildID().String()),
		zap.String("tenant_id", event.TenantID().String()),
		zap.String("status", event.Status()),
		zap.String("message", event.Message()),
		zap.Any("metadata", event.Metadata()),
		zap.Time("occurred_at", event.OccurredAt()),
	)
	return nil
}
