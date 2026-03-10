package worker

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Worker is an aggregate root representing a build execution host
type Worker struct {
	id                    uuid.UUID
	tenantID              uuid.UUID
	name                  string
	workerType            WorkerType
	capacity              Capacity
	currentLoad           int
	status                Status
	lastHeartbeat         *time.Time
	consecutiveFailures   int
	createdAt             time.Time
	updatedAt             time.Time
	uncommittedEvents     []interface{}
}

// New creates a new Worker aggregate
func New(
	id uuid.UUID,
	tenantID uuid.UUID,
	name string,
	workerType WorkerType,
	capacity Capacity,
) (*Worker, error) {
	if id == uuid.Nil {
		return nil, errors.New("worker id cannot be nil")
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant id cannot be nil")
	}
	if name == "" {
		return nil, errors.New("worker name cannot be empty")
	}
	if !capacity.IsValid() {
		return nil, errors.New("capacity must be greater than 0")
	}

	w := &Worker{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		workerType:  workerType,
		capacity:    capacity,
		currentLoad: 0,
		status:      StatusAvailable,
		createdAt:   time.Now().UTC(),
		updatedAt:   time.Now().UTC(),
	}

	w.addEvent(&WorkerRegistered{
		ID:         w.id,
		TenantID:   w.tenantID,
		Name:       w.name,
		WorkerType: string(w.workerType),
		Capacity:   int(w.capacity),
		Timestamp:  w.createdAt,
	})

	return w, nil
}

// === Accessors ===

func (w *Worker) ID() uuid.UUID {
	return w.id
}

func (w *Worker) TenantID() uuid.UUID {
	return w.tenantID
}

func (w *Worker) Name() string {
	return w.name
}

func (w *Worker) WorkerType() WorkerType {
	return w.workerType
}

func (w *Worker) Capacity() Capacity {
	return w.capacity
}

func (w *Worker) CurrentLoad() int {
	return w.currentLoad
}

func (w *Worker) Status() Status {
	return w.status
}

func (w *Worker) LastHeartbeat() *time.Time {
	return w.lastHeartbeat
}

func (w *Worker) ConsecutiveFailures() int {
	return w.consecutiveFailures
}

func (w *Worker) AvailableCapacity() int {
	return int(w.capacity) - w.currentLoad
}

func (w *Worker) IsAvailable() bool {
	return w.status == StatusAvailable && w.currentLoad < int(w.capacity)
}

func (w *Worker) CreatedAt() time.Time {
	return w.createdAt
}

func (w *Worker) UpdatedAt() time.Time {
	return w.updatedAt
}

// === Commands ===

// IncrementLoad increases the worker's current load
func (w *Worker) IncrementLoad(delta int) error {
	if delta <= 0 {
		return errors.New("load delta must be positive")
	}

	newLoad := w.currentLoad + delta
	if newLoad > int(w.capacity) {
		return errors.New("load would exceed capacity")
	}

	w.currentLoad = newLoad

	// Update status based on load
	if w.currentLoad >= int(w.capacity) {
		w.status = StatusBusy
	} else if w.status == StatusBusy && w.currentLoad < int(w.capacity) {
		w.status = StatusAvailable
	}

	w.updatedAt = time.Now().UTC()

	w.addEvent(&LoadUpdated{
		ID:           w.id,
		TenantID:     w.tenantID,
		CurrentLoad:  w.currentLoad,
		Capacity:     int(w.capacity),
		Status:       string(w.status),
		Timestamp:    w.updatedAt,
	})

	return nil
}

// DecrementLoad decreases the worker's current load
func (w *Worker) DecrementLoad(delta int) error {
	if delta <= 0 {
		return errors.New("load delta must be positive")
	}

	newLoad := w.currentLoad - delta
	if newLoad < 0 {
		newLoad = 0
	}

	w.currentLoad = newLoad

	// Update status based on load
	if w.currentLoad < int(w.capacity) && w.status == StatusBusy {
		w.status = StatusAvailable
	}

	w.updatedAt = time.Now().UTC()

	w.addEvent(&LoadUpdated{
		ID:           w.id,
		TenantID:     w.tenantID,
		CurrentLoad:  w.currentLoad,
		Capacity:     int(w.capacity),
		Status:       string(w.status),
		Timestamp:    w.updatedAt,
	})

	return nil
}

// RecordHeartbeat updates the last heartbeat time and resets failure count
func (w *Worker) RecordHeartbeat() error {
	if w.status == StatusOffline {
		return errors.New("cannot record heartbeat for offline worker")
	}

	now := time.Now().UTC()
	w.lastHeartbeat = &now
	w.consecutiveFailures = 0
	w.updatedAt = now

	w.addEvent(&HeartbeatReceived{
		ID:        w.id,
		TenantID:  w.tenantID,
		Timestamp: now,
	})

	return nil
}

// RecordFailure increments the failure counter
func (w *Worker) RecordFailure() error {
	w.consecutiveFailures++
	w.updatedAt = time.Now().UTC()

	// Mark offline if too many consecutive failures
	if w.consecutiveFailures >= 5 {
		w.status = StatusOffline
		w.addEvent(&WorkerFailed{
			ID:                  w.id,
			TenantID:            w.tenantID,
			ConsecutiveFailures: w.consecutiveFailures,
			Reason:              "Too many consecutive failures",
			Timestamp:           w.updatedAt,
		})
	}

	return nil
}

// GoOnline marks the worker as available
func (w *Worker) GoOnline() error {
	if w.status == StatusAvailable {
		return nil // Already online
	}

	w.status = StatusAvailable
	w.consecutiveFailures = 0
	w.updatedAt = time.Now().UTC()

	w.addEvent(&WorkerOnline{
		ID:        w.id,
		TenantID:  w.tenantID,
		Timestamp: w.updatedAt,
	})

	return nil
}

// GoOffline marks the worker as offline
func (w *Worker) GoOffline(reason string) error {
	if w.status == StatusOffline {
		return nil // Already offline
	}

	w.status = StatusOffline
	w.updatedAt = time.Now().UTC()

	w.addEvent(&WorkerOffline{
		ID:        w.id,
		TenantID:  w.tenantID,
		Reason:    reason,
		Timestamp: w.updatedAt,
	})

	return nil
}

// === Event Management ===

func (w *Worker) addEvent(event interface{}) {
	w.uncommittedEvents = append(w.uncommittedEvents, event)
}

// UncommittedEvents returns all events raised since the aggregate was loaded
func (w *Worker) UncommittedEvents() []interface{} {
	return w.uncommittedEvents
}

// MarkEventsAsCommitted clears the uncommitted events
func (w *Worker) MarkEventsAsCommitted() {
	w.uncommittedEvents = nil
}
