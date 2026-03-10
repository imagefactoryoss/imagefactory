package project

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrMemberNotFound       = errors.New("project member not found")
	ErrMemberAlreadyExists  = errors.New("user is already a member of this project")
	ErrMemberNotAuthorized  = errors.New("user not authorized to access this project")
)

// MemberRepository defines operations for project member persistence
type MemberRepository interface {
	// CreateMember adds a new member to a project
	CreateMember(ctx context.Context, member *Member) error

	// GetMember retrieves a specific project member
	GetMember(ctx context.Context, projectID, userID uuid.UUID) (*Member, error)

	// ListMembers retrieves all members of a project
	ListMembers(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Member, int, error)

	// ListUserProjects retrieves all projects a user is a member of
	ListUserProjects(ctx context.Context, userID, tenantID uuid.UUID, limit, offset int) ([]*Project, int, error)

	// UpdateMember updates a member's role override
	UpdateMember(ctx context.Context, member *Member) error

	// DeleteMember removes a user from a project
	DeleteMember(ctx context.Context, projectID, userID uuid.UUID) error

	// IsMember checks if user is a member of the project
	IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error)

	// CountMembers returns the count of members in a project
	CountMembers(ctx context.Context, projectID uuid.UUID) (int, error)

	// DeleteProjectMembers removes all members when a project is deleted
	DeleteProjectMembers(ctx context.Context, projectID uuid.UUID) error
}
