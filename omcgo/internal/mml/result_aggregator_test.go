package mml

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// =============================================================================
// T-0102-d — per-device frame SSE push tests
// =============================================================================

// fakeSSEHub captures PublishSimple calls so tests can assert on the
// sequence + payloads emitted by ResultAggregator. Mirrors the
// SSEPublisher contract without spinning up the real events.MessageHub.
type fakeSSEHub struct {
	mu       sync.Mutex
	events   []fakeSSEEvent
}

type fakeSSEEvent struct {
	UserID    string
	EventType string
	Data      []byte
}

func (h *fakeSSEHub) PublishSimple(userID, eventType string, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, fakeSSEEvent{
		UserID: userID, EventType: eventType, Data: append([]byte(nil), data...),
	})
}

func (h *fakeSSEHub) byType(eventType string) []fakeSSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []fakeSSEEvent
	for _, e := range h.events {
		if e.EventType == eventType {
			out = append(out, e)
		}
	}
	return out
}

// aggregatorTestRepo extends mockTaskRepo with IncrementStats capture
// and a default GetByID that returns a sensible MMLTask snapshot.
type aggregatorTestRepo struct {
	*mockTaskRepo
	incCalls    []incCall
	incErr      error
	getMMLTask  *MMLTask
	getErr      error
	updatedTask *MMLTask // 最近一次 Update 的快照（finalize 断言用）
}

type incCall struct {
	ID            uuid.UUID
	SuccessDelta  int
	FailedDelta   int
}

func (r *aggregatorTestRepo) IncrementStats(_ context.Context, id uuid.UUID, successDelta, failedDelta int) error {
	r.incCalls = append(r.incCalls, incCall{ID: id, SuccessDelta: successDelta, FailedDelta: failedDelta})
	return r.incErr
}

func (r *aggregatorTestRepo) GetByID(_ context.Context, _ uuid.UUID) (*MMLTask, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.getMMLTask, nil
}

func (r *aggregatorTestRepo) Update(_ context.Context, t *MMLTask) error {
	if t != nil {
		snap := *t
		r.updatedTask = &snap
	}
	return nil
}

// fakeDeviceStats 实现 DeviceTaskStatsReader，按 sourceID 返回预置的终态聚合，
// 用于验证 finalizeIfComplete 以实际 device_tasks 终态为完成判据。
type fakeDeviceStats struct {
	stats task.DeviceTaskSourceStats
	err   error
	calls int
}

func (f *fakeDeviceStats) AggregateStatusBySourceID(_ context.Context, _ task.TaskSource, _ string) (task.DeviceTaskSourceStats, error) {
	f.calls++
	return f.stats, f.err
}

func newAggregatorTestRepo(mmlTask *MMLTask) *aggregatorTestRepo {
	return &aggregatorTestRepo{mockTaskRepo: &mockTaskRepo{}, getMMLTask: mmlTask}
}

func newAggregatorWithHub(repo TaskRepository, hub SSEPublisher) *ResultAggregator {
	return NewResultAggregator(repo, nil, nil, hub, zap.NewNop())
}

// V1 — completed device_task emits mml_device_frame with device_sn + result
func TestAggregator_OnTaskCompleted_PublishesDeviceFrame_Success(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{
		ID:           mmlID,
		Executor:     "alice",
		TotalDevices: 3,
		Commands:     []map[string]interface{}{{}, {}},
		SuccessCount: 1, FailedCount: 0,
	})
	hub := &fakeSSEHub{}
	agg := newAggregatorWithHub(repo, hub)

	completedAt := time.Now()
	dt := &task.Task{
		ID:           "device-task-001",
		SourceID:     mmlID.String(),
		DeviceSN:     "SN-1",
		Method:       "Reboot",
		Status:       task.TaskStatusCompleted,
		Result:       json.RawMessage(`{"reboot_status":"ok"}`),
		CompletedAt:  &completedAt,
		CommandIndex: 0,
		DeviceIndex:  0,
	}

	agg.OnTaskCompleted(context.Background(), dt)

	// Stats incremented
	require.Len(t, repo.incCalls, 1)
	assert.Equal(t, 1, repo.incCalls[0].SuccessDelta)
	assert.Equal(t, 0, repo.incCalls[0].FailedDelta)

	// Frame emitted to executor
	frames := hub.byType("mml_device_frame")
	require.Len(t, frames, 1, "exactly one frame per device_task terminal state")
	assert.Equal(t, "alice", frames[0].UserID)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(frames[0].Data, &payload))
	assert.Equal(t, mmlID.String(), payload["task_id"])
	assert.Equal(t, "device-task-001", payload["device_task_id"])
	assert.Equal(t, "SN-1", payload["device_sn"])
	assert.Equal(t, "Reboot", payload["method"])
	assert.Equal(t, "completed", payload["status"])
	assert.Equal(t, float64(0), payload["command_index"])
	assert.Equal(t, float64(0), payload["device_index"])
	// result is json.RawMessage → unmarshalled back into map
	resultMap, ok := payload["result"].(map[string]interface{})
	require.True(t, ok, "result must be a structured object")
	assert.Equal(t, "ok", resultMap["reboot_status"])
	assert.NotContains(t, payload, "error_message",
		"success frame must not carry error_message")
}

// V2 — failed device_task emits frame carrying error_message
func TestAggregator_OnTaskCompleted_PublishesDeviceFrame_Failure(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{
		ID: mmlID, Executor: "bob", TotalDevices: 1, Commands: []map[string]interface{}{{}},
	})
	hub := &fakeSSEHub{}
	agg := newAggregatorWithHub(repo, hub)

	dt := &task.Task{
		ID: "device-task-002", SourceID: mmlID.String(), DeviceSN: "SN-2",
		Method: "FactoryReset", Status: task.TaskStatusFailed,
		ErrorMessage: "device offline",
	}
	agg.OnTaskCompleted(context.Background(), dt)

	require.Len(t, repo.incCalls, 1)
	assert.Equal(t, 0, repo.incCalls[0].SuccessDelta)
	assert.Equal(t, 1, repo.incCalls[0].FailedDelta)

	frames := hub.byType("mml_device_frame")
	require.Len(t, frames, 1)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(frames[0].Data, &payload))
	assert.Equal(t, "failed", payload["status"])
	assert.Equal(t, "device offline", payload["error_message"])
}

// V3 — non-terminal status (sent / pending) emits no frame and no stat update
func TestAggregator_OnTaskCompleted_NonTerminal_NoFrame(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "carol"})
	hub := &fakeSSEHub{}
	agg := newAggregatorWithHub(repo, hub)

	for _, status := range []task.TaskStatus{
		task.TaskStatusSent, task.TaskStatusPending, task.TaskStatusCancelled,
	} {
		dt := &task.Task{SourceID: mmlID.String(), DeviceSN: "SN-x", Status: status}
		agg.OnTaskCompleted(context.Background(), dt)
	}

	assert.Empty(t, repo.incCalls, "non-terminal status must not increment stats")
	assert.Empty(t, hub.byType("mml_device_frame"),
		"non-terminal status must not emit frames")
}

// V4 — nil hub or empty executor → no panic, no publish (best-effort)
func TestAggregator_OnTaskCompleted_NilHub_NoPanic(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "dave"})
	agg := NewResultAggregator(repo, nil, nil, nil, zap.NewNop())

	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-3", Status: task.TaskStatusCompleted,
	}
	assert.NotPanics(t, func() {
		agg.OnTaskCompleted(context.Background(), dt)
	})
}

// =============================================================================
// T-0174 / T-0176-PR-E — auto-learn is_unsupported from SetParameterValues 9005
// =============================================================================
//
// T-0176-PR-E：写入真值源从 mml_command_sub_fields 迁到 param_mappings，
// MarkUnsupportedByStandardPath 签名加 paramModelID 入参。ResultAggregator 在调用
// repo 之前先用注入的 paramModelResolver 把 dt.DeviceSN 解析成 paramModelID；
// resolver 未注入 / 返 nil / 返 err 时整个 auto-learn skip 不污染数据。

// fakeSubFieldRepoAutoLearn 捕获 MarkUnsupportedByStandardPath 调用，
// 同时记录 paramModelID 入参以便测试断言 PR-E 解析链路被正确穿透。
type fakeSubFieldRepoAutoLearn struct {
	SubFieldRepository // 嵌入接口让未实现的方法 panic 时立刻暴露（仅 Mark 路径被测）
	mu          sync.Mutex
	markedPaths []string
	markedPMIDs []uuid.UUID
	rowsPerPath int64
}

func (f *fakeSubFieldRepoAutoLearn) MarkUnsupportedByStandardPath(
	_ context.Context, paramModelID uuid.UUID, path string,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markedPaths = append(f.markedPaths, path)
	f.markedPMIDs = append(f.markedPMIDs, paramModelID)
	return f.rowsPerPath, nil
}

// resolverConst 返回固定 paramModelID 的 resolver（happy path 用）。
func resolverConst(pmID uuid.UUID) func(context.Context, string) (*uuid.UUID, error) {
	return func(_ context.Context, _ string) (*uuid.UUID, error) {
		return &pmID, nil
	}
}

// V5 — SPV fault 9005 触发 auto-learn，paramModelID 透传正确
func TestAggregator_AutoLearn_MarksUnsupportedOn9005(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{rowsPerPath: 1}
	pmID := uuid.New()
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(resolverConst(pmID))

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.DeviceInfo.UserLabel", "fault_code": 9005, "fault_string": "AttributeIdNotFound"},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(),
		DeviceSN: "SN-X",
		Status:   task.TaskStatusFailed,
		Method:   "SetParameterValues",
		Result:   result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Equal(t, []string{"Device.DeviceInfo.UserLabel"}, sf.markedPaths)
	require.Len(t, sf.markedPMIDs, 1)
	assert.Equal(t, pmID, sf.markedPMIDs[0], "paramModelID 必须从 resolver 透传到 repo")
}

// V5b — 非 9005 fault code（如 9008 read-only）不触发 auto-learn
func TestAggregator_AutoLearn_SkipsNon9005Codes(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{}
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(resolverConst(uuid.New()))

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.DeviceInfo.Foo", "fault_code": 9008, "fault_string": "read-only"},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-Y",
		Status: task.TaskStatusFailed, Method: "SetParameterValues",
		Result: result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, sf.markedPaths)
}

// V5c — 非 SPV 方法即使失败也不触发（GPV 等没有 SetParameterValuesFault 语义）
func TestAggregator_AutoLearn_SkipsNonSPVMethods(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{}
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(resolverConst(uuid.New()))

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.X.Y", "fault_code": 9005, "fault_string": "nope"},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-Z",
		Status: task.TaskStatusFailed, Method: "GetParameterValues",
		Result: result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, sf.markedPaths)
}

// V5d — subFieldRepo 为 nil（未注入 auto-learn）时 silent skip，不 panic
func TestAggregator_AutoLearn_NilRepo_NoPanic(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	agg := NewResultAggregator(repo, nil, nil, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(resolverConst(uuid.New()))

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.A", "fault_code": 9005},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), Status: task.TaskStatusFailed,
		Method: "SetParameterValues", Result: result,
	}
	assert.NotPanics(t, func() { agg.OnTaskCompleted(context.Background(), dt) })
}

// V5e — paramModelResolver 未注入（nil） → 整个 auto-learn skip，不调 repo
func TestAggregator_AutoLearn_ResolverNil_Skip(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{rowsPerPath: 1}
	// 不调 SetParamModelResolver — resolver 保持 nil
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.A", "fault_code": 9005},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-W",
		Status: task.TaskStatusFailed, Method: "SetParameterValues",
		Result: result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, sf.markedPaths, "resolver nil → 不调 MarkUnsupportedByStandardPath")
}

// V5f — paramModelResolver 返 nil（孤儿设备 / 无 paramModel） → skip
func TestAggregator_AutoLearn_ResolverReturnsNil_Skip(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{rowsPerPath: 1}
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(func(_ context.Context, _ string) (*uuid.UUID, error) {
		return nil, nil // 孤儿设备 / 无 paramModel
	})

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.B", "fault_code": 9005},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-ORPHAN",
		Status: task.TaskStatusFailed, Method: "SetParameterValues",
		Result: result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, sf.markedPaths, "resolver 返 nil → 孤儿设备不污染数据")
}

// V5g — paramModelResolver 返 error → silent skip + 不调 repo
func TestAggregator_AutoLearn_ResolverReturnsErr_Skip(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{rowsPerPath: 1}
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(func(_ context.Context, _ string) (*uuid.UUID, error) {
		return nil, errors.New("db blip resolving param_model")
	})

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.C", "fault_code": 9005},
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-ERR",
		Status: task.TaskStatusFailed, Method: "SetParameterValues",
		Result: result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Empty(t, sf.markedPaths, "resolver 返 err → silent skip 不调 repo")
}

// V5h — 同一 task 多个 9005 path 全部 mark 到同一 paramModelID 下
func TestAggregator_AutoLearn_MultiplePaths_AllMarkedUnderSamePmID(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "u", TotalDevices: 1, Commands: []map[string]interface{}{{}}})
	sf := &fakeSubFieldRepoAutoLearn{rowsPerPath: 1}
	pmID := uuid.New()
	agg := NewResultAggregator(repo, nil, sf, &fakeSSEHub{}, zap.NewNop())
	agg.SetParamModelResolver(resolverConst(pmID))

	result, _ := json.Marshal(map[string]interface{}{
		"param_faults": []map[string]interface{}{
			{"parameter_name": "Device.A.Foo", "fault_code": 9005},
			{"parameter_name": "Device.A.Bar", "fault_code": 9005},
			{"parameter_name": "Device.A.Baz", "fault_code": 9008}, // 不该被 mark
		},
	})
	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-MULTI",
		Status: task.TaskStatusFailed, Method: "SetParameterValues",
		Result: result,
	}
	agg.OnTaskCompleted(context.Background(), dt)
	assert.Equal(t, []string{"Device.A.Foo", "Device.A.Bar"}, sf.markedPaths)
	require.Len(t, sf.markedPMIDs, 2)
	for _, got := range sf.markedPMIDs {
		assert.Equal(t, pmID, got, "所有调用必须用同一 paramModelID")
	}
}

func TestAggregator_OnTaskCompleted_EmptyExecutor_SkipsFrame(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{ID: mmlID, Executor: "" /* anonymous */})
	hub := &fakeSSEHub{}
	agg := newAggregatorWithHub(repo, hub)

	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-4", Status: task.TaskStatusCompleted,
	}
	agg.OnTaskCompleted(context.Background(), dt)

	assert.Empty(t, hub.byType("mml_device_frame"),
		"empty executor → no per-user channel to push to; skip frame")
}

// V5 — empty source_id → early return, no work done
func TestAggregator_OnTaskCompleted_EmptySourceID(t *testing.T) {
	hub := &fakeSSEHub{}
	repo := newAggregatorTestRepo(nil)
	agg := newAggregatorWithHub(repo, hub)

	dt := &task.Task{SourceID: "", Status: task.TaskStatusCompleted}
	agg.OnTaskCompleted(context.Background(), dt)

	assert.Empty(t, repo.incCalls)
	assert.Empty(t, hub.events)
}

// V6 — frame publish survives GetByID transient error: the increment is
// already committed, the user just misses ONE frame; the next device's
// frame still flows. Multi-frame fan-out doesn't get gated by one
// flaky frame lookup.
func TestAggregator_OnTaskCompleted_GetByIDError_StillIncrements(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(nil)
	repo.getErr = errors.New("db blip")
	hub := &fakeSSEHub{}
	agg := newAggregatorWithHub(repo, hub)

	dt := &task.Task{SourceID: mmlID.String(), DeviceSN: "SN-5", Status: task.TaskStatusCompleted}
	agg.OnTaskCompleted(context.Background(), dt)

	require.Len(t, repo.incCalls, 1, "stat increment must commit even if frame lookup fails")
	assert.Empty(t, hub.byType("mml_device_frame"),
		"GetByID error → skip frame publish, do not crash")
}

// =============================================================================
// finalize 判据：以实际 device_tasks 终态为准（修复理论 total 卡死 running）
// =============================================================================

// 回归：fanout/sequencer 跳过了某设备 → 实际派发 < 理论 total（设备数×命令数），
// 但已派发的全部成功并进入终态。旧逻辑 done(2) < total(3) 永远不 finalize → 卡
// running；新逻辑以 device_tasks 终态判定（Active==0）→ 正常 finalize 为成功。
func TestAggregator_Finalize_UsesActualDeviceTasks_NotTheoreticalTotal(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{
		ID:           mmlID,
		Status:       TaskRunning,
		TotalDevices: 3, // 理论 total = 3×1 = 3
		Commands:     []map[string]interface{}{{}},
		SuccessCount: 2, FailedCount: 0, // 仅 2 台真正派发并完成（1 台 fanout 跳过）
	})
	agg := newAggregatorWithHub(repo, &fakeSSEHub{})
	// 实际 device_tasks：仅 2 条，均 completed，无在途。
	agg.SetDeviceTaskStatsReader(&fakeDeviceStats{
		stats: task.DeviceTaskSourceStats{Total: 2, Completed: 2, Failed: 0, Active: 0},
	})

	dt := &task.Task{SourceID: mmlID.String(), DeviceSN: "SN-2", Status: task.TaskStatusCompleted}
	agg.OnTaskCompleted(context.Background(), dt)

	require.NotNil(t, repo.updatedTask, "应以实际 device_tasks 终态 finalize，而非卡 running")
	assert.Equal(t, TaskCompleted, repo.updatedTask.Status)
	require.NotNil(t, repo.updatedTask.Result)
	assert.Equal(t, ResultSuccess, *repo.updatedTask.Result)
	assert.NotNil(t, repo.updatedTask.FinishedAt)
	// 计数器以真值覆盖
	assert.Equal(t, 2, repo.updatedTask.SuccessCount)
}

// 仍有在途 device_task（Active>0）→ 不 finalize（顺序模式中途等待下一行）。
func TestAggregator_Finalize_ActiveInFlight_NoFinalize(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{
		ID: mmlID, Status: TaskRunning, TotalDevices: 1,
		Commands: []map[string]interface{}{{}, {}}, // 2 命令（顺序链）
	})
	agg := newAggregatorWithHub(repo, &fakeSSEHub{})
	agg.SetDeviceTaskStatsReader(&fakeDeviceStats{
		stats: task.DeviceTaskSourceStats{Total: 2, Completed: 1, Failed: 0, Active: 1},
	})

	dt := &task.Task{SourceID: mmlID.String(), DeviceSN: "SN-1", Status: task.TaskStatusCompleted, CommandIndex: 0}
	agg.OnTaskCompleted(context.Background(), dt)

	assert.Nil(t, repo.updatedTask, "仍有在途 device_task 时不得 finalize")
}

// 部分成功部分失败、全部终态 → finalize 为 partial。
func TestAggregator_Finalize_MixedTerminal_Partial(t *testing.T) {
	mmlID := uuid.New()
	repo := newAggregatorTestRepo(&MMLTask{
		ID: mmlID, Status: TaskRunning, TotalDevices: 3,
		Commands: []map[string]interface{}{{}},
	})
	agg := newAggregatorWithHub(repo, &fakeSSEHub{})
	agg.SetDeviceTaskStatsReader(&fakeDeviceStats{
		stats: task.DeviceTaskSourceStats{Total: 3, Completed: 2, Failed: 1, Active: 0},
	})

	dt := &task.Task{SourceID: mmlID.String(), DeviceSN: "SN-3", Status: task.TaskStatusFailed}
	agg.OnTaskCompleted(context.Background(), dt)

	require.NotNil(t, repo.updatedTask)
	assert.Equal(t, TaskCompleted, repo.updatedTask.Status)
	require.NotNil(t, repo.updatedTask.Result)
	assert.Equal(t, ResultPartial, *repo.updatedTask.Result)
	assert.Equal(t, 2, repo.updatedTask.SuccessCount)
	assert.Equal(t, 1, repo.updatedTask.FailedCount)
}
