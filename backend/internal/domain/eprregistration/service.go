package eprregistration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repo   Repository
	logger *zap.Logger
}

type LifecycleTransitionResult struct {
	ExpiringCount   int
	ExpiredCount    int
	ExpiringRecords []LifecycleTransitionRecord
	ExpiredRecords  []LifecycleTransitionRecord
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

type CreateInput struct {
	TenantID              uuid.UUID
	RequestedByUserID     uuid.UUID
	EPRRecordID           string
	ProductName           string
	TechnologyName        string
	BusinessJustification string
}

func (s *Service) CreateRequest(ctx context.Context, input CreateInput) (*Request, error) {
	input.EPRRecordID = strings.TrimSpace(input.EPRRecordID)
	if input.EPRRecordID == "" {
		input.EPRRecordID = generateEPRRecordID()
	}

	req, err := NewRequest(
		input.TenantID,
		input.RequestedByUserID,
		input.EPRRecordID,
		input.ProductName,
		input.TechnologyName,
		input.BusinessJustification,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID, statusRaw string, limit, offset int) ([]*Request, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	status, err := parseStatus(statusRaw)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByTenant(ctx, tenantID, status, limit, offset)
}

func (s *Service) ListAll(ctx context.Context, statusRaw string, limit, offset int) ([]*Request, error) {
	status, err := parseStatus(statusRaw)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAll(ctx, status, limit, offset)
}

func (s *Service) Approve(ctx context.Context, id, actorUserID uuid.UUID, reason string) (*Request, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Approve(actorUserID, reason); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDecision(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) Reject(ctx context.Context, id, actorUserID uuid.UUID, reason string) (*Request, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Reject(actorUserID, reason); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDecision(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) Withdraw(ctx context.Context, tenantID, id, actorUserID uuid.UUID, reason string) (*Request, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.TenantID != tenantID {
		return nil, ErrNotFound
	}

	rejectReason := strings.TrimSpace(reason)
	if rejectReason == "" {
		rejectReason = "withdrawn by tenant user"
	}
	if err := req.Withdraw(actorUserID, rejectReason); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDecision(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) IsApprovedEPRRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error) {
	if tenantID == uuid.Nil {
		return false, ErrInvalidTenantID
	}
	eprRecordID = strings.TrimSpace(eprRecordID)
	if eprRecordID == "" {
		return false, ErrInvalidEPRRecord
	}
	return s.repo.HasApprovedRegistration(ctx, tenantID, eprRecordID)
}

func (s *Service) GetApprovedEPRRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*LifecycleStatus, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	eprRecordID = strings.TrimSpace(eprRecordID)
	if eprRecordID == "" {
		return nil, ErrInvalidEPRRecord
	}
	return s.repo.GetApprovedRegistrationLifecycleStatus(ctx, tenantID, eprRecordID)
}

func (s *Service) Suspend(ctx context.Context, id, actorUserID uuid.UUID, reason string) (*Request, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Suspend(actorUserID, reason); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateLifecycle(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) Reactivate(ctx context.Context, id, actorUserID uuid.UUID, reason string) (*Request, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Reactivate(actorUserID, reason); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateLifecycle(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) Revalidate(ctx context.Context, id, actorUserID uuid.UUID, reason string) (*Request, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Revalidate(actorUserID, reason); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateLifecycle(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) RunLifecycleTransitions(ctx context.Context, now time.Time, expiringWindow time.Duration) (*LifecycleTransitionResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if expiringWindow <= 0 {
		expiringWindow = 7 * 24 * time.Hour
	}
	expiringBefore := now.Add(expiringWindow)
	expiringRecords, expiredRecords, err := s.repo.TransitionLifecycleStates(ctx, now, expiringBefore)
	if err != nil {
		return nil, err
	}
	return &LifecycleTransitionResult{
		ExpiringCount:   len(expiringRecords),
		ExpiredCount:    len(expiredRecords),
		ExpiringRecords: expiringRecords,
		ExpiredRecords:  expiredRecords,
	}, nil
}

func generateEPRRecordID() string {
	now := time.Now().UTC().Format("20060102")
	suffix := strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]
	return fmt.Sprintf("EPR-%s-%s", now, suffix)
}

func parseStatus(raw string) (*Status, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "all" {
		return nil, nil
	}
	status := Status(raw)
	switch status {
	case StatusPending, StatusApproved, StatusRejected, StatusWithdrawn:
		return &status, nil
	default:
		return nil, ErrInvalidStatus
	}
}
