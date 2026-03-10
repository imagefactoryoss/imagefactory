package worker

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewWorkerValidationAndDefaults(t *testing.T) {
	_, err := New(uuid.Nil, uuid.New(), "w1", WorkerTypeDocker, Capacity(1))
	if err == nil {
		t.Fatal("expected error for nil worker id")
	}
	_, err = New(uuid.New(), uuid.Nil, "w1", WorkerTypeDocker, Capacity(1))
	if err == nil {
		t.Fatal("expected error for nil tenant id")
	}
	_, err = New(uuid.New(), uuid.New(), "", WorkerTypeDocker, Capacity(1))
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	_, err = New(uuid.New(), uuid.New(), "w1", WorkerTypeDocker, Capacity(0))
	if err == nil {
		t.Fatal("expected error for invalid capacity")
	}

	w, err := New(uuid.New(), uuid.New(), "w1", WorkerTypeDocker, Capacity(2))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if w.Status() != StatusAvailable || w.CurrentLoad() != 0 || !w.IsAvailable() {
		t.Fatalf("unexpected default worker state: %+v", w)
	}
	if len(w.UncommittedEvents()) == 0 {
		t.Fatal("expected WorkerRegistered event")
	}
}

func TestWorkerLoadLifecycle(t *testing.T) {
	w, _ := New(uuid.New(), uuid.New(), "w1", WorkerTypeDocker, Capacity(2))

	if err := w.IncrementLoad(0); err == nil {
		t.Fatal("expected error for non-positive delta")
	}
	if err := w.IncrementLoad(3); err == nil {
		t.Fatal("expected capacity overflow error")
	}

	if err := w.IncrementLoad(2); err != nil {
		t.Fatalf("expected increment success, got %v", err)
	}
	if w.Status() != StatusBusy || w.CurrentLoad() != 2 {
		t.Fatalf("expected busy/full load, got status=%s load=%d", w.Status(), w.CurrentLoad())
	}

	if err := w.DecrementLoad(1); err != nil {
		t.Fatalf("expected decrement success, got %v", err)
	}
	if w.Status() != StatusAvailable || w.CurrentLoad() != 1 {
		t.Fatalf("expected available after decrement, got status=%s load=%d", w.Status(), w.CurrentLoad())
	}

	if err := w.DecrementLoad(5); err != nil {
		t.Fatalf("expected decrement clamp success, got %v", err)
	}
	if w.CurrentLoad() != 0 {
		t.Fatalf("expected load clamped at 0, got %d", w.CurrentLoad())
	}
}

func TestWorkerHeartbeatAndFailure(t *testing.T) {
	w, _ := New(uuid.New(), uuid.New(), "w1", WorkerTypeDocker, Capacity(1))

	if err := w.RecordHeartbeat(); err != nil {
		t.Fatalf("expected heartbeat success, got %v", err)
	}
	if w.LastHeartbeat() == nil || w.ConsecutiveFailures() != 0 {
		t.Fatalf("unexpected heartbeat state: hb=%v failures=%d", w.LastHeartbeat(), w.ConsecutiveFailures())
	}

	for i := 0; i < 5; i++ {
		if err := w.RecordFailure(); err != nil {
			t.Fatalf("expected failure record success, got %v", err)
		}
	}
	if w.Status() != StatusOffline {
		t.Fatalf("expected offline after repeated failures, got %s", w.Status())
	}
	if err := w.RecordHeartbeat(); err == nil {
		t.Fatal("expected heartbeat error while offline")
	}
}

func TestWorkerOnlineOfflineAndEvents(t *testing.T) {
	w, _ := New(uuid.New(), uuid.New(), "w1", WorkerTypeDocker, Capacity(1))
	initialEventCount := len(w.UncommittedEvents())

	if err := w.GoOffline("maintenance"); err != nil {
		t.Fatalf("expected go offline success, got %v", err)
	}
	if w.Status() != StatusOffline {
		t.Fatalf("expected offline status, got %s", w.Status())
	}
	if err := w.GoOnline(); err != nil {
		t.Fatalf("expected go online success, got %v", err)
	}
	if w.Status() != StatusAvailable {
		t.Fatalf("expected available status, got %s", w.Status())
	}
	if len(w.UncommittedEvents()) <= initialEventCount {
		t.Fatal("expected additional events after online/offline transitions")
	}

	w.MarkEventsAsCommitted()
	if len(w.UncommittedEvents()) != 0 {
		t.Fatal("expected uncommitted events to be cleared")
	}
}

func TestStatusAndWorkerTypeValidity(t *testing.T) {
	if !StatusAvailable.IsValid() || !StatusBusy.IsValid() || !StatusOffline.IsValid() {
		t.Fatal("expected built-in statuses to be valid")
	}
	if Status("x").IsValid() {
		t.Fatal("expected unknown status to be invalid")
	}

	if !WorkerTypeDocker.IsValid() || !WorkerTypeKubernetes.IsValid() || !WorkerTypeLambda.IsValid() {
		t.Fatal("expected built-in worker types to be valid")
	}
	if WorkerType("x").IsValid() {
		t.Fatal("expected unknown worker type invalid")
	}
	if !Capacity(1).IsValid() || Capacity(0).IsValid() {
		t.Fatal("capacity validity mismatch")
	}
}

func TestWorkerTimestampsAdvance(t *testing.T) {
	w, _ := New(uuid.New(), uuid.New(), "w1", WorkerTypeDocker, Capacity(2))
	before := w.UpdatedAt()
	time.Sleep(1 * time.Millisecond)
	_ = w.IncrementLoad(1)
	if !w.UpdatedAt().After(before) {
		t.Fatal("expected updated_at to advance")
	}
}
