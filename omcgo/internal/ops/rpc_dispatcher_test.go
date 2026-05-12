package ops

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// =============================================================================
// T-0102-c — actionToRPCMethod mapping + TaskExecutor inline RPC dispatch
// =============================================================================

func TestActionToRPCMethod(t *testing.T) {
	cases := []struct {
		action  string
		want    string
		wantErr bool
	}{
		{"reboot", "Reboot", false},
		{"REBOOT", "Reboot", false},
		{" reboot ", "Reboot", false}, // whitespace tolerated
		{"factory_reset", "FactoryReset", false},
		{"factoryreset", "FactoryReset", false},
		{"get_param", "GetParameterValues", false},
		{"get_parameter_values", "GetParameterValues", false},
		{"GetParameterValues", "GetParameterValues", false},
		{"set_param", "SetParameterValues", false},
		{"get_rpc_methods", "GetRPCMethods", false},
		{"getrpcmethods", "GetRPCMethods", false},
		{"", "", true},                // empty
		{"unknown_action", "", true},
		{"download_firmware", "", true}, // not in T-0102-c scope
	}
	for _, c := range cases {
		c := c
		t.Run(c.action, func(t *testing.T) {
			got, err := actionToRPCMethod(c.action)
			if c.wantErr {
				assert.Error(t, err, "action %q should map to error", c.action)
				return
			}
			require.NoError(t, err, "action %q should map cleanly", c.action)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestParseInlineRPCEnvelope(t *testing.T) {
	t.Run("valid rpc envelope", func(t *testing.T) {
		msg := `{"kind":"rpc","action":"reboot","params":{}}`
		env, ok := parseInlineRPCEnvelope(msg)
		require.True(t, ok)
		assert.Equal(t, "rpc", env.Kind)
		assert.Equal(t, "reboot", env.Action)
	})
	t.Run("empty message → not envelope", func(t *testing.T) {
		_, ok := parseInlineRPCEnvelope("")
		assert.False(t, ok)
	})
	t.Run("non-json message → not envelope", func(t *testing.T) {
		_, ok := parseInlineRPCEnvelope("just a status message")
		assert.False(t, ok)
	})
	t.Run("wrong kind → not envelope", func(t *testing.T) {
		_, ok := parseInlineRPCEnvelope(`{"kind":"mml","action":"reboot"}`)
		assert.False(t, ok)
	})
}

// execCaptor wraps stubTaskExecRepo with a thread-safe slice of created
// execution rows so tests can inspect what the dispatcher wrote without
// touching the real DB.
type execCaptor struct {
	mu   sync.Mutex
	rows []*OpsTaskExecution
}

func (c *execCaptor) repo() *stubTaskExecRepo {
	return &stubTaskExecRepo{createFn: func(_ context.Context, ex *OpsTaskExecution) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		cp := *ex
		c.rows = append(c.rows, &cp)
		return nil
	}}
}

func (c *execCaptor) all() []*OpsTaskExecution {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*OpsTaskExecution, len(c.rows))
	copy(out, c.rows)
	return out
}

// fakeEnqueuer captures device-task creates so tests can assert on them.
type fakeEnqueuer struct {
	mu       sync.Mutex
	requests []*task.CreateTaskRequest
	createErr error
	failOnSN string // if set, CreateTask returns an error for this device_sn
	created  atomic.Int32
}

func (f *fakeEnqueuer) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.failOnSN != "" && req.DeviceSN == f.failOnSN {
		return nil, errors.New("enqueue refused for " + req.DeviceSN)
	}
	f.created.Add(1)
	return &task.Task{ID: uuid.NewString(), DeviceSN: req.DeviceSN, Method: req.Method}, nil
}

func (f *fakeEnqueuer) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// newRPCTestExecutor wires a TaskExecutor with our in-memory taskRepo +
// stub exec repo + fake enqueuer; SSE hub is real so tests can subscribe.
func newRPCTestExecutor(t *testing.T, taskRepo TaskRepository, execRepo TaskExecutionRepository, enq task.Enqueuer) (*TaskExecutor, *SSEHub) {
	t.Helper()
	logger := zap.NewNop()
	auditSvc := NewAuditLogService(noopAuditRepo{}, logger)
	exec := NewTaskExecutor(taskRepo, execRepo, auditSvc, logger)
	hub := NewSSEHub()
	exec.SetEnqueuer(enq)
	exec.SetSSEHub(hub)
	return exec, hub
}

// V8 — TaskExecutor.Run on inline-RPC envelope fans out one
// device-task per device_sn via the injected enqueuer.
func TestTaskExecutor_Run_InlineRPC_FansOut(t *testing.T) {
	opsTaskID := uuid.New()
	devices := []string{"SN-1", "SN-2", "SN-3"}
	deviceSNsJSON, _ := json.Marshal(devices)
	envelope := RPCInlineEnvelope{
		Kind: RPCInlineKind, Action: "reboot",
	}
	envJSON, _ := json.Marshal(envelope)

	opsTask := OpsTask{
		ID:        opsTaskID,
		Status:    OpsTaskPending,
		DeviceSNs: deviceSNsJSON,
		Message:   string(envJSON),
		Creator:   "alice",
		RiskLevel: RiskCautious,
	}
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			require.Equal(t, opsTaskID, id)
			cp := opsTask
			return &cp, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	require.NoError(t, executor.Run(context.Background(), opsTaskID))

	// One CreateTask call per device_sn, all carrying method=Reboot.
	require.Len(t, enq.requests, 3)
	for i, req := range enq.requests {
		assert.Equal(t, devices[i], req.DeviceSN, "device index %d", i)
		assert.Equal(t, "Reboot", req.Method)
		assert.Equal(t, task.TaskSourceOps, req.Source)
		assert.Equal(t, opsTaskID.String(), req.SourceID)
		assert.Equal(t, "alice", req.CreatorID)
		assert.Equal(t, i, req.DeviceIndex)
	}
	assert.Equal(t, int32(3), enq.created.Load(), "all 3 devices enqueued")

	rows := captor.all()
	assert.Len(t, rows, 3)
	for _, ex := range rows {
		assert.Equal(t, "rpc", ex.StepType)
		assert.Equal(t, "reboot", ex.StepName)
		assert.Equal(t, "running", ex.Status, "all 3 should succeed")
	}
}

// V8b — when the enqueuer fails for one device, the other devices still
// dispatch and the failing device's execution row is recorded as failed.
func TestTaskExecutor_Run_InlineRPC_PartialFailure(t *testing.T) {
	opsTaskID := uuid.New()
	devices := []string{"SN-1", "SN-bad", "SN-3"}
	deviceSNsJSON, _ := json.Marshal(devices)
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "factory_reset"})

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID: opsTaskID, Status: OpsTaskPending,
				DeviceSNs: deviceSNsJSON, Message: string(envJSON),
				Creator: "bob", RiskLevel: RiskDangerous,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{failOnSN: "SN-bad"}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	require.NoError(t, executor.Run(context.Background(), opsTaskID),
		"partial failure must not bubble out of Run")

	// All 3 CreateTask attempted; 2 succeeded.
	assert.Len(t, enq.requests, 3)
	assert.Equal(t, int32(2), enq.created.Load())

	// Execution rows: 2 running, 1 failed.
	rows := captor.all()
	require.Len(t, rows, 3)
	var running, failed int
	for _, ex := range rows {
		switch ex.Status {
		case "running":
			running++
		case "failed":
			failed++
			assert.Contains(t, ex.ErrorMessage, "SN-bad",
				"error message must identify the failing device")
		}
	}
	assert.Equal(t, 2, running)
	assert.Equal(t, 1, failed)
}

// V8c — Run with a non-RPC Message (template-driven task) falls back to
// the MVP placeholder path — no enqueuer fan-out.
func TestTaskExecutor_Run_NonInlineMessage_FallsBackToPlaceholder(t *testing.T) {
	opsTaskID := uuid.New()
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID: opsTaskID, Status: OpsTaskPending,
				DeviceSNs: []byte(`["SN-1"]`),
				Message:   "freeform task description, no rpc envelope",
				Creator:   "carol",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	require.NoError(t, executor.Run(context.Background(), opsTaskID))

	// Enqueuer untouched; one placeholder execution row written.
	assert.Equal(t, int32(0), enq.created.Load(), "non-RPC task must not fan out")
	rows := captor.all()
	require.Len(t, rows, 1)
	assert.Equal(t, "executor_dispatched", rows[0].StepName)
	assert.Equal(t, "system", rows[0].StepType)
}

// V8d — When TaskExecutor.enqueuer is nil (DI not wired), an inline RPC
// task must also fall back to the placeholder path so we never silently
// drop the task.
func TestTaskExecutor_Run_InlineRPC_NoEnqueuer_FallsBackToPlaceholder(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "reboot"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID: opsTaskID, Status: OpsTaskPending,
				DeviceSNs: []byte(`["SN-1"]`),
				Message:   string(envJSON),
				Creator:   "dave",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	logger := zap.NewNop()
	auditSvc := NewAuditLogService(noopAuditRepo{}, logger)
	executor := NewTaskExecutor(taskRepo, captor.repo(), auditSvc, logger)
	// Deliberately do NOT SetEnqueuer.

	require.NoError(t, executor.Run(context.Background(), opsTaskID))

	// Falls back: placeholder row written.
	rows := captor.all()
	require.Len(t, rows, 1)
	assert.Equal(t, "executor_dispatched", rows[0].StepName)
}

// V8e — SSE command.dispatched event is published per device on the
// per-ops-task channel when the hub is wired.
func TestTaskExecutor_Run_InlineRPC_SSEDispatchEvents(t *testing.T) {
	opsTaskID := uuid.New()
	devices := []string{"SN-1", "SN-2"}
	deviceSNsJSON, _ := json.Marshal(devices)
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "get_param"})

	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID: opsTaskID, Status: OpsTaskPending,
				DeviceSNs: deviceSNsJSON, Message: string(envJSON), Creator: "erin",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{}
	executor, hub := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	ch, unsub := hub.Subscribe("command:" + opsTaskID.String())
	defer unsub()

	require.NoError(t, executor.Run(context.Background(), opsTaskID))

	// Two SSE events expected — drain non-blocking and assert.
	var got []SSEEvent
	for i := 0; i < 2; i++ {
		select {
		case evt := <-ch:
			got = append(got, evt)
		default:
			t.Fatalf("expected 2 SSE events, got %d", i)
		}
	}
	for _, evt := range got {
		assert.Equal(t, "command.dispatched", evt.Event)
		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(evt.Data, &data))
		assert.Equal(t, "get_param", data["action"])
		assert.Equal(t, "GetParameterValues", data["method"])
		assert.Equal(t, opsTaskID.String(), data["ops_task_id"])
	}
}
