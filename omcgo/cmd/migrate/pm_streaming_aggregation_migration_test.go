package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPMStreamingAggregationMigrationContract(t *testing.T) {
	mainSQL := readMigration(t, filepath.Join("..", "..", "migrations", "000006_pm_streaming_aggregation.sql"))
	tsdbSQL := readMigration(t, filepath.Join("..", "..", "migrations", "tsdb", "000002_pm_streaming_aggregation.sql"))

	for _, table := range []string{
		"pm_aggregation_tasks",
		"pm_aggregation_task_versions",
		"pm_aggregation_version_metrics",
		"pm_aggregation_version_members",
	} {
		require.Contains(t, mainSQL, "CREATE TABLE public."+table)
	}
	for _, table := range []string{
		"pm_aggregation_outbox",
		"pm_aggregation_windows",
		"pm_aggregation_results",
	} {
		require.Contains(t, tsdbSQL, "CREATE TABLE public."+table)
	}
	require.Contains(t, tsdbSQL, "CREATE UNIQUE INDEX uq_pm_aggregation_results_business")
	require.Contains(t, tsdbSQL, "DROP TABLE IF EXISTS public.pm_metrics_daily")
	require.Contains(t, mainSQL, "DROP TABLE IF EXISTS public.pm_completion_watermarks")
	require.NotContains(t, strings.ToUpper(tsdbSQL), "INSERT INTO PUBLIC.PM_AGGREGATION_RESULTS SELECT")
}

func TestStreamingWorkerDoesNotReferenceRawPMTables(t *testing.T) {
	streamDir := filepath.Join("..", "..", "internal", "pm", "stream")
	entries, err := os.ReadDir(streamDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(streamDir, entry.Name()))
		require.NoError(t, readErr)
		source := strings.ToLower(string(data))
		require.NotContains(t, source, "pm_measurement_anchors", entry.Name())
		require.NotContains(t, source, "pm_metric_values", entry.Name())
		require.NotContains(t, source, "from pm_metrics", entry.Name())
	}
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
