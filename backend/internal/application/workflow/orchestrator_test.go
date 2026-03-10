package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

type mockRepo struct {
	steps       []*workflow.Step
	claimErr    error
	claimIndex  int
	updatedStep *workflow.Step
	events      []*workflow.Event
}

func (m *mockRepo) ClaimNextRunnableStep(ctx context.Context) (*workflow.Step, error) {
	if m.claimErr != nil {
		return nil, m.claimErr
	}
	if len(m.steps) == 0 {
		return nil, nil
	}
	if m.claimIndex >= len(m.steps) {
		return nil, nil
	}
	step := m.steps[m.claimIndex]
	m.claimIndex++
	return step, nil
}

func (m *mockRepo) UpdateStep(ctx context.Context, step *workflow.Step) error {
	m.updatedStep = step
	return nil
}

func (m *mockRepo) AppendEvent(ctx context.Context, event *workflow.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockRepo) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockRepo) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status workflow.InstanceStatus) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockRepo) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []workflow.StepDefinition) error {
	return nil
}

func (m *mockRepo) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status workflow.InstanceStatus) error {
	return nil
}

func (m *mockRepo) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status workflow.StepStatus, errMsg *string) error {
	return nil
}

func (m *mockRepo) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*workflow.Instance, []workflow.Step, error) {
	return nil, nil, nil
}

func (m *mockRepo) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*workflow.BlockedStepDiagnostics, error) {
	return &workflow.BlockedStepDiagnostics{SubjectType: subjectType}, nil
}

type mockHandler struct {
	key string
	res StepResult
}

func (m *mockHandler) Key() string { return m.key }

func (m *mockHandler) Execute(ctx context.Context, step *workflow.Step) (StepResult, error) {
	if m.res.Status != "" {
		return m.res, nil
	}
	return StepResult{
		Status: workflow.StepStatusSucceeded,
		Data:   map[string]interface{}{"ok": true},
	}, nil
}

func TestOrchestratorRunOnce_NoStep(t *testing.T) {
	repo := &mockRepo{}
	orch := NewOrchestrator(repo, nil, zap.NewNop())

	ran, err := orch.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ran {
		t.Fatalf("expected ran=false when no step is available")
	}
}

func TestOrchestratorRunOnce_ExecutesStep(t *testing.T) {
	step := &workflow.Step{
		ID:         uuid.New(),
		InstanceID: uuid.New(),
		StepKey:    "queue_build",
		Status:     workflow.StepStatusRunning,
		Attempts:   1,
		StartedAt:  ptrTime(time.Now().UTC()),
	}
	repo := &mockRepo{steps: []*workflow.Step{step}}
	orch := NewOrchestrator(repo, []StepHandler{&mockHandler{key: "queue_build"}}, zap.NewNop())

	ran, err := orch.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ran {
		t.Fatalf("expected ran=true when step executed")
	}
	if repo.updatedStep == nil {
		t.Fatalf("expected step to be updated")
	}
	if repo.updatedStep.Status != workflow.StepStatusSucceeded {
		t.Fatalf("expected step status succeeded, got %s", repo.updatedStep.Status)
	}
	if len(repo.events) == 0 {
		t.Fatalf("expected event to be appended")
	}
}

func TestOrchestratorRunOnce_RequeuesBlockedStep(t *testing.T) {
	step := &workflow.Step{
		ID:         uuid.New(),
		InstanceID: uuid.New(),
		StepKey:    "build.monitor",
		Status:     workflow.StepStatusRunning,
		Attempts:   1,
		StartedAt:  ptrTime(time.Now().UTC()),
	}
	repo := &mockRepo{steps: []*workflow.Step{step}}
	orch := NewOrchestrator(repo, []StepHandler{&mockHandler{
		key: "build.monitor",
		res: StepResult{
			Status: workflow.StepStatusBlocked,
			Error:  "build still running",
		},
	}}, zap.NewNop())

	ran, err := orch.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ran {
		t.Fatalf("expected ran=true")
	}
	if repo.updatedStep == nil {
		t.Fatalf("expected step update")
	}
	if repo.updatedStep.Status != workflow.StepStatusPending {
		t.Fatalf("expected pending for blocked requeue, got %s", repo.updatedStep.Status)
	}
	if repo.updatedStep.CompletedAt != nil {
		t.Fatalf("expected no completed_at for requeued step")
	}
}

func TestOrchestratorRunOnce_HandlerErrorMarksStepFailed(t *testing.T) {
	step := &workflow.Step{
		ID:         uuid.New(),
		InstanceID: uuid.New(),
		StepKey:    "build.dispatch",
		Status:     workflow.StepStatusRunning,
		Attempts:   1,
		StartedAt:  ptrTime(time.Now().UTC()),
	}
	repo := &mockRepo{steps: []*workflow.Step{step}}
	failing := &errorHandler{key: "build.dispatch", err: errors.New("dispatcher unavailable")}
	orch := NewOrchestrator(repo, []StepHandler{failing}, zap.NewNop())

	ran, err := orch.RunOnce(context.Background())
	if !ran {
		t.Fatalf("expected ran=true")
	}
	if err == nil {
		t.Fatal("expected handler error")
	}
	if repo.updatedStep == nil {
		t.Fatalf("expected updated step")
	}
	if repo.updatedStep.Status != workflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", repo.updatedStep.Status)
	}
	if len(repo.events) == 0 || repo.events[len(repo.events)-1].Type != "workflow.step.failed" {
		t.Fatalf("expected failed event appended")
	}
}

func TestOrchestratorRunOnce_ClaimErrorReturnsFailure(t *testing.T) {
	repo := &mockRepo{claimErr: errors.New("claim failed")}
	orch := NewOrchestrator(repo, nil, zap.NewNop())

	ran, err := orch.RunOnce(context.Background())
	if ran {
		t.Fatalf("expected ran=false on claim error")
	}
	if err == nil {
		t.Fatal("expected claim error")
	}
}

func TestOrchestratorRunOnce_PartialRestartBlockedThenSucceeded(t *testing.T) {
	step := &workflow.Step{
		ID:         uuid.New(),
		InstanceID: uuid.New(),
		StepKey:    "build.monitor",
		Status:     workflow.StepStatusRunning,
		Attempts:   1,
		StartedAt:  ptrTime(time.Now().UTC()),
	}
	repo := &mockRepo{steps: []*workflow.Step{step, step}}
	handler := &sequencedHandler{
		key: "build.monitor",
		results: []StepResult{
			{Status: workflow.StepStatusBlocked, Error: "build still running"},
			{Status: workflow.StepStatusSucceeded, Data: map[string]interface{}{"build_status": "completed"}},
		},
	}
	orch := NewOrchestrator(repo, []StepHandler{handler}, zap.NewNop())

	ran, err := orch.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("expected first run to process blocked step, ran=%v err=%v", ran, err)
	}
	if repo.updatedStep == nil || repo.updatedStep.Status != workflow.StepStatusPending {
		t.Fatalf("expected blocked step to requeue as pending")
	}

	ran, err = orch.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("expected second run to succeed recovered step, ran=%v err=%v", ran, err)
	}
	if repo.updatedStep == nil || repo.updatedStep.Status != workflow.StepStatusSucceeded {
		t.Fatalf("expected recovered step to succeed, got %v", repo.updatedStep.Status)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

type errorHandler struct {
	key string
	err error
}

func (h *errorHandler) Key() string { return h.key }
func (h *errorHandler) Execute(ctx context.Context, step *workflow.Step) (StepResult, error) {
	return StepResult{}, h.err
}

type sequencedHandler struct {
	key     string
	results []StepResult
	index   int
}

func (h *sequencedHandler) Key() string { return h.key }
func (h *sequencedHandler) Execute(ctx context.Context, step *workflow.Step) (StepResult, error) {
	if len(h.results) == 0 {
		return StepResult{Status: workflow.StepStatusSucceeded}, nil
	}
	if h.index >= len(h.results) {
		return h.results[len(h.results)-1], nil
	}
	result := h.results[h.index]
	h.index++
	return result, nil
}
