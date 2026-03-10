package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

type runnerStub struct {
	results []bool
	err     error
	calls   int
}

func (r *runnerStub) RunNextTektonInstallerJob(ctx context.Context) (bool, error) {
	r.calls++
	if r.err != nil {
		return false, r.err
	}
	if len(r.results) == 0 {
		return false, nil
	}
	out := r.results[0]
	r.results = r.results[1:]
	return out, nil
}

func TestTektonInstallerDispatcherRunOnce(t *testing.T) {
	runner := &runnerStub{results: []bool{true, true, false}}
	dispatcher := NewTektonInstallerDispatcher(runner, zap.NewNop(), TektonInstallerDispatcherConfig{
		MaxJobsPerTick: 5,
	})

	processed, err := dispatcher.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if processed != 2 {
		t.Fatalf("expected 2 processed jobs, got %d", processed)
	}
}

func TestTektonInstallerDispatcherRunOnceError(t *testing.T) {
	runner := &runnerStub{err: errors.New("boom")}
	dispatcher := NewTektonInstallerDispatcher(runner, zap.NewNop(), TektonInstallerDispatcherConfig{})

	processed, err := dispatcher.RunOnce(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if processed != 0 {
		t.Fatalf("expected 0 processed jobs, got %d", processed)
	}
}

func TestTektonInstallerDispatcherRunStopsOnContextCancel(t *testing.T) {
	runner := &runnerStub{results: []bool{false}}
	dispatcher := NewTektonInstallerDispatcher(runner, zap.NewNop(), TektonInstallerDispatcherConfig{
		PollInterval: 10 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dispatcher.Run(ctx)
}
