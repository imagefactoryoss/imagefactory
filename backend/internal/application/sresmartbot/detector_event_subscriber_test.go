package sresmartbot

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type sreRepoStub struct {
	incidentsByCorrelation map[string]*domainsresmartbot.Incident
	findings               []*domainsresmartbot.Finding
	evidence               []*domainsresmartbot.Evidence
	actions                []*domainsresmartbot.ActionAttempt
	suggestions            []*domainsresmartbot.DetectorRuleSuggestion
}

func (s *sreRepoStub) CreateIncident(ctx context.Context, incident *domainsresmartbot.Incident) error {
	if s.incidentsByCorrelation == nil {
		s.incidentsByCorrelation = make(map[string]*domainsresmartbot.Incident)
	}
	copy := *incident
	s.incidentsByCorrelation[incident.CorrelationKey] = &copy
	return nil
}

func (s *sreRepoStub) UpdateIncident(ctx context.Context, incident *domainsresmartbot.Incident) error {
	copy := *incident
	s.incidentsByCorrelation[incident.CorrelationKey] = &copy
	return nil
}

func (s *sreRepoStub) GetIncident(ctx context.Context, id uuid.UUID) (*domainsresmartbot.Incident, error) {
	for _, incident := range s.incidentsByCorrelation {
		if incident.ID == id {
			copy := *incident
			return &copy, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *sreRepoStub) GetIncidentByCorrelationKey(ctx context.Context, correlationKey string) (*domainsresmartbot.Incident, error) {
	incident, ok := s.incidentsByCorrelation[correlationKey]
	if !ok {
		return nil, sql.ErrNoRows
	}
	copy := *incident
	return &copy, nil
}

func (s *sreRepoStub) ListIncidents(ctx context.Context, filter domainsresmartbot.IncidentFilter) ([]*domainsresmartbot.Incident, error) {
	return nil, nil
}

func (s *sreRepoStub) CountIncidents(ctx context.Context, filter domainsresmartbot.IncidentFilter) (int, error) {
	return 0, nil
}

func (s *sreRepoStub) CreateFinding(ctx context.Context, finding *domainsresmartbot.Finding) error {
	copy := *finding
	s.findings = append(s.findings, &copy)
	return nil
}

func (s *sreRepoStub) ListFindingsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.Finding, error) {
	return nil, nil
}

func (s *sreRepoStub) AddEvidence(ctx context.Context, evidence *domainsresmartbot.Evidence) error {
	copy := *evidence
	s.evidence = append(s.evidence, &copy)
	return nil
}

func (s *sreRepoStub) ListEvidenceByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.Evidence, error) {
	return nil, nil
}

func (s *sreRepoStub) CreateActionAttempt(ctx context.Context, attempt *domainsresmartbot.ActionAttempt) error {
	copy := *attempt
	s.actions = append(s.actions, &copy)
	return nil
}

func (s *sreRepoStub) UpdateActionAttempt(ctx context.Context, attempt *domainsresmartbot.ActionAttempt) error {
	return nil
}

func (s *sreRepoStub) ListActionAttemptsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.ActionAttempt, error) {
	return nil, nil
}

func (s *sreRepoStub) CreateApproval(ctx context.Context, approval *domainsresmartbot.Approval) error {
	return nil
}

func (s *sreRepoStub) UpdateApproval(ctx context.Context, approval *domainsresmartbot.Approval) error {
	return nil
}

func (s *sreRepoStub) ListApprovalsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*domainsresmartbot.Approval, error) {
	return nil, nil
}

func (s *sreRepoStub) ListApprovals(ctx context.Context, filter domainsresmartbot.ApprovalFilter) ([]*domainsresmartbot.Approval, error) {
	return nil, nil
}

func (s *sreRepoStub) CountApprovals(ctx context.Context, filter domainsresmartbot.ApprovalFilter) (int, error) {
	return 0, nil
}

func (s *sreRepoStub) CreateDetectorRuleSuggestion(ctx context.Context, suggestion *domainsresmartbot.DetectorRuleSuggestion) error {
	copy := *suggestion
	s.suggestions = append(s.suggestions, &copy)
	return nil
}

func (s *sreRepoStub) UpdateDetectorRuleSuggestion(ctx context.Context, suggestion *domainsresmartbot.DetectorRuleSuggestion) error {
	return nil
}

func (s *sreRepoStub) GetDetectorRuleSuggestion(ctx context.Context, id uuid.UUID) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	for _, suggestion := range s.suggestions {
		if suggestion.ID == id {
			copy := *suggestion
			return &copy, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *sreRepoStub) GetDetectorRuleSuggestionByFingerprint(ctx context.Context, fingerprint string) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	for _, suggestion := range s.suggestions {
		if suggestion.Fingerprint == fingerprint {
			copy := *suggestion
			return &copy, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *sreRepoStub) ListDetectorRuleSuggestions(ctx context.Context, filter domainsresmartbot.DetectorRuleSuggestionFilter) ([]*domainsresmartbot.DetectorRuleSuggestion, error) {
	return s.suggestions, nil
}

type srePublisherStub struct {
	findingEvents  int
	evidenceEvents int
}

func (s *srePublisherStub) PublishFindingObserved(ctx context.Context, incident *domainsresmartbot.Incident, finding *domainsresmartbot.Finding) error {
	s.findingEvents++
	return nil
}

func (s *srePublisherStub) PublishIncidentResolved(ctx context.Context, incident *domainsresmartbot.Incident) error {
	return nil
}

func (s *srePublisherStub) PublishEvidenceAdded(ctx context.Context, incident *domainsresmartbot.Incident, evidence *domainsresmartbot.Evidence) error {
	s.evidenceEvents++
	return nil
}

func (s *srePublisherStub) PublishActionProposed(ctx context.Context, incident *domainsresmartbot.Incident, attempt *domainsresmartbot.ActionAttempt) error {
	return nil
}

func TestDetectorEventSubscriber_RecordsObservationAndEvidence(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	service := NewService(repo, publisher, zap.NewNop())
	subscriber := NewDetectorEventSubscriber(service, zap.NewNop())

	observedAt := time.Now().UTC().Truncate(time.Second)
	subscriber.HandleFindingEvent(context.Background(), messaging.Event{
		Type:       messaging.EventTypeSREDetectorFindingObserved,
		OccurredAt: observedAt,
		Payload: map[string]interface{}{
			"correlation_key": "logs:image_pull_rate_limit",
			"domain":          "runtime_services",
			"incident_type":   "registry_pull_failure",
			"display_name":    "Image pull rate limit detected",
			"summary":         "Repeated toomanyrequests responses were detected",
			"source":          "log_detector",
			"severity":        "warning",
			"confidence":      "high",
			"finding_title":   "Docker Hub rate limit signature matched",
			"finding_message": "Matched repeated toomanyrequests log lines in image-factory namespace",
			"signal_type":     "log_signature",
			"signal_key":      "toomanyrequests",
			"metadata": map[string]interface{}{
				"detector_name": "runtime-log-detector",
			},
			"raw_payload": map[string]interface{}{
				"query": "{namespace=\"image-factory\"} |= \"toomanyrequests\"",
			},
			"evidence_type":    "loki_log_match_window",
			"evidence_summary": "3 matching log lines in the last 5 minutes",
			"evidence_payload": map[string]interface{}{"match_count": 3},
		},
	})

	if len(repo.incidentsByCorrelation) != 1 {
		t.Fatalf("expected one incident, got %d", len(repo.incidentsByCorrelation))
	}
	incident, ok := repo.incidentsByCorrelation["logs:image_pull_rate_limit"]
	if !ok {
		t.Fatalf("expected incident to be stored by correlation key")
	}
	if incident.Source != "log_detector" {
		t.Fatalf("expected source log_detector, got %s", incident.Source)
	}
	if len(repo.findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(repo.findings))
	}
	if repo.findings[0].SignalType != "log_signature" {
		t.Fatalf("expected signal type log_signature, got %s", repo.findings[0].SignalType)
	}
	if len(repo.evidence) != 1 {
		t.Fatalf("expected one evidence row, got %d", len(repo.evidence))
	}
	if repo.evidence[0].EvidenceType != "loki_log_match_window" {
		t.Fatalf("expected evidence type loki_log_match_window, got %s", repo.evidence[0].EvidenceType)
	}
	if publisher.findingEvents != 1 || publisher.evidenceEvents != 1 {
		t.Fatalf("expected finding and evidence events to publish once, got finding=%d evidence=%d", publisher.findingEvents, publisher.evidenceEvents)
	}
}

func TestMapDetectorEvent_ParsesJSONPayloadMaps(t *testing.T) {
	metadata, _ := json.Marshal(map[string]interface{}{"detector_name": "loki-rules"})
	rawPayload, _ := json.Marshal(map[string]interface{}{"namespace": "image-factory"})

	observation, evidenceType, _, _, ok := mapDetectorEvent(messaging.Event{
		Type: messaging.EventTypeSREDetectorFindingObserved,
		Payload: map[string]interface{}{
			"correlation_key": "logs:nats_disconnect",
			"domain":          "runtime_services",
			"incident_type":   "runtime_dependency_outage",
			"summary":         "NATS connection failures detected",
			"source":          "detector",
			"severity":        "critical",
			"confidence":      "low",
			"finding_title":   "NATS disconnect pattern",
			"finding_message": "Matched no servers available for connection",
			"signal_type":     "log_signature",
			"signal_key":      "nats_disconnect",
			"metadata":        string(metadata),
			"raw_payload":     json.RawMessage(rawPayload),
		},
	})
	if !ok {
		t.Fatal("expected detector event to parse")
	}
	if observation.Severity != domainsresmartbot.IncidentSeverityCritical {
		t.Fatalf("expected critical severity, got %s", observation.Severity)
	}
	if observation.Confidence != domainsresmartbot.IncidentConfidenceLow {
		t.Fatalf("expected low confidence, got %s", observation.Confidence)
	}
	if observation.Metadata["detector_name"] != "loki-rules" {
		t.Fatalf("expected parsed metadata detector_name")
	}
	if observation.RawPayload["namespace"] != "image-factory" {
		t.Fatalf("expected parsed raw payload namespace")
	}
	if evidenceType != "" {
		t.Fatalf("expected no evidence type, got %s", evidenceType)
	}
}
