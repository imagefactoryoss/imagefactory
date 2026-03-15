package sresmartbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	domaininfrastructure "github.com/srikarm/image-factory/internal/domain/infrastructure"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
	"go.uber.org/zap"
)

type RuntimeDependencyIssue struct {
	Key      string
	Severity string
	Message  string
}

type GoldenSignalThresholds struct {
	NodeCPUSaturationPercent    int
	NodeMemorySaturationPercent int
	PodRestartThreshold         int32
}

type GoldenSignalIssue struct {
	Key       string
	Kind      string
	NodeName  string
	Summary   string
	Severity  domainsresmartbot.IncidentSeverity
	Percent   int
	Threshold int
	Payload   map[string]interface{}
}

type ClusterNodeSignalSnapshot struct {
	ClusterName              string
	NodeName                 string
	CPUUsageMillicores       int64
	MemoryUsageBytes         int64
	CPUAllocatableMillicores int64
	MemoryAllocatableBytes   int64
}

type ClusterPodSignalSnapshot struct {
	ClusterName    string
	Namespace      string
	PodName        string
	NodeName       string
	RestartCount   int32
	Phase          string
	Reason         string
	ContainerCount int
}

type HTTPRequestSignalSnapshot struct {
	WindowStartedAt  time.Time
	WindowEndedAt    time.Time
	RequestCount     int64
	ServerErrorCount int64
	ClientErrorCount int64
	TotalLatencyMs   int64
	MaxLatencyMs     int64
}

type HTTPGoldenSignalThresholds struct {
	MinRequestCount         int64
	ErrorRatePercent        int
	AverageLatencyThreshold int64
	RequestVolumeThreshold  int64
}

type HTTPGoldenSignalIssue struct {
	Key     string
	Summary string
	Payload map[string]interface{}
}

type AsyncBacklogThresholds struct {
	BuildQueueThreshold      int64
	EmailQueueThreshold      int64
	MessagingOutboxThreshold int64
}

type AsyncBacklogSignalSnapshot struct {
	BuildQueueDepth        int64
	EmailQueueDepth        int64
	MessagingOutboxPending int64
}

type AsyncBacklogIssue struct {
	Key       string
	Kind      string
	Summary   string
	Severity  domainsresmartbot.IncidentSeverity
	Count     int64
	Threshold int64
	Payload   map[string]interface{}
}

func ObserveRuntimeDependencyIssues(ctx context.Context, svc *Service, logger *zap.Logger, issues []RuntimeDependencyIssue, now time.Time, previous map[string]RuntimeDependencyIssue) map[string]RuntimeDependencyIssue {
	current := make(map[string]RuntimeDependencyIssue, len(issues))
	for _, issue := range issues {
		current[issue.Key] = issue
		if svc == nil {
			continue
		}
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: "runtime_dependency:" + strings.TrimSpace(issue.Key),
			Domain:         "runtime_services",
			IncidentType:   "runtime_dependency_outage",
			DisplayName:    "Runtime dependency degraded: " + strings.TrimSpace(issue.Key),
			Summary:        issue.Message,
			Source:         "runtime_dependency_watcher",
			Severity:       runtimeIssueSeverity(issue),
			Confidence:     runtimeIssueConfidence(issues),
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"component": issue.Key,
				"severity":  strings.ToLower(strings.TrimSpace(issue.Severity)),
			},
			FindingTitle:   "Runtime dependency issue detected",
			FindingMessage: issue.Message,
			SignalType:     "runtime_dependency_issue",
			SignalKey:      strings.TrimSpace(issue.Key),
			RawPayload: map[string]interface{}{
				"component": issue.Key,
				"severity":  issue.Severity,
				"message":   issue.Message,
			},
		}); err != nil && logger != nil {
			logger.Warn("Failed to record runtime dependency incident observation",
				zap.String("issue_key", issue.Key),
				zap.Error(err),
			)
		}
	}

	for issueKey := range previous {
		if _, stillPresent := current[issueKey]; stillPresent {
			continue
		}
		if svc == nil {
			continue
		}
		if err := svc.ResolveIncident(ctx,
			"runtime_dependency:"+strings.TrimSpace(issueKey),
			now,
			"Runtime dependency recovered",
			map[string]interface{}{
				"component": issueKey,
				"source":    "runtime_dependency_watcher",
			},
		); err != nil && logger != nil {
			logger.Warn("Failed to resolve runtime dependency incident",
				zap.String("issue_key", issueKey),
				zap.Error(err),
			)
		}
	}

	return current
}

func ObserveClusterMetricsIngesterFailure(ctx context.Context, svc *Service, logger *zap.Logger, err error, now time.Time) {
	if svc == nil || err == nil {
		return
	}
	if incidentErr := svc.RecordObservation(ctx, SignalObservation{
		CorrelationKey: "runtime_service:cluster_metrics_snapshot_ingester",
		Domain:         "runtime_services",
		IncidentType:   "metrics_snapshot_ingester_degraded",
		DisplayName:    "Cluster metrics snapshot ingester degraded",
		Summary:        err.Error(),
		Source:         "cluster_metrics_snapshot_ingester",
		Severity:       domainsresmartbot.IncidentSeverityWarning,
		Confidence:     domainsresmartbot.IncidentConfidenceHigh,
		OccurredAt:     now,
		Metadata: map[string]interface{}{
			"component": "cluster_metrics_snapshot_ingester",
		},
		FindingTitle:   "Cluster metrics snapshot collection failed",
		FindingMessage: err.Error(),
		SignalType:     "metrics_collection_failure",
		SignalKey:      "cluster_metrics_snapshot_ingester",
		RawPayload: map[string]interface{}{
			"error": err.Error(),
		},
	}); incidentErr != nil && logger != nil {
		logger.Warn("Failed to record cluster metrics snapshot ingester incident", zap.Error(incidentErr))
	}
}

func ResolveClusterMetricsIngester(ctx context.Context, svc *Service, logger *zap.Logger, now time.Time, nodesCollected int, podsCollected int) {
	if svc == nil {
		return
	}
	if err := svc.ResolveIncident(ctx,
		"runtime_service:cluster_metrics_snapshot_ingester",
		now,
		fmt.Sprintf("Cluster metrics snapshot ingester healthy; collected %d node snapshots and %d pod snapshots", nodesCollected, podsCollected),
		map[string]interface{}{
			"component":       "cluster_metrics_snapshot_ingester",
			"nodes_collected": nodesCollected,
			"pods_collected":  podsCollected,
		},
	); err != nil && logger != nil {
		logger.Warn("Failed to resolve cluster metrics snapshot ingester incident", zap.Error(err))
	}
}

func ObserveClusterGoldenSignals(
	ctx context.Context,
	svc *Service,
	logger *zap.Logger,
	nodes []ClusterNodeSignalSnapshot,
	pods []ClusterPodSignalSnapshot,
	now time.Time,
	previous map[string]GoldenSignalIssue,
	thresholds GoldenSignalThresholds,
) map[string]GoldenSignalIssue {
	if thresholds.NodeCPUSaturationPercent < 1 {
		thresholds.NodeCPUSaturationPercent = 85
	}
	if thresholds.NodeMemorySaturationPercent < 1 {
		thresholds.NodeMemorySaturationPercent = 85
	}
	if thresholds.PodRestartThreshold < 1 {
		thresholds.PodRestartThreshold = 3
	}

	current := make(map[string]GoldenSignalIssue)
	for _, node := range nodes {
		if issue, ok := evaluateNodeCPUSaturation(node, thresholds.NodeCPUSaturationPercent); ok {
			current[issue.Key] = issue
			recordGoldenSignalIssue(ctx, svc, logger, issue, now)
		}
		if issue, ok := evaluateNodeMemorySaturation(node, thresholds.NodeMemorySaturationPercent); ok {
			current[issue.Key] = issue
			recordGoldenSignalIssue(ctx, svc, logger, issue, now)
		}
	}
	for _, pod := range pods {
		if issue, ok := evaluatePodRestartPressure(pod, thresholds.PodRestartThreshold); ok {
			current[issue.Key] = issue
			recordGoldenSignalIssue(ctx, svc, logger, issue, now)
		}
		if issue, ok := evaluatePodEvictionPressure(pod); ok {
			current[issue.Key] = issue
			recordGoldenSignalIssue(ctx, svc, logger, issue, now)
		}
	}

	for issueKey, issue := range previous {
		if _, stillPresent := current[issueKey]; stillPresent {
			continue
		}
		if svc == nil {
			continue
		}
		if err := svc.ResolveIncident(ctx, goldenSignalCorrelationKey(issue), now,
			fmt.Sprintf("Golden signal recovered for node %s", issue.NodeName),
			map[string]interface{}{
				"kind":      issue.Kind,
				"node_name": issue.NodeName,
				"source":    "cluster_metrics_snapshot_ingester",
			},
		); err != nil && logger != nil {
			logger.Warn("Failed to resolve golden signal incident",
				zap.String("issue_key", issue.Key),
				zap.Error(err),
			)
		}
	}

	return current
}

func evaluateNodeCPUSaturation(node ClusterNodeSignalSnapshot, thresholdPercent int) (GoldenSignalIssue, bool) {
	if node.CPUAllocatableMillicores <= 0 {
		return GoldenSignalIssue{}, false
	}
	percent := int((node.CPUUsageMillicores * 100) / node.CPUAllocatableMillicores)
	if percent < thresholdPercent {
		return GoldenSignalIssue{}, false
	}
	return GoldenSignalIssue{
		Key:       fmt.Sprintf("node_cpu_saturation:%s", node.NodeName),
		Kind:      "node_cpu_saturation",
		NodeName:  node.NodeName,
		Summary:   fmt.Sprintf("Node %s CPU usage is at %d%% of allocatable capacity", node.NodeName, percent),
		Severity:  severityForPercent(percent),
		Percent:   percent,
		Threshold: thresholdPercent,
		Payload: map[string]interface{}{
			"node_name":                  node.NodeName,
			"cpu_usage_millicores":       node.CPUUsageMillicores,
			"cpu_allocatable_millicores": node.CPUAllocatableMillicores,
			"percent":                    percent,
			"threshold_percent":          thresholdPercent,
			"cluster_name":               node.ClusterName,
		},
	}, true
}

func evaluateNodeMemorySaturation(node ClusterNodeSignalSnapshot, thresholdPercent int) (GoldenSignalIssue, bool) {
	if node.MemoryAllocatableBytes <= 0 {
		return GoldenSignalIssue{}, false
	}
	percent := int((node.MemoryUsageBytes * 100) / node.MemoryAllocatableBytes)
	if percent < thresholdPercent {
		return GoldenSignalIssue{}, false
	}
	return GoldenSignalIssue{
		Key:       fmt.Sprintf("node_memory_saturation:%s", node.NodeName),
		Kind:      "node_memory_saturation",
		NodeName:  node.NodeName,
		Summary:   fmt.Sprintf("Node %s memory usage is at %d%% of allocatable capacity", node.NodeName, percent),
		Severity:  severityForPercent(percent),
		Percent:   percent,
		Threshold: thresholdPercent,
		Payload: map[string]interface{}{
			"node_name":                node.NodeName,
			"memory_usage_bytes":       node.MemoryUsageBytes,
			"memory_allocatable_bytes": node.MemoryAllocatableBytes,
			"percent":                  percent,
			"threshold_percent":        thresholdPercent,
			"cluster_name":             node.ClusterName,
		},
	}, true
}

func evaluatePodRestartPressure(pod ClusterPodSignalSnapshot, threshold int32) (GoldenSignalIssue, bool) {
	if pod.RestartCount < threshold {
		return GoldenSignalIssue{}, false
	}
	podRef := pod.Namespace + "/" + pod.PodName
	return GoldenSignalIssue{
		Key:       fmt.Sprintf("pod_restart_pressure:%s", podRef),
		Kind:      "pod_restart_pressure",
		NodeName:  pod.NodeName,
		Summary:   fmt.Sprintf("Pod %s has restarted %d times", podRef, pod.RestartCount),
		Severity:  severityForRestartCount(pod.RestartCount),
		Percent:   int(pod.RestartCount),
		Threshold: int(threshold),
		Payload: map[string]interface{}{
			"namespace":       pod.Namespace,
			"pod_name":        pod.PodName,
			"node_name":       pod.NodeName,
			"restart_count":   pod.RestartCount,
			"threshold":       threshold,
			"phase":           pod.Phase,
			"reason":          pod.Reason,
			"container_count": pod.ContainerCount,
			"cluster_name":    pod.ClusterName,
		},
	}, true
}

func evaluatePodEvictionPressure(pod ClusterPodSignalSnapshot) (GoldenSignalIssue, bool) {
	if !strings.EqualFold(strings.TrimSpace(pod.Reason), "Evicted") {
		return GoldenSignalIssue{}, false
	}
	podRef := pod.Namespace + "/" + pod.PodName
	return GoldenSignalIssue{
		Key:       fmt.Sprintf("pod_eviction_pressure:%s", podRef),
		Kind:      "pod_eviction_pressure",
		NodeName:  pod.NodeName,
		Summary:   fmt.Sprintf("Pod %s was evicted from node %s", podRef, pod.NodeName),
		Severity:  domainsresmartbot.IncidentSeverityWarning,
		Threshold: 1,
		Payload: map[string]interface{}{
			"namespace":       pod.Namespace,
			"pod_name":        pod.PodName,
			"node_name":       pod.NodeName,
			"phase":           pod.Phase,
			"reason":          pod.Reason,
			"container_count": pod.ContainerCount,
			"cluster_name":    pod.ClusterName,
		},
	}, true
}

func recordGoldenSignalIssue(ctx context.Context, svc *Service, logger *zap.Logger, issue GoldenSignalIssue, now time.Time) {
	if svc == nil {
		return
	}
	if err := svc.RecordObservation(ctx, SignalObservation{
		CorrelationKey: goldenSignalCorrelationKey(issue),
		Domain:         "golden_signals",
		IncidentType:   goldenSignalIncidentType(issue),
		DisplayName:    fmt.Sprintf("Golden signal saturation risk: %s", strings.ReplaceAll(issue.Kind, "_", " ")),
		Summary:        issue.Summary,
		Source:         "cluster_metrics_snapshot_ingester",
		Severity:       issue.Severity,
		Confidence:     domainsresmartbot.IncidentConfidenceHigh,
		OccurredAt:     now,
		Metadata: map[string]interface{}{
			"kind":              issue.Kind,
			"node_name":         issue.NodeName,
			"percent":           issue.Percent,
			"threshold_percent": issue.Threshold,
		},
		FindingTitle:   "Cluster saturation signal observed",
		FindingMessage: issue.Summary,
		SignalType:     issue.Kind,
		SignalKey:      issue.NodeName,
		RawPayload:     issue.Payload,
	}); err != nil && logger != nil {
		logger.Warn("Failed to record golden signal incident observation",
			zap.String("issue_key", issue.Key),
			zap.Error(err),
		)
	}
	_ = svc.AddEvidence(ctx, goldenSignalCorrelationKey(issue), issue.Kind, issue.Summary, issue.Payload, now)
	_ = svc.EnsureActionAttempt(ctx, goldenSignalCorrelationKey(issue), ActionAttemptSpec{
		ActionKey:     goldenSignalActionKey(issue),
		ActionClass:   "recommendation",
		TargetKind:    goldenSignalTargetKind(issue),
		TargetRef:     goldenSignalTargetRef(issue),
		Status:        "proposed",
		ActorType:     "system",
		ResultPayload: map[string]interface{}{"kind": issue.Kind, "percent": issue.Percent},
	}, now)
}

func goldenSignalCorrelationKey(issue GoldenSignalIssue) string {
	return "golden_signal:" + strings.TrimSpace(issue.Key)
}

func severityForPercent(percent int) domainsresmartbot.IncidentSeverity {
	if percent >= 95 {
		return domainsresmartbot.IncidentSeverityCritical
	}
	return domainsresmartbot.IncidentSeverityWarning
}

func severityForRestartCount(restartCount int32) domainsresmartbot.IncidentSeverity {
	if restartCount >= 6 {
		return domainsresmartbot.IncidentSeverityCritical
	}
	return domainsresmartbot.IncidentSeverityWarning
}

func goldenSignalIncidentType(issue GoldenSignalIssue) string {
	if strings.Contains(issue.Kind, "saturation") {
		return "saturation_risk"
	}
	return "error_pressure"
}

func goldenSignalActionKey(issue GoldenSignalIssue) string {
	if strings.Contains(issue.Kind, "saturation") {
		return "review_cluster_capacity"
	}
	return "review_workload_stability"
}

func goldenSignalTargetKind(issue GoldenSignalIssue) string {
	if strings.Contains(issue.Kind, "node_") {
		return "node"
	}
	return "pod"
}

func goldenSignalTargetRef(issue GoldenSignalIssue) string {
	if ref, ok := issue.Payload["namespace"].(string); ok && ref != "" {
		if podName, ok := issue.Payload["pod_name"].(string); ok && podName != "" {
			return ref + "/" + podName
		}
	}
	return issue.NodeName
}

func ObserveHTTPGoldenSignals(
	ctx context.Context,
	svc *Service,
	logger *zap.Logger,
	snapshot HTTPRequestSignalSnapshot,
	now time.Time,
	previous map[string]HTTPGoldenSignalIssue,
	thresholds HTTPGoldenSignalThresholds,
) map[string]HTTPGoldenSignalIssue {
	if thresholds.MinRequestCount < 1 {
		thresholds.MinRequestCount = 20
	}
	if thresholds.ErrorRatePercent < 1 {
		thresholds.ErrorRatePercent = 10
	}
	if thresholds.AverageLatencyThreshold < 1 {
		thresholds.AverageLatencyThreshold = 800
	}
	if thresholds.RequestVolumeThreshold < 1 {
		thresholds.RequestVolumeThreshold = 250
	}

	current := make(map[string]HTTPGoldenSignalIssue)
	if issue, ok := evaluateHTTPErrorRate(snapshot, thresholds); ok {
		current[issue.Key] = issue
		recordHTTPGoldenSignalIssue(ctx, svc, logger, issue, now)
	}
	if issue, ok := evaluateHTTPLatency(snapshot, thresholds); ok {
		current[issue.Key] = issue
		recordHTTPGoldenSignalIssue(ctx, svc, logger, issue, now)
	}
	if issue, ok := evaluateHTTPRequestVolume(snapshot, thresholds); ok {
		current[issue.Key] = issue
		recordHTTPGoldenSignalIssue(ctx, svc, logger, issue, now)
	}

	for issueKey, issue := range previous {
		if _, stillPresent := current[issueKey]; stillPresent {
			continue
		}
		if svc == nil {
			continue
		}
		if err := svc.ResolveIncident(ctx, "http_signal:"+issueKey, now,
			fmt.Sprintf("HTTP golden signal recovered: %s", issue.Summary),
			issue.Payload,
		); err != nil && logger != nil {
			logger.Warn("Failed to resolve http golden signal incident",
				zap.String("issue_key", issueKey),
				zap.Error(err),
			)
		}
	}

	return current
}

func evaluateHTTPErrorRate(snapshot HTTPRequestSignalSnapshot, thresholds HTTPGoldenSignalThresholds) (HTTPGoldenSignalIssue, bool) {
	if snapshot.RequestCount < thresholds.MinRequestCount {
		return HTTPGoldenSignalIssue{}, false
	}
	errorRate := int((snapshot.ServerErrorCount * 100) / snapshot.RequestCount)
	if errorRate < thresholds.ErrorRatePercent {
		return HTTPGoldenSignalIssue{}, false
	}
	return HTTPGoldenSignalIssue{
		Key:     "error_rate_spike",
		Summary: fmt.Sprintf("API server error rate reached %d%% over %d requests", errorRate, snapshot.RequestCount),
		Payload: map[string]interface{}{
			"request_count":       snapshot.RequestCount,
			"server_error_count":  snapshot.ServerErrorCount,
			"client_error_count":  snapshot.ClientErrorCount,
			"error_rate_percent":  errorRate,
			"window_started_at":   snapshot.WindowStartedAt,
			"window_ended_at":     snapshot.WindowEndedAt,
			"threshold_percent":   thresholds.ErrorRatePercent,
			"window_duration_sec": int64(snapshot.WindowEndedAt.Sub(snapshot.WindowStartedAt).Seconds()),
		},
	}, true
}

func evaluateHTTPLatency(snapshot HTTPRequestSignalSnapshot, thresholds HTTPGoldenSignalThresholds) (HTTPGoldenSignalIssue, bool) {
	if snapshot.RequestCount < thresholds.MinRequestCount {
		return HTTPGoldenSignalIssue{}, false
	}
	avgLatencyMs := snapshot.TotalLatencyMs / snapshot.RequestCount
	if avgLatencyMs < thresholds.AverageLatencyThreshold {
		return HTTPGoldenSignalIssue{}, false
	}
	return HTTPGoldenSignalIssue{
		Key:     "latency_regression",
		Summary: fmt.Sprintf("API average latency reached %dms over %d requests", avgLatencyMs, snapshot.RequestCount),
		Payload: map[string]interface{}{
			"request_count":        snapshot.RequestCount,
			"average_latency_ms":   avgLatencyMs,
			"max_latency_ms":       snapshot.MaxLatencyMs,
			"window_started_at":    snapshot.WindowStartedAt,
			"window_ended_at":      snapshot.WindowEndedAt,
			"threshold_latency_ms": thresholds.AverageLatencyThreshold,
			"window_duration_sec":  int64(snapshot.WindowEndedAt.Sub(snapshot.WindowStartedAt).Seconds()),
		},
	}, true
}

func evaluateHTTPRequestVolume(snapshot HTTPRequestSignalSnapshot, thresholds HTTPGoldenSignalThresholds) (HTTPGoldenSignalIssue, bool) {
	if snapshot.RequestCount < thresholds.RequestVolumeThreshold {
		return HTTPGoldenSignalIssue{}, false
	}
	return HTTPGoldenSignalIssue{
		Key:     "traffic_anomaly",
		Summary: fmt.Sprintf("API traffic volume reached %d requests in the latest window", snapshot.RequestCount),
		Payload: map[string]interface{}{
			"request_count":       snapshot.RequestCount,
			"window_started_at":   snapshot.WindowStartedAt,
			"window_ended_at":     snapshot.WindowEndedAt,
			"threshold_requests":  thresholds.RequestVolumeThreshold,
			"window_duration_sec": int64(snapshot.WindowEndedAt.Sub(snapshot.WindowStartedAt).Seconds()),
		},
	}, true
}

func recordHTTPGoldenSignalIssue(ctx context.Context, svc *Service, logger *zap.Logger, issue HTTPGoldenSignalIssue, now time.Time) {
	if svc == nil {
		return
	}

	incidentType := "traffic_anomaly"
	severity := domainsresmartbot.IncidentSeverityInfo
	actionKey := "review_http_traffic"
	switch issue.Key {
	case "error_rate_spike":
		incidentType = "error_pressure"
		severity = domainsresmartbot.IncidentSeverityWarning
		actionKey = "review_application_health"
	case "latency_regression":
		incidentType = "latency_regression"
		severity = domainsresmartbot.IncidentSeverityWarning
		actionKey = "review_application_health"
	case "traffic_anomaly":
		incidentType = "traffic_anomaly"
		severity = domainsresmartbot.IncidentSeverityInfo
		actionKey = "review_http_traffic"
	}

	if err := svc.RecordObservation(ctx, SignalObservation{
		CorrelationKey: "http_signal:" + issue.Key,
		Domain:         "golden_signals",
		IncidentType:   incidentType,
		DisplayName:    "HTTP golden signal observed: " + strings.ReplaceAll(issue.Key, "_", " "),
		Summary:        issue.Summary,
		Source:         "http_golden_signal_runner",
		Severity:       severity,
		Confidence:     domainsresmartbot.IncidentConfidenceMedium,
		OccurredAt:     now,
		Metadata: map[string]interface{}{
			"signal_key": issue.Key,
		},
		FindingTitle:   "HTTP golden signal observed",
		FindingMessage: issue.Summary,
		SignalType:     "http_golden_signal",
		SignalKey:      issue.Key,
		RawPayload:     issue.Payload,
	}); err != nil && logger != nil {
		logger.Warn("Failed to record http golden signal observation", zap.String("issue_key", issue.Key), zap.Error(err))
	}
	_ = svc.AddEvidence(ctx, "http_signal:"+issue.Key, "http_golden_signal_window", issue.Summary, issue.Payload, now)
	_ = svc.EnsureActionAttempt(ctx, "http_signal:"+issue.Key, ActionAttemptSpec{
		ActionKey:     actionKey,
		ActionClass:   "recommendation",
		TargetKind:    "service",
		TargetRef:     "backend-api",
		Status:        "proposed",
		ActorType:     "system",
		ResultPayload: issue.Payload,
	}, now)
}

func runtimeIssueSeverity(issue RuntimeDependencyIssue) domainsresmartbot.IncidentSeverity {
	if strings.EqualFold(strings.TrimSpace(issue.Severity), "critical") {
		return domainsresmartbot.IncidentSeverityCritical
	}
	return domainsresmartbot.IncidentSeverityWarning
}

func runtimeIssueConfidence(issues []RuntimeDependencyIssue) domainsresmartbot.IncidentConfidence {
	if len(issues) >= 2 {
		return domainsresmartbot.IncidentConfidenceHigh
	}
	return domainsresmartbot.IncidentConfidenceMedium
}

func ObserveProviderReadinessTick(ctx context.Context, svc *Service, logger *zap.Logger, result *domaininfrastructure.ProviderReadinessWatchTickResult, tickErr error, now time.Time) {
	const correlationKey = "provider_readiness:global"
	if svc == nil {
		return
	}
	if tickErr != nil {
		message := fmt.Sprintf("Provider readiness watcher failed: %v", tickErr)
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: correlationKey,
			Domain:         "infrastructure",
			IncidentType:   "provider_readiness_watcher_failed",
			DisplayName:    "Provider readiness watcher degraded",
			Summary:        message,
			Source:         "provider_readiness_watcher",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"mode": "watch_tick_failed",
			},
			FindingTitle:   "Provider readiness watch tick failed",
			FindingMessage: message,
			SignalType:     "provider_readiness_watch_failure",
			SignalKey:      "global",
			RawPayload: map[string]interface{}{
				"error": tickErr.Error(),
			},
		}); err != nil && logger != nil {
			logger.Warn("Failed to record provider readiness watcher failure", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, correlationKey, "provider_readiness_watch_failure", message, map[string]interface{}{
			"error": tickErr.Error(),
		}, now)
		_ = svc.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:     "review_provider_connectivity",
			ActionClass:   "recommendation",
			TargetKind:    "provider",
			TargetRef:     "global",
			Status:        "proposed",
			ActorType:     "system",
			ResultPayload: map[string]interface{}{"reason": "watch_tick_failed"},
		}, now)
		return
	}
	if result == nil {
		return
	}
	if result.Failed > 0 || result.NotReady > 0 {
		summary := fmt.Sprintf("Provider readiness degraded: %d failed refreshes, %d providers not ready", result.Failed, result.NotReady)
		severity := domainsresmartbot.IncidentSeverityWarning
		confidence := domainsresmartbot.IncidentConfidenceMedium
		if result.Failed > 0 && result.NotReady > 0 {
			confidence = domainsresmartbot.IncidentConfidenceHigh
		}
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: correlationKey,
			Domain:         "infrastructure",
			IncidentType:   "provider_readiness_degraded",
			DisplayName:    "Provider readiness degraded",
			Summary:        summary,
			Source:         "provider_readiness_watcher",
			Severity:       severity,
			Confidence:     confidence,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"attempted": result.Attempted,
				"succeeded": result.Succeeded,
				"failed":    result.Failed,
				"skipped":   result.Skipped,
				"ready":     result.Ready,
				"not_ready": result.NotReady,
			},
			FindingTitle:   "Provider readiness degraded",
			FindingMessage: summary,
			SignalType:     "provider_readiness_summary",
			SignalKey:      "global",
			RawPayload: map[string]interface{}{
				"total_providers": result.TotalProviders,
				"attempted":       result.Attempted,
				"succeeded":       result.Succeeded,
				"failed":          result.Failed,
				"skipped":         result.Skipped,
				"ready":           result.Ready,
				"not_ready":       result.NotReady,
			},
		}); err != nil && logger != nil {
			logger.Warn("Failed to record provider readiness incident observation", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, correlationKey, "provider_readiness_tick", summary, map[string]interface{}{
			"total_providers": result.TotalProviders,
			"attempted":       result.Attempted,
			"succeeded":       result.Succeeded,
			"failed":          result.Failed,
			"skipped":         result.Skipped,
			"ready":           result.Ready,
			"not_ready":       result.NotReady,
		}, now)
		_ = svc.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:     "review_provider_connectivity",
			ActionClass:   "recommendation",
			TargetKind:    "provider",
			TargetRef:     "global",
			Status:        "proposed",
			ActorType:     "system",
			ResultPayload: map[string]interface{}{"not_ready": result.NotReady, "failed": result.Failed},
		}, now)
		return
	}
	if err := svc.ResolveIncident(ctx, correlationKey, now, "Provider readiness healthy", map[string]interface{}{
		"total_providers": result.TotalProviders,
		"ready":           result.Ready,
		"not_ready":       result.NotReady,
		"failed":          result.Failed,
		"source":          "provider_readiness_watcher",
	}); err != nil && logger != nil {
		logger.Warn("Failed to resolve provider readiness incident", zap.Error(err))
	}
}

func ObserveAsyncBacklogSignals(
	ctx context.Context,
	svc *Service,
	logger *zap.Logger,
	snapshot AsyncBacklogSignalSnapshot,
	now time.Time,
	previous map[string]AsyncBacklogIssue,
	thresholds AsyncBacklogThresholds,
) map[string]AsyncBacklogIssue {
	if thresholds.BuildQueueThreshold < 1 {
		thresholds.BuildQueueThreshold = 10
	}
	if thresholds.EmailQueueThreshold < 1 {
		thresholds.EmailQueueThreshold = 20
	}
	if thresholds.MessagingOutboxThreshold < 1 {
		thresholds.MessagingOutboxThreshold = 15
	}

	current := make(map[string]AsyncBacklogIssue)
	if issue, ok := evaluateBuildQueueBacklog(snapshot, thresholds); ok {
		current[issue.Key] = issue
		recordAsyncBacklogIssue(ctx, svc, logger, issue, now)
	}
	if issue, ok := evaluateEmailQueueBacklog(snapshot, thresholds); ok {
		current[issue.Key] = issue
		recordAsyncBacklogIssue(ctx, svc, logger, issue, now)
	}
	if issue, ok := evaluateMessagingOutboxBacklog(snapshot, thresholds); ok {
		current[issue.Key] = issue
		recordAsyncBacklogIssue(ctx, svc, logger, issue, now)
	}

	for issueKey, issue := range previous {
		if _, stillPresent := current[issueKey]; stillPresent {
			continue
		}
		if svc == nil {
			continue
		}
		if err := svc.ResolveIncident(ctx, asyncBacklogCorrelationKey(issue), now,
			fmt.Sprintf("Async backlog recovered for %s", issue.Kind),
			map[string]interface{}{
				"kind":   issue.Kind,
				"source": "async_backlog_signal_runner",
			},
		); err != nil && logger != nil {
			logger.Warn("Failed to resolve async backlog incident",
				zap.String("issue_key", issue.Key),
				zap.Error(err),
			)
		}
	}

	return current
}

func evaluateBuildQueueBacklog(snapshot AsyncBacklogSignalSnapshot, thresholds AsyncBacklogThresholds) (AsyncBacklogIssue, bool) {
	if snapshot.BuildQueueDepth < thresholds.BuildQueueThreshold {
		return AsyncBacklogIssue{}, false
	}
	return AsyncBacklogIssue{
		Key:       "build_queue_backlog",
		Kind:      "build_queue_backlog",
		Summary:   fmt.Sprintf("Build queue depth is %d, above threshold %d", snapshot.BuildQueueDepth, thresholds.BuildQueueThreshold),
		Severity:  severityForCount(snapshot.BuildQueueDepth, thresholds.BuildQueueThreshold*2),
		Count:     snapshot.BuildQueueDepth,
		Threshold: thresholds.BuildQueueThreshold,
		Payload: map[string]interface{}{
			"build_queue_depth": snapshot.BuildQueueDepth,
			"threshold":         thresholds.BuildQueueThreshold,
		},
	}, true
}

func evaluateEmailQueueBacklog(snapshot AsyncBacklogSignalSnapshot, thresholds AsyncBacklogThresholds) (AsyncBacklogIssue, bool) {
	if snapshot.EmailQueueDepth < thresholds.EmailQueueThreshold {
		return AsyncBacklogIssue{}, false
	}
	return AsyncBacklogIssue{
		Key:       "email_queue_backlog",
		Kind:      "email_queue_backlog",
		Summary:   fmt.Sprintf("Email queue depth is %d, above threshold %d", snapshot.EmailQueueDepth, thresholds.EmailQueueThreshold),
		Severity:  severityForCount(snapshot.EmailQueueDepth, thresholds.EmailQueueThreshold*2),
		Count:     snapshot.EmailQueueDepth,
		Threshold: thresholds.EmailQueueThreshold,
		Payload: map[string]interface{}{
			"email_queue_depth": snapshot.EmailQueueDepth,
			"threshold":         thresholds.EmailQueueThreshold,
		},
	}, true
}

func evaluateMessagingOutboxBacklog(snapshot AsyncBacklogSignalSnapshot, thresholds AsyncBacklogThresholds) (AsyncBacklogIssue, bool) {
	if snapshot.MessagingOutboxPending < thresholds.MessagingOutboxThreshold {
		return AsyncBacklogIssue{}, false
	}
	return AsyncBacklogIssue{
		Key:       "messaging_outbox_backlog",
		Kind:      "messaging_outbox_backlog",
		Summary:   fmt.Sprintf("Messaging outbox has %d pending records, above threshold %d", snapshot.MessagingOutboxPending, thresholds.MessagingOutboxThreshold),
		Severity:  severityForCount(snapshot.MessagingOutboxPending, thresholds.MessagingOutboxThreshold*2),
		Count:     snapshot.MessagingOutboxPending,
		Threshold: thresholds.MessagingOutboxThreshold,
		Payload: map[string]interface{}{
			"messaging_outbox_pending_count": snapshot.MessagingOutboxPending,
			"threshold":                      thresholds.MessagingOutboxThreshold,
		},
	}, true
}

func recordAsyncBacklogIssue(ctx context.Context, svc *Service, logger *zap.Logger, issue AsyncBacklogIssue, now time.Time) {
	if svc == nil {
		return
	}
	if err := svc.RecordObservation(ctx, SignalObservation{
		CorrelationKey: asyncBacklogCorrelationKey(issue),
		Domain:         "golden_signals",
		IncidentType:   "backlog_pressure",
		DisplayName:    asyncBacklogDisplayName(issue),
		Summary:        issue.Summary,
		Source:         "async_backlog_signal_runner",
		Severity:       issue.Severity,
		Confidence:     domainsresmartbot.IncidentConfidenceHigh,
		OccurredAt:     now,
		Metadata: map[string]interface{}{
			"kind":      issue.Kind,
			"count":     issue.Count,
			"threshold": issue.Threshold,
		},
		FindingTitle:   "Async backlog pressure detected",
		FindingMessage: issue.Summary,
		SignalType:     "async_backlog_pressure",
		SignalKey:      issue.Key,
		RawPayload:     issue.Payload,
	}); err != nil && logger != nil {
		logger.Warn("Failed to record async backlog incident observation",
			zap.String("issue_key", issue.Key),
			zap.Error(err),
		)
	}
	_ = svc.AddEvidence(ctx, asyncBacklogCorrelationKey(issue), "async_backlog_snapshot", issue.Summary, issue.Payload, now)
	_ = svc.EnsureActionAttempt(ctx, asyncBacklogCorrelationKey(issue), ActionAttemptSpec{
		ActionKey:     "review_async_backlog",
		ActionClass:   "recommendation",
		TargetKind:    "async_pipeline",
		TargetRef:     issue.Kind,
		Status:        "proposed",
		ActorType:     "system",
		ResultPayload: issue.Payload,
	}, now)
}

func asyncBacklogCorrelationKey(issue AsyncBacklogIssue) string {
	return "golden_signal:backlog:" + strings.TrimSpace(issue.Kind)
}

func asyncBacklogDisplayName(issue AsyncBacklogIssue) string {
	switch issue.Kind {
	case "build_queue_backlog":
		return "Build queue backlog pressure"
	case "email_queue_backlog":
		return "Email queue backlog pressure"
	case "messaging_outbox_backlog":
		return "Messaging outbox backlog pressure"
	default:
		return "Async backlog pressure"
	}
}

func severityForCount(count int64, criticalThreshold int64) domainsresmartbot.IncidentSeverity {
	if count >= criticalThreshold && criticalThreshold > 0 {
		return domainsresmartbot.IncidentSeverityCritical
	}
	return domainsresmartbot.IncidentSeverityWarning
}

func ObserveTenantAssetDriftTick(ctx context.Context, svc *Service, logger *zap.Logger, result *domaininfrastructure.TenantAssetDriftWatchTickResult, metrics domaininfrastructure.TenantAssetDriftMetricsSnapshot, tickErr error, now time.Time) {
	const correlationKey = "tenant_asset_drift:global"
	if svc == nil {
		return
	}
	if tickErr != nil {
		message := fmt.Sprintf("Tenant asset drift watcher failed: %v", tickErr)
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: correlationKey,
			Domain:         "application_services",
			IncidentType:   "tenant_asset_drift_watcher_failed",
			DisplayName:    "Tenant asset drift watcher degraded",
			Summary:        message,
			Source:         "tenant_asset_drift_watcher",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata:       map[string]interface{}{"mode": "watch_tick_failed"},
			FindingTitle:   "Tenant asset drift watch tick failed",
			FindingMessage: message,
			SignalType:     "tenant_asset_drift_watch_failure",
			SignalKey:      "global",
			RawPayload: map[string]interface{}{
				"error":   tickErr.Error(),
				"metrics": tenantAssetDriftMetricsPayload(metrics),
			},
		}); err != nil && logger != nil {
			logger.Warn("Failed to record tenant asset drift failure", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, correlationKey, "tenant_asset_drift_watch_failure", message, map[string]interface{}{
			"error":   tickErr.Error(),
			"metrics": tenantAssetDriftMetricsPayload(metrics),
		}, now)
		_ = svc.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:     "reconcile_tenant_assets",
			ActionClass:   "recommendation",
			TargetKind:    "tenant_namespace",
			TargetRef:     "global",
			Status:        "proposed",
			ActorType:     "system",
			ResultPayload: map[string]interface{}{"reason": "watch_tick_failed"},
		}, now)
		return
	}
	if result == nil {
		return
	}
	if result.Stale > 0 || result.Unknown > 0 || result.Failed > 0 {
		summary := fmt.Sprintf("Tenant asset drift detected: %d stale, %d unknown, %d failed", result.Stale, result.Unknown, result.Failed)
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: correlationKey,
			Domain:         "application_services",
			IncidentType:   "tenant_asset_drift_detected",
			DisplayName:    "Tenant asset drift detected",
			Summary:        summary,
			Source:         "tenant_asset_drift_watcher",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"total_namespaces": result.TotalNamespaces,
				"current":          result.Current,
				"stale":            result.Stale,
				"unknown":          result.Unknown,
				"failed":           result.Failed,
			},
			FindingTitle:   "Tenant asset drift detected",
			FindingMessage: summary,
			SignalType:     "tenant_asset_drift_summary",
			SignalKey:      "global",
			RawPayload: map[string]interface{}{
				"result":  result,
				"metrics": tenantAssetDriftMetricsPayload(metrics),
			},
		}); err != nil && logger != nil {
			logger.Warn("Failed to record tenant asset drift incident observation", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, correlationKey, "tenant_asset_drift_tick", summary, map[string]interface{}{
			"result":  result,
			"metrics": tenantAssetDriftMetricsPayload(metrics),
		}, now)
		_ = svc.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:     "reconcile_tenant_assets",
			ActionClass:   "recommendation",
			TargetKind:    "tenant_namespace",
			TargetRef:     "global",
			Status:        "proposed",
			ActorType:     "system",
			ResultPayload: map[string]interface{}{"stale": result.Stale, "unknown": result.Unknown, "failed": result.Failed},
		}, now)
		return
	}
	if err := svc.ResolveIncident(ctx, correlationKey, now, "Tenant asset drift healthy", map[string]interface{}{
		"total_namespaces": result.TotalNamespaces,
		"current":          result.Current,
		"stale":            result.Stale,
		"unknown":          result.Unknown,
		"failed":           result.Failed,
		"source":           "tenant_asset_drift_watcher",
	}); err != nil && logger != nil {
		logger.Warn("Failed to resolve tenant asset drift incident", zap.Error(err))
	}
}

func ObserveReleaseComplianceTick(ctx context.Context, svc *Service, logger *zap.Logger, detected []releasecompliance.DriftRecord, recovered []releasecompliance.DriftRecord, snapshot releasecompliance.Snapshot, tickErr error, now time.Time) {
	if svc == nil {
		return
	}
	const watcherCorrelationKey = "release_compliance_watcher:global"
	if tickErr != nil {
		message := fmt.Sprintf("Release compliance watcher failed: %v", tickErr)
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: watcherCorrelationKey,
			Domain:         "release_configuration",
			IncidentType:   "release_compliance_watcher_failed",
			DisplayName:    "Release compliance watcher degraded",
			Summary:        message,
			Source:         "quarantine_release_compliance_watcher",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata:       map[string]interface{}{"mode": "watch_tick_failed"},
			FindingTitle:   "Release compliance watch tick failed",
			FindingMessage: message,
			SignalType:     "release_compliance_watch_failure",
			SignalKey:      "global",
			RawPayload: map[string]interface{}{
				"error":    tickErr.Error(),
				"snapshot": releaseComplianceSnapshotPayload(snapshot),
			},
		}); err != nil && logger != nil {
			logger.Warn("Failed to record release compliance watcher failure", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, watcherCorrelationKey, "release_compliance_watch_failure", message, map[string]interface{}{
			"error":    tickErr.Error(),
			"snapshot": releaseComplianceSnapshotPayload(snapshot),
		}, now)
		return
	}

	for _, rec := range detected {
		correlationKey := "release_compliance:" + rec.ExternalImageImportID.String()
		summary := fmt.Sprintf("Release compliance drift detected for import %s", rec.ExternalImageImportID.String())
		if err := svc.RecordObservation(ctx, SignalObservation{
			CorrelationKey: correlationKey,
			Domain:         "release_configuration",
			IncidentType:   "release_compliance_drift_detected",
			DisplayName:    "Release compliance drift detected",
			Summary:        summary,
			Source:         "quarantine_release_compliance_watcher",
			Severity:       domainsresmartbot.IncidentSeverityWarning,
			Confidence:     domainsresmartbot.IncidentConfidenceHigh,
			OccurredAt:     now,
			Metadata: map[string]interface{}{
				"external_image_import_id": rec.ExternalImageImportID.String(),
				"tenant_id":                rec.TenantID.String(),
				"release_state":            rec.ReleaseState,
			},
			FindingTitle:   "Release compliance drift detected",
			FindingMessage: summary,
			SignalType:     "release_compliance_drift_detected",
			SignalKey:      rec.ExternalImageImportID.String(),
			RawPayload:     releaseComplianceDriftPayload(rec),
		}); err != nil && logger != nil {
			logger.Warn("Failed to record release compliance drift incident", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, correlationKey, "release_compliance_drift_record", summary, releaseComplianceDriftPayload(rec), now)
		_ = svc.EnsureActionAttempt(ctx, correlationKey, ActionAttemptSpec{
			ActionKey:     "review_release_drift",
			ActionClass:   "recommendation",
			TargetKind:    "external_image_import",
			TargetRef:     rec.ExternalImageImportID.String(),
			Status:        "proposed",
			ActorType:     "system",
			ResultPayload: map[string]interface{}{"release_state": rec.ReleaseState},
		}, now)
	}

	for _, rec := range recovered {
		correlationKey := "release_compliance:" + rec.ExternalImageImportID.String()
		if err := svc.ResolveIncident(ctx, correlationKey, now, "Release compliance drift recovered", map[string]interface{}{
			"external_image_import_id": rec.ExternalImageImportID.String(),
			"tenant_id":                rec.TenantID.String(),
			"release_state":            rec.ReleaseState,
			"source":                   "quarantine_release_compliance_watcher",
		}); err != nil && logger != nil {
			logger.Warn("Failed to resolve release compliance drift incident", zap.Error(err))
		}
		_ = svc.AddEvidence(ctx, correlationKey, "release_compliance_recovered", "Release compliance drift recovered", releaseComplianceDriftPayload(rec), now)
	}
}

func tenantAssetDriftMetricsPayload(metrics domaininfrastructure.TenantAssetDriftMetricsSnapshot) map[string]interface{} {
	return map[string]interface{}{
		"watch_ticks_total":           metrics.WatchTicksTotal,
		"watch_failures_total":        metrics.WatchFailuresTotal,
		"watch_current_namespaces":    metrics.WatchCurrentNamespaces,
		"watch_stale_namespaces":      metrics.WatchStaleNamespaces,
		"watch_unknown_namespaces":    metrics.WatchUnknownNamespaces,
		"watch_duration_count":        metrics.WatchDurationCount,
		"watch_duration_total_ms":     metrics.WatchDurationTotalMs,
		"watch_duration_max_ms":       metrics.WatchDurationMaxMs,
		"reconcile_requests_total":    metrics.ReconcileRequestsTotal,
		"reconcile_requests_success":  metrics.ReconcileRequestsSuccess,
		"reconcile_requests_failures": metrics.ReconcileRequestsFailures,
	}
}

func releaseComplianceDriftPayload(rec releasecompliance.DriftRecord) map[string]interface{} {
	return map[string]interface{}{
		"external_image_import_id": rec.ExternalImageImportID.String(),
		"tenant_id":                rec.TenantID.String(),
		"release_state":            rec.ReleaseState,
		"internal_image_ref":       rec.InternalImageRef,
		"source_image_digest":      rec.SourceImageDigest,
		"released_at":              rec.ReleasedAt.UTC().Format(time.RFC3339),
	}
}

func releaseComplianceSnapshotPayload(snapshot releasecompliance.Snapshot) map[string]interface{} {
	return map[string]interface{}{
		"watch_ticks_total":     snapshot.WatchTicksTotal,
		"watch_failures_total":  snapshot.WatchFailuresTotal,
		"drift_detected_total":  snapshot.DriftDetectedTotal,
		"drift_recovered_total": snapshot.DriftRecoveredTotal,
		"active_drift_count":    snapshot.ActiveDriftCount,
		"released_count":        snapshot.ReleasedCount,
	}
}
