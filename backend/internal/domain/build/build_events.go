package build

import (
	"time"

	"github.com/google/uuid"
)

// BuildEvent is the base interface for build domain events.
type BuildEvent interface {
	BuildID() uuid.UUID
	TenantID() uuid.UUID
	OccurredAt() time.Time
}

type BuildCreated struct {
	buildID    uuid.UUID
	tenantID   uuid.UUID
	manifest   BuildManifest
	occurredAt time.Time
}

func NewBuildCreated(buildID, tenantID uuid.UUID, manifest BuildManifest) *BuildCreated {
	return &BuildCreated{buildID: buildID, tenantID: tenantID, manifest: manifest, occurredAt: time.Now().UTC()}
}

func (e *BuildCreated) BuildID() uuid.UUID      { return e.buildID }
func (e *BuildCreated) TenantID() uuid.UUID     { return e.tenantID }
func (e *BuildCreated) Manifest() BuildManifest { return e.manifest }
func (e *BuildCreated) OccurredAt() time.Time   { return e.occurredAt }

type BuildStarted struct {
	buildID    uuid.UUID
	tenantID   uuid.UUID
	occurredAt time.Time
}

func NewBuildStarted(buildID, tenantID uuid.UUID) *BuildStarted {
	return &BuildStarted{buildID: buildID, tenantID: tenantID, occurredAt: time.Now().UTC()}
}

func (e *BuildStarted) BuildID() uuid.UUID    { return e.buildID }
func (e *BuildStarted) TenantID() uuid.UUID   { return e.tenantID }
func (e *BuildStarted) OccurredAt() time.Time { return e.occurredAt }

type BuildCompleted struct {
	buildID    uuid.UUID
	tenantID   uuid.UUID
	result     BuildResult
	occurredAt time.Time
}

func NewBuildCompleted(buildID, tenantID uuid.UUID, result BuildResult) *BuildCompleted {
	return &BuildCompleted{buildID: buildID, tenantID: tenantID, result: result, occurredAt: time.Now().UTC()}
}

func (e *BuildCompleted) BuildID() uuid.UUID    { return e.buildID }
func (e *BuildCompleted) TenantID() uuid.UUID   { return e.tenantID }
func (e *BuildCompleted) Result() BuildResult   { return e.result }
func (e *BuildCompleted) OccurredAt() time.Time { return e.occurredAt }

type BuildStatusUpdated struct {
	buildID    uuid.UUID
	tenantID   uuid.UUID
	status     string
	message    string
	metadata   map[string]interface{}
	occurredAt time.Time
}

func NewBuildStatusUpdated(buildID, tenantID uuid.UUID, status, message string, metadata map[string]interface{}) *BuildStatusUpdated {
	return &BuildStatusUpdated{buildID: buildID, tenantID: tenantID, status: status, message: message, metadata: metadata, occurredAt: time.Now().UTC()}
}

func (e *BuildStatusUpdated) BuildID() uuid.UUID               { return e.buildID }
func (e *BuildStatusUpdated) TenantID() uuid.UUID              { return e.tenantID }
func (e *BuildStatusUpdated) Status() string                   { return e.status }
func (e *BuildStatusUpdated) Message() string                  { return e.message }
func (e *BuildStatusUpdated) Metadata() map[string]interface{} { return e.metadata }
func (e *BuildStatusUpdated) OccurredAt() time.Time            { return e.occurredAt }
