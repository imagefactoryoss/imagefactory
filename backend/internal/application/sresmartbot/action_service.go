package sresmartbot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domaininfrastructure "github.com/srikarm/image-factory/internal/domain/infrastructure"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	domainsystemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

var (
	ErrSREActionNotFound         = errors.New("sre action attempt not found")
	ErrSREApprovalNotFound       = errors.New("sre approval not found")
	ErrSREApprovalRequired       = errors.New("approved decision required before execution")
	ErrSREActionNotExecutable    = errors.New("sre action is not executable")
	ErrSREApprovalAlreadyDecided = errors.New("sre approval already decided")
)

type InfrastructureActionService interface {
	ListProvidersAll(ctx context.Context, opts *domaininfrastructure.ListProvidersOptions) (*domaininfrastructure.ListProvidersResult, error)
	ReconcileStaleTenantNamespaces(ctx context.Context, providerID uuid.UUID, requestedBy *uuid.UUID) (*domaininfrastructure.TenantNamespaceReconcileSummary, error)
	RunProviderReadinessWatchTick(ctx context.Context, pageSize int) (*domaininfrastructure.ProviderReadinessWatchTickResult, error)
}

type AdminRecipientResolver interface {
	ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error)
	ListSystemAdminUserIDs(ctx context.Context) ([]uuid.UUID, error)
	ListUserEmailsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]string, error)
}

type EmailNotificationSender interface {
	SendEmail(ctx context.Context, tenantID uuid.UUID, toEmail, subject, bodyText, bodyHTML string, priority int) error
}

type RobotSREPolicyReader interface {
	GetRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*domainsystemconfig.RobotSREPolicyConfig, error)
}

type ActionService struct {
	repo              domainsresmartbot.Repository
	infra             InfrastructureActionService
	policyReader      RobotSREPolicyReader
	adminRecipients   AdminRecipientResolver
	emailNotification EmailNotificationSender
	logger            *zap.Logger
}

func NewActionService(repo domainsresmartbot.Repository, infra InfrastructureActionService, policyReader RobotSREPolicyReader, adminRecipients AdminRecipientResolver, emailNotification EmailNotificationSender, logger *zap.Logger) *ActionService {
	return &ActionService{
		repo:              repo,
		infra:             infra,
		policyReader:      policyReader,
		adminRecipients:   adminRecipients,
		emailNotification: emailNotification,
		logger:            logger,
	}
}

func (s *ActionService) RequestApproval(ctx context.Context, incidentID uuid.UUID, actionAttemptID uuid.UUID, requestedBy string, channelProviderID string, requestMessage string) (*domainsresmartbot.Approval, *domainsresmartbot.ActionAttempt, error) {
	if s == nil || s.repo == nil {
		return nil, nil, nil
	}
	incident, action, err := s.loadIncidentAndAction(ctx, incidentID, actionAttemptID)
	if err != nil {
		return nil, nil, err
	}
	if channelProviderID == "" && s.policyReader != nil {
		policy, policyErr := s.policyReader.GetRobotSREPolicyConfig(ctx, incident.TenantID)
		if policyErr == nil && policy != nil {
			channelProviderID = strings.TrimSpace(policy.DefaultChannelProviderID)
		}
	}
	if channelProviderID == "" {
		channelProviderID = "in-app-default"
	}
	now := time.Now().UTC()
	existingApprovals, err := s.repo.ListApprovalsByIncident(ctx, incidentID)
	if err != nil {
		return nil, nil, err
	}
	for _, approval := range existingApprovals {
		if approval == nil || approval.ActionAttemptID == nil {
			continue
		}
		if *approval.ActionAttemptID == actionAttemptID && strings.EqualFold(approval.Status, "pending") {
			return approval, action, nil
		}
	}
	approval := &domainsresmartbot.Approval{
		ID:                uuid.New(),
		IncidentID:        incidentID,
		ActionAttemptID:   &actionAttemptID,
		ChannelProviderID: channelProviderID,
		Status:            "pending",
		RequestMessage:    requestMessageOrDefault(requestMessage, action),
		RequestedBy:       strings.TrimSpace(requestedBy),
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.CreateApproval(ctx, approval); err != nil {
		return nil, nil, err
	}
	action.ApprovalRequired = true
	action.Status = "awaiting_approval"
	action.UpdatedAt = now
	if err := s.repo.UpdateActionAttempt(ctx, action); err != nil {
		return nil, nil, err
	}
	return approval, action, nil
}

func (s *ActionService) DecideApproval(ctx context.Context, incidentID uuid.UUID, approvalID uuid.UUID, decision string, decidedBy string, comment string) (*domainsresmartbot.Approval, *domainsresmartbot.ActionAttempt, error) {
	if s == nil || s.repo == nil {
		return nil, nil, nil
	}
	approval, action, err := s.loadApprovalAndAction(ctx, incidentID, approvalID)
	if err != nil {
		return nil, nil, err
	}
	if approval.DecidedAt != nil {
		return nil, nil, ErrSREApprovalAlreadyDecided
	}
	now := time.Now().UTC()
	normalizedDecision := strings.ToLower(strings.TrimSpace(decision))
	switch normalizedDecision {
	case "approved":
		approval.Status = "approved"
		action.Status = "approved"
	case "rejected":
		approval.Status = "rejected"
		action.Status = "rejected"
		action.CompletedAt = &now
	default:
		return nil, nil, fmt.Errorf("invalid decision: %s", decision)
	}
	approval.DecidedBy = strings.TrimSpace(decidedBy)
	approval.DecisionComment = strings.TrimSpace(comment)
	approval.DecidedAt = &now
	approval.UpdatedAt = now
	if err := s.repo.UpdateApproval(ctx, approval); err != nil {
		return nil, nil, err
	}
	action.UpdatedAt = now
	if err := s.repo.UpdateActionAttempt(ctx, action); err != nil {
		return nil, nil, err
	}
	return approval, action, nil
}

func (s *ActionService) ExecuteAction(ctx context.Context, incidentID uuid.UUID, actionAttemptID uuid.UUID, actorID string, actorTenantID *uuid.UUID) (*domainsresmartbot.ActionAttempt, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	incident, action, err := s.loadIncidentAndAction(ctx, incidentID, actionAttemptID)
	if err != nil {
		return nil, err
	}
	if action.ApprovalRequired {
		approved, approvalErr := s.isActionApproved(ctx, incidentID, actionAttemptID)
		if approvalErr != nil {
			return nil, approvalErr
		}
		if !approved {
			return nil, ErrSREApprovalRequired
		}
	}
	if !isExecutableAction(action.ActionKey) {
		return nil, ErrSREActionNotExecutable
	}
	now := time.Now().UTC()
	action.Status = "running"
	action.ActorType = "operator"
	action.ActorID = strings.TrimSpace(actorID)
	action.StartedAt = &now
	action.UpdatedAt = now
	if err := s.repo.UpdateActionAttempt(ctx, action); err != nil {
		return nil, err
	}

	var resultPayload map[string]interface{}
	switch action.ActionKey {
	case "reconcile_tenant_assets":
		resultPayload, err = s.executeTenantAssetReconcile(ctx)
	case "review_provider_connectivity":
		resultPayload, err = s.executeProviderReadinessRefresh(ctx)
	case "email_incident_summary":
		resultPayload, err = s.executeIncidentSummaryEmail(ctx, incident, actorTenantID)
	default:
		err = ErrSREActionNotExecutable
	}

	completedAt := time.Now().UTC()
	action.CompletedAt = &completedAt
	action.UpdatedAt = completedAt
	if err != nil {
		action.Status = "failed"
		action.ErrorMessage = err.Error()
		action.ResultPayload = mustJSON(map[string]interface{}{
			"error": err.Error(),
		})
		if updateErr := s.repo.UpdateActionAttempt(ctx, action); updateErr != nil && s.logger != nil {
			s.logger.Warn("Failed to persist failed SRE action attempt", zap.Error(updateErr))
		}
		return nil, err
	}

	action.Status = "completed"
	action.ErrorMessage = ""
	action.ResultPayload = mustJSON(resultPayload)
	if err := s.repo.UpdateActionAttempt(ctx, action); err != nil {
		return nil, err
	}
	incident.Status = domainsresmartbot.IncidentStatusContained
	incident.ContainedAt = &completedAt
	incident.LastObservedAt = completedAt
	incident.UpdatedAt = completedAt
	incident.Metadata = mustJSON(map[string]interface{}{
		"last_action_key": action.ActionKey,
		"last_action_at":  completedAt.Format(time.RFC3339),
	})
	if err := s.repo.UpdateIncident(ctx, incident); err != nil && s.logger != nil {
		s.logger.Warn("Failed to persist contained incident state after action execution", zap.Error(err))
	}
	return action, nil
}

func (s *ActionService) TriggerIncidentSummaryEmail(ctx context.Context, incidentID uuid.UUID, actorID string, actorTenantID *uuid.UUID) (*domainsresmartbot.ActionAttempt, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	action := &domainsresmartbot.ActionAttempt{
		ID:               uuid.New(),
		IncidentID:       incidentID,
		ActionKey:        "email_incident_summary",
		ActionClass:      "notify",
		TargetKind:       "incident",
		TargetRef:        incident.ID.String(),
		Status:           "running",
		ActorType:        "operator",
		ActorID:          strings.TrimSpace(actorID),
		ApprovalRequired: false,
		RequestedAt:      now,
		StartedAt:        &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateActionAttempt(ctx, action); err != nil {
		return nil, err
	}

	resultPayload, execErr := s.executeIncidentSummaryEmail(ctx, incident, actorTenantID)
	completedAt := time.Now().UTC()
	action.CompletedAt = &completedAt
	action.UpdatedAt = completedAt
	if execErr != nil {
		action.Status = "failed"
		action.ErrorMessage = execErr.Error()
		action.ResultPayload = mustJSON(map[string]interface{}{
			"error": execErr.Error(),
		})
		if updateErr := s.repo.UpdateActionAttempt(ctx, action); updateErr != nil && s.logger != nil {
			s.logger.Warn("Failed to persist failed incident summary email action", zap.Error(updateErr))
		}
		return nil, execErr
	}

	action.Status = "completed"
	action.ErrorMessage = ""
	action.ResultPayload = mustJSON(resultPayload)
	if err := s.repo.UpdateActionAttempt(ctx, action); err != nil {
		return nil, err
	}
	return action, nil
}

func (s *ActionService) executeTenantAssetReconcile(ctx context.Context) (map[string]interface{}, error) {
	if s.infra == nil {
		return nil, fmt.Errorf("infrastructure action service not configured")
	}
	providers, err := s.infra.ListProvidersAll(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	summaries := make([]map[string]interface{}, 0)
	totalApplied := 0
	totalFailed := 0
	for _, provider := range providers.Providers {
		summary, reconcileErr := s.infra.ReconcileStaleTenantNamespaces(ctx, provider.ID, nil)
		if reconcileErr != nil {
			totalFailed++
			summaries = append(summaries, map[string]interface{}{
				"provider_id": provider.ID.String(),
				"name":        provider.Name,
				"error":       reconcileErr.Error(),
			})
			continue
		}
		totalApplied += summary.Applied
		totalFailed += summary.Failed
		summaries = append(summaries, map[string]interface{}{
			"provider_id": provider.ID.String(),
			"name":        provider.Name,
			"targeted":    summary.Targeted,
			"applied":     summary.Applied,
			"failed":      summary.Failed,
			"skipped":     summary.Skipped,
			"mode":        summary.Mode,
		})
	}
	if totalFailed > 0 {
		return map[string]interface{}{
			"providers":      summaries,
			"total_applied":  totalApplied,
			"total_failed":   totalFailed,
			"executed_scope": "all_providers",
		}, fmt.Errorf("tenant asset reconcile completed with %d failed provider(s)", totalFailed)
	}
	return map[string]interface{}{
		"providers":      summaries,
		"total_applied":  totalApplied,
		"total_failed":   totalFailed,
		"executed_scope": "all_providers",
	}, nil
}

func (s *ActionService) executeProviderReadinessRefresh(ctx context.Context) (map[string]interface{}, error) {
	if s.infra == nil {
		return nil, fmt.Errorf("infrastructure action service not configured")
	}
	result, err := s.infra.RunProviderReadinessWatchTick(ctx, 200)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh provider readiness: %w", err)
	}
	if result == nil {
		return map[string]interface{}{
			"message": "provider readiness refresh completed with no result payload",
		}, nil
	}
	payload := map[string]interface{}{
		"total_providers": result.TotalProviders,
		"attempted":       result.Attempted,
		"succeeded":       result.Succeeded,
		"failed":          result.Failed,
		"skipped":         result.Skipped,
		"ready":           result.Ready,
		"not_ready":       result.NotReady,
	}
	if result.Failed > 0 || result.NotReady > 0 {
		return payload, fmt.Errorf("provider readiness still degraded: %d failed refreshes, %d providers not ready", result.Failed, result.NotReady)
	}
	return payload, nil
}

func (s *ActionService) executeIncidentSummaryEmail(ctx context.Context, incident *domainsresmartbot.Incident, actorTenantID *uuid.UUID) (map[string]interface{}, error) {
	if incident == nil {
		return nil, fmt.Errorf("incident is required")
	}
	if s.adminRecipients == nil || s.emailNotification == nil {
		return nil, fmt.Errorf("email summary services are not configured")
	}

	recipientUserIDs, err := s.resolveAdminRecipients(ctx, incident)
	if err != nil {
		return nil, err
	}
	if len(recipientUserIDs) == 0 {
		return nil, fmt.Errorf("no admin recipients available for incident summary")
	}
	emails, err := s.adminRecipients.ListUserEmailsByIDs(ctx, recipientUserIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve admin recipient emails: %w", err)
	}
	if len(emails) == 0 {
		return nil, fmt.Errorf("no active admin email addresses available for incident summary")
	}

	findings, _ := s.repo.ListFindingsByIncident(ctx, incident.ID)
	evidence, _ := s.repo.ListEvidenceByIncident(ctx, incident.ID)
	actions, _ := s.repo.ListActionAttemptsByIncident(ctx, incident.ID)
	approvals, _ := s.repo.ListApprovalsByIncident(ctx, incident.ID)

	subject := fmt.Sprintf("SRE Smart Bot incident summary: %s [%s/%s]", incident.DisplayName, incident.Severity, incident.Status)
	bodyText, bodyHTML := buildIncidentSummaryEmail(incident, findings, evidence, actions, approvals)
	routingTenantID := resolveEmailRoutingTenantID(incident, actorTenantID)

	sent := 0
	for _, email := range emails {
		if sendErr := s.emailNotification.SendEmail(ctx, routingTenantID, email, subject, bodyText, bodyHTML, 2); sendErr != nil {
			return map[string]interface{}{
				"recipient_count": len(emails),
				"sent_count":      sent,
				"last_error":      sendErr.Error(),
			}, fmt.Errorf("failed to send incident summary email: %w", sendErr)
		}
		sent++
	}

	return map[string]interface{}{
		"recipient_count": len(emails),
		"sent_count":      sent,
		"subject":         subject,
		"recipients":      emails,
		"routing_tenant":  routingTenantID.String(),
	}, nil
}

func resolveEmailRoutingTenantID(incident *domainsresmartbot.Incident, actorTenantID *uuid.UUID) uuid.UUID {
	if incident != nil && incident.TenantID != nil && *incident.TenantID != uuid.Nil {
		return *incident.TenantID
	}
	if actorTenantID != nil && *actorTenantID != uuid.Nil {
		return *actorTenantID
	}
	return uuid.Nil
}

func (s *ActionService) isActionApproved(ctx context.Context, incidentID uuid.UUID, actionAttemptID uuid.UUID) (bool, error) {
	approvals, err := s.repo.ListApprovalsByIncident(ctx, incidentID)
	if err != nil {
		return false, err
	}
	for _, approval := range approvals {
		if approval == nil || approval.ActionAttemptID == nil {
			continue
		}
		if *approval.ActionAttemptID == actionAttemptID && strings.EqualFold(approval.Status, "approved") {
			return true, nil
		}
	}
	return false, nil
}

func (s *ActionService) resolveAdminRecipients(ctx context.Context, incident *domainsresmartbot.Incident) ([]uuid.UUID, error) {
	if s.adminRecipients == nil {
		return nil, nil
	}
	if incident != nil && incident.TenantID != nil && *incident.TenantID != uuid.Nil {
		return s.adminRecipients.ListTenantAdminUserIDs(ctx, *incident.TenantID)
	}
	return s.adminRecipients.ListSystemAdminUserIDs(ctx)
}

func (s *ActionService) loadIncidentAndAction(ctx context.Context, incidentID uuid.UUID, actionAttemptID uuid.UUID) (*domainsresmartbot.Incident, *domainsresmartbot.ActionAttempt, error) {
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, nil, err
	}
	actions, err := s.repo.ListActionAttemptsByIncident(ctx, incidentID)
	if err != nil {
		return nil, nil, err
	}
	for _, action := range actions {
		if action != nil && action.ID == actionAttemptID {
			return incident, action, nil
		}
	}
	return nil, nil, ErrSREActionNotFound
}

func (s *ActionService) loadApprovalAndAction(ctx context.Context, incidentID uuid.UUID, approvalID uuid.UUID) (*domainsresmartbot.Approval, *domainsresmartbot.ActionAttempt, error) {
	approvals, err := s.repo.ListApprovalsByIncident(ctx, incidentID)
	if err != nil {
		return nil, nil, err
	}
	for _, approval := range approvals {
		if approval == nil || approval.ID != approvalID {
			continue
		}
		if approval.ActionAttemptID == nil {
			return approval, nil, nil
		}
		_, action, err := s.loadIncidentAndAction(ctx, incidentID, *approval.ActionAttemptID)
		return approval, action, err
	}
	return nil, nil, ErrSREApprovalNotFound
}

func requestMessageOrDefault(message string, action *domainsresmartbot.ActionAttempt) string {
	if strings.TrimSpace(message) != "" {
		return strings.TrimSpace(message)
	}
	if action == nil {
		return "SRE Smart Bot action approval requested"
	}
	return fmt.Sprintf("Approval requested for action %s on %s %s", action.ActionKey, action.TargetKind, action.TargetRef)
}

func isExecutableAction(actionKey string) bool {
	switch strings.TrimSpace(actionKey) {
	case "reconcile_tenant_assets":
		return true
	case "review_provider_connectivity":
		return true
	case "email_incident_summary":
		return true
	default:
		return false
	}
}

func tenantIDOrNil(tenantID *uuid.UUID) uuid.UUID {
	if tenantID == nil {
		return uuid.Nil
	}
	return *tenantID
}

func buildIncidentSummaryEmail(
	incident *domainsresmartbot.Incident,
	findings []*domainsresmartbot.Finding,
	evidence []*domainsresmartbot.Evidence,
	actions []*domainsresmartbot.ActionAttempt,
	approvals []*domainsresmartbot.Approval,
) (string, string) {
	topFindingTitles := make([]string, 0, 3)
	for _, finding := range findings {
		if finding == nil || strings.TrimSpace(finding.Title) == "" {
			continue
		}
		topFindingTitles = append(topFindingTitles, finding.Title)
		if len(topFindingTitles) == 3 {
			break
		}
	}
	pendingApprovals := 0
	executableActions := 0
	for _, approval := range approvals {
		if approval != nil && approval.DecidedAt == nil {
			pendingApprovals++
		}
	}
	for _, action := range actions {
		if action != nil && isExecutableAction(action.ActionKey) {
			executableActions++
		}
	}

	lines := []string{
		"SRE Smart Bot Incident Summary",
		"",
		fmt.Sprintf("Incident: %s", incident.DisplayName),
		fmt.Sprintf("Type: %s", incident.IncidentType),
		fmt.Sprintf("Domain: %s", incident.Domain),
		fmt.Sprintf("Severity / Status: %s / %s", incident.Severity, incident.Status),
		fmt.Sprintf("Source: %s", incident.Source),
		fmt.Sprintf("First observed: %s", incident.FirstObservedAt.Format(time.RFC1123)),
		fmt.Sprintf("Last observed: %s", incident.LastObservedAt.Format(time.RFC1123)),
		"",
		"Summary:",
		incident.Summary,
		"",
		fmt.Sprintf("Findings: %d", len(findings)),
		fmt.Sprintf("Evidence records: %d", len(evidence)),
		fmt.Sprintf("Action attempts: %d", len(actions)),
		fmt.Sprintf("Pending approvals: %d", pendingApprovals),
		fmt.Sprintf("Executable actions available: %d", executableActions),
	}
	if len(topFindingTitles) > 0 {
		lines = append(lines, "", "Top findings:")
		for _, title := range topFindingTitles {
			lines = append(lines, fmt.Sprintf("- %s", title))
		}
	}

	bodyText := strings.Join(lines, "\n")
	bodyHTML := fmt.Sprintf(
		"<h1>SRE Smart Bot Incident Summary</h1><p><strong>Incident:</strong> %s</p><p><strong>Type:</strong> %s<br/><strong>Domain:</strong> %s<br/><strong>Severity / Status:</strong> %s / %s<br/><strong>Source:</strong> %s</p><p><strong>Summary:</strong> %s</p><p><strong>Findings:</strong> %d<br/><strong>Evidence records:</strong> %d<br/><strong>Action attempts:</strong> %d<br/><strong>Pending approvals:</strong> %d<br/><strong>Executable actions available:</strong> %d</p>",
		escapeHTML(incident.DisplayName),
		escapeHTML(incident.IncidentType),
		escapeHTML(incident.Domain),
		escapeHTML(string(incident.Severity)),
		escapeHTML(string(incident.Status)),
		escapeHTML(incident.Source),
		escapeHTML(incident.Summary),
		len(findings),
		len(evidence),
		len(actions),
		pendingApprovals,
		executableActions,
	)
	if len(topFindingTitles) > 0 {
		bodyHTML += "<p><strong>Top findings:</strong></p><ul>"
		for _, title := range topFindingTitles {
			bodyHTML += "<li>" + escapeHTML(title) + "</li>"
		}
		bodyHTML += "</ul>"
	}
	return bodyText, bodyHTML
}

func escapeHTML(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(value)
}
