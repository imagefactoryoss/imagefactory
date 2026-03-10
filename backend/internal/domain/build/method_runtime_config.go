package build

import (
	"context"
	"fmt"
)

func configForMethod(ctx context.Context, method BuildMethod) (*BuildConfigData, error) {
	build, ok := BuildFromContext(ctx)
	if !ok || build == nil {
		return nil, fmt.Errorf("build context is not available")
	}

	config := build.Config()
	if config == nil {
		return nil, fmt.Errorf("build config is required")
	}

	cfgMethod := BuildMethod(config.BuildMethod)
	if cfgMethod == BuildMethod("container") {
		cfgMethod = BuildMethodDocker
	}
	if cfgMethod != method {
		return nil, fmt.Errorf("build config method mismatch: expected %s, got %s", method, cfgMethod)
	}

	return config, nil
}

func metadataString(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := metadata[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
	}
	return ""
}

func metadataBool(metadata map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if val, ok := metadata[key]; ok {
			if b, ok := val.(bool); ok {
				return b
			}
		}
	}
	return false
}

func metadataStringMap(metadata map[string]interface{}, key string) map[string]string {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	out := map[string]string{}
	switch typed := raw.(type) {
	case map[string]string:
		for k, v := range typed {
			out[k] = v
		}
	case map[string]interface{}:
		for k, v := range typed {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func metadataStringSlice(metadata map[string]interface{}, key string) []string {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	out := []string{}
	switch typed := raw.(type) {
	case []string:
		out = append(out, typed...)
	case []interface{}:
		for _, v := range typed {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
