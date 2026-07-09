package aggregator

import (
	"context"
	"encoding/json"
	"errors"
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
	assert.Equal(t, "pm_metrics_daily", r.Source())
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
	assert.Equal(t, []any{"hourly", start, end, start, end, ""}, db.execArgs)
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

func shanghaiLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	return loc
}

func Test_Runner_Run_ChainsDailyAfterBusinessDayLastHourlyBucket(t *testing.T) {
	loc := shanghaiLoc(t)
	start := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC) // 北京 2026-07-07 23:00
	end := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)   // 北京 2026-07-08 00:00
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}
	enq := &stubEnqueuer{}
	r := NewHourlyRunner(New(db, nil, nil))
	r.SetRollupChain(enq, func() *time.Location { return loc })

	_, err = r.Run(context.Background(), &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload})
	require.NoError(t, err)
	require.Len(t, enq.calls, 1)
	require.Equal(t, JobTypeDaily, enq.calls[0].JobType)
	require.NotNil(t, enq.calls[0].BucketStart)
	require.NotNil(t, enq.calls[0].BucketEnd)
	require.True(t, enq.calls[0].BucketStart.Equal(time.Date(2026, 7, 6, 16, 0, 0, 0, time.UTC)))
	require.True(t, enq.calls[0].BucketEnd.Equal(time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)))
}

func Test_Runner_Run_DoesNotChainDailyForMiddleHourlyBucket(t *testing.T) {
	loc := shanghaiLoc(t)
	start := time.Date(2026, 7, 7, 14, 0, 0, 0, time.UTC) // 北京 22:00
	end := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)   // 北京 23:00
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}
	enq := &stubEnqueuer{}
	r := NewHourlyRunner(New(db, nil, nil))
	r.SetRollupChain(enq, func() *time.Location { return loc })

	_, err = r.Run(context.Background(), &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload})
	require.NoError(t, err)
	require.Empty(t, enq.calls)
}

func Test_Runner_Run_ChainsWeeklyAndMonthlyFromDailyBoundaries(t *testing.T) {
	loc := shanghaiLoc(t)
	// 北京 2026-06-28(日) 00:00 到 2026-06-29(一) 00:00，是 ISO 业务周最后一天。
	start := time.Date(2026, 6, 27, 16, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 28, 16, 0, 0, 0, time.UTC)
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}
	enq := &stubEnqueuer{}
	r := NewDailyRunner(New(db, nil, nil))
	r.SetRollupChain(enq, func() *time.Location { return loc })

	_, err = r.Run(context.Background(), &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload})
	require.NoError(t, err)
	require.Len(t, enq.calls, 1)
	require.Equal(t, JobTypeWeekly, enq.calls[0].JobType)
	require.True(t, enq.calls[0].BucketStart.Equal(time.Date(2026, 6, 21, 16, 0, 0, 0, time.UTC)))
	require.True(t, enq.calls[0].BucketEnd.Equal(time.Date(2026, 6, 28, 16, 0, 0, 0, time.UTC)))

	// 北京 2026-06-30 到 2026-07-01，是自然月最后一天。
	enq.calls = nil
	monthPayload, err := BuildPayload(
		time.Date(2026, 6, 29, 16, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	_, err = r.Run(context.Background(), &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: monthPayload})
	require.NoError(t, err)
	require.Len(t, enq.calls, 1)
	require.Equal(t, JobTypeMonthly, enq.calls[0].JobType)
	require.True(t, enq.calls[0].BucketStart.Equal(time.Date(2026, 5, 31, 16, 0, 0, 0, time.UTC)))
	require.True(t, enq.calls[0].BucketEnd.Equal(time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC)))
}

func Test_Runner_Run_RollupChainFailureFailsCurrentJob(t *testing.T) {
	loc := shanghaiLoc(t)
	payload, err := BuildPayload(
		time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}
	r := NewHourlyRunner(New(db, nil, nil))
	r.SetRollupChain(&stubEnqueuer{failErr: errors.New("queue unavailable")}, func() *time.Location { return loc })

	_, err = r.Run(context.Background(), &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload})
	require.Error(t, err)
	require.Contains(t, err.Error(), "enqueue rollup chain")
}
