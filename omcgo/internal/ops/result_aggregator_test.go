package ops

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
)

// ============================================================
// result_aggregator_test.go — T-0102 残债 2 / B′
//
// 覆盖：
//   - 成功路径：execution row 翻转 + SSE command.completed
//   - 失败路径：execution row 翻转 + SSE command.failed
//   - expired 状态映射
//   - source 不为 ops → 短路（防 AddCompletionCallback 全量分发误触）
//   - dt.Status 非终态 (pending/sent/cancelled) → 短路
//   - SourceID 空 / 非法 UUID → log warn，不 panic
//   - nil sseHub → 仍更行（best-effort SSE 不阻塞持久化）
//   - row not found → log warn，不更新（孤儿设备 graceful）
//   - List 错 / Update 错 → log warn，不阻塞 SSE 推送
// ============================================================

// fakeExecRepoForAgg 是 TaskExecutionRepository 的可控 stub，
// 记录 Update 调用便于断言。
type fakeExecRepoForAgg struct {
	mu         sync.Mutex
	listRows   []OpsTaskExecution
	listErr    error
	updateErr  error
	updateCall []OpsTaskExecution // 捕获顺序排序
}

func (f *fakeExecRepoForAgg) Create(_ context.Context, _ *OpsTaskExecution) error { return nil }

func (f *fakeExecRepoForAgg) Update(_ context.Context, e *OpsTaskExecution) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *e
	f.updateCall = append(f.updateCall, cp)
	return nil
}

func (f *fakeExecRepoForAgg) List(_ context.Context, _ TaskExecutionFilter) (*model.ListResponse[OpsTaskExecution], error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]OpsTaskExecution, len(f.listRows))
	copy(items, f.listRows)
	return model.NewListResponse(items, int64(len(items)), 1, 20), nil
}

func (f *fakeExecRepoForAgg) CountByTaskStatus(_ context.Context, _ uuid.UUID) (map[string]int, error) {
	return nil, nil
}

func (f *fakeExecRepoForAgg) updates() []OpsTaskExecution {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]OpsTaskExecution, len(f.updateCall))
	copy(out, f.updateCall)
	return out
}

// drainOne 从 channel 拿一个事件，超时返 nil。
func drainOne(ch <-chan SSEEvent, timeout time.Duration) *SSEEvent {
	select {
	case e := <-ch:
		return &e
	case <-time.After(timeout):
		return nil
	}
}

// buildRunningRow 构造一个 dispatchInlineRPC 入队时创建的"running" execution row。
func buildRunningRow(opsTaskID uuid.UUID, deviceSN, action string, startedAt time.Time) OpsTaskExecution {
	return OpsTaskExecution{
		ID:        uuid.New(),
		TaskID:    opsTaskID,
		DeviceSN:  deviceSN,
		StepIndex: 0,
		StepName:  action,
		StepType:  "rpc",
		Status:    "running",
		StartedAt: &startedAt,
	}
}

// buildOpsTaskCompleted 构造 source=ops 的终态 device_task。
func buildOpsTaskCompleted(opsTaskID uuid.UUID, deviceSN string, status task.TaskStatus, completedAt time.Time) *task.Task {
	return &task.Task{
		ID:          uuid.New().String(),
		DeviceSN:    deviceSN,
		Method:      "Reboot",
		Source:      task.TaskSourceOps,
		SourceID:    opsTaskID.String(),
		Status:      status,
		CompletedAt: &completedAt,
	}
}

// ============================================================
// 成功路径
// ============================================================

func TestResultAggregator_Success_UpdatesRowAndPublishesEvent(t *testing.T) {
	opsTaskID := uuid.New()
	startedAt := time.Now().Add(-2 * time.Second)
	completedAt := startedAt.Add(2 * time.Second)
	row := buildRunningRow(opsTaskID, "SN1", "reboot", startedAt)

	repo := &fakeExecRepoForAgg{listRows: []OpsTaskExecution{row}}
	hub := NewSSEHub()
	events, unsub := hub.Subscribe("command:" + opsTaskID.String())
	defer unsub()

	agg := NewResultAggregator(repo, hub, nil)
	dt := buildOpsTaskCompleted(opsTaskID, "SN1", task.TaskStatusCompleted, completedAt)
	dt.Result = json.RawMessage(`{"reboot_status":"ok"}`)
	agg.OnTaskCompleted(context.Background(), dt)

	// row updated
	updates := repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, "completed", updates[0].Status)
	assert.NotNil(t, updates[0].CompletedAt)
	assert.Equal(t, 2000, updates[0].DurationMS, "2s startedAt → completedAt")
	assert.JSONEq(t, `{"reboot_status":"ok"}`, string(updates[0].Response))

	// SSE event command.completed
	ev := drainOne(events, 200*time.Millisecond)
	require.NotNil(t, ev)
	assert.Equal(t, "command.completed", ev.Event)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(ev.Data, &payload))
	assert.Equal(t, opsTaskID.String(), payload["ops_task_id"])
	assert.Equal(t, "SN1", payload["device_sn"])
	assert.Equal(t, "Reboot", payload["method"])
	assert.Equal(t, "completed", payload["status"])
}

// ============================================================
// 失败 / expired 路径
// ============================================================

func TestResultAggregator_Failed_UpdatesRowAndPublishesEvent(t *testing.T) {
	opsTaskID := uuid.New()
	row := buildRunningRow(opsTaskID, "SN-bad", "factory_reset", time.Now())
	repo := &fakeExecRepoForAgg{listRows: []OpsTaskExecution{row}}
	hub := NewSSEHub()
	events, unsub := hub.Subscribe("command:" + opsTaskID.String())
	defer unsub()

	agg := NewResultAggregator(repo, hub, nil)
	dt := buildOpsTaskCompleted(opsTaskID, "SN-bad", task.TaskStatusFailed, time.Now())
	dt.ErrorMessage = "device offline"
	agg.OnTaskCompleted(context.Background(), dt)

	updates := repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, "failed", updates[0].Status)
	assert.Equal(t, "device offline", updates[0].ErrorMessage)

	ev := drainOne(events, 200*time.Millisecond)
	require.NotNil(t, ev)
	assert.Equal(t, "command.failed", ev.Event)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(ev.Data, &payload))
	assert.Equal(t, "device offline", payload["error_message"])
	assert.Equal(t, "failed", payload["status"])
}

func TestResultAggregator_Expired_MapsToExpiredStatus(t *testing.T) {
	opsTaskID := uuid.New()
	row := buildRunningRow(opsTaskID, "SN-expired", "get_param", time.Now())
	repo := &fakeExecRepoForAgg{listRows: []OpsTaskExecution{row}}
	hub := NewSSEHub()
	events, unsub := hub.Subscribe("command:" + opsTaskID.String())
	defer unsub()

	agg := NewResultAggregator(repo, hub, nil)
	dt := buildOpsTaskCompleted(opsTaskID, "SN-expired", task.TaskStatusExpired, time.Now())
	agg.OnTaskCompleted(context.Background(), dt)

	updates := repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, "expired", updates[0].Status)

	ev := drainOne(events, 200*time.Millisecond)
	require.NotNil(t, ev)
	// expired 也归类到 command.failed（前端不需细分子状态，error_message 已携带）
	assert.Equal(t, "command.failed", ev.Event)
}

// ============================================================
// 短路场景
// ============================================================

func TestResultAggregator_NonOpsSource_Ignored(t *testing.T) {
	repo := &fakeExecRepoForAgg{}
	hub := NewSSEHub()
	agg := NewResultAggregator(repo, hub, nil)

	dt := &task.Task{
		ID:       uuid.New().String(),
		DeviceSN: "SN1",
		Source:   task.TaskSourceMML, // 非 ops
		SourceID: uuid.New().String(),
		Status:   task.TaskStatusCompleted,
	}
	agg.OnTaskCompleted(context.Background(), dt)

	assert.Empty(t, repo.updates(), "non-ops source should not trigger update")
}

func TestResultAggregator_NonTerminal_Ignored(t *testing.T) {
	cases := []task.TaskStatus{
		task.TaskStatusPending,
		task.TaskStatusSent,
		task.TaskStatusCancelled,
	}
	for _, status := range cases {
		t.Run(string(status), func(t *testing.T) {
			repo := &fakeExecRepoForAgg{}
			agg := NewResultAggregator(repo, nil, nil)
			dt := buildOpsTaskCompleted(uuid.New(), "SN1", status, time.Now())
			agg.OnTaskCompleted(context.Background(), dt)
			assert.Empty(t, repo.updates())
		})
	}
}

func TestResultAggregator_NilTask_NoPanic(t *testing.T) {
	repo := &fakeExecRepoForAgg{}
	agg := NewResultAggregator(repo, nil, nil)
	agg.OnTaskCompleted(context.Background(), nil) // no panic
	assert.Empty(t, repo.updates())
}

func TestResultAggregator_EmptySourceID_Ignored(t *testing.T) {
	repo := &fakeExecRepoForAgg{}
	agg := NewResultAggregator(repo, nil, nil)
	dt := &task.Task{
		ID:       uuid.New().String(),
		Source:   task.TaskSourceOps,
		SourceID: "", // 空
		Status:   task.TaskStatusCompleted,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, repo.updates())
}

func TestResultAggregator_BadSourceID_Ignored(t *testing.T) {
	repo := &fakeExecRepoForAgg{}
	agg := NewResultAggregator(repo, nil, nil)
	dt := &task.Task{
		ID:       uuid.New().String(),
		Source:   task.TaskSourceOps,
		SourceID: "not-a-uuid",
		Status:   task.TaskStatusCompleted,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, repo.updates())
}

// ============================================================
// 持久化错误 / SSE 缺席 graceful
// ============================================================

func TestResultAggregator_NilHub_StillUpdatesRow(t *testing.T) {
	opsTaskID := uuid.New()
	row := buildRunningRow(opsTaskID, "SN1", "reboot", time.Now())
	repo := &fakeExecRepoForAgg{listRows: []OpsTaskExecution{row}}

	agg := NewResultAggregator(repo, nil, nil) // hub nil
	dt := buildOpsTaskCompleted(opsTaskID, "SN1", task.TaskStatusCompleted, time.Now())
	agg.OnTaskCompleted(context.Background(), dt)

	assert.Len(t, repo.updates(), 1, "nil hub must not block row update")
}

func TestResultAggregator_RowNotFound_LogsWarn_NoUpdate(t *testing.T) {
	repo := &fakeExecRepoForAgg{listRows: nil} // empty
	hub := NewSSEHub()
	events, unsub := hub.Subscribe("command:" + uuid.New().String())
	defer unsub()

	agg := NewResultAggregator(repo, hub, nil)
	dt := buildOpsTaskCompleted(uuid.New(), "SN-orphan", task.TaskStatusCompleted, time.Now())
	agg.OnTaskCompleted(context.Background(), dt)

	assert.Empty(t, repo.updates(), "row not found should not trigger Update")
	// SSE 推送仍发（hub.Publish 是 best-effort，但 channel 不匹配所以本订阅不收到）
	_ = events
}

func TestResultAggregator_ListError_LogsWarn_SkipsUpdate(t *testing.T) {
	repo := &fakeExecRepoForAgg{listErr: assertableErr{"db down"}}
	agg := NewResultAggregator(repo, nil, nil)
	dt := buildOpsTaskCompleted(uuid.New(), "SN1", task.TaskStatusCompleted, time.Now())
	agg.OnTaskCompleted(context.Background(), dt) // no panic
	assert.Empty(t, repo.updates())
}

func TestResultAggregator_UpdateError_StillPublishesSSE(t *testing.T) {
	opsTaskID := uuid.New()
	row := buildRunningRow(opsTaskID, "SN1", "reboot", time.Now())
	repo := &fakeExecRepoForAgg{
		listRows:  []OpsTaskExecution{row},
		updateErr: assertableErr{"update conflict"},
	}
	hub := NewSSEHub()
	events, unsub := hub.Subscribe("command:" + opsTaskID.String())
	defer unsub()

	agg := NewResultAggregator(repo, hub, nil)
	dt := buildOpsTaskCompleted(opsTaskID, "SN1", task.TaskStatusCompleted, time.Now())
	agg.OnTaskCompleted(context.Background(), dt)

	// Update 失败但 SSE 仍推送（best-effort 解耦）
	ev := drainOne(events, 200*time.Millisecond)
	require.NotNil(t, ev, "SSE must still be published when row update fails")
	assert.Equal(t, "command.completed", ev.Event)
}

type assertableErr struct{ msg string }

func (e assertableErr) Error() string { return e.msg }
