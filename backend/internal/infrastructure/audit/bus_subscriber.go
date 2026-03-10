package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type eventAuditMapping struct {
	eventType string
	auditType AuditEventType
	resource  string
	action    string
	severity  AuditEventSeverity
	message   string
}

// RegisterEventBusSubscriber wires audit logging to the event bus.
func RegisterEventBusSubscriber(bus messaging.EventBus, service AuditService, logger *zap.Logger) (unsubscribe func()) {
	if bus == nil || service == nil {
		return func() {}
	}

	mappings := []eventAuditMapping{
		{
			eventType: messaging.EventTypeBuildCreated,
			auditType: AuditEventBuildCreate,
			resource:  "build",
			action:    "create",
			severity:  AuditSeverityInfo,
			message:   "Build created",
		},
		{
			eventType: messaging.EventTypeBuildStarted,
			auditType: AuditEventBuildStart,
			resource:  "build",
			action:    "start",
			severity:  AuditSeverityInfo,
			message:   "Build started",
		},
		{
			eventType: messaging.EventTypeBuildCompleted,
			auditType: AuditEventBuildComplete,
			resource:  "build",
			action:    "complete",
			severity:  AuditSeverityInfo,
			message:   "Build completed",
		},
		{
			eventType: messaging.EventTypeBuildExecutionFailed,
			auditType: AuditEventBuildFail,
			resource:  "build",
			action:    "fail",
			severity:  AuditSeverityWarning,
			message:   "Build failed",
		},
		{
			eventType: messaging.EventTypeTenantCreated,
			auditType: AuditEventTenantCreate,
			resource:  "tenant",
			action:    "create",
			severity:  AuditSeverityInfo,
			message:   "Tenant created",
		},
		{
			eventType: messaging.EventTypeTenantActivated,
			auditType: AuditEventTenantActivate,
			resource:  "tenant",
			action:    "activate",
			severity:  AuditSeverityInfo,
			message:   "Tenant activated",
		},
		{
			eventType: messaging.EventTypeInfraProviderCreated,
			auditType: AuditEventConfigChange,
			resource:  "infrastructure_provider",
			action:    "create",
			severity:  AuditSeverityInfo,
			message:   "Infrastructure provider created",
		},
		{
			eventType: messaging.EventTypeInfraProviderUpdated,
			auditType: AuditEventConfigChange,
			resource:  "infrastructure_provider",
			action:    "update",
			severity:  AuditSeverityInfo,
			message:   "Infrastructure provider updated",
		},
		{
			eventType: messaging.EventTypeInfraProviderDeleted,
			auditType: AuditEventConfigChange,
			resource:  "infrastructure_provider",
			action:    "delete",
			severity:  AuditSeverityInfo,
			message:   "Infrastructure provider deleted",
		},
		{
			eventType: messaging.EventTypeProjectCreated,
			auditType: AuditEventProjectCreate,
			resource:  "projects",
			action:    "create",
			severity:  AuditSeverityInfo,
			message:   "Project created",
		},
		{
			eventType: messaging.EventTypeProjectUpdated,
			auditType: AuditEventProjectUpdate,
			resource:  "projects",
			action:    "update",
			severity:  AuditSeverityInfo,
			message:   "Project updated",
		},
		{
			eventType: messaging.EventTypeProjectDeleted,
			auditType: AuditEventProjectDelete,
			resource:  "projects",
			action:    "delete",
			severity:  AuditSeverityInfo,
			message:   "Project deleted",
		},
	}

	unsubscribers := make([]func(), 0, len(mappings))
	for _, mapping := range mappings {
		unsub := bus.Subscribe(mapping.eventType, func(ctx context.Context, event messaging.Event) {
			// Use a fresh background context to avoid request cancellation dropping audit writes.
			auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			tenantID := parseUUID(event.TenantID)
			actorID := parseUUID(event.ActorID)
			eventDetails := withEventMetadata(event)
			if actorID != uuid.Nil {
				_ = service.LogUserAction(auditCtx, tenantID, actorID, mapping.auditType, mapping.resource, mapping.action, mapping.message, eventDetails)
				return
			}
			if tenantID != uuid.Nil {
				_ = service.LogSystemAction(auditCtx, tenantID, mapping.auditType, mapping.resource, mapping.action, mapping.message, eventDetails)
				return
			}
			auditEvent := &AuditEvent{
				EventType: mapping.auditType,
				Severity:  mapping.severity,
				Resource:  mapping.resource,
				Action:    mapping.action,
				Message:   mapping.message,
				Details:   eventDetails,
				Timestamp: time.Now().UTC(),
			}
			if err := service.LogEvent(auditCtx, auditEvent); err != nil && logger != nil {
				logger.Warn("Failed to log audit event from bus", zap.Error(err))
			}
		})
		unsubscribers = append(unsubscribers, unsub)
	}

	return func() {
		for _, unsub := range unsubscribers {
			unsub()
		}
	}
}

func parseUUID(value string) uuid.UUID {
	if value == "" {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func withEventMetadata(event messaging.Event) map[string]interface{} {
	details := map[string]interface{}{}
	for key, value := range event.Payload {
		details[key] = value
	}
	if event.CorrelationID != "" {
		details["correlation_id"] = event.CorrelationID
	}
	if event.RequestID != "" {
		details["request_id"] = event.RequestID
	}
	if event.TraceID != "" {
		details["trace_id"] = event.TraceID
	}
	if event.Source != "" {
		details["source"] = event.Source
	}
	if event.SchemaVersion != "" {
		details["schema_version"] = event.SchemaVersion
	}
	return details
}
