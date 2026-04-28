package postgres

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolMetrics_RegistersAllFiveMetricNames(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	sampler := func() poolSnapshot {
		return poolSnapshot{InUse: 3, Idle: 7, Max: 10, AcquireCount: 100}
	}
	pm := registerPoolMetricsWithSampler(sampler, reg, WithPoolMetricsInterval(time.Hour))
	defer pm.Stop()

	// Trigger one extra sample on top of the constructor's initial sample
	// to make assertions independent of background goroutine timing.
	pm.sample()

	require.Equal(t, float64(3), testutil.ToFloat64(pm.InUse))
	require.Equal(t, float64(7), testutil.ToFloat64(pm.Idle))
	require.Equal(t, float64(10), testutil.ToFloat64(pm.Max))
	require.Equal(t, float64(100), testutil.ToFloat64(pm.AcquireTotal))

	families, err := reg.Gather()
	require.NoError(t, err)
	names := metricNames(families)
	for _, want := range []string{"pgxpool_in_use", "pgxpool_idle", "pgxpool_max", "pgxpool_acquire_total"} {
		assert.Contains(t, names, want, "missing metric %q", want)
	}
}

func TestPoolMetrics_AcquireTotalIsMonotonic(t *testing.T) {
	t.Parallel()

	var count atomic.Int64
	count.Store(50)
	sampler := func() poolSnapshot {
		return poolSnapshot{InUse: 1, Idle: 1, Max: 5, AcquireCount: count.Load()}
	}

	reg := prometheus.NewRegistry()
	pm := registerPoolMetricsWithSampler(sampler, reg, WithPoolMetricsInterval(time.Hour))
	defer pm.Stop()

	require.Equal(t, float64(50), testutil.ToFloat64(pm.AcquireTotal))

	count.Store(75)
	pm.sample()
	require.Equal(t, float64(75), testutil.ToFloat64(pm.AcquireTotal))

	// counter 必须单调递增；pool 重建（计数回退）时 baseline 重置但 prom counter 不下跌。
	count.Store(10)
	pm.sample()
	require.Equal(t, float64(75), testutil.ToFloat64(pm.AcquireTotal))

	count.Store(40)
	pm.sample()
	require.Equal(t, float64(105), testutil.ToFloat64(pm.AcquireTotal))
}

func TestPoolMetrics_StopIdempotent(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	pm := registerPoolMetricsWithSampler(
		func() poolSnapshot { return poolSnapshot{} },
		reg,
		WithPoolMetricsInterval(10*time.Millisecond),
	)

	// 多次 Stop 不应 panic / 卡死。
	pm.Stop()
	pm.Stop()
	pm.Stop()
}

func TestPoolMetrics_BackgroundSampling(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	sampler := func() poolSnapshot {
		calls.Add(1)
		return poolSnapshot{InUse: 2, Idle: 8, Max: 10, AcquireCount: calls.Load()}
	}

	reg := prometheus.NewRegistry()
	pm := registerPoolMetricsWithSampler(sampler, reg, WithPoolMetricsInterval(20*time.Millisecond))
	defer pm.Stop()

	// 初始构造时已调用一次。等几个 tick 看后台是否在跑。
	require.Eventually(t, func() bool {
		return calls.Load() >= 3
	}, 2*time.Second, 20*time.Millisecond)
}

func TestPoolMetrics_NilSamplerPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		require.NotNil(t, r, "nil sampler should panic")
	}()
	registerPoolMetricsWithSampler(nil, prometheus.NewRegistry())
}

func TestPoolMetrics_NilRegistererPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		require.NotNil(t, r, "nil registerer should panic")
	}()
	registerPoolMetricsWithSampler(func() poolSnapshot { return poolSnapshot{} }, nil)
}

func TestRegisterPoolMetrics_NilPoolPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		require.NotNil(t, r, "nil pool should panic")
	}()
	RegisterPoolMetrics(nil, prometheus.NewRegistry())
}

// metricNames 提取 gather 出的所有 metric family 名字。
func metricNames(families []*dto.MetricFamily) []string {
	out := make([]string, 0, len(families))
	for _, f := range families {
		out = append(out, f.GetName())
	}
	return out
}
