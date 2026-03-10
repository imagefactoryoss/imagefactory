package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/project"
)

// ProjectMemberRepository implements the project.MemberRepository interface for PostgreSQL
type ProjectMemberRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewProjectMemberRepository creates a new PostgreSQL project member repository
func NewProjectMemberRepository(db *sqlx.DB, logger *zap.Logger) *ProjectMemberRepository {
	return &ProjectMemberRepository{
		db:     db,
		logger: logger,
	}
}

// CreateMember adds a new member to a project
func (r *ProjectMemberRepository) CreateMember(ctx context.Context, member *project.Member) error {
	const query = `
		INSERT INTO project_members (id, project_id, user_id, assigned_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		member.ID(),
		member.ProjectID(),
		member.UserID(),
		member.AssignedByUserID(),
		member.CreatedAt(),
		member.UpdatedAt(),
	)
	if err != nil {
		r.logger.Error("Failed to create project member", zap.Error(err),
			zap.String("project_id", member.ProjectID().String()),
			zap.String("user_id", member.UserID().String()))
		return fmt.Errorf("failed to create project member: %w", err)
	}

	return nil
}

// GetMember retrieves a specific project member
func (r *ProjectMemberRepository) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*project.Member, error) {
	const query = `
		SELECT id, project_id, user_id, role_id, assigned_by_user_id, created_at, updated_at
		FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`

	var id, pID, uID, roleID, abuID sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(
		&id, &pID, &uID, &roleID, &abuID, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to retrieve project member", zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("failed to retrieve project member: %w", err)
	}

	// Reconstruct member from database values
	uUID, _ := uuid.Parse(uID.String)
	pUID, _ := uuid.Parse(pID.String)

	var abUserID *uuid.UUID
	if abuID.Valid {
		if rid, err := uuid.Parse(abuID.String); err == nil {
			abUserID = &rid
		}
	}

	member, _ := project.NewMember(pUID, uUID, abUserID)

	// Set role override if exists
	if roleID.Valid {
		if rid, err := uuid.Parse(roleID.String); err == nil {
			member.SetRoleID(&rid)
		}
	}

	return member, nil
}

// ListMembers retrieves all members of a project
func (r *ProjectMemberRepository) ListMembers(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*project.Member, int, error) {
	const query = `
		SELECT id, project_id, user_id, role_id, assigned_by_user_id, created_at, updated_at
		FROM project_members
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	const countQuery = `
		SELECT COUNT(*)
		FROM project_members
		WHERE project_id = $1
	`

	// Get total count
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, projectID).Scan(&totalCount); err != nil {
		r.logger.Error("Failed to count project members", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count project members: %w", err)
	}

	// Get paginated results
	rows, err := r.db.QueryContext(ctx, query, projectID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list project members", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list project members: %w", err)
	}
	defer rows.Close()

	var members []*project.Member
	for rows.Next() {
		var id, pID, uID, roleID, abuID sql.NullString
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&id, &pID, &uID, &roleID, &abuID, &createdAt, &updatedAt); err != nil {
			r.logger.Error("Failed to scan project member row", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to scan project member: %w", err)
		}

		pUID, _ := uuid.Parse(pID.String)
		uUID, _ := uuid.Parse(uID.String)

		var abUserID *uuid.UUID
		if abuID.Valid {
			if rid, err := uuid.Parse(abuID.String); err == nil {
				abUserID = &rid
			}
		}

		member, _ := project.NewMember(pUID, uUID, abUserID)

		// Set role override if exists
		if roleID.Valid {
			if rid, err := uuid.Parse(roleID.String); err == nil {
				member.SetRoleID(&rid)
			}
		}

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating project members", zap.Error(err))
		return nil, 0, fmt.Errorf("error iterating project members: %w", err)
	}

	return members, totalCount, nil
}

// ListUserProjects retrieves all projects a user is a member of
func (r *ProjectMemberRepository) ListUserProjects(ctx context.Context, userID, tenantID uuid.UUID, limit, offset int) ([]*project.Project, int, error) {
	const query = `
		SELECT DISTINCT p.id, p.tenant_id, p.name, p.slug, p.description,
		       COALESCE(ps.repository_url, '') AS git_repository_url,
		       COALESCE(ps.default_branch, 'main') AS git_branch,
		       p.git_provider_key, p.repository_auth_id, p.created_by, p.is_draft,
		       p.status, p.visibility, p.created_at, p.updated_at, p.deleted_at
		FROM projects p
		LEFT JOIN LATERAL (
			SELECT repository_url, default_branch
			FROM project_sources
			WHERE project_id = p.id AND is_active = true
			ORDER BY is_default DESC, created_at ASC
			LIMIT 1
		) ps ON true
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1 AND p.tenant_id = $2 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`
	const countQuery = `
		SELECT COUNT(DISTINCT p.id)
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1 AND p.tenant_id = $2 AND p.deleted_at IS NULL
	`

	// Get total count
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, userID, tenantID).Scan(&totalCount); err != nil {
		r.logger.Error("Failed to count user projects", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count user projects: %w", err)
	}

	// Get paginated results - using base project query function
	rows, err := r.db.QueryContext(ctx, query, userID, tenantID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list user projects", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list user projects: %w", err)
	}
	defer rows.Close()

	var projects []*project.Project
	for rows.Next() {
		var idStr, tenantIDStr sql.NullString
		var name, slug, description, gitRepo, gitBranch, gitProvider, status, visibility string
		var repoAuthID sql.NullString
		var createdBy sql.NullString
		var isDraft sql.NullBool
		var createdAt, updatedAt sql.NullTime
		var deletedAt sql.NullTime

		if err := rows.Scan(&idStr, &tenantIDStr, &name, &slug, &description,
			&gitRepo, &gitBranch, &gitProvider, &repoAuthID, &createdBy, &isDraft,
			&status, &visibility, &createdAt, &updatedAt, &deletedAt); err != nil {
			r.logger.Error("Failed to scan project row", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}

		projectID, _ := uuid.Parse(idStr.String)
		tenantUUID, _ := uuid.Parse(tenantIDStr.String)
		var repoAuthUUID *uuid.UUID
		if repoAuthID.Valid {
			parsed, err := uuid.Parse(repoAuthID.String)
			if err == nil {
				repoAuthUUID = &parsed
			}
		}

		var createdByUUID *uuid.UUID
		if createdBy.Valid {
			parsed, err := uuid.Parse(createdBy.String)
			if err == nil {
				createdByUUID = &parsed
			}
		}

		draftValue := false
		if isDraft.Valid {
			draftValue = isDraft.Bool
		}

		var deletedAtPtr *time.Time
		if deletedAt.Valid {
			deletedAtPtr = &deletedAt.Time
		}

		p := project.NewProjectFromExisting(
			projectID,
			tenantUUID,
			name,
			slug,
			description,
			gitRepo,
			gitBranch,
			gitProvider,
			project.ProjectStatus(status),
			visibility,
			repoAuthUUID,
			createdByUUID,
			draftValue,
			0,
			createdAt.Time,
			updatedAt.Time,
			deletedAtPtr,
			1,
		)

		projects = append(projects, p)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating user projects", zap.Error(err))
		return nil, 0, fmt.Errorf("error iterating user projects: %w", err)
	}

	return projects, totalCount, nil
}

// UpdateMember updates a member's role override
func (r *ProjectMemberRepository) UpdateMember(ctx context.Context, member *project.Member) error {
	const query = `
		UPDATE project_members
		SET role_id = $1, updated_at = $2
		WHERE project_id = $3 AND user_id = $4
	`

	_, err := r.db.ExecContext(ctx, query,
		member.RoleID(),
		member.UpdatedAt(),
		member.ProjectID(),
		member.UserID(),
	)
	if err != nil {
		r.logger.Error("Failed to update project member", zap.Error(err),
			zap.String("project_id", member.ProjectID().String()),
			zap.String("user_id", member.UserID().String()))
		return fmt.Errorf("failed to update project member: %w", err)
	}

	return nil
}

// DeleteMember removes a user from a project
func (r *ProjectMemberRepository) DeleteMember(ctx context.Context, projectID, userID uuid.UUID) error {
	const query = `
		DELETE FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, projectID, userID)
	if err != nil {
		r.logger.Error("Failed to delete project member", zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("user_id", userID.String()))
		return fmt.Errorf("failed to delete project member: %w", err)
	}

	return nil
}

// IsMember checks if user is a member of the project
func (r *ProjectMemberRepository) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM project_members
			WHERE project_id = $1 AND user_id = $2
		)
	`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(&exists); err != nil {
		r.logger.Error("Failed to check project membership", zap.Error(err))
		return false, fmt.Errorf("failed to check project membership: %w", err)
	}

	return exists, nil
}

// CountMembers returns the count of members in a project
func (r *ProjectMemberRepository) CountMembers(ctx context.Context, projectID uuid.UUID) (int, error) {
	const query = `
		SELECT COUNT(*) FROM project_members
		WHERE project_id = $1
	`

	var count int
	if err := r.db.QueryRowContext(ctx, query, projectID).Scan(&count); err != nil {
		r.logger.Error("Failed to count project members", zap.Error(err))
		return 0, fmt.Errorf("failed to count project members: %w", err)
	}

	return count, nil
}

// DeleteProjectMembers removes all members when a project is deleted
func (r *ProjectMemberRepository) DeleteProjectMembers(ctx context.Context, projectID uuid.UUID) error {
	const query = `
		DELETE FROM project_members
		WHERE project_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, projectID)
	if err != nil {
		r.logger.Error("Failed to delete project members", zap.Error(err),
			zap.String("project_id", projectID.String()))
		return fmt.Errorf("failed to delete project members: %w", err)
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================
