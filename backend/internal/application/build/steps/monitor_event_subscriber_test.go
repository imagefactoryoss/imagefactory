package steps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type monitorEventRepoStub struct {
	instance            *domainworkflow.Instance
	steps               []domainworkflow.Step
	err                 error
	updateInstanceID    uuid.UUID
	updateStepKey       string
	updateStatus        domainworkflow.StepStatus
	updateErrMsg        *string
	updateCalls         int
	appendCalls         int
	lastAppendedEvent   *domainworkflow.Event
	updateStepStatusErr error
}

func (m *monitorEventRepoStub) ClaimNextRunnableStep(ctx context.Context) (*domainworkflow.Step, error) {
	return nil, nil
}
func (m *monitorEventRepoStub) UpdateStep(ctx context.Context, step *domainworkflow.Step) error {
	return nil
}
func (m *monitorEventRepoStub) AppendEvent(ctx context.Context, event *domainworkflow.Event) error {
	m.appendCalls++
	m.lastAppendedEvent = event
	return nil
}
func (m *monitorEventRepoStub) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *monitorEventRepoStub) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status domainworkflow.InstanceStatus) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *monitorEventRepoStub) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []domainworkflow.StepDefinition) error {
	return nil
}
func (m *monitorEventRepoStub) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status domainworkflow.InstanceStatus) error {
	return nil
}
func (m *monitorEventRepoStub) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status domainworkflow.StepStatus, errMsg *string) error {
	m.updateCalls++
	m.updateInstanceID = instanceID
	m.updateStepKey = stepKey
	m.updateStatus = status
	m.updateErrMsg = errMsg
	if m.updateStepStatusErr != nil {
		return m.updateStepStatusErr
	}
	return nil
}
func (m *monitorEventRepoStub) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*domainworkflow.Instance, []domainworkflow.Step, error) {
	return m.instance, m.steps, m.err
}
func (m *monitorEventRepoStub) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*domainworkflow.BlockedStepDiagnostics, error) {
	return nil, nil
}

func TestBuildMonitorEventSubscriberCountsStatuses(t *testing.T) {
	buildID := uuid.New()
	instance := &domainworkflow.Instance{ID: uuid.New()}
	monitorStepID := uuid.New()
	repo := &monitorEventRepoStub{
		instance: instance,
		steps: []domainworkflow.Step{
			{ID: monitorStepID, StepKey: StepMonitorBuild, Status: domainworkflow.StepStatusPending},
			{StepKey: StepFinalizeBuild, Status: domainworkflow.StepStatusRunning},
		},
	}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())
	subscriber.HandleExecutionTerminalEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionCompleted,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
		},
	})

	snapshot := subscriber.Snapshot()
	if snapshot.EventsReceived != 1 {
		t.Fatalf("expected EventsReceived=1, got %d", snapshot.EventsReceived)
	}
	if snapshot.MonitorPending != 1 {
		t.Fatalf("expected MonitorPending=1, got %d", snapshot.MonitorPending)
	}
	if snapshot.FinalizeRunning != 1 {
		t.Fatalf("expected FinalizeRunning=1, got %d", snapshot.FinalizeRunning)
	}
	if snapshot.Transitioned != 1 {
		t.Fatalf("expected Transitioned=1, got %d", snapshot.Transitioned)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected UpdateStepStatus call once, got %d", repo.updateCalls)
	}
	if repo.updateInstanceID != instance.ID {
		t.Fatalf("unexpected instance ID update: %s", repo.updateInstanceID.String())
	}
	if repo.updateStepKey != StepMonitorBuild {
		t.Fatalf("unexpected step key update: %s", repo.updateStepKey)
	}
	if repo.updateStatus != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected monitor update to succeeded, got %s", repo.updateStatus)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendEvent call once, got %d", repo.appendCalls)
	}
	if repo.lastAppendedEvent == nil || repo.lastAppendedEvent.StepID == nil || *repo.lastAppendedEvent.StepID != monitorStepID {
		t.Fatalf("expected appended event step_id to be monitor step")
	}
}

func TestBuildMonitorEventSubscriberHandlesInvalidBuildID(t *testing.T) {
	repo := &monitorEventRepoStub{}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	subscriber.HandleExecutionTerminalEvent(context.Background(), messaging.Event{
		Type:    messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{"build_id": "not-a-uuid"},
	})

	snapshot := subscriber.Snapshot()
	if snapshot.ParseFailures != 1 {
		t.Fatalf("expected ParseFailures=1, got %d", snapshot.ParseFailures)
	}
}

func TestBuildMonitorEventSubscriberHandlesMissingWorkflow(t *testing.T) {
	repo := &monitorEventRepoStub{instance: nil}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	subscriber.HandleExecutionTerminalEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
		},
	})

	snapshot := subscriber.Snapshot()
	if snapshot.WorkflowMissing != 1 {
		t.Fatalf("expected WorkflowMissing=1, got %d", snapshot.WorkflowMissing)
	}
}

func TestBuildMonitorEventSubscriberRepoErrorDoesNotPanic(t *testing.T) {
	repo := &monitorEventRepoStub{err: errors.New("db down")}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	subscriber.HandleExecutionTerminalEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionCompleted,
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
		},
	})

	snapshot := subscriber.Snapshot()
	if snapshot.EventsReceived != 1 {
		t.Fatalf("expected EventsReceived=1, got %d", snapshot.EventsReceived)
	}
}

func TestBuildMonitorEventSubscriberNoopWhenMonitorAlreadyTerminal(t *testing.T) {
	repo := &monitorEventRepoStub{
		instance: &domainworkflow.Instance{ID: uuid.New()},
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: StepMonitorBuild, Status: domainworkflow.StepStatusSucceeded},
		},
	}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	subscriber.HandleExecutionTerminalEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionCompleted,
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
		},
	})

	if subscriber.Snapshot().NoopTerminal != 1 {
		t.Fatalf("expected NoopTerminal=1, got %d", subscriber.Snapshot().NoopTerminal)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no update call, got %d", repo.updateCalls)
	}
}

func TestBuildMonitorEventSubscriberFailedTransitionSetsError(t *testing.T) {
	repo := &monitorEventRepoStub{
		instance: &domainworkflow.Instance{ID: uuid.New()},
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: StepMonitorBuild, Status: domainworkflow.StepStatusRunning},
		},
	}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	subscriber.HandleExecutionTerminalEvent(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
			"message":  "pipeline failed in kaniko step",
		},
	})

	if repo.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", repo.updateCalls)
	}
	if repo.updateStatus != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status update, got %s", repo.updateStatus)
	}
	if repo.updateErrMsg == nil || *repo.updateErrMsg != "pipeline failed in kaniko step" {
		t.Fatalf("expected propagated error message, got %+v", repo.updateErrMsg)
	}
}

func TestBuildMonitorEventSubscriberRegisterWiresHandlers(t *testing.T) {
	repo := &monitorEventRepoStub{
		instance: &domainworkflow.Instance{ID: uuid.New()},
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: StepMonitorBuild, Status: domainworkflow.StepStatusPending},
		},
	}
	subscriber := NewBuildMonitorEventSubscriber(repo, zap.NewNop())
	bus := messaging.NewInProcessBus(zap.NewNop())
	unsub := RegisterBuildMonitorEventSubscriber(bus, subscriber)
	defer unsub()

	_ = bus.Publish(context.Background(), messaging.Event{
		Type: messaging.EventTypeBuildExecutionCompleted,
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
		},
	})

	// in-process bus dispatches async; assert eventually.
	for i := 0; i < 20; i++ {
		if subscriber.Snapshot().EventsReceived > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected at least one received event")
}

func TestBuildMonitorEventSubscriberCrossInstanceSharedStateNoop(t *testing.T) {
	buildID := uuid.New()
	instance := &domainworkflow.Instance{ID: uuid.New()}
	repo := &monitorEventRepoStub{
		instance: instance,
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: StepMonitorBuild, Status: domainworkflow.StepStatusPending},
		},
	}

	subscriberA := NewBuildMonitorEventSubscriber(repo, zap.NewNop())
	subscriberB := NewBuildMonitorEventSubscriber(repo, zap.NewNop())
	event := messaging.Event{
		Type: messaging.EventTypeBuildExecutionCompleted,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
		},
	}

	subscriberA.HandleExecutionTerminalEvent(context.Background(), event)
	if repo.updateCalls != 1 {
		t.Fatalf("expected subscriber A to perform one transition, got %d", repo.updateCalls)
	}

	// Simulate shared DB state observed by a second instance after A transitioned monitor step.
	repo.steps[0].Status = domainworkflow.StepStatusSucceeded
	subscriberB.HandleExecutionTerminalEvent(context.Background(), event)

	if repo.updateCalls != 1 {
		t.Fatalf("expected subscriber B to no-op on terminal state, got update calls=%d", repo.updateCalls)
	}
	if subscriberB.Snapshot().NoopTerminal != 1 {
		t.Fatalf("expected subscriber B noop terminal count 1, got %d", subscriberB.Snapshot().NoopTerminal)
	}
}
