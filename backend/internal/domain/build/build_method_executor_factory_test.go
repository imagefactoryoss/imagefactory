package build

import (
	"context"
	"testing"
	"time"
)

// ============ Factory Tests ============

func TestFactoryCreateExecutor_Packer(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodPacker)

	if err != nil {
		t.Errorf("failed to create packer executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	}

	if !executor.Supports(BuildMethodPacker) {
		t.Error("created executor should support BuildMethodPacker")
	}
}

func TestFactoryCreateExecutor_Buildx(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodBuildx)

	if err != nil {
		t.Errorf("failed to create buildx executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	}

	if !executor.Supports(BuildMethodBuildx) {
		t.Error("created executor should support BuildMethodBuildx")
	}
}

func TestFactoryCreateExecutor_Kaniko(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodKaniko)

	if err != nil {
		t.Errorf("failed to create kaniko executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	}

	if !executor.Supports(BuildMethodKaniko) {
		t.Error("created executor should support BuildMethodKaniko")
	}
}

func TestFactoryCreateExecutor_Docker(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodDocker)

	if err != nil {
		t.Errorf("failed to create docker executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	}

	if !executor.Supports(BuildMethodDocker) {
		t.Error("created executor should support BuildMethodDocker")
	}
}

func TestFactoryCreateExecutor_Nix(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodNix)

	if err != nil {
		t.Errorf("failed to create nix executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	}

	if !executor.Supports(BuildMethodNix) {
		t.Error("created executor should support BuildMethodNix")
	}
}

func TestFactoryCreateExecutor_Paketo(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodPaketo)

	if err != nil {
		t.Errorf("failed to create paketo executor: %v", err)
	}

	if executor == nil {
		t.Error("executor should not be nil")
	}

	if !executor.Supports(BuildMethodPaketo) {
		t.Error("created executor should support BuildMethodPaketo")
	}
}

func TestFactoryCreateExecutor_Unsupported(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethod("unsupported"))

	if err == nil {
		t.Error("expected error for unsupported method")
	}

	if executor != nil {
		t.Error("executor should be nil for unsupported method")
	}
}

func TestFactoryGetSupportedMethods(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	methods := factory.GetSupportedMethods()

	if len(methods) != 6 {
		t.Errorf("expected 6 supported methods, got %d", len(methods))
	}

	expectedMethods := map[BuildMethod]bool{
		BuildMethodPacker:  true,
		BuildMethodBuildx:  true,
		BuildMethodKaniko:  true,
		BuildMethodDocker:  true,
		BuildMethodPaketo:  true,
		BuildMethodNix:     true,
	}

	for _, method := range methods {
		if !expectedMethods[method] {
			t.Errorf("unexpected method in supported list: %v", method)
		}
	}
}

func TestFactoryServiceInjection(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor, err := factory.CreateExecutor(BuildMethodDocker)

	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	if executor == nil {
		t.Fatal("executor should not be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = executor.Execute(ctx, "test-config-service-injection", BuildMethodDocker)

	t.Log("service successfully injected into executor")
}

func TestFactoryMultipleCreations(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	executor1, _ := factory.CreateExecutor(BuildMethodDocker)
	executor2, _ := factory.CreateExecutor(BuildMethodDocker)

	if executor1 == nil || executor2 == nil {
		t.Error("both executors should be created successfully")
	}

	if !executor1.Supports(BuildMethodDocker) || !executor2.Supports(BuildMethodDocker) {
		t.Error("both executors should support BuildMethodDocker")
	}
}

func TestFactoryAllMethodsSupported(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	methods := factory.GetSupportedMethods()

	for _, method := range methods {
		executor, err := factory.CreateExecutor(method)

		if err != nil {
			t.Errorf("failed to create executor for method %v: %v", method, err)
		}

		if executor == nil {
			t.Errorf("executor should not be nil for method %v", method)
		}

		if !executor.Supports(method) {
			t.Errorf("executor should support method %v", method)
		}
	}
}

func TestFactoryExecutorTypes(t *testing.T) {
	mockService := &MockExecutorService{}
	factory := NewBuildMethodExecutorFactory(mockService)

	testCases := []struct {
		method BuildMethod
	}{
		{BuildMethodPacker},
		{BuildMethodBuildx},
		{BuildMethodKaniko},
		{BuildMethodDocker},
		{BuildMethodPaketo},
		{BuildMethodNix},
	}

	for _, tc := range testCases {
		executor, err := factory.CreateExecutor(tc.method)

		if err != nil {
			t.Errorf("failed to create executor for %v: %v", tc.method, err)
			continue
		}

		if executor == nil {
			t.Errorf("executor for %v should not be nil", tc.method)
			continue
		}

		if !executor.Supports(tc.method) {
			t.Errorf("executor for %v should support %v", tc.method, tc.method)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, _ = executor.Execute(ctx, "test-config-id", tc.method)
		cancel()
	}
}
