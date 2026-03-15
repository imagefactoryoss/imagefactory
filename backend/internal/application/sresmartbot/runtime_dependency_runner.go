package sresmartbot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"go.uber.org/zap"
)

type RuntimeDependencyWatcherConfig struct {
	Enabled                         bool
	Interval                        time.Duration
	AlertCooldown                   time.Duration
	CheckTimeout                    time.Duration
	InternalRegistryGCWorkerEnabled bool
	InternalRegistryGCWorkerURL     string
	InternalRegistryGCWorkerTimeout time.Duration
	DispatcherEnabled               bool
	WorkflowEnabled                 bool
}

type RuntimeDependencyNotificationEmitter func(ctx context.Context, title string, message string, notificationType string) (int, error)

type DispatcherRuntimeStatusReader interface {
	GetLatestRuntimeStatus(ctx context.Context, mode string) (*appdispatcher.RuntimeStatus, error)
}

func StartRuntimeDependencyWatcher(
	logger *zap.Logger,
	db *sqlx.DB,
	processHealthStore *runtimehealth.Store,
	sreSmartBotService *Service,
	dispatcherRuntimeStore DispatcherRuntimeStatusReader,
	monitorSubscriber interface{},
	buildNotificationEventSubscriber interface{},
	relay interface{},
	notify RuntimeDependencyNotificationEmitter,
	cfg RuntimeDependencyWatcherConfig,
) {
	if cfg.Interval < 15*time.Second {
		cfg.Interval = 60 * time.Second
	}
	if cfg.CheckTimeout < time.Second {
		cfg.CheckTimeout = 5 * time.Second
	}
	if cfg.InternalRegistryGCWorkerTimeout < time.Second {
		cfg.InternalRegistryGCWorkerTimeout = cfg.CheckTimeout
	}
	if cfg.AlertCooldown < 0 {
		cfg.AlertCooldown = 0
	}

	processHealthStore.Upsert("internal_registry_gc_worker", runtimehealth.ProcessStatus{
		Enabled:      cfg.InternalRegistryGCWorkerEnabled,
		Running:      false,
		LastActivity: time.Now().UTC(),
		Message:      "internal registry gc worker status pending",
		Metrics: map[string]int64{
			"last_run_candidates":      0,
			"last_run_deleted":         0,
			"last_run_errors":          0,
			"last_run_reclaimed_bytes": 0,
			"total_deleted":            0,
			"total_reclaimed_bytes":    0,
		},
	})

	processHealthStore.Upsert("runtime_dependency_watcher", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      cfg.Enabled,
		LastActivity: time.Now().UTC(),
		Message:      "runtime dependency watcher initialized",
		Metrics: map[string]int64{
			"runtime_dependency_checks_total":       0,
			"runtime_dependency_check_failures":     0,
			"runtime_dependency_degraded_count":     0,
			"runtime_dependency_critical_count":     0,
			"runtime_dependency_alerts_emitted":     0,
			"runtime_dependency_recoveries_emitted": 0,
		},
	})
	if !cfg.Enabled {
		processHealthStore.Upsert("runtime_dependency_watcher", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "runtime dependency watcher disabled",
			Metrics: map[string]int64{
				"runtime_dependency_checks_total":       0,
				"runtime_dependency_check_failures":     0,
				"runtime_dependency_degraded_count":     0,
				"runtime_dependency_critical_count":     0,
				"runtime_dependency_alerts_emitted":     0,
				"runtime_dependency_recoveries_emitted": 0,
			},
		})
		return
	}

	logger.Info("Background process starting",
		zap.String("component", "runtime_dependency_watcher"),
		zap.Duration("interval", cfg.Interval),
		zap.Duration("check_timeout", cfg.CheckTimeout),
		zap.Duration("alert_cooldown", cfg.AlertCooldown),
	)

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		var checksTotal int64
		var checkFailures int64
		var alertsEmitted int64
		var recoveriesEmitted int64
		var lastAlertSignature string
		var lastAlertAt time.Time
		lastObservedIssueKeys := make(map[string]RuntimeDependencyIssue)

		runTick := func() {
			checksTotal++
			now := time.Now().UTC()
			checkCtx, cancel := context.WithTimeout(context.Background(), cfg.CheckTimeout)
			defer cancel()

			issues := make([]RuntimeDependencyIssue, 0, 8)
			if pingErr := db.PingContext(checkCtx); pingErr != nil {
				issues = append(issues, RuntimeDependencyIssue{
					Key:      "database",
					Severity: "critical",
					Message:  pingErr.Error(),
				})
				checkFailures++
			}

			if !cfg.InternalRegistryGCWorkerEnabled {
				processHealthStore.Upsert("internal_registry_gc_worker", runtimehealth.ProcessStatus{
					Enabled:      false,
					Running:      false,
					LastActivity: now,
					Message:      "disabled",
					Metrics: map[string]int64{
						"last_run_candidates":      0,
						"last_run_deleted":         0,
						"last_run_errors":          0,
						"last_run_reclaimed_bytes": 0,
						"total_deleted":            0,
						"total_reclaimed_bytes":    0,
					},
				})
			} else {
				healthCtx, healthCancel := context.WithTimeout(context.Background(), cfg.InternalRegistryGCWorkerTimeout)
				req, reqErr := http.NewRequestWithContext(healthCtx, http.MethodGet, cfg.InternalRegistryGCWorkerURL, nil)
				if reqErr != nil {
					processHealthStore.Upsert("internal_registry_gc_worker", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      false,
						LastActivity: now,
						Message:      fmt.Sprintf("invalid health URL: %v", reqErr),
					})
					checkFailures++
					healthCancel()
				} else {
					resp, callErr := http.DefaultClient.Do(req)
					if callErr != nil {
						processHealthStore.Upsert("internal_registry_gc_worker", runtimehealth.ProcessStatus{
							Enabled:      true,
							Running:      false,
							LastActivity: now,
							Message:      fmt.Sprintf("health check failed: %v", callErr),
						})
						checkFailures++
						healthCancel()
					} else {
						var payload map[string]interface{}
						decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
						_ = resp.Body.Close()
						healthCancel()

						toInt64 := func(v interface{}) int64 {
							switch t := v.(type) {
							case float64:
								return int64(t)
							case int:
								return int64(t)
							case int64:
								return t
							case json.Number:
								n, _ := t.Int64()
								return n
							default:
								return 0
							}
						}

						gcMetrics := map[string]int64{
							"last_run_candidates":      toInt64(payload["last_run_candidates"]),
							"last_run_deleted":         toInt64(payload["last_run_deleted"]),
							"last_run_errors":          toInt64(payload["last_run_errors"]),
							"last_run_reclaimed_bytes": toInt64(payload["last_run_reclaimed_bytes"]),
							"total_deleted":            toInt64(payload["total_deleted"]),
							"total_reclaimed_bytes":    toInt64(payload["total_reclaimed_bytes"]),
						}

						gcMessage := "gc worker healthy"
						if decodeErr != nil {
							gcMessage = fmt.Sprintf("health response parse failed: %v", decodeErr)
						} else if message, ok := payload["message"].(string); ok && strings.TrimSpace(message) != "" {
							gcMessage = strings.TrimSpace(message)
						}

						gcRunning := resp.StatusCode >= 200 && resp.StatusCode < 300 && decodeErr == nil
						if !gcRunning {
							checkFailures++
							if resp.StatusCode < 200 || resp.StatusCode >= 300 {
								gcMessage = fmt.Sprintf("health check returned HTTP %d", resp.StatusCode)
							}
						}

						processHealthStore.Upsert("internal_registry_gc_worker", runtimehealth.ProcessStatus{
							Enabled:      true,
							Running:      gcRunning,
							LastActivity: now,
							Message:      gcMessage,
							Metrics:      gcMetrics,
						})
					}
				}
			}

			checkRuntimeProcess := func(name string, severity string, required bool) {
				if !required {
					return
				}
				status, ok := processHealthStore.GetStatus(name)
				if !ok {
					issues = append(issues, RuntimeDependencyIssue{Key: name, Severity: severity, Message: "status not reported"})
					checkFailures++
					return
				}
				if !status.Enabled {
					issues = append(issues, RuntimeDependencyIssue{Key: name, Severity: severity, Message: "component disabled"})
					checkFailures++
					return
				}
				if !status.Running {
					msg := strings.TrimSpace(status.Message)
					if msg == "" {
						msg = "component not running"
					}
					issues = append(issues, RuntimeDependencyIssue{Key: name, Severity: severity, Message: msg})
					checkFailures++
				}
			}

			dispatcherRequired := cfg.DispatcherEnabled
			if !dispatcherRequired && dispatcherRuntimeStore != nil {
				if externalStatus, err := dispatcherRuntimeStore.GetLatestRuntimeStatus(checkCtx, appdispatcher.DispatcherModeExternal); err != nil || externalStatus == nil || !externalStatus.Running || time.Since(externalStatus.LastHeartbeat) > 90*time.Second {
					dispatcherRequired = true
				}
			}

			checkRuntimeProcess("dispatcher", "critical", dispatcherRequired)
			checkRuntimeProcess("workflow_orchestrator", "critical", cfg.WorkflowEnabled)
			checkRuntimeProcess("messaging_outbox_relay", "critical", relay != nil)
			checkRuntimeProcess("build_monitor_event_subscriber", "degraded", monitorSubscriber != nil)
			checkRuntimeProcess("build_notification_event_subscriber", "degraded", buildNotificationEventSubscriber != nil)
			checkRuntimeProcess("internal_registry_gc_worker", "degraded", cfg.InternalRegistryGCWorkerEnabled)

			for _, name := range []string{"provider_readiness_watcher", "tenant_asset_drift_watcher", "quarantine_release_compliance_watcher"} {
				status, ok := processHealthStore.GetStatus(name)
				if !ok {
					issues = append(issues, RuntimeDependencyIssue{Key: name, Severity: "degraded", Message: "status not reported"})
					checkFailures++
					continue
				}
				if status.Enabled && !status.Running {
					msg := strings.TrimSpace(status.Message)
					if msg == "" {
						msg = "component not running"
					}
					issues = append(issues, RuntimeDependencyIssue{Key: name, Severity: "degraded", Message: msg})
					checkFailures++
				}
			}

			criticalCount := int64(0)
			degradedCount := int64(0)
			issueMessages := make([]string, 0, len(issues))
			for _, issue := range issues {
				if strings.EqualFold(issue.Severity, "critical") {
					criticalCount++
				} else {
					degradedCount++
				}
				issueMessages = append(issueMessages, fmt.Sprintf("%s (%s): %s", issue.Key, strings.ToUpper(issue.Severity), issue.Message))
			}
			lastObservedIssueKeys = ObserveRuntimeDependencyIssues(context.Background(), sreSmartBotService, logger, issues, now, lastObservedIssueKeys)

			signature := runtimeDependencyIssuesSignature(issues)
			if signature != "" {
				shouldNotify := false
				if signature != lastAlertSignature {
					shouldNotify = true
				} else if cfg.AlertCooldown > 0 && time.Since(lastAlertAt) >= cfg.AlertCooldown {
					shouldNotify = true
				}

				if shouldNotify && notify != nil {
					count, notifyErr := notify(
						context.Background(),
						"Runtime dependency alert",
						"Required runtime dependencies are degraded: "+strings.Join(issueMessages, " | "),
						"system_alert",
					)
					if notifyErr != nil {
						logger.Warn("Runtime dependency alert notification failed", zap.Error(notifyErr))
					} else {
						alertsEmitted++
						lastAlertSignature = signature
						lastAlertAt = now
						logger.Warn("Runtime dependency alert emitted",
							zap.Int("issues", len(issues)),
							zap.Int("notifications", count),
							zap.String("signature", signature),
						)
					}
				}
			} else if lastAlertSignature != "" && notify != nil {
				count, notifyErr := notify(
					context.Background(),
					"Runtime dependencies recovered",
					"Runtime dependency watcher reports all required services as healthy.",
					"system_alert",
				)
				if notifyErr != nil {
					logger.Warn("Runtime dependency recovery notification failed", zap.Error(notifyErr))
				} else {
					recoveriesEmitted++
					lastAlertSignature = ""
					lastAlertAt = now
					logger.Info("Runtime dependency recovery notification emitted",
						zap.Int("notifications", count),
					)
				}
			}

			message := "all dependencies healthy"
			if len(issueMessages) > 0 {
				message = "degraded dependencies: " + strings.Join(issueMessages, " | ")
			}
			processHealthStore.Upsert("runtime_dependency_watcher", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      message,
				Metrics: map[string]int64{
					"runtime_dependency_checks_total":       checksTotal,
					"runtime_dependency_check_failures":     checkFailures,
					"runtime_dependency_degraded_count":     degradedCount,
					"runtime_dependency_critical_count":     criticalCount,
					"runtime_dependency_alerts_emitted":     alertsEmitted,
					"runtime_dependency_recoveries_emitted": recoveriesEmitted,
				},
			})
		}

		runTick()
		for {
			<-ticker.C
			runTick()
		}
	}()
}

func runtimeDependencyIssuesSignature(issues []RuntimeDependencyIssue) string {
	if len(issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		key := strings.TrimSpace(strings.ToLower(issue.Key))
		severity := strings.TrimSpace(strings.ToLower(issue.Severity))
		if key == "" {
			continue
		}
		if severity == "" {
			severity = "degraded"
		}
		parts = append(parts, key+":"+severity)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}
