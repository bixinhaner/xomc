package tsdbsync

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMirrorMergeSQLUsesIncrementalUpsertAndDelete(t *testing.T) {
	upsert, prune, err := buildMirrorMergeSQL(
		"alarm_definition_dim",
		"sync_stage_alarm_definition_dim",
		[]string{"id", "cn_name", "severity"},
		[]string{"id"},
	)
	require.NoError(t, err)
	assert.Contains(t, upsert, `ON CONFLICT ("id") DO UPDATE`)
	assert.Contains(t, upsert, `IS DISTINCT FROM`)
	assert.NotContains(t, strings.ToUpper(upsert), "TRUNCATE")
	assert.Contains(t, prune, `DELETE FROM "alarm_definition_dim"`)
	assert.Contains(t, prune, `d."id" = s."id"`)
	assert.NotContains(t, prune, `IS NOT DISTINCT FROM`)
}

func TestBuildMirrorMergeSQLSupportsCompositePrimaryKey(t *testing.T) {
	upsert, prune, err := buildMirrorMergeSQL(
		"device_group_member_dim",
		"sync_stage_device_group_member_dim",
		[]string{"group_id", "device_id"},
		[]string{"group_id", "device_id"},
	)
	require.NoError(t, err)
	assert.Contains(t, upsert, `ON CONFLICT ("group_id", "device_id") DO NOTHING`)
	assert.Contains(t, prune, `d."group_id" = s."group_id"`)
	assert.Contains(t, prune, `d."device_id" = s."device_id"`)
}

func TestBuildStageIndexSQLUsesCompositePrimaryKey(t *testing.T) {
	sql, err := buildStageIndexSQL(
		"sync_stage_device_group_member_dim",
		[]string{"group_id", "device_id"},
	)
	require.NoError(t, err)
	assert.Contains(t, sql, `CREATE UNIQUE INDEX`)
	assert.Contains(t, sql, `ON "sync_stage_device_group_member_dim" ("group_id", "device_id")`)
}

func TestBuildMirrorMergeSQLRejectsMissingKeyColumn(t *testing.T) {
	_, _, err := buildMirrorMergeSQL("dst", "stage", []string{"name"}, []string{"id"})
	assert.Error(t, err)
}

func TestMetricDictionarySyncDoesNotRewriteStableRowsOrPromoteKPIReportKeys(t *testing.T) {
	assert.Contains(t, metricDictionarySyncSQL, "AND s.is_counter='1'",
		"only configured counters may enrich dynamically discovered report-key rows")
	assert.GreaterOrEqual(t, strings.Count(metricDictionarySyncSQL, "IS DISTINCT FROM"), 2,
		"both configured-path upserts and unknown-path enrichment must skip unchanged rows")
}

func TestCellBandSyncReadsParametersOnceAndKeepsTechnologyPairsSeparate(t *testing.T) {
	normalized := strings.Join(strings.Fields(cellBandSyncSQL), " ")
	assert.Equal(t, 1, strings.Count(normalized, "FROM device_parameters"),
		"cell/band dimension sync must not rescan the full parameter table for every path family")
	assert.Contains(t, normalized, "technology")
	assert.Contains(t, normalized, "GROUP BY device_id, fap_instance, technology")
}

func TestCellBandDimSyncIsThrottledSeparatelyFromShadowDimCycle(t *testing.T) {
	runner := NewSyncRunner(nil, nil, DefaultInterval, nil)
	now := time.Unix(1_800_000_000, 0)

	require.Equal(t, 10*time.Minute, runner.cellBandInterval)
	assert.True(t, runner.shouldSyncCellBandDim(now), "first cycle must seed cell_band_dim")

	runner.markCellBandDimAttempt(now)
	assert.False(t, runner.shouldSyncCellBandDim(now.Add(DefaultInterval)),
		"ordinary shadow dimensions may run every minute, but cell_band_dim must not rescan device_parameters that often")
	assert.False(t, runner.shouldSyncCellBandDim(now.Add(9*time.Minute+59*time.Second)))
	assert.True(t, runner.shouldSyncCellBandDim(now.Add(10*time.Minute)))
}

func TestCellBandDimSyncUsesBoundedQueryTimeout(t *testing.T) {
	runner := NewSyncRunner(nil, nil, DefaultInterval, nil)
	require.Equal(t, 2*time.Minute, runner.cellBandQueryTimeout)
	assert.Greater(t, runner.cellBandQueryTimeout, DefaultInterval)
	assert.Less(t, runner.cellBandQueryTimeout, runner.cellBandInterval)
}

func TestCellBandPartialIndexCoversTheSyncPredicate(t *testing.T) {
	normalized := strings.Join(strings.Fields(cellBandRelevantPredicate), " ")
	assert.Contains(t, cellBandParentIndexSQL, "WHERE "+cellBandRelevantPredicate)
	assert.Contains(t, normalized, "CellIdentity")
	assert.Contains(t, normalized, "FreqBandIndicator")
	assert.Contains(t, normalized, "IpaUnitId")
}

func TestBuildCellBandChildIndexSQLUsesConcurrentPartialCoveringIndex(t *testing.T) {
	indexName, sql := buildCellBandChildIndexSQL("public", "device_parameters_p07")
	assert.Equal(t, "device_parameters_p07_cell_band_dim_idx", indexName)
	assert.Contains(t, sql, `CREATE INDEX CONCURRENTLY IF NOT EXISTS "device_parameters_p07_cell_band_dim_idx"`)
	assert.Contains(t, sql, `ON "public"."device_parameters_p07" (device_id, fap_instance, parameter_path)`)
	assert.Contains(t, sql, `INCLUDE (parameter_value)`)
	assert.Contains(t, sql, "WHERE "+cellBandRelevantPredicate)
}
