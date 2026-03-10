package build

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubMethodExecutorAdapterFactory struct {
	executor BuildMethodExecutor
}

func (f *stubMethodExecutorAdapterFactory) CreateExecutor(method BuildMethod) (BuildMethodExecutor, error) {
	return f.executor, nil
}

func (f *stubMethodExecutorAdapterFactory) GetSupportedMethods() []BuildMethod {
	return []BuildMethod{BuildMethodDocker}
}

type stubMethodExecutorAdapterExec struct {
	output *MethodExecutionOutput
	err    error
}

func (e *stubMethodExecutorAdapterExec) Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error) {
	return e.output, e.err
}
func (e *stubMethodExecutorAdapterExec) Cancel(ctx context.Context, executionID string) error {
	return nil
}
func (e *stubMethodExecutorAdapterExec) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
	return ExecutionPending, nil
}
func (e *stubMethodExecutorAdapterExec) Supports(method BuildMethod) bool {
	return true
}

func makeMethodAdapterTestBuild(t *testing.T) *Build {
	t.Helper()
	b, err := NewBuild(uuid.New(), uuid.New(), BuildManifest{
		Name:         "adapter-test",
		Type:         BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}, nil)
	if err != nil {
		t.Fatalf("failed to create build: %v", err)
	}
	b.SetConfig(&BuildConfigData{
		ID:          uuid.New(),
		BuildID:     b.ID(),
		BuildMethod: string(BuildMethodDocker),
	})
	return b
}

func TestMethodExecutorAdapter_ReturnsInProgressForRunningOutput(t *testing.T) {
	b := makeMethodAdapterTestBuild(t)
	adapter := NewMethodExecutorAdapter(
		&stubMethodExecutorAdapterFactory{
			executor: &stubMethodExecutorAdapterExec{
				output: &MethodExecutionOutput{
					Status: ExecutionRunning,
					Output: "scheduled",
				},
			},
		},
		zap.NewNop(),
	)

	result, err := adapter.Execute(context.Background(), b)
	if !errors.Is(err, ErrBuildExecutionInProgress) {
		t.Fatalf("expected ErrBuildExecutionInProgress, got err=%v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result while in progress, got %#v", result)
	}
}

func TestMethodExecutorAdapter_ReturnsResultOnSuccess(t *testing.T) {
	b := makeMethodAdapterTestBuild(t)
	adapter := NewMethodExecutorAdapter(
		&stubMethodExecutorAdapterFactory{
			executor: &stubMethodExecutorAdapterExec{
				output: &MethodExecutionOutput{
					Status:   ExecutionSuccess,
					Output:   "done",
					Duration: 1,
				},
			},
		},
		zap.NewNop(),
	)

	result, err := adapter.Execute(context.Background(), b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Logs) != 1 || result.Logs[0] != "done" {
		t.Fatalf("unexpected result logs: %#v", result.Logs)
	}
}
