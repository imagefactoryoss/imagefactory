package build

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type statusEventPublisher struct {
	statuses []*BuildStatusUpdated
}

func (p *statusEventPublisher) PublishBuildCreated(ctx context.Context, event *BuildCreated) error {
	return nil
}

func (p *statusEventPublisher) PublishBuildStarted(ctx context.Context, event *BuildStarted) error {
	return nil
}

func (p *statusEventPublisher) PublishBuildCompleted(ctx context.Context, event *BuildCompleted) error {
	return nil
}

func (p *statusEventPublisher) PublishBuildStatusUpdated(ctx context.Context, event *BuildStatusUpdated) error {
	p.statuses = append(p.statuses, event)
	return nil
}

type preflightFailExecutor struct{}

func (e *preflightFailExecutor) Execute(ctx context.Context, build *Build) (*BuildResult, error) {
	return nil, errors.New("tekton preflight failed: missing tekton task")
}

func (e *preflightFailExecutor) Cancel(ctx context.Context, buildID uuid.UUID) error {
	return nil
}

type inProgressExecutor struct{}

func (e *inProgressExecutor) Execute(ctx context.Context, build *Build) (*BuildResult, error) {
	return nil, ErrBuildExecutionInProgress
}

func (e *inProgressExecutor) Cancel(ctx context.Context, buildID uuid.UUID) error {
	return nil
}

func TestServiceExecuteBuild_PublishesPreflightBlockedStatus(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
	events := &statusEventPublisher{}
	executor := &preflightFailExecutor{}

	b, err := NewBuild(uuid.New(), uuid.New(), validManifestForServiceLifecycleTests(), nil)
	if err != nil {
		t.Fatalf("failed to create build aggregate: %v", err)
	}
	_ = b.Queue()
	_ = b.Start()
	repo.byID[b.ID()] = b

	svc := NewService(
		repo,
		&stubTriggerRepo{},
		events,
		executor,
		nil,
		nil,
		nil,
		nil,
		&stubSystemConfigService{},
		nil,
		zap.NewNop(),
	)

	svc.executeBuild(context.Background(), b, false)

	if len(events.statuses) == 0 {
		t.Fatal("expected at least one status update event")
	}
	got := events.statuses[len(events.statuses)-1]
	if got.Status() != "preflight_blocked" {
		t.Fatalf("expected status preflight_blocked, got %q", got.Status())
	}
	if got.Message() != "Build preflight blocked execution" {
		t.Fatalf("expected preflight blocked message, got %q", got.Message())
	}
}

func TestServiceRetryBuild_PublishesRetryStartedStatus(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
	events := &statusEventPublisher{}
	executor := &inProgressExecutor{}

	b, err := NewBuild(uuid.New(), uuid.New(), validManifestForServiceLifecycleTests(), nil)
	if err != nil {
		t.Fatalf("failed to create build aggregate: %v", err)
	}
	_ = b.Queue()
	_ = b.Start()
	_ = b.Fail("failed-on-purpose")
	repo.byID[b.ID()] = b

	svc := NewService(
		repo,
		&stubTriggerRepo{},
		events,
		executor,
		nil,
		nil,
		nil,
		nil,
		&stubSystemConfigService{},
		nil,
		zap.NewNop(),
	)

	if err := svc.RetryBuild(context.Background(), b.ID()); err != nil {
		t.Fatalf("retry build failed: %v", err)
	}

	// Retry dispatch is async; allow a brief window for goroutine scheduling.
	time.Sleep(10 * time.Millisecond)

	if len(events.statuses) == 0 {
		t.Fatal("expected retry_started status update event")
	}
	got := events.statuses[0]
	if got.Status() != "retry_started" {
		t.Fatalf("expected status retry_started, got %q", got.Status())
	}
	if got.Message() != "Build retry started" {
		t.Fatalf("expected retry started message, got %q", got.Message())
	}
}
