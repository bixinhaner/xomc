package redis

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolMetrics_RegistersAllRedisMetricNames(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	sampler := func() poolSnapshot {
		return poolSnapshot{InUse: 4, Idle: 6, Max: 10}
	}
	pm := registerPoolMetricsWithSampler(sampler, reg, WithPoolMetricsInterval(time.Hour))
	defer pm.Stop()

	require.Equal(t, float64(4), testutil.ToFloat64(pm.InUse))
	require.Equal(t, float64(6), testutil.ToFloat64(pm.Idle))
	require.Equal(t, float64(10), testutil.ToFloat64(pm.Max))

	families, err := reg.Gather()
	require.NoError(t, err)
	names := metricNames(families)
	for _, want := range []string{"redis_pool_in_use", "redis_pool_idle", "redis_pool_max"} {
		assert.Contains(t, names, want, "missing metric %q", want)
	}
}

func TestPoolMetrics_SamplerUpdatesGauges(t *testing.T) {
	t.Parallel()

	var inUse, idle, max atomic.Int32
	inUse.Store(2)
	idle.Store(8)
	max.Store(10)

	sampler := func() poolSnapshot {
		return poolSnapshot{InUse: inUse.Load(), Idle: idle.Load(), Max: max.Load()}
	}

	reg := prometheus.NewRegistry()
	pm := registerPoolMetricsWithSampler(sampler, reg, WithPoolMetricsInterval(time.Hour))
	defer pm.Stop()

	require.Equal(t, float64(2), testutil.ToFloat64(pm.InUse))

	inUse.Store(9)
	idle.Store(1)
	pm.sample()

	require.Equal(t, float64(9), testutil.ToFloat64(pm.InUse))
	require.Equal(t, float64(1), testutil.ToFloat64(pm.Idle))
}

func TestRegisterPoolMetrics_FromPoolStatsProvider(t *testing.T) {
	t.Parallel()

	stub := &stubPoolStats{stats: &goredis.PoolStats{
		TotalConns: 7,
		IdleConns:  3,
	}}

	reg := prometheus.NewRegistry()
	pm := RegisterPoolMetrics(stub, 10, reg, WithPoolMetricsInterval(time.Hour))
	defer pm.Stop()

	// in_use = total - idle = 7 - 3 = 4
	require.Equal(t, float64(4), testutil.ToFloat64(pm.InUse))
	require.Equal(t, float64(3), testutil.ToFloat64(pm.Idle))
	require.Equal(t, float64(10), testutil.ToFloat64(pm.Max))
}

func TestRegisterPoolMetrics_PoolSizeUnknownFallsBackToTotal(t *testing.T) {
	t.Parallel()

	stub := &stubPoolStats{stats: &goredis.PoolStats{TotalConns: 5, IdleConns: 5}}

	reg := prometheus.NewRegistry()
	pm := RegisterPoolMetrics(stub, 0, reg, WithPoolMetricsInterval(time.Hour))
	defer pm.Stop()

	require.Equal(t, float64(5), testutil.ToFloat64(pm.Max))
}

func TestPoolMetrics_StopIdempotent(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	pm := registerPoolMetricsWithSampler(
		func() poolSnapshot { return poolSnapshot{} },
		reg,
		WithPoolMetricsInterval(10*time.Millisecond),
	)
	pm.Stop()
	pm.Stop()
}

func TestRegisterPoolMetrics_NilClientPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		require.NotNil(t, recover(), "nil client should panic")
	}()
	RegisterPoolMetrics(nil, 10, prometheus.NewRegistry())
}

// stubPoolStats 实现 poolStatsProvider，便于不连真实 redis 也能跑测试。
type stubPoolStats struct {
	stats *goredis.PoolStats
}

func (s *stubPoolStats) PoolStats() *goredis.PoolStats {
	return s.stats
}

// metricNames 提取 gather 出的所有 metric family 名字。
func metricNames(families []*dto.MetricFamily) []string {
	out := make([]string, 0, len(families))
	for _, f := range families {
		out = append(out, f.GetName())
	}
	return out
}
