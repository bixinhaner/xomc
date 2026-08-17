package alarm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestBuildAlarmEmailQueriesUseLifecycleWindowAndSameScope(t *testing.T) {
	deviceID := uuid.New()
	window := AlarmEmailWindow{
		Start: time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 17, 8, 10, 0, 0, time.UTC),
	}
	subscription := &AlarmEmailSubscription{
		AlarmIdentifiers: []string{"10"},
		Severities:       []int16{1},
		AlarmSources:     []string{"OMC"},
		EventTypes:       []string{"30003"},
	}

	active := buildActiveAlarmEmailQuery(subscription, window, []uuid.UUID{deviceID}, true)
	history := buildHistoryAlarmEmailQuery(subscription, window, []uuid.UUID{deviceID}, true)
	require.NoError(t, active.err)
	require.NoError(t, history.err)
	assert.Contains(t, active.sql, "FROM alarms_active")
	assert.Contains(t, active.sql, "NULL::timestamptz")
	assert.Contains(t, history.sql, "FROM alarms_history")
	assert.Contains(t, history.sql, "cleared_at")
	for _, query := range []alarmEmailQuery{active, history} {
		assert.Contains(t, query.sql, "alarm_identifier IN")
		assert.Contains(t, query.sql, "device_id IN")
		assert.Contains(t, query.args, model.AlarmCritical)
		assert.Contains(t, query.args, model.AlarmSeverity(31001))
	}
	assert.Contains(t, active.sql, "raised_at >=")
	assert.Contains(t, active.sql, "raised_at <")
	assert.NotContains(t, active.sql, "cleared_at >=")
	assert.Contains(t, history.sql, "raised_at >=")
	assert.Contains(t, history.sql, "raised_at <")
	assert.Contains(t, history.sql, "cleared_at >=")
	assert.Contains(t, history.sql, "cleared_at <")
	assert.Contains(t, history.sql, " OR ")
}

func TestBuildAlarmEmailQueryWithEmptySelectedGroupMatchesNothing(t *testing.T) {
	window := AlarmEmailWindow{Start: time.Now().Add(-time.Hour), End: time.Now()}
	query := buildActiveAlarmEmailQuery(&AlarmEmailSubscription{}, window, nil, true)
	require.NoError(t, query.err)
	assert.Contains(t, query.sql, "FALSE")
}
