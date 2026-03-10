package worker

import (
	"time"

	"github.com/google/uuid"
)

// Domain Events

// WorkerRegistered is raised when a new worker is registered
type WorkerRegistered struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	Name       string
	WorkerType string
	Capacity   int
	Timestamp  time.Time
}

// LoadUpdated is raised when a worker's load changes
type LoadUpdated struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	CurrentLoad int
	Capacity    int
	Status      string
	Timestamp   time.Time
}

// HeartbeatReceived is raised when a worker sends a heartbeat
type HeartbeatReceived struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Timestamp time.Time
}

// WorkerFailed is raised when a worker fails
type WorkerFailed struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	ConsecutiveFailures int
	Reason              string
	Timestamp           time.Time
}

// WorkerOnline is raised when a worker comes online
type WorkerOnline struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Timestamp time.Time
}

// WorkerOffline is raised when a worker goes offline
type WorkerOffline struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Reason   string
	Timestamp time.Time
}
