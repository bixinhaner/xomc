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
	incCalls   []incCall
	incErr     error
	getMMLTask *MMLTask
	getErr     error
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

func (r *aggregatorTestRepo) Update(_ context.Context, _ *MMLTask) error { return nil }

func newAggregatorTestRepo(mmlTask *MMLTask) *aggregatorTestRepo {
	return &aggregatorTestRepo{mockTaskRepo: &mockTaskRepo{}, getMMLTask: mmlTask}
}

func newAggregatorWithHub(repo TaskRepository, hub SSEPublisher) *ResultAggregator {
	return NewResultAggregator(repo, nil, hub, zap.NewNop())
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
	agg := NewResultAggregator(repo, nil, nil, zap.NewNop())

	dt := &task.Task{
		SourceID: mmlID.String(), DeviceSN: "SN-3", Status: task.TaskStatusCompleted,
	}
	assert.NotPanics(t, func() {
		agg.OnTaskCompleted(context.Background(), dt)
	})
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
