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
	RecommendedAction string                  `json:"recommended_action"`
	EvidenceRefs      []AgentDraftEvidenceRef `json:"evidence_refs,omitempty"`
	HumanConfirmation bool                    `json:"human_confirmation_required"`
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

	return &AgentTriageResponse{
		IncidentID:        draft.IncidentID,
		Mode:              "deterministic_triage",
		Summary:           draft.Summary,
		ProbableCause:     probableCause,
		Confidence:        confidence,
		NextChecks:        buildTriageNextChecks(draft),
		RecommendedAction: deriveTriageRecommendedAction(draft),
		EvidenceRefs:      evidenceRefs,
		HumanConfirmation: draft.HumanConfirmation,
	}
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
