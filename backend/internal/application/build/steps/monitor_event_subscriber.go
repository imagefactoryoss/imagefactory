package steps

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

// MonitorEventSubscriberSnapshot captures diagnostics from event-driven monitor subscriber.
type MonitorEventSubscriberSnapshot struct {
	EventsReceived   int64
	ParseFailures    int64
	WorkflowMissing  int64
	Transitioned     int64
	TransitionErrors int64
	NoopTerminal     int64
	MonitorFound     int64
	MonitorPending   int64
	MonitorRunning   int64
	MonitorTerminal  int64
	FinalizePending  int64
	FinalizeRunning  int64
	FinalizeTerminal int64
}

// BuildMonitorEventSubscriber observes terminal execution events and applies idempotent
// monitor-step terminal transitions while recording diagnostics snapshots.
type BuildMonitorEventSubscriber struct {
	repo             domainworkflow.Repository
	logger           *zap.Logger
	eventsReceived   atomic.Int64
	parseFailures    atomic.Int64
	workflowMissing  atomic.Int64
	transitioned     atomic.Int64
	transitionErrors atomic.Int64
	noopTerminal     atomic.Int64
	monitorFound     atomic.Int64
	monitorPending   atomic.Int64
	monitorRunning   atomic.Int64
	monitorTerminal  atomic.Int64
	finalizePending  atomic.Int64
	finalizeRunning  atomic.Int64
	finalizeTerminal atomic.Int64
}

func NewBuildMonitorEventSubscriber(repo domainworkflow.Repository, logger *zap.Logger) *BuildMonitorEventSubscriber {
	return &BuildMonitorEventSubscriber{
		repo:   repo,
		logger: logger,
	}
}

func (s *BuildMonitorEventSubscriber) Snapshot() MonitorEventSubscriberSnapshot {
	return MonitorEventSubscriberSnapshot{
		EventsReceived:   s.eventsReceived.Load(),
		ParseFailures:    s.parseFailures.Load(),
		WorkflowMissing:  s.workflowMissing.Load(),
		Transitioned:     s.transitioned.Load(),
		TransitionErrors: s.transitionErrors.Load(),
		NoopTerminal:     s.noopTerminal.Load(),
		MonitorFound:     s.monitorFound.Load(),
		MonitorPending:   s.monitorPending.Load(),
		MonitorRunning:   s.monitorRunning.Load(),
		MonitorTerminal:  s.monitorTerminal.Load(),
		FinalizePending:  s.finalizePending.Load(),
		FinalizeRunning:  s.finalizeRunning.Load(),
		FinalizeTerminal: s.finalizeTerminal.Load(),
	}
}

func (s *BuildMonitorEventSubscriber) HandleExecutionTerminalEvent(ctx context.Context, event messaging.Event) {
	s.eventsReceived.Add(1)
	if s.repo == nil {
		return
	}

	buildIDStr, _ := event.Payload["build_id"].(string)
	buildIDStr = strings.TrimSpace(buildIDStr)
	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		s.parseFailures.Add(1)
		if s.logger != nil {
			s.logger.Debug("Build monitor event subscriber ignored event with invalid build_id",
				zap.String("event_type", event.Type),
				zap.String("build_id", buildIDStr),
				zap.Error(err))
		}
		return
	}

	instance, steps, err := s.repo.GetInstanceWithStepsBySubject(ctx, "build", buildID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("Build monitor event subscriber failed to load workflow state",
				zap.String("event_type", event.Type),
				zap.String("build_id", buildID.String()),
				zap.Error(err))
		}
		return
	}
	if instance == nil {
		s.workflowMissing.Add(1)
		if s.logger != nil {
			s.logger.Debug("Build monitor event subscriber found no workflow instance",
				zap.String("event_type", event.Type),
				zap.String("build_id", buildID.String()))
		}
		return
	}

	var monitorStatus domainworkflow.StepStatus
	var finalizeStatus domainworkflow.StepStatus
	for _, step := range steps {
		switch step.StepKey {
		case StepMonitorBuild:
			monitorStatus = step.Status
		case StepFinalizeBuild:
			finalizeStatus = step.Status
		}
	}

	if monitorStatus != "" {
		s.monitorFound.Add(1)
		switch monitorStatus {
		case domainworkflow.StepStatusPending:
			s.monitorPending.Add(1)
		case domainworkflow.StepStatusRunning:
			s.monitorRunning.Add(1)
		default:
			s.monitorTerminal.Add(1)
		}
	}

	switch finalizeStatus {
	case domainworkflow.StepStatusPending:
		s.finalizePending.Add(1)
	case domainworkflow.StepStatusRunning:
		s.finalizeRunning.Add(1)
	case domainworkflow.StepStatusSucceeded, domainworkflow.StepStatusFailed:
		s.finalizeTerminal.Add(1)
	}

	var monitorStepID *uuid.UUID
	for _, step := range steps {
		if step.StepKey == StepMonitorBuild {
			id := step.ID
			monitorStepID = &id
			break
		}
	}

	targetStatus, errMsg := desiredMonitorStepOutcome(event)
	if monitorStatus == targetStatus {
		s.noopTerminal.Add(1)
		if s.logger != nil {
			s.logger.Debug("Build monitor event transition skipped: step already terminal",
				zap.String("event_type", event.Type),
				zap.String("build_id", buildID.String()),
				zap.String("workflow_instance_id", instance.ID.String()),
				zap.String("monitor_step_status", string(monitorStatus)))
		}
		return
	}

	if updateErr := s.repo.UpdateStepStatus(ctx, instance.ID, StepMonitorBuild, targetStatus, errMsg); updateErr != nil {
		s.transitionErrors.Add(1)
		if s.logger != nil {
			s.logger.Warn("Build monitor event transition failed",
				zap.String("event_type", event.Type),
				zap.String("build_id", buildID.String()),
				zap.String("workflow_instance_id", instance.ID.String()),
				zap.Error(updateErr))
		}
		return
	}
	s.transitioned.Add(1)
	now := time.Now().UTC()
	payload := map[string]interface{}{
		"source":                 "event_subscriber",
		"build_id":               buildID.String(),
		"execution_event_type":   event.Type,
		"monitor_target_status":  string(targetStatus),
		"finalize_current_state": string(finalizeStatus),
	}
	if errMsg != nil {
		payload["error"] = *errMsg
	}
	_ = s.repo.AppendEvent(ctx, &domainworkflow.Event{
		ID:         uuid.New(),
		InstanceID: instance.ID,
		StepID:     monitorStepID,
		Type:       "workflow.step." + string(targetStatus),
		Payload:    payload,
		CreatedAt:  now,
	})

	if s.logger != nil {
		s.logger.Info("Build monitor event diagnostic snapshot",
			zap.String("event_type", event.Type),
			zap.String("build_id", buildID.String()),
			zap.String("workflow_instance_id", instance.ID.String()),
			zap.String("monitor_step_status", string(monitorStatus)),
			zap.String("finalize_step_status", string(finalizeStatus)),
			zap.String("summary", fmt.Sprintf("%+v", s.Snapshot())),
		)
	}
}

func desiredMonitorStepOutcome(event messaging.Event) (domainworkflow.StepStatus, *string) {
	switch event.Type {
	case messaging.EventTypeBuildExecutionCompleted:
		return domainworkflow.StepStatusSucceeded, nil
	case messaging.EventTypeBuildExecutionFailed:
		msg := "build execution failed"
		if event.Payload != nil {
			if raw, ok := event.Payload["message"].(string); ok && strings.TrimSpace(raw) != "" {
				msg = strings.TrimSpace(raw)
			}
		}
		return domainworkflow.StepStatusFailed, &msg
	default:
		status := strings.TrimSpace(strings.ToLower(stringValue(event.Payload, "status")))
		if status == "completed" || status == "success" || status == "succeeded" {
			return domainworkflow.StepStatusSucceeded, nil
		}
		msg := "build execution failed"
		if raw := strings.TrimSpace(stringValue(event.Payload, "message")); raw != "" {
			msg = raw
		}
		return domainworkflow.StepStatusFailed, &msg
	}
}

func stringValue(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func RegisterBuildMonitorEventSubscriber(bus messaging.EventBus, subscriber *BuildMonitorEventSubscriber) func() {
	if bus == nil || subscriber == nil {
		return func() {}
	}
	unsubscribers := []func(){
		bus.Subscribe(messaging.EventTypeBuildExecutionCompleted, subscriber.HandleExecutionTerminalEvent),
		bus.Subscribe(messaging.EventTypeBuildExecutionFailed, subscriber.HandleExecutionTerminalEvent),
	}
	return func() {
		for _, unsub := range unsubscribers {
			unsub()
		}
	}
}
