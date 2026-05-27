package devsweep

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

// Test_NormalizeForProbe_NoPlaceholder：路径无 {i} 时原样返回。
func Test_NormalizeForProbe_NoPlaceholder(t *testing.T) {
	got := NormalizeForProbe("Device.DeviceInfo.UserLabel")
	require.Equal(t, "Device.DeviceInfo.UserLabel", got)
}

// Test_NormalizeForProbe_SinglePlaceholder：单个 {i} → 0。
func Test_NormalizeForProbe_SinglePlaceholder(t *testing.T) {
	got := NormalizeForProbe("Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation")
	require.Equal(t, "Device.FaultMgmt.CurrentAlarm.0.AdditionalInformation", got)
}

// Test_NormalizeForProbe_MultiPlaceholder：多个 {i} 全部 → 0。
func Test_NormalizeForProbe_MultiPlaceholder(t *testing.T) {
	got := NormalizeForProbe("Device.Services.FAPService.{i}.LTECell.{i}.X_VENDOR")
	require.Equal(t, "Device.Services.FAPService.0.LTECell.0.X_VENDOR", got)
}

// Test_NormalizeForProbe_Empty：空串原样。
func Test_NormalizeForProbe_Empty(t *testing.T) {
	require.Equal(t, "", NormalizeForProbe(""))
}

// Test_extractBadPathFromMessage table-driven 覆盖 ACS 几种错误消息样式。
func Test_extractBadPathFromMessage(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "including-style",
			msg:  "[Client] Invalid Parameter Names [1], including: Device.Services.FAPService.2.LTECell.",
			want: "Device.Services.FAPService.2.LTECell.",
		},
		{
			name: "name-style",
			msg:  "Invalid parameter name: Device.DeviceInfo.UserLabel",
			want: "Device.DeviceInfo.UserLabel",
		},
		{
			name: "quoted-parameter",
			msg:  "Parameter 'Device.X.Y.Z' is not supported",
			want: "Device.X.Y.Z",
		},
		{
			name: "empty",
			msg:  "",
			want: "",
		},
		{
			name: "fallback-longest-token",
			msg:  "weird vendor format, badPath Device.Foo.Bar.Baz blah",
			want: "Device.Foo.Bar.Baz",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractBadPathFromMessage(c.msg)
			require.Equal(t, c.want, got)
		})
	}
}

// Test_classifyGPVFailure_9005_BatchOne：batch=1 时 9005 → unsupported。
func Test_classifyGPVFailure_9005_BatchOne(t *testing.T) {
	tk := &task.Task{
		Status:       task.TaskStatusFailed,
		ErrorCode:    9005,
		ErrorMessage: "[Client] Invalid parameter name: Device.X",
	}
	out := classifyGPVFailure(tk, []string{"Device.X"})
	require.Len(t, out, 1)
	require.Equal(t, OutcomeUnsupported, out[0])
}

// Test_classifyGPVFailure_9005_BatchMany：batch>1 时仅 ErrorMessage 命中的 path
// 标 unsupported；其余 unknown。
func Test_classifyGPVFailure_9005_BatchMany(t *testing.T) {
	tk := &task.Task{
		Status:       task.TaskStatusFailed,
		ErrorCode:    9005,
		ErrorMessage: "Invalid parameter name: Device.B",
	}
	out := classifyGPVFailure(tk, []string{"Device.A", "Device.B", "Device.C"})
	require.Len(t, out, 3)
	require.Equal(t, OutcomeUnknown, out[0])
	require.Equal(t, OutcomeUnsupported, out[1])
	require.Equal(t, OutcomeUnknown, out[2])
}

// Test_classifyGPVFailure_Non9005：非 9005 全 unknown，绝不误标。
func Test_classifyGPVFailure_Non9005(t *testing.T) {
	tk := &task.Task{
		Status:    task.TaskStatusFailed,
		ErrorCode: 9007, // 值越界
	}
	out := classifyGPVFailure(tk, []string{"Device.X"})
	require.Equal(t, OutcomeUnknown, out[0])
}

// ── taskProber 集成（用 fake TaskSubmitter）────────────────────────────

type fakeTaskSubmitter struct {
	createErr error
	getErr    error
	tasks     map[string]*task.Task
	calls     int
}

func newFakeTaskSubmitter() *fakeTaskSubmitter {
	return &fakeTaskSubmitter{tasks: map[string]*task.Task{}}
}

func (f *fakeTaskSubmitter) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	f.calls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	id := uuid.New().String()
	tk := &task.Task{
		ID:        id,
		DeviceSN:  req.DeviceSN,
		Method:    req.Method,
		Params:    req.Params,
		Status:    task.TaskStatusCompleted, // 默认 happy
		CreatedAt: time.Now(),
	}
	f.tasks[id] = tk
	return tk, nil
}

func (f *fakeTaskSubmitter) GetTask(_ context.Context, id string) (*task.Task, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.tasks[id], nil
}

// Test_TaskProber_Probe_Supported：CreateTask 立即返 completed，全部 path supported。
func Test_TaskProber_Probe_Supported(t *testing.T) {
	sub := newFakeTaskSubmitter()
	p := NewTaskProber(sub, 10*time.Millisecond, 2*time.Second, nil)
	got := p.Probe(context.Background(), "SN1", []string{"Device.A"}, 0)
	require.Len(t, got, 1)
	require.Equal(t, OutcomeSupported, got[0].Outcome)
}

// Test_TaskProber_Probe_Unsupported_9005：fake submitter 注入 failed+9005 → unsupported。
func Test_TaskProber_Probe_Unsupported_9005(t *testing.T) {
	sub := newFakeTaskSubmitter()
	// wrapped 在 CreateTask 后立即翻 status，模拟 ACS handler 写 9005 fault
	wrapped := &wrappedSubmitter{
		inner: sub,
		afterCreate: func(tk *task.Task) {
			tk.Status = task.TaskStatusFailed
			tk.ErrorCode = 9005
			tk.ErrorMessage = "Invalid parameter name: Device.X"
		},
	}
	p := NewTaskProber(wrapped, 10*time.Millisecond, 2*time.Second, nil)
	got := p.Probe(context.Background(), "SN1", []string{"Device.X"}, 0)
	require.Len(t, got, 1)
	require.Equal(t, OutcomeUnsupported, got[0].Outcome)
	require.Equal(t, 9005, got[0].FaultCode)
}

// Test_TaskProber_Probe_Unknown_NonFaultCode：failed 但非 9005 → unknown。
func Test_TaskProber_Probe_Unknown_NonFaultCode(t *testing.T) {
	sub := newFakeTaskSubmitter()
	wrapped := &wrappedSubmitter{
		inner: sub,
		afterCreate: func(tk *task.Task) {
			tk.Status = task.TaskStatusFailed
			tk.ErrorCode = 9007
		},
	}
	p := NewTaskProber(wrapped, 10*time.Millisecond, 2*time.Second, nil)
	got := p.Probe(context.Background(), "SN1", []string{"Device.X"}, 0)
	require.Equal(t, OutcomeUnknown, got[0].Outcome)
}

// Test_TaskProber_Probe_CreateError：入队失败 → 全 unknown，不阻塞后续批次。
func Test_TaskProber_Probe_CreateError(t *testing.T) {
	sub := newFakeTaskSubmitter()
	sub.createErr = errors.New("redis down")
	p := NewTaskProber(sub, 10*time.Millisecond, 2*time.Second, nil)
	got := p.Probe(context.Background(), "SN1", []string{"Device.A", "Device.B"}, 0)
	require.Len(t, got, 2)
	for _, r := range got {
		require.Equal(t, OutcomeUnknown, r.Outcome)
	}
}

// Test_TaskProber_Probe_MarshalParams：验证发出的 GPV params 形态为 {"names":[...]}。
func Test_TaskProber_Probe_MarshalParams(t *testing.T) {
	sub := newFakeTaskSubmitter()
	p := NewTaskProber(sub, 10*time.Millisecond, 2*time.Second, nil)
	_ = p.Probe(context.Background(), "SN1", []string{"Device.X", "Device.Y"}, 0)
	require.Len(t, sub.tasks, 1)
	var tk *task.Task
	for _, v := range sub.tasks {
		tk = v
	}
	require.NotNil(t, tk)
	require.Equal(t, "GetParameterValues", tk.Method)
	var payload map[string][]string
	require.NoError(t, json.Unmarshal(tk.Params, &payload))
	require.Equal(t, []string{"Device.X", "Device.Y"}, payload["names"])
}

// wrappedSubmitter 在 CreateTask 后让测试改 task state（模拟 CPE 响应）。
type wrappedSubmitter struct {
	inner       *fakeTaskSubmitter
	afterCreate func(*task.Task)
}

func (w *wrappedSubmitter) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	tk, err := w.inner.CreateTask(ctx, req)
	if err == nil && w.afterCreate != nil {
		w.afterCreate(tk)
	}
	return tk, err
}

func (w *wrappedSubmitter) GetTask(ctx context.Context, id string) (*task.Task, error) {
	return w.inner.GetTask(ctx, id)
}

// ──────────────────────────────────────────────────────────────────────
// T-0180: ACS GPV failure 写结构化 param_faults[] + Probe retry 收敛
// ──────────────────────────────────────────────────────────────────────

// Test_classifyGPVFailure_PrefersStructuredParamFaults: t.Result 含
// param_faults[] 时优先用,而非 ErrorMessage 文本抽取。
func Test_classifyGPVFailure_PrefersStructuredParamFaults(t *testing.T) {
	// ErrorMessage 故意写错 path,如果优先级正确应被 result.param_faults 覆盖
	tk := &task.Task{
		Status:       task.TaskStatusFailed,
		ErrorCode:    9005,
		ErrorMessage: "Invalid parameter name: Device.WRONG",
		Result: []byte(`{"param_faults":[{"parameter_name":"Device.B","fault_code":9005,"fault_string":"..."}]}`),
	}
	out := classifyGPVFailure(tk, []string{"Device.A", "Device.B", "Device.C"})
	require.Equal(t, OutcomeUnknown, out[0], "A 不在 param_faults 应 unknown")
	require.Equal(t, OutcomeUnsupported, out[1], "B 在 param_faults 应 unsupported")
	require.Equal(t, OutcomeUnknown, out[2], "C 不在 param_faults 应 unknown")
}

// Test_classifyGPVFailure_FallsBackToErrorMessage: result 空时回退 ErrorMessage。
func Test_classifyGPVFailure_FallsBackToErrorMessage(t *testing.T) {
	tk := &task.Task{
		Status:       task.TaskStatusFailed,
		ErrorCode:    9005,
		ErrorMessage: "Invalid parameter name: Device.B",
		Result:       nil,
	}
	out := classifyGPVFailure(tk, []string{"Device.A", "Device.B"})
	require.Equal(t, OutcomeUnknown, out[0])
	require.Equal(t, OutcomeUnsupported, out[1])
}

// Test_classifyGPVFailure_IgnoresNon9005Faults: result 含非 9005 fault
// (如 9007 值越界)不应映射到 unsupported。
func Test_classifyGPVFailure_IgnoresNon9005Faults(t *testing.T) {
	tk := &task.Task{
		Status:    task.TaskStatusFailed,
		ErrorCode: 9005,
		Result:    []byte(`{"param_faults":[{"parameter_name":"Device.B","fault_code":9007}]}`),
	}
	out := classifyGPVFailure(tk, []string{"Device.A", "Device.B"})
	require.Equal(t, OutcomeUnknown, out[0])
	require.Equal(t, OutcomeUnknown, out[1], "9007 非 9005 应 unknown")
}

// Test_extractBadPathsFromResult_Empty: 空 / 损坏 JSON 返 nil 不 panic。
func Test_extractBadPathsFromResult_Empty(t *testing.T) {
	require.Nil(t, extractBadPathsFromResult(nil))
	require.Nil(t, extractBadPathsFromResult([]byte(``)))
	require.Nil(t, extractBadPathsFromResult([]byte(`not-json`)))
	require.Nil(t, extractBadPathsFromResult([]byte(`{}`)))
	require.Nil(t, extractBadPathsFromResult([]byte(`{"param_faults":[]}`)))
}

// Test_TaskProber_Probe_RetriesOnPartialFailure: batch=3,首次 9005 暴露 1 个
// badPath,剩余 2 个进入下一轮 retry。下一轮全 supported → 总结果 1 unsupported + 2 supported。
func Test_TaskProber_Probe_RetriesOnPartialFailure(t *testing.T) {
	sub := newFakeTaskSubmitter()
	iteration := 0
	wrapped := &wrappedSubmitter{
		inner: sub,
		afterCreate: func(tk *task.Task) {
			if iteration == 0 {
				// 第 1 轮: 3 path 中 Device.B 不支持
				tk.Status = task.TaskStatusFailed
				tk.ErrorCode = 9005
				tk.Result = []byte(`{"param_faults":[{"parameter_name":"Device.B","fault_code":9005,"fault_string":"..."}]}`)
			} else {
				// 第 2 轮: 剩余 [Device.A, Device.C] 全支持
				tk.Status = task.TaskStatusCompleted
			}
			iteration++
		},
	}
	p := NewTaskProber(wrapped, 1*time.Millisecond, 2*time.Second, nil)
	got := p.Probe(context.Background(), "SN1",
		[]string{"Device.A", "Device.B", "Device.C"}, 0)

	require.Len(t, got, 3)
	// 按入参顺序返回 — index 与 Probe input 一致
	require.Equal(t, "Device.A", got[0].StandardPath)
	require.Equal(t, OutcomeSupported, got[0].Outcome)
	require.Equal(t, "Device.B", got[1].StandardPath)
	require.Equal(t, OutcomeUnsupported, got[1].Outcome)
	require.Equal(t, "Device.C", got[2].StandardPath)
	require.Equal(t, OutcomeSupported, got[2].Outcome)
	require.Equal(t, 2, sub.calls, "应有 2 次 CreateTask(原批 + retry)")
}

// Test_TaskProber_Probe_RetryStopsWhenNoProgress: 整批失败但无 path 归因
// (param_faults 空,ErrorMessage 也无 badPath) → 剩余全 Unknown,不无限循环。
func Test_TaskProber_Probe_RetryStopsWhenNoProgress(t *testing.T) {
	sub := newFakeTaskSubmitter()
	calls := 0
	wrapped := &wrappedSubmitter{
		inner: sub,
		afterCreate: func(tk *task.Task) {
			calls++
			tk.Status = task.TaskStatusFailed
			tk.ErrorCode = 9001 // 非 9005,classifyGPVFailure 全标 Unknown
		},
	}
	p := NewTaskProber(wrapped, 1*time.Millisecond, 2*time.Second, nil)
	got := p.Probe(context.Background(), "SN1",
		[]string{"Device.A", "Device.B"}, 0)
	require.Len(t, got, 2)
	require.Equal(t, OutcomeUnknown, got[0].Outcome)
	require.Equal(t, OutcomeUnknown, got[1].Outcome)
	require.Equal(t, 1, sub.calls, "非 9005 fault 不应触发 retry")
}
