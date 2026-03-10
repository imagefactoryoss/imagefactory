package workflowapproval

import (
	"context"
	"fmt"

	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/workflow"
)

const (
	imageImportApprovalWorkflowName    = "external_image_import_approval"
	imageImportApprovalWorkflowVersion = 1
)

type ImageImportApprovalRequester struct {
	workflowRepo workflow.Repository
}

func NewImageImportApprovalRequester(workflowRepo workflow.Repository) *ImageImportApprovalRequester {
	return &ImageImportApprovalRequester{workflowRepo: workflowRepo}
}

func (r *ImageImportApprovalRequester) CreateImportApproval(ctx context.Context, req *imageimport.ImportRequest) error {
	if r == nil || r.workflowRepo == nil || req == nil {
		return nil
	}

	definition := map[string]interface{}{
		"name":    imageImportApprovalWorkflowName,
		"version": imageImportApprovalWorkflowVersion,
		"steps": []string{
			"approval.request",
			"approval.decision",
			"import.dispatch",
			"import.monitor",
		},
	}

	definitionID, err := r.workflowRepo.UpsertDefinition(ctx, imageImportApprovalWorkflowName, imageImportApprovalWorkflowVersion, definition)
	if err != nil {
		return fmt.Errorf("failed to upsert image import approval workflow definition: %w", err)
	}

	tenantID := req.TenantID
	instanceID, err := r.workflowRepo.CreateInstance(
		ctx,
		definitionID,
		&tenantID,
		"external_image_import",
		req.ID,
		workflow.InstanceStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("failed to create image import approval workflow instance: %w", err)
	}

	payload := map[string]interface{}{
		"external_image_import_id": req.ID.String(),
		"tenant_id":                req.TenantID.String(),
		"requested_by_user_id":     req.RequestedByUserID.String(),
		"request_type":             string(req.RequestType),
		"sor_record_id":            req.SORRecordID,
		"source_registry":          req.SourceRegistry,
		"source_image_ref":         req.SourceImageRef,
	}

	steps := []workflow.StepDefinition{
		{
			StepKey: "approval.request",
			Payload: payload,
			Status:  workflow.StepStatusPending,
		},
		{
			StepKey: "approval.decision",
			Payload: payload,
			Status:  workflow.StepStatusBlocked,
		},
		{
			StepKey: "import.dispatch",
			Payload: payload,
			Status:  workflow.StepStatusBlocked,
		},
		{
			StepKey: "import.monitor",
			Payload: payload,
			Status:  workflow.StepStatusBlocked,
		},
	}

	if err := r.workflowRepo.CreateSteps(ctx, instanceID, steps); err != nil {
		return fmt.Errorf("failed to create image import approval workflow steps: %w", err)
	}

	return nil
}
