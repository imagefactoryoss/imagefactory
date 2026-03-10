package policy

import (
	"context"

	"github.com/google/uuid"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

type OperationCapabilityChecker struct {
	systemConfigService *systemconfig.Service
}

func NewOperationCapabilityChecker(systemConfigService *systemconfig.Service) *OperationCapabilityChecker {
	return &OperationCapabilityChecker{systemConfigService: systemConfigService}
}

func (c *OperationCapabilityChecker) IsImportEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if c == nil || c.systemConfigService == nil {
		return false, nil
	}
	if tenantID == uuid.Nil {
		return false, nil
	}
	scope := tenantID
	cfg, err := c.systemConfigService.GetOperationCapabilitiesConfig(ctx, &scope)
	if err != nil {
		return false, err
	}
	// Quarantine request is the canonical tenant trigger for import/scan flow.
	return cfg.QuarantineRequest, nil
}

func (c *OperationCapabilityChecker) IsOnDemandScanEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if c == nil || c.systemConfigService == nil {
		return false, nil
	}
	if tenantID == uuid.Nil {
		return false, nil
	}
	scope := tenantID
	cfg, err := c.systemConfigService.GetOperationCapabilitiesConfig(ctx, &scope)
	if err != nil {
		return false, err
	}
	return cfg.OnDemandImageScan, nil
}

func (c *OperationCapabilityChecker) IsQuarantineReleaseEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if c == nil || c.systemConfigService == nil {
		return false, nil
	}
	if tenantID == uuid.Nil {
		return false, nil
	}
	scope := tenantID
	cfg, err := c.systemConfigService.GetOperationCapabilitiesConfig(ctx, &scope)
	if err != nil {
		return false, err
	}
	return cfg.QuarantineRelease, nil
}
