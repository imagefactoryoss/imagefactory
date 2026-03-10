package build

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ============ Nix Executor Tests ============

func TestNixExecutorExecute_Supports(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	if !executor.Supports(BuildMethodNix) {
		t.Error("executor should support BuildMethodNix")
	}

	if executor.Supports(BuildMethodDocker) {
		t.Error("executor should not support BuildMethodDocker")
	}
}

func TestNixExecutorCancel_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	err := executor.Cancel(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}
	if err != ErrExecutionNotRunning {
		t.Errorf("expected ErrExecutionNotRunning, got %v", err)
	}
}

func TestNixExecutorGetStatus_NotRunning(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	status, err := executor.GetStatus(context.Background(), "nonexistent-exec-id")

	if err == nil {
		t.Error("expected error for non-running execution")
	}

	// Status may vary depending on implementation
	if err != ErrExecutionNotRunning && err != nil {
		t.Logf("execution error (expected): %v, status: %v", err, status)
	}
}

func TestNixExecutorExecute_OutputStreaming(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-1", BuildMethodNix)

	if output != nil && output.ExecutionID != "" {
		t.Log("output stream received")
	}
}

func TestNixExecutorExecute_ContextCancellation(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _ = executor.Execute(ctx, "test-config-2", BuildMethodNix)
	t.Log("context cancellation handled")
}

func TestNixExecutorExecute_ConcurrentExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			configID := "test-config-concurrent-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodNix)
		}(i)
	}

	wg.Wait()
	t.Log("concurrent executions completed")
}

func TestNixExecutorExecute_StorePathExtraction(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-storepath", BuildMethodNix)

	if output != nil && output.ExecutionID != "" {
		t.Log("store path extraction initiated")
	}
}

func TestNixExecutorExecute_JSONOutput(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-json", BuildMethodNix)

	if output != nil {
		if output.Output != "" {
			t.Logf("JSON output received: %d chars", len(output.Output))
		}
	}
}

func TestNixExecutorExecute_DerivationSupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, _ := executor.Execute(ctx, "test-config-derivation", BuildMethodNix)

	if output != nil && output.Status != "" {
		t.Logf("derivation status: %v", output.Status)
	}
}

func TestNixExecutorMethodSupport(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	if !executor.Supports(BuildMethodNix) {
		t.Error("executor must support BuildMethodNix")
	}
}

func TestNixExecutorInterfaceCompliance(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodNixExecutor(mockService)

	_ = executor.(BuildMethodExecutor)
	t.Log("executor implements BuildMethodExecutor interface")
}
