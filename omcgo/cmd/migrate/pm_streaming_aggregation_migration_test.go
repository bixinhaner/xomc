package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPMStreamingAggregationMigrationContract(t *testing.T) {
	mainSQL := readMigration(t, filepath.Join("..", "..", "migrations", "000001_init_schema.sql"))
	tsdbSQL := readMigration(t, filepath.Join("..", "..", "migrations", "tsdb", "000001_tsdb_schema.sql"))
	composeSQL := readMigration(t, filepath.Join("..", "..", "..", "deployments", "docker", "docker-compose.yml"))
	dockerfile := readMigration(t, filepath.Join("..", "..", "..", "deployments", "docker", "Dockerfile.app"))

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
		"pm_aggregation_rollup_outbox",
		"pm_aggregation_counter_rollups",
		"pm_aggregation_windows",
		"pm_aggregation_publications",
		"pm_aggregation_results",
	} {
		require.Contains(t, tsdbSQL, "CREATE TABLE public."+table)
	}
	require.Contains(t, tsdbSQL, "CREATE UNIQUE INDEX uq_pm_aggregation_results_business")
	require.Contains(t, tsdbSQL, "PRIMARY KEY (task_version_id, granularity, window_start)")
	require.Contains(t, tsdbSQL, "status IN ('preparing', 'published')")
	require.Contains(t, tsdbSQL, "idx_pm_aggregation_publications_due")
	require.Contains(t, tsdbSQL, "idx_pm_windows_prepared_publication")
	require.Contains(t, tsdbSQL, "idx_pm_rollup_outbox_publication")
	require.Contains(t, tsdbSQL, "publication_task_version_id uuid NOT NULL")
	require.Contains(t, tsdbSQL, "ADD COLUMN published_revision integer NOT NULL DEFAULT 0")
	require.Contains(t, tsdbSQL, "publication_eligible boolean NOT NULL DEFAULT true")
	require.GreaterOrEqual(t, strings.Count(tsdbSQL, "JOIN public.pm_aggregation_publications published_window"), 10,
		"every compatibility result view must pin the atomically published revision")
	require.GreaterOrEqual(t, strings.Count(tsdbSQL, "published_window.revision = r.revision"), 10,
		"every compatibility result view must hide preparing revisions")
	require.Contains(t, tsdbSQL, "'prepared'")
	for _, fragment := range []string{
		"ADD COLUMN finalize_lease_owner uuid",
		"ADD COLUMN finalize_lease_until timestamptz",
		"ADD COLUMN finalize_attempts integer NOT NULL DEFAULT 0",
		"ADD COLUMN finalize_next_attempt_at timestamptz NOT NULL DEFAULT '-infinity'",
		"ADD COLUMN version_audit_fingerprint text",
		"CREATE INDEX idx_pm_windows_due_claim",
		"CREATE INDEX idx_pm_windows_oldest_due",
		"CREATE INDEX idx_pm_windows_version_audit",
		"granularity, window_end, task_version_id, entity_key, window_start",
		"ADD COLUMN device_id uuid",
		"CREATE INDEX idx_pm_aggregation_outbox_device_period_replay",
		"device_id, event_window_start, event_id",
		"CREATE INDEX idx_pm_replay_sources_device_period",
		"timescaledb.compress_segmentby",
		"'device_id'",
	} {
		require.Contains(t, tsdbSQL, fragment)
	}
	require.Contains(t, tsdbSQL, "DROP TABLE IF EXISTS public.pm_metrics_daily")
	require.Contains(t, mainSQL, "DROP TABLE IF EXISTS public.pm_completion_watermarks")
	require.NotContains(t, strings.ToUpper(tsdbSQL), "INSERT INTO PUBLIC.PM_AGGREGATION_RESULTS SELECT")
	require.Contains(t, composeSQL, `"--reconcile", "/etc/omcgo/tsdb-schema-reconcile.sql"`)
	require.Contains(t, dockerfile, "COPY deployments/release/bundle/deploy/tsdb-schema-reconcile.sql")
}

func TestPMDeviceViewQueryIndexesMigrationContract(t *testing.T) {
	tsdbSQL := readMigration(t, filepath.Join("..", "..", "migrations", "tsdb", "000001_tsdb_schema.sql"))
	reconcileSQL := readMigration(t, filepath.Join("..", "..", "..", "deployments", "release", "bundle", "deploy", "tsdb-schema-reconcile.sql"))

	for _, sql := range []string{tsdbSQL, reconcileSQL} {
		require.Contains(t, sql, "idx_pm_anchors_15min_device_time_object")
		require.Contains(t, sql, "ON public.pm_measurement_anchors (device_dim_id, \"time\" DESC, object_ldn)")
		require.Contains(t, sql, "WHERE granularity = '15min'")
		require.Contains(t, sql, "idx_pm_anchors_15min_device_object")
		require.Contains(t, sql, "ON public.pm_measurement_anchors (device_dim_id, object_ldn)")
		require.Contains(t, sql, "idx_pm_anchors_15min_device_object_time")
		require.Contains(t, sql, "ON public.pm_measurement_anchors (device_dim_id, object_ldn, \"time\" DESC)")
		require.Contains(t, sql, "idx_pm_metric_values_anchor_metric_time")
		require.Contains(t, sql, "ON public.pm_metric_values (anchor_id, metric_id, \"time\" DESC)")
		require.Contains(t, sql, "idx_pm_aggregation_results_device_view")
		require.Contains(t, sql, "granularity,\n        dimension_key,\n        window_start DESC,\n        object_ldn,\n        metric_path,\n        task_version_id,\n        revision")
		require.Contains(t, sql, "idx_pm_aggregation_results_device_metric_time")
		require.Contains(t, sql, "granularity,\n        dimension_key,\n        metric_path,\n        window_start DESC,\n        object_ldn,\n        task_version_id,\n        revision")
		require.Contains(t, sql, "WHERE dimension = 'device'")
	}
}

func TestPMWindowVersionMetadataBackfillPendingIndexMigrationContract(t *testing.T) {
	tsdbSQL := readMigration(t, filepath.Join("..", "..", "migrations", "tsdb", "000001_tsdb_schema.sql"))

	for _, fragment := range []string{
		"CREATE INDEX IF NOT EXISTS idx_pm_aggregation_windows_version_backfill_pending",
		"task_version_id",
		"entity_key",
		"granularity",
		"window_start",
		"WHERE version_effective_from IS NULL",
	} {
		require.Contains(t, tsdbSQL, fragment)
	}
}

func TestPMBuiltinTaskRecoveryMigrationContract(t *testing.T) {
	mainSQL := readMigration(t, filepath.Join("..", "..", "migrations", "000001_init_schema.sql"))
	seedSQL := readMigration(t, filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql"))
	baselineSQL := readMigration(t, filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql"))
	testCompose := readMigration(t, filepath.Join("..", "..", "..", "deployments", "docker", "docker-compose.test.yml"))

	require.Contains(t, mainSQL, "ADD COLUMN content_hash bytea")
	require.Contains(t, seedSQL, ") ON CONFLICT DO NOTHING;")
	require.NotContains(t, baselineSQL, "pm_completion_watermarks")
	require.Contains(t, testCompose, `GOOSE_TABLE: "goose_db_version_seed"`)
	for _, id := range []string{
		"0184dddd-0001-4000-8000-000000000001",
		"0184dddd-0001-4000-8000-000000000002",
		"0184dddd-0001-4000-8000-000000000003",
		"0184dddd-0002-4000-8000-000000000001",
		"0184dddd-0002-4000-8000-000000000002",
		"0184dddd-0002-4000-8000-000000000003",
		"0184dddd-0003-4000-8000-000000000001",
		"0184dddd-0003-4000-8000-000000000002",
		"0184dddd-0003-4000-8000-000000000003",
		"0184dddd-0004-4000-8000-000000000001",
		"0184dddd-0004-4000-8000-000000000002",
		"0184dddd-0004-4000-8000-000000000003",
	} {
		require.Equalf(t, 1, countBuiltinPMTaskInsertID(t, seedSQL, id), "builtin task %s must be inserted exactly once", id)
	}
}

func TestPMBuiltinTaskMetricPathsMatchEnabledDefaults(t *testing.T) {
	seedSQL := readMigration(t, filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql"))
	enabledDefaults := map[string]map[string]struct{}{
		"lte": parseDefaultEnabledIndicators(t, seedSQL, "enabled_pm_indicators_enb"),
		"nr":  parseDefaultEnabledIndicators(t, seedSQL, "enabled_pm_indicators_gnb"),
		"gsm": parseDefaultEnabledIndicators(t, seedSQL, "enabled_pm_indicators_gsm"),
	}

	enbDefault := enabledDefaults["lte"]
	for _, dependency := range []string{
		"C000060216", "C000060273",
		"C000190005", "C000190006", "C000190007",
		"C000190008", "C000190009", "C000190010",
	} {
		require.Containsf(t, enbDefault, dependency,
			"clean install must explicitly enable dashboard KPI dependency %s", dependency)
	}
	require.Contains(t, enbDefault, "K900010076")
	require.Equal(t, 1, countDefaultEnabledIndicator(t, seedSQL, "enabled_pm_indicators_enb", "C000060216"))
	require.Equal(t, 1, countDefaultEnabledIndicator(t, seedSQL, "enabled_pm_indicators_enb", "C000060273"))

	tasks := parseBuiltinPMTasks(t, seedSQL)
	require.Len(t, tasks, 12)
	for _, task := range tasks {
		enabled, ok := enabledDefaults[task.technology]
		require.Truef(t, ok, "builtin task %s uses unsupported technology %q", task.name, task.technology)
		for _, metricPath := range task.metricPaths {
			require.Containsf(t, enabled, metricPath, "builtin task %s (%s) references metric %s outside %s default enabled list",
				task.name, task.technology, metricPath, task.technology)
		}
		if task.technology == "lte" {
			require.Contains(t, task.metricPaths, "K900010076",
				"LTE builtin task must output the availability KPI")
			require.NotContains(t, task.metricPaths, "C000060216",
				"formula dependency must not replace the availability KPI output")
		}
	}
}

func TestPMEnabledIndicatorDependencyClosureRepairMigrationContract(t *testing.T) {
	seedSQL := readMigration(t, filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql"))

	for _, suffix := range []string{"enb", "gnb", "gsm"} {
		require.Contains(t, seedSQL, "enabled_pm_indicators_"+suffix)
		require.Contains(t, seedSQL, "perf_indicators_"+suffix)
	}
	require.Contains(t, seedSQL, "WITH RECURSIVE dependency_closure")
	require.Contains(t, seedSQL, "ON CONFLICT (operator_code, indicator_id) DO NOTHING")
	require.Contains(t, seedSQL, "K900010076")
	require.Contains(t, seedSQL, "C000060216")
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

type builtinPMTask struct {
	name        string
	technology  string
	metricPaths []string
}

func parseDefaultEnabledIndicators(t *testing.T, seedSQL, table string) map[string]struct{} {
	t.Helper()
	ids := parseDefaultEnabledIndicatorIDs(t, seedSQL, table)
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func countDefaultEnabledIndicator(t *testing.T, seedSQL, table, indicatorID string) int {
	t.Helper()
	count := 0
	for _, id := range parseDefaultEnabledIndicatorIDs(t, seedSQL, table) {
		if id == indicatorID {
			count++
		}
	}
	return count
}

func countBuiltinPMTaskInsertID(t *testing.T, seedSQL, id string) int {
	t.Helper()
	insertMarker := "INSERT INTO public.pm_tasks ("
	insertStart := strings.Index(seedSQL, insertMarker)
	require.NotEqual(t, -1, insertStart, "missing pm_tasks seed insert")
	insertEnd := strings.Index(seedSQL[insertStart:], ";\n")
	require.NotEqual(t, -1, insertEnd, "unterminated pm_tasks seed insert")
	return strings.Count(seedSQL[insertStart:insertStart+insertEnd], id)
}

func parseDefaultEnabledIndicatorIDs(t *testing.T, seedSQL, table string) []string {
	t.Helper()
	insertMarker := "INSERT INTO public." + table + " (operator_code, indicator_id)"
	insertStart := strings.Index(seedSQL, insertMarker)
	require.NotEqualf(t, -1, insertStart, "missing %s default seed insert", table)

	block := seedSQL[insertStart:]
	require.Containsf(t, block, "SELECT 'default', indicator_id", "%s seed must populate the default operator", table)

	idsStartMarker := "FROM regexp_split_to_table($ids$\n"
	idsStart := strings.Index(block, idsStartMarker)
	require.NotEqualf(t, -1, idsStart, "missing %s $ids$ start", table)
	idsStart += len(idsStartMarker)

	idsEnd := strings.Index(block[idsStart:], "\n$ids$")
	require.NotEqualf(t, -1, idsEnd, "missing %s $ids$ end", table)
	return strings.Fields(block[idsStart : idsStart+idsEnd])
}

func parseBuiltinPMTasks(t *testing.T, seedSQL string) []builtinPMTask {
	t.Helper()
	insertMarker := "INSERT INTO public.pm_tasks ("
	insertStart := strings.Index(seedSQL, insertMarker)
	require.NotEqual(t, -1, insertStart, "missing pm_tasks seed insert")

	block := seedSQL[insertStart:]
	valuesStart := strings.Index(block, ") VALUES\n")
	require.NotEqual(t, -1, valuesStart, "missing pm_tasks VALUES")
	valuesStart += len(") VALUES\n")
	valuesEnd := strings.Index(block[valuesStart:], " ON CONFLICT DO NOTHING;")
	require.NotEqual(t, -1, valuesEnd, "missing pm_tasks ON CONFLICT")

	var tasks []builtinPMTask
	for _, line := range strings.Split(block[valuesStart:valuesStart+valuesEnd], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		row := strings.TrimSuffix(line, ",")
		row = strings.TrimPrefix(row, "(")
		row = strings.TrimSuffix(row, ")")
		fields := splitSQLValues(t, row)
		require.Len(t, fields, 25)
		if unquoteSQLValue(fields[12]) != "adhoc_aggregation" || fields[22] != "true" {
			continue
		}
		tasks = append(tasks, builtinPMTask{
			name:        unquoteSQLValue(fields[1]),
			technology:  unquoteSQLValue(fields[21]),
			metricPaths: parsePGTextArray(unquoteSQLValue(fields[15])),
		})
	}
	return tasks
}

func splitSQLValues(t *testing.T, row string) []string {
	t.Helper()
	var fields []string
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(row); i++ {
		ch := row[i]
		if ch == '\'' {
			current.WriteByte(ch)
			if inQuote && i+1 < len(row) && row[i+1] == '\'' {
				i++
				current.WriteByte(row[i])
				continue
			}
			inQuote = !inQuote
			continue
		}
		if ch == ',' && !inQuote {
			fields = append(fields, strings.TrimSpace(current.String()))
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}
	require.False(t, inQuote, "unterminated SQL string in pm_tasks row")
	fields = append(fields, strings.TrimSpace(current.String()))
	return fields
}

func unquoteSQLValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
		value = strings.ReplaceAll(value, "''", "'")
	}
	return value
}

func parsePGTextArray(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "{")
	value = strings.TrimSuffix(value, "}")
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
