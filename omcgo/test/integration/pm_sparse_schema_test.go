package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTSDBBaselineDefinesSparsePMStorage(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "migrations", "tsdb", "000001_tsdb_schema.sql"))
	require.NoError(t, err)
	sql := string(raw)

	for _, object := range []string{
		"pm_metric_dictionary", "pm_metric_sets", "pm_ingest_batches",
		"pm_measurement_anchors", "pm_metric_values",
		"pm_hourly_bucket_versions", "pm_hourly_rollup_batches",
		"pm_hourly_anchors", "pm_hourly_values",
	} {
		require.Contains(t, sql, "CREATE TABLE public."+object)
	}
	require.Contains(t, sql, "CREATE VIEW public.pm_metrics AS")
	require.Contains(t, sql, "CREATE VIEW public.pm_metrics_hourly AS")
	require.NotContains(t, sql, "CREATE TABLE public.pm_metrics (")
	require.NotContains(t, sql, "CREATE TABLE public.pm_metrics_hourly (")
	require.Contains(t, sql, "add_retention_policy('public.pm_metric_values', INTERVAL '30 days')")
	require.Contains(t, sql, "add_retention_policy('public.pm_measurement_anchors', INTERVAL '30 days')")
}
