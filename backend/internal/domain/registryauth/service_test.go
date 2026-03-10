package registryauth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
)

type repoStub struct {
	byID map[uuid.UUID]*RegistryAuth
}

func (r *repoStub) Save(ctx context.Context, auth *RegistryAuth) error   { return nil }
func (r *repoStub) Update(ctx context.Context, auth *RegistryAuth) error { return nil }
func (r *repoStub) FindByID(ctx context.Context, id uuid.UUID) (*RegistryAuth, error) {
	return r.byID[id], nil
}
func (r *repoStub) ListByProjectID(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RegistryAuth, error) {
	return nil, nil
}
func (r *repoStub) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*RegistryAuth, error) {
	return nil, nil
}
func (r *repoStub) FindDefaultByProjectID(ctx context.Context, projectID uuid.UUID) (*RegistryAuth, error) {
	return nil, nil
}
func (r *repoStub) FindDefaultByTenantID(ctx context.Context, tenantID uuid.UUID) (*RegistryAuth, error) {
	return nil, nil
}
func (r *repoStub) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *repoStub) ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error) {
	return false, nil
}

func newEncryptorForTest(t *testing.T) *crypto.AESGCMEncryptor {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := crypto.NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}
	return enc
}

func encryptCredentials(t *testing.T, enc *crypto.AESGCMEncryptor, credentials map[string]interface{}) []byte {
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

func TestResolveDockerConfigJSON_BasicAuth(t *testing.T) {
	enc := newEncryptorForTest(t)
	id := uuid.New()
	tenantID := uuid.New()
	createdBy := uuid.New()

	repo := &repoStub{
		byID: map[uuid.UUID]*RegistryAuth{
			id: NewRegistryAuthFromExisting(
				id, tenantID, nil, "basic", "", "generic",
				AuthTypeBasicAuth, "ghcr.io",
				encryptCredentials(t, enc, map[string]interface{}{
					"username": "alice",
					"password": "secret",
				}),
				true, true, createdBy, time.Now().UTC(), time.Now().UTC(),
			),
		},
	}
	svc := NewService(repo, enc)

	raw, err := svc.ResolveDockerConfigJSON(context.Background(), id)
	if err != nil {
		t.Fatalf("ResolveDockerConfigJSON returned error: %v", err)
	}
	var parsed map[string]map[string]map[string]string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to parse docker config json: %v", err)
	}
	entry := parsed["auths"]["ghcr.io"]
	if entry["username"] != "alice" || entry["password"] != "secret" || entry["auth"] == "" {
		t.Fatalf("unexpected docker auth entry: %+v", entry)
	}
}

func TestResolveDockerConfigJSON_DockerConfigJSONPassthrough(t *testing.T) {
	enc := newEncryptorForTest(t)
	id := uuid.New()
	tenantID := uuid.New()
	createdBy := uuid.New()
	dockerConfig := `{"auths":{"registry.example.com":{"auth":"YWxpY2U6c2VjcmV0"}}}`

	repo := &repoStub{
		byID: map[uuid.UUID]*RegistryAuth{
			id: NewRegistryAuthFromExisting(
				id, tenantID, nil, "dockerjson", "", "generic",
				AuthTypeDockerConfigJSON, "registry.example.com",
				encryptCredentials(t, enc, map[string]interface{}{
					"dockerconfigjson": dockerConfig,
				}),
				true, true, createdBy, time.Now().UTC(), time.Now().UTC(),
			),
		},
	}
	svc := NewService(repo, enc)

	raw, err := svc.ResolveDockerConfigJSON(context.Background(), id)
	if err != nil {
		t.Fatalf("ResolveDockerConfigJSON returned error: %v", err)
	}
	if string(raw) != dockerConfig {
		t.Fatalf("expected passthrough docker config json, got %s", string(raw))
	}
}
