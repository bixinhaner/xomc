package metrics

import (
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
