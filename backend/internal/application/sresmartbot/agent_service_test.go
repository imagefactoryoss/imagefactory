package sresmartbot

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildDraftHypotheses_IncludesAsyncBacklogAndMessagingTransport(t *testing.T) {
	workspace := &IncidentWorkspace{
		Incident: (&incidentFixture{
			Domain:       "golden_signals",
			IncidentType: "backlog_pressure",
			DisplayName:  "Async backlog pressure",
		}).toDomain(),
	}
	toolRuns := []MCPToolInvocationResult{
		{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{
			"build_queue_depth":        int64(12),
			"email_queue_depth":        int64(5),
			"messaging_outbox_pending": int64(8),
		}},
		{ToolName: "messaging_transport.recent", ServerName: "Observability", Payload: map[string]any{
			"reconnects":          int64(4),
			"disconnects":         int64(2),
			"reconnect_threshold": int64(3),
		}},
	}

	hypotheses := buildDraftHypotheses(workspace, toolRuns)
	if len(hypotheses) == 0 {
		t.Fatal("expected hypotheses to be generated")
	}

	foundBacklog := false
	foundTransport := false
	for _, hypothesis := range hypotheses {
		for _, signal := range hypothesis.SignalsUsed {
			if signal == "async_backlog.recent" {
				foundBacklog = true
			}
			if signal == "messaging_transport.recent" {
				foundTransport = true
			}
		}
	}

	if !foundBacklog {
		t.Fatal("expected async_backlog.recent to influence hypotheses")
	}
	if !foundTransport {
		t.Fatal("expected messaging_transport.recent to influence hypotheses")
	}
}

func TestSummarizeToolRun_MessagingTransport(t *testing.T) {
	summary := summarizeToolRun(MCPToolInvocationResult{
		ToolName:   "messaging_transport.recent",
		ServerName: "Observability",
		Payload: map[string]any{
			"reconnects":          int64(4),
			"disconnects":         int64(2),
			"reconnect_threshold": int64(3),
		},
	})

	expected := "Messaging transport shows reconnects=4, disconnects=2, threshold=3."
	if summary != expected {
		t.Fatalf("expected %q, got %q", expected, summary)
	}
}

func TestBuildTriageFromDraft_UsesTopHypothesis(t *testing.T) {
	incidentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	draft := &AgentDraftResponse{
		IncidentID: incidentID,
		Summary:    "Dispatcher backlog is growing with mixed pressure.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Service health is degrading through one or more golden signals",
				Confidence: "high",
				SignalsUsed: []string{
					"messaging_transport.recent",
					"evidence.list",
				},
				EvidenceRefs: []AgentDraftEvidenceRef{
					{ToolName: "messaging_transport.recent", ServerName: "Observability", Summary: "Messaging transport shows reconnects=4, disconnects=2, threshold=3."},
				},
			},
		},
		InvestigationPlan: []AgentDraftPlanStep{
			{Title: "Check transport status", Description: "Confirm reconnect and disconnect pressure is still active."},
			{Title: "Check backlog trend", Description: "Compare latest backlog depth against threshold."},
		},
		HumanConfirmation: true,
	}

	triage := buildTriageFromDraft(draft)
	if triage == nil {
		t.Fatal("expected triage response")
	}
	if triage.ProbableCause != "Service health is degrading through one or more golden signals" {
		t.Fatalf("unexpected probable cause: %q", triage.ProbableCause)
	}
	if triage.Confidence != "high" {
		t.Fatalf("expected high confidence, got %q", triage.Confidence)
	}
	if len(triage.NextChecks) != 3 {
		t.Fatalf("expected 3 next checks, got %d", len(triage.NextChecks))
	}
	if !strings.Contains(triage.RecommendedAction, "review_messaging_transport_health") {
		t.Fatalf("expected transport recommendation, got %q", triage.RecommendedAction)
	}
}

func TestBuildTriageFromDraft_FallsBackWhenHypothesesMissing(t *testing.T) {
	incidentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	draft := &AgentDraftResponse{
		IncidentID: incidentID,
		Summary:    "Evidence remains inconclusive.",
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "findings.list", ServerName: "Observability", Payload: map[string]any{"count": int64(2)}},
			{ToolName: "evidence.list", ServerName: "Observability", Payload: map[string]any{"count": int64(1)}},
		},
	}

	triage := buildTriageFromDraft(draft)
	if triage == nil {
		t.Fatal("expected triage response")
	}
	if triage.ProbableCause != "Evidence remains inconclusive." {
		t.Fatalf("expected summary fallback probable cause, got %q", triage.ProbableCause)
	}
	if triage.Confidence != "medium" {
		t.Fatalf("expected medium confidence fallback, got %q", triage.Confidence)
	}
	if len(triage.EvidenceRefs) == 0 {
		t.Fatal("expected fallback evidence refs")
	}
}
