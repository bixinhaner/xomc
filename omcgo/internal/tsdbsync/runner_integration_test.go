//go:build integration

package tsdbsync

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSyncMetricDictionaryDoesNotPromoteKPIReportKeyToKPI(t *testing.T) {
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		INSERT INTO perf_indicators_enb
		    (id,report_key,en_name,cn_name,group_id,is_counter,unit_id,statis_type)
		VALUES
		  ('KSYNCINT','vendor.sync.key','Sync KPI','同步指标','integration','0','%','pct'),
		  ('CSYNCINT','vendor.counter.key','Sync Counter','同步计数器','integration','1','number','sum')
		ON CONFLICT (id) DO UPDATE SET
		  report_key=EXCLUDED.report_key,cn_name=EXCLUDED.cn_name,
		  unit_id=EXCLUDED.unit_id,statis_type=EXCLUDED.statis_type,
		  is_counter=EXCLUDED.is_counter`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,report_key,metric_type)
		VALUES
		  ('vendor.sync.key','vendor.sync.key','counter'),
		  ('vendor.counter.key','vendor.counter.key','counter')
		ON CONFLICT (metric_path) DO NOTHING`)
	require.NoError(t, err)

	runner := NewSyncRunner(pool, pool, time.Minute, nil)
	_, err = runner.syncMetricDictionary(ctx)
	require.NoError(t, err)

	var name, unit, statis, metricType string
	err = pool.QueryRow(ctx, `
		SELECT metric_name,unit,statis_type,metric_type
		  FROM pm_metric_dictionary WHERE metric_path='KSYNCINT'`).
		Scan(&name, &unit, &statis, &metricType)
	require.NoError(t, err)
	require.Equal(t, "同步指标", name)
	require.Equal(t, "%", unit)
	require.Equal(t, "pct", statis)
	require.Equal(t, "kpi", metricType)

	var reportKeyType string
	err = pool.QueryRow(ctx, `
		SELECT metric_type
		  FROM pm_metric_dictionary WHERE metric_path='vendor.sync.key'`).
		Scan(&reportKeyType)
	require.NoError(t, err)
	require.Equal(t, "counter", reportKeyType)

	err = pool.QueryRow(ctx, `
		SELECT metric_name,unit,statis_type,metric_type
		  FROM pm_metric_dictionary WHERE metric_path='vendor.counter.key'`).
		Scan(&name, &unit, &statis, &metricType)
	require.NoError(t, err)
	require.Equal(t, "同步计数器", name)
	require.Equal(t, "number", unit)
	require.Equal(t, "sum", statis)
	require.Equal(t, "counter", metricType)
}
