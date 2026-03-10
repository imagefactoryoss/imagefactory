package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// BulkOperationsHandler handles bulk operations HTTP requests
type BulkOperationsHandler struct {
	userService  user.BulkOperationService
	auditService *audit.Service
	logger       *zap.Logger
}

// NewBulkOperationsHandler creates a new bulk operations handler
func NewBulkOperationsHandler(userService user.BulkOperationService, auditService *audit.Service, logger *zap.Logger) *BulkOperationsHandler {
	return &BulkOperationsHandler{
		userService:  userService,
		auditService: auditService,
		logger:       logger,
	}
}

// StartBulkImportRequest represents a bulk import request
type StartBulkImportRequest struct {
	TenantID    string                   `json:"tenant_id" validate:"required,uuid"`
	Users       []user.BulkImportUser    `json:"users" validate:"required,min=1"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

// StartBulkImportResponse represents a bulk import response
type StartBulkImportResponse struct {
	Operation BulkOperationResponse `json:"operation"`
}

// BulkOperationResponse represents a bulk operation response
type BulkOperationResponse struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	OperationType   string                 `json:"operation_type"`
	Status          string                 `json:"status"`
	InitiatedBy     string                 `json:"initiated_by"`
	TotalItems      int                    `json:"total_items"`
	ProcessedItems  int                    `json:"processed_items"`
	SuccessfulItems int                    `json:"successful_items"`
	FailedItems     int                    `json:"failed_items"`
	Progress        float64                `json:"progress"`
	Results         []BulkOperationResultResponse `json:"results,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	StartedAt       *string                `json:"started_at,omitempty"`
	CompletedAt     *string                `json:"completed_at,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// BulkOperationResultResponse represents a bulk operation result response
type BulkOperationResultResponse struct {
	ItemID      string    `json:"item_id"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ProcessedAt string    `json:"processed_at"`
}

// ListBulkOperationsResponse represents a list of bulk operations response
type ListBulkOperationsResponse struct {
	Operations []BulkOperationResponse `json:"operations"`
	Total      int                     `json:"total"`
}

// StartBulkImport handles POST /bulk-operations/import
func (h *BulkOperationsHandler) StartBulkImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Get user context from middleware
	userCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	var req StartBulkImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode bulk import request", zap.Error(err))
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	// Parse tenant ID
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid tenant ID"))
		return
	}

	// Start bulk import operation
	operation, err := h.userService.StartBulkImport(
		r.Context(),
		tenantID,
		userCtx.UserID,
		req.Users,
		req.Metadata,
	)
	if err != nil {
		h.logger.Error("Failed to start bulk import",
			zap.String("tenant_id", req.TenantID),
			zap.Int("user_count", len(req.Users)),
			zap.Error(err),
		)
		WriteError(w, r.Context(), InternalServer("Failed to start bulk import").WithCause(err))
		return
	}

	response := StartBulkImportResponse{
		Operation: h.buildBulkOperationResponse(operation),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Bulk import operation started",
		zap.String("operation_id", operation.ID().String()),
		zap.String("tenant_id", req.TenantID),
		zap.Int("user_count", len(req.Users)),
		zap.String("initiated_by", userCtx.UserID.String()),
	)

	// Audit bulk import start
	if h.auditService != nil {
		h.auditService.LogUserAction(r.Context(), tenantID, userCtx.UserID,
			audit.AuditEventBulkImport, "bulk_operations", "start_import",
			"Bulk import operation started",
			map[string]interface{}{
				"operation_id": operation.ID().String(),
				"tenant_id":    tenantID.String(),
				"user_count":   len(req.Users),
			})
	}
}

// GetBulkOperation handles GET /bulk-operations/{id}
func (h *BulkOperationsHandler) GetBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	operationIDStr := chi.URLParam(r, "id")
	if operationIDStr == "" {
		WriteError(w, r.Context(), BadRequest("Operation ID is required"))
		return
	}

	operationID, err := uuid.Parse(operationIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid operation ID"))
		return
	}

	operation, err := h.userService.GetBulkOperationStatus(r.Context(), operationID)
	if err != nil {
		h.logger.Error("Failed to get bulk operation",
			zap.String("operation_id", operationID.String()),
			zap.Error(err),
		)

		switch err {
		case user.ErrBulkOperationNotFound:
			WriteError(w, r.Context(), NotFound("Bulk operation not found"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to get bulk operation").WithCause(err))
		}
		return
	}

	response := h.buildBulkOperationResponse(operation)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListBulkOperations handles GET /bulk-operations
func (h *BulkOperationsHandler) ListBulkOperations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	// Parse query parameters
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		WriteError(w, r.Context(), BadRequest("Tenant ID is required"))
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid tenant ID"))
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0 // default

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

	operations, err := h.userService.ListBulkOperations(r.Context(), tenantID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list bulk operations",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		WriteError(w, r.Context(), InternalServer("Failed to list bulk operations").WithCause(err))
		return
	}

	response := ListBulkOperationsResponse{
		Operations: make([]BulkOperationResponse, len(operations)),
		Total:      len(operations), // TODO: Implement proper total count with pagination
	}

	for i, op := range operations {
		response.Operations[i] = h.buildBulkOperationResponse(op)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CancelBulkOperation handles DELETE /bulk-operations/{id}
func (h *BulkOperationsHandler) CancelBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, r.Context(), Forbidden("Method not allowed"))
		return
	}

	operationIDStr := chi.URLParam(r, "id")
	if operationIDStr == "" {
		WriteError(w, r.Context(), BadRequest("Operation ID is required"))
		return
	}

	operationID, err := uuid.Parse(operationIDStr)
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid operation ID"))
		return
	}

	err = h.userService.CancelBulkOperation(r.Context(), operationID)
	if err != nil {
		h.logger.Error("Failed to cancel bulk operation",
			zap.String("operation_id", operationID.String()),
			zap.Error(err),
		)

		switch err {
		case user.ErrBulkOperationNotFound:
			WriteError(w, r.Context(), NotFound("Bulk operation not found"))
		default:
			WriteError(w, r.Context(), InternalServer("Failed to cancel bulk operation").WithCause(err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Bulk operation cancelled successfully"})

	h.logger.Info("Bulk operation cancelled",
		zap.String("operation_id", operationID.String()),
	)
}

// buildBulkOperationResponse converts a domain bulk operation to a response
func (h *BulkOperationsHandler) buildBulkOperationResponse(operation *user.BulkOperation) BulkOperationResponse {
	response := BulkOperationResponse{
		ID:              operation.ID().String(),
		TenantID:        operation.TenantID().String(),
		OperationType:   string(operation.OperationType()),
		Status:          string(operation.Status()),
		InitiatedBy:     operation.InitiatedBy().String(),
		TotalItems:      operation.TotalItems(),
		ProcessedItems:  operation.ProcessedItems(),
		SuccessfulItems: operation.SuccessfulItems(),
		FailedItems:     operation.FailedItems(),
		Progress:        operation.Progress(),
		ErrorMessage:    operation.ErrorMessage(),
		CreatedAt:       operation.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       operation.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}

	if startedAt := operation.StartedAt(); startedAt != nil {
		startedAtStr := startedAt.Format("2006-01-02T15:04:05Z")
		response.StartedAt = &startedAtStr
	}

	if completedAt := operation.CompletedAt(); completedAt != nil {
		completedAtStr := completedAt.Format("2006-01-02T15:04:05Z")
		response.CompletedAt = &completedAtStr
	}

	// Include results if operation is complete
	if operation.IsComplete() {
		results := operation.Results()
		response.Results = make([]BulkOperationResultResponse, len(results))
		for i, result := range results {
			response.Results[i] = BulkOperationResultResponse{
				ItemID:      result.ItemID,
				Success:     result.Success,
				Error:       result.Error,
				Data:        result.Data,
				ProcessedAt: result.ProcessedAt.Format("2006-01-02T15:04:05Z"),
			}
		}
	}

	return response
}