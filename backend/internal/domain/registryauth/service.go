package registryauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
)

type ServiceInterface interface {
	Create(ctx context.Context, input RegistryAuthCreate, credentials map[string]interface{}, createdBy uuid.UUID) (*RegistryAuth, error)
	Update(ctx context.Context, id uuid.UUID, input RegistryAuthCreate, credentials map[string]interface{}) (*RegistryAuth, error)
	GetByID(ctx context.Context, id uuid.UUID) (*RegistryAuth, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RegistryAuth, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*RegistryAuth, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DecryptCredentials(ctx context.Context, id uuid.UUID) (map[string]interface{}, error)
	ResolveForBuild(ctx context.Context, tenantID, projectID uuid.UUID, selectedID *uuid.UUID) (*uuid.UUID, error)
	ResolveDockerConfigJSON(ctx context.Context, id uuid.UUID) ([]byte, error)
}

type Service struct {
	repo      Repository
	encryptor *crypto.AESGCMEncryptor
}

func NewService(repo Repository, encryptor *crypto.AESGCMEncryptor) *Service {
	return &Service{repo: repo, encryptor: encryptor}
}

func (s *Service) Create(ctx context.Context, input RegistryAuthCreate, credentials map[string]interface{}, createdBy uuid.UUID) (*RegistryAuth, error) {
	if len(credentials) == 0 {
		return nil, fmt.Errorf("credentials are required")
	}
	exists, err := s.repo.ExistsByNameInScope(ctx, input.TenantID, input.ProjectID, input.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check name uniqueness: %w", err)
	}
	if exists {
		return nil, ErrDuplicateName
	}

	credentialJSON, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}
	encryptedData, err := s.encryptor.Encrypt(credentialJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	auth, err := NewRegistryAuth(input, encryptedData, createdBy)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to save registry auth: %w", err)
	}
	return auth, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input RegistryAuthCreate, credentials map[string]interface{}) (*RegistryAuth, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.Name != input.Name {
		exists, err := s.repo.ExistsByNameInScope(ctx, existing.TenantID, existing.ProjectID, input.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check name uniqueness: %w", err)
		}
		if exists {
			return nil, ErrDuplicateName
		}
	}

	encryptedData := existing.CredentialData()
	if len(credentials) == 0 && existing.AuthType != input.AuthType {
		return nil, fmt.Errorf("credentials are required when changing auth_type")
	}
	if len(credentials) > 0 {
		credentialJSON, err := json.Marshal(credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal credentials: %w", err)
		}
		encryptedData, err = s.encryptor.Encrypt(credentialJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
	}

	updated := NewRegistryAuthFromExisting(
		existing.ID,
		existing.TenantID,
		existing.ProjectID,
		input.Name,
		input.Description,
		input.RegistryType,
		input.AuthType,
		input.RegistryHost,
		encryptedData,
		existing.IsActive,
		input.IsDefault,
		existing.CreatedBy,
		existing.CreatedAt,
		time.Now().UTC(),
	)

	if err := s.repo.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf("failed to update registry auth: %w", err)
	}
	return updated, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*RegistryAuth, error) {
	auth, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry auth: %w", err)
	}
	if auth == nil {
		return nil, ErrRegistryAuthNotFound
	}
	return auth, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RegistryAuth, error) {
	return s.repo.ListByProjectID(ctx, projectID, includeTenant)
}

func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*RegistryAuth, error) {
	return s.repo.ListByTenantID(ctx, tenantID)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) DecryptCredentials(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	auth, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	decrypted, err := s.encryptor.Decrypt(auth.CredentialData())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	var credentials map[string]interface{}
	if err := json.Unmarshal(decrypted, &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}
	return credentials, nil
}

func (s *Service) ResolveForBuild(ctx context.Context, tenantID, projectID uuid.UUID, selectedID *uuid.UUID) (*uuid.UUID, error) {
	if selectedID != nil {
		auth, err := s.GetByID(ctx, *selectedID)
		if err != nil {
			return nil, err
		}
		if !auth.IsActive {
			return nil, fmt.Errorf("selected registry authentication is inactive")
		}
		if auth.TenantID != tenantID {
			return nil, fmt.Errorf("selected registry authentication belongs to a different tenant")
		}
		if auth.ProjectID != nil && *auth.ProjectID != projectID {
			return nil, fmt.Errorf("selected registry authentication belongs to a different project")
		}
		return selectedID, nil
	}

	projectDefault, err := s.repo.FindDefaultByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to load project default registry auth: %w", err)
	}
	if projectDefault != nil && projectDefault.IsActive {
		id := projectDefault.ID
		return &id, nil
	}

	tenantDefault, err := s.repo.FindDefaultByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant default registry auth: %w", err)
	}
	if tenantDefault != nil && tenantDefault.IsActive {
		id := tenantDefault.ID
		return &id, nil
	}

	return nil, ErrNoRegistryAuthFound
}

func (s *Service) ResolveDockerConfigJSON(ctx context.Context, id uuid.UUID) ([]byte, error) {
	auth, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	credentials, err := s.DecryptCredentials(ctx, id)
	if err != nil {
		return nil, err
	}

	type dockerAuthEntry struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Auth     string `json:"auth,omitempty"`
	}
	payload := map[string]interface{}{
		"auths": map[string]dockerAuthEntry{},
	}
	auths := payload["auths"].(map[string]dockerAuthEntry)

	switch auth.AuthType {
	case AuthTypeDockerConfigJSON:
		raw := credentialString(credentials, "dockerconfigjson")
		if raw == "" {
			return nil, fmt.Errorf("dockerconfigjson credential is required for auth_type=dockerconfigjson")
		}
		if !json.Valid([]byte(raw)) {
			return nil, fmt.Errorf("dockerconfigjson credential must be valid JSON")
		}
		return []byte(raw), nil

	case AuthTypeBasicAuth:
		username := credentialString(credentials, "username")
		password := credentialString(credentials, "password")
		if username == "" || password == "" {
			return nil, fmt.Errorf("username and password credentials are required for auth_type=basic_auth")
		}
		auths[auth.RegistryHost] = dockerAuthEntry{
			Username: username,
			Password: password,
			Auth:     base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
		}

	case AuthTypeToken:
		token := credentialString(credentials, "token")
		if token == "" {
			return nil, fmt.Errorf("token credential is required for auth_type=token")
		}
		username := credentialString(credentials, "username")
		if username == "" {
			username = "token"
		}
		auths[auth.RegistryHost] = dockerAuthEntry{
			Username: username,
			Password: token,
			Auth:     base64.StdEncoding.EncodeToString([]byte(username + ":" + token)),
		}

	default:
		return nil, fmt.Errorf("unsupported registry auth type %q", auth.AuthType)
	}

	out, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal docker config json: %w", err)
	}
	return out, nil
}

func credentialString(credentials map[string]interface{}, key string) string {
	val, ok := credentials[key]
	if !ok || val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", val)
}
