package eprregistration

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusWithdrawn Status = "withdrawn"
)

type LifecycleStatus string

const (
	LifecycleStatusActive    LifecycleStatus = "active"
	LifecycleStatusExpiring  LifecycleStatus = "expiring"
	LifecycleStatusExpired   LifecycleStatus = "expired"
	LifecycleStatusSuspended LifecycleStatus = "suspended"
)

var (
	ErrInvalidTenantID   = errors.New("invalid tenant id")
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrInvalidEPRRecord  = errors.New("epr record id is required")
	ErrInvalidProduct    = errors.New("product name is required")
	ErrInvalidTechnology = errors.New("technology name is required")
	ErrInvalidStatus     = errors.New("invalid status filter")
	ErrInvalidLifecycle  = errors.New("invalid lifecycle status")
	ErrNotFound          = errors.New("epr registration request not found")
	ErrAlreadyDecided    = errors.New("epr registration request already decided")
	ErrLifecycleAction   = errors.New("invalid lifecycle action")
)

type Request struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	EPRRecordID           string
	ProductName           string
	TechnologyName        string
	BusinessJustification string
	RequestedByUserID     uuid.UUID
	Status                Status
	DecidedByUserID       *uuid.UUID
	DecisionReason        string
	DecidedAt             *time.Time
	ApprovedAt            *time.Time
	ExpiresAt             *time.Time
	LifecycleStatus       LifecycleStatus
	SuspensionReason      string
	LastReviewedAt        *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewRequest(tenantID, requestedByUserID uuid.UUID, eprRecordID, productName, technologyName, businessJustification string) (*Request, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if requestedByUserID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	eprRecordID = strings.TrimSpace(eprRecordID)
	if eprRecordID == "" {
		return nil, ErrInvalidEPRRecord
	}
	productName = strings.TrimSpace(productName)
	if productName == "" {
		return nil, ErrInvalidProduct
	}
	technologyName = strings.TrimSpace(technologyName)
	if technologyName == "" {
		return nil, ErrInvalidTechnology
	}
	now := time.Now().UTC()
	return &Request{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		EPRRecordID:           eprRecordID,
		ProductName:           productName,
		TechnologyName:        technologyName,
		BusinessJustification: strings.TrimSpace(businessJustification),
		RequestedByUserID:     requestedByUserID,
		Status:                StatusPending,
		LifecycleStatus:       LifecycleStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

func (r *Request) Approve(actorUserID uuid.UUID, reason string) error {
	if actorUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Status != StatusPending {
		return ErrAlreadyDecided
	}
	now := time.Now().UTC()
	r.Status = StatusApproved
	r.DecidedByUserID = &actorUserID
	r.DecisionReason = strings.TrimSpace(reason)
	r.DecidedAt = &now
	r.ApprovedAt = &now
	if r.LifecycleStatus == "" {
		r.LifecycleStatus = LifecycleStatusActive
	}
	r.LastReviewedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Request) Reject(actorUserID uuid.UUID, reason string) error {
	if actorUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Status != StatusPending {
		return ErrAlreadyDecided
	}
	now := time.Now().UTC()
	r.Status = StatusRejected
	r.DecidedByUserID = &actorUserID
	r.DecisionReason = strings.TrimSpace(reason)
	r.DecidedAt = &now
	if r.LifecycleStatus == "" {
		r.LifecycleStatus = LifecycleStatusActive
	}
	r.LastReviewedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Request) Withdraw(actorUserID uuid.UUID, reason string) error {
	if actorUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Status != StatusPending {
		return ErrAlreadyDecided
	}
	now := time.Now().UTC()
	r.Status = StatusWithdrawn
	r.DecidedByUserID = &actorUserID
	r.DecisionReason = strings.TrimSpace(reason)
	r.DecidedAt = &now
	if r.LifecycleStatus == "" {
		r.LifecycleStatus = LifecycleStatusActive
	}
	r.LastReviewedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Request) Suspend(actorUserID uuid.UUID, reason string) error {
	if actorUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Status != StatusApproved {
		return ErrLifecycleAction
	}
	if r.LifecycleStatus == LifecycleStatusSuspended {
		return ErrLifecycleAction
	}
	now := time.Now().UTC()
	r.LifecycleStatus = LifecycleStatusSuspended
	r.SuspensionReason = strings.TrimSpace(reason)
	r.DecidedByUserID = &actorUserID
	r.LastReviewedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Request) Reactivate(actorUserID uuid.UUID, reason string) error {
	if actorUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Status != StatusApproved {
		return ErrLifecycleAction
	}
	if r.LifecycleStatus != LifecycleStatusSuspended && r.LifecycleStatus != LifecycleStatusExpired && r.LifecycleStatus != LifecycleStatusExpiring {
		return ErrLifecycleAction
	}
	now := time.Now().UTC()
	r.LifecycleStatus = LifecycleStatusActive
	r.SuspensionReason = ""
	r.DecidedByUserID = &actorUserID
	r.DecisionReason = strings.TrimSpace(reason)
	r.LastReviewedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Request) Revalidate(actorUserID uuid.UUID, reason string) error {
	if actorUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Status != StatusApproved {
		return ErrLifecycleAction
	}
	now := time.Now().UTC()
	r.LifecycleStatus = LifecycleStatusActive
	r.SuspensionReason = ""
	r.DecidedByUserID = &actorUserID
	r.DecisionReason = strings.TrimSpace(reason)
	r.LastReviewedAt = &now
	r.UpdatedAt = now
	return nil
}

func ParseLifecycleStatus(raw string) (LifecycleStatus, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	status := LifecycleStatus(raw)
	switch status {
	case LifecycleStatusActive, LifecycleStatusExpiring, LifecycleStatusExpired, LifecycleStatusSuspended:
		return status, nil
	default:
		return "", ErrInvalidLifecycle
	}
}
