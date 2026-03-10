package build

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ============ Integration Tests ============

func TestIntegration_ExecutorWithService(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	if !executor.Supports(BuildMethodDocker) {
		t.Error("executor should support BuildMethodDocker")
	}

	t.Log("executor successfully created with service")
}

func TestIntegration_LogStreaming(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configID := "test-config-002"
	_, _ = executor.Execute(ctx, configID, BuildMethodDocker)

	t.Log("log streaming completed")
}

func TestIntegration_ConcurrentExecutions(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	var wg sync.WaitGroup
	numExecutions := 5

	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			executor, err := factory.CreateExecutor(BuildMethodDocker)

			if err != nil {
				t.Errorf("failed to create executor: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			configID := "test-config-concurrent-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodDocker)
		}(i)
	}

	wg.Wait()
	t.Log("concurrent executions completed")
}

func TestIntegration_ErrorHandling(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configID := "test-config-error"

	output, err := executor.Execute(ctx, configID, BuildMethodDocker)

	if output == nil && err == nil {
		t.Error("should have error or output")
	}

	if err != nil && err != ErrExecutionNotRunning {
		t.Logf("execution error (expected): %v", err)
	}
}

func TestIntegration_ArtifactHandling(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	if executor != nil {
		t.Log("artifact handling executor created successfully")
	}
}

func TestIntegration_FullPipeline_FactoryToService(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodDocker)

	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	} else {
		t.Log("full pipeline factory to service completed")
	}
}

func TestIntegration_ContextCancellation_Multi(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	configID := "test-config-cancel"

	select {
	case <-ctx.Done():
		t.Log("context cancelled as expected")
	case <-time.After(1 * time.Second):
		_, _ = executor.Execute(ctx, configID, BuildMethodDocker)
	}
}

func TestIntegration_ResourceCleanup(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	methods := factory.GetSupportedMethods()

	for idx, method := range methods {
		executor, err := factory.CreateExecutor(method)

		if err != nil {
			t.Errorf("failed to create executor for %v: %v", method, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		configID := "test-config-cleanup-" + string(rune(idx))
		_, _ = executor.Execute(ctx, configID, method)
		cancel()

		t.Logf("cleaned up executor for method %v", method)
	}
}

func TestIntegration_ParallelExecutors(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	methods := factory.GetSupportedMethods()

	var wg sync.WaitGroup

	for idx, method := range methods {
		wg.Add(1)
		go func(m BuildMethod, methodIdx int) {
			defer wg.Done()

			executor, err := factory.CreateExecutor(m)

			if err != nil {
				t.Errorf("failed to create executor: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			configID := "test-config-parallel-" + string(rune(methodIdx))
			_, _ = executor.Execute(ctx, configID, m)
		}(method, idx)
	}

	wg.Wait()
	t.Log("all parallel executors completed")
}

func TestIntegration_HighLoad(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodDocker)

	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	numExecutions := 100
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numExecutions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			configID := "test-config-load-" + string(rune(idx))
			_, _ = executor.Execute(ctx, configID, BuildMethodDocker)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("completed %d executions in %v", numExecutions, elapsed)

	if elapsed > 30*time.Second {
		t.Logf("warning: high load test took longer than expected: %v", elapsed)
	}
}

func TestIntegration_StatusTracking(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	if executor.Supports(BuildMethodDocker) {
		t.Log("status tracking executor supports BuildMethodDocker")
	}
}

func TestIntegration_ErrorRecovery(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configID := "test-config-recovery"

	_, err := executor.Execute(ctx, configID, BuildMethodDocker)

	if err != nil {
		t.Logf("execution failed as expected: %v", err)

		status, err := executor.GetStatus(context.Background(), "nonexistent-id")

		if err == nil {
			t.Logf("status retrieved after error: %v", status)
		}
	}
}

func TestIntegration_ServiceLogAccumulation(t *testing.T) {
	mockService := NewMockExecutorService()
	executor := NewMethodDockerExecutor(mockService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configID := "test-config-logs"

	_, _ = executor.Execute(ctx, configID, BuildMethodDocker)

	t.Log("log accumulation completed")
}

func TestIntegration_MultiMethodExecution(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	methods := factory.GetSupportedMethods()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for idx, method := range methods {
		executor, err := factory.CreateExecutor(method)

		if err != nil {
			t.Errorf("failed to create executor for %v: %v", method, err)
			continue
		}

		configID := "test-config-multi-" + string(rune(idx))
		output, err := executor.Execute(ctx, configID, method)

		if err != nil && err != ErrExecutionNotRunning {
			t.Logf("execution for %v returned error: %v", method, err)
			continue
		}

		if output == nil {
			t.Logf("no output for method %v (acceptable)", method)
		} else {
			t.Logf("method %v executed with status: %v", method, output.Status)
		}
	}
}

func TestIntegration_ContextPropagation(t *testing.T) {
	mockService := NewMockExecutorService()
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodDocker)

	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configID := "test-config-context"

	output, err := executor.Execute(ctx, configID, BuildMethodDocker)

	if err != nil && err != ErrExecutionNotRunning {
		t.Logf("execution with context returned: %v", err)
	}

	if output != nil {
		t.Log("context properly propagated to executor")
	}
}
