package build

import "github.com/google/uuid"

func supportsMethod(factory BuildMethodExecutorFactory, method BuildMethod) bool {
	if factory == nil {
		return false
	}
	for _, supported := range factory.GetSupportedMethods() {
		if supported == method {
			return true
		}
	}
	return false
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func requiresRegistryAuth(buildType BuildType) bool {
	switch buildType {
	case BuildTypeContainer, BuildTypeBuildx, BuildTypeKaniko, BuildTypePaketo:
		return true
	default:
		return false
	}
}
