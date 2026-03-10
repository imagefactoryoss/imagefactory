package releasetelemetry

import (
	"strings"
	"sync"

	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
)

// Snapshot is an aggregate view of quarantine release workflow events.
type Snapshot struct {
	Requested int64 `json:"requested"`
	Released  int64 `json:"released"`
	Failed    int64 `json:"failed"`
	Consumed  int64 `json:"consumed"`
	Total     int64 `json:"total"`
}

// Metrics stores in-memory counters for release lifecycle events.
type Metrics struct {
	mu     sync.RWMutex
	counts map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		counts: make(map[string]int64),
	}
}

func (m *Metrics) Record(eventType string) {
	if m == nil {
		return
	}
	trimmed := strings.TrimSpace(eventType)
	if trimmed == "" {
		return
	}
	m.mu.Lock()
	m.counts[trimmed]++
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() Snapshot {
	if m == nil {
		return Snapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	requested := m.counts[messaging.EventTypeQuarantineReleaseRequested]
	released := m.counts[messaging.EventTypeQuarantineReleased]
	failed := m.counts[messaging.EventTypeQuarantineReleaseFailed]
	consumed := m.counts[messaging.EventTypeQuarantineReleaseConsumed]
	return Snapshot{
		Requested: requested,
		Released:  released,
		Failed:    failed,
		Consumed:  consumed,
		Total:     requested + released + failed,
	}
}
