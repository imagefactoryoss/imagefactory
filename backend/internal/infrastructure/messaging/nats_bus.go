package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"go.uber.org/zap"
)

// NATSBus publishes and subscribes to events using NATS.
type NATSBus struct {
	conn     *nats.Conn
	subject  string
	logger   *zap.Logger
	statusMu sync.RWMutex
	status   NATSTransportStatus
}

type NATSTransportStatus struct {
	ConnectedAt      time.Time
	LastDisconnectAt time.Time
	LastReconnectAt  time.Time
	LastError        string
	Reconnects       int64
	Disconnects      int64
	Status           string
	ConnectedURL     string
}

func NewNATSBus(config config.NATSConfig, logger *zap.Logger) (*NATSBus, error) {
	bus := &NATSBus{
		subject: config.Subject,
		logger:  logger,
		status: NATSTransportStatus{
			Status: "connecting",
		},
	}
	opts := []nats.Option{
		nats.Timeout(config.Timeout),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			bus.updateStatus(func(status *NATSTransportStatus) {
				status.Status = "disconnected"
				status.LastDisconnectAt = time.Now().UTC()
				status.Disconnects++
				status.ConnectedURL = conn.ConnectedUrl()
				if err != nil {
					status.LastError = err.Error()
				}
			})
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			bus.updateStatus(func(status *NATSTransportStatus) {
				status.Status = "connected"
				status.LastReconnectAt = time.Now().UTC()
				status.Reconnects++
				status.ConnectedURL = conn.ConnectedUrl()
				status.LastError = ""
			})
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			bus.updateStatus(func(status *NATSTransportStatus) {
				status.Status = "closed"
				status.ConnectedURL = conn.ConnectedUrl()
				if err := conn.LastError(); err != nil {
					status.LastError = err.Error()
				}
			})
		}),
	}

	urls := strings.Join(config.URLs, ",")
	conn, err := nats.Connect(urls, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	bus.conn = conn
	bus.updateStatus(func(status *NATSTransportStatus) {
		now := time.Now().UTC()
		status.Status = "connected"
		status.ConnectedAt = now
		status.LastReconnectAt = now
		status.ConnectedURL = conn.ConnectedUrl()
		status.LastError = ""
	})
	return bus, nil
}

func (b *NATSBus) Publish(ctx context.Context, event Event) error {
	subject := b.subjectFor(event.Type)
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return b.conn.Publish(subject, payload)
}

func (b *NATSBus) Subscribe(eventType string, handler Handler) (unsubscribe func()) {
	subject := b.subjectFor(eventType)
	if eventType == "*" {
		subject = fmt.Sprintf("%s.>", b.subject)
	}
	sub, err := b.conn.Subscribe(subject, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			if b.logger != nil {
				b.logger.Warn("Failed to unmarshal NATS event", zap.Error(err))
			}
			return
		}
		handler(context.Background(), event)
	})
	if err != nil {
		if b.logger != nil {
			b.logger.Warn("Failed to subscribe to NATS subject", zap.String("subject", subject), zap.Error(err))
		}
		return func() {}
	}
	return func() {
		_ = sub.Unsubscribe()
	}
}

func (b *NATSBus) Close() {
	if b.conn == nil {
		return
	}
	b.conn.Drain()
	_ = b.conn.FlushTimeout(2 * time.Second)
	b.conn.Close()
}

func (b *NATSBus) TransportStatus() NATSTransportStatus {
	if b == nil {
		return NATSTransportStatus{Status: "unavailable"}
	}
	b.statusMu.RLock()
	defer b.statusMu.RUnlock()
	return b.status
}

func (b *NATSBus) updateStatus(update func(status *NATSTransportStatus)) {
	if b == nil || update == nil {
		return
	}
	b.statusMu.Lock()
	defer b.statusMu.Unlock()
	update(&b.status)
}

func (b *NATSBus) subjectFor(eventType string) string {
	if b.subject == "" {
		return eventType
	}
	return fmt.Sprintf("%s.%s", b.subject, eventType)
}
