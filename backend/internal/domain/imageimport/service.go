package imageimport

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repository        Repository
	sorValidator      SORValidator
	capabilityChecker CapabilityChecker
	approvalRequester ApprovalRequester
	logger            *zap.Logger
}

type retryPolicy struct {
	MaxAttempts int
	Cooldown    time.Duration
}

func NewService(repository Repository, sorValidator SORValidator, capabilityChecker CapabilityChecker, approvalRequester ApprovalRequester, logger *zap.Logger) *Service {
	return &Service{
		repository:        repository,
		sorValidator:      sorValidator,
		capabilityChecker: capabilityChecker,
		approvalRequester: approvalRequester,
		logger:            logger,
	}
}

type CreateImportRequestInput struct {
	TenantID          uuid.UUID
	RequestedByUserID uuid.UUID
	RequestType       RequestType
	SORRecordID       string
	SourceRegistry    string
	SourceImageRef    string
	RegistryAuthID    *uuid.UUID
}

func (s *Service) CreateImportRequest(ctx context.Context, input CreateImportRequestInput) (*ImportRequest, error) {
	req, err := NewImportRequest(
		input.TenantID,
		input.RequestedByUserID,
		input.RequestType,
		input.SORRecordID,
		input.SourceRegistry,
		input.SourceImageRef,
		input.RegistryAuthID,
	)
	if err != nil {
		return nil, err
	}

	if s.capabilityChecker != nil {
		entitled, capabilityErr := s.capabilityChecker.IsImportEntitled(ctx, req.TenantID)
		if capabilityErr != nil {
			return nil, fmt.Errorf("failed to validate operation capability: %w", capabilityErr)
		}
		if !entitled {
			return nil, ErrOperationNotEntitled
		}
	}

	if s.sorValidator != nil && req.RequestType == RequestTypeQuarantine {
		ok, validateErr := s.sorValidator.ValidateRegistration(ctx, req.TenantID, req.SORRecordID)
		if validateErr != nil {
			return nil, fmt.Errorf("failed to validate epr registration: %w", validateErr)
		}
		if !ok {
			return nil, ErrSORRegistrationRequired
		}
	}

	if s.approvalRequester != nil {
		if err := s.approvalRequester.CreateImportApproval(ctx, req); err != nil {
			return nil, fmt.Errorf("failed to create approval request: %w", err)
		}
	}

	if err := s.repository.Create(ctx, req); err != nil {
		s.logger.Error("Failed to create external image import request", zap.Error(err), zap.String("tenant_id", req.TenantID.String()))
		return nil, err
	}

	return req, nil
}

func (s *Service) GetImportRequest(ctx context.Context, tenantID, id uuid.UUID) (*ImportRequest, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if id == uuid.Nil {
		return nil, ErrInvalidImportID
	}
	return s.repository.GetByID(ctx, tenantID, id)
}

func (s *Service) ListImportRequests(ctx context.Context, tenantID uuid.UUID, requestType RequestType, limit, offset int) ([]*ImportRequest, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
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
	return s.repository.ListByTenant(ctx, tenantID, requestType, limit, offset)
}

func (s *Service) ListAllImportRequests(ctx context.Context, requestType RequestType, limit, offset int) ([]*ImportRequest, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repository.ListAll(ctx, requestType, limit, offset)
}

func (s *Service) ListReleasedArtifacts(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*ReleasedArtifact, int, error) {
	if tenantID == uuid.Nil {
		return nil, 0, ErrInvalidTenantID
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
	return s.repository.ListReleasedByTenant(ctx, tenantID, search, limit, offset)
}

func (s *Service) RetryImportRequest(ctx context.Context, tenantID, requestedByUserID, importID uuid.UUID) (*ImportRequest, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if requestedByUserID == uuid.Nil {
		return nil, errors.New("requested by user id is required")
	}
	if importID == uuid.Nil {
		return nil, ErrInvalidImportID
	}

	existing, err := s.repository.GetByID(ctx, tenantID, importID)
	if err != nil {
		return nil, err
	}

	switch existing.Status {
	case StatusFailed, StatusQuarantined:
		// Retryable terminal states.
	default:
		return nil, ErrImportNotRetryable
	}

	policy := deriveRetryPolicy(existing)
	attempts, err := s.countRetryAttemptsForKey(ctx, existing)
	if err != nil {
		return nil, err
	}
	if policy.MaxAttempts > 0 && attempts >= policy.MaxAttempts {
		return nil, &RetryAttemptLimitError{
			MaxAttempts: policy.MaxAttempts,
			Current:     attempts,
		}
	}

	if policy.Cooldown > 0 {
		nextEligibleAt := existing.UpdatedAt.UTC().Add(policy.Cooldown)
		now := time.Now().UTC()
		if nextEligibleAt.After(now) {
			return nil, &RetryBackoffError{Remaining: nextEligibleAt.Sub(now)}
		}
	}

	return s.CreateImportRequest(ctx, CreateImportRequestInput{
		TenantID:          tenantID,
		RequestedByUserID: requestedByUserID,
		RequestType:       existing.RequestType,
		SORRecordID:       existing.SORRecordID,
		SourceRegistry:    existing.SourceRegistry,
		SourceImageRef:    existing.SourceImageRef,
		RegistryAuthID:    existing.RegistryAuthID,
	})
}

func (s *Service) countRetryAttemptsForKey(ctx context.Context, existing *ImportRequest) (int, error) {
	if existing == nil {
		return 0, nil
	}
	rows, err := s.repository.ListByTenant(ctx, existing.TenantID, existing.RequestType, 500, 0)
	if err != nil {
		return 0, err
	}
	targetSOR := strings.TrimSpace(strings.ToLower(existing.SORRecordID))
	targetRegistry := strings.TrimSpace(strings.ToLower(existing.SourceRegistry))
	targetRef := strings.TrimSpace(strings.ToLower(existing.SourceImageRef))

	attempts := 0
	for _, row := range rows {
		if row == nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(row.SORRecordID)) != targetSOR {
			continue
		}
		if strings.TrimSpace(strings.ToLower(row.SourceRegistry)) != targetRegistry {
			continue
		}
		if strings.TrimSpace(strings.ToLower(row.SourceImageRef)) != targetRef {
			continue
		}
		attempts++
	}
	return attempts, nil
}

func deriveRetryPolicy(existing *ImportRequest) retryPolicy {
	failureClass := classifyFailureClass(existing)
	switch failureClass {
	case "dispatch":
		return retryPolicy{MaxAttempts: 5, Cooldown: 30 * time.Second}
	case "connectivity":
		return retryPolicy{MaxAttempts: 3, Cooldown: 2 * time.Minute}
	case "auth":
		return retryPolicy{MaxAttempts: 2, Cooldown: 5 * time.Minute}
	case "policy":
		return retryPolicy{MaxAttempts: 1, Cooldown: 10 * time.Minute}
	default:
		return retryPolicy{MaxAttempts: 3, Cooldown: 1 * time.Minute}
	}
}

func classifyFailureClass(existing *ImportRequest) string {
	if existing == nil {
		return "runtime"
	}
	if existing.Status == StatusQuarantined {
		return "policy"
	}
	message := strings.ToLower(strings.TrimSpace(existing.ErrorMessage))
	switch {
	case strings.HasPrefix(message, "dispatch_failed:"),
		strings.Contains(message, "waiting_for_dispatch"),
		strings.Contains(message, "dispatcher"):
		return "dispatch"
	case strings.Contains(message, "forbidden"),
		strings.Contains(message, "unauthorized"),
		strings.Contains(message, "authentication"):
		return "auth"
	case strings.Contains(message, "no such host"),
		strings.Contains(message, "connection refused"),
		strings.Contains(message, "i/o timeout"),
		strings.Contains(message, "dial tcp"),
		strings.Contains(message, "deadline exceeded"),
		strings.Contains(message, "timeout"):
		return "connectivity"
	case strings.Contains(message, "policy"),
		strings.Contains(message, "quarantine"):
		return "policy"
	default:
		return "runtime"
	}
}

func (s *Service) WithdrawImportRequest(ctx context.Context, tenantID, actorUserID, importID uuid.UUID, reason string) (*ImportRequest, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if actorUserID == uuid.Nil {
		return nil, errors.New("actor user id is required")
	}
	if importID == uuid.Nil {
		return nil, ErrInvalidImportID
	}

	req, err := s.repository.GetByID(ctx, tenantID, importID)
	if err != nil {
		return nil, err
	}

	if req.Status != StatusPending {
		return nil, ErrImportNotWithdrawable
	}

	withdrawReason := strings.TrimSpace(reason)
	if withdrawReason == "" {
		withdrawReason = "withdrawn by tenant user"
	}
	errorMessage := "Withdrawn: " + withdrawReason
	if err := s.repository.UpdateStatus(ctx, tenantID, importID, StatusFailed, errorMessage, req.InternalImageRef); err != nil {
		return nil, err
	}
	return s.repository.GetByID(ctx, tenantID, importID)
}

func (s *Service) MarkImportReleased(ctx context.Context, tenantID, importID, actorUserID uuid.UUID, reason string) (*ImportRequest, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if importID == uuid.Nil {
		return nil, ErrInvalidImportID
	}
	if actorUserID == uuid.Nil {
		return nil, errors.New("actor user id is required")
	}

	req, err := s.repository.GetByID(ctx, tenantID, importID)
	if err != nil {
		return nil, err
	}

	currentState := req.ReleaseState
	if currentState == "" || currentState == ReleaseStateUnknown {
		currentState = DeriveReleaseProjection(req).State
	}
	if currentState == ReleaseStateReleased {
		return req, nil
	}
	derivedProjection := DeriveReleaseProjection(req)
	if !derivedProjection.Eligible {
		return nil, ErrReleaseNotEligible
	}

	if err := ValidateReleaseTransition(currentState, ReleaseStateReleaseApproved); err != nil {
		return nil, err
	}
	if err := ValidateReleaseTransition(ReleaseStateReleaseApproved, ReleaseStateReleased); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	trimmedReason := strings.TrimSpace(reason)
	if err := s.repository.UpdateReleaseState(
		ctx,
		tenantID,
		importID,
		ReleaseStateReleased,
		"",
		&actorUserID,
		trimmedReason,
		&now,
		&now,
	); err != nil {
		return nil, err
	}
	return s.repository.GetByID(ctx, tenantID, importID)
}
