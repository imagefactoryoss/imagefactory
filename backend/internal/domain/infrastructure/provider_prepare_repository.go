package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ProviderPrepareRunRepository interface {
	CreateProviderPrepareRun(ctx context.Context, run *ProviderPrepareRun) error
	UpdateProviderPrepareRunStatus(ctx context.Context, id uuid.UUID, status ProviderPrepareRunStatus, startedAt, completedAt *time.Time, errorMessage *string, resultSummary map[string]interface{}) error
	GetProviderPrepareRun(ctx context.Context, id uuid.UUID) (*ProviderPrepareRun, error)
	FindActiveProviderPrepareRunByProvider(ctx context.Context, providerID uuid.UUID) (*ProviderPrepareRun, error)
	ListProviderPrepareRunsByProvider(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*ProviderPrepareRun, error)

	AddProviderPrepareRunCheck(ctx context.Context, check *ProviderPrepareRunCheck) error
	ListProviderPrepareRunChecks(ctx context.Context, runID uuid.UUID, limit, offset int) ([]*ProviderPrepareRunCheck, error)
}

type ProviderPrepareSummaryRepository interface {
	ListLatestProviderPrepareSummaries(ctx context.Context, providerIDs []uuid.UUID) (map[uuid.UUID]*ProviderPrepareLatestSummary, error)
}
