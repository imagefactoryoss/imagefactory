package rest

import (
	"net/http"
	"strings"
)

func classifyBuildLifecycleError(err error) *buildHTTPError {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())

	switch {
	case strings.Contains(lower, "invalid repo build config"):
		return &buildHTTPError{status: http.StatusBadRequest, message: err.Error(), code: "repo_build_config_invalid"}
	case strings.Contains(lower, "repo build config policy violation"):
		return &buildHTTPError{status: http.StatusBadRequest, message: err.Error(), code: "repo_build_config_policy_violation"}
	case strings.Contains(lower, "repo build config validation failed"):
		return &buildHTTPError{status: http.StatusBadRequest, message: err.Error(), code: "repo_build_config_validation_failed"}
	case strings.Contains(lower, "repo build config violates tool availability"),
		strings.Contains(lower, "repo build config violates build capabilities"):
		return &buildHTTPError{status: http.StatusBadRequest, message: err.Error(), code: "repo_build_config_not_allowed"}
	case strings.Contains(lower, "failed to resolve repo build config"):
		return &buildHTTPError{status: http.StatusUnprocessableEntity, message: err.Error(), code: "repo_build_config_unavailable"}
	default:
		return nil
	}
}
