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