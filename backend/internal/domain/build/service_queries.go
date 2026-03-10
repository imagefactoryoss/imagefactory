package build

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type allTenantBuildRepository interface {
	FindAll(ctx context.Context, limit, offset int) ([]*Build, error)
	CountAll(ctx context.Context) (int, error)
}

// SupportsKubernetesInfrastructure returns whether Kubernetes infrastructure can be used
// for build execution in the current tenant/server configuration.
func (s *Service) SupportsKubernetesInfrastructure(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if err := s.validateTektonExecutable(ctx, tenantID); err != nil {
		return false, err
	}
	return true, nil
}

// GetBuild retrieves a build by ID.
func (s *Service) GetBuild(ctx context.Context, buildID uuid.UUID) (*Build, error) {
	s.logger.Debug("Retrieving build", zap.String("build_id", buildID.String()))

	build, err := s.repository.FindByID(ctx, buildID)
	if err != nil {
		s.logger.Error("Failed to find build", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to find build: %w", err)
	}

	if build == nil {
		s.logger.Warn("Build not found", zap.String("build_id", buildID.String()))
		return nil, ErrBuildNotFound
	}

	return build, nil
}

// GetBuildExecutions retrieves executions for a build.
func (s *Service) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error) {
	return s.executionService.GetBuildExecutions(ctx, buildID, limit, offset)
}

// ListBuilds retrieves builds for a tenant with pagination.
func (s *Service) ListBuilds(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Build, error) {
	s.logger.Debug("Listing builds", zap.String("tenant_id", tenantID.String()), zap.Int("limit", limit), zap.Int("offset", offset))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	builds, err := s.repository.FindByTenantID(ctx, tenantID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list builds", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}

	return builds, nil
}

// ListBuildsAllTenants retrieves builds across all tenants with pagination.
func (s *Service) ListBuildsAllTenants(ctx context.Context, limit, offset int) ([]*Build, error) {
	s.logger.Debug("Listing builds across all tenants", zap.Int("limit", limit), zap.Int("offset", offset))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	repo, ok := s.repository.(allTenantBuildRepository)
	if !ok {
		return nil, fmt.Errorf("build repository does not support all-tenant build listing")
	}

	builds, err := repo.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list builds across all tenants", zap.Error(err))
		return nil, fmt.Errorf("failed to list builds across all tenants: %w", err)
	}
	return builds, nil
}

// ListBuildsByProject retrieves builds for a specific project with pagination.
func (s *Service) ListBuildsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Build, error) {
	s.logger.Debug("Listing builds by project", zap.String("project_id", projectID.String()), zap.Int("limit", limit), zap.Int("offset", offset))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	builds, err := s.repository.FindByProjectID(ctx, projectID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list builds by project", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, fmt.Errorf("failed to list builds by project: %w", err)
	}

	return builds, nil
}

// GetBuildCountByProject retrieves the total number of builds for a specific project.
func (s *Service) GetBuildCountByProject(ctx context.Context, projectID uuid.UUID) (int, error) {
	s.logger.Debug("Getting build count by project", zap.String("project_id", projectID.String()))

	count, err := s.repository.CountByProjectID(ctx, projectID)
	if err != nil {
		s.logger.Error("Failed to get build count by project", zap.Error(err), zap.String("project_id", projectID.String()))
		return 0, fmt.Errorf("failed to get build count by project: %w", err)
	}

	return count, nil
}

// GetBuildCount returns the count of builds for a tenant.
func (s *Service) GetBuildCount(ctx context.Context, tenantID uuid.UUID) (int, error) {
	count, err := s.repository.CountByTenantID(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to count builds", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return 0, fmt.Errorf("failed to count builds: %w", err)
	}
	return count, nil
}

// GetBuildCountAllTenants returns build count across all tenants.
func (s *Service) GetBuildCountAllTenants(ctx context.Context) (int, error) {
	repo, ok := s.repository.(allTenantBuildRepository)
	if !ok {
		return 0, fmt.Errorf("build repository does not support all-tenant build counting")
	}

	count, err := repo.CountAll(ctx)
	if err != nil {
		s.logger.Error("Failed to count builds across all tenants", zap.Error(err))
		return 0, fmt.Errorf("failed to count builds across all tenants: %w", err)
	}
	return count, nil
}
