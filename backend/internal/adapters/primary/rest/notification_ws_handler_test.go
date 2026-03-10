package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestHandleNotificationEvents_RequiresAuthContext(t *testing.T) {
	hub := NewWebSocketHub(zap.NewNop())
	handler := NewBuildWSHandler(hub, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/events", nil)
	rr := httptest.NewRecorder()

	handler.HandleNotificationEvents(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestWebSocketHub_BroadcastNotificationEvent_TargetedByTenantAndUser(t *testing.T) {
	hub := NewWebSocketHub(zap.NewNop())

	targetTenant := uuid.New()
	targetUser := uuid.New()
	otherTenant := uuid.New()
	otherUser := uuid.New()

	newClient := func(tenantID, userID uuid.UUID) *WebSocketClient {
		ctx, cancel := context.WithCancel(context.Background())
		return &WebSocketClient{
			hub:        hub,
			streamType: streamTypeNotificationEvents,
			buildID:    uuid.Nil,
			tenantID:   tenantID,
			userID:     userID,
			send:       make(chan interface{}, 2),
			ctx:        ctx,
			cancel:     cancel,
		}
	}

	targetClient := newClient(targetTenant, targetUser)
	nonTargetSameTenant := newClient(targetTenant, otherUser)
	nonTargetSameUser := newClient(otherTenant, targetUser)

	hub.register <- targetClient
	hub.register <- nonTargetSameTenant
	hub.register <- nonTargetSameUser
	defer func() {
		hub.unregister <- targetClient
		hub.unregister <- nonTargetSameTenant
		hub.unregister <- nonTargetSameUser
	}()

	time.Sleep(25 * time.Millisecond)

	notificationID := uuid.New()
	hub.BroadcastNotificationEvent(
		targetTenant,
		targetUser,
		"notification.created",
		&notificationID,
		map[string]interface{}{"notification_type": "build_started"},
	)

	select {
	case raw := <-targetClient.send:
		event, ok := raw.(NotificationEventMessage)
		if !ok {
			t.Fatalf("expected NotificationEventMessage, got %T", raw)
		}
		if event.Type != "notification.created" {
			t.Fatalf("expected event type notification.created, got %q", event.Type)
		}
		if event.NotificationID != notificationID.String() {
			t.Fatalf("expected notification_id %q, got %q", notificationID.String(), event.NotificationID)
		}
		if event.Timestamp == "" {
			t.Fatalf("expected timestamp to be set")
		}
		if event.Metadata == nil || event.Metadata["notification_type"] != "build_started" {
			t.Fatalf("expected metadata.notification_type=build_started, got %#v", event.Metadata)
		}
	case <-time.After(600 * time.Millisecond):
		t.Fatalf("expected target client to receive notification event")
	}

	select {
	case raw := <-nonTargetSameTenant.send:
		t.Fatalf("non-target same-tenant client should not receive event, got %T", raw)
	case <-time.After(120 * time.Millisecond):
	}

	select {
	case raw := <-nonTargetSameUser.send:
		t.Fatalf("non-target same-user client should not receive event, got %T", raw)
	case <-time.After(120 * time.Millisecond):
	}
}
