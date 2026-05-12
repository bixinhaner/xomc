package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// =============================================================================
// T-0102-b — ExecuteRPC endpoint tests
//
// The endpoint persists an OpsTask via TaskCreator, optionally auto-dispatches
// via the executor when no approval is required, and publishes an SSE event
// for any active /commands/:id/stream subscriber. Tests drive the success +
// approval-gating + error paths without standing up a real Service or DB.
// =============================================================================

// mockTaskCreator captures the OpsTask the handler creates and returns it
// with a fresh UUID populated, mimicking the real Service.CreateTask
// behavior (which lets the DB or repo assign the ID).
type mockTaskCreator struct {
	mu        sync.Mutex
	created   *OpsTask
	createErr error
	idToAssign uuid.UUID
}

func (m *mockTaskCreator) CreateTask(_ context.Context, task *OpsTask) (*OpsTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return nil, m.createErr
	}
	cp := *task
	if m.idToAssign != uuid.Nil {
		cp.ID = m.idToAssign
	} else {
		cp.ID = uuid.New()
	}
	cp.Status = OpsTaskPending
	m.created = &cp
	return &cp, nil
}

// stubExecutor counts Run invocations so tests can verify auto-dispatch
// happened (or did not happen, in the approval-gated path).
type stubExecutor struct {
	runCount atomic.Int32
	runErr   error
	runIDs   chan uuid.UUID
}

func newStubExecutor() *stubExecutor {
	return &stubExecutor{runIDs: make(chan uuid.UUID, 4)}
}

func (s *stubExecutor) Run(_ context.Context, id uuid.UUID) error {
	s.runCount.Add(1)
	select {
	case s.runIDs <- id:
	default:
	}
	return s.runErr
}

func (s *stubExecutor) Pause(_ context.Context, _ uuid.UUID) error    { return nil }
func (s *stubExecutor) Resume(_ context.Context, _ uuid.UUID) error   { return nil }
func (s *stubExecutor) Cancel(_ context.Context, _ uuid.UUID) error   { return nil }
func (s *stubExecutor) Rollback(_ context.Context, _ uuid.UUID) error { return nil }
func (s *stubExecutor) ListExecutions(_ context.Context, _ TaskExecutionFilter) (*model.ListResponse[OpsTaskExecution], error) {
	return nil, nil
}

// newExtHandlerForRPCTest wires the minimum set of dependencies ExecuteRPC
// reaches. Unused services use zero-value structs since this endpoint only
// touches taskCreator, approvalSvc, auditSvc, executor, sseHub.
func newExtHandlerForRPCTest(t *testing.T, tc TaskCreator, exec TaskExecutorEngine) (*ExtHandler, *SSEHub) {
	t.Helper()
	logger := zap.NewNop()
	auditSvc := NewAuditLogService(noopAuditRepo{}, logger)
	approvalSvc := NewApprovalService(nil, auditSvc, logger) // RequiresApproval is purely Go
	hub := NewSSEHub()
	h := &ExtHandler{
		auditSvc:    auditSvc,
		executor:    exec,
		approvalSvc: approvalSvc,
		taskCreator: tc,
		sseHub:      hub,
		logger:      logger.Named("ops.ext"),
	}
	return h, hub
}

// noopAuditRepo satisfies AuditLogRepository for the tiny audit-write path.
type noopAuditRepo struct{}

func (noopAuditRepo) Create(_ context.Context, _ *OpsAuditLog) error { return nil }
func (noopAuditRepo) List(_ context.Context, _ AuditLogFilter) (*model.ListResponse[OpsAuditLog], error) {
	return nil, nil
}

func setupExtRouter(h *ExtHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/ops/commands/rpc", h.ExecuteRPC)
	return r
}

// =============================================================================
// V1 — single-device safe RPC (get_param) auto-dispatches + returns task_id
// =============================================================================
func TestExecuteRPC_V1_SingleDevice_SafeAutoDispatch(t *testing.T) {
	tc := &mockTaskCreator{}
	exec := newStubExecutor()
	h, _ := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	body := bytes.NewBufferString(`{"device_sn":"SN-1","action":"get_param","params":{"name":"Device.X"}}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["task_id"], "must return DB-assigned task id")
	assert.Equal(t, false, resp["approval_required"], "L1 safe action should not require approval")
	assert.Equal(t, string(RiskSafe), resp["risk_level"])
	assert.Contains(t, resp["stream_url"], "/api/v1/ops/commands/")

	require.NotNil(t, tc.created, "CreateTask must be invoked")
	assert.Equal(t, RiskSafe, tc.created.RiskLevel)
	assert.Equal(t, ApprovalNotRequired, tc.created.ApprovalState)
	assert.Equal(t, "system", tc.created.Creator, "creator falls back to system when no header set")
	assert.Equal(t, 1, tc.created.TotalSteps)

	// Inline command envelope persisted in Message for T-0102-c dispatch.
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(tc.created.Message), &envelope))
	assert.Equal(t, "rpc", envelope["kind"])
	assert.Equal(t, "get_param", envelope["action"])

	// Auto-dispatch fires in a goroutine; wait briefly with a deadline rather
	// than an unbounded sleep.
	select {
	case <-exec.runIDs:
	case <-time.After(time.Second):
		t.Fatal("executor.Run was not invoked within 1s")
	}
	assert.Equal(t, int32(1), exec.runCount.Load())
}

// =============================================================================
// V2 — factory_reset (L3 dangerous) requires approval, NO auto-dispatch
// =============================================================================
func TestExecuteRPC_V2_FactoryReset_RequiresApproval(t *testing.T) {
	tc := &mockTaskCreator{}
	exec := newStubExecutor()
	h, _ := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	body := bytes.NewBufferString(`{"device_sn":"SN-1","action":"factory_reset"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["approval_required"], "L3 dangerous action must gate on approval")
	assert.Equal(t, string(RiskDangerous), resp["risk_level"])

	require.NotNil(t, tc.created)
	assert.Equal(t, ApprovalPending, tc.created.ApprovalState)

	// Auto-dispatch MUST NOT fire — wait long enough that a stray goroutine
	// would have run already if the gate had failed.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), exec.runCount.Load(),
		"approval-gated task must not auto-dispatch")
}

// =============================================================================
// V3 — batch device_sns[] with reboot upgrades risk via device count >10
// =============================================================================
func TestExecuteRPC_V3_BatchReboot_RiskEscalation(t *testing.T) {
	tc := &mockTaskCreator{}
	exec := newStubExecutor()
	h, _ := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	// 11 devices: ApprovalService.EvaluateRiskLevel escalates >10 to cautious.
	// reboot is already cautious; we test the count-driven path is wired.
	devices := make([]string, 11)
	for i := range devices {
		devices[i] = "SN-batch"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"device_sns": devices,
		"action":     "reboot",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(RiskCautious), resp["risk_level"], "11 devices + reboot → cautious")
	assert.Equal(t, false, resp["approval_required"], "cautious does not require approval")

	require.NotNil(t, tc.created)
	// DeviceSNs JSON should round-trip the 11 entries.
	var stored []string
	require.NoError(t, json.Unmarshal(tc.created.DeviceSNs, &stored))
	assert.Len(t, stored, 11)
	assert.Equal(t, "rpc:reboot on 11 devices", tc.created.TaskName)
}

// =============================================================================
// V4 — batch >50 escalates to dangerous + approval required
// =============================================================================
func TestExecuteRPC_V4_LargeBatch_DangerousEscalation(t *testing.T) {
	tc := &mockTaskCreator{}
	exec := newStubExecutor()
	h, _ := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	devices := make([]string, 51) // >50 triggers RiskDangerous
	for i := range devices {
		devices[i] = "SN-large"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"device_sns": devices,
		"action":     "set_param",
		"params":     map[string]string{"k": "v"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(RiskDangerous), resp["risk_level"])
	assert.Equal(t, true, resp["approval_required"])

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), exec.runCount.Load(),
		"large batch must gate on approval")
}

// =============================================================================
// V5 — missing device_sn AND device_sns rejected with 400
// =============================================================================
func TestExecuteRPC_V5_MissingDevice_400(t *testing.T) {
	tc := &mockTaskCreator{}
	exec := newStubExecutor()
	h, _ := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	body := bytes.NewBufferString(`{"action":"get_param"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, tc.created, "CreateTask must not be invoked on validation failure")
}

// =============================================================================
// V6 — TaskCreator failure propagates as 500
// =============================================================================
func TestExecuteRPC_V6_CreateTaskError_500(t *testing.T) {
	tc := &mockTaskCreator{createErr: errors.New("db down")}
	exec := newStubExecutor()
	h, _ := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	body := bytes.NewBufferString(`{"device_sn":"SN-1","action":"reboot"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, int32(0), exec.runCount.Load(), "executor must not run when persistence fails")
}

// =============================================================================
// V7 — SSE command.enqueued event fires on the per-task channel
// =============================================================================
func TestExecuteRPC_V7_SSEEnqueuedEvent(t *testing.T) {
	fixedID := uuid.New()
	tc := &mockTaskCreator{idToAssign: fixedID}
	exec := newStubExecutor()
	h, hub := newExtHandlerForRPCTest(t, tc, exec)
	router := setupExtRouter(h)

	// Subscribe BEFORE making the request so the publisher has someone to
	// deliver to. The channel name is "command:<task_id>".
	ch, unsub := hub.Subscribe("command:" + fixedID.String())
	defer unsub()

	body := bytes.NewBufferString(`{"device_sn":"SN-1","action":"get_param"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/commands/rpc", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)

	// The publish is synchronous within ExecuteRPC; the event should be
	// waiting in the channel.
	select {
	case evt := <-ch:
		assert.Equal(t, "command.enqueued", evt.Event)
		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(evt.Data, &data))
		assert.Equal(t, fixedID.String(), data["task_id"])
		assert.Equal(t, "get_param", data["action"])
	case <-time.After(time.Second):
		t.Fatal("SSE enqueued event not delivered within 1s")
	}
}

// Compile-time guard: stubAuditRepo wasn't enough on its own; ensure the
// noop variant satisfies AuditLogRepository.
var _ AuditLogRepository = noopAuditRepo{}
