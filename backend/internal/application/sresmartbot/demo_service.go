package sresmartbot

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"go.uber.org/zap"
)

type DemoScenarioDescriptor struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Summary                string `json:"summary"`
	RecommendedWalkthrough string `json:"recommended_walkthrough"`
}

type DemoService struct {
	signals *Service
	repo    domainsresmartbot.Repository
	logger  *zap.Logger
}

func NewDemoService(signals *Service, repo domainsresmartbot.Repository, logger *zap.Logger) *DemoService {
	return &DemoService{signals: signals, repo: repo, logger: logger}
}

func (s *DemoService) ListScenarios() []DemoScenarioDescriptor {
	return []DemoScenarioDescriptor{
		{
			ID:                     "ldap_timeout",
			Name:                   "LDAP Login Timeout",
			Summary:                "Simulates identity-provider timeouts so the bot can explain auth impact, surface grounded evidence, and recommend safe next steps.",
			RecommendedWalkthrough: "Open the incident, review the executive summary, run grounded MCP tools, then generate the AI draft and local interpretation.",
		},
		{
			ID:                     "provider_connectivity",
			Name:                   "Provider Connectivity Degradation",
			Summary:                "Creates a provider-readiness incident with an executable review action so you can demo approval and safe action flow end to end.",
			RecommendedWalkthrough: "Open the incident, inspect evidence, request or approve the proposed action, then execute provider connectivity review.",
		},
		{
			ID:                     "release_drift",
			Name:                   "Release Drift And Partial Apply",
			Summary:                "Shows configuration drift with release-focused evidence so the AI layer can build a hypothesis and investigation plan from MCP tool output.",
			RecommendedWalkthrough: "Open the incident, inspect release evidence, run release MCP tools, and show the deterministic draft investigation plan.",
		},
	}
}

func (s *DemoService) GenerateIncident(ctx context.Context, tenantID *uuid.UUID, scenarioID string) (*domainsresmartbot.Incident, error) {
	if s == nil || s.signals == nil || s.repo == nil {
		return nil, fmt.Errorf("demo service is not configured")
	}

	now := time.Now().UTC()
	correlationKey := fmt.Sprintf("demo.%s.%d", scenarioID, now.UnixNano())

	switch scenarioID {
	case "ldap_timeout":
		if err := s.signals.RecordObservation(ctx, SignalObservation{
			TenantID:       tenantID,
			CorrelationKey: correlationKey,
			Domain:         "identity_security",
			IncidentType:   "identity_provider_unreachable",
			DisplayName:    "LDAP login requests timing out",
			Summary:        "Authentication requests are timing out against the configured LDAP provider, which is degrading operator and user sign-in flows.",
			Source:         "demo.generator",
			Severity:       domainsresmartbot.IncidentSeverityCritical,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"demo":             true,
				"scenario_id":      scenarioID,
				"affected_surface": "login",
				"provider_host":    "image-factory-glauth:3893",
			},
			FindingTitle:   "LDAP provider timed out during login checks",
			FindingMessage: "Recent auth attempts exceeded the expected timeout threshold while binding to the LDAP provider.",
			SignalType:     "ldap_timeout",
			SignalKey:      "image-factory-glauth",
			RawPayload: map[string]interface{}{
				"error":         "LDAP Result Code 200 \"Network Error\": context deadline exceeded",
				"timeout_ms":    5000,
				"provider_host": "image-factory-glauth:3893",
			},
		}); err != nil {
			return nil, err
		}
		if err := s.signals.AddEvidence(ctx, correlationKey, "log_excerpt", "Representative LDAP timeout evidence captured for demo walkthrough.", map[string]interface{}{
			"query":         `{app="backend"} |= "LDAP Result Code 200"`,
			"sample_lines":  []string{"LDAP Result Code 200 \"Network Error\": dial tcp 10.96.7.40:3893: i/o timeout"},
			"affected_path": "/api/v1/auth/login",
		}, now); err != nil {
			return nil, err
		}
		if err := s.signals.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:        "email_incident_summary",
			ActionClass:      "communication",
			TargetKind:       "incident",
			TargetRef:        "identity_security",
			Status:           "proposed",
			ActorType:        "system",
			ApprovalRequired: false,
			ResultPayload: map[string]interface{}{
				"reason": "Share a concise operator update while auth troubleshooting proceeds.",
			},
		}, now); err != nil {
			return nil, err
		}
	case "provider_connectivity":
		if err := s.signals.RecordObservation(ctx, SignalObservation{
			TenantID:       tenantID,
			CorrelationKey: correlationKey,
			Domain:         "runtime_services",
			IncidentType:   "provider_readiness_degraded",
			DisplayName:    "Provider connectivity checks are failing",
			Summary:        "One or more infrastructure providers are failing readiness checks, which can block reconciliation and runtime provisioning workflows.",
			Source:         "demo.generator",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"demo":              true,
				"scenario_id":       scenarioID,
				"failed_providers":  1,
				"provider_identity": "aws-prod",
			},
			FindingTitle:   "Provider readiness degraded for aws-prod",
			FindingMessage: "Connectivity checks against aws-prod are failing and the provider is currently marked not ready.",
			SignalType:     "provider_readiness",
			SignalKey:      "aws-prod",
			RawPayload: map[string]interface{}{
				"provider_name":    "aws-prod",
				"readiness_status": "not_ready",
				"last_error":       "failed to connect to provider API: dial tcp 10.0.0.15:443: i/o timeout",
				"failed_checks":    3,
			},
		}); err != nil {
			return nil, err
		}
		if err := s.signals.AddEvidence(ctx, correlationKey, "provider_readiness_snapshot", "Synthetic provider readiness snapshot for demo action workflow.", map[string]interface{}{
			"provider_name":          "aws-prod",
			"status":                 "not_ready",
			"failed_checks":          3,
			"successful_checks":      1,
			"recommended_action":     "review_provider_connectivity",
			"operator_talking_point": "Use this incident to demo approval plus bounded execution.",
		}, now); err != nil {
			return nil, err
		}
		if err := s.signals.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:        "review_provider_connectivity",
			ActionClass:      "investigation",
			TargetKind:       "provider",
			TargetRef:        "aws-prod",
			Status:           "proposed",
			ActorType:        "system",
			ApprovalRequired: true,
			ResultPayload: map[string]interface{}{
				"reason": "Run a safe connectivity refresh against the affected provider.",
			},
		}, now); err != nil {
			return nil, err
		}
	case "release_drift":
		if err := s.signals.RecordObservation(ctx, SignalObservation{
			TenantID:       tenantID,
			CorrelationKey: correlationKey,
			Domain:         "release_configuration",
			IncidentType:   "release_drift_or_partial_apply",
			DisplayName:    "Release drift detected after partial apply",
			Summary:        "A release drift condition was detected between intended and actual runtime state after a partial apply, leaving the environment in an inconsistent posture.",
			Source:         "demo.generator",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceMedium,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"demo":             true,
				"scenario_id":      scenarioID,
				"release_name":     "image-factory",
				"drifted_resource": "Deployment/image-factory-backend",
			},
			FindingTitle:   "Release state differs from expected runtime configuration",
			FindingMessage: "Observed resource state no longer matches the expected release plan for the backend deployment.",
			SignalType:     "release_drift",
			SignalKey:      "image-factory-backend",
			RawPayload: map[string]interface{}{
				"release_name":       "image-factory",
				"resource_kind":      "Deployment",
				"resource_name":      "image-factory-backend",
				"expected_image_tag": "v0.1.0-demo-expected",
				"actual_image_tag":   "v0.1.0-demo-running",
			},
		}); err != nil {
			return nil, err
		}
		if err := s.signals.AddEvidence(ctx, correlationKey, "release_drift_summary", "Synthetic release drift evidence for MCP and AI walkthroughs.", map[string]interface{}{
			"helm_release":          "image-factory",
			"expected_revision":     12,
			"observed_revision":     11,
			"drifted_resources":     []string{"Deployment/image-factory-backend", "ConfigMap/image-factory-config"},
			"suggested_mcp_tooling": []string{"release_drift.summary", "logs.recent"},
		}, now); err != nil {
			return nil, err
		}
		if err := s.signals.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:        "email_incident_summary",
			ActionClass:      "communication",
			TargetKind:       "release",
			TargetRef:        "image-factory",
			Status:           "proposed",
			ActorType:        "system",
			ApprovalRequired: false,
			ResultPayload: map[string]interface{}{
				"reason": "Share the drift summary with administrators before manual reconciliation.",
			},
		}, now); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported demo scenario: %s", scenarioID)
	}

	incident, err := s.repo.GetIncidentByCorrelationKey(ctx, correlationKey)
	if err != nil {
		return nil, fmt.Errorf("load generated incident: %w", err)
	}
	return incident, nil
}
