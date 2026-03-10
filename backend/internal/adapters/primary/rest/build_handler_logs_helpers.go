package rest

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/srikarm/image-factory/internal/domain/build"
)

func parseBuildLogsFilter(query url.Values) (buildLogsFilter, error) {
	filter := buildLogsFilter{
		source:   buildLogSourceAll,
		minLevel: "",
	}

	source := strings.TrimSpace(strings.ToLower(query.Get("source")))
	switch source {
	case "", string(buildLogSourceAll):
		filter.source = buildLogSourceAll
	case string(buildLogSourceTekton):
		filter.source = buildLogSourceTekton
	case string(buildLogSourceLifecycle):
		filter.source = buildLogSourceLifecycle
	default:
		return buildLogsFilter{}, fmt.Errorf("invalid source filter: %q (supported: all, tekton, lifecycle)", source)
	}

	minLevel := strings.TrimSpace(strings.ToLower(query.Get("min_level")))
	if minLevel == "" {
		return filter, nil
	}
	switch build.LogLevel(minLevel) {
	case build.LogDebug, build.LogInfo, build.LogWarn, build.LogError:
		filter.minLevel = build.LogLevel(minLevel)
		return filter, nil
	default:
		return buildLogsFilter{}, fmt.Errorf("invalid min_level filter: %q (supported: debug, info, warn, error)", minLevel)
	}
}

func applyBuildLogsFilter(logs []LogEntry, filter buildLogsFilter) []LogEntry {
	if len(logs) == 0 {
		return logs
	}
	filtered := make([]LogEntry, 0, len(logs))
	minRank := buildLogLevelRank(filter.minLevel)

	for _, entry := range logs {
		if !matchesBuildLogSourceFilter(entry, filter.source) {
			continue
		}
		if filter.minLevel != "" {
			entryLevel := build.LogLevel(strings.TrimSpace(strings.ToLower(entry.Level)))
			if buildLogLevelRank(entryLevel) < minRank {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func matchesBuildLogSourceFilter(entry LogEntry, source buildLogSourceFilter) bool {
	if source == buildLogSourceAll {
		return true
	}

	entrySource := ""
	if entry.Metadata != nil {
		if raw, ok := entry.Metadata["source"]; ok {
			entrySource = strings.TrimSpace(strings.ToLower(fmt.Sprint(raw)))
		}
	}

	isTekton := entrySource == "tekton"
	if source == buildLogSourceTekton {
		return isTekton
	}
	return !isTekton
}

func buildLogLevelRank(level build.LogLevel) int {
	switch level {
	case build.LogDebug:
		return 0
	case build.LogInfo:
		return 1
	case build.LogWarn:
		return 2
	case build.LogError:
		return 3
	default:
		return 1
	}
}
