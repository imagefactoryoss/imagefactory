package steps

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	domainbuild "github.com/srikarm/image-factory/internal/domain/build"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

func validStepPayloadMap() map[string]interface{} {
	return map[string]interface{}{
		"tenant_id":  uuid.New().String(),
		"project_id": uuid.New().String(),
		"manifest": map[string]interface{}{
			"name":         "wf-build",
			"type":         "container",
			"base_image":   "alpine:3.19",
			"instructions": []string{"RUN echo ok"},
		},
	}
}

type stubBuildControlPlaneService struct {
	createdBuild *domainbuild.Build
	fetchedBuild *domainbuild.Build
	executions   []domainbuild.BuildExecution
	createErr    error
	startErr     error
	getErr       error
	execErr      error
	failErr      error
	failReason   string
	startedIDs   []uuid.UUID
}

func (s *stubBuildControlPlaneService) CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest domainbuild.BuildManifest, actorID *uuid.UUID) (*domainbuild.Build, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createdBuild != nil {
		return s.createdBuild, nil
	}
	b, _ := domainbuild.NewBuild(tenantID, projectID, manifest, nil)
	_ = b.Queue()
	return b, nil
}

func (s *stubBuildControlPlaneService) StartBuild(ctx context.Context, buildID uuid.UUID) error {
	if s.startErr != nil {
		return s.startErr
	}
	s.startedIDs = append(s.startedIDs, buildID)
	return nil
}

func (s *stubBuildControlPlaneService) GetBuild(ctx context.Context, id uuid.UUID) (*domainbuild.Build, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.fetchedBuild, nil
}

func (s *stubBuildControlPlaneService) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]domainbuild.BuildExecution, int64, error) {
	if s.execErr != nil {
		return nil, 0, s.execErr
	}
	return s.executions, int64(len(s.executions)), nil
}

func (s *stubBuildControlPlaneService) MarkBuildFailed(ctx context.Context, buildID uuid.UUID, reason string) error {
	if s.failErr != nil {
		return s.failErr
	}
	s.failReason = reason
	return nil
}

func mustBuild(t *testing.T, status domainbuild.BuildStatus) *domainbuild.Build {
	t.Helper()
	manifest := domainbuild.BuildManifest{
		Name:         "wf-build",
		Type:         domainbuild.BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}
	b, err := domainbuild.NewBuild(uuid.New(), uuid.New(), manifest, nil)
	if err != nil {
		t.Fatalf("failed to create build: %v", err)
	}
	switch status {
	case domainbuild.BuildStatusQueued:
		_ = b.Queue()
	case domainbuild.BuildStatusRunning:
		_ = b.Queue()
		_ = b.Start()
	case domainbuild.BuildStatusCompleted:
		_ = b.Queue()
		_ = b.Start()
		_ = b.Complete(domainbuild.BuildResult{})
	case domainbuild.BuildStatusFailed:
		_ = b.Queue()
		_ = b.Start()
		_ = b.Fail("boom")
	case domainbuild.BuildStatusCancelled:
		_ = b.Cancel()
	}
	return b
}

func withBuildID(payload map[string]interface{}, buildID uuid.UUID) map[string]interface{} {
	copy := map[string]interface{}{}
	for k, v := range payload {
		copy[k] = v
	}
	copy["build_id"] = buildID.String()
	return copy
}

func TestValidateBuildHandler_Success(t *testing.T) {
	h := NewValidateBuildHandler(zap.NewNop())
	result, err := h.Execute(context.Background(), &domainworkflow.Step{Payload: validStepPayloadMap()})
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", result.Status)
	}
}

func TestValidateBuildHandler_FailsForMissingManifestName(t *testing.T) {
	payload := validStepPayloadMap()
	payload["manifest"].(map[string]interface{})["name"] = ""
	h := NewValidateBuildHandler(zap.NewNop())
	result, err := h.Execute(context.Background(), &domainworkflow.Step{Payload: payload})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed, got %s", result.Status)
	}
}

func TestSelectInfrastructureHandler_SuccessAndMutatesPayload(t *testing.T) {
	expectedProviderID := uuid.New()
	h := NewSelectInfrastructureHandler(func(ctx context.Context, tenantID uuid.UUID, manifest *domainbuild.BuildManifest) error {
		manifest.InfrastructureType = "kubernetes"
		manifest.InfrastructureProviderID = &expectedProviderID
		return nil
	}, zap.NewNop())

	step := &domainworkflow.Step{Payload: validStepPayloadMap()}
	result, err := h.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", result.Status)
	}
	manifest := step.Payload["manifest"].(map[string]interface{})
	if manifest["infrastructure_type"] != "kubernetes" {
		t.Fatalf("expected infrastructure_type kubernetes, got %v", manifest["infrastructure_type"])
	}
	if manifest["infrastructure_provider_id"] != expectedProviderID.String() {
		t.Fatalf("expected provider id %s, got %v", expectedProviderID.String(), manifest["infrastructure_provider_id"])
	}
}

func TestEnqueueBuildHandler_SetsBuildID(t *testing.T) {
	svc := &stubBuildControlPlaneService{createdBuild: mustBuild(t, domainbuild.BuildStatusQueued)}
	h := NewEnqueueBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{Payload: validStepPayloadMap()}
	result, err := h.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected enqueue success, got err: %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", result.Status)
	}
	if step.Payload["build_id"] == "" {
		t.Fatal("expected build_id injected into step payload")
	}
}

func TestDispatchBuildHandler_StartsBuild(t *testing.T) {
	b := mustBuild(t, domainbuild.BuildStatusQueued)
	buildID := b.ID()
	svc := &stubBuildControlPlaneService{fetchedBuild: b}
	h := NewDispatchBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), buildID)}
	result, err := h.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected dispatch success, got err: %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", result.Status)
	}
	if len(svc.startedIDs) != 1 || svc.startedIDs[0] != buildID {
		t.Fatalf("expected StartBuild called with %s", buildID)
	}
}

func TestDispatchBuildHandler_FailsFastWhenStartFails(t *testing.T) {
	b := mustBuild(t, domainbuild.BuildStatusQueued)
	buildID := b.ID()
	svc := &stubBuildControlPlaneService{fetchedBuild: b, startErr: errors.New("dispatcher unavailable")}
	h := NewDispatchBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), buildID)}

	result, err := h.Execute(context.Background(), step)
	if err == nil {
		t.Fatal("expected dispatch failure when start fails")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if result.Error == "" {
		t.Fatal("expected surfaced dispatch error message")
	}
}

func TestDispatchBuildHandler_SkipsStartWhenNotQueued(t *testing.T) {
	tests := []struct {
		name   string
		status domainbuild.BuildStatus
	}{
		{name: "running", status: domainbuild.BuildStatusRunning},
		{name: "completed", status: domainbuild.BuildStatusCompleted},
		{name: "failed", status: domainbuild.BuildStatusFailed},
		{name: "cancelled", status: domainbuild.BuildStatusCancelled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := mustBuild(t, tc.status)
			svc := &stubBuildControlPlaneService{fetchedBuild: b}
			h := NewDispatchBuildHandler(svc, zap.NewNop())
			step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), b.ID())}

			result, err := h.Execute(context.Background(), step)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != domainworkflow.StepStatusSucceeded {
				t.Fatalf("expected succeeded, got %s", result.Status)
			}
			if len(svc.startedIDs) != 0 {
				t.Fatalf("expected StartBuild not to be called, got %v", svc.startedIDs)
			}
		})
	}
}

func TestMonitorBuildHandler_StatusOutcomes(t *testing.T) {
	tests := []struct {
		name         string
		status       domainbuild.BuildStatus
		expectedStep domainworkflow.StepStatus
		expectsErr   bool
	}{
		{name: "completed", status: domainbuild.BuildStatusCompleted, expectedStep: domainworkflow.StepStatusSucceeded, expectsErr: false},
		{name: "failed", status: domainbuild.BuildStatusFailed, expectedStep: domainworkflow.StepStatusFailed, expectsErr: true},
		{name: "running", status: domainbuild.BuildStatusRunning, expectedStep: domainworkflow.StepStatusBlocked, expectsErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := mustBuild(t, tc.status)
			svc := &stubBuildControlPlaneService{fetchedBuild: b}
			h := NewMonitorBuildHandler(svc, zap.NewNop())
			step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), b.ID())}
			result, err := h.Execute(context.Background(), step)
			if tc.expectsErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.expectsErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != tc.expectedStep {
				t.Fatalf("expected %s, got %s", tc.expectedStep, result.Status)
			}
		})
	}
}

func TestMonitorBuildHandler_MarksOrphanedKubernetesExecutionFailed(t *testing.T) {
	b := mustBuild(t, domainbuild.BuildStatusRunning)
	b.SetInfrastructureSelection("kubernetes", "selected")
	svc := &stubBuildControlPlaneService{
		fetchedBuild: b,
		executions: []domainbuild.BuildExecution{
			{
				ID:       uuid.New(),
				BuildID:  b.ID(),
				Status:   domainbuild.ExecutionRunning,
				Metadata: []byte(`null`),
			},
		},
	}
	h := NewMonitorBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{
		Payload:  withBuildID(validStepPayloadMap(), b.ID()),
		Attempts: orphanedExecutionDetectionAttempts,
	}
	result, err := h.Execute(context.Background(), step)
	if err == nil {
		t.Fatal("expected orphaned execution failure")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if svc.failReason == "" {
		t.Fatal("expected MarkBuildFailed to be invoked")
	}
}

func TestMonitorBuildHandler_DoesNotMarkOrphanedBeforeThreshold(t *testing.T) {
	b := mustBuild(t, domainbuild.BuildStatusRunning)
	b.SetInfrastructureSelection("kubernetes", "selected")
	svc := &stubBuildControlPlaneService{
		fetchedBuild: b,
		executions: []domainbuild.BuildExecution{
			{
				ID:       uuid.New(),
				BuildID:  b.ID(),
				Status:   domainbuild.ExecutionRunning,
				Metadata: []byte(`null`),
			},
		},
	}
	h := NewMonitorBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{
		Payload:  withBuildID(validStepPayloadMap(), b.ID()),
		Attempts: orphanedExecutionDetectionAttempts - 1,
	}
	result, err := h.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status, got %s", result.Status)
	}
	if svc.failReason != "" {
		t.Fatal("did not expect MarkBuildFailed before detection threshold")
	}
}

func TestMonitorBuildHandler_RespectsBackoffWindow(t *testing.T) {
	b := mustBuild(t, domainbuild.BuildStatusRunning)
	next := time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339)
	step := &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":             uuid.New().String(),
			"project_id":            uuid.New().String(),
			"build_id":              b.ID().String(),
			"manifest":              validStepPayloadMap()["manifest"],
			"monitor_next_check_at": next,
		},
	}

	svc := &stubBuildControlPlaneService{fetchedBuild: b}
	h := NewMonitorBuildHandler(svc, zap.NewNop())
	result, err := h.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status, got %s", result.Status)
	}
	if result.Error == "" {
		t.Fatalf("expected backoff error message")
	}
}

func TestMonitorBuildHandler_UsesExecutionTerminalState(t *testing.T) {
	b := mustBuild(t, domainbuild.BuildStatusRunning)
	svc := &stubBuildControlPlaneService{
		fetchedBuild: b,
		executions: []domainbuild.BuildExecution{
			{
				ID:           uuid.New(),
				BuildID:      b.ID(),
				Status:       domainbuild.ExecutionFailed,
				ErrorMessage: "tekton failed",
			},
		},
	}
	h := NewMonitorBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), b.ID())}

	result, err := h.Execute(context.Background(), step)
	if err == nil {
		t.Fatal("expected error for failed execution")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if svc.failReason != "tekton failed" {
		t.Fatalf("expected build failure propagation from execution, got %q", svc.failReason)
	}
}

func TestMonitorBuildHandler_FailsWhenExecutionTimeoutExceeded(t *testing.T) {
	t.Setenv("IF_BUILD_MONITOR_TIMEOUT_SECONDS", "60")

	b := mustBuild(t, domainbuild.BuildStatusRunning)
	startedAt := time.Now().UTC().Add(-3 * time.Minute)
	svc := &stubBuildControlPlaneService{
		fetchedBuild: b,
		executions: []domainbuild.BuildExecution{
			{
				ID:        uuid.New(),
				BuildID:   b.ID(),
				Status:    domainbuild.ExecutionRunning,
				StartedAt: &startedAt,
			},
		},
	}
	h := NewMonitorBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), b.ID())}

	result, err := h.Execute(context.Background(), step)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if svc.failReason == "" || !strings.Contains(svc.failReason, "timeout exceeded") {
		t.Fatalf("expected timeout failure reason, got %q", svc.failReason)
	}
}

func TestMonitorBuildHandler_DoesNotFailWhenExecutionWithinTimeout(t *testing.T) {
	t.Setenv("IF_BUILD_MONITOR_TIMEOUT_SECONDS", "3600")

	b := mustBuild(t, domainbuild.BuildStatusRunning)
	startedAt := time.Now().UTC().Add(-2 * time.Minute)
	svc := &stubBuildControlPlaneService{
		fetchedBuild: b,
		executions: []domainbuild.BuildExecution{
			{
				ID:        uuid.New(),
				BuildID:   b.ID(),
				Status:    domainbuild.ExecutionRunning,
				StartedAt: &startedAt,
			},
		},
	}
	h := NewMonitorBuildHandler(svc, zap.NewNop())
	step := &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), b.ID())}

	result, err := h.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status, got %s", result.Status)
	}
	if svc.failReason != "" {
		t.Fatalf("did not expect build failure, got %q", svc.failReason)
	}
}

func TestFinalizeBuildHandler_TerminalCheck(t *testing.T) {
	nonTerminal := mustBuild(t, domainbuild.BuildStatusRunning)
	svc := &stubBuildControlPlaneService{fetchedBuild: nonTerminal}
	h := NewFinalizeBuildHandler(svc, zap.NewNop())
	result, err := h.Execute(context.Background(), &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), nonTerminal.ID())})
	if err != nil {
		t.Fatalf("unexpected error for non-terminal finalize: %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked for non-terminal build, got %s", result.Status)
	}

	terminal := mustBuild(t, domainbuild.BuildStatusCompleted)
	svc.fetchedBuild = terminal
	result, err = h.Execute(context.Background(), &domainworkflow.Step{Payload: withBuildID(validStepPayloadMap(), terminal.ID())})
	if err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded for terminal build, got %s", result.Status)
	}
}

func TestSelectInfrastructureHandler_FailsOnPreflightError(t *testing.T) {
	h := NewSelectInfrastructureHandler(func(ctx context.Context, tenantID uuid.UUID, manifest *domainbuild.BuildManifest) error {
		return errors.New("boom")
	}, zap.NewNop())
	result, err := h.Execute(context.Background(), &domainworkflow.Step{Payload: validStepPayloadMap()})
	if err == nil {
		t.Fatal("expected preflight error")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed, got %s", result.Status)
	}
}

func TestNewPhase2ControlPlaneHandlers_OrderAndKeys(t *testing.T) {
	svc := &stubBuildControlPlaneService{}
	handlers := NewPhase2ControlPlaneHandlers(svc, func(ctx context.Context, tenantID uuid.UUID, manifest *domainbuild.BuildManifest) error { return nil }, zap.NewNop())
	if len(handlers) != 6 {
		t.Fatalf("expected 6 handlers, got %d", len(handlers))
	}
	expectedKeys := []string{
		StepValidateBuild,
		StepSelectInfrastructure,
		StepEnqueueBuild,
		StepDispatchBuild,
		StepMonitorBuild,
		StepFinalizeBuild,
	}
	for i, h := range handlers {
		if h.Key() != expectedKeys[i] {
			t.Fatalf("unexpected key at index %d: want %s got %s", i, expectedKeys[i], h.Key())
		}
	}
}
