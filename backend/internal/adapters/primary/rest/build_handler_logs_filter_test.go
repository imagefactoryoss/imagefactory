package rest

import (
	"net/url"
	"testing"
)

func TestParseBuildLogsFilter_Valid(t *testing.T) {
	query := url.Values{}
	query.Set("source", "tekton")
	query.Set("min_level", "warn")

	filter, err := parseBuildLogsFilter(query)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filter.source != buildLogSourceTekton {
		t.Fatalf("expected tekton source, got %q", filter.source)
	}
	if string(filter.minLevel) != "warn" {
		t.Fatalf("expected warn min level, got %q", filter.minLevel)
	}
}

func TestParseBuildLogsFilter_Invalid(t *testing.T) {
	query := url.Values{}
	query.Set("source", "bad")

	if _, err := parseBuildLogsFilter(query); err == nil {
		t.Fatalf("expected invalid source error")
	}

	query = url.Values{}
	query.Set("min_level", "bad")
	if _, err := parseBuildLogsFilter(query); err == nil {
		t.Fatalf("expected invalid min_level error")
	}
}

func TestApplyBuildLogsFilter(t *testing.T) {
	logs := []LogEntry{
		{Level: "info", Message: "lifecycle info"},
		{Level: "warn", Message: "lifecycle warn"},
		{Level: "info", Message: "tekton info", Metadata: map[string]interface{}{"source": "tekton"}},
		{Level: "error", Message: "tekton error", Metadata: map[string]interface{}{"source": "tekton"}},
	}

	filtered := applyBuildLogsFilter(logs, buildLogsFilter{
		source:   buildLogSourceTekton,
		minLevel: "warn",
	})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 log after filter, got %d", len(filtered))
	}
	if filtered[0].Message != "tekton error" {
		t.Fatalf("unexpected filtered message: %q", filtered[0].Message)
	}

	filtered = applyBuildLogsFilter(logs, buildLogsFilter{
		source:   buildLogSourceLifecycle,
		minLevel: "info",
	})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 lifecycle logs, got %d", len(filtered))
	}
}
