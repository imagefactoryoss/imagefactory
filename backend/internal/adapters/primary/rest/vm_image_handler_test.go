package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
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

func TestResolveVMLifecycleExecutionMode(t *testing.T) {
	t.Run("defaults to metadata_only", func(t *testing.T) {
		t.Setenv("IF_VM_LIFECYCLE_EXECUTION_MODE", "")
		got := resolveVMLifecycleExecutionMode(zap.NewNop())
		if got != vmLifecycleExecutionModeMetadataOnly {
			t.Fatalf("expected metadata_only default, got %q", got)
		}
	})

	t.Run("accepts require_provider_native", func(t *testing.T) {
		t.Setenv("IF_VM_LIFECYCLE_EXECUTION_MODE", "require_provider_native")
		got := resolveVMLifecycleExecutionMode(zap.NewNop())
		if got != vmLifecycleExecutionModeRequireProviderNative {
			t.Fatalf("expected require_provider_native mode, got %q", got)
		}
	})

	t.Run("invalid mode falls back", func(t *testing.T) {
		t.Setenv("IF_VM_LIFECYCLE_EXECUTION_MODE", "bad-value")
		got := resolveVMLifecycleExecutionMode(zap.NewNop())
		if got != vmLifecycleExecutionModeMetadataOnly {
			t.Fatalf("expected metadata_only fallback, got %q", got)
		}
	})
}

func TestVMDispatchLifecycleExecutor(t *testing.T) {
	t.Run("metadata_only mode returns metadata_only transition", func(t *testing.T) {
		exec := vmDispatchLifecycleExecutor{mode: vmLifecycleExecutionModeMetadataOnly}
		result, err := exec.ExecuteTransition(context.Background(), vmLifecycleTransitionRequest{
			TargetProvider: "aws",
			TargetState:    "deprecated",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.TransitionMode != "metadata_only" {
			t.Fatalf("expected metadata_only transition mode, got %q", result.TransitionMode)
		}
	})

	t.Run("require_provider_native fails closed", func(t *testing.T) {
		exec := vmDispatchLifecycleExecutor{mode: vmLifecycleExecutionModeRequireProviderNative}
		_, err := exec.ExecuteTransition(context.Background(), vmLifecycleTransitionRequest{
			TargetProvider: "gcp",
			TargetState:    "deprecated",
		})
		if !errors.Is(err, errUnsupportedProviderLifecycleTransition) {
			t.Fatalf("expected unsupported provider transition error, got %v", err)
		}
	})

	t.Run("require_provider_native aws requires identifiers", func(t *testing.T) {
		exec := vmDispatchLifecycleExecutor{mode: vmLifecycleExecutionModeRequireProviderNative}
		_, err := exec.ExecuteTransition(context.Background(), vmLifecycleTransitionRequest{
			TargetProvider: "aws",
			TargetState:    "deprecated",
		})
		if !errors.Is(err, errInvalidProviderLifecycleTransitionInput) {
			t.Fatalf("expected invalid provider lifecycle input error, got %v", err)
		}
	})

	t.Run("prefer_provider_native executes aws delete", func(t *testing.T) {
		fake := &fakeVMAWSLifecycleClient{}
		exec := vmDispatchLifecycleExecutor{
			mode: vmLifecycleExecutionModePreferProviderNative,
			awsClientFactory: func(ctx context.Context, region string) (vmAWSLifecycleClient, error) {
				if region != "us-west-2" {
					t.Fatalf("expected region us-west-2, got %q", region)
				}
				return fake, nil
			},
		}
		result, err := exec.ExecuteTransition(context.Background(), vmLifecycleTransitionRequest{
			TargetProvider: "aws",
			TargetState:    "deleted",
			ProviderArtifactIdentifiers: map[string][]string{
				"aws": {"us-west-2:ami-0123456789abcdef0"},
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.TransitionMode != "provider_native" {
			t.Fatalf("expected provider_native mode, got %q", result.TransitionMode)
		}
		if fake.lastImageID != "ami-0123456789abcdef0" {
			t.Fatalf("expected deregister image id ami-0123456789abcdef0, got %q", fake.lastImageID)
		}
		if fake.lastDeprecateImageID != "" {
			t.Fatalf("expected no deprecate call for delete flow, got %q", fake.lastDeprecateImageID)
		}
	})

	t.Run("prefer_provider_native executes aws deprecate", func(t *testing.T) {
		fake := &fakeVMAWSLifecycleClient{}
		exec := vmDispatchLifecycleExecutor{
			mode: vmLifecycleExecutionModePreferProviderNative,
			awsClientFactory: func(ctx context.Context, region string) (vmAWSLifecycleClient, error) {
				if region != "us-east-1" {
					t.Fatalf("expected region us-east-1, got %q", region)
				}
				return fake, nil
			},
		}
		result, err := exec.ExecuteTransition(context.Background(), vmLifecycleTransitionRequest{
			TargetProvider: "aws",
			TargetState:    "deprecated",
			ProviderArtifactIdentifiers: map[string][]string{
				"aws": {"us-east-1:ami-0a1b2c3d4e5f67890"},
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.TransitionMode != "provider_native" {
			t.Fatalf("expected provider_native mode, got %q", result.TransitionMode)
		}
		if fake.lastDeprecateImageID != "ami-0a1b2c3d4e5f67890" {
			t.Fatalf("expected deprecate image id ami-0a1b2c3d4e5f67890, got %q", fake.lastDeprecateImageID)
		}
		if fake.lastImageID != "" {
			t.Fatalf("expected no deregister call for deprecate flow, got %q", fake.lastImageID)
		}
		if fake.lastDeprecateAt.IsZero() {
			t.Fatal("expected deprecate timestamp to be set")
		}
	})
}

func TestParseAWSLifecycleImageReference(t *testing.T) {
	t.Run("parses region prefixed identifier", func(t *testing.T) {
		ref, ok := parseAWSLifecycleImageReference("us-east-1:ami-0123456789abcdef0", "")
		if !ok {
			t.Fatal("expected parse success")
		}
		if ref.Region != "us-east-1" || ref.ImageID != "ami-0123456789abcdef0" {
			t.Fatalf("unexpected ref: %+v", ref)
		}
	})

	t.Run("parses arn identifier", func(t *testing.T) {
		ref, ok := parseAWSLifecycleImageReference("arn:aws:ec2:us-west-2:123456789012:image/ami-0a1b2c3d4e5f67890", "")
		if !ok {
			t.Fatal("expected parse success")
		}
		if ref.Region != "us-west-2" || ref.ImageID != "ami-0a1b2c3d4e5f67890" {
			t.Fatalf("unexpected ref: %+v", ref)
		}
	})

	t.Run("uses default region for bare ami", func(t *testing.T) {
		ref, ok := parseAWSLifecycleImageReference("ami-0123456789abcdef0", "us-east-2")
		if !ok {
			t.Fatal("expected parse success")
		}
		if ref.Region != "us-east-2" || ref.ImageID != "ami-0123456789abcdef0" {
			t.Fatalf("unexpected ref: %+v", ref)
		}
	})
}

type fakeVMAWSLifecycleClient struct {
	lastImageID          string
	lastDeprecateImageID string
	lastDeprecateAt      time.Time
}

func (f *fakeVMAWSLifecycleClient) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	if params != nil && params.ImageId != nil {
		f.lastImageID = *params.ImageId
	}
	return &ec2.DeregisterImageOutput{}, nil
}

func (f *fakeVMAWSLifecycleClient) EnableImageDeprecation(ctx context.Context, params *ec2.EnableImageDeprecationInput, optFns ...func(*ec2.Options)) (*ec2.EnableImageDeprecationOutput, error) {
	if params != nil && params.ImageId != nil {
		f.lastDeprecateImageID = *params.ImageId
	}
	if params != nil && params.DeprecateAt != nil {
		f.lastDeprecateAt = *params.DeprecateAt
	}
	return &ec2.EnableImageDeprecationOutput{}, nil
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
					"at": "2026-03-27T20:00:00Z",
					"transition_mode": "metadata_only"
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
	if lifecycle.History[0].TransitionMode != "metadata_only" {
		t.Fatalf("unexpected lifecycle history transition mode: %q", lifecycle.History[0].TransitionMode)
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
		"provider_native",
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
	if lifecycle.TransitionMode != "provider_native" {
		t.Fatalf("expected lifecycle transition mode provider_native, got %q", lifecycle.TransitionMode)
	}
	if lifecycle.History[0].TransitionMode != "provider_native" {
		t.Fatalf("expected lifecycle history mode provider_native, got %q", lifecycle.History[0].TransitionMode)
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
	if got := vmImageLifecycleTransitionMode("hybrid"); got != "hybrid" {
		t.Fatalf("expected hybrid mode, got %q", got)
	}
	if got := vmImageLifecycleTransitionMode("unknown_mode"); got != "metadata_only" {
		t.Fatalf("expected unknown mode to fallback to metadata_only, got %q", got)
	}
}

func TestVMImageCatalogItemFromRow(t *testing.T) {
	row := vmImageRow{
		ExecutionID:     uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		BuildID:         uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		ProjectID:       uuid.MustParse("33333333-3333-4333-8333-333333333333"),
		ProjectName:     "demo-project",
		BuildNumber:     42,
		BuildStatus:     "success",
		ExecutionStatus: "success",
		CreatedAt:       time.Date(2026, 3, 27, 20, 0, 0, 0, time.UTC),
		MetadataRaw: json.RawMessage(`{
			"packer": {
				"target_provider": "aws",
				"target_profile_id": "profile-1",
				"lifecycle_state": "released",
				"lifecycle_transition_mode": "metadata_only",
				"provider_artifact_identifiers": {
					"aws": ["ami-2", "ami-1"]
				}
			}
		}`),
		ArtifactsRaw: json.RawMessage(`[{"name":"a","value":"ami-1","path":"p"}]`),
	}

	item := vmImageCatalogItemFromRow(row)
	if item.ExecutionID != row.ExecutionID {
		t.Fatalf("unexpected execution id: %s", item.ExecutionID)
	}
	if item.LifecycleState != "released" {
		t.Fatalf("expected released lifecycle state, got %q", item.LifecycleState)
	}
	if item.LifecycleTransitionMode != "metadata_only" {
		t.Fatalf("expected metadata_only transition mode, got %q", item.LifecycleTransitionMode)
	}
	if !item.ActionPermissions.CanDeprecate || item.ActionPermissions.CanDelete {
		t.Fatalf("unexpected action permissions for released state: %+v", item.ActionPermissions)
	}
	if !reflect.DeepEqual(item.ProviderArtifactIdentifiers["aws"], []string{"ami-1", "ami-2"}) {
		t.Fatalf("unexpected provider artifact identifiers: %+v", item.ProviderArtifactIdentifiers)
	}
}

func TestParsePackerMetadataFields_DefaultLifecycleModeOnEmptyOrInvalid(t *testing.T) {
	_, _, _, lifecycle := parsePackerMetadataFields(nil)
	if lifecycle.TransitionMode != "metadata_only" {
		t.Fatalf("expected default metadata_only on empty metadata, got %q", lifecycle.TransitionMode)
	}

	_, _, _, lifecycle = parsePackerMetadataFields(json.RawMessage(`{"packer":`))
	if lifecycle.TransitionMode != "metadata_only" {
		t.Fatalf("expected default metadata_only on invalid metadata, got %q", lifecycle.TransitionMode)
	}
}
