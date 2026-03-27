package build

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

var awsAMIIDPattern = regexp.MustCompile(`\bami-[0-9a-fA-F]{8,17}\b`)

func packerTargetProfileIDFromConfig(config *BuildConfigData) string {
	if config == nil {
		return ""
	}
	profileID := strings.TrimSpace(config.PackerTargetProfileID)
	if profileID != "" {
		return profileID
	}
	return strings.TrimSpace(metadataString(config.Metadata, "packer_target_profile_id", "packerTargetProfileId"))
}

func packerTargetProviderFromConfig(config *BuildConfigData) string {
	if config == nil {
		return ""
	}
	return strings.TrimSpace(metadataString(config.Metadata, "packer_target_provider", "packerTargetProvider"))
}

func parseExecutionMetadata(raw json.RawMessage) map[string]interface{} {
	metadata := map[string]interface{}{}
	if len(raw) == 0 {
		return metadata
	}
	if err := json.Unmarshal(raw, &metadata); err != nil || metadata == nil {
		return map[string]interface{}{}
	}
	return metadata
}

func deriveProviderArtifactIdentifiers(raw []byte) map[string][]string {
	if len(raw) == 0 {
		return nil
	}

	values := collectArtifactValues(raw)
	if len(values) == 0 {
		return nil
	}

	providerValues := map[string]map[string]struct{}{
		"aws":    {},
		"azure":  {},
		"gcp":    {},
		"vmware": {},
	}

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		for _, ami := range awsAMIIDPattern.FindAllString(value, -1) {
			providerValues["aws"][strings.ToLower(strings.TrimSpace(ami))] = struct{}{}
		}

		lower := strings.ToLower(value)
		if strings.Contains(lower, "/subscriptions/") && strings.Contains(lower, "/resourcegroups/") {
			providerValues["azure"][value] = struct{}{}
		}
		if (strings.Contains(lower, "/projects/") || strings.HasPrefix(lower, "projects/")) && strings.Contains(lower, "/global/images/") {
			providerValues["gcp"][value] = struct{}{}
		}
		if strings.Contains(lower, "googleapis.com/compute") && strings.Contains(lower, "/images/") {
			providerValues["gcp"][value] = struct{}{}
		}
		if strings.Contains(lower, "vsphere") ||
			(strings.Contains(lower, "vmware") && (strings.ContainsAny(lower, "/:\\") || strings.Contains(lower, "template"))) ||
			strings.Contains(lower, "/templates/") ||
			strings.HasSuffix(lower, ".ova") {
			providerValues["vmware"][value] = struct{}{}
		}
	}

	out := map[string][]string{}
	for provider, identifiers := range providerValues {
		if len(identifiers) == 0 {
			continue
		}
		list := make([]string, 0, len(identifiers))
		for identifier := range identifiers {
			list = append(list, identifier)
		}
		sort.Strings(list)
		out[provider] = list
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func collectArtifactValues(raw []byte) []string {
	values := make([]string, 0, 16)
	seen := map[string]struct{}{}

	appendValue := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, exists := seen[v]; exists {
			return
		}
		seen[v] = struct{}{}
		values = append(values, v)
	}

	var tektonArtifacts []Artifact
	if err := json.Unmarshal(raw, &tektonArtifacts); err == nil && len(tektonArtifacts) > 0 {
		for _, artifact := range tektonArtifacts {
			appendValue(artifact.Name)
			appendValue(artifact.Value)
			appendValue(artifact.Path)
		}
		return values
	}

	var methodArtifacts []MethodArtifact
	if err := json.Unmarshal(raw, &methodArtifacts); err == nil && len(methodArtifacts) > 0 {
		for _, artifact := range methodArtifacts {
			appendValue(artifact.Name)
			appendValue(artifact.Path)
		}
		return values
	}

	var generic []string
	if err := json.Unmarshal(raw, &generic); err == nil && len(generic) > 0 {
		for _, value := range generic {
			appendValue(value)
		}
		return values
	}

	var payload interface{}
	if err := json.Unmarshal(raw, &payload); err == nil {
		collectGenericArtifactValues(payload, appendValue)
	}

	return values
}

func collectGenericArtifactValues(value interface{}, appendValue func(string)) {
	switch typed := value.(type) {
	case string:
		appendValue(typed)
	case []interface{}:
		for _, item := range typed {
			collectGenericArtifactValues(item, appendValue)
		}
	case map[string]interface{}:
		for _, item := range typed {
			collectGenericArtifactValues(item, appendValue)
		}
	}
}
