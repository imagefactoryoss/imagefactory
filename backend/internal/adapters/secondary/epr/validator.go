package epr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

type ExternalValidator struct {
	logger              *zap.Logger
	systemConfigService *systemconfig.Service
	httpClient          *http.Client
	approvedChecker     ApprovedRegistrationChecker
}

type ApprovedRegistrationChecker interface {
	IsApprovedEPRRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error)
	GetApprovedEPRRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*eprregistration.LifecycleStatus, error)
}

func NewExternalValidator(logger *zap.Logger, systemConfigService *systemconfig.Service, httpClient *http.Client) *ExternalValidator {
	return &ExternalValidator{
		logger:              logger,
		systemConfigService: systemConfigService,
		httpClient:          httpClient,
	}
}

func (v *ExternalValidator) SetApprovedRegistrationChecker(checker ApprovedRegistrationChecker) {
	if v == nil {
		return
	}
	v.approvedChecker = checker
}

type eprRecordResponse struct {
	ID      string `json:"id"`
	Tenant  string `json:"tenant_id"`
	Active  bool   `json:"active"`
	Success bool   `json:"success"`
}

func (v *ExternalValidator) ValidateRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error) {
	eprRecordID = strings.TrimSpace(eprRecordID)
	if eprRecordID == "" {
		return false, nil
	}
	if v.approvedChecker != nil {
		lifecycleStatus, lifecycleErr := v.approvedChecker.GetApprovedEPRRegistrationLifecycleStatus(ctx, tenantID, eprRecordID)
		if lifecycleErr != nil {
			return false, fmt.Errorf("failed to check approved epr lifecycle status: %w", lifecycleErr)
		}
		if lifecycleStatus != nil {
			switch *lifecycleStatus {
			case eprregistration.LifecycleStatusSuspended, eprregistration.LifecycleStatusExpired:
				return false, nil
			case eprregistration.LifecycleStatusActive, eprregistration.LifecycleStatusExpiring:
				return true, nil
			}
		}
		approved, err := v.approvedChecker.IsApprovedEPRRegistration(ctx, tenantID, eprRecordID)
		if err != nil {
			return false, fmt.Errorf("failed to check approved epr registration request: %w", err)
		}
		if approved {
			return true, nil
		}
	}

	eprConfig, cfgErr := v.systemConfigService.GetSORRegistrationConfig(ctx, &tenantID)
	if cfgErr != nil {
		return false, fmt.Errorf("failed to load epr registration policy: %w", cfgErr)
	}
	if !eprConfig.Enforce {
		v.logger.Info("EPR validation bypassed because enforcement is disabled", zap.String("tenant_id", tenantID.String()))
		return true, nil
	}

	serviceConfig, err := v.systemConfigService.GetExternalService(ctx, nil, "epr-service")
	if err != nil {
		return v.handleRuntimeError(tenantID, eprRecordID, eprConfig, fmt.Errorf("external EPR service configuration not found"))
	}
	if !serviceConfig.Enabled {
		return v.handleRuntimeError(tenantID, eprRecordID, eprConfig, fmt.Errorf("external EPR service is disabled"))
	}
	if v.httpClient == nil {
		return v.handleRuntimeError(tenantID, eprRecordID, eprConfig, fmt.Errorf("http client not configured"))
	}

	requestURL := fmt.Sprintf(
		"%s/api/epr/records/%s?tenant_id=%s",
		strings.TrimRight(serviceConfig.URL, "/"),
		url.PathEscape(eprRecordID),
		url.QueryEscape(tenantID.String()),
	)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		ok, retry, callErr := v.callEPR(ctx, requestURL, serviceConfig)
		if callErr == nil {
			return ok, nil
		}
		lastErr = callErr
		if !retry || attempt == 3 {
			break
		}
		if sleepErr := sleepWithContext(ctx, backoffDelay(attempt)); sleepErr != nil {
			if errors.Is(sleepErr, context.Canceled) || errors.Is(sleepErr, context.DeadlineExceeded) {
				return false, sleepErr
			}
			return v.handleRuntimeError(tenantID, eprRecordID, eprConfig, sleepErr)
		}
	}

	v.logger.Warn("EPR validation call failed", zap.String("tenant_id", tenantID.String()), zap.String("epr_record_id", eprRecordID), zap.Error(lastErr))
	return v.handleRuntimeError(tenantID, eprRecordID, eprConfig, lastErr)
}

func (v *ExternalValidator) callEPR(ctx context.Context, requestURL string, serviceConfig *systemconfig.ExternalServiceConfig) (ok bool, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return false, false, err
	}
	if len(serviceConfig.Headers) > 0 {
		for key, value := range serviceConfig.Headers {
			req.Header.Set(key, value)
		}
	} else {
		req.Header.Set("X-API-Key", serviceConfig.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return false, true, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload eprRecordResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
			return false, false, decodeErr
		}
		return payload.Active, false, nil
	case http.StatusNotFound, http.StatusForbidden:
		return false, false, nil
	case http.StatusTooManyRequests:
		return false, true, fmt.Errorf("epr service throttled request")
	default:
		if resp.StatusCode >= 500 {
			return false, true, fmt.Errorf("epr service returned status %d", resp.StatusCode)
		}
		return false, false, fmt.Errorf("epr service returned status %d", resp.StatusCode)
	}
}

func backoffDelay(attempt int) time.Duration {
	multiplier := 1 << (attempt - 1)
	return time.Duration(multiplier) * 200 * time.Millisecond
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (v *ExternalValidator) handleRuntimeError(tenantID uuid.UUID, eprRecordID string, cfg *systemconfig.SORRegistrationConfig, runtimeErr error) (bool, error) {
	if cfg == nil {
		return false, runtimeErr
	}

	switch strings.ToLower(strings.TrimSpace(cfg.RuntimeErrorMode)) {
	case "deny":
		v.logger.Warn("EPR validation runtime error handled in deny mode",
			zap.String("tenant_id", tenantID.String()),
			zap.String("epr_record_id", eprRecordID),
			zap.Error(runtimeErr))
		return false, nil
	case "allow":
		v.logger.Warn("EPR validation runtime error handled in allow mode",
			zap.String("tenant_id", tenantID.String()),
			zap.String("epr_record_id", eprRecordID),
			zap.Error(runtimeErr))
		return true, nil
	default:
		return false, runtimeErr
	}
}
