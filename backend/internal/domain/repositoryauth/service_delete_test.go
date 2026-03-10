package repositoryauth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
)

type deleteRepoStub struct {
	byID    map[uuid.UUID]*RepositoryAuth
	usages  map[uuid.UUID][]ProjectUsage
	deleted []uuid.UUID
}

func (r *deleteRepoStub) Save(ctx context.Context, auth *RepositoryAuth) error { return nil }
func (r *deleteRepoStub) FindByID(ctx context.Context, id uuid.UUID) (*RepositoryAuth, error) {
	return r.byID[id], nil
}
func (r *deleteRepoStub) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*RepositoryAuth, error) {
	return nil, nil
}
func (r *deleteRepoStub) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*RepositoryAuth, error) {
	return nil, nil
}
func (r *deleteRepoStub) FindByProjectIDWithTenant(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RepositoryAuth, error) {
	return nil, nil
}
func (r *deleteRepoStub) FindActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*RepositoryAuth, error) {
	return nil, nil
}
func (r *deleteRepoStub) FindByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (*RepositoryAuth, error) {
	return nil, nil
}
func (r *deleteRepoStub) Update(ctx context.Context, auth *RepositoryAuth) error { return nil }
func (r *deleteRepoStub) Delete(ctx context.Context, id uuid.UUID) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (r *deleteRepoStub) ExistsByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (bool, error) {
	return false, nil
}
func (r *deleteRepoStub) ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error) {
	return false, nil
}
func (r *deleteRepoStub) FindSummariesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]RepositoryAuthSummary, error) {
	return nil, nil
}
func (r *deleteRepoStub) FindActiveProjectUsages(ctx context.Context, authID uuid.UUID) ([]ProjectUsage, error) {
	return r.usages[authID], nil
}

func TestDeleteRepositoryAuth_BlocksTenantScopedAuthInUse(t *testing.T) {
	now := time.Now().UTC()
	authID := uuid.New()
	tenantID := uuid.New()

	tenantScoped := NewRepositoryAuthFromExisting(
		authID, tenantID, nil, "shared-git-token", "", string(AuthTypeToken),
		[]byte("encrypted"), true, uuid.New(), now, now, 1,
	)

	repo := &deleteRepoStub{
		byID: map[uuid.UUID]*RepositoryAuth{
			authID: tenantScoped,
		},
		usages: map[uuid.UUID][]ProjectUsage{
			authID: {
				{ProjectID: uuid.New(), ProjectName: "payments-service"},
				{ProjectID: uuid.New(), ProjectName: "catalog-service"},
			},
		},
	}

	enc, _ := crypto.NewAESGCMEncryptor(make([]byte, 32))
	svc := NewService(repo, enc)

	err := svc.DeleteRepositoryAuth(context.Background(), authID)
	if err == nil {
		t.Fatalf("expected in-use delete guard error")
	}

	inUse, ok := err.(*RepositoryAuthInUseError)
	if !ok {
		t.Fatalf("expected RepositoryAuthInUseError, got %T", err)
	}
	if len(inUse.Projects) != 2 {
		t.Fatalf("expected 2 active project references, got %d", len(inUse.Projects))
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("expected delete not to execute while in use")
	}
}

func TestDeleteRepositoryAuth_AllowsProjectScopedDelete(t *testing.T) {
	now := time.Now().UTC()
	authID := uuid.New()
	tenantID := uuid.New()
	projectID := uuid.New()

	projectScoped := NewRepositoryAuthFromExisting(
		authID, tenantID, &projectID, "project-git-token", "", string(AuthTypeToken),
		[]byte("encrypted"), true, uuid.New(), now, now, 1,
	)

	repo := &deleteRepoStub{
		byID: map[uuid.UUID]*RepositoryAuth{
			authID: projectScoped,
		},
	}

	enc, _ := crypto.NewAESGCMEncryptor(make([]byte, 32))
	svc := NewService(repo, enc)

	if err := svc.DeleteRepositoryAuth(context.Background(), authID); err != nil {
		t.Fatalf("expected project-scoped delete to succeed, got %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != authID {
		t.Fatalf("expected delete to execute for project-scoped auth")
	}
}

