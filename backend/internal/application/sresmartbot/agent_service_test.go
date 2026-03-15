package sresmartbot

import "testing"

func TestBuildDraftHypotheses_IncludesAsyncBacklogAndMessagingTransport(t *testing.T) {
	workspace := &IncidentWorkspace{
		Incident: (&incidentFixture{
			Domain:       "golden_signals",
			IncidentType: "backlog_pressure",
			DisplayName:  "Async backlog pressure",
		}).toDomain(),
	}
	toolRuns := []MCPToolInvocationResult{
		{ToolName: "async_backlog.recent", ServerName: "Observability", Payload: map[string]any{
			"build_queue_depth":        int64(12),
			"email_queue_depth":        int64(5),
			"messaging_outbox_pending": int64(8),
		}},
		{ToolName: "messaging_transport.recent", ServerName: "Observability", Payload: map[string]any{
			"reconnects":          int64(4),
			"disconnects":         int64(2),
			"reconnect_threshold": int64(3),
		}},
	}

	hypotheses := buildDraftHypotheses(workspace, toolRuns)
	if len(hypotheses) == 0 {
		t.Fatal("expected hypotheses to be generated")
	}

	foundBacklog := false
	foundTransport := false
	for _, hypothesis := range hypotheses {
		for _, signal := range hypothesis.SignalsUsed {
			if signal == "async_backlog.recent" {
				foundBacklog = true
			}
			if signal == "messaging_transport.recent" {
				foundTransport = true
			}
		}
	}

	if !foundBacklog {
		t.Fatal("expected async_backlog.recent to influence hypotheses")
	}
	if !foundTransport {
		t.Fatal("expected messaging_transport.recent to influence hypotheses")
	}
}

func TestSummarizeToolRun_MessagingTransport(t *testing.T) {
	summary := summarizeToolRun(MCPToolInvocationResult{
		ToolName:   "messaging_transport.recent",
		ServerName: "Observability",
		Payload: map[string]any{
			"reconnects":          int64(4),
			"disconnects":         int64(2),
			"reconnect_threshold": int64(3),
		},
	})

	expected := "Messaging transport shows reconnects=4, disconnects=2, threshold=3."
	if summary != expected {
		t.Fatalf("expected %q, got %q", expected, summary)
	}
}
