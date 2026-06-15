package postgres

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestRegisterPoolMetrics_DualPoolConstLabelsNoCollision 验证：主库与时序库两个
// pgxpool 通过 prometheus.WrapRegistererWith 各加 pool 常量标签后，注册到同一
// registry 不会因同名指标（pgxpool_in_use 等）冲突 panic，且各自产出带 pool 标签
// 的独立序列。这正是 infra.go ConnectPostgres(pool="main") / ConnectTimescale(
// pool="tsdb") 采用的双池接线策略。
func TestRegisterPoolMetrics_DualPoolConstLabelsNoCollision(t *testing.T) {
	reg := prometheus.NewRegistry()

	mainSampler := func() poolSnapshot { return poolSnapshot{InUse: 3, Idle: 7, Max: 10, AcquireCount: 100} }
	tsdbSampler := func() poolSnapshot { return poolSnapshot{InUse: 1, Idle: 4, Max: 5, AcquireCount: 20} }

	pmMain := registerPoolMetricsWithSampler(mainSampler,
		prometheus.WrapRegistererWith(prometheus.Labels{"pool": "main"}, reg),
		WithPoolMetricsInterval(time.Hour))
	defer pmMain.Stop()

	// 第二个池注册同名指标 —— 若没有 pool 常量标签区分，这里会 panic（重复注册）。
	pmTsdb := registerPoolMetricsWithSampler(tsdbSampler,
		prometheus.WrapRegistererWith(prometheus.Labels{"pool": "tsdb"}, reg),
		WithPoolMetricsInterval(time.Hour))
	defer pmTsdb.Stop()

	families, err := reg.Gather()
	require.NoError(t, err)

	byPool := make(map[string]float64)
	for _, f := range families {
		if f.GetName() != "pgxpool_in_use" {
			continue
		}
		for _, m := range f.GetMetric() {
			var pool string
			for _, l := range m.GetLabel() {
				if l.GetName() == "pool" {
					pool = l.GetValue()
				}
			}
			byPool[pool] = m.GetGauge().GetValue()
		}
	}

	require.Equal(t, 3.0, byPool["main"], "pgxpool_in_use{pool=\"main\"} 应为 3")
	require.Equal(t, 1.0, byPool["tsdb"], "pgxpool_in_use{pool=\"tsdb\"} 应为 1")
}
