package middleware

import (
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/tenant"
)

// TenantStatusMiddleware enforces tenant status access control policies
type TenantStatusMiddleware struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTenantStatusMiddleware creates a new tenant status middleware
func NewTenantStatusMiddleware(db *sqlx.DB, logger *zap.Logger) *TenantStatusMiddleware {
	return &TenantStatusMiddleware{
		db:     db,
		logger: logger,
	}
}

// EnforceTenantStatus wraps request handling to enforce tenant status access control
// This middleware checks if the tenant is suspended or deleted, and if so:
// - Allows READ operations (GET, OPTIONS, HEAD)
// - Denies WRITE operations (POST, PUT, PATCH, DELETE)
func (m *TenantStatusMiddleware) EnforceTenantStatus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get auth context
		authCtx, ok := r.Context().Value("auth").(*AuthContext)
		if !ok {
			// No auth context, let other middleware handle it
			next.ServeHTTP(w, r)
			return
		}

		// System admins (TenantID == nil) bypass tenant status checks
		if authCtx.IsSystemAdmin {
			next.ServeHTTP(w, r)
			return
		}

		// Check if this is a read-only operation
		isReadOnly := isReadOnlyMethod(r.Method)

		// Query tenant status
		tenantRepo := postgres.NewTenantRepository(m.db, m.logger)
		t, err := tenantRepo.FindByID(r.Context(), authCtx.TenantID)
		if err != nil {
			m.logger.Error("Failed to fetch tenant status",
				zap.Error(err),
				zap.String("tenant_id", authCtx.TenantID.String()),
				zap.String("user_id", authCtx.UserID.String()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if t == nil {
			m.logger.Warn("Tenant not found",
				zap.String("tenant_id", authCtx.TenantID.String()),
				zap.String("user_id", authCtx.UserID.String()))
			http.Error(w, "Tenant Not Found", http.StatusNotFound)
			return
		}

		// Check tenant status
		if t.Status() == tenant.TenantStatusSuspended || t.Status() == tenant.TenantStatusDeleted {
			// For suspended/deleted tenants, only allow read operations
			if !isReadOnly {
				m.logger.Warn("Write operation denied for suspended/deleted tenant",
					zap.String("tenant_id", authCtx.TenantID.String()),
					zap.String("user_id", authCtx.UserID.String()),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("tenant_status", string(t.Status())))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"Tenant is suspended or deleted. Only read-only operations are allowed."}`))
				return
			}

			m.logger.Debug("Read-only operation allowed for suspended/deleted tenant",
				zap.String("tenant_id", authCtx.TenantID.String()),
				zap.String("user_id", authCtx.UserID.String()),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path))
		}

		// Continue with next handler
		next.ServeHTTP(w, r)
	})
}

// isReadOnlyMethod returns true if the HTTP method is read-only
func isReadOnlyMethod(method string) bool {
	readOnlyMethods := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodOptions: true,
	}
	return readOnlyMethods[strings.ToUpper(method)]
}

// GetWriteRestrictedMethods returns a list of HTTP methods that trigger write restrictions
func GetWriteRestrictedMethods() []string {
	return []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}
}
