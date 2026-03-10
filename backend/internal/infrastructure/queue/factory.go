package queue

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// QueueBackendType represents the type of queue backend
type QueueBackendType string

const (
	QueueBackendPostgres QueueBackendType = "postgres"
	QueueBackendRedis    QueueBackendType = "redis"
	QueueBackendNATS     QueueBackendType = "nats"
)

// QueueConfig holds configuration for queue backend creation
type QueueConfig struct {
	Backend QueueBackendType
	DB      *sqlx.DB // For PostgreSQL backend
	// RedisClient redis.Client // For Redis backend (future)
	// NATSConn *nats.Conn       // For NATS backend (future)
}

// NewEmailQueueBackend creates a new email queue backend based on configuration
// Follows Dependency Inversion Principle - returns interface, not concrete type
func NewEmailQueueBackend(config QueueConfig) (notification.EmailQueueBackend, error) {
	switch config.Backend {
	case QueueBackendPostgres:
		if config.DB == nil {
			return nil, fmt.Errorf("postgres backend requires DB connection")
		}
		return NewPostgresEmailQueue(config.DB), nil

	case QueueBackendRedis:
		// Future implementation
		return nil, fmt.Errorf("redis backend not yet implemented")

	case QueueBackendNATS:
		// Future implementation
		return nil, fmt.Errorf("nats backend not yet implemented")

	default:
		return nil, fmt.Errorf("unknown queue backend type: %s", config.Backend)
	}
}

// MustNewEmailQueueBackend creates a queue backend or panics
// Use only in initialization code where failure should be fatal
func MustNewEmailQueueBackend(config QueueConfig) notification.EmailQueueBackend {
	backend, err := NewEmailQueueBackend(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create email queue backend: %v", err))
	}
	return backend
}
