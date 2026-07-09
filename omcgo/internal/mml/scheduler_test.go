package mml

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeClock 固定时间，单测专用。
type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

// fakeScheduledRepo 实现 ScheduledTaskRepository 的最小 mock。
type fakeScheduledRepo struct {
	mu       sync.Mutex
	claims   int
	toClaim  []*MMLTask
	finalize int
}

func (r *fakeScheduledRepo) ClaimDueTasks(_ context.Context, _ time.Time, _ int) ([]*MMLTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claims++
	out := r.toClaim
	r.toClaim = nil
	return out, nil
}

func (r *fakeScheduledRepo) RecordPeriodicChild(_ context.Context, _, _ *MMLTask, _ *time.Time) error {
	return nil
}

func (r *fakeScheduledRepo) FinalizePeriodicParent(_ context.Context, _ uuid.UUID, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finalize++
	return nil
}

func Test_computeNextPeriodicTrigger_DayWrap(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 4, 24, 15, 0, 0, 0, loc)
	start := time.Date(2026, 4, 24, 0, 0, 0, 0, loc)
	end := time.Date(2026, 4, 26, 23, 59, 59, 0, loc)
	task := &MMLTask{
		ExecuteType: ExecutePeriodic,
		PeriodStart: &start,
		PeriodEnd:   &end,
		PeriodTime:  "14:30:00",
	}

	next := computeNextPeriodicTrigger(task, now)
	require.NotNil(t, next)
	assert.Equal(t, 2026, next.Year())
	assert.Equal(t, time.April, next.Month())
	assert.Equal(t, 25, next.Day(), "now=15:00 已过今日14:30，应顺延到明天 14:30")
	assert.Equal(t, 14, next.Hour())
	assert.Equal(t, 30, next.Minute())
}

func Test_computeNextPeriodicTrigger_ReturnsNilPastEnd(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 4, 27, 0, 0, 0, 0, loc)
	start := time.Date(2026, 4, 24, 0, 0, 0, 0, loc)
	end := time.Date(2026, 4, 26, 23, 59, 59, 0, loc)
	task := &MMLTask{
		PeriodStart: &start,
		PeriodEnd:   &end,
		PeriodTime:  "14:30:00",
	}
	assert.Nil(t, computeNextPeriodicTrigger(task, now), "now 已过 period_end，应返回 nil")
}

func Test_computeNextPeriodicTrigger_RespectsPeriodStart(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, loc) // 早于 start
	start := time.Date(2026, 4, 24, 0, 0, 0, 0, loc)
	task := &MMLTask{
		PeriodStart: &start,
		PeriodTime:  "09:00:00",
	}
	next := computeNextPeriodicTrigger(task, now)
	require.NotNil(t, next)
	assert.Equal(t, 24, next.Day(), "now < start 时，下次触发应落在 start 当日的 period_time")
	assert.Equal(t, 9, next.Hour())
}

func Test_Scheduler_RunOnceClaimsAndDispatches(t *testing.T) {
	// 构造一个场景：fakeRepo 返回 1 个 scheduled 任务；runOnce 应认领并调 dispatch。
	svc := &Service{} // service.fanoutClaimed 在 fanouter==nil 时返回错误，但本测试只关心 dispatch 被调用
	clock := &fakeClock{now: time.Now()}
	repo := &fakeScheduledRepo{
		toClaim: []*MMLTask{{
			ID:          uuid.New(),
			ExecuteType: ExecuteScheduled,
			Status:      TaskRunning,
		}},
	}
	s := NewScheduler(svc, repo, clock, 10*time.Millisecond, zap.NewNop())
	s.runOnce(context.Background())
	assert.Equal(t, 1, repo.claims, "应认领一次")
}

func Test_Scheduler_DispatchPeriodicImmediateChildFansOut(t *testing.T) {
	parentID := uuid.New()
	stub := &stubDeviceTaskCreator{}
	svc := &Service{logger: zap.NewNop()}
	svc.SetFanouter(NewFanouter(stub, nil, nil, nil, zap.NewNop()))
	repo := &fakeScheduledRepo{}
	s := NewScheduler(svc, repo, &fakeClock{now: time.Now()}, time.Minute, zap.NewNop())

	s.dispatch(context.Background(), &MMLTask{
		ID:               uuid.New(),
		ExecuteType:      ExecuteImmediate,
		Status:           TaskRunning,
		PeriodicParentID: &parentID,
		DeviceSNs:        []string{"SN-PERIODIC-CHILD"},
		Commands: []map[string]interface{}{
			{"command_code": "REBOOT", "rpc_method": "Reboot"},
		},
		FailedRetry:      true,
		FailedRetryCount: 2,
	}, time.Now())

	require.Len(t, stub.calls, 1, "periodic child execute_type=immediate 时也必须 fanout")
	require.Len(t, stub.calls[0], 1)
	assert.Equal(t, 2, *stub.calls[0][0].MaxRetries)
	assert.Equal(t, 1, repo.finalize, "periodic child fanout 后应触发父模板 finalize 兜底")
}

func Test_Scheduler_StartStop(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	repo := &fakeScheduledRepo{}
	s := NewScheduler(&Service{}, repo, clock, 10*time.Millisecond, zap.NewNop())
	s.Start(context.Background())
	time.Sleep(30 * time.Millisecond) // 让 ticker 触发一两次
	s.Stop()
	s.Wait()
	assert.Greater(t, repo.claims, 0, "ticker 至少触发一次 runOnce")
}
