package build

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// SystemConfigService defines the interface for system configuration operations.
type SystemConfigService interface {
	GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.BuildConfig, error)
	GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.ToolAvailabilityConfig, error)
}

type BuildCapabilitiesConfigProvider interface {
	GetBuildCapabilitiesConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.BuildCapabilitiesConfig, error)
}

// RegistryAuthResolver validates and resolves registry auth selection for builds.
type RegistryAuthResolver interface {
	ResolveForBuild(ctx context.Context, tenantID, projectID uuid.UUID, selectedID *uuid.UUID) (*uuid.UUID, error)
}

// Service defines the business logic for build management.
type Service struct {
	repository                 Repository
	triggerRepository          TriggerRepository
	eventPublisher             EventPublisher
	containerExecutor          BuildExecutor
	vmExecutor                 BuildExecutor
	executionService           BuildExecutionService
	localExecutorFactory       BuildMethodExecutorFactory
	tektonExecutorFactory      BuildMethodExecutorFactory
	systemConfigService        SystemConfigService
	registryAuthResolver       RegistryAuthResolver
	projectGitAuthLookup       projectGitAuthLookup
	projectSourceGitAuthLookup projectSourceGitAuthLookup
	projectBuildSettingsLookup projectBuildSettingsLookup
	projectService             interface{} // ProjectService for access control
	logger                     *zap.Logger
}

// NewService creates a new build service.
func NewService(
	repository Repository,
	triggerRepository TriggerRepository,
	eventPublisher EventPublisher,
	containerExecutor, vmExecutor BuildExecutor,
	executionService BuildExecutionService,
	localExecutorFactory BuildMethodExecutorFactory,
	tektonExecutorFactory BuildMethodExecutorFactory,
	systemConfigService SystemConfigService,
	projectService interface{},
	logger *zap.Logger,
) *Service {
	return &Service{
		repository:            repository,
		triggerRepository:     triggerRepository,
		eventPublisher:        eventPublisher,
		containerExecutor:     containerExecutor,
		vmExecutor:            vmExecutor,
		executionService:      executionService,
		localExecutorFactory:  localExecutorFactory,
		tektonExecutorFactory: tektonExecutorFactory,
		systemConfigService:   systemConfigService,
		projectService:        projectService,
		logger:                logger,
	}
}

// SetRegistryAuthResolver configures registry authentication validation and default resolution.
func (s *Service) SetRegistryAuthResolver(resolver RegistryAuthResolver) {
	s.registryAuthResolver = resolver
}

func (s *Service) SetProjectBuildSettingsLookup(lookup func(ctx context.Context, projectID uuid.UUID) (*ProjectBuildSettings, error)) {
	if lookup == nil {
		s.projectBuildSettingsLookup = nil
		return
	}
	s.projectBuildSettingsLookup = projectBuildSettingsLookup(lookup)
}
