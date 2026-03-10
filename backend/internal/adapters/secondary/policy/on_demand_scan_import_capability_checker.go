package policy

import (
	"context"

	"github.com/google/uuid"
)

// OnDemandScanImportCapabilityChecker adapts on-demand scan entitlement to the image import capability contract.
type OnDemandScanImportCapabilityChecker struct {
	base *OperationCapabilityChecker
}

func NewOnDemandScanImportCapabilityChecker(base *OperationCapabilityChecker) *OnDemandScanImportCapabilityChecker {
	return &OnDemandScanImportCapabilityChecker{base: base}
}

func (c *OnDemandScanImportCapabilityChecker) IsImportEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if c == nil || c.base == nil {
		return false, nil
	}
	return c.base.IsOnDemandScanEntitled(ctx, tenantID)
}
