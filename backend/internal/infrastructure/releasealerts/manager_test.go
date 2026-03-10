package releasealerts

import (
	"testing"

	"github.com/google/uuid"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
)

func TestManager_RecordAndEvaluate_DedupesAndRecovers(t *testing.T) {
	manager := NewManager()
	tenantID := uuid.New()
	policy := systemconfig.ReleaseGovernancePolicyConfig{
		Enabled:                      true,
		FailureRatioThreshold:        0.90,
		ConsecutiveFailuresThreshold: 2,
		MinimumSamples:               1,
		WindowMinutes:                60,
	}

	// First failure transitions to degraded.
	first, emit := manager.RecordAndEvaluate(
		tenantID,
		messaging.EventTypeQuarantineReleaseFailed,
		releasetelemetry.Snapshot{Failed: 1, Total: 1},
		policy,
	)
	if !emit || first == nil {
		t.Fatalf("expected degraded transition on first failure")
	}
	if !first.CurrentDegraded {
		t.Fatalf("expected degraded=true after first failure")
	}

	// Second failure keeps degraded; should not emit duplicate alert.
	second, emit := manager.RecordAndEvaluate(
		tenantID,
		messaging.EventTypeQuarantineReleaseFailed,
		releasetelemetry.Snapshot{Failed: 2, Total: 2},
		policy,
	)
	if emit || second != nil {
		t.Fatalf("expected no transition while already degraded")
	}

	// One release lowers ratio and resets failure burst; should recover.
	third, emit := manager.RecordAndEvaluate(
		tenantID,
		messaging.EventTypeQuarantineReleased,
		releasetelemetry.Snapshot{Failed: 2, Released: 1, Total: 3},
		policy,
	)
	if !emit || third == nil {
		t.Fatalf("expected recovery transition after successful release")
	}
	if third.CurrentDegraded {
		t.Fatalf("expected degraded=false after recovery")
	}
	if third.ConsecutiveFailures != 0 {
		t.Fatalf("expected consecutive failures reset to 0, got %d", third.ConsecutiveFailures)
	}
}

