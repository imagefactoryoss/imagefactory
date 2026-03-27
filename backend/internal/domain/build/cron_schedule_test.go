package build

import (
	"testing"
	"time"
)

func TestNextTriggerTimeFromCron_DailyMidnightUTC(t *testing.T) {
	from := time.Date(2026, 3, 27, 23, 40, 0, 0, time.UTC)
	next, err := nextTriggerTimeFromCron("0 0 * * *", "UTC", from)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if next == nil {
		t.Fatal("expected next trigger time")
	}
	expected := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.Format(time.RFC3339), next.Format(time.RFC3339))
	}
}

func TestNextTriggerTimeFromCron_EveryFiveMinutes(t *testing.T) {
	from := time.Date(2026, 3, 27, 10, 3, 0, 0, time.UTC)
	next, err := nextTriggerTimeFromCron("*/5 * * * *", "UTC", from)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := time.Date(2026, 3, 27, 10, 5, 0, 0, time.UTC)
	if next == nil || !next.Equal(expected) {
		t.Fatalf("expected %s, got %v", expected.Format(time.RFC3339), next)
	}
}

func TestNextTriggerTimeFromCron_InvalidExpr(t *testing.T) {
	if _, err := nextTriggerTimeFromCron("* * *", "UTC", time.Now().UTC()); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}
