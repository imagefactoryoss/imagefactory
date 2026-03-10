package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/srikarm/image-factory/internal/adapters/primary/rest"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	ldappkg "github.com/srikarm/image-factory/internal/infrastructure/ldap"
	"github.com/srikarm/image-factory/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// AuthIntegrationTestSuite tests authentication flows end-to-end
type AuthIntegrationTestSuite struct {
	suite.Suite
	db         *sqlx.DB
	router     *rest.Router
	tenantSvc  *tenant.Service
	userSvc    *user.Service
	ldapSvc    *user.LDAPService
	rbacSvc    *rbac.Service
	configSvc  *systemconfig.Service
	logger     *zap.Logger
	testTenant *tenant.Tenant
	testUser   *user.User
}

// SetupSuite sets up the test suite
func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Integration tests should be self-contained; the router requires ENCRYPTION_KEY for
	// registry/repository auth encryption. Use a deterministic test key if not set.
	if os.Getenv("ENCRYPTION_KEY") == "" {
		_ = os.Setenv("ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=") // base64("0123456789abcdef0123456789abcdef")
	}

	// Setup test database connection
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/image_factory_test?sslmode=disable"
	}
	testutil.RequireSafeTestDSN(suite.T(), dbURL, "TEST_DATABASE_URL")

	var err error
	suite.db, err = sqlx.Connect("postgres", dbURL)
	suite.Require().NoError(err)

	// Setup logger
	suite.logger = zaptest.NewLogger(suite.T())

	// Initialize repositories
	tenantRepo := postgres.NewTenantRepository(suite.db, suite.logger)
	userRepo := postgres.NewUserRepository(suite.db, suite.logger)
	rbacRepo := postgres.NewRBACRepository(suite.db, suite.logger)
	systemConfigRepo := postgres.NewSystemConfigRepository(suite.db, suite.logger)
	notificationTemplateRepo := postgres.NewNotificationRepository(suite.db, suite.logger)
	buildExecutionRepo := postgres.NewBuildExecutionRepository(suite.db, suite.logger)

	// Initialize services
	eventPublisher := tenant.NewNoOpEventPublisher(suite.logger)
	suite.tenantSvc = tenant.NewService(tenantRepo, eventPublisher, suite.logger)
	suite.userSvc = user.NewService(userRepo, suite.logger, "test-jwt-secret")
	suite.rbacSvc = rbac.NewService(rbacRepo, suite.logger)
	suite.configSvc = systemconfig.NewService(systemConfigRepo, suite.logger)

	// Initialize LDAP service (placeholder - would use real LDAP in integration tests)
	suite.ldapSvc = user.NewLDAPService(userRepo, suite.logger, "test-jwt-secret", suite.newTestLDAPClient(), suite.configSvc)

	// Initialize build execution service
	buildExecutionSvc := build.NewBuildExecutionService(buildExecutionRepo)

	// Setup router
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			URL: dbURL,
		},
	}

	suite.router = rest.NewRouter(cfg, suite.logger, suite.db, nil)
	rest.SetupRoutes(
		suite.router,
		tenantRepo,
		suite.tenantSvc,
		nil,               // build repo
		nil,               // build service
		buildExecutionSvc, // build execution service
		nil,               // project repo
		nil,               // project service
		userRepo,
		suite.userSvc,
		rbacRepo,
		suite.rbacSvc,
		systemConfigRepo,
		suite.logger,
		suite.ldapSvc, // ldap service
		nil,           // audit service
		notificationTemplateRepo,
		nil,      // ws hub
		suite.db, // db interface
		cfg,      // config
		nil,      // build policy service
		nil,      // dispatcher metrics provider
		nil,      // dispatcher controller
		nil,      // orchestrator controller
		nil,      // dispatcher runtime reader
		nil,      // process status provider
		false,    // dispatcher enabled
		nil,      // infrastructure event publisher
		nil,      // release compliance metrics
		nil,      // event bus
	)
}

func (suite *AuthIntegrationTestSuite) newTestLDAPClient() *ldappkg.Client {
	if strings.TrimSpace(os.Getenv("LDAP_IT_ENABLED")) == "" {
		return nil
	}

	cfg := &ldappkg.Config{
		Host:         envOrFirst("127.0.0.1", "LDAP_IT_HOST", "IF_AUTH_LDAP_SERVER"),
		Port:         envIntOrFirst(3893, "LDAP_IT_PORT", "IF_AUTH_LDAP_PORT"),
		BaseDN:       envOrFirst("dc=imgfactory,dc=com", "LDAP_IT_BASE_DN", "IF_AUTH_LDAP_BASE_DN"),
		BindDN:       envOrFirst("cn=ldap_search,dc=imgfactory,dc=com", "LDAP_IT_BIND_DN", "IF_AUTH_LDAP_BIND_DN"),
		BindPassword: envOrFirst("search_password", "LDAP_IT_BIND_PASSWORD", "IF_AUTH_LDAP_BIND_PASSWORD"),
		UserFilter:   envOrFirst("(uid=%s)", "LDAP_IT_USER_FILTER", "LDAP_USER_FILTER"),
		GroupFilter:  envOrFirst("(member=%s)", "LDAP_IT_GROUP_FILTER", "LDAP_GROUP_FILTER"),
		UseTLS:       envBoolOrFirst(false, "LDAP_IT_USE_TLS", "IF_AUTH_LDAP_USE_TLS"),
		StartTLS:     envBoolOrFirst(false, "LDAP_IT_START_TLS", "IF_AUTH_LDAP_START_TLS"),
	}

	return ldappkg.NewClient(cfg, suite.logger)
}

func envOrDefault(key, fallback string) string {
	return envOrFirst(fallback, key)
}

func envOrFirst(fallback string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return fallback
}

func envIntOrFirst(fallback int, keys ...string) int {
	for _, key := range keys {
		if parsed, ok := parseIntEnv(key); ok {
			return parsed
		}
	}
	return fallback
}

func parseIntEnv(key string) (int, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func envBoolOrFirst(fallback bool, keys ...string) bool {
	for _, key := range keys {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		if strings.EqualFold(v, "true") || v == "1" {
			return true
		}
		if strings.EqualFold(v, "false") || v == "0" {
			return false
		}
	}
	return fallback
}

// SetupTest sets up each test
func (suite *AuthIntegrationTestSuite) SetupTest() {
	// Clean up test data
	suite.Require().NoError(suite.cleanupTestData())

	// Create test tenant
	testTenant, err := suite.tenantSvc.CreateTenant(
		context.Background(),
		uuid.New(), // company ID
		"TEST001",
		"Test Tenant",
		"test-tenant",
		"Test tenant for integration tests",
	)
	suite.Require().NoError(err)
	suite.testTenant = testTenant

	// Create test user
	testUser, err := suite.userSvc.CreateUser(
		context.Background(),
		testTenant.ID(),
		"test@example.com",
		"Test",
		"User",
		"password123",
	)
	suite.Require().NoError(err)

	// Activate the test user account
	testUser.Activate()
	err = suite.userSvc.UpdateUser(context.Background(), testUser)
	suite.Require().NoError(err)

	suite.testUser = testUser

	// Ensure wildcard permissions exist for system role initialization
	_, _ = suite.db.Exec(
		`INSERT INTO permissions (resource, action, description, category, is_system_permission)
		 VALUES ('*', '*', 'Wildcard system access', 'system', true)
		 ON CONFLICT (resource, action) DO NOTHING`,
	)
	_, _ = suite.db.Exec(
		`INSERT INTO permissions (resource, action, description, category, is_system_permission)
		 VALUES ('*', 'read', 'Wildcard read access', 'system', true)
		 ON CONFLICT (resource, action) DO NOTHING`,
	)
	_, _ = suite.db.Exec(
		`INSERT INTO permissions (resource, action, description, category, is_system_permission) VALUES
		 ('users', 'read', 'Read users', 'user_management', true),
		 ('users', 'write', 'Write users', 'user_management', true),
		 ('users', 'delete', 'Delete users', 'user_management', true),
		 ('roles', 'read', 'Read roles', 'rbac', true),
		 ('roles', 'assign', 'Assign roles', 'rbac', true)
		 ON CONFLICT (resource, action) DO NOTHING`,
	)

	// Initialize system roles
	err = suite.rbacSvc.InitializeSystemRoles(context.Background())
	suite.Require().NoError(err)
}

// TearDownTest cleans up after each test
func (suite *AuthIntegrationTestSuite) TearDownTest() {
	suite.Require().NoError(suite.cleanupTestData())
}

// cleanupTestData removes test data
func (suite *AuthIntegrationTestSuite) cleanupTestData() error {
	// Use schema-aware TRUNCATE to support both current and legacy RBAC table names.
	_, err := suite.db.Exec(`
DO $$
DECLARE
	existing_tables TEXT;
BEGIN
	SELECT string_agg(format('%I', table_name), ', ')
	INTO existing_tables
	FROM (
		SELECT unnest(ARRAY[
			'user_roles',
			'user_role_assignments',
			'role_permissions',
			'roles',
			'rbac_roles',
			'user_sessions',
			'users',
			'tenant_groups',
			'tenants',
			'companies'
		]) AS table_name
	) candidates
	WHERE to_regclass('public.' || table_name) IS NOT NULL;

	IF existing_tables IS NOT NULL THEN
		EXECUTE 'TRUNCATE TABLE ' || existing_tables || ' RESTART IDENTITY CASCADE';
	END IF;
END $$;`)
	return err
}

// TestStandardLogin tests standard username/password login
func (suite *AuthIntegrationTestSuite) TestStandardLogin() {
	// Test successful login
	loginReq := rest.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response rest.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	assert.NotEmpty(suite.T(), response.AccessToken)
	assert.NotEmpty(suite.T(), response.RefreshToken)
	assert.Equal(suite.T(), "test@example.com", response.User.Email)
	assert.True(suite.T(), response.User.IsActive)
}

// TestInvalidCredentials tests login with invalid credentials
func (suite *AuthIntegrationTestSuite) TestInvalidCredentials() {
	loginReq := rest.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "invalid credentials", response["message"])
}

// TestTokenRefresh tests token refresh functionality
func (suite *AuthIntegrationTestSuite) TestTokenRefresh() {
	// First login to get tokens
	loginReq := rest.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var loginResp rest.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &loginResp)

	// Now refresh the token
	refreshReq := rest.RefreshTokenRequest{
		RefreshToken: loginResp.RefreshToken,
	}

	body, _ = json.Marshal(refreshReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var refreshResp rest.RefreshTokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &refreshResp)
	suite.Require().NoError(err)

	assert.NotEmpty(suite.T(), refreshResp.AccessToken)
	assert.NotEmpty(suite.T(), refreshResp.RefreshToken)
	assert.Equal(suite.T(), "test@example.com", refreshResp.User.Email)
}

// TestUserCreationAndLogin tests creating a user and then logging in
func (suite *AuthIntegrationTestSuite) TestUserCreationAndLogin() {
	// Create a new user via service (bypassing authentication for testing)
	newUser, err := suite.userSvc.CreateUser(
		context.Background(),
		suite.testTenant.ID(),
		"newuser@example.com",
		"New",
		"User",
		"newpassword123",
	)
	suite.Require().NoError(err)

	// Activate the user account
	newUser.Activate()

	// Save the activated user
	err = suite.userSvc.UpdateUser(context.Background(), newUser)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "newuser@example.com", newUser.Email())
	assert.Equal(suite.T(), "New", newUser.FirstName())
	assert.Equal(suite.T(), "User", newUser.LastName())

	// Now login with the new user
	loginReq := rest.LoginRequest{
		Email:    "newuser@example.com",
		Password: "newpassword123",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response rest.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "newuser@example.com", response.User.Email)
	assert.Equal(suite.T(), "New", response.User.FirstName)
	assert.Equal(suite.T(), "User", response.User.LastName)
}

// TestLDAPLoginSimulation tests LDAP login flow (simulated)
func (suite *AuthIntegrationTestSuite) TestLDAPLoginSimulation() {
	if strings.TrimSpace(os.Getenv("LDAP_IT_ENABLED")) == "" {
		suite.T().Skip("Set LDAP_IT_ENABLED=1 to run against local LDAP/glauth")
	}
	if suite.ldapSvc == nil {
		suite.T().Skip("LDAP service is not configured for integration test")
	}

	loginEmail := envOrDefault("LDAP_IT_LOGIN_EMAIL", "alice.johnson@imagefactory.local")
	loginPassword := envOrFirst("password", "LDAP_IT_LOGIN_PASSWORD")

	parts := strings.Split(loginEmail, "@")
	suite.Require().Len(parts, 2)
	allowedDomain := parts[1]

	_, err := suite.configSvc.CreateOrUpdateCategoryConfig(
		context.Background(),
		nil,
		systemconfig.ConfigTypeLDAP,
		"ldap_glauth_test",
		systemconfig.LDAPConfig{
			ProviderName:    "glauth-test",
			ProviderType:    "active_directory",
			Host:            envOrFirst("127.0.0.1", "LDAP_IT_HOST", "IF_AUTH_LDAP_SERVER"),
			Port:            envIntOrFirst(3893, "LDAP_IT_PORT", "IF_AUTH_LDAP_PORT"),
			BaseDN:          envOrFirst("dc=imgfactory,dc=com", "LDAP_IT_BASE_DN", "IF_AUTH_LDAP_BASE_DN"),
			UserSearchBase:  envOrFirst("ou=people,dc=imgfactory,dc=com", "LDAP_IT_USER_SEARCH_BASE", "IF_AUTH_LDAP_USER_SEARCH_BASE"),
			GroupSearchBase: envOrFirst("ou=groups,dc=imgfactory,dc=com", "LDAP_IT_GROUP_SEARCH_BASE", "IF_AUTH_LDAP_GROUP_SEARCH_BASE"),
			BindDN:          envOrFirst("cn=ldap_search,dc=imgfactory,dc=com", "LDAP_IT_BIND_DN", "IF_AUTH_LDAP_BIND_DN"),
			BindPassword:    envOrFirst("search_password", "LDAP_IT_BIND_PASSWORD", "IF_AUTH_LDAP_BIND_PASSWORD"),
			UserFilter:      envOrFirst("(uid=%s)", "LDAP_IT_USER_FILTER", "LDAP_USER_FILTER"),
			GroupFilter:     envOrFirst("(member=%s)", "LDAP_IT_GROUP_FILTER", "LDAP_GROUP_FILTER"),
			StartTLS:        envBoolOrFirst(false, "LDAP_IT_START_TLS", "IF_AUTH_LDAP_START_TLS"),
			SSL:             envBoolOrFirst(false, "LDAP_IT_USE_TLS", "IF_AUTH_LDAP_USE_TLS"),
			AllowedDomains:  []string{allowedDomain},
			Enabled:         true,
		},
		suite.testUser.ID(),
	)
	suite.Require().NoError(err)

	loginReq := rest.LoginRequest{
		Email:    loginEmail,
		Password: loginPassword,
		UseLDAP:  true,
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, w.Body.String())

	var response rest.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), loginEmail, response.User.Email)
	assert.NotEmpty(suite.T(), response.AccessToken)
	assert.NotEmpty(suite.T(), response.RefreshToken)
}

// TestConcurrentLogins tests multiple concurrent login attempts
func (suite *AuthIntegrationTestSuite) TestConcurrentLogins() {
	// Test concurrent logins to ensure thread safety
	done := make(chan int, 10)
	successCount := 0

	for i := 0; i < 10; i++ {
		go func() {
			loginReq := rest.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			}

			body, _ := json.Marshal(loginReq)
			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				done <- 1 // Success
			} else {
				done <- 0 // Failure
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case result := <-done:
			successCount += result
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Concurrent login test timed out")
		}
	}

	// Due to optimistic locking, not all concurrent logins may succeed
	// But at least some should succeed
	assert.Greater(suite.T(), successCount, 0, "At least one concurrent login should succeed")
	assert.LessOrEqual(suite.T(), successCount, 10, "Success count should not exceed total attempts")
}

// TestRateLimitingSimulation tests rate limiting behavior
func (suite *AuthIntegrationTestSuite) TestRateLimitingSimulation() {
	// Simulate multiple failed login attempts
	for i := 0; i < 5; i++ {
		loginReq := rest.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		// First few attempts should return 401
		if i < 4 {
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
		}
	}

	// Check if account is locked (would require account status check)
	user, err := suite.userSvc.GetUserByID(context.Background(), suite.testUser.ID())
	suite.Require().NoError(err)

	// Account should be locked after 5 failed attempts
	assert.True(suite.T(), user.IsLocked() || user.Status() == "locked")
}

// TestRunSuite runs the test suite
func TestAuthIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Skip if database is not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(AuthIntegrationTestSuite))
}
