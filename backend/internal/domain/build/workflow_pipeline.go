package build

import "github.com/srikarm/image-factory/internal/domain/workflow"

const (
	buildWorkflowName        = "build_pipeline"
	buildWorkflowVersion     = 1
	buildWorkflowSubjectType = "build_execution"
)

var buildWorkflowStepKeys = []string{
	"queued",
	"validated",
	"dispatched",
	"build",
	"scan",
	"sbom",
	"publish",
	"complete",
}

func buildWorkflowDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":    buildWorkflowName,
		"version": buildWorkflowVersion,
		"steps":   buildWorkflowStepKeys,
	}
}

func buildWorkflowSteps(payload map[string]interface{}) []workflow.StepDefinition {
	steps := make([]workflow.StepDefinition, 0, len(buildWorkflowStepKeys))
	for _, key := range buildWorkflowStepKeys {
		steps = append(steps, workflow.StepDefinition{
			StepKey: key,
			Payload: payload,
			Status:  workflow.StepStatusPending,
		})
	}
	return steps
}
