package build

import "strings"

func packerMetadataFromBuildConfig(config *BuildConfig) map[string]interface{} {
	if config == nil {
		return nil
	}

	metadata := map[string]interface{}{}
	if profileID := strings.TrimSpace(config.PackerTargetProfileID); profileID != "" {
		metadata["packer_target_profile_id"] = profileID
	}
	if len(config.Variables) > 0 {
		metadata["variables"] = config.Variables
	}
	if len(config.BuildVars) > 0 {
		metadata["build_vars"] = config.BuildVars
	}

	onError := strings.TrimSpace(config.OnError)
	if onError == "" {
		onError = "cleanup"
	}
	metadata["on_error"] = onError
	metadata["parallel"] = config.Parallel

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}
