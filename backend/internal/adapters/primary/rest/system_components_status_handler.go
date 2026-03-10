package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type SystemComponentsStatusHandler struct {
	systemConfigService *systemconfig.Service
	logger              *zap.Logger
}

func NewSystemComponentsStatusHandler(systemConfigService *systemconfig.Service, logger *zap.Logger) *SystemComponentsStatusHandler {
	return &SystemComponentsStatusHandler{
		systemConfigService: systemConfigService,
		logger:              logger,
	}
}

type ComponentStatus struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	LastCheck  string                 `json:"last_check"`
	Message    string                 `json:"message,omitempty"`
	Endpoint   string                 `json:"endpoint,omitempty"`
	HTTPStatus int                    `json:"http_status,omitempty"`
	LatencyMS  int64                  `json:"latency_ms,omitempty"`
	Configured bool                   `json:"configured"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

type SystemComponentsStatusResponse struct {
	Status     string                     `json:"status"`
	CheckedAt  string                     `json:"checked_at"`
	Components map[string]ComponentStatus `json:"components"`
}

func (h *SystemComponentsStatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	authCtx, _ := middleware.GetAuthContext(r)
	if authCtx == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	components := map[string]ComponentStatus{
		"api": {
			Name:       "API Server",
			Status:     "healthy",
			LastCheck:  now,
			Message:    "API process is responding",
			Configured: true,
		},
	}

	runtimeCfg, runtimeErr := h.getRuntimeServicesConfig(r.Context(), authCtx)
	if runtimeErr != nil {
		h.logger.Warn("Runtime services config not found for status check", zap.Error(runtimeErr))
	}

	timeoutSeconds := 5
	if runtimeCfg != nil && runtimeCfg.HealthCheckTimeoutSecond > 0 {
		timeoutSeconds = runtimeCfg.HealthCheckTimeoutSecond
	}

	dispatcherEndpoint := composeServiceEndpoint(runtimeCfg, runtimeCfg != nil, "dispatcher")
	components["dispatcher"] = h.checkComponent(r.Context(), "Dispatcher", dispatcherEndpoint, timeoutSeconds)

	emailEndpoint := composeServiceEndpoint(runtimeCfg, runtimeCfg != nil, "email")
	components["email_worker"] = h.checkComponent(r.Context(), "Email Worker", emailEndpoint, timeoutSeconds)

	notificationEndpoint := composeServiceEndpoint(runtimeCfg, runtimeCfg != nil, "notification")
	components["notification_worker"] = h.checkComponent(r.Context(), "Notification Worker", notificationEndpoint, timeoutSeconds)
	internalRegistryGCEnabled := true
	if runtimeCfg != nil && runtimeCfg.InternalRegistryTempCleanupEnabled != nil {
		internalRegistryGCEnabled = *runtimeCfg.InternalRegistryTempCleanupEnabled
	} else {
		internalRegistryGCEnabled = parseBoolEnv("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_ENABLED", true)
	}
	if internalRegistryGCEnabled {
		registryGCEndpoint := composeServiceEndpoint(runtimeCfg, runtimeCfg != nil, "internal_registry_gc")
		components["internal_registry_gc_worker"] = h.checkComponent(r.Context(), "Internal Registry GC Worker", registryGCEndpoint, timeoutSeconds)
	} else {
		components["internal_registry_gc_worker"] = ComponentStatus{
			Name:       "Internal Registry GC Worker",
			Status:     "warning",
			LastCheck:  now,
			Message:    "Internal registry temp cleanup is disabled",
			Configured: true,
		}
	}

	messagingStatus := ComponentStatus{
		Name:       "Messaging",
		Status:     "warning",
		LastCheck:  now,
		Message:    "Messaging configuration not found",
		Configured: false,
	}
	if msgCfg, err := h.systemConfigService.GetConfigByTypeAndKey(r.Context(), nil, systemconfig.ConfigTypeMessaging, "messaging"); err == nil {
		if parsed, parseErr := msgCfg.GetMessagingConfig(); parseErr == nil {
			messagingStatus.Configured = true
			if parsed.EnableNATS {
				if components["notification_worker"].Status == "healthy" {
					messagingStatus.Status = "healthy"
					messagingStatus.Message = "NATS messaging enabled"
				} else {
					messagingStatus.Status = "warning"
					messagingStatus.Message = "NATS enabled, but notification worker is not healthy"
				}
			} else {
				messagingStatus.Status = "warning"
				messagingStatus.Message = "NATS messaging disabled"
			}
		}
	}
	components["messaging"] = messagingStatus

	response := SystemComponentsStatusResponse{
		Status:     summarizeStatus(components),
		CheckedAt:  now,
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *SystemComponentsStatusHandler) getRuntimeServicesConfig(ctx context.Context, authCtx *middleware.AuthContext) (*systemconfig.RuntimeServicesConfig, error) {
	if authCtx == nil {
		return nil, errors.New("missing auth context")
	}

	tenantID := authCtx.TenantID
	configs, err := h.systemConfigService.GetConfigsByType(ctx, &tenantID, systemconfig.ConfigTypeRuntimeServices)
	if err == nil {
		for _, cfg := range configs {
			if strings.EqualFold(cfg.ConfigKey(), "runtime_services") && cfg.IsActive() {
				return cfg.GetRuntimeServicesConfig()
			}
		}
	}

	globalCfg, err := h.systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services")
	if err != nil {
		return nil, err
	}
	return globalCfg.GetRuntimeServicesConfig()
}

func (h *SystemComponentsStatusHandler) checkComponent(ctx context.Context, name, endpoint string, timeoutSeconds int) ComponentStatus {
	now := time.Now().UTC().Format(time.RFC3339)
	status := ComponentStatus{
		Name:       name,
		Status:     "warning",
		LastCheck:  now,
		Endpoint:   endpoint,
		Configured: strings.TrimSpace(endpoint) != "",
	}

	if strings.TrimSpace(endpoint) == "" {
		status.Message = "Runtime endpoint is not configured"
		return status
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		status.Message = fmt.Sprintf("Invalid endpoint: %v", err)
		return status
	}

	start := time.Now()
	resp, err := client.Do(req)
	status.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		status.Status = "critical"
		status.Message = classifyNetworkError(err)
		return status
	}
	defer resp.Body.Close()

	status.HTTPStatus = resp.StatusCode

	// Read a bounded response body section for debugging/details parsing.
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	bodyText := strings.TrimSpace(string(bodyBytes))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		status.Status = "healthy"
		status.Message = "Service reachable"
		if details := parseComponentHealthDetails(name, bodyBytes); len(details) > 0 {
			status.Details = details
			if msg, ok := details["summary"].(string); ok && strings.TrimSpace(msg) != "" {
				status.Message = msg
			}
		}
		return status
	}

	if bodyText != "" {
		status.Message = fmt.Sprintf("Health check returned %d: %s", resp.StatusCode, bodyText)
	} else {
		status.Message = fmt.Sprintf("Health check returned HTTP %d", resp.StatusCode)
	}

	if resp.StatusCode >= 500 {
		status.Status = "critical"
	} else {
		status.Status = "warning"
	}

	return status
}

func parseComponentHealthDetails(componentName string, body []byte) map[string]interface{} {
	if strings.TrimSpace(componentName) != "Internal Registry GC Worker" || len(body) == 0 {
		return nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	deleted := toInt64(payload["last_run_deleted"])
	reclaimed := toInt64(payload["last_run_reclaimed_bytes"])
	totalDeleted := toInt64(payload["total_deleted"])
	totalReclaimed := toInt64(payload["total_reclaimed_bytes"])
	lastRunAt, _ := payload["last_run_at"].(string)

	summary := fmt.Sprintf(
		"Last run deleted %d images, reclaimed ~%s (total deleted: %d, total reclaimed: ~%s)",
		deleted,
		formatBytes(reclaimed),
		totalDeleted,
		formatBytes(totalReclaimed),
	)

	return map[string]interface{}{
		"last_run_deleted":         deleted,
		"last_run_reclaimed_bytes": reclaimed,
		"total_deleted":            totalDeleted,
		"total_reclaimed_bytes":    totalReclaimed,
		"last_run_at":              lastRunAt,
		"summary":                  summary,
	}
}

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	const unit = 1024
	div := int64(unit)
	exp := 0
	for n/div >= unit && exp < 5 {
		div *= unit
		exp++
	}
	value := float64(n) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB", "PB", "EB"}[exp]
	return fmt.Sprintf("%.2f %s", value, suffix)
}

func composeServiceEndpoint(cfg *systemconfig.RuntimeServicesConfig, configured bool, service string) string {
	if service == "internal_registry_gc" {
		if healthURL := strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_GC_WORKER_HEALTH_URL")); healthURL != "" {
			return healthURL
		}
		if configured && cfg != nil {
			if endpoint := buildHealthEndpoint(cfg.InternalRegistryGCWorkerURL, cfg.InternalRegistryGCWorkerPort, cfg.InternalRegistryGCWorkerTLSEnabled); endpoint != "" {
				return endpoint
			}
		}

		host := strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_GC_WORKER_URL"))
		port := parseIntEnv("IF_INTERNAL_REGISTRY_GC_WORKER_PORT", 8085)
		tlsEnabled := parseBoolEnv("IF_INTERNAL_REGISTRY_GC_TLS_ENABLED", false)
		if host == "" {
			host = "http://localhost"
		}
		return buildHealthEndpoint(host, port, tlsEnabled)
	}

	if !configured || cfg == nil {
		return ""
	}

	switch service {
	case "dispatcher":
		return buildHealthEndpoint(cfg.DispatcherURL, cfg.DispatcherPort, cfg.DispatcherMTLSEnabled)
	case "email":
		return buildHealthEndpoint(cfg.EmailWorkerURL, cfg.EmailWorkerPort, cfg.EmailWorkerTLSEnabled)
	case "notification":
		return buildHealthEndpoint(cfg.NotificationWorkerURL, cfg.NotificationWorkerPort, cfg.NotificationTLSEnabled)
	case "internal_registry_gc":
		return buildHealthEndpoint(cfg.InternalRegistryGCWorkerURL, cfg.InternalRegistryGCWorkerPort, cfg.InternalRegistryGCWorkerTLSEnabled)
	default:
		return ""
	}
}

func parseIntEnv(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}

func parseBoolEnv(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return defaultValue
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func buildHealthEndpoint(rawBase string, port int, tlsEnabled bool) string {
	base := strings.TrimSpace(rawBase)
	if base == "" || port <= 0 {
		return ""
	}

	if !strings.Contains(base, "://") {
		scheme := "http"
		if tlsEnabled {
			scheme = "https"
		}
		base = scheme + "://" + base
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	parsed.Host = fmt.Sprintf("%s:%d", host, port)
	parsed.Path = "/health"
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String()
}

func summarizeStatus(components map[string]ComponentStatus) string {
	hasWarning := false
	for _, component := range components {
		if component.Status == "critical" {
			return "critical"
		}
		if component.Status == "warning" {
			hasWarning = true
		}
	}
	if hasWarning {
		return "warning"
	}
	return "healthy"
}

func classifyNetworkError(err error) string {
	if err == nil {
		return "service unreachable"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "health check timeout"
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		var netErr net.Error
		if errors.As(urlErr, &netErr) && netErr.Timeout() {
			return "health check timeout"
		}
		return urlErr.Error()
	}
	return err.Error()
}
