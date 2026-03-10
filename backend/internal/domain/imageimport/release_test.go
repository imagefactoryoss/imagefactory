package imageimport

import (
	"testing"
	"time"
)

func TestDeriveReleaseProjection(t *testing.T) {
	freshEvidenceTime := time.Now().UTC().Add(-1 * time.Hour)
	testCases := []struct {
		name         string
		req          *ImportRequest
		wantState    ReleaseState
		wantEligible bool
		wantBlocker  string
	}{
		{
			name:         "success is ready for release",
			req:          &ImportRequest{Status: StatusSuccess, PolicyDecision: "pass", PolicySnapshotJSON: `{"decision":"pass"}`, ScanSummaryJSON: `{"critical":0}`, SBOMSummaryJSON: `{"packages":42}`, SourceImageDigest: "sha256:abc123", UpdatedAt: freshEvidenceTime},
			wantState:    ReleaseStateReadyForRelease,
			wantEligible: true,
			wantBlocker:  "",
		},
		{
			name:         "success with incomplete evidence is blocked",
			req:          &ImportRequest{Status: StatusSuccess, PolicyDecision: "pass", PolicySnapshotJSON: `{"decision":"pass"}`, ScanSummaryJSON: `{}`, SBOMSummaryJSON: `{"packages":42}`, SourceImageDigest: "sha256:abc123", UpdatedAt: freshEvidenceTime},
			wantState:    ReleaseStateReleaseBlocked,
			wantEligible: false,
			wantBlocker:  "evidence_incomplete",
		},
		{
			name:         "success with stale evidence is blocked",
			req:          &ImportRequest{Status: StatusSuccess, PolicyDecision: "pass", PolicySnapshotJSON: `{"decision":"pass"}`, ScanSummaryJSON: `{"critical":0}`, SBOMSummaryJSON: `{"packages":42}`, SourceImageDigest: "sha256:abc123", UpdatedAt: time.Now().UTC().Add(-31 * 24 * time.Hour)},
			wantState:    ReleaseStateReleaseBlocked,
			wantEligible: false,
			wantBlocker:  "evidence_stale",
		},
		{
			name:         "success with denied policy decision is blocked",
			req:          &ImportRequest{Status: StatusSuccess, PolicyDecision: "deny", PolicySnapshotJSON: `{"decision":"deny"}`, ScanSummaryJSON: `{"critical":0}`, SBOMSummaryJSON: `{"packages":42}`, SourceImageDigest: "sha256:abc123", UpdatedAt: freshEvidenceTime},
			wantState:    ReleaseStateReleaseBlocked,
			wantEligible: false,
			wantBlocker:  "policy_not_passed",
		},
		{
			name:         "quarantined is blocked",
			req:          &ImportRequest{Status: StatusQuarantined},
			wantState:    ReleaseStateReleaseBlocked,
			wantEligible: false,
			wantBlocker:  "policy_quarantined",
		},
		{
			name:         "failed is blocked",
			req:          &ImportRequest{Status: StatusFailed},
			wantState:    ReleaseStateReleaseBlocked,
			wantEligible: false,
			wantBlocker:  "import_failed",
		},
		{
			name:         "pending is not ready",
			req:          &ImportRequest{Status: StatusPending},
			wantState:    ReleaseStateNotReady,
			wantEligible: false,
			wantBlocker:  "import_not_terminal",
		},
		{
			name:         "approved is not ready",
			req:          &ImportRequest{Status: StatusApproved},
			wantState:    ReleaseStateNotReady,
			wantEligible: false,
			wantBlocker:  "import_not_terminal",
		},
		{
			name:         "importing is not ready",
			req:          &ImportRequest{Status: StatusImporting},
			wantState:    ReleaseStateNotReady,
			wantEligible: false,
			wantBlocker:  "import_not_terminal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveReleaseProjection(tc.req)
			if got.State != tc.wantState {
				t.Fatalf("expected state %q, got %q", tc.wantState, got.State)
			}
			if got.Eligible != tc.wantEligible {
				t.Fatalf("expected eligible %v, got %v", tc.wantEligible, got.Eligible)
			}
			if got.BlockerReason != tc.wantBlocker {
				t.Fatalf("expected blocker %q, got %q", tc.wantBlocker, got.BlockerReason)
			}
		})
	}
}

func TestDeriveReleaseProjection_NilRequest(t *testing.T) {
	got := DeriveReleaseProjection(nil)
	if got.State != ReleaseStateUnknown {
		t.Fatalf("expected unknown state, got %q", got.State)
	}
	if got.Eligible {
		t.Fatalf("expected eligible false")
	}
	if got.BlockerReason != "import_request_missing" {
		t.Fatalf("expected blocker import_request_missing, got %q", got.BlockerReason)
	}
}

func TestValidateReleaseTransition(t *testing.T) {
	testCases := []struct {
		name      string
		current   ReleaseState
		target    ReleaseState
		wantError bool
	}{
		{name: "not_ready to ready_for_release", current: ReleaseStateNotReady, target: ReleaseStateReadyForRelease, wantError: false},
		{name: "not_ready to release_blocked", current: ReleaseStateNotReady, target: ReleaseStateReleaseBlocked, wantError: false},
		{name: "ready_for_release to release_approved", current: ReleaseStateReadyForRelease, target: ReleaseStateReleaseApproved, wantError: false},
		{name: "ready_for_release to release_blocked", current: ReleaseStateReadyForRelease, target: ReleaseStateReleaseBlocked, wantError: false},
		{name: "release_approved to released", current: ReleaseStateReleaseApproved, target: ReleaseStateReleased, wantError: false},
		{name: "release_blocked to ready_for_release", current: ReleaseStateReleaseBlocked, target: ReleaseStateReadyForRelease, wantError: false},
		{name: "same state no-op", current: ReleaseStateReadyForRelease, target: ReleaseStateReadyForRelease, wantError: false},
		{name: "invalid ready_for_release to released", current: ReleaseStateReadyForRelease, target: ReleaseStateReleased, wantError: true},
		{name: "invalid released to ready_for_release", current: ReleaseStateReleased, target: ReleaseStateReadyForRelease, wantError: true},
		{name: "invalid unknown current", current: ReleaseStateUnknown, target: ReleaseStateReadyForRelease, wantError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateReleaseTransition(tc.current, tc.target)
			if tc.wantError && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestResolveReleaseProjection_PrefersPersistedState(t *testing.T) {
	req := &ImportRequest{
		Status:               StatusSuccess,
		ReleaseState:         ReleaseStateReleased,
		ReleaseBlockerReason: "",
	}
	got := ResolveReleaseProjection(req)
	if got.State != ReleaseStateReleased {
		t.Fatalf("expected released state, got %q", got.State)
	}
	if got.Eligible {
		t.Fatalf("expected eligible false for released state")
	}
}

func TestResolveReleaseProjection_DoesNotTrustPersistedReadyWhenEvidenceIncomplete(t *testing.T) {
	req := &ImportRequest{
		Status:               StatusSuccess,
		ReleaseState:         ReleaseStateReadyForRelease,
		PolicyDecision:       "pass",
		PolicySnapshotJSON:   `{"decision":"pass"}`,
		ScanSummaryJSON:      `{}`,
		SBOMSummaryJSON:      `{"packages":42}`,
		SourceImageDigest:    "sha256:abc123",
		UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
		ReleaseBlockerReason: "",
	}
	got := ResolveReleaseProjection(req)
	if got.State != ReleaseStateReleaseBlocked {
		t.Fatalf("expected state %q, got %q", ReleaseStateReleaseBlocked, got.State)
	}
	if got.Eligible {
		t.Fatalf("expected eligible false")
	}
	if got.BlockerReason != "evidence_incomplete" {
		t.Fatalf("expected blocker evidence_incomplete, got %q", got.BlockerReason)
	}
}
