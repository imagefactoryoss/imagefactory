package build

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CreateBuild creates a new build and queues it for execution.
func (s *Service) CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest BuildManifest, actorID *uuid.UUID) (*Build, error) {
	return s.createBuild(ctx, tenantID, projectID, manifest, actorID, true)
}

// CreateBuildDraft creates a new build in pending status without queueing execution.
func (s *Service) CreateBuildDraft(ctx context.Context, tenantID, projectID uuid.UUID, manifest BuildManifest, actorID *uuid.UUID) (*Build, error) {
	return s.createBuild(ctx, tenantID, projectID, manifest, actorID, false)
}

func (s *Service) createBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest BuildManifest, actorID *uuid.UUID, autoQueue bool) (*Build, error) {
	s.logger.Info("Creating new build",
		zap.String("tenant_id", tenantID.String()),
		zap.String("project_id", projectID.String()),
		zap.String("build_name", manifest.Name),
		zap.Bool("auto_queue", autoQueue))

	if err := s.preflightCreateBuild(ctx, tenantID, projectID, &manifest); err != nil {
		return nil, err
	}

	build, err := NewBuild(tenantID, projectID, manifest, actorID)
	if err != nil {
		s.logger.Error("Failed to create build aggregate", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, err
	}

	if autoQueue {
		if err := build.Queue(); err != nil {
			s.logger.Error("Failed to queue build after creation", zap.Error(err), zap.String("build_id", build.ID().String()))
			return nil, err
		}
	}

	if err := s.repository.Save(ctx, build); err != nil {
		s.logger.Error("Failed to save build", zap.Error(err), zap.String("build_id", build.ID().String()))
		return nil, fmt.Errorf("failed to save build: %w", err)
	}

	if s.persistBuildConfigFromManifest(ctx, build, manifest) {
		return build, nil
	}

	event := NewBuildCreated(build.ID(), build.TenantID(), build.Manifest())
	if err := s.eventPublisher.PublishBuildCreated(ctx, event); err != nil {
		s.logger.Error("Failed to publish build created event", zap.Error(err), zap.String("build_id", build.ID().String()))
	}

	s.logger.Info("Build created successfully", zap.String("build_id", build.ID().String()))
	return build, nil
}
