package rest

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/auth"
)

const (
	// APIKeyHeader is the HTTP header name for API key authentication
	APIKeyHeader = "X-API-Key"

	// ContextAPIKeyKey is the context key for storing the API key
	ContextAPIKeyKey = "api_key"

	// ContextTenantIDKey is the context key for storing the tenant ID from API key
	ContextTenantIDKey = "tenant_id"
)

// APIKeyMiddleware validates API keys in incoming requests
type APIKeyMiddleware struct {
	apiKeyService *auth.APIKeyService
	logger        *zap.Logger
}

// NewAPIKeyMiddleware creates a new API key middleware
func NewAPIKeyMiddleware(apiKeyService *auth.APIKeyService, logger *zap.Logger) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		apiKeyService: apiKeyService,
		logger:        logger,
	}
}

// Middleware returns the HTTP middleware function for API key validation
func (m *APIKeyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header
		apiKey := m.extractAPIKey(r)
		if apiKey == "" {
			m.logger.Debug("missing API key in request")
			http.Error(w, `{"error":"API key required"}`, http.StatusUnauthorized)
			return
		}

		// Validate API key
		keyData, err := m.apiKeyService.ValidateAPIKey(apiKey)
		if err != nil {
			m.logger.Warn("invalid API key", zap.Error(err))
			http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
			return
		}

		// Store key and tenant ID in context
		ctx := context.WithValue(r.Context(), ContextAPIKeyKey, keyData)
		ctx = context.WithValue(ctx, ContextTenantIDKey, keyData.TenantID)

		// Continue with next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SkipPathsMiddleware wraps the middleware and skips certain paths
// Use this for routes that don't require API key authentication
func (m *APIKeyMiddleware) SkipPathsMiddleware(skipPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip authentication
			for _, skipPath := range skipPaths {
				if m.matchPath(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Apply API key validation
			m.Middleware(next).ServeHTTP(w, r)
		})
	}
}

// extractAPIKey extracts the API key from the request
// Supports both "X-API-Key" header and "Authorization: Bearer" header
func (m *APIKeyMiddleware) extractAPIKey(r *http.Request) string {
	// Try X-API-Key header first
	if apiKey := r.Header.Get(APIKeyHeader); apiKey != "" {
		return apiKey
	}

	// Try Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	return ""
}

// matchPath checks if a request path matches a skip path pattern
// Simple implementation: exact match or prefix match with wildcard
func (m *APIKeyMiddleware) matchPath(path, skipPath string) bool {
	if skipPath == path {
		return true
	}

	// Support wildcard patterns like "/health*"
	if strings.HasSuffix(skipPath, "*") {
		prefix := strings.TrimSuffix(skipPath, "*")
		return strings.HasPrefix(path, prefix)
	}

	return false
}

// GetAPIKeyFromContext retrieves the API key from request context
func GetAPIKeyFromContext(r *http.Request) *auth.APIKey {
	apiKey, ok := r.Context().Value(ContextAPIKeyKey).(*auth.APIKey)
	if !ok {
		return nil
	}
	return apiKey
}

// GetTenantIDFromContext retrieves the tenant ID from request context (from API key)
func GetTenantIDFromContext(r *http.Request) string {
	tenantID, ok := r.Context().Value(ContextTenantIDKey).(string)
	if !ok {
		return ""
	}
	return tenantID
}

// CheckAPIKeyScope verifies that the API key has the required scope
func CheckAPIKeyScope(r *http.Request, requiredScope string) bool {
	apiKey := GetAPIKeyFromContext(r)
	if apiKey == nil {
		return false
	}
	return apiKey.HasScope(requiredScope)
}
