package sresmartbot

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	EnabledMCPServers    []systemconfig.RobotSREMCPServer        `json:"enabled_mcp_servers"`
	AgentRuntime         systemconfig.RobotSREAgentRuntimeConfig `json:"agent_runtime"`
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
