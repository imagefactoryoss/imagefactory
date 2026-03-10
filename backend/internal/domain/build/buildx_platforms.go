package build

import "strings"

// normalizeBuildxPlatforms trims entries and removes duplicates while preserving order.
func normalizeBuildxPlatforms(platforms []string) []string {
	if len(platforms) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(platforms))
	normalized := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		trimmed := strings.TrimSpace(platform)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}
