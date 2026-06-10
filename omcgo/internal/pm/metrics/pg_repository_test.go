package metrics

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
// LIMIT 边界（#11）：Query 经 buildQuerySQL 强制收口，防 OOM
// ---------------------------------------------------------------------------

func Test_clampLimit(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero falls back to default", 0, DefaultQueryLimit},
		{"negative falls back to default", -5, DefaultQueryLimit},
		{"in range passes through", 500, 500},
		{"exactly max passes through", MaxQueryLimit, MaxQueryLimit},
		{"over max clamped to max", MaxQueryLimit + 1, MaxQueryLimit},
		{"huge clamped to max", 1 << 30, MaxQueryLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clampLimit(tt.in))
		})
	}
}

// 成功路径：合理 Limit 原样下推。
func Test_buildQuerySQL_LimitInRange(t *testing.T) {
	sql, _, err := buildQuerySQL(QueryRequest{Limit: 200})
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT 200")
}

// 防 OOM 路径：Limit<=0 不再生成无上界查询，落 DefaultQueryLimit。
func Test_buildQuerySQL_NoLimit_ForcesDefault(t *testing.T) {
	sql, _, err := buildQuerySQL(QueryRequest{Limit: 0})
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT 1000")
	assert.NotContains(t, sql, "LIMIT 0", "Limit=0 必须落默认上界而非裸全表扫")
}

// 防 OOM 路径：超大 Limit 被收口到 MaxQueryLimit。
func Test_buildQuerySQL_OversizedLimit_ClampedToMax(t *testing.T) {
	sql, _, err := buildQuerySQL(QueryRequest{Limit: 5_000_000})
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT 100000")
	assert.NotContains(t, sql, "LIMIT 5000000")
}

// Offset 与 LIMIT 共存：分页游标仍可推进。
func Test_buildQuerySQL_WithOffset(t *testing.T) {
	sql, _, err := buildQuerySQL(QueryRequest{Limit: 100, Offset: 300})
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT 100")
	assert.Contains(t, sql, "OFFSET 300")
}

// 任意 Query 都必须带 LIMIT（即使空过滤），防止裸 Query() 拉全表。
func Test_buildQuerySQL_EmptyRequest_StillBounded(t *testing.T) {
	sql, _, err := buildQuerySQL(QueryRequest{})
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT")
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

// nil ObjectLDN 落参数时统一为 ”（migration 000171 要求 NOT NULL DEFAULT ”）。
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

// ---------------------------------------------------------------------------
// issue #14: COPY 路径行值与 VALUES INSERT 参数等价（数据写出一致性）
// ---------------------------------------------------------------------------

// buildRows（COPY 路径）每行的列值必须与 buildBatchInsertSQL（VALUES 路径）拆出的
// per-row 参数逐列一致 —— 两条路径写出的数据完全相同，只是传输方式不同。
// id / ingest_time 含随机 / 时钟默认值，单独按位置比对其余确定列。
func Test_buildRows_ParityWith_buildBatchInsertSQL(t *testing.T) {
	ms := []PMMetric{
		metricWithLDN("L.Cell.Avail", "cell-1", true),
		metricWithLDN("L.Cell.Drop", "cell-2", true),
		metricWithLDN("L.Cell.Att", "", false), // ObjectLDN nil → ''
	}

	rows, err := buildRows(ms)
	require.NoError(t, err)
	require.Len(t, rows, len(ms))

	_, args, err := buildBatchInsertSQL(ms)
	require.NoError(t, err)
	const cols = 14
	require.Len(t, args, cols*len(ms))

	// 列序：id, oui, sn, path, type, value, statis, gran, time, start, end, ingest, ldn, extra
	// 索引 0(id) 与 11(ingest_time) 是默认值列（uuid.New / time.Now），不参与确定性比对。
	for i := range ms {
		rowVals := rows[i]
		valuesArgs := args[i*cols : (i+1)*cols]
		require.Len(t, rowVals, cols)
		for col := 0; col < cols; col++ {
			if col == 0 || col == 11 {
				continue // id / ingest_time 默认值列
			}
			assert.EqualValues(t, valuesArgs[col], rowVals[col],
				"row %d col %d 必须 COPY 与 VALUES 一致", i, col)
		}
		// object_ldn（列 12）必须为非 NULL 字符串（含 nil→'' 兜底）。
		_, ok := rowVals[12].(string)
		assert.True(t, ok, "object_ldn 必须落字符串而非 nil")
	}
}

// buildRows 对 ObjectLDN nil 也落空字符串（与 VALUES 路径、UNIQUE 索引语义一致）。
func Test_buildRows_NilObjectLDN_EmptyString(t *testing.T) {
	rows, err := buildRows([]PMMetric{metricWithLDN("L.Cell.Avail", "", false)})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "", rows[0][12], "ObjectLDN=nil 必须落空字符串")
}

// 列序常量稳定性：pmMetricsColumns 必须与 buildBatchInsertSQL 的 14 列一致，
// 否则 COPY 写入列错位。
func Test_pmMetricsColumns_Count(t *testing.T) {
	require.Len(t, pmMetricsColumns, 14)
	assert.Equal(t, "id", pmMetricsColumns[0])
	assert.Equal(t, "object_ldn", pmMetricsColumns[12])
	assert.Equal(t, "extra", pmMetricsColumns[13])
}

// joinCols 生成无前后逗号的列清单。
func Test_joinCols(t *testing.T) {
	assert.Equal(t, "a, b, c", joinCols([]string{"a", "b", "c"}))
	assert.Equal(t, "x", joinCols([]string{"x"}))
	assert.Equal(t, "", joinCols(nil))
}

// ---------------------------------------------------------------------------
// issue #14: 迟到数据降级 — 压缩 chunk 错误分类
// ---------------------------------------------------------------------------

// 0A000（feature_not_supported，TimescaleDB 压缩 chunk 拒 ON CONFLICT）→ 识别为迟到数据。
func Test_isLateArrivalError_CompressedChunk(t *testing.T) {
	err := &pgconn.PgError{Code: sqlstateFeatureNotSupported, Message: "invalid ON CONFLICT clause on compressed chunk"}
	assert.True(t, isLateArrivalError(err))
	// errors.As 透过 wrap 也应命中。
	assert.True(t, isLateArrivalError(fmt.Errorf("exec failed: %w", err)))
}

// 非压缩类 PG 错误（如 unique_violation 23505）不被误判为迟到数据。
func Test_isLateArrivalError_OtherPgError(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	assert.False(t, isLateArrivalError(err))
}

// 普通非 PG 错误不被误判。
func Test_isLateArrivalError_PlainError(t *testing.T) {
	assert.False(t, isLateArrivalError(errors.New("connection refused")))
	assert.False(t, isLateArrivalError(nil))
}

// classifyInsertError：压缩 chunk → 包成 ErrLateArrival（errors.Is 可识别）。
func Test_classifyInsertError_LateArrival_WrapsSentinel(t *testing.T) {
	pgErr := &pgconn.PgError{Code: sqlstateFeatureNotSupported, Message: "compressed chunk"}
	got := classifyInsertError(pgErr)
	require.Error(t, got)
	assert.True(t, errors.Is(got, ErrLateArrival), "迟到数据错误必须 wrap ErrLateArrival 供上层 errors.Is 识别")
}

// classifyInsertError：其他错误不被误判为 ErrLateArrival，但仍被 wrap。
func Test_classifyInsertError_OtherError_NotLateArrival(t *testing.T) {
	got := classifyInsertError(errors.New("boom"))
	require.Error(t, got)
	assert.False(t, errors.Is(got, ErrLateArrival))
	assert.Contains(t, got.Error(), "insert pm_metrics")
}

// classifyInsertError(nil) == nil（成功路径不构造错误）。
func Test_classifyInsertError_Nil(t *testing.T) {
	assert.NoError(t, classifyInsertError(nil))
}

// 批量阈值常量稳定（防误改导致小批量也走建表开销 / 大批量回退慢路径）。
func Test_batchInsertThreshold_Stable(t *testing.T) {
	assert.Equal(t, 50, batchInsertThreshold)
}
