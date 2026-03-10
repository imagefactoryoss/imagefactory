package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type NotificationCenterHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
	wsHub  *WebSocketHub
}

type notificationCenterItem struct {
	ID                  uuid.UUID  `db:"id" json:"id"`
	Title               *string    `db:"title" json:"title,omitempty"`
	Message             *string    `db:"message" json:"message,omitempty"`
	NotificationType    *string    `db:"notification_type" json:"notification_type,omitempty"`
	RelatedResourceType *string    `db:"related_resource_type" json:"related_resource_type,omitempty"`
	RelatedResourceID   *uuid.UUID `db:"related_resource_id" json:"related_resource_id,omitempty"`
	IsRead              bool       `db:"is_read" json:"is_read"`
	ReadAt              *string    `db:"read_at" json:"read_at,omitempty"`
	CreatedAt           string     `db:"created_at" json:"created_at"`
}

type deleteNotificationsBulkRequest struct {
	IDs []string `json:"ids"`
}

func NewNotificationCenterHandler(db *sqlx.DB, logger *zap.Logger, wsHub *WebSocketHub) *NotificationCenterHandler {
	return &NotificationCenterHandler{db: db, logger: logger, wsHub: wsHub}
}

func (h *NotificationCenterHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := parseQueryInt(r, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	offset := parseQueryInt(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	unreadOnly := parseQueryBool(r, "unread", false)

	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1
		  AND tenant_id = $2
		  AND channel = 'in_app'`
	countArgs := []interface{}{authCtx.UserID, authCtx.TenantID}
	if unreadOnly {
		countQuery += ` AND is_read = false`
	}
	if err := h.db.QueryRowxContext(r.Context(), countQuery, countArgs...).Scan(&total); err != nil {
		h.logger.Error("Failed to count notifications", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	query := `
		SELECT
			id,
			title,
			message,
			notification_type,
			related_resource_type,
			related_resource_id,
			is_read,
			CASE WHEN read_at IS NULL THEN NULL ELSE to_char(read_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') END AS read_at,
			to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM notifications
		WHERE user_id = $1
		  AND tenant_id = $2
		  AND channel = 'in_app'`
	if unreadOnly {
		query += ` AND is_read = false`
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`

	items := make([]notificationCenterItem, 0, limit)
	if err := h.db.SelectContext(r.Context(), &items, query, authCtx.UserID, authCtx.TenantID, limit, offset); err != nil {
		h.logger.Error("Failed to query notifications", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": items,
		"total_count":   total,
		"limit":         limit,
		"offset":        offset,
	})
}

func (h *NotificationCenterHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var unread int
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1
		  AND tenant_id = $2
		  AND channel = 'in_app'
		  AND is_read = false`
	if err := h.db.QueryRowxContext(r.Context(), query, authCtx.UserID, authCtx.TenantID).Scan(&unread); err != nil {
		h.logger.Error("Failed to count unread notifications", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to load unread count")
		return
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{"unread_count": unread})
}

func (h *NotificationCenterHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notificationID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil || notificationID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	query := `
		UPDATE notifications
		SET is_read = true,
		    read_at = COALESCE(read_at, NOW())
		WHERE id = $1
		  AND user_id = $2
		  AND tenant_id = $3
		  AND channel = 'in_app'`
	result, execErr := h.db.ExecContext(r.Context(), query, notificationID, authCtx.UserID, authCtx.TenantID)
	if execErr != nil {
		h.logger.Error("Failed to mark notification as read", zap.Error(execErr), zap.String("notification_id", notificationID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to update notification")
		return
	}
	rows, rowsErr := result.RowsAffected()
	if rowsErr == nil && rows == 0 {
		writeNotificationCenterError(w, http.StatusNotFound, "notification not found")
		return
	}
	if h.wsHub != nil {
		h.wsHub.BroadcastNotificationEvent(
			authCtx.TenantID,
			authCtx.UserID,
			"notification.read",
			&notificationID,
			map[string]interface{}{"is_read": true},
		)
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *NotificationCenterHandler) DeleteOne(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notificationID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil || notificationID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	query := `
		DELETE FROM notifications
		WHERE id = $1
		  AND user_id = $2
		  AND tenant_id = $3
		  AND channel = 'in_app'`
	result, execErr := h.db.ExecContext(r.Context(), query, notificationID, authCtx.UserID, authCtx.TenantID)
	if execErr != nil {
		h.logger.Error("Failed to delete notification", zap.Error(execErr), zap.String("notification_id", notificationID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to delete notification")
		return
	}
	rows, rowsErr := result.RowsAffected()
	if rowsErr == nil && rows == 0 {
		writeNotificationCenterError(w, http.StatusNotFound, "notification not found")
		return
	}

	if h.wsHub != nil {
		h.wsHub.BroadcastNotificationEvent(
			authCtx.TenantID,
			authCtx.UserID,
			"notification.deleted",
			&notificationID,
			map[string]interface{}{"deleted": true},
		)
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *NotificationCenterHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := `
		UPDATE notifications
		SET is_read = true,
		    read_at = COALESCE(read_at, NOW())
		WHERE user_id = $1
		  AND tenant_id = $2
		  AND channel = 'in_app'
		  AND is_read = false`
	result, err := h.db.ExecContext(r.Context(), query, authCtx.UserID, authCtx.TenantID)
	if err != nil {
		h.logger.Error("Failed to mark all notifications as read", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to update notifications")
		return
	}
	rows, _ := result.RowsAffected()
	if h.wsHub != nil && rows > 0 {
		h.wsHub.BroadcastNotificationEvent(
			authCtx.TenantID,
			authCtx.UserID,
			"notification.read_all",
			nil,
			map[string]interface{}{"updated": rows},
		)
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"updated": rows,
	})
}

func (h *NotificationCenterHandler) DeleteRead(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := `
		DELETE FROM notifications
		WHERE user_id = $1
		  AND tenant_id = $2
		  AND channel = 'in_app'
		  AND is_read = true`
	result, err := h.db.ExecContext(r.Context(), query, authCtx.UserID, authCtx.TenantID)
	if err != nil {
		h.logger.Error("Failed to delete read notifications", zap.Error(err), zap.String("user_id", authCtx.UserID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to delete notifications")
		return
	}
	rows, _ := result.RowsAffected()
	if h.wsHub != nil && rows > 0 {
		h.wsHub.BroadcastNotificationEvent(
			authCtx.TenantID,
			authCtx.UserID,
			"notification.deleted_read",
			nil,
			map[string]interface{}{"deleted": rows},
		)
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"deleted": rows,
	})
}

func (h *NotificationCenterHandler) DeleteBulk(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil || authCtx.TenantID == uuid.Nil {
		writeNotificationCenterError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload deleteNotificationsBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeNotificationCenterError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trimmed := make([]uuid.UUID, 0, len(payload.IDs))
	seen := map[uuid.UUID]struct{}{}
	for _, rawID := range payload.IDs {
		notificationID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil || notificationID == uuid.Nil {
			writeNotificationCenterError(w, http.StatusBadRequest, "invalid notification id")
			return
		}
		if _, exists := seen[notificationID]; exists {
			continue
		}
		seen[notificationID] = struct{}{}
		trimmed = append(trimmed, notificationID)
	}

	if len(trimmed) == 0 {
		writeNotificationCenterError(w, http.StatusBadRequest, "at least one notification id is required")
		return
	}
	if len(trimmed) > 500 {
		writeNotificationCenterError(w, http.StatusBadRequest, "too many notification ids")
		return
	}

	baseQuery := `
		DELETE FROM notifications
		WHERE user_id = ?
		  AND tenant_id = ?
		  AND channel = 'in_app'
		  AND id IN (?)`
	query, args, err := sqlx.In(baseQuery, authCtx.UserID, authCtx.TenantID, trimmed)
	if err != nil {
		h.logger.Error("Failed to build bulk delete notifications query", zap.Error(err))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to delete notifications")
		return
	}
	query = h.db.Rebind(query)

	result, execErr := h.db.ExecContext(r.Context(), query, args...)
	if execErr != nil {
		h.logger.Error("Failed to delete notifications in bulk", zap.Error(execErr), zap.String("user_id", authCtx.UserID.String()))
		writeNotificationCenterError(w, http.StatusInternalServerError, "failed to delete notifications")
		return
	}
	rows, _ := result.RowsAffected()

	if h.wsHub != nil && rows > 0 {
		h.wsHub.BroadcastNotificationEvent(
			authCtx.TenantID,
			authCtx.UserID,
			"notification.deleted_bulk",
			nil,
			map[string]interface{}{
				"deleted":   rows,
				"requested": len(trimmed),
			},
		)
	}

	writeNotificationCenterJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"deleted":   rows,
		"requested": len(trimmed),
	})
}

func writeNotificationCenterJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeNotificationCenterError(w http.ResponseWriter, status int, message string) {
	writeNotificationCenterJSON(w, status, map[string]string{"error": message})
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseQueryBool(r *http.Request, key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	if raw == "" {
		return fallback
	}
	return raw == "1" || raw == "true" || raw == "yes"
}
