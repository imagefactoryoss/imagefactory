package repositoryauth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
)

type repoAuthRepoStub struct {
	byID            map[uuid.UUID]*RepositoryAuth
	activeByProject map[uuid.UUID]*RepositoryAuth
}

func (r *repoAuthRepoStub) Save(ctx context.Context, auth *RepositoryAuth) error { return nil }
func (r *repoAuthRepoStub) FindByID(ctx context.Context, id uuid.UUID) (*RepositoryAuth, error) {
	return r.byID[id], nil
}
func (r *repoAuthRepoStub) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*RepositoryAuth, error) {
	return nil, nil
}
func (r *repoAuthRepoStub) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*RepositoryAuth, error) {
	return nil, nil
}
func (r *repoAuthRepoStub) FindByProjectIDWithTenant(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RepositoryAuth, error) {
	return nil, nil
}
func (r *repoAuthRepoStub) FindActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*RepositoryAuth, error) {
	if r.activeByProject == nil {
		return nil, nil
	}
	return r.activeByProject[projectID], nil
}
func (r *repoAuthRepoStub) FindByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (*RepositoryAuth, error) {
	return nil, nil
}
func (r *repoAuthRepoStub) Update(ctx context.Context, auth *RepositoryAuth) error { return nil }
func (r *repoAuthRepoStub) Delete(ctx context.Context, id uuid.UUID) error         { return nil }
func (r *repoAuthRepoStub) ExistsByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (bool, error) {
	return false, nil
}
func (r *repoAuthRepoStub) ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error) {
	return false, nil
}
func (r *repoAuthRepoStub) FindSummariesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]RepositoryAuthSummary, error) {
	return nil, nil
}
func (r *repoAuthRepoStub) FindActiveProjectUsages(ctx context.Context, authID uuid.UUID) ([]ProjectUsage, error) {
	return nil, nil
}

func testEncryptor(t *testing.T) *crypto.AESGCMEncryptor {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := crypto.NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create test encryptor: %v", err)
	}
	return enc
}

func testEncryptCredentials(t *testing.T, enc *crypto.AESGCMEncryptor, credentials map[string]interface{}) []byte {
	t.Helper()
	raw, err := json.Marshal(credentials)
	if err != nil {
		t.Fatalf("failed to marshal credentials: %v", err)
	}
	out, err := enc.Encrypt(raw)
	if err != nil {
		t.Fatalf("failed to encrypt credentials: %v", err)
	}
	return out
}

func TestResolveGitAuthSecretData_Token(t *testing.T) {
	enc := testEncryptor(t)
	id := uuid.New()
	projectID := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()
	auth := NewRepositoryAuthFromExisting(
		id, tenantID, &projectID, "repo-token", "", string(AuthTypeToken),
		testEncryptCredentials(t, enc, map[string]interface{}{
			"token": "abc123",
		}),
		true, uuid.New(), now, now, 1,
	)

	repo := &repoAuthRepoStub{
		byID: map[uuid.UUID]*RepositoryAuth{id: auth},
		activeByProject: map[uuid.UUID]*RepositoryAuth{
			projectID: auth,
		},
	}
	svc := NewService(repo, enc)
	data, err := svc.ResolveGitAuthSecretData(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ResolveGitAuthSecretData returned error: %v", err)
	}
	if string(data["token"]) != "abc123" {
		t.Fatalf("expected token credential in secret data")
	}
	if string(data["username"]) != "token" {
		t.Fatalf("expected default username token, got %q", string(data["username"]))
	}
}

func TestResolveGitAuthSecretData_SSH(t *testing.T) {
	enc := testEncryptor(t)
	id := uuid.New()
	projectID := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()
	auth := NewRepositoryAuthFromExisting(
		id, tenantID, &projectID, "repo-ssh", "", string(AuthTypeSSH),
		testEncryptCredentials(t, enc, map[string]interface{}{
			"ssh_key":     "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----",
			"known_hosts": "github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
		}),
		true, uuid.New(), now, now, 1,
	)

	repo := &repoAuthRepoStub{
		byID: map[uuid.UUID]*RepositoryAuth{id: auth},
		activeByProject: map[uuid.UUID]*RepositoryAuth{
			projectID: auth,
		},
	}
	svc := NewService(repo, enc)
	data, err := svc.ResolveGitAuthSecretData(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ResolveGitAuthSecretData returned error: %v", err)
	}
	if string(data["ssh-privatekey"]) == "" {
		t.Fatalf("expected ssh-privatekey in secret data")
	}
	if string(data["known_hosts"]) == "" {
		t.Fatalf("expected known_hosts in secret data")
	}
}
