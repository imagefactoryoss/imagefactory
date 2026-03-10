package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"go.uber.org/zap"
)

// AuditMiddleware provides audit logging for HTTP requests
type AuditMiddleware struct {
	auditService *audit.Service
	logger       *zap.Logger
}

// NewAuditMiddleware creates a new AuditMiddleware instance
func NewAuditMiddleware(auditService *audit.Service, logger *zap.Logger) *AuditMiddleware {
	return &AuditMiddleware{
		auditService: auditService,
		logger:       logger,
	}
}

// Wrap applies audit logging middleware to the next handler
func (am *AuditMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &ResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			bodySize:       0,
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Calculate request duration
		duration := time.Since(start)

		// Extract user and tenant information from context
		authCtx, _ := r.Context().Value("auth").(*AuthContext)
		var userID *uuid.UUID
		var tenantID *uuid.UUID
		isSystemAdmin := false
		if authCtx != nil {
			userID = &authCtx.UserID
			isSystemAdmin = authCtx.IsSystemAdmin
			if authCtx.TenantID != uuid.Nil {
				tenantID = &authCtx.TenantID
			}
		}

		// Skip audit logging for health checks and static assets
		if am.shouldSkipAudit(r.URL.Path) {
			return
		}

		// Log API call to audit service
		if am.auditService != nil && (tenantID != nil || isSystemAdmin) {
			details := map[string]interface{}{
				"method":        r.Method,
				"path":          r.URL.Path,
				"query":         r.URL.RawQuery,
				"status_code":   wrapped.statusCode,
				"response_size": wrapped.bodySize,
				"duration_ms":   duration.Milliseconds(),
				"remote_addr":   am.extractClientIP(r),
				"user_agent":    r.UserAgent(),
			}

			severity := audit.AuditSeverityInfo
			if wrapped.statusCode >= 400 && wrapped.statusCode < 500 {
				severity = audit.AuditSeverityWarning
			} else if wrapped.statusCode >= 500 {
				severity = audit.AuditSeverityError
			}

			message := "API call completed"
			if wrapped.statusCode >= 400 {
				message = "API call failed"
			}

			err := am.auditService.LogEvent(r.Context(), &audit.AuditEvent{
				TenantID:  tenantID,
				UserID:    userID,
				EventType: audit.AuditEventType("api_call"),
				Severity:  severity,
				Resource:  "api",
				Action:    strings.ToLower(r.Method),
				IPAddress: am.extractClientIP(r),
				UserAgent: r.UserAgent(),
				Details:   details,
				Message:   message,
				Timestamp: time.Now().UTC(),
			})

			if err != nil {
				am.logger.Error("Failed to log API call to audit",
					zap.Error(err),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
			}
		}
	})
}

// shouldSkipAudit determines if the request should be skipped from audit logging
func (am *AuditMiddleware) shouldSkipAudit(path string) bool {
	// Skip health checks, metrics, and static assets
	skipPaths := []string{
		"/health",
		"/ready",
		"/alive",
		"/metrics",
		"/favicon.ico",
		"/static/",
		"/assets/",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// extractUserID extracts user ID from request context
func (am *AuditMiddleware) extractUserID(ctx context.Context) *uuid.UUID {
	authCtx, ok := ctx.Value("auth").(*AuthContext)
	if !ok {
		return nil
	}
	return &authCtx.UserID
}

// extractTenantID extracts tenant ID from request context
func (am *AuditMiddleware) extractTenantID(ctx context.Context) *uuid.UUID {
	authCtx, ok := ctx.Value("auth").(*AuthContext)
	if !ok {
		return nil
	}
	return &authCtx.TenantID
}

// extractClientIP extracts the real client IP from the request
func (am *AuditMiddleware) extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
