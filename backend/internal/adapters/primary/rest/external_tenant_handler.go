package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// ExternalTenantHandler handles requests to fetch tenants from external service
type ExternalTenantHandler struct {
	logger              *zap.Logger
	systemConfigService *systemconfig.Service
	httpClient          *http.Client
}

// NewExternalTenantHandler creates a new external tenant handler
func NewExternalTenantHandler(logger *zap.Logger, systemConfigService *systemconfig.Service, httpClient *http.Client) *ExternalTenantHandler {
	return &ExternalTenantHandler{
		logger:              logger,
		systemConfigService: systemConfigService,
		httpClient:          httpClient,
	}
}

// ExternalTenant represents a tenant from the external service
type ExternalTenant struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	Industry     string `json:"industry"`
	Country      string `json:"country"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ExternalTenantsResponse represents the response from the external tenant service
type ExternalTenantsResponse struct {
	Count   int              `json:"count"`
	Data    []ExternalTenant `json:"data"`
	Success bool             `json:"success"`
}

// makeAuthenticatedRequest creates an HTTP request with API key authentication
func (h *ExternalTenantHandler) makeAuthenticatedRequest(ctx context.Context, method, url string) (*http.Response, error) {
	// Get external service configuration
	// For now, we'll look for a service named "tenant-service" - this could be made configurable later
	serviceConfig, err := h.systemConfigService.GetExternalService(ctx, nil, "tenant-service")
	if err != nil {
		h.logger.Error("Failed to get external service configuration", zap.Error(err))
		return nil, fmt.Errorf("external service configuration not found")
	}

	if !serviceConfig.Enabled {
		return nil, fmt.Errorf("external service is disabled")
	}

	if h.httpClient == nil {
		return nil, fmt.Errorf("http client not configured")
	}

	var lastErr error
	for attempt := 1; attempt <= externalTenantMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}

		// Add custom headers if configured, otherwise use default X-API-Key header
		if len(serviceConfig.Headers) > 0 {
			for key, value := range serviceConfig.Headers {
				req.Header.Set(key, value)
			}
		} else {
			// Fallback to default X-API-Key header for external tenant service
			req.Header.Set("X-API-Key", serviceConfig.APIKey)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.httpClient.Do(req)
		if err == nil && !shouldRetryExternalTenantStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			lastErr = err
			h.logger.Warn("External tenant request retry", zap.Int("attempt", attempt), zap.String("url", url), zap.Error(err))
		} else {
			lastErr = fmt.Errorf("external tenant service error: %s", resp.Status)
			h.logger.Warn("External tenant request retry", zap.Int("attempt", attempt), zap.String("url", url), zap.Int("status", resp.StatusCode), zap.Error(lastErr))
			resp.Body.Close()
		}

		if attempt < externalTenantMaxAttempts {
			if sleepErr := sleepWithContext(ctx, backoffDelay(attempt, externalTenantBaseDelay)); sleepErr != nil {
				return nil, sleepErr
			}
		}
	}

	return nil, lastErr
}

const (
	externalTenantMaxAttempts = 3
	externalTenantBaseDelay   = 200 * time.Millisecond
)

func shouldRetryExternalTenantStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
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
// Returns a searchable list of tenants from external service
func (h *ExternalTenantHandler) ListExternalTenants(w http.ResponseWriter, r *http.Request) {
	// Get search query from URL parameters
	query := r.URL.Query().Get("q")

	// Get external service configuration
	serviceConfig, err := h.systemConfigService.GetExternalService(r.Context(), nil, "tenant-service")
	if err != nil {
		h.logger.Error("Failed to get external service configuration", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "External service configuration not found"})
		return
	}

	if !serviceConfig.Enabled {
		h.logger.Warn("External service is disabled")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "External service is disabled"})
		return
	}

	// Build URL to external service
	externalURL := fmt.Sprintf("%s/api/tenants", serviceConfig.URL)
	if query != "" {
		externalURL += "?q=" + url.QueryEscape(query)
	}

	// Call external service with authentication
	resp, err := h.makeAuthenticatedRequest(r.Context(), "GET", externalURL)
	if err != nil {
		h.logger.Error("Failed to call external tenant service", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "External service unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Warn("External service returned error", zap.Int("status", resp.StatusCode))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	// Decode response from external service
	var externalResp ExternalTenantsResponse
	if err := json.NewDecoder(resp.Body).Decode(&externalResp); err != nil {
		h.logger.Error("Failed to decode external tenant service response", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse response"})
		return
	}

	// Transform response to expected format
	clientResp := map[string]interface{}{
		"tenants": externalResp.Data,
		"total":   externalResp.Count,
	}

	// Return response to client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(clientResp)
}

// GetExternalTenant handles GET /api/v1/external-tenants/:id
// Returns a specific tenant from external service by ID
func (h *ExternalTenantHandler) GetExternalTenant(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	id := r.URL.Path[len("/api/v1/external-tenants/"):]
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tenant ID required"})
		return
	}

	// Get external service configuration
	serviceConfig, err := h.systemConfigService.GetExternalService(r.Context(), nil, "tenant-service")
	if err != nil {
		h.logger.Error("Failed to get external service configuration", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "External service configuration not found"})
		return
	}

	if !serviceConfig.Enabled {
		h.logger.Warn("External service is disabled")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "External service is disabled"})
		return
	}

	// Call external service with authentication
	externalURL := fmt.Sprintf("%s/api/tenants/%s", serviceConfig.URL, id)
	resp, err := h.makeAuthenticatedRequest(r.Context(), "GET", externalURL)
	if err != nil {
		h.logger.Error("Failed to call external tenant service", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "External service unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	// Decode and return tenant
	var tenant ExternalTenant
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		h.logger.Error("Failed to decode external tenant service response", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse response"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tenant)
}
