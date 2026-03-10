package rest

import (
	"context"
	"os/exec"
	"strings"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
)

func requiresTektonGitContext(manifest *build.BuildManifest) bool {
	if manifest == nil || manifest.InfrastructureType != "kubernetes" {
		return false
	}
	switch manifest.Type {
	case build.BuildTypeKaniko, build.BuildTypeBuildx, build.BuildTypeContainer, build.BuildTypePacker:
		return true
	default:
		return false
	}
}

func hasManifestStringMetadata(metadata map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if val, ok := metadata[key].(string); ok && strings.TrimSpace(val) != "" {
			return true
		}
	}
	return false
}

func manifestStringMetadata(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := metadata[key].(string); ok && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

func defaultPublicRepoProbe(ctx context.Context, repoURL string) (bool, error) {
	if strings.TrimSpace(repoURL) == "" {
		return false, nil
	}
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", "--tags", repoURL)
	if err := cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
}

func isTektonEnabled(provider *infrastructure.Provider) bool {
	if provider == nil || provider.Config == nil {
		return false
	}
	enabled, ok := provider.Config["tekton_enabled"].(bool)
	return ok && enabled
}
