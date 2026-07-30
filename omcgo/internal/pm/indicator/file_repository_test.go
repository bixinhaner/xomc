package indicator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPlatformSummarySQLUsesSamePlatformScopeAsIndicatorList(t *testing.T) {
	sql := buildPlatformSummarySQL("enb")

	assert.Contains(t, sql, "FROM rela_platform_indicator_formula_enb")
	assert.Contains(t, sql, "JOIN perf_indicators_enb i")
	assert.Contains(t, sql, "COUNT(DISTINCT rf.indicator_id)")
	assert.Contains(t, sql, "GROUP BY rf.platform_name")
	assert.NotContains(t, sql, "platform_name <>")
	assert.NotContains(t, sql, "rf.platform_name = $2")
}
