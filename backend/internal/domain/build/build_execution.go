package build

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus represents the status of a build execution
type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "pending"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionSuccess   ExecutionStatus = "success"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCancelled ExecutionStatus = "cancelled"
)

// LogLevel represents the severity level of an execution log
type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// BuildExecution represents a single execution of a build configuration
type BuildExecution struct {
	ID              uuid.UUID
	BuildID         uuid.UUID
	ConfigID        uuid.UUID
	Status          ExecutionStatus
	StartedAt       *time.Time
	CompletedAt     *time.Time
	DurationSeconds *int
	Output          string
	ErrorMessage    string
	Artifacts       json.RawMessage
	Metadata        json.RawMessage
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ExecutionLog represents a single log entry from a build execution
type ExecutionLog struct {
	ID          uuid.UUID
	ExecutionID uuid.UUID
	Timestamp   time.Time
	Level       LogLevel
	Message     string
	Metadata    json.RawMessage
}

// StartBuildRequest is the request to start a new build execution
type StartBuildRequest struct {
	ConfigID uuid.UUID `json:"config_id" binding:"required"`
}

// StartBuildResponse is the response when a build execution is started
type StartBuildResponse struct {
	ID        uuid.UUID       `json:"id"`
	BuildID   uuid.UUID       `json:"build_id"`
	ConfigID  uuid.UUID       `json:"config_id"`
	Status    ExecutionStatus `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

// ExecutionStatusResponse represents the current status of a build execution
type ExecutionStatusResponse struct {
	ID              uuid.UUID       `json:"id"`
	BuildID         uuid.UUID       `json:"build_id"`
	ConfigID        uuid.UUID       `json:"config_id"`
	Status          ExecutionStatus `json:"status"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	DurationSeconds *int            `json:"duration_seconds,omitempty"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ExecutionLogResponse represents a single log entry
type ExecutionLogResponse struct {
	ID          uuid.UUID       `json:"id"`
	ExecutionID uuid.UUID       `json:"execution_id"`
	Timestamp   time.Time       `json:"timestamp"`
	Level       LogLevel        `json:"level"`
	Message     string          `json:"message"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// ExecutionListResponse represents a paginated list of executions
type ExecutionListResponse struct {
	Executions []ExecutionStatusResponse `json:"executions"`
	Total      int64                     `json:"total"`
	Limit      int                       `json:"limit"`
	Offset     int                       `json:"offset"`
}

// LogListResponse represents a paginated list of logs
type LogListResponse struct {
	Logs   []ExecutionLogResponse `json:"logs"`
	Total  int64                  `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// ExecutionOutputMessage represents a single output item from a build execution
type ExecutionOutputMessage struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Status    *ExecutionStatus
	Artifact  *ExecutionArtifact
}

// ExecutionArtifact represents a build artifact produced during execution
type ExecutionArtifact struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Domain errors for execution
var (
	ErrExecutionNotFound       = errors.New("execution not found")
	ErrBuildAlreadyExecuting   = errors.New("build is already executing")
	ErrInvalidExecutionStatus  = errors.New("invalid execution status")
	ErrUnauthorizedExecution   = errors.New("unauthorized to access this execution")
	ErrExecutionNotCancellable = errors.New("execution cannot be cancelled in current state")
	ErrExecutionNotRetryable   = errors.New("execution cannot be retried in current state")
	ErrInvalidLogLevel         = errors.New("invalid log level")
	ErrExecutionNotRunning     = errors.New("execution is not running")
)

// ToResponse converts a BuildExecution to ExecutionStatusResponse
func (e *BuildExecution) ToResponse() ExecutionStatusResponse {
	return ExecutionStatusResponse{
		ID:              e.ID,
		BuildID:         e.BuildID,
		ConfigID:        e.ConfigID,
		Status:          e.Status,
		StartedAt:       e.StartedAt,
		CompletedAt:     e.CompletedAt,
		DurationSeconds: e.DurationSeconds,
		ErrorMessage:    e.ErrorMessage,
		Metadata:        e.Metadata,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

// ToLogResponse converts an ExecutionLog to ExecutionLogResponse
func (l *ExecutionLog) ToResponse() ExecutionLogResponse {
	return ExecutionLogResponse{
		ID:          l.ID,
		ExecutionID: l.ExecutionID,
		Timestamp:   l.Timestamp,
		Level:       l.Level,
		Message:     l.Message,
		Metadata:    l.Metadata,
	}
}

// IsTerminalStatus returns true if the status is a terminal state
func (s ExecutionStatus) IsTerminalStatus() bool {
	return s == ExecutionSuccess || s == ExecutionFailed || s == ExecutionCancelled
}

// CanTransitionTo checks if a transition from current status to target status is valid
func (s ExecutionStatus) CanTransitionTo(target ExecutionStatus) bool {
	// From pending, can go to running or cancelled
	if s == ExecutionPending {
		return target == ExecutionRunning || target == ExecutionCancelled
	}
	// From running, can go to success, failed, or cancelled
	if s == ExecutionRunning {
		return target == ExecutionSuccess || target == ExecutionFailed || target == ExecutionCancelled
	}
	// Terminal states cannot transition
	return false
}

// Scan implements sql.Scanner for database scanning
func (s *ExecutionStatus) Scan(value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return errors.New("cannot scan non-string value into ExecutionStatus")
	}
	*s = ExecutionStatus(strVal)
	return nil
}

// Value implements driver.Valuer for database writing
func (s ExecutionStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements sql.Scanner for LogLevel
func (l *LogLevel) Scan(value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return errors.New("cannot scan non-string value into LogLevel")
	}
	*l = LogLevel(strVal)
	return nil
}

// Value implements driver.Valuer for LogLevel
func (l LogLevel) Value() (driver.Value, error) {
	return string(l), nil
}
