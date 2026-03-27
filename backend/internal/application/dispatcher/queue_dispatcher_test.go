package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockBuildRepo struct {
	mu                sync.Mutex
	queue             []*build.Build
	claimErr          error
	claimCalls        int
	updateStatusCalls []updateStatusCall
	requeueCalls      []requeueCall
	countRunning      int
}

type updateStatusCall struct {
	id           uuid.UUID
	status       build.BuildStatus
	errorMessage *string
}

type requeueCall struct {
	id        uuid.UUID
	nextRunAt time.Time
	errMsg    *string
}

func (m *mockBuildRepo) ClaimNextQueuedBuild(ctx context.Context) (*build.Build, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.claimCalls++
	if m.claimErr != nil {
		return nil, m.claimErr
	}
	if len(m.queue) == 0 {
		return nil, nil
	}
	b := m.queue[0]
	m.queue = m.queue[1:]
	return b, nil
}

func (m *mockBuildRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status build.BuildStatus, startedAt, completedAt *time.Time, errorMessage *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateStatusCalls = append(m.updateStatusCalls, updateStatusCall{
		id:           id,
		status:       status,
		errorMessage: errorMessage,
	})
	return nil
}

func (m *mockBuildRepo) RequeueBuild(ctx context.Context, id uuid.UUID, nextRunAt time.Time, errorMessage *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requeueCalls = append(m.requeueCalls, requeueCall{
		id:        id,
		nextRunAt: nextRunAt,
		errMsg:    errorMessage,
	})
	return nil
}

func (m *mockBuildRepo) Save(ctx context.Context, build *build.Build) error { panic("not implemented") }
func (m *mockBuildRepo) FindByID(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*build.Build, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*build.Build, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) FindByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*build.Build, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) FindByStatus(ctx context.Context, status build.BuildStatus, limit, offset int) ([]*build.Build, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) Update(ctx context.Context, build *build.Build) error {
	panic("not implemented")
}
func (m *mockBuildRepo) Delete(ctx context.Context, id uuid.UUID) error { panic("not implemented") }
func (m *mockBuildRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) CountByStatus(ctx context.Context, tenantID uuid.UUID, status build.BuildStatus) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.countRunning, nil
}
func (m *mockBuildRepo) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) FindRunningBuilds(ctx context.Context) ([]*build.Build, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) SaveBuildConfig(ctx context.Context, config *build.BuildConfigData) error {
	panic("not implemented")
}
func (m *mockBuildRepo) GetBuildConfig(ctx context.Context, buildID uuid.UUID) (*build.BuildConfigData, error) {
	panic("not implemented")
}
func (m *mockBuildRepo) UpdateBuildConfig(ctx context.Context, config *build.BuildConfigData) error {
	panic("not implemented")
}
func (m *mockBuildRepo) DeleteBuildConfig(ctx context.Context, buildID uuid.UUID) error {
	panic("not implemented")
}
func (m *mockBuildRepo) UpdateInfrastructureSelection(ctx context.Context, build *build.Build) error {
	panic("not implemented")
}

type mockDispatchService struct {
	mu                    sync.Mutex
	calls                 []uuid.UUID
	err                   error
	callCh                chan uuid.UUID
	scheduledProcessCalls int
	scheduledQueued       int
	scheduledErr          error
}

func (m *mockDispatchService) DispatchBuild(ctx context.Context, build *build.Build) error {
	m.mu.Lock()
	m.calls = append(m.calls, build.ID())
	m.mu.Unlock()
	if m.callCh != nil {
		m.callCh <- build.ID()
	}
	return m.err
}

func (m *mockDispatchService) ProcessScheduledTriggers(ctx context.Context, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scheduledProcessCalls++
	if m.scheduledErr != nil {
		return 0, m.scheduledErr
	}
	return m.scheduledQueued, nil
}

type mockSystemConfigService struct {
	maxConcurrent int
}

func (m *mockSystemConfigService) GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.BuildConfig, error) {
	return &systemconfig.BuildConfig{MaxConcurrentJobs: m.maxConcurrent}, nil
}

func (m *mockSystemConfigService) GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.ToolAvailabilityConfig, error) {
	return nil, nil
}

func newQueuedBuild(id uuid.UUID) *build.Build {
	manifest := build.BuildManifest{
		Name: "Test Build",
		Type: build.BuildTypeContainer,
	}
	return build.NewBuildFromDB(id, uuid.New(), uuid.New(), manifest, build.BuildStatusQueued, time.Now().UTC(), time.Now().UTC(), nil)
}

func TestQueuedBuildDispatcher_DispatchesQueuedBuild(t *testing.T) {
	buildID := uuid.New()
	repo := &mockBuildRepo{
		queue: []*build.Build{newQueuedBuild(buildID)},
	}
	dispatcherSvc := &mockDispatchService{
		callCh: make(chan uuid.UUID, 1),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case id := <-dispatcherSvc.callCh:
		require.Equal(t, buildID, id)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not dispatch build")
	}
}

func TestQueuedBuildDispatcher_RequeuesOnDispatchFailure(t *testing.T) {
	buildID := uuid.New()
	repo := &mockBuildRepo{
		queue: []*build.Build{newQueuedBuild(buildID)},
	}
	dispatcherSvc := &mockDispatchService{
		err:    errors.New("dispatch failed"),
		callCh: make(chan uuid.UUID, 1),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not attempt dispatch")
	}

	time.Sleep(10 * time.Millisecond)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.requeueCalls, 1)
	require.Equal(t, buildID, repo.requeueCalls[0].id)
}

func TestQueuedBuildDispatcher_FailsAfterMaxRetries(t *testing.T) {
	buildID := uuid.New()
	b := newQueuedBuild(buildID)
	b.SetDispatchState(3, nil)

	repo := &mockBuildRepo{
		queue: []*build.Build{b},
	}
	dispatcherSvc := &mockDispatchService{
		err:    errors.New("dispatch failed"),
		callCh: make(chan uuid.UUID, 1),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
		MaxRetries:         3,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not attempt dispatch")
	}

	time.Sleep(10 * time.Millisecond)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.updateStatusCalls, 1)
	require.Equal(t, buildID, repo.updateStatusCalls[0].id)
	require.Equal(t, build.BuildStatusFailed, repo.updateStatusCalls[0].status)
}

func TestQueuedBuildDispatcher_ProcessesScheduledTriggersBeforeDispatch(t *testing.T) {
	buildID := uuid.New()
	repo := &mockBuildRepo{
		queue: []*build.Build{newQueuedBuild(buildID)},
	}
	dispatcherSvc := &mockDispatchService{
		callCh:          make(chan uuid.UUID, 1),
		scheduledQueued: 1,
		scheduledErr:    nil,
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	_, err := d.RunOnce(context.Background())
	require.NoError(t, err)
	dispatcherSvc.mu.Lock()
	defer dispatcherSvc.mu.Unlock()
	require.Equal(t, 1, dispatcherSvc.scheduledProcessCalls)
}

func TestQueuedBuildDispatcher_FailsImmediatelyOnCapabilityEntitlementError(t *testing.T) {
	buildID := uuid.New()
	repo := &mockBuildRepo{
		queue: []*build.Build{newQueuedBuild(buildID)},
	}
	dispatcherSvc := &mockDispatchService{
		err:    fmt.Errorf("%w: gpu build capability is not entitled for this tenant", build.ErrBuildCapabilityNotEntitled),
		callCh: make(chan uuid.UUID, 1),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not attempt dispatch")
	}

	time.Sleep(10 * time.Millisecond)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Empty(t, repo.requeueCalls)
	require.Len(t, repo.updateStatusCalls, 1)
	require.Equal(t, buildID, repo.updateStatusCalls[0].id)
	require.Equal(t, build.BuildStatusFailed, repo.updateStatusCalls[0].status)
}

func TestQueuedBuildDispatcher_RespectsMaxDispatchPerTick(t *testing.T) {
	repo := &mockBuildRepo{
		queue: []*build.Build{
			newQueuedBuild(uuid.New()),
			newQueuedBuild(uuid.New()),
		},
	}
	dispatcherSvc := &mockDispatchService{
		callCh: make(chan uuid.UUID, 2),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       200 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not dispatch build")
	}

	repo.mu.Lock()
	claimCalls := repo.claimCalls
	repo.mu.Unlock()
	require.Equal(t, 1, claimCalls)
}

func TestQueuedBuildDispatcher_DispatchesMultiplePerTick(t *testing.T) {
	repo := &mockBuildRepo{
		queue: []*build.Build{
			newQueuedBuild(uuid.New()),
			newQueuedBuild(uuid.New()),
		},
	}
	dispatcherSvc := &mockDispatchService{
		callCh: make(chan uuid.UUID, 2),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       500 * time.Millisecond,
		MaxDispatchPerTick: 2,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not dispatch first build")
	}
	select {
	case <-dispatcherSvc.callCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("dispatcher did not dispatch second build in same tick")
	}
}

func TestQueuedBuildDispatcher_ClaimErrorDoesNotDispatch(t *testing.T) {
	repo := &mockBuildRepo{
		claimErr: errors.New("claim failed"),
	}
	dispatcherSvc := &mockDispatchService{
		callCh: make(chan uuid.UUID, 1),
	}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, nil, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
		t.Fatal("dispatcher should not dispatch when claim fails")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestQueuedBuildDispatcher_RespectsConcurrencyLimit(t *testing.T) {
	buildID := uuid.New()
	repo := &mockBuildRepo{
		queue:        []*build.Build{newQueuedBuild(buildID)},
		countRunning: 5,
	}
	dispatcherSvc := &mockDispatchService{
		callCh: make(chan uuid.UUID, 1),
	}
	systemConfigSvc := &mockSystemConfigService{maxConcurrent: 3}

	d := NewQueuedBuildDispatcher(repo, dispatcherSvc, systemConfigSvc, zap.NewNop(), QueueDispatcherConfig{
		PollInterval:       10 * time.Millisecond,
		MaxDispatchPerTick: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	select {
	case <-dispatcherSvc.callCh:
		t.Fatal("dispatcher should not dispatch when concurrency limit exceeded")
	case <-time.After(100 * time.Millisecond):
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.updateStatusCalls, 0)
	require.Len(t, repo.requeueCalls, 1)
	require.Equal(t, buildID, repo.requeueCalls[0].id)
}
