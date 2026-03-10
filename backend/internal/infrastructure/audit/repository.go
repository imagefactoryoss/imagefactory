package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for audit event persistence
type Repository interface {
	SaveEvent(ctx context.Context, event *AuditEvent) error
	QueryEvents(ctx context.Context, tenantID *uuid.UUID, filter AuditEventFilter, limit, offset int) ([]*AuditEvent, error)
	DeleteOldEvents(ctx context.Context, tenantID uuid.UUID, before time.Time) error
	CountEvents(ctx context.Context, tenantID *uuid.UUID, filter AuditEventFilter) (int, error)
}
