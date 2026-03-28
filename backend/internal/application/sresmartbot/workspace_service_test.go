package sresmartbot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
)

func TestBuildIncidentWorkspace_ProjectsAsyncPressureSummaryAndDefaultTools(t *testing.T) {
	incidentID := uuid.New()
	correlationKey := "golden_signal:backlog:email_queue_backlog"
	capturedAt := time.Date(2026, time.March, 15, 12, 10, 0, 0, time.UTC)
	repo := &sreRepoStub{
		incidentsByCorrelation: map[string]*domainsresmartbot.Incident{
			correlationKey: {
				ID:             incidentID,
				CorrelationKey: correlationKey,
				Domain:         "golden_signals",
				IncidentType:   "email_queue_backlog_pressure",
				DisplayName:    "Email queue backlog pressure",
				Status:         domainsresmartbot.IncidentStatusObserved,
				Severity:       domainsresmartbot.IncidentSeverityWarning,
			},
		},
		evidence: []*domainsresmartbot.Evidence{
			{
				ID:           uuid.New(),
				IncidentID:   incidentID,
				EvidenceType: "email_queue_backlog_snapshot",
				Summary:      "Email queue depth is 24, above threshold 20",
				Payload:      []byte(`{"count":24,"threshold":20,"threshold_delta":4,"threshold_ratio_percent":120,"trend":"growing","queue_kind":"email_queue","subsystem":"worker_pool","recent_observations":{"build_queue_depth":4,"email_queue_depth":24,"messaging_outbox_pending_count":6},"correlation_hints":{"transport_tool":"messaging_transport.recent","async_backlog_tool":"async_backlog.recent","transport_correlation_key":"messaging_transport:nats_transport_degraded"}}`),
				CapturedAt:   capturedAt,
				CreatedAt:    capturedAt,
			},
		},
	}

	service := NewWorkspaceService(repo, nil)
	workspace, err := service.BuildIncidentWorkspace(context.Background(), nil, incidentID)
	if err != nil {
		t.Fatalf("build workspace: %v", err)
	}
	if workspace.AsyncPressureSummary == nil || workspace.AsyncPressureSummary.Backlog == nil {
		t.Fatal("expected async backlog workspace summary")
	}
	backlog := workspace.AsyncPressureSummary.Backlog
	if backlog.QueueKind != "email_queue" {
		t.Fatalf("expected queue kind %q, got %q", "email_queue", backlog.QueueKind)
	}
	if backlog.OperatorStatus != "backlog growing" {
		t.Fatalf("expected operator status %q, got %q", "backlog growing", backlog.OperatorStatus)
	}
	if backlog.ThresholdDelta != 4 {
		t.Fatalf("expected threshold delta %d, got %d", 4, backlog.ThresholdDelta)
	}
	if len(backlog.CorrelationHints) == 0 || backlog.CorrelationHints["transport_tool"] != "messaging_transport.recent" {
		t.Fatal("expected transport tool correlation hint in backlog summary")
	}
	if !containsString(workspace.DefaultToolBundle, "async_backlog.recent") {
		t.Fatal("expected async_backlog.recent in default tool bundle")
	}
	if !containsString(workspace.DefaultToolBundle, "messaging_transport.recent") {
		t.Fatal("expected messaging_transport.recent in default tool bundle")
	}
	if !containsString(workspace.DefaultToolBundle, "evidence.list") {
		t.Fatal("expected evidence.list in default tool bundle")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
