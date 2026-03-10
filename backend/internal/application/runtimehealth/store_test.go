package runtimehealth

import (
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	if got := normalize("  Email-Worker  "); got != "email-worker" {
		t.Fatalf("expected normalized name, got %q", got)
	}
	if got := normalize("   "); got != "" {
		t.Fatalf("expected empty normalized name, got %q", got)
	}
}

func TestStoreUpsertAndGetStatus(t *testing.T) {
	s := NewStore()
	st := ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: time.Now().UTC(),
		Message:      "ok",
	}

	s.Upsert(" Worker-A ", st)
	got, ok := s.GetStatus("worker-a")
	if !ok {
		t.Fatal("expected status to exist")
	}
	if !got.Enabled || !got.Running || got.Message != "ok" {
		t.Fatalf("unexpected status: %+v", got)
	}
}

func TestStoreTouch(t *testing.T) {
	s := NewStore()
	s.Touch("worker-b")

	got, ok := s.GetStatus("worker-b")
	if !ok {
		t.Fatal("expected status to exist after touch")
	}
	if got.LastActivity.IsZero() {
		t.Fatal("expected LastActivity to be populated")
	}
}

func TestStoreIgnoresEmptyNames(t *testing.T) {
	s := NewStore()
	s.Upsert("   ", ProcessStatus{Enabled: true})
	s.Touch("   ")

	if _, ok := s.GetStatus("   "); ok {
		t.Fatal("expected empty-name status to be ignored")
	}
}
