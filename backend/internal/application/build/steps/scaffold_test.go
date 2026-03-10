package steps

import (
	"context"
	"testing"

	appworkflow "github.com/srikarm/image-factory/internal/application/workflow"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

func TestBuildStepKeys_OrderAndCount(t *testing.T) {
	keys := BuildStepKeys()
	if len(keys) != 6 {
		t.Fatalf("expected 6 scaffold step keys, got %d", len(keys))
	}
	expected := []string{
		StepValidateBuild,
		StepSelectInfrastructure,
		StepEnqueueBuild,
		StepDispatchBuild,
		StepMonitorBuild,
		StepFinalizeBuild,
	}
	for i := range expected {
		if keys[i] != expected[i] {
			t.Fatalf("unexpected key at index %d: expected %s got %s", i, expected[i], keys[i])
		}
	}
}

func TestNewPhase2ScaffoldHandlers(t *testing.T) {
	handlers := NewPhase2ScaffoldHandlers(zap.NewNop())
	if len(handlers) != len(BuildStepKeys()) {
		t.Fatalf("expected %d handlers, got %d", len(BuildStepKeys()), len(handlers))
	}
	for i, handler := range handlers {
		if handler.Key() != BuildStepKeys()[i] {
			t.Fatalf("handler key mismatch at index %d: expected %s got %s", i, BuildStepKeys()[i], handler.Key())
		}
	}
}

func TestScaffoldHandler_FailsFast(t *testing.T) {
	handlers := NewPhase2ScaffoldHandlers(zap.NewNop())
	handler := handlers[0]
	result, err := handler.Execute(context.Background(), &domainworkflow.Step{StepKey: handler.Key()})
	if err == nil {
		t.Fatal("expected fail-fast scaffold error")
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
}

var _ appworkflow.StepHandler = (*phase2ScaffoldStep)(nil)
