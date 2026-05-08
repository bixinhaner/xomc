package parammodel

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMiniRedis(t *testing.T) redis.UniversalClient {
	t.Helper()
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func sampleMappings(modelID uuid.UUID) []ParamMapping {
	v := "1.0.0"
	max := int64(256)
	return []ParamMapping{
		{
			ID:           uuid.New(),
			ParamModelID: modelID,
			StandardPath: "Device.DeviceInfo.SAS.CpiId",
			PrivatePath:  "Device.DeviceInfo.CPI_Id",
			EntryType:    "parameter",
			Access:       "READ_WRITE",
			DataType:     "STRING",
			MaxValue:     &max,
			IsStorable:   true,
			IsActive:     true,
		},
		{
			ID:              uuid.New(),
			StandardPath:    "Device.X_Vendor.Build",
			PrivatePath:     "Device.X_Vendor.Build",
			EntryType:       "parameter",
			IsStorable:      false,
			IsActive:        true,
			SoftwareVersion: &v,
		},
	}
}

func TestRedisCache_DefaultRoundTrip(t *testing.T) {
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	id := uuid.New()
	mappings := sampleMappings(id)

	got, err := c.GetDefault(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, got)

	require.NoError(t, c.SetDefault(ctx, id, mappings))
	got, err = c.GetDefault(ctx, id)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, mappings[0].StandardPath, got[0].StandardPath)
	assert.Equal(t, mappings[0].MaxValue, got[0].MaxValue)

	require.NoError(t, c.InvalidateDefault(ctx, id))
	got, err = c.GetDefault(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisCache_DefaultEmptySliceCached(t *testing.T) {
	// 空切片是合法的"无映射"事实，必须能被缓存且回读为空切片（非 nil）
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	id := uuid.New()
	require.NoError(t, c.SetDefault(ctx, id, []ParamMapping{}))
	got, err := c.GetDefault(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Empty(t, got)
}

func TestRedisCache_DiscoveredRoundTrip(t *testing.T) {
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	productID := uuid.New()
	swVer := "1.2.3"
	mappings := sampleMappings(uuid.Nil)

	got, err := c.GetDiscovered(ctx, productID, swVer)
	require.NoError(t, err)
	assert.Nil(t, got)

	require.NoError(t, c.SetDiscovered(ctx, productID, swVer, mappings))
	got, err = c.GetDiscovered(ctx, productID, swVer)
	require.NoError(t, err)
	require.Len(t, got, 2)

	// 不同 swVersion 是独立条目
	got2, err := c.GetDiscovered(ctx, productID, "9.9.9")
	require.NoError(t, err)
	assert.Nil(t, got2)

	require.NoError(t, c.InvalidateDiscovered(ctx, productID, swVer))
	got, err = c.GetDiscovered(ctx, productID, swVer)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisCache_Version(t *testing.T) {
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	v0, err := c.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), v0)

	v1, err := c.BumpVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), v1)

	v2, err := c.BumpVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), v2)

	got, err := c.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), got)
}

func TestRedisCache_TTLOverride(t *testing.T) {
	// 覆写 TTL 必须生效；非正数回退默认值。
	client := newMiniRedis(t)
	c := NewRedisCacheWithTTL(client, 5*time.Minute, 0) // discoveredTTL ≤0 应 fallback
	assert.Equal(t, 5*time.Minute, c.defaultTTL)
	assert.Equal(t, discoveredMappingTTL, c.discoveredTTL)
}

func TestNopCache(t *testing.T) {
	ctx := context.Background()
	var c Cache = NopCache{}

	got, err := c.GetDefault(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, got)

	assert.NoError(t, c.SetDefault(ctx, uuid.New(), []ParamMapping{{}}))
	assert.NoError(t, c.InvalidateDefault(ctx, uuid.New()))

	got, err = c.GetDiscovered(ctx, uuid.New(), "1")
	assert.NoError(t, err)
	assert.Nil(t, got)

	assert.NoError(t, c.SetDiscovered(ctx, uuid.New(), "1", nil))
	assert.NoError(t, c.InvalidateDiscovered(ctx, uuid.New(), "1"))

	v, err := c.GetVersion(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), v)

	bumped, err := c.BumpVersion(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), bumped)
}
