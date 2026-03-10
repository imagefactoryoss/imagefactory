package buildnotification

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	ListTenantTriggerPreferences(ctx context.Context, tenantID uuid.UUID) ([]TenantTriggerPreference, error)
	UpsertTenantTriggerPreferences(ctx context.Context, tenantID, actorID uuid.UUID, prefs []TenantTriggerPreference) error
	ListProjectTriggerPreferences(ctx context.Context, tenantID, projectID uuid.UUID) ([]ProjectTriggerPreference, error)
	UpsertProjectTriggerPreferences(ctx context.Context, tenantID, projectID, actorID uuid.UUID, prefs []ProjectTriggerPreference) error
	DeleteProjectTriggerPreference(ctx context.Context, tenantID, projectID uuid.UUID, triggerID TriggerID) error
}
