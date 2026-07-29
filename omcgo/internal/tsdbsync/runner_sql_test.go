package tsdbsync

import (
	"strings"
	"testing"

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
