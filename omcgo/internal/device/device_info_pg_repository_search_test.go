package device

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceListSearchFields(t *testing.T) {
	assert.Equal(t, []string{
		"d.serial_number",
		"d.site_name",
		"d.manufacturer",
		"d.model_name",
		"di.device_name",
		"di.address",
		"host(d.ip_address)",
		"di.mac",
		"di.pci",
	}, deviceListSearchFields())
}

func TestDeviceListCountSelect_DeduplicatesDeviceRows(t *testing.T) {
	q, _, err := deviceListCountSelect().
		From("devices d").
		LeftJoin("device_group_members dgm ON dgm.device_id = d.id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "COUNT(DISTINCT d.id)")
	assert.NotContains(t, q, "COUNT(*)")
}

func TestComputeListStats_UsesAlarmTotalNotAlarmedDeviceCount(t *testing.T) {
	const subQ = `SELECT DISTINCT d.id, d.lifecycle_state, d.is_online, (
		SELECT COUNT(*)
		FROM alarms_active aa
		WHERE aa.device_id = d.id AND aa.status <> 'cleared'
	) AS active_alarm_count FROM devices d`
	groupQ := "SELECT lifecycle_state, is_online, COUNT(*), " +
		"COALESCE(SUM(active_alarm_count), 0) FROM (" + subQ +
		") s GROUP BY lifecycle_state, is_online"

	assert.Contains(t, groupQ, "SUM(active_alarm_count)")
	assert.NotContains(t, groupQ, "COUNT(*) FILTER (WHERE alarmed)")
}

func TestComputeListStats_CurrentUEOnlyIncludesOnlineDevices(t *testing.T) {
	const subQ = `SELECT DISTINCT d.id, d.lifecycle_state, d.is_online, ` + deviceListStatsCurrentUECountExpr + ` FROM devices d`
	groupQ := deviceListStatsGroupQuery(subQ)

	assert.Contains(t, subQ, "WHEN d.is_online THEN COALESCE(di.ue_count, 0)")
	assert.Contains(t, subQ, "ELSE 0")
	assert.Contains(t, groupQ, "SUM(current_ue_count)")
}
