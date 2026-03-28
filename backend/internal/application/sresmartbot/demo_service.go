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
		{
			ID:                     "async_backlog_without_transport",
			Name:                   "Async Backlog Without Transport Instability",
			Summary:                "Creates localized dispatcher backlog pressure while transport remains stable so operators can distinguish worker congestion from message-bus instability.",
			RecommendedWalkthrough: "Open the incident, confirm the summary tab shows isolated backlog pressure, then generate the deterministic draft to show worker-throughput wording.",
		},
		{
			ID:                     "async_backlog_with_transport",
			Name:                   "Async Backlog With Transport Correlation",
			Summary:                "Creates messaging outbox backlog pressure with reconnect and disconnect evidence so operators can demo transport-driven async pressure.",
			RecommendedWalkthrough: "Open the incident, review backlog plus transport summaries, then generate the draft to show transport-related causality and bounded messaging review guidance.",
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
	case "async_backlog_without_transport":
		if err := s.signals.RecordObservation(ctx, SignalObservation{
			TenantID:       tenantID,
			CorrelationKey: correlationKey,
			Domain:         "golden_signals",
			IncidentType:   "dispatcher_backlog_pressure",
			DisplayName:    "Dispatcher backlog pressure without transport instability",
			Summary:        "Dispatcher backlog is elevated above threshold while messaging transport remains stable, which points to worker throughput pressure rather than bus instability.",
			Source:         "demo.generator",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"demo":                       true,
				"scenario_id":                scenarioID,
				"queue_kind":                 "build_queue",
				"subsystem":                  "dispatcher",
				"transport_correlation_hint": "messaging_transport:stable",
			},
			FindingTitle:   "Dispatcher backlog exceeded its configured threshold",
			FindingMessage: "Dispatcher backlog remains elevated without matching reconnect or disconnect pressure on the message bus.",
			SignalType:     "dispatcher_backlog_pressure",
			SignalKey:      "dispatcher.build_queue",
			RawPayload: map[string]interface{}{
				"count":                   16,
				"threshold":               8,
				"threshold_delta":         8,
				"threshold_ratio_percent": 200,
				"trend":                   "rising",
				"recent_observations": map[string]interface{}{
					"previous": 11,
					"current":  16,
				},
				"correlation_hints": map[string]interface{}{
					"messaging_transport": "stable",
				},
			},
		}); err != nil {
			return nil, err
		}
		if err := s.signals.AddEvidence(ctx, correlationKey, "dispatcher_backlog_snapshot", "Synthetic dispatcher backlog snapshot showing localized worker congestion without transport instability.", map[string]interface{}{
			"count":                   16,
			"threshold":               8,
			"threshold_delta":         8,
			"threshold_ratio_percent": 200,
			"trend":                   "rising",
			"queue_kind":              "build_queue",
			"subsystem":               "dispatcher",
			"operator_status":         "Dispatcher backlog is growing without current messaging transport instability.",
			"latest_summary":          "Dispatcher queue depth rose from 11 to 16 while reconnect and disconnect counts stayed at zero.",
			"recent_observations": map[string]interface{}{
				"previous": 11,
				"current":  16,
			},
			"correlation_hints": map[string]interface{}{
				"messaging_transport": "stable",
			},
		}, now); err != nil {
			return nil, err
		}
		if err := s.signals.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:        "review_dispatcher_backlog_pressure",
			ActionClass:      "recommendation",
			TargetKind:       "worker_domain",
			TargetRef:        "dispatcher",
			Status:           "proposed",
			ActorType:        "system",
			ApprovalRequired: false,
			ResultPayload: map[string]interface{}{
				"reason": "Review dispatcher worker throughput and downstream processing before making transport changes.",
			},
		}, now); err != nil {
			return nil, err
		}
	case "async_backlog_with_transport":
		if err := s.signals.RecordObservation(ctx, SignalObservation{
			TenantID:       tenantID,
			CorrelationKey: correlationKey,
			Domain:         "golden_signals",
			IncidentType:   "messaging_outbox_backlog_pressure",
			DisplayName:    "Messaging outbox backlog correlated with transport instability",
			Summary:        "Messaging outbox backlog is above threshold and recent reconnect pressure suggests the message bus is contributing to delayed delivery.",
			Source:         "demo.generator",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"demo":                       true,
				"scenario_id":                scenarioID,
				"queue_kind":                 "messaging_outbox",
				"subsystem":                  "messaging",
				"transport_correlation_hint": "messaging_transport:nats_transport_degraded",
			},
			FindingTitle:   "Messaging outbox backlog exceeded threshold alongside reconnect pressure",
			FindingMessage: "Outbox pending count is growing while the message bus is reconnecting repeatedly.",
			SignalType:     "messaging_outbox_backlog_pressure",
			SignalKey:      "messaging.outbox",
			RawPayload: map[string]interface{}{
				"count":                   21,
				"threshold":               9,
				"threshold_delta":         12,
				"threshold_ratio_percent": 233,
				"trend":                   "rising",
				"recent_observations": map[string]interface{}{
					"previous": 13,
					"current":  21,
				},
				"correlation_hints": map[string]interface{}{
					"messaging_transport": "reconnect_pressure",
				},
			},
		}); err != nil {
			return nil, err
		}
		if err := s.signals.AddEvidence(ctx, correlationKey, "messaging_outbox_backlog_snapshot", "Synthetic outbox backlog snapshot correlated with reconnect pressure on NATS.", map[string]interface{}{
			"count":                   21,
			"threshold":               9,
			"threshold_delta":         12,
			"threshold_ratio_percent": 233,
			"trend":                   "rising",
			"queue_kind":              "messaging_outbox",
			"subsystem":               "messaging",
			"operator_status":         "Messaging outbox backlog is growing while messaging transport is unstable.",
			"latest_summary":          "Outbox pending count rose from 13 to 21 after repeated reconnect and disconnect events on NATS.",
			"recent_observations": map[string]interface{}{
				"previous": 13,
				"current":  21,
			},
			"correlation_hints": map[string]interface{}{
				"messaging_transport": "reconnect_pressure",
			},
		}, now); err != nil {
			return nil, err
		}
		if err := s.signals.AddEvidence(ctx, correlationKey, "nats_transport_status", "Synthetic transport snapshot showing reconnect pressure during outbox backlog growth.", map[string]interface{}{
			"status":              "degraded",
			"reconnects":          6,
			"disconnects":         2,
			"reconnect_threshold": 3,
			"operator_status":     "Transport instability is likely contributing to outbox backlog pressure.",
			"latest_summary":      "Reconnect spikes occurred before the outbox crossed its threshold.",
		}, now); err != nil {
			return nil, err
		}
		if err := s.signals.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:        "review_messaging_transport_health",
			ActionClass:      "recommendation",
			TargetKind:       "message_bus",
			TargetRef:        "nats",
			Status:           "proposed",
			ActorType:        "system",
			ApprovalRequired: false,
			ResultPayload: map[string]interface{}{
				"reason": "Validate message-bus reconnect pressure before changing worker capacity or draining the backlog manually.",
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
