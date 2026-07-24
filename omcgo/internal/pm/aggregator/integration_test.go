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

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func Test_Integration_IndicatorMeta_DictionaryPriorityAndLegacyFallback(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	for _, ddl := range []string{
		`CREATE TEMP TABLE perf_indicators_enb (id text, unit_id text, statis_type text) ON COMMIT DROP`,
		`CREATE TEMP TABLE perf_indicators_gnb (id text, unit_id text, statis_type text) ON COMMIT DROP`,
		`CREATE TEMP TABLE perf_indicators_gsm (id text, unit_id text, statis_type text) ON COMMIT DROP`,
		`CREATE TEMP TABLE pm_metric_dictionary (
			metric_path text PRIMARY KEY, unit text, statis_type text, metric_type text
		) ON COMMIT DROP`,
	} {
		_, err = tx.Exec(ctx, ddl)
		require.NoError(t, err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO perf_indicators_enb (id, unit_id, statis_type) VALUES
    ('same', 'number', 'sum'),
    ('conflict', 'number', 'sum'),
    ('fallback', 'number', 'sum'),
    ('legacy-complete', NULL, NULL);
INSERT INTO perf_indicators_gnb (id, unit_id, statis_type) VALUES
    ('legacy-complete', 'number', 'max');
INSERT INTO pm_metric_dictionary (metric_path, unit, statis_type, metric_type) VALUES
    ('dictionary-only', 'number', 'sum', 'counter'),
    ('same', 'number', 'sum', 'counter'),
    ('conflict', '%', 'pct', 'counter'),
    ('fallback', '', 'avg', 'counter'),
    ('ignored-kpi', '%', 'pct', 'kpi')`)
	require.NoError(t, err)

	rows, err := tx.Query(ctx, "WITH "+indicatorMetaSQL()+`
SELECT id, unit_id, statis_type FROM indicator_meta ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	type meta struct{ unit, statis string }
	got := make(map[string]meta)
	for rows.Next() {
		var id, unit, statis string
		require.NoError(t, rows.Scan(&id, &unit, &statis))
		got[id] = meta{unit: unit, statis: statis}
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]meta{
		"conflict":        {unit: "%", statis: "pct"},
		"dictionary-only": {unit: "number", statis: "sum"},
		"fallback":        {unit: "number", statis: "avg"},
		"legacy-complete": {unit: "number", statis: "max"},
		"same":            {unit: "number", statis: "sum"},
	}, got)
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
		testOUI = "ISSU31"
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
	require.NoError(t, ensureAggregationMetadata(ctx, pool, sumPath, "sum"))
	require.NoError(t, ensureAggregationMetadata(ctx, pool, avgPath, "avg"))

	cleanup := func() {
		cleanupVersionedHourlyDevice(ctx, pool, testOUI, testSN)
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
			require.NoError(t, insertVersionedHourlyMetric(
				ctx, pool, testOUI, testSN, metric.path, metric.statis, float64(i+1), start, end))
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

func insertVersionedHourlyMetric(
	ctx context.Context,
	pool *pgxpool.Pool,
	oui, sn, path, statis string,
	value float64,
	start, end time.Time,
) error {
	deviceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(oui+"\x00"+sn))
	if _, err := pool.Exec(ctx, `
		INSERT INTO device_dim (id,oui,serial_number)
		VALUES ($1,$2,$3)
		ON CONFLICT (id) DO UPDATE SET oui=EXCLUDED.oui, serial_number=EXCLUDED.serial_number`,
		deviceID, oui, sn); err != nil {
		return err
	}
	var metricID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,metric_type,statis_type)
		VALUES ($1,'counter',$2)
		ON CONFLICT (metric_path) DO UPDATE SET statis_type=EXCLUDED.statis_type
		RETURNING metric_id`, path, statis).Scan(&metricID); err != nil {
		return err
	}
	var setID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO pm_metric_sets (product_key,counter_group,content_hash,metric_ids)
		VALUES ('integration-hourly','integration',decode(md5($1::bigint::text),'hex'),ARRAY[$1::bigint])
		ON CONFLICT (product_key,counter_group,content_hash)
		DO UPDATE SET metric_ids=EXCLUDED.metric_ids
		RETURNING metric_set_id`, metricID).Scan(&setID); err != nil {
		return err
	}
	var version int64
	err := pool.QueryRow(ctx,
		`SELECT bucket_version FROM pm_hourly_bucket_versions WHERE bucket_start=$1 AND status='active'`,
		start).Scan(&version)
	if err != nil {
		if err := pool.QueryRow(ctx, `
			INSERT INTO pm_hourly_bucket_versions
			    (bucket_start,bucket_end,status,published_at)
			VALUES ($1,$2,'active',now()) RETURNING bucket_version`,
			start, end).Scan(&version); err != nil {
			return err
		}
	}
	var anchorID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO pm_hourly_anchors
		    ("time",bucket_version,device_dim_id,object_ldn,counter_group,metric_set_id,
		     granularity,start_time,end_time)
		VALUES ($1,$2,$3,'','integration',$4,'hourly',$1,$5)
		RETURNING anchor_id`,
		start, version, deviceID, setID, end).Scan(&anchorID); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_hourly_values ("time",bucket_version,anchor_id,metric_id,metric_value)
		VALUES ($1,$2,$3,$4,$5)`, start, version, anchorID, metricID, value)
	return err
}

func cleanupVersionedHourlyDevice(ctx context.Context, pool *pgxpool.Pool, oui, sn string) {
	deviceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(oui+"\x00"+sn))
	_, _ = pool.Exec(ctx, `
		WITH target_versions AS MATERIALIZED (
		    SELECT DISTINCT bucket_version
		      FROM pm_hourly_anchors
		     WHERE device_dim_id=$1
		),
		deleted_values AS (
		    DELETE FROM pm_hourly_values v USING pm_hourly_anchors a
		     WHERE a.device_dim_id=$1
		       AND v."time"=a."time" AND v.anchor_id=a.anchor_id
		),
		deleted_anchors AS (
		    DELETE FROM pm_hourly_anchors WHERE device_dim_id=$1
		),
		deleted_batches AS (
		    DELETE FROM pm_hourly_rollup_batches b USING target_versions t
		     WHERE b.bucket_version=t.bucket_version
		       AND NOT EXISTS (
		           SELECT 1 FROM pm_hourly_anchors a
		            WHERE a.bucket_version=t.bucket_version
		       )
		)
		DELETE FROM pm_hourly_bucket_versions v USING target_versions t
		 WHERE v.bucket_version=t.bucket_version
		   AND NOT EXISTS (
		       SELECT 1 FROM pm_hourly_anchors a
		        WHERE a.bucket_version=t.bucket_version
		   )`, deviceID)
}

// #30：七个完整业务日必须能卷成一条周数据；周桶按系统时区周一 00:00 对齐。
func Test_Integration_WeeklyAggregateCounters_CompleteBusinessWeek(t *testing.T) {
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
		testOUI = "ISSUE30"
		testSN  = "ISSUE30-WEEKLY-001"
		sumPath = "C010070002"
		avgPath = "C010070004"
	)
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	localStart := time.Date(2026, 5, 4, 0, 0, 0, 0, loc) // Monday；避开运行时验收周
	localEnd := localStart.AddDate(0, 0, 7)
	bucketStart := localStart.UTC()
	bucketEnd := localEnd.UTC()
	require.NoError(t, ensureAggregationMetadata(ctx, pool, sumPath, "sum"))
	require.NoError(t, ensureAggregationMetadata(ctx, pool, avgPath, "avg"))

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_daily WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_weekly WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
	}
	cleanup()
	defer cleanup()

	for i := 0; i < 7; i++ {
		start := bucketStart.AddDate(0, 0, i)
		end := start.AddDate(0, 0, 1)
		for _, metric := range []struct {
			path   string
			statis string
			value  float64
		}{
			{path: sumPath, statis: "sum", value: float64((i + 1) * 10)},
			{path: avgPath, statis: "avg", value: float64(i + 1)},
		} {
			_, err = pool.Exec(ctx, `
INSERT INTO pm_metrics_daily
    (device_oui, device_sn, metric_path, metric_type, metric_value, statis_type,
     granularity, time, start_time, end_time, object_ldn)
VALUES ($1, $2, $3, 'counter', $4, $5, 'daily', $6, $6, $7, '')`,
				testOUI, testSN, metric.path, metric.value, metric.statis, start, end)
			require.NoError(t, err)
		}
	}

	a := NewWithPool(pool, nil, nil)
	w := WindowSpec{Granularity: metrics.GranularityWeekly, Start: bucketStart, End: bucketEnd}
	n, err := a.AggregateCounters(ctx, "pm_metrics_daily", "pm_metrics_weekly", w)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	type weeklyRow struct {
		value            float64
		start, end, time time.Time
	}
	rows := make(map[string]weeklyRow, 2)
	dbRows, err := pool.Query(ctx, `
SELECT metric_path, metric_value, start_time, end_time, time
FROM pm_metrics_weekly
WHERE device_oui=$1 AND device_sn=$2
ORDER BY metric_path`, testOUI, testSN)
	require.NoError(t, err)
	defer dbRows.Close()
	for dbRows.Next() {
		var path string
		var row weeklyRow
		require.NoError(t, dbRows.Scan(&path, &row.value, &row.start, &row.end, &row.time))
		rows[path] = row
	}
	require.NoError(t, dbRows.Err())
	require.Len(t, rows, 2)

	assert.InDelta(t, 280.0, rows[sumPath].value, 0.001)
	assert.InDelta(t, 4.0, rows[avgPath].value, 0.001)
	for _, path := range []string{sumPath, avgPath} {
		row := rows[path]
		assert.True(t, row.start.Equal(bucketStart), "%s start=%s", path, row.start)
		assert.True(t, row.time.Equal(bucketStart), "%s time=%s", path, row.time)
		assert.True(t, row.end.Equal(bucketEnd), "%s end=%s", path, row.end)
		assert.Equal(t, time.Monday, row.start.In(loc).Weekday())
		assert.Equal(t, 0, row.start.In(loc).Hour())
		assert.Equal(t, time.Monday, row.end.In(loc).Weekday())
		assert.Equal(t, 0, row.end.In(loc).Hour())
	}
}

// #29：自定义查询范围采用 [start,end)，end 恰好等于下一桶桶头时只能返回首桶。
func Test_Integration_QueryCustomRange_ExcludesBucketAtEndBoundary(t *testing.T) {
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
		testOUI = "ISSUE29"
		testSN  = "ISSUE29-RANGE-001"
		path    = "C010070002"
	)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	require.NoError(t, ensureAggregationMetadata(ctx, pool, path, "sum"))
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_metrics_daily WHERE device_oui=$1 AND device_sn=$2`, testOUI, testSN)
	}
	cleanup()
	defer cleanup()

	for i, bucketStart := range []time.Time{start, end} {
		_, err = pool.Exec(ctx, `
INSERT INTO pm_metrics_daily
    (device_oui, device_sn, metric_path, metric_type, metric_value, statis_type,
     granularity, time, start_time, end_time, object_ldn)
VALUES ($1, $2, $3, 'counter', $4, 'sum', 'daily', $5, $5, $6, '')`,
			testOUI, testSN, path, float64(i+1), bucketStart, bucketStart.AddDate(0, 0, 1))
		require.NoError(t, err)
	}

	a := NewWithPool(pool, nil, nil)
	rows, err := a.Query(ctx, QueryRequest{
		Granularity: metrics.GranularityDaily,
		Dimension:   DimensionDevice,
		DeviceOUIs:  []string{testOUI},
		DeviceSNs:   []string{testSN},
		MetricPaths: []string{path},
		StartTime:   start,
		EndTime:     end,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].Time.Equal(start), "returned bucket head=%s", rows[0].Time)
	assert.InDelta(t, 1.0, float64(rows[0].MetricValue), 0.001)
}

func ensureAggregationMetadata(ctx context.Context, pool *pgxpool.Pool, path, statis string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO perf_indicators_enb
		    (id,report_key,en_name,cn_name,group_id,unit_id,is_counter,statis_type)
		VALUES ($1,$1,$1,$1,'integration','number','1',$2)
		ON CONFLICT (id) DO UPDATE
		    SET unit_id=EXCLUDED.unit_id, statis_type=EXCLUDED.statis_type`,
		path, statis)
	return err
}
