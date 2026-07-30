package product

import (
	"context"
	"testing"

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

func TestRedisCache_RoundTrip(t *testing.T) {
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	id := uuid.New()
	pmID := uuid.New()
	p := &Product{
		ID:                  id,
		Name:                "QRTB",
		Vendor:              "Baicells",
		Tech:                "4G",
		RadioModes:          "SC,CA",
		Description:         "test product",
		ParamModelID:        &pmID,
		IndicatorDeviceType: "enb",
		IndicatorPlatform:   "BLQ",
		AlarmNeType:         "ENB",
		EnableFileType11:    true,
		DeviceAttrsOverride: map[string]any{"access": true, "data_type": false},
		EnableUnknownAlarm:  false,
	}

	// 初次 miss
	got, err := c.GetProduct(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Set + Get
	require.NoError(t, c.SetProduct(ctx, p))
	got, err = c.GetProduct(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.Name, got.Name)
	assert.Equal(t, p.IndicatorPlatform, got.IndicatorPlatform)
	assert.Equal(t, p.EnableFileType11, got.EnableFileType11)
	assert.NotNil(t, got.ParamModelID)
	assert.Equal(t, *p.ParamModelID, *got.ParamModelID)

	// Invalidate
	require.NoError(t, c.InvalidateProduct(ctx, id))
	got, err = c.GetProduct(ctx, id)
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

func TestRedisCache_ProductMissesAfterVersionBump(t *testing.T) {
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	p := &Product{
		ID:                 uuid.New(),
		Name:               "BM",
		EnableUnknownAlarm: false,
	}

	require.NoError(t, c.SetProduct(ctx, p))
	got, err := c.GetProduct(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.EnableUnknownAlarm)

	_, err = c.BumpVersion(ctx)
	require.NoError(t, err)

	got, err = c.GetProduct(ctx, p.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisCache_SetNilProductRejected(t *testing.T) {
	client := newMiniRedis(t)
	c := NewRedisCache(client)
	ctx := context.Background()

	err := c.SetProduct(ctx, nil)
	assert.Error(t, err)
}

func TestNopCache(t *testing.T) {
	ctx := context.Background()
	var c Cache = NopCache{}

	got, err := c.GetProduct(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, got)

	assert.NoError(t, c.SetProduct(ctx, &Product{ID: uuid.New(), Name: "x"}))
	assert.NoError(t, c.InvalidateProduct(ctx, uuid.New()))

	v, err := c.GetVersion(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), v)

	bumped, err := c.BumpVersion(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), bumped)
}
