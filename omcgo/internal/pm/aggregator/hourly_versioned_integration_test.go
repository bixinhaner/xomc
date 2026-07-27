//go:build integration

package aggregator

import (
	"context"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/kpi/router"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestIntegrationVersionedHourlyPublishesSparseBucketAtomically(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	deviceID := uuid.New()
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration(binary.BigEndian.Uint32(deviceID[:4])%40000) * time.Hour)
	end := start.Add(time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO device_dim (id,oui,serial_number,technology,carrier)
		VALUES ($1,'INT001','SPARSE-HOURLY-001','lte','cmcc')`, deviceID)
	require.NoError(t, err)
	var metricID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,metric_type,statis_type,unit)
		VALUES ('CINTSUM','counter','sum','number')
		ON CONFLICT (metric_path) DO UPDATE
		SET statis_type=EXCLUDED.statis_type,unit=EXCLUDED.unit
		RETURNING metric_id`).Scan(&metricID)
	require.NoError(t, err)
	var avgMetricID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,metric_type,statis_type,unit)
		VALUES ('CINTAVG','counter','avg','%')
		ON CONFLICT (metric_path) DO UPDATE
		SET statis_type=EXCLUDED.statis_type,unit=EXCLUDED.unit
		RETURNING metric_id`).Scan(&avgMetricID)
	require.NoError(t, err)
	var setID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO pm_metric_sets (product_key,counter_group,content_hash,metric_ids)
		VALUES ('integration','INT',decode(md5($1::bigint::text||','||$2::bigint::text),'hex'),
		        ARRAY[$1::bigint,$2::bigint])
		ON CONFLICT (product_key,counter_group,content_hash)
		DO UPDATE SET metric_ids=EXCLUDED.metric_ids
		RETURNING metric_set_id`, metricID, avgMetricID).Scan(&setID)
	require.NoError(t, err)
	for i, value := range []float64{0.4, 0.4, 0.4, 0.4} {
		point := start.Add(time.Duration(i) * 15 * time.Minute)
		var anchorID int64
		err = pool.QueryRow(ctx, `
			INSERT INTO pm_measurement_anchors
			    ("time",device_dim_id,object_ldn,counter_group,metric_set_id,granularity,start_time,end_time)
			VALUES ($1,$2,'','INT',$3,'15min',$1,$4)
			RETURNING anchor_id`, point, deviceID, setID, point.Add(15*time.Minute)).Scan(&anchorID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO pm_metric_values ("time",anchor_id,metric_id,metric_value)
			VALUES ($1,$2,$3,$4),($1,$2,$5,$6)`,
			point, anchorID, metricID, value, avgMetricID,
			[]float64{10.111, 20.222, 30.333, 40.444}[i])
		require.NoError(t, err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_hourly_bucket_versions (bucket_start,bucket_end,status)
		VALUES ($1,$2,'building')`, start, end)
	require.NoError(t, err)

	a := NewWithPool(pool, &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		"SPARSE-HOURLY-001": {
			KPIs: []router.KPIDef{{
				IndicatorID: "KINTPCT",
				StatisType:  "pct",
				Unit:        "%",
				Formula:     "CINTSUM",
			}},
		},
	}}, nil)
	stats, err := a.RunHourlyVersioned(ctx, WindowSpec{
		Granularity: metrics.GranularityHourly, Start: start, End: end,
	}, 2)
	require.NoError(t, err)
	require.Equal(t, 1, stats.DeviceCount)
	require.Equal(t, int64(2), stats.AnchorCount)
	require.Equal(t, int64(3), stats.ValueCount)

	var value float64
	err = pool.QueryRow(ctx, `
		SELECT metric_value FROM pm_metrics_hourly
		 WHERE device_sn='SPARSE-HOURLY-001' AND metric_path='CINTSUM'
		   AND "time"=$1`, start).Scan(&value)
	require.NoError(t, err)
	require.Equal(t, float64(2), value)
	targetedRequest := QueryRequest{
		MetricPaths: []string{"CINTSUM"},
		DeviceSNs:   []string{"SPARSE-HOURLY-001"},
		StartTime:   start,
		EndTime:     end,
	}
	targeted := newRawAwareDeviceSelect(
		storage.Psql, "pm_metrics_hourly", targetedRequest, "COUNT(*)",
	)
	targeted = applyDeviceFilters(targeted, targetedRequest)
	targetedSQL, targetedArgs, err := targeted.ToSql()
	require.NoError(t, err)
	var targetedCount int
	err = pool.QueryRow(ctx, targetedSQL, targetedArgs...).Scan(&targetedCount)
	require.NoError(t, err)
	require.Equal(t, 1, targetedCount)
	err = pool.QueryRow(ctx, `
		SELECT metric_value FROM pm_metrics_hourly
		 WHERE device_sn='SPARSE-HOURLY-001' AND metric_path='CINTAVG'
		   AND "time"=$1`, start).Scan(&value)
	require.NoError(t, err)
	require.Equal(t, 25.28, value)
	err = pool.QueryRow(ctx, `
		SELECT metric_value FROM pm_metrics_hourly
		 WHERE device_sn='SPARSE-HOURLY-001' AND metric_path='KINTPCT'
		   AND "time"=$1`, start).Scan(&value)
	require.NoError(t, err)
	require.Equal(t, float64(2), value)

	var active int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM pm_hourly_bucket_versions
		 WHERE bucket_start=$1 AND status='active'`, start).Scan(&active)
	require.NoError(t, err)
	require.Equal(t, 1, active)
	var failed int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM pm_hourly_bucket_versions
		 WHERE bucket_start=$1 AND status='failed'`, start).Scan(&failed)
	require.NoError(t, err)
	require.Equal(t, 1, failed)

	require.NoError(t, sampleSparseState(ctx, pool, NewMetrics(nil)))
	_, err = eligibleChunks(ctx, pool, "pm_metric_values", DefaultLateDataWindow, true)
	require.NoError(t, err)
	require.NoError(t, cleanupObsoleteHourlyVersions(ctx, pool))
}

func TestIntegrationVersionedHourlyLockOrderAvoidsIngestDeadlock(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()
	rollupConfig, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	rollupConfig.MaxConns = 1
	rollupPool, err := pgxpool.NewWithConfig(ctx, rollupConfig)
	require.NoError(t, err)
	defer rollupPool.Close()

	deviceID := uuid.New()
	suffix := deviceID.String()[:8]
	sn := "LOCK-ORDER-" + suffix
	counterPath := "CLOCK-" + suffix
	formulaPath := "KLOCK-" + suffix
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration(binary.BigEndian.Uint32(deviceID[:4])%200000) * time.Hour)
	end := start.Add(time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO device_dim (id,oui,serial_number,technology,carrier)
		VALUES ($1,'INT003',$2,'lte','cmcc')`, deviceID, sn)
	require.NoError(t, err)
	var counterMetricID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,metric_type,statis_type,unit)
		VALUES ($1,'counter','sum','number')
		RETURNING metric_id`, counterPath).Scan(&counterMetricID)
	require.NoError(t, err)
	var sourceSetID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO pm_metric_sets (product_key,counter_group,content_hash,metric_ids)
		VALUES ($1,'LOCK',decode(md5($2),'hex'),ARRAY[$3::bigint])
		RETURNING metric_set_id`, "lock-order-"+suffix, counterPath, counterMetricID).Scan(&sourceSetID)
	require.NoError(t, err)
	var sourceAnchorID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO pm_measurement_anchors
		    ("time",device_dim_id,object_ldn,counter_group,metric_set_id,granularity,start_time,end_time)
		VALUES ($1,$2,'','LOCK',$3,'15min',$1,$4)
		RETURNING anchor_id`, start, deviceID, sourceSetID, start.Add(15*time.Minute)).Scan(&sourceAnchorID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_metric_values ("time",anchor_id,metric_id,metric_value)
		VALUES ($1,$2,$3,5)`, start, sourceAnchorID, counterMetricID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_hourly_bucket_versions (bucket_start,bucket_end,status,published_at)
		VALUES ($1,$2,'active',now())`, start, end)
	require.NoError(t, err)

	ingestConn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer ingestConn.Release()
	ingestTx, err := ingestConn.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = ingestTx.Rollback(context.Background()) }()
	// Recreate the former ingestion half-cycle deliberately: hold dictionary/set/
	// value locks first, then request the bucket lock only after rollup is waiting.
	// The fixed rollup must wait during pre-transaction dictionary preparation,
	// so it cannot already hold the bucket lock needed to close the deadlock cycle.
	var formulaMetricID int64
	err = ingestTx.QueryRow(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,report_key,metric_type,statis_type,unit)
		VALUES ($1,$1,'kpi','pct','%')
		RETURNING metric_id`, formulaPath).Scan(&formulaMetricID)
	require.NoError(t, err)
	var ingestSetID int64
	err = ingestTx.QueryRow(ctx, `
		INSERT INTO pm_metric_sets (product_key,counter_group,content_hash,metric_ids)
		VALUES ($1,'__kpi__',decode(md5($2),'hex'),ARRAY[$3::bigint])
		RETURNING metric_set_id`, "ingest-lock-order-"+suffix, formulaPath, formulaMetricID).Scan(&ingestSetID)
	require.NoError(t, err)
	var ingestAnchorID int64
	err = ingestTx.QueryRow(ctx, `
		INSERT INTO pm_measurement_anchors
		    ("time",device_dim_id,object_ldn,counter_group,metric_set_id,granularity,start_time,end_time)
		VALUES ($1,$2,'','__kpi__',$3,'15min',$1,$4)
		RETURNING anchor_id`,
		start, deviceID, ingestSetID, start.Add(15*time.Minute)).Scan(&ingestAnchorID)
	require.NoError(t, err)
	_, err = ingestTx.Exec(ctx, `
		INSERT INTO pm_metric_values ("time",anchor_id,metric_id,metric_value)
		VALUES ($1,$2,$3,5)`, start, ingestAnchorID, formulaMetricID)
	require.NoError(t, err)

	probe, err := rollupPool.Acquire(ctx)
	require.NoError(t, err)
	var rollupPID int32
	require.NoError(t, probe.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&rollupPID))
	probe.Release()

	a := NewWithPool(rollupPool, &stubKPIRouter{byDevice: map[string]*router.KPIRoute{
		sn: {
			KPIs: []router.KPIDef{{
				IndicatorID: formulaPath, StatisType: "pct", Unit: "%", Formula: counterPath,
			}},
		},
	}}, nil)
	type rollupResult struct {
		stats RollupStats
		err   error
	}
	resultCh := make(chan rollupResult, 1)
	go func() {
		stats, runErr := a.RunHourlyVersioned(ctx, WindowSpec{
			Granularity: metrics.GranularityHourly, Start: start, End: end,
		}, 1)
		resultCh <- rollupResult{stats: stats, err: runErr}
	}()

	require.Eventually(t, func() bool {
		var waiting bool
		queryErr := pool.QueryRow(ctx, `
			SELECT COALESCE(wait_event_type='Lock',false)
			  FROM pg_stat_activity
			 WHERE pid=$1`, rollupPID).Scan(&waiting)
		return queryErr == nil && waiting
	}, 5*time.Second, 20*time.Millisecond,
		"rollup must reach its pre-transaction dictionary barrier while ingestion holds the dictionary lock")
	_, err = ingestTx.Exec(ctx, `
		UPDATE pm_hourly_bucket_versions
		   SET dirty=true
		 WHERE bucket_start=$1 AND status IN ('active','building')`, start)
	require.NoError(t, err,
		"rollup must still be outside its bucket/version transaction while dictionary preparation waits")
	require.NoError(t, ingestTx.Commit(ctx))

	select {
	case result := <-resultCh:
		require.Error(t, result.err, "late ingestion must prevent publishing the now-dirty building version")
		require.NotContains(t, result.err.Error(), "deadlock detected")
		require.Equal(t, 1, result.stats.DeviceCount)
	case <-ctx.Done():
		t.Fatalf("concurrent ingestion and hourly rollup did not finish before deadline: %v", ctx.Err())
	}
	var active int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM pm_hourly_bucket_versions
		 WHERE bucket_start=$1 AND status='active'`, start).Scan(&active)
	require.NoError(t, err)
	require.Equal(t, 1, active)
}
