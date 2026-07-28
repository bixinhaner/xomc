package dashboard

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTopAlarmDevicesQueryUsesCurrentInventoryAndSeverityBuckets(t *testing.T) {
	query, args, err := buildTopAlarmDevicesQuery(nil)
	require.NoError(t, err)

	assert.Contains(t, query, "FROM alarms_active")
	assert.NotContains(t, query, "alarms_history")
	assert.Contains(t, query, "severity IN (1,31001)")
	assert.Contains(t, query, "severity IN (2,31002)")
	assert.Contains(t, query, "severity IN (3,31003)")
	assert.Contains(t, query, "severity IN (4,31004)")
	assert.Contains(t, query, "GROUP BY device_sn, technology")
	assert.Contains(t, query, "ORDER BY alarm_count DESC, device_sn ASC")
	assert.Contains(t, query, "LIMIT 10")
	assert.Equal(t, []any{""}, args)
}

func TestBuildTopAlarmDevicesQueryAppliesVisibleGroups(t *testing.T) {
	groupID := uuid.New()
	query, args, err := buildTopAlarmDevicesQuery([]uuid.UUID{groupID})
	require.NoError(t, err)

	assert.Contains(t, query, "device_group_members")
	require.Len(t, args, 2)
	assert.Equal(t, groupID, args[1])
}

func TestBuildTopAlarmDevicesQueryFailsClosedForNoVisibleGroups(t *testing.T) {
	query, _, err := buildTopAlarmDevicesQuery([]uuid.UUID{})
	require.NoError(t, err)
	assert.Contains(t, query, "AND FALSE")
}
