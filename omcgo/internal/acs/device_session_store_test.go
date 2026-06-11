package acs

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDeviceSessionStore(t *testing.T, mr *miniredis.Miniredis) *RedisDeviceSessionStore {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return NewRedisDeviceSessionStore(rdb, 10*time.Minute)
}

// TestDeviceSessionStore_SwapReturnsOld 验证 Swap 原子换指针并返回被顶替的旧 sessionID。
func TestDeviceSessionStore_SwapReturnsOld(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s := newTestDeviceSessionStore(t, mr)

	// 首次：无旧会话。
	old, err := s.Swap(ctx, "SN-1", "sess-A")
	require.NoError(t, err)
	assert.Empty(t, old, "first swap has no prior session")

	cur, err := s.Get(ctx, "SN-1")
	require.NoError(t, err)
	assert.Equal(t, "sess-A", cur)

	// 第二次 Inform：返回被顶替的 sess-A，并把指针换成 sess-B。
	old, err = s.Swap(ctx, "SN-1", "sess-B")
	require.NoError(t, err)
	assert.Equal(t, "sess-A", old, "swap must return the orphaned previous session")

	cur, _ = s.Get(ctx, "SN-1")
	assert.Equal(t, "sess-B", cur)
}

// TestDeviceSessionStore_SwapSameSessionNoOrphan 验证用同一 sessionID 重复 Swap 不报孤儿。
func TestDeviceSessionStore_SwapSameSessionNoOrphan(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s := newTestDeviceSessionStore(t, mr)

	_, _ = s.Swap(ctx, "SN-2", "sess-X")
	old, err := s.Swap(ctx, "SN-2", "sess-X")
	require.NoError(t, err)
	assert.Empty(t, old, "swapping to the same sessionID is not an orphan")
}

// TestDeviceSessionStore_CompareAndDelete 验证 CAS 删除语义：仅当指针仍等于本会话 ID 时删除。
func TestDeviceSessionStore_CompareAndDelete(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	s := newTestDeviceSessionStore(t, mr)

	_, _ = s.Swap(ctx, "SN-3", "sess-1")

	// 指针已被新 Inform 覆盖为 sess-2：旧会话 completeSession 时 CAS 删除不应误删。
	_, _ = s.Swap(ctx, "SN-3", "sess-2")
	require.NoError(t, s.CompareAndDelete(ctx, "SN-3", "sess-1"))
	cur, _ := s.Get(ctx, "SN-3")
	assert.Equal(t, "sess-2", cur, "stale completeSession must not delete the newer pointer")

	// 当前会话 sess-2 自己 completeSession：CAS 命中，指针清空。
	require.NoError(t, s.CompareAndDelete(ctx, "SN-3", "sess-2"))
	cur, _ = s.Get(ctx, "SN-3")
	assert.Empty(t, cur, "matching completeSession clears the pointer")
}

// TestDeviceSessionStore_CrossInstanceOrphanCleanup 模拟 issue #65 跨实例孤儿清理：
// 实例 A 起了会话但没 complete；实例 B 收到同 SN 的新 Inform → 读到 A 的旧 sessionID。
func TestDeviceSessionStore_CrossInstanceOrphanCleanup(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	// 两个实例共享同一 Redis store。
	storeA := NewRedisDeviceSessionStore(rdb, 10*time.Minute)
	storeB := NewRedisDeviceSessionStore(rdb, 10*time.Minute)

	// A 起会话（不 complete，模拟 CPE 失联 / RPC 未响应）。
	old, err := storeA.Swap(ctx, "SN-XINST", "sessA-orphan")
	require.NoError(t, err)
	assert.Empty(t, old)

	// B 收到新 Inform，读到 A 的旧 sessionID → 可据此跨实例清理旧会话。
	orphan, err := storeB.Swap(ctx, "SN-XINST", "sessB-new")
	require.NoError(t, err)
	assert.Equal(t, "sessA-orphan", orphan, "instance B must see instance A's orphaned session")

	cur, _ := storeB.Get(ctx, "SN-XINST")
	assert.Equal(t, "sessB-new", cur)
}
