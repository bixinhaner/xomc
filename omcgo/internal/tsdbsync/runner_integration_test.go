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

func TestIntegrationSyncMetricDictionaryEnrichesConfiguredAndUnknownPaths(t *testing.T) {
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
		VALUES ('KSYNCINT','vendor.sync.key','Sync KPI','同步指标','integration','0','%','pct')
		ON CONFLICT (id) DO UPDATE SET
		  report_key=EXCLUDED.report_key,cn_name=EXCLUDED.cn_name,
		  unit_id=EXCLUDED.unit_id,statis_type=EXCLUDED.statis_type`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO pm_metric_dictionary (metric_path,report_key,metric_type)
		VALUES ('vendor.sync.key','vendor.sync.key','counter')
		ON CONFLICT (metric_path) DO NOTHING`)
	require.NoError(t, err)

	runner := NewSyncRunner(pool, pool, time.Minute, nil)
	_, err = runner.syncMetricDictionary(ctx)
	require.NoError(t, err)

	for _, path := range []string{"KSYNCINT", "vendor.sync.key"} {
		var name, unit, statis, metricType string
		err = pool.QueryRow(ctx, `
			SELECT metric_name,unit,statis_type,metric_type
			  FROM pm_metric_dictionary WHERE metric_path=$1`, path).
			Scan(&name, &unit, &statis, &metricType)
		require.NoError(t, err)
		require.Equal(t, "同步指标", name)
		require.Equal(t, "%", unit)
		require.Equal(t, "pct", statis)
		require.Equal(t, "kpi", metricType)
	}
}
