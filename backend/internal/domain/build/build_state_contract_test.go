package build

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func validManifestForStateTests() BuildManifest {
	return BuildManifest{
		Name:         "state-contract-test",
		Type:         BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo 'ok'"},
	}
}

func newPendingBuildForStateTests(t *testing.T) *Build {
	t.Helper()
	b, err := NewBuild(uuid.New(), uuid.New(), validManifestForStateTests(), nil)
	if err != nil {
		t.Fatalf("failed to create test build: %v", err)
	}
	return b
}

func TestBuildStateContract_QueueOnlyFromPending(t *testing.T) {
	b := newPendingBuildForStateTests(t)

	if err := b.Queue(); err != nil {
		t.Fatalf("queue from pending should succeed: %v", err)
	}
	if b.Status() != BuildStatusQueued {
		t.Fatalf("expected status queued, got %s", b.Status())
	}

	if err := b.Queue(); err == nil {
		t.Fatal("queue from queued should fail")
	}
}

func TestBuildStateContract_StartAllowedOnlyFromQueued(t *testing.T) {
	pending := newPendingBuildForStateTests(t)
	if err := pending.Start(); err == nil {
		t.Fatal("start from pending should fail")
	}

	queued := newPendingBuildForStateTests(t)
	if err := queued.Queue(); err != nil {
		t.Fatalf("queue failed: %v", err)
	}
	if err := queued.Start(); err != nil {
		t.Fatalf("start from queued should succeed: %v", err)
	}
	if queued.Status() != BuildStatusRunning {
		t.Fatalf("expected running from queued start, got %s", queued.Status())
	}
}

func TestBuildStateContract_CompleteOnlyFromRunning(t *testing.T) {
	b := newPendingBuildForStateTests(t)
	if err := b.Complete(BuildResult{}); err == nil {
		t.Fatal("complete from pending should fail")
	}

	if err := b.Queue(); err != nil {
		t.Fatalf("queue failed: %v", err)
	}
	if err := b.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if err := b.Complete(BuildResult{}); err != nil {
		t.Fatalf("complete from running should succeed: %v", err)
	}
	if b.Status() != BuildStatusCompleted {
		t.Fatalf("expected completed, got %s", b.Status())
	}
}

func TestBuildStateContract_CancelRejectedForCompletedOrFailed(t *testing.T) {
	completed := newPendingBuildForStateTests(t)
	_ = completed.Queue()
	_ = completed.Start()
	_ = completed.Complete(BuildResult{})
	if err := completed.Cancel(); !errors.Is(err, ErrCannotCancelBuild) {
		t.Fatalf("expected ErrCannotCancelBuild for completed build, got %v", err)
	}

	failed := newPendingBuildForStateTests(t)
	_ = failed.Fail("boom")
	if err := failed.Cancel(); !errors.Is(err, ErrCannotCancelBuild) {
		t.Fatalf("expected ErrCannotCancelBuild for failed build, got %v", err)
	}
}

func TestBuildStateContract_RetryStartOnlyFromFailedOrCancelled(t *testing.T) {
	running := newPendingBuildForStateTests(t)
	_ = running.Queue()
	_ = running.Start()
	if err := running.RetryStart(); err == nil {
		t.Fatal("retry from running should fail")
	}

	failed := newPendingBuildForStateTests(t)
	_ = failed.Queue()
	_ = failed.Start()
	_ = failed.Fail("failed-on-purpose")
	if failed.CompletedAt() == nil {
		t.Fatal("failed build should have completed_at before retry")
	}
	if err := failed.RetryStart(); err != nil {
		t.Fatalf("retry from failed should succeed: %v", err)
	}
	if failed.Status() != BuildStatusRunning {
		t.Fatalf("expected running after retry, got %s", failed.Status())
	}
	if failed.CompletedAt() != nil {
		t.Fatal("completed_at should be reset on retry")
	}
	if failed.ErrorMessage() != "" {
		t.Fatal("error message should be cleared on retry")
	}

	cancelled := newPendingBuildForStateTests(t)
	_ = cancelled.Cancel()
	if err := cancelled.RetryStart(); err != nil {
		t.Fatalf("retry from cancelled should succeed: %v", err)
	}
	if cancelled.Status() != BuildStatusRunning {
		t.Fatalf("expected running after cancelled retry, got %s", cancelled.Status())
	}
}
