package device

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// newCache 用 miniredis 起一个内存 Redis，构造 DeviceCache。
func newCache(t *testing.T) (*DeviceCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewDeviceCache(client, zap.NewNop()), mr
}

func sampleDevice(sn string) *model.Device {
	return &model.Device{
		ID:           uuid.New(),
		SerialNumber: sn,
		ProductClass: "PC-1",
		Carrier:      model.CarrierCMCC,
	}
}

// --- Set + Get 往返（成功路径）---

func TestDeviceCache_SetGetRoundTrip(t *testing.T) {
	cache, _ := newCache(t)
	ctx := context.Background()

	dev := sampleDevice("SN-001")
	cache.Set(ctx, dev)

	got, ok := cache.Get(ctx, "SN-001")
	require.True(t, ok, "Set 后应命中缓存")
	assert.Equal(t, dev.ID, got.ID)
	assert.Equal(t, "SN-001", got.SerialNumber)
	assert.Equal(t, model.CarrierCMCC, got.Carrier)
}

// --- Get miss：键不存在返回 (nil,false)，不报错 ---

func TestDeviceCache_GetMiss(t *testing.T) {
	cache, _ := newCache(t)
	got, ok := cache.Get(context.Background(), "SN-NONE")
	assert.False(t, ok)
	assert.Nil(t, got)
}

// --- Get 反序列化失败：缓存里是脏数据 → (nil,false)，安全降级 ---

func TestDeviceCache_GetCorruptData(t *testing.T) {
	cache, mr := newCache(t)
	// 直接往底层写非法 JSON，模拟缓存被污染。
	require.NoError(t, mr.Set(deviceCacheKey("SN-BAD"), "{not-json"))

	got, ok := cache.Get(context.Background(), "SN-BAD")
	assert.False(t, ok, "脏数据应被当作 miss，不 panic")
	assert.Nil(t, got)
}

// --- Delete：删除后再 Get 应 miss ---

func TestDeviceCache_Delete(t *testing.T) {
	cache, _ := newCache(t)
	ctx := context.Background()
	cache.Set(ctx, sampleDevice("SN-DEL"))

	cache.Delete(ctx, "SN-DEL")
	_, ok := cache.Get(ctx, "SN-DEL")
	assert.False(t, ok)
}

// --- GetOrLoad 缓存命中：不调 loader ---

func TestDeviceCache_GetOrLoad_HitSkipsLoader(t *testing.T) {
	cache, _ := newCache(t)
	ctx := context.Background()
	cache.Set(ctx, sampleDevice("SN-HIT"))

	loaderCalled := false
	got, err := cache.GetOrLoad(ctx, "SN-HIT", func(context.Context, string) (*model.Device, error) {
		loaderCalled = true
		return nil, errors.New("should not be called")
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, loaderCalled, "命中缓存时不应触发 DB loader")
}

// --- GetOrLoad 缓存未命中：调 loader 并回填 ---

func TestDeviceCache_GetOrLoad_MissLoadsAndCaches(t *testing.T) {
	cache, _ := newCache(t)
	ctx := context.Background()
	dev := sampleDevice("SN-MISS")

	calls := 0
	loader := func(context.Context, string) (*model.Device, error) {
		calls++
		return dev, nil
	}

	// 第一次 miss → loader 被调，结果回填缓存。
	got1, err := cache.GetOrLoad(ctx, "SN-MISS", loader)
	require.NoError(t, err)
	assert.Equal(t, dev.ID, got1.ID)
	assert.Equal(t, 1, calls)

	// 第二次应命中缓存，loader 不再被调。
	got2, err := cache.GetOrLoad(ctx, "SN-MISS", loader)
	require.NoError(t, err)
	assert.Equal(t, dev.ID, got2.ID)
	assert.Equal(t, 1, calls, "第二次应走缓存，loader 调用次数仍为 1")
}

// --- GetOrLoad 失败路径：loader（DB/参数更新）报错应 wrap 上抛，不回填 ---

func TestDeviceCache_GetOrLoad_LoaderError(t *testing.T) {
	cache, _ := newCache(t)
	ctx := context.Background()
	sentinel := errors.New("db connection lost")

	got, err := cache.GetOrLoad(ctx, "SN-ERR", func(context.Context, string) (*model.Device, error) {
		return nil, sentinel
	})
	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "load device")

	// loader 失败不应污染缓存。
	_, ok := cache.Get(ctx, "SN-ERR")
	assert.False(t, ok, "loader 失败不应回填缓存")
}

// --- GetOrLoad 找不到设备（loader 返回 nil,nil）：不缓存、不报错 ---

func TestDeviceCache_GetOrLoad_NotFoundSkipsCache(t *testing.T) {
	cache, _ := newCache(t)
	ctx := context.Background()

	got, err := cache.GetOrLoad(ctx, "SN-404", func(context.Context, string) (*model.Device, error) {
		return nil, nil
	})
	require.NoError(t, err)
	assert.Nil(t, got)
	_, ok := cache.Get(ctx, "SN-404")
	assert.False(t, ok, "DB 未找到的设备不应写入缓存")
}

// --- GetOrLoad 在 Redis 不可用时仍能降级到 loader（缓存层不阻断主流程）---

func TestDeviceCache_GetOrLoad_RedisDownFallsBackToLoader(t *testing.T) {
	cache, mr := newCache(t)
	ctx := context.Background()
	mr.Close() // 关掉 Redis 模拟连接失败

	dev := sampleDevice("SN-RD")
	got, err := cache.GetOrLoad(ctx, "SN-RD", func(context.Context, string) (*model.Device, error) {
		return dev, nil
	})
	// Get 出错被当作 miss，loader 仍被调用并成功返回；Set 失败仅记日志不影响结果。
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, dev.ID, got.ID)
}
