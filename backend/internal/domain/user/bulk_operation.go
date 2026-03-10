package user

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors for bulk operations
var (
	ErrBulkOperationNotFound = errors.New("bulk operation not found")
	ErrBulkOperationFailed   = errors.New("bulk operation failed")
	ErrInvalidBulkOperation  = errors.New("invalid bulk operation")
)

// BulkOperationType represents the type of bulk operation
type BulkOperationType string

const (
	BulkOperationTypeImportUsers    BulkOperationType = "import_users"
	BulkOperationTypeUpdateUsers    BulkOperationType = "update_users"
	BulkOperationTypeDeleteUsers    BulkOperationType = "delete_users"
	BulkOperationTypeAssignRoles    BulkOperationType = "assign_roles"
	BulkOperationTypeRemoveRoles    BulkOperationType = "remove_roles"
	BulkOperationTypeChangeStatus   BulkOperationType = "change_status"
)

// BulkOperationStatus represents the status of a bulk operation
type BulkOperationStatus string

const (
	BulkOperationStatusPending   BulkOperationStatus = "pending"
	BulkOperationStatusRunning   BulkOperationStatus = "running"
	BulkOperationStatusCompleted BulkOperationStatus = "completed"
	BulkOperationStatusFailed    BulkOperationStatus = "failed"
	BulkOperationStatusCancelled BulkOperationStatus = "cancelled"
)

// BulkOperationResult represents the result of processing a single item in a bulk operation
type BulkOperationResult struct {
	ItemID      string                 `json:"item_id"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// BulkOperation represents a bulk operation aggregate
type BulkOperation struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	operationType BulkOperationType
	status        BulkOperationStatus
	initiatedBy   uuid.UUID
	totalItems    int
	processedItems int
	successfulItems int
	failedItems    int
	results       []BulkOperationResult
	metadata      map[string]interface{}
	errorMessage  string
	startedAt     *time.Time
	completedAt   *time.Time
	createdAt     time.Time
	updatedAt     time.Time
	version       int
}

// NewBulkOperation creates a new bulk operation
func NewBulkOperation(
	tenantID uuid.UUID,
	operationType BulkOperationType,
	initiatedBy uuid.UUID,
	totalItems int,
	metadata map[string]interface{},
) (*BulkOperation, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if initiatedBy == uuid.Nil {
		return nil, errors.New("initiated by user ID is required")
	}
	if totalItems <= 0 {
		return nil, errors.New("total items must be greater than 0")
	}

	now := time.Now().UTC()

	return &BulkOperation{
		id:             uuid.New(),
		tenantID:       tenantID,
		operationType:  operationType,
		status:         BulkOperationStatusPending,
		initiatedBy:    initiatedBy,
		totalItems:     totalItems,
		processedItems: 0,
		successfulItems: 0,
		failedItems:    0,
		results:        make([]BulkOperationResult, 0),
		metadata:       metadata,
		createdAt:      now,
		updatedAt:      now,
		version:        1,
	}, nil
}

// NewBulkOperationFromExisting creates a bulk operation from existing data
func NewBulkOperationFromExisting(
	id, tenantID uuid.UUID,
	operationType BulkOperationType,
	status BulkOperationStatus,
	initiatedBy uuid.UUID,
	totalItems, processedItems, successfulItems, failedItems int,
	resultsJSON string,
	metadata map[string]interface{},
	errorMessage string,
	startedAt, completedAt *time.Time,
	createdAt, updatedAt time.Time,
	version int,
) (*BulkOperation, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid bulk operation ID")
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if initiatedBy == uuid.Nil {
		return nil, errors.New("initiated by user ID is required")
	}

	var results []BulkOperationResult
	if resultsJSON != "" {
		if err := json.Unmarshal([]byte(resultsJSON), &results); err != nil {
			return nil, errors.New("invalid results JSON")
		}
	} else {
		results = make([]BulkOperationResult, 0)
	}

	return &BulkOperation{
		id:              id,
		tenantID:        tenantID,
		operationType:   operationType,
		status:          status,
		initiatedBy:     initiatedBy,
		totalItems:      totalItems,
		processedItems:  processedItems,
		successfulItems: successfulItems,
		failedItems:     failedItems,
		results:         results,
		metadata:        metadata,
		errorMessage:    errorMessage,
		startedAt:       startedAt,
		completedAt:     completedAt,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		version:         version,
	}, nil
}

// ID returns the bulk operation ID
func (b *BulkOperation) ID() uuid.UUID {
	return b.id
}

// TenantID returns the tenant ID
func (b *BulkOperation) TenantID() uuid.UUID {
	return b.tenantID
}

// OperationType returns the operation type
func (b *BulkOperation) OperationType() BulkOperationType {
	return b.operationType
}

// Status returns the operation status
func (b *BulkOperation) Status() BulkOperationStatus {
	return b.status
}

// InitiatedBy returns the user who initiated the operation
func (b *BulkOperation) InitiatedBy() uuid.UUID {
	return b.initiatedBy
}

// TotalItems returns the total number of items to process
func (b *BulkOperation) TotalItems() int {
	return b.totalItems
}

// ProcessedItems returns the number of processed items
func (b *BulkOperation) ProcessedItems() int {
	return b.processedItems
}

// SuccessfulItems returns the number of successful items
func (b *BulkOperation) SuccessfulItems() int {
	return b.successfulItems
}

// FailedItems returns the number of failed items
func (b *BulkOperation) FailedItems() int {
	return b.failedItems
}

// Progress returns the completion percentage (0-100)
func (b *BulkOperation) Progress() float64 {
	if b.totalItems == 0 {
		return 100.0
	}
	return float64(b.processedItems) / float64(b.totalItems) * 100.0
}

// IsComplete returns true if the operation is finished
func (b *BulkOperation) IsComplete() bool {
	return b.status == BulkOperationStatusCompleted ||
		b.status == BulkOperationStatusFailed ||
		b.status == BulkOperationStatusCancelled
}

// StartOperation marks the operation as started
func (b *BulkOperation) StartOperation() error {
	if b.status != BulkOperationStatusPending {
		return errors.New("operation is not in pending status")
	}

	now := time.Now().UTC()
	b.status = BulkOperationStatusRunning
	b.startedAt = &now
	b.updatedAt = now
	b.version++
	return nil
}

// AddResult adds a result for a processed item
func (b *BulkOperation) AddResult(result BulkOperationResult) {
	b.results = append(b.results, result)
	b.processedItems++

	if result.Success {
		b.successfulItems++
	} else {
		b.failedItems++
	}

	b.updatedAt = time.Now().UTC()
	b.version++
}

// CompleteOperation marks the operation as completed
func (b *BulkOperation) CompleteOperation() error {
	if b.status != BulkOperationStatusRunning {
		return errors.New("operation is not running")
	}

	now := time.Now().UTC()
	b.status = BulkOperationStatusCompleted
	b.completedAt = &now
	b.updatedAt = now
	b.version++
	return nil
}

// FailOperation marks the operation as failed with an error message
func (b *BulkOperation) FailOperation(errorMessage string) {
	now := time.Now().UTC()
	b.status = BulkOperationStatusFailed
	b.errorMessage = errorMessage
	b.completedAt = &now
	b.updatedAt = now
	b.version++
}

// CancelOperation marks the operation as cancelled
func (b *BulkOperation) CancelOperation() {
	now := time.Now().UTC()
	b.status = BulkOperationStatusCancelled
	b.completedAt = &now
	b.updatedAt = now
	b.version++
}

// Results returns the operation results
func (b *BulkOperation) Results() []BulkOperationResult {
	return b.results
}

// Metadata returns the operation metadata
func (b *BulkOperation) Metadata() map[string]interface{} {
	return b.metadata
}

// ErrorMessage returns the error message if the operation failed
func (b *BulkOperation) ErrorMessage() string {
	return b.errorMessage
}

// StartedAt returns when the operation started
func (b *BulkOperation) StartedAt() *time.Time {
	return b.startedAt
}

// CompletedAt returns when the operation completed
func (b *BulkOperation) CompletedAt() *time.Time {
	return b.completedAt
}

// CreatedAt returns the creation time
func (b *BulkOperation) CreatedAt() time.Time {
	return b.createdAt
}

// UpdatedAt returns the last update time
func (b *BulkOperation) UpdatedAt() time.Time {
	return b.updatedAt
}

// Version returns the version for optimistic locking
func (b *BulkOperation) Version() int {
	return b.version
}