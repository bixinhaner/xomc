package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// applyFilters 各条件单测（不依赖 pgx pool）
// ---------------------------------------------------------------------------

func Test_applyFilters_Empty(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	result := applyFilters(qb, QueryRequest{})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM pm_metrics", sql)
	assert.Empty(t, args)
}

func Test_applyFilters_DeviceSNs_IN(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	result := applyFilters(qb, QueryRequest{DeviceSNs: []string{"BLQ-001", "BLQ-002"}})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "device_sn IN")
	assert.Len(t, args, 2)
}

func Test_applyFilters_DeviceOUIs_IN(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	result := applyFilters(qb, QueryRequest{DeviceOUIs: []string{"48BF74", "00E0FC"}})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "device_oui IN")
	assert.Len(t, args, 2)
}

func Test_applyFilters_OUI_SN_Paired(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	result := applyFilters(qb, QueryRequest{
		DeviceOUIs: []string{"48BF74", "00E0FC"},
		DeviceSNs:  []string{"BLQ-001", "BLQ-002"},
	})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	// 应该是 (oui=$1 AND sn=$2) OR (oui=$3 AND sn=$4)
	assert.Contains(t, sql, "device_oui = $")
	assert.Contains(t, sql, "device_sn = $")
	assert.Contains(t, sql, " OR ")
	assert.Len(t, args, 4)
}

func Test_applyFilters_OUI_SN_UnequalLength_TruncatesToMin(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	// OUIs 2 个、SNs 1 个 → 按 min(2,1)=1 配对
	result := applyFilters(qb, QueryRequest{
		DeviceOUIs: []string{"48BF74", "00E0FC"},
		DeviceSNs:  []string{"BLQ-001"},
	})
	_, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Len(t, args, 2, "should pair 1 OUI+SN, not 3 args")
}

func Test_applyFilters_MetricPaths_IN(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	result := applyFilters(qb, QueryRequest{MetricPaths: []string{"L.Cell.Avail", "KPI.Avail.Rate"}})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "metric_path IN")
	assert.Len(t, args, 2)
}

func Test_applyFilters_MetricType(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	mt := MetricTypeCounter
	result := applyFilters(qb, QueryRequest{MetricType: &mt})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "metric_type = $1")
	assert.Equal(t, "counter", args[0])
}

func Test_applyFilters_Granularity(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	result := applyFilters(qb, QueryRequest{Granularity: Granularity15Min})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "granularity = $1")
	assert.Equal(t, "15min", args[0])
}

func Test_applyFilters_TimeRange(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	start := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 23, 1, 0, 0, 0, time.UTC)
	result := applyFilters(qb, QueryRequest{StartTime: start, EndTime: end})
	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "time >= $1")
	assert.Contains(t, sql, "time <= $2")
	assert.Len(t, args, 2)
}

func Test_applyFilters_Combined(t *testing.T) {
	qb := storage.Psql.Select("*").From("pm_metrics")
	mt := MetricTypeKPI
	result := applyFilters(qb, QueryRequest{
		DeviceSNs:   []string{"BLQ-001"},
		MetricType:  &mt,
		Granularity: Granularity15Min,
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
	})
	sql, _, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "device_sn IN")
	assert.Contains(t, sql, "metric_type = ")
	assert.Contains(t, sql, "granularity = ")
	assert.Contains(t, sql, "time >= ")
	assert.Contains(t, sql, "time <= ")
}

// ---------------------------------------------------------------------------
// buildBatchInsertSQL：BUG-6 自然键含 object_ldn 回归测试
// ---------------------------------------------------------------------------

// metricWithLDN 构造一个最小有效 PMMetric，object_ldn 通过参数控制（nil = 不设）。
func metricWithLDN(path, ldn string, setLDN bool) PMMetric {
	t := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC)
	m := PMMetric{
		DeviceOUI:   "48BF74",
		DeviceSN:    "1202000240194DP0015",
		MetricPath:  path,
		MetricType:  MetricTypeCounter,
		MetricValue: 123,
		Granularity: Granularity15Min,
		Time:        t,
		StartTime:   t.Add(-15 * time.Minute),
		EndTime:     t,
		IngestTime:  t,
	}
	if setLDN {
		v := ldn
		m.ObjectLDN = &v
	}
	return m
}

// BUG-6 回归：ON CONFLICT 子句必须含 object_ldn 列，否则同 PM 文件多 cell 同 counter_name
// 会撞自然键二次命中触发 SQLSTATE 21000。
func Test_buildBatchInsertSQL_ON_CONFLICT_Includes_ObjectLDN(t *testing.T) {
	ms := []PMMetric{metricWithLDN("L.Cell.Avail", "cell-1", true)}
	sql, _, err := buildBatchInsertSQL(ms)
	require.NoError(t, err)
	assert.Contains(t, sql,
		"ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn)",
		"自然键必须含 object_ldn（BUG-6 回归）")
	assert.Contains(t, sql, "DO UPDATE SET metric_value = EXCLUDED.metric_value")
}

// BUG-6 回归：同 device+path+time 跨多个 cell（不同 object_ldn）在 SQL 参数中
// 必须出现各自的 ldn 值（不被合并 / 不被丢弃 / 不为 NULL）。
func Test_buildBatchInsertSQL_MultiCell_NoNaturalKeyCollision(t *testing.T) {
	ms := []PMMetric{
		metricWithLDN("L.Cell.Avail", "cell-1", true),
		metricWithLDN("L.Cell.Avail", "cell-2", true),
		metricWithLDN("L.Cell.Avail", "cell-3", true),
	}
	sql, args, err := buildBatchInsertSQL(ms)
	require.NoError(t, err)

	// 三行 → 14 列 × 3 = 42 占位符；其中 object_ldn 是第 13 列（0-index 12）
	// 不同 cell-N 都应作为独立参数出现。
	require.Len(t, args, 14*3)

	ldnValues := []string{}
	for _, a := range args {
		if s, ok := a.(string); ok && strings.HasPrefix(s, "cell-") {
			ldnValues = append(ldnValues, s)
		}
	}
	assert.ElementsMatch(t, []string{"cell-1", "cell-2", "cell-3"}, ldnValues,
		"三 cell 的 object_ldn 必须各自落参数（不合并）")

	// SQL 不应出现 NULL 字面量（object_ldn nil → '' 落值后由 squirrel 占位符承载）
	assert.NotContains(t, sql, "NULL")
}

// nil ObjectLDN 落参数时统一为 ''（migration 000171 要求 NOT NULL DEFAULT ''）。
func Test_buildBatchInsertSQL_NilObjectLDN_FallsBackToEmptyString(t *testing.T) {
	ms := []PMMetric{metricWithLDN("L.Cell.Avail", "", false)}
	_, args, err := buildBatchInsertSQL(ms)
	require.NoError(t, err)

	// 第 13 个 args（0-index 12）是 object_ldn。
	// 14 列分别：id, oui, sn, path, type, value, statis, gran, time, start, end, ingest, ldn, extra
	require.Len(t, args, 14)
	assert.Equal(t, "", args[12], "ObjectLDN=nil 必须落空字符串而非 NULL")
}

// ---------------------------------------------------------------------------
// MetricType / StatisType / Granularity 常量稳定性（避免误改字符串值）
// ---------------------------------------------------------------------------

func Test_Constants_Stable(t *testing.T) {
	assert.Equal(t, "counter", string(MetricTypeCounter))
	assert.Equal(t, "kpi", string(MetricTypeKPI))
	assert.Equal(t, "sum", string(StatisSum))
	assert.Equal(t, "avg", string(StatisAvg))
	assert.Equal(t, "max", string(StatisMax))
	assert.Equal(t, "pct", string(StatisPct))
	assert.Equal(t, "15min", string(Granularity15Min))
	assert.Equal(t, "hourly", string(GranularityHourly))
	assert.Equal(t, "daily", string(GranularityDaily))
	assert.Equal(t, "weekly", string(GranularityWeekly))
	assert.Equal(t, "monthly", string(GranularityMonthly))
}
