package ops

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

// =============================================================================
// T-0102-e — per-device rate limit + retry-with-backoff + per-cmd timeout
// =============================================================================

// flakyEnqueuer fails the first N CreateTask calls then succeeds.
// Captures all requests so tests assert on ExpiresIn + Source + retries.
type flakyEnqueuer struct {
	mu        sync.Mutex
	failFirst int
	count     atomic.Int32
	requests  []*task.CreateTaskRequest
}

func (f *flakyEnqueuer) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	c := f.count.Add(1)
	if int(c) <= f.failFirst {
		return nil, errors.New("transient db error")
	}
	return &task.Task{ID: uuid.NewString(), DeviceSN: req.DeviceSN, Method: req.Method}, nil
}

func (f *flakyEnqueuer) GetQueueLength(_ context.Context, _ string) (int64, error) { return 0, nil }

// V1 — CreateTaskRequest.ExpiresIn is populated to PRD §4.2.3 default 60s
func TestDispatchInlineRPC_PerCmdTimeoutInExpiresIn(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "reboot"})
	devices, _ := json.Marshal([]string{"SN-1"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: opsTaskID, Status: OpsTaskPending, DeviceSNs: devices, Message: string(envJSON), Creator: "alice"}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	require.NoError(t, executor.Run(context.Background(), opsTaskID))

	require.Len(t, enq.requests, 1)
	assert.Equal(t, DefaultPerCmdTimeoutSeconds, enq.requests[0].ExpiresIn,
		"CreateTaskRequest.ExpiresIn must be set to PRD §4.2.3 default 60s")
}

// V2 — transient enqueue failure retries up to DefaultEnqueueMaxRetries
// and succeeds on attempt 3 (fail-first=2)
func TestDispatchInlineRPC_RetryRecoversFromTransientFailure(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "reboot"})
	devices, _ := json.Marshal([]string{"SN-1"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: opsTaskID, Status: OpsTaskPending, DeviceSNs: devices, Message: string(envJSON), Creator: "alice"}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &flakyEnqueuer{failFirst: 2}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	start := time.Now()
	require.NoError(t, executor.Run(context.Background(), opsTaskID))
	elapsed := time.Since(start)

	assert.Equal(t, int32(3), enq.count.Load(), "should attempt 3 times (2 fail + 1 success)")
	rows := captor.all()
	require.Len(t, rows, 1)
	assert.Equal(t, "running", rows[0].Status, "final attempt succeeded → execution row running")
	// Backoff for attempt 2 = 2s, attempt 3 = 4s; total >= 6s
	assert.GreaterOrEqual(t, elapsed, 6*time.Second,
		"retry backoff should add ~6s for 2 retries (2s + 4s)")
}

// V3 — exhausted retries write a failed execution row with attempt count in message
func TestDispatchInlineRPC_RetryExhaustedRecordsFailure(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "reboot"})
	devices, _ := json.Marshal([]string{"SN-1"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: opsTaskID, Status: OpsTaskPending, DeviceSNs: devices, Message: string(envJSON), Creator: "alice"}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &flakyEnqueuer{failFirst: 99} // always fail
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	require.NoError(t, executor.Run(context.Background(), opsTaskID),
		"exhausted retries must not bubble out of Run")

	assert.Equal(t, int32(DefaultEnqueueMaxRetries), enq.count.Load(),
		"exactly DefaultEnqueueMaxRetries attempts before giving up")
	rows := captor.all()
	require.Len(t, rows, 1)
	assert.Equal(t, "failed", rows[0].Status)
	assert.Contains(t, rows[0].ErrorMessage, "enqueue after")
	assert.Contains(t, rows[0].ErrorMessage, "transient db error")
}

// V4 — retry loop respects ctx cancellation between attempts
func TestDispatchInlineRPC_RetryCtxCancelExitsEarly(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "reboot"})
	devices, _ := json.Marshal([]string{"SN-1"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: opsTaskID, Status: OpsTaskPending, DeviceSNs: devices, Message: string(envJSON), Creator: "alice"}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &flakyEnqueuer{failFirst: 99}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	require.NoError(t, executor.Run(ctx, opsTaskID))
	elapsed := time.Since(start)

	assert.LessOrEqual(t, elapsed, 3*time.Second,
		"ctx cancel should short-circuit retry backoff well before full 6s budget")
	rows := captor.all()
	require.Len(t, rows, 1)
	assert.Equal(t, "failed", rows[0].Status)
	assert.Contains(t, rows[0].ErrorMessage, "cancelled")
}

// V5 — per-device rate limiter wired via SetLimiter throttles back-to-back
// RPC against the same device_sn. We use a low rate (1/sec) and verify
// the second device call waits.
func TestDispatchInlineRPC_PerDeviceRateLimit(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "get_param"})
	// Same SN twice — exercises per-device limiter.
	devices, _ := json.Marshal([]string{"SN-shared", "SN-shared"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: opsTaskID, Status: OpsTaskPending, DeviceSNs: devices, Message: string(envJSON), Creator: "alice"}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)
	lim := NewConcurrencyLimiter(0, 0)
	lim.SetPerDeviceRate(1) // 1 RPC/sec/device — second call must wait ~1s
	executor.SetLimiter(lim)

	start := time.Now()
	require.NoError(t, executor.Run(context.Background(), opsTaskID))
	elapsed := time.Since(start)

	assert.Equal(t, int32(2), enq.created.Load(), "both calls eventually succeed")
	assert.GreaterOrEqual(t, elapsed, 800*time.Millisecond,
		"second call to same device must wait for rate.Limiter (1/sec ≈ 1s)")
	assert.Equal(t, 1, lim.DeviceLimiterCount(),
		"single device_sn → single per-device limiter created")
}

// V6 — different device_sns get independent limiters; no inter-device blocking
func TestDispatchInlineRPC_PerDeviceLimiterIndependent(t *testing.T) {
	opsTaskID := uuid.New()
	envJSON, _ := json.Marshal(RPCInlineEnvelope{Kind: RPCInlineKind, Action: "get_param"})
	devices, _ := json.Marshal([]string{"SN-A", "SN-B", "SN-C"})
	taskRepo := &mockTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*OpsTask, error) {
			return &OpsTask{ID: opsTaskID, Status: OpsTaskPending, DeviceSNs: devices, Message: string(envJSON), Creator: "alice"}, nil
		},
		updateStatusFn: func(_ context.Context, _ *OpsTask) error { return nil },
	}
	captor := &execCaptor{}
	enq := &fakeEnqueuer{}
	executor, _ := newRPCTestExecutor(t, taskRepo, captor.repo(), enq)
	lim := NewConcurrencyLimiter(0, 0)
	lim.SetPerDeviceRate(1) // 1/sec but 3 different devices — each gets its own slot
	executor.SetLimiter(lim)

	start := time.Now()
	require.NoError(t, executor.Run(context.Background(), opsTaskID))
	elapsed := time.Since(start)

	assert.Equal(t, int32(3), enq.created.Load())
	// Each device gets initial burst slot; 3 distinct devices should complete fast.
	assert.Less(t, elapsed, 500*time.Millisecond,
		"3 distinct devices have 3 independent limiters → no inter-device wait")
	assert.Equal(t, 3, lim.DeviceLimiterCount())
}

// V7 — ConcurrencyLimiter.SetPerDeviceRate clamps non-positive to default
func TestConcurrencyLimiter_SetPerDeviceRate_DefaultsOnZero(t *testing.T) {
	lim := NewConcurrencyLimiter(0, 0)
	assert.Equal(t, DefaultPerDeviceRPCRateLimit, lim.perDeviceRate)

	lim.SetPerDeviceRate(0)
	assert.Equal(t, DefaultPerDeviceRPCRateLimit, lim.perDeviceRate)

	lim.SetPerDeviceRate(-5)
	assert.Equal(t, DefaultPerDeviceRPCRateLimit, lim.perDeviceRate)

	lim.SetPerDeviceRate(20)
	assert.Equal(t, 20, lim.perDeviceRate)
}

// V8 — WaitForDevice early-returns on empty device_sn (no limiter created)
func TestConcurrencyLimiter_WaitForDevice_EmptySN_NoOp(t *testing.T) {
	lim := NewConcurrencyLimiter(0, 0)
	require.NoError(t, lim.WaitForDevice(context.Background(), ""))
	assert.Equal(t, 0, lim.DeviceLimiterCount(),
		"empty device_sn must not create a limiter entry")
}
