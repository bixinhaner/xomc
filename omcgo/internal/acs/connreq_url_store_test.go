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

// TestConnReqURLStore_CrossInstanceSetGet 验证 issue #65 CR-URL 场景：Inform 落在实例 A
// 写入设备 ConnectionRequestURL；会话后续唤在实例 B 读共享存储 → 拿到同一 URL（修复前
// 非 Inform 实例上进程缓存 miss → httpURL 为空 → 纯 HTTP-CR 设备续唤失败）。
func TestConnReqURLStore_CrossInstanceSetGet(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	instanceA := NewRedisConnReqURLStore(rdb, 30*time.Minute)
	instanceB := NewRedisConnReqURLStore(rdb, 30*time.Minute)

	const url = "http://10.1.2.3:7547/cr/abc"
	require.NoError(t, instanceA.Set(ctx, "SN-CR", url))

	got, err := instanceB.Get(ctx, "SN-CR")
	require.NoError(t, err)
	assert.Equal(t, url, got, "post-session wake on another instance must read the shared CR URL")
}

// TestConnReqURLStore_MissReturnsEmpty 验证未缓存设备返回空串且无错误（等价改造前 miss）。
func TestConnReqURLStore_MissReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	s := NewRedisConnReqURLStore(rdb, 30*time.Minute)
	got, err := s.Get(ctx, "SN-UNKNOWN")
	require.NoError(t, err)
	assert.Empty(t, got)
}
