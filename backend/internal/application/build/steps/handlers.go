package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	appworkflow "github.com/srikarm/image-factory/internal/application/workflow"
	domainbuild "github.com/srikarm/image-factory/internal/domain/build"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

var (
	ErrInvalidWorkflowPayload = errors.New("invalid build workflow payload")
)

// InfrastructurePreflight resolves and validates infrastructure selection for a build manifest.
type InfrastructurePreflight func(ctx context.Context, tenantID uuid.UUID, manifest *domainbuild.BuildManifest) error

// BuildControlPlaneService exposes build operations required by workflow step handlers.
type BuildControlPlaneService interface {
	CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest domainbuild.BuildManifest, actorID *uuid.UUID) (*domainbuild.Build, error)
	StartBuild(ctx context.Context, buildID uuid.UUID) error
	GetBuild(ctx context.Context, id uuid.UUID) (*domainbuild.Build, error)
	GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]domainbuild.BuildExecution, int64, error)
	MarkBuildFailed(ctx context.Context, buildID uuid.UUID, reason string) error
}

type ValidateBuildHandler struct {
	logger *zap.Logger
}

func NewValidateBuildHandler(logger *zap.Logger) *ValidateBuildHandler {
	return &ValidateBuildHandler{logger: logger}
}

func (h *ValidateBuildHandler) Key() string { return StepValidateBuild }

func (h *ValidateBuildHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	payload, err := parseBuildWorkflowPayload(step.Payload)
	if err != nil {
		return failStepResult(ErrInvalidWorkflowPayload, err)
	}

	if payload.TenantUUID == uuid.Nil || payload.ProjectUUID == uuid.Nil {
		return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("tenant_id and project_id are required"))
	}
	if payload.Manifest.Name == "" {
		return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("manifest.name is required"))
	}
	if payload.Manifest.Type == "" {
		return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("manifest.type is required"))
	}

	if payload.Manifest.Type == domainbuild.BuildTypeContainer {
		if payload.Manifest.BaseImage == "" {
			return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("manifest.base_image is required for container builds"))
		}
		if len(payload.Manifest.Instructions) == 0 {
			return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("manifest.instructions are required for container builds"))
		}
	}

	h.logger.Debug("Build workflow validate step passed",
		zap.String("tenant_id", payload.TenantUUID.String()),
		zap.String("project_id", payload.ProjectUUID.String()),
		zap.String("build_type", string(payload.Manifest.Type)),
	)

	return appworkflow.StepResult{Status: domainworkflow.StepStatusSucceeded}, nil
}

type SelectInfrastructureHandler struct {
	preflight InfrastructurePreflight
	logger    *zap.Logger
}

func NewSelectInfrastructureHandler(preflight InfrastructurePreflight, logger *zap.Logger) *SelectInfrastructureHandler {
	return &SelectInfrastructureHandler{preflight: preflight, logger: logger}
}

func (h *SelectInfrastructureHandler) Key() string { return StepSelectInfrastructure }

func (h *SelectInfrastructureHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	if h.preflight == nil {
		return failStepResult(errors.New("build infrastructure preflight is not configured"), nil)
	}

	payload, err := parseBuildWorkflowPayload(step.Payload)
	if err != nil {
		return failStepResult(ErrInvalidWorkflowPayload, err)
	}

	manifest := payload.Manifest
	if err := h.preflight(ctx, payload.TenantUUID, &manifest); err != nil {
		return failStepResult(err, nil)
	}

	if err := setManifestOnStepPayload(step, manifest); err != nil {
		return failStepResult(err, nil)
	}

	h.logger.Debug("Build workflow infrastructure selected",
		zap.String("tenant_id", payload.TenantUUID.String()),
		zap.String("infrastructure_type", manifest.InfrastructureType),
	)

	return appworkflow.StepResult{Status: domainworkflow.StepStatusSucceeded}, nil
}

type EnqueueBuildHandler struct {
	buildService BuildControlPlaneService
	logger       *zap.Logger
}

func NewEnqueueBuildHandler(buildService BuildControlPlaneService, logger *zap.Logger) *EnqueueBuildHandler {
	return &EnqueueBuildHandler{buildService: buildService, logger: logger}
}

func (h *EnqueueBuildHandler) Key() string { return StepEnqueueBuild }

func (h *EnqueueBuildHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	if h.buildService == nil {
		return failStepResult(errors.New("build service is not configured"), nil)
	}
	payload, err := parseBuildWorkflowPayload(step.Payload)
	if err != nil {
		return failStepResult(ErrInvalidWorkflowPayload, err)
	}

	created, err := h.buildService.CreateBuild(ctx, payload.TenantUUID, payload.ProjectUUID, payload.Manifest, nil)
	if err != nil {
		return failStepResult(err, nil)
	}

	if err := setBuildIDOnStepPayload(step, created.ID()); err != nil {
		return failStepResult(err, nil)
	}

	h.logger.Debug("Build workflow enqueue step created build",
		zap.String("build_id", created.ID().String()),
		zap.String("tenant_id", payload.TenantUUID.String()),
	)

	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusSucceeded,
		Data: map[string]interface{}{
			"build_id": created.ID().String(),
		},
	}, nil
}

type DispatchBuildHandler struct {
	buildService BuildControlPlaneService
	logger       *zap.Logger
}

func NewDispatchBuildHandler(buildService BuildControlPlaneService, logger *zap.Logger) *DispatchBuildHandler {
	return &DispatchBuildHandler{buildService: buildService, logger: logger}
}

func (h *DispatchBuildHandler) Key() string { return StepDispatchBuild }

func (h *DispatchBuildHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	if h.buildService == nil {
		return failStepResult(errors.New("build service is not configured"), nil)
	}
	payload, err := parseBuildWorkflowPayload(step.Payload)
	if err != nil {
		return failStepResult(ErrInvalidWorkflowPayload, err)
	}
	if payload.BuildUUID == uuid.Nil {
		return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("build_id is required for dispatch"))
	}

	// The build may already have been started (or even completed/failed) by another control-plane component
	// (e.g. dispatcher) by the time this step runs. Only attempt to start when the build is still queued.
	b, err := h.buildService.GetBuild(ctx, payload.BuildUUID)
	if err != nil || b == nil {
		return failStepResult(domainbuild.ErrBuildNotFound, err)
	}

	switch b.Status() {
	case domainbuild.BuildStatusQueued:
		if err := h.buildService.StartBuild(ctx, payload.BuildUUID); err != nil {
			return failStepResult(err, nil)
		}
	case domainbuild.BuildStatusRunning, domainbuild.BuildStatusCompleted, domainbuild.BuildStatusFailed, domainbuild.BuildStatusCancelled:
		// No-op: let monitor/finalize steps report final state and logs.
		h.logger.Debug("Build workflow dispatch step skipped start due to current status",
			zap.String("build_id", payload.BuildUUID.String()),
			zap.String("status", string(b.Status())),
		)
	default:
		// Pending indicates the build was created but not queued; this is unexpected for workflow-managed builds.
		return failStepResult(fmt.Errorf("build is not dispatchable (status=%s)", b.Status()), nil)
	}

	h.logger.Debug("Build workflow dispatch step started build", zap.String("build_id", payload.BuildUUID.String()))
	return appworkflow.StepResult{Status: domainworkflow.StepStatusSucceeded}, nil
}

type MonitorBuildHandler struct {
	buildService BuildControlPlaneService
	logger       *zap.Logger
}

const (
	orphanedExecutionDetectionAttempts = 3
	defaultBuildMonitorTimeout         = 2 * time.Hour
	minBuildMonitorTimeout             = 1 * time.Minute
)

func NewMonitorBuildHandler(buildService BuildControlPlaneService, logger *zap.Logger) *MonitorBuildHandler {
	return &MonitorBuildHandler{buildService: buildService, logger: logger}
}

func (h *MonitorBuildHandler) Key() string { return StepMonitorBuild }

func (h *MonitorBuildHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	if h.buildService == nil {
		return failStepResult(errors.New("build service is not configured"), nil)
	}
	payload, err := parseBuildWorkflowPayload(step.Payload)
	if err != nil {
		return failStepResult(ErrInvalidWorkflowPayload, err)
	}
	if payload.BuildUUID == uuid.Nil {
		return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("build_id is required for monitor"))
	}

	if notBefore, due := monitorCheckDueAt(step.Payload, time.Now().UTC()); !due {
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  "monitor waiting for next check window",
			Data: map[string]interface{}{
				"monitor_next_check_at": notBefore.Format(time.RFC3339),
			},
		}, nil
	}

	executions, _, execErr := h.buildService.GetBuildExecutions(ctx, payload.BuildUUID, 1, 0)
	if execErr != nil {
		h.logger.Warn("Build monitor: failed to inspect latest execution",
			zap.String("build_id", payload.BuildUUID.String()),
			zap.Error(execErr))
	}
	if len(executions) > 0 {
		latest := executions[0]
		switch latest.Status {
		case domainbuild.ExecutionFailed, domainbuild.ExecutionCancelled:
			reason := latest.ErrorMessage
			if reason == "" {
				reason = fmt.Sprintf("build execution ended in terminal state: %s", latest.Status)
			}
			if failErr := h.buildService.MarkBuildFailed(ctx, payload.BuildUUID, reason); failErr != nil {
				h.logger.Warn("Build monitor: failed to propagate terminal execution state to build",
					zap.String("build_id", payload.BuildUUID.String()),
					zap.String("execution_id", latest.ID.String()),
					zap.Error(failErr))
			}
			return failStepResult(errors.New(reason), nil)
		case domainbuild.ExecutionRunning, domainbuild.ExecutionPending:
			if reason, exceeded := monitorTimeoutFailureReason(nil, &latest, time.Now().UTC()); exceeded {
				if failErr := h.buildService.MarkBuildFailed(ctx, payload.BuildUUID, reason); failErr != nil {
					h.logger.Warn("Build monitor: failed to mark timed out build as failed",
						zap.String("build_id", payload.BuildUUID.String()),
						zap.String("execution_id", latest.ID.String()),
						zap.Error(failErr))
				} else {
					return failStepResult(errors.New(reason), nil)
				}
			}
			if step.Attempts >= orphanedExecutionDetectionAttempts && !executionHasTektonRef(latest.Metadata) {
				reason := fmt.Sprintf("orphaned build execution detected: execution %s missing tekton metadata", latest.ID.String())
				if failErr := h.buildService.MarkBuildFailed(ctx, payload.BuildUUID, reason); failErr != nil {
					h.logger.Warn("Build monitor: failed to mark orphaned build as failed",
						zap.String("build_id", payload.BuildUUID.String()),
						zap.String("execution_id", latest.ID.String()),
						zap.Error(failErr))
				} else {
					return failStepResult(errors.New(reason), nil)
				}
			}
			next := setMonitorNextCheckAt(step.Payload, time.Now().UTC(), nextMonitorBackoff(step.Attempts))
			return appworkflow.StepResult{
				Status: domainworkflow.StepStatusBlocked,
				Error:  fmt.Sprintf("build execution is still in progress: %s", latest.Status),
				Data: map[string]interface{}{
					"execution_status":      string(latest.Status),
					"monitor_next_check_at": next.Format(time.RFC3339),
				},
			}, nil
		}
	}

	b, err := h.buildService.GetBuild(ctx, payload.BuildUUID)
	if err != nil || b == nil {
		return failStepResult(domainbuild.ErrBuildNotFound, err)
	}

	switch b.Status() {
	case domainbuild.BuildStatusCompleted:
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusSucceeded,
			Data: map[string]interface{}{
				"build_status": string(b.Status()),
			},
		}, nil
	case domainbuild.BuildStatusFailed, domainbuild.BuildStatusCancelled:
		return failStepResult(fmt.Errorf("build reached terminal non-success state: %s", b.Status()), nil)
	default:
		var latestExec *domainbuild.BuildExecution
		if len(executions) > 0 {
			latestExec = &executions[0]
		}
		if reason, exceeded := monitorTimeoutFailureReason(b, latestExec, time.Now().UTC()); exceeded {
			if failErr := h.buildService.MarkBuildFailed(ctx, payload.BuildUUID, reason); failErr != nil {
				h.logger.Warn("Build monitor: failed to mark timed out build as failed",
					zap.String("build_id", payload.BuildUUID.String()),
					zap.Error(failErr))
			} else {
				return failStepResult(errors.New(reason), nil)
			}
		}
		if b.Status() == domainbuild.BuildStatusRunning && b.InfrastructureType() == "kubernetes" && step.Attempts >= orphanedExecutionDetectionAttempts {
			if len(executions) == 0 {
				reason := "orphaned build execution detected: running build has no execution records"
				if failErr := h.buildService.MarkBuildFailed(ctx, payload.BuildUUID, reason); failErr != nil {
					h.logger.Warn("Build monitor: failed to mark orphaned build as failed",
						zap.String("build_id", payload.BuildUUID.String()),
						zap.Error(failErr))
				} else {
					return failStepResult(errors.New(reason), nil)
				}
			} else if !executionHasTektonRef(executions[0].Metadata) {
				reason := fmt.Sprintf("orphaned build execution detected: execution %s missing tekton metadata", executions[0].ID.String())
				if failErr := h.buildService.MarkBuildFailed(ctx, payload.BuildUUID, reason); failErr != nil {
					h.logger.Warn("Build monitor: failed to mark orphaned build as failed",
						zap.String("build_id", payload.BuildUUID.String()),
						zap.String("execution_id", executions[0].ID.String()),
						zap.Error(failErr))
				} else {
					return failStepResult(errors.New(reason), nil)
				}
			}
		}
		// One-shot monitor step: return blocked when build is still in-flight.
		next := setMonitorNextCheckAt(step.Payload, time.Now().UTC(), nextMonitorBackoff(step.Attempts))
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  fmt.Sprintf("build is still in progress: %s", b.Status()),
			Data: map[string]interface{}{
				"build_status":          string(b.Status()),
				"monitor_next_check_at": next.Format(time.RFC3339),
			},
		}, nil
	}
}

func nextMonitorBackoff(attempts int) time.Duration {
	if monitorEventDrivenEnabled() {
		switch {
		case attempts < 5:
			return 10 * time.Second
		case attempts < 20:
			return 30 * time.Second
		default:
			return 60 * time.Second
		}
	}

	switch {
	case attempts < 5:
		return 3 * time.Second
	case attempts < 20:
		return 10 * time.Second
	default:
		return 30 * time.Second
	}
}

func monitorTimeoutFailureReason(build *domainbuild.Build, execution *domainbuild.BuildExecution, now time.Time) (string, bool) {
	timeout := buildMonitorTimeout()
	if timeout <= 0 {
		return "", false
	}

	var startedAt *time.Time
	reference := "build"
	if execution != nil {
		if execution.StartedAt != nil && !execution.StartedAt.IsZero() {
			startedAt = execution.StartedAt
			reference = "execution"
		} else if !execution.UpdatedAt.IsZero() {
			startedAt = &execution.UpdatedAt
			reference = "execution"
		}
	}
	if startedAt == nil && build != nil {
		if build.StartedAt() != nil && !build.StartedAt().IsZero() {
			startedAt = build.StartedAt()
		} else if !build.UpdatedAt().IsZero() {
			updatedAt := build.UpdatedAt()
			startedAt = &updatedAt
		}
	}
	if startedAt == nil || startedAt.IsZero() {
		return "", false
	}

	elapsed := now.Sub(*startedAt)
	if elapsed < timeout {
		return "", false
	}

	return fmt.Sprintf("%s timeout exceeded after %s (limit %s)", reference, elapsed.Round(time.Second), timeout.Round(time.Second)), true
}

func buildMonitorTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("IF_BUILD_MONITOR_TIMEOUT_SECONDS"))
	if raw == "" {
		return defaultBuildMonitorTimeout
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return defaultBuildMonitorTimeout
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout < minBuildMonitorTimeout {
		return minBuildMonitorTimeout
	}
	return timeout
}

var (
	monitorEventDrivenEnabledOnce  sync.Once
	monitorEventDrivenEnabledValue bool
	monitorEventDrivenEnabledMu    sync.RWMutex
	monitorEventDrivenOverride     *bool
)

// SetMonitorEventDrivenEnabledOverride applies a runtime override for monitor event-driven behavior.
// Pass nil to clear override and fall back to IF_BUILD_MONITOR_EVENT_DRIVEN_ENABLED.
func SetMonitorEventDrivenEnabledOverride(value *bool) {
	monitorEventDrivenEnabledMu.Lock()
	defer monitorEventDrivenEnabledMu.Unlock()
	if value == nil {
		monitorEventDrivenOverride = nil
		return
	}
	v := *value
	monitorEventDrivenOverride = &v
}

func monitorEventDrivenEnabled() bool {
	monitorEventDrivenEnabledMu.RLock()
	override := monitorEventDrivenOverride
	monitorEventDrivenEnabledMu.RUnlock()
	if override != nil {
		return *override
	}

	monitorEventDrivenEnabledOnce.Do(func() {
		value := strings.TrimSpace(strings.ToLower(os.Getenv("IF_BUILD_MONITOR_EVENT_DRIVEN_ENABLED")))
		switch value {
		case "1", "true", "yes", "y", "on":
			monitorEventDrivenEnabledValue = true
		default:
			monitorEventDrivenEnabledValue = false
		}
	})
	return monitorEventDrivenEnabledValue
}

func monitorCheckDueAt(payload map[string]interface{}, now time.Time) (time.Time, bool) {
	return checkDueAt(payload, "monitor_next_check_at", now)
}

func finalizeCheckDueAt(payload map[string]interface{}, now time.Time) (time.Time, bool) {
	return checkDueAt(payload, "finalize_next_check_at", now)
}

func checkDueAt(payload map[string]interface{}, key string, now time.Time) (time.Time, bool) {
	if payload == nil {
		return time.Time{}, true
	}
	raw, ok := payload[key]
	if !ok {
		return time.Time{}, true
	}
	value, ok := raw.(string)
	if !ok || value == "" {
		return time.Time{}, true
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, true
	}
	return ts, !now.Before(ts)
}

func setMonitorNextCheckAt(payload map[string]interface{}, now time.Time, backoff time.Duration) time.Time {
	if payload != nil {
		next := now.Add(backoff).UTC()
		payload["monitor_next_check_at"] = next.Format(time.RFC3339)
		return next
	}
	return now.Add(backoff).UTC()
}

func setFinalizeNextCheckAt(payload map[string]interface{}, now time.Time, backoff time.Duration) time.Time {
	if payload != nil {
		next := now.Add(backoff).UTC()
		payload["finalize_next_check_at"] = next.Format(time.RFC3339)
		return next
	}
	return now.Add(backoff).UTC()
}

func executionHasTektonRef(metadata []byte) bool {
	if len(metadata) == 0 || string(metadata) == "null" {
		return false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(metadata, &payload); err != nil || payload == nil {
		return false
	}
	tektonRaw, ok := payload["tekton"]
	if !ok {
		return false
	}
	tekton, ok := tektonRaw.(map[string]interface{})
	if !ok {
		return false
	}
	namespace, _ := tekton["namespace"].(string)
	pipelineRun, _ := tekton["pipeline_run"].(string)
	return namespace != "" && pipelineRun != ""
}

type FinalizeBuildHandler struct {
	buildService BuildControlPlaneService
	logger       *zap.Logger
}

func NewFinalizeBuildHandler(buildService BuildControlPlaneService, logger *zap.Logger) *FinalizeBuildHandler {
	return &FinalizeBuildHandler{buildService: buildService, logger: logger}
}

func (h *FinalizeBuildHandler) Key() string { return StepFinalizeBuild }

func (h *FinalizeBuildHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	if h.buildService == nil {
		return failStepResult(errors.New("build service is not configured"), nil)
	}
	payload, err := parseBuildWorkflowPayload(step.Payload)
	if err != nil {
		return failStepResult(ErrInvalidWorkflowPayload, err)
	}
	if payload.BuildUUID == uuid.Nil {
		return failStepResult(ErrInvalidWorkflowPayload, fmt.Errorf("build_id is required for finalize"))
	}
	if notBefore, due := finalizeCheckDueAt(step.Payload, time.Now().UTC()); !due {
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  "finalize waiting for next check window",
			Data: map[string]interface{}{
				"finalize_next_check_at": notBefore.Format(time.RFC3339),
			},
		}, nil
	}

	b, err := h.buildService.GetBuild(ctx, payload.BuildUUID)
	if err != nil || b == nil {
		return failStepResult(domainbuild.ErrBuildNotFound, err)
	}
	if !b.IsTerminal() {
		next := setFinalizeNextCheckAt(step.Payload, time.Now().UTC(), nextMonitorBackoff(step.Attempts))
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  fmt.Sprintf("build is not terminal: %s", b.Status()),
			Data: map[string]interface{}{
				"finalize_next_check_at": next.Format(time.RFC3339),
			},
		}, nil
	}
	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusSucceeded,
		Data: map[string]interface{}{
			"build_status": string(b.Status()),
		},
	}, nil
}

// NewPhase2ControlPlaneHandlers returns the planned Phase 2 full handler set.
func NewPhase2ControlPlaneHandlers(buildService BuildControlPlaneService, preflight InfrastructurePreflight, logger *zap.Logger) []appworkflow.StepHandler {
	return []appworkflow.StepHandler{
		NewValidateBuildHandler(logger),
		NewSelectInfrastructureHandler(preflight, logger),
		NewEnqueueBuildHandler(buildService, logger),
		NewDispatchBuildHandler(buildService, logger),
		NewMonitorBuildHandler(buildService, logger),
		NewFinalizeBuildHandler(buildService, logger),
	}
}

type buildWorkflowPayload struct {
	TenantID    string                    `json:"tenant_id"`
	ProjectID   string                    `json:"project_id"`
	BuildID     string                    `json:"build_id,omitempty"`
	ExecutionID string                    `json:"execution_id,omitempty"`
	Manifest    domainbuild.BuildManifest `json:"manifest"`

	TenantUUID  uuid.UUID `json:"-"`
	ProjectUUID uuid.UUID `json:"-"`
	BuildUUID   uuid.UUID `json:"-"`
}

func parseBuildWorkflowPayload(raw map[string]interface{}) (*buildWorkflowPayload, error) {
	if raw == nil {
		return nil, fmt.Errorf("payload is required")
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	var payload buildWorkflowPayload
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	payload.TenantUUID, err = uuid.Parse(payload.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}
	payload.ProjectUUID, err = uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project_id: %w", err)
	}
	if payload.BuildID != "" {
		payload.BuildUUID, err = uuid.Parse(payload.BuildID)
		if err != nil {
			return nil, fmt.Errorf("invalid build_id: %w", err)
		}
	}
	return &payload, nil
}

func setManifestOnStepPayload(step *domainworkflow.Step, manifest domainbuild.BuildManifest) error {
	if step.Payload == nil {
		step.Payload = map[string]interface{}{}
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	var manifestMap map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifestMap); err != nil {
		return fmt.Errorf("failed to unmarshal manifest map: %w", err)
	}
	step.Payload["manifest"] = manifestMap
	return nil
}

func setBuildIDOnStepPayload(step *domainworkflow.Step, buildID uuid.UUID) error {
	if step.Payload == nil {
		step.Payload = map[string]interface{}{}
	}
	step.Payload["build_id"] = buildID.String()
	return nil
}

func failStepResult(baseErr error, wrapped error) (appworkflow.StepResult, error) {
	err := baseErr
	if wrapped != nil {
		err = fmt.Errorf("%w: %v", baseErr, wrapped)
	}
	return appworkflow.StepResult{Status: domainworkflow.StepStatusFailed, Error: err.Error()}, err
}
