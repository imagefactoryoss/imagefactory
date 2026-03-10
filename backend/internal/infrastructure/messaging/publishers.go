package messaging

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/tenant"
)

// BuildEventPublisher publishes build events onto the bus.
type BuildEventPublisher struct {
	bus    EventBus
	source string
	schema string
}

func NewBuildEventPublisher(bus EventBus, source, schemaVersion string) *BuildEventPublisher {
	return &BuildEventPublisher{bus: bus, source: source, schema: schemaVersion}
}

func (p *BuildEventPublisher) PublishBuildCreated(ctx context.Context, event *build.BuildCreated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeBuildCreated,
		TenantID:      event.TenantID().String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"build_id":   event.BuildID().String(),
			"build_name": event.Manifest().Name,
			"build_type": string(event.Manifest().Type),
		},
	})
}

func (p *BuildEventPublisher) PublishBuildStarted(ctx context.Context, event *build.BuildStarted) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeBuildStarted,
		TenantID:      event.TenantID().String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"build_id": event.BuildID().String(),
		},
	})
}

func (p *BuildEventPublisher) PublishBuildCompleted(ctx context.Context, event *build.BuildCompleted) error {
	if p == nil || p.bus == nil {
		return nil
	}
	result := event.Result()
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeBuildCompleted,
		TenantID:      event.TenantID().String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"build_id":   event.BuildID().String(),
			"image_id":   result.ImageID,
			"image_size": result.Size,
			"duration":   result.Duration.Seconds(),
		},
	})
}

func (p *BuildEventPublisher) PublishBuildStatusUpdated(ctx context.Context, event *build.BuildStatusUpdated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	payload := map[string]interface{}{
		"build_id": event.BuildID().String(),
		"status":   event.Status(),
		"message":  event.Message(),
		"metadata": event.Metadata(),
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeBuildExecutionStatusUpdate,
		TenantID:      event.TenantID().String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload:       payload,
	})
}

// TenantEventPublisher publishes tenant events onto the bus.
type TenantEventPublisher struct {
	bus    EventBus
	source string
	schema string
}

func NewTenantEventPublisher(bus EventBus, source, schemaVersion string) *TenantEventPublisher {
	return &TenantEventPublisher{bus: bus, source: source, schema: schemaVersion}
}

func (p *TenantEventPublisher) PublishTenantCreated(ctx context.Context, event *tenant.TenantCreated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeTenantCreated,
		TenantID:      event.TenantID().String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"tenant_id":   event.TenantID().String(),
			"tenant_name": event.TenantName(),
		},
	})
}

func (p *TenantEventPublisher) PublishTenantActivated(ctx context.Context, event *tenant.TenantActivated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeTenantActivated,
		TenantID:      event.TenantID().String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"tenant_id": event.TenantID().String(),
		},
	})
}

// InfrastructureEventPublisher publishes infrastructure provider events onto the bus.
type InfrastructureEventPublisher struct {
	bus    EventBus
	source string
	schema string
}

func NewInfrastructureEventPublisher(bus EventBus, source, schemaVersion string) *InfrastructureEventPublisher {
	return &InfrastructureEventPublisher{bus: bus, source: source, schema: schemaVersion}
}

func (p *InfrastructureEventPublisher) PublishProviderCreated(ctx context.Context, event *infrastructure.ProviderCreated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeInfraProviderCreated,
		TenantID:      event.TenantID.String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt,
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"provider_id":   event.ProviderID.String(),
			"provider_type": event.ProviderType,
			"name":          event.Name,
			"created_by":    event.CreatedBy.String(),
		},
	})
}

func (p *InfrastructureEventPublisher) PublishProviderUpdated(ctx context.Context, event *infrastructure.ProviderUpdated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeInfraProviderUpdated,
		TenantID:      event.TenantID.String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt,
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"provider_id": event.ProviderID.String(),
			"updated_by":  event.UpdatedBy.String(),
		},
	})
}

func (p *InfrastructureEventPublisher) PublishProviderDeleted(ctx context.Context, event *infrastructure.ProviderDeleted) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeInfraProviderDeleted,
		TenantID:      event.TenantID.String(),
		Source:        p.source,
		OccurredAt:    event.OccurredAt,
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"provider_id": event.ProviderID.String(),
			"deleted_by":  event.DeletedBy.String(),
		},
	})
}

// BuildStatusBroadcaster publishes build status updates onto the bus.
type BuildStatusBroadcaster struct {
	bus    EventBus
	source string
	schema string
}

func NewBuildStatusBroadcaster(bus EventBus, source, schemaVersion string) *BuildStatusBroadcaster {
	return &BuildStatusBroadcaster{bus: bus, source: source, schema: schemaVersion}
}

func (b *BuildStatusBroadcaster) BroadcastBuildEvent(
	tenantID uuid.UUID,
	eventType, buildID, buildNumber, projectID, status, message string,
	duration int,
	metadata map[string]interface{},
) {
	if b == nil || b.bus == nil || eventType == "" {
		return
	}

	busEventType := eventType
	switch eventType {
	case EventTypeBuildCompleted:
		busEventType = EventTypeBuildExecutionCompleted
	case EventTypeBuildFailed:
		busEventType = EventTypeBuildExecutionFailed
	case EventTypeBuildStatusUpdate:
		busEventType = EventTypeBuildExecutionStatusUpdate
	}

	payload := map[string]interface{}{
		"build_id":     buildID,
		"build_number": buildNumber,
		"project_id":   projectID,
		"status":       status,
		"message":      message,
		"duration":     duration,
		"metadata":     metadata,
	}

	_ = b.bus.Publish(context.Background(), Event{
		Type:          busEventType,
		TenantID:      tenantID.String(),
		Source:        b.source,
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: b.schema,
		Payload:       payload,
	})
}

// ProjectEventPublisher publishes project events onto the bus.
type ProjectEventPublisher struct {
	bus    EventBus
	source string
	schema string
}

func NewProjectEventPublisher(bus EventBus, source, schemaVersion string) *ProjectEventPublisher {
	return &ProjectEventPublisher{bus: bus, source: source, schema: schemaVersion}
}

func (p *ProjectEventPublisher) PublishProjectCreated(ctx context.Context, event *project.ProjectCreated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeProjectCreated,
		TenantID:      event.TenantID().String(),
		ActorID:       actorID(event.ActorID()),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"project_id":   event.ProjectID().String(),
			"project_name": event.ProjectName(),
			"visibility":   event.Visibility(),
			"git_repo":     event.GitRepo(),
			"git_branch":   event.GitBranch(),
			"git_provider": event.GitProvider(),
		},
	})
}

func (p *ProjectEventPublisher) PublishProjectUpdated(ctx context.Context, event *project.ProjectUpdated) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeProjectUpdated,
		TenantID:      event.TenantID().String(),
		ActorID:       actorID(event.ActorID()),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"project_id":   event.ProjectID().String(),
			"project_name": event.ProjectName(),
			"visibility":   event.Visibility(),
			"git_repo":     event.GitRepo(),
			"git_branch":   event.GitBranch(),
			"git_provider": event.GitProvider(),
		},
	})
}

func (p *ProjectEventPublisher) PublishProjectDeleted(ctx context.Context, event *project.ProjectDeleted) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, Event{
		Type:          EventTypeProjectDeleted,
		TenantID:      event.TenantID().String(),
		ActorID:       actorID(event.ActorID()),
		Source:        p.source,
		OccurredAt:    event.OccurredAt(),
		SchemaVersion: p.schema,
		Payload: map[string]interface{}{
			"project_id": event.ProjectID().String(),
		},
	})
}

func actorID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
