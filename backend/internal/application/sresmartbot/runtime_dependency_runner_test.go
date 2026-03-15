package sresmartbot

import "testing"

func TestRuntimeDependencyIssuesSignature_IsStableAndSorted(t *testing.T) {
	a := []RuntimeDependencyIssue{
		{Key: "dispatcher", Severity: "critical", Message: "down"},
		{Key: "database", Severity: "critical", Message: "timeout"},
	}
	b := []RuntimeDependencyIssue{
		{Key: "database", Severity: "critical", Message: "something else"},
		{Key: "dispatcher", Severity: "critical", Message: "different text"},
	}

	sigA := runtimeDependencyIssuesSignature(a)
	sigB := runtimeDependencyIssuesSignature(b)
	if sigA == "" {
		t.Fatal("expected non-empty signature")
	}
	if sigA != sigB {
		t.Fatalf("expected signatures to match for equivalent issue keys/severity, got %q vs %q", sigA, sigB)
	}
}

func TestRuntimeDependencyIssuesSignature_EmptyWhenNoIssues(t *testing.T) {
	if got := runtimeDependencyIssuesSignature(nil); got != "" {
		t.Fatalf("expected empty signature, got %q", got)
	}
}