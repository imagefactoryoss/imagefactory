package rest

import (
	"context"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

// RegisterBuildEventBusSubscriber wires build event bus messages to WebSocket broadcasts.
func RegisterBuildEventBusSubscriber(bus messaging.EventBus, hub *WebSocketHub, logger *zap.Logger) (unsubscribe func()) {
	if bus == nil || hub == nil {
		return func() {}
	}

	handler := func(ctx context.Context, event messaging.Event) {
		buildID := stringValue(event.Payload, "build_id")
		buildNumber := stringValue(event.Payload, "build_number")
		projectID := stringValue(event.Payload, "project_id")
		status := stringValue(event.Payload, "status")
		message := stringValue(event.Payload, "message")
		duration := intValue(event.Payload, "duration")
		metadata := mapValue(event.Payload, "metadata")
		tenantID := parseUUID(event.TenantID)

		if buildID == "" {
			if logger != nil {
				logger.Debug("Skipping build event without build_id", zap.String("event_type", event.Type))
			}
			return
		}

		hub.BroadcastBuildEvent(tenantID, wsEventType(event.Type), buildID, buildNumber, projectID, status, message, duration, metadata)
	}

	unsubscribers := []func(){
		bus.Subscribe(messaging.EventTypeBuildExecutionStatusUpdate, handler),
		bus.Subscribe(messaging.EventTypeBuildExecutionCompleted, handler),
		bus.Subscribe(messaging.EventTypeBuildExecutionFailed, handler),
		bus.Subscribe(messaging.EventTypeBuildStarted, handler),
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

func stringValue(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func intValue(payload map[string]interface{}, key string) int {
	if payload == nil {
		return 0
	}
	if value, ok := payload[key]; ok {
		switch cast := value.(type) {
		case int:
			return cast
		case int64:
			return int(cast)
		case float64:
			return int(cast)
		}
	}
	return 0
}

func mapValue(payload map[string]interface{}, key string) map[string]interface{} {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		if result, ok := value.(map[string]interface{}); ok {
			return result
		}
	}
	return nil
}

func wsEventType(eventType string) string {
	switch eventType {
	case messaging.EventTypeBuildExecutionCompleted:
		return messaging.EventTypeBuildCompleted
	case messaging.EventTypeBuildExecutionFailed:
		return messaging.EventTypeBuildFailed
	case messaging.EventTypeBuildExecutionStatusUpdate:
		return messaging.EventTypeBuildStatusUpdate
	default:
		return eventType
	}
}
