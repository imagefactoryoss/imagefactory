package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	buildsteps "github.com/srikarm/image-factory/internal/application/build/steps"
	domainbuild "github.com/srikarm/image-factory/internal/domain/build"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

var ErrProjectTenantMismatch = errors.New("project belongs to a different tenant")

type ErrBuildNotRetriable struct {
	Status domainbuild.BuildStatus
}

func (e ErrBuildNotRetriable) Error() string {
	return fmt.Sprintf("Cannot retry build in %s state", e.Status)
}

// ProjectTenantLookup returns the owning tenant for a project.
// `found=false` keeps compatibility with current behavior when the project cannot be resolved.
type ProjectTenantLookup func(ctx context.Context, projectID uuid.UUID) (tenantID uuid.UUID, found bool, err error)

type RetryBuildPreflight func(ctx context.Context, build *domainbuild.Build) error
type CreateBuildPreflight func(ctx context.Context, tenantID, projectID uuid.UUID, manifest *domainbuild.BuildManifest) error

type BuildWorkflowWriter interface {
	UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error)
	CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status domainworkflow.InstanceStatus) (uuid.UUID, error)
	CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []domainworkflow.StepDefinition) error
}

// BuildDomainService defines domain operations needed by application orchestration.
type BuildDomainService interface {
	CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest domainbuild.BuildManifest, actorID *uuid.UUID) (*domainbuild.Build, error)
	RetryBuild(ctx context.Context, buildID uuid.UUID) error
	GetBuild(ctx context.Context, id uuid.UUID) (*domainbuild.Build, error)
}

// Service is the application-layer entrypoint for build orchestration use cases.
// Phase 1 compatibility mode delegates to the existing domain service.
type Service struct {
	buildService        BuildDomainService
	projectTenantLookup ProjectTenantLookup
	retryPreflight      RetryBuildPreflight
	createPreflight     CreateBuildPreflight
	workflowWriter      BuildWorkflowWriter
	logger              *zap.Logger
}

func NewService(buildService BuildDomainService, logger *zap.Logger) *Service {
	return &Service{
		buildService: buildService,
		logger:       logger,
	}
}

func (s *Service) SetProjectTenantLookup(lookup ProjectTenantLookup) {
	s.projectTenantLookup = lookup
}

func (s *Service) SetRetryBuildPreflight(preflight RetryBuildPreflight) {
	s.retryPreflight = preflight
}

func (s *Service) SetCreateBuildPreflight(preflight CreateBuildPreflight) {
	s.createPreflight = preflight
}

func (s *Service) SetWorkflowWriter(workflowWriter BuildWorkflowWriter) {
	s.workflowWriter = workflowWriter
}

func (s *Service) CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest domainbuild.BuildManifest, actorID *uuid.UUID) (*domainbuild.Build, error) {
	s.logger.Debug("BuildApplicationService.CreateBuild",
		zap.String("tenant_id", tenantID.String()),
		zap.String("project_id", projectID.String()),
		zap.String("build_type", string(manifest.Type)),
	)

	if s.projectTenantLookup != nil {
		projectTenantID, found, err := s.projectTenantLookup(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("project lookup failed: %w", err)
		}
		if found && projectTenantID != tenantID {
			return nil, ErrProjectTenantMismatch
		}
	}

	if s.createPreflight != nil {
		if preflightErr := s.createPreflight(ctx, tenantID, projectID, &manifest); preflightErr != nil {
			return nil, preflightErr
		}
	}

	created, err := s.buildService.CreateBuild(ctx, tenantID, projectID, manifest, actorID)
	if err != nil {
		return nil, classifyCreateBuildError(err)
	}
	s.seedBuildWorkflow(ctx, created, false)
	return created, nil
}

func (s *Service) RetryBuild(ctx context.Context, buildID uuid.UUID) error {
	s.logger.Debug("BuildApplicationService.RetryBuild", zap.String("build_id", buildID.String()))

	existing, err := s.buildService.GetBuild(ctx, buildID)
	if err != nil || existing == nil {
		return domainbuild.ErrBuildNotFound
	}
	if existing.Status() != domainbuild.BuildStatusFailed && existing.Status() != domainbuild.BuildStatusCancelled {
		return ErrBuildNotRetriable{Status: existing.Status()}
	}
	if s.retryPreflight != nil {
		if preflightErr := s.retryPreflight(ctx, existing); preflightErr != nil {
			return preflightErr
		}
	}

	if err := s.buildService.RetryBuild(ctx, buildID); err != nil {
		return err
	}
	s.seedBuildWorkflow(ctx, existing, true)
	return nil
}

const (
	buildControlPlaneWorkflowName    = "build_control_plane"
	buildControlPlaneWorkflowVersion = 1
)

func (s *Service) seedBuildWorkflow(ctx context.Context, b *domainbuild.Build, retry bool) {
	if s.workflowWriter == nil || b == nil {
		return
	}

	manifestBytes, err := json.Marshal(b.Manifest())
	if err != nil {
		s.logger.Warn("Skipping build workflow seed: failed to marshal manifest", zap.Error(err), zap.String("build_id", b.ID().String()))
		return
	}
	var manifestMap map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifestMap); err != nil {
		s.logger.Warn("Skipping build workflow seed: failed to unmarshal manifest map", zap.Error(err), zap.String("build_id", b.ID().String()))
		return
	}

	payload := map[string]interface{}{
		"tenant_id":  b.TenantID().String(),
		"project_id": b.ProjectID().String(),
		"build_id":   b.ID().String(),
		"manifest":   manifestMap,
	}

	definition := map[string]interface{}{
		"name":    buildControlPlaneWorkflowName,
		"version": buildControlPlaneWorkflowVersion,
		"steps": []string{
			buildsteps.StepValidateBuild,
			buildsteps.StepSelectInfrastructure,
			buildsteps.StepEnqueueBuild,
			buildsteps.StepDispatchBuild,
			buildsteps.StepMonitorBuild,
			buildsteps.StepFinalizeBuild,
		},
	}

	definitionID, err := s.workflowWriter.UpsertDefinition(ctx, buildControlPlaneWorkflowName, buildControlPlaneWorkflowVersion, definition)
	if err != nil {
		s.logger.Warn("Skipping build workflow seed: failed to upsert definition", zap.Error(err), zap.String("build_id", b.ID().String()))
		return
	}

	tenantID := b.TenantID()
	instanceID, err := s.workflowWriter.CreateInstance(
		ctx,
		definitionID,
		&tenantID,
		"build",
		b.ID(),
		domainworkflow.InstanceStatusRunning,
	)
	if err != nil {
		s.logger.Warn("Skipping build workflow seed: failed to create instance", zap.Error(err), zap.String("build_id", b.ID().String()))
		return
	}

	steps := []domainworkflow.StepDefinition{
		{StepKey: buildsteps.StepValidateBuild, Payload: payload, Status: domainworkflow.StepStatusSucceeded},
		{StepKey: buildsteps.StepSelectInfrastructure, Payload: payload, Status: domainworkflow.StepStatusSucceeded},
		{StepKey: buildsteps.StepEnqueueBuild, Payload: payload, Status: domainworkflow.StepStatusSucceeded},
		{StepKey: buildsteps.StepDispatchBuild, Payload: payload, Status: domainworkflow.StepStatusPending},
		{StepKey: buildsteps.StepMonitorBuild, Payload: payload, Status: domainworkflow.StepStatusPending},
		{StepKey: buildsteps.StepFinalizeBuild, Payload: payload, Status: domainworkflow.StepStatusPending},
	}
	if retry {
		steps[3].Status = domainworkflow.StepStatusSucceeded
	}

	if err := s.workflowWriter.CreateSteps(ctx, instanceID, steps); err != nil {
		s.logger.Warn("Skipping build workflow seed: failed to create steps", zap.Error(err), zap.String("build_id", b.ID().String()))
	}
}
