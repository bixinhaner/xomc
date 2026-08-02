package components

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/stretchr/testify/require"
)

func TestInfraConnectsCoreAndPMRedisWithDistinctHealthAndMetrics(t *testing.T) {
	coreServer := miniredis.RunT(t)
	pmServer := miniredis.RunT(t)
	inf, err := NewInfra(appconfig.LogConfig{
		Level: "error", Format: "console", OutputPaths: []string{"stdout"},
	}, 0)
	require.NoError(t, err)

	require.NoError(t, inf.ConnectRedis(appconfig.RedisConfig{Addrs: []string{coreServer.Addr()}, PoolSize: 4}))
	require.NoError(t, inf.ConnectPMRedis(appconfig.RedisConfig{Addrs: []string{pmServer.Addr()}, PoolSize: 6}))
	require.NotNil(t, inf.Redis)
	require.NotNil(t, inf.PMRedis)
	require.NoError(t, inf.Redis.Set(context.Background(), "core-only", "1", 0).Err())
	require.NoError(t, inf.PMRedis.Set(context.Background(), "pm-only", "1", 0).Err())
	require.True(t, coreServer.Exists("core-only"))
	require.False(t, coreServer.Exists("pm-only"))
	require.True(t, pmServer.Exists("pm-only"))
	require.False(t, pmServer.Exists("core-only"))

	checks := map[string]bool{}
	for _, check := range inf.Health.checks {
		checks[check.name] = true
	}
	require.True(t, checks["redis-core"])
	require.True(t, checks["redis-pm"])

	families, err := inf.MetricsReg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "redis_pool_max" {
			continue
		}
		require.Len(t, family.Metric, 2)
		return
	}
	t.Fatal("redis_pool_max metric family not registered")
}
