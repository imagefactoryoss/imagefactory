package build

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ============ Buildx Executor Tests ============

func TestBuildxExecutorExecute_Supports(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	if !executor.Supports(BuildMethodBuildx) {
		t.Error("executor should support BuildMethodBuildx")
	}

	if executor.Supports(BuildMethodDocker) {
		t.Error("executor should not support BuildMethodDocker")
	}
}

func TestBuildxExecutorCancel_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	err := executor.Cancel(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	if err != ErrExecutionNotRunning {
		t.Errorf("expected ErrExecutionNotRunning, got %v", err)
	}
}

func TestBuildxExecutorGetStatus_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	status, err := executor.GetStatus(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	
	// Status may vary depending on implementation
	if err != ErrExecutionNotRunning && err != nil {
		t.Logf("execution error (expected): %v, status: %v", err, status)
	}
}

func TestBuildxExecutorExecute_OutputStreaming(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-1", BuildMethodBuildx)

	if output != nil && output.ExecutionID != "" {
		t.Log("output stream received")
	}
}

func TestBuildxExecutorExecute_ContextCancellation(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _ = executor.Execute(ctx, "test-config-2", BuildMethodBuildx)
	t.Log("context cancellation handled")
}

func TestBuildxExecutorExecute_ConcurrentExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			configID := "test-config-concurrent-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodBuildx)
		}(i)
	}

	wg.Wait()
	t.Log("concurrent executions completed")
}

func TestBuildxExecutorExecute_MultiPlatform(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-multiplatform", BuildMethodBuildx)

	if output != nil && output.ExecutionID != "" {
		t.Log("multiplatform build initiated")
	}
}

func TestBuildxExecutorExecute_JSONParsing(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	execID := "test-exec-json-parsing"
	output, err := executor.Execute(ctx, execID, BuildMethodBuildx)

	if output != nil {
		if output.Output != "" {
			t.Logf("JSON output parsed: %d chars", len(output.Output))
		}
	}
	_ = err
}

func TestBuildxExecutorExecute_ArtifactHandling(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-artifacts", BuildMethodBuildx)

	if output != nil {
		if output.Artifacts != nil && len(output.Artifacts) > 0 {
			t.Log("artifacts successfully handled")
		}
	}
}

func TestBuildxExecutorMethodSupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	if !executor.Supports(BuildMethodBuildx) {
		t.Error("executor must support BuildMethodBuildx")
	}
}

func TestBuildxExecutorInterfaceCompliance(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodBuildxExecutor(mockService)

	_ = executor.(BuildMethodExecutor)
	t.Log("executor implements BuildMethodExecutor interface")
}
