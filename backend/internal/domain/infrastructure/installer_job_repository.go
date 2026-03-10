package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TektonInstallerJobRepository interface {
	ClaimNextPendingInstallerJob(ctx context.Context) (*TektonInstallerJob, error)
	CreateInstallerJob(ctx context.Context, job *TektonInstallerJob) error
	FindInstallerJobByProviderAndIdempotencyKey(ctx context.Context, providerID uuid.UUID, operation TektonInstallerOperation, idempotencyKey string) (*TektonInstallerJob, error)
	UpdateInstallerJobStatus(ctx context.Context, id uuid.UUID, status TektonInstallerJobStatus, startedAt, completedAt *time.Time, errorMessage *string) error
	GetInstallerJob(ctx context.Context, id uuid.UUID) (*TektonInstallerJob, error)
	ListInstallerJobsByProvider(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*TektonInstallerJob, error)

	AddInstallerJobEvent(ctx context.Context, event *TektonInstallerJobEvent) error
	ListInstallerJobEvents(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*TektonInstallerJobEvent, error)
}
