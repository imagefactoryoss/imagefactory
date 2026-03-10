package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

// StepHandler executes a workflow step.
type StepHandler interface {
	Key() string
	Execute(ctx context.Context, step *workflow.Step) (StepResult, error)
}

// StepResult represents the outcome of a step execution.
type StepResult struct {
	Status workflow.StepStatus
	Data   map[string]interface{}
	Error  string
}

// Orchestrator polls runnable steps and executes handlers.
type Orchestrator struct {
	repo     workflow.Repository
	handlers map[string]StepHandler
	logger   *zap.Logger
}

func NewOrchestrator(repo workflow.Repository, handlers []StepHandler, logger *zap.Logger) *Orchestrator {
	index := make(map[string]StepHandler, len(handlers))
	for _, h := range handlers {
		index[h.Key()] = h
	}
	return &Orchestrator{
		repo:     repo,
		handlers: index,
		logger:   logger,
	}
}

// Run starts the orchestration loop until context cancellation.
func (o *Orchestrator) Run(ctx context.Context, pollInterval time.Duration, maxStepsPerTick int) {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	if maxStepsPerTick <= 0 {
		maxStepsPerTick = 1
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	o.logger.Info("Workflow orchestrator started",
		zap.Duration("poll_interval", pollInterval),
		zap.Int("max_steps_per_tick", maxStepsPerTick),
	)

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("Workflow orchestrator stopped")
			return
		default:
		}

		for i := 0; i < maxStepsPerTick; i++ {
			ran, err := o.RunOnce(ctx)
			if err != nil {
				o.logger.Error("Workflow orchestrator step failed", zap.Error(err))
				break
			}
			if !ran {
				break
			}
		}

		select {
		case <-ctx.Done():
			o.logger.Info("Workflow orchestrator stopped")
			return
		case <-ticker.C:
		}
	}
}

// RunOnce claims and executes a single runnable step.
func (o *Orchestrator) RunOnce(ctx context.Context) (bool, error) {
	step, err := o.repo.ClaimNextRunnableStep(ctx)
	if err != nil {
		return false, err
	}
	if step == nil {
		return false, nil
	}

	handler, ok := o.handlers[step.StepKey]
	if !ok {
		errMsg := fmt.Sprintf("handler not found for step key: %s", step.StepKey)
		o.failStep(ctx, step, errMsg)
		return true, fmt.Errorf("%s", errMsg)
	}

	result, err := handler.Execute(ctx, step)
	if err != nil {
		o.failStep(ctx, step, err.Error())
		return true, err
	}
	if result.Status == "" {
		result.Status = workflow.StepStatusSucceeded
	}
	// Blocked steps are retried by placing them back in pending state.
	if result.Status == workflow.StepStatusBlocked {
		result.Status = workflow.StepStatusPending
	}

	now := time.Now().UTC()
	step.Status = result.Status
	if step.Status == workflow.StepStatusSucceeded || step.Status == workflow.StepStatusFailed {
		step.CompletedAt = &now
	} else {
		step.CompletedAt = nil
	}
	if result.Error != "" {
		step.LastError = &result.Error
	}

	if err := o.repo.UpdateStep(ctx, step); err != nil {
		return true, err
	}

	eventID := uuid.New()
	_ = o.repo.AppendEvent(ctx, &workflow.Event{
		ID:         eventID,
		InstanceID: step.InstanceID,
		StepID:     &step.ID,
		Type:       "workflow.step." + string(result.Status),
		Payload:    result.Data,
		CreatedAt:  now,
	})

	return true, nil
}

func (o *Orchestrator) failStep(ctx context.Context, step *workflow.Step, errMsg string) {
	now := time.Now().UTC()
	step.Status = workflow.StepStatusFailed
	step.CompletedAt = &now
	step.LastError = &errMsg
	_ = o.repo.UpdateStep(ctx, step)
	eventID := uuid.New()
	_ = o.repo.AppendEvent(ctx, &workflow.Event{
		ID:         eventID,
		InstanceID: step.InstanceID,
		StepID:     &step.ID,
		Type:       "workflow.step.failed",
		Payload:    map[string]interface{}{"error": errMsg},
		CreatedAt:  now,
	})
}
