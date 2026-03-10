package rest

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

// InfrastructureHandler handles infrastructure management HTTP requests
type InfrastructureHandler struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewInfrastructureHandler creates a new infrastructure handler
func NewInfrastructureHandler(db *sql.DB, logger *zap.Logger) *InfrastructureHandler {
	return &InfrastructureHandler{
		db:     db,
		logger: logger,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// InfrastructureNodeResponse represents a single infrastructure node
type InfrastructureNodeResponse struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	TotalCPUCores    float64           `json:"total_cpu_cores"`
	TotalMemoryGB    float64           `json:"total_memory_gb"`
	TotalDiskGB      float64           `json:"total_disk_gb"`
	LastHeartbeat    *string           `json:"last_heartbeat"`
	MaintenanceMode  bool              `json:"maintenance_mode"`
	Labels           map[string]string `json:"labels"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
	CurrentResources *ResourceUsage    `json:"current_resources"`
}

// ResourceUsage represents current resource usage for a node
type ResourceUsage struct {
	UsedCPUCores    float64 `json:"used_cpu_cores"`
	UsedMemoryGB    float64 `json:"used_memory_gb"`
	UsedDiskGB      float64 `json:"used_disk_gb"`
	RecordedAt      string  `json:"recorded_at"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemoryPercent   float64 `json:"memory_percent"`
	DiskPercent     float64 `json:"disk_percent"`
}

// GetNodesResponse represents the response for listing nodes
type GetNodesResponse struct {
	Nodes   []InfrastructureNodeResponse `json:"nodes"`
	Total   int                          `json:"total"`
	Limit   int                          `json:"limit"`
	Offset  int                          `json:"offset"`
	HasMore bool                         `json:"has_more"`
}

// CreateNodeRequest represents a request to create a node
type CreateNodeRequest struct {
	Name        string            `json:"name" validate:"required,min=3"`
	TotalCPU    float64           `json:"total_cpu_cores" validate:"required,min=1"`
	TotalMemory float64           `json:"total_memory_gb" validate:"required,min=1"`
	TotalDisk   float64           `json:"total_disk_gb" validate:"required,min=1"`
	Labels      map[string]string `json:"labels"`
}

// UpdateNodeRequest represents a request to update a node
type UpdateNodeRequest struct {
	Name            *string           `json:"name"`
	Status          *string           `json:"status" validate:"omitempty,oneof=ready offline maintenance"`
	MaintenanceMode *bool             `json:"maintenance_mode"`
	Labels          map[string]string `json:"labels"`
}

// InfrastructureHealthResponse represents infrastructure health metrics
type InfrastructureHealthResponse struct {
	TotalNodes          int                 `json:"total_nodes"`
	HealthyNodes        int                 `json:"healthy_nodes"`
	OfflineNodes        int                 `json:"offline_nodes"`
	MaintenanceNodes    int                 `json:"maintenance_nodes"`
	TotalCPUCapacity    float64             `json:"total_cpu_capacity"`
	TotalMemoryCapacity float64             `json:"total_memory_capacity_gb"`
	TotalDiskCapacity   float64             `json:"total_disk_capacity_gb"`
	UsedCPU             float64             `json:"used_cpu_cores"`
	UsedMemory          float64             `json:"used_memory_gb"`
	UsedDisk            float64             `json:"used_disk_gb"`
	AverageCPUUsage     float64             `json:"average_cpu_usage_percent"`
	AverageMemoryUsage  float64             `json:"average_memory_usage_percent"`
	AverageDiskUsage    float64             `json:"average_disk_usage_percent"`
	NodeHealthBreakdown []NodeHealthSummary `json:"node_health_breakdown"`
}

// NodeHealthSummary represents health summary for a single node
type NodeHealthSummary struct {
	NodeID             string  `json:"node_id"`
	NodeName           string  `json:"node_name"`
	Status             string  `json:"status"`
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	UpSince            *string `json:"up_since"`
}

// ResourceHistoryResponse represents resource usage history
type ResourceHistoryResponse struct {
	NodeID   string           `json:"node_id"`
	NodeName string           `json:"node_name"`
	History  []ResourceRecord `json:"history"`
}

// ResourceRecord represents a single resource usage record
type ResourceRecord struct {
	RecordedAt      string  `json:"recorded_at"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemoryPercent   float64 `json:"memory_percent"`
	DiskPercent     float64 `json:"disk_percent"`
}

// ============================================================================
// Handler Methods
// ============================================================================

// GetNodes returns a paginated list of infrastructure nodes
// GET /api/v1/admin/infrastructure/nodes?limit=20&offset=0&status=ready
func (h *InfrastructureHandler) GetNodes(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	scopeTenantID, allTenants, scopeStatus, scopeMessage := resolveTenantScopeFromRequest(r, authCtx, true)
	if scopeStatus != 0 {
		h.respondError(w, scopeStatus, scopeMessage)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	status := r.URL.Query().Get("status")

	limit := 20
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Count total nodes
	countQuery := `SELECT COUNT(*) FROM infrastructure_nodes WHERE 1=1`
	countArgs := []interface{}{}
	argIndex := 1

	if !allTenants {
		countQuery += ` AND tenant_id = $` + strconv.Itoa(argIndex)
		countArgs = append(countArgs, scopeTenantID)
		argIndex++
	}

	if status != "" {
		countQuery += ` AND status = $` + strconv.Itoa(argIndex)
		countArgs = append(countArgs, status)
		argIndex++
	}

	var total int
	if err := h.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		h.logger.Error("Failed to count nodes", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to count nodes")
		return
	}

	// Get nodes
	query := `
		SELECT 
			id,
			tenant_id,
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
	args := []interface{}{}
	argIndex = 1

	if !allTenants {
		query += ` AND tenant_id = $` + strconv.Itoa(argIndex)
		args = append(args, scopeTenantID)
		argIndex++
	}

	if status != "" {
		query += ` AND status = $` + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.logger.Error("Failed to query nodes", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve nodes")
		return
	}
	defer rows.Close()

	nodes := []InfrastructureNodeResponse{}
	for rows.Next() {
		var node InfrastructureNodeResponse
		var lastHeartbeat, createdAt, updatedAt sql.NullTime
		var labels sql.NullString
		var tenantID uuid.UUID

		if err := rows.Scan(
			&node.ID,
			&tenantID,
			&node.Name,
			&node.Status,
			&node.TotalCPUCores,
			&node.TotalMemoryGB,
			&node.TotalDiskGB,
			&lastHeartbeat,
			&node.MaintenanceMode,
			&labels,
			&createdAt,
			&updatedAt,
		); err != nil {
			h.logger.Error("Failed to scan node", zap.Error(err))
			continue
		}

		if lastHeartbeat.Valid {
			ts := lastHeartbeat.Time.Format("2006-01-02T15:04:05Z07:00")
			node.LastHeartbeat = &ts
		}

		node.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
		node.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")

		// Parse labels JSON
		if labels.Valid {
			json.Unmarshal([]byte(labels.String), &node.Labels)
		}

		// Get current resource usage
		node.CurrentResources = h.getNodeResourceUsage(node.ID, node.TotalCPUCores, node.TotalMemoryGB, node.TotalDiskGB)

		nodes = append(nodes, node)
	}

	response := GetNodesResponse{
		Nodes:   nodes,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetNode returns a specific infrastructure node
// GET /api/v1/admin/infrastructure/nodes/:id
func (h *InfrastructureHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	scopeTenantID, allTenants, scopeStatus, scopeMessage := resolveTenantScopeFromRequest(r, authCtx, true)
	if scopeStatus != 0 {
		h.respondError(w, scopeStatus, scopeMessage)
		return
	}

	nodeID := chi.URLParam(r, "id")

	var node InfrastructureNodeResponse
	var lastHeartbeat, createdAt, updatedAt sql.NullTime
	var labels sql.NullString

	query := `
		SELECT 
			id,
			tenant_id,
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
		WHERE id = $1
	`
	args := []interface{}{nodeID}
	if !allTenants {
		query += ` AND tenant_id = $2`
		args = append(args, scopeTenantID)
	}

	var tenantID uuid.UUID
	if err := h.db.QueryRow(query, args...).Scan(
		&node.ID,
		&tenantID,
		&node.Name,
		&node.Status,
		&node.TotalCPUCores,
		&node.TotalMemoryGB,
		&node.TotalDiskGB,
		&lastHeartbeat,
		&node.MaintenanceMode,
		&labels,
		&createdAt,
		&updatedAt,
	); err == sql.ErrNoRows {
		h.respondError(w, http.StatusNotFound, "Node not found")
		return
	} else if err != nil {
		h.logger.Error("Failed to query node", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve node")
		return
	}

	if lastHeartbeat.Valid {
		ts := lastHeartbeat.Time.Format("2006-01-02T15:04:05Z07:00")
		node.LastHeartbeat = &ts
	}

	node.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	node.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")

	if labels.Valid {
		json.Unmarshal([]byte(labels.String), &node.Labels)
	}

	node.CurrentResources = h.getNodeResourceUsage(node.ID, node.TotalCPUCores, node.TotalMemoryGB, node.TotalDiskGB)

	h.respondJSON(w, http.StatusOK, node)
}

// CreateNode creates a new infrastructure node
// POST /api/v1/admin/infrastructure/nodes
func (h *InfrastructureHandler) CreateNode(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	scopeTenantID, _, scopeStatus, scopeMessage := resolveTenantScopeFromRequest(r, authCtx, false)
	if scopeStatus != 0 {
		h.respondError(w, scopeStatus, scopeMessage)
		return
	}

	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Name == "" || len(req.Name) < 3 {
		h.respondError(w, http.StatusBadRequest, "name must be at least 3 characters")
		return
	}

	if req.TotalCPU < 1 || req.TotalMemory < 1 || req.TotalDisk < 1 {
		h.respondError(w, http.StatusBadRequest, "CPU, memory, and disk must be at least 1")
		return
	}

	nodeID := uuid.New().String()
	now := time.Now()

	labelsJSON := "{}"
	if req.Labels != nil {
		if data, err := json.Marshal(req.Labels); err == nil {
			labelsJSON = string(data)
		}
	}

	insertQuery := `
		INSERT INTO infrastructure_nodes 
		(id, tenant_id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, maintenance_mode, labels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11)
		RETURNING id, tenant_id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, last_heartbeat, maintenance_mode, labels, created_at, updated_at
	`

	var node InfrastructureNodeResponse
	var lastHeartbeat, createdAt, updatedAt sql.NullTime
	var labels sql.NullString

	var createdTenantID uuid.UUID
	if err := h.db.QueryRow(
		insertQuery,
		nodeID, scopeTenantID, req.Name, "ready", req.TotalCPU, req.TotalMemory, req.TotalDisk, false, labelsJSON, now, now,
	).Scan(
		&node.ID, &createdTenantID, &node.Name, &node.Status, &node.TotalCPUCores, &node.TotalMemoryGB, &node.TotalDiskGB,
		&lastHeartbeat, &node.MaintenanceMode, &labels, &createdAt, &updatedAt,
	); err != nil {
		h.logger.Error("Failed to create node", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to create node")
		return
	}

	node.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	node.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	if labels.Valid {
		json.Unmarshal([]byte(labels.String), &node.Labels)
	}

	h.respondJSON(w, http.StatusCreated, node)
}

// UpdateNode updates an infrastructure node
// PUT /api/v1/admin/infrastructure/nodes/:id
func (h *InfrastructureHandler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	scopeTenantID, allTenants, scopeStatus, scopeMessage := resolveTenantScopeFromRequest(r, authCtx, true)
	if scopeStatus != 0 {
		h.respondError(w, scopeStatus, scopeMessage)
		return
	}

	nodeID := chi.URLParam(r, "id")

	var req UpdateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

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

	updateQuery += ` WHERE id = $` + strconv.Itoa(argIndex)
	args = append(args, nodeID)
	argIndex++
	if !allTenants {
		updateQuery += ` AND tenant_id = $` + strconv.Itoa(argIndex)
		args = append(args, scopeTenantID)
		argIndex++
	}
	updateQuery += ` RETURNING id, tenant_id, name, status, total_cpu_cores, total_memory_gb, total_disk_gb, last_heartbeat, maintenance_mode, labels, created_at, updated_at`

	var node InfrastructureNodeResponse
	var lastHeartbeat, createdAt, updatedAt sql.NullTime
	var labels sql.NullString

	var updatedTenantID uuid.UUID
	if err := h.db.QueryRow(updateQuery, args...).Scan(
		&node.ID, &updatedTenantID, &node.Name, &node.Status, &node.TotalCPUCores, &node.TotalMemoryGB, &node.TotalDiskGB,
		&lastHeartbeat, &node.MaintenanceMode, &labels, &createdAt, &updatedAt,
	); err == sql.ErrNoRows {
		h.respondError(w, http.StatusNotFound, "Node not found")
		return
	} else if err != nil {
		h.logger.Error("Failed to update node", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to update node")
		return
	}

	node.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	node.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	if labels.Valid {
		json.Unmarshal([]byte(labels.String), &node.Labels)
	}

	h.respondJSON(w, http.StatusOK, node)
}

// DeleteNode deletes an infrastructure node
// DELETE /api/v1/admin/infrastructure/nodes/:id
func (h *InfrastructureHandler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	scopeTenantID, allTenants, scopeStatus, scopeMessage := resolveTenantScopeFromRequest(r, authCtx, true)
	if scopeStatus != 0 {
		h.respondError(w, scopeStatus, scopeMessage)
		return
	}

	nodeID := chi.URLParam(r, "id")

	deleteQuery := `DELETE FROM infrastructure_nodes WHERE id = $1`
	args := []interface{}{nodeID}
	if !allTenants {
		deleteQuery += ` AND tenant_id = $2`
		args = append(args, scopeTenantID)
	}

	result, err := h.db.Exec(deleteQuery, args...)
	if err != nil {
		h.logger.Error("Failed to delete node", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete node")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		h.logger.Error("Failed to get rows affected", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete node")
		return
	}

	if rowsAffected == 0 {
		h.respondError(w, http.StatusNotFound, "Node not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetInfrastructureHealth returns overall infrastructure health metrics
// GET /api/v1/admin/infrastructure/health
func (h *InfrastructureHandler) GetInfrastructureHealth(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	scopeTenantID, allTenants, scopeStatus, scopeMessage := resolveTenantScopeFromRequest(r, authCtx, true)
	if scopeStatus != 0 {
		h.respondError(w, scopeStatus, scopeMessage)
		return
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
	args := []interface{}{}
	if !allTenants {
		query += ` WHERE tenant_id = $1`
		args = append(args, scopeTenantID)
	}

	var resp InfrastructureHealthResponse

	if err := h.db.QueryRow(query, args...).Scan(
		&resp.TotalNodes,
		&resp.HealthyNodes,
		&resp.OfflineNodes,
		&resp.MaintenanceNodes,
		&resp.TotalCPUCapacity,
		&resp.TotalMemoryCapacity,
		&resp.TotalDiskCapacity,
		&resp.UsedCPU,
		&resp.UsedMemory,
		&resp.UsedDisk,
		&resp.AverageCPUUsage,
		&resp.AverageMemoryUsage,
		&resp.AverageDiskUsage,
	); err != nil {
		h.logger.Error("Failed to query health metrics", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve health metrics")
		return
	}

	// Get node health breakdown
	nodeQuery := `
		SELECT 
			id,
			name,
			status,
			COALESCE((SELECT used_cpu_cores FROM node_resource_usage WHERE node_id = n.id ORDER BY recorded_at DESC LIMIT 1) / NULLIF(n.total_cpu_cores, 0) * 100, 0),
			COALESCE((SELECT used_memory_gb FROM node_resource_usage WHERE node_id = n.id ORDER BY recorded_at DESC LIMIT 1) / NULLIF(n.total_memory_gb, 0) * 100, 0),
			COALESCE((SELECT used_disk_gb FROM node_resource_usage WHERE node_id = n.id ORDER BY recorded_at DESC LIMIT 1) / NULLIF(n.total_disk_gb, 0) * 100, 0),
			last_heartbeat
		FROM infrastructure_nodes n
		WHERE 1=1
	`
	nodeArgs := []interface{}{}
	if !allTenants {
		nodeQuery += ` AND tenant_id = $1`
		nodeArgs = append(nodeArgs, scopeTenantID)
	}
	nodeQuery += `
		ORDER BY created_at
	`

	rows, err := h.db.Query(nodeQuery, nodeArgs...)
	if err != nil {
		h.logger.Error("Failed to query node health", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve node health")
		return
	}
	defer rows.Close()

	resp.NodeHealthBreakdown = []NodeHealthSummary{}
	for rows.Next() {
		var summary NodeHealthSummary
		var lastHeartbeat sql.NullTime

		if err := rows.Scan(
			&summary.NodeID,
			&summary.NodeName,
			&summary.Status,
			&summary.CPUUsagePercent,
			&summary.MemoryUsagePercent,
			&summary.DiskUsagePercent,
			&lastHeartbeat,
		); err != nil {
			h.logger.Error("Failed to scan node health", zap.Error(err))
			continue
		}

		if lastHeartbeat.Valid {
			ts := lastHeartbeat.Time.Format("2006-01-02T15:04:05Z07:00")
			summary.UpSince = &ts
		}

		resp.NodeHealthBreakdown = append(resp.NodeHealthBreakdown, summary)
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Helper Methods
// ============================================================================

// getNodeResourceUsage fetches the latest resource usage for a node
func (h *InfrastructureHandler) getNodeResourceUsage(nodeID string, totalCPU, totalMemory, totalDisk float64) *ResourceUsage {
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

	if err := h.db.QueryRow(query, nodeID).Scan(
		&usage.UsedCPUCores,
		&usage.UsedMemoryGB,
		&usage.UsedDiskGB,
		&recordedAt,
	); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		h.logger.Error("Failed to query resource usage", zap.Error(err))
		return nil
	}

	if recordedAt.Valid {
		usage.RecordedAt = recordedAt.Time.Format("2006-01-02T15:04:05Z07:00")
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

// respondJSON writes a JSON response
func (h *InfrastructureHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// respondError writes an error response
func (h *InfrastructureHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}
