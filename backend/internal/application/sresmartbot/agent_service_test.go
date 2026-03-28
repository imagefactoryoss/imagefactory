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

func TestBuildSeverityFromDraft_CorrelatesSignals(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Summary:    "Degradation detected across transport and HTTP windows.",
		Hypotheses: []AgentDraftHypothesis{
			{Title: "Service health is degrading through one or more golden signals", Confidence: "high"},
		},
		ToolRuns: []MCPToolInvocationResult{
			{
				ToolName:   "logs.recent",
				ServerName: "Observability",
				Payload: map[string]any{
					"match_count": int64(9),
				},
			},
			{
				ToolName:   "http_signals.recent",
				ServerName: "Observability",
				Payload: map[string]any{
					"error_rate_percent": int64(7),
					"average_latency_ms": int64(1200),
				},
			},
			{
				ToolName:   "messaging_transport.recent",
				ServerName: "Observability",
				Payload: map[string]any{
					"reconnects":          int64(5),
					"disconnects":         int64(1),
					"reconnect_threshold": int64(3),
				},
			},
		},
	}

	severity := buildSeverityFromDraft(draft)
	if severity == nil {
		t.Fatal("expected severity response")
	}
	if severity.Score < 80 {
		t.Fatalf("expected score >= 80, got %d", severity.Score)
	}
	if severity.Level != "critical" {
		t.Fatalf("expected critical level, got %q", severity.Level)
	}
	if len(severity.Factors) == 0 {
		t.Fatal("expected correlated severity factors")
	}
}

func TestBuildSeverityFromDraft_UsesFallbackForSparseEvidence(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Summary:    "Initial signal observed.",
	}

	severity := buildSeverityFromDraft(draft)
	if severity == nil {
		t.Fatal("expected severity response")
	}
	if severity.Score != 25 {
		t.Fatalf("expected baseline score 25, got %d", severity.Score)
	}
	if severity.Level != "low" {
		t.Fatalf("expected low level, got %q", severity.Level)
	}
	if len(severity.Factors) != 0 {
		t.Fatalf("expected no factors for sparse evidence, got %d", len(severity.Factors))
	}
}

func TestBuildSuggestedActionFromDraft_IsAdvisoryAndApprovalBound(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		Summary:    "Messaging transport and backlog pressure are rising.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Messaging transport instability may be contributing to backlog growth",
				Confidence: "high",
				SignalsUsed: []string{
					"messaging_transport.recent",
					"async_backlog.recent",
				},
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "messaging_transport.recent", ServerName: "Observability", Payload: map[string]any{"reconnects": int64(4), "disconnects": int64(2), "reconnect_threshold": int64(3)}},
			{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{"build_queue_depth": int64(9), "email_queue_depth": int64(4), "messaging_outbox_pending": int64(7)}},
		},
	}

	suggestion := buildSuggestedActionFromDraft(draft)
	if suggestion == nil {
		t.Fatal("expected suggested action response")
	}
	if suggestion.Mode != "deterministic_advisory_suggested_action" {
		t.Fatalf("unexpected mode: %q", suggestion.Mode)
	}
	if suggestion.ActionKey != "review_messaging_transport_health" {
		t.Fatalf("expected transport advisory action, got %q", suggestion.ActionKey)
	}
	if !suggestion.AdvisoryOnly {
		t.Fatal("expected advisory-only suggestion")
	}
	if !suggestion.ExecutionRequiresApproval {
		t.Fatal("expected approval-required execution guard")
	}
	guard := strings.ToLower(suggestion.ExecutionGuardrail)
	if !strings.Contains(guard, "advisory-only") || !strings.Contains(guard, "approval") {
		t.Fatalf("expected guardrail text to enforce approval, got %q", suggestion.ExecutionGuardrail)
	}
}

func TestBuildSuggestedActionFromDraft_CategorizesBlastRadius(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		Summary:    "Release drift detected with cross-service impact.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Release drift or partial rollout is the primary cause",
				Confidence: "high",
				SignalsUsed: []string{
					"release_drift.summary",
				},
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "release_drift.summary", ServerName: "Release", Payload: map[string]any{"active_drift_count": int64(3)}},
		},
	}

	suggestion := buildSuggestedActionFromDraft(draft)
	if suggestion == nil {
		t.Fatal("expected suggested action response")
	}
	if suggestion.BlastRadius != "high" {
		t.Fatalf("expected high blast radius for release drift, got %q", suggestion.BlastRadius)
	}
	if suggestion.ActionKey != "review_release_drift" {
		t.Fatalf("expected release drift advisory action, got %q", suggestion.ActionKey)
	}
}

func TestBuildIncidentScorecardFromDraft_ProjectsSeverityAndTriage(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		Summary:    "Transport instability and backlog pressure are worsening.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Messaging transport instability may be contributing to backlog growth",
				Confidence: "high",
				SignalsUsed: []string{
					"messaging_transport.recent",
					"async_backlog.recent",
				},
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "messaging_transport.recent", ServerName: "Observability", Payload: map[string]any{"reconnects": int64(5), "disconnects": int64(2), "reconnect_threshold": int64(3)}},
			{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{"build_queue_depth": int64(12), "email_queue_depth": int64(4), "messaging_outbox_pending": int64(8)}},
		},
		HumanConfirmation: true,
	}

	scorecard := buildIncidentScorecardFromDraft(draft)
	if scorecard == nil {
		t.Fatal("expected incident scorecard response")
	}
	if scorecard.Mode != "deterministic_incident_scorecard" {
		t.Fatalf("unexpected mode: %q", scorecard.Mode)
	}
	if scorecard.SeverityScore < 60 {
		t.Fatalf("expected high severity score, got %d", scorecard.SeverityScore)
	}
	if scorecard.SeverityLevel != "high" {
		t.Fatalf("expected high severity level, got %q", scorecard.SeverityLevel)
	}
	if scorecard.ProbableCause == "" || scorecard.Confidence == "" {
		t.Fatalf("expected populated probable cause/confidence, got cause=%q confidence=%q", scorecard.ProbableCause, scorecard.Confidence)
	}
	if scorecard.ActionKey != "review_messaging_transport_health" {
		t.Fatalf("expected transport review action key, got %q", scorecard.ActionKey)
	}
	if !scorecard.ExecutionRequiresApproval {
		t.Fatal("expected scorecard action to remain approval-bound")
	}
	if len(scorecard.WhySevereCards) == 0 {
		t.Fatal("expected at least one why-severe card")
	}
}

func TestBuildIncidentScorecardFromDraft_TrimsWhySevereCards(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		Summary:    "Correlated degradation across logs/http/backlog/transport.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Service health is degrading through one or more golden signals",
				Confidence: "high",
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "logs.recent", ServerName: "Observability", Payload: map[string]any{"match_count": int64(15)}},
			{ToolName: "http_signals.recent", ServerName: "Observability", Payload: map[string]any{"error_rate_percent": int64(9), "average_latency_ms": int64(1300)}},
			{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{"build_queue_depth": int64(25), "email_queue_depth": int64(11), "messaging_outbox_pending": int64(16)}},
			{ToolName: "messaging_transport.recent", ServerName: "Observability", Payload: map[string]any{"reconnects": int64(8), "disconnects": int64(3), "reconnect_threshold": int64(3)}},
		},
	}

	scorecard := buildIncidentScorecardFromDraft(draft)
	if scorecard == nil {
		t.Fatal("expected incident scorecard response")
	}
	if len(scorecard.WhySevereCards) > 3 {
		t.Fatalf("expected why-severe cards to be trimmed to 3, got %d", len(scorecard.WhySevereCards))
	}
}

func TestBuildIncidentSnapshotFromDraft_ComposesDeterministicViews(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("99999999-9999-9999-9999-999999999999"),
		Summary:    "Transport and backlog pressure are correlated.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Messaging transport instability may be contributing to backlog growth",
				Confidence: "high",
				SignalsUsed: []string{
					"messaging_transport.recent",
					"async_backlog.recent",
				},
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "messaging_transport.recent", ServerName: "Observability", Payload: map[string]any{"reconnects": int64(6), "disconnects": int64(2), "reconnect_threshold": int64(3)}},
			{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{"build_queue_depth": int64(10), "email_queue_depth": int64(4), "messaging_outbox_pending": int64(8)}},
		},
		HumanConfirmation: true,
	}

	snapshot := buildIncidentSnapshotFromDraft(draft)
	if snapshot == nil {
		t.Fatal("expected incident snapshot response")
	}
	if snapshot.Mode != "deterministic_incident_snapshot" {
		t.Fatalf("unexpected mode: %q", snapshot.Mode)
	}
	if snapshot.Triage == nil || snapshot.Severity == nil || snapshot.Scorecard == nil || snapshot.SuggestedAction == nil {
		t.Fatal("expected snapshot to include triage, severity, scorecard, and suggested action")
	}
	if snapshot.Scorecard.ActionKey != snapshot.SuggestedAction.ActionKey {
		t.Fatalf("expected scorecard/suggested-action keys to align, got %q and %q", snapshot.Scorecard.ActionKey, snapshot.SuggestedAction.ActionKey)
	}
	if snapshot.OperatorHandoff == "" {
		t.Fatal("expected snapshot operator handoff note")
	}
	if len(snapshot.PolicyGuardrails) == 0 {
		t.Fatal("expected snapshot policy guardrails")
	}
	if snapshot.EvidenceCoveragePercent <= 0 {
		t.Fatalf("expected positive evidence coverage, got %d", snapshot.EvidenceCoveragePercent)
	}
	if len(snapshot.EvidenceSignalsExpected) == 0 {
		t.Fatal("expected evidence expected signals")
	}
	if len(snapshot.EvidenceSignalsObserved) == 0 {
		t.Fatal("expected evidence observed signals")
	}
}

func TestBuildIncidentSnapshotFromDraft_RemainsApprovalBound(t *testing.T) {
	draft := &AgentDraftResponse{
		IncidentID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Summary:    "Release drift detected.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Release drift or partial rollout is the primary cause",
				Confidence: "high",
				SignalsUsed: []string{
					"release_drift.summary",
				},
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "release_drift.summary", ServerName: "Release", Payload: map[string]any{"active_drift_count": int64(3)}},
		},
	}

	snapshot := buildIncidentSnapshotFromDraft(draft)
	if snapshot == nil || snapshot.Scorecard == nil || snapshot.SuggestedAction == nil {
		t.Fatal("expected snapshot scorecard and suggested action")
	}
	if !snapshot.Scorecard.ExecutionRequiresApproval || !snapshot.SuggestedAction.ExecutionRequiresApproval {
		t.Fatal("expected incident snapshot actions to remain approval-bound")
	}
	if !strings.Contains(strings.ToLower(snapshot.OperatorHandoff), "approval-bound") {
		t.Fatalf("expected operator handoff to preserve approval-bound language, got %q", snapshot.OperatorHandoff)
	}
	guardrails := strings.ToLower(strings.Join(snapshot.PolicyGuardrails, " "))
	if !strings.Contains(guardrails, "advisory") || !strings.Contains(guardrails, "approval") {
		t.Fatalf("expected guardrails to include advisory+approval language, got %q", strings.Join(snapshot.PolicyGuardrails, " | "))
	}
}

func TestBuildSnapshotEvidenceHealth_CoverageBands(t *testing.T) {
	draft := &AgentDraftResponse{
		ToolRuns: []MCPToolInvocationResult{
			{ToolName: "findings.list"},
			{ToolName: "evidence.list"},
			{ToolName: "http_signals.recent"},
			{ToolName: "async_backlog.recent"},
			{ToolName: "messaging_transport.recent"},
			{ToolName: "logs.recent"},
		},
	}
	expected, observed, coverage, note := buildSnapshotEvidenceHealth(draft)
	if len(expected) == 0 || len(observed) == 0 {
		t.Fatal("expected non-empty expected/observed signals")
	}
	if coverage != 100 {
		t.Fatalf("expected full evidence coverage, got %d", coverage)
	}
	if !strings.Contains(strings.ToLower(note), "strong") {
		t.Fatalf("expected strong coverage note, got %q", note)
	}
}
