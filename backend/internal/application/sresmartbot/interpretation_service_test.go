package sresmartbot

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

func TestBuildBoundedSummaries_CacheHitByEvidenceHash(t *testing.T) {
	draft := interpretationDraftFixture()
	runtime := systemconfig.RobotSREAgentRuntimeConfig{
		Enabled:  true,
		Provider: "ollama",
		Model:    "llama3.2:3b",
		BaseURL:  "http://127.0.0.1:11434",
	}
	calls := 0
	svc := &InterpretationService{
		generate: func(ctx context.Context, baseURL string, model string, prompt string) (string, error) {
			calls++
			return `{"timeline_summary":"Timeline from local model","change_detection_15m":"Error rate rose in last 15m","operator_handoff_note":"Check HTTP and backlog, then hold approval gates.","likely_root_cause":"HTTP degradation","watchouts":["avoid direct remediation"]}`, nil
		},
		cache: make(map[string]cachedInterpretation),
	}

	first := svc.buildBoundedSummaries(context.Background(), draft, runtime)
	second := svc.buildBoundedSummaries(context.Background(), draft, runtime)

	if !first.Generated || first.CacheHit {
		t.Fatalf("expected first call to generate without cache hit, got generated=%v cache_hit=%v", first.Generated, first.CacheHit)
	}
	if !second.Generated || !second.CacheHit {
		t.Fatalf("expected second call to return cached generated output, got generated=%v cache_hit=%v", second.Generated, second.CacheHit)
	}
	if calls != 1 {
		t.Fatalf("expected one runtime generation call, got %d", calls)
	}
	if first.EvidenceHash == "" || second.EvidenceHash == "" {
		t.Fatal("expected evidence hash to be present")
	}
	if first.EvidenceHash != second.EvidenceHash {
		t.Fatalf("expected stable evidence hash, got %q vs %q", first.EvidenceHash, second.EvidenceHash)
	}
}

func TestBuildBoundedSummaries_CacheMissWhenEvidenceChanges(t *testing.T) {
	draft := interpretationDraftFixture()
	changed := interpretationDraftFixture()
	changed.Summary = draft.Summary + " Updated signal pressure."

	runtime := systemconfig.RobotSREAgentRuntimeConfig{
		Enabled:  true,
		Provider: "ollama",
		Model:    "llama3.2:3b",
		BaseURL:  "http://127.0.0.1:11434",
	}
	calls := 0
	svc := &InterpretationService{
		generate: func(ctx context.Context, baseURL string, model string, prompt string) (string, error) {
			calls++
			return `{"timeline_summary":"Timeline","change_detection_15m":"Change","operator_handoff_note":"Handoff","likely_root_cause":"Cause","watchouts":["watchout"]}`, nil
		},
		cache: make(map[string]cachedInterpretation),
	}

	first := svc.buildBoundedSummaries(context.Background(), draft, runtime)
	second := svc.buildBoundedSummaries(context.Background(), changed, runtime)

	if calls != 2 {
		t.Fatalf("expected two generation calls for evidence hash miss, got %d", calls)
	}
	if first.EvidenceHash == second.EvidenceHash {
		t.Fatalf("expected evidence hash to change when evidence changes, got %q", first.EvidenceHash)
	}
	if second.CacheHit {
		t.Fatal("expected second call to be cache miss")
	}
}

func TestBuildBoundedSummaries_FallbackWhenModelUnavailable(t *testing.T) {
	draft := interpretationDraftFixture()
	runtime := systemconfig.RobotSREAgentRuntimeConfig{
		Enabled:  true,
		Provider: "ollama",
		Model:    "llama3.2:3b",
		BaseURL:  "http://127.0.0.1:11434",
	}
	svc := &InterpretationService{
		generate: func(ctx context.Context, baseURL string, model string, prompt string) (string, error) {
			return "", errors.New("runtime unreachable")
		},
		cache: make(map[string]cachedInterpretation),
	}

	outcome := svc.buildBoundedSummaries(context.Background(), draft, runtime)

	if outcome.Generated {
		t.Fatal("expected fallback mode when runtime call fails")
	}
	if outcome.CacheHit {
		t.Fatal("expected no cache hit in fallback mode")
	}
	if outcome.FallbackReason == "" {
		t.Fatal("expected fallback reason to be populated")
	}
	if outcome.TimelineSummary == "" || outcome.ChangeDetection15m == "" || outcome.OperatorHandoffNote == "" {
		t.Fatalf("expected deterministic fallback summaries, got timeline=%q change=%q handoff=%q", outcome.TimelineSummary, outcome.ChangeDetection15m, outcome.OperatorHandoffNote)
	}
}

func interpretationDraftFixture() *AgentDraftResponse {
	incidentID := uuid.MustParse("0f7af5f3-5ca4-4294-b985-16ad507c54ad")
	return &AgentDraftResponse{
		IncidentID: incidentID,
		Mode:       "deterministic_draft",
		Summary:    "Request latency and errors are elevated alongside backlog pressure.",
		Hypotheses: []AgentDraftHypothesis{
			{
				Title:      "Service health is degrading through one or more golden signals",
				Confidence: "high",
				Rationale:  "HTTP and backlog signals are correlated in the current evidence.",
				SignalsUsed: []string{
					"http_signals.recent",
					"async_backlog.recent",
				},
			},
		},
		InvestigationPlan: []AgentDraftPlanStep{
			{
				Title:       "Validate app-level golden signals",
				Description: "Compare request/error/latency trends against async pressure.",
			},
			{
				Title:       "Escalate only after bounded review",
				Description: "Use approval-bound action flow if evidence still supports intervention.",
			},
		},
		ToolRuns: []MCPToolInvocationResult{
			{
				ToolName:   "http_signals.recent",
				ServerName: "Observability",
				Payload: map[string]any{
					"request_count":      int64(220),
					"error_rate_percent": int64(9),
					"average_latency_ms": int64(980),
				},
			},
			{
				ToolName:   "http_signals.history",
				ServerName: "Observability",
				Payload: map[string]any{
					"trend": "worsening",
				},
			},
			{
				ToolName:   "async_backlog.recent",
				ServerName: "Observability",
				Payload: map[string]any{
					"build_queue_depth":        int64(16),
					"email_queue_depth":        int64(7),
					"messaging_outbox_pending": int64(9),
				},
			},
		},
		HumanConfirmation: true,
	}
}
