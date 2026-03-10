package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors for user deactivations
var (
	ErrUserDeactivationNotFound = errors.New("user deactivation not found")
	ErrInvalidDeactivation      = errors.New("invalid user deactivation")
)

// UserDeactivationReason represents the reason for user deactivation
type UserDeactivationReason string

const (
	UserDeactivationReasonVoluntary    UserDeactivationReason = "voluntary"
	UserDeactivationReasonAdminAction  UserDeactivationReason = "admin_action"
	UserDeactivationReasonSecurity     UserDeactivationReason = "security"
	UserDeactivationReasonPolicy       UserDeactivationReason = "policy_violation"
	UserDeactivationReasonInactivity   UserDeactivationReason = "inactivity"
	UserDeactivationReasonAccountMerge UserDeactivationReason = "account_merge"
)

// UserDeactivation represents a user deactivation aggregate
type UserDeactivation struct {
	id            uuid.UUID
	userID        uuid.UUID
	tenantID      uuid.UUID
	deactivatedBy uuid.UUID
	reason        UserDeactivationReason
	notes         string
	deactivatedAt time.Time
	reactivatedAt *time.Time
	reactivatedBy *uuid.UUID
	ipAddress     string
	userAgent     string
	createdAt     time.Time
	updatedAt     time.Time
	version       int
}

// NewUserDeactivation creates a new user deactivation record
func NewUserDeactivation(
	userID, tenantID, deactivatedBy uuid.UUID,
	reason UserDeactivationReason,
	notes, ipAddress, userAgent string,
) (*UserDeactivation, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if deactivatedBy == uuid.Nil {
		return nil, errors.New("deactivated by user ID is required")
	}

	now := time.Now().UTC()

	return &UserDeactivation{
		id:            uuid.New(),
		userID:        userID,
		tenantID:      tenantID,
		deactivatedBy: deactivatedBy,
		reason:        reason,
		notes:         notes,
		deactivatedAt: now,
		ipAddress:     ipAddress,
		userAgent:     userAgent,
		createdAt:     now,
		updatedAt:     now,
		version:       1,
	}, nil
}

// NewUserDeactivationFromExisting creates a user deactivation from existing data
func NewUserDeactivationFromExisting(
	id, userID, tenantID, deactivatedBy uuid.UUID,
	reason UserDeactivationReason,
	notes string,
	deactivatedAt time.Time,
	reactivatedAt *time.Time,
	reactivatedBy *uuid.UUID,
	ipAddress, userAgent string,
	createdAt, updatedAt time.Time,
	version int,
) (*UserDeactivation, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid deactivation ID")
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if deactivatedBy == uuid.Nil {
		return nil, errors.New("deactivated by user ID is required")
	}

	return &UserDeactivation{
		id:            id,
		userID:        userID,
		tenantID:      tenantID,
		deactivatedBy: deactivatedBy,
		reason:        reason,
		notes:         notes,
		deactivatedAt: deactivatedAt,
		reactivatedAt: reactivatedAt,
		reactivatedBy: reactivatedBy,
		ipAddress:     ipAddress,
		userAgent:     userAgent,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		version:       version,
	}, nil
}

// ID returns the deactivation ID
func (d *UserDeactivation) ID() uuid.UUID {
	return d.id
}

// UserID returns the deactivated user ID
func (d *UserDeactivation) UserID() uuid.UUID {
	return d.userID
}

// TenantID returns the tenant ID
func (d *UserDeactivation) TenantID() uuid.UUID {
	return d.tenantID
}

// DeactivatedBy returns the user who performed the deactivation
func (d *UserDeactivation) DeactivatedBy() uuid.UUID {
	return d.deactivatedBy
}

// Reason returns the deactivation reason
func (d *UserDeactivation) Reason() UserDeactivationReason {
	return d.reason
}

// Notes returns additional notes about the deactivation
func (d *UserDeactivation) Notes() string {
	return d.notes
}

// DeactivatedAt returns when the user was deactivated
func (d *UserDeactivation) DeactivatedAt() time.Time {
	return d.deactivatedAt
}

// IsReactivated returns true if the user has been reactivated
func (d *UserDeactivation) IsReactivated() bool {
	return d.reactivatedAt != nil
}

// ReactivatedAt returns when the user was reactivated
func (d *UserDeactivation) ReactivatedAt() *time.Time {
	return d.reactivatedAt
}

// ReactivatedBy returns the user who performed the reactivation
func (d *UserDeactivation) ReactivatedBy() *uuid.UUID {
	return d.reactivatedBy
}

// Reactivate marks the deactivation as reactivated
func (d *UserDeactivation) Reactivate(reactivatedBy uuid.UUID, ipAddress, userAgent string) error {
	if d.IsReactivated() {
		return errors.New("user is already reactivated")
	}

	now := time.Now().UTC()
	d.reactivatedAt = &now
	d.reactivatedBy = &reactivatedBy
	d.updatedAt = now
	d.version++
	return nil
}

// IpAddress returns the IP address of the deactivation request
func (d *UserDeactivation) IpAddress() string {
	return d.ipAddress
}

// UserAgent returns the user agent of the deactivation request
func (d *UserDeactivation) UserAgent() string {
	return d.userAgent
}

// CreatedAt returns the creation time
func (d *UserDeactivation) CreatedAt() time.Time {
	return d.createdAt
}

// UpdatedAt returns the last update time
func (d *UserDeactivation) UpdatedAt() time.Time {
	return d.updatedAt
}

// Version returns the version for optimistic locking
func (d *UserDeactivation) Version() int {
	return d.version
}