package build

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) preflightCreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest *BuildManifest) error {
	if manifest != nil {
		if buildConfig, err := s.systemConfigService.GetBuildConfig(ctx, tenantID); err == nil {
			applyBuildRuntimeDefaults(manifest, buildConfig)
		}
		var toolScope *uuid.UUID
		if tenantID != uuid.Nil {
			toolScope = &tenantID
		}
		if toolConfig, err := s.systemConfigService.GetToolAvailabilityConfig(ctx, toolScope); err == nil {
			applyTrivyRuntimeDefaults(manifest, toolConfig)
		}
	}

	// Validate tool availability (Phase 2.2.3 - placeholder for now)
	if err := s.validateToolAvailability(ctx, tenantID, *manifest); err != nil {
		s.logger.Error("Tool availability validation failed", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return fmt.Errorf("tool availability validation failed: %w", err)
	}
	if err := s.validateBuildCapabilities(ctx, tenantID, *manifest); err != nil {
		s.logger.Error("Build capability entitlement validation failed", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return fmt.Errorf("build capability validation failed: %w", err)
	}

	// Validate build limits using system configuration
	if err := s.validateBuildLimits(ctx, tenantID); err != nil {
		s.logger.Error("Build limit validation failed", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return fmt.Errorf("build limit validation failed: %w", err)
	}

	// Enforce Tekton-enabled when kubernetes infrastructure is selected
	if manifest.InfrastructureType == "kubernetes" {
		if err := s.validateTektonAllowed(ctx, tenantID); err != nil {
			s.logger.Warn("Tekton disabled for Kubernetes build", zap.String("tenant_id", tenantID.String()), zap.Error(err))
			return err
		}
	}

	// Enforce registry authentication for methods that push images.
	if manifest.BuildConfig != nil && requiresRegistryAuth(manifest.Type) && s.registryAuthResolver != nil {
		resolvedAuthID, err := s.registryAuthResolver.ResolveForBuild(
			ctx,
			tenantID,
			projectID,
			manifest.BuildConfig.RegistryAuthID,
		)
		if err != nil {
			s.logger.Warn("Registry authentication validation failed", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("project_id", projectID.String()))
			return fmt.Errorf("registry authentication is required: %w", err)
		}
		manifest.BuildConfig.RegistryAuthID = resolvedAuthID
	}

	return nil
}
