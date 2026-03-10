package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"go.uber.org/zap"
)

type imageImportLogStreamEntry struct {
	ImportRequestID string                 `json:"import_request_id"`
	Timestamp       string                 `json:"timestamp"`
	Level           string                 `json:"level"`
	Message         string                 `json:"message"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func (h *ImageImportHandler) StreamImportRequestLogs(w http.ResponseWriter, r *http.Request) {
	h.streamImportRequestLogs(w, r, false)
}

func (h *ImageImportHandler) StreamImportRequestLogsAdmin(w http.ResponseWriter, r *http.Request) {
	h.streamImportRequestLogs(w, r, true)
}

func (h *ImageImportHandler) streamImportRequestLogs(w http.ResponseWriter, r *http.Request, admin bool) {
	item, authCtx, err := h.resolveImportRequestForRead(r.Context(), r, admin)
	if err != nil {
		h.writeResolveImportError(w, err)
		return
	}
	if item == nil {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}
	if !admin && authCtx != nil && authCtx.TenantID != item.TenantID {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	logFilter, filterErr := parseBuildLogsFilter(r.URL.Query())
	if filterErr != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", filterErr.Error(), nil)
		return
	}

	conn, upgradeErr := upgrader.Upgrade(w, r, nil)
	if upgradeErr != nil {
		h.logger.Warn("Failed to upgrade import log stream websocket",
			zap.Error(upgradeErr),
			zap.String("import_id", item.ID.String()),
		)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				cancel()
				return
			}
		}
	}()

	seen := make(map[string]struct{})
	refreshItem := item
	sendDelta := func(current *imageimport.ImportRequest) error {
		if current == nil {
			return nil
		}
		entries := h.listImportLogs(ctx, current, logFilter)
		if len(entries) == 0 {
			return nil
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Timestamp < entries[j].Timestamp
		})
		for _, entry := range entries {
			key := entry.Timestamp + "|" + entry.Level + "|" + entry.Message
			if source, ok := entry.Metadata["source"]; ok && source != nil {
				key += "|" + fmt.Sprintf("%v", source)
			}
			if taskRun, ok := entry.Metadata["task_run"]; ok && taskRun != nil {
				key += "|" + fmt.Sprintf("%v", taskRun)
			}
			if step, ok := entry.Metadata["step"]; ok && step != nil {
				key += "|" + fmt.Sprintf("%v", step)
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			msg := imageImportLogStreamEntry{
				ImportRequestID: current.ID.String(),
				Timestamp:       entry.Timestamp,
				Level:           entry.Level,
				Message:         entry.Message,
				Metadata:        entry.Metadata,
			}
			if writeErr := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); writeErr != nil {
				return writeErr
			}
			if writeErr := conn.WriteJSON(msg); writeErr != nil {
				return writeErr
			}
		}
		return nil
	}

	if err := sendDelta(refreshItem); err != nil {
		return
	}

	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshed, getErr := h.service.GetImportRequest(ctx, refreshItem.TenantID, refreshItem.ID)
			if getErr != nil {
				if errors.Is(getErr, imageimport.ErrImportNotFound) {
					return
				}
				h.logger.Warn("Failed to refresh import request during stream",
					zap.Error(getErr),
					zap.String("import_id", refreshItem.ID.String()),
				)
				continue
			}
			refreshItem = refreshed
			if err := sendDelta(refreshItem); err != nil {
				return
			}
		}
	}
}

func (h *ImageImportHandler) listImportLogs(ctx context.Context, item *imageimport.ImportRequest, filter buildLogsFilter) []LogEntry {
	if item == nil {
		return nil
	}
	timeline := h.loadDecisionTimeline(ctx, item.ID)
	reconciliation := h.loadNotificationReconciliation(ctx, item, timeline)
	logs := h.buildImportLifecycleLogs(item, timeline, reconciliation)
	tektonLogs := h.fetchImportTektonLogs(ctx, item)
	if len(tektonLogs) > 0 {
		logs = append(logs, tektonLogs...)
	}
	return applyBuildLogsFilter(logs, filter)
}
