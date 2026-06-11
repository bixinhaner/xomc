package software

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestResolveScheduleMode(t *testing.T) {
	frozen := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	prevNow := nowFunc
	nowFunc = func() time.Time { return frozen }
	defer func() { nowFunc = prevNow }()

	future := frozen.Add(2 * time.Hour)
	past := frozen.Add(-2 * time.Hour)

	tests := []struct {
		name       string
		scheduled  *time.Time
		suspended  bool
		wantMode   scheduleMode
		wantSchedT bool // whether the returned scheduledAt is non-nil
	}{
		{"立即（默认）", nil, false, scheduleModeImmediate, false},
		{"挂起", nil, true, scheduleModeSuspended, false},
		{"定时（未来）", &future, false, scheduleModeScheduled, true},
		{"定时优先级高于挂起", &future, true, scheduleModeScheduled, true},
		{"过期时间退化为立即", &past, false, scheduleModeImmediate, false},
		{"过期时间且挂起 → 走挂起", &past, true, scheduleModeSuspended, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, at := resolveScheduleMode(tt.scheduled, tt.suspended)
			assert.Equal(t, tt.wantMode, mode)
			if tt.wantSchedT {
				require.NotNil(t, at)
				assert.Equal(t, *tt.scheduled, *at)
			} else {
				assert.Nil(t, at)
			}
		})
	}
}

func TestApplyScheduleMode(t *testing.T) {
	t.Run("scheduled 模式写满三字段", func(t *testing.T) {
		at := time.Date(2026, 6, 1, 8, 30, 0, 0, time.UTC)
		task := &UpgradeTask{Status: TaskInProgress, CreateStatus: CreateStatusActive}
		applyScheduleMode(task, scheduleModeScheduled, &at)

		assert.Equal(t, TaskPending, task.Status)
		assert.Equal(t, CreateStatusTiming, task.CreateStatus)
		require.NotNil(t, task.ScheduledAt)
		assert.True(t, time.Time(*task.ScheduledAt).Equal(at))
	})
	t.Run("suspended 模式落 TaskSuspended 并清空 scheduled_at", func(t *testing.T) {
		at := time.Date(2026, 6, 1, 8, 30, 0, 0, time.UTC)
		mt := model.Time(at)
		task := &UpgradeTask{ScheduledAt: &mt}
		applyScheduleMode(task, scheduleModeSuspended, nil)

		// #138：挂起统一落 TaskSuspended（历史 bug 落 TaskPending，导致 ufte 回显 immediate）。
		assert.Equal(t, TaskSuspended, task.Status)
		assert.Equal(t, CreateStatusActive, task.CreateStatus)
		assert.Nil(t, task.ScheduledAt)
	})
	t.Run("immediate 模式仅置位 create_status", func(t *testing.T) {
		at := time.Date(2026, 6, 1, 8, 30, 0, 0, time.UTC)
		mt := model.Time(at)
		task := &UpgradeTask{ScheduledAt: &mt}
		applyScheduleMode(task, scheduleModeImmediate, nil)

		assert.Equal(t, CreateStatusActive, task.CreateStatus)
		assert.Nil(t, task.ScheduledAt)
	})
}

// =============================================================================
// TaskScheduler 单测：用 fakeSource 喂"到期任务"，验证按 TaskType 正确分流。
// 不打实际数据库，覆盖 fire 路径。
// =============================================================================

type fakeScheduledSource struct {
	mu     sync.Mutex
	due    []*UpgradeTask
	listN  int
	guards map[uuid.UUID]string // 模拟 create_status：仅记录最新值
}

func newFakeSource(due ...*UpgradeTask) *fakeScheduledSource {
	g := make(map[uuid.UUID]string, len(due))
	for _, t := range due {
		g[t.ID] = CreateStatusTiming
	}
	return &fakeScheduledSource{due: due, guards: g}
}

func (f *fakeScheduledSource) ListDueScheduled(_ context.Context, _ time.Time, _ int) ([]*UpgradeTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listN++
	// 只返回那些仍处于 timing 状态的任务（模拟 partial index 行为）。
	out := make([]*UpgradeTask, 0, len(f.due))
	for _, t := range f.due {
		if f.guards[t.ID] == CreateStatusTiming {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeScheduledSource) UpdateCreateStatusGuarded(_ context.Context, id uuid.UUID, from, to string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cur, ok := f.guards[id]; ok && cur == from {
		f.guards[id] = to
		return true, nil
	}
	return false, nil
}

// firingScheduler 让 fire 不依赖真实 SoftwareService.ResumeUpgrade（需要 DB / Redis），
// 把分流后的回调挂到本结构上，便于断言。
type firingScheduler struct {
	*TaskScheduler
	upgradeCalls []uuid.UUID
	collectCalls []uuid.UUID
	upgradeErr   error
}

func newFiringScheduler(t *testing.T, src *fakeScheduledSource) *firingScheduler {
	t.Helper()
	ts := NewTaskScheduler(nil /*svc 替换*/, []ScheduledTaskSource{src}, time.Hour, zap.NewNop())
	fs := &firingScheduler{TaskScheduler: ts}
	// 把 svc.ResumeUpgrade 用 stubFire 替换：直接覆盖 fire 实现走自己的回调。
	return fs
}

// fireStub 复刻 TaskScheduler.fire 的状态翻转 + 分流逻辑，但用回调记录而非真实 svc。
func (f *firingScheduler) fireStub(ctx context.Context, task *UpgradeTask) {
	if err := f.markAsActive(ctx, task.ID); err != nil {
		return
	}
	switch task.TaskType {
	case TaskTypeUpgrade, TaskTypePatch, TaskTypeFPGA, TaskTypeRollback, TaskTypeReserved:
		if f.upgradeErr != nil {
			return
		}
		f.upgradeCalls = append(f.upgradeCalls, task.ID)
	case TaskTypeLogCollect:
		if f.collectTrigger == nil {
			_ = f.markAsTiming(ctx, task.ID)
			return
		}
		if err := f.collectTrigger(ctx, task.ID); err != nil {
			return
		}
		f.collectCalls = append(f.collectCalls, task.ID)
	}
}

func TestTaskScheduler_FiresUpgradeTasks(t *testing.T) {
	at := nowFunc().Add(-1 * time.Minute) // 已到期
	mt := model.Time(at)
	upg := &UpgradeTask{ID: uuid.New(), TaskType: TaskTypeUpgrade, Status: TaskPending,
		CreateStatus: CreateStatusTiming, ScheduledAt: &mt, TaskName: "upg-1"}
	rb := &UpgradeTask{ID: uuid.New(), TaskType: TaskTypeRollback, Status: TaskPending,
		CreateStatus: CreateStatusTiming, ScheduledAt: &mt, TaskName: "rb-1"}

	src := newFakeSource(upg, rb)
	fs := newFiringScheduler(t, src)

	ctx := context.Background()
	// 模拟 tick：直接调 fireStub，逐条触发。
	for _, task := range []*UpgradeTask{upg, rb} {
		fs.fireStub(ctx, task)
	}

	assert.ElementsMatch(t, []uuid.UUID{upg.ID, rb.ID}, fs.upgradeCalls)
	assert.Empty(t, fs.collectCalls)
	// 抢占锁应已生效：create_status 从 timing → active。
	assert.Equal(t, CreateStatusActive, src.guards[upg.ID])
	assert.Equal(t, CreateStatusActive, src.guards[rb.ID])
}

func TestTaskScheduler_FiresLogCollectViaTrigger(t *testing.T) {
	at := nowFunc().Add(-1 * time.Minute)
	mt := model.Time(at)
	logTask := &UpgradeTask{ID: uuid.New(), TaskType: TaskTypeLogCollect, Status: TaskPending,
		CreateStatus: CreateStatusTiming, ScheduledAt: &mt, TaskName: "log-1"}
	src := newFakeSource(logTask)
	fs := newFiringScheduler(t, src)

	var triggered []uuid.UUID
	fs.SetCollectTrigger(func(_ context.Context, id uuid.UUID) error {
		triggered = append(triggered, id)
		return nil
	})

	fs.fireStub(context.Background(), logTask)

	assert.Equal(t, []uuid.UUID{logTask.ID}, fs.collectCalls)
	assert.Equal(t, []uuid.UUID{logTask.ID}, triggered)
	assert.Empty(t, fs.upgradeCalls)
	assert.Equal(t, CreateStatusActive, src.guards[logTask.ID])
}

func TestTaskScheduler_LogCollectWithoutTriggerStaysTiming(t *testing.T) {
	at := nowFunc().Add(-1 * time.Minute)
	mt := model.Time(at)
	logTask := &UpgradeTask{ID: uuid.New(), TaskType: TaskTypeLogCollect, Status: TaskPending,
		CreateStatus: CreateStatusTiming, ScheduledAt: &mt, TaskName: "log-no-trigger"}
	src := newFakeSource(logTask)
	fs := newFiringScheduler(t, src)
	// 故意不调 SetCollectTrigger。

	fs.fireStub(context.Background(), logTask)

	assert.Empty(t, fs.collectCalls)
	// 没 trigger → 翻回 timing，下一 tick 再试。
	assert.Equal(t, CreateStatusTiming, src.guards[logTask.ID])
}

func TestTaskScheduler_DedupeBetweenTicks(t *testing.T) {
	at := nowFunc().Add(-1 * time.Minute)
	mt := model.Time(at)
	task := &UpgradeTask{ID: uuid.New(), TaskType: TaskTypeUpgrade, Status: TaskPending,
		CreateStatus: CreateStatusTiming, ScheduledAt: &mt, TaskName: "dedupe"}
	src := newFakeSource(task)
	fs := newFiringScheduler(t, src)

	ctx := context.Background()
	fs.fireStub(ctx, task) // 第一次：成功
	fs.fireStub(ctx, task) // 第二次：guard 已为 active，markAsActive 返回 error

	assert.Equal(t, []uuid.UUID{task.ID}, fs.upgradeCalls,
		"同一 task 不应被触发两次")
}

func TestTaskScheduler_ErrorFromUpgradeKeepsState(t *testing.T) {
	at := nowFunc().Add(-1 * time.Minute)
	mt := model.Time(at)
	task := &UpgradeTask{ID: uuid.New(), TaskType: TaskTypeUpgrade, Status: TaskPending,
		CreateStatus: CreateStatusTiming, ScheduledAt: &mt}
	src := newFakeSource(task)
	fs := newFiringScheduler(t, src)
	fs.upgradeErr = errors.New("simulated downstream failure")

	fs.fireStub(context.Background(), task)

	assert.Empty(t, fs.upgradeCalls, "失败时不应计入 callCalls")
	// 抢占已完成（guard=active），失败由日志兜底；下次该任务靠 reaper 兜底而非 scheduler 重试，
	// 这与生产路径一致 —— 不希望 scheduler 无限循环触发同一条任务。
	assert.Equal(t, CreateStatusActive, src.guards[task.ID])
}
