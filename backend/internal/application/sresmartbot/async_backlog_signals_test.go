package sresmartbot

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"go.uber.org/zap"
)

func TestObserveAsyncBacklogSignals_NormalizesPerQueueArtifacts(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name                string
		snapshot            AsyncBacklogSignalSnapshot
		expectedKey         string
		expectedIncident    string
		expectedSignalType  string
		expectedEvidence    string
		expectedAction      string
		expectedTargetKind  string
		expectedTargetRef   string
		expectedDisplayName string
		expectedCount       int64
		expectedThreshold   int64
	}{
		{
			name: "email queue backlog",
			snapshot: AsyncBacklogSignalSnapshot{
				EmailQueueDepth: 24,
			},
			expectedKey:         "golden_signal:backlog:email_queue_backlog",
			expectedIncident:    "email_queue_backlog_pressure",
			expectedSignalType:  "email_queue_backlog_pressure",
			expectedEvidence:    "email_queue_backlog_snapshot",
			expectedAction:      "review_async_worker_capacity",
			expectedTargetKind:  "worker_pool",
			expectedTargetRef:   "email_queue",
			expectedDisplayName: "Email queue backlog pressure",
			expectedCount:       24,
			expectedThreshold:   20,
		},
		{
			name: "messaging outbox backlog",
			snapshot: AsyncBacklogSignalSnapshot{
				MessagingOutboxPending: 18,
			},
			expectedKey:         "golden_signal:backlog:messaging_outbox_backlog",
			expectedIncident:    "messaging_outbox_backlog_pressure",
			expectedSignalType:  "messaging_outbox_backlog_pressure",
			expectedEvidence:    "messaging_outbox_backlog_snapshot",
			expectedAction:      "review_messaging_transport_health",
			expectedTargetKind:  "message_bus",
			expectedTargetRef:   "messaging_outbox",
			expectedDisplayName: "Messaging outbox backlog pressure",
			expectedCount:       18,
			expectedThreshold:   15,
		},
		{
			name: "dispatcher workflow backlog",
			snapshot: AsyncBacklogSignalSnapshot{
				BuildQueueDepth: 14,
			},
			expectedKey:         "golden_signal:backlog:build_queue_backlog",
			expectedIncident:    "dispatcher_backlog_pressure",
			expectedSignalType:  "dispatcher_backlog_pressure",
			expectedEvidence:    "dispatcher_backlog_snapshot",
			expectedAction:      "review_dispatcher_backlog_pressure",
			expectedTargetKind:  "async_pipeline",
			expectedTargetRef:   "dispatcher_workflow",
			expectedDisplayName: "Dispatcher/workflow backlog pressure",
			expectedCount:       14,
			expectedThreshold:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &sreRepoStub{}
			publisher := &srePublisherStub{}
			service := NewService(repo, publisher, zap.NewNop())

			current := ObserveAsyncBacklogSignals(
				context.Background(),
				service,
				zap.NewNop(),
				tt.snapshot,
				now,
				nil,
				AsyncBacklogThresholds{
					BuildQueueThreshold:      10,
					EmailQueueThreshold:      20,
					MessagingOutboxThreshold: 15,
				},
			)

			if len(current) != 1 {
				t.Fatalf("expected 1 current issue, got %d", len(current))
			}

			incident, ok := repo.incidentsByCorrelation[tt.expectedKey]
			if !ok {
				t.Fatalf("expected incident for correlation key %q", tt.expectedKey)
			}
			if incident.IncidentType != tt.expectedIncident {
				t.Fatalf("expected incident type %q, got %q", tt.expectedIncident, incident.IncidentType)
			}
			if incident.DisplayName != tt.expectedDisplayName {
				t.Fatalf("expected display name %q, got %q", tt.expectedDisplayName, incident.DisplayName)
			}

			if len(repo.findings) != 1 {
				t.Fatalf("expected 1 finding, got %d", len(repo.findings))
			}
			finding := repo.findings[0]
			if finding.SignalType != tt.expectedSignalType {
				t.Fatalf("expected signal type %q, got %q", tt.expectedSignalType, finding.SignalType)
			}
			if finding.Severity != domainsresmartbot.IncidentSeverityWarning {
				t.Fatalf("expected warning severity, got %q", finding.Severity)
			}

			if len(repo.evidence) != 1 {
				t.Fatalf("expected 1 evidence record, got %d", len(repo.evidence))
			}
			evidence := repo.evidence[0]
			if evidence.EvidenceType != tt.expectedEvidence {
				t.Fatalf("expected evidence type %q, got %q", tt.expectedEvidence, evidence.EvidenceType)
			}
			var evidencePayload map[string]any
			if err := json.Unmarshal(evidence.Payload, &evidencePayload); err != nil {
				t.Fatalf("unmarshal evidence payload: %v", err)
			}
			if got := int64(evidencePayload["count"].(float64)); got != tt.expectedCount {
				t.Fatalf("expected evidence count %d, got %d", tt.expectedCount, got)
			}
			if got := int64(evidencePayload["threshold"].(float64)); got != tt.expectedThreshold {
				t.Fatalf("expected evidence threshold %d, got %d", tt.expectedThreshold, got)
			}
			if got := evidencePayload["trend"]; got != "elevated" {
				t.Fatalf("expected trend %q, got %#v", "elevated", got)
			}
			correlationHints, ok := evidencePayload["correlation_hints"].(map[string]any)
			if !ok {
				t.Fatal("expected evidence correlation_hints map")
			}
			if correlationHints["transport_tool"] != "messaging_transport.recent" {
				t.Fatalf("expected transport tool hint, got %#v", correlationHints["transport_tool"])
			}
			recentObservations, ok := evidencePayload["recent_observations"].(map[string]any)
			if !ok {
				t.Fatal("expected evidence recent_observations map")
			}
			if _, exists := recentObservations["build_queue_depth"]; !exists {
				t.Fatal("expected build_queue_depth in recent observations")
			}

			if len(repo.actions) != 1 {
				t.Fatalf("expected 1 action attempt, got %d", len(repo.actions))
			}
			action := repo.actions[0]
			if action.ActionKey != tt.expectedAction {
				t.Fatalf("expected action key %q, got %q", tt.expectedAction, action.ActionKey)
			}
			if action.TargetKind != tt.expectedTargetKind {
				t.Fatalf("expected target kind %q, got %q", tt.expectedTargetKind, action.TargetKind)
			}
			if action.TargetRef != tt.expectedTargetRef {
				t.Fatalf("expected target ref %q, got %q", tt.expectedTargetRef, action.TargetRef)
			}

			var incidentMetadata map[string]any
			if err := json.Unmarshal(incident.Metadata, &incidentMetadata); err != nil {
				t.Fatalf("unmarshal incident metadata: %v", err)
			}
			if incidentMetadata["kind"] == nil {
				t.Fatal("expected incident metadata to include issue kind")
			}
		})
	}
}

func TestObserveAsyncBacklogSignals_EvidenceCapturesTrendFromPreviousObservation(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	service := NewService(repo, publisher, zap.NewNop())
	now := time.Date(2026, time.March, 15, 12, 5, 0, 0, time.UTC)

	ObserveAsyncBacklogSignals(
		context.Background(),
		service,
		zap.NewNop(),
		AsyncBacklogSignalSnapshot{EmailQueueDepth: 28, BuildQueueDepth: 3, MessagingOutboxPending: 4},
		now,
		map[string]AsyncBacklogIssue{
			"email_queue_backlog": {
				Key:       "email_queue_backlog",
				Kind:      "email_queue_backlog",
				Count:     22,
				Threshold: 20,
			},
		},
		AsyncBacklogThresholds{
			BuildQueueThreshold:      10,
			EmailQueueThreshold:      20,
			MessagingOutboxThreshold: 15,
		},
	)

	if len(repo.evidence) != 1 {
		t.Fatalf("expected 1 evidence record, got %d", len(repo.evidence))
	}

	var evidencePayload map[string]any
	if err := json.Unmarshal(repo.evidence[0].Payload, &evidencePayload); err != nil {
		t.Fatalf("unmarshal evidence payload: %v", err)
	}
	if got := evidencePayload["trend"]; got != "growing" {
		t.Fatalf("expected trend %q, got %#v", "growing", got)
	}
	if got := int64(evidencePayload["previous_observation_count"].(float64)); got != 22 {
		t.Fatalf("expected previous observation count %d, got %d", 22, got)
	}
	if got := int64(evidencePayload["count_delta"].(float64)); got != 6 {
		t.Fatalf("expected count delta %d, got %d", 6, got)
	}
}

func TestObserveAsyncBacklogSignals_ReusesStableIncidentThread(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	service := NewService(repo, publisher, zap.NewNop())
	firstObservedAt := time.Date(2026, time.March, 15, 12, 15, 0, 0, time.UTC)
	secondObservedAt := firstObservedAt.Add(2 * time.Minute)

	current := ObserveAsyncBacklogSignals(
		context.Background(),
		service,
		zap.NewNop(),
		AsyncBacklogSignalSnapshot{EmailQueueDepth: 21},
		firstObservedAt,
		nil,
		AsyncBacklogThresholds{
			BuildQueueThreshold:      10,
			EmailQueueThreshold:      20,
			MessagingOutboxThreshold: 15,
		},
	)

	ObserveAsyncBacklogSignals(
		context.Background(),
		service,
		zap.NewNop(),
		AsyncBacklogSignalSnapshot{EmailQueueDepth: 29},
		secondObservedAt,
		current,
		AsyncBacklogThresholds{
			BuildQueueThreshold:      10,
			EmailQueueThreshold:      20,
			MessagingOutboxThreshold: 15,
		},
	)

	if got := len(repo.incidentsByCorrelation); got != 1 {
		t.Fatalf("expected 1 incident thread, got %d", got)
	}
	incident := repo.incidentsByCorrelation["golden_signal:backlog:email_queue_backlog"]
	if incident == nil {
		t.Fatal("expected incident thread for email queue backlog")
	}
	if !incident.LastObservedAt.Equal(secondObservedAt) {
		t.Fatalf("expected last observed at %v, got %v", secondObservedAt, incident.LastObservedAt)
	}
	if incident.Status != domainsresmartbot.IncidentStatusObserved {
		t.Fatalf("expected observed status, got %q", incident.Status)
	}
	if got := len(repo.findings); got != 2 {
		t.Fatalf("expected 2 findings for repeated observations, got %d", got)
	}
	if got := len(repo.actions); got != 1 {
		t.Fatalf("expected 1 action attempt for stable incident thread, got %d", got)
	}
	if got := len(repo.evidence); got != 2 {
		t.Fatalf("expected 2 evidence records, got %d", got)
	}
}

func TestObserveAsyncBacklogSignals_ResolvesRecoveredIncident(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	service := NewService(repo, publisher, zap.NewNop())
	observedAt := time.Date(2026, time.March, 15, 12, 20, 0, 0, time.UTC)
	resolvedAt := observedAt.Add(3 * time.Minute)

	previous := ObserveAsyncBacklogSignals(
		context.Background(),
		service,
		zap.NewNop(),
		AsyncBacklogSignalSnapshot{MessagingOutboxPending: 17},
		observedAt,
		nil,
		AsyncBacklogThresholds{
			BuildQueueThreshold:      10,
			EmailQueueThreshold:      20,
			MessagingOutboxThreshold: 15,
		},
	)

	current := ObserveAsyncBacklogSignals(
		context.Background(),
		service,
		zap.NewNop(),
		AsyncBacklogSignalSnapshot{},
		resolvedAt,
		previous,
		AsyncBacklogThresholds{
			BuildQueueThreshold:      10,
			EmailQueueThreshold:      20,
			MessagingOutboxThreshold: 15,
		},
	)

	if len(current) != 0 {
		t.Fatalf("expected no active issues after recovery, got %d", len(current))
	}
	incident := repo.incidentsByCorrelation["golden_signal:backlog:messaging_outbox_backlog"]
	if incident == nil {
		t.Fatal("expected resolved incident thread for messaging outbox backlog")
	}
	if incident.Status != domainsresmartbot.IncidentStatusResolved {
		t.Fatalf("expected resolved status, got %q", incident.Status)
	}
	if incident.ResolvedAt == nil || !incident.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("expected resolved at %v, got %#v", resolvedAt, incident.ResolvedAt)
	}
	if incident.Summary != "Async backlog recovered for messaging_outbox_backlog" {
		t.Fatalf("unexpected resolution summary %q", incident.Summary)
	}
}
