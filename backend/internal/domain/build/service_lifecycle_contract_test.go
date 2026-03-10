package build

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type lifecycleBuildRepo struct {
	stubBuildRepo
	byID    map[uuid.UUID]*Build
	updated []*Build
}

func (r *lifecycleBuildRepo) Save(ctx context.Context, b *Build) error {
	r.lastSaved = b
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*Build)
	}
	r.byID[b.ID()] = b
	return nil
}

func (r *lifecycleBuildRepo) FindByID(ctx context.Context, id uuid.UUID) (*Build, error) {
	if r.byID == nil {
		return nil, nil
	}
	return r.byID[id], nil
}

func (r *lifecycleBuildRepo) Update(ctx context.Context, b *Build) error {
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*Build)
	}
	r.byID[b.ID()] = b
	r.updated = append(r.updated, b)
	return nil
}

func newLifecycleServiceForContractTests(repo Repository) *Service {
	return NewService(
		repo,
		&stubTriggerRepo{},
		&stubEventPublisher{},
		nil,
		nil,
		nil,
		nil,
		nil,
		&stubSystemConfigService{},
		nil,
		zap.NewNop(),
	)
}

func validManifestForServiceLifecycleTests() BuildManifest {
	return BuildManifest{
		Name:         "service-contract-test",
		Type:         BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}
}

func TestServiceLifecycleContract_CreateBuildQueuesByDefault(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
	svc := newLifecycleServiceForContractTests(repo)

	created, err := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), validManifestForServiceLifecycleTests(), nil)
	if err != nil {
		t.Fatalf("create build failed: %v", err)
	}
	if created.Status() != BuildStatusQueued {
		t.Fatalf("expected queued status after create, got %s", created.Status())
	}
	if repo.lastSaved == nil {
		t.Fatal("expected repository Save to be called")
	}
	if repo.lastSaved.ID() != created.ID() {
		t.Fatalf("expected saved build id %s, got %s", created.ID(), repo.lastSaved.ID())
	}
}

func TestServiceLifecycleContract_RetryBuildTransitionsFailedToRunning(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
	svc := newLifecycleServiceForContractTests(repo)

	b, err := NewBuild(uuid.New(), uuid.New(), validManifestForServiceLifecycleTests(), nil)
	if err != nil {
		t.Fatalf("failed to create build aggregate: %v", err)
	}
	_ = b.Queue()
	_ = b.Start()
	_ = b.Fail("failed-on-purpose")
	repo.byID[b.ID()] = b

	if err := svc.RetryBuild(context.Background(), b.ID()); err != nil {
		t.Fatalf("retry build failed: %v", err)
	}

	if len(repo.updated) == 0 {
		t.Fatal("expected repository Update to be called during retry")
	}
	if repo.byID[b.ID()].Status() != BuildStatusRunning {
		t.Fatalf("expected running status after retry, got %s", repo.byID[b.ID()].Status())
	}

	// Give async dispatch goroutine a tiny window; no executor is configured so no extra state changes expected.
	time.Sleep(10 * time.Millisecond)
	if repo.byID[b.ID()].Status() != BuildStatusRunning {
		t.Fatalf("expected build to remain running without executor, got %s", repo.byID[b.ID()].Status())
	}
}

func TestServiceLifecycleContract_RetryBuildRejectsPending(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
	svc := newLifecycleServiceForContractTests(repo)

	b, err := NewBuild(uuid.New(), uuid.New(), validManifestForServiceLifecycleTests(), nil)
	if err != nil {
		t.Fatalf("failed to create build aggregate: %v", err)
	}
	repo.byID[b.ID()] = b

	if err := svc.RetryBuild(context.Background(), b.ID()); err == nil {
		t.Fatal("expected retry from pending state to fail")
	}
}

func TestServiceLifecycleContract_StartBuildIsIdempotentWhenRunning(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
	svc := newLifecycleServiceForContractTests(repo)

	b, err := NewBuild(uuid.New(), uuid.New(), validManifestForServiceLifecycleTests(), nil)
	if err != nil {
		t.Fatalf("failed to create build aggregate: %v", err)
	}
	_ = b.Queue()
	_ = b.Start()
	repo.byID[b.ID()] = b

	if err := svc.StartBuild(context.Background(), b.ID()); err != nil {
		t.Fatalf("expected StartBuild to be idempotent when already running, got %v", err)
	}
}

type asyncLifecycleExecutor struct{}

func (e *asyncLifecycleExecutor) Execute(ctx context.Context, build *Build) (*BuildResult, error) {
	return nil, ErrBuildExecutionInProgress
}

func (e *asyncLifecycleExecutor) Cancel(ctx context.Context, buildID uuid.UUID) error {
	return nil
}

func TestServiceLifecycleContract_ExecuteBuildKeepsRunningWhenAsync(t *testing.T) {
	repo := &lifecycleBuildRepo{byID: make(map[uuid.UUID]*Build)}
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
		&stubEventPublisher{},
		&asyncLifecycleExecutor{},
		nil,
		nil,
		nil,
		nil,
		&stubSystemConfigService{},
		nil,
		zap.NewNop(),
	)

	svc.executeBuild(context.Background(), b, false)

	got, err := repo.FindByID(context.Background(), b.ID())
	if err != nil {
		t.Fatalf("failed to reload build: %v", err)
	}
	if got == nil {
		t.Fatal("expected build to exist")
	}
	if got.Status() != BuildStatusRunning {
		t.Fatalf("expected build to remain running for async execution, got %s", got.Status())
	}
	if len(repo.updated) != 0 {
		t.Fatalf("expected no immediate terminal update, got %d updates", len(repo.updated))
	}
}
