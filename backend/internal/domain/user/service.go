package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// SystemConfigService defines the interface for system configuration operations
type SystemConfigService interface {
	GetSecurityConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.SecurityConfig, error)
	GetConfigsByType(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error)
}

// SecurityConfig represents security-related configuration settings
type SecurityConfig = systemconfig.SecurityConfig

// Token TTL Configuration - Default values (can be overridden by system config)
const (
	DefaultAccessTokenTTL  = 15 * time.Minute   // Default short-lived access token
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour // Default long-lived refresh token
)

// Service defines the business logic for user authentication and management
type Service struct {
	repository              Repository
	invitationRepository    UserInvitationRepository
	passwordResetRepository PasswordResetTokenRepository
	notificationService     *email.NotificationService
	systemConfigService     SystemConfigService
	logger                  *zap.Logger
	jwtSecret               []byte
	accessTokenTTL          time.Duration
	refreshTokenTTL         time.Duration
	frontendBaseURL         string
}

// NewService creates a new user service with default token TTL
func NewService(repository Repository, logger *zap.Logger, jwtSecret string) *Service {
	return &Service{
		repository:      repository,
		logger:          logger,
		jwtSecret:       []byte(jwtSecret),
		accessTokenTTL:  DefaultAccessTokenTTL,
		refreshTokenTTL: DefaultRefreshTokenTTL,
		frontendBaseURL: "http://localhost:3000", // Default value
	}
}

// NewServiceWithTokenTTL creates a new user service with custom token TTL
func NewServiceWithTokenTTL(repository Repository, logger *zap.Logger, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) *Service {
	return &Service{
		repository:      repository,
		logger:          logger,
		jwtSecret:       []byte(jwtSecret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// NewServiceWithInvitations creates a new user service with invitation support
func NewServiceWithInvitations(repository Repository, invitationRepository UserInvitationRepository, notificationService *email.NotificationService, logger *zap.Logger, jwtSecret string) *Service {
	return &Service{
		repository:           repository,
		invitationRepository: invitationRepository,
		notificationService:  notificationService,
		logger:               logger,
		jwtSecret:            []byte(jwtSecret),
		accessTokenTTL:       DefaultAccessTokenTTL,
		refreshTokenTTL:      DefaultRefreshTokenTTL,
	}
}

// NewServiceWithInvitationsAndTokenTTL creates a new user service with invitation support and custom token TTL
func NewServiceWithInvitationsAndTokenTTL(repository Repository, invitationRepository UserInvitationRepository, notificationService *email.NotificationService, logger *zap.Logger, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) *Service {
	return &Service{
		repository:           repository,
		invitationRepository: invitationRepository,
		notificationService:  notificationService,
		logger:               logger,
		jwtSecret:            []byte(jwtSecret),
		accessTokenTTL:       accessTokenTTL,
		refreshTokenTTL:      refreshTokenTTL,
	}
}

// NewServiceWithPasswordReset creates a new user service with password reset support
func NewServiceWithPasswordReset(repository Repository, invitationRepository UserInvitationRepository, passwordResetRepository PasswordResetTokenRepository, notificationService *email.NotificationService, logger *zap.Logger, jwtSecret string) *Service {
	return &Service{
		repository:              repository,
		invitationRepository:    invitationRepository,
		passwordResetRepository: passwordResetRepository,
		notificationService:     notificationService,
		logger:                  logger,
		jwtSecret:               []byte(jwtSecret),
		accessTokenTTL:          DefaultAccessTokenTTL,
		refreshTokenTTL:         DefaultRefreshTokenTTL,
	}
}

// NewServiceWithConfig creates a new user service with system configuration support
func NewServiceWithConfig(repository Repository, systemConfigService SystemConfigService, logger *zap.Logger, jwtSecret string) *Service {
	return &Service{
		repository:          repository,
		systemConfigService: systemConfigService,
		logger:              logger,
		jwtSecret:           []byte(jwtSecret),
		accessTokenTTL:      DefaultAccessTokenTTL,
		refreshTokenTTL:     DefaultRefreshTokenTTL,
	}
}

// NewServiceWithInvitationsAndConfig creates a new user service with invitation support and system configuration
func NewServiceWithInvitationsAndConfig(repository Repository, invitationRepository UserInvitationRepository, notificationService *email.NotificationService, systemConfigService SystemConfigService, logger *zap.Logger, jwtSecret string, frontendBaseURL string) *Service {
	return &Service{
		repository:           repository,
		invitationRepository: invitationRepository,
		notificationService:  notificationService,
		systemConfigService:  systemConfigService,
		logger:               logger,
		jwtSecret:            []byte(jwtSecret),
		accessTokenTTL:       DefaultAccessTokenTTL,
		refreshTokenTTL:      DefaultRefreshTokenTTL,
		frontendBaseURL:      frontendBaseURL,
	}
}

// NewServiceWithInvitationsPasswordResetAndConfig creates a new user service with invitation support, password reset, and system configuration
func NewServiceWithInvitationsPasswordResetAndConfig(repository Repository, invitationRepository UserInvitationRepository, passwordResetRepository PasswordResetTokenRepository, notificationService *email.NotificationService, systemConfigService SystemConfigService, logger *zap.Logger, jwtSecret string, frontendBaseURL string) *Service {
	return &Service{
		repository:              repository,
		invitationRepository:    invitationRepository,
		passwordResetRepository: passwordResetRepository,
		notificationService:     notificationService,
		systemConfigService:     systemConfigService,
		logger:                  logger,
		jwtSecret:               []byte(jwtSecret),
		accessTokenTTL:          DefaultAccessTokenTTL,
		refreshTokenTTL:         DefaultRefreshTokenTTL,
		frontendBaseURL:         frontendBaseURL,
	}
}

// AuthResult represents the result of an authentication attempt
type AuthResult struct {
	User                   *User
	AccessToken            string
	RefreshToken           string
	RequiresMFA            bool
	RequiresPasswordChange bool
}

// PasswordResetTokenValidation represents the validation result of a password reset token
type PasswordResetTokenValidation struct {
	Valid     bool
	Email     string
	ExpiresAt string
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string
	Password string
	MFAToken string // Optional MFA token
}

// Login authenticates a user and returns authentication tokens
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResult, error) {
	// Find user by email
	user, err := s.repository.FindByEmail(ctx, req.Email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check if account is active
	if !user.IsActive() {
		return nil, ErrAccountDisabled
	}

	// Check if account is locked
	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	// Verify password
	if !user.VerifyPassword(req.Password) {
		user.RecordFailedLogin()
		if err := s.repository.Update(ctx, user); err != nil {
			s.logger.Error("Failed to update user after failed login",
				zap.String("user_id", user.ID().String()),
				zap.Error(err),
			)
		}
		return nil, errors.New("invalid credentials")
	}

	// Check if MFA is required
	if user.IsMFAEnabled() && req.MFAToken == "" {
		return &AuthResult{
			User:        user,
			RequiresMFA: true,
		}, nil
	}

	// If MFA is enabled, verify the token (simplified - in real implementation, use proper TOTP verification)
	if user.IsMFAEnabled() {
		if !s.verifyMFAToken(user, req.MFAToken) {
			user.RecordFailedLogin()
			if err := s.repository.Update(ctx, user); err != nil {
				s.logger.Error("Failed to update user after failed MFA",
					zap.String("user_id", user.ID().String()),
					zap.Error(err),
				)
			}
			return nil, errors.New("invalid MFA token")
		}
	}

	// Record successful login
	user.RecordLogin()
	if err := s.repository.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user after successful login",
			zap.String("user_id", user.ID().String()),
			zap.Error(err),
		)
		return nil, err
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	s.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID().String()),
		zap.String("email", user.Email()),
	)

	return &AuthResult{
		User:                   user,
		AccessToken:            accessToken,
		RefreshToken:           refreshToken,
		RequiresMFA:            false,
		RequiresPasswordChange: user.MustChangePassword(),
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid user ID in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	// Find user
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user is still active
	if !user.IsActive() {
		return nil, ErrAccountDisabled
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:                   user,
		AccessToken:            accessToken,
		RefreshToken:           newRefreshToken,
		RequiresMFA:            false,
		RequiresPasswordChange: user.MustChangePassword(),
	}, nil
}

// ValidateToken validates an access token and returns the user
func (s *Service) ValidateToken(ctx context.Context, accessToken string) (*User, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		// Check if token is expired
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token has expired, please login again")
		}
		return nil, errors.New("invalid access token")
	}

	if !token.Valid {
		return nil, errors.New("invalid access token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid user ID in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user is still active
	if !user.IsActive() {
		return nil, ErrAccountDisabled
	}

	return user, nil
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, tenantID uuid.UUID, email, firstName, lastName, password string) (*User, error) {
	// Check if user already exists
	exists, err := s.repository.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	// Create new user
	user, err := NewUser(email, firstName, lastName, password)
	if err != nil {
		return nil, err
	}

	// Save user
	if err := s.repository.Save(ctx, user); err != nil {
		return nil, err
	}

	s.logger.Info("User created successfully",
		zap.String("user_id", user.ID().String()),
		zap.String("email", user.Email()),
		zap.String("tenant_id", tenantID.String()),
	)

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repository.FindByID(ctx, id)
}

// GetUserByEmail retrieves a user by email address
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.repository.FindByEmail(ctx, email)
}

// GetUsersByTenantID retrieves all users for a tenant
func (s *Service) GetUsersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*User, error) {
	return s.repository.FindByTenantID(ctx, tenantID)
}

// GetAllUsers retrieves all users in the system (used by system administrators)
func (s *Service) GetAllUsers(ctx context.Context) ([]*User, error) {
	return s.repository.FindAll(ctx)
}

// UpdateUser updates user information
func (s *Service) UpdateUser(ctx context.Context, user *User) error {
	// Prevent edits to suspended users, but allow status updates (suspend/reactivate operations)
	// This is determined by checking if the user in the database is different from what we're trying to save
	dbUser, err := s.repository.FindByID(ctx, user.ID())
	if err != nil {
		return err
	}

	// Allow updates if:
	// 1. User is not currently suspended, OR
	// 2. User is currently suspended but the status is being changed (suspend/reactivate operation), OR
	// 3. Only the status field is being changed
	if dbUser.Status() == UserStatusSuspended && user.Status() == UserStatusSuspended {
		return errors.New("cannot modify suspended user accounts; reactivate user first")
	}

	return s.repository.Update(ctx, user)
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repository.Delete(ctx, id)
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := user.ChangePassword(newPassword); err != nil {
		return err
	}

	return s.repository.Update(ctx, user)
}

// EnableMFA enables MFA for a user
func (s *Service) EnableMFA(ctx context.Context, userID uuid.UUID, mfaType MFAType, secret string) error {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.EnableMFA(mfaType, secret)
	return s.repository.Update(ctx, user)
}

// DisableMFA disables MFA for a user
func (s *Service) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.DisableMFA()
	return s.repository.Update(ctx, user)
}

// VerifyEmail marks a user's email as verified
func (s *Service) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.VerifyEmail()
	return s.repository.Update(ctx, user)
}

// UnlockAccount unlocks a locked user account
func (s *Service) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.UnlockAccount()
	return s.repository.Update(ctx, user)
}

// generateTokens generates access and refresh tokens for a user
func (s *Service) generateTokens(ctx context.Context, user *User) (string, string, error) {
	// Get configurable TTL values from system config if available
	accessTTL := s.accessTokenTTL
	refreshTTL := s.refreshTokenTTL

	// Per-tenant JWT config selection should be derived from explicit tenant context,
	// not legacy user attributes. Use service defaults when no tenant context is provided.

	// Access token (short-lived)
	accessClaims := jwt.MapClaims{
		"user_id": user.ID().String(),
		"email":   user.Email(),
		"exp":     time.Now().Add(accessTTL).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "access",
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (long-lived)
	refreshClaims := jwt.MapClaims{
		"user_id": user.ID().String(),
		"exp":     time.Now().Add(refreshTTL).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "refresh",
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

// verifyMFAToken verifies an MFA token (simplified implementation)
func (s *Service) verifyMFAToken(user *User, token string) bool {
	// In a real implementation, this would verify TOTP, SMS codes, etc.
	// For now, just check if token is not empty and user has MFA enabled
	return user.IsMFAEnabled() && token != ""
}

// PasswordResetService defines methods for password reset functionality
type PasswordResetService interface {
	RequestPasswordReset(ctx context.Context, email, ipAddress, userAgent string) error
	ResetPassword(ctx context.Context, token, newPassword, ipAddress, userAgent string) error
	ValidateResetToken(ctx context.Context, token string) (*PasswordResetTokenValidation, error)
}

// RequestPasswordReset initiates a password reset for a user
func (s *Service) RequestPasswordReset(ctx context.Context, email, ipAddress, userAgent string) error {
	// Find user by email
	user, err := s.repository.FindByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			// If the email domain is managed by LDAP, direct users to corporate reset flow.
			if ldapManaged, ldapErr := s.isAllowedLDAPDomainEmail(ctx, email); ldapErr == nil && ldapManaged {
				s.logger.Warn("Password reset requested for LDAP-managed email domain without local account",
					zap.String("email", email),
					zap.String("ip_address", ipAddress),
				)
				return ErrPasswordResetNotAllowed
			}
			// Don't reveal if email exists or not for security
			s.logger.Info("Password reset requested for non-existent email",
				zap.String("email", email),
				zap.String("ip_address", ipAddress),
			)
			return nil
		}
		return err
	}

	// Check if user is active
	if !user.IsActive() {
		s.logger.Warn("Password reset requested for inactive user",
			zap.String("user_id", user.ID().String()),
			zap.String("email", email),
		)
		return nil // Don't reveal account status
	}

	// Check if user is LDAP-authenticated - LDAP users cannot reset passwords
	if user.AuthMethod() == AuthMethodLDAP {
		s.logger.Warn("Password reset requested for LDAP user",
			zap.String("user_id", user.ID().String()),
			zap.String("email", email),
		)
		return ErrPasswordResetNotAllowed
	}

	// Create password reset token
	token, plainToken, err := NewPasswordResetToken(user.ID(), ipAddress, userAgent)
	if err != nil {
		return err
	}

	// Save token to repository
	if s.passwordResetRepository != nil {
		if err := s.passwordResetRepository.Save(ctx, token); err != nil {
			s.logger.Error("Failed to save password reset token",
				zap.String("user_id", user.ID().String()),
				zap.Error(err))
			return err
		}
	} else {
		s.logger.Warn("Password reset repository not configured, token not persisted")
	}

	// Send email with reset link
	if s.notificationService != nil {
		resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendBaseURL, plainToken)
		subject := "Password Reset Request"
		bodyText := fmt.Sprintf("Hello,\n\nYou have requested to reset your password. Please click the following link to reset your password:\n\n%s\n\nThis link will expire in 24 hours.\n\nIf you did not request this password reset, please ignore this email.", resetLink)
		bodyHTML := fmt.Sprintf("<p>Hello,</p><p>You have requested to reset your password. Please click the following link to reset your password:</p><p><a href=\"%s\">Reset Password</a></p><p>This link will expire in 24 hours.</p><p>If you did not request this password reset, please ignore this email.</p>", resetLink)

		if err := s.notificationService.SendEmail(ctx, uuid.Nil, user.Email(), subject, bodyText, bodyHTML, 1); err != nil {
			s.logger.Error("Failed to send password reset email",
				zap.String("user_id", user.ID().String()),
				zap.String("email", user.Email()),
				zap.Error(err))
			// Don't return error - token was created successfully
		}
	} else {
		s.logger.Warn("Notification service not configured, password reset email not sent")
	}

	s.logger.Info("Password reset token created",
		zap.String("user_id", user.ID().String()),
		zap.String("email", email),
		zap.String("token_id", token.ID().String()),
	)

	// For development/testing, log the plain token
	s.logger.Info("Password reset token (for development only)",
		zap.String("token", plainToken),
		zap.String("reset_link", fmt.Sprintf("https://app.example.com/reset-password?token=%s", plainToken)),
	)

	return nil
}

func (s *Service) isAllowedLDAPDomainEmail(ctx context.Context, email string) (bool, error) {
	if s.systemConfigService == nil {
		return false, nil
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, nil
	}
	domain := strings.ToLower(strings.TrimSpace(parts[1]))
	if domain == "" {
		return false, nil
	}

	configs, err := s.systemConfigService.GetConfigsByType(ctx, nil, systemconfig.ConfigTypeLDAP)
	if err != nil {
		return false, err
	}

	for _, cfg := range configs {
		if cfg.Status() != systemconfig.ConfigStatusActive {
			continue
		}
		ldapCfg, parseErr := cfg.GetLDAPConfig()
		if parseErr != nil || ldapCfg == nil || !ldapCfg.Enabled {
			continue
		}

		for _, allowed := range ldapCfg.AllowedDomains {
			if domain == strings.ToLower(strings.TrimSpace(allowed)) {
				return true, nil
			}
		}
	}

	return false, nil
}

// ResetPassword resets a user's password using a valid reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword, ipAddress, userAgent string) error {
	// Validate new password first
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash the token to find it in the repository
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Find token by hash
	resetToken, err := s.passwordResetRepository.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		s.logger.Error("Failed to find password reset token",
			zap.String("token_hash", tokenHash),
			zap.Error(err))
		return ErrTokenNotFound
	}

	// Verify token is valid
	if resetToken.Status() != PasswordResetTokenStatusActive {
		s.logger.Warn("Attempted to use inactive password reset token",
			zap.String("token_id", resetToken.ID().String()),
			zap.String("status", string(resetToken.Status())))
		return ErrTokenAlreadyUsed
	}

	if time.Now().UTC().After(resetToken.ExpiresAt()) {
		// Mark token as expired
		resetToken.MarkExpired()
		if err := s.passwordResetRepository.Update(ctx, resetToken); err != nil {
			s.logger.Error("Failed to mark expired token",
				zap.String("token_id", resetToken.ID().String()),
				zap.Error(err))
		}
		return ErrTokenExpired
	}

	// Get the user
	user, err := s.repository.FindByID(ctx, resetToken.UserID())
	if err != nil {
		s.logger.Error("Failed to find user for password reset",
			zap.String("user_id", resetToken.UserID().String()),
			zap.Error(err))
		return err
	}

	// Check if user is LDAP-authenticated - LDAP users cannot reset passwords
	if user.AuthMethod() == AuthMethodLDAP {
		s.logger.Warn("Password reset attempted for LDAP user",
			zap.String("user_id", user.ID().String()),
			zap.String("token_id", resetToken.ID().String()),
		)
		return ErrTokenNotFound // Return generic error for security
	}

	// Update user's password
	if err := user.ChangePassword(newPassword); err != nil {
		s.logger.Error("Failed to set new password",
			zap.String("user_id", user.ID().String()),
			zap.Error(err))
		return err
	}

	if err := s.repository.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user with new password",
			zap.String("user_id", user.ID().String()),
			zap.Error(err))
		return err
	}

	// Mark token as used
	now := time.Now().UTC()
	resetToken.MarkUsed(ipAddress, userAgent, now)
	if err := s.passwordResetRepository.Update(ctx, resetToken); err != nil {
		s.logger.Error("Failed to mark token as used",
			zap.String("token_id", resetToken.ID().String()),
			zap.Error(err))
		// Don't return error - password was successfully changed
	}

	s.logger.Info("Password reset successful",
		zap.String("user_id", user.ID().String()),
		zap.String("token_id", resetToken.ID().String()),
		zap.String("ip_address", ipAddress))

	return nil
}

// ValidateResetToken validates a password reset token and returns validation information
func (s *Service) ValidateResetToken(ctx context.Context, token string) (*PasswordResetTokenValidation, error) {
	// Hash the token to find it in the repository
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Find token by hash
	resetToken, err := s.passwordResetRepository.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		s.logger.Error("Failed to find password reset token",
			zap.String("token_hash", tokenHash),
			zap.Error(err))
		return &PasswordResetTokenValidation{Valid: false}, nil
	}

	// Check if token is active
	if resetToken.Status() != PasswordResetTokenStatusActive {
		s.logger.Warn("Attempted to validate inactive password reset token",
			zap.String("token_id", resetToken.ID().String()),
			zap.String("status", string(resetToken.Status())))
		return &PasswordResetTokenValidation{Valid: false}, nil
	}

	// Check if token is expired
	if time.Now().UTC().After(resetToken.ExpiresAt()) {
		// Mark token as expired
		resetToken.MarkExpired()
		if err := s.passwordResetRepository.Update(ctx, resetToken); err != nil {
			s.logger.Error("Failed to mark expired token",
				zap.String("token_id", resetToken.ID().String()),
				zap.Error(err))
		}
		return &PasswordResetTokenValidation{Valid: false}, nil
	}

	// Get the user to get their email
	user, err := s.repository.FindByID(ctx, resetToken.UserID())
	if err != nil {
		s.logger.Error("Failed to find user for token validation",
			zap.String("user_id", resetToken.UserID().String()),
			zap.Error(err))
		return &PasswordResetTokenValidation{Valid: false}, nil
	}

	// Check if user is LDAP-authenticated - LDAP users cannot reset passwords
	if user.AuthMethod() == AuthMethodLDAP {
		s.logger.Warn("Token validation attempted for LDAP user",
			zap.String("user_id", user.ID().String()),
			zap.String("token_id", resetToken.ID().String()),
		)
		return &PasswordResetTokenValidation{Valid: false}, nil
	}

	return &PasswordResetTokenValidation{
		Valid:     true,
		Email:     user.Email(),
		ExpiresAt: resetToken.ExpiresAt().Format(time.RFC3339),
	}, nil
}

// GenerateAccessToken generates an access token for a user
func (s *Service) GenerateAccessToken(ctx context.Context, user *User) (string, error) {
	accessToken, _, err := s.generateTokens(ctx, user)
	return accessToken, err
}

type UserInvitationService interface {
	CreateInvitation(ctx context.Context, tenantID, invitedBy uuid.UUID, email, roleID, message string) (*UserInvitation, string, error)
	AcceptInvitation(ctx context.Context, token, firstName, lastName, password string, isLDAP bool, ipAddress, userAgent string) (*User, error)
	GetInvitationByToken(ctx context.Context, token string) (*UserInvitation, error)
	CancelInvitation(ctx context.Context, invitationID uuid.UUID) error
	ListInvitationsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*UserInvitation, error)
	ResendInvitation(ctx context.Context, invitationID uuid.UUID) error
	GenerateAccessToken(ctx context.Context, user *User) (string, error)
}

// CreateInvitation creates a new user invitation
func (s *Service) CreateInvitation(ctx context.Context, tenantID, invitedBy uuid.UUID, email, roleID, message string) (*UserInvitation, string, error) {
	// Note: We allow inviting existing users - no check for email existence
	// Both new and existing users must accept the invitation before being added to the tenant

	// Check if invitation already exists for this email and tenant
	if s.invitationRepository != nil {
		exists, err := s.invitationRepository.ExistsByEmailAndTenant(ctx, email, tenantID)
		if err != nil {
			return nil, "", err
		}
		if exists {
			return nil, "", ErrInvitationAlreadyExists
		}
	}

	// Parse role ID
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return nil, "", errors.New("invalid role ID")
	}

	// Create invitation
	invitation, plainToken, err := NewUserInvitation(tenantID, email, roleUUID, invitedBy, message)
	if err != nil {
		return nil, "", err
	}

	// Save invitation to repository
	if s.invitationRepository != nil {
		err = s.invitationRepository.CreateInvitation(ctx, invitation, plainToken)
		if err != nil {
			return nil, "", err
		}
	}

	// Send invitation email
	if s.notificationService != nil {
		// Get tenant name for email
		tenantName := "Your Organization" // Default fallback

		// Get inviter name
		inviterName := "System Administrator" // Default fallback
		if inviter, err := s.repository.FindByID(ctx, invitedBy); err == nil && inviter != nil {
			inviterName = inviter.FirstName() + " " + inviter.LastName()
		}

		// Create invitation URL
		invitationURL := fmt.Sprintf("%s/accept-invitation?token=%s", s.frontendBaseURL, plainToken)

		// Try to send email - log warning if it fails but don't fail the invitation
		if err := s.notificationService.SendUserInvitationEmail(ctx, email, tenantName, inviterName, invitationURL, message, invitation.ExpiresAt().Format("January 2, 2006 at 3:04 PM"), tenantID); err != nil {
			s.logger.Warn("Failed to send invitation email", zap.Error(err))
		}
	}

	s.logger.Info("User invitation created",
		zap.String("invitation_id", invitation.ID().String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("email", email),
		zap.String("invited_by", invitedBy.String()),
	)

	return invitation, plainToken, nil
}

// AcceptInvitation accepts a user invitation
// For new users: creates account and assigns role
// For existing users: just creates role assignment in the tenant
func (s *Service) AcceptInvitation(ctx context.Context, token, firstName, lastName, password string, isLDAP bool, ipAddress, userAgent string) (*User, error) {
	if s.invitationRepository == nil {
		return nil, errors.New("invitation repository not available")
	}

	// Find invitation by token
	invitation, err := s.invitationRepository.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Verify invitation is still valid
	if invitation.Status() != UserInvitationStatusPending {
		return nil, errors.New("invitation is no longer valid")
	}

	if time.Now().After(invitation.ExpiresAt()) {
		return nil, errors.New("invitation has expired")
	}

	// Check if user already exists by email
	existingUser, err := s.repository.FindByEmail(ctx, invitation.Email())
	var user *User

	if err != nil && err.Error() != "user not found" && err.Error() != "sql: no rows in result set" {
		// Some other error occurred
		return nil, err
	}

	if existingUser != nil {
		// Existing user: use the found user
		user = existingUser
		s.logger.Info("Existing user accepted invitation",
			zap.String("user_id", user.ID().String()),
			zap.String("email", invitation.Email()),
			zap.String("tenant_id", invitation.TenantID().String()),
			zap.Bool("is_ldap", isLDAP),
		)
	} else {
		// New user: create account
		// For LDAP users, use a dummy password (like in LDAP login flow)
		// For regular users, use the provided password
		passwordToUse := password
		if isLDAP {
			passwordToUse = uuid.New().String() // Dummy password for LDAP users
			s.logger.Info("Creating LDAP user via invitation - using dummy password",
				zap.String("email", invitation.Email()),
			)
		}

		user, err = s.CreateUser(ctx, invitation.TenantID(), invitation.Email(), firstName, lastName, passwordToUse)
		if err != nil {
			return nil, err
		}

		// Set auth method based on user type
		if isLDAP {
			user.SetAuthMethod(AuthMethodLDAP)
		} else {
			user.SetAuthMethod(AuthMethodCredentials)
		}

		s.logger.Info("New user created via invitation",
			zap.String("user_id", user.ID().String()),
			zap.String("email", invitation.Email()),
			zap.String("tenant_id", invitation.TenantID().String()),
			zap.Bool("is_ldap", isLDAP),
			zap.String("auth_method", string(user.AuthMethod())),
		)
	}

	// Assign role to user if specified in invitation
	if invitation.RoleID() != uuid.Nil {
		// Note: Role assignment would be handled by the RBAC system
		// This might require additional service calls to assign the role
		s.logger.Info("User accepted invitation with role assignment pending",
			zap.String("user_id", user.ID().String()),
			zap.String("role_id", invitation.RoleID().String()),
			zap.String("tenant_id", invitation.TenantID().String()),
		)
	}

	// Mark invitation as accepted
	now := time.Now()
	err = s.invitationRepository.UpdateInvitationStatus(ctx, invitation.ID(), UserInvitationStatusAccepted, &now)
	if err != nil {
		// Log error but don't fail the operation since user was created/found successfully
		s.logger.Error("Failed to update invitation status", zap.Error(err), zap.String("invitation_id", invitation.ID().String()))
	}

	s.logger.Info("Invitation accepted successfully",
		zap.String("invitation_id", invitation.ID().String()),
		zap.String("user_id", user.ID().String()),
		zap.String("email", invitation.Email()),
	)

	return user, nil
}

// GetInvitationByToken retrieves an invitation by its token
func (s *Service) GetInvitationByToken(ctx context.Context, token string) (*UserInvitation, error) {
	if s.invitationRepository == nil {
		return nil, errors.New("invitation repository not available")
	}

	return s.invitationRepository.GetInvitationByToken(ctx, token)
}

// CancelInvitation cancels a pending invitation
func (s *Service) CancelInvitation(ctx context.Context, invitationID uuid.UUID) error {
	if s.invitationRepository == nil {
		return errors.New("invitation repository not available")
	}

	// Get the invitation first to check if it exists and is in pending status
	invitation, err := s.invitationRepository.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}

	if invitation.Status() != UserInvitationStatusPending {
		return errors.New("can only cancel pending invitations")
	}

	// Update the invitation status to cancelled
	err = s.invitationRepository.UpdateInvitationStatus(ctx, invitationID, UserInvitationStatusCancelled, nil)
	if err != nil {
		return err
	}

	// Get tenant name for email
	tenantName := "Your Organization" // Default fallback

	// Try to send cancellation email - log warning if it fails but don't fail the operation
	if s.notificationService != nil {
		if err := s.notificationService.SendUserInvitationCancelledEmail(ctx, invitation.Email(), tenantName, invitation.TenantID()); err != nil {
			s.logger.Warn("Failed to send invitation cancellation email", zap.Error(err))
		}
	}

	return nil
}

// ResendInvitation resends an invitation email for a pending invitation
func (s *Service) ResendInvitation(ctx context.Context, invitationID uuid.UUID) error {
	if s.invitationRepository == nil {
		return errors.New("invitation repository not available")
	}

	// Get the invitation
	invitation, err := s.invitationRepository.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return err
	}

	if invitation.Status() != UserInvitationStatusPending {
		return errors.New("can only resend pending invitations")
	}

	// Regenerate a new token
	plainToken, err := invitation.RegenerateToken()
	if err != nil {
		return err
	}

	// Update the invitation token in the repository
	if err := s.invitationRepository.UpdateInvitationToken(ctx, invitationID, invitation.TokenHash()); err != nil {
		return err
	}

	// Get tenant name for email
	tenantName := "Your Organization" // Default fallback

	// Get inviter name
	inviterName := "System Administrator" // Default fallback
	if inviter, err := s.repository.FindByID(ctx, invitation.InvitedBy()); err == nil && inviter != nil {
		inviterName = inviter.FirstName() + " " + inviter.LastName()
	}

	// Create invitation URL with the new token
	invitationURL := fmt.Sprintf("%s/accept-invitation?token=%s", s.frontendBaseURL, plainToken)

	// Try to send email - log warning if it fails but don't fail the operation
	if err := s.notificationService.SendUserInvitationEmail(ctx, invitation.Email(), tenantName, inviterName, invitationURL, invitation.Message(), invitation.ExpiresAt().Format("January 2, 2006 at 3:04 PM"), invitation.TenantID()); err != nil {
		s.logger.Warn("Failed to resend invitation email", zap.Error(err))
		return err
	}

	return nil
}

// ListInvitationsByTenant lists all invitations for a tenant
func (s *Service) ListInvitationsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*UserInvitation, error) {
	s.logger.Info("ListInvitationsByTenant called", zap.String("tenant_id", tenantID.String()))
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant_id is required")
	}
	if s.invitationRepository == nil {
		s.logger.Error("Invitation repository not available")
		return nil, errors.New("invitation repository not available")
	}
	invitations, err := s.invitationRepository.ListInvitationsByTenant(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to get invitations from repository", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Found invitations", zap.Int("count", len(invitations)))
	return invitations, nil
}

// BulkOperationService defines methods for bulk user operations
type BulkOperationService interface {
	StartBulkImport(ctx context.Context, tenantID, initiatedBy uuid.UUID, users []BulkImportUser, metadata map[string]interface{}) (*BulkOperation, error)
	GetBulkOperationStatus(ctx context.Context, operationID uuid.UUID) (*BulkOperation, error)
	ListBulkOperations(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*BulkOperation, error)
	CancelBulkOperation(ctx context.Context, operationID uuid.UUID) error
}

// BulkImportUser represents a user to be imported in bulk
type BulkImportUser struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	RoleID    string `json:"role_id"`
}

// StartBulkImport starts a bulk user import operation
func (s *Service) StartBulkImport(ctx context.Context, tenantID, initiatedBy uuid.UUID, users []BulkImportUser, metadata map[string]interface{}) (*BulkOperation, error) {
	if len(users) == 0 {
		return nil, errors.New("no users to import")
	}

	// Create bulk operation
	operation, err := NewBulkOperation(tenantID, BulkOperationTypeImportUsers, initiatedBy, len(users), metadata)
	if err != nil {
		return nil, err
	}

	// TODO: Save operation to repository
	// TODO: Start background processing

	s.logger.Info("Bulk import operation started",
		zap.String("operation_id", operation.ID().String()),
		zap.String("tenant_id", tenantID.String()),
		zap.Int("user_count", len(users)),
		zap.String("initiated_by", initiatedBy.String()),
	)

	return operation, nil
}

// GetBulkOperationStatus retrieves the status of a bulk operation
func (s *Service) GetBulkOperationStatus(ctx context.Context, operationID uuid.UUID) (*BulkOperation, error) {
	// TODO: Find operation by ID
	return nil, ErrBulkOperationNotFound
}

// ListBulkOperations lists bulk operations for a tenant
func (s *Service) ListBulkOperations(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*BulkOperation, error) {
	// TODO: Find operations by tenant with pagination
	return []*BulkOperation{}, nil
}

// CancelBulkOperation cancels a running bulk operation
func (s *Service) CancelBulkOperation(ctx context.Context, operationID uuid.UUID) error {
	// TODO: Find and cancel operation
	return ErrBulkOperationNotFound
}

// UserDeactivationService defines methods for user deactivation
type UserDeactivationService interface {
	DeactivateUser(ctx context.Context, userID, deactivatedBy uuid.UUID, reason UserDeactivationReason, notes, ipAddress, userAgent string) error
	ReactivateUser(ctx context.Context, userID, reactivatedBy uuid.UUID, ipAddress, userAgent string) error
	GetUserDeactivationHistory(ctx context.Context, userID uuid.UUID) ([]*UserDeactivation, error)
}

// DeactivateUser deactivates a user account
func (s *Service) DeactivateUser(ctx context.Context, userID, deactivatedBy uuid.UUID, reason UserDeactivationReason, notes, ipAddress, userAgent string) error {
	// Find user
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Check if user is already inactive
	if !user.IsActive() {
		return errors.New("user is already deactivated")
	}

	// Create deactivation record
	// TODO: Save deactivation record to repository when tenant scope is explicitly modeled.
	user.Deactivate()
	if err := s.repository.Update(ctx, user); err != nil {
		return err
	}

	// TODO: Save deactivation record to repository

	s.logger.Info("User deactivated",
		zap.String("user_id", userID.String()),
		zap.String("deactivated_by", deactivatedBy.String()),
		zap.String("reason", string(reason)),
	)

	return nil
}

// ReactivateUser reactivates a deactivated user account
func (s *Service) ReactivateUser(ctx context.Context, userID, reactivatedBy uuid.UUID, ipAddress, userAgent string) error {
	// Find user
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Check if user is active
	if user.IsActive() {
		return errors.New("user is already active")
	}

	// Find latest deactivation record
	// TODO: Get deactivation history and reactivate

	// Update user status to active
	user.Activate()
	if err := s.repository.Update(ctx, user); err != nil {
		return err
	}

	// TODO: Update deactivation record with reactivation info

	s.logger.Info("User reactivated",
		zap.String("user_id", userID.String()),
		zap.String("reactivated_by", reactivatedBy.String()),
	)

	return nil
}

// GetUserDeactivationHistory retrieves the deactivation history for a user
func (s *Service) GetUserDeactivationHistory(ctx context.Context, userID uuid.UUID) ([]*UserDeactivation, error) {
	// TODO: Find deactivation records by user ID
	return []*UserDeactivation{}, nil
}

// GetTotalUserCount returns the total number of users in the system
func (s *Service) GetTotalUserCount(ctx context.Context) (int, error) {
	return s.repository.GetTotalUserCount(ctx)
}

// GetActiveUserCount returns the number of users active within the specified number of days
func (s *Service) GetActiveUserCount(ctx context.Context, days int) (int, error) {
	return s.repository.GetActiveUserCount(ctx, days)
}

// SuspendUser suspends a user account and sends a notification
func (s *Service) SuspendUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.SuspendAccount()
	if err := s.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Send notification if notification service is available
	if s.notificationService != nil {
		// Get tenant name - for now, use a default or get from context
		// TODO: Get actual tenant name from tenant service
		tenantName := "Image Factory" // Default fallback

		data := &email.UserSuspendedData{
			UserEmail:   user.Email(),
			UserName:    user.FirstName() + " " + user.LastName(),
			TenantName:  tenantName,
			TenantID:    tenantID,
			SuspendedAt: time.Now().Format("January 2, 2006 at 3:04 PM"),
			Reason:      "Account suspended by administrator",
		}

		if err := s.notificationService.SendUserSuspendedEmail(ctx, data); err != nil {
			s.logger.Error("Failed to send user suspension notification", zap.Error(err))
			// Don't fail the operation if notification fails
		}
	}

	return nil
}

// ActivateUser activates a suspended user account and sends a notification
func (s *Service) ActivateUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	s.logger.Info("Activating user in service", zap.String("user_id", userID.String()), zap.String("tenant_id", tenantID.String()))

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := user.ReactivateAccount(); err != nil {
		return err
	}

	if err := s.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Send notification if notification service is available
	if s.notificationService != nil {
		// Get tenant name - for now, use a default or get from context
		// TODO: Get actual tenant name from tenant service
		tenantName := "Image Factory"           // Default fallback
		dashboardURL := "http://localhost:3000" // Default dashboard URL
		if s.frontendBaseURL != "" {
			dashboardURL = s.frontendBaseURL
		}

		data := &email.UserActivatedData{
			UserEmail:     user.Email(),
			UserName:      user.FirstName() + " " + user.LastName(),
			TenantName:    tenantName,
			TenantID:      tenantID,
			ReactivatedAt: time.Now().Format("January 2, 2006 at 3:04 PM"),
			DashboardURL:  dashboardURL,
		}

		if err := s.notificationService.SendUserActivatedEmail(ctx, data); err != nil {
			s.logger.Error("Failed to send user activation notification", zap.Error(err))
			// Don't fail the operation if notification fails
		}
	}

	return nil
}

// UpdateUserRoles updates all roles for a user
func (s *Service) UpdateUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	// Check if user is suspended before allowing role updates
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Status() == UserStatusSuspended {
		return errors.New("cannot modify roles for suspended user accounts; reactivate user first")
	}

	// TODO: Implement bulk role assignment logic
	// This should remove all existing roles for the user and assign the new ones
	return nil
}

// GetTenantUsers retrieves all users in a tenant
func (s *Service) GetTenantUsers(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*User, error) {
	users, err := s.repository.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Apply pagination manually
	if offset > len(users) {
		return []*User{}, nil
	}

	end := offset + limit
	if end > len(users) {
		end = len(users)
	}

	return users[offset:end], nil
}

// AddUserToTenant validates that a user can be added to a tenant
// The actual role assignment is handled by the caller (e.g., HTTP handler) using RBAC service
func (s *Service) AddUserToTenant(ctx context.Context, userID, tenantID uuid.UUID, roleIDs []uuid.UUID) error {
	// Check if user is suspended before allowing tenant assignment
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Status() == UserStatusSuspended {
		return errors.New("cannot add suspended user to tenant; reactivate user first")
	}

	// Validation passed - caller is responsible for role assignment
	return nil
}

// RemoveUserFromTenant validates that a user can be removed from a tenant
// The actual role cleanup is handled by the caller (e.g., HTTP handler) using RBAC service
func (s *Service) RemoveUserFromTenant(ctx context.Context, userID, tenantID uuid.UUID) error {
	// Verify user exists
	_, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// Validation passed - caller is responsible for role cleanup
	return nil
}
