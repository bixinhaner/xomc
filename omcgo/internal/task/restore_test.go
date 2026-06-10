package task

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// fakePendingLister 实现 pendingTaskLister，用于单测。
type fakePendingLister struct {
	tasks []*Task
	err   error
}

func (f *fakePendingLister) ListPendingAllDevices(_ context.Context, _ int) ([]*Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tasks, nil
}

// fakePagingLister 实现 pendingTaskLister + pendingPageLister，用于验证 #11 流式恢复路径。
// pages 是按页预置的返回；recordedCursors 记录每次 ListPendingPage 收到的游标，
// 用于断言 keyset 是否正确推进。allDevicesCalled 标记是否回退到一次性全量加载路径。
type fakePagingLister struct {
	pages            [][]*Task
	idx              int
	recordedCursors  []PendingCursor
	allDevicesCalled bool
	pageErrAt        int // <0 表示不注入错误；否则在第 N 次 ListPendingPage 返回错误
}

func (f *fakePagingLister) ListPendingAllDevices(_ context.Context, _ int) ([]*Task, error) {
	f.allDevicesCalled = true
	var all []*Task
	for _, p := range f.pages {
		all = append(all, p...)
	}
	return all, nil
}

func (f *fakePagingLister) ListPendingPage(_ context.Context, after PendingCursor, _ int) ([]*Task, error) {
	f.recordedCursors = append(f.recordedCursors, after)
	if f.pageErrAt >= 0 && f.idx == f.pageErrAt {
		return nil, fmt.Errorf("pg page down")
	}
	if f.idx >= len(f.pages) {
		return nil, nil
	}
	p := f.pages[f.idx]
	f.idx++
	return p, nil
}

// pendingTaskWithCursor 构造带 created_at/id 的 pending 任务（供 keyset 推进断言）。
func pendingTaskWithCursor(id, sn string, createdAt time.Time) *Task {
	t := newPendingTask(id, sn)
	t.CreatedAt = createdAt
	return t
}

// newMiniRedisQueue 构造一个挂在 miniredis 上的真实 RedisTaskQueue。
func newMiniRedisQueue(t *testing.T) (*RedisTaskQueue, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	return NewRedisTaskQueue(client), m
}

func newPendingTask(id, sn string) *Task {
	return &Task{
		ID:       id,
		DeviceSN: sn,
		Method:   "Reboot",
		Params:   json.RawMessage("{}"),
		Priority: 10,
		Status:   TaskStatusPending,
	}
}

func Test_RestorePendingQueues_PushesAbsentTasks(t *testing.T) {
	q, _ := newMiniRedisQueue(t)
	ctx := context.Background()

	lister := &fakePendingLister{tasks: []*Task{
		newPendingTask("t1", "SN001"),
		newPendingTask("t2", "SN001"),
		newPendingTask("t3", "SN002"),
	}}

	stats, err := restorePendingQueues(ctx, lister, q, zap.NewNop(), 0)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.Scanned)
	assert.Equal(t, 3, stats.Pushed)
	assert.Equal(t, 0, stats.Skipped)
	assert.Equal(t, 0, stats.Failed)

	// 验证队列中真的有任务
	sn1Len, err := q.Len(ctx, "SN001")
	require.NoError(t, err)
	assert.Equal(t, int64(2), sn1Len)
	sn2Len, err := q.Len(ctx, "SN002")
	require.NoError(t, err)
	assert.Equal(t, int64(1), sn2Len)
}

func Test_RestorePendingQueues_SkipsExistingTasks(t *testing.T) {
	q, _ := newMiniRedisQueue(t)
	ctx := context.Background()

	// 先 Push 一个任务，模拟 Redis 已有 — 再跑 restore 时应跳过
	existing := newPendingTask("t1", "SN001")
	require.NoError(t, q.Push(ctx, existing))

	lister := &fakePendingLister{tasks: []*Task{
		existing, // 已存在 → skip
		newPendingTask("t2", "SN001"),
	}}

	stats, err := restorePendingQueues(ctx, lister, q, zap.NewNop(), 0)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Scanned)
	assert.Equal(t, 1, stats.Pushed)
	assert.Equal(t, 1, stats.Skipped)
	assert.Equal(t, 0, stats.Failed)

	// 队列中 t1 只应存在 1 份
	length, err := q.Len(ctx, "SN001")
	require.NoError(t, err)
	assert.Equal(t, int64(2), length)
}

func Test_RestorePendingQueues_PropagatesListerError(t *testing.T) {
	q, _ := newMiniRedisQueue(t)
	lister := &fakePendingLister{err: fmt.Errorf("pg down")}

	_, err := restorePendingQueues(context.Background(), lister, q, zap.NewNop(), 0)
	assert.ErrorContains(t, err, "list pending tasks")
}

func Test_RestorePendingQueues_Idempotent(t *testing.T) {
	// 模拟进程多次重启：同一份 pending 任务不应重复入队
	q, _ := newMiniRedisQueue(t)
	ctx := context.Background()

	tasks := []*Task{
		newPendingTask("t1", "SN001"),
		newPendingTask("t2", "SN002"),
	}
	lister := &fakePendingLister{tasks: tasks}

	// 第一次：两个都 Push
	stats, err := restorePendingQueues(ctx, lister, q, zap.NewNop(), 0)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Pushed)
	assert.Equal(t, 0, stats.Skipped)

	// 第二次：都已存在，全部 Skip
	stats, err = restorePendingQueues(ctx, lister, q, zap.NewNop(), 0)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Pushed)
	assert.Equal(t, 2, stats.Skipped)

	// 队列里每个设备仍只有 1 个任务
	l1, _ := q.Len(ctx, "SN001")
	l2, _ := q.Len(ctx, "SN002")
	assert.Equal(t, int64(1), l1)
	assert.Equal(t, int64(1), l2)
}

// #11 流式恢复：limit<=0 且 lister 支持分页时，按页拉取并灌入 Redis，cursor 正确推进，
// 不走一次性全量加载路径。
func Test_RestorePendingQueues_StreamsPages(t *testing.T) {
	q, _ := newMiniRedisQueue(t)
	ctx := context.Background()

	t0 := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	// 两满页（每页 restorePageBatchSize 条）+ 一非满末页，强制多轮翻页。
	full := restorePageBatchSize
	page1 := make([]*Task, full)
	page2 := make([]*Task, full)
	for i := 0; i < full; i++ {
		page1[i] = pendingTaskWithCursor(fmt.Sprintf("p1-%d", i), "SN-A", t0.Add(time.Duration(i)*time.Second))
		page2[i] = pendingTaskWithCursor(fmt.Sprintf("p2-%d", i), "SN-B", t0.Add(time.Duration(full+i)*time.Second))
	}
	page3 := []*Task{pendingTaskWithCursor("p3-0", "SN-C", t0.Add(time.Duration(2*full)*time.Second))}

	lister := &fakePagingLister{
		pages:     [][]*Task{page1, page2, page3},
		pageErrAt: -1,
	}

	stats, err := restorePendingQueues(ctx, lister, q, zap.NewNop(), 0)
	require.NoError(t, err)

	wantTotal := 2*full + 1
	assert.Equal(t, wantTotal, stats.Scanned)
	assert.Equal(t, wantTotal, stats.Pushed)
	assert.False(t, lister.allDevicesCalled, "应走分页流式路径，不一次性全量加载")

	// cursor 推进：首页零游标，第二页游标 = 第一页末条，第三页游标 = 第二页末条。
	require.GreaterOrEqual(t, len(lister.recordedCursors), 3)
	assert.True(t, lister.recordedCursors[0].IsZero(), "首页应是零游标")
	assert.Equal(t, page1[full-1].ID, lister.recordedCursors[1].ID)
	assert.Equal(t, page2[full-1].ID, lister.recordedCursors[2].ID)
}

// #11 流式恢复：分页过程中 PG 报错应包装并冒泡，已处理的统计不丢。
func Test_RestorePendingQueues_StreamPageError(t *testing.T) {
	q, _ := newMiniRedisQueue(t)
	ctx := context.Background()

	full := restorePageBatchSize
	page1 := make([]*Task, full)
	for i := 0; i < full; i++ {
		page1[i] = newPendingTask(fmt.Sprintf("e-%d", i), "SN-E")
	}
	lister := &fakePagingLister{
		pages:     [][]*Task{page1},
		pageErrAt: 1, // 第二次翻页报错
	}

	_, err := restorePendingQueues(ctx, lister, q, zap.NewNop(), 0)
	assert.ErrorContains(t, err, "list pending tasks")
}

// #11：调用方给了显式上界（limit>0）时仍走单批 ListPendingAllDevices（向后兼容）。
func Test_RestorePendingQueues_ExplicitLimit_UsesSingleBatch(t *testing.T) {
	q, _ := newMiniRedisQueue(t)
	ctx := context.Background()

	lister := &fakePagingLister{
		pages: [][]*Task{{
			newPendingTask("s1", "SN-X"),
			newPendingTask("s2", "SN-Y"),
		}},
		pageErrAt: -1,
	}

	stats, err := restorePendingQueues(ctx, lister, q, zap.NewNop(), 50)
	require.NoError(t, err)
	assert.True(t, lister.allDevicesCalled, "limit>0 应走单批全量路径")
	assert.Equal(t, 2, stats.Scanned)
	assert.Equal(t, 2, stats.Pushed)
}

// ---- notifyCompletion ----

// capturingEventBus 记录被发布的事件，Subscribe 返回空实现。
type capturingEventBus struct {
	published []event.Event
	subjects  []string
}

func (b *capturingEventBus) Publish(_ context.Context, subject string, evt event.Event) error {
	b.subjects = append(b.subjects, subject)
	b.published = append(b.published, evt)
	return nil
}
func (b *capturingEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &noopSub{}, nil
}
func (b *capturingEventBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &noopSub{}, nil
}
func (b *capturingEventBus) Close() error { return nil }

type noopSub struct{}

func (noopSub) Unsubscribe() error { return nil }

type recordingCallback struct {
	tasks []*Task
}

func (r *recordingCallback) OnTaskCompleted(_ context.Context, t *Task) {
	r.tasks = append(r.tasks, t)
}

func Test_NotifyCompletion_PublishesWhenBusSet(t *testing.T) {
	bus := &capturingEventBus{}
	cb := &recordingCallback{}

	svc := &TaskService{logger: zap.NewNop(), callbacks: []TaskCompletionCallback{cb}, eventBus: bus}

	mmlTask := &Task{
		ID:       "t1",
		DeviceSN: "SN001",
		Source:   TaskSourceMML,
		SourceID: "mml-uuid-1",
		Status:   TaskStatusCompleted,
	}
	svc.notifyCompletion(context.Background(), mmlTask)

	require.Len(t, bus.published, 1, "应发布一次事件")
	assert.Equal(t, event.SubjectTaskCompleted, bus.subjects[0])
	assert.Len(t, cb.tasks, 0, "EventBus 存在时不再触发本地 callback，避免重复计数")
}

func Test_NotifyCompletion_FallbacksToCallbackWhenBusNil(t *testing.T) {
	cb := &recordingCallback{}
	svc := &TaskService{logger: zap.NewNop(), callbacks: []TaskCompletionCallback{cb}}

	mmlTask := &Task{
		ID:       "t1",
		DeviceSN: "SN001",
		Source:   TaskSourceMML,
		SourceID: "mml-uuid-1",
		Status:   TaskStatusFailed,
	}
	svc.notifyCompletion(context.Background(), mmlTask)

	require.Len(t, cb.tasks, 1)
	assert.Equal(t, "t1", cb.tasks[0].ID)
}

// P1 重构后（docs/design/mml-task-flow-design-20260424.md §3.3 C2）：
// source 过滤取消，任何带 source_id 的终态都发事件；由 APP 侧 CompletionRouter
// 按 source 分发。仅当 source_id 为空（匿名 / 临时任务）时才跳过。
func Test_NotifyCompletion_PublishesForAnySourceWithSourceID(t *testing.T) {
	bus := &capturingEventBus{}
	cb := &recordingCallback{}
	svc := &TaskService{logger: zap.NewNop(), callbacks: []TaskCompletionCallback{cb}, eventBus: bus}

	// API 来源 + 非空 source_id：应发事件（让未来 API 聚合器可接入）
	svc.notifyCompletion(context.Background(), &Task{
		ID: "t1", Source: TaskSourceAPI, SourceID: "x", Status: TaskStatusCompleted,
	})
	// MML 来源 + 空 source_id：跳过
	svc.notifyCompletion(context.Background(), &Task{
		ID: "t2", Source: TaskSourceMML, SourceID: "", Status: TaskStatusCompleted,
	})

	require.Len(t, bus.published, 1, "有 source_id 即发事件（无论 source 类型）")
	var decoded Task
	require.NoError(t, bus.published[0].DecodePayload(&decoded))
	assert.Equal(t, "t1", decoded.ID)
	assert.Len(t, cb.tasks, 0, "eventBus 存在时不重复触发本地 callback")
}

func Test_NotifyCompletion_SkipsWhenSourceIDEmpty(t *testing.T) {
	bus := &capturingEventBus{}
	cb := &recordingCallback{}
	svc := &TaskService{logger: zap.NewNop(), callbacks: []TaskCompletionCallback{cb}, eventBus: bus}

	svc.notifyCompletion(context.Background(), &Task{
		ID: "t1", Source: TaskSourceAPI, SourceID: "", Status: TaskStatusCompleted,
	})

	assert.Len(t, bus.published, 0, "source_id 为空时不发事件")
	assert.Len(t, cb.tasks, 0)
}

func Test_SubjectForStatus(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{TaskStatusCompleted, event.SubjectTaskCompleted},
		{TaskStatusFailed, event.SubjectTaskFailed},
		{TaskStatusExpired, event.SubjectTaskFailed},
		{TaskStatusPending, ""},
		{TaskStatusSent, ""},
		{TaskStatusCancelled, ""},
	}
	for _, tc := range tests {
		got := SubjectForStatus(tc.status)
		assert.Equal(t, tc.want, got, "status=%s", tc.status)
	}
}
