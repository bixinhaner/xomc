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

// 4 个 runner 的契约（JobType / Source / Target / Granularity）与 worker/main.go 注册
// 时的依赖耦合，单测固定下来避免误改。

func Test_HourlyRunner_Wiring(t *testing.T) {
	r := NewHourlyRunner(nil)
	assert.Equal(t, "pm_aggregate_hourly", r.JobType())
	assert.Equal(t, "pm_metrics", r.Source())
	assert.Equal(t, "pm_metrics_hourly", r.Target())
	assert.Equal(t, metrics.GranularityHourly, r.Granularity())
}

func Test_DailyRunner_Wiring(t *testing.T) {
	r := NewDailyRunner(nil)
	assert.Equal(t, "pm_aggregate_daily", r.JobType())
	assert.Equal(t, "pm_metrics_hourly", r.Source())
	assert.Equal(t, "pm_metrics_daily", r.Target())
	assert.Equal(t, metrics.GranularityDaily, r.Granularity())
}

func Test_WeeklyRunner_Wiring(t *testing.T) {
	r := NewWeeklyRunner(nil)
	assert.Equal(t, "pm_aggregate_weekly", r.JobType())
	assert.Equal(t, "pm_metrics_daily", r.Source())
	assert.Equal(t, "pm_metrics_weekly", r.Target())
	assert.Equal(t, metrics.GranularityWeekly, r.Granularity())
}

func Test_MonthlyRunner_Wiring(t *testing.T) {
	r := NewMonthlyRunner(nil)
	assert.Equal(t, "pm_aggregate_monthly", r.JobType())
	assert.Equal(t, "pm_metrics_weekly", r.Source())
	assert.Equal(t, "pm_metrics_monthly", r.Target())
	assert.Equal(t, metrics.GranularityMonthly, r.Granularity())
}

// Runner.Run：payload 解析正确 + 两步调用顺序符合预期。
// 用 stubDB 让 AggregateCounters 走通；AggregateKPIs 没 kpiRouter 所以返 0。

func Test_Runner_Run_PayloadParsesAndCallsAggregator(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	aggr := New(db, nil, nil)
	r := NewHourlyRunner(aggr)

	job := &asyncjob.Job{
		ID:      uuid.New(),
		JobType: r.JobType(),
		Payload: payload,
	}
	result, err := r.Run(context.Background(), job)
	require.NoError(t, err)

	var resultMap map[string]any
	require.NoError(t, json.Unmarshal(result, &resultMap))
	assert.Equal(t, float64(5), resultMap["counter_rows"])
	assert.Equal(t, float64(0), resultMap["kpi_rows"]) // 无 kpiRouter
	assert.Equal(t, "pm_metrics", resultMap["source"])
	assert.Equal(t, "pm_metrics_hourly", resultMap["target"])
	assert.Equal(t, "hourly", resultMap["granularity"])

	// AggregateCounters 被调到底层 Exec 的 args 含 hourly + start + end
	assert.Contains(t, db.execSQL, "INSERT INTO pm_metrics_hourly")
	assert.Equal(t, []any{"hourly", start, end, start, end}, db.execArgs)
}

func Test_Runner_Run_RejectsMissingPayload(t *testing.T) {
	r := NewHourlyRunner(New(&stubDB{}, nil, nil))
	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: nil}
	_, err := r.Run(context.Background(), job)
	assert.Error(t, err)
}

func Test_Runner_Run_RejectsReverseRange(t *testing.T) {
	start := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC) // end < start
	payload, _ := BuildPayload(start, end)

	r := NewHourlyRunner(New(&stubDB{}, nil, nil))
	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	_, err := r.Run(context.Background(), job)
	assert.Error(t, err)
}
