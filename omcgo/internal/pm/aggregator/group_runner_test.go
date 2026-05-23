package aggregator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// 4 个 GroupRunner 的契约固定（JobType / DeviceTarget / GroupTarget / Granularity）—
// 与 worker/aggregator.go 装配时的依赖耦合，单测固定下来避免误改。

func Test_HourlyGroupRunner_Wiring(t *testing.T) {
	r := NewHourlyGroupRunner(nil)
	assert.Equal(t, "pm_aggregate_hourly_group", r.JobType())
	assert.Equal(t, "pm_metrics_hourly", r.DeviceTarget())
	assert.Equal(t, "pm_group_metrics_hourly", r.GroupTarget())
	assert.Equal(t, metrics.GranularityHourly, r.Granularity())
}

func Test_DailyGroupRunner_Wiring(t *testing.T) {
	r := NewDailyGroupRunner(nil)
	assert.Equal(t, "pm_aggregate_daily_group", r.JobType())
	assert.Equal(t, "pm_metrics_daily", r.DeviceTarget())
	assert.Equal(t, "pm_group_metrics_daily", r.GroupTarget())
	assert.Equal(t, metrics.GranularityDaily, r.Granularity())
}

func Test_WeeklyGroupRunner_Wiring(t *testing.T) {
	r := NewWeeklyGroupRunner(nil)
	assert.Equal(t, "pm_aggregate_weekly_group", r.JobType())
	assert.Equal(t, "pm_metrics_weekly", r.DeviceTarget())
	assert.Equal(t, "pm_group_metrics_weekly", r.GroupTarget())
	assert.Equal(t, metrics.GranularityWeekly, r.Granularity())
}

func Test_MonthlyGroupRunner_Wiring(t *testing.T) {
	r := NewMonthlyGroupRunner(nil)
	assert.Equal(t, "pm_aggregate_monthly_group", r.JobType())
	assert.Equal(t, "pm_metrics_monthly", r.DeviceTarget())
	assert.Equal(t, "pm_group_metrics_monthly", r.GroupTarget())
	assert.Equal(t, metrics.GranularityMonthly, r.Granularity())
}

// GroupRunner.Run 路径：payload 解析 → AggregateDeviceGroup
func Test_GroupRunner_Run_CallsAggregateDeviceGroup(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 7")}
	aggr := New(db, nil, nil)
	r := NewHourlyGroupRunner(aggr)

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	result, err := r.Run(context.Background(), job)
	require.NoError(t, err)

	var resultMap map[string]any
	require.NoError(t, json.Unmarshal(result, &resultMap))
	assert.Equal(t, float64(7), resultMap["rows"])
	assert.Equal(t, "pm_metrics_hourly", resultMap["device_target"])
	assert.Equal(t, "pm_group_metrics_hourly", resultMap["group_target"])

	// 底层 Exec SQL 应是 device_group 聚合（含 JOIN device_group_members）
	assert.Contains(t, db.execSQL, "INSERT INTO pm_group_metrics_hourly")
	assert.Contains(t, db.execSQL, "JOIN device_group_members")
}

func Test_GroupRunner_Run_RejectsMissingPayload(t *testing.T) {
	r := NewHourlyGroupRunner(New(&stubDB{}, nil, nil))
	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: nil}
	_, err := r.Run(context.Background(), job)
	assert.Error(t, err)
}
