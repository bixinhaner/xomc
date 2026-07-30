package indicator

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyIndicatorFiltersPlatformNameUsesExactPlatformRelationForAllIndicators(t *testing.T) {
	platform := "BLQ"

	sql, args, err := applyIndicatorFilters(
		storage.Psql.Select("i.id").From("perf_indicators_enb AS i"),
		IndicatorListFilter{PlatformName: &platform},
		DeviceTypeENB,
	).ToSql()

	require.NoError(t, err)
	assert.Contains(t, sql, "EXISTS (SELECT 1 FROM rela_platform_indicator_formula_enb f WHERE f.indicator_id = i.id AND f.platform_name = ANY($1))")
	assert.NotContains(t, sql, "i.is_counter")
	assert.Equal(t, []any{[]string{"BLQ"}}, args)
}

func TestApplyIndicatorFiltersPlatformNameKeepsAllScopedToAllOnly(t *testing.T) {
	platform := PlatformAll

	_, args, err := applyIndicatorFilters(
		storage.Psql.Select("i.id").From("perf_indicators_enb AS i"),
		IndicatorListFilter{PlatformName: &platform},
		DeviceTypeENB,
	).ToSql()

	require.NoError(t, err)
	assert.Equal(t, []any{[]string{PlatformAll}}, args)
}
