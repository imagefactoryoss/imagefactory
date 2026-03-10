package build

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// BuildHistory is a value object representing a completed build record
// It is immutable once created and used for ETA learning and performance analysis
type BuildHistory struct {
	id          uuid.UUID
	buildID     uuid.UUID
	tenantID    uuid.UUID
	projectID   uuid.UUID
	buildMethod string
	workerID    *uuid.UUID
	duration    time.Duration
	success     bool
	startedAt   *time.Time
	completedAt time.Time
	createdAt   time.Time
}

// New creates a new BuildHistory value object
func New(
	id uuid.UUID,
	buildID uuid.UUID,
	tenantID uuid.UUID,
	projectID uuid.UUID,
	buildMethod string,
	workerID *uuid.UUID,
	duration time.Duration,
	success bool,
	startedAt *time.Time,
	completedAt time.Time,
) (*BuildHistory, error) {
	if id == uuid.Nil {
		return nil, errors.New("history id cannot be nil")
	}
	if buildID == uuid.Nil {
		return nil, errors.New("build id cannot be nil")
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant id cannot be nil")
	}
	if projectID == uuid.Nil {
		return nil, errors.New("project id cannot be nil")
	}
	if buildMethod == "" {
		return nil, errors.New("build method cannot be empty")
	}
	if !isValidBuildMethod(buildMethod) {
		return nil, errors.New("invalid build method")
	}
	if duration <= 0 {
		return nil, errors.New("duration must be positive")
	}

	return &BuildHistory{
		id:          id,
		buildID:     buildID,
		tenantID:    tenantID,
		projectID:   projectID,
		buildMethod: buildMethod,
		workerID:    workerID,
		duration:    duration,
		success:     success,
		startedAt:   startedAt,
		completedAt: completedAt,
		createdAt:   time.Now().UTC(),
	}, nil
}

// === Accessors ===

func (bh *BuildHistory) ID() uuid.UUID {
	return bh.id
}

func (bh *BuildHistory) BuildID() uuid.UUID {
	return bh.buildID
}

func (bh *BuildHistory) TenantID() uuid.UUID {
	return bh.tenantID
}

func (bh *BuildHistory) ProjectID() uuid.UUID {
	return bh.projectID
}

func (bh *BuildHistory) BuildMethod() string {
	return bh.buildMethod
}

func (bh *BuildHistory) WorkerID() *uuid.UUID {
	return bh.workerID
}

func (bh *BuildHistory) Duration() time.Duration {
	return bh.duration
}

func (bh *BuildHistory) DurationSeconds() int {
	return int(bh.duration.Seconds())
}

func (bh *BuildHistory) Success() bool {
	return bh.success
}

func (bh *BuildHistory) StartedAt() *time.Time {
	return bh.startedAt
}

func (bh *BuildHistory) CompletedAt() time.Time {
	return bh.completedAt
}

func (bh *BuildHistory) CreatedAt() time.Time {
	return bh.createdAt
}

// === Value Object Equality ===

// Equals checks if this BuildHistory equals another
func (bh *BuildHistory) Equals(other *BuildHistory) bool {
	if other == nil {
		return false
	}
	return bh.id == other.id
}

// === Helper Functions ===

func isValidBuildMethod(method string) bool {
	switch method {
	case "kaniko", "buildx", "container", "paketo", "packer":
		return true
	default:
		return false
	}
}
