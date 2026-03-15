package sresmartbot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	domainappsignals "github.com/srikarm/image-factory/internal/domain/appsignals"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/logdetector"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
)

type MCPToolDescriptor struct {
	ServerID         string `json:"server_id"`
	ServerName       string `json:"server_name"`
	ServerKind       string `json:"server_kind"`
	ToolName         string `json:"tool_name"`
	DisplayName      string `json:"display_name"`
	Description      string `json:"description"`
	ReadOnly         bool   `json:"read_only"`
	IncidentRequired bool   `json:"incident_required"`
}

type MCPToolInvocationRequest struct {
	IncidentID uuid.UUID `json:"incident_id"`
	ServerID   string    `json:"server_id"`
	ToolName   string    `json:"tool_name"`
}

type MCPToolInvocationResult struct {
	ServerID   string         `json:"server_id"`
	ServerName string         `json:"server_name"`
	ServerKind string         `json:"server_kind"`
	ToolName   string         `json:"tool_name"`
	ExecutedAt time.Time      `json:"executed_at"`
	Payload    map[string]any `json:"payload"`
}

type toolSpec struct {
	Name             string
	DisplayName      string
	Description      string
	IncidentRequired bool
}

type MCPService struct {
	repo                domainsresmartbot.Repository
	systemConfigService *systemconfig.Service
	processHealth       runtimehealth.Provider
	appSignalRepo       domainappsignals.Repository
	releaseCompliance   *releasecompliance.Metrics
	logQueryClient      *logdetector.LokiClient
	db                  *sqlx.DB
}

func NewMCPService(
	repo domainsresmartbot.Repository,
	systemConfigService *systemconfig.Service,
	processHealth runtimehealth.Provider,
	appSignalRepo domainappsignals.Repository,
	releaseCompliance *releasecompliance.Metrics,
	logQueryClient *logdetector.LokiClient,
	db *sqlx.DB,
) *MCPService {
	return &MCPService{
		repo:                repo,
		systemConfigService: systemConfigService,
		processHealth:       processHealth,
		appSignalRepo:       appSignalRepo,
		releaseCompliance:   releaseCompliance,
		logQueryClient:      logQueryClient,
		db:                  db,
	}
}

func (s *MCPService) ListAvailableTools(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) ([]MCPToolDescriptor, error) {
	if s == nil || s.systemConfigService == nil {
		return nil, fmt.Errorf("mcp service is not configured")
	}
	if incidentID != uuid.Nil && s.repo != nil {
		if _, err := s.repo.GetIncident(ctx, incidentID); err != nil {
			return nil, err
		}
	}

	policy, err := s.systemConfigService.GetRobotSREPolicyConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	descriptors := make([]MCPToolDescriptor, 0)
	for _, server := range policy.MCPServers {
		if !server.Enabled {
			continue
		}
		for _, spec := range supportedToolsForServer(server) {
			descriptors = append(descriptors, MCPToolDescriptor{
				ServerID:         server.ID,
				ServerName:       server.Name,
				ServerKind:       server.Kind,
				ToolName:         spec.Name,
				DisplayName:      spec.DisplayName,
				Description:      spec.Description,
				ReadOnly:         true,
				IncidentRequired: spec.IncidentRequired,
			})
		}
	}
	return descriptors, nil
}

func (s *MCPService) InvokeTool(ctx context.Context, tenantID *uuid.UUID, req MCPToolInvocationRequest) (*MCPToolInvocationResult, error) {
	if s == nil || s.systemConfigService == nil {
		return nil, fmt.Errorf("mcp service is not configured")
	}
	if req.ServerID == "" || req.ToolName == "" {
		return nil, fmt.Errorf("server_id and tool_name are required")
	}

	policy, err := s.systemConfigService.GetRobotSREPolicyConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	server, spec, err := resolveTool(policy.MCPServers, req.ServerID, req.ToolName)
	if err != nil {
		return nil, err
	}
	if spec.IncidentRequired {
		if req.IncidentID == uuid.Nil {
			return nil, fmt.Errorf("incident_id is required for tool %s", req.ToolName)
		}
	}

	var payload map[string]any
	switch req.ToolName {
	case "incidents.list":
		payload, err = s.invokeIncidentsList(ctx, tenantID)
	case "incidents.get":
		payload, err = s.invokeIncidentGet(ctx, req.IncidentID)
	case "findings.list":
		payload, err = s.invokeFindingsList(ctx, req.IncidentID)
	case "evidence.list":
		payload, err = s.invokeEvidenceList(ctx, req.IncidentID)
	case "runtime_health.get":
		payload, err = s.invokeRuntimeHealth()
	case "http_signals.recent":
		payload, err = s.invokeHTTPRecentSignals()
	case "http_signals.history":
		payload, err = s.invokeHTTPSignalHistory(ctx)
	case "async_backlog.recent":
		payload, err = s.invokeAsyncBacklogSignals()
	case "messaging_transport.recent":
		payload, err = s.invokeMessagingTransport()
	case "logs.recent":
		payload, err = s.invokeRecentLogs(ctx, req.IncidentID)
	case "cluster_overview.get":
		payload, err = s.invokeClusterOverview(ctx, tenantID)
	case "nodes.list":
		payload, err = s.invokeNodesList(ctx, tenantID)
	case "release_drift.summary":
		payload, err = s.invokeReleaseDriftSummary(ctx, tenantID)
	default:
		err = fmt.Errorf("unsupported tool: %s", req.ToolName)
	}
	if err != nil {
		return nil, err
	}

	return &MCPToolInvocationResult{
		ServerID:   server.ID,
		ServerName: server.Name,
		ServerKind: server.Kind,
		ToolName:   spec.Name,
		ExecutedAt: time.Now().UTC(),
		Payload:    payload,
	}, nil
}

func supportedToolsForServer(server systemconfig.RobotSREMCPServer) []toolSpec {
	supported := supportedToolCatalog()[server.Kind]
	if len(supported) == 0 {
		return nil
	}
	if len(server.AllowedTools) == 0 {
		return supported
	}
	allowed := make(map[string]struct{}, len(server.AllowedTools))
	for _, tool := range server.AllowedTools {
		allowed[strings.TrimSpace(tool)] = struct{}{}
	}
	filtered := make([]toolSpec, 0, len(supported))
	for _, spec := range supported {
		if _, ok := allowed[spec.Name]; ok {
			filtered = append(filtered, spec)
		}
	}
	return filtered
}

func supportedToolCatalog() map[string][]toolSpec {
	return map[string][]toolSpec{
		"observability": {
			{Name: "incidents.list", DisplayName: "Incident Index", Description: "List recent incident threads in the current scope.", IncidentRequired: false},
			{Name: "incidents.get", DisplayName: "Incident Record", Description: "Return the normalized incident record for the selected thread.", IncidentRequired: true},
			{Name: "findings.list", DisplayName: "Findings", Description: "Return recent findings attached to the selected incident.", IncidentRequired: true},
			{Name: "evidence.list", DisplayName: "Evidence", Description: "Return stored evidence snapshots for the selected incident.", IncidentRequired: true},
			{Name: "runtime_health.get", DisplayName: "Runtime Health", Description: "Return current health of embedded control-plane workers.", IncidentRequired: false},
			{Name: "http_signals.recent", DisplayName: "HTTP Signals", Description: "Return the latest app-level HTTP traffic, error-rate, and latency window captured by SRE Smart Bot.", IncidentRequired: false},
			{Name: "http_signals.history", DisplayName: "HTTP Signal History", Description: "Return recent app-level HTTP signal windows for trend-aware investigation.", IncidentRequired: false},
			{Name: "async_backlog.recent", DisplayName: "Async Backlog", Description: "Return the latest build queue, email queue, and outbox backlog pressure captured by SRE Smart Bot.", IncidentRequired: false},
			{Name: "messaging_transport.recent", DisplayName: "Messaging Transport", Description: "Return current NATS transport health, reconnect activity, and recent disconnect state.", IncidentRequired: false},
			{Name: "logs.recent", DisplayName: "Recent Logs", Description: "Run a bounded Loki query using incident-derived search terms and return recent matching log lines.", IncidentRequired: true},
		},
		"kubernetes": {
			{Name: "cluster_overview.get", DisplayName: "Cluster Overview", Description: "Return read-only cluster health and capacity context.", IncidentRequired: false},
			{Name: "nodes.list", DisplayName: "Nodes", Description: "Return recent node health and capacity details.", IncidentRequired: false},
		},
		"release": {
			{Name: "release_drift.summary", DisplayName: "Release Drift Summary", Description: "Return release compliance watcher metrics and recent release-drift incidents.", IncidentRequired: false},
		},
	}
}

func resolveTool(servers []systemconfig.RobotSREMCPServer, serverID string, toolName string) (systemconfig.RobotSREMCPServer, toolSpec, error) {
	for _, server := range servers {
		if server.ID != serverID {
			continue
		}
		if !server.Enabled {
			return systemconfig.RobotSREMCPServer{}, toolSpec{}, fmt.Errorf("mcp server %s is disabled", serverID)
		}
		for _, spec := range supportedToolsForServer(server) {
			if spec.Name == toolName {
				return server, spec, nil
			}
		}
		return systemconfig.RobotSREMCPServer{}, toolSpec{}, fmt.Errorf("tool %s is not allowed for server %s", toolName, serverID)
	}
	return systemconfig.RobotSREMCPServer{}, toolSpec{}, fmt.Errorf("mcp server %s not found", serverID)
}

func (s *MCPService) invokeIncidentsList(ctx context.Context, tenantID *uuid.UUID) (map[string]any, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("incident repository is not configured")
	}
	incidents, err := s.repo.ListIncidents(ctx, domainsresmartbot.IncidentFilter{
		TenantID: tenantID,
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"count":     len(incidents),
		"incidents": incidents,
	}, nil
}

func (s *MCPService) invokeIncidentGet(ctx context.Context, incidentID uuid.UUID) (map[string]any, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("incident repository is not configured")
	}
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"incident": incident}, nil
}

func (s *MCPService) invokeFindingsList(ctx context.Context, incidentID uuid.UUID) (map[string]any, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("incident repository is not configured")
	}
	findings, err := s.repo.ListFindingsByIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"count":    len(findings),
		"findings": findings,
	}, nil
}

func (s *MCPService) invokeEvidenceList(ctx context.Context, incidentID uuid.UUID) (map[string]any, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("incident repository is not configured")
	}
	evidence, err := s.repo.ListEvidenceByIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"count":    len(evidence),
		"evidence": evidence,
	}, nil
}

func (s *MCPService) invokeRuntimeHealth() (map[string]any, error) {
	names := []string{
		"dispatcher",
		"workflow_orchestrator",
		"provider_readiness_watcher",
		"tenant_asset_drift_watcher",
		"quarantine_release_compliance_watcher",
		"runtime_dependency_watcher",
		"cluster_metrics_snapshot_ingester",
		"http_golden_signal_runner",
		"async_backlog_signal_runner",
		"nats_transport_signal_runner",
		"stale_execution_watchdog",
	}
	processes := make([]map[string]any, 0, len(names))
	for _, name := range names {
		status, ok := runtimehealth.ProcessStatus{}, false
		if s.processHealth != nil {
			status, ok = s.processHealth.GetStatus(name)
		}
		processes = append(processes, map[string]any{
			"name":          name,
			"configured":    ok,
			"enabled":       status.Enabled,
			"running":       status.Running,
			"last_activity": status.LastActivity,
			"message":       status.Message,
			"metrics":       status.Metrics,
		})
	}
	return map[string]any{
		"count":     len(processes),
		"processes": processes,
	}, nil
}

func (s *MCPService) invokeHTTPRecentSignals() (map[string]any, error) {
	if s.processHealth == nil {
		return nil, fmt.Errorf("http_signals.recent is unavailable because runtime health is not configured")
	}
	status, ok := s.processHealth.GetStatus("http_golden_signal_runner")
	if !ok {
		return nil, fmt.Errorf("http_signals.recent is unavailable because the HTTP golden signal runner has not reported yet")
	}

	requestCount := metricInt64(status.Metrics, "http_signal_last_request_count")
	serverErrorCount := metricInt64(status.Metrics, "http_signal_last_server_error_count")
	clientErrorCount := metricInt64(status.Metrics, "http_signal_last_client_error_count")
	avgLatencyMs := metricInt64(status.Metrics, "http_signal_last_avg_latency_ms")
	maxLatencyMs := metricInt64(status.Metrics, "http_signal_last_max_latency_ms")
	errorRatePercent := int64(0)
	if requestCount > 0 {
		errorRatePercent = (serverErrorCount * 100) / requestCount
	}

	return map[string]any{
		"runner_enabled":     status.Enabled,
		"runner_running":     status.Running,
		"message":            status.Message,
		"last_activity":      status.LastActivity,
		"request_count":      requestCount,
		"server_error_count": serverErrorCount,
		"client_error_count": clientErrorCount,
		"error_rate_percent": errorRatePercent,
		"average_latency_ms": avgLatencyMs,
		"max_latency_ms":     maxLatencyMs,
		"raw_metrics":        status.Metrics,
	}, nil
}

func (s *MCPService) invokeAsyncBacklogSignals() (map[string]any, error) {
	if s.processHealth == nil {
		return nil, fmt.Errorf("async_backlog.recent is unavailable because runtime health is not configured")
	}
	status, ok := s.processHealth.GetStatus("async_backlog_signal_runner")
	if !ok {
		return nil, fmt.Errorf("async_backlog.recent is unavailable because the async backlog signal runner has not reported yet")
	}

	buildQueueDepth := metricInt64(status.Metrics, "async_backlog_build_queue_depth")
	emailQueueDepth := metricInt64(status.Metrics, "async_backlog_email_queue_depth")
	outboxPendingCount := metricInt64(status.Metrics, "async_backlog_outbox_pending_count")
	buildQueueThreshold := metricInt64(status.Metrics, "build_queue_threshold")
	emailQueueThreshold := metricInt64(status.Metrics, "email_queue_threshold")
	outboxThreshold := metricInt64(status.Metrics, "messaging_outbox_threshold")

	return map[string]any{
		"runner_enabled":             status.Enabled,
		"runner_running":             status.Running,
		"message":                    status.Message,
		"last_activity":              status.LastActivity,
		"build_queue_depth":          buildQueueDepth,
		"email_queue_depth":          emailQueueDepth,
		"messaging_outbox_pending":   outboxPendingCount,
		"build_queue_threshold":      buildQueueThreshold,
		"email_queue_threshold":      emailQueueThreshold,
		"messaging_outbox_threshold": outboxThreshold,
		"build_queue_pressure":       buildQueueDepth >= buildQueueThreshold && buildQueueThreshold > 0,
		"email_queue_pressure":       emailQueueDepth >= emailQueueThreshold && emailQueueThreshold > 0,
		"messaging_outbox_pressure":  outboxPendingCount >= outboxThreshold && outboxThreshold > 0,
	}, nil
}

func (s *MCPService) invokeMessagingTransport() (map[string]any, error) {
	if s.processHealth == nil {
		return nil, fmt.Errorf("messaging_transport.recent is unavailable because runtime health is not configured")
	}
	status, ok := s.processHealth.GetStatus("nats_transport_signal_runner")
	if !ok {
		return nil, fmt.Errorf("messaging_transport.recent is unavailable because the NATS transport signal runner has not reported yet")
	}
	return map[string]any{
		"runner_enabled":      status.Enabled,
		"runner_running":      status.Running,
		"message":             status.Message,
		"last_activity":       status.LastActivity,
		"reconnect_threshold": metricInt64(status.Metrics, "nats_reconnect_threshold"),
		"reconnects":          metricInt64(status.Metrics, "nats_transport_reconnects"),
		"disconnects":         metricInt64(status.Metrics, "nats_transport_disconnects"),
	}, nil
}

func (s *MCPService) invokeHTTPSignalHistory(ctx context.Context) (map[string]any, error) {
	if s.appSignalRepo == nil {
		return nil, fmt.Errorf("http_signals.history is unavailable because app signal history is not configured")
	}
	windows, err := s.appSignalRepo.ListRecentHTTPWindows(ctx, 12)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"count":   len(windows),
		"windows": windows,
	}, nil
}

func metricInt64(metrics map[string]int64, key string) int64 {
	if len(metrics) == 0 {
		return 0
	}
	return metrics[key]
}

func (s *MCPService) invokeClusterOverview(ctx context.Context, tenantID *uuid.UUID) (map[string]any, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	query := `
		SELECT 
			COALESCE(SUM(total_nodes), 0) AS total_nodes,
			COALESCE(SUM(healthy_nodes), 0) AS healthy_nodes,
			COALESCE(SUM(offline_nodes), 0) AS offline_nodes,
			COALESCE(SUM(maintenance_nodes), 0) AS maintenance_nodes,
			COALESCE(SUM(total_cpu_capacity), 0) AS total_cpu_capacity,
			COALESCE(SUM(total_memory_capacity_gb), 0) AS total_memory_capacity_gb,
			COALESCE(SUM(total_disk_capacity_gb), 0) AS total_disk_capacity_gb,
			COALESCE(SUM(used_cpu_cores), 0) AS used_cpu_cores,
			COALESCE(SUM(used_memory_gb), 0) AS used_memory_gb,
			COALESCE(SUM(used_disk_gb), 0) AS used_disk_gb,
			COALESCE(AVG(avg_cpu_usage_percent), 0) AS avg_cpu_usage_percent,
			COALESCE(AVG(avg_memory_usage_percent), 0) AS avg_memory_usage_percent,
			COALESCE(AVG(avg_disk_usage_percent), 0) AS avg_disk_usage_percent
		FROM v_infrastructure_health
	`
	args := []any{}
	if tenantID != nil {
		query += ` WHERE tenant_id = $1`
		args = append(args, *tenantID)
	}
	row := s.db.QueryRowContext(ctx, query, args...)
	var totalNodes, healthyNodes, offlineNodes, maintenanceNodes int
	var totalCPU, totalMemory, totalDisk, usedCPU, usedMemory, usedDisk, avgCPU, avgMemory, avgDisk float64
	if err := row.Scan(
		&totalNodes,
		&healthyNodes,
		&offlineNodes,
		&maintenanceNodes,
		&totalCPU,
		&totalMemory,
		&totalDisk,
		&usedCPU,
		&usedMemory,
		&usedDisk,
		&avgCPU,
		&avgMemory,
		&avgDisk,
	); err != nil {
		return nil, err
	}
	return map[string]any{
		"total_nodes":                  totalNodes,
		"healthy_nodes":                healthyNodes,
		"offline_nodes":                offlineNodes,
		"maintenance_nodes":            maintenanceNodes,
		"total_cpu_capacity":           totalCPU,
		"total_memory_capacity_gb":     totalMemory,
		"total_disk_capacity_gb":       totalDisk,
		"used_cpu_cores":               usedCPU,
		"used_memory_gb":               usedMemory,
		"used_disk_gb":                 usedDisk,
		"average_cpu_usage_percent":    avgCPU,
		"average_memory_usage_percent": avgMemory,
		"average_disk_usage_percent":   avgDisk,
	}, nil
}

func (s *MCPService) invokeNodesList(ctx context.Context, tenantID *uuid.UUID) (map[string]any, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	query := `
		SELECT 
			id,
			name,
			status,
			total_cpu_capacity,
			total_memory_capacity_gb,
			total_disk_capacity_gb,
			last_heartbeat,
			maintenance_mode,
			labels,
			created_at,
			updated_at
		FROM v_infrastructure_nodes
		WHERE 1=1
	`
	args := []any{}
	if tenantID != nil {
		query += ` AND tenant_id = $1`
		args = append(args, *tenantID)
	}
	query += ` ORDER BY created_at DESC LIMIT 10`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]map[string]any, 0)
	for rows.Next() {
		var (
			id, name, status                    string
			totalCPU, totalMemory, totalDisk    float64
			lastHeartbeat, createdAt, updatedAt sql.NullTime
			maintenanceMode                     bool
			labelsRaw                           sql.NullString
		)
		if err := rows.Scan(
			&id,
			&name,
			&status,
			&totalCPU,
			&totalMemory,
			&totalDisk,
			&lastHeartbeat,
			&maintenanceMode,
			&labelsRaw,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		labels := map[string]string{}
		if labelsRaw.Valid && strings.TrimSpace(labelsRaw.String) != "" {
			_ = json.Unmarshal([]byte(labelsRaw.String), &labels)
		}
		nodes = append(nodes, map[string]any{
			"id":               id,
			"name":             name,
			"status":           status,
			"total_cpu_cores":  totalCPU,
			"total_memory_gb":  totalMemory,
			"total_disk_gb":    totalDisk,
			"last_heartbeat":   nullableTime(lastHeartbeat),
			"maintenance_mode": maintenanceMode,
			"labels":           labels,
			"created_at":       nullableTime(createdAt),
			"updated_at":       nullableTime(updatedAt),
		})
	}
	return map[string]any{
		"count": len(nodes),
		"nodes": nodes,
	}, nil
}

func nullableTime(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func (s *MCPService) invokeRecentLogs(ctx context.Context, incidentID uuid.UUID) (map[string]any, error) {
	if s.logQueryClient == nil {
		return nil, fmt.Errorf("logs.recent is unavailable because Loki is not configured")
	}
	if s.repo == nil {
		return nil, fmt.Errorf("incident repository is not configured")
	}
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	findings, _ := s.repo.ListFindingsByIncident(ctx, incidentID)

	queries := buildLokiQueriesForIncident(incident, findings)
	if len(queries) == 0 {
		return nil, fmt.Errorf("logs.recent could not derive a Loki query for this incident")
	}
	start := time.Now().UTC().Add(-15 * time.Minute)
	end := time.Now().UTC()
	queryErrors := make([]string, 0, len(queries))
	for _, query := range queries {
		result, queryErr := s.logQueryClient.QueryRange(ctx, query, start, end, 20)
		if queryErr != nil {
			queryErrors = append(queryErrors, fmt.Sprintf("%s: %v", query, queryErr))
			continue
		}
		matches := make([]map[string]any, 0, len(result.Matches))
		for _, match := range result.Matches {
			matches = append(matches, map[string]any{
				"timestamp": match.Timestamp,
				"line":      match.Line,
				"labels":    match.Labels,
			})
		}
		return map[string]any{
			"query":       query,
			"lookback":    "15m",
			"match_count": len(matches),
			"matches":     matches,
		}, nil
	}
	if len(queryErrors) > 0 {
		return nil, fmt.Errorf("logs.recent could not query Loki successfully: %s", strings.Join(queryErrors, "; "))
	}
	return nil, fmt.Errorf("logs.recent returned no usable Loki response")
}

func (s *MCPService) invokeReleaseDriftSummary(ctx context.Context, tenantID *uuid.UUID) (map[string]any, error) {
	snapshot := releasecompliance.Snapshot{}
	if s.releaseCompliance != nil {
		snapshot = s.releaseCompliance.Snapshot()
	}

	recentIncidents := []*domainsresmartbot.Incident{}
	if s.repo != nil {
		incidents, err := s.repo.ListIncidents(ctx, domainsresmartbot.IncidentFilter{
			TenantID: tenantID,
			Domain:   "release_configuration",
			Limit:    10,
			Offset:   0,
		})
		if err == nil {
			recentIncidents = incidents
		}
	}

	return map[string]any{
		"watch_ticks_total":        snapshot.WatchTicksTotal,
		"watch_failures_total":     snapshot.WatchFailuresTotal,
		"drift_detected_total":     snapshot.DriftDetectedTotal,
		"drift_recovered_total":    snapshot.DriftRecoveredTotal,
		"active_drift_count":       snapshot.ActiveDriftCount,
		"released_count":           snapshot.ReleasedCount,
		"last_tick_unix":           snapshot.LastTickUnix,
		"recent_release_incidents": recentIncidents,
	}, nil
}

func buildLokiQueriesForIncident(incident *domainsresmartbot.Incident, findings []*domainsresmartbot.Finding) []string {
	baseSelector := `{job=~".+"}`
	candidates := []string{}
	addCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		value = strings.ReplaceAll(value, `"`, "")
		if len(value) > 64 {
			value = value[:64]
		}
		candidates = append(candidates, value)
	}

	if incident != nil {
		addCandidate(incident.Source)
		addCandidate(incident.IncidentType)
		addCandidate(incident.DisplayName)
	}
	for _, finding := range findings {
		if finding == nil {
			continue
		}
		addCandidate(finding.SignalKey)
		addCandidate(finding.Title)
		if len(candidates) >= 5 {
			break
		}
	}

	seen := map[string]struct{}{}
	queries := make([]string, 0, len(candidates)+1)
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		queries = append(queries, fmt.Sprintf(`%s |= %q`, baseSelector, candidate))
	}
	if len(queries) == 0 {
		queries = append(queries, baseSelector)
	}
	return queries
}
