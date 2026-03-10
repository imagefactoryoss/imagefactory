package secondary

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// InfrastructureAdminService provides infrastructure management operations
type InfrastructureAdminService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewInfrastructureAdminService creates a new infrastructure admin service
func NewInfrastructureAdminService(db *sql.DB, logger *zap.Logger) *InfrastructureAdminService {
	return &InfrastructureAdminService{
		db:     db,
		logger: logger,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// CreateNodeRequest represents a request to create a node
type CreateNodeRequest struct {
	Name        string  `validate:"required,min=3"`
	TotalCPU    float64 `validate:"required,min=1"`
	TotalMemory float64 `validate:"required,min=1"`
	TotalDisk   float64 `validate:"required,min=1"`
	Labels      map[string]string
}

// UpdateNodeRequest represents a request to update a node
type UpdateNodeRequest struct {
	Name            *string
	Status          *string
	MaintenanceMode *bool
	Labels          map[string]string
}

// Node represents an infrastructure node
type Node struct {
	ID               string
	Name             string
	Status           string
	TotalCPUCores    float64
	TotalMemoryGB    float64
	TotalDiskGB      float64
	LastHeartbeat    *time.Time
	MaintenanceMode  bool
	Labels           map[string]string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CurrentResources *ResourceUsage
}

// ResourceUsage represents current resource usage for a node
type ResourceUsage struct {
	UsedCPUCores    float64
	UsedMemoryGB    float64
	UsedDiskGB      float64
	RecordedAt      time.Time
	CPUUsagePercent float64
	MemoryPercent   float64
	DiskPercent     float64
}

// ListNodesOptions represents options for listing nodes
type ListNodesOptions struct {
	Status string
	Limit  int
	Offset int
}

// ListNodesResult represents the result of listing nodes
type ListNodesResult struct {
	Nodes   []Node
	Total   int
	Limit   int
	Offset  int
	HasMore bool
}

// HealthMetrics represents infrastructure health metrics
type HealthMetrics struct {
	TotalNodes          int
	HealthyNodes        int
	OfflineNodes        int
	MaintenanceNodes    int
	TotalCPUCapacity    float64
	TotalMemoryCapacity float64
	TotalDiskCapacity   float64
	UsedCPU             float64
	UsedMemory          float64
	UsedDisk            float64
	AverageCPUUsage     float64
	AverageMemoryUsage  float64
	AverageDiskUsage    float64
}

// NodeHealth represents health metrics for a single node
type NodeHealth struct {
	NodeID          string
	NodeName        string
	Status          string
	CPUUsagePercent float64
	MemoryPercent   float64
	DiskPercent     float64
	UpSince         *time.Time
}

// ============================================================================
// Public Methods
// ============================================================================

// CreateNode creates a new infrastructure node
func (s *InfrastructureAdminService) CreateNode(ctx context.Context, req *CreateNodeRequest) (*Node, error) {
	// Validate request
	if req.Name == "" || len(req.Name) < 3 {
		return nil, errors.New("name must be at least 3 characters")
	}

	if req.TotalCPU < 1 || req.TotalMemory < 1 || req.TotalDisk < 1 {
		return nil, errors.New("CPU, memory, and disk must be at least 1")
	}

	nodeID := uuid.New().String()
	now := time.Now()

	labelsJSON := "{}"
	if req.Labels != nil {
		if data, err := json.Marshal(req.Labels); err == nil {
			labelsJSON = string(data)
		}
	}

	query := `
		INSERT INTO infrastructure_nodes 
		(id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, maintenance_mode, labels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
		RETURNING id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, last_heartbeat, maintenance_mode, labels, created_at, updated_at
	`

	var node Node
	var lastHeartbeat, createdAt, updatedAt sql.NullTime
	var labels sql.NullString

	if err := s.db.QueryRowContext(ctx,
		query,
		nodeID, req.Name, "ready", req.TotalCPU, req.TotalMemory, req.TotalDisk, false, labelsJSON, now, now,
	).Scan(
		&node.ID, &node.Name, &node.Status, &node.TotalCPUCores, &node.TotalMemoryGB, &node.TotalDiskGB,
		&lastHeartbeat, &node.MaintenanceMode, &labels, &createdAt, &updatedAt,
	); err != nil {
		s.logger.Error("Failed to create node", zap.Error(err))
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	if lastHeartbeat.Valid {
		node.LastHeartbeat = &lastHeartbeat.Time
	}
	node.CreatedAt = createdAt.Time
	node.UpdatedAt = updatedAt.Time
	if labels.Valid {
		json.Unmarshal([]byte(labels.String), &node.Labels)
	}

	s.logger.Info("Node created", zap.String("nodeID", node.ID), zap.String("nodeName", node.Name))
	return &node, nil
}

// GetNode retrieves a specific infrastructure node
func (s *InfrastructureAdminService) GetNode(ctx context.Context, nodeID string) (*Node, error) {
	var node Node
	var lastHeartbeat, createdAt, updatedAt sql.NullTime
	var labels sql.NullString

	query := `
		SELECT 
			id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb,
			last_heartbeat, maintenance_mode, labels, created_at, updated_at
		FROM v_infrastructure_nodes
		WHERE id = $1
	`

	if err := s.db.QueryRowContext(ctx, query, nodeID).Scan(
		&node.ID, &node.Name, &node.Status, &node.TotalCPUCores, &node.TotalMemoryGB, &node.TotalDiskGB,
		&lastHeartbeat, &node.MaintenanceMode, &labels, &createdAt, &updatedAt,
	); err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	} else if err != nil {
		s.logger.Error("Failed to query node", zap.Error(err))
		return nil, fmt.Errorf("failed to query node: %w", err)
	}

	if lastHeartbeat.Valid {
		node.LastHeartbeat = &lastHeartbeat.Time
	}
	node.CreatedAt = createdAt.Time
	node.UpdatedAt = updatedAt.Time
	if labels.Valid {
		json.Unmarshal([]byte(labels.String), &node.Labels)
	}

	node.CurrentResources = s.getNodeResourceUsage(ctx, node.ID, node.TotalCPUCores, node.TotalMemoryGB, node.TotalDiskGB)

	return &node, nil
}

// ListNodes returns a paginated list of infrastructure nodes
func (s *InfrastructureAdminService) ListNodes(ctx context.Context, opts *ListNodesOptions) (*ListNodesResult, error) {
	if opts == nil {
		opts = &ListNodesOptions{
			Limit:  20,
			Offset: 0,
		}
	}

	// Validate and normalize options
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	// Count total nodes
	countQuery := `SELECT COUNT(*) FROM infrastructure_nodes WHERE 1=1`
	countArgs := []interface{}{}

	if opts.Status != "" {
		countQuery += ` AND status = $1`
		countArgs = append(countArgs, opts.Status)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		s.logger.Error("Failed to count nodes", zap.Error(err))
		return nil, fmt.Errorf("failed to count nodes: %w", err)
	}

	// Get nodes
	query := `
		SELECT 
			id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb,
			last_heartbeat, maintenance_mode, labels, created_at, updated_at
		FROM v_infrastructure_nodes
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if opts.Status != "" {
		query += ` AND status = $` + strconv.Itoa(argIndex)
		args = append(args, opts.Status)
		argIndex++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, opts.Limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to query nodes", zap.Error(err))
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var lastHeartbeat, createdAt, updatedAt sql.NullTime
		var labels sql.NullString

		if err := rows.Scan(
			&node.ID, &node.Name, &node.Status, &node.TotalCPUCores, &node.TotalMemoryGB, &node.TotalDiskGB,
			&lastHeartbeat, &node.MaintenanceMode, &labels, &createdAt, &updatedAt,
		); err != nil {
			s.logger.Error("Failed to scan node", zap.Error(err))
			continue
		}

		if lastHeartbeat.Valid {
			node.LastHeartbeat = &lastHeartbeat.Time
		}
		node.CreatedAt = createdAt.Time
		node.UpdatedAt = updatedAt.Time
		if labels.Valid {
			json.Unmarshal([]byte(labels.String), &node.Labels)
		}

		node.CurrentResources = s.getNodeResourceUsage(ctx, node.ID, node.TotalCPUCores, node.TotalMemoryGB, node.TotalDiskGB)

		nodes = append(nodes, node)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Error iterating nodes", zap.Error(err))
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return &ListNodesResult{
		Nodes:   nodes,
		Total:   total,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
		HasMore: opts.Offset+opts.Limit < total,
	}, nil
}

// UpdateNode updates an infrastructure node
func (s *InfrastructureAdminService) UpdateNode(ctx context.Context, nodeID string, req *UpdateNodeRequest) (*Node, error) {
	// Build dynamic update query
	updateQuery := `UPDATE infrastructure_nodes SET updated_at = $1`
	args := []interface{}{time.Now()}
	argIndex := 2

	if req.Name != nil {
		updateQuery += `, name = $` + strconv.Itoa(argIndex)
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Status != nil {
		updateQuery += `, status = $` + strconv.Itoa(argIndex)
		args = append(args, *req.Status)
		argIndex++
	}

	if req.MaintenanceMode != nil {
		updateQuery += `, maintenance_mode = $` + strconv.Itoa(argIndex)
		args = append(args, *req.MaintenanceMode)
		argIndex++
	}

	if req.Labels != nil {
		labelsJSON, _ := json.Marshal(req.Labels)
		updateQuery += `, labels = $` + strconv.Itoa(argIndex) + `::jsonb`
		args = append(args, string(labelsJSON))
		argIndex++
	}

	updateQuery += ` WHERE id = $` + strconv.Itoa(argIndex) + ` RETURNING id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, last_heartbeat, maintenance_mode, labels, created_at, updated_at`
	args = append(args, nodeID)

	var node Node
	var lastHeartbeat, createdAt, updatedAt sql.NullTime
	var labels sql.NullString

	if err := s.db.QueryRowContext(ctx, updateQuery, args...).Scan(
		&node.ID, &node.Name, &node.Status, &node.TotalCPUCores, &node.TotalMemoryGB, &node.TotalDiskGB,
		&lastHeartbeat, &node.MaintenanceMode, &labels, &createdAt, &updatedAt,
	); err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	} else if err != nil {
		s.logger.Error("Failed to update node", zap.Error(err))
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	node.CreatedAt = createdAt.Time
	node.UpdatedAt = updatedAt.Time
	if labels.Valid {
		json.Unmarshal([]byte(labels.String), &node.Labels)
	}

	s.logger.Info("Node updated", zap.String("nodeID", nodeID))
	return &node, nil
}

// DeleteNode deletes an infrastructure node
func (s *InfrastructureAdminService) DeleteNode(ctx context.Context, nodeID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM infrastructure_nodes WHERE id = $1`, nodeID)
	if err != nil {
		s.logger.Error("Failed to delete node", zap.Error(err))
		return fmt.Errorf("failed to delete node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	s.logger.Info("Node deleted", zap.String("nodeID", nodeID))
	return nil
}

// GetHealth returns overall infrastructure health metrics
func (s *InfrastructureAdminService) GetHealth(ctx context.Context) (*HealthMetrics, error) {
	query := `
		SELECT 
			total_nodes,
			healthy_nodes,
			offline_nodes,
			maintenance_nodes,
			total_cpu_capacity,
			total_memory_capacity_gb,
			total_disk_capacity_gb,
			used_cpu_cores,
			used_memory_gb,
			used_disk_gb,
			avg_cpu_usage_percent,
			avg_memory_usage_percent,
			avg_disk_usage_percent
		FROM v_infrastructure_health
	`

	var health HealthMetrics

	if err := s.db.QueryRowContext(ctx, query).Scan(
		&health.TotalNodes,
		&health.HealthyNodes,
		&health.OfflineNodes,
		&health.MaintenanceNodes,
		&health.TotalCPUCapacity,
		&health.TotalMemoryCapacity,
		&health.TotalDiskCapacity,
		&health.UsedCPU,
		&health.UsedMemory,
		&health.UsedDisk,
		&health.AverageCPUUsage,
		&health.AverageMemoryUsage,
		&health.AverageDiskUsage,
	); err != nil {
		s.logger.Error("Failed to query health metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to query health metrics: %w", err)
	}

	return &health, nil
}

// GetNodeHealth returns health metrics for a specific node
func (s *InfrastructureAdminService) GetNodeHealth(ctx context.Context, nodeID string) (*NodeHealth, error) {
	query := `
		SELECT 
			n.id,
			n.name,
			n.status,
			COALESCE((SELECT used_cpu_cores FROM node_resource_usage WHERE node_id = $1 ORDER BY recorded_at DESC LIMIT 1) / NULLIF(n.total_cpu_cores, 0) * 100, 0),
			COALESCE((SELECT used_memory_gb FROM node_resource_usage WHERE node_id = $1 ORDER BY recorded_at DESC LIMIT 1) / NULLIF(n.total_memory_gb, 0) * 100, 0),
			COALESCE((SELECT used_disk_gb FROM node_resource_usage WHERE node_id = $1 ORDER BY recorded_at DESC LIMIT 1) / NULLIF(n.total_disk_gb, 0) * 100, 0),
			n.last_heartbeat
		FROM infrastructure_nodes n
		WHERE n.id = $1
	`

	var health NodeHealth
	var lastHeartbeat sql.NullTime

	if err := s.db.QueryRowContext(ctx, query, nodeID).Scan(
		&health.NodeID,
		&health.NodeName,
		&health.Status,
		&health.CPUUsagePercent,
		&health.MemoryPercent,
		&health.DiskPercent,
		&lastHeartbeat,
	); err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	} else if err != nil {
		s.logger.Error("Failed to query node health", zap.Error(err))
		return nil, fmt.Errorf("failed to query node health: %w", err)
	}

	if lastHeartbeat.Valid {
		health.UpSince = &lastHeartbeat.Time
	}

	return &health, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// getNodeResourceUsage fetches the latest resource usage for a node
func (s *InfrastructureAdminService) getNodeResourceUsage(ctx context.Context, nodeID string, totalCPU, totalMemory, totalDisk float64) *ResourceUsage {
	query := `
		SELECT 
			used_cpu_cores,
			used_memory_gb,
			used_disk_gb,
			recorded_at
		FROM node_resource_usage
		WHERE node_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`

	var usage ResourceUsage
	var recordedAt sql.NullTime

	if err := s.db.QueryRowContext(ctx, query, nodeID).Scan(
		&usage.UsedCPUCores,
		&usage.UsedMemoryGB,
		&usage.UsedDiskGB,
		&recordedAt,
	); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		s.logger.Error("Failed to query resource usage", zap.Error(err))
		return nil
	}

	if recordedAt.Valid {
		usage.RecordedAt = recordedAt.Time
	}

	// Calculate percentages
	if totalCPU > 0 {
		usage.CPUUsagePercent = (usage.UsedCPUCores / totalCPU) * 100
	}
	if totalMemory > 0 {
		usage.MemoryPercent = (usage.UsedMemoryGB / totalMemory) * 100
	}
	if totalDisk > 0 {
		usage.DiskPercent = (usage.UsedDiskGB / totalDisk) * 100
	}

	return &usage
}
