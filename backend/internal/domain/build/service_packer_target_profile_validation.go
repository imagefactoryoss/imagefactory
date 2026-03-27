package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/packertarget"
)

func (s *Service) validatePackerTargetProfileForManifest(ctx context.Context, tenantID uuid.UUID, manifest BuildManifest) error {
	if manifest.Type != BuildTypePacker || manifest.BuildConfig == nil {
		return nil
	}
	if s.packerTargetProfileLookup == nil {
		return fmt.Errorf("packer target profile lookup is not configured")
	}

	profileIDRaw := strings.TrimSpace(manifest.BuildConfig.PackerTargetProfileID)
	if profileIDRaw == "" {
		profileIDRaw = strings.TrimSpace(metadataString(manifest.Metadata, "packer_target_profile_id", "packerTargetProfileId"))
	}
	if profileIDRaw == "" {
		return fmt.Errorf("packer preflight failed: packer_target_profile_id is required for packer builds")
	}

	profileID, err := uuid.Parse(profileIDRaw)
	if err != nil {
		return fmt.Errorf("packer preflight failed: invalid packer_target_profile_id %q", profileIDRaw)
	}

	profile, err := s.packerTargetProfileLookup.GetByID(ctx, tenantID, profileID)
	if err != nil {
		if err == packertarget.ErrNotFound {
			return fmt.Errorf("packer preflight failed: target profile %s is not entitled to this tenant", profileID.String())
		}
		return fmt.Errorf("packer preflight failed: failed to resolve target profile %s: %w", profileID.String(), err)
	}
	if profile == nil {
		return fmt.Errorf("packer preflight failed: target profile %s is not entitled to this tenant", profileID.String())
	}
	if strings.TrimSpace(profile.ValidationStatus) != packertarget.ValidationStatusValid {
		return fmt.Errorf("packer preflight failed: target profile %s is %q; validate the profile before starting builds", profileID.String(), strings.TrimSpace(profile.ValidationStatus))
	}

	return nil
}

func (s *Service) validatePackerTargetProfileForBuildConfig(ctx context.Context, build *Build) error {
	if build == nil {
		return nil
	}
	cfg := build.Config()
	if cfg == nil || cfg.BuildMethod != string(BuildMethodPacker) {
		return nil
	}
	if s.packerTargetProfileLookup == nil {
		return fmt.Errorf("packer target profile lookup is not configured")
	}

	profileIDRaw := strings.TrimSpace(cfg.PackerTargetProfileID)
	if profileIDRaw == "" {
		profileIDRaw = strings.TrimSpace(metadataString(cfg.Metadata, "packer_target_profile_id", "packerTargetProfileId"))
	}
	if profileIDRaw == "" {
		return fmt.Errorf("packer preflight failed: packer_target_profile_id is required for packer build config")
	}

	profileID, err := uuid.Parse(profileIDRaw)
	if err != nil {
		return fmt.Errorf("packer preflight failed: invalid packer_target_profile_id %q", profileIDRaw)
	}

	profile, err := s.packerTargetProfileLookup.GetByID(ctx, build.TenantID(), profileID)
	if err != nil {
		if err == packertarget.ErrNotFound {
			return fmt.Errorf("packer preflight failed: target profile %s is not entitled to this tenant", profileID.String())
		}
		return fmt.Errorf("packer preflight failed: failed to resolve target profile %s: %w", profileID.String(), err)
	}
	if profile == nil {
		return fmt.Errorf("packer preflight failed: target profile %s is not entitled to this tenant", profileID.String())
	}
	if strings.TrimSpace(profile.ValidationStatus) != packertarget.ValidationStatusValid {
		return fmt.Errorf("packer preflight failed: target profile %s is %q; validate the profile before starting builds", profileID.String(), strings.TrimSpace(profile.ValidationStatus))
	}

	if cfg.Metadata == nil {
		cfg.Metadata = map[string]interface{}{}
	}
	cfg.PackerTargetProfileID = profileID.String()
	cfg.Metadata["packer_target_profile_id"] = profileID.String()
	cfg.Metadata["packer_target_provider"] = profile.Provider
	build.SetConfig(cfg)
	return nil
}
