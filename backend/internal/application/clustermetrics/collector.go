package clustermetrics

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type NodeSnapshot struct {
	ClusterName                      string
	NodeName                         string
	CPUUsageMillicores               int64
	MemoryUsageBytes                 int64
	CPUAllocatableMillicores         int64
	MemoryAllocatableBytes           int64
	EphemeralStorageAllocatableBytes int64
	WindowSeconds                    int
	CollectedAt                      time.Time
}

type PodSnapshot struct {
	ClusterName        string
	Namespace          string
	PodName            string
	NodeName           string
	ContainerCount     int
	CPUUsageMillicores int64
	MemoryUsageBytes   int64
	WindowSeconds      int
	CollectedAt        time.Time
}

type PodSignalSnapshot struct {
	ClusterName    string
	Namespace      string
	PodName        string
	NodeName       string
	RestartCount   int32
	Phase          string
	Reason         string
	ContainerCount int
}

type SnapshotRepository interface {
	StoreSnapshotBatch(ctx context.Context, nodes []NodeSnapshot, pods []PodSnapshot, retentionDays int) error
}

type CollectResult struct {
	NodesCollected     int
	PodsCollected      int
	CollectedAt        time.Time
	NodeSnapshots      []NodeSnapshot
	PodSnapshots       []PodSnapshot
	PodSignalSnapshots []PodSignalSnapshot
}

type Collector struct {
	clusterName   string
	coreClient    kubernetes.Interface
	metricsClient metricsclient.Interface
	repo          SnapshotRepository
}

func NewCollector(clusterName string, coreClient kubernetes.Interface, metricsClient metricsclient.Interface, repo SnapshotRepository) *Collector {
	return &Collector{
		clusterName:   clusterName,
		coreClient:    coreClient,
		metricsClient: metricsClient,
		repo:          repo,
	}
}

func (c *Collector) CollectAndStore(ctx context.Context, retentionDays int) (CollectResult, error) {
	if c == nil || c.coreClient == nil || c.metricsClient == nil || c.repo == nil {
		return CollectResult{}, fmt.Errorf("cluster metrics collector is not fully configured")
	}

	collectedAt := time.Now().UTC()

	nodes, err := c.coreClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return CollectResult{}, fmt.Errorf("list nodes: %w", err)
	}

	pods, err := c.coreClient.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return CollectResult{}, fmt.Errorf("list pods: %w", err)
	}

	nodeMetrics, err := c.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return CollectResult{}, fmt.Errorf("list node metrics: %w", err)
	}

	podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return CollectResult{}, fmt.Errorf("list pod metrics: %w", err)
	}

	nodeAllocatable := make(map[string]corev1.ResourceList, len(nodes.Items))
	for _, node := range nodes.Items {
		nodeAllocatable[node.Name] = node.Status.Allocatable
	}

	podNodeNames := make(map[string]string, len(pods.Items))
	for _, pod := range pods.Items {
		key := pod.Namespace + "/" + pod.Name
		podNodeNames[key] = pod.Spec.NodeName
	}

	nodeSnapshots := make([]NodeSnapshot, 0, len(nodeMetrics.Items))
	for _, metric := range nodeMetrics.Items {
		alloc := nodeAllocatable[metric.Name]
		nodeSnapshots = append(nodeSnapshots, NodeSnapshot{
			ClusterName:                      c.clusterName,
			NodeName:                         metric.Name,
			CPUUsageMillicores:               metric.Usage.Cpu().MilliValue(),
			MemoryUsageBytes:                 metric.Usage.Memory().Value(),
			CPUAllocatableMillicores:         alloc.Cpu().MilliValue(),
			MemoryAllocatableBytes:           alloc.Memory().Value(),
			EphemeralStorageAllocatableBytes: alloc.StorageEphemeral().Value(),
			WindowSeconds:                    int(metric.Window.Duration.Seconds()),
			CollectedAt:                      collectedAt,
		})
	}

	podSnapshots := make([]PodSnapshot, 0, len(podMetrics.Items))
	podSignalSnapshots := make([]PodSignalSnapshot, 0, len(pods.Items))
	for _, pod := range pods.Items {
		var restartCount int32
		for _, status := range pod.Status.InitContainerStatuses {
			restartCount += status.RestartCount
		}
		for _, status := range pod.Status.ContainerStatuses {
			restartCount += status.RestartCount
		}
		podSignalSnapshots = append(podSignalSnapshots, PodSignalSnapshot{
			ClusterName:    c.clusterName,
			Namespace:      pod.Namespace,
			PodName:        pod.Name,
			NodeName:       pod.Spec.NodeName,
			RestartCount:   restartCount,
			Phase:          string(pod.Status.Phase),
			Reason:         pod.Status.Reason,
			ContainerCount: len(pod.Spec.Containers),
		})
	}

	for _, metric := range podMetrics.Items {
		var cpuMillicores int64
		var memoryBytes int64
		for _, container := range metric.Containers {
			cpuMillicores += container.Usage.Cpu().MilliValue()
			memoryBytes += container.Usage.Memory().Value()
		}
		key := metric.Namespace + "/" + metric.Name
		podSnapshots = append(podSnapshots, PodSnapshot{
			ClusterName:        c.clusterName,
			Namespace:          metric.Namespace,
			PodName:            metric.Name,
			NodeName:           podNodeNames[key],
			ContainerCount:     len(metric.Containers),
			CPUUsageMillicores: cpuMillicores,
			MemoryUsageBytes:   memoryBytes,
			WindowSeconds:      int(metric.Window.Duration.Seconds()),
			CollectedAt:        collectedAt,
		})
	}

	if err := c.repo.StoreSnapshotBatch(ctx, nodeSnapshots, podSnapshots, retentionDays); err != nil {
		return CollectResult{}, fmt.Errorf("store cluster metrics snapshot batch: %w", err)
	}

	return CollectResult{
		NodesCollected:     len(nodeSnapshots),
		PodsCollected:      len(podSnapshots),
		CollectedAt:        collectedAt,
		NodeSnapshots:      nodeSnapshots,
		PodSnapshots:       podSnapshots,
		PodSignalSnapshots: podSignalSnapshots,
	}, nil
}
