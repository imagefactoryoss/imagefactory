package worker

import (
	"sync"
	"sync/atomic"
	"time"
)

// WorkerStatus represents the operational status of a worker
type WorkerStatus string

const (
	WorkerStatusIdle       WorkerStatus = "idle"
	WorkerStatusProcessing WorkerStatus = "processing"
	WorkerStatusStopped    WorkerStatus = "stopped"
	WorkerStatusError      WorkerStatus = "error"
)

// WorkerMetrics holds metrics for email worker pool
// Uses atomic operations for thread-safe updates
type WorkerMetrics struct {
	// EmailsSent is the total number of emails successfully sent
	emailsSent atomic.Int64

	// EmailsFailed is the total number of emails that failed permanently
	emailsFailed atomic.Int64

	// EmailsRetried is the total number of emails retried
	emailsRetried atomic.Int64

	// QueueDepth is the current number of pending emails in the queue
	queueDepth atomic.Int64

	// WorkerStatus tracks the status of each worker (protected by mutex)
	workerStatuses   map[int]WorkerStatus
	workerStatusesMu sync.RWMutex

	// LastHealthCheck is the timestamp of the last health check
	lastHealthCheck atomic.Int64 // Unix timestamp
}

// NewWorkerMetrics creates a new WorkerMetrics instance
func NewWorkerMetrics(workerCount int) *WorkerMetrics {
	m := &WorkerMetrics{
		workerStatuses: make(map[int]WorkerStatus, workerCount),
	}
	// Initialize all workers as idle
	for i := 0; i < workerCount; i++ {
		m.workerStatuses[i] = WorkerStatusIdle
	}
	return m
}

// IncrementEmailsSent increments the emails sent counter
func (m *WorkerMetrics) IncrementEmailsSent() {
	m.emailsSent.Add(1)
}

// IncrementEmailsFailed increments the emails failed counter
func (m *WorkerMetrics) IncrementEmailsFailed() {
	m.emailsFailed.Add(1)
}

// IncrementEmailsRetried increments the emails retried counter
func (m *WorkerMetrics) IncrementEmailsRetried() {
	m.emailsRetried.Add(1)
}

// UpdateQueueDepth updates the current queue depth
func (m *WorkerMetrics) UpdateQueueDepth(depth int64) {
	m.queueDepth.Store(depth)
}

// UpdateLastHealthCheck updates the last health check timestamp
func (m *WorkerMetrics) UpdateLastHealthCheck() {
	m.lastHealthCheck.Store(time.Now().Unix())
}

// GetEmailsSent returns the total emails sent
func (m *WorkerMetrics) GetEmailsSent() int64 {
	return m.emailsSent.Load()
}

// GetEmailsFailed returns the total emails failed
func (m *WorkerMetrics) GetEmailsFailed() int64 {
	return m.emailsFailed.Load()
}

// GetEmailsRetried returns the total emails retried
func (m *WorkerMetrics) GetEmailsRetried() int64 {
	return m.emailsRetried.Load()
}

// GetQueueDepth returns the current queue depth
func (m *WorkerMetrics) GetQueueDepth() int64 {
	return m.queueDepth.Load()
}

// GetLastHealthCheck returns the last health check timestamp
func (m *WorkerMetrics) GetLastHealthCheck() time.Time {
	ts := m.lastHealthCheck.Load()
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// SetWorkerStatus sets the status for a specific worker
// Thread-safe with mutex protection
func (m *WorkerMetrics) SetWorkerStatus(workerID int, status WorkerStatus) {
	m.workerStatusesMu.Lock()
	defer m.workerStatusesMu.Unlock()
	m.workerStatuses[workerID] = status
}

// GetWorkerStatus gets the status for a specific worker
// Thread-safe with mutex protection
func (m *WorkerMetrics) GetWorkerStatus(workerID int) WorkerStatus {
	m.workerStatusesMu.RLock()
	defer m.workerStatusesMu.RUnlock()
	return m.workerStatuses[workerID]
}

// GetAllWorkerStatuses returns a copy of all worker statuses
// Thread-safe with mutex protection
func (m *WorkerMetrics) GetAllWorkerStatuses() map[int]WorkerStatus {
	m.workerStatusesMu.RLock()
	defer m.workerStatusesMu.RUnlock()

	statuses := make(map[int]WorkerStatus, len(m.workerStatuses))
	for id, status := range m.workerStatuses {
		statuses[id] = status
	}
	return statuses
}

// Snapshot returns a snapshot of all metrics
type MetricsSnapshot struct {
	EmailsSent      int64                `json:"emails_sent"`
	EmailsFailed    int64                `json:"emails_failed"`
	EmailsRetried   int64                `json:"emails_retried"`
	QueueDepth      int64                `json:"queue_depth"`
	WorkerStatuses  map[int]WorkerStatus `json:"worker_statuses"`
	LastHealthCheck time.Time            `json:"last_health_check"`
}

// Snapshot returns a point-in-time snapshot of all metrics
func (m *WorkerMetrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		EmailsSent:      m.GetEmailsSent(),
		EmailsFailed:    m.GetEmailsFailed(),
		EmailsRetried:   m.GetEmailsRetried(),
		QueueDepth:      m.GetQueueDepth(),
		WorkerStatuses:  m.GetAllWorkerStatuses(),
		LastHealthCheck: m.GetLastHealthCheck(),
	}
}
