package sresmartbot

import (
	"context"
	"testing"
	"time"

	"github.com/srikarm/image-factory/internal/infrastructure/logdetector"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type fakeLogQueryClient struct {
	results map[string]*logdetector.QueryResult
	errors  map[string]error
}

func (f *fakeLogQueryClient) QueryRange(ctx context.Context, query string, start time.Time, end time.Time, limit int) (*logdetector.QueryResult, error) {
	if err, ok := f.errors[query]; ok {
		return nil, err
	}
	if result, ok := f.results[query]; ok {
		return result, nil
	}
	return &logdetector.QueryResult{}, nil
}

type captureBus struct {
	events []messaging.Event
}

func (c *captureBus) Publish(ctx context.Context, event messaging.Event) error {
	c.events = append(c.events, event)
	return nil
}

func (c *captureBus) Subscribe(eventType string, handler messaging.Handler) (unsubscribe func()) {
	return func() {}
}

func TestRunLogDetectorTick_PublishesObservedAndRecovered(t *testing.T) {
	bus := &captureBus{}
	publisher := newLogFindingPublisher(bus, "server", "v1")
	rules := []LogDetectorRule{
		{
			Name:         "docker_hub_rate_limit",
			Query:        `{namespace="image-factory"} |= "toomanyrequests"`,
			Threshold:    1,
			Domain:       "runtime_services",
			IncidentType: "registry_pull_failure",
			DisplayName:  "Registry pull rate limit detected",
			Summary:      "Repeated Docker Hub pull-rate limit errors were detected",
			Severity:     "warning",
			Confidence:   "high",
			SignalKey:    "toomanyrequests",
		},
	}
	client := &fakeLogQueryClient{
		results: map[string]*logdetector.QueryResult{
			rules[0].Query: {
				Matches: []logdetector.QueryMatch{{
					Timestamp: time.Now().UTC(),
					Line:      "toomanyrequests: rate limit exceeded",
					Labels:    map[string]string{"namespace": "image-factory"},
				}},
			},
		},
	}
	active := map[string]activeRuleState{}
	start := time.Now().Add(-5 * time.Minute).UTC()
	end := time.Now().UTC()

	activeCount, findingsDelta, failuresDelta := runLogDetectorTick(zap.NewNop(), client, publisher, rules, active, start, end, 5*time.Minute, 5*time.Second, 5)
	if activeCount != 1 || findingsDelta != 1 || failuresDelta != 0 {
		t.Fatalf("unexpected tick counts: active=%d findings=%d failures=%d", activeCount, findingsDelta, failuresDelta)
	}
	if len(bus.events) != 1 || bus.events[0].Type != messaging.EventTypeSREDetectorFindingObserved {
		t.Fatalf("expected one observed event, got %+v", bus.events)
	}

	client.results[rules[0].Query] = &logdetector.QueryResult{}
	activeCount, findingsDelta, failuresDelta = runLogDetectorTick(zap.NewNop(), client, publisher, rules, active, start, end.Add(1*time.Minute), 5*time.Minute, 5*time.Second, 5)
	if activeCount != 0 || findingsDelta != 0 || failuresDelta != 0 {
		t.Fatalf("unexpected recovery tick counts: active=%d findings=%d failures=%d", activeCount, findingsDelta, failuresDelta)
	}
	if len(bus.events) != 2 || bus.events[1].Type != messaging.EventTypeSREDetectorFindingRecovered {
		t.Fatalf("expected recovery event, got %+v", bus.events)
	}
}

func TestDefaultLogDetectorRules_IncludeNotificationFailures(t *testing.T) {
	rules := defaultLogDetectorRules()
	foundDelivery := false
	foundQueuePersistence := false
	for _, rule := range rules {
		switch rule.Name {
		case "notification_delivery_failure":
			foundDelivery = true
			if rule.IncidentType != "notification_delivery_failure" {
				t.Fatalf("expected notification delivery rule incident type, got %q", rule.IncidentType)
			}
		case "notification_queue_persistence_failure":
			foundQueuePersistence = true
			if rule.IncidentType != "notification_delivery_failure" {
				t.Fatalf("expected queue persistence rule incident type, got %q", rule.IncidentType)
			}
		}
	}
	if !foundDelivery {
		t.Fatal("expected notification_delivery_failure rule to be registered")
	}
	if !foundQueuePersistence {
		t.Fatal("expected notification_queue_persistence_failure rule to be registered")
	}
}
