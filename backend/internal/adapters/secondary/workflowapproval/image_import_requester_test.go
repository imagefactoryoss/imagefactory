package workflowapproval

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/workflow"
)

type workflowRepoStub struct {
	definitionName string
	definitionVer  int
	subjectType    string
	subjectID      uuid.UUID
	instanceStatus workflow.InstanceStatus
	steps          []workflow.StepDefinition
}

func (s *workflowRepoStub) ClaimNextRunnableStep(ctx context.Context) (*workflow.Step, error) {
	return nil, nil
}

func (s *workflowRepoStub) UpdateStep(ctx context.Context, step *workflow.Step) error {
	return nil
}

func (s *workflowRepoStub) AppendEvent(ctx context.Context, event *workflow.Event) error {
	return nil
}

func (s *workflowRepoStub) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	s.definitionName = name
	s.definitionVer = version
	return uuid.New(), nil
}

func (s *workflowRepoStub) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status workflow.InstanceStatus) (uuid.UUID, error) {
	s.subjectType = subjectType
	s.subjectID = subjectID
	s.instanceStatus = status
	return uuid.New(), nil
}

func (s *workflowRepoStub) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []workflow.StepDefinition) error {
	s.steps = steps
	return nil
}

func (s *workflowRepoStub) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status workflow.InstanceStatus) error {
	return nil
}

func (s *workflowRepoStub) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status workflow.StepStatus, errMsg *string) error {
	return nil
}

func (s *workflowRepoStub) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*workflow.Instance, []workflow.Step, error) {
	return nil, nil, nil
}

func (s *workflowRepoStub) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*workflow.BlockedStepDiagnostics, error) {
	return nil, nil
}

func TestImageImportApprovalRequester_CreateImportApproval(t *testing.T) {
	repo := &workflowRepoStub{}
	requester := NewImageImportApprovalRequester(repo)

	req, err := imageimport.NewImportRequest(uuid.New(), uuid.New(), imageimport.RequestTypeQuarantine, "APP-123", "ghcr.io", "ghcr.io/org/app:1.0.0", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if err := requester.CreateImportApproval(context.Background(), req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.definitionName != imageImportApprovalWorkflowName || repo.definitionVer != imageImportApprovalWorkflowVersion {
		t.Fatalf("unexpected workflow definition upsert: %s@%d", repo.definitionName, repo.definitionVer)
	}
	if repo.subjectType != "external_image_import" {
		t.Fatalf("unexpected subject type: %s", repo.subjectType)
	}
	if repo.subjectID != req.ID {
		t.Fatalf("unexpected subject id: %s", repo.subjectID.String())
	}
	if len(repo.steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(repo.steps))
	}
	if repo.instanceStatus != workflow.InstanceStatusRunning {
		t.Fatalf("expected running instance status, got %s", repo.instanceStatus)
	}
	if repo.steps[0].Status != workflow.StepStatusPending {
		t.Fatalf("expected first step pending, got %s", repo.steps[0].Status)
	}
	if repo.steps[3].StepKey != "import.monitor" {
		t.Fatalf("expected import.monitor as final step, got %s", repo.steps[3].StepKey)
	}
}
