package runtimehealth

import (
	"strings"
	"sync"
	"time"
)

// ProcessStatus captures runtime health for a background process.
type ProcessStatus struct {
	Enabled      bool
	Running      bool
	LastActivity time.Time
	Message      string
	Metrics      map[string]int64
}

// Provider exposes runtime status for named processes.
type Provider interface {
	GetStatus(name string) (ProcessStatus, bool)
}

// Store is an in-memory runtime status store for background processes.
type Store struct {
	mu     sync.RWMutex
	status map[string]ProcessStatus
}

func NewStore() *Store {
	return &Store{status: make(map[string]ProcessStatus)}
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (s *Store) Upsert(name string, status ProcessStatus) {
	key := normalize(name)
	if key == "" {
		return
	}
	status.Metrics = cloneMetrics(status.Metrics)
	s.mu.Lock()
	s.status[key] = status
	s.mu.Unlock()
}

func (s *Store) Touch(name string) {
	key := normalize(name)
	if key == "" {
		return
	}
	s.mu.Lock()
	st, ok := s.status[key]
	if !ok {
		st = ProcessStatus{}
	}
	st.LastActivity = time.Now().UTC()
	s.status[key] = st
	s.mu.Unlock()
}

func (s *Store) GetStatus(name string) (ProcessStatus, bool) {
	key := normalize(name)
	if key == "" {
		return ProcessStatus{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.status[key]
	if ok {
		st.Metrics = cloneMetrics(st.Metrics)
	}
	return st, ok
}

func cloneMetrics(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
