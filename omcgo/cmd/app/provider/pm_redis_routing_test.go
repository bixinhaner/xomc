package provider

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAppKPIRouteCacheUsesPMRedis(t *testing.T) {
	coreServer := miniredis.RunT(t)
	pmServer := miniredis.RunT(t)
	coreClient := redis.NewClient(&redis.Options{Addr: coreServer.Addr()})
	pmClient := redis.NewClient(&redis.Options{Addr: pmServer.Addr()})
	t.Cleanup(func() {
		_ = coreClient.Close()
		_ = pmClient.Close()
	})
	c := &Container{Redis: coreClient, PMRedis: pmClient}

	cache := c.kpiRouteRedisCache()
	require.NotNil(t, cache)
	require.NoError(t, cache.Put(
		context.Background(), &router.KPIRoute{ProductID: uuid.New()}, 0,
	))
	_, err := cache.BumpVersion(context.Background())
	require.NoError(t, err)
	require.Empty(t, coreServer.Keys())
	require.NotEmpty(t, pmServer.Keys())
}
