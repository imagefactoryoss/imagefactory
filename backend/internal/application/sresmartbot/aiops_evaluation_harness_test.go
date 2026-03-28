package sresmartbot

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

type aiopsReplayCase struct {
	name                string
	draft               *AgentDraftResponse
	expectedCausePart   string
	expectedConfidence  string
	expectedActionKey   string
	expectedBlastRadius string
	expectedSeverity    string
	minSeverityScore    int64
}

func TestAIOpsEvaluationHarness_ReplaySuite(t *testing.T) {
	replays := []aiopsReplayCase{
		{
			name: "transport_backlog_pressure",
			draft: &AgentDraftResponse{
				IncidentID: stableIncidentID(),
				Summary:    "Messaging transport and async backlog pressure are both increasing.",
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
					{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{"build_queue_depth": int64(11), "email_queue_depth": int64(5), "messaging_outbox_pending": int64(9)}},
				},
				HumanConfirmation: true,
			},
			expectedCausePart:   "transport instability",
			expectedConfidence:  "high",
			expectedActionKey:   "review_messaging_transport_health",
			expectedBlastRadius: "low",
			expectedSeverity:    "high",
			minSeverityScore:    60,
		},
		{
			name: "release_drift_cross_service",
			draft: &AgentDraftResponse{
				IncidentID: stableIncidentID(),
				Summary:    "Release drift is visible across dependent services.",
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
					{ToolName: "release_drift.summary", ServerName: "Release", Payload: map[string]any{"active_drift_count": int64(4)}},
				},
				HumanConfirmation: true,
			},
			expectedCausePart:   "release drift",
			expectedConfidence:  "high",
			expectedActionKey:   "review_release_drift",
			expectedBlastRadius: "high",
			expectedSeverity:    "medium",
			minSeverityScore:    35,
		},
		{
			name: "sparse_evidence_fallback",
			draft: &AgentDraftResponse{
				IncidentID: stableIncidentID(),
				Summary:    "Evidence remains inconclusive.",
				ToolRuns: []MCPToolInvocationResult{
					{ToolName: "findings.list", ServerName: "Observability", Payload: map[string]any{"count": int64(2)}},
					{ToolName: "evidence.list", ServerName: "Observability", Payload: map[string]any{"count": int64(1)}},
				},
				HumanConfirmation: true,
			},
			expectedCausePart:   "inconclusive",
			expectedConfidence:  "medium",
			expectedActionKey:   "review_runtime_health",
			expectedBlastRadius: "low",
			expectedSeverity:    "low",
			minSeverityScore:    25,
		},
	}

	for _, replay := range replays {
		replay := replay
		t.Run(replay.name, func(t *testing.T) {
			triage := buildTriageFromDraft(replay.draft)
			if triage == nil {
				t.Fatal("expected triage output")
			}
			if !strings.Contains(strings.ToLower(triage.ProbableCause), strings.ToLower(replay.expectedCausePart)) {
				t.Fatalf("correctness: probable_cause mismatch, expected substring %q, got %q", replay.expectedCausePart, triage.ProbableCause)
			}
			if triage.Confidence != replay.expectedConfidence {
				t.Fatalf("correctness: expected confidence %q, got %q", replay.expectedConfidence, triage.Confidence)
			}

			severity := buildSeverityFromDraft(replay.draft)
			if severity == nil {
				t.Fatal("expected severity output")
			}
			if severity.Level != replay.expectedSeverity {
				t.Fatalf("correctness: expected severity level %q, got %q", replay.expectedSeverity, severity.Level)
			}
			if severity.Score < replay.minSeverityScore {
				t.Fatalf("correctness: expected severity score >= %d, got %d", replay.minSeverityScore, severity.Score)
			}

			suggestion := buildSuggestedActionFromDraft(replay.draft)
			if suggestion == nil {
				t.Fatal("expected suggested action output")
			}
			if suggestion.ActionKey != replay.expectedActionKey {
				t.Fatalf("correctness: expected action key %q, got %q", replay.expectedActionKey, suggestion.ActionKey)
			}
			if suggestion.BlastRadius != replay.expectedBlastRadius {
				t.Fatalf("correctness: expected blast radius %q, got %q", replay.expectedBlastRadius, suggestion.BlastRadius)
			}

			if !suggestion.AdvisoryOnly || !suggestion.ExecutionRequiresApproval {
				t.Fatalf("policy_compliance: expected advisory_only=true and execution_requires_approval=true, got advisory_only=%v execution_requires_approval=%v", suggestion.AdvisoryOnly, suggestion.ExecutionRequiresApproval)
			}
			if !containsAllParts(strings.ToLower(suggestion.ExecutionGuardrail), "advisory", "approval") {
				t.Fatalf("policy_compliance: execution guardrail must mention advisory+approval, got %q", suggestion.ExecutionGuardrail)
			}

			assertNoHallucinatedEvidenceRefs(t, replay.draft, triage.EvidenceRefs, "triage")
			assertNoHallucinatedEvidenceRefs(t, replay.draft, suggestion.EvidenceRefs, "suggested_action")
			assertNoUnsafeLanguage(t, triage.ProbableCause, "triage_probable_cause")
			assertNoUnsafeLanguage(t, suggestion.Justification, "suggested_action_justification")
		})
	}
}

func assertNoHallucinatedEvidenceRefs(t *testing.T, draft *AgentDraftResponse, refs []AgentDraftEvidenceRef, scope string) {
	t.Helper()
	if draft == nil || len(refs) == 0 {
		return
	}
	allowed := map[string]struct{}{
		"findings.list":              {},
		"evidence.list":              {},
		"runtime_health.get":         {},
		"cluster_overview.get":       {},
		"release_drift.summary":      {},
		"http_signals.recent":        {},
		"http_signals.history":       {},
		"logs.recent":                {},
		"async_backlog.recent":       {},
		"messaging_transport.recent": {},
		"messaging_consumers.recent": {},
	}
	for _, run := range draft.ToolRuns {
		name := strings.TrimSpace(run.ToolName)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	for _, ref := range refs {
		toolName := strings.TrimSpace(ref.ToolName)
		if toolName == "" {
			continue
		}
		if _, ok := allowed[toolName]; !ok {
			t.Fatalf("hallucination_guard: %s contains non-grounded evidence ref tool %q", scope, toolName)
		}
	}
}

func assertNoUnsafeLanguage(t *testing.T, value string, scope string) {
	t.Helper()
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return
	}
	blocked := []string{
		"bypass approval",
		"without approval",
		"auto-execute",
		"automatically execute",
	}
	for _, phrase := range blocked {
		if strings.Contains(lower, phrase) {
			t.Fatalf("hallucination_guard: %s contains unsafe phrase %q", scope, phrase)
		}
	}
}

func stableIncidentID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func containsAllParts(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
