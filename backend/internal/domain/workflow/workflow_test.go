package workflow

import (
	"testing"

	"github.com/google/uuid"
)

func TestWorkflowConstantsAndErrors(t *testing.T) {
	if ErrInvalidDefinition == nil || ErrInvalidInstance == nil || ErrInvalidStep == nil {
		t.Fatal("expected workflow error variables to be initialized")
	}

	if InstanceStatusRunning == "" || InstanceStatusBlocked == "" || InstanceStatusFailed == "" || InstanceStatusCompleted == "" {
		t.Fatal("expected instance status constants to be non-empty")
	}
	if StepStatusPending == "" || StepStatusRunning == "" || StepStatusSucceeded == "" || StepStatusFailed == "" || StepStatusBlocked == "" {
		t.Fatal("expected step status constants to be non-empty")
	}
}

func TestWorkflowStructsConstructible(t *testing.T) {
	def := Definition{
		ID:         uuid.New(),
		Name:       "wf",
		Version:    1,
		Definition: map[string]interface{}{"steps": []string{"a"}},
	}
	if def.Name != "wf" || def.Version != 1 {
		t.Fatalf("unexpected definition values: %+v", def)
	}

	inst := Instance{
		ID:           uuid.New(),
		DefinitionID: def.ID,
		SubjectType:  "build",
		SubjectID:    uuid.New(),
		Status:       InstanceStatusRunning,
	}
	step := Step{
		ID:         uuid.New(),
		InstanceID: inst.ID,
		StepKey:    "dispatch",
		Status:     StepStatusPending,
	}
	ev := Event{
		ID:         uuid.New(),
		InstanceID: inst.ID,
		Type:       "step.created",
	}
	if step.StepKey == "" || ev.Type == "" {
		t.Fatalf("unexpected zero values: step=%+v event=%+v", step, ev)
	}
}
