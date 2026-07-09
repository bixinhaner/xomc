package provider

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/mml"
)

type fakeMMLScheduledRepo struct {
	mu     sync.Mutex
	claims int
}

func (r *fakeMMLScheduledRepo) ClaimDueTasks(context.Context, time.Time, int) ([]*mml.MMLTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claims++
	return nil, nil
}

func (r *fakeMMLScheduledRepo) RecordPeriodicChild(context.Context, *mml.MMLTask, *mml.MMLTask, *time.Time) error {
	return nil
}

func (r *fakeMMLScheduledRepo) FinalizePeriodicParent(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (r *fakeMMLScheduledRepo) claimCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.claims
}

func TestStartMMLSchedulerWiresRunLoopAndShutdown(t *testing.T) {
	logger := zap.NewNop()
	repo := &fakeMMLScheduledRepo{}
	container := &Container{
		GS: components.NewGracefulShutdown(2*time.Second, logger),
	}

	scheduler := startMMLScheduler(container, &mml.Service{}, repo, logger)
	require.NotNil(t, scheduler)
	require.Same(t, scheduler, container.miscDeps.mmlScheduler)
	require.GreaterOrEqual(t, repo.claimCount(), 1, "MML scheduler 启动时必须至少执行一次到期任务扫描")

	require.NoError(t, container.GS.Shutdown(context.Background()))
}
