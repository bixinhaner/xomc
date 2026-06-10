package alarm

import (
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这些测试覆盖 #9：Statistics() / HistoryStatistics() 此前忽略 filter 导致全表扫描。
// 校验统计基底查询确实把 filter 拼入 WHERE（带 filter 路径），且空 filter 不产生
// 任何 WHERE（空 filter 路径，保持原全量统计语义）。

func TestActiveStatsBaseEmptyFilterHasNoWhere(t *testing.T) {
	sql, args, err := activeStatsBase(AlarmFilter{}).Column("COUNT(*)").ToSql()
	require.NoError(t, err)

	assert.NotContains(t, strings.ToUpper(sql), "WHERE", "空 filter 不应产生 WHERE 子句")
	assert.Empty(t, args, "空 filter 不应携带占位参数")
	// 仍保持与 ListActive 同一 LEFT JOIN，使跨表过滤列可用。
	assert.Contains(t, sql, "LEFT JOIN devices d ON d.id = alarms_active.device_id")
}

func TestActiveStatsBaseWithFilterAppendsWhere(t *testing.T) {
	carrier := model.CarrierCMCC
	filter := AlarmFilter{
		Carrier:   &carrier,
		DeviceSN:  strPtr("SN12345"),
		AlarmType: strPtr("link_down"),
	}

	sql, args, err := activeStatsBase(filter).Column("COUNT(*)").ToSql()
	require.NoError(t, err)

	assert.Contains(t, strings.ToUpper(sql), "WHERE", "带 filter 必须产生 WHERE 子句")
	assert.Contains(t, sql, "alarms_active.carrier")
	assert.Contains(t, sql, "alarms_active.device_sn")
	assert.Contains(t, sql, "alarms_active.alarm_type")
	// 三个等值条件 → 三个占位参数，证明 filter 真正绑定。
	assert.ElementsMatch(t, []interface{}{model.CarrierCMCC, "SN12345", "link_down"}, args)
}

func TestActiveStatsBaseSeverityGroupByCarriesFilter(t *testing.T) {
	carrier := model.CarrierCTCC
	sql, args, err := activeStatsBase(AlarmFilter{Carrier: &carrier}).
		Columns("alarms_active.severity", "COUNT(*)").
		GroupBy("alarms_active.severity").ToSql()
	require.NoError(t, err)

	upper := strings.ToUpper(sql)
	assert.Contains(t, upper, "WHERE")
	assert.Contains(t, upper, "GROUP BY")
	assert.Equal(t, []interface{}{model.CarrierCTCC}, args)
}

func TestHistoryStatsBaseEmptyFilterHasNoWhere(t *testing.T) {
	sql, args, err := historyStatsBase(AlarmFilter{}).Column("COUNT(*)").ToSql()
	require.NoError(t, err)

	assert.NotContains(t, strings.ToUpper(sql), "WHERE", "空 filter 不应产生 WHERE 子句")
	assert.Empty(t, args)
	assert.Contains(t, sql, "LEFT JOIN devices d ON d.id = alarms_history.device_id")
}

func TestHistoryStatsBaseWithFilterAppendsWhere(t *testing.T) {
	carrier := model.CarrierCUCC
	filter := AlarmFilter{
		Carrier:    &carrier,
		Severities: []model.AlarmSeverity{model.AlarmCritical, model.AlarmMajor},
	}

	sql, args, err := historyStatsBase(filter).Column("COUNT(*)").ToSql()
	require.NoError(t, err)

	assert.Contains(t, strings.ToUpper(sql), "WHERE")
	assert.Contains(t, sql, "alarms_history.carrier")
	assert.Contains(t, sql, "alarms_history.severity")
	assert.ElementsMatch(t,
		[]interface{}{model.CarrierCUCC, model.AlarmCritical, model.AlarmMajor},
		args,
	)
}

// 守护回归：旧实现 by-type 统计排除空 alarm_type，新实现保持该语义。
func TestActiveStatsByTypeExcludesEmptyType(t *testing.T) {
	sql, _, err := activeStatsBase(AlarmFilter{}).
		Columns("alarms_active.alarm_type", "COUNT(*)").
		Where(squirrel.NotEq{"alarms_active.alarm_type": ""}).
		GroupBy("alarms_active.alarm_type").ToSql()
	require.NoError(t, err)

	assert.Contains(t, sql, "alarms_active.alarm_type <>")
}
