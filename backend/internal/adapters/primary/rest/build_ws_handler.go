package rest

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// WebSocketHub manages WebSocket connections for build log streaming
type WebSocketHub struct {
	clients               map[string]map[*WebSocketClient]bool // buildID -> clients
	tenantClients         map[string]map[*WebSocketClient]bool // tenantID -> clients for general events
	notificationClients   map[string]map[*WebSocketClient]bool // tenantID:userID -> clients for notification events
	broadcast             chan LogMessage
	tenantBroadcast       chan tenantBuildEvent
	notificationBroadcast chan notificationTargetedEvent
	register              chan *WebSocketClient
	unregister            chan *WebSocketClient
	mu                    sync.RWMutex
	logger                *zap.Logger
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	hub        *WebSocketHub
	conn       *websocket.Conn
	streamType string
	buildID    uuid.UUID // uuid.Nil for general events
	userID     uuid.UUID
	tenantID   uuid.UUID
	send       chan interface{} // Can be LogMessage or BuildEventMessage
	ctx        context.Context
	cancel     context.CancelFunc
}

const (
	streamTypeBuildLogs          = "build_logs"
	streamTypeBuildEvents        = "build_events"
	streamTypeNotificationEvents = "notification_events"
)

// LogMessage represents a log entry sent via WebSocket
type LogMessage struct {
	BuildID   string                 `json:"build_id"`
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BuildEventMessage represents a build status event sent via WebSocket
type BuildEventMessage struct {
	Type        string                 `json:"type"`
	BuildID     string                 `json:"build_id"`
	BuildNumber string                 `json:"build_number,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Duration    int                    `json:"duration,omitempty"`
	Timestamp   string                 `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type NotificationEventMessage struct {
	Type           string                 `json:"type"`
	NotificationID string                 `json:"notification_id,omitempty"`
	Timestamp      string                 `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type tenantBuildEvent struct {
	tenantID uuid.UUID
	event    BuildEventMessage
}

type notificationTargetedEvent struct {
	tenantID uuid.UUID
	userID   uuid.UUID
	event    NotificationEventMessage
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *zap.Logger) *WebSocketHub {
	hub := &WebSocketHub{
		clients:               make(map[string]map[*WebSocketClient]bool),
		tenantClients:         make(map[string]map[*WebSocketClient]bool),
		notificationClients:   make(map[string]map[*WebSocketClient]bool),
		broadcast:             make(chan LogMessage, 256),
		tenantBroadcast:       make(chan tenantBuildEvent, 256),
		notificationBroadcast: make(chan notificationTargetedEvent, 256),
		register:              make(chan *WebSocketClient),
		unregister:            make(chan *WebSocketClient),
		logger:                logger,
	}
	go hub.run()
	return hub
}

// run manages hub operations
func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			switch client.streamType {
			case streamTypeBuildLogs:
				// Register for specific build logs
				buildIDStr := client.buildID.String()
				if h.clients[buildIDStr] == nil {
					h.clients[buildIDStr] = make(map[*WebSocketClient]bool)
				}
				h.clients[buildIDStr][client] = true
			case streamTypeNotificationEvents:
				userStreamKey := notificationStreamKey(client.tenantID, client.userID)
				if h.notificationClients[userStreamKey] == nil {
					h.notificationClients[userStreamKey] = make(map[*WebSocketClient]bool)
				}
				h.notificationClients[userStreamKey][client] = true
			default:
				// Register for general build events
				tenantIDStr := client.tenantID.String()
				if h.tenantClients[tenantIDStr] == nil {
					h.tenantClients[tenantIDStr] = make(map[*WebSocketClient]bool)
				}
				h.tenantClients[tenantIDStr][client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			switch client.streamType {
			case streamTypeBuildLogs:
				// Unregister from build logs
				buildIDStr := client.buildID.String()
				if clients, ok := h.clients[buildIDStr]; ok {
					if _, exists := clients[client]; exists {
						delete(clients, client)
						// Intentionally silent for normal connection lifecycle; errors are logged.
						close(client.send)
						if len(clients) == 0 {
							delete(h.clients, buildIDStr)
						}
					}
				}
			case streamTypeNotificationEvents:
				userStreamKey := notificationStreamKey(client.tenantID, client.userID)
				if clients, ok := h.notificationClients[userStreamKey]; ok {
					if _, exists := clients[client]; exists {
						delete(clients, client)
						close(client.send)
						if len(clients) == 0 {
							delete(h.notificationClients, userStreamKey)
						}
					}
				}
			default:
				// Unregister from build events
				tenantIDStr := client.tenantID.String()
				if clients, ok := h.tenantClients[tenantIDStr]; ok {
					if _, exists := clients[client]; exists {
						delete(clients, client)
						// Intentionally silent for normal connection lifecycle; errors are logged.
						close(client.send)
						if len(clients) == 0 {
							delete(h.tenantClients, tenantIDStr)
						}
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.clients[msg.BuildID]; ok {
				for client := range clients {
					select {
					case client.send <- msg:
					default:
						// Client's send channel is full, close it
						go func(c *WebSocketClient) {
							h.unregister <- c
						}(client)
					}
				}
			}
			h.mu.RUnlock()

		case tenantEvent := <-h.tenantBroadcast:
			h.mu.RLock()
			if tenantEvent.tenantID == uuid.Nil {
				for _, clients := range h.tenantClients {
					for client := range clients {
						select {
						case client.send <- tenantEvent.event:
						default:
							// Client's send channel is full, close it
							go func(c *WebSocketClient) {
								h.unregister <- c
							}(client)
						}
					}
				}
			} else if clients, ok := h.tenantClients[tenantEvent.tenantID.String()]; ok {
				for client := range clients {
					select {
					case client.send <- tenantEvent.event:
					default:
						// Client's send channel is full, close it
						go func(c *WebSocketClient) {
							h.unregister <- c
						}(client)
					}
				}
			}
			h.mu.RUnlock()

		case notificationEvent := <-h.notificationBroadcast:
			h.mu.RLock()
			streamKey := notificationStreamKey(notificationEvent.tenantID, notificationEvent.userID)
			if clients, ok := h.notificationClients[streamKey]; ok {
				for client := range clients {
					select {
					case client.send <- notificationEvent.event:
					default:
						go func(c *WebSocketClient) {
							h.unregister <- c
						}(client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func notificationStreamKey(tenantID, userID uuid.UUID) string {
	return tenantID.String() + ":" + userID.String()
}

// BuildWSHandler handles build WebSocket connections
type BuildWSHandler struct {
	hub          *WebSocketHub
	buildService *build.Service
	logger       *zap.Logger
}

// NewBuildWSHandler creates a new WebSocket handler
func NewBuildWSHandler(hub *WebSocketHub, buildService *build.Service, logger *zap.Logger) *BuildWSHandler {
	return &BuildWSHandler{
		hub:          hub,
		buildService: buildService,
		logger:       logger,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// HandleBuildLogs handles WebSocket connections for build logs
func (h *BuildWSHandler) HandleBuildLogs(w http.ResponseWriter, r *http.Request) {
	// Extract buildID from URL
	buildIDStr := chi.URLParam(r, "id")
	if buildIDStr == "" {
		http.Error(w, "Build ID is required", http.StatusBadRequest)
		return
	}

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.logger.Warn("Invalid build ID format", zap.String("build_id", buildIDStr), zap.Error(err))
		http.Error(w, "Invalid build ID format", http.StatusBadRequest)
		return
	}

	// Extract auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("Missing auth context for WebSocket connection")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify build exists and user has access
	build, err := h.buildService.GetBuild(r.Context(), buildID)
	if err != nil {
		h.logger.Error("Failed to verify build access",
			zap.String("build_id", buildIDStr),
			zap.Error(err))
		http.Error(w, "Failed to verify build", http.StatusInternalServerError)
		return
	}

	if build == nil {
		h.logger.Warn("Build not found",
			zap.String("build_id", buildIDStr))
		http.Error(w, "Build not found", http.StatusNotFound)
		return
	}

	// Verify tenant access
	if build.TenantID() != authCtx.TenantID {
		h.logger.Warn("Unauthorized tenant access",
			zap.String("build_id", buildIDStr),
			zap.String("tenant_id", authCtx.TenantID.String()))
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection",
			zap.Error(err),
			zap.String("build_id", buildIDStr))
		return
	}

	// Use a detached context for websocket lifecycle.
	// Request context is canceled once the HTTP handler returns, which would
	// otherwise close the websocket immediately after upgrade.
	ctx, cancel := context.WithCancel(context.Background())

	// Create and register client
	client := &WebSocketClient{
		hub:        h.hub,
		conn:       conn,
		streamType: streamTypeBuildLogs,
		buildID:    buildID,
		userID:     authCtx.UserID,
		tenantID:   authCtx.TenantID,
		send:       make(chan interface{}, 256),
		ctx:        ctx,
		cancel:     cancel,
	}

	h.hub.register <- client

	// Start goroutines to handle client communication
	go client.readPump(h.logger)
	go client.writePump(h.logger)

	// Intentionally silent for normal connection lifecycle; errors are logged.
}

// HandleBuildEvents handles WebSocket connections for general build events
func (h *BuildWSHandler) HandleBuildEvents(w http.ResponseWriter, r *http.Request) {
	// Extract auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("Missing auth context for WebSocket connection")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection for build events",
			zap.Error(err))
		return
	}

	// Use a detached context for websocket lifecycle.
	// Request context is canceled once the HTTP handler returns, which would
	// otherwise close the websocket immediately after upgrade.
	ctx, cancel := context.WithCancel(context.Background())

	// Create and register client for general events (buildID = uuid.Nil)
	client := &WebSocketClient{
		hub:        h.hub,
		conn:       conn,
		streamType: streamTypeBuildEvents,
		buildID:    uuid.Nil, // Indicates general events client
		userID:     authCtx.UserID,
		tenantID:   authCtx.TenantID,
		send:       make(chan interface{}, 256),
		ctx:        ctx,
		cancel:     cancel,
	}

	h.hub.register <- client

	// Start goroutines to handle client communication
	go client.readPump(h.logger)
	go client.writePump(h.logger)

	// Intentionally silent for normal connection lifecycle; errors are logged.
}

// HandleNotificationEvents handles websocket connections for user notification events.
func (h *BuildWSHandler) HandleNotificationEvents(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Warn("Missing auth context for notification websocket connection")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade notification websocket connection", zap.Error(err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &WebSocketClient{
		hub:        h.hub,
		conn:       conn,
		streamType: streamTypeNotificationEvents,
		buildID:    uuid.Nil,
		userID:     authCtx.UserID,
		tenantID:   authCtx.TenantID,
		send:       make(chan interface{}, 256),
		ctx:        ctx,
		cancel:     cancel,
	}

	h.hub.register <- client
	go client.readPump(h.logger)
	go client.writePump(h.logger)
}

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump(logger *zap.Logger) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		c.cancel()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if c.streamType != streamTypeBuildLogs {
			_, _, err := c.conn.ReadMessage()
			if err != nil {
				if !isBenignWebSocketDisconnect(err) {
					logger.Error("WebSocket error",
						zap.String("stream_type", c.streamType),
						zap.Error(err))
				}
				return
			}
			continue
		}

		var msg LogMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if !isBenignWebSocketDisconnect(err) {
				logger.Error("WebSocket error",
					zap.String("build_id", c.buildID.String()),
					zap.Error(err))
			}
			return
		}

		// Validate message origin
		if msg.BuildID != c.buildID.String() {
			logger.Warn("Client attempted to send logs for different build",
				zap.String("client_build_id", c.buildID.String()),
				zap.String("msg_build_id", msg.BuildID))
			continue
		}

		// Broadcast message to other clients
		c.hub.broadcast <- msg
	}
}

// writePump writes messages to the WebSocket connection
func (c *WebSocketClient) writePump(logger *zap.Logger) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(msg)
			if err != nil {
				if !isBenignWebSocketDisconnect(err) {
					logger.Error("Failed to write WebSocket message",
						zap.String("build_id", c.buildID.String()),
						zap.Error(err))
				}
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if !isBenignWebSocketDisconnect(err) {
					logger.Error("Failed to send WebSocket ping",
						zap.String("build_id", c.buildID.String()),
						zap.Error(err))
				}
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func isBenignWebSocketDisconnect(err error) bool {
	if err == nil {
		return true
	}
	if websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "broken pipe") ||
		strings.Contains(text, "connection reset by peer") ||
		strings.Contains(text, "use of closed network connection")
}

// BroadcastBuildEvent sends a build event to all connected clients for a tenant
func (h *WebSocketHub) BroadcastBuildEvent(tenantID uuid.UUID, eventType, buildID, buildNumber, projectID, status, message string, duration int, metadata map[string]interface{}) {
	event := BuildEventMessage{
		Type:        eventType,
		BuildID:     buildID,
		BuildNumber: buildNumber,
		ProjectID:   projectID,
		Status:      status,
		Message:     message,
		Duration:    duration,
		Timestamp:   time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Metadata:    metadata,
	}

	select {
	case h.tenantBroadcast <- tenantBuildEvent{
		tenantID: tenantID,
		event:    event,
	}:
	default:
		h.logger.Warn("WebSocket tenant broadcast channel full",
			zap.String("tenant_id", tenantID.String()),
			zap.String("event_type", eventType))
	}
}

// BroadcastNotificationEvent sends a user-scoped notification event to notification stream clients.
func (h *WebSocketHub) BroadcastNotificationEvent(tenantID, userID uuid.UUID, eventType string, notificationID *uuid.UUID, metadata map[string]interface{}) {
	if h == nil || tenantID == uuid.Nil || userID == uuid.Nil {
		return
	}

	event := NotificationEventMessage{
		Type:      strings.TrimSpace(eventType),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metadata:  metadata,
	}
	if notificationID != nil && *notificationID != uuid.Nil {
		event.NotificationID = notificationID.String()
	}

	select {
	case h.notificationBroadcast <- notificationTargetedEvent{
		tenantID: tenantID,
		userID:   userID,
		event:    event,
	}:
	default:
		h.logger.Warn("WebSocket notification broadcast channel full",
			zap.String("tenant_id", tenantID.String()),
			zap.String("user_id", userID.String()),
			zap.String("event_type", eventType))
	}
}

// BroadcastBuildLog sends a build log entry to clients connected to a specific build stream.
func (h *WebSocketHub) BroadcastBuildLog(buildID uuid.UUID, timestamp time.Time, level, message string, metadata map[string]interface{}) {
	if h == nil || buildID == uuid.Nil {
		return
	}
	msg := LogMessage{
		BuildID:   buildID.String(),
		Timestamp: timestamp.UTC().Format(time.RFC3339),
		Level:     strings.ToUpper(strings.TrimSpace(level)),
		Message:   message,
		Metadata:  metadata,
	}
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("WebSocket build log broadcast channel full",
			zap.String("build_id", buildID.String()))
	}
}

// BroadcastSystemEvent sends a non-build system event to connected event clients.
func (h *WebSocketHub) BroadcastSystemEvent(eventType, status, message string, metadata map[string]interface{}) {
	if h == nil {
		return
	}
	h.BroadcastBuildEvent(uuid.Nil, eventType, "", "", "", status, message, 0, metadata)
}

// GetConnectedClients returns the number of connected clients for a build
func (h *WebSocketHub) GetConnectedClients(buildID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[buildID.String()]; ok {
		return len(clients)
	}
	return 0
}

// CloseClientConnections closes all connections for a build
func (h *WebSocketHub) CloseClientConnections(buildID uuid.UUID) {
	h.mu.Lock()
	clients := h.clients[buildID.String()]
	delete(h.clients, buildID.String())
	h.mu.Unlock()

	for client := range clients {
		client.cancel()
		close(client.send)
	}
}
