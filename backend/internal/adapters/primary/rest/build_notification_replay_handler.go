package rest

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type BuildNotificationReplayHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type replayFailedNotificationsRequest struct {
	Limit           int  `json:"limit"`
	IncludeTerminal bool `json:"include_terminal"`
}

func NewBuildNotificationReplayHandler(db *sqlx.DB, logger *zap.Logger) *BuildNotificationReplayHandler {
	return &BuildNotificationReplayHandler{db: db, logger: logger}
}

func (h *BuildNotificationReplayHandler) GetTenantBuildNotificationReplayStatus(w http.ResponseWriter, r *http.Request) {
	_, tenantID, ok := h.resolveTenantContext(w, r)
	if !ok {
		return
	}

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_count,
			COALESCE(SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END), 0) AS processing_count,
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0) AS sent_count,
			COALESCE(SUM(CASE WHEN status = 'failed' AND retry_count < max_retries THEN 1 ELSE 0 END), 0) AS failed_retryable_count,
			COALESCE(SUM(CASE WHEN status = 'failed' AND retry_count >= max_retries THEN 1 ELSE 0 END), 0) AS failed_terminal_count,
			MIN(CASE WHEN status = 'failed' THEN updated_at END) AS oldest_failed_at
		FROM email_queue
		WHERE tenant_id = $1
		  AND email_type = 'notification'
		  AND (COALESCE(metadata->>'notification_type', '') LIKE 'build_%'
		       OR COALESCE(metadata->>'notification_type', '') = 'preflight_blocked')`

	var out struct {
		PendingCount         int64      `db:"pending_count"`
		ProcessingCount      int64      `db:"processing_count"`
		SentCount            int64      `db:"sent_count"`
		FailedRetryableCount int64      `db:"failed_retryable_count"`
		FailedTerminalCount  int64      `db:"failed_terminal_count"`
		OldestFailedAt       *time.Time `db:"oldest_failed_at"`
	}
	if err := h.db.GetContext(r.Context(), &out, query, tenantID); err != nil {
		h.logger.Error("Failed to load build notification replay status",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to load replay status")
		return
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id": tenantID,
		"status": map[string]int64{
			"pending":          out.PendingCount,
			"processing":       out.ProcessingCount,
			"sent":             out.SentCount,
			"failed_retryable": out.FailedRetryableCount,
			"failed_terminal":  out.FailedTerminalCount,
		},
		"oldest_failed_at": func() interface{} {
			if out.OldestFailedAt == nil || out.OldestFailedAt.IsZero() {
				return nil
			}
			return out.OldestFailedAt.UTC().Format(time.RFC3339)
		}(),
	})
}

func (h *BuildNotificationReplayHandler) ReplayTenantBuildNotificationFailures(w http.ResponseWriter, r *http.Request) {
	_, tenantID, ok := h.resolveTenantContext(w, r)
	if !ok {
		return
	}

	req := replayFailedNotificationsRequest{
		Limit:           200,
		IncludeTerminal: false,
	}
	if r.Body != nil {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Limit <= 0 {
		req.Limit = 200
	}
	if req.Limit > 2000 {
		req.Limit = 2000
	}

	query := `
		WITH selected AS (
			SELECT id
			FROM email_queue
			WHERE tenant_id = $1
			  AND email_type = 'notification'
			  AND status = 'failed'
			  AND (retry_count < max_retries OR $3::boolean)
			  AND (COALESCE(metadata->>'notification_type', '') LIKE 'build_%'
			       OR COALESCE(metadata->>'notification_type', '') = 'preflight_blocked')
			ORDER BY updated_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE email_queue e
		SET status = 'pending',
			next_retry_at = NULL,
			last_error = NULL,
			retry_count = CASE
				WHEN e.retry_count >= e.max_retries THEN GREATEST(e.max_retries - 1, 0)
				ELSE e.retry_count
			END,
			updated_at = NOW()
		FROM selected s
		WHERE e.id = s.id`

	result, err := h.db.ExecContext(r.Context(), query, tenantID, req.Limit, req.IncludeTerminal)
	if err != nil {
		h.logger.Error("Failed to replay failed build notification emails",
			zap.String("tenant_id", tenantID.String()),
			zap.Int("limit", req.Limit),
			zap.Bool("include_terminal", req.IncludeTerminal),
			zap.Error(err))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to replay notifications")
		return
	}
	rows, _ := result.RowsAffected()

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id":         tenantID,
		"replayed_count":    rows,
		"limit":             req.Limit,
		"include_terminal":  req.IncludeTerminal,
		"replay_started_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *BuildNotificationReplayHandler) resolveTenantContext(w http.ResponseWriter, r *http.Request) (*middleware.AuthContext, uuid.UUID, bool) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "authentication required")
		return nil, uuid.Nil, false
	}

	tenantIDStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(strings.TrimSpace(tenantIDStr))
	if err != nil || tenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusBadRequest, "invalid tenant id")
		return nil, uuid.Nil, false
	}

	if authCtx.IsSystemAdmin || authCtx.TenantID == tenantID || authCtx.HasTenant(tenantID) {
		return authCtx, tenantID, true
	}

	writeNotificationCenterError(w, http.StatusForbidden, "forbidden")
	return nil, uuid.Nil, false
}
