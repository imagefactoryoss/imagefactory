package project

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Member represents a user's membership in a project with optional role override
type Member struct {
	id                uuid.UUID
	projectID         uuid.UUID
	userID            uuid.UUID
	roleID            *uuid.UUID // Optional project-level role override
	assignedByUserID  *uuid.UUID // Audit: who assigned this member
	createdAt         time.Time
	updatedAt         time.Time
}

// NewMember creates a new project member
func NewMember(projectID, userID uuid.UUID, assignedByUserID *uuid.UUID) (*Member, error) {
	if projectID == uuid.Nil {
		return nil, fmt.Errorf("project_id cannot be nil")
	}
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id cannot be nil")
	}

	return &Member{
		id:               uuid.New(),
		projectID:        projectID,
		userID:           userID,
		assignedByUserID: assignedByUserID,
		createdAt:        time.Now().UTC(),
		updatedAt:        time.Now().UTC(),
	}, nil
}

// ID returns the member ID
func (m *Member) ID() uuid.UUID {
	return m.id
}

// ProjectID returns the project ID
func (m *Member) ProjectID() uuid.UUID {
	return m.projectID
}

// UserID returns the user ID
func (m *Member) UserID() uuid.UUID {
	return m.userID
}

// RoleID returns the optional project-level role override
func (m *Member) RoleID() *uuid.UUID {
	return m.roleID
}

// SetRoleID sets a project-level role override
func (m *Member) SetRoleID(roleID *uuid.UUID) {
	m.roleID = roleID
	m.updatedAt = time.Now().UTC()
}

// AssignedByUserID returns the user who assigned this membership
func (m *Member) AssignedByUserID() *uuid.UUID {
	return m.assignedByUserID
}

// CreatedAt returns the creation timestamp
func (m *Member) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m *Member) UpdatedAt() time.Time {
	return m.updatedAt
}

// HasProjectRoleOverride returns true if this member has a project-level role override
func (m *Member) HasProjectRoleOverride() bool {
	return m.roleID != nil
}
