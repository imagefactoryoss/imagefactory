package build

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) persistPackerExecutionMetadata(ctx context.Context, executionID uuid.UUID, b *Build, artifactsPayload []byte) {
	if s == nil || s.executionService == nil || b == nil {
		return
	}
	cfg := b.Config()
	if cfg == nil || cfg.BuildMethod != string(BuildMethodPacker) {
		return
	}

	profileID := packerTargetProfileIDFromConfig(cfg)
	provider := packerTargetProviderFromConfig(cfg)
	providerArtifacts := deriveProviderArtifactIdentifiers(artifactsPayload)
	if profileID == "" && provider == "" && len(providerArtifacts) == 0 {
		return
	}

	execution, err := s.executionService.GetExecution(ctx, executionID)
	if err != nil {
		s.logger.Warn("Failed to load execution for packer metadata update",
			zap.String("execution_id", executionID.String()),
			zap.Error(err))
		return
	}

	metadata := parseExecutionMetadata(execution.Metadata)
	packerMetadata, _ := metadata["packer"].(map[string]interface{})
	if packerMetadata == nil {
		packerMetadata = map[string]interface{}{}
	}
	if profileID != "" {
		packerMetadata["target_profile_id"] = profileID
	}
	if provider != "" {
		packerMetadata["target_provider"] = provider
	}
	if len(providerArtifacts) > 0 {
		packerMetadata["provider_artifact_identifiers"] = providerArtifacts
	}
	metadata["packer"] = packerMetadata

	payload, err := json.Marshal(metadata)
	if err != nil {
		s.logger.Warn("Failed to marshal packer execution metadata",
			zap.String("execution_id", executionID.String()),
			zap.Error(err))
		return
	}
	if err := s.executionService.UpdateExecutionMetadata(ctx, executionID, payload); err != nil {
		s.logger.Warn("Failed to persist packer execution metadata",
			zap.String("execution_id", executionID.String()),
			zap.Error(err))
	}
}
