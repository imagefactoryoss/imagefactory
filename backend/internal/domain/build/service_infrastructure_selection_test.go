package build

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

type stubBuildRepo struct {
	lastSaved *Build
}

func (s *stubBuildRepo) Save(ctx context.Context, build *Build) error {
	s.lastSaved = build
	return nil
}
func (s *stubBuildRepo) FindByID(ctx context.Context, id uuid.UUID) (*Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Build, error) {
	return map[uuid.UUID]*Build{}, nil
}
func (s *stubBuildRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) FindByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) FindByStatus(ctx context.Context, status BuildStatus, limit, offset int) ([]*Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) Update(ctx context.Context, build *Build) error { return nil }
func (s *stubBuildRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status BuildStatus, startedAt, completedAt *time.Time, errorMessage *string) error {
	return nil
}
func (s *stubBuildRepo) ClaimNextQueuedBuild(ctx context.Context) (*Build, error) { return nil, nil }
func (s *stubBuildRepo) RequeueBuild(ctx context.Context, id uuid.UUID, nextRunAt time.Time, errorMessage *string) error {
	return nil
}
func (s *stubBuildRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (s *stubBuildRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubBuildRepo) CountByStatus(ctx context.Context, tenantID uuid.UUID, status BuildStatus) (int, error) {
	return 0, nil
}
func (s *stubBuildRepo) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubBuildRepo) FindRunningBuilds(ctx context.Context) ([]*Build, error) { return nil, nil }
func (s *stubBuildRepo) SaveBuildConfig(ctx context.Context, config *BuildConfigData) error {
	return nil
}
func (s *stubBuildRepo) GetBuildConfig(ctx context.Context, buildID uuid.UUID) (*BuildConfigData, error) {
	return nil, nil
}
func (s *stubBuildRepo) UpdateBuildConfig(ctx context.Context, config *BuildConfigData) error {
	return nil
}
func (s *stubBuildRepo) DeleteBuildConfig(ctx context.Context, buildID uuid.UUID) error { return nil }
func (s *stubBuildRepo) UpdateInfrastructureSelection(ctx context.Context, build *Build) error {
	return nil
}

type stubTriggerRepo struct{}

func (s *stubTriggerRepo) SaveTrigger(ctx context.Context, trigger *BuildTrigger) error { return nil }
func (s *stubTriggerRepo) GetTrigger(ctx context.Context, triggerID uuid.UUID) (*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) GetTriggersByBuild(ctx context.Context, buildID uuid.UUID) ([]*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) GetTriggersByProject(ctx context.Context, projectID uuid.UUID) ([]*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) GetActiveScheduledTriggers(ctx context.Context, tenantID uuid.UUID) ([]*BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) UpdateTrigger(ctx context.Context, trigger *BuildTrigger) error { return nil }
func (s *stubTriggerRepo) DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error   { return nil }

type stubEventPublisher struct{}

func (s *stubEventPublisher) PublishBuildCreated(ctx context.Context, event *BuildCreated) error {
	return nil
}
func (s *stubEventPublisher) PublishBuildStarted(ctx context.Context, event *BuildStarted) error {
	return nil
}
func (s *stubEventPublisher) PublishBuildCompleted(ctx context.Context, event *BuildCompleted) error {
	return nil
}

func (s *stubEventPublisher) PublishBuildStatusUpdated(ctx context.Context, event *BuildStatusUpdated) error {
	return nil
}

type stubSystemConfigService struct{}

func (s *stubSystemConfigService) GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.BuildConfig, error) {
	return &systemconfig.BuildConfig{MaxConcurrentJobs: 10, MaxQueueSize: 100, TektonEnabled: true}, nil
}
func (s *stubSystemConfigService) GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.ToolAvailabilityConfig, error) {
	return &systemconfig.ToolAvailabilityConfig{
		BuildMethods:   systemconfig.BuildMethodAvailability{Kaniko: true},
		SBOMTools:      systemconfig.SBOMToolAvailability{Syft: true, Trivy: true},
		ScanTools:      systemconfig.ScanToolAvailability{Trivy: true},
		RegistryTypes:  systemconfig.RegistryTypeAvailability{Harbor: true},
		SecretManagers: systemconfig.SecretManagerAvailability{Vault: true},
	}, nil
}

type stubExecutionService struct{}

func (s *stubExecutionService) StartBuild(ctx context.Context, configID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error) {
	return nil, nil
}
func (s *stubExecutionService) CancelBuild(ctx context.Context, executionID uuid.UUID) error {
	return nil
}
func (s *stubExecutionService) RetryBuild(ctx context.Context, executionID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error) {
	return nil, nil
}
func (s *stubExecutionService) GetExecution(ctx context.Context, executionID uuid.UUID) (*BuildExecution, error) {
	return nil, nil
}
func (s *stubExecutionService) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error) {
	return nil, 0, nil
}
func (s *stubExecutionService) ListRunningExecutions(ctx context.Context) ([]BuildExecution, error) {
	return nil, nil
}
func (s *stubExecutionService) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error) {
	return nil, 0, nil
}
func (s *stubExecutionService) AddLog(ctx context.Context, executionID uuid.UUID, level LogLevel, message string, metadata []byte) error {
	return nil
}
func (s *stubExecutionService) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error {
	return nil
}
func (s *stubExecutionService) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	return nil
}
func (s *stubExecutionService) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return true, nil
}
func (s *stubExecutionService) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return true, nil
}
func (s *stubExecutionService) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	return nil
}
func (s *stubExecutionService) CompleteExecution(ctx context.Context, executionID uuid.UUID, success bool, errorMsg string, artifacts []byte) error {
	return nil
}
func (s *stubExecutionService) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) error {
	return nil
}

func TestCreateBuild_PersistsProviderSelection(t *testing.T) {
	repo := &stubBuildRepo{}
	execService := &stubExecutionService{}
	localFactory := NewBuildMethodExecutorFactory(execService)
	noopExecutor := NewNoOpBuildExecutor(zap.NewNop())
	service := NewService(
		repo,
		&stubTriggerRepo{},
		&stubEventPublisher{},
		noopExecutor,
		noopExecutor,
		execService,
		localFactory,
		nil,
		&stubSystemConfigService{},
		nil,
		zap.NewNop(),
	)

	providerID := uuid.New()
	manifest := BuildManifest{
		Name:                     "Build",
		Type:                     BuildTypeKaniko,
		InfrastructureType:       "kubernetes",
		InfrastructureProviderID: &providerID,
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeHarbor,
			SecretManagerType: SecretManagerVault,
			Dockerfile:        "FROM alpine",
			BuildContext:      ".",
			RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
		},
	}

	_, err := service.CreateBuild(context.Background(), uuid.New(), uuid.New(), manifest, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSaved == nil {
		t.Fatalf("expected build to be saved")
	}
	if repo.lastSaved.InfrastructureProviderID() == nil || *repo.lastSaved.InfrastructureProviderID() != providerID {
		t.Fatalf("expected provider ID to be persisted")
	}
}
