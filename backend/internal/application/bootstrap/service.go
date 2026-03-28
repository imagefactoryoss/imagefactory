package bootstrap

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	StatusNotStarted          = "not_started"
	StatusAdminPasswordIssued = "admin_password_issued"
	StatusSetupInProgress     = "setup_in_progress"
	StatusSetupComplete       = "setup_complete"
)

// State captures first-run bootstrap lifecycle.
type State struct {
	ID                           uuid.UUID  `db:"id" json:"id"`
	Status                       string     `db:"status" json:"status"`
	SetupRequired                bool       `db:"setup_required" json:"setup_required"`
	SeedVersion                  int        `db:"seed_version" json:"seed_version"`
	InitialAdminUserID           *uuid.UUID `db:"initial_admin_user_id" json:"initial_admin_user_id,omitempty"`
	InitialAdminPasswordIssuedAt *time.Time `db:"initial_admin_password_issued_at" json:"initial_admin_password_issued_at,omitempty"`
	SetupCompletedAt             *time.Time `db:"setup_completed_at" json:"setup_completed_at,omitempty"`
	CreatedAt                    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                    time.Time  `db:"updated_at" json:"updated_at"`
}

// Service manages first-run bootstrap state.
type Service struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewService(db *sqlx.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

func (s *Service) GetState(ctx context.Context) (*State, error) {
	var state State
	err := s.db.GetContext(ctx, &state, `
		SELECT id, status, setup_required, seed_version,
		       initial_admin_user_id, initial_admin_password_issued_at, setup_completed_at,
		       created_at, updated_at
		FROM system_bootstrap_state
		ORDER BY created_at ASC
		LIMIT 1
	`)
	if err != nil {
		if isMissingBootstrapTableError(err) {
			return nil, nil
		}
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (s *Service) IsSetupRequired(ctx context.Context) (bool, error) {
	state, err := s.GetState(ctx)
	if err != nil {
		return false, err
	}
	if state == nil {
		return false, nil
	}
	return state.SetupRequired, nil
}

// EnsureInitialized initializes first-run bootstrap state and local admin password exactly once.
func (s *Service) EnsureInitialized(ctx context.Context, adminEmail string) (*State, string, error) {
	if adminEmail == "" {
		adminEmail = "admin@imgfactory.com"
	}

	state, err := s.GetState(ctx)
	if err != nil {
		return nil, "", err
	}
	if state != nil {
		return state, "", nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var adminUserID uuid.UUID
	if err := tx.GetContext(ctx, &adminUserID, `
		SELECT id FROM users WHERE email = $1 LIMIT 1
	`, adminEmail); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("admin user %s not found", adminEmail)
		}
		return nil, "", err
	}

	plainPassword, err := generateInitialPassword(32)
	if err != nil {
		return nil, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $1,
		    is_ldap_user = false,
		    auth_method = 'credentials',
		    must_change_password = true,
		    failed_login_count = 0,
		    locked_until = NULL,
		    status = 'active',
		    password_changed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, string(hash), adminUserID); err != nil {
		return nil, "", err
	}

	stateID := uuid.New()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO system_bootstrap_state (
			id, status, setup_required, seed_version,
			initial_admin_user_id, initial_admin_password_issued_at,
			created_at, updated_at
		) VALUES (
			$1, $2, true, 1,
			$3, CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`, stateID, StatusAdminPasswordIssued, adminUserID); err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	tx = nil

	state, err = s.GetState(ctx)
	if err != nil {
		return nil, "", err
	}

	return state, plainPassword, nil
}

func (s *Service) MarkSetupInProgress(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE system_bootstrap_state
		SET status = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE setup_required = true
	`, StatusSetupInProgress)
	return err
}

func (s *Service) MarkSetupComplete(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE system_bootstrap_state
		SET status = $1,
		    setup_required = false,
		    setup_completed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE setup_required = true
	`, StatusSetupComplete)
	return err
}

// ReissueInitialAdminPassword rotates the local admin password and marks it as must_change_password.
// This is intended for first-run recovery when the generated password was lost.
func (s *Service) ReissueInitialAdminPassword(ctx context.Context, adminEmail string) (string, error) {
	if adminEmail == "" {
		adminEmail = "admin@imgfactory.com"
	}

	state, err := s.GetState(ctx)
	if err != nil {
		return "", err
	}
	if state == nil {
		return "", fmt.Errorf("bootstrap state not initialized")
	}
	if !state.SetupRequired {
		return "", fmt.Errorf("setup is already complete; refusing to reissue bootstrap password")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var adminUserID uuid.UUID
	if err := tx.GetContext(ctx, &adminUserID, `
		SELECT id FROM users WHERE email = $1 LIMIT 1
	`, adminEmail); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("admin user %s not found", adminEmail)
		}
		return "", err
	}

	plainPassword, err := generateInitialPassword(32)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $1,
		    is_ldap_user = false,
		    auth_method = 'credentials',
		    must_change_password = true,
		    failed_login_count = 0,
		    locked_until = NULL,
		    status = 'active',
		    password_changed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, string(hash), adminUserID); err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE system_bootstrap_state
		SET status = $1,
		    initial_admin_user_id = $2,
		    initial_admin_password_issued_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, StatusAdminPasswordIssued, adminUserID, state.ID); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	tx = nil

	return plainPassword, nil
}

func generateInitialPassword(length int) (string, error) {
	if length < 16 {
		length = 16
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = charset[int(random[i])%len(charset)]
	}
	return string(buf), nil
}

func isMissingBootstrapTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "system_bootstrap_state") &&
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
