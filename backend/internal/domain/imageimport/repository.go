package imageimport

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, req *ImportRequest) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*ImportRequest, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, requestType RequestType, limit, offset int) ([]*ImportRequest, error)
	ListAll(ctx context.Context, requestType RequestType, limit, offset int) ([]*ImportRequest, error)
	ListReleasedByTenant(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*ReleasedArtifact, int, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status Status, errorMessage, internalImageRef string) error
	UpdatePipelineRefs(ctx context.Context, tenantID, id uuid.UUID, pipelineRunName, pipelineNamespace string) error
	UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, evidence ImportEvidence) error
	UpdateReleaseState(ctx context.Context, tenantID, id uuid.UUID, state ReleaseState, blockerReason string, actorUserID *uuid.UUID, reason string, requestedAt, releasedAt *time.Time) error
	SyncEvidenceToCatalog(ctx context.Context, tenantID, id uuid.UUID) error
}

type ImportEvidence struct {
	PolicyDecision     string
	PolicyReasonsJSON  string
	PolicySnapshotJSON string
	ScanSummaryJSON    string
	SBOMSummaryJSON    string
	SBOMEvidenceJSON   string
	SourceImageDigest  string
}

type SORValidator interface {
	ValidateRegistration(ctx context.Context, tenantID uuid.UUID, sorRecordID string) (bool, error)
}

type CapabilityChecker interface {
	IsImportEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error)
}

type ApprovalRequester interface {
	CreateImportApproval(ctx context.Context, req *ImportRequest) error
}
