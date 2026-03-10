package releasealerts

import (
	"sync"

	"github.com/google/uuid"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
)

// Transition captures one release-governance alert state transition.
type Transition struct {
	PreviousDegraded       bool
	CurrentDegraded        bool
	FailureRatio           float64
	ConsecutiveFailures    int
	BreachByFailureRatio   bool
	BreachByFailureBurst   bool
	FailureRatioThreshold  float64
	FailureBurstThreshold  int
	MinimumSamples         int
	ReleaseMetricsSnapshot releasetelemetry.Snapshot
}

type tenantState struct {
	degraded            bool
	consecutiveFailures int
}

// Manager tracks per-tenant release alert state and emits transitions only.
type Manager struct {
	mu     sync.Mutex
	states map[uuid.UUID]tenantState
}

func NewManager() *Manager {
	return &Manager{
		states: make(map[uuid.UUID]tenantState),
	}
}

// RecordAndEvaluate updates internal counters from eventType and evaluates
// whether alert state transitioned (degraded <-> healthy).
func (m *Manager) RecordAndEvaluate(
	tenantID uuid.UUID,
	eventType string,
	snapshot releasetelemetry.Snapshot,
	policy systemconfig.ReleaseGovernancePolicyConfig,
) (*Transition, bool) {
	if m == nil || tenantID == uuid.Nil {
		return nil, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.states[tenantID]
	switch eventType {
	case messaging.EventTypeQuarantineReleaseFailed:
		state.consecutiveFailures++
	case messaging.EventTypeQuarantineReleased:
		state.consecutiveFailures = 0
	}

	total := float64(snapshot.Total)
	failed := float64(snapshot.Failed)
	failureRatio := 0.0
	if total > 0 {
		failureRatio = failed / total
	}

	sufficientSamples := int(snapshot.Total) >= policy.MinimumSamples
	breachByRatio := policy.Enabled && sufficientSamples && failureRatio >= policy.FailureRatioThreshold
	breachByBurst := policy.Enabled && state.consecutiveFailures >= policy.ConsecutiveFailuresThreshold
	currentDegraded := breachByRatio || breachByBurst
	previousDegraded := state.degraded

	state.degraded = currentDegraded
	m.states[tenantID] = state

	if previousDegraded == currentDegraded {
		return nil, false
	}

	return &Transition{
		PreviousDegraded:       previousDegraded,
		CurrentDegraded:        currentDegraded,
		FailureRatio:           failureRatio,
		ConsecutiveFailures:    state.consecutiveFailures,
		BreachByFailureRatio:   breachByRatio,
		BreachByFailureBurst:   breachByBurst,
		FailureRatioThreshold:  policy.FailureRatioThreshold,
		FailureBurstThreshold:  policy.ConsecutiveFailuresThreshold,
		MinimumSamples:         policy.MinimumSamples,
		ReleaseMetricsSnapshot: snapshot,
	}, true
}

