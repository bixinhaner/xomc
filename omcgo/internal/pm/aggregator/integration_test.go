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
	assert.InDelta(t, 450.0, float64(rows[0].MetricValue), 0.001)
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

// #31/#32/#33：完整业务日必须包含 24 个 hourly 源桶；sum/avg 结果与上海本地
// [00:00, 24:00) 桶边界必须同时正确。历史独立 daily cron 会抢在最后一个 hourly
// 桶完成前执行，固定漏最后一小时；本测试锁住链式触发后的最终聚合口径。
func Test_Integration_DailyAggregateCounters_CompleteBusinessDay(t *testing.T) {
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
		testOUI = "ISSUE31"
		testSN  = "ISSUE31-33-DAILY-001"
		sumPath = "C010070002"
		avgPath = "C010070004"
	)
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	localStart := time.Date(2026, 7, 10, 0, 0, 0, 0, loc)
	localEnd := localStart.AddDate(0, 0, 1)
	bucketStart := localStart.UTC()
	bucketEnd := localEnd.UTC()

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_hourly WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_daily WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
	}
	cleanup()
	defer cleanup()

	for i := 0; i < 24; i++ {
		start := bucketStart.Add(time.Duration(i) * time.Hour)
		end := start.Add(time.Hour)
		for _, metric := range []struct {
			path   string
			statis string
		}{
			{path: sumPath, statis: "sum"},
			{path: avgPath, statis: "avg"},
		} {
			_, err = pool.Exec(ctx, `
INSERT INTO pm_metrics_hourly
    (device_oui, device_sn, metric_path, metric_type, metric_value, statis_type,
     granularity, time, start_time, end_time, object_ldn)
VALUES ($1, $2, $3, 'counter', $4, $5, 'hourly', $6, $6, $7, '')`,
				testOUI, testSN, metric.path, float64(i+1), metric.statis, start, end)
			require.NoError(t, err)
		}
	}

	a := NewWithPool(pool, nil, nil)
	w := WindowSpec{Granularity: metrics.GranularityDaily, Start: bucketStart, End: bucketEnd}
	n, err := a.AggregateCounters(ctx, "pm_metrics_hourly", "pm_metrics_daily", w)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	type dailyRow struct {
		value            float64
		start, end, time time.Time
	}
	rows := make(map[string]dailyRow, 2)
	dbRows, err := pool.Query(ctx, `
SELECT metric_path, metric_value, start_time, end_time, time
FROM pm_metrics_daily
WHERE device_oui=$1 AND device_sn=$2
ORDER BY metric_path`, testOUI, testSN)
	require.NoError(t, err)
	defer dbRows.Close()
	for dbRows.Next() {
		var path string
		var row dailyRow
		require.NoError(t, dbRows.Scan(&path, &row.value, &row.start, &row.end, &row.time))
		rows[path] = row
	}
	require.NoError(t, dbRows.Err())
	require.Len(t, rows, 2)

	assert.InDelta(t, 300.0, rows[sumPath].value, 0.001, "sum(1..24) must include the last hourly bucket")
	assert.InDelta(t, 12.5, rows[avgPath].value, 0.001, "avg(1..24) must include the last hourly bucket")
	for _, path := range []string{sumPath, avgPath} {
		row := rows[path]
		assert.True(t, row.start.Equal(bucketStart), "%s start=%s", path, row.start)
		assert.True(t, row.time.Equal(bucketStart), "%s time=%s", path, row.time)
		assert.True(t, row.end.Equal(bucketEnd), "%s end=%s", path, row.end)
		assert.Equal(t, 0, row.start.In(loc).Hour())
		assert.Equal(t, 0, row.end.In(loc).Hour())
		assert.Equal(t, localStart.Day(), row.start.In(loc).Day())
		assert.Equal(t, localEnd.Day(), row.end.In(loc).Day())
	}
}
