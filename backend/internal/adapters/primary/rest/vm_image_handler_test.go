package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestVMImageHandler_ListTenantVMImages_RequiresAuthContext(t *testing.T) {
	handler := NewVMImageHandler(nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/vm", nil)
	w := httptest.NewRecorder()
	handler.ListTenantVMImages(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestVMImageHandler_GetTenantVMImage_RequiresAuthContext(t *testing.T) {
	handler := NewVMImageHandler(nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/vm/abc", nil)
	w := httptest.NewRecorder()
	handler.GetTenantVMImage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestVMImageLifecycleState(t *testing.T) {
	cases := map[string]string{
		"success":   "available",
		"running":   "building",
		"pending":   "queued",
		"failed":    "failed",
		"cancelled": "cancelled",
		"":          "unknown",
	}
	for input, want := range cases {
		if got := vmImageLifecycleState(input, ""); got != want {
			t.Fatalf("vmImageLifecycleState(%q): expected %q, got %q", input, want, got)
		}
	}
}

func TestVMImageLifecycleState_UsesOverride(t *testing.T) {
	got := vmImageLifecycleState("success", "released")
	if got != "released" {
		t.Fatalf("expected released lifecycle override, got %q", got)
	}
}

func TestVMImageLifecycleActionPermissions(t *testing.T) {
	cases := []struct {
		name            string
		executionStatus string
		lifecycle       string
		wantPromote     bool
		wantDeprecate   bool
		wantDelete      bool
	}{
		{name: "available", executionStatus: "success", lifecycle: "available", wantPromote: true, wantDeprecate: true, wantDelete: false},
		{name: "released", executionStatus: "success", lifecycle: "released", wantPromote: false, wantDeprecate: true, wantDelete: false},
		{name: "deprecated", executionStatus: "success", lifecycle: "deprecated", wantPromote: true, wantDeprecate: false, wantDelete: true},
		{name: "deleted", executionStatus: "success", lifecycle: "deleted", wantPromote: false, wantDeprecate: false, wantDelete: false},
		{name: "running blocked", executionStatus: "running", lifecycle: "available", wantPromote: false, wantDeprecate: false, wantDelete: false},
	}
	for _, tc := range cases {
		got := vmImageLifecycleActionPermissions(tc.executionStatus, tc.lifecycle)
		if got.CanPromote != tc.wantPromote || got.CanDeprecate != tc.wantDeprecate || got.CanDelete != tc.wantDelete {
			t.Fatalf("%s: unexpected permissions: %+v", tc.name, got)
		}
	}
}

func TestParsePackerMetadataFields(t *testing.T) {
	raw := json.RawMessage(`{
		"packer": {
			"target_provider": "aws",
			"target_profile_id": "11111111-1111-1111-1111-111111111111",
			"lifecycle_state": "released",
			"lifecycle_last_action_at": "2026-03-27T20:00:00Z",
			"lifecycle_last_action_by": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			"lifecycle_last_reason": "promoted by operator",
			"lifecycle_transition_mode": "metadata_only",
			"lifecycle_history": [
				{
					"state": "released",
					"reason": "promoted by operator",
					"actor_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
					"at": "2026-03-27T20:00:00Z"
				}
			],
			"provider_artifact_identifiers": {
				"aws": ["ami-b", "ami-a"],
				"gcp": ["projects/demo/global/images/example"]
			}
		}
	}`)
	provider, profileID, identifiers, lifecycle := parsePackerMetadataFields(raw)
	if provider != "aws" {
		t.Fatalf("expected provider aws, got %q", provider)
	}
	if profileID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected profile id: %q", profileID)
	}
	if !reflect.DeepEqual(identifiers["aws"], []string{"ami-a", "ami-b"}) {
		t.Fatalf("unexpected aws identifiers: %+v", identifiers["aws"])
	}
	if lifecycle.State != "released" {
		t.Fatalf("expected lifecycle override released, got %q", lifecycle.State)
	}
	if lifecycle.LastActionBy != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected lifecycle actor: %q", lifecycle.LastActionBy)
	}
	if lifecycle.TransitionMode != "metadata_only" {
		t.Fatalf("unexpected lifecycle transition mode: %q", lifecycle.TransitionMode)
	}
	if len(lifecycle.History) != 1 || lifecycle.History[0].State != "released" {
		t.Fatalf("unexpected lifecycle history: %+v", lifecycle.History)
	}
}

func TestExtractArtifactValues(t *testing.T) {
	raw := json.RawMessage(`[
		{"name":"artifact-a","value":"ami-123","path":"path-a"},
		{"name":"artifact-b","value":"ami-123","path":"path-b"}
	]`)
	values := extractArtifactValues(raw)
	expected := []string{"ami-123", "artifact-a", "artifact-b", "path-a", "path-b"}
	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("unexpected artifact values: expected %+v, got %+v", expected, values)
	}
}

func TestUpdatePackerLifecycleMetadata(t *testing.T) {
	input := json.RawMessage(`{"packer":{"target_provider":"aws"}}`)
	out, err := updatePackerLifecycleMetadata(
		input,
		"deprecated",
		"stale image",
		uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"),
		time.Date(2026, 3, 27, 20, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("updatePackerLifecycleMetadata returned error: %v", err)
	}
	_, _, _, lifecycle := parsePackerMetadataFields(out)
	if lifecycle.State != "deprecated" {
		t.Fatalf("expected lifecycle override deprecated, got %q", lifecycle.State)
	}
	if len(lifecycle.History) != 1 || lifecycle.History[0].State != "deprecated" {
		t.Fatalf("expected lifecycle history to include deprecated entry, got %+v", lifecycle.History)
	}
	if lifecycle.TransitionMode != "metadata_only" {
		t.Fatalf("expected lifecycle transition mode metadata_only, got %q", lifecycle.TransitionMode)
	}
}

func TestValidateVMLifecycleReason(t *testing.T) {
	t.Run("required reason missing", func(t *testing.T) {
		_, err := validateVMLifecycleReason("   ", true)
		if err == nil || err.Error() != "reason is required for this lifecycle transition" {
			t.Fatalf("expected required reason error, got %v", err)
		}
	})

	t.Run("reason length exceeds max", func(t *testing.T) {
		_, err := validateVMLifecycleReason(strings.Repeat("a", vmImageLifecycleReasonMaxLength+1), true)
		if err == nil || err.Error() != "reason must be 500 characters or fewer" {
			t.Fatalf("expected max length error, got %v", err)
		}
	})

	t.Run("valid required reason", func(t *testing.T) {
		reason, err := validateVMLifecycleReason("  stale provider image  ", true)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if reason != "stale provider image" {
			t.Fatalf("unexpected reason value %q", reason)
		}
	})

	t.Run("optional empty reason allowed", func(t *testing.T) {
		reason, err := validateVMLifecycleReason("  ", false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if reason != "" {
			t.Fatalf("expected empty reason, got %q", reason)
		}
	})
}

func TestVMLifecycleTransitionMode_DefaultsToMetadataOnly(t *testing.T) {
	if got := vmImageLifecycleTransitionMode(""); got != "metadata_only" {
		t.Fatalf("expected default metadata_only mode, got %q", got)
	}
	if got := vmImageLifecycleTransitionMode("  provider_native  "); got != "provider_native" {
		t.Fatalf("expected normalized provider_native mode, got %q", got)
	}
}
