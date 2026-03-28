package sresmartbot

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// IncidentWorkspace is the MCP/AI-ready bundle for a single incident.
// It gives the future standalone agent runtime a bounded, structured view
// without coupling it directly to repository internals.
type IncidentWorkspace struct {
	Incident             *domainsresmartbot.Incident             `json:"incident"`
	ExecutiveSummary     []string                                `json:"executive_summary"`
	RecommendedQuestions []string                                `json:"recommended_questions"`
	SuggestedTooling     []string                                `json:"suggested_tooling"`
	DefaultToolBundle    []string                                `json:"default_tool_bundle,omitempty"`
	AsyncPressureSummary *AsyncPressureWorkspaceSummary          `json:"async_pressure_summary,omitempty"`
	EnabledMCPServers    []systemconfig.RobotSREMCPServer        `json:"enabled_mcp_servers"`
	AgentRuntime         systemconfig.RobotSREAgentRuntimeConfig `json:"agent_runtime"`
}

type AsyncPressureWorkspaceSummary struct {
	Backlog            *AsyncBacklogWorkspaceSummary       `json:"backlog,omitempty"`
	MessagingTransport *MessagingTransportWorkspaceSummary `json:"messaging_transport,omitempty"`
	MessagingConsumer  *MessagingConsumerWorkspaceSummary  `json:"messaging_consumer,omitempty"`
}

type AsyncBacklogWorkspaceSummary struct {
	IncidentType          string            `json:"incident_type"`
	DisplayName           string            `json:"display_name"`
	QueueKind             string            `json:"queue_kind"`
	Subsystem             string            `json:"subsystem"`
	Count                 int64             `json:"count"`
	Threshold             int64             `json:"threshold"`
	ThresholdDelta        int64             `json:"threshold_delta"`
	ThresholdRatioPercent int64             `json:"threshold_ratio_percent"`
	Trend                 string            `json:"trend"`
	OperatorStatus        string            `json:"operator_status"`
	LatestSummary         string            `json:"latest_summary"`
	LatestCapturedAt      *time.Time        `json:"latest_captured_at,omitempty"`
	RecentObservations    map[string]int64  `json:"recent_observations,omitempty"`
	CorrelationHints      map[string]string `json:"correlation_hints,omitempty"`
}

type MessagingTransportWorkspaceSummary struct {
	Status             string     `json:"status"`
	Reconnects         int64      `json:"reconnects"`
	Disconnects        int64      `json:"disconnects"`
	ReconnectThreshold int64      `json:"reconnect_threshold"`
	OperatorStatus     string     `json:"operator_status"`
	LatestSummary      string     `json:"latest_summary"`
	LatestCapturedAt   *time.Time `json:"latest_captured_at,omitempty"`
}

type MessagingConsumerWorkspaceSummary struct {
	IncidentType          string            `json:"incident_type"`
	DisplayName           string            `json:"display_name"`
	Kind                  string            `json:"kind"`
	Stream                string            `json:"stream"`
	Consumer              string            `json:"consumer"`
	TargetRef             string            `json:"target_ref"`
	Count                 int64             `json:"count"`
	Threshold             int64             `json:"threshold"`
	ThresholdDelta        int64             `json:"threshold_delta"`
	ThresholdRatioPercent int64             `json:"threshold_ratio_percent"`
	PendingCount          int64             `json:"pending_count"`
	AckPendingCount       int64             `json:"ack_pending_count"`
	WaitingCount          int64             `json:"waiting_count"`
	RedeliveredCount      int64             `json:"redelivered_count"`
	Trend                 string            `json:"trend"`
	OperatorStatus        string            `json:"operator_status"`
	LatestSummary         string            `json:"latest_summary"`
	LatestCapturedAt      *time.Time        `json:"latest_captured_at,omitempty"`
	LastActive            *time.Time        `json:"last_active,omitempty"`
	CorrelationHints      map[string]string `json:"correlation_hints,omitempty"`
}

type WorkspaceService struct {
	repo                domainsresmartbot.Repository
	systemConfigService *systemconfig.Service
}

func NewWorkspaceService(repo domainsresmartbot.Repository, systemConfigService *systemconfig.Service) *WorkspaceService {
	return &WorkspaceService{
		repo:                repo,
		systemConfigService: systemConfigService,
	}
}

func (s *WorkspaceService) BuildIncidentWorkspace(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*IncidentWorkspace, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("workspace service is not configured")
	}

	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	findings, _ := s.repo.ListFindingsByIncident(ctx, incidentID)
	evidence, _ := s.repo.ListEvidenceByIncident(ctx, incidentID)
	actions, _ := s.repo.ListActionAttemptsByIncident(ctx, incidentID)
	approvals, _ := s.repo.ListApprovalsByIncident(ctx, incidentID)

	policy := defaultRobotWorkspacePolicy()
	if s.systemConfigService != nil {
		cfg, cfgErr := s.systemConfigService.GetRobotSREPolicyConfig(ctx, tenantID)
		if cfgErr == nil && cfg != nil {
			policy = *cfg
		}
	}

	return &IncidentWorkspace{
		Incident:             incident,
		ExecutiveSummary:     buildWorkspaceExecutiveSummary(incident, findings, actions, approvals),
		RecommendedQuestions: buildWorkspaceRecommendedQuestions(incident, findings, evidence, actions),
		SuggestedTooling:     buildWorkspaceSuggestedTooling(policy.MCPServers, incident),
		DefaultToolBundle:    buildWorkspaceDefaultToolBundle(incident),
		AsyncPressureSummary: buildAsyncPressureWorkspaceSummary(incident, evidence),
		EnabledMCPServers:    enabledMCPServers(policy.MCPServers),
		AgentRuntime:         policy.AgentRuntime,
	}, nil
}

func defaultRobotWorkspacePolicy() systemconfig.RobotSREPolicyConfig {
	return systemconfig.RobotSREPolicyConfig{
		AgentRuntime: systemconfig.RobotSREAgentRuntimeConfig{
			Provider:                           "custom",
			SystemPromptRef:                    "sre_smart_bot_default",
			OperatorSummaryEnabled:             true,
			HypothesisRankingEnabled:           true,
			DraftActionPlansEnabled:            true,
			MaxToolCallsPerTurn:                6,
			MaxIncidentsPerSummary:             5,
			RequireHumanConfirmationForMessage: true,
		},
	}
}

func buildWorkspaceExecutiveSummary(
	incident *domainsresmartbot.Incident,
	findings []*domainsresmartbot.Finding,
	actions []*domainsresmartbot.ActionAttempt,
	approvals []*domainsresmartbot.Approval,
) []string {
	if incident == nil {
		return nil
	}

	pendingApprovals := 0
	for _, approval := range approvals {
		if approval != nil && approval.DecidedAt == nil {
			pendingApprovals++
		}
	}

	executableReady := 0
	for _, action := range actions {
		if action == nil {
			continue
		}
		if slices.Contains([]string{"reconcile_tenant_assets", "review_provider_connectivity", "email_incident_summary"}, action.ActionKey) &&
			slices.Contains([]string{"approved", "proposed"}, strings.ToLower(strings.TrimSpace(action.Status))) {
			executableReady++
		}
	}

	topFinding := "No finding titles recorded yet."
	if len(findings) > 0 && findings[0] != nil {
		if strings.TrimSpace(findings[0].Title) != "" {
			topFinding = findings[0].Title
		} else if strings.TrimSpace(findings[0].Message) != "" {
			topFinding = findings[0].Message
		}
	}

	latestAction := "No remediation actions have been attempted yet."
	if len(actions) > 0 && actions[0] != nil {
		latestAction = fmt.Sprintf("Latest action activity: %s is %s.", actions[0].ActionKey, strings.TrimSpace(actions[0].Status))
	}

	return []string{
		fmt.Sprintf("%s is currently %s with %s severity in %s.", incident.DisplayName, incident.Status, incident.Severity, incident.Domain),
		func() string {
			if pendingApprovals == 0 {
				return "There are no pending approval requests on this incident thread."
			}
			return fmt.Sprintf("%d approval request(s) still need operator attention.", pendingApprovals)
		}(),
		func() string {
			if executableReady == 0 {
				return "No executable actions are currently waiting to run."
			}
			return fmt.Sprintf("%d executable action(s) are ready or nearly ready for operator review.", executableReady)
		}(),
		"Most recent signal: " + topFinding,
		latestAction,
	}
}

func buildWorkspaceRecommendedQuestions(
	incident *domainsresmartbot.Incident,
	findings []*domainsresmartbot.Finding,
	evidence []*domainsresmartbot.Evidence,
	actions []*domainsresmartbot.ActionAttempt,
) []string {
	if incident == nil {
		return nil
	}
	questions := []string{
		fmt.Sprintf("What changed just before %s entered %s status?", incident.DisplayName, incident.Status),
		fmt.Sprintf("Which evidence items best explain the %s incident type?", incident.IncidentType),
	}

	switch incident.Domain {
	case "infrastructure":
		questions = append(questions,
			"Are node, storage, or cluster runtime signals corroborating the incident?",
			"Would a bounded containment action reduce blast radius without adding churn?",
		)
	case "runtime_services":
		questions = append(questions,
			"Which runtime dependency is degraded first and what is the concrete failure mode?",
			"Do the newest findings suggest pull failures, health-check failures, or config drift?",
		)
	case "application_services":
		questions = append(questions,
			"Is this isolated to one service or part of a wider release or dependency issue?",
			"What customer-facing symptom should be communicated if this persists?",
		)
	case "identity_security":
		questions = append(questions,
			"Is the failure rooted in connectivity, credentials, or provider-side availability?",
			"Should outbound approvals or operator notifications be limited until identity recovers?",
		)
	default:
		questions = append(questions,
			"What is the smallest safe next investigation step?",
			"Which proposed action has the highest evidence support and lowest operational risk?",
		)
	}

	if len(findings) == 0 {
		questions = append(questions, "Do we need more detector coverage or watcher evidence before taking action?")
	}
	if len(evidence) == 0 {
		questions = append(questions, "Should the bot collect more structured evidence before escalating?")
	}
	if len(actions) > 0 {
		questions = append(questions, "Why did earlier proposed actions succeed, stall, or fail?")
	}

	return dedupeStrings(questions)
}

func buildWorkspaceSuggestedTooling(servers []systemconfig.RobotSREMCPServer, incident *domainsresmartbot.Incident) []string {
	tooling := make([]string, 0)
	for _, server := range enabledMCPServers(servers) {
		switch server.Kind {
		case "observability":
			tooling = append(tooling, "Use observability MCP tools to review findings, evidence, and runtime health around the incident window.")
		case "kubernetes":
			tooling = append(tooling, "Use Kubernetes MCP tools for node, pod, event, and workload inspection with bounded read paths first.")
		case "oci":
			tooling = append(tooling, "Use OCI MCP tools for instance, volume, or node-pool context before any disruptive recovery action.")
		case "database":
			tooling = append(tooling, "Use database MCP tools for read-only schema and health checks before changing runtime state.")
		case "release":
			tooling = append(tooling, "Use release MCP tools to confirm drift, rollout state, and policy posture.")
		case "chat":
			tooling = append(tooling, "Use channel-provider MCP tools only after human confirmation for outbound summaries or approvals.")
		}
	}
	if incident != nil && incident.Domain == "identity_security" {
		tooling = append(tooling, "Prefer read-only tooling until identity or approval paths are stable again.")
	}
	return dedupeStrings(tooling)
}

func buildWorkspaceDefaultToolBundle(incident *domainsresmartbot.Incident) []string {
	if incident == nil {
		return nil
	}
	tools := []string{"incidents.get", "findings.list", "evidence.list"}
	if isAsyncBacklogIncidentType(incident.IncidentType) {
		tools = append(tools, "async_backlog.recent", "messaging_transport.recent")
	}
	if isMessagingConsumerIncidentType(incident.IncidentType) {
		tools = append(tools, "messaging_consumers.recent", "messaging_transport.recent", "async_backlog.recent")
	}
	if strings.TrimSpace(incident.IncidentType) == "messaging_transport_degraded" {
		tools = append(tools, "messaging_transport.recent", "async_backlog.recent")
	}
	if incident.Domain == "golden_signals" {
		tools = append(tools, "runtime_health.get")
	}
	if strings.Contains(strings.TrimSpace(incident.IncidentType), "http") || strings.TrimSpace(incident.IncidentType) == "error_pressure" || strings.TrimSpace(incident.IncidentType) == "saturation_risk" {
		tools = append(tools, "http_signals.recent", "http_signals.history")
	}
	return dedupeStrings(tools)
}

func buildAsyncPressureWorkspaceSummary(incident *domainsresmartbot.Incident, evidence []*domainsresmartbot.Evidence) *AsyncPressureWorkspaceSummary {
	if incident == nil {
		return nil
	}
	summary := &AsyncPressureWorkspaceSummary{}
	if isAsyncBacklogIncidentType(incident.IncidentType) {
		if backlog := buildAsyncBacklogWorkspaceSummary(incident, evidence); backlog != nil {
			summary.Backlog = backlog
		}
	}
	if strings.TrimSpace(incident.IncidentType) == "messaging_transport_degraded" || isAsyncBacklogIncidentType(incident.IncidentType) {
		if transport := buildMessagingTransportWorkspaceSummary(evidence); transport != nil {
			summary.MessagingTransport = transport
		}
	}
	if isMessagingConsumerIncidentType(incident.IncidentType) {
		if consumer := buildMessagingConsumerWorkspaceSummary(incident, evidence); consumer != nil {
			summary.MessagingConsumer = consumer
		}
		if transport := buildMessagingTransportWorkspaceSummary(evidence); transport != nil {
			summary.MessagingTransport = transport
		}
	}
	if summary.Backlog == nil && summary.MessagingTransport == nil && summary.MessagingConsumer == nil {
		return nil
	}
	return summary
}

func buildAsyncBacklogWorkspaceSummary(incident *domainsresmartbot.Incident, evidence []*domainsresmartbot.Evidence) *AsyncBacklogWorkspaceSummary {
	latest := latestEvidenceByPrefix(evidence, "_backlog_snapshot")
	if latest == nil {
		return nil
	}
	payload := parseEvidencePayload(latest.Payload)
	summary := &AsyncBacklogWorkspaceSummary{
		IncidentType:          strings.TrimSpace(incident.IncidentType),
		DisplayName:           strings.TrimSpace(incident.DisplayName),
		QueueKind:             stringMap(payload, "queue_kind"),
		Subsystem:             stringMap(payload, "subsystem"),
		Count:                 int64Map(payload, "count"),
		Threshold:             int64Map(payload, "threshold"),
		ThresholdDelta:        int64Map(payload, "threshold_delta"),
		ThresholdRatioPercent: int64Map(payload, "threshold_ratio_percent"),
		Trend:                 stringMap(payload, "trend"),
		LatestSummary:         strings.TrimSpace(latest.Summary),
		RecentObservations:    int64MapMap(payload, "recent_observations"),
		CorrelationHints:      stringMapMap(payload, "correlation_hints"),
	}
	if !latest.CapturedAt.IsZero() {
		capturedAt := latest.CapturedAt
		summary.LatestCapturedAt = &capturedAt
	}
	summary.OperatorStatus = asyncBacklogOperatorStatus(summary.Trend)
	return summary
}

func buildMessagingTransportWorkspaceSummary(evidence []*domainsresmartbot.Evidence) *MessagingTransportWorkspaceSummary {
	latest := latestEvidenceByType(evidence, "nats_transport_status")
	if latest == nil {
		return nil
	}
	payload := parseEvidencePayload(latest.Payload)
	summary := &MessagingTransportWorkspaceSummary{
		Status:             stringMap(payload, "status"),
		Reconnects:         int64Map(payload, "reconnects"),
		Disconnects:        int64Map(payload, "disconnects"),
		ReconnectThreshold: int64Map(payload, "reconnect_threshold"),
		LatestSummary:      strings.TrimSpace(latest.Summary),
	}
	if !latest.CapturedAt.IsZero() {
		capturedAt := latest.CapturedAt
		summary.LatestCapturedAt = &capturedAt
	}
	summary.OperatorStatus = messagingTransportOperatorStatus(summary)
	return summary
}

func buildMessagingConsumerWorkspaceSummary(incident *domainsresmartbot.Incident, evidence []*domainsresmartbot.Evidence) *MessagingConsumerWorkspaceSummary {
	latest := latestEvidenceByTypes(evidence,
		"nats_consumer_lag_snapshot",
		"nats_consumer_progress_snapshot",
		"nats_consumer_ack_pressure_snapshot",
	)
	if latest == nil {
		return nil
	}
	payload := parseEvidencePayload(latest.Payload)
	summary := &MessagingConsumerWorkspaceSummary{
		IncidentType:          strings.TrimSpace(incident.IncidentType),
		DisplayName:           strings.TrimSpace(incident.DisplayName),
		Kind:                  stringMap(payload, "kind"),
		Stream:                stringMap(payload, "stream"),
		Consumer:              stringMap(payload, "consumer"),
		TargetRef:             stringMap(payload, "target_ref"),
		Count:                 int64Map(payload, "count"),
		Threshold:             int64Map(payload, "threshold"),
		ThresholdDelta:        int64Map(payload, "threshold_delta"),
		ThresholdRatioPercent: int64Map(payload, "threshold_ratio_percent"),
		PendingCount:          int64Map(payload, "pending_count"),
		AckPendingCount:       int64Map(payload, "ack_pending_count"),
		WaitingCount:          int64Map(payload, "waiting_count"),
		RedeliveredCount:      int64Map(payload, "redelivered_count"),
		Trend:                 stringMap(payload, "trend"),
		LatestSummary:         strings.TrimSpace(latest.Summary),
		CorrelationHints:      stringMapMap(payload, "correlation_hints"),
	}
	if !latest.CapturedAt.IsZero() {
		capturedAt := latest.CapturedAt
		summary.LatestCapturedAt = &capturedAt
	}
	if lastActive, ok := timeMap(payload, "last_active"); ok {
		summary.LastActive = &lastActive
	}
	summary.OperatorStatus = messagingConsumerOperatorStatus(summary)
	return summary
}

func latestEvidenceByPrefix(evidence []*domainsresmartbot.Evidence, suffix string) *domainsresmartbot.Evidence {
	for _, item := range evidence {
		if item == nil {
			continue
		}
		if strings.HasSuffix(strings.TrimSpace(item.EvidenceType), suffix) {
			return item
		}
	}
	return nil
}

func latestEvidenceByType(evidence []*domainsresmartbot.Evidence, evidenceType string) *domainsresmartbot.Evidence {
	for _, item := range evidence {
		if item == nil {
			continue
		}
		if strings.TrimSpace(item.EvidenceType) == evidenceType {
			return item
		}
	}
	return nil
}

func latestEvidenceByTypes(evidence []*domainsresmartbot.Evidence, evidenceTypes ...string) *domainsresmartbot.Evidence {
	if len(evidenceTypes) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(evidenceTypes))
	for _, evidenceType := range evidenceTypes {
		allowed[strings.TrimSpace(evidenceType)] = struct{}{}
	}
	for _, item := range evidence {
		if item == nil {
			continue
		}
		if _, ok := allowed[strings.TrimSpace(item.EvidenceType)]; ok {
			return item
		}
	}
	return nil
}

func parseEvidencePayload(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func stringMap(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func timeMap(payload map[string]any, key string) (time.Time, bool) {
	value, ok := payload[key]
	if !ok {
		return time.Time{}, false
	}
	text, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(text))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func int64Map(payload map[string]any, key string) int64 {
	value, ok := payload[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func int64MapMap(payload map[string]any, key string) map[string]int64 {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	entries, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]int64, len(entries))
	for entryKey, entryValue := range entries {
		switch typed := entryValue.(type) {
		case float64:
			result[entryKey] = int64(typed)
		case int64:
			result[entryKey] = typed
		case int:
			result[entryKey] = int64(typed)
		}
	}
	return result
}

func stringMapMap(payload map[string]any, key string) map[string]string {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	entries, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(entries))
	for entryKey, entryValue := range entries {
		text, ok := entryValue.(string)
		if ok {
			result[entryKey] = strings.TrimSpace(text)
		}
	}
	return result
}

func asyncBacklogOperatorStatus(trend string) string {
	switch strings.TrimSpace(trend) {
	case "growing":
		return "backlog growing"
	case "stable":
		return "backlog stable but elevated"
	case "improving":
		return "backlog improving"
	default:
		return "backlog elevated"
	}
}

func messagingTransportOperatorStatus(summary *MessagingTransportWorkspaceSummary) string {
	if summary == nil {
		return ""
	}
	if summary.ReconnectThreshold > 0 && summary.Reconnects >= summary.ReconnectThreshold {
		return "transport unstable"
	}
	if summary.Disconnects > 0 {
		return "transport interruptions observed"
	}
	if strings.TrimSpace(summary.Status) != "" {
		return strings.ReplaceAll(strings.TrimSpace(summary.Status), "_", " ")
	}
	return "transport context available"
}

func messagingConsumerOperatorStatus(summary *MessagingConsumerWorkspaceSummary) string {
	if summary == nil {
		return ""
	}
	switch strings.TrimSpace(summary.Kind) {
	case "stalled_consumer_progress":
		return "consumer progress stalled"
	case "pending_ack_saturation":
		switch strings.TrimSpace(summary.Trend) {
		case "growing":
			return "pending-ack pressure growing"
		case "stable":
			return "pending-ack pressure stable but elevated"
		default:
			return "pending-ack pressure elevated"
		}
	default:
		switch strings.TrimSpace(summary.Trend) {
		case "growing":
			return "consumer lag growing"
		case "stable":
			return "consumer lag stable but elevated"
		case "improving":
			return "consumer lag improving"
		default:
			return "consumer lag elevated"
		}
	}
}

func isAsyncBacklogIncidentType(incidentType string) bool {
	switch strings.TrimSpace(incidentType) {
	case "backlog_pressure", "dispatcher_backlog_pressure", "email_queue_backlog_pressure", "messaging_outbox_backlog_pressure":
		return true
	default:
		return false
	}
}

func isMessagingConsumerIncidentType(incidentType string) bool {
	switch strings.TrimSpace(incidentType) {
	case "nats_consumer_lag_pressure", "nats_consumer_stalled_progress", "nats_consumer_ack_pressure":
		return true
	default:
		return false
	}
}

func enabledMCPServers(servers []systemconfig.RobotSREMCPServer) []systemconfig.RobotSREMCPServer {
	if len(servers) == 0 {
		return nil
	}
	enabled := make([]systemconfig.RobotSREMCPServer, 0, len(servers))
	for _, server := range servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	return enabled
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
