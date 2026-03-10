package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/user"
)

// UserRepository implements the user.Repository interface
type UserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// stringToNullString converts a string to sql.NullString
func stringToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// userModel represents the database model for user
type userModel struct {
	ID                 uuid.UUID      `db:"id"`
	Email              string         `db:"email"`
	PasswordHash       sql.NullString `db:"password_hash"`
	FirstName          string         `db:"first_name"`
	LastName           string         `db:"last_name"`
	Status             string         `db:"status"`
	EmailVerified      bool           `db:"email_verified"`
	EmailVerifiedAt    *time.Time     `db:"email_verified_at"`
	MFAEnabled         bool           `db:"mfa_enabled"`
	MFAType            sql.NullString `db:"mfa_type"`
	MFASecret          sql.NullString `db:"mfa_secret"`
	FailedLoginCount   int            `db:"failed_login_count"`
	LastLoginAt        *time.Time     `db:"last_login_at"`
	PasswordChangedAt  *time.Time     `db:"password_changed_at"`
	MustChangePassword bool           `db:"must_change_password"`
	LockedUntil        *time.Time     `db:"locked_until"`
	AuthMethod         string         `db:"auth_method"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
	Version            int            `db:"version"`
}

func uuidToNullable(tenantID uuid.UUID) interface{} {
	if tenantID == uuid.Nil {
		return nil
	}
	return tenantID
}

// Save persists a user
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, status,
			email_verified, email_verified_at, mfa_enabled, mfa_type, mfa_secret,
			failed_login_count, last_login_at, password_changed_at, must_change_password, locked_until,
			auth_method, created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	passwordHash := stringToNullString(u.PasswordHash())

	_, err := r.db.ExecContext(ctx, query,
		u.ID(),
		u.Email(),
		passwordHash,
		u.FirstName(),
		u.LastName(),
		string(u.Status()),
		u.IsEmailVerified(),
		u.EmailVerifiedAt(),
		u.IsMFAEnabled(),
		stringToNullString(string(u.MFAType())),
		stringToNullString(u.MFASecret()),
		u.FailedLoginCount(),
		u.LastLoginAt(),
		u.PasswordChangedAt(),
		u.MustChangePassword(),
		u.LockedUntil(),
		string(u.AuthMethod()),
		u.CreatedAt(),
		u.UpdatedAt(),
		u.Version(),
	)

	if err != nil {
		r.logger.Error("Failed to save user",
			zap.String("user_id", u.ID().String()),
			zap.String("email", u.Email()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save user: %w", err)
	}

	r.logger.Info("User saved successfully",
		zap.String("user_id", u.ID().String()),
		zap.String("email", u.Email()),
	)

	return nil
}

// FindByID retrieves a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var model userModel

	query := `
		SELECT id, email, password_hash, first_name, last_name, status,
			   email_verified, email_verified_at, mfa_enabled, mfa_type, mfa_secret,
			   failed_login_count, last_login_at, password_changed_at, must_change_password, locked_until,
			   auth_method, created_at, updated_at, version
		FROM users
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrUserNotFound
		}
		r.logger.Error("Failed to find user by ID",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	return r.modelToUser(&model)
}

// FindByIDsBatch retrieves multiple users by IDs in a single query (avoids N+1)
func (r *UserRepository) FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*user.User), nil
	}

	var models []userModel

	query := `
		SELECT id, email, password_hash, first_name, last_name, status,
			   email_verified, email_verified_at, mfa_enabled, mfa_type, mfa_secret,
			   failed_login_count, last_login_at, password_changed_at, must_change_password, locked_until,
			   auth_method, created_at, updated_at, version
		FROM users
		WHERE id = ANY($1)
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &models, query, ids)
	if err != nil {
		r.logger.Error("Failed to find users in batch",
			zap.Int("count", len(ids)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find users in batch: %w", err)
	}

	result := make(map[uuid.UUID]*user.User, len(models))
	for i := range models {
		u, err := r.modelToUser(&models[i])
		if err != nil {
			r.logger.Error("Failed to convert user model in batch query",
				zap.String("user_id", models[i].ID.String()),
				zap.Error(err),
			)
			continue
		}
		result[u.ID()] = u
	}

	r.logger.Debug("Found users in batch",
		zap.Int("requested_count", len(ids)),
		zap.Int("found_count", len(result)),
	)

	return result, nil
}

// FindByEmail retrieves a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var model userModel

	query := `
		SELECT id, email, password_hash, first_name, last_name, status,
			   email_verified, email_verified_at, mfa_enabled, mfa_type, mfa_secret,
			   failed_login_count, last_login_at, password_changed_at, must_change_password, locked_until,
			   auth_method, created_at, updated_at, version
		FROM users
		WHERE email = $1
	`

	err := r.db.GetContext(ctx, &model, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrUserNotFound
		}
		r.logger.Error("Failed to find user by email",
			zap.String("email", email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return r.modelToUser(&model)
}

// FindByTenantID retrieves all users for a tenant with optimized pagination
func (r *UserRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	return r.FindByTenantIDWithPagination(ctx, tenantID, 0, 100)
}

// FindByTenantIDWithPagination retrieves users for a tenant with pagination to avoid large result sets
// This includes both direct role assignments and group memberships
// Excludes users with pending status (invited but not yet accepted)
func (r *UserRepository) FindByTenantIDWithPagination(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*user.User, error) {
	var models []userModel

	query := `
		SELECT DISTINCT id, email, password_hash, first_name, last_name, status,
			   email_verified, email_verified_at, mfa_enabled, mfa_type, mfa_secret,
			   failed_login_count, last_login_at, password_changed_at, must_change_password, locked_until,
			   auth_method, created_at, updated_at, version
		FROM users
		WHERE status != 'pending'
		AND id IN (
			-- Users with direct role assignment to tenant
			SELECT DISTINCT user_id FROM user_role_assignments WHERE tenant_id = $1
			UNION
			-- Users who are members of groups that belong to tenant
			SELECT DISTINCT user_id FROM group_members
			WHERE group_id IN (
				SELECT id FROM tenant_groups WHERE tenant_id = $1
			)
		)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &models, query, tenantID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to find users by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find users by tenant ID: %w", err)
	}

	users := make([]*user.User, len(models))
	for i, model := range models {
		user, err := r.modelToUser(&model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to user: %w", err)
		}
		users[i] = user
	}

	return users, nil
}

// FindAll retrieves all users in the system (used by system administrators)
func (r *UserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	var models []userModel

	query := `
		SELECT DISTINCT id, email, password_hash, first_name, last_name, status,
			   email_verified, email_verified_at, mfa_enabled, mfa_type, mfa_secret,
			   failed_login_count, last_login_at, password_changed_at, must_change_password, locked_until,
			   auth_method, created_at, updated_at, version
		FROM users
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &models, query)
	if err != nil {
		r.logger.Error("Failed to find all users",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find all users: %w", err)
	}

	users := make([]*user.User, len(models))
	for i, model := range models {
		user, err := r.modelToUser(&model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to user: %w", err)
		}
		users[i] = user
	}

	return users, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users SET
			email = $2, password_hash = $3, first_name = $4, last_name = $5, status = $6,
			email_verified = $7, email_verified_at = $8, mfa_enabled = $9, mfa_type = $10,
			mfa_secret = $11, failed_login_count = $12, last_login_at = $13,
			password_changed_at = $14, must_change_password = $15, locked_until = $16, auth_method = $17, updated_at = $18, version = $19
		WHERE id = $1 AND version = $20
	`

	passwordHash := stringToNullString(u.PasswordHash())

	result, err := r.db.ExecContext(ctx, query,
		u.ID(),
		u.Email(),
		passwordHash,
		u.FirstName(),
		u.LastName(),
		string(u.Status()),
		u.IsEmailVerified(),
		u.EmailVerifiedAt(),
		u.IsMFAEnabled(),
		stringToNullString(string(u.MFAType())),
		stringToNullString(u.MFASecret()),
		u.FailedLoginCount(),
		u.LastLoginAt(),
		u.PasswordChangedAt(),
		u.MustChangePassword(),
		u.LockedUntil(),
		string(u.AuthMethod()),
		u.UpdatedAt(),
		u.Version(),
		u.Version()-1, // Optimistic concurrency check - check OLD version
	)

	if err != nil {
		r.logger.Error("Failed to update user",
			zap.String("user_id", u.ID().String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or version conflict")
	}

	r.logger.Info("User updated successfully",
		zap.String("user_id", u.ID().String()),
		zap.String("email", u.Email()),
	)

	return nil
}

// Delete removes a user and cascades cleanup of role assignments and group memberships
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Start transaction to ensure atomic cleanup
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to start transaction for user deletion",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, remove all user role assignments
	deleteRoleAssignmentsQuery := `DELETE FROM user_role_assignments WHERE user_id = $1`
	result, err := tx.ExecContext(ctx, deleteRoleAssignmentsQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete user role assignments",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete user role assignments: %w", err)
	}

	roleAssignmentsDeleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for role assignments: %w", err)
	}

	if roleAssignmentsDeleted > 0 {
		r.logger.Info("Cleaned up user role assignments during user deletion",
			zap.String("user_id", id.String()),
			zap.Int64("role_assignments_deleted", roleAssignmentsDeleted),
		)
	}

	// Second, remove all group memberships
	deleteGroupMembersQuery := `DELETE FROM group_members WHERE user_id = $1`
	result, err = tx.ExecContext(ctx, deleteGroupMembersQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete user group memberships",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete user group memberships: %w", err)
	}

	groupMembershipsDeleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for group memberships: %w", err)
	}

	if groupMembershipsDeleted > 0 {
		r.logger.Info("Cleaned up user group memberships during user deletion",
			zap.String("user_id", id.String()),
			zap.Int64("group_memberships_deleted", groupMembershipsDeleted),
		)
	}

	// Third, remove user from all projects
	deleteProjectMembersQuery := `DELETE FROM project_members WHERE user_id = $1`
	result, err = tx.ExecContext(ctx, deleteProjectMembersQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete user project memberships",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete user project memberships: %w", err)
	}

	projectMembershipsDeleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for project memberships: %w", err)
	}

	if projectMembershipsDeleted > 0 {
		r.logger.Info("Cleaned up user project memberships during user deletion",
			zap.String("user_id", id.String()),
			zap.Int64("project_memberships_deleted", projectMembershipsDeleted),
		)
	}

	// Finally, delete the user itself
	deleteUserQuery := `DELETE FROM users WHERE id = $1`
	result, err = tx.ExecContext(ctx, deleteUserQuery, id)
	if err != nil {
		r.logger.Error("Failed to delete user",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return user.ErrUserNotFound
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit user deletion transaction",
			zap.String("user_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("User deleted successfully with cascading cleanup",
		zap.String("user_id", id.String()),
		zap.Int64("role_assignments_deleted", roleAssignmentsDeleted),
		zap.Int64("group_memberships_deleted", groupMembershipsDeleted),
		zap.Int64("project_memberships_deleted", projectMembershipsDeleted),
	)

	return nil
}

// ExistsByEmail checks if a user exists by email
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	err := r.db.GetContext(ctx, &exists, query, email)
	if err != nil {
		r.logger.Error("Failed to check user existence by email",
			zap.String("email", email),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to check user existence by email: %w", err)
	}

	return exists, nil
}

// CountByTenantID counts users for a tenant
func (r *UserRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int

	query := `SELECT COUNT(DISTINCT u.id) FROM users u 
	           INNER JOIN user_role_assignments ura ON u.id = ura.user_id 
	           WHERE ura.tenant_id = $1`

	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to count users by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count users by tenant ID: %w", err)
	}

	return count, nil
}

// modelToUser converts a database model to a domain user
func (r *UserRepository) modelToUser(model *userModel) (*user.User, error) {
	status := user.UserStatus(model.Status)

	// Handle nullable mfa_type field
	mfaType := user.MFATypeNone
	if model.MFAType.Valid {
		mfaType = user.MFAType(model.MFAType.String)
	}

	// Handle nullable mfa_secret field
	mfaSecret := ""
	if model.MFASecret.Valid {
		mfaSecret = model.MFASecret.String
	}

	// Handle nullable password_hash field
	passwordHash := ""
	if model.PasswordHash.Valid {
		passwordHash = model.PasswordHash.String
	}

	// Handle nullable password_changed_at field
	passwordChangedAt := model.CreatedAt // Default to created_at if null
	if model.PasswordChangedAt != nil {
		passwordChangedAt = *model.PasswordChangedAt
	}

	// Handle auth_method with default fallback
	authMethod := user.AuthMethod(model.AuthMethod)
	if authMethod == "" {
		authMethod = user.AuthMethodCredentials // Default to credentials if not set
	}

	return user.NewUserFromExisting(
		model.ID,
		model.Email,
		passwordHash,
		model.FirstName,
		model.LastName,
		status,
		model.EmailVerified,
		model.EmailVerifiedAt,
		model.MFAEnabled,
		mfaType,
		mfaSecret,
		model.FailedLoginCount,
		model.LastLoginAt,
		passwordChangedAt,
		model.MustChangePassword,
		model.LockedUntil,
		authMethod,
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
	)
}

// FindByTenantIDWithRoles retrieves users for a tenant with their roles in a single optimized query
// This eliminates N+1 queries by using PostgreSQL ARRAY_AGG to batch-load roles
func (r *UserRepository) FindByTenantIDWithRoles(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	return r.FindByTenantIDWithRolesAndPagination(ctx, tenantID, 0, 100)
}

// FindByTenantIDWithRolesAndPagination retrieves users for a tenant with their roles using a single query
// Optimized to avoid N+1 problem by loading user and role data together
// Excludes users with pending status (invited but not yet accepted)
func (r *UserRepository) FindByTenantIDWithRolesAndPagination(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*user.User, error) {
	var models []userModel

	// Optimized query with pagination and role data in single query
	query := `
		SELECT DISTINCT
			u.id, u.email, u.password_hash, u.first_name, u.last_name, u.status,
			u.email_verified, u.email_verified_at, u.mfa_enabled, u.mfa_type, u.mfa_secret,
			u.failed_login_count, u.last_login_at, u.password_changed_at, u.must_change_password, u.locked_until,
			u.auth_method, u.created_at, u.updated_at, u.version
		FROM users u
		INNER JOIN user_role_assignments ura ON u.id = ura.user_id
		WHERE u.status != 'pending'
		AND ura.tenant_id = $1
		ORDER BY u.created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &models, query, tenantID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to find users with roles by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find users with roles by tenant ID: %w", err)
	}

	// If no users found, return early
	if len(models) == 0 {
		return []*user.User{}, nil
	}

	// Extract user IDs for batch role lookup
	userIDs := make([]uuid.UUID, len(models))
	for i, model := range models {
		userIDs[i] = model.ID
	}

	// Load all roles for these users in a single query
	var roleAssignments []struct {
		UserID uuid.UUID `db:"user_id"`
		RoleID uuid.UUID `db:"role_id"`
	}

	roleQuery := `
		SELECT user_id, role_id
		FROM user_role_assignments
		WHERE user_id = ANY($1) AND tenant_id = $2
	`

	err = r.db.SelectContext(ctx, &roleAssignments, roleQuery, userIDs, tenantID)
	if err != nil {
		r.logger.Error("Failed to find role assignments",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find role assignments: %w", err)
	}

	// Group roles by user ID
	userRoles := make(map[uuid.UUID][]uuid.UUID)
	for _, assignment := range roleAssignments {
		userRoles[assignment.UserID] = append(userRoles[assignment.UserID], assignment.RoleID)
	}

	// Convert models to users with roles
	users := make([]*user.User, len(models))
	for i, model := range models {
		usr, err := r.modelToUser(&model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to user: %w", err)
		}
		users[i] = usr
	}

	return users, nil
}

// GetUserTenants retrieves all tenants a user has roles in
func (r *UserRepository) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var tenantIDs []uuid.UUID

	query := `
		SELECT DISTINCT tenant_id
		FROM (
			SELECT ura.tenant_id
			FROM user_role_assignments ura
			WHERE ura.user_id = $1
			  AND ura.tenant_id IS NOT NULL

			UNION

			SELECT tg.tenant_id
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			WHERE gm.user_id = $1
			  AND gm.removed_at IS NULL
			  AND tg.status = 'active'
			  AND tg.tenant_id IS NOT NULL
		) user_tenants
		ORDER BY tenant_id
	`

	err := r.db.SelectContext(ctx, &tenantIDs, query, userID)
	if err != nil {
		r.logger.Error("Failed to get user tenants",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get user tenants: %w", err)
	}

	return tenantIDs, nil
}

// GetUserPrimaryTenant retrieves the primary (first) tenant for a user
func (r *UserRepository) GetUserPrimaryTenant(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID

	query := `
		SELECT tenant_id
		FROM user_role_assignments
		WHERE user_id = $1 AND tenant_id IS NOT NULL
		ORDER BY created_at ASC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &tenantID, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// User has no tenant assignments - might be a system admin
			return uuid.Nil, nil
		}
		r.logger.Error("Failed to get user primary tenant",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("failed to get user primary tenant: %w", err)
	}

	return tenantID, nil
}

// GetTotalUserCount returns the total number of users in the system
func (r *UserRepository) GetTotalUserCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		r.logger.Error("Failed to get total user count", zap.Error(err))
		return 0, fmt.Errorf("failed to get total user count: %w", err)
	}

	return count, nil
}

// GetActiveUserCount returns the number of users active within the specified number of days
func (r *UserRepository) GetActiveUserCount(ctx context.Context, days int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM users
		WHERE last_login_at >= NOW() - INTERVAL '%d days'
	`

	err := r.db.GetContext(ctx, &count, fmt.Sprintf(query, days))
	if err != nil {
		r.logger.Error("Failed to get active user count",
			zap.Int("days", days),
			zap.Error(err))
		return 0, fmt.Errorf("failed to get active user count: %w", err)
	}

	return count, nil
}
