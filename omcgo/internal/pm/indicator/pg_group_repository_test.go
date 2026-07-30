package indicator

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupPlatformFilterExprUsesExactPlatformOnly(t *testing.T) {
	sql, args, err := storage.Psql.
		Select("g.id").
		From("indicator_groups_enb g").
		Where(groupPlatformFilterExpr(DeviceTypeENB, " BLQ ")).
		ToSql()

	require.NoError(t, err)
	assert.Contains(t, sql, "rela_platform_indicator_formula_enb f")
	assert.Contains(t, sql, "f.platform_name = $1")
	assert.NotContains(t, sql, "ANY")
	assert.Equal(t, []any{"BLQ"}, args)
}

func TestBuildGroupIndicatorCountSQLUsesExactPlatformWhenProvided(t *testing.T) {
	sql, args, err := buildGroupIndicatorCountSQL(DeviceTypeENB, " BLQ ")

	require.NoError(t, err)
	assert.Contains(t, sql, "LEFT JOIN perf_indicators_enb i ON i.group_id = g.id")
	assert.Contains(t, sql, "LEFT JOIN rela_platform_indicator_formula_enb f ON f.indicator_id = i.id AND f.platform_name = $1")
	assert.Contains(t, sql, "COUNT(DISTINCT i.id) FILTER (WHERE f.indicator_id IS NOT NULL)")
	assert.NotContains(t, sql, "ANY")
	assert.Equal(t, []any{"BLQ"}, args)
}

func TestBuildGroupIndicatorCountSQLCountsAllIndicatorsWithoutPlatform(t *testing.T) {
	sql, args, err := buildGroupIndicatorCountSQL(DeviceTypeENB, "")

	require.NoError(t, err)
	assert.Contains(t, sql, "COALESCE(COUNT(i.id), 0)")
	assert.NotContains(t, sql, "rela_platform_indicator_formula_enb")
	assert.Empty(t, args)
}
