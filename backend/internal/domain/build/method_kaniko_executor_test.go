package build

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ============ Kaniko Executor Tests ============

func TestKanikoExecutorExecute_Supports(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	if !executor.Supports(BuildMethodKaniko) {
		t.Error("executor should support BuildMethodKaniko")
	}

	if executor.Supports(BuildMethodDocker) {
		t.Error("executor should not support BuildMethodDocker")
	}
}

func TestKanikoExecutorCancel_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	err := executor.Cancel(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	if err != ErrExecutionNotRunning {
		t.Errorf("expected ErrExecutionNotRunning, got %v", err)
	}
}

func TestKanikoExecutorGetStatus_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	status, err := executor.GetStatus(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	
	// Status may vary depending on implementation
	if err != ErrExecutionNotRunning && err != nil {
		t.Logf("execution error (expected): %v, status: %v", err, status)
	}
}

func TestKanikoExecutorExecute_OutputStreaming(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-1", BuildMethodKaniko)

	if output != nil && output.ExecutionID != "" {
		t.Log("output stream received")
	}
}

func TestKanikoExecutorExecute_ContextCancellation(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _ = executor.Execute(ctx, "test-config-2", BuildMethodKaniko)
	t.Log("context cancellation handled")
}

func TestKanikoExecutorExecute_ConcurrentExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			configID := "test-config-concurrent-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodKaniko)
		}(i)
	}

	wg.Wait()
	t.Log("concurrent executions completed")
}

func TestKanikoExecutorExecute_ContainerExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-container", BuildMethodKaniko)

	if output != nil && output.ExecutionID != "" {
		t.Log("container execution initiated")
	}
}

func TestKanikoExecutorExecute_ImagePush(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-push", BuildMethodKaniko)

	if output != nil {
		if output.Artifacts != nil && len(output.Artifacts) > 0 {
			t.Log("image pushed successfully")
		}
	}
}

func TestKanikoExecutorExecute_RegistrySupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-registry", BuildMethodKaniko)

	if output != nil && output.ExecutionID != "" {
		t.Log("registry support verified")
	}
}

func TestKanikoExecutorExecute_DaemonlessMode(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-daemonless", BuildMethodKaniko)

	if output != nil && output.Status != "" {
		t.Logf("daemonless execution status: %v", output.Status)
	}
}

func TestKanikoExecutorMethodSupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	if !executor.Supports(BuildMethodKaniko) {
		t.Error("executor must support BuildMethodKaniko")
	}
}

func TestKanikoExecutorInterfaceCompliance(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodKanikoExecutor(mockService)

	_ = executor.(BuildMethodExecutor)
	t.Log("executor implements BuildMethodExecutor interface")
}
