package systemconfig

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestService_GetBuildConfig_DefaultsTempScanStageEnabled(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	mockRepo.On("FindActiveByType", ctx, tenantID, ConfigTypeBuild).Return([]*SystemConfig{}, nil).Once()
	mockRepo.On("FindUniversalByType", ctx, ConfigTypeBuild).Return([]*SystemConfig{}, nil).Once()

	cfg, err := service.GetBuildConfig(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.EnableTempScanStage)
	mockRepo.AssertExpectations(t)
}

func TestService_GetBuildConfig_OverridesTempScanStageFromBuildConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	createdBy := uuid.New()
	buildCfg, err := NewSystemConfig(&tenantID, ConfigTypeBuild, "build", map[string]interface{}{
		"enable_temp_scan_stage": false,
	}, "tenant build config", createdBy)
	require.NoError(t, err)
	require.NoError(t, buildCfg.Activate(createdBy))

	mockRepo.On("FindActiveByType", ctx, tenantID, ConfigTypeBuild).Return([]*SystemConfig{buildCfg}, nil).Once()
	mockRepo.On("FindUniversalByType", ctx, ConfigTypeBuild).Return([]*SystemConfig{}, nil).Once()

	cfg, err := service.GetBuildConfig(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.False(t, cfg.EnableTempScanStage)
	mockRepo.AssertExpectations(t)
}

func TestService_GetBuildConfig_OverridesTempScanStageFromSingleKey(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	createdBy := uuid.New()
	singleKeyCfg, err := NewSystemConfig(&tenantID, ConfigTypeBuild, "enable_temp_scan_stage", true, "temp scan stage single key", createdBy)
	require.NoError(t, err)
	require.NoError(t, singleKeyCfg.Activate(createdBy))

	mockRepo.On("FindActiveByType", ctx, tenantID, ConfigTypeBuild).Return([]*SystemConfig{singleKeyCfg}, nil).Once()
	mockRepo.On("FindUniversalByType", ctx, ConfigTypeBuild).Return([]*SystemConfig{}, nil).Once()

	cfg, err := service.GetBuildConfig(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.EnableTempScanStage)
	mockRepo.AssertExpectations(t)
}

func TestService_GetBuildConfig_GlobalFallbackTempScanStage(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	createdBy := uuid.New()
	globalCfg, err := NewSystemConfig(nil, ConfigTypeBuild, "build", map[string]interface{}{
		"enable_temp_scan_stage": false,
	}, "global build config", createdBy)
	require.NoError(t, err)
	require.NoError(t, globalCfg.Activate(createdBy))

	mockRepo.On("FindActiveByType", ctx, tenantID, ConfigTypeBuild).Return([]*SystemConfig{}, nil).Once()
	mockRepo.On("FindUniversalByType", ctx, ConfigTypeBuild).Return([]*SystemConfig{globalCfg}, nil).Once()

	cfg, err := service.GetBuildConfig(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.False(t, cfg.EnableTempScanStage)
	mockRepo.AssertExpectations(t)
}
