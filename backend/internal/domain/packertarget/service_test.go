package packertarget

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type testRepo struct {
	profiles map[uuid.UUID]*Profile
}

func newTestRepo() *testRepo {
	return &testRepo{profiles: make(map[uuid.UUID]*Profile)}
}

func (r *testRepo) Create(_ context.Context, profile *Profile) error {
	cp := *profile
	r.profiles[profile.ID] = &cp
	return nil
}

func (r *testRepo) Update(_ context.Context, profile *Profile) error {
	if _, ok := r.profiles[profile.ID]; !ok {
		return ErrNotFound
	}
	cp := *profile
	r.profiles[profile.ID] = &cp
	return nil
}

func (r *testRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	profile, ok := r.profiles[id]
	if !ok || profile.TenantID != tenantID {
		return ErrNotFound
	}
	delete(r.profiles, id)
	return nil
}

func (r *testRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*Profile, error) {
	profile, ok := r.profiles[id]
	if !ok || profile.TenantID != tenantID {
		return nil, ErrNotFound
	}
	cp := *profile
	return &cp, nil
}

func (r *testRepo) List(_ context.Context, tenantID uuid.UUID, allTenants bool, provider string) ([]*Profile, error) {
	out := make([]*Profile, 0)
	for _, profile := range r.profiles {
		if !allTenants && profile.TenantID != tenantID {
			continue
		}
		if provider != "" && profile.Provider != provider {
			continue
		}
		cp := *profile
		out = append(out, &cp)
	}
	return out, nil
}

func (r *testRepo) UpdateValidation(_ context.Context, tenantID, id uuid.UUID, result ValidationResult) error {
	profile, ok := r.profiles[id]
	if !ok || profile.TenantID != tenantID {
		return ErrNotFound
	}
	profile.ValidationStatus = result.Status
	profile.LastValidatedAt = &result.CheckedAt
	msg := result.Message
	profile.LastValidationMessage = &msg
	profile.LastRemediationHints = append([]string(nil), result.RemediationHints...)
	return nil
}

func TestServiceValidate_InvalidWhenRequiredOptionsMissing(t *testing.T) {
	repo := newTestRepo()
	service := NewService(repo)
	tenantID := uuid.New()
	userID := uuid.New()

	profile, err := service.Create(context.Background(), CreateRequest{
		TenantID:  tenantID,
		CreatedBy: userID,
		Name:      "vmware-base",
		Provider:  ProviderVMware,
		SecretRef: "vault/vmware-creds",
		Options: map[string]interface{}{
			"vcenter_server": "vc.example.local",
			"datacenter":     "dc1",
		},
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	result, err := service.Validate(context.Background(), tenantID, profile.ID)
	if err != nil {
		t.Fatalf("validate profile: %v", err)
	}
	if result.Status != ValidationStatusInvalid {
		t.Fatalf("expected invalid status, got %q", result.Status)
	}
	if len(result.RemediationHints) == 0 {
		t.Fatalf("expected remediation hints when validation fails")
	}

	stored, err := service.GetByID(context.Background(), tenantID, profile.ID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if stored.ValidationStatus != ValidationStatusInvalid {
		t.Fatalf("expected persisted invalid status, got %q", stored.ValidationStatus)
	}
	if stored.LastValidatedAt == nil || stored.LastValidatedAt.IsZero() {
		t.Fatalf("expected last_validated_at to be set")
	}
}

func TestServiceUpdate_ResetsValidationState(t *testing.T) {
	repo := newTestRepo()
	service := NewService(repo)
	tenantID := uuid.New()
	userID := uuid.New()

	profile, err := service.Create(context.Background(), CreateRequest{
		TenantID:  tenantID,
		CreatedBy: userID,
		Name:      "aws-base",
		Provider:  ProviderAWS,
		SecretRef: "vault/aws-creds",
		Options: map[string]interface{}{
			"region":        "us-east-1",
			"source_ami":    "ami-123",
			"instance_type": "t3.micro",
			"subnet_id":     "subnet-123",
		},
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	validatedAt := time.Now().UTC()
	profile.ValidationStatus = ValidationStatusValid
	profile.LastValidatedAt = &validatedAt
	message := "ok"
	profile.LastValidationMessage = &message
	profile.LastRemediationHints = []string{"none"}
	if err := repo.Update(context.Background(), profile); err != nil {
		t.Fatalf("seed profile state: %v", err)
	}

	newSecret := "vault/aws-rotated"
	updated, err := service.Update(context.Background(), tenantID, profile.ID, UpdateRequest{SecretRef: &newSecret})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.ValidationStatus != ValidationStatusUntested {
		t.Fatalf("expected untested after update, got %q", updated.ValidationStatus)
	}
	if updated.LastValidatedAt != nil {
		t.Fatalf("expected last_validated_at reset on update")
	}
	if updated.LastValidationMessage != nil {
		t.Fatalf("expected validation message reset on update")
	}
	if len(updated.LastRemediationHints) != 0 {
		t.Fatalf("expected remediation hints reset on update")
	}
}
