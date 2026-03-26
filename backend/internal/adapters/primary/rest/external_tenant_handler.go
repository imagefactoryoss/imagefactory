package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// ExternalTenantHandler handles requests to look up tenants from AppHQ.
type ExternalTenantHandler struct {
	logger              *zap.Logger
	systemConfigService *systemconfig.Service
	httpClient          *http.Client
}

// NewExternalTenantHandler creates a new external tenant handler.
func NewExternalTenantHandler(logger *zap.Logger, systemConfigService *systemconfig.Service, httpClient *http.Client) *ExternalTenantHandler {
	return &ExternalTenantHandler{
		logger:              logger,
		systemConfigService: systemConfigService,
		httpClient:          httpClient,
	}
}

// ExternalTenant represents a tenant record returned to the frontend.
type ExternalTenant struct {
	ID                string `json:"id"`
	TenantID          string `json:"tenant_id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Description       string `json:"description"`
	ContactEmail      string `json:"contact_email"`
	Status            string `json:"status"`
	Company           string `json:"company"`
	CriticalApp       string `json:"critical_app"`
	Org               string `json:"org"`
	AppStrategy       string `json:"app_strategy"`
	RecordType        string `json:"record_type"`
	InternalFlag      string `json:"internal_flag"`
	ProdDate          string `json:"prod_date"`
	TechExecEmail     string `json:"tech_exec_email"`
	LOBPrimaryEmail   string `json:"lob_primary_email"`
	AppMgrNetID       string `json:"app_mgr_netid"`
	AppMgrFirstName   string `json:"app_mgr_first_name"`
	AppMgrLastName    string `json:"app_mgr_last_name"`
	AppMgrEmail       string `json:"app_mgr_email"`
}

// apphqRequest is the JSON body sent to the AppHQ exec API.
type apphqRequest struct {
	System     string       `json:"system"`
	SystemName string       `json:"system_name"`
	Run        string       `json:"run"`
	ObjCode    string       `json:"obj_cd"`
	Params     []apphqParam `json:"params"`
	ViewJSON   string       `json:"viewjson"`
}

type apphqParam struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Comparator string `json:"comparator"`
}

// apphqResponse is the JSON response from the AppHQ exec API.
type apphqResponse struct {
	TableData []map[string]interface{} `json:"tableData"`
}

// oauthTokenResponse represents the OAuth token endpoint response.
type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

const (
	apphqMaxAttempts = 3
	apphqBaseDelay   = 200 * time.Millisecond
)

// getAppHQConfig retrieves the AppHQ configuration from runtime services config.
func (h *ExternalTenantHandler) getAppHQConfig(ctx context.Context) (*systemconfig.RuntimeServicesConfig, error) {
	config, err := h.systemConfigService.GetConfigByKey(ctx, nil, "runtime_services")
	if err != nil {
		return nil, fmt.Errorf("failed to load runtime services config: %w", err)
	}

	runtimeCfg, err := config.GetRuntimeServicesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse runtime services config: %w", err)
	}

	if runtimeCfg.AppHQEnabled == nil || !*runtimeCfg.AppHQEnabled {
		return nil, fmt.Errorf("AppHQ tenant lookup is not enabled")
	}

	if runtimeCfg.AppHQOAuthTokenURL == "" || runtimeCfg.AppHQClientID == "" || runtimeCfg.AppHQClientSecret == "" || runtimeCfg.AppHQAPIURL == "" {
		return nil, fmt.Errorf("AppHQ configuration is incomplete")
	}

	return runtimeCfg, nil
}

// fetchOAuthToken obtains a client-credentials OAuth token from the configured endpoint.
func (h *ExternalTenantHandler) fetchOAuthToken(ctx context.Context, cfg *systemconfig.RuntimeServicesConfig) (string, error) {
	form := url.Values{}
	form.Set("client_id", cfg.AppHQClientID)
	form.Set("client_secret", cfg.AppHQClientSecret)
	form.Set("grant_type", "client_credentials")

	var lastErr error
	for attempt := 1; attempt <= apphqMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.AppHQOAuthTokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return "", fmt.Errorf("failed to build token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := h.httpClient.Do(req)
		if err != nil {
			lastErr = err
			h.logger.Warn("OAuth token request retry", zap.Int("attempt", attempt), zap.Error(err))
			if attempt < apphqMaxAttempts {
				if sleepErr := sleepWithContext(ctx, backoffDelay(attempt, apphqBaseDelay)); sleepErr != nil {
					return "", sleepErr
				}
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("OAuth token endpoint returned %d", resp.StatusCode)
			h.logger.Warn("OAuth token request retry", zap.Int("attempt", attempt), zap.Int("status", resp.StatusCode))
			if attempt < apphqMaxAttempts {
				if sleepErr := sleepWithContext(ctx, backoffDelay(attempt, apphqBaseDelay)); sleepErr != nil {
					return "", sleepErr
				}
			}
			continue
		}

		var tokenResp oauthTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return "", fmt.Errorf("failed to decode token response: %w", err)
		}

		if tokenResp.AccessToken == "" {
			return "", fmt.Errorf("empty access token in response")
		}

		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("OAuth token request failed after %d attempts: %w", apphqMaxAttempts, lastErr)
}

// queryAppHQ calls the AppHQ exec API with the given search parameters.
func (h *ExternalTenantHandler) queryAppHQ(ctx context.Context, cfg *systemconfig.RuntimeServicesConfig, token string, params []apphqParam) (*apphqResponse, error) {
	reqBody := apphqRequest{
		System:     cfg.AppHQSystem,
		SystemName: cfg.AppHQSystemName,
		Run:        cfg.AppHQRun,
		ObjCode:    cfg.AppHQObjCode,
		Params:     params,
		ViewJSON:   "true",
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AppHQ request: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= apphqMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.AppHQAPIURL, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to build AppHQ request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.httpClient.Do(req)
		if err != nil {
			lastErr = err
			h.logger.Warn("AppHQ API request retry", zap.Int("attempt", attempt), zap.Error(err))
			if attempt < apphqMaxAttempts {
				if sleepErr := sleepWithContext(ctx, backoffDelay(attempt, apphqBaseDelay)); sleepErr != nil {
					return nil, sleepErr
				}
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			lastErr = fmt.Errorf("AppHQ API returned %d", resp.StatusCode)
			h.logger.Warn("AppHQ API request retry", zap.Int("attempt", attempt), zap.Int("status", resp.StatusCode))
			if attempt < apphqMaxAttempts {
				if sleepErr := sleepWithContext(ctx, backoffDelay(attempt, apphqBaseDelay)); sleepErr != nil {
					return nil, sleepErr
				}
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("AppHQ API returned status %d", resp.StatusCode)
		}

		var apphqResp apphqResponse
		if err := json.Unmarshal(body, &apphqResp); err != nil {
			return nil, fmt.Errorf("failed to decode AppHQ response: %w", err)
		}

		return &apphqResp, nil
	}

	return nil, fmt.Errorf("AppHQ API request failed after %d attempts: %w", apphqMaxAttempts, lastErr)
}

// mapAppHQToTenant converts a raw AppHQ table row into an ExternalTenant.
func mapAppHQToTenant(row map[string]interface{}) ExternalTenant {
	str := func(key string) string {
		if v, ok := row[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
		return ""
	}

	appID := str("APP_ID")
	shortName := str("APP_SHORT_NAME")
	slug := strings.ToLower(strings.ReplaceAll(shortName, " ", "-"))

	// Prefer APP_FULL_NAME, fall back to APP_DESC
	description := str("APP_FULL_NAME")
	if description == "" {
		description = str("APP_DESC")
	}

	// Best-effort contact email from available fields
	contactEmail := str("TECHCONTACT_MAILID")
	if contactEmail == "" {
		contactEmail = str("SECONDLVL_PROD_MAILID")
	}
	if contactEmail == "" {
		contactEmail = str("APP_ACCESS_ADMIN_MAILID")
	}

	// Owner fields from APP_MGR
	appMgrNetID := str("APP_MGR_NETID")
	appMgrFirstName := str("APP_MGR_FNAME")
	appMgrLastName := str("APP_MGR_LNAME")
	// Try to get email for APP_MGR
	appMgrEmail := contactEmail
	// If TECHCONTACT_MAILID doesn't match APP_MGR, try to find a better match if available
	// (for now, use contactEmail as best effort)

	return ExternalTenant{
		ID:                appID,
		TenantID:          appID,
		Name:              shortName,
		Slug:              slug,
		Description:       description,
		ContactEmail:      contactEmail,
		Status:            str("APPSTATUS"),
		Company:           str("COMPANY"),
		CriticalApp:       str("CRITICAL_APP"),
		Org:               str("ALGN_CIO_ORG"),
		AppStrategy:       str("APPSTRATEGY"),
		RecordType:        str("RECORD_TYPE"),
		InternalFlag:      str("INTERNAL_FLAG"),
		ProdDate:          str("APP_STATUS_PROD_DATE"),
		TechExecEmail:     str("TECHEXEC_MAILID"),
		LOBPrimaryEmail:   str("LOB_PRIMARY_MAILID"),
		AppMgrNetID:       appMgrNetID,
		AppMgrFirstName:   appMgrFirstName,
		AppMgrLastName:    appMgrLastName,
		AppMgrEmail:       appMgrEmail,
	}
}

func backoffDelay(attempt int, base time.Duration) time.Duration {
	multiplier := 1 << (attempt - 1)
	return time.Duration(multiplier) * base
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

// ListExternalTenants handles GET /api/v1/external-tenants?q=search_query
// Searches AppHQ for tenants matching the query by APP_ID or APP_SHORT_NAME.
func (h *ExternalTenantHandler) ListExternalTenants(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	cfg, err := h.getAppHQConfig(r.Context())
	if err != nil {
		h.logger.Error("AppHQ configuration unavailable", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	token, err := h.fetchOAuthToken(r.Context(), cfg)
	if err != nil {
		h.logger.Error("Failed to obtain AppHQ OAuth token", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to authenticate with AppHQ"})
		return
	}

	// Build search parameters – include active statuses and optionally filter by query
	params := []apphqParam{
		{Key: "APPSTATUS", Value: "In Production, In Development, Added, Projected Production", Comparator: "IN"},
	}
	if query != "" {
		params = append(params, apphqParam{Key: "APP_ID", Value: query, Comparator: "IN"})
	}

	apphqResp, err := h.queryAppHQ(r.Context(), cfg, token, params)
	if err != nil {
		h.logger.Error("Failed to query AppHQ", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query AppHQ"})
		return
	}

	tenants := make([]ExternalTenant, 0, len(apphqResp.TableData))
	for _, row := range apphqResp.TableData {
		tenants = append(tenants, mapAppHQToTenant(row))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenants": tenants,
		"total":   len(tenants),
	})
}

// GetExternalTenant handles GET /api/v1/external-tenants/{id}
// Looks up a single tenant in AppHQ by APP_ID.
func (h *ExternalTenantHandler) GetExternalTenant(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/external-tenants/"):]
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tenant ID required"})
		return
	}

	cfg, err := h.getAppHQConfig(r.Context())
	if err != nil {
		h.logger.Error("AppHQ configuration unavailable", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	token, err := h.fetchOAuthToken(r.Context(), cfg)
	if err != nil {
		h.logger.Error("Failed to obtain AppHQ OAuth token", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to authenticate with AppHQ"})
		return
	}

	params := []apphqParam{
		{Key: "APP_ID", Value: id, Comparator: "IN"},
	}

	apphqResp, err := h.queryAppHQ(r.Context(), cfg, token, params)
	if err != nil {
		h.logger.Error("Failed to query AppHQ", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query AppHQ"})
		return
	}

	if len(apphqResp.TableData) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tenant not found"})
		return
	}

	tenant := mapAppHQToTenant(apphqResp.TableData[0])

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tenant)
}
