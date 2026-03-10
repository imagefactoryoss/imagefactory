package imageimport

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrImportNotFound          = errors.New("external image import request not found")
	ErrInvalidImportID         = errors.New("invalid import request id")
	ErrInvalidTenantID         = errors.New("invalid tenant id")
	ErrInvalidSourceImageRef   = errors.New("invalid source image reference")
	ErrInvalidSourceRegistry   = errors.New("invalid source registry")
	ErrInvalidSORRecordID      = errors.New("invalid SOR record id")
	ErrSORRegistrationRequired = errors.New("epr registration required")
	ErrOperationNotEntitled    = errors.New("operation capability not entitled")
	ErrCatalogImageNotReady    = errors.New("catalog image not ready for evidence sync")
	ErrImportNotRetryable      = errors.New("import request is not retryable")
	ErrRetryBackoffActive      = errors.New("retry backoff active")
	ErrRetryAttemptLimitReached = errors.New("retry attempt limit reached")
	ErrImportNotWithdrawable   = errors.New("import request is not withdrawable")
	ErrReleaseNotEligible      = errors.New("import request is not eligible for release")
)

type RetryBackoffError struct {
	Remaining time.Duration
}

func (e *RetryBackoffError) Error() string {
	return ErrRetryBackoffActive.Error()
}

func (e *RetryBackoffError) Unwrap() error {
	return ErrRetryBackoffActive
}

type RetryAttemptLimitError struct {
	MaxAttempts int
	Current     int
}

func (e *RetryAttemptLimitError) Error() string {
	return ErrRetryAttemptLimitReached.Error()
}

func (e *RetryAttemptLimitError) Unwrap() error {
	return ErrRetryAttemptLimitReached
}

type Status string

const (
	StatusPending     Status = "pending"
	StatusApproved    Status = "approved"
	StatusImporting   Status = "importing"
	StatusSuccess     Status = "success"
	StatusFailed      Status = "failed"
	StatusQuarantined Status = "quarantined"
)

type RequestType string

const (
	RequestTypeQuarantine RequestType = "quarantine"
	RequestTypeScan       RequestType = "scan"
)

type ImportRequest struct {
	ID                   uuid.UUID
	TenantID             uuid.UUID
	RequestedByUserID    uuid.UUID
	RequestType          RequestType
	SORRecordID          string
	SourceRegistry       string
	SourceImageRef       string
	RegistryAuthID       *uuid.UUID
	Status               Status
	ErrorMessage         string
	InternalImageRef     string
	PipelineRunName      string
	PipelineNamespace    string
	PolicyDecision       string
	PolicyReasonsJSON    string
	PolicySnapshotJSON   string
	ScanSummaryJSON      string
	SBOMSummaryJSON      string
	SBOMEvidenceJSON     string
	SourceImageDigest    string
	ReleaseState         ReleaseState
	ReleaseBlockerReason string
	ReleaseActorUserID   *uuid.UUID
	ReleaseReason        string
	ReleaseRequestedAt   *time.Time
	ReleasedAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ReleasedArtifact struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	RequestedByUserID  uuid.UUID
	SORRecordID        string
	SourceRegistry     string
	SourceImageRef     string
	InternalImageRef   string
	SourceImageDigest  string
	PolicyDecision     string
	PolicySnapshotJSON string
	ReleaseState       ReleaseState
	ReleaseReason      string
	ReleaseActorUserID *uuid.UUID
	ReleaseRequestedAt *time.Time
	ReleasedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewImportRequest(
	tenantID, requestedByUserID uuid.UUID,
	requestType RequestType,
	sorRecordID, sourceRegistry, sourceImageRef string,
	registryAuthID *uuid.UUID,
) (*ImportRequest, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if requestedByUserID == uuid.Nil {
		return nil, errors.New("requested by user id is required")
	}
	if requestType == "" {
		requestType = RequestTypeQuarantine
	}
	if requestType != RequestTypeQuarantine && requestType != RequestTypeScan {
		return nil, errors.New("invalid request type")
	}
	sorRecordID = strings.TrimSpace(sorRecordID)
	if requestType == RequestTypeQuarantine && sorRecordID == "" {
		return nil, ErrInvalidSORRecordID
	}
	if requestType == RequestTypeScan && sorRecordID == "" {
		sorRecordID = "on-demand-scan"
	}
	sourceRegistry = strings.TrimSpace(sourceRegistry)
	if sourceRegistry == "" {
		return nil, ErrInvalidSourceRegistry
	}
	sourceImageRef = strings.TrimSpace(sourceImageRef)
	if sourceImageRef == "" {
		return nil, ErrInvalidSourceImageRef
	}

	now := time.Now().UTC()
	return &ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: requestedByUserID,
		RequestType:       requestType,
		SORRecordID:       sorRecordID,
		SourceRegistry:    sourceRegistry,
		SourceImageRef:    sourceImageRef,
		RegistryAuthID:    registryAuthID,
		Status:            StatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}
