package sresmartbot

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

func TestObserveNATSConsumerLagSignals_NormalizesConsumerArtifacts(t *testing.T) {
	now := time.Date(2026, time.March, 15, 13, 0, 0, 0, time.UTC)
	tests := []struct {
		name                string
		snapshot            messaging.NATSConsumerLagSnapshot
		expectedKey         string
		expectedIncident    string
		expectedSignalType  string
		expectedEvidence    string
		expectedAction      string
		expectedDisplayName string
		expectedCount       int64
		expectedThreshold   int64
		expectedTargetRef   string
	}{
		{
			name: "consumer lag pressure",
			snapshot: messaging.NATSConsumerLagSnapshot{
				Stream:       "build-events",
				Consumer:     "dispatcher",
				PendingCount: 42,
			},
			expectedKey:         "golden_signal:nats_consumer:build-events:dispatcher:consumer_lag_pressure",
			expectedIncident:    "nats_consumer_lag_pressure",
			expectedSignalType:  "nats_consumer_lag_pressure",
			expectedEvidence:    "nats_consumer_lag_snapshot",
			expectedAction:      "review_nats_consumer_lag",
			expectedDisplayName: "NATS consumer lag pressure for build-events/dispatcher",
			expectedCount:       42,
			expectedThreshold:   25,
			expectedTargetRef:   "build-events/dispatcher",
		},
		{
			name: "pending ack pressure",
			snapshot: messaging.NATSConsumerLagSnapshot{
				Stream:          "notifications",
				Consumer:        "email-worker",
				PendingCount:    4,
				AckPendingCount: 12,
			},
			expectedKey:         "golden_signal:nats_consumer:notifications:email-worker:pending_ack_saturation",
			expectedIncident:    "nats_consumer_ack_pressure",
			expectedSignalType:  "nats_consumer_ack_pressure",
			expectedEvidence:    "nats_consumer_ack_pressure_snapshot",
			expectedAction:      "review_nats_consumer_progress",
			expectedDisplayName: "NATS consumer pending-ack pressure for notifications/email-worker",
			expectedCount:       12,
			expectedThreshold:   10,
			expectedTargetRef:   "notifications/email-worker",
		},
		{
			name: "stalled consumer progress",
			snapshot: messaging.NATSConsumerLagSnapshot{
				Stream:       "image-sync",
				Consumer:     "sync-worker",
				PendingCount: 9,
				LastActive:   now.Add(-8 * time.Minute),
			},
			expectedKey:         "golden_signal:nats_consumer:image-sync:sync-worker:stalled_consumer_progress",
			expectedIncident:    "nats_consumer_stalled_progress",
			expectedSignalType:  "nats_consumer_stalled_progress",
			expectedEvidence:    "nats_consumer_progress_snapshot",
			expectedAction:      "review_nats_consumer_progress",
			expectedDisplayName: "NATS consumer stalled progress for image-sync/sync-worker",
			expectedCount:       480,
			expectedThreshold:   300,
			expectedTargetRef:   "image-sync/sync-worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &sreRepoStub{}
			publisher := &srePublisherStub{}
			service := NewService(repo, publisher, zap.NewNop())

			current := ObserveNATSConsumerLagSignals(
				context.Background(),
				service,
				zap.NewNop(),
				[]messaging.NATSConsumerLagSnapshot{tt.snapshot},
				now,
				nil,
				NATSConsumerLagThresholds{
					PendingMessagesThreshold: 25,
					AckPendingThreshold:      10,
					StalledDuration:          5 * time.Minute,
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
			if repo.findings[0].SignalType != tt.expectedSignalType {
				t.Fatalf("expected signal type %q, got %q", tt.expectedSignalType, repo.findings[0].SignalType)
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
				t.Fatalf("expected threshold %d, got %d", tt.expectedThreshold, got)
			}
			if got := evidencePayload["trend"]; got != "elevated" {
				t.Fatalf("expected trend elevated, got %#v", got)
			}

			if len(repo.actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(repo.actions))
			}
			if repo.actions[0].ActionKey != tt.expectedAction {
				t.Fatalf("expected action key %q, got %q", tt.expectedAction, repo.actions[0].ActionKey)
			}
			if repo.actions[0].TargetRef != tt.expectedTargetRef {
				t.Fatalf("expected target ref %q, got %q", tt.expectedTargetRef, repo.actions[0].TargetRef)
			}

			var incidentMetadata map[string]any
			if err := json.Unmarshal(incident.Metadata, &incidentMetadata); err != nil {
				t.Fatalf("unmarshal incident metadata: %v", err)
			}
			if incidentMetadata["messaging_consumers_tool"] != "messaging_consumers.recent" {
				t.Fatalf("expected messaging consumer tool hint, got %#v", incidentMetadata["messaging_consumers_tool"])
			}
		})
	}
}

func TestObserveNATSConsumerLagSignals_EvidenceCapturesTrendFromPreviousObservation(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	service := NewService(repo, publisher, zap.NewNop())
	now := time.Date(2026, time.March, 15, 13, 10, 0, 0, time.UTC)

	ObserveNATSConsumerLagSignals(
		context.Background(),
		service,
		zap.NewNop(),
		[]messaging.NATSConsumerLagSnapshot{{
			Stream:       "build-events",
			Consumer:     "dispatcher",
			PendingCount: 36,
		}},
		now,
		map[string]NATSConsumerLagIssue{
			"build-events:dispatcher:consumer_lag_pressure": {
				Key:       "build-events:dispatcher:consumer_lag_pressure",
				Kind:      "consumer_lag_pressure",
				Count:     28,
				Threshold: 25,
			},
		},
		NATSConsumerLagThresholds{
			PendingMessagesThreshold: 25,
			AckPendingThreshold:      10,
			StalledDuration:          5 * time.Minute,
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
		t.Fatalf("expected trend growing, got %#v", got)
	}
	if got := int64(evidencePayload["previous_observation_count"].(float64)); got != 28 {
		t.Fatalf("expected previous observation count 28, got %d", got)
	}
	if got := int64(evidencePayload["count_delta"].(float64)); got != 8 {
		t.Fatalf("expected count delta 8, got %d", got)
	}
}

func TestObserveNATSConsumerLagSignals_ResolvesRecoveredIncident(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	service := NewService(repo, publisher, zap.NewNop())
	observedAt := time.Date(2026, time.March, 15, 13, 20, 0, 0, time.UTC)
	resolvedAt := observedAt.Add(2 * time.Minute)

	previous := ObserveNATSConsumerLagSignals(
		context.Background(),
		service,
		zap.NewNop(),
		[]messaging.NATSConsumerLagSnapshot{{
			Stream:       "notifications",
			Consumer:     "email-worker",
			PendingCount: 31,
		}},
		observedAt,
		nil,
		NATSConsumerLagThresholds{
			PendingMessagesThreshold: 25,
			AckPendingThreshold:      10,
			StalledDuration:          5 * time.Minute,
		},
	)

	current := ObserveNATSConsumerLagSignals(
		context.Background(),
		service,
		zap.NewNop(),
		nil,
		resolvedAt,
		previous,
		NATSConsumerLagThresholds{
			PendingMessagesThreshold: 25,
			AckPendingThreshold:      10,
			StalledDuration:          5 * time.Minute,
		},
	)

	if len(current) != 0 {
		t.Fatalf("expected no active issues after recovery, got %d", len(current))
	}
	incident := repo.incidentsByCorrelation["golden_signal:nats_consumer:notifications:email-worker:consumer_lag_pressure"]
	if incident == nil {
		t.Fatal("expected resolved incident thread for consumer lag")
	}
	if incident.Status != domainsresmartbot.IncidentStatusResolved {
		t.Fatalf("expected resolved status, got %q", incident.Status)
	}
	if incident.ResolvedAt == nil || !incident.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("expected resolved at %v, got %#v", resolvedAt, incident.ResolvedAt)
	}
}
