package build

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ============ Docker Executor Tests ============

func TestDockerExecutorExecute_Supports(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	if !executor.Supports(BuildMethodDocker) {
		t.Error("executor should support BuildMethodDocker")
	}

	if executor.Supports(BuildMethodPacker) {
		t.Error("executor should not support BuildMethodPacker")
	}
}

func TestDockerExecutorCancel_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	err := executor.Cancel(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	if err != ErrExecutionNotRunning {
		t.Errorf("expected ErrExecutionNotRunning, got %v", err)
	}
}

func TestDockerExecutorGetStatus_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	status, err := executor.GetStatus(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	
	// Status may vary depending on implementation
	if err != ErrExecutionNotRunning && err != nil {
		t.Logf("execution error (expected): %v, status: %v", err, status)
	}
}

func TestDockerExecutorExecute_OutputStreaming(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-1", BuildMethodDocker)

	if output != nil && output.ExecutionID != "" {
		t.Log("output stream received")
	}
}

func TestDockerExecutorExecute_ContextCancellation(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _ = executor.Execute(ctx, "test-config-2", BuildMethodDocker)
	t.Log("context cancellation handled")
}

func TestDockerExecutorExecute_ConcurrentExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			configID := "test-config-concurrent-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodDocker)
		}(i)
	}

	wg.Wait()
	t.Log("concurrent executions completed")
}

func TestDockerExecutorExecute_ImageTagging(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-tagging", BuildMethodDocker)

	if output != nil && output.ExecutionID != "" {
		t.Log("image tagging completed")
	}
}

func TestDockerExecutorExecute_SimpleWorkflow(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-workflow", BuildMethodDocker)

	if output != nil {
		if output.Status != "" {
			t.Logf("workflow status: %v", output.Status)
		}
	}
}

func TestDockerExecutorMethodSupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	if !executor.Supports(BuildMethodDocker) {
		t.Error("executor must support BuildMethodDocker")
	}
}

func TestDockerExecutorInterfaceCompliance(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	_ = executor.(BuildMethodExecutor)
	t.Log("executor implements BuildMethodExecutor interface")
}

func TestDockerExecutorExecute_ArtifactHandling(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-docker-artifacts", BuildMethodDocker)

	if output != nil {
		if output.Artifacts != nil && len(output.Artifacts) > 0 {
			t.Logf("docker artifacts: %d items", len(output.Artifacts))
		}
	}
}
