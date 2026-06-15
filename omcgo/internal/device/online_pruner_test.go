package device

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOnlinePruner_TickPrunesAndCounts(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	idx := redisx.NewOnlineIndex(client)
	ctx := context.Background()
	const now = int64(10000)

	// 默认窗口：prune 保留 1h(3600s)，count 统计 600s。
	idx.Mark(ctx, "SN-old", now-7200)       // 早于 now-3600 → 应被 Prune 删
	idx.Mark(ctx, "SN-staleKept", now-1800) // 早于统计窗 600 但晚于保留窗 3600 → 保留但不计在线
	idx.Mark(ctx, "SN-fresh", now-100)      // 600s 内 → 在线

	p := NewOnlinePruner(idx, zap.NewNop())
	p.tick(ctx, now)

	// Prune 后集合剩 SN-staleKept + SN-fresh = 2。
	total, err := idx.Count(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)

	// 600s 在线窗内只有 SN-fresh = 1。
	online, err := idx.Count(ctx, now-600)
	require.NoError(t, err)
	require.Equal(t, int64(1), online)
}

func TestOnlinePruner_NilIndex_NoOp(t *testing.T) {
	p := NewOnlinePruner(nil, zap.NewNop())
	p.Start() // 不应启动 goroutine / 不 panic
	p.tick(context.Background(), 0)
	p.Stop()
}
