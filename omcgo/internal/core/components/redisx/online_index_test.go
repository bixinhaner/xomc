package redisx

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestOnlineIndex(t *testing.T) (*OnlineIndex, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewOnlineIndex(client), mr
}

func TestOnlineIndex_MarkAndCount(t *testing.T) {
	idx, _ := newTestOnlineIndex(t)
	ctx := context.Background()

	require.NoError(t, idx.Mark(ctx, "SN-A", 1000))
	require.NoError(t, idx.Mark(ctx, "SN-B", 1000))
	require.NoError(t, idx.Mark(ctx, "SN-C", 1000))

	// cutoff=1000：三台 score>=1000 → 在线 3。
	n, err := idx.Count(ctx, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)

	// cutoff=1001：无人 score>=1001 → 在线 0（全部已"过期"）。
	n, err = idx.Count(ctx, 1001)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}

func TestOnlineIndex_MarkIsIdempotentOverwrite(t *testing.T) {
	idx, _ := newTestOnlineIndex(t)
	ctx := context.Background()

	require.NoError(t, idx.Mark(ctx, "SN-A", 1000))
	require.NoError(t, idx.Mark(ctx, "SN-A", 2000)) // 同 SN 覆盖 score，不新增成员

	// 仍只有 1 个成员；按新 score 计：cutoff=1500 → 在线 1。
	n, err := idx.Count(ctx, 1500)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}

func TestOnlineIndex_Prune(t *testing.T) {
	idx, _ := newTestOnlineIndex(t)
	ctx := context.Background()

	require.NoError(t, idx.Mark(ctx, "SN-old", 1000))
	require.NoError(t, idx.Mark(ctx, "SN-mid", 2000))
	require.NoError(t, idx.Mark(ctx, "SN-new", 3000))

	// 删 score<2000（开区间）→ 只删 SN-old，返回 1。
	removed, err := idx.Prune(ctx, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(1), removed)

	// 余下 SN-mid(2000) + SN-new(3000)。
	n, err := idx.Count(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}

func TestOnlineIndex_NilClientAndEmptySN_NoOp(t *testing.T) {
	ctx := context.Background()

	// nil client：所有方法安全降级，不 panic、不报错。
	nilIdx := NewOnlineIndex(nil)
	require.NoError(t, nilIdx.Mark(ctx, "SN-A", 1000))
	n, err := nilIdx.Count(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
	removed, err := nilIdx.Prune(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), removed)

	// 空 SN：Mark no-op，不写入。
	idx, _ := newTestOnlineIndex(t)
	require.NoError(t, idx.Mark(ctx, "", 1000))
	n, err = idx.Count(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}
