package metrics

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildQuerySQLTargetsDictionaryBeforeMetricSetExpansion(t *testing.T) {
	sql, args, err := buildQuerySQL(QueryRequest{MetricPaths: []string{"C1", "K1"}, Limit: 50})
	require.NoError(t, err)
	assert.Contains(t, sql, "d.metric_id=ANY(s.metric_ids)")
	assert.NotContains(t, sql, "unnest(s.metric_ids)")
	assert.Contains(t, sql, "d.metric_path IN")
	assert.NotEmpty(t, args)
}

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

// #64 设备组数据权限：applyFilters 按 device_sn 两层子查询 fail-closed 收口。
func Test_applyFilters_VisibleGroups_ThreeWay(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	t.Run("nil 超管不过滤", func(t *testing.T) {
		qb := storage.Psql.Select("*").From("pm_metrics")
		sql, args, err := applyFilters(qb, QueryRequest{VisibleGroups: nil}).ToSql()
		require.NoError(t, err)
		assert.Equal(t, "SELECT * FROM pm_metrics", sql)
		assert.Empty(t, args)
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		qb := storage.Psql.Select("*").From("pm_metrics")
		sql, _, err := applyFilters(qb, QueryRequest{VisibleGroups: []uuid.UUID{}}).ToSql()
		require.NoError(t, err)
		assert.Contains(t, sql, "FALSE")
	})

	t.Run("限定到可见分组下设备（sn 两层子查询）", func(t *testing.T) {
		qb := storage.Psql.Select("*").From("pm_metrics")
		sql, args, err := applyFilters(qb, QueryRequest{VisibleGroups: []uuid.UUID{g1, g2}}).ToSql()
		require.NoError(t, err)
		assert.Contains(t, sql, "device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id IN (")
		assert.Contains(t, args, g1)
		assert.Contains(t, args, g2)
	})
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
	assert.Contains(t, sql, "time < $2")
	assert.NotContains(t, sql, "time <= $2")
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
	assert.Contains(t, sql, "time < ")
	assert.NotContains(t, sql, "time <= ")
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
// MetricType / StatisType / Granularity 常量稳定性（避免误改字符串值）
// ---------------------------------------------------------------------------

func Test_Constants_Stable(t *testing.T) {
	assert.Equal(t, "counter", string(MetricTypeCounter))
	assert.Equal(t, "kpi", string(MetricTypeKPI))
	assert.Equal(t, "sum", string(StatisSum))
	assert.Equal(t, "avg", string(StatisAvg))
	assert.Equal(t, "max", string(StatisMax))
	assert.Equal(t, "min", string(StatisMin))
	assert.Equal(t, "pct", string(StatisPct))
	assert.Equal(t, "15min", string(Granularity15Min))
	assert.Equal(t, "hourly", string(GranularityHourly))
	assert.Equal(t, "daily", string(GranularityDaily))
	assert.Equal(t, "weekly", string(GranularityWeekly))
	assert.Equal(t, "monthly", string(GranularityMonthly))
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
