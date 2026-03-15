package clustermetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type RunnerConfig struct {
	Enabled                     bool
	Interval                    time.Duration
	Timeout                     time.Duration
	RetentionDays               int
	ClusterName                 string
	NodeCPUSaturationPercent    int
	NodeMemorySaturationPercent int
	PodRestartThreshold         int32
}

func StartSnapshotIngester(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	coreClient kubernetes.Interface,
	metricsClient metricsclient.Interface,
	repo SnapshotRepository,
	sreSmartBotService *appsresmartbot.Service,
	cfg RunnerConfig,
) {
	if cfg.Interval < time.Minute {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.Timeout < 5*time.Second {
		cfg.Timeout = 20 * time.Second
	}
	if cfg.RetentionDays < 1 {
		cfg.RetentionDays = 14
	}
	if cfg.ClusterName == "" {
		cfg.ClusterName = "image-factory"
	}
	if cfg.NodeCPUSaturationPercent < 1 {
		cfg.NodeCPUSaturationPercent = 85
	}
	if cfg.NodeMemorySaturationPercent < 1 {
		cfg.NodeMemorySaturationPercent = 85
	}
	if cfg.PodRestartThreshold < 1 {
		cfg.PodRestartThreshold = 3
	}

	statusMessage := "cluster metrics snapshot ingester initialized"
	running := cfg.Enabled
	if !cfg.Enabled {
		statusMessage = "cluster metrics snapshot ingester disabled"
		running = false
	} else if coreClient == nil || metricsClient == nil {
		statusMessage = "cluster metrics snapshot ingester unavailable: kubernetes metrics client not configured"
		running = false
	}
	processHealthStore.Upsert("cluster_metrics_snapshot_ingester", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      running,
		LastActivity: time.Now().UTC(),
		Message:      statusMessage,
		Metrics: map[string]int64{
			"cluster_metrics_collections_total":         0,
			"cluster_metrics_collection_failures_total": 0,
			"cluster_metrics_nodes_last_collected":      0,
			"cluster_metrics_pods_last_collected":       0,
			"cluster_metrics_retention_days":            int64(cfg.RetentionDays),
			"cluster_metrics_interval_seconds":          int64(cfg.Interval / time.Second),
			"cluster_metrics_timeout_seconds":           int64(cfg.Timeout / time.Second),
		},
	})
	if !cfg.Enabled || coreClient == nil || metricsClient == nil {
		return
	}

	collector := NewCollector(cfg.ClusterName, coreClient, metricsClient, repo)
	logger.Info("Background process starting",
		zap.String("component", "cluster_metrics_snapshot_ingester"),
		zap.String("cluster_name", cfg.ClusterName),
		zap.Duration("interval", cfg.Interval),
		zap.Duration("timeout", cfg.Timeout),
		zap.Int("retention_days", cfg.RetentionDays),
	)

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		var collectionsTotal int64
		var collectionFailures int64
		var nodesLastCollected int64
		var podsLastCollected int64
		var previousGoldenSignalIssues map[string]appsresmartbot.GoldenSignalIssue

		runTick := func() {
			now := time.Now().UTC()
			collectionsTotal++

			collectCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
			result, err := collector.CollectAndStore(collectCtx, cfg.RetentionDays)
			cancel()
			if err != nil {
				collectionFailures++
				appsresmartbot.ObserveClusterMetricsIngesterFailure(context.Background(), sreSmartBotService, logger, err, now)
				processHealthStore.Upsert("cluster_metrics_snapshot_ingester", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      false,
					LastActivity: now,
					Message:      fmt.Sprintf("collection failed: %v", err),
					Metrics: map[string]int64{
						"cluster_metrics_collections_total":         collectionsTotal,
						"cluster_metrics_collection_failures_total": collectionFailures,
						"cluster_metrics_nodes_last_collected":      nodesLastCollected,
						"cluster_metrics_pods_last_collected":       podsLastCollected,
						"cluster_metrics_retention_days":            int64(cfg.RetentionDays),
						"cluster_metrics_interval_seconds":          int64(cfg.Interval / time.Second),
						"cluster_metrics_timeout_seconds":           int64(cfg.Timeout / time.Second),
					},
				})
				logger.Warn("Cluster metrics snapshot collection failed", zap.Error(err))
				return
			}

			nodesLastCollected = int64(result.NodesCollected)
			podsLastCollected = int64(result.PodsCollected)
			nodeSignals := make([]appsresmartbot.ClusterNodeSignalSnapshot, 0, len(result.NodeSnapshots))
			for _, node := range result.NodeSnapshots {
				nodeSignals = append(nodeSignals, appsresmartbot.ClusterNodeSignalSnapshot{
					ClusterName:              node.ClusterName,
					NodeName:                 node.NodeName,
					CPUUsageMillicores:       node.CPUUsageMillicores,
					MemoryUsageBytes:         node.MemoryUsageBytes,
					CPUAllocatableMillicores: node.CPUAllocatableMillicores,
					MemoryAllocatableBytes:   node.MemoryAllocatableBytes,
				})
			}
			podSignals := make([]appsresmartbot.ClusterPodSignalSnapshot, 0, len(result.PodSignalSnapshots))
			for _, pod := range result.PodSignalSnapshots {
				podSignals = append(podSignals, appsresmartbot.ClusterPodSignalSnapshot{
					ClusterName:    pod.ClusterName,
					Namespace:      pod.Namespace,
					PodName:        pod.PodName,
					NodeName:       pod.NodeName,
					RestartCount:   pod.RestartCount,
					Phase:          pod.Phase,
					Reason:         pod.Reason,
					ContainerCount: pod.ContainerCount,
				})
			}
			previousGoldenSignalIssues = appsresmartbot.ObserveClusterGoldenSignals(
				context.Background(),
				sreSmartBotService,
				logger,
				nodeSignals,
				podSignals,
				now,
				previousGoldenSignalIssues,
				appsresmartbot.GoldenSignalThresholds{
					NodeCPUSaturationPercent:    cfg.NodeCPUSaturationPercent,
					NodeMemorySaturationPercent: cfg.NodeMemorySaturationPercent,
					PodRestartThreshold:         cfg.PodRestartThreshold,
				},
			)
			appsresmartbot.ResolveClusterMetricsIngester(context.Background(), sreSmartBotService, logger, now, result.NodesCollected, result.PodsCollected)
			processHealthStore.Upsert("cluster_metrics_snapshot_ingester", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("collected %d node snapshots and %d pod snapshots", result.NodesCollected, result.PodsCollected),
				Metrics: map[string]int64{
					"cluster_metrics_collections_total":         collectionsTotal,
					"cluster_metrics_collection_failures_total": collectionFailures,
					"cluster_metrics_nodes_last_collected":      nodesLastCollected,
					"cluster_metrics_pods_last_collected":       podsLastCollected,
					"cluster_metrics_retention_days":            int64(cfg.RetentionDays),
					"cluster_metrics_interval_seconds":          int64(cfg.Interval / time.Second),
					"cluster_metrics_timeout_seconds":           int64(cfg.Timeout / time.Second),
				},
			})
		}

		runTick()
		for {
			<-ticker.C
			runTick()
		}
	}()
}
