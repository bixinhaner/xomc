package alarm

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

func TestApplyAlarmFilterRuleOrdering_UsesStableTiebreakers(t *testing.T) {
	query := applyAlarmFilterRuleOrdering(
		storage.Psql.Select("id").From("alarm_filters"),
	)

	sql, _, err := query.ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "ORDER BY priority ASC, created_at ASC, id ASC")
}

func TestApplyAlarmFilterRuleFilters_AppliesKeywordAcrossVisibleFields(t *testing.T) {
	keyword := "operator"
	query := applyAlarmFilterRuleFilters(
		storage.Psql.Select("id").From("alarm_filters"),
		AlarmFilterRuleFilter{Keyword: &keyword},
	)

	sql, args, err := query.ToSql()
	require.NoError(t, err)
	require.Len(t, args, 10)
	require.Contains(t, sql, "name ILIKE")
	require.Contains(t, sql, "created_by ILIKE")
	require.Contains(t, sql, "updated_by ILIKE")
	require.Contains(t, sql, "filter_type ILIKE")
	require.Contains(t, sql, "action ILIKE")
	require.Contains(t, sql, "array_to_string(alarm_identifiers, ',') ILIKE")
	require.Contains(t, sql, "array_to_string(alarm_sources, ',') ILIKE")
	require.Contains(t, sql, "array_to_string(device_ids, ',') ILIKE")
	require.Contains(t, sql, "array_to_string(device_group_ids, ',') ILIKE")
	require.Contains(t, sql, "d.serial_number ILIKE")
}

func TestApplyAlarmFilterRuleFilters_AppliesAnySelectedDimension(t *testing.T) {
	query := applyAlarmFilterRuleFilters(
		storage.Psql.Select("id").From("alarm_filters"),
		AlarmFilterRuleFilter{FilterTypes: []string{FilterTypeAlarmIdentifier, FilterTypeDevice}},
	)

	sql, args, err := query.ToSql()
	require.NoError(t, err)
	require.Empty(t, args)
	require.Contains(t, sql, "cardinality(alarm_identifiers) > 0")
	require.Contains(t, sql, "cardinality(device_ids) > 0")
}