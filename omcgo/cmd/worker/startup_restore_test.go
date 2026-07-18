package main

import (
	"context"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartPendingQueueRestoreDoesNotBlockWorkerStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entered := make(chan struct{})
	done := startPendingQueueRestore(ctx, func(ctx context.Context) (task.RestoreStats, error) {
		close(entered)
		<-ctx.Done()
		return task.RestoreStats{}, ctx.Err()
	}, zap.NewNop())

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("后台恢复任务未启动")
	}

	select {
	case <-done:
		t.Fatal("恢复任务阻塞时不应提前结束")
	default:
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("取消上下文后恢复任务未退出")
	}
}

func TestStartPendingQueueRestoreLogsAndCompletesOnSuccess(t *testing.T) {
	done := startPendingQueueRestore(context.Background(), func(context.Context) (task.RestoreStats, error) {
		return task.RestoreStats{Scanned: 3, Pushed: 2, Skipped: 1}, nil
	}, zap.NewNop())

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("恢复任务成功后未退出")
	}

	require.NotNil(t, done)
}
