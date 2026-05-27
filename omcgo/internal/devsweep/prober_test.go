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
