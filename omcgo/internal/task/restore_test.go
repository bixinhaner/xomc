package task

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

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

func Test_NotifyCompletion_SkipsNonMMLOrMissingSourceID(t *testing.T) {
	bus := &capturingEventBus{}
	cb := &recordingCallback{}
	svc := &TaskService{logger: zap.NewNop(), callbacks: []TaskCompletionCallback{cb}, eventBus: bus}

	cases := []*Task{
		{ID: "t1", Source: TaskSourceAPI, SourceID: "x", Status: TaskStatusCompleted},
		{ID: "t2", Source: TaskSourceMML, SourceID: "", Status: TaskStatusCompleted},
	}
	for _, task := range cases {
		svc.notifyCompletion(context.Background(), task)
	}

	assert.Len(t, bus.published, 0, "非 MML 或无 source_id 不应发事件")
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
