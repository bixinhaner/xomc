package alarm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/global"
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
		assert.Contains(t, query.sql, "REPLACE(REPLACE(REPLACE(LOWER(COALESCE(event_type, ''))")
		assert.Contains(t, query.args, model.AlarmCritical)
		assert.Contains(t, query.args, model.AlarmSeverity(31001))
		assert.Contains(t, query.args, "equipmentalarm")
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

func TestBuildAlarmEmailDeviceScopeQueryIncludesSelectedGroupsAndChildren(t *testing.T) {
	groupID := uuid.New()

	query, args, err := buildAlarmEmailDeviceScopeQuery([]uuid.UUID{groupID})

	require.NoError(t, err)
	assert.Contains(t, query, "FROM devices d")
	assert.Contains(t, query, "FROM device_group_members")
	assert.Contains(t, query, "FROM device_groups")
	assert.Contains(t, query, "parent_id IN")
	assert.Equal(t, []any{groupID, groupID}, args)
	assert.NotContains(t, query, "NOT EXISTS")
}

func TestBuildAlarmEmailDeviceScopeQueryDefaultGroupsIncludeLegacyUngroupedDevices(t *testing.T) {
	for _, groupID := range []string{global.DefaultLevel1GroupID, global.DefaultLevel2GroupID} {
		t.Run(groupID, func(t *testing.T) {
			query, _, err := buildAlarmEmailDeviceScopeQuery([]uuid.UUID{uuid.MustParse(groupID)})

			require.NoError(t, err)
			assert.Contains(t, query, "NOT EXISTS (SELECT 1 FROM device_group_members m WHERE m.device_id = d.id)")
		})
	}
}

func TestBuildAlarmEmailQueryNormalizesEverySelectedEventType(t *testing.T) {
	window := AlarmEmailWindow{Start: time.Now().Add(-time.Hour), End: time.Now()}
	subscription := &AlarmEmailSubscription{EventTypes: []string{"30000", "Equipment Alarm"}}

	query := buildActiveAlarmEmailQuery(subscription, window, nil, false)

	require.NoError(t, query.err)
	assert.Contains(t, query.sql, " OR ")
	assert.Contains(t, query.args, "communicationsalarm")
	assert.Contains(t, query.args, "equipmentalarm")
	assert.Contains(t, query.args, "30000")
	assert.Contains(t, query.args, "30003")
}
