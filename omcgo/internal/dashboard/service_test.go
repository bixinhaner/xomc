package dashboard

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 告警趋势查询级别归一（issue #219 同根残留）
// 库内 severity 列存的是 5 位字典码（31001~31004），趋势按天分级取数若只认旧的
// 1~4 小编号，四条曲线会全读 0。断言每个级别桶同时匹配旧值与字典码。
// ---------------------------------------------------------------------------

func TestAlarmTrendQueryIncludesLegacySeverityCodes(t *testing.T) {
	cases := []struct {
		bucket string
		expr   string
	}{
		{"critical", "severity IN (1, 31001)"},
		{"major", "severity IN (2, 31002)"},
		{"minor", "severity IN (3, 31003)"},
		{"warning", "severity IN (4, 31004)"},
	}
	for _, c := range cases {
		t.Run(c.bucket, func(t *testing.T) {
			assert.Contains(t, alarmTrendByDateQuery, c.expr)
		})
	}
}

func TestLatestPMSlotHealthQueryReadsOnlyPersistedSummaries(t *testing.T) {
	query, args, err := latestPMSlotHealthQuery()

	require.NoError(t, err)
	assert.Contains(t, query, "FROM pm_slot_health")
	assert.Contains(t, query, "SELECT MAX(latest.slot_end) FROM pm_slot_health latest")
	assert.NotContains(t, query, "DISTINCT ON")
	assert.NotContains(t, query, "pm_files")
	assert.NotContains(t, query, "pm_measurement_anchors")
	assert.NotContains(t, query, "pm_metric_values")
	assert.Empty(t, args)
}

func TestAlarmTrendQueryHasTablePlaceholder(t *testing.T) {
	assert.Contains(t, alarmTrendByDateQuery, "FROM %s")
}

func TestBuildAlarmTrendSnapshotsUsesDayEndAndNow(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, time.July, 27, 18, 30, 15, 123, location)

	got := buildAlarmTrendSnapshots(now, 3)

	require.Len(t, got, 3)
	assert.Equal(t, "2026-07-25", got[0].Date)
	assert.Equal(t, time.Date(2026, time.July, 25, 23, 59, 59, int(time.Second-time.Nanosecond), location), got[0].At)
	assert.Equal(t, "2026-07-26", got[1].Date)
	assert.Equal(t, time.Date(2026, time.July, 26, 23, 59, 59, int(time.Second-time.Nanosecond), location), got[1].At)
	assert.Equal(t, "2026-07-27", got[2].Date)
	assert.Equal(t, now, got[2].At)
}

func TestMergeAlarmTrendSnapshotCountsAddsSourcesAndKeepsZeroDays(t *testing.T) {
	entries := []AlarmTrendEntry{
		{Date: "2026-07-25"},
		{Date: "2026-07-26"},
		{Date: "2026-07-27"},
	}

	err := mergeAlarmTrendSnapshotCounts(entries, []alarmTrendSnapshotCount{
		{Ordinal: 1, Critical: 2, Major: 1},
		{Ordinal: 3, Critical: 5, Minor: 2},
	})
	require.NoError(t, err)
	err = mergeAlarmTrendSnapshotCounts(entries, []alarmTrendSnapshotCount{
		{Ordinal: 1, Warning: 3},
		{Ordinal: 3, Major: 1},
	})
	require.NoError(t, err)

	assert.Equal(t, AlarmTrendEntry{Date: "2026-07-25", Critical: 2, Major: 1, Warning: 3}, entries[0])
	assert.Equal(t, AlarmTrendEntry{Date: "2026-07-26"}, entries[1])
	assert.Equal(t, AlarmTrendEntry{Date: "2026-07-27", Critical: 5, Major: 1, Minor: 2}, entries[2])
}

func TestMergeAlarmTrendSnapshotCountsRejectsInvalidOrdinal(t *testing.T) {
	entries := []AlarmTrendEntry{{Date: "2026-07-27"}}
	err := mergeAlarmTrendSnapshotCounts(entries, []alarmTrendSnapshotCount{{Ordinal: 2}})
	require.Error(t, err)
}

func TestActiveAlarmTrendQueriesUseSnapshotInventorySemantics(t *testing.T) {
	activeSQL, _, err := buildActiveAlarmTrendSnapshotQuery([]time.Time{time.Now()}, nil)
	require.NoError(t, err)
	historySQL, _, err := buildHistoryAlarmTrendSnapshotQuery([]time.Time{time.Now()}, nil, nil)
	require.NoError(t, err)

	assert.Contains(t, activeSQL, "FROM unnest($1::timestamptz[]) WITH ORDINALITY")
	assert.Contains(t, activeSQL, "alarm.raised_at <= snapshots.snapshot_at")
	assert.Contains(t, historySQL, "alarm.raised_at <= snapshots.snapshot_at")
	assert.Contains(t, historySQL, "alarm.cleared_at > snapshots.snapshot_at")
}

func TestActiveAlarmTrendQueriesApplyVisibleGroups(t *testing.T) {
	groupID := uuid.New()
	activeSQL, activeArgs, err := buildActiveAlarmTrendSnapshotQuery(
		[]time.Time{time.Now()},
		[]uuid.UUID{groupID},
	)
	require.NoError(t, err)
	historySQL, historyArgs, err := buildHistoryAlarmTrendSnapshotQuery(
		[]time.Time{time.Now()},
		[]uuid.UUID{groupID},
		nil,
	)
	require.NoError(t, err)

	assert.Contains(t, activeSQL, "device_group_members")
	assert.Contains(t, historySQL, "device_group_members")
	require.Len(t, activeArgs, 2)
	require.Len(t, historyArgs, 3)
	assert.Equal(t, groupID, activeArgs[1])
	assert.Equal(t, groupID, historyArgs[1])
}

func TestHistoryAlarmTrendQueryExcludesIDsAlreadyCountedAsActive(t *testing.T) {
	activeID := uuid.New()
	query, args, err := buildHistoryAlarmTrendSnapshotQuery(
		[]time.Time{time.Now()},
		nil,
		[]uuid.UUID{activeID},
	)
	require.NoError(t, err)

	assert.Contains(t, query, "NOT (alarm.alarm_id = ANY($2::uuid[]))")
	require.Len(t, args, 2)
	assert.Equal(t, []uuid.UUID{activeID}, args[1])
}

func TestCollectAlarmTrendSnapshotIDsDeduplicatesAcrossBuckets(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	got := collectAlarmTrendSnapshotIDs([]alarmTrendSnapshotCount{
		{AlarmIDs: []uuid.UUID{firstID}},
		{AlarmIDs: []uuid.UUID{firstID, secondID}},
	})

	assert.ElementsMatch(t, []uuid.UUID{firstID, secondID}, got)
}

// ---------------------------------------------------------------------------
// parseKPINames
// ---------------------------------------------------------------------------

func TestParseKPINames_Empty(t *testing.T) {
	got := parseKPINames("")
	assert.Nil(t, got)
}

func TestParseKPINames_Single(t *testing.T) {
	got := parseKPINames("rrc_success_rate")
	assert.Equal(t, []string{"rrc_success_rate"}, got)
}

func TestParseKPINames_Multiple(t *testing.T) {
	got := parseKPINames("a,b,c")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestParseKPINames_Whitespace(t *testing.T) {
	got := parseKPINames(" a , b , c ")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestParseKPINames_EmptyParts(t *testing.T) {
	got := parseKPINames("a,,b")
	assert.Equal(t, []string{"a", "b"}, got)
}

// ---------------------------------------------------------------------------
// severityToLabel
// ---------------------------------------------------------------------------

func TestSeverityToLabel(t *testing.T) {
	tests := []struct {
		name     string
		severity model.AlarmSeverity
		want     string
	}{
		{"critical", model.AlarmCritical, "critical"},
		{"major", model.AlarmMajor, "major"},
		{"minor", model.AlarmMinor, "minor"},
		{"warning", model.AlarmWarning, "warning"},
		{"critical dictionary code", model.AlarmSeverity(31001), "critical"},
		{"major dictionary code", model.AlarmSeverity(31002), "major"},
		{"minor dictionary code", model.AlarmSeverity(31003), "minor"},
		{"warning dictionary code", model.AlarmSeverity(31004), "warning"},
		{"unknown value", model.AlarmSeverity(99), "unknown"},
		{"zero value", model.AlarmSeverity(0), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityToLabel(tt.severity)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// severityFromLabel
// ---------------------------------------------------------------------------

func TestSeverityFromLabel(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  int
	}{
		{"critical", "critical", 1},
		{"major", "major", 2},
		{"minor", "minor", 3},
		{"warning", "warning", 4},
		{"unknown label", "info", 5},
		{"empty label", "", 5},
		{"random string", "foobar", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityFromLabel(tt.label)
			assert.Equal(t, tt.want, got)
		})
	}
}

// strPtr is a helper function to create a string pointer.
func strPtr(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// derefOrEmpty
// ---------------------------------------------------------------------------

func TestDerefOrEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{
			name:  "非空指针返回值",
			input: strPtr("test-value"),
			want:  "test-value",
		},
		{
			name:  "nil指针返回空字符串",
			input: nil,
			want:  "",
		},
		{
			name:  "空字符串指针返回空字符串",
			input: strPtr(""),
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derefOrEmpty(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// 历史快照查询（Issue D / 同环比对比）
// countDevicesAtTime / countAlarmsAtTime 的 SQL 必须满足两点：
//   1. 时间过滤覆盖「在 t 之前出现且截至 t 仍存活」语义；
//   2. devices 查询打到主库（pgPool），alarms_history 查询打到时序库（tsPool）。
// 这里仅断言 SQL 文本，避免起容器；与既有 alarmTrendByDateQuery 断言风格一致。
// ---------------------------------------------------------------------------

func TestCountDevicesAtTimeQuery(t *testing.T) {
	assert.Contains(t, countDevicesAtTimeQuery, "FROM devices")
	assert.Contains(t, countDevicesAtTimeQuery, "created_at <= $1")
	assert.Contains(t, countDevicesAtTimeQuery, "deleted_at IS NULL OR deleted_at > $1")
}

func TestCountAlarmsAtTimeQuery(t *testing.T) {
	assert.Contains(t, listActiveAlarmIDsAtTimeQuery, "SELECT id")
	assert.NotContains(t, listActiveAlarmIDsAtTimeQuery, "SELECT alarm_id")
	assert.Contains(t, listActiveAlarmIDsAtTimeQuery, "FROM alarms_active")
	assert.Contains(t, listActiveAlarmIDsAtTimeQuery, "raised_at <= $1")
	assert.Contains(t, countHistoricalAlarmsAtTimeQuery, "FROM alarms_history")
	assert.Contains(t, countHistoricalAlarmsAtTimeQuery, "COUNT(DISTINCT alarm_id)")
	assert.Contains(t, countHistoricalAlarmsAtTimeQuery, "raised_at <= $1")
	assert.Contains(t, countHistoricalAlarmsAtTimeQuery, "cleared_at > $1")
	assert.Contains(t, countHistoricalAlarmsAtTimeQuery, "ANY($2::uuid[])")
}

// summaryDeviceCountsQuery 是 /summary 接口 KPI 卡的设备总数 + 在线数取数 SQL。
// 必须满足：
//   1. 取自 devices 父表（按 carrier 分区，父表查询贯穿所有分区）；
//   2. "在线"语义 = is_online=TRUE（T-0162 后与 lifecycle 解耦，与设备状态柱图同源）；
//   3. 排除软删（deleted_at IS NULL）。
// 任何 SQL 改动只要破坏上述三条，本测试立即失败。

func TestSummaryDeviceCountsQuery(t *testing.T) {
	assert.Contains(t, summaryDeviceCountsQuery, "FROM devices")
	assert.Contains(t, summaryDeviceCountsQuery, "COUNT(*) FILTER (WHERE is_online = TRUE)")
	assert.Contains(t, summaryDeviceCountsQuery, "deleted_at IS NULL")
}

func TestComputeKPIDeltaRequiresComparableBaseline(t *testing.T) {
	t.Run("无历史基线时明确标记不可比较且不伪造百分比", func(t *testing.T) {
		got := computeKPIDelta(12, 0, false, "last_week")

		assert.False(t, got.HasComparison)
		assert.Zero(t, got.ChangePercent)
		assert.Equal(t, "stable", got.Trend)
	})

	t.Run("存在非零历史基线时计算真实变化", func(t *testing.T) {
		got := computeKPIDelta(12, 10, true, "last_week")

		assert.True(t, got.HasComparison)
		assert.InDelta(t, 20, got.ChangePercent, 0.001)
		assert.Equal(t, "up", got.Trend)
	})

	t.Run("零值历史样本不能作为百分比基线", func(t *testing.T) {
		got := computeKPIDelta(12, 0, true, "yesterday")

		assert.False(t, got.HasComparison)
		assert.Zero(t, got.ChangePercent)
	})
}

func TestAlarmComparisonRequiresPreviousDeviceBaseline(t *testing.T) {
	assert.False(t, hasPreviousAlarmBaseline(0, nil))
	assert.True(t, hasPreviousAlarmBaseline(11, nil))
	assert.False(t, hasPreviousAlarmBaseline(11, assert.AnError))
}

func TestAverageKPITrendEntries(t *testing.T) {
	avg, ok := averageKPITrendEntries([]KPITrendEntry{{Value: 8}, {Value: 12}})
	assert.True(t, ok)
	assert.Equal(t, 10.0, avg)

	_, ok = averageKPITrendEntries(nil)
	assert.False(t, ok)

	avg, ok = averageKPITrendEntries([]KPITrendEntry{
		{Value: math.NaN()},
		{Value: math.Inf(1)},
		{Value: 9},
	})
	assert.True(t, ok)
	assert.Equal(t, 9.0, avg)

	_, ok = averageKPITrendEntries([]KPITrendEntry{{Value: math.NaN()}, {Value: math.Inf(-1)}})
	assert.False(t, ok)
}

func TestSetDashboardKPIOverviewValueMapsActiveUEIndicatorID(t *testing.T) {
	overview := map[string]float64{}
	setDashboardKPIOverviewValue(overview, activeUEKPIID, 17)

	assert.Equal(t, 17.0, overview[activeUEKPIID])
	assert.Equal(t, 17.0, overview[activeUEKPIAlias])
}

func TestDashboardKPIDeltaWindowsUseLocalCalendarBoundaries(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	now := time.Date(2026, 7, 21, 19, 15, 0, 0, location)

	windows := dashboardKPIDeltaWindows(now)

	assert.Equal(t, time.Date(2026, 7, 14, 19, 15, 0, 0, location), windows.DeviceCompareAt)
	assert.Equal(t, time.Date(2026, 7, 20, 19, 15, 0, 0, location), windows.AlarmCompareAt)
	assert.Equal(t, time.Date(2026, 7, 21, 0, 0, 0, 0, location), windows.UECurrentStart)
	assert.Equal(t, now, windows.UECurrentEnd)
	assert.Equal(t, time.Date(2026, 7, 14, 0, 0, 0, 0, location), windows.UEPreviousStart)
	assert.Equal(t, time.Date(2026, 7, 14, 19, 15, 0, 0, location), windows.UEPreviousEnd)
}

func TestComputeSeriesKPIDeltaUsesSameAggregationOnBothPeriods(t *testing.T) {
	got := computeSeriesKPIDelta(
		[]KPITrendEntry{{Value: 12}, {Value: 18}},
		[]KPITrendEntry{{Value: 8}, {Value: 12}},
		"last_week",
	)
	assert.True(t, got.HasComparison)
	assert.Equal(t, 15.0, got.CurrentValue)
	assert.Equal(t, 10.0, got.PreviousValue)
	assert.Equal(t, 50.0, got.ChangePercent)

	missing := computeSeriesKPIDelta(nil, []KPITrendEntry{{Value: 10}}, "last_week")
	assert.False(t, missing.HasComparison)
}
