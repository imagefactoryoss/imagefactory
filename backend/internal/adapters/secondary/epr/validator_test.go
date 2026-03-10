package epr

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"go.uber.org/zap"
)

type approvedCheckerStub struct {
	approved     bool
	called       bool
	statusCalled bool
	status       *eprregistration.LifecycleStatus
}

func (s *approvedCheckerStub) IsApprovedEPRRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error) {
	s.called = true
	return s.approved, nil
}

func (s *approvedCheckerStub) GetApprovedEPRRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*eprregistration.LifecycleStatus, error) {
	s.statusCalled = true
	return s.status, nil
}

func TestValidateRegistration_UsesApprovedCheckerBeforeExternalLookup(t *testing.T) {
	v := NewExternalValidator(zap.NewNop(), nil, nil)
	status := eprregistration.LifecycleStatusActive
	checker := &approvedCheckerStub{approved: true, status: &status}
	v.SetApprovedRegistrationChecker(checker)

	ok, err := v.ValidateRegistration(context.Background(), uuid.New(), "SOR-100")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatal("expected registration to be approved")
	}
	if !checker.statusCalled {
		t.Fatal("expected approved lifecycle checker to be called")
	}
}

func TestValidateRegistration_DeniesWhenApprovedLifecycleIsSuspended(t *testing.T) {
	v := NewExternalValidator(zap.NewNop(), nil, nil)
	status := eprregistration.LifecycleStatusSuspended
	checker := &approvedCheckerStub{approved: true, status: &status}
	v.SetApprovedRegistrationChecker(checker)

	ok, err := v.ValidateRegistration(context.Background(), uuid.New(), "EPR-100")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatal("expected validation deny for suspended lifecycle")
	}
}
