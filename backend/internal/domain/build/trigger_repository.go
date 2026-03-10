package build

import (
	"context"

	"github.com/google/uuid"
)

// TriggerRepository defines trigger persistence operations
type TriggerRepository interface {
	// Create or update a trigger
	SaveTrigger(ctx context.Context, trigger *BuildTrigger) error

	// Read operations
	GetTrigger(ctx context.Context, triggerID uuid.UUID) (*BuildTrigger, error)
	GetTriggersByBuild(ctx context.Context, buildID uuid.UUID) ([]*BuildTrigger, error)
	GetTriggersByProject(ctx context.Context, projectID uuid.UUID) ([]*BuildTrigger, error)
	GetActiveScheduledTriggers(ctx context.Context, tenantID uuid.UUID) ([]*BuildTrigger, error)

	// Update
	UpdateTrigger(ctx context.Context, trigger *BuildTrigger) error

	// Delete
	DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error
}
