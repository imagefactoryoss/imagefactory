package sresmartbot

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type AgentDraftHypothesis struct {
	Title        string                  `json:"title"`
	Confidence   string                  `json:"confidence"`
	Rationale    string                  `json:"rationale"`
	SignalsUsed  []string                `json:"signals_used"`
	EvidenceRefs []AgentDraftEvidenceRef `json:"evidence_refs,omitempty"`
}

type AgentDraftPlanStep struct {
	Title        string                  `json:"title"`
	Description  string                  `json:"description"`
	EvidenceRefs []AgentDraftEvidenceRef `json:"evidence_refs,omitempty"`
}

type AgentDraftEvidenceRef struct {
	ToolName   string `json:"tool_name"`
	ServerName string `json:"server_name"`
	Summary    string `json:"summary"`
}

type AgentDraftResponse struct {
	IncidentID        uuid.UUID                 `json:"incident_id"`
	Mode              string                    `json:"mode"`
	Summary           string                    `json:"summary"`
	Hypotheses        []AgentDraftHypothesis    `json:"hypotheses"`
	InvestigationPlan []AgentDraftPlanStep      `json:"investigation_plan"`
	ToolRuns          []MCPToolInvocationResult `json:"tool_runs"`
	HumanConfirmation bool                      `json:"human_confirmation_required"`
}

type AgentTriageResponse struct {
	IncidentID        uuid.UUID               `json:"incident_id"`
	Mode              string                  `json:"mode"`
	Summary           string                  `json:"summary"`
	ProbableCause     string                  `json:"probable_cause"`
	Confidence        string                  `json:"confidence"`
	NextChecks        []string                `json:"next_checks"`
	NextCheckRefs     []AgentTriageCheckRef   `json:"next_check_refs,omitempty"`
	RecommendedAction string                  `json:"recommended_action"`
	EvidenceRefs      []AgentDraftEvidenceRef `json:"evidence_refs,omitempty"`
	HumanConfirmation bool                    `json:"human_confirmation_required"`
}

type AgentTriageCheckRef struct {
	Check           string   `json:"check"`
	RunbookSource   string   `json:"runbook_source"`
	RunbookSection  string   `json:"runbook_section"`
	EvidenceSignals []string `json:"evidence_signals,omitempty"`
	EvidenceNote    string   `json:"evidence_note"`
}

type AgentSeverityFactor struct {
	Key               string `json:"key"`
	Label             string `json:"label"`
	Contribution      int64  `json:"contribution"`
	WeightPercent     int64  `json:"weight_percent"`
	Reason            string `json:"reason"`
	OperatorRationale string `json:"operator_rationale"`
}

type AgentSeverityResponse struct {
	IncidentID        uuid.UUID             `json:"incident_id"`
	Mode              string                `json:"mode"`
	Score             int64                 `json:"score"`
	Level             string                `json:"level"`
	Summary           string                `json:"summary"`
	Factors           []AgentSeverityFactor `json:"factors"`
	HumanConfirmation bool                  `json:"human_confirmation_required"`
}

type AgentSuggestedActionResponse struct {
	IncidentID                uuid.UUID               `json:"incident_id"`
	Mode                      string                  `json:"mode"`
	ActionKey                 string                  `json:"action_key"`
	ActionSummary             string                  `json:"action_summary"`
	Justification             string                  `json:"justification"`
	BlastRadius               string                  `json:"blast_radius"`
	AdvisoryOnly              bool                    `json:"advisory_only"`
	ExecutionRequiresApproval bool                    `json:"execution_requires_approval"`
	ExecutionGuardrail        string                  `json:"execution_guardrail"`
	EvidenceRefs              []AgentDraftEvidenceRef `json:"evidence_refs,omitempty"`
	HumanConfirmation         bool                    `json:"human_confirmation_required"`
}

type AgentIncidentScorecardResponse struct {
	IncidentID                uuid.UUID             `json:"incident_id"`
	Mode                      string                `json:"mode"`
	Summary                   string                `json:"summary"`
	ProbableCause             string                `json:"probable_cause"`
	Confidence                string                `json:"confidence"`
	SeverityScore             int64                 `json:"severity_score"`
	SeverityLevel             string                `json:"severity_level"`
	WhySevereCards            []AgentSeverityFactor `json:"why_severe_cards,omitempty"`
	RecommendedAction         string                `json:"recommended_action"`
	ActionKey                 string                `json:"action_key"`
	BlastRadius               string                `json:"blast_radius"`
	ExecutionRequiresApproval bool                  `json:"execution_requires_approval"`
	HumanConfirmation         bool                  `json:"human_confirmation_required"`
}

type AgentIncidentSnapshotResponse struct {
	IncidentID              uuid.UUID                       `json:"incident_id"`
	Mode                    string                          `json:"mode"`
	Summary                 string                          `json:"summary"`
	Triage                  *AgentTriageResponse            `json:"triage,omitempty"`
	Severity                *AgentSeverityResponse          `json:"severity,omitempty"`
	Scorecard               *AgentIncidentScorecardResponse `json:"scorecard,omitempty"`
	SuggestedAction         *AgentSuggestedActionResponse   `json:"suggested_action,omitempty"`
	OperatorHandoff         string                          `json:"operator_handoff_note"`
	PolicyGuardrails        []string                        `json:"policy_guardrails,omitempty"`
	EvidenceSignalsExpected []string                        `json:"evidence_signals_expected,omitempty"`
	EvidenceSignalsObserved []string                        `json:"evidence_signals_observed,omitempty"`
	EvidenceCoveragePercent int64                           `json:"evidence_coverage_percent"`
	EvidenceHealthNote      string                          `json:"evidence_health_note"`
	HumanConfirmation       bool                            `json:"human_confirmation_required"`
}

type AgentService struct {
	workspaceService *WorkspaceService
	mcpService       *MCPService
}

func NewAgentService(workspaceService *WorkspaceService, mcpService *MCPService) *AgentService {
	return &AgentService{
		workspaceService: workspaceService,
		mcpService:       mcpService,
	}
}

func (s *AgentService) BuildDraft(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentDraftResponse, error) {
	if s == nil || s.workspaceService == nil {
		return nil, fmt.Errorf("agent service is not configured")
	}

	workspace, err := s.workspaceService.BuildIncidentWorkspace(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}

	toolRuns := make([]MCPToolInvocationResult, 0)
	if s.mcpService != nil {
		tools, toolErr := s.mcpService.ListAvailableTools(ctx, tenantID, incidentID)
		if toolErr == nil {
			selected := selectDraftTools(workspace, tools)
			maxRuns := workspace.AgentRuntime.MaxToolCallsPerTurn
			if maxRuns <= 0 {
				maxRuns = 4
			}
			if len(selected) > maxRuns {
				selected = selected[:maxRuns]
			}
			for _, tool := range selected {
				result, invokeErr := s.mcpService.InvokeTool(ctx, tenantID, MCPToolInvocationRequest{
					IncidentID: incidentID,
					ServerID:   tool.ServerID,
					ToolName:   tool.ToolName,
				})
				if invokeErr == nil && result != nil {
					toolRuns = append(toolRuns, *result)
				}
			}
		}
	}

	hypotheses := buildDraftHypotheses(workspace, toolRuns)
	plan := buildDraftPlan(workspace, hypotheses, toolRuns)
	summary := ""
	if len(workspace.ExecutiveSummary) > 0 {
		summary = workspace.ExecutiveSummary[0]
	}
	if summary == "" && workspace.Incident != nil {
		summary = workspace.Incident.DisplayName
	}

	return &AgentDraftResponse{
		IncidentID:        incidentID,
		Mode:              "deterministic_draft",
		Summary:           summary,
		Hypotheses:        hypotheses,
		InvestigationPlan: plan,
		ToolRuns:          toolRuns,
		HumanConfirmation: workspace.AgentRuntime.RequireHumanConfirmationForMessage,
	}, nil
}

func (s *AgentService) BuildTriage(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentTriageResponse, error) {
	draft, err := s.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	return buildTriageFromDraft(draft), nil
}

func (s *AgentService) BuildSeverity(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentSeverityResponse, error) {
	draft, err := s.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	return buildSeverityFromDraft(draft), nil
}

func (s *AgentService) BuildSuggestedAction(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentSuggestedActionResponse, error) {
	draft, err := s.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	return buildSuggestedActionFromDraft(draft), nil
}

func (s *AgentService) BuildIncidentScorecard(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentIncidentScorecardResponse, error) {
	draft, err := s.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	return buildIncidentScorecardFromDraft(draft), nil
}

func (s *AgentService) BuildIncidentSnapshot(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentIncidentSnapshotResponse, error) {
	draft, err := s.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	return buildIncidentSnapshotFromDraft(draft), nil
}

func buildTriageFromDraft(draft *AgentDraftResponse) *AgentTriageResponse {
	if draft == nil {
		return nil
	}
	probableCause := strings.TrimSpace(draft.Summary)
	confidence := "medium"
	evidenceRefs := make([]AgentDraftEvidenceRef, 0)
	if len(draft.Hypotheses) > 0 {
		top := draft.Hypotheses[0]
		if title := strings.TrimSpace(top.Title); title != "" {
			probableCause = title
		}
		if value := normalizeTriageConfidence(top.Confidence); value != "" {
			confidence = value
		}
		if len(top.EvidenceRefs) > 0 {
			evidenceRefs = append(evidenceRefs, top.EvidenceRefs...)
		}
	}
	if probableCause == "" {
		probableCause = "Most recent stored findings and evidence require additional confirmation."
	}
	if len(evidenceRefs) == 0 {
		evidenceRefs = evidenceRefsForSignals([]string{"findings.list", "evidence.list"}, draft.ToolRuns)
	}

	nextChecks := buildTriageNextChecks(draft)
	return &AgentTriageResponse{
		IncidentID:        draft.IncidentID,
		Mode:              "deterministic_triage",
		Summary:           draft.Summary,
		ProbableCause:     probableCause,
		Confidence:        confidence,
		NextChecks:        nextChecks,
		NextCheckRefs:     buildTriageCheckRefs(draft, nextChecks),
		RecommendedAction: deriveTriageRecommendedAction(draft),
		EvidenceRefs:      evidenceRefs,
		HumanConfirmation: draft.HumanConfirmation,
	}
}

func buildSeverityFromDraft(draft *AgentDraftResponse) *AgentSeverityResponse {
	if draft == nil {
		return nil
	}
	score := int64(25)
	factors := make([]AgentSeverityFactor, 0, 6)
	addFactor := func(key, label string, contribution int64, reason string) {
		if contribution <= 0 {
			return
		}
		factors = append(factors, AgentSeverityFactor{
			Key:          key,
			Label:        label,
			Contribution: contribution,
			Reason:       reason,
		})
		score += contribution
	}

	if len(draft.Hypotheses) > 0 {
		top := draft.Hypotheses[0]
		switch strings.ToLower(strings.TrimSpace(top.Confidence)) {
		case "high":
			addFactor("top_hypothesis_confidence", "Top Hypothesis Confidence", 15, "The strongest grounded hypothesis is high confidence.")
		case "medium":
			addFactor("top_hypothesis_confidence", "Top Hypothesis Confidence", 8, "The strongest grounded hypothesis is medium confidence.")
		}
	}

	for _, run := range draft.ToolRuns {
		switch run.ToolName {
		case "logs.recent":
			matches := metricFromPayload(run.Payload, "match_count")
			if matches > 0 {
				contribution := matches * 2
				if contribution > 20 {
					contribution = 20
				}
				addFactor("logs_recent", "Log Pressure", contribution, fmt.Sprintf("Recent logs returned %d matching error signatures.", matches))
			}
		case "http_signals.recent":
			errRate := metricFromPayload(run.Payload, "error_rate_percent")
			latency := metricFromPayload(run.Payload, "average_latency_ms")
			contribution := int64(0)
			if errRate >= 5 {
				contribution += 12
			} else if errRate > 0 {
				contribution += 6
			}
			if latency >= 1000 {
				contribution += 8
			} else if latency >= 400 {
				contribution += 4
			}
			addFactor("http_signals_recent", "HTTP Signal Degradation", contribution, fmt.Sprintf("Recent HTTP window shows %d%% server-error rate and %dms average latency.", errRate, latency))
		case "async_backlog.recent":
			buildQueue := metricFromPayload(run.Payload, "build_queue_depth")
			emailQueue := metricFromPayload(run.Payload, "email_queue_depth")
			outboxPending := metricFromPayload(run.Payload, "messaging_outbox_pending")
			total := buildQueue + emailQueue + outboxPending
			contribution := int64(0)
			if total >= 50 {
				contribution = 12
			} else if total >= 15 {
				contribution = 7
			} else if total > 0 {
				contribution = 3
			}
			addFactor("async_backlog_recent", "Async Backlog Pressure", contribution, fmt.Sprintf("Backlog snapshot totals build=%d, email=%d, outbox=%d.", buildQueue, emailQueue, outboxPending))
		case "messaging_transport.recent":
			reconnects := metricFromPayload(run.Payload, "reconnects")
			disconnects := metricFromPayload(run.Payload, "disconnects")
			threshold := metricFromPayload(run.Payload, "reconnect_threshold")
			contribution := int64(0)
			if disconnects > 0 {
				contribution += 10
			}
			if threshold > 0 && reconnects >= threshold {
				contribution += 8
			} else if reconnects > 0 {
				contribution += 4
			}
			addFactor("messaging_transport_recent", "Messaging Transport Instability", contribution, fmt.Sprintf("Transport shows reconnects=%d, disconnects=%d, threshold=%d.", reconnects, disconnects, threshold))
		}
	}

	if score > 100 {
		score = 100
	}
	level := "low"
	switch {
	case score >= 80:
		level = "critical"
	case score >= 60:
		level = "high"
	case score >= 35:
		level = "medium"
	}

	sort.SliceStable(factors, func(i, j int) bool {
		return factors[i].Contribution > factors[j].Contribution
	})
	if len(factors) > 4 {
		factors = factors[:4]
	}
	applySeverityFactorWeights(factors, score, level)

	summary := fmt.Sprintf("Severity score is %d (%s) based on correlated logs, HTTP signals, async backlog, and transport evidence.", score, level)
	return &AgentSeverityResponse{
		IncidentID:        draft.IncidentID,
		Mode:              "deterministic_severity_correlation",
		Score:             score,
		Level:             level,
		Summary:           summary,
		Factors:           factors,
		HumanConfirmation: draft.HumanConfirmation,
	}
}

func applySeverityFactorWeights(factors []AgentSeverityFactor, score int64, level string) {
	if len(factors) == 0 {
		return
	}
	base := score
	if base <= 0 {
		base = 1
	}
	for i := range factors {
		weight := int64((factors[i].Contribution * 100) / base)
		if weight > 100 {
			weight = 100
		}
		factors[i].WeightPercent = weight
		factors[i].OperatorRationale = buildOperatorRationaleForSeverityFactor(factors[i], level)
	}
}

func buildOperatorRationaleForSeverityFactor(factor AgentSeverityFactor, level string) string {
	levelText := strings.ToLower(strings.TrimSpace(level))
	if levelText == "" {
		levelText = "active"
	}
	return fmt.Sprintf(
		"%s contributes %d%% of the current severity score; verify this signal first while the incident remains %s.",
		factor.Label,
		factor.WeightPercent,
		levelText,
	)
}

func buildSuggestedActionFromDraft(draft *AgentDraftResponse) *AgentSuggestedActionResponse {
	if draft == nil {
		return nil
	}
	triage := buildTriageFromDraft(draft)
	actionKey := "review_runtime_health"
	actionSummary := "Review runtime and cluster health to confirm if escalation is needed."
	blastRadius := "low"
	justification := "Current grounded evidence supports additional read-only verification before any disruptive intervention."
	evidenceRefs := evidenceRefsForSignals([]string{"runtime_health.get", "cluster_overview.get", "findings.list", "evidence.list"}, draft.ToolRuns)

	signals := []string{}
	if len(draft.Hypotheses) > 0 {
		signals = draft.Hypotheses[0].SignalsUsed
		if len(draft.Hypotheses[0].EvidenceRefs) > 0 {
			evidenceRefs = draft.Hypotheses[0].EvidenceRefs
		}
	}
	recommended := ""
	if triage != nil {
		recommended = strings.TrimSpace(triage.RecommendedAction)
	}

	switch {
	case containsSignal(signals, "messaging_transport.recent"):
		actionKey = "review_messaging_transport_health"
		actionSummary = "Inspect transport reconnect/disconnect pressure and confirm bus stability."
		blastRadius = "low"
		justification = "Transport instability is present; runbooks recommend validating bus stability before throughput or restart actions."
		evidenceRefs = evidenceRefsForSignals([]string{"messaging_transport.recent", "async_backlog.recent"}, draft.ToolRuns)
	case containsSignal(signals, "async_backlog.recent"):
		actionKey = "review_async_worker_capacity"
		actionSummary = "Validate async queue pressure and worker throughput limits."
		blastRadius = "medium"
		justification = "Backlog pressure is elevated and should be verified before wider incident response actions."
		evidenceRefs = evidenceRefsForSignals([]string{"async_backlog.recent", "http_signals.history", "logs.recent"}, draft.ToolRuns)
	case containsSignal(signals, "release_drift.summary"):
		actionKey = "review_release_drift"
		actionSummary = "Assess rollout drift and configuration mismatch prior to remediation."
		blastRadius = "high"
		justification = "Release drift can impact multiple components; approval-gated remediation is required before execution."
		evidenceRefs = evidenceRefsForSignals([]string{"release_drift.summary", "runtime_health.get", "findings.list"}, draft.ToolRuns)
	case containsSignal(signals, "cluster_overview.get"), containsSignal(signals, "runtime_health.get"):
		actionKey = "review_provider_connectivity"
		actionSummary = "Check infrastructure/provider health pathways for degradation."
		blastRadius = "medium"
		justification = "Infrastructure health may be driving symptoms; confirm dependencies first through bounded checks."
		evidenceRefs = evidenceRefsForSignals([]string{"cluster_overview.get", "runtime_health.get", "findings.list"}, draft.ToolRuns)
	}

	if recommended != "" {
		justification = fmt.Sprintf("%s Recommended triage path: %s", justification, recommended)
	}
	if len(evidenceRefs) == 0 {
		evidenceRefs = evidenceRefsForSignals([]string{"findings.list", "evidence.list"}, draft.ToolRuns)
	}

	return &AgentSuggestedActionResponse{
		IncidentID:                draft.IncidentID,
		Mode:                      "deterministic_advisory_suggested_action",
		ActionKey:                 actionKey,
		ActionSummary:             actionSummary,
		Justification:             justification,
		BlastRadius:               blastRadius,
		AdvisoryOnly:              true,
		ExecutionRequiresApproval: true,
		ExecutionGuardrail:        "Suggestion is advisory-only. Execution must use the existing action + approval workflow and cannot bypass deterministic policy gates.",
		EvidenceRefs:              evidenceRefs,
		HumanConfirmation:         draft.HumanConfirmation,
	}
}

func buildIncidentScorecardFromDraft(draft *AgentDraftResponse) *AgentIncidentScorecardResponse {
	if draft == nil {
		return nil
	}
	triage := buildTriageFromDraft(draft)
	severity := buildSeverityFromDraft(draft)
	suggestion := buildSuggestedActionFromDraft(draft)
	if triage == nil || severity == nil || suggestion == nil {
		return nil
	}

	whySevereCards := append([]AgentSeverityFactor(nil), severity.Factors...)
	if len(whySevereCards) > 3 {
		whySevereCards = whySevereCards[:3]
	}

	return &AgentIncidentScorecardResponse{
		IncidentID:                draft.IncidentID,
		Mode:                      "deterministic_incident_scorecard",
		Summary:                   severity.Summary,
		ProbableCause:             triage.ProbableCause,
		Confidence:                triage.Confidence,
		SeverityScore:             severity.Score,
		SeverityLevel:             severity.Level,
		WhySevereCards:            whySevereCards,
		RecommendedAction:         triage.RecommendedAction,
		ActionKey:                 suggestion.ActionKey,
		BlastRadius:               suggestion.BlastRadius,
		ExecutionRequiresApproval: suggestion.ExecutionRequiresApproval,
		HumanConfirmation:         draft.HumanConfirmation,
	}
}

func buildIncidentSnapshotFromDraft(draft *AgentDraftResponse) *AgentIncidentSnapshotResponse {
	if draft == nil {
		return nil
	}
	triage := buildTriageFromDraft(draft)
	severity := buildSeverityFromDraft(draft)
	scorecard := buildIncidentScorecardFromDraft(draft)
	suggested := buildSuggestedActionFromDraft(draft)
	if triage == nil || severity == nil || scorecard == nil || suggested == nil {
		return nil
	}
	operatorHandoff := buildSnapshotOperatorHandoff(triage, scorecard, suggested)
	expectedSignals, observedSignals, evidenceCoveragePercent, evidenceHealthNote := buildSnapshotEvidenceHealth(draft)
	policyGuardrails := []string{
		"AI output is advisory-only and cannot execute actions directly.",
		"Execution must go through deterministic action + approval workflow.",
		"Approval gate remains mandatory before any disruptive remediation.",
	}
	if draft.HumanConfirmation {
		policyGuardrails = append(policyGuardrails, "Human confirmation is required before operator-facing messages are sent.")
	}
	return &AgentIncidentSnapshotResponse{
		IncidentID:              draft.IncidentID,
		Mode:                    "deterministic_incident_snapshot",
		Summary:                 severity.Summary,
		Triage:                  triage,
		Severity:                severity,
		Scorecard:               scorecard,
		SuggestedAction:         suggested,
		OperatorHandoff:         operatorHandoff,
		PolicyGuardrails:        policyGuardrails,
		EvidenceSignalsExpected: expectedSignals,
		EvidenceSignalsObserved: observedSignals,
		EvidenceCoveragePercent: evidenceCoveragePercent,
		EvidenceHealthNote:      evidenceHealthNote,
		HumanConfirmation:       draft.HumanConfirmation,
	}
}

func buildSnapshotOperatorHandoff(
	triage *AgentTriageResponse,
	scorecard *AgentIncidentScorecardResponse,
	suggested *AgentSuggestedActionResponse,
) string {
	if triage == nil || scorecard == nil || suggested == nil {
		return ""
	}
	checks := triage.NextChecks
	if len(checks) > 2 {
		checks = checks[:2]
	}
	return fmt.Sprintf(
		"Probable cause: %s (confidence: %s). Severity: %d (%s). Next checks: %s. Advisory action: %s (%s blast radius). Execution remains approval-bound.",
		strings.TrimSpace(triage.ProbableCause),
		strings.TrimSpace(triage.Confidence),
		scorecard.SeverityScore,
		strings.TrimSpace(scorecard.SeverityLevel),
		strings.Join(checks, " | "),
		strings.TrimSpace(suggested.ActionKey),
		strings.TrimSpace(suggested.BlastRadius),
	)
}

func buildSnapshotEvidenceHealth(draft *AgentDraftResponse) ([]string, []string, int64, string) {
	expected := []string{
		"findings.list",
		"evidence.list",
		"http_signals.recent",
		"async_backlog.recent",
		"messaging_transport.recent",
		"logs.recent",
	}
	if draft == nil {
		return expected, nil, 0, "No deterministic draft evidence available yet."
	}
	seen := make(map[string]struct{}, len(draft.ToolRuns))
	for _, run := range draft.ToolRuns {
		name := strings.TrimSpace(run.ToolName)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	observed := make([]string, 0, len(expected))
	for _, signal := range expected {
		if _, ok := seen[signal]; ok {
			observed = append(observed, signal)
		}
	}
	coverage := int64(0)
	if len(expected) > 0 {
		coverage = int64((len(observed) * 100) / len(expected))
	}
	healthNote := "Evidence coverage is partial; use bounded checks before escalation."
	switch {
	case coverage >= 80:
		healthNote = "Evidence coverage is strong for deterministic operator guidance."
	case coverage >= 50:
		healthNote = "Evidence coverage is moderate; confirm missing signals if risk increases."
	}
	return expected, observed, coverage, healthNote
}

func selectDraftTools(workspace *IncidentWorkspace, tools []MCPToolDescriptor) []MCPToolDescriptor {
	if workspace == nil || len(tools) == 0 {
		return nil
	}
	preferred := []string{"incidents.get", "findings.list", "evidence.list", "runtime_health.get", "http_signals.recent", "http_signals.history", "async_backlog.recent", "messaging_transport.recent", "cluster_overview.get", "release_drift.summary", "logs.recent"}
	index := map[string]MCPToolDescriptor{}
	for _, tool := range tools {
		index[tool.ToolName] = tool
	}
	selected := make([]MCPToolDescriptor, 0, len(preferred))
	for _, name := range preferred {
		if tool, ok := index[name]; ok {
			if name == "logs.recent" && workspace.Incident == nil {
				continue
			}
			selected = append(selected, tool)
		}
	}
	return selected
}

func buildDraftHypotheses(workspace *IncidentWorkspace, toolRuns []MCPToolInvocationResult) []AgentDraftHypothesis {
	if workspace == nil || workspace.Incident == nil {
		return nil
	}
	incident := workspace.Incident
	hypotheses := make([]AgentDraftHypothesis, 0, 3)
	switch incident.Domain {
	case "infrastructure":
		hypotheses = append(hypotheses,
			AgentDraftHypothesis{
				Title:      "Cluster capacity or node health is constraining normal operation",
				Confidence: "high",
				Rationale:  "Infrastructure incidents usually correlate with node pressure, runtime health degradation, or capacity imbalance.",
				SignalsUsed: []string{
					incident.Domain,
					"cluster_overview.get",
					"runtime_health.get",
				},
			},
		)
	case "golden_signals":
		hypotheses = append(hypotheses,
			AgentDraftHypothesis{
				Title:      "Service health is degrading through one or more golden signals",
				Confidence: "high",
				Rationale:  "Golden-signal incidents usually reflect rising latency, error pressure, traffic anomalies, saturation, or backlog pressure that can be cross-checked with recent HTTP windows, async backlog signals, and findings.",
				SignalsUsed: []string{
					incident.Domain,
					"http_signals.recent",
					"http_signals.history",
					"async_backlog.recent",
					"messaging_transport.recent",
					"findings.list",
				},
			},
		)
	case "runtime_services":
		hypotheses = append(hypotheses,
			AgentDraftHypothesis{
				Title:      "A runtime dependency is degraded or misconfigured",
				Confidence: "high",
				Rationale:  "Runtime-service incidents usually stem from dependency availability, image pulls, or config drift rather than business logic regressions.",
				SignalsUsed: []string{
					incident.Domain,
					"findings.list",
					"logs.recent",
				},
			},
		)
	case "release_configuration":
		hypotheses = append(hypotheses,
			AgentDraftHypothesis{
				Title:      "Release drift or partial rollout is the primary cause",
				Confidence: "high",
				Rationale:  "Release configuration incidents often align with drift records, partial applies, or rollout/compliance mismatches.",
				SignalsUsed: []string{
					incident.Domain,
					"release_drift.summary",
					"incidents.get",
				},
			},
		)
	default:
		hypotheses = append(hypotheses,
			AgentDraftHypothesis{
				Title:      "The incident is best explained by the newest stored findings and evidence",
				Confidence: "medium",
				Rationale:  "The current draft is staying conservative until more evidence is collected from read-only MCP tools.",
				SignalsUsed: []string{
					incident.Domain,
					"findings.list",
					"evidence.list",
				},
			},
		)
	}

	if hasToolRun(toolRuns, "logs.recent") {
		hypotheses = append(hypotheses, AgentDraftHypothesis{
			Title:      "Recent log signatures likely narrow the failure to one concrete component or symptom",
			Confidence: "medium",
			Rationale:  "Recent log lines can validate whether the issue is active now and whether it maps to a known error family.",
			SignalsUsed: []string{
				"logs.recent",
			},
		})
	}
	if hasToolRun(toolRuns, "runtime_health.get") {
		hypotheses = append(hypotheses, AgentDraftHypothesis{
			Title:      "Control-plane worker health may be amplifying or masking the incident",
			Confidence: "medium",
			Rationale:  "Watcher and dispatcher health can change whether the system is detecting, recovering from, or compounding the issue.",
			SignalsUsed: []string{
				"runtime_health.get",
			},
		})
	}
	if hasToolRun(toolRuns, "http_signals.recent") {
		hypotheses = append(hypotheses, AgentDraftHypothesis{
			Title:      "Recent request patterns can help distinguish between traffic, latency, and error pressure",
			Confidence: "medium",
			Rationale:  "A bounded HTTP window gives the draft a quick read on whether the service is overloaded, erroring, or just seeing benign traffic variation.",
			SignalsUsed: []string{
				"http_signals.recent",
			},
		})
	}
	if hasToolRun(toolRuns, "http_signals.history") {
		hypotheses = append(hypotheses, AgentDraftHypothesis{
			Title:      "Recent HTTP trend direction can separate a spike from a sustained regression",
			Confidence: "medium",
			Rationale:  "Trend windows help determine whether traffic, latency, or error pressure is building, stabilizing, or already recovering.",
			SignalsUsed: []string{
				"http_signals.history",
			},
		})
	}
	if hasToolRun(toolRuns, "async_backlog.recent") {
		hypotheses = append(hypotheses, AgentDraftHypothesis{
			Title:      "Async backlog pressure may be amplifying user-facing symptoms or delaying recovery",
			Confidence: "medium",
			Rationale:  "Queue and outbox backlog can indicate downstream processing pressure even when the primary symptom first appears in HTTP or logs.",
			SignalsUsed: []string{
				"async_backlog.recent",
				"logs.recent",
			},
		})
	}
	if hasToolRun(toolRuns, "messaging_transport.recent") {
		hypotheses = append(hypotheses, AgentDraftHypothesis{
			Title:      "Messaging transport instability may be contributing to backlog growth or delayed event delivery",
			Confidence: "medium",
			Rationale:  "Reconnect storms or disconnects can explain why async pressure builds even when application traffic is only moderately elevated.",
			SignalsUsed: []string{
				"messaging_transport.recent",
				"async_backlog.recent",
			},
		})
	}

	sort.SliceStable(hypotheses, func(i, j int) bool {
		return confidenceRank(hypotheses[i].Confidence) < confidenceRank(hypotheses[j].Confidence)
	})
	if len(hypotheses) > 3 {
		hypotheses = hypotheses[:3]
	}
	for i := range hypotheses {
		hypotheses[i].EvidenceRefs = evidenceRefsForSignals(hypotheses[i].SignalsUsed, toolRuns)
	}
	return hypotheses
}

func buildDraftPlan(workspace *IncidentWorkspace, hypotheses []AgentDraftHypothesis, toolRuns []MCPToolInvocationResult) []AgentDraftPlanStep {
	if workspace == nil || workspace.Incident == nil {
		return nil
	}
	steps := []AgentDraftPlanStep{
		{
			Title:        "Confirm the incident frame",
			Description:  "Review the incident summary, severity, and most recent signal to make sure the current thread still matches the active symptom.",
			EvidenceRefs: evidenceRefsForSignals([]string{"incidents.get", "findings.list"}, toolRuns),
		},
	}

	for _, question := range workspace.RecommendedQuestions {
		steps = append(steps, AgentDraftPlanStep{
			Title:        "Investigate a recommended question",
			Description:  question,
			EvidenceRefs: evidenceRefsForSignals([]string{"findings.list", "evidence.list"}, toolRuns),
		})
		if len(steps) >= 4 {
			break
		}
	}

	if len(hypotheses) > 0 {
		steps = append(steps, AgentDraftPlanStep{
			Title:        "Challenge the top hypothesis",
			Description:  fmt.Sprintf("Look for confirming and disconfirming evidence for: %s", hypotheses[0].Title),
			EvidenceRefs: hypotheses[0].EvidenceRefs,
		})
	}
	if hasToolRun(toolRuns, "logs.recent") {
		steps = append(steps, AgentDraftPlanStep{
			Title:        "Validate recency in logs",
			Description:  "Use the recent log output to confirm the symptom is current and scoped to the expected component.",
			EvidenceRefs: evidenceRefsForSignals([]string{"logs.recent"}, toolRuns),
		})
	}
	if hasToolRun(toolRuns, "http_signals.recent") {
		steps = append(steps, AgentDraftPlanStep{
			Title:        "Validate app-level golden signals",
			Description:  "Compare recent request volume, server-error rate, and latency to decide whether the incident is primarily traffic-driven, failure-driven, or performance-driven.",
			EvidenceRefs: evidenceRefsForSignals([]string{"http_signals.recent"}, toolRuns),
		})
	}
	if hasToolRun(toolRuns, "http_signals.history") {
		steps = append(steps, AgentDraftPlanStep{
			Title:        "Check whether the signal is worsening or recovering",
			Description:  "Use recent HTTP history windows to decide whether the symptom is a one-off spike, an active regression, or a recovering incident.",
			EvidenceRefs: evidenceRefsForSignals([]string{"http_signals.history"}, toolRuns),
		})
	}
	if hasToolRun(toolRuns, "async_backlog.recent") {
		steps = append(steps, AgentDraftPlanStep{
			Title:        "Check whether backlog pressure is contributing to the symptom",
			Description:  "Compare build queue, email queue, and messaging outbox pressure with recent HTTP trends and logs to decide whether async congestion is amplifying the incident.",
			EvidenceRefs: evidenceRefsForSignals([]string{"async_backlog.recent", "messaging_transport.recent", "http_signals.history", "logs.recent"}, toolRuns),
		})
	}
	steps = append(steps, AgentDraftPlanStep{
		Title:        "Escalate only after bounded review",
		Description:  "If the evidence still supports intervention, move to an approval-bound action rather than extending the read-only investigation loop.",
		EvidenceRefs: evidenceRefsForSignals([]string{"runtime_health.get", "cluster_overview.get", "release_drift.summary", "async_backlog.recent"}, toolRuns),
	})
	if len(steps) > 6 {
		steps = steps[:6]
	}
	return steps
}

func hasToolRun(runs []MCPToolInvocationResult, toolName string) bool {
	for _, run := range runs {
		if strings.TrimSpace(run.ToolName) == toolName {
			return true
		}
	}
	return false
}

func confidenceRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}

func evidenceRefsForSignals(signals []string, runs []MCPToolInvocationResult) []AgentDraftEvidenceRef {
	if len(signals) == 0 || len(runs) == 0 {
		return nil
	}
	index := make(map[string]MCPToolInvocationResult, len(runs))
	for _, run := range runs {
		index[strings.TrimSpace(run.ToolName)] = run
	}
	refs := make([]AgentDraftEvidenceRef, 0, len(signals))
	for _, signal := range signals {
		run, ok := index[strings.TrimSpace(signal)]
		if !ok {
			continue
		}
		refs = append(refs, AgentDraftEvidenceRef{
			ToolName:   run.ToolName,
			ServerName: run.ServerName,
			Summary:    summarizeToolRun(run),
		})
	}
	return refs
}

func summarizeToolRun(run MCPToolInvocationResult) string {
	if len(run.Payload) == 0 {
		return "No payload captured."
	}
	if count, ok := run.Payload["match_count"].(int); ok {
		return fmt.Sprintf("%d recent log matches returned.", count)
	}
	if count, ok := run.Payload["request_count"].(int64); ok {
		return fmt.Sprintf("HTTP window captured %d requests with %d%% server-error rate and %dms average latency.",
			count,
			metricFromPayload(run.Payload, "error_rate_percent"),
			metricFromPayload(run.Payload, "average_latency_ms"),
		)
	}
	if count, ok := run.Payload["count"].(int); ok && run.ToolName == "http_signals.history" {
		return fmt.Sprintf("%d recent HTTP signal windows captured for trend review.", count)
	}
	if run.ToolName == "async_backlog.recent" {
		return fmt.Sprintf("Backlog snapshot shows build queue=%d, email queue=%d, outbox pending=%d.",
			metricFromPayload(run.Payload, "build_queue_depth"),
			metricFromPayload(run.Payload, "email_queue_depth"),
			metricFromPayload(run.Payload, "messaging_outbox_pending"),
		)
	}
	if run.ToolName == "messaging_transport.recent" {
		return fmt.Sprintf("Messaging transport shows reconnects=%d, disconnects=%d, threshold=%d.",
			metricFromPayload(run.Payload, "reconnects"),
			metricFromPayload(run.Payload, "disconnects"),
			metricFromPayload(run.Payload, "reconnect_threshold"),
		)
	}
	if count, ok := run.Payload["count"].(int); ok {
		return fmt.Sprintf("%d records returned.", count)
	}
	if active, ok := run.Payload["active_drift_count"].(int64); ok {
		return fmt.Sprintf("Active drift count is %d.", active)
	}
	if active, ok := run.Payload["active_drift_count"].(int); ok {
		return fmt.Sprintf("Active drift count is %d.", active)
	}
	if count, ok := run.Payload["healthy_nodes"].(int); ok {
		return fmt.Sprintf("%d healthy nodes reported.", count)
	}
	if count, ok := run.Payload["healthy_nodes"].(int64); ok {
		return fmt.Sprintf("%d healthy nodes reported.", count)
	}
	if count, ok := run.Payload["count"].(float64); ok {
		return fmt.Sprintf("%.0f records returned.", count)
	}
	if count, ok := run.Payload["match_count"].(float64); ok {
		return fmt.Sprintf("%.0f recent log matches returned.", count)
	}
	return "Structured tool output captured for this draft."
}

func buildTriageNextChecks(draft *AgentDraftResponse) []string {
	if draft == nil {
		return nil
	}
	checks := make([]string, 0, 3)
	for _, step := range draft.InvestigationPlan {
		if len(checks) >= 3 {
			break
		}
		title := strings.TrimSpace(step.Title)
		description := strings.TrimSpace(step.Description)
		switch {
		case title != "" && description != "":
			checks = append(checks, fmt.Sprintf("%s: %s", title, description))
		case title != "":
			checks = append(checks, title)
		case description != "":
			checks = append(checks, description)
		}
	}
	if len(checks) == 0 {
		if summary := strings.TrimSpace(draft.Summary); summary != "" {
			checks = append(checks, fmt.Sprintf("Confirm incident summary against stored evidence: %s", summary))
		}
	}
	for len(checks) < 3 {
		switch len(checks) {
		case 0:
			checks = append(checks, "Review the latest findings and evidence rows for this incident.")
		case 1:
			checks = append(checks, "Run bounded read-only MCP tools from the default workspace bundle.")
		default:
			checks = append(checks, "Request approval before any disruptive remediation path.")
		}
	}
	return checks
}

func buildTriageCheckRefs(draft *AgentDraftResponse, checks []string) []AgentTriageCheckRef {
	if len(checks) == 0 {
		return nil
	}
	signals := triageReferenceSignals(draft)
	refs := make([]AgentTriageCheckRef, 0, len(checks))
	for idx, check := range checks {
		signal := "findings.list"
		if idx < len(signals) && strings.TrimSpace(signals[idx]) != "" {
			signal = strings.TrimSpace(signals[idx])
		}
		runbookSource, runbookSection := triageRunbookForSignal(signal)
		evidenceNote := triageEvidenceNoteForSignal(draft, signal)
		refs = append(refs, AgentTriageCheckRef{
			Check:           strings.TrimSpace(check),
			RunbookSource:   runbookSource,
			RunbookSection:  runbookSection,
			EvidenceSignals: []string{signal},
			EvidenceNote:    evidenceNote,
		})
	}
	return refs
}

func triageReferenceSignals(draft *AgentDraftResponse) []string {
	ordered := make([]string, 0, 8)
	seen := map[string]struct{}{}
	push := func(signal string) {
		value := strings.TrimSpace(signal)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}
	if draft != nil && len(draft.Hypotheses) > 0 {
		for _, signal := range draft.Hypotheses[0].SignalsUsed {
			push(signal)
		}
	}
	if draft != nil {
		for _, run := range draft.ToolRuns {
			push(run.ToolName)
		}
	}
	push("findings.list")
	push("evidence.list")
	return ordered
}

func triageRunbookForSignal(signal string) (string, string) {
	switch strings.TrimSpace(signal) {
	case "async_backlog.recent", "messaging_transport.recent":
		return "docs/implementation/SRE_SMART_BOT_ASYNC_BACKLOG_TRANSPORT_PRESSURE_EPIC.md", "Async backlog and transport pressure correlation"
	case "messaging_consumers.recent":
		return "docs/implementation/SRE_SMART_BOT_NATS_CONSUMER_LAG_PRESSURE_EPIC.md", "NATS consumer lag diagnostics"
	case "cluster_overview.get", "runtime_health.get":
		return "docs/implementation/SRE_SMART_BOT_EXTERNAL_CLUSTER_DEPLOYMENT_RUNBOOK.md", "External cluster health and rollout checks"
	case "release_drift.summary", "http_signals.recent", "http_signals.history", "logs.recent":
		return "docs/implementation/ROBOT_SRE_INCIDENT_TAXONOMY_AND_POLICY_MATRIX.md", "Incident taxonomy and response policy"
	default:
		return "docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md", "Deterministic and approval-safe actions"
	}
}

func triageEvidenceNoteForSignal(draft *AgentDraftResponse, signal string) string {
	if draft != nil {
		for _, run := range draft.ToolRuns {
			if strings.TrimSpace(run.ToolName) == strings.TrimSpace(signal) {
				return summarizeToolRun(run)
			}
		}
	}
	return fmt.Sprintf("No direct tool payload for %s in this draft; use findings/evidence rows to validate the check.", strings.TrimSpace(signal))
}

func deriveTriageRecommendedAction(draft *AgentDraftResponse) string {
	if draft == nil {
		return "Collect more evidence before selecting a remediation path."
	}
	topSignals := make([]string, 0)
	if len(draft.Hypotheses) > 0 {
		topSignals = draft.Hypotheses[0].SignalsUsed
	}
	if containsSignal(topSignals, "messaging_transport.recent") {
		return "Prefer recommendation-only review_messaging_transport_health first, then re-check backlog pressure before scaling workers."
	}
	if containsSignal(topSignals, "async_backlog.recent") {
		return "Prefer recommendation-only review_async_worker_capacity or review_dispatcher_backlog_pressure before any disruptive intervention."
	}
	if containsSignal(topSignals, "release_drift.summary") {
		return "Validate release drift and rollout state first; avoid runtime restarts until drift direction is confirmed."
	}
	if containsSignal(topSignals, "cluster_overview.get") || containsSignal(topSignals, "runtime_health.get") {
		return "Review cluster and dependency health before config mutations; keep remediation approval-bound."
	}
	return "Follow the top bounded investigation step and keep execution in deterministic approval-gated action flows."
}

func containsSignal(signals []string, wanted string) bool {
	for _, signal := range signals {
		if strings.TrimSpace(signal) == wanted {
			return true
		}
	}
	return false
}

func normalizeTriageConfidence(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func metricFromPayload(payload map[string]any, key string) int64 {
	switch value := payload[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}
