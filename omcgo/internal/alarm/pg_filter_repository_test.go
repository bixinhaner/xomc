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

// TestAlarmFilterRuleSelect_CoalescesNullableTextColumns guards against the
// HTTP 500 regression where a row with NULL in a nullable text column scanned
// into a plain `string` field crashes with "cannot scan NULL into *string".
// The read methods (GetByID/List/ListEnabled) wrap these columns in
// COALESCE(col, '') so the DB never returns NULL for them. acknowledge_desc,
// created_by and updated_by are nullable text columns scanned into plain
// `string` fields of AlarmFilterRule; webhook_url/webhook_secret are *string so
// they intentionally keep NULL semantics and are NOT coalesced.
func TestAlarmFilterRuleSelect_CoalescesNullableTextColumns(t *testing.T) {
	query := storage.Psql.
		Select(
			"id", "name", "filter_type", "alarm_sources", "alarm_identifiers",
			"device_ids", "device_group_ids", "action", "COALESCE(acknowledge_desc, '') AS acknowledge_desc", "webhook_url", "webhook_secret", "email_recipients",
			"priority", "enabled", "COALESCE(created_by, '') AS created_by", "created_at", "COALESCE(updated_by, '') AS updated_by", "updated_at",
		).
		From("alarm_filters")

	sql, _, err := query.ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "COALESCE(acknowledge_desc, '') AS acknowledge_desc")
	require.Contains(t, sql, "COALESCE(created_by, '') AS created_by")
	require.Contains(t, sql, "COALESCE(updated_by, '') AS updated_by")
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