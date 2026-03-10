package eprregistration

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type LifecycleTransitionRecord struct {
	RequestID         uuid.UUID
	TenantID          uuid.UUID
	EPRRecordID       string
	RequestedByUserID uuid.UUID
	LifecycleStatus   LifecycleStatus
}

type Repository interface {
	Create(ctx context.Context, req *Request) error
	GetByID(ctx context.Context, id uuid.UUID) (*Request, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*Request, error)
	ListAll(ctx context.Context, status *Status, limit, offset int) ([]*Request, error)
	UpdateDecision(ctx context.Context, req *Request) error
	UpdateLifecycle(ctx context.Context, req *Request) error
	TransitionLifecycleStates(ctx context.Context, now time.Time, expiringBefore time.Time) (expiring []LifecycleTransitionRecord, expired []LifecycleTransitionRecord, err error)
	HasApprovedRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error)
	GetApprovedRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*LifecycleStatus, error)
}
