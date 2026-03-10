package releasecompliance

import (
	"sync"
	"time"
)

// Snapshot captures in-memory runtime drift compliance counters.
type Snapshot struct {
	WatchTicksTotal     int64 `json:"watch_ticks_total"`
	WatchFailuresTotal  int64 `json:"watch_failures_total"`
	DriftDetectedTotal  int64 `json:"drift_detected_total"`
	DriftRecoveredTotal int64 `json:"drift_recovered_total"`
	ActiveDriftCount    int64 `json:"active_drift_count"`
	ReleasedCount       int64 `json:"released_count"`
	LastTickUnix        int64 `json:"last_tick_unix"`
}

// Metrics stores watcher counters in-memory for admin stats/reporting.
type Metrics struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RecordTick(activeDrift, released int64) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.snapshot.WatchTicksTotal++
	m.snapshot.ActiveDriftCount = activeDrift
	m.snapshot.ReleasedCount = released
	m.snapshot.LastTickUnix = time.Now().UTC().Unix()
	m.mu.Unlock()
}

func (m *Metrics) RecordFailure() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.snapshot.WatchFailuresTotal++
	m.mu.Unlock()
}

func (m *Metrics) AddDetected(count int64) {
	if m == nil || count <= 0 {
		return
	}
	m.mu.Lock()
	m.snapshot.DriftDetectedTotal += count
	m.mu.Unlock()
}

func (m *Metrics) AddRecovered(count int64) {
	if m == nil || count <= 0 {
		return
	}
	m.mu.Lock()
	m.snapshot.DriftRecoveredTotal += count
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() Snapshot {
	if m == nil {
		return Snapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshot
}
