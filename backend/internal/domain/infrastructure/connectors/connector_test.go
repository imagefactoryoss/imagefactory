package connectors

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestKubernetesConnectorRejectsLegacyTopLevelAuth(t *testing.T) {
	connector := NewKubernetesConnector(map[string]interface{}{
		"auth_method": "token",
		"endpoint":    "https://k8s.example.com:6443",
		"token":       "dummy-token",
	}, zap.NewNop())

	result, err := connector.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.Success {
		t.Fatalf("expected failure for legacy top-level auth config")
	}
	if !strings.Contains(result.Message, "runtime auth config") {
		t.Fatalf("expected runtime auth error, got %q", result.Message)
	}
}

func TestKubernetesConnectorRequiresRuntimeAuthObject(t *testing.T) {
	connector := NewKubernetesConnector(map[string]interface{}{
		"runtime_auth": map[string]interface{}{
			"endpoint": "https://k8s.example.com:6443",
			"token":    "dummy-token",
		},
	}, zap.NewNop())

	result, err := connector.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.Success {
		t.Fatalf("expected failure when runtime_auth.auth_method is missing")
	}
	if !strings.Contains(result.Message, "auth_method is required") {
		t.Fatalf("expected missing auth_method message, got %q", result.Message)
	}
}
