package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// OperationMetrics tracks operation performance
type OperationMetrics struct {
	Count         int64
	TotalDurationMs int64
	MinDurationMs int64
	MaxDurationMs int64
	mu            sync.RWMutex
}

// RecordOperation records an operation execution
func (om *OperationMetrics) RecordOperation(durationMs int64) {
	atomic.AddInt64(&om.Count, 1)
	atomic.AddInt64(&om.TotalDurationMs, durationMs)
	
	om.mu.Lock()
	defer om.mu.Unlock()
	
	if om.MinDurationMs == 0 || durationMs < om.MinDurationMs {
		om.MinDurationMs = durationMs
	}
	if durationMs > om.MaxDurationMs {
		om.MaxDurationMs = durationMs
	}
}

// GetStats returns operation statistics
func (om *OperationMetrics) GetStats() (count int64, totalMs, minMs, maxMs int64, avgMs float64) {
	count = atomic.LoadInt64(&om.Count)
	totalMs = atomic.LoadInt64(&om.TotalDurationMs)
	
	om.mu.RLock()
	minMs = om.MinDurationMs
	maxMs = om.MaxDurationMs
	om.mu.RUnlock()
	
	if count > 0 {
		avgMs = float64(totalMs) / float64(count)
	}
	
	return
}

// Reset clears all metrics
func (om *OperationMetrics) Reset() {
	atomic.StoreInt64(&om.Count, 0)
	atomic.StoreInt64(&om.TotalDurationMs, 0)
	om.mu.Lock()
	om.MinDurationMs = 0
	om.MaxDurationMs = 0
	om.mu.Unlock()
}

// Measurement is a helper for timing operations
type Measurement struct {
	start time.Time
	om    *OperationMetrics
}

// NewMeasurement creates a new measurement
func NewMeasurement(om *OperationMetrics) *Measurement {
	return &Measurement{
		start: time.Now(),
		om:    om,
	}
}

// Record finalizes the measurement
func (m *Measurement) Record() {
	duration := time.Since(m.start)
	m.om.RecordOperation(duration.Milliseconds())
}
