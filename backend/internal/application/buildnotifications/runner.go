package buildnotifications

import (
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"go.uber.org/zap"
)

const processHealthKey = "build_notification_event_subscriber"

func StartHealthReporter(logger *zap.Logger, processHealthStore *runtimehealth.Store, subscriber *EventSubscriber) {
	if processHealthStore == nil {
		return
	}
	if subscriber == nil {
		processHealthStore.Upsert(processHealthKey, runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "build notification event subscriber disabled",
		})
		return
	}

	processHealthStore.Upsert(processHealthKey, runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: time.Now().UTC(),
		Message:      "build notification event subscriber started",
		Metrics:      zeroSnapshotMetrics(),
	})

	if logger != nil {
		logger.Info("Background process starting",
			zap.String("component", processHealthKey),
			zap.Duration("interval", 15*time.Second),
		)
	}

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			snapshot := subscriber.Snapshot()
			processHealthStore.Upsert(processHealthKey, runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: time.Now().UTC(),
				Message: fmt.Sprintf(
					"events=%d mapped=%d in_app=%d email=%d failures=%d",
					snapshot.EventsReceived,
					snapshot.MappedEvents,
					snapshot.InAppDelivered,
					snapshot.EmailQueued,
					snapshot.Failures,
				),
				Metrics: map[string]int64{
					"build_notification_events_received_total":  snapshot.EventsReceived,
					"build_notification_mapped_events_total":    snapshot.MappedEvents,
					"build_notification_in_app_delivered_total": snapshot.InAppDelivered,
					"build_notification_email_queued_total":     snapshot.EmailQueued,
					"build_notification_failures_total":         snapshot.Failures,
				},
			})
			<-ticker.C
		}
	}()
}

func zeroSnapshotMetrics() map[string]int64 {
	return map[string]int64{
		"build_notification_events_received_total":  0,
		"build_notification_mapped_events_total":    0,
		"build_notification_in_app_delivered_total": 0,
		"build_notification_email_queued_total":     0,
		"build_notification_failures_total":         0,
	}
}
