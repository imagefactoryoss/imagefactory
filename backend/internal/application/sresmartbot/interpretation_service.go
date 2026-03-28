package sresmartbot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/llm"
)

type AgentInterpretationResponse struct {
	Draft                *AgentDraftResponse `json:"draft"`
	Provider             string              `json:"provider"`
	Model                string              `json:"model"`
	Generated            bool                `json:"generated"`
	CacheHit             bool                `json:"cache_hit"`
	EvidenceHash         string              `json:"evidence_hash,omitempty"`
	SummaryMode          string              `json:"summary_mode,omitempty"`
	TimelineSummary      string              `json:"timeline_summary,omitempty"`
	ChangeDetection15m   string              `json:"change_detection_15m,omitempty"`
	OperatorHandoffNote  string              `json:"operator_handoff_note,omitempty"`
	FallbackReason       string              `json:"fallback_reason,omitempty"`
	OperatorSummary      string              `json:"operator_summary,omitempty"`
	LikelyRootCause      string              `json:"likely_root_cause,omitempty"`
	Watchouts            []string            `json:"watchouts,omitempty"`
	Citations            []AgentCitation     `json:"citations,omitempty"`
	OperatorMessageDraft string              `json:"operator_message_draft,omitempty"`
	RawResponse          string              `json:"raw_response,omitempty"`
}

type interpretationJSON struct {
	TimelineSummary      string   `json:"timeline_summary"`
	ChangeDetection15m   string   `json:"change_detection_15m"`
	OperatorHandoffNote  string   `json:"operator_handoff_note"`
	OperatorSummary      string   `json:"operator_summary"`
	LikelyRootCause      string   `json:"likely_root_cause"`
	Watchouts            []string `json:"watchouts"`
	OperatorMessageDraft string   `json:"operator_message_draft"`
}

type cachedInterpretation struct {
	TimelineSummary     string
	ChangeDetection15m  string
	OperatorHandoffNote string
	LikelyRootCause     string
	Watchouts           []string
	RawResponse         string
}

type boundedSummaryOutcome struct {
	TimelineSummary     string
	ChangeDetection15m  string
	OperatorHandoffNote string
	LikelyRootCause     string
	Watchouts           []string
	Citations           []AgentCitation
	RawResponse         string
	Generated           bool
	CacheHit            bool
	EvidenceHash        string
	SummaryMode         string
	FallbackReason      string
}

type InterpretationService struct {
	agentService     *AgentService
	workspaceService *WorkspaceService
	generate         func(ctx context.Context, baseURL string, model string, prompt string) (string, error)
	runbookIndex     *runbookGroundingIndex
	mu               sync.RWMutex
	cache            map[string]cachedInterpretation
}

func NewInterpretationService(agentService *AgentService, workspaceService *WorkspaceService) *InterpretationService {
	return &InterpretationService{
		agentService:     agentService,
		workspaceService: workspaceService,
		generate: func(ctx context.Context, baseURL string, model string, prompt string) (string, error) {
			client := llm.NewOllamaClient(baseURL, nil)
			return client.Generate(ctx, model, prompt)
		},
		runbookIndex: newRunbookGroundingIndex(),
		cache:        make(map[string]cachedInterpretation),
	}
}

func (s *InterpretationService) BuildInterpretation(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentInterpretationResponse, error) {
	if s == nil || s.agentService == nil || s.workspaceService == nil {
		return nil, fmt.Errorf("interpretation service is not configured")
	}
	draft, err := s.agentService.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	workspace, err := s.workspaceService.BuildIncidentWorkspace(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}

	resp := &AgentInterpretationResponse{
		Draft:    draft,
		Provider: workspace.AgentRuntime.Provider,
		Model:    workspace.AgentRuntime.Model,
	}

	outcome := s.buildBoundedSummaries(ctx, draft, workspace.AgentRuntime)
	resp.Generated = outcome.Generated
	resp.CacheHit = outcome.CacheHit
	resp.EvidenceHash = outcome.EvidenceHash
	resp.SummaryMode = outcome.SummaryMode
	resp.TimelineSummary = outcome.TimelineSummary
	resp.ChangeDetection15m = outcome.ChangeDetection15m
	resp.OperatorHandoffNote = outcome.OperatorHandoffNote
	resp.FallbackReason = outcome.FallbackReason
	resp.LikelyRootCause = outcome.LikelyRootCause
	resp.Watchouts = outcome.Watchouts
	resp.Citations = outcome.Citations
	resp.RawResponse = outcome.RawResponse
	if err := validateCitationContract(resp.Citations); err != nil {
		return nil, err
	}

	// Backward compatibility for existing UI consumers.
	resp.OperatorSummary = outcome.TimelineSummary
	resp.OperatorMessageDraft = outcome.OperatorHandoffNote
	return resp, nil
}

func buildInterpretationPrompt(draft *AgentDraftResponse, runtime systemconfig.RobotSREAgentRuntimeConfig) string {
	payload, _ := json.MarshalIndent(draft, "", "  ")
	return strings.TrimSpace(fmt.Sprintf(`
You are SRE Smart Bot's local bounded summary layer.
You must stay grounded in the deterministic draft and never invent facts.
Return only valid JSON with this shape:
{
  "timeline_summary": "short grounded timeline summary",
  "change_detection_15m": "what changed in the last 15m from evidence",
  "operator_handoff_note": "brief handoff note with next checks and safe action",
  "likely_root_cause": "single concise statement",
  "watchouts": ["item 1", "item 2"]
}

System prompt ref: %s

Deterministic draft:
%s
`, runtime.SystemPromptRef, string(payload)))
}

func (s *InterpretationService) buildBoundedSummaries(ctx context.Context, draft *AgentDraftResponse, runtime systemconfig.RobotSREAgentRuntimeConfig) boundedSummaryOutcome {
	outcome := fallbackBoundedSummaries(draft)
	outcome.EvidenceHash = evidenceHashForDraft(draft)
	outcome.SummaryMode = "grounded_fallback"
	outcome.Citations = s.buildGroundedCitations(draft)

	if !runtimeEligibleForLocalModel(runtime) || strings.TrimSpace(outcome.EvidenceHash) == "" {
		outcome.FallbackReason = "local runtime unavailable; returned deterministic grounded fallback"
		return outcome
	}

	cacheKey := interpretationCacheKey(draft.IncidentID, outcome.EvidenceHash, runtime.Provider, runtime.Model)
	if cached, ok := s.getCachedInterpretation(cacheKey); ok {
		outcome.TimelineSummary = cached.TimelineSummary
		outcome.ChangeDetection15m = cached.ChangeDetection15m
		outcome.OperatorHandoffNote = cached.OperatorHandoffNote
		outcome.LikelyRootCause = cached.LikelyRootCause
		outcome.Watchouts = append([]string(nil), cached.Watchouts...)
		outcome.RawResponse = cached.RawResponse
		outcome.Generated = true
		outcome.CacheHit = true
		outcome.SummaryMode = "local_model_cached"
		return outcome
	}

	raw, err := s.generate(ctx, runtime.BaseURL, runtime.Model, buildInterpretationPrompt(draft, runtime))
	if err != nil {
		outcome.FallbackReason = fmt.Sprintf("local model unavailable (%s); returned deterministic grounded fallback", strings.TrimSpace(err.Error()))
		return outcome
	}

	parsed, ok := parseInterpretationJSON(raw)
	if !ok {
		outcome.TimelineSummary = strings.TrimSpace(raw)
		outcome.ChangeDetection15m = fallbackChangeDetection15m(draft)
		outcome.OperatorHandoffNote = fallbackOperatorHandoffNote(draft)
		outcome.LikelyRootCause = fallbackLikelyRootCause(draft)
		outcome.Watchouts = fallbackWatchouts(draft)
		outcome.RawResponse = raw
		outcome.Generated = true
		outcome.SummaryMode = "local_model_generated_raw"
		s.setCachedInterpretation(cacheKey, cachedInterpretation{
			TimelineSummary:     outcome.TimelineSummary,
			ChangeDetection15m:  outcome.ChangeDetection15m,
			OperatorHandoffNote: outcome.OperatorHandoffNote,
			LikelyRootCause:     outcome.LikelyRootCause,
			Watchouts:           append([]string(nil), outcome.Watchouts...),
			RawResponse:         outcome.RawResponse,
		})
		return outcome
	}

	outcome.TimelineSummary = parsed.TimelineSummary
	outcome.ChangeDetection15m = parsed.ChangeDetection15m
	outcome.OperatorHandoffNote = parsed.OperatorHandoffNote
	outcome.LikelyRootCause = parsed.LikelyRootCause
	outcome.Watchouts = append([]string(nil), parsed.Watchouts...)
	outcome.RawResponse = raw
	outcome.Generated = true
	outcome.SummaryMode = "local_model_generated"

	s.setCachedInterpretation(cacheKey, cachedInterpretation{
		TimelineSummary:     outcome.TimelineSummary,
		ChangeDetection15m:  outcome.ChangeDetection15m,
		OperatorHandoffNote: outcome.OperatorHandoffNote,
		LikelyRootCause:     outcome.LikelyRootCause,
		Watchouts:           append([]string(nil), outcome.Watchouts...),
		RawResponse:         outcome.RawResponse,
	})
	return outcome
}

func parseInterpretationJSON(raw string) (cachedInterpretation, bool) {
	var parsed interpretationJSON
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return cachedInterpretation{}, false
	}

	timeline := strings.TrimSpace(parsed.TimelineSummary)
	if timeline == "" {
		timeline = strings.TrimSpace(parsed.OperatorSummary)
	}
	handoff := strings.TrimSpace(parsed.OperatorHandoffNote)
	if handoff == "" {
		handoff = strings.TrimSpace(parsed.OperatorMessageDraft)
	}
	change := strings.TrimSpace(parsed.ChangeDetection15m)
	if change == "" && len(parsed.Watchouts) > 0 {
		change = strings.TrimSpace(parsed.Watchouts[0])
	}
	likelyRootCause := strings.TrimSpace(parsed.LikelyRootCause)
	watchouts := make([]string, 0, len(parsed.Watchouts))
	for _, watchout := range parsed.Watchouts {
		value := strings.TrimSpace(watchout)
		if value == "" {
			continue
		}
		watchouts = append(watchouts, value)
	}
	if timeline == "" || handoff == "" || change == "" {
		return cachedInterpretation{}, false
	}
	return cachedInterpretation{
		TimelineSummary:     timeline,
		ChangeDetection15m:  change,
		OperatorHandoffNote: handoff,
		LikelyRootCause:     likelyRootCause,
		Watchouts:           watchouts,
		RawResponse:         strings.TrimSpace(raw),
	}, true
}

func runtimeEligibleForLocalModel(runtime systemconfig.RobotSREAgentRuntimeConfig) bool {
	provider := strings.ToLower(strings.TrimSpace(runtime.Provider))
	if !runtime.Enabled || provider == "" || provider == "none" || provider == "custom" {
		return false
	}
	if provider != "ollama" {
		return false
	}
	return strings.TrimSpace(runtime.Model) != "" && strings.TrimSpace(runtime.BaseURL) != ""
}

func interpretationCacheKey(incidentID uuid.UUID, evidenceHash string, provider string, model string) string {
	return fmt.Sprintf("%s|%s|%s|%s", incidentID.String(), strings.TrimSpace(evidenceHash), strings.ToLower(strings.TrimSpace(provider)), strings.TrimSpace(model))
}

func (s *InterpretationService) getCachedInterpretation(key string) (cachedInterpretation, bool) {
	if s == nil {
		return cachedInterpretation{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return cachedInterpretation{}, false
	}
	value, ok := s.cache[key]
	return value, ok
}

func (s *InterpretationService) setCachedInterpretation(key string, value cachedInterpretation) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache == nil {
		s.cache = make(map[string]cachedInterpretation)
	}
	s.cache[key] = value
}

func evidenceHashForDraft(draft *AgentDraftResponse) string {
	if draft == nil {
		return ""
	}
	type canonicalToolRun struct {
		ServerID   string         `json:"server_id"`
		ServerName string         `json:"server_name"`
		ServerKind string         `json:"server_kind"`
		ToolName   string         `json:"tool_name"`
		Payload    map[string]any `json:"payload"`
	}
	type canonicalDraft struct {
		IncidentID        uuid.UUID              `json:"incident_id"`
		Mode              string                 `json:"mode"`
		Summary           string                 `json:"summary"`
		Hypotheses        []AgentDraftHypothesis `json:"hypotheses"`
		InvestigationPlan []AgentDraftPlanStep   `json:"investigation_plan"`
		ToolRuns          []canonicalToolRun     `json:"tool_runs"`
		HumanConfirmation bool                   `json:"human_confirmation_required"`
	}
	canonical := canonicalDraft{
		IncidentID:        draft.IncidentID,
		Mode:              draft.Mode,
		Summary:           draft.Summary,
		Hypotheses:        draft.Hypotheses,
		InvestigationPlan: draft.InvestigationPlan,
		HumanConfirmation: draft.HumanConfirmation,
	}
	canonical.ToolRuns = make([]canonicalToolRun, 0, len(draft.ToolRuns))
	for _, run := range draft.ToolRuns {
		canonical.ToolRuns = append(canonical.ToolRuns, canonicalToolRun{
			ServerID:   run.ServerID,
			ServerName: run.ServerName,
			ServerKind: run.ServerKind,
			ToolName:   run.ToolName,
			Payload:    run.Payload,
		})
	}
	blob, err := json.Marshal(canonical)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

func fallbackBoundedSummaries(draft *AgentDraftResponse) boundedSummaryOutcome {
	return boundedSummaryOutcome{
		TimelineSummary:     fallbackTimelineSummary(draft),
		ChangeDetection15m:  fallbackChangeDetection15m(draft),
		OperatorHandoffNote: fallbackOperatorHandoffNote(draft),
		LikelyRootCause:     fallbackLikelyRootCause(draft),
		Watchouts:           fallbackWatchouts(draft),
		Generated:           false,
	}
}

func (s *InterpretationService) buildGroundedCitations(draft *AgentDraftResponse) []AgentCitation {
	runbookCitations := make([]AgentCitation, 0, 2)
	if s != nil && s.runbookIndex != nil {
		runbookCitations = append(runbookCitations, s.runbookIndex.FindRelevantCitations(draft, 2)...)
	} else {
		runbookCitations = append(runbookCitations, fallbackRunbookCitation())
	}
	evidenceCitations := buildEvidenceCitations(draft, 2)

	citations := make([]AgentCitation, 0, len(runbookCitations)+len(evidenceCitations))
	citations = append(citations, runbookCitations...)
	citations = append(citations, evidenceCitations...)
	return citations
}

func fallbackTimelineSummary(draft *AgentDraftResponse) string {
	if draft == nil {
		return "No incident draft was available, so the timeline summary is pending bounded evidence retrieval."
	}
	triage := buildTriageFromDraft(draft)
	severity := buildSeverityFromDraft(draft)
	summary := strings.TrimSpace(draft.Summary)
	if summary == "" {
		summary = "Deterministic draft is available but did not include a summary line."
	}
	if triage != nil && severity != nil {
		return fmt.Sprintf("%s Probable cause currently points to %s with %s confidence; correlated severity score is %d (%s).", summary, strings.TrimSpace(triage.ProbableCause), strings.TrimSpace(triage.Confidence), severity.Score, severity.Level)
	}
	return summary
}

func fallbackChangeDetection15m(draft *AgentDraftResponse) string {
	if draft == nil {
		return "No 15-minute change evidence is currently available."
	}
	var recentErrRate int64
	var recentLatency int64
	var historyTrend string
	var backlogTotal int64
	var reconnects int64
	var disconnects int64

	for _, run := range draft.ToolRuns {
		switch strings.TrimSpace(run.ToolName) {
		case "http_signals.recent":
			recentErrRate = metricFromPayload(run.Payload, "error_rate_percent")
			recentLatency = metricFromPayload(run.Payload, "average_latency_ms")
		case "http_signals.history":
			if trend, ok := run.Payload["trend"].(string); ok {
				historyTrend = strings.TrimSpace(trend)
			}
		case "async_backlog.recent":
			backlogTotal = metricFromPayload(run.Payload, "build_queue_depth") + metricFromPayload(run.Payload, "email_queue_depth") + metricFromPayload(run.Payload, "messaging_outbox_pending")
		case "messaging_transport.recent":
			reconnects = metricFromPayload(run.Payload, "reconnects")
			disconnects = metricFromPayload(run.Payload, "disconnects")
		}
	}

	parts := make([]string, 0, 4)
	if historyTrend != "" {
		parts = append(parts, fmt.Sprintf("HTTP trend is %s", historyTrend))
	}
	if recentErrRate > 0 || recentLatency > 0 {
		parts = append(parts, fmt.Sprintf("latest HTTP window shows %d%% server-error rate and %dms average latency", recentErrRate, recentLatency))
	}
	if backlogTotal > 0 {
		parts = append(parts, fmt.Sprintf("async backlog snapshot totals %d pending items", backlogTotal))
	}
	if reconnects > 0 || disconnects > 0 {
		parts = append(parts, fmt.Sprintf("messaging transport shows reconnects=%d and disconnects=%d", reconnects, disconnects))
	}
	if len(parts) == 0 {
		return "No significant 15-minute drift was detected from bounded evidence snapshots."
	}
	return strings.Join(parts, "; ") + "."
}

func fallbackOperatorHandoffNote(draft *AgentDraftResponse) string {
	if draft == nil {
		return "Handoff: bounded evidence is not loaded yet; start by generating deterministic draft and severity/triage summaries."
	}
	triage := buildTriageFromDraft(draft)
	if triage == nil {
		return "Handoff: deterministic draft is present but triage output was unavailable; re-run triage and validate top evidence refs."
	}
	nextChecks := triage.NextChecks
	if len(nextChecks) > 3 {
		nextChecks = nextChecks[:3]
	}
	if len(nextChecks) == 0 {
		nextChecks = []string{
			"Review incident summary against findings/evidence ledger.",
			"Run bounded read-only MCP tools for current scope.",
			"Escalate only via approval-gated action flow.",
		}
	}
	return fmt.Sprintf("Handoff: probable cause %s (%s confidence). Next checks: 1) %s 2) %s 3) %s Safe action: %s",
		triage.ProbableCause,
		triage.Confidence,
		nextChecks[0],
		nextChecks[minIndex(1, len(nextChecks)-1)],
		nextChecks[minIndex(2, len(nextChecks)-1)],
		triage.RecommendedAction,
	)
}

func fallbackLikelyRootCause(draft *AgentDraftResponse) string {
	if draft == nil {
		return ""
	}
	triage := buildTriageFromDraft(draft)
	if triage == nil {
		return ""
	}
	return strings.TrimSpace(triage.ProbableCause)
}

func fallbackWatchouts(draft *AgentDraftResponse) []string {
	watchouts := []string{
		"Keep actions advisory-only until deterministic approval gates are satisfied.",
	}
	change := fallbackChangeDetection15m(draft)
	if change != "" {
		watchouts = append(watchouts, change)
	}
	return watchouts
}

func minIndex(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
