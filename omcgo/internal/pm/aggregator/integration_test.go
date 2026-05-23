//go:build integration

package aggregator

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// Integration test：注入 15min pm_metrics 数据 → 调 AggregateCounters → 验证 pm_metrics_hourly 行正确。
//
// 运行方式：
//
//	OMCGO_DB_DSN=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
//	  go test -tags integration -count=1 -run Integration ./internal/pm/aggregator/...
//
// 注意：本测试会修改 pm_metrics / pm_metrics_hourly；用唯一 device_sn 'AGGR-INT-TEST-001' 隔离。
func Test_Integration_HourlyAggregateCounters_SumPath(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	const (
		testOUI = "INTTST"
		testSN  = "AGGR-INT-TEST-001"
		path    = "L.Cell.Avail.Dur"
	)
	// 唯一 bucket 时间：2026-05-22 10:00 ~ 11:00 UTC（避免与现网数据冲突）
	bucketStart := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(time.Hour)

	// 1) 清理上次运行残留
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_hourly WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
	}
	cleanup()
	defer cleanup()

	// 2) 注入 4 个 15min 行：值 100/200/150/250 → SUM=700
	values := []float64{100, 200, 150, 250}
	for i, v := range values {
		endTime := bucketStart.Add(time.Duration(i+1) * 15 * time.Minute) // 10:15, 10:30, 10:45, 11:00
		startTime := endTime.Add(-15 * time.Minute)
		_, err := pool.Exec(ctx, `
INSERT INTO pm_metrics (device_oui, device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time)
VALUES ($1, $2, $3, 'counter', $4, 'sum', '15min', $5, $6, $5)`,
			testOUI, testSN, path, v, endTime, startTime)
		require.NoError(t, err)
	}

	// 3) 跑 AggregateCounters (window 10:00–11:00 排除 11:00 的最后一行 → end_time>=$4 AND end_time<$5)
	//    bucketEnd 是开区间，所以 endTime=11:00 的行不会被 SUM。期望取 10:15/10:30/10:45 三行 = 100+200+150 = 450
	a := NewWithPool(pool, nil, nil)
	w := WindowSpec{Granularity: metrics.GranularityHourly, Start: bucketStart, End: bucketEnd}
	n, err := a.AggregateCounters(ctx, "pm_metrics", "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 1)

	// 4) 验证 pm_metrics_hourly 行
	var aggValue float64
	var aggStatis, aggGran string
	err = pool.QueryRow(ctx, `
SELECT metric_value, statis_type, granularity
FROM pm_metrics_hourly
WHERE device_oui=$1 AND device_sn=$2 AND metric_path=$3 AND end_time=$4`,
		testOUI, testSN, path, bucketEnd).Scan(&aggValue, &aggStatis, &aggGran)
	require.NoError(t, err)
	assert.InDelta(t, 450.0, aggValue, 0.001, "10:15+10:30+10:45 = 100+200+150 = 450 (11:00 falls outside half-open [10:00,11:00))")
	assert.Equal(t, "sum", aggStatis)
	assert.Equal(t, "hourly", aggGran)

	// 5) 再次跑：UPSERT 幂等
	n2, err := a.AggregateCounters(ctx, "pm_metrics", "pm_metrics_hourly", w)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n2, 1)
	err = pool.QueryRow(ctx, `
SELECT metric_value FROM pm_metrics_hourly
WHERE device_oui=$1 AND device_sn=$2 AND metric_path=$3 AND end_time=$4`,
		testOUI, testSN, path, bucketEnd).Scan(&aggValue)
	require.NoError(t, err)
	assert.InDelta(t, 450.0, aggValue, 0.001, "idempotent: second run yields same value")

	// 6) Query 路由：hourly × device → pm_metrics_hourly
	rows, err := a.Query(ctx, QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDevice,
		DeviceOUIs:  []string{testOUI},
		DeviceSNs:   []string{testSN},
		StartTime:   bucketStart,
		EndTime:     bucketEnd,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.Equal(t, path, rows[0].MetricPath)
	assert.InDelta(t, 450.0, rows[0].MetricValue, 0.001)
}

// Test runner via asyncjob payload — 验证 Runner.Run 能完整跑通 hourly pipeline
func Test_Integration_HourlyRunner_PayloadDrivesPipeline(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	const (
		testOUI = "INTTST"
		testSN  = "AGGR-RUNNER-TEST-001"
		path    = "L.Cell.Throughput.Sum"
	)
	bucketStart := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(time.Hour)

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_hourly WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
	}
	cleanup()
	defer cleanup()

	// 注入 2 个 15min 行：50 + 75 = 125
	for i, v := range []float64{50, 75} {
		endTime := bucketStart.Add(time.Duration(i+1) * 15 * time.Minute)
		startTime := endTime.Add(-15 * time.Minute)
		_, err := pool.Exec(ctx, `
INSERT INTO pm_metrics (device_oui, device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time)
VALUES ($1, $2, $3, 'counter', $4, 'sum', '15min', $5, $6, $5)`,
			testOUI, testSN, path, v, endTime, startTime)
		require.NoError(t, err)
	}

	a := NewWithPool(pool, nil, nil)
	runner := NewHourlyRunner(a)

	payload, err := BuildPayload(bucketStart, bucketEnd)
	require.NoError(t, err)

	// 走 Runner.Run（模拟 asyncjob 调度）
	job := &asyncjob.Job{ID: uuid.New(), JobType: runner.JobType(), Payload: payload}
	result, err := runner.Run(ctx, job)
	require.NoError(t, err)
	assert.Contains(t, string(result), `"counter_rows"`)

	var aggValue float64
	err = pool.QueryRow(ctx, `
SELECT metric_value FROM pm_metrics_hourly
WHERE device_oui=$1 AND device_sn=$2 AND metric_path=$3`,
		testOUI, testSN, path).Scan(&aggValue)
	require.NoError(t, err)
	assert.InDelta(t, 125.0, aggValue, 0.001)
}
