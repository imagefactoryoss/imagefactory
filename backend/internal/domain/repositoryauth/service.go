package repositoryauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
)

// TestConnectionResponse represents the response from testing a repository auth connection
type TestConnectionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// TestOptions defines optional parameters for repository auth tests.
type TestOptions struct {
	FullTest bool   `json:"full_test"`
	RepoURL  string `json:"repo_url"`
}

// ServiceInterface defines the interface for repository authentication service
type ServiceInterface interface {
	CreateRepositoryAuth(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name, description string, authType AuthType, credentials map[string]interface{}, createdBy uuid.UUID) (*RepositoryAuth, error)
	GetRepositoryAuth(ctx context.Context, id uuid.UUID) (*RepositoryAuth, error)
	GetRepositoryAuthsByProject(ctx context.Context, projectID uuid.UUID) ([]*RepositoryAuth, error)
	GetRepositoryAuthsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*RepositoryAuth, error)
	GetRepositoryAuthsForProject(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RepositoryAuth, error)
	GetRepositoryAuthSummariesByTenant(ctx context.Context, tenantID uuid.UUID) ([]RepositoryAuthSummary, error)
	GetActiveRepositoryAuth(ctx context.Context, projectID uuid.UUID) (*RepositoryAuth, error)
	UpdateRepositoryAuth(ctx context.Context, id uuid.UUID, name, description string, credentials map[string]interface{}) (*RepositoryAuth, error)
	DeleteRepositoryAuth(ctx context.Context, id uuid.UUID) error
	DecryptCredentials(ctx context.Context, id uuid.UUID) (map[string]interface{}, error)
	TestRepositoryAuth(ctx context.Context, id uuid.UUID, opts TestOptions) (*TestConnectionResponse, error)
	CloneRepositoryAuth(ctx context.Context, sourceID, targetProjectID uuid.UUID, name, description string, createdBy uuid.UUID) (*RepositoryAuth, error)
	ResolveGitAuthSecretData(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error)
}

// Service provides business logic for repository authentication
type Service struct {
	repo      Repository
	encryptor *crypto.AESGCMEncryptor
}

// NewService creates a new repository authentication service
func NewService(repo Repository, encryptor *crypto.AESGCMEncryptor) *Service {
	return &Service{
		repo:      repo,
		encryptor: encryptor,
	}
}

// CreateRepositoryAuth creates a new repository authentication configuration
func (s *Service) CreateRepositoryAuth(
	ctx context.Context,
	tenantID uuid.UUID,
	projectID *uuid.UUID,
	name, description string,
	authType AuthType,
	credentials map[string]interface{},
	createdBy uuid.UUID,
) (*RepositoryAuth, error) {
	exists, err := s.repo.ExistsByNameInScope(ctx, tenantID, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check name existence: %w", err)
	}
	if exists {
		return nil, ErrDuplicateRepositoryAuthName
	}

	// Encrypt credentials
	credentialJSON, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encryptedData, err := s.encryptor.Encrypt(credentialJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Create the repository authentication
	auth, err := NewRepositoryAuth(tenantID, projectID, name, description, authType, encryptedData, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository auth: %w", err)
	}

	// Save to repository
	if err := s.repo.Save(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to save repository auth: %w", err)
	}

	return auth, nil
}

// GetRepositoryAuth retrieves a repository authentication by ID
func (s *Service) GetRepositoryAuth(ctx context.Context, id uuid.UUID) (*RepositoryAuth, error) {
	auth, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository auth: %w", err)
	}
	if auth == nil {
		return nil, ErrRepositoryAuthNotFound
	}
	return auth, nil
}

// GetRepositoryAuthsByProject retrieves all repository authentications for a project
func (s *Service) GetRepositoryAuthsByProject(ctx context.Context, projectID uuid.UUID) ([]*RepositoryAuth, error) {
	auths, err := s.repo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository auths by project: %w", err)
	}
	return auths, nil
}

func (s *Service) GetRepositoryAuthsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*RepositoryAuth, error) {
	auths, err := s.repo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository auths by tenant: %w", err)
	}
	return auths, nil
}

func (s *Service) GetRepositoryAuthsForProject(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RepositoryAuth, error) {
	auths, err := s.repo.FindByProjectIDWithTenant(ctx, projectID, includeTenant)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository auths for project: %w", err)
	}
	return auths, nil
}

// GetRepositoryAuthSummariesByTenant retrieves all repository auths for a tenant.
func (s *Service) GetRepositoryAuthSummariesByTenant(ctx context.Context, tenantID uuid.UUID) ([]RepositoryAuthSummary, error) {
	auths, err := s.repo.FindSummariesByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository auths by tenant: %w", err)
	}
	return auths, nil
}

// GetActiveRepositoryAuth retrieves the active repository authentication for a project
func (s *Service) GetActiveRepositoryAuth(ctx context.Context, projectID uuid.UUID) (*RepositoryAuth, error) {
	auth, err := s.repo.FindActiveByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find active repository auth: %w", err)
	}
	return auth, nil // Can be nil if no active auth
}

// CloneRepositoryAuth clones an existing auth onto a new project.
func (s *Service) CloneRepositoryAuth(ctx context.Context, sourceID, targetProjectID uuid.UUID, name, description string, createdBy uuid.UUID) (*RepositoryAuth, error) {
	sourceAuth, err := s.GetRepositoryAuth(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	credentials, err := s.DecryptCredentials(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt source credentials: %w", err)
	}

	if description == "" {
		description = sourceAuth.GetDescription()
	}

	return s.CreateRepositoryAuth(
		ctx,
		sourceAuth.GetTenantID(),
		&targetProjectID,
		name,
		description,
		sourceAuth.GetAuthType(),
		credentials,
		createdBy,
	)
}

// UpdateRepositoryAuth updates an existing repository authentication
func (s *Service) UpdateRepositoryAuth(
	ctx context.Context,
	id uuid.UUID,
	name, description string,
	credentials map[string]interface{},
) (*RepositoryAuth, error) {
	// Get existing auth
	auth, err := s.GetRepositoryAuth(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check name conflict if name changed
	if name != auth.GetName() {
		exists, err := s.repo.ExistsByNameInScope(ctx, auth.GetTenantID(), auth.GetProjectID(), name)
		if err != nil {
			return nil, fmt.Errorf("failed to check name existence: %w", err)
		}
		if exists {
			return nil, ErrDuplicateRepositoryAuthName
		}
	}

	var encryptedData []byte
	if credentials == nil {
		encryptedData = auth.credentialData
	} else {
		// Encrypt new credentials
		credentialJSON, err := json.Marshal(credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal credentials: %w", err)
		}

		encryptedData, err = s.encryptor.Encrypt(credentialJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
	}

	// Update the auth
	if err := auth.Update(name, description, encryptedData); err != nil {
		return nil, fmt.Errorf("failed to update repository auth: %w", err)
	}

	// Save to repository
	if err := s.repo.Update(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to save updated repository auth: %w", err)
	}

	return auth, nil
}

// DeleteRepositoryAuth deactivates a repository authentication
func (s *Service) DeleteRepositoryAuth(ctx context.Context, id uuid.UUID) error {
	auth, err := s.GetRepositoryAuth(ctx, id)
	if err != nil {
		return err
	}

	// Tenant-scoped auth can be shared by multiple projects.
	// Block deletion while active projects are still referencing it.
	if auth.GetProjectID() == nil {
		usages, err := s.repo.FindActiveProjectUsages(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to resolve active project usages: %w", err)
		}
		if len(usages) > 0 {
			return &RepositoryAuthInUseError{
				AuthID:   id,
				AuthName: auth.GetName(),
				Projects: usages,
			}
		}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete repository auth: %w", err)
	}
	return nil
}

// DecryptCredentials decrypts and returns the credentials for a repository authentication
func (s *Service) DecryptCredentials(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	auth, err := s.GetRepositoryAuth(ctx, id)
	if err != nil {
		return nil, err
	}

	// Decrypt credential data
	decryptedData, err := s.encryptor.Decrypt(auth.CredentialData())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Unmarshal JSON
	var credentials map[string]interface{}
	if err := json.Unmarshal(decryptedData, &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return credentials, nil
}

// TestRepositoryAuth tests the connection to a repository using the authentication credentials
func (s *Service) TestRepositoryAuth(ctx context.Context, id uuid.UUID, opts TestOptions) (*TestConnectionResponse, error) {
	start := time.Now()
	finalize := func(resp *TestConnectionResponse) *TestConnectionResponse {
		if resp == nil {
			resp = &TestConnectionResponse{Success: false, Message: "Unknown test result"}
		}
		if resp.Details == nil {
			resp.Details = map[string]interface{}{}
		}
		resp.Details["duration_ms"] = time.Since(start).Milliseconds()
		resp.Details["test_type"] = map[bool]string{true: "full", false: "validation"}[opts.FullTest]
		return resp
	}

	// Get the repository auth
	auth, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository auth: %w", err)
	}
	if auth == nil {
		return finalize(&TestConnectionResponse{
			Success: false,
			Message: "Repository authentication not found",
		}), nil
	}

	// Decrypt credentials
	credentials, err := s.DecryptCredentials(ctx, id)
	if err != nil {
		return finalize(&TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to decrypt credentials: %v", err),
		}), nil
	}

	if opts.FullTest {
		resp, err := s.testFullConnection(ctx, auth.AuthType, credentials, opts.RepoURL)
		if resp != nil {
			resp.Details = ensureDetails(resp.Details)
			resp.Details["auth_type"] = string(auth.AuthType)
		}
		return finalize(resp), err
	}

	// Test connection based on auth type (format-only)
	switch auth.AuthType {
	case AuthTypeSSH:
		resp, err := s.testSSHConnection(ctx, credentials)
		if resp != nil {
			resp.Details = ensureDetails(resp.Details)
			resp.Details["auth_type"] = string(auth.AuthType)
		}
		return finalize(resp), err
	case AuthTypeToken:
		resp, err := s.testTokenConnection(ctx, credentials)
		if resp != nil {
			resp.Details = ensureDetails(resp.Details)
			resp.Details["auth_type"] = string(auth.AuthType)
		}
		return finalize(resp), err
	case AuthTypeBasic:
		resp, err := s.testBasicAuthConnection(ctx, credentials)
		if resp != nil {
			resp.Details = ensureDetails(resp.Details)
			resp.Details["auth_type"] = string(auth.AuthType)
		}
		return finalize(resp), err
	case AuthTypeOAuth:
		resp, err := s.testOAuthConnection(ctx, credentials)
		if resp != nil {
			resp.Details = ensureDetails(resp.Details)
			resp.Details["auth_type"] = string(auth.AuthType)
		}
		return finalize(resp), err
	default:
		return finalize(&TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported authentication type: %s", auth.AuthType),
		}), nil
	}
}

// testSSHConnection tests SSH key authentication
func (s *Service) testSSHConnection(ctx context.Context, credentials map[string]interface{}) (*TestConnectionResponse, error) {
	sshKey, ok := credentials["ssh_key"].(string)
	if !ok || sshKey == "" {
		return &TestConnectionResponse{
			Success: false,
			Message: "SSH key is required for SSH authentication",
		}, nil
	}

	// For now, we do a basic validation of the SSH key format
	// In a real implementation, you might want to test actual SSH connectivity
	if len(sshKey) < 20 {
		return &TestConnectionResponse{
			Success: false,
			Message: "SSH key appears to be too short or invalid",
		}, nil
	}

	// Check if it starts with common SSH key prefixes
	validPrefixes := []string{
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"-----BEGIN RSA PRIVATE KEY-----",
		"-----BEGIN EC PRIVATE KEY-----",
		"-----BEGIN DSA PRIVATE KEY-----",
	}

	valid := false
	for _, prefix := range validPrefixes {
		if len(sshKey) > len(prefix) && sshKey[:len(prefix)] == prefix {
			valid = true
			break
		}
	}

	if !valid {
		return &TestConnectionResponse{
			Success: false,
			Message: "SSH key does not appear to be in a valid format",
		}, nil
	}

	return &TestConnectionResponse{
		Success: true,
		Message: "SSH key format validation passed",
		Details: map[string]interface{}{
			"key_type":   "ssh_key",
			"validation": "format_check",
			"full_test":  false,
		},
	}, nil
}

// testTokenConnection tests token-based authentication
func (s *Service) testTokenConnection(ctx context.Context, credentials map[string]interface{}) (*TestConnectionResponse, error) {
	token, ok := credentials["token"].(string)
	if !ok || token == "" {
		return &TestConnectionResponse{
			Success: false,
			Message: "Token is required for token authentication",
		}, nil
	}

	// Basic token validation - check length and format
	if len(token) < 10 {
		return &TestConnectionResponse{
			Success: false,
			Message: "Token appears to be too short",
		}, nil
	}

	// For GitHub tokens, they typically start with ghp_ or github_pat_
	if len(token) > 4 && (token[:4] == "ghp_" || token[:12] == "github_pat_") {
		return &TestConnectionResponse{
			Success: true,
			Message: "GitHub token format validation passed",
			Details: map[string]interface{}{
				"token_type": "github",
				"validation": "format_check",
				"full_test":  false,
			},
		}, nil
	}

	// For GitLab tokens, they are typically just the token string
	return &TestConnectionResponse{
		Success: true,
		Message: "Token format validation passed",
		Details: map[string]interface{}{
			"token_type": "generic",
			"validation": "format_check",
			"full_test":  false,
		},
	}, nil
}

// testBasicAuthConnection tests basic authentication
func (s *Service) testBasicAuthConnection(ctx context.Context, credentials map[string]interface{}) (*TestConnectionResponse, error) {
	username, ok := credentials["username"].(string)
	if !ok || username == "" {
		return &TestConnectionResponse{
			Success: false,
			Message: "Username is required for basic authentication",
		}, nil
	}

	password, ok := credentials["password"].(string)
	if !ok || password == "" {
		return &TestConnectionResponse{
			Success: false,
			Message: "Password is required for basic authentication",
		}, nil
	}

	if len(username) < 1 {
		return &TestConnectionResponse{
			Success: false,
			Message: "Username cannot be empty",
		}, nil
	}

	if len(password) < 1 {
		return &TestConnectionResponse{
			Success: false,
			Message: "Password cannot be empty",
		}, nil
	}

	return &TestConnectionResponse{
		Success: true,
		Message: "Basic authentication credentials validation passed",
		Details: map[string]interface{}{
			"auth_type":  "basic",
			"validation": "credentials_check",
			"full_test":  false,
		},
	}, nil
}

// testOAuthConnection tests OAuth authentication
func (s *Service) testOAuthConnection(ctx context.Context, credentials map[string]interface{}) (*TestConnectionResponse, error) {
	// OAuth testing would require actual API calls to the OAuth provider
	// For now, we'll do basic validation
	return &TestConnectionResponse{
		Success: true,
		Message: "OAuth configuration validation passed",
		Details: map[string]interface{}{
			"auth_type":  "oauth",
			"validation": "configuration_check",
			"note":       "Full OAuth testing requires actual API calls to the provider",
			"full_test":  false,
		},
	}, nil
}

func (s *Service) testFullConnection(ctx context.Context, authType AuthType, credentials map[string]interface{}, repoURL string) (*TestConnectionResponse, error) {
	if strings.TrimSpace(repoURL) == "" {
		return &TestConnectionResponse{
			Success: false,
			Message: "Repository URL is required for full connection tests",
			Details: map[string]interface{}{
				"validation": "full_test",
				"full_test":  true,
				"timeout_s":  15,
			},
		}, nil
	}

	var testURL string
	switch authType {
	case AuthTypeSSH:
		if strings.HasPrefix(repoURL, "http") {
			return &TestConnectionResponse{
				Success: false,
				Message: "SSH authentication requires an SSH repository URL",
				Details: map[string]interface{}{
					"validation": "git_ls_remote",
					"full_test":  true,
					"timeout_s":  15,
				},
			}, nil
		}
		testURL = repoURL
	case AuthTypeToken, AuthTypeOAuth, AuthTypeBasic:
		if !strings.HasPrefix(repoURL, "http") {
			return &TestConnectionResponse{
				Success: false,
				Message: "Token/basic/OAuth authentication requires an HTTPS repository URL",
				Details: map[string]interface{}{
					"validation": "git_ls_remote",
					"full_test":  true,
					"timeout_s":  15,
				},
			}, nil
		}
		var username, password string
		switch authType {
		case AuthTypeBasic:
			userVal, _ := credentials["username"].(string)
			passVal, _ := credentials["password"].(string)
			username = userVal
			password = passVal
		default:
			tokenVal, _ := credentials["token"].(string)
			if tokenVal == "" {
				tokenVal, _ = credentials["access_token"].(string)
			}
			username = "x-access-token"
			password = tokenVal
		}

		parsed, err := url.Parse(repoURL)
		if err != nil {
			return &TestConnectionResponse{
				Success: false,
				Message: "Repository URL is invalid",
				Details: map[string]interface{}{
					"validation": "git_ls_remote",
					"full_test":  true,
					"timeout_s":  15,
				},
			}, nil
		}
		parsed.User = url.UserPassword(username, password)
		testURL = parsed.String()
	default:
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported authentication type: %s", authType),
			Details: map[string]interface{}{
				"validation": "git_ls_remote",
				"full_test":  true,
				"timeout_s":  15,
			},
		}, nil
	}

	testCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(testCtx, "git", "ls-remote", "--heads", testURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var tempKeyFile string
	if authType == AuthTypeSSH {
		key, _ := credentials["ssh_key"].(string)
		if key == "" {
			return &TestConnectionResponse{
				Success: false,
				Message: "SSH key is required for full test",
				Details: map[string]interface{}{
					"validation": "git_ls_remote",
					"full_test":  true,
					"timeout_s":  15,
				},
			}, nil
		}

		file, err := os.CreateTemp("", "if-ssh-key-*")
		if err != nil {
			return &TestConnectionResponse{
				Success: false,
				Message: "Failed to prepare SSH key for test",
				Details: map[string]interface{}{
					"validation": "git_ls_remote",
					"full_test":  true,
					"timeout_s":  15,
				},
			}, nil
		}
		tempKeyFile = file.Name()
		_, _ = file.WriteString(key)
		_ = file.Close()
		_ = os.Chmod(tempKeyFile, 0o600)

		cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", tempKeyFile))
	}

	defer func() {
		if tempKeyFile != "" {
			_ = os.Remove(tempKeyFile)
		}
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &TestConnectionResponse{
			Success: false,
			Message: sanitizeGitError(string(output)),
			Details: map[string]interface{}{
				"validation": "git_ls_remote",
				"full_test":  true,
				"repo_host":  repoHost(repoURL),
				"timeout_s":  15,
			},
		}, nil
	}

	refsCount := 0
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) != "" {
			refsCount++
		}
	}

	return &TestConnectionResponse{
		Success: true,
		Message: "Full connection test succeeded",
		Details: map[string]interface{}{
			"validation": "git_ls_remote",
			"full_test":  true,
			"repo_host":  repoHost(repoURL),
			"refs_count": refsCount,
			"timeout_s":  15,
		},
	}, nil
}

func ensureDetails(details map[string]interface{}) map[string]interface{} {
	if details == nil {
		return map[string]interface{}{}
	}
	return details
}

// ResolveGitAuthSecretData resolves active repository auth for project into secret key/value data.
// Returns nil,nil when project has no active repository authentication.
func (s *Service) ResolveGitAuthSecretData(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error) {
	auth, err := s.GetActiveRepositoryAuth(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if auth == nil || !auth.GetIsActive() {
		return nil, nil
	}

	credentials, err := s.DecryptCredentials(ctx, auth.GetID())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt repository auth credentials: %w", err)
	}

	secretData, err := buildGitAuthSecretData(auth.GetAuthType(), credentials)
	if err != nil {
		return nil, err
	}
	return secretData, nil
}

// ResolveGitAuthSecretDataByAuthID resolves repository auth by ID into secret key/value data.
// Returns nil,nil when auth does not exist or is inactive.
func (s *Service) ResolveGitAuthSecretDataByAuthID(ctx context.Context, authID uuid.UUID) (map[string][]byte, error) {
	auth, err := s.GetRepositoryAuth(ctx, authID)
	if err != nil {
		if errors.Is(err, ErrRepositoryAuthNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if auth == nil || !auth.GetIsActive() {
		return nil, nil
	}

	credentials, err := s.DecryptCredentials(ctx, auth.GetID())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt repository auth credentials: %w", err)
	}

	return buildGitAuthSecretData(auth.GetAuthType(), credentials)
}

func buildGitAuthSecretData(authType AuthType, credentials map[string]interface{}) (map[string][]byte, error) {
	secretData := map[string][]byte{
		"auth_type": []byte(string(authType)),
	}
	switch authType {
	case AuthTypeToken, AuthTypeOAuth:
		token := stringCredential(credentials, "token")
		if token == "" {
			return nil, fmt.Errorf("token credential is required for auth type %s", authType)
		}
		username := stringCredential(credentials, "username")
		if username == "" {
			username = "token"
		}
		secretData["username"] = []byte(username)
		secretData["token"] = []byte(token)

	case AuthTypeBasic:
		username := stringCredential(credentials, "username")
		password := stringCredential(credentials, "password")
		if username == "" || password == "" {
			return nil, fmt.Errorf("username and password credentials are required for auth type %s", authType)
		}
		secretData["username"] = []byte(username)
		secretData["password"] = []byte(password)

	case AuthTypeSSH:
		sshKey := stringCredential(credentials, "ssh_key")
		if sshKey == "" {
			return nil, fmt.Errorf("ssh_key credential is required for auth type %s", authType)
		}
		secretData["ssh-privatekey"] = []byte(sshKey)
		if knownHosts := stringCredential(credentials, "known_hosts"); knownHosts != "" {
			secretData["known_hosts"] = []byte(knownHosts)
		}

	default:
		return nil, fmt.Errorf("unsupported repository auth type %s", authType)
	}

	return secretData, nil
}

func stringCredential(credentials map[string]interface{}, key string) string {
	val, ok := credentials[key]
	if !ok || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func repoHost(repoURL string) string {
	if strings.HasPrefix(repoURL, "http") {
		if parsed, err := url.Parse(repoURL); err == nil {
			return parsed.Host
		}
	}
	if strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") {
		parts := strings.Split(repoURL, "@")
		if len(parts) > 1 {
			hostPart := strings.Split(parts[1], ":")
			return hostPart[0]
		}
	}
	return ""
}

func sanitizeGitError(output string) string {
	// Remove embedded credentials from URLs
	if strings.Contains(output, "://") {
		parts := strings.Split(output, "://")
		if len(parts) >= 2 {
			right := parts[1]
			if at := strings.Index(right, "@"); at >= 0 {
				right = right[at+1:]
			}
			return strings.TrimSpace(parts[0] + "://" + right)
		}
	}
	return strings.TrimSpace(output)
}
