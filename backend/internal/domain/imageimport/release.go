package imageimport

import (
	"fmt"
	"strings"
	"time"
)

var ErrInvalidReleaseTransition = fmt.Errorf("invalid release state transition")

type ReleaseState string

const (
	ReleaseStateNotReady        ReleaseState = "not_ready"
	ReleaseStateReadyForRelease ReleaseState = "ready_for_release"
	ReleaseStateReleaseApproved ReleaseState = "release_approved"
	ReleaseStateReleased        ReleaseState = "released"
	ReleaseStateReleaseBlocked  ReleaseState = "release_blocked"
	ReleaseStateUnknown         ReleaseState = "unknown"
)

type ReleaseProjection struct {
	State         ReleaseState
	Eligible      bool
	BlockerReason string
}

const releaseEvidenceFreshnessWindow = 30 * 24 * time.Hour

// ResolveReleaseProjection returns persisted release state when present and
// falls back to status-derived projection for backward compatibility.
func ResolveReleaseProjection(req *ImportRequest) ReleaseProjection {
	derived := DeriveReleaseProjection(req)
	if req == nil {
		return derived
	}

	persisted := ReleaseState(strings.TrimSpace(string(req.ReleaseState)))
	if persisted == "" {
		return derived
	}
	// Persisted terminal states are authoritative, but pre-release states
	// must continue to reflect current evidence-based readiness.
	if persisted != ReleaseStateReleaseApproved && persisted != ReleaseStateReleased {
		return derived
	}

	projection := ReleaseProjection{
		State:         persisted,
		Eligible:      persisted == ReleaseStateReadyForRelease,
		BlockerReason: strings.TrimSpace(req.ReleaseBlockerReason),
	}
	if projection.BlockerReason == "" {
		projection.BlockerReason = derived.BlockerReason
	}
	return projection
}

// DeriveReleaseProjection returns deterministic release-readiness semantics for an import.
func DeriveReleaseProjection(req *ImportRequest) ReleaseProjection {
	if req == nil {
		return ReleaseProjection{State: ReleaseStateUnknown, Eligible: false, BlockerReason: "import_request_missing"}
	}

	switch req.Status {
	case StatusSuccess:
		if !policyDecisionAllowsRelease(req.PolicyDecision) {
			return ReleaseProjection{State: ReleaseStateReleaseBlocked, Eligible: false, BlockerReason: "policy_not_passed"}
		}
		if !hasReleaseEvidence(req) {
			return ReleaseProjection{State: ReleaseStateReleaseBlocked, Eligible: false, BlockerReason: "evidence_incomplete"}
		}
		if evidenceIsStale(req.UpdatedAt) {
			return ReleaseProjection{State: ReleaseStateReleaseBlocked, Eligible: false, BlockerReason: "evidence_stale"}
		}
		return ReleaseProjection{State: ReleaseStateReadyForRelease, Eligible: true}
	case StatusQuarantined:
		return ReleaseProjection{State: ReleaseStateReleaseBlocked, Eligible: false, BlockerReason: "policy_quarantined"}
	case StatusFailed:
		return ReleaseProjection{State: ReleaseStateReleaseBlocked, Eligible: false, BlockerReason: "import_failed"}
	case StatusPending, StatusApproved, StatusImporting:
		return ReleaseProjection{State: ReleaseStateNotReady, Eligible: false, BlockerReason: "import_not_terminal"}
	default:
		return ReleaseProjection{State: ReleaseStateUnknown, Eligible: false, BlockerReason: "unknown_import_status"}
	}
}

func policyDecisionAllowsRelease(raw string) bool {
	decision := strings.TrimSpace(strings.ToLower(raw))
	return decision == "" || decision == "pass" || decision == "allow" || decision == "approved"
}

func hasReleaseEvidence(req *ImportRequest) bool {
	if req == nil {
		return false
	}
	return hasEvidencePayload(req.PolicySnapshotJSON) &&
		hasEvidencePayload(req.ScanSummaryJSON) &&
		hasEvidencePayload(req.SBOMSummaryJSON) &&
		strings.TrimSpace(req.SourceImageDigest) != ""
}

func hasEvidencePayload(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	switch strings.ToLower(trimmed) {
	case "{}", "[]", "null":
		return false
	default:
		return true
	}
}

func evidenceIsStale(updatedAt time.Time) bool {
	if updatedAt.IsZero() {
		return true
	}
	return time.Since(updatedAt.UTC()) > releaseEvidenceFreshnessWindow
}

// ValidateReleaseTransition enforces allowed release lifecycle transitions.
func ValidateReleaseTransition(current, target ReleaseState) error {
	if current == target {
		return nil
	}

	allowedTargets := map[ReleaseState]map[ReleaseState]struct{}{
		ReleaseStateNotReady: {
			ReleaseStateReadyForRelease: {},
			ReleaseStateReleaseBlocked:  {},
		},
		ReleaseStateReadyForRelease: {
			ReleaseStateReleaseApproved: {},
			ReleaseStateReleaseBlocked:  {},
		},
		ReleaseStateReleaseApproved: {
			ReleaseStateReleased: {},
		},
		ReleaseStateReleaseBlocked: {
			ReleaseStateReadyForRelease: {},
		},
		ReleaseStateReleased: {},
	}

	targets, ok := allowedTargets[current]
	if !ok {
		return fmt.Errorf("%w: current=%s target=%s", ErrInvalidReleaseTransition, current, target)
	}
	if _, ok := targets[target]; !ok {
		return fmt.Errorf("%w: current=%s target=%s", ErrInvalidReleaseTransition, current, target)
	}
	return nil
}
