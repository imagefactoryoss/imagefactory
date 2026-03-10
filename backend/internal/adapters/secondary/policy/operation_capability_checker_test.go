package policy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

type operationCapabilityRepoStub struct {
	mock.Mock
}

func (m *operationCapabilityRepoStub) Save(ctx context.Context, config *systemconfig.SystemConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *operationCapabilityRepoStub) SaveAll(ctx context.Context, configs []*systemconfig.SystemConfig) error {
	args := m.Called(ctx, configs)
	return args.Error(0)
}

func (m *operationCapabilityRepoStub) FindByID(ctx context.Context, id uuid.UUID) (*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, id)
	cfg, _ := args.Get(0).(*systemconfig.SystemConfig)
	return cfg, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, tenantID)
	cfgs, _ := args.Get(0).([]*systemconfig.SystemConfig)
	return cfgs, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindByType(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, tenantID, configType)
	cfgs, _ := args.Get(0).([]*systemconfig.SystemConfig)
	return cfgs, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindAllByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, configType)
	cfgs, _ := args.Get(0).([]*systemconfig.SystemConfig)
	return cfgs, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, tenantID, configKey)
	cfg, _ := args.Get(0).(*systemconfig.SystemConfig)
	return cfg, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType, configKey string) (*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, tenantID, configType, configKey)
	cfg, _ := args.Get(0).(*systemconfig.SystemConfig)
	return cfg, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindActiveByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, tenantID, configType)
	cfgs, _ := args.Get(0).([]*systemconfig.SystemConfig)
	return cfgs, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindAll(ctx context.Context) ([]*systemconfig.SystemConfig, error) {
	args := m.Called(ctx)
	cfgs, _ := args.Get(0).([]*systemconfig.SystemConfig)
	return cfgs, args.Error(1)
}

func (m *operationCapabilityRepoStub) FindUniversalByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	args := m.Called(ctx, configType)
	cfgs, _ := args.Get(0).([]*systemconfig.SystemConfig)
	return cfgs, args.Error(1)
}

func (m *operationCapabilityRepoStub) Update(ctx context.Context, config *systemconfig.SystemConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *operationCapabilityRepoStub) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *operationCapabilityRepoStub) ExistsByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (bool, error) {
	args := m.Called(ctx, tenantID, configKey)
	return args.Bool(0), args.Error(1)
}

func (m *operationCapabilityRepoStub) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *operationCapabilityRepoStub) CountByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) (int, error) {
	args := m.Called(ctx, tenantID, configType)
	return args.Int(0), args.Error(1)
}

func TestOperationCapabilityChecker_IsImportEntitled(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := &operationCapabilityRepoStub{}
	configValue := systemconfig.OperationCapabilitiesConfig{
		QuarantineRequest: true,
	}
	cfg, err := systemconfig.NewSystemConfig(&tenantID, systemconfig.ConfigTypeToolSettings, "operation_capabilities", configValue, "test", uuid.New())
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	repo.On("FindByKey", ctx, &tenantID, "operation_capabilities").Return(cfg, nil).Once()
	svc := systemconfig.NewService(repo, zap.NewNop())
	checker := NewOperationCapabilityChecker(svc)

	ok, err := checker.IsImportEntitled(ctx, tenantID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected import entitlement to be true")
	}
}

func TestOperationCapabilityChecker_IsOnDemandScanEntitled(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := &operationCapabilityRepoStub{}
	configValue := systemconfig.OperationCapabilitiesConfig{
		OnDemandImageScan: true,
	}
	cfg, err := systemconfig.NewSystemConfig(&tenantID, systemconfig.ConfigTypeToolSettings, "operation_capabilities", configValue, "test", uuid.New())
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	repo.On("FindByKey", ctx, &tenantID, "operation_capabilities").Return(cfg, nil).Once()
	svc := systemconfig.NewService(repo, zap.NewNop())
	checker := NewOperationCapabilityChecker(svc)

	ok, err := checker.IsOnDemandScanEntitled(ctx, tenantID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected on-demand scan entitlement to be true")
	}
}

func TestOperationCapabilityChecker_IsQuarantineReleaseEntitled(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := &operationCapabilityRepoStub{}
	configValue := systemconfig.OperationCapabilitiesConfig{
		QuarantineRelease: true,
	}
	cfg, err := systemconfig.NewSystemConfig(&tenantID, systemconfig.ConfigTypeToolSettings, "operation_capabilities", configValue, "test", uuid.New())
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	repo.On("FindByKey", ctx, &tenantID, "operation_capabilities").Return(cfg, nil).Once()
	svc := systemconfig.NewService(repo, zap.NewNop())
	checker := NewOperationCapabilityChecker(svc)

	ok, err := checker.IsQuarantineReleaseEntitled(ctx, tenantID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected quarantine release entitlement to be true")
	}
}
