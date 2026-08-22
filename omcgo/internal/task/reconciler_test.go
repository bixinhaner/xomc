package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeActiveLister 返回预设的 PG 活跃态任务（pending/sent）。
type fakeActiveLister struct {
	mu        sync.Mutex
	active    []*Task
	listErr   error
	callCount int
	lastOlder time.Time
	lastLimit int
}

func (f *fakeActiveLister) ListActiveTasks(_ context.Context, olderThan time.Time, limit int) ([]*Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	f.lastOlder = olderThan
	f.lastLimit = limit
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.active, nil
}

type fakeCursorActiveLister struct {
	mu          sync.Mutex
	first       []*Task
	second      []*Task
	afterCalls  []*ActiveTaskCursor
	failAfterID string
	failOnce    bool
	legacyCalls int
}

func (f *fakeCursorActiveLister) ListActiveTasks(
	_ context.Context,
	_ time.Time,
	_ int,
) ([]*Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.legacyCalls++
	return f.first, nil
}

func (f *fakeCursorActiveLister) ListActiveTasksAfter(
	_ context.Context,
	_ time.Time,
	after *ActiveTaskCursor,
	_ int,
) ([]*Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var copied *ActiveTaskCursor
	if after != nil {
		value := *after
		copied = &value
	}
	f.afterCalls = append(f.afterCalls, copied)
	if after == nil {
		return f.first, nil
	}
	if after.ID == f.failAfterID && f.failOnce {
		f.failOnce = false
		return nil, errors.New("list active page failed")
	}
	if len(f.first) > 0 && after.ID == f.first[len(f.first)-1].ID {
		return f.second, nil
	}
	return nil, nil
}

func (f *fakeCursorActiveLister) observedAfterIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.afterCalls))
	for _, after := range f.afterCalls {
		if after == nil {
			out = append(out, "")
			continue
		}
		out = append(out, after.ID)
	}
	return out
}

// fakeQueueReader 按 taskID 返回 Redis 真相；可注入读错误。
type fakeQueueReader struct {
	mu      sync.Mutex
	byID    map[string]*Task
	readErr error
	calls   int
}

func (f *fakeQueueReader) GetByID(_ context.Context, taskID string) (*Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.byID[taskID], nil
}

// fakeRepairer 记录每次 Update 调用；可注入失败。
type fakeRepairer struct {
	mu       sync.Mutex
	updated  []*Task
	failNext bool
	failErr  error
}

func (f *fakeRepairer) Update(_ context.Context, t *Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return f.failErr
	}
	f.updated = append(f.updated, t)
	return nil
}

func (f *fakeRepairer) updatedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.updated))
	for _, t := range f.updated {
		ids = append(ids, t.ID)
	}
	return ids
}

type fakeConditionalRepairer struct {
	mu           sync.Mutex
	transitioned []*Task
	fromStatuses []TaskStatus
}

func (f *fakeConditionalRepairer) Update(_ context.Context, _ *Task) error {
	return nil
}

func (f *fakeConditionalRepairer) TransitionIfStatus(
	_ context.Context,
	t *Task,
	from TaskStatus,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transitioned = append(f.transitioned, t)
	f.fromStatuses = append(f.fromStatuses, from)
	return true, nil
}

func (f *fakeConditionalRepairer) GetByID(_ context.Context, _ string) (*Task, error) {
	return nil, nil
}

func pgActiveTask(id, sn string, status TaskStatus) *Task {
	return &Task{ID: id, DeviceSN: sn, Method: "SetParameterValues", Status: status}
}

func redisTerminalTask(id, sn string, status TaskStatus) *Task {
	now := time.Now()
	return &Task{ID: id, DeviceSN: sn, Method: "SetParameterValues", Status: status, CompletedAt: &now}
}

func Test_Reconciler_CursorAdvancesAndWrapsAcrossRounds(t *testing.T) {
	first := pgActiveTask("t1", "SN1", TaskStatusPending)
	first.CreatedAt = time.Date(2026, 7, 31, 1, 0, 0, 0, time.UTC)
	second := pgActiveTask("t2", "SN2", TaskStatusSent)
	second.CreatedAt = first.CreatedAt.Add(time.Second)
	lister := &fakeCursorActiveLister{
		first:  []*Task{first},
		second: []*Task{second},
	}
	rc := NewReconciler(
		lister,
		&fakeQueueReader{byID: map[string]*Task{}},
		&fakeRepairer{},
		nil,
		0,
		0,
		1,
		zap.NewNop(),
	)

	for range 4 {
		_, err := rc.ReconcileOnce(context.Background())
		require.NoError(t, err)
	}

	assert.Equal(t, []string{"", "t1", "t2", ""}, lister.observedAfterIDs())
	assert.Zero(t, lister.legacyCalls, "cursor-capable lister must not use the legacy first page")
}

func Test_Reconciler_CursorDoesNotAdvanceWhenPageQueryFails(t *testing.T) {
	first := pgActiveTask("t1", "SN1", TaskStatusPending)
	first.CreatedAt = time.Date(2026, 7, 31, 1, 0, 0, 0, time.UTC)
	second := pgActiveTask("t2", "SN2", TaskStatusSent)
	second.CreatedAt = first.CreatedAt.Add(time.Second)
	lister := &fakeCursorActiveLister{
		first:       []*Task{first},
		second:      []*Task{second},
		failAfterID: first.ID,
		failOnce:    true,
	}
	rc := NewReconciler(
		lister,
		&fakeQueueReader{byID: map[string]*Task{}},
		&fakeRepairer{},
		nil,
		0,
		0,
		1,
		zap.NewNop(),
	)

	_, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)
	_, err = rc.ReconcileOnce(context.Background())
	require.ErrorContains(t, err, "list active page failed")
	_, err = rc.ReconcileOnce(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{"", "t1", "t1"}, lister.observedAfterIDs())
}

// counterValue 读出某 outcome 标签下的 counter 当前值（测试断言用）。
func counterValue(t *testing.T, vec *prometheus.CounterVec, label string) float64 {
	t.Helper()
	m := &dto.Metric{}
	c, err := vec.GetMetricWithLabelValues(label)
	require.NoError(t, err)
	require.NoError(t, c.Write(m))
	return m.GetCounter().GetValue()
}

func Test_ReconcileOnce_RepairsPGStaleVsRedisTerminal(t *testing.T) {
	// PG 认为 t1 still pending、t2 still sent；Redis 真相是两者都已 completed/failed → 应修复。
	lister := &fakeActiveLister{active: []*Task{
		pgActiveTask("t1", "SN1", TaskStatusPending),
		pgActiveTask("t2", "SN2", TaskStatusSent),
	}}
	reader := &fakeQueueReader{byID: map[string]*Task{
		"t1": redisTerminalTask("t1", "SN1", TaskStatusCompleted),
		"t2": redisTerminalTask("t2", "SN2", TaskStatusFailed),
	}}
	repairer := &fakeRepairer{}
	metrics := NewTaskMetrics(prometheus.NewRegistry())
	rc := NewReconciler(lister, reader, repairer, metrics, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Scanned)
	assert.Equal(t, 2, stats.Repaired)
	assert.Equal(t, 0, stats.RepairFailed)
	assert.ElementsMatch(t, []string{"t1", "t2"}, repairer.updatedIDs())

	// 回写 PG 的对象必须带 Redis 终态，而非 PG 的旧状态。
	for _, u := range repairer.updated {
		assert.True(t, isTerminal(u.Status), "repaired task must carry terminal status")
	}
	assert.Equal(t, float64(2), counterValue(t, metrics.ReconcileTotal, "repaired"))
}

func Test_ReconcileOnce_RepairsLegacyRedisTerminalWithoutDeviceSN(t *testing.T) {
	lister := &fakeActiveLister{active: []*Task{
		pgActiveTask("t1", "SN1", TaskStatusSent),
	}}
	reader := &fakeQueueReader{byID: map[string]*Task{
		"t1": redisTerminalTask("t1", "", TaskStatusCompleted),
	}}
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, stats.Repaired)
	require.Len(t, repairer.updated, 1)
	assert.Equal(t, "SN1", repairer.updated[0].DeviceSN)
	assert.Equal(t, TaskStatusCompleted, repairer.updated[0].Status)
	assert.Empty(t, reader.byID["t1"].DeviceSN, "redis snapshot must not be mutated")
}

func Test_ReconcileOnce_ConditionalRepairCopiesDeviceSNFromDurableTask(t *testing.T) {
	lister := &fakeActiveLister{active: []*Task{
		pgActiveTask("t1", "SN1", TaskStatusSent),
	}}
	reader := &fakeQueueReader{byID: map[string]*Task{
		"t1": redisTerminalTask("t1", "", TaskStatusCompleted),
	}}
	repairer := &fakeConditionalRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, stats.Repaired)
	require.Len(t, repairer.transitioned, 1)
	assert.Equal(t, "SN1", repairer.transitioned[0].DeviceSN)
	assert.Equal(t, TaskStatusCompleted, repairer.transitioned[0].Status)
	assert.Equal(t, []TaskStatus{TaskStatusSent}, repairer.fromStatuses)
	assert.Empty(t, reader.byID["t1"].DeviceSN, "redis snapshot must not be mutated")
}

// issue #20：分叉检出应记 stale-detection counter + recovery-action counter，
// 活跃任务积压应反映到 backlog gauge。
func Test_ReconcileOnce_LifecycleMetrics(t *testing.T) {
	lister := &fakeActiveLister{active: []*Task{
		pgActiveTask("t1", "SN1", TaskStatusPending), // 分叉（Redis 已 completed）
		pgActiveTask("t2", "SN2", TaskStatusSent),    // 无分叉（Redis 仍 sent）
		pgActiveTask("t3", "SN3", TaskStatusPending), // 分叉（Redis 已 failed）
	}}
	reader := &fakeQueueReader{byID: map[string]*Task{
		"t1": redisTerminalTask("t1", "SN1", TaskStatusCompleted),
		"t2": pgActiveTask("t2", "SN2", TaskStatusSent),
		"t3": redisTerminalTask("t3", "SN3", TaskStatusFailed),
	}}
	repairer := &fakeRepairer{}
	metrics := NewTaskMetrics(prometheus.NewRegistry())
	rc := NewReconciler(lister, reader, repairer, metrics, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, stats.Scanned)
	assert.Equal(t, 2, stats.Repaired)

	// backlog gauge = 本轮扫描到的活跃任务数。
	m := &dto.Metric{}
	require.NoError(t, metrics.BacklogTotal.Write(m))
	assert.Equal(t, float64(3), m.GetGauge().GetValue(), "backlog gauge 应为扫描到的活跃任务数")

	// stale-detection counter = 检出分叉的任务数（2 条）。
	sm := &dto.Metric{}
	require.NoError(t, metrics.StaleDetectedTotal.Write(sm))
	assert.Equal(t, float64(2), sm.GetCounter().GetValue(), "应检出 2 条分叉")

	// recovery-action counter（reconcile_repair）= 成功修复数（2 条）。
	assert.Equal(t, float64(2),
		counterValue(t, metrics.RecoveryActionTotal, RecoveryActionReconcileRepair))
}

func Test_ReconcileOnce_NoDivergence_WhenRedisStillActive(t *testing.T) {
	// Redis 也还是 pending/sent → 无分叉，不修复（正常在飞任务）。
	lister := &fakeActiveLister{active: []*Task{
		pgActiveTask("t1", "SN1", TaskStatusPending),
		pgActiveTask("t2", "SN2", TaskStatusSent),
	}}
	reader := &fakeQueueReader{byID: map[string]*Task{
		"t1": pgActiveTask("t1", "SN1", TaskStatusPending),
		"t2": pgActiveTask("t2", "SN2", TaskStatusSent),
	}}
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Scanned)
	assert.Equal(t, 0, stats.Repaired)
	assert.Empty(t, repairer.updatedIDs(), "无分叉时不应回写 PG")
}

func Test_ReconcileOnce_NoDivergence_WhenRedisMissing(t *testing.T) {
	// Redis 无记录（TTL 过期 / rollback 孤儿）→ 不在对账器职责内（Restore/Sweeper 兜底）。
	lister := &fakeActiveLister{active: []*Task{pgActiveTask("t1", "SN1", TaskStatusPending)}}
	reader := &fakeQueueReader{byID: map[string]*Task{}} // GetByID 返回 nil,nil
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Repaired)
	assert.Empty(t, repairer.updatedIDs())
}

func Test_ReconcileOnce_ListerError_IsFatal(t *testing.T) {
	lister := &fakeActiveLister{listErr: errors.New("pg connection refused")}
	reader := &fakeQueueReader{byID: map[string]*Task{}}
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, stats.Scanned)
	assert.Empty(t, repairer.updatedIDs())
}

func Test_ReconcileOnce_RedisReadError_SkipsTaskNotFatal(t *testing.T) {
	// Redis 读失败：本轮跳过该任务，不致命（待 Redis 恢复后再对账）。
	lister := &fakeActiveLister{active: []*Task{pgActiveTask("t1", "SN1", TaskStatusPending)}}
	reader := &fakeQueueReader{readErr: errors.New("redis dial timeout")}
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err, "Redis 读失败不应是致命错误")
	assert.Equal(t, 1, stats.Scanned)
	assert.Equal(t, 0, stats.Repaired)
	assert.Empty(t, repairer.updatedIDs())
}

func Test_ReconcileOnce_RepairError_CountedAndContinues(t *testing.T) {
	// 第一条 PG 回写失败，第二条仍被处理（单条失败不中断整轮）。
	lister := &fakeActiveLister{active: []*Task{
		pgActiveTask("t1", "SN1", TaskStatusPending),
		pgActiveTask("t2", "SN2", TaskStatusSent),
	}}
	reader := &fakeQueueReader{byID: map[string]*Task{
		"t1": redisTerminalTask("t1", "SN1", TaskStatusCompleted),
		"t2": redisTerminalTask("t2", "SN2", TaskStatusCompleted),
	}}
	repairer := &fakeRepairer{failNext: true, failErr: errors.New("pg update conflict")}
	metrics := NewTaskMetrics(prometheus.NewRegistry())
	rc := NewReconciler(lister, reader, repairer, metrics, 0, 0, 0, zap.NewNop())

	stats, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err, "lister 成功，单条回写失败不应致命")
	assert.Equal(t, 2, stats.Scanned)
	assert.Equal(t, 1, stats.Repaired, "第二条成功")
	assert.Equal(t, 1, stats.RepairFailed, "第一条失败计入")
	assert.Equal(t, []string{"t2"}, repairer.updatedIDs())
	assert.Equal(t, float64(1), counterValue(t, metrics.ReconcileTotal, "repaired"))
	assert.Equal(t, float64(1), counterValue(t, metrics.ReconcileTotal, "repair_failed"))
}

func Test_ReconcileOnce_PassesGraceAndBatchSize(t *testing.T) {
	lister := &fakeActiveLister{active: nil}
	reader := &fakeQueueReader{byID: map[string]*Task{}}
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, time.Minute, 90*time.Second, 42, zap.NewNop())

	before := time.Now().Add(-90 * time.Second)
	_, err := rc.ReconcileOnce(context.Background())
	require.NoError(t, err)
	after := time.Now().Add(-90 * time.Second)

	assert.Equal(t, 42, lister.lastLimit)
	assert.True(t, !lister.lastOlder.Before(before) && !lister.lastOlder.After(after),
		"olderThan 应为 now-grace")
}

func Test_NewReconciler_DefaultsApplied(t *testing.T) {
	rc := NewReconciler(&fakeActiveLister{}, &fakeQueueReader{}, &fakeRepairer{}, nil, 0, 0, 0, nil)
	assert.Equal(t, 30*time.Second, rc.interval)
	assert.Equal(t, 60*time.Second, rc.grace)
	assert.Equal(t, 100, rc.batchSize)
	assert.NotNil(t, rc.logger)
}

func Test_Reconciler_Run_StopsOnContextCancel(t *testing.T) {
	lister := &fakeActiveLister{active: nil}
	reader := &fakeQueueReader{byID: map[string]*Task{}}
	repairer := &fakeRepairer{}
	rc := NewReconciler(lister, reader, repairer, nil, 20*time.Millisecond, time.Second, 10, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		rc.Run(ctx)
		close(done)
	}()

	time.Sleep(60 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run 未在 ctx 取消后退出")
	}

	lister.mu.Lock()
	calls := lister.callCount
	lister.mu.Unlock()
	assert.GreaterOrEqual(t, calls, 1, "应至少跑过一轮对账")
}

// Test_Reconciler_ConcurrentRepairAndExpire 验证 Reconciler 修复分叉的同时，
// ExpiredSweeper / 其它写者并发触达同一批任务时不发生数据竞争（-race 下运行）。
// 模拟 #13 的并发失败路径：双写 sync 与对账器同时操作。
func Test_Reconciler_ConcurrentRepairAndExpire(t *testing.T) {
	const n = 50
	pgTasks := make([]*Task, 0, n)
	redisTruth := make(map[string]*Task, n)
	for i := 0; i < n; i++ {
		id := "task-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		pgTasks = append(pgTasks, pgActiveTask(id, "SN", TaskStatusSent))
		redisTruth[id] = redisTerminalTask(id, "SN", TaskStatusCompleted)
	}

	lister := &fakeActiveLister{active: pgTasks}
	reader := &fakeQueueReader{byID: redisTruth}
	repairer := &fakeRepairer{}
	metrics := NewTaskMetrics(prometheus.NewRegistry())
	rc := NewReconciler(lister, reader, repairer, metrics, 0, 0, 0, zap.NewNop())

	var wg sync.WaitGroup
	// 多个 goroutine 并发跑 ReconcileOnce（reconciler 须对并发安全 / 至少不竞态）。
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := rc.ReconcileOnce(context.Background())
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	// 每轮都应把全部 n 条判为分叉并尝试修复；并发下 repair 计数 = 4*n（幂等回写，安全）。
	assert.Equal(t, float64(4*n), counterValue(t, metrics.ReconcileTotal, "repaired"))
}
