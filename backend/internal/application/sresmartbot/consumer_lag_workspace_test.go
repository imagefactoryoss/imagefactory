package sresmartbot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
)

func TestBuildIncidentWorkspace_IncludesMessagingConsumerSummaryAndBundle(t *testing.T) {
	incidentID := uuid.New()
	now := time.Date(2026, time.March, 15, 13, 30, 0, 0, time.UTC)
	repo := &workspaceRepoStub{
		incident: &domainsresmartbot.Incident{
			ID:           incidentID,
			Domain:       "golden_signals",
			IncidentType: "nats_consumer_lag_pressure",
			DisplayName:  "NATS consumer lag pressure for build-events/dispatcher",
			Status:       domainsresmartbot.IncidentStatusObserved,
			Severity:     domainsresmartbot.IncidentSeverityWarning,
		},
		evidence: []*domainsresmartbot.Evidence{
			{
				IncidentID:   incidentID,
				EvidenceType: "nats_consumer_lag_snapshot",
				Summary:      "NATS consumer build-events/dispatcher has 42 pending messages, above threshold 25",
				Payload:      []byte(`{"kind":"consumer_lag_pressure","stream":"build-events","consumer":"dispatcher","target_ref":"build-events/dispatcher","count":42,"threshold":25,"threshold_delta":17,"threshold_ratio_percent":168,"pending_count":42,"ack_pending_count":4,"waiting_count":1,"redelivered_count":0,"trend":"growing","last_active":"2026-03-15T13:24:00Z","correlation_hints":{"messaging_consumers_tool":"messaging_consumers.recent","transport_tool":"messaging_transport.recent","async_backlog_tool":"async_backlog.recent"}}`),
				CapturedAt:   now,
			},
		},
	}

	service := NewWorkspaceService(repo, nil)
	workspace, err := service.BuildIncidentWorkspace(context.Background(), nil, incidentID)
	if err != nil {
		t.Fatalf("expected workspace build to succeed, got %v", err)
	}
	if workspace.AsyncPressureSummary == nil || workspace.AsyncPressureSummary.MessagingConsumer == nil {
		t.Fatal("expected messaging consumer workspace summary")
	}
	consumer := workspace.AsyncPressureSummary.MessagingConsumer
	if consumer.Stream != "build-events" || consumer.Consumer != "dispatcher" {
		t.Fatalf("unexpected consumer summary: %+v", consumer)
	}
	if consumer.OperatorStatus != "consumer lag growing" {
		t.Fatalf("expected operator status consumer lag growing, got %q", consumer.OperatorStatus)
	}
	if !containsString(workspace.DefaultToolBundle, "messaging_consumers.recent") {
		t.Fatal("expected messaging_consumers.recent in default tool bundle")
	}
	if !containsString(workspace.DefaultToolBundle, "messaging_transport.recent") {
		t.Fatal("expected messaging_transport.recent in default tool bundle")
	}
	if !containsString(workspace.DefaultToolBundle, "async_backlog.recent") {
		t.Fatal("expected async_backlog.recent in default tool bundle")
	}
}

type workspaceRepoStub struct {
	incident *domainsresmartbot.Incident
	evidence []*domainsresmartbot.Evidence
}

func (s *workspaceRepoStub) CreateIncident(ctx context.Context, incident *domainsresmartbot.Incident) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) UpdateIncident(ctx context.Context, incident *domainsresmartbot.Incident) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) GetIncident(ctx context.Context, id uuid.UUID) (*domainsresmartbot.Incident, error) {
	return s.incident, nil
}

func (s *workspaceRepoStub) GetIncidentByCorrelationKey(ctx context.Context, correlationKey string) (*domainsresmartbot.Incident, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListIncidents(ctx context.Context, filter domainsresmartbot.IncidentFilter) ([]*domainsresmartbot.Incident, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) CountIncidents(ctx context.Context, filter domainsresmartbot.IncidentFilter) (int, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) CreateFinding(ctx context.Context, finding *domainsresmartbot.Finding) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListFindingsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.Finding, error) {
	return nil, nil
}

func (s *workspaceRepoStub) AddEvidence(ctx context.Context, evidence *domainsresmartbot.Evidence) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListEvidenceByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.Evidence, error) {
	return s.evidence, nil
}

func (s *workspaceRepoStub) CreateActionAttempt(ctx context.Context, attempt *domainsresmartbot.ActionAttempt) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) UpdateActionAttempt(ctx context.Context, attempt *domainsresmartbot.ActionAttempt) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListActionAttemptsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.ActionAttempt, error) {
	return nil, nil
}

func (s *workspaceRepoStub) CreateApproval(ctx context.Context, approval *domainsresmartbot.Approval) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) UpdateApproval(ctx context.Context, approval *domainsresmartbot.Approval) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListApprovalsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.Approval, error) {
	return nil, nil
}

func (s *workspaceRepoStub) ListApprovals(ctx context.Context, filter domainsresmartbot.ApprovalFilter) ([]*domainsresmartbot.Approval, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) CountApprovals(ctx context.Context, filter domainsresmartbot.ApprovalFilter) (int, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) CreateDetectorRuleSuggestion(ctx context.Context, suggestion *domainsresmartbot.DetectorRuleSuggestion) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) UpdateDetectorRuleSuggestion(ctx context.Context, suggestion *domainsresmartbot.DetectorRuleSuggestion) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) GetDetectorRuleSuggestion(ctx context.Context, id uuid.UUID) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) GetDetectorRuleSuggestionByFingerprint(ctx context.Context, fingerprint string) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListDetectorRuleSuggestions(ctx context.Context, filter domainsresmartbot.DetectorRuleSuggestionFilter) ([]*domainsresmartbot.DetectorRuleSuggestion, error) {
	panic("unexpected call")
}

func (s *workspaceRepoStub) CreateRemediationPackRun(ctx context.Context, run *domainsresmartbot.RemediationPackRun) error {
	panic("unexpected call")
}

func (s *workspaceRepoStub) ListRemediationPackRunsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.RemediationPackRun, error) {
	panic("unexpected call")
}
