package router

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/indicator"
)

// 共享 helper：构造一份带 KPI 子集的 KPIRoute fixture。
func sampleRoute(productID uuid.UUID) *KPIRoute {
	return &KPIRoute{
		ProductID:           productID,
		IndicatorPlatform:   "BLQ-LTE-V1",
		IndicatorDeviceType: indicator.DeviceTypeENB,
		Counters: []CounterDef{
			{IndicatorID: "C-001", Name: "RRC_Conn_Att", StatisType: "sum"},
		},
		KPIs: []KPIDef{
			{IndicatorID: "K-001", Name: "RRC_Succ_Rate", StatisType: "pct",
				Formula:      "RRC_Conn_Succ / RRC_Conn_Att",
				Dependencies: []string{"RRC_Conn_Succ", "RRC_Conn_Att"}},
		},
	}
}

func newRedisHarness(t *testing.T) *RedisCache {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewRedisCache(rdb)
}

// Case 1: 默认 cache_version=0，Put → Get 应命中。
func TestRedisCache_PutThenGet_Hit(t *testing.T) {
	c := newRedisHarness(t)
	productID := uuid.New()
	want := sampleRoute(productID)

	require.NoError(t, c.Put(context.Background(), want))

	got, err := c.Get(context.Background(), productID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, want.IndicatorPlatform, got.IndicatorPlatform)
	require.Equal(t, want.IndicatorDeviceType, got.IndicatorDeviceType)
	require.Len(t, got.KPIs, 1)
	require.Equal(t, "RRC_Succ_Rate", got.KPIs[0].Name)
}

// Case 2: BumpVersion 后旧条目立即视为 stale → Get miss。
func TestRedisCache_BumpVersion_InvalidatesAll(t *testing.T) {
	c := newRedisHarness(t)
	productID := uuid.New()

	require.NoError(t, c.Put(context.Background(), sampleRoute(productID)))

	_, err := c.BumpVersion(context.Background())
	require.NoError(t, err)

	got, err := c.Get(context.Background(), productID)
	require.NoError(t, err)
	require.Nil(t, got, "version mismatch should be treated as miss")
}

// Case 3: 不同进程间不一致 — Put 写入 v=0 后另一进程 INCR；
// 此时本进程再 Get 该 productID 应 miss（防止读到 stale 值）。
// miniredis 共享 Redis 后端模拟两进程同时读写。
func TestRedisCache_StaleAfterRemoteIncr(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c1 := NewRedisCache(rdb)
	c2 := NewRedisCache(rdb)

	productID := uuid.New()
	require.NoError(t, c1.Put(context.Background(), sampleRoute(productID)))

	got, err := c1.Get(context.Background(), productID)
	require.NoError(t, err)
	require.NotNil(t, got, "c1 just wrote it; same-version Get should hit")

	// 另一个 Router 进程 INCR cache_version（指令端：indicator 表变更后）。
	_, err = c2.BumpVersion(context.Background())
	require.NoError(t, err)

	// 现在 c1 / c2 任一再读都应 miss。
	got1, err := c1.Get(context.Background(), productID)
	require.NoError(t, err)
	require.Nil(t, got1, "after remote INCR, c1's stale entry should miss")

	got2, err := c2.Get(context.Background(), productID)
	require.NoError(t, err)
	require.Nil(t, got2)
}
