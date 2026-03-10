package build

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBuildExecutionRepository is a mock implementation for testing
type MockBuildExecutionRepository struct {
	Executions         map[uuid.UUID]*BuildExecution
	Logs               map[uuid.UUID][]ExecutionLog
	LeaseOwners        map[uuid.UUID]string
	LeaseExpiries      map[uuid.UUID]time.Time
	NowFunc            func() time.Time
	GetExecutionErr    error
	SaveExecutionErr   error
	UpdateExecutionErr error
	GetRunningErr      error
	GetBuildIDErr      error
}

func NewMockBuildExecutionRepository() *MockBuildExecutionRepository {
	return &MockBuildExecutionRepository{
		Executions:    make(map[uuid.UUID]*BuildExecution),
		Logs:          make(map[uuid.UUID][]ExecutionLog),
		LeaseOwners:   make(map[uuid.UUID]string),
		LeaseExpiries: make(map[uuid.UUID]time.Time),
		NowFunc:       time.Now,
	}
}

func (m *MockBuildExecutionRepository) SaveExecution(ctx context.Context, execution *BuildExecution) error {
	if m.SaveExecutionErr != nil {
		return m.SaveExecutionErr
	}
	m.Executions[execution.ID] = execution
	m.Logs[execution.ID] = []ExecutionLog{}
	return nil
}

func (m *MockBuildExecutionRepository) UpdateExecution(ctx context.Context, execution *BuildExecution) error {
	if m.UpdateExecutionErr != nil {
		return m.UpdateExecutionErr
	}
	m.Executions[execution.ID] = execution
	return nil
}

func (m *MockBuildExecutionRepository) GetExecution(ctx context.Context, id uuid.UUID) (*BuildExecution, error) {
	if m.GetExecutionErr != nil {
		return nil, m.GetExecutionErr
	}
	if exec, ok := m.Executions[id]; ok {
		return exec, nil
	}
	return nil, ErrExecutionNotFound
}

func (m *MockBuildExecutionRepository) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error) {
	var executions []BuildExecution
	for _, exec := range m.Executions {
		if exec.BuildID == buildID {
			executions = append(executions, *exec)
		}
	}

	total := int64(len(executions))

	// Apply pagination
	if offset >= len(executions) {
		return []BuildExecution{}, total, nil
	}

	end := offset + limit
	if end > len(executions) {
		end = len(executions)
	}

	return executions[offset:end], total, nil
}

func (m *MockBuildExecutionRepository) ListRunningExecutions(ctx context.Context) ([]BuildExecution, error) {
	var executions []BuildExecution
	for _, exec := range m.Executions {
		if exec.Status == ExecutionRunning {
			executions = append(executions, *exec)
		}
	}
	return executions, nil
}

func (m *MockBuildExecutionRepository) GetRunningExecutionForConfig(ctx context.Context, configID uuid.UUID) (*BuildExecution, error) {
	if m.GetRunningErr != nil {
		return nil, m.GetRunningErr
	}
	for _, exec := range m.Executions {
		if exec.ConfigID == configID && exec.Status == ExecutionRunning {
			return exec, nil
		}
	}
	return nil, ErrExecutionNotFound
}

func (m *MockBuildExecutionRepository) AddLog(ctx context.Context, log *ExecutionLog) error {
	m.Logs[log.ExecutionID] = append(m.Logs[log.ExecutionID], *log)
	return nil
}

func (m *MockBuildExecutionRepository) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error) {
	logs := m.Logs[executionID]
	return logs, int64(len(logs)), nil
}

func (m *MockBuildExecutionRepository) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error {
	if exec, ok := m.Executions[executionID]; ok {
		exec.Status = status
		exec.UpdatedAt = time.Now()
		return nil
	}
	return ErrExecutionNotFound
}

func (m *MockBuildExecutionRepository) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	if exec, ok := m.Executions[executionID]; ok {
		exec.Metadata = metadata
		exec.UpdatedAt = time.Now()
		return nil
	}
	return ErrExecutionNotFound
}

func (m *MockBuildExecutionRepository) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	exec, ok := m.Executions[executionID]
	if !ok {
		return false, ErrExecutionNotFound
	}
	if exec.Status != ExecutionPending && exec.Status != ExecutionRunning {
		return false, nil
	}

	now := m.NowFunc()
	currentOwner := m.LeaseOwners[executionID]
	expiresAt := m.LeaseExpiries[executionID]
	canAcquire := currentOwner == "" || currentOwner == owner || expiresAt.IsZero() || expiresAt.Before(now)
	if !canAcquire {
		return false, nil
	}

	m.LeaseOwners[executionID] = owner
	m.LeaseExpiries[executionID] = now.Add(ttl)
	return true, nil
}

func (m *MockBuildExecutionRepository) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	exec, ok := m.Executions[executionID]
	if !ok {
		return false, ErrExecutionNotFound
	}
	if exec.Status != ExecutionPending && exec.Status != ExecutionRunning {
		return false, nil
	}
	if m.LeaseOwners[executionID] != owner {
		return false, nil
	}
	m.LeaseExpiries[executionID] = m.NowFunc().Add(ttl)
	return true, nil
}

func (m *MockBuildExecutionRepository) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	if _, ok := m.Executions[executionID]; !ok {
		return ErrExecutionNotFound
	}
	if m.LeaseOwners[executionID] == owner {
		delete(m.LeaseOwners, executionID)
		delete(m.LeaseExpiries, executionID)
	}
	return nil
}

func (m *MockBuildExecutionRepository) DeleteOldExecutions(ctx context.Context, olderThan time.Duration) error {
	return nil
}

func (m *MockBuildExecutionRepository) GetBuildIDFromConfig(ctx context.Context, configID uuid.UUID) (uuid.UUID, error) {
	if m.GetBuildIDErr != nil {
		return uuid.Nil, m.GetBuildIDErr
	}
	return uuid.New(), nil
}

// Tests

func TestStartBuild_Success(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()
	exec, err := service.StartBuild(ctx, configID, createdBy)

	assert.NoError(t, err)
	require.NotNil(t, exec)
	assert.Equal(t, ExecutionPending, exec.Status)
	assert.Equal(t, configID, exec.ConfigID)
	assert.Equal(t, createdBy, exec.CreatedBy)
	assert.NotEqual(t, uuid.Nil, exec.ID)

	// Verify execution was saved
	saved, err := mock.GetExecution(ctx, exec.ID)
	assert.NoError(t, err)
	assert.Equal(t, ExecutionPending, saved.Status)
}

func TestStartBuild_AlreadyRunning(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start first execution
	exec1, err := service.StartBuild(ctx, configID, createdBy)
	assert.NoError(t, err)

	// Transition to running
	err = service.UpdateExecutionStatus(ctx, exec1.ID, ExecutionRunning)
	assert.NoError(t, err)

	// Try to start second execution
	exec2, err := service.StartBuild(ctx, configID, createdBy)

	assert.Error(t, err)
	assert.Equal(t, ErrBuildAlreadyExecuting, err)
	assert.Nil(t, exec2)
}

func TestCancelBuild_Success(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start execution
	exec, err := service.StartBuild(ctx, configID, createdBy)
	assert.NoError(t, err)

	// Cancel it
	err = service.CancelBuild(ctx, exec.ID)
	assert.NoError(t, err)

	// Verify status
	updated, err := mock.GetExecution(ctx, exec.ID)
	assert.NoError(t, err)
	assert.Equal(t, ExecutionCancelled, updated.Status)
}

func TestCancelBuild_AlreadyCompleted(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start and complete execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)
	service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)
	service.CompleteExecution(ctx, exec.ID, true, "", json.RawMessage("[]"))

	// Try to cancel
	err := service.CancelBuild(ctx, exec.ID)

	assert.Error(t, err)
	assert.Equal(t, ErrExecutionNotCancellable, err)
}

func TestRetryBuild_Success(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start, run, and fail execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)
	service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)
	service.CompleteExecution(ctx, exec.ID, false, "Build failed", json.RawMessage("[]"))

	// Retry
	retried, err := service.RetryBuild(ctx, exec.ID, createdBy)

	assert.NoError(t, err)
	require.NotNil(t, retried)
	assert.Equal(t, ExecutionPending, retried.Status)
	assert.Equal(t, configID, retried.ConfigID)
	assert.NotEqual(t, exec.ID, retried.ID)
}

func TestMonitoringLease_StaleLeaseCanBeReclaimed(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()
	ctx := context.Background()

	exec, err := service.StartBuild(ctx, configID, createdBy)
	require.NoError(t, err)
	require.NotNil(t, exec)

	acquiredByOwner1, err := service.TryAcquireMonitoringLease(ctx, exec.ID, "owner-1", 20*time.Millisecond)
	require.NoError(t, err)
	require.True(t, acquiredByOwner1)

	acquiredByOwner2, err := service.TryAcquireMonitoringLease(ctx, exec.ID, "owner-2", 20*time.Millisecond)
	require.NoError(t, err)
	require.False(t, acquiredByOwner2)

	time.Sleep(25 * time.Millisecond)

	acquiredByOwner2, err = service.TryAcquireMonitoringLease(ctx, exec.ID, "owner-2", 20*time.Millisecond)
	require.NoError(t, err)
	require.True(t, acquiredByOwner2)

	renewedByOwner1, err := service.RenewMonitoringLease(ctx, exec.ID, "owner-1", time.Second)
	require.NoError(t, err)
	require.False(t, renewedByOwner1)

	renewedByOwner2, err := service.RenewMonitoringLease(ctx, exec.ID, "owner-2", time.Second)
	require.NoError(t, err)
	require.True(t, renewedByOwner2)
}

func TestRetryBuild_SuccessfulExecution(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start, run, and succeed execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)
	service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)
	service.CompleteExecution(ctx, exec.ID, true, "", json.RawMessage("[]"))

	// Try to retry
	retried, err := service.RetryBuild(ctx, exec.ID, createdBy)

	assert.Error(t, err)
	assert.Equal(t, ErrExecutionNotRetryable, err)
	assert.Nil(t, retried)
}

func TestGetExecution_NotFound(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	ctx := context.Background()
	exec, err := service.GetExecution(ctx, uuid.New())

	assert.Error(t, err)
	assert.Equal(t, ErrExecutionNotFound, err)
	assert.Nil(t, exec)
}

func TestGetBuildExecutions_Pagination(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	buildID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Create 3 executions with different configs
	for i := 0; i < 3; i++ {
		configID := uuid.New()
		exec, _ := service.StartBuild(ctx, configID, createdBy)
		mock.Executions[exec.ID].BuildID = buildID
	}

	// Get with pagination
	executions, total, err := service.GetBuildExecutions(ctx, buildID, 2, 0)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 2, len(executions))
}

func TestListRunningExecutions(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	ctx := context.Background()

	// Create 2 running and 1 pending
	config1, config2, config3 := uuid.New(), uuid.New(), uuid.New()
	createdBy := uuid.New()

	exec1, _ := service.StartBuild(ctx, config1, createdBy)
	exec2, _ := service.StartBuild(ctx, config2, createdBy)
	_, _ = service.StartBuild(ctx, config3, createdBy)

	service.UpdateExecutionStatus(ctx, exec1.ID, ExecutionRunning)
	service.UpdateExecutionStatus(ctx, exec2.ID, ExecutionRunning)
	// third execution stays pending

	running, err := service.ListRunningExecutions(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(running))
	for _, r := range running {
		assert.Equal(t, ExecutionRunning, r.Status)
	}
}

func TestAddLog(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)

	// Add log
	err := service.AddLog(ctx, exec.ID, LogInfo, "Test message", json.RawMessage("{}"))

	assert.NoError(t, err)

	// Verify log
	logs, _, err := service.GetLogs(ctx, exec.ID, 10, 0)
	assert.NoError(t, err)
	assert.True(t, len(logs) > 0)

	// Verify last log is our message
	lastLog := logs[len(logs)-1]
	assert.Equal(t, LogInfo, lastLog.Level)
	assert.Equal(t, "Test message", lastLog.Message)
}

func TestCompleteExecution_Success(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)

	// Move to running
	service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)

	// Complete it
	artifacts := json.RawMessage(`[{"name": "image", "size": 1024}]`)
	err := service.CompleteExecution(ctx, exec.ID, true, "", artifacts)

	assert.NoError(t, err)

	// Verify
	completed, _ := service.GetExecution(ctx, exec.ID)
	assert.Equal(t, ExecutionSuccess, completed.Status)
	assert.NotNil(t, completed.CompletedAt)
	assert.NotNil(t, completed.DurationSeconds)
	assert.Equal(t, artifacts, completed.Artifacts)
}

func TestCompleteExecution_Failure(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)

	// Move to running
	service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)

	// Complete with failure
	err := service.CompleteExecution(ctx, exec.ID, false, "Build timed out", json.RawMessage("[]"))

	assert.NoError(t, err)

	// Verify
	completed, _ := service.GetExecution(ctx, exec.ID)
	assert.Equal(t, ExecutionFailed, completed.Status)
	assert.Equal(t, "Build timed out", completed.ErrorMessage)
}

func TestExecutionStatusTransitions(t *testing.T) {
	tests := []struct {
		from  ExecutionStatus
		to    ExecutionStatus
		valid bool
		name  string
	}{
		{ExecutionPending, ExecutionRunning, true, "pending to running"},
		{ExecutionPending, ExecutionCancelled, true, "pending to cancelled"},
		{ExecutionPending, ExecutionSuccess, false, "pending to success"},
		{ExecutionRunning, ExecutionSuccess, true, "running to success"},
		{ExecutionRunning, ExecutionFailed, true, "running to failed"},
		{ExecutionRunning, ExecutionCancelled, true, "running to cancelled"},
		{ExecutionSuccess, ExecutionRunning, false, "success to running"},
		{ExecutionFailed, ExecutionRunning, false, "failed to running"},
		{ExecutionCancelled, ExecutionRunning, false, "cancelled to running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   ExecutionStatus
		terminal bool
		name     string
	}{
		{ExecutionPending, false, "pending"},
		{ExecutionRunning, false, "running"},
		{ExecutionSuccess, true, "success"},
		{ExecutionFailed, true, "failed"},
		{ExecutionCancelled, true, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsTerminalStatus()
			assert.Equal(t, tt.terminal, result)
		})
	}
}

func TestCleanupOldExecutions(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	ctx := context.Background()

	// Cleanup should work without error
	err := service.CleanupOldExecutions(ctx, 24*time.Hour)
	assert.NoError(t, err)
}

func TestUpdateExecutionStatus_InvalidTransition(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()

	// Start execution
	exec, _ := service.StartBuild(ctx, configID, createdBy)

	// Complete it
	service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)
	service.CompleteExecution(ctx, exec.ID, true, "", json.RawMessage("[]"))

	// Try invalid transition
	err := service.UpdateExecutionStatus(ctx, exec.ID, ExecutionRunning)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidExecutionStatus, err)
}

func TestBuildExecutionToResponse(t *testing.T) {
	now := time.Now()
	duration := 42

	exec := &BuildExecution{
		ID:              uuid.New(),
		BuildID:         uuid.New(),
		ConfigID:        uuid.New(),
		Status:          ExecutionSuccess,
		StartedAt:       &now,
		CompletedAt:     &now,
		DurationSeconds: &duration,
		ErrorMessage:    "",
		CreatedBy:       uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	resp := exec.ToResponse()

	assert.Equal(t, exec.ID, resp.ID)
	assert.Equal(t, exec.BuildID, resp.BuildID)
	assert.Equal(t, exec.ConfigID, resp.ConfigID)
	assert.Equal(t, ExecutionSuccess, resp.Status)
	assert.Equal(t, &duration, resp.DurationSeconds)
	assert.Equal(t, "", resp.ErrorMessage)
}

func TestExecutionLogToResponse(t *testing.T) {
	now := time.Now()

	log := &ExecutionLog{
		ID:          uuid.New(),
		ExecutionID: uuid.New(),
		Timestamp:   now,
		Level:       LogInfo,
		Message:     "Test log",
		Metadata:    json.RawMessage("{}"),
	}

	resp := log.ToResponse()

	assert.Equal(t, log.ID, resp.ID)
	assert.Equal(t, log.ExecutionID, resp.ExecutionID)
	assert.Equal(t, log.Timestamp, resp.Timestamp)
	assert.Equal(t, LogInfo, resp.Level)
	assert.Equal(t, "Test log", resp.Message)
}

func TestStartBuild_RepositoryError(t *testing.T) {
	mock := NewMockBuildExecutionRepository()
	mock.SaveExecutionErr = errors.New("database error")

	service := NewBuildExecutionService(mock)

	configID := uuid.New()
	createdBy := uuid.New()

	ctx := context.Background()
	exec, err := service.StartBuild(ctx, configID, createdBy)

	assert.Error(t, err)
	assert.Nil(t, exec)
	assert.Equal(t, "database error", err.Error())
}
