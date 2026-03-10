package releasecompliance

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DriftRecord identifies one released artifact currently in drift state.
type DriftRecord struct {
	ExternalImageImportID uuid.UUID
	TenantID              uuid.UUID
	ReleaseState          string
	InternalImageRef      string
	SourceImageDigest     string
	ReleasedAt            time.Time
}

// Manager deduplicates drift/recovery transitions between watcher ticks.
type Manager struct {
	mu     sync.Mutex
	active map[uuid.UUID]DriftRecord
}

func NewManager() *Manager {
	return &Manager{
		active: make(map[uuid.UUID]DriftRecord),
	}
}

// Evaluate compares the current drift set to previous state and returns
// transition slices for newly detected and recovered drift records.
func (m *Manager) Evaluate(current []DriftRecord) (detected []DriftRecord, recovered []DriftRecord) {
	if m == nil {
		return nil, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	currentMap := make(map[uuid.UUID]DriftRecord, len(current))
	for _, record := range current {
		if record.ExternalImageImportID == uuid.Nil || record.TenantID == uuid.Nil {
			continue
		}
		record.ReleaseState = strings.TrimSpace(record.ReleaseState)
		record.InternalImageRef = strings.TrimSpace(record.InternalImageRef)
		record.SourceImageDigest = strings.TrimSpace(record.SourceImageDigest)
		currentMap[record.ExternalImageImportID] = record
		if _, exists := m.active[record.ExternalImageImportID]; !exists {
			detected = append(detected, record)
		}
	}

	for id, previous := range m.active {
		if _, exists := currentMap[id]; !exists {
			recovered = append(recovered, previous)
		}
	}

	m.active = currentMap
	return detected, recovered
}

func (m *Manager) ActiveCount() int {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.active)
}
