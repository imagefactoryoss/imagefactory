package workflow

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines persistence operations for workflow engine.
type Repository interface {
	ClaimNextRunnableStep(ctx context.Context) (*Step, error)
	UpdateStep(ctx context.Context, step *Step) error
	AppendEvent(ctx context.Context, event *Event) error
	UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error)
	CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status InstanceStatus) (uuid.UUID, error)
	CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []StepDefinition) error
	UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status InstanceStatus) error
	UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status StepStatus, errMsg *string) error
	GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*Instance, []Step, error)
	GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*BlockedStepDiagnostics, error)
}
