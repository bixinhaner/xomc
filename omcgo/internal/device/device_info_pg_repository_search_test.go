package device

import (
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceListSearchFields(t *testing.T) {
	assert.Equal(t, []string{
		"d.serial_number",
		"d.site_name",
		"COALESCE(udi.site_id, d.site_id)",
		"d.manufacturer",
		"d.model_name",
		"COALESCE(udi.device_name, di.device_name)",
		"COALESCE(udi.address, di.address)",
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

func TestDeviceListPageIDsSelect_UsesLightPagedIDQuery(t *testing.T) {
	q, _, err := deviceListPageIDsSelect(DeviceFilter{
		DeviceType: DeviceListDeviceTypeBaseStation,
		ListRequest: model.ListRequest{
			Page:     2,
			PageSize: 20,
		},
	}).PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "SELECT d.id, ROW_NUMBER() OVER")
	assert.Contains(t, q, "FROM devices d")
	assert.Contains(t, q, "LIMIT 20")
	assert.Contains(t, q, "OFFSET 20")
	assert.NotContains(t, q, "device_location_observations", "分页 ID 查询不应补位置观测字段")
	assert.NotContains(t, q, "device_groups dg", "分页 ID 查询不应补分组名称")
	assert.NotContains(t, q, "active_alarm_count", "默认分页 ID 查询不应聚合告警")
}

func TestDeviceListDetailSelect_HydratesOnlyPagedDevices(t *testing.T) {
	pageIDs := deviceListPageIDsSelect(DeviceFilter{
		DeviceType: DeviceListDeviceTypeBaseStation,
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: 20,
		},
	})
	q, _, err := deviceListDetailSelect(pageIDs).PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "JOIN ( SELECT d.id")
	assert.Contains(t, q, ") page_ids ON page_ids.id = d.id")
	assert.Contains(t, q, "ORDER BY page_ids.page_order ASC")
	assert.Contains(t, q, "LEFT JOIN LATERAL")
	assert.Contains(t, q, "aa_page.device_id = d.id")
}

func TestDeviceListPageIDsSelect_UsesAlarmAggOnlyForAlarmSort(t *testing.T) {
	q, _, err := deviceListPageIDsSelect(DeviceFilter{
		ListRequest: model.ListRequest{
			SortBy:   "alarm_severity",
			SortDir:  "asc",
			Page:     1,
			PageSize: 20,
		},
	}).PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)

	assert.Contains(t, q, "GROUP BY device_id")
	assert.Contains(t, q, "aa.top_sev ASC NULLS LAST")
}

func TestDeviceListPageIDsSelect_SortKeysStayOnWhitelist(t *testing.T) {
	for sortKey, sortCol := range allowedSortColumnsWithInfo {
		t.Run(sortKey, func(t *testing.T) {
			q, args, err := deviceListPageIDsSelect(DeviceFilter{
				ListRequest: model.ListRequest{
					SortBy:   sortKey,
					SortDir:  "asc",
					Page:     1,
					PageSize: 20,
				},
			}).PlaceholderFormat(sq.Dollar).ToSql()
			require.NoError(t, err)

			assert.Contains(t, q, "ORDER BY "+sortCol+" ASC")
			assert.NotContains(t, q, ";")
			assert.Empty(t, args)
		})
	}
}

func TestDeviceListFilterQueriesShareCommonFilters(t *testing.T) {
	groupID := uuid.New()
	productID := uuid.New()
	online := true
	source := DeviceControlSourceGeofence
	filter := DeviceFilter{
		DeviceType:    DeviceListDeviceTypeBaseStation,
		GroupID:       &groupID,
		Search:        stringPointer("SN-001"),
		ProductID:     &productID,
		RFStatus:      stringPointer("enabled"),
		CellStatus:    stringPointer("active"),
		ProjectStatus: stringPointer("commissioned"),
		GPSStatus:     stringPointer("locked"),
		AlarmSeverity: stringPointer("critical"),
		LicenseStatus: stringPointer("valid"),
		OpState:       stringPointer("1"),
		IsOnline:      &online,
		ControlSource: &source,
		ListRequest: model.ListRequest{
			Page:     1,
			PageSize: 20,
		},
	}

	listSQL, _, err := deviceListPageIDsSelect(filter).PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)
	countSQL, _, err := applyDeviceFilters(deviceListFilterBaseSelect("COUNT(DISTINCT d.id)"), filter).
		PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)
	statsSQL, _, err := applyDeviceFilters(
		deviceListFilterBaseSelect("DISTINCT d.id", "d.lifecycle_state", "d.is_online", deviceListStatsCurrentUECountExpr),
		filter,
	).PlaceholderFormat(sq.Dollar).ToSql()
	require.NoError(t, err)

	for _, q := range []string{listSQL, countSQL, statsSQL} {
		assert.Contains(t, q, "COALESCE(d.product_class, '') LIKE 'UPS%'")
		assert.Contains(t, q, "dgm.group_id")
		assert.Contains(t, q, "d.product_id")
		assert.Contains(t, q, "di.rf_status")
		assert.Contains(t, q, "di.cell_status")
		assert.Contains(t, q, "di.project_status")
		assert.Contains(t, q, "di.gps_status")
		assert.Contains(t, q, "di.license_status")
		assert.Contains(t, q, "di.op_state")
		assert.Contains(t, q, "d.is_online")
		assert.Contains(t, q, "SELECT MIN(aaf.severity)")
		assert.Contains(t, q, "EXISTS")
		assert.Contains(t, strings.ToLower(q), "like")
	}
}

func TestComputeListStats_UsesAlarmTotalNotAlarmedDeviceCount(t *testing.T) {
	const subQ = `SELECT DISTINCT d.id, d.lifecycle_state, d.is_online, 0 AS current_ue_count FROM devices d`
	groupQ := deviceListStatsGroupQuery(subQ)

	assert.Contains(t, groupQ, "active_alarm_counts")
	assert.Contains(t, groupQ, "JOIN filtered_devices fd ON fd.id = aa.device_id")
	assert.Contains(t, groupQ, "SUM(COALESCE(aac.active_alarm_count, 0))")
	assert.NotContains(t, groupQ, "COUNT(*) FILTER (WHERE alarmed)")
	assert.NotContains(t, groupQ, "SELECT COUNT(*) FROM alarms_active")
}

func TestComputeListStats_CurrentUEOnlyIncludesOnlineDevices(t *testing.T) {
	const subQ = `SELECT DISTINCT d.id, d.lifecycle_state, d.is_online, ` + deviceListStatsCurrentUECountExpr + ` FROM devices d`
	groupQ := deviceListStatsGroupQuery(subQ)

	assert.Contains(t, subQ, "WHEN d.is_online THEN COALESCE(di.ue_count, 0)")
	assert.Contains(t, subQ, "ELSE 0")
	assert.Contains(t, groupQ, "SUM(fd.current_ue_count)")
}
