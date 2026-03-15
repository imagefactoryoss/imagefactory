package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/application/clustermetrics"
	"go.uber.org/zap"
)

type ClusterMetricsSnapshotRepository struct {
	db *sqlx.DB
}

func NewClusterMetricsSnapshotRepository(db *sqlx.DB, _ *zap.Logger) clustermetrics.SnapshotRepository {
	return &ClusterMetricsSnapshotRepository{db: db}
}

func (r *ClusterMetricsSnapshotRepository) StoreSnapshotBatch(ctx context.Context, nodes []clustermetrics.NodeSnapshot, pods []clustermetrics.PodSnapshot, retentionDays int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const insertNode = `
		INSERT INTO cluster_node_metrics_snapshots (
			cluster_name,
			node_name,
			cpu_usage_millicores,
			memory_usage_bytes,
			cpu_allocatable_millicores,
			memory_allocatable_bytes,
			ephemeral_storage_allocatable_bytes,
			window_seconds,
			collected_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`

	for _, snapshot := range nodes {
		if _, err = tx.ExecContext(ctx, insertNode,
			snapshot.ClusterName,
			snapshot.NodeName,
			snapshot.CPUUsageMillicores,
			snapshot.MemoryUsageBytes,
			snapshot.CPUAllocatableMillicores,
			snapshot.MemoryAllocatableBytes,
			snapshot.EphemeralStorageAllocatableBytes,
			snapshot.WindowSeconds,
			snapshot.CollectedAt.UTC(),
		); err != nil {
			return fmt.Errorf("insert node snapshot for %s: %w", snapshot.NodeName, err)
		}
	}

	const insertPod = `
		INSERT INTO cluster_pod_metrics_snapshots (
			cluster_name,
			namespace,
			pod_name,
			node_name,
			container_count,
			cpu_usage_millicores,
			memory_usage_bytes,
			window_seconds,
			collected_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`

	for _, snapshot := range pods {
		if _, err = tx.ExecContext(ctx, insertPod,
			snapshot.ClusterName,
			snapshot.Namespace,
			snapshot.PodName,
			snapshot.NodeName,
			snapshot.ContainerCount,
			snapshot.CPUUsageMillicores,
			snapshot.MemoryUsageBytes,
			snapshot.WindowSeconds,
			snapshot.CollectedAt.UTC(),
		); err != nil {
			return fmt.Errorf("insert pod snapshot for %s/%s: %w", snapshot.Namespace, snapshot.PodName, err)
		}
	}

	if retentionDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
		if _, err = tx.ExecContext(ctx, `DELETE FROM cluster_node_metrics_snapshots WHERE collected_at < $1`, cutoff); err != nil {
			return fmt.Errorf("delete expired node metrics snapshots: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM cluster_pod_metrics_snapshots WHERE collected_at < $1`, cutoff); err != nil {
			return fmt.Errorf("delete expired pod metrics snapshots: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit cluster metrics snapshot transaction: %w", err)
	}

	return nil
}
