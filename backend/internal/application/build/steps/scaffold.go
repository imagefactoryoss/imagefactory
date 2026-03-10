package steps

import (
	"context"
	"errors"
	"fmt"

	appworkflow "github.com/srikarm/image-factory/internal/application/workflow"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

const (
	StepValidateBuild        = "build.validate"
	StepSelectInfrastructure = "build.select_infrastructure"
	StepEnqueueBuild         = "build.enqueue"
	StepDispatchBuild        = "build.dispatch"
	StepMonitorBuild         = "build.monitor"
	StepFinalizeBuild        = "build.finalize"
)

// BuildStepKeys defines the planned Phase 2 control-plane steps in order.
func BuildStepKeys() []string {
	return []string{
		StepValidateBuild,
		StepSelectInfrastructure,
		StepEnqueueBuild,
		StepDispatchBuild,
		StepMonitorBuild,
		StepFinalizeBuild,
	}
}

// BuildWorkflowPayload is the common payload contract for build orchestration steps.
type BuildWorkflowPayload struct {
	TenantID    string `json:"tenant_id"`
	ProjectID   string `json:"project_id"`
	BuildID     string `json:"build_id,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
}

type phase2ScaffoldStep struct {
	key    string
	logger *zap.Logger
}

func (s *phase2ScaffoldStep) Key() string {
	return s.key
}

func (s *phase2ScaffoldStep) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	msg := fmt.Sprintf("workflow step %s is scaffolded but not wired", s.key)
	s.logger.Warn(msg)
	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusFailed,
		Error:  msg,
	}, errors.New(msg)
}

// NewPhase2ScaffoldHandlers returns placeholder step handlers for Phase 2.
// They are intentionally fail-fast and must not be wired into runtime until each step is implemented.
func NewPhase2ScaffoldHandlers(logger *zap.Logger) []appworkflow.StepHandler {
	keys := BuildStepKeys()
	handlers := make([]appworkflow.StepHandler, 0, len(keys))
	for _, key := range keys {
		handlers = append(handlers, &phase2ScaffoldStep{key: key, logger: logger})
	}
	return handlers
}
