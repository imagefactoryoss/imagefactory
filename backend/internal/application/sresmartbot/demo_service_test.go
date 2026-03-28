package sresmartbot

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestDemoService_ListScenarios_IncludesAsyncPressureDemos(t *testing.T) {
	service := NewDemoService(nil, nil, zap.NewNop())
	scenarios := service.ListScenarios()

	foundWithoutTransport := false
	foundWithTransport := false
	for _, scenario := range scenarios {
		if scenario.ID == "async_backlog_without_transport" {
			foundWithoutTransport = true
		}
		if scenario.ID == "async_backlog_with_transport" {
			foundWithTransport = true
		}
	}

	if !foundWithoutTransport || !foundWithTransport {
		t.Fatalf("expected async backlog demo scenarios to be listed, got %+v", scenarios)
	}
}

func TestDemoService_GenerateIncident_AsyncBacklogWithoutTransport(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	signals := NewService(repo, publisher, zap.NewNop())
	service := NewDemoService(signals, repo, zap.NewNop())

	incident, err := service.GenerateIncident(context.Background(), nil, "async_backlog_without_transport")
	if err != nil {
		t.Fatalf("expected incident to be generated, got error %v", err)
	}
	if incident.IncidentType != "dispatcher_backlog_pressure" {
		t.Fatalf("expected dispatcher backlog incident, got %s", incident.IncidentType)
	}

	evidence, err := repo.ListEvidenceByIncident(context.Background(), incident.ID)
	if err != nil {
		t.Fatalf("expected evidence list to succeed, got error %v", err)
	}
	if len(evidence) != 1 || evidence[0].EvidenceType != "dispatcher_backlog_snapshot" {
		t.Fatalf("expected dispatcher backlog evidence, got %+v", evidence)
	}

	actions, err := repo.ListActionAttemptsByIncident(context.Background(), incident.ID)
	if err != nil {
		t.Fatalf("expected action list to succeed, got error %v", err)
	}
	if len(actions) != 1 || actions[0].ActionKey != "review_dispatcher_backlog_pressure" {
		t.Fatalf("expected dispatcher review action, got %+v", actions)
	}
}

func TestDemoService_GenerateIncident_AsyncBacklogWithTransport(t *testing.T) {
	repo := &sreRepoStub{}
	publisher := &srePublisherStub{}
	signals := NewService(repo, publisher, zap.NewNop())
	service := NewDemoService(signals, repo, zap.NewNop())

	incident, err := service.GenerateIncident(context.Background(), nil, "async_backlog_with_transport")
	if err != nil {
		t.Fatalf("expected incident to be generated, got error %v", err)
	}
	if incident.IncidentType != "messaging_outbox_backlog_pressure" {
		t.Fatalf("expected messaging outbox backlog incident, got %s", incident.IncidentType)
	}

	evidence, err := repo.ListEvidenceByIncident(context.Background(), incident.ID)
	if err != nil {
		t.Fatalf("expected evidence list to succeed, got error %v", err)
	}
	if len(evidence) != 2 {
		t.Fatalf("expected two evidence records, got %+v", evidence)
	}

	actions, err := repo.ListActionAttemptsByIncident(context.Background(), incident.ID)
	if err != nil {
		t.Fatalf("expected action list to succeed, got error %v", err)
	}
	if len(actions) != 1 || actions[0].ActionKey != "review_messaging_transport_health" {
		t.Fatalf("expected transport review action, got %+v", actions)
	}
}
