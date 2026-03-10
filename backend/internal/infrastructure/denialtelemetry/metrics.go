package denialtelemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type SnapshotRow struct {
	TenantID      string
	CapabilityKey string
	Reason        string
	Labels        map[string]string
	Count         int64
}

type Metrics struct {
	mu     sync.RWMutex
	counts map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		counts: make(map[string]int64),
	}
}

func (m *Metrics) RecordDenied(tenantID uuid.UUID, capabilityKey, reason string) {
	m.RecordDeniedWithLabels(tenantID, capabilityKey, reason, nil)
}

func (m *Metrics) RecordDeniedWithLabels(tenantID uuid.UUID, capabilityKey, reason string, labels map[string]string) {
	if m == nil {
		return
	}
	key := fmt.Sprintf("%s|%s|%s|%s", tenantID.String(), capabilityKey, reason, labelsKey(labels))
	m.mu.Lock()
	m.counts[key]++
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() []SnapshotRow {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	out := make([]SnapshotRow, 0, len(m.counts))
	for key, count := range m.counts {
		parts := splitKey(key)
		out = append(out, SnapshotRow{
			TenantID:      parts[0],
			CapabilityKey: parts[1],
			Reason:        parts[2],
			Labels:        parseLabels(parts[3]),
			Count:         count,
		})
	}
	m.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].TenantID != out[j].TenantID {
			return out[i].TenantID < out[j].TenantID
		}
		if out[i].CapabilityKey != out[j].CapabilityKey {
			return out[i].CapabilityKey < out[j].CapabilityKey
		}
		if out[i].Reason != out[j].Reason {
			return out[i].Reason < out[j].Reason
		}
		return labelsKey(out[i].Labels) < labelsKey(out[j].Labels)
	})
	return out
}

func splitKey(key string) [4]string {
	out := [4]string{}
	start := 0
	idx := 0
	for i := 0; i < len(key) && idx < 3; i++ {
		if key[i] == '|' {
			out[idx] = key[start:i]
			start = i + 1
			idx++
		}
	}
	out[idx] = key[start:]
	return out
}

func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(labels[key])
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, "&")
}

func parseLabels(encoded string) map[string]string {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	out := make(map[string]string)
	for _, part := range strings.Split(encoded, "&") {
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		value := strings.TrimSpace(pair[1])
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
