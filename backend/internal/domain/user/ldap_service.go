package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/ldap"
)

// LDAPService extends the user service with LDAP authentication
type LDAPService struct {
	*Service
	ldapClient          *ldap.Client
	systemConfigService *systemconfig.Service
	logger              *zap.Logger
}

// NewLDAPService creates a new LDAP-enabled user service
func NewLDAPService(repository Repository, logger *zap.Logger, jwtSecret string, ldapClient *ldap.Client, systemConfigService *systemconfig.Service) *LDAPService {
	return &LDAPService{
		Service:             NewService(repository, logger, jwtSecret),
		ldapClient:          ldapClient,
		systemConfigService: systemConfigService,
		logger:              logger,
	}
}

// LDAPLogin authenticates a user against LDAP and creates/updates local user record.
// Users may be created without tenant assignment and must be assigned by an administrator.
func (s *LDAPService) LDAPLogin(ctx context.Context, req LoginRequest) (*AuthResult, error) {
	s.logger.Info("Starting LDAP login", zap.String("email", req.Email))

	if req.Email == "" || req.Password == "" {
		s.logger.Warn("LDAP login failed: missing email or password")
		return nil, errors.New("email and password are required")
	}

	// Extract username from email (before @)
	username := req.Email
	if atIndex := len(req.Email) - 1; atIndex > 0 {
		for i := atIndex; i >= 0; i-- {
			if req.Email[i] == '@' {
				username = req.Email[:i]
				break
			}
		}
	}

	s.logger.Debug("Extracted username from email",
		zap.String("email", req.Email),
		zap.String("username", username))

	// Validate email domain for LDAP login
	if err := s.validateLDAPEmailDomain(ctx, req.Email); err != nil {
		s.logger.Warn("LDAP login rejected: invalid email domain",
			zap.String("email", req.Email),
			zap.Error(err))
		return nil, errors.New("email domain not allowed for LDAP authentication")
	}

	// Authenticate against LDAP
	ldapUser, err := s.selectLDAPClient(ctx).Authenticate(ctx, username, req.Password)
	if err != nil {
		s.logger.Warn("LDAP authentication failed",
			zap.String("username", username),
			zap.String("email", req.Email),
			zap.Error(err))
		return nil, errors.New("invalid credentials")
	}

	s.logger.Info("LDAP authentication successful",
		zap.String("username", username),
		zap.String("email", req.Email),
		zap.String("ldap_dn", ldapUser.DN))

	// Check if user exists locally
	localUser, err := s.repository.FindByEmail(ctx, req.Email)
	if err != nil && err != ErrUserNotFound {
		return nil, err
	}

	// If user doesn't exist locally, create them
	if err == ErrUserNotFound {
		localUser, err = s.createUserFromLDAP(ctx, req.Email, ldapUser, req.Password)
		if err != nil {
			s.logger.Error("Failed to create user from LDAP",
				zap.String("email", req.Email),
				zap.Error(err))
			return nil, fmt.Errorf("failed to provision user account")
		}
	} else {
		// Update local user with latest LDAP info
		if err := s.updateUserFromLDAP(ctx, localUser, ldapUser); err != nil {
			s.logger.Warn("Failed to update user from LDAP",
				zap.String("email", req.Email),
				zap.Error(err))
			// Don't fail login if update fails
		}
	}

	// Check if account is active
	if !localUser.IsActive() {
		return nil, ErrAccountDisabled
	}

	// Check if account is locked
	if localUser.IsLocked() {
		return nil, ErrAccountLocked
	}

	// Record successful login
	localUser.RecordLogin()
	if err := s.repository.Update(ctx, localUser); err != nil {
		s.logger.Error("Failed to update user after successful login",
			zap.String("user_id", localUser.ID().String()),
			zap.Error(err))
		return nil, err
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(ctx, localUser)
	if err != nil {
		return nil, err
	}

	s.logger.Info("LDAP login successful",
		zap.String("user_id", localUser.ID().String()),
		zap.String("email", localUser.Email()),
		zap.Strings("ldap_groups", ldapUser.Groups))

	return &AuthResult{
		User:         localUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		RequiresMFA:  false, // LDAP handles MFA separately if needed
	}, nil
}

// createUserFromLDAP creates a local user from LDAP user info
// Users are created without a tenant - they will be assigned via group membership
func (s *LDAPService) createUserFromLDAP(ctx context.Context, email string, ldapUser *ldap.UserInfo, password string) (*User, error) {
	// Use a dummy password hash since LDAP authentication will be used
	// The actual password validation happens against LDAP
	dummyPassword := uuid.New().String()

	// Create user without tenant (tenantID = uuid.Nil)
	// User will be assigned to tenants through group membership
	// Use the email from the login request, not from LDAP
	// Provide fallback values for first/last name if not available from LDAP
	firstName := ldapUser.FirstName
	lastName := ldapUser.LastName
	if firstName == "" {
		firstName = "User"
	}
	if lastName == "" {
		lastName = "Account"
	}

	user, err := NewUser(email, firstName, lastName, dummyPassword)
	if err != nil {
		return nil, err
	}

	// Mark as active since LDAP authentication succeeded
	user.status = UserStatusActive

	// Set auth method to LDAP
	user.SetAuthMethod(AuthMethodLDAP)

	if err := s.repository.Save(ctx, user); err != nil {
		return nil, err
	}

	s.logger.Info("User created from LDAP",
		zap.String("user_id", user.ID().String()),
		zap.String("email", user.Email()),
		zap.String("auth_method", string(user.AuthMethod())))

	return user, nil
}

// updateUserFromLDAP updates local user with latest LDAP information
func (s *LDAPService) updateUserFromLDAP(ctx context.Context, user *User, ldapUser *ldap.UserInfo) error {
	// Update user details if they've changed
	needsUpdate := false

	if user.FirstName() != ldapUser.FirstName {
		// Note: User domain doesn't have UpdateDetails method
		// This would need to be added to the domain model
		s.logger.Info("User first name changed in LDAP",
			zap.String("user_id", user.ID().String()),
			zap.String("old", user.FirstName()),
			zap.String("new", ldapUser.FirstName))
		needsUpdate = true
	}

	if user.LastName() != ldapUser.LastName {
		s.logger.Info("User last name changed in LDAP",
			zap.String("user_id", user.ID().String()),
			zap.String("old", user.LastName()),
			zap.String("new", ldapUser.LastName))
		needsUpdate = true
	}

	if needsUpdate {
		// TODO: Add UpdateDetails method to User domain
		// For now, just log the change
		s.logger.Info("User details need updating from LDAP",
			zap.String("user_id", user.ID().String()))
	}

	return nil
}

// EnsureLocalUser ensures a local LDAP user exists for the given email.
// If the user exists and is LDAP-authenticated, it updates details if needed.
// If the user doesn't exist, it creates a new LDAP user with a dummy password.
func (s *LDAPService) EnsureLocalUser(ctx context.Context, email string) (*User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	if err := s.validateLDAPEmailDomain(ctx, email); err != nil {
		return nil, err
	}

	ldapUsers, err := s.SearchLDAPUsers(ctx, email, 1)
	if err != nil {
		return nil, err
	}
	if len(ldapUsers) == 0 {
		return nil, errors.New("ldap user not found")
	}

	ldapUser := ldapUsers[0]

	existingUser, err := s.repository.FindByEmail(ctx, email)
	if err != nil && err.Error() != "user not found" && err.Error() != "sql: no rows in result set" {
		return nil, err
	}

	if existingUser != nil {
		if existingUser.AuthMethod() != AuthMethodLDAP {
			return nil, errors.New("user exists with non-ldap auth method")
		}
		_ = s.updateUserFromLDAP(ctx, existingUser, ldapUser)
		return existingUser, nil
	}

	return s.createUserFromLDAP(ctx, email, ldapUser, "")
}

// GetLDAPUserInfo retrieves user information from LDAP without authentication
func (s *LDAPService) GetLDAPUserInfo(ctx context.Context, username string) (*ldap.UserInfo, error) {
	return s.selectLDAPClient(ctx).SearchUser(ctx, username)
}

// SearchLDAPUsers searches for LDAP users matching a query
func (s *LDAPService) SearchLDAPUsers(ctx context.Context, query string, limit int) ([]*ldap.UserInfo, error) {
	return s.selectLDAPClient(ctx).SearchUsers(ctx, query, limit)
}

func (s *LDAPService) selectLDAPClient(ctx context.Context) *ldap.Client {
	if s.systemConfigService == nil {
		return s.ldapClient
	}

	configs, err := s.systemConfigService.GetConfigsByType(ctx, nil, systemconfig.ConfigTypeLDAP)
	if err != nil {
		s.logger.Warn("Falling back to startup LDAP client: failed to read LDAP system config", zap.Error(err))
		return s.ldapClient
	}

	for _, cfg := range configs {
		if cfg.Status() != systemconfig.ConfigStatusActive {
			continue
		}
		ldapCfg, parseErr := cfg.GetLDAPConfig()
		if parseErr != nil || ldapCfg == nil {
			continue
		}
		if !ldapCfg.Enabled {
			continue
		}
		if strings.TrimSpace(ldapCfg.Host) == "" || ldapCfg.Port <= 0 || strings.TrimSpace(ldapCfg.BaseDN) == "" {
			continue
		}

		userFilter := strings.TrimSpace(ldapCfg.UserFilter)
		if userFilter == "" {
			userFilter = "(uid=%s)"
		}
		groupFilter := strings.TrimSpace(ldapCfg.GroupFilter)
		if groupFilter == "" {
			groupFilter = "(member=%s)"
		}

		return ldap.NewClient(&ldap.Config{
			Host:         ldapCfg.Host,
			Port:         ldapCfg.Port,
			BaseDN:       ldapCfg.BaseDN,
			BindDN:       ldapCfg.BindDN,
			BindPassword: ldapCfg.BindPassword,
			UserFilter:   userFilter,
			GroupFilter:  groupFilter,
			UseTLS:       ldapCfg.SSL,
			StartTLS:     ldapCfg.StartTLS,
		}, s.logger)
	}

	return s.ldapClient
}

// IsLDAPLoginEnabled reports whether at least one active LDAP provider is enabled and usable for login.
func (s *LDAPService) IsLDAPLoginEnabled(ctx context.Context) (bool, error) {
	// Corporate LDAP model: login only evaluates global LDAP provider configuration.
	configs, err := s.systemConfigService.GetConfigsByType(ctx, nil, systemconfig.ConfigTypeLDAP)
	if err != nil {
		return false, err
	}

	for _, cfg := range configs {
		if cfg.Status() != systemconfig.ConfigStatusActive {
			continue
		}
		parsed, parseErr := cfg.GetLDAPConfig()
		if parseErr != nil || parsed == nil {
			continue
		}
		if !parsed.Enabled {
			continue
		}
		if strings.TrimSpace(parsed.Host) == "" || parsed.Port <= 0 || strings.TrimSpace(parsed.BaseDN) == "" {
			continue
		}
		if !hasUsableAllowedDomain(parsed.AllowedDomains) {
			continue
		}
		return true, nil
	}

	return false, nil
}

// validateLDAPEmailDomain validates that the email domain is allowed for LDAP login
func (s *LDAPService) validateLDAPEmailDomain(ctx context.Context, email string) error {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email format")
	}
	domain := normalizeDomain(parts[1])
	if domain == "" {
		return errors.New("invalid email format")
	}

	// Corporate LDAP model: domain validation only evaluates global LDAP provider configuration.
	configs, err := s.systemConfigService.GetConfigsByType(ctx, nil, systemconfig.ConfigTypeLDAP)
	if err != nil {
		s.logger.Error("Failed to retrieve LDAP configurations",
			zap.String("email", email),
			zap.Error(err))
		return errors.New("failed to validate LDAP configuration")
	}

	hasActiveEnabledProvider := false
	hasAnyAllowedDomain := false
	var configuredDomains []string
	for _, cfg := range configs {
		if cfg.Status() != systemconfig.ConfigStatusActive {
			continue
		}
		parsed, parseErr := cfg.GetLDAPConfig()
		if parseErr != nil || parsed == nil {
			continue
		}
		if !parsed.Enabled {
			continue
		}

		hasActiveEnabledProvider = true
		for _, allowed := range parsed.AllowedDomains {
			normalizedAllowed := normalizeDomain(allowed)
			if normalizedAllowed == "" {
				continue
			}
			hasAnyAllowedDomain = true
			configuredDomains = append(configuredDomains, normalizedAllowed)
			if domain == normalizedAllowed {
				return nil
			}
		}
	}

	if !hasActiveEnabledProvider {
		s.logger.Warn("LDAP login rejected: no active LDAP provider configured",
			zap.String("email", email))
		return errors.New("LDAP authentication is not configured")
	}

	// If no domains configured, deny all LDAP logins for security
	if !hasAnyAllowedDomain {
		s.logger.Warn("LDAP login rejected: no allowed domains configured",
			zap.String("email", email))
		return errors.New("LDAP authentication is not configured")
	}

	s.logger.Warn("LDAP login rejected: email domain not allowed",
		zap.String("email", email),
		zap.String("email_domain", domain),
		zap.Strings("configured_allowed_domains", configuredDomains))

	return fmt.Errorf("email domain '%s' is not allowed for LDAP authentication", domain)
}

func normalizeDomain(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(trimmed, "@")
}

func hasUsableAllowedDomain(domains []string) bool {
	for _, d := range domains {
		if normalizeDomain(d) != "" {
			return true
		}
	}
	return false
}
