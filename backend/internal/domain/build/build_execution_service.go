package build

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BuildExecutionService defines the interface for managing build executions
type BuildExecutionService interface {
	// Execution Management
	StartBuild(ctx context.Context, configID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error)
	CancelBuild(ctx context.Context, executionID uuid.UUID) error
	RetryBuild(ctx context.Context, executionID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error)

	// Status & History
	GetExecution(ctx context.Context, executionID uuid.UUID) (*BuildExecution, error)
	GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error)
	ListRunningExecutions(ctx context.Context) ([]BuildExecution, error)

	// Logs
	GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error)
	AddLog(ctx context.Context, executionID uuid.UUID, level LogLevel, message string, metadata []byte) error

	// Status Updates
	UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error
	UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error
	TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error)
	RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error)
	ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error
	CompleteExecution(ctx context.Context, executionID uuid.UUID, success bool, errorMsg string, artifacts []byte) error

	// Cleanup
	CleanupOldExecutions(ctx context.Context, olderThan time.Duration) error
}

// WebSocketEventBroadcaster defines interface for broadcasting WebSocket events
type WebSocketEventBroadcaster interface {
	BroadcastBuildEvent(tenantID uuid.UUID, eventType, buildID, buildNumber, projectID, status, message string, duration int, metadata map[string]interface{})
}

type BuildLogBroadcaster interface {
	BroadcastBuildLog(buildID uuid.UUID, timestamp time.Time, level, message string, metadata map[string]interface{})
}

// BuildStatusUpdateEmitter is an optional extension interface used by components that need
// deterministic build status.updated emissions outside execution state transitions.
type BuildStatusUpdateEmitter interface {
	EmitBuildStatusUpdate(ctx context.Context, buildID uuid.UUID, status, message string, metadata map[string]interface{})
}

// BuildExecutionServiceImpl implements BuildExecutionService
type BuildExecutionServiceImpl struct {
	repo          BuildExecutionRepository
	wsBroadcaster WebSocketEventBroadcaster
	buildIDCache  sync.Map // execution_id -> build_id
}

// NewBuildExecutionService creates a new BuildExecutionService
func NewBuildExecutionService(repo BuildExecutionRepository) BuildExecutionService {
	return &BuildExecutionServiceImpl{
		repo: repo,
	}
}

// NewBuildExecutionServiceWithWebSocket creates a new BuildExecutionService with WebSocket broadcasting
func NewBuildExecutionServiceWithWebSocket(repo BuildExecutionRepository, wsBroadcaster WebSocketEventBroadcaster) BuildExecutionService {
	return &BuildExecutionServiceImpl{
		repo:          repo,
		wsBroadcaster: wsBroadcaster,
	}
}

// StartBuild creates a new build execution
func (s *BuildExecutionServiceImpl) StartBuild(ctx context.Context, configID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error) {
	// Validate config exists and user has permission
	// (This will be expanded with actual validation in later implementation)

	// Check if build is already executing
	running, err := s.repo.GetRunningExecutionForConfig(ctx, configID)
	if err != nil && err != ErrExecutionNotFound {
		return nil, err
	}
	if running != nil {
		return nil, ErrBuildAlreadyExecuting
	}

	// Create execution record
	execution := &BuildExecution{
		ID:        uuid.New(),
		ConfigID:  configID,
		Status:    ExecutionPending,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Get build ID from config
	buildID, err := s.repo.GetBuildIDFromConfig(ctx, configID)
	if err != nil {
		return nil, err
	}
	execution.BuildID = buildID

	// Save execution
	if err := s.repo.SaveExecution(ctx, execution); err != nil {
		return nil, err
	}

	// Add initial log
	_ = s.AddLog(ctx, execution.ID, LogInfo, "Build execution started", nil)

	return execution, nil
}

// CancelBuild cancels a running build execution
func (s *BuildExecutionServiceImpl) CancelBuild(ctx context.Context, executionID uuid.UUID) error {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	if !execution.Status.CanTransitionTo(ExecutionCancelled) {
		return ErrExecutionNotCancellable
	}

	return s.UpdateExecutionStatus(ctx, executionID, ExecutionCancelled)
}

// RetryBuild retries a failed build execution
func (s *BuildExecutionServiceImpl) RetryBuild(ctx context.Context, executionID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	if execution.Status != ExecutionFailed {
		return nil, ErrExecutionNotRetryable
	}

	// Create new execution with same config
	return s.StartBuild(ctx, execution.ConfigID, createdBy)
}

// GetExecution retrieves a specific execution
func (s *BuildExecutionServiceImpl) GetExecution(ctx context.Context, executionID uuid.UUID) (*BuildExecution, error) {
	return s.repo.GetExecution(ctx, executionID)
}

// GetBuildExecutions retrieves all executions for a build with pagination
func (s *BuildExecutionServiceImpl) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error) {
	return s.repo.GetBuildExecutions(ctx, buildID, limit, offset)
}

// ListRunningExecutions retrieves all running executions
func (s *BuildExecutionServiceImpl) ListRunningExecutions(ctx context.Context) ([]BuildExecution, error) {
	return s.repo.ListRunningExecutions(ctx)
}

// GetLogs retrieves logs for an execution with pagination
func (s *BuildExecutionServiceImpl) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error) {
	return s.repo.GetLogs(ctx, executionID, limit, offset)
}

// AddLog adds a log entry for an execution
func (s *BuildExecutionServiceImpl) AddLog(ctx context.Context, executionID uuid.UUID, level LogLevel, message string, metadata []byte) error {
	log := &ExecutionLog{
		ID:          uuid.New(),
		ExecutionID: executionID,
		Timestamp:   time.Now(),
		Level:       level,
		Message:     message,
		Metadata:    metadata,
	}
	if err := s.repo.AddLog(ctx, log); err != nil {
		return err
	}

	logBroadcaster, ok := s.wsBroadcaster.(BuildLogBroadcaster)
	if !ok || logBroadcaster == nil {
		return nil
	}

	buildID, found := s.buildIDForExecution(ctx, executionID)
	if !found {
		return nil
	}

	var metadataMap map[string]interface{}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &metadataMap)
	}
	if metadataMap == nil {
		metadataMap = map[string]interface{}{}
	}
	if _, exists := metadataMap["execution_id"]; !exists {
		metadataMap["execution_id"] = executionID.String()
	}

	logBroadcaster.BroadcastBuildLog(buildID, log.Timestamp, strings.ToUpper(string(level)), message, metadataMap)
	return nil
}

func (s *BuildExecutionServiceImpl) buildIDForExecution(ctx context.Context, executionID uuid.UUID) (uuid.UUID, bool) {
	if cached, ok := s.buildIDCache.Load(executionID.String()); ok {
		if id, castOK := cached.(uuid.UUID); castOK && id != uuid.Nil {
			return id, true
		}
	}

	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil || execution == nil || execution.BuildID == uuid.Nil {
		return uuid.Nil, false
	}
	s.buildIDCache.Store(executionID.String(), execution.BuildID)
	return execution.BuildID, true
}

// UpdateExecutionStatus updates the execution status
func (s *BuildExecutionServiceImpl) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	if !execution.Status.CanTransitionTo(status) {
		return ErrInvalidExecutionStatus
	}

	execution.Status = status
	execution.UpdatedAt = time.Now()

	if status == ExecutionRunning && execution.StartedAt == nil {
		execution.StartedAt = &execution.UpdatedAt
	}

	err = s.repo.UpdateExecution(ctx, execution)
	if err != nil {
		return err
	}

	// Broadcast status update via WebSocket
	if s.wsBroadcaster != nil {
		eventType := "build.status.updated"
		message := fmt.Sprintf("Build status changed to %s", string(status))
		s.wsBroadcaster.BroadcastBuildEvent(
			uuid.Nil,
			eventType,
			execution.BuildID.String(),
			"",
			"",
			string(status),
			message,
			0,
			map[string]interface{}{
				"execution_id": executionID.String(),
			},
		)
	}

	return nil
}

// EmitBuildStatusUpdate emits a build.status.updated event via the configured broadcaster.
// This is intentionally decoupled from execution status transitions for reconciliation paths.
func (s *BuildExecutionServiceImpl) EmitBuildStatusUpdate(ctx context.Context, buildID uuid.UUID, status, message string, metadata map[string]interface{}) {
	if s.wsBroadcaster == nil || buildID == uuid.Nil {
		return
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	s.wsBroadcaster.BroadcastBuildEvent(
		uuid.Nil,
		"build.status.updated",
		buildID.String(),
		"",
		"",
		status,
		message,
		0,
		metadata,
	)
}

// UpdateExecutionMetadata updates execution metadata payload.
func (s *BuildExecutionServiceImpl) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	return s.repo.UpdateExecutionMetadata(ctx, executionID, metadata)
}

func (s *BuildExecutionServiceImpl) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return s.repo.TryAcquireMonitoringLease(ctx, executionID, owner, ttl)
}

func (s *BuildExecutionServiceImpl) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return s.repo.RenewMonitoringLease(ctx, executionID, owner, ttl)
}

func (s *BuildExecutionServiceImpl) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	return s.repo.ReleaseMonitoringLease(ctx, executionID, owner)
}

// CompleteExecution marks an execution as complete
func (s *BuildExecutionServiceImpl) CompleteExecution(ctx context.Context, executionID uuid.UUID, success bool, errorMsg string, artifacts []byte) error {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	now := time.Now()
	execution.CompletedAt = &now
	execution.ErrorMessage = errorMsg
	execution.Artifacts = artifacts

	if execution.StartedAt != nil {
		duration := int(now.Sub(*execution.StartedAt).Seconds())
		execution.DurationSeconds = &duration
	}

	if success {
		execution.Status = ExecutionSuccess
	} else {
		execution.Status = ExecutionFailed
	}

	execution.UpdatedAt = now

	err = s.repo.UpdateExecution(ctx, execution)
	if err != nil {
		return err
	}

	// Broadcast completion via WebSocket
	if s.wsBroadcaster != nil {
		var eventType, message string
		if success {
			eventType = "build.completed"
			message = "Build completed successfully"
		} else {
			eventType = "build.failed"
			message = fmt.Sprintf("Build failed: %s", errorMsg)
		}
		duration := 0
		if execution.DurationSeconds != nil {
			duration = *execution.DurationSeconds
		}
		s.wsBroadcaster.BroadcastBuildEvent(
			uuid.Nil,
			eventType,
			execution.BuildID.String(),
			"",
			"",
			string(execution.Status),
			message,
			duration,
			map[string]interface{}{
				"execution_id": executionID.String(),
			},
		)
	}

	return nil
}

// CleanupOldExecutions deletes executions older than the specified duration
func (s *BuildExecutionServiceImpl) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) error {
	return s.repo.DeleteOldExecutions(ctx, olderThan)
}
