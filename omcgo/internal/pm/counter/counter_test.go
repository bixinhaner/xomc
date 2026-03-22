package counter

import (
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test applyCounterFilters — builds correct SQL WHERE clauses
// ---------------------------------------------------------------------------

func Test_applyCounterFilters_NoFilters(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	filter := CounterFilter{}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM pm_counters", sql)
	assert.Empty(t, args)
}

func Test_applyCounterFilters_DeviceID(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	deviceID := uuid.New()
	filter := CounterFilter{DeviceID: &deviceID}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "device_id = $1")
	assert.Len(t, args, 1)
}

func Test_applyCounterFilters_CellID(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	cellID := "cell-1"
	filter := CounterFilter{CellID: &cellID}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "cell_id = $1")
	assert.Equal(t, "cell-1", args[0])
}

func Test_applyCounterFilters_CounterGroup(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	group := "LTE.CellMeasReport"
	filter := CounterFilter{CounterGroup: &group}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "counter_group = $1")
	assert.Equal(t, "LTE.CellMeasReport", args[0])
}

func Test_applyCounterFilters_CounterName(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	name := "PRB.UlAvailProcMeas"
	filter := CounterFilter{CounterName: &name}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "counter_name = $1")
	assert.Equal(t, "PRB.UlAvailProcMeas", args[0])
}

func Test_applyCounterFilters_TimeRange(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	start := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 22, 23, 59, 59, 0, time.UTC)
	filter := CounterFilter{
		StartTime: start,
		EndTime:   end,
	}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "time >= $1")
	assert.Contains(t, sql, "time <= $2")
	assert.Equal(t, start, args[0])
	assert.Equal(t, end, args[1])
}

func Test_applyCounterFilters_AllFilters(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	deviceID := uuid.New()
	cellID := "cell-1"
	group := "LTE.CellMeasReport"
	name := "PRB.UlAvailProcMeas"
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)

	filter := CounterFilter{
		DeviceID:     &deviceID,
		CellID:       &cellID,
		CounterGroup: &group,
		CounterName:  &name,
		StartTime:    start,
		EndTime:      end,
	}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "device_id = $1")
	assert.Contains(t, sql, "cell_id = $2")
	assert.Contains(t, sql, "counter_group = $3")
	assert.Contains(t, sql, "counter_name = $4")
	assert.Contains(t, sql, "time >= $5")
	assert.Contains(t, sql, "time <= $6")
	assert.Len(t, args, 6)
}

func Test_applyCounterFilters_ZeroTimeIgnored(t *testing.T) {
	qb := psql.Select("*").From("pm_counters")
	filter := CounterFilter{
		StartTime: time.Time{}, // zero value
		EndTime:   time.Time{}, // zero value
	}

	result := applyCounterFilters(qb, filter)

	sql, args, err := result.ToSql()
	require.NoError(t, err)
	assert.NotContains(t, sql, "time")
	assert.Empty(t, args)
}

// ---------------------------------------------------------------------------
// Test allowedSortColumns — SQL injection prevention
// ---------------------------------------------------------------------------

func Test_allowedSortColumns_ValidColumns(t *testing.T) {
	validColumns := []string{"time", "counter_name", "counter_value", "device_id", "created_at"}

	for _, col := range validColumns {
		t.Run(col, func(t *testing.T) {
			assert.True(t, allowedSortColumns[col], "column %s should be allowed", col)
		})
	}
}

func Test_allowedSortColumns_InvalidColumns(t *testing.T) {
	invalidColumns := []string{
		"DROP TABLE",
		"counter_group",
		"cell_id",
		"1; DROP TABLE pm_counters",
		"",
	}

	for _, col := range invalidColumns {
		t.Run(col, func(t *testing.T) {
			assert.False(t, allowedSortColumns[col], "column %s should not be allowed", col)
		})
	}
}

// ---------------------------------------------------------------------------
// Test CounterFilter struct — embedded ListRequest
// ---------------------------------------------------------------------------

func Test_CounterFilter_WithListRequest(t *testing.T) {
	filter := CounterFilter{
		ListRequest: model.ListRequest{
			Page:     2,
			PageSize: 10,
			SortBy:   "time",
			SortDir:  "asc",
		},
	}

	assert.Equal(t, 10, filter.Offset(), "page 2, size 10 should offset 10")
	assert.Equal(t, 10, filter.Limit())
}

func Test_CounterFilter_DefaultListRequest(t *testing.T) {
	filter := CounterFilter{
		ListRequest: model.DefaultListRequest(),
	}

	assert.Equal(t, 0, filter.Offset(), "page 1 should offset 0")
	assert.Equal(t, 20, filter.Limit())
}

// ---------------------------------------------------------------------------
// Test AggregatedCounter struct — verify fields
// ---------------------------------------------------------------------------

func Test_AggregatedCounter_Struct(t *testing.T) {
	now := time.Now()
	deviceID := uuid.New()

	ac := AggregatedCounter{
		Bucket:       now,
		DeviceID:     deviceID,
		CellID:       "cell-1",
		CounterGroup: "LTE.CellMeasReport",
		CounterName:  "PRB.UlAvailProcMeas",
		SumValue:     100.0,
		AvgValue:     25.0,
		MinValue:     10.0,
		MaxValue:     50.0,
		SampleCount:  4,
	}

	assert.Equal(t, now, ac.Bucket)
	assert.Equal(t, deviceID, ac.DeviceID)
	assert.Equal(t, "cell-1", ac.CellID)
	assert.Equal(t, 100.0, ac.SumValue)
	assert.Equal(t, 25.0, ac.AvgValue)
	assert.Equal(t, 10.0, ac.MinValue)
	assert.Equal(t, 50.0, ac.MaxValue)
	assert.Equal(t, int64(4), ac.SampleCount)
}

// ---------------------------------------------------------------------------
// Test psql statement builder — uses Dollar placeholder format
// ---------------------------------------------------------------------------

func Test_psql_DollarPlaceholder(t *testing.T) {
	sql, args, err := psql.Select("counter_name").
		From("pm_counters").
		Where(squirrel.Eq{"device_id": "test-id"}).
		ToSql()

	require.NoError(t, err)
	assert.Contains(t, sql, "$1", "should use dollar placeholders")
	assert.NotContains(t, sql, "?", "should not use question mark placeholders")
	assert.Equal(t, "test-id", args[0])
}
