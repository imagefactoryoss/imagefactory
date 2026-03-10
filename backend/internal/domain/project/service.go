package project

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service defines the business logic for project management
type Service struct {
	repository       Repository
	memberRepository MemberRepository
	eventPublisher   EventPublisher
	logger           *zap.Logger
}

// NewService creates a new project service
func NewService(repository Repository, memberRepository MemberRepository, eventPublisher EventPublisher, logger *zap.Logger) *Service {
	return &Service{
		repository:       repository,
		memberRepository: memberRepository,
		eventPublisher:   eventPublisher,
		logger:           logger,
	}
}

// CreateProject creates a new project
func (s *Service) CreateProject(ctx context.Context, tenantID uuid.UUID, name, slug, description, gitRepo, gitBranch, visibility, gitProvider string, repoAuthID *uuid.UUID, actorID *uuid.UUID, isDraft bool) (*Project, error) {
	s.logger.Info("Creating new project",
		zap.String("tenant_id", tenantID.String()),
		zap.String("name", name))

	// Check if project name already exists for this tenant
	exists, err := s.repository.ExistsByNameAndTenantID(ctx, name, tenantID)
	if err != nil {
		s.logger.Error("Failed to check project name existence", zap.Error(err))
		return nil, err
	}
	if exists {
		s.logger.Warn("Project name already exists",
			zap.String("name", name),
			zap.String("tenant_id", tenantID.String()))
		return nil, ErrDuplicateProjectName
	}

	if slug == "" {
		slug = GenerateSlug(name)
	}
	exists, err = s.repository.ExistsBySlugAndTenantID(ctx, slug, tenantID)
	if err != nil {
		s.logger.Error("Failed to check project slug existence", zap.Error(err))
		return nil, err
	}
	if exists {
		s.logger.Warn("Project slug already exists",
			zap.String("slug", slug),
			zap.String("tenant_id", tenantID.String()))
		return nil, ErrDuplicateProjectSlug
	}

	// Create new project aggregate
	project, err := NewProject(tenantID, name, slug, description, gitRepo, gitBranch, visibility, gitProvider, repoAuthID, actorID, isDraft)
	if err != nil {
		s.logger.Error("Failed to create project aggregate", zap.Error(err))
		return nil, err
	}

	// Save to repository
	if err := s.repository.Save(ctx, project); err != nil {
		s.logger.Error("Failed to save project", zap.Error(err), zap.String("project_id", project.ID().String()))
		return nil, err
	}

	if s.eventPublisher != nil {
		_ = s.eventPublisher.PublishProjectCreated(ctx, NewProjectCreated(project, actorID))
	}

	s.logger.Info("Project created successfully", zap.String("project_id", project.ID().String()))
	return project, nil
}

// GetProject retrieves a project by ID
func (s *Service) GetProject(ctx context.Context, id uuid.UUID) (*Project, error) {
	s.logger.Debug("Retrieving project", zap.String("project_id", id.String()))

	project, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find project", zap.Error(err), zap.String("project_id", id.String()))
		return nil, err
	}

	if project == nil {
		s.logger.Warn("Project not found", zap.String("project_id", id.String()))
		return nil, ErrProjectNotFound
	}

	return project, nil
}

// ListProjects retrieves projects for a tenant with pagination
func (s *Service) ListProjects(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID, limit, offset int) ([]*Project, int, error) {
	s.logger.Debug("Listing projects",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	projects, err := s.repository.FindByTenantID(ctx, tenantID, viewerID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list projects", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, 0, err
	}

	totalCount, err := s.repository.CountByTenantID(ctx, tenantID, viewerID)
	if err != nil {
		s.logger.Error("Failed to count projects", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, 0, err
	}

	return projects, totalCount, nil
}

// UpdateProject updates an existing project
func (s *Service) UpdateProject(ctx context.Context, id uuid.UUID, name, slug, description, gitRepo, gitBranch, gitProvider string, repoAuthID *uuid.UUID, actorID *uuid.UUID, isDraft *bool) (*Project, error) {
	s.logger.Info("Updating project",
		zap.String("project_id", id.String()),
		zap.String("name", name))

	// Get existing project
	project, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find project for update", zap.Error(err), zap.String("project_id", id.String()))
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}

	// Check if name change would create a duplicate
	if name != project.Name() {
		exists, err := s.repository.ExistsByNameAndTenantID(ctx, name, project.TenantID())
		if err != nil {
			s.logger.Error("Failed to check project name existence for update", zap.Error(err))
			return nil, err
		}
		if exists {
			return nil, ErrDuplicateProjectName
		}
	}

	if slug == "" {
		slug = GenerateSlug(name)
	}
	if slug != project.Slug() {
		exists, err := s.repository.ExistsBySlugAndTenantID(ctx, slug, project.TenantID())
		if err != nil {
			s.logger.Error("Failed to check project slug existence for update", zap.Error(err))
			return nil, err
		}
		if exists {
			return nil, ErrDuplicateProjectSlug
		}
	}

	// Update project
	if err := project.Update(name, slug, description, gitRepo, gitBranch, gitProvider, repoAuthID); err != nil {
		s.logger.Error("Failed to update project aggregate", zap.Error(err))
		return nil, err
	}

	if isDraft != nil {
		project.SetDraft(*isDraft)
	}

	// Save to repository
	if err := s.repository.Update(ctx, project); err != nil {
		s.logger.Error("Failed to save updated project", zap.Error(err), zap.String("project_id", id.String()))
		return nil, err
	}

	if s.eventPublisher != nil {
		_ = s.eventPublisher.PublishProjectUpdated(ctx, NewProjectUpdated(project, actorID))
	}

	s.logger.Info("Project updated successfully", zap.String("project_id", id.String()))
	return project, nil
}

// DeleteProject performs soft delete
func (s *Service) DeleteProject(ctx context.Context, id uuid.UUID, actorID *uuid.UUID) error {
	s.logger.Info("Deleting project", zap.String("project_id", id.String()))

	// Get existing project
	project, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find project for deletion", zap.Error(err), zap.String("project_id", id.String()))
		return err
	}
	if project == nil {
		return ErrProjectNotFound
	}

	// Perform soft delete
	if err := project.Delete(); err != nil {
		s.logger.Error("Failed to delete project aggregate", zap.Error(err))
		return err
	}

	// Save to repository
	if err := s.repository.Update(ctx, project); err != nil {
		s.logger.Error("Failed to save deleted project", zap.Error(err), zap.String("project_id", id.String()))
		return err
	}

	if s.eventPublisher != nil {
		_ = s.eventPublisher.PublishProjectDeleted(ctx, NewProjectDeleted(project, actorID))
	}

	s.logger.Info("Project deleted successfully", zap.String("project_id", id.String()))
	return nil
}

// PurgeDeletedProjects permanently deletes projects older than retentionDays.
func (s *Service) PurgeDeletedProjects(ctx context.Context, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	return s.repository.PurgeDeletedBefore(ctx, cutoff)
}

// ============================================================================
// PROJECT MEMBER MANAGEMENT
// ============================================================================

// AddMember adds a user to a project
func (s *Service) AddMember(ctx context.Context, projectID, userID uuid.UUID, assignedByUserID *uuid.UUID) (*Member, error) {
	s.logger.Info("Adding member to project",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))

	// Verify project exists
	project, err := s.repository.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error("Failed to find project", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}

	// Check if user is already a member
	isMember, err := s.memberRepository.IsMember(ctx, projectID, userID)
	if err != nil {
		s.logger.Error("Failed to check membership", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}
	if isMember {
		s.logger.Warn("User is already a member of the project",
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()))
		return nil, ErrMemberAlreadyExists
	}

	// Create new member
	member, err := NewMember(projectID, userID, assignedByUserID)
	if err != nil {
		s.logger.Error("Failed to create member aggregate", zap.Error(err))
		return nil, err
	}

	// Save to repository
	if err := s.memberRepository.CreateMember(ctx, member); err != nil {
		s.logger.Error("Failed to save member", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}

	s.logger.Info("Member added to project successfully",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))
	return member, nil
}

// RemoveMember removes a user from a project
func (s *Service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	s.logger.Info("Removing member from project",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))

	// Verify member exists
	member, err := s.memberRepository.GetMember(ctx, projectID, userID)
	if err != nil {
		s.logger.Error("Failed to find member", zap.Error(err), zap.String("project_id", projectID.String()))
		return err
	}
	if member == nil {
		return ErrMemberNotFound
	}

	// Delete member
	if err := s.memberRepository.DeleteMember(ctx, projectID, userID); err != nil {
		s.logger.Error("Failed to delete member", zap.Error(err), zap.String("project_id", projectID.String()))
		return err
	}

	s.logger.Info("Member removed from project successfully",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))
	return nil
}

// GetMember retrieves a project member
func (s *Service) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*Member, error) {
	s.logger.Debug("Retrieving project member",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))

	member, err := s.memberRepository.GetMember(ctx, projectID, userID)
	if err != nil {
		s.logger.Error("Failed to find member", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}

	if member == nil {
		return nil, ErrMemberNotFound
	}

	return member, nil
}

// ListMembers retrieves all members of a project
func (s *Service) ListMembers(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Member, int, error) {
	s.logger.Debug("Listing project members",
		zap.String("project_id", projectID.String()),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	members, total, err := s.memberRepository.ListMembers(ctx, projectID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list members", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, 0, err
	}

	return members, total, nil
}

// ListUserProjects retrieves all projects a user is a member of
func (s *Service) ListUserProjects(ctx context.Context, userID, tenantID uuid.UUID, limit, offset int) ([]*Project, int, error) {
	s.logger.Debug("Listing user's projects",
		zap.String("user_id", userID.String()),
		zap.String("tenant_id", tenantID.String()))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	projects, total, err := s.memberRepository.ListUserProjects(ctx, userID, tenantID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list user projects", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, 0, err
	}

	return projects, total, nil
}

// UpdateMemberRole updates a member's project-level role override
func (s *Service) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, roleID *uuid.UUID) (*Member, error) {
	s.logger.Info("Updating member role",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))

	// Get existing member
	member, err := s.memberRepository.GetMember(ctx, projectID, userID)
	if err != nil {
		s.logger.Error("Failed to find member", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}
	if member == nil {
		return nil, ErrMemberNotFound
	}

	// Update role
	member.SetRoleID(roleID)

	// Save to repository
	if err := s.memberRepository.UpdateMember(ctx, member); err != nil {
		s.logger.Error("Failed to update member", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, err
	}

	s.logger.Info("Member role updated successfully",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))
	return member, nil
}

// UserHasProjectAccess checks if a user has access to a specific project
func (s *Service) UserHasProjectAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	s.logger.Debug("Checking project access",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))

	isMember, err := s.memberRepository.IsMember(ctx, projectID, userID)
	if err != nil {
		s.logger.Error("Failed to check project access", zap.Error(err), zap.String("project_id", projectID.String()))
		return false, err
	}

	return isMember, nil
}
