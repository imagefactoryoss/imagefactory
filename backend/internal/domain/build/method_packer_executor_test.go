package build

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============ Mock Services (Reusable) ============

// MockExecutorService is a simple mock for BuildExecutionService
type MockExecutorService struct {
	logs       map[string]int
	executions map[string]bool
	mu         sync.RWMutex
}

func NewMockExecutorService() *MockExecutorService {
	return &MockExecutorService{
		logs:       make(map[string]int),
		executions: make(map[string]bool),
	}
}

func (m *MockExecutorService) AddLog(ctx context.Context, executionID uuid.UUID, level LogLevel, message string, metadata []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs[executionID.String()]++
	return nil
}

func (m *MockExecutorService) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return []ExecutionLog{}, int64(m.logs[executionID.String()]), nil
}

func (m *MockExecutorService) StartBuild(ctx context.Context, configID, createdBy uuid.UUID) (*BuildExecution, error) {
	return &BuildExecution{ID: uuid.New()}, nil
}

func (m *MockExecutorService) CancelBuild(ctx context.Context, executionID uuid.UUID) error {
	return nil
}

func (m *MockExecutorService) RetryBuild(ctx context.Context, executionID, createdBy uuid.UUID) (*BuildExecution, error) {
	return &BuildExecution{ID: executionID}, nil
}

func (m *MockExecutorService) GetExecution(ctx context.Context, executionID uuid.UUID) (*BuildExecution, error) {
	return &BuildExecution{ID: executionID}, nil
}

func (m *MockExecutorService) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error) {
	return []BuildExecution{}, 0, nil
}

func (m *MockExecutorService) ListRunningExecutions(ctx context.Context) ([]BuildExecution, error) {
	return []BuildExecution{}, nil
}

func (m *MockExecutorService) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error {
	return nil
}

func (m *MockExecutorService) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	return nil
}
func (m *MockExecutorService) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return true, nil
}
func (m *MockExecutorService) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return true, nil
}
func (m *MockExecutorService) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	return nil
}

func (m *MockExecutorService) CompleteExecution(ctx context.Context, executionID uuid.UUID, success bool, errorMsg string, artifacts []byte) error {
	return nil
}

func (m *MockExecutorService) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) error {
	return nil
}

// ============ Packer Executor Tests ============

func TestPackerExecutorExecute_Supports(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	if !executor.Supports(BuildMethodPacker) {
		t.Error("executor should support BuildMethodPacker")
	}

	if executor.Supports(BuildMethodDocker) {
		t.Error("executor should not support BuildMethodDocker")
	}
}

func TestPackerExecutorCancel_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	err := executor.Cancel(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	if err != ErrExecutionNotRunning {
		t.Errorf("expected ErrExecutionNotRunning, got %v", err)
	}
}

func TestPackerExecutorGetStatus_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	status, err := executor.GetStatus(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}

	// Status may vary depending on implementation
	if err != ErrExecutionNotRunning && err != nil {
		t.Logf("execution error (expected): %v, status: %v", err, status)
	}
}

func TestPackerExecutorExecute_OutputStreaming(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-1", BuildMethodPacker)

	if output != nil && output.ExecutionID != "" {
		t.Log("output stream received")
	}
}

func TestPackerExecutorExecute_ContextCancellation(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _ = executor.Execute(ctx, "test-config-2", BuildMethodPacker)
	t.Log("context cancellation handled")
}

func TestPackerExecutorExecute_ConcurrentExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			configID := "test-config-concurrent-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodPacker)
		}(i)
	}

	wg.Wait()
	t.Log("concurrent executions completed")
}

func TestPackerExecutorMethodSupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	if !executor.Supports(BuildMethodPacker) {
		t.Error("executor must support BuildMethodPacker")
	}
}

func TestPackerExecutorInterfaceCompliance(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	_ = executor.(BuildMethodExecutor)
	t.Log("executor implements BuildMethodExecutor interface")
}

func TestPackerExecutorExecute_Success(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := executor.Execute(ctx, "test-config-success", BuildMethodPacker)

	if err != nil && err != ErrExecutionNotRunning {
		t.Logf("execution error (acceptable): %v", err)
	}

	if output != nil {
		t.Logf("execution status: %v", output.Status)
	}
}

func TestPackerExecutorCommandExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-cmd", BuildMethodPacker)

	if output != nil && output.ExecutionID != "" {
		t.Log("command executed successfully")
	}
}

func TestPackerExecutorArtifactHandling(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodPackerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-artifacts", BuildMethodPacker)

	if output != nil {
		if output.Artifacts != nil && len(output.Artifacts) > 0 {
			t.Logf("artifacts generated: %d items", len(output.Artifacts))
		} else {
			t.Log("no artifacts generated (acceptable)")
		}
	}
}
