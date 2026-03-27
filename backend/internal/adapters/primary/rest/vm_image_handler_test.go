package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

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
		if got := vmImageLifecycleState(input); got != want {
			t.Fatalf("vmImageLifecycleState(%q): expected %q, got %q", input, want, got)
		}
	}
}

func TestParsePackerMetadataFields(t *testing.T) {
	raw := json.RawMessage(`{
		"packer": {
			"target_provider": "aws",
			"target_profile_id": "11111111-1111-1111-1111-111111111111",
			"provider_artifact_identifiers": {
				"aws": ["ami-b", "ami-a"],
				"gcp": ["projects/demo/global/images/example"]
			}
		}
	}`)
	provider, profileID, identifiers := parsePackerMetadataFields(raw)
	if provider != "aws" {
		t.Fatalf("expected provider aws, got %q", provider)
	}
	if profileID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected profile id: %q", profileID)
	}
	if !reflect.DeepEqual(identifiers["aws"], []string{"ami-a", "ami-b"}) {
		t.Fatalf("unexpected aws identifiers: %+v", identifiers["aws"])
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
