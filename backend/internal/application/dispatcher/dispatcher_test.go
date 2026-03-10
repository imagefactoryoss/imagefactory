package dispatcher

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/infrastructure/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBuildExecutor is a mock implementation of BuildExecutor
type MockBuildExecutor struct {
	mock.Mock
}

func (m *MockBuildExecutor) Execute(ctx context.Context, build *build.Build) (any, error) {
	args := m.Called(ctx, build)
	return args.Get(0), args.Error(1)
}

func (m *MockBuildExecutor) Cancel(ctx context.Context, buildID uuid.UUID) error {
	args := m.Called(ctx, buildID)
	return args.Error(0)
}

// MockQueueManager is a mock implementation of QueueManager
type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) QueueBuild(ctx context.Context, build *build.Build, executor BuildExecutor) error {
	args := m.Called(ctx, build, executor)
	return args.Error(0)
}

// MockMetricsCollector is a mock implementation of MetricsCollector
type MockMetricsCollector struct {
	mock.Mock
}

func (m *MockMetricsCollector) RecordInfrastructureUsage(ctx context.Context, build *build.Build, decision *k8s.InfrastructureDecision, duration time.Duration, err error) {
	m.Called(ctx, build, decision, duration, err)
}

// MockBuildRepository is a mock implementation of BuildRepository
type MockBuildRepository struct {
	mock.Mock
}

func (m *MockBuildRepository) UpdateInfrastructureSelection(ctx context.Context, build *build.Build) error {
	args := m.Called(ctx, build)
	return args.Error(0)
}

func TestSmartBuildDispatcher_Dispatch_Kubernetes(t *testing.T) {
	// Setup mocks
	k8sExecutor := &MockBuildExecutor{}
	nodeExecutor := &MockBuildExecutor{}
	queueManager := &MockQueueManager{}
	metrics := &MockMetricsCollector{}
	repo := &MockBuildRepository{}

	// Create infrastructure selector that returns Kubernetes
	selector := k8s.NewInfrastructureSelector(true)

	// Create dispatcher
	dispatcher := NewSmartBuildDispatcher(selector, k8sExecutor, nodeExecutor, queueManager, metrics, repo)

	// Create test build
	tenantID := uuid.New()
	projectID := uuid.New()
	manifest := build.BuildManifest{
		Name:         "test-build",
		Type:         build.BuildTypeContainer,
		BaseImage:    "ubuntu:20.04",
		Instructions: []string{"RUN apt-get update", "RUN apt-get install -y curl"},
	}
	testBuild, err := build.NewBuild(tenantID, projectID, manifest, nil)
	assert.NoError(t, err)

	// Setup expectations
	repo.On("UpdateInfrastructureSelection", mock.Anything, mock.AnythingOfType("*build.Build")).Return(nil)
	metrics.On("RecordInfrastructureUsage", mock.Anything, mock.AnythingOfType("*build.Build"), mock.AnythingOfType("*k8s.InfrastructureDecision"), mock.AnythingOfType("time.Duration"), mock.Anything).Return()
	queueManager.On("QueueBuild", mock.Anything, mock.AnythingOfType("*build.Build"), k8sExecutor).Return(nil)

	// Execute dispatch
	err = dispatcher.Dispatch(context.Background(), testBuild)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, string(k8s.InfrastructureKubernetes), testBuild.InfrastructureType())
	assert.Contains(t, testBuild.InfrastructureReason(), "Selected")
	assert.NotNil(t, testBuild.SelectedAt())

	// Verify mocks
	repo.AssertExpectations(t)
	queueManager.AssertExpectations(t)
}

func TestSmartBuildDispatcher_Dispatch_BuildNodes(t *testing.T) {
	// Setup mocks
	k8sExecutor := &MockBuildExecutor{}
	nodeExecutor := &MockBuildExecutor{}
	queueManager := &MockQueueManager{}
	metrics := &MockMetricsCollector{}
	repo := &MockBuildRepository{}

	// Create infrastructure selector that returns build nodes (K8s unavailable)
	selector := k8s.NewInfrastructureSelector(false)

	// Create dispatcher
	dispatcher := NewSmartBuildDispatcher(selector, k8sExecutor, nodeExecutor, queueManager, metrics, repo)

	// Create test build
	tenantID := uuid.New()
	projectID := uuid.New()
	manifest := build.BuildManifest{
		Name:         "test-build",
		Type:         build.BuildTypeContainer,
		BaseImage:    "ubuntu:20.04",
		Instructions: []string{"RUN apt-get update", "RUN apt-get install -y curl"},
	}
	testBuild, err := build.NewBuild(tenantID, projectID, manifest, nil)
	assert.NoError(t, err)

	// Setup expectations
	repo.On("UpdateInfrastructureSelection", mock.Anything, mock.AnythingOfType("*build.Build")).Return(nil)
	metrics.On("RecordInfrastructureUsage", mock.Anything, mock.AnythingOfType("*build.Build"), mock.AnythingOfType("*k8s.InfrastructureDecision"), mock.AnythingOfType("time.Duration"), mock.Anything).Return()
	queueManager.On("QueueBuild", mock.Anything, mock.AnythingOfType("*build.Build"), nodeExecutor).Return(nil)

	// Execute dispatch
	err = dispatcher.Dispatch(context.Background(), testBuild)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, string(k8s.InfrastructureBuildNodes), testBuild.InfrastructureType())
	assert.Contains(t, testBuild.InfrastructureReason(), "unavailable")
	assert.NotNil(t, testBuild.SelectedAt())

	// Verify mocks
	repo.AssertExpectations(t)
	queueManager.AssertExpectations(t)
}

func TestSmartBuildDispatcher_Dispatch_SelectionFailure(t *testing.T) {
	// Setup mocks
	k8sExecutor := &MockBuildExecutor{}
	nodeExecutor := &MockBuildExecutor{}
	queueManager := &MockQueueManager{}
	metrics := &MockMetricsCollector{}
	repo := &MockBuildRepository{}

	// Create dispatcher with nil selector (will cause selection failure)
	dispatcher := NewSmartBuildDispatcher(nil, k8sExecutor, nodeExecutor, queueManager, metrics, repo)

	// Create test build
	tenantID := uuid.New()
	projectID := uuid.New()
	manifest := build.BuildManifest{
		Name:         "test-build",
		Type:         build.BuildTypeContainer,
		BaseImage:    "ubuntu:20.04",
		Instructions: []string{"RUN apt-get update", "RUN apt-get install -y curl"},
	}
	testBuild, err := build.NewBuild(tenantID, projectID, manifest, nil)
	assert.NoError(t, err)

	// Setup expectations
	metrics.On("RecordInfrastructureUsage", mock.Anything, mock.AnythingOfType("*build.Build"), mock.Anything, mock.AnythingOfType("time.Duration"), mock.Anything).Return()

	// Execute dispatch
	err = dispatcher.Dispatch(context.Background(), testBuild)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "infrastructure selector not configured")
}

func TestSmartBuildDispatcher_Dispatch_RepositoryFailure(t *testing.T) {
	// Setup mocks
	k8sExecutor := &MockBuildExecutor{}
	nodeExecutor := &MockBuildExecutor{}
	queueManager := &MockQueueManager{}
	metrics := &MockMetricsCollector{}
	repo := &MockBuildRepository{}

	// Create dispatcher
	dispatcher := NewSmartBuildDispatcher(k8s.NewInfrastructureSelector(true), k8sExecutor, nodeExecutor, queueManager, metrics, repo)

	// Create test build
	tenantID := uuid.New()
	projectID := uuid.New()
	manifest := build.BuildManifest{
		Name:         "test-build",
		Type:         build.BuildTypeContainer,
		BaseImage:    "ubuntu:20.04",
		Instructions: []string{"RUN apt-get update", "RUN apt-get install -y curl"},
	}
	testBuild, err := build.NewBuild(tenantID, projectID, manifest, nil)
	assert.NoError(t, err)

	// Setup expectations - repository update fails
	repo.On("UpdateInfrastructureSelection", mock.Anything, mock.AnythingOfType("*build.Build")).Return(assert.AnError)
	metrics.On("RecordInfrastructureUsage", mock.Anything, mock.AnythingOfType("*build.Build"), mock.AnythingOfType("*k8s.InfrastructureDecision"), mock.AnythingOfType("time.Duration"), mock.Anything).Return()

	// Execute dispatch
	err = dispatcher.Dispatch(context.Background(), testBuild)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save infrastructure selection")

	// Verify mocks
	repo.AssertExpectations(t)
}