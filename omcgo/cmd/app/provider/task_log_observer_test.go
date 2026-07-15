package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
)

// stubLogRepo 实现 admin.LogRepository，仅捕获 CreateTaskLog 调用供断言。
// 其余 5 个方法为满足接口的空实现。
type stubLogRepo struct {
	calls chan admin.CreateTaskLogRequest
}

func newStubLogRepo() *stubLogRepo {
	return &stubLogRepo{calls: make(chan admin.CreateTaskLogRequest, 4)}
}

func (s *stubLogRepo) CreateTaskLog(_ context.Context, req admin.CreateTaskLogRequest) error {
	s.calls <- req
	return nil
}

func (s *stubLogRepo) CreateLoginLog(context.Context, admin.CreateLoginLogRequest) error {
	return nil
}
func (s *stubLogRepo) ListLoginLogs(context.Context, admin.LoginLogFilter) (*model.ListResponse[admin.LoginLog], error) {
	return nil, nil
}
func (s *stubLogRepo) CreateOperLog(context.Context, admin.CreateOperLogRequest) error { return nil }
func (s *stubLogRepo) ListOperLogs(context.Context, admin.OperLogFilter) (*model.ListResponse[admin.OperLog], error) {
	return nil, nil
}
func (s *stubLogRepo) ListTaskLogs(context.Context, admin.TaskLogFilter) (*model.ListResponse[admin.TaskLog], error) {
	return nil, nil
}

func (s *stubLogRepo) waitOne(t *testing.T) admin.CreateTaskLogRequest {
	t.Helper()
	select {
	case req := <-s.calls:
		return req
	case <-time.After(2 * time.Second):
		t.Fatal("expected CreateTaskLog to be called, timed out")
		return admin.CreateTaskLogRequest{}
	}
}

func (s *stubLogRepo) assertNoCall(t *testing.T) {
	t.Helper()
	select {
	case req := <-s.calls:
		t.Fatalf("expected no CreateTaskLog call, got: %+v", req)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestTaskLogObserver_WritesTerminalTask(t *testing.T) {
	repo := newStubLogRepo()
	obs := newTaskLogObserver(repo, zap.NewNop())

	operatorID := uuid.New()
	sent := time.Now().Add(-2 * time.Second)
	done := sent.Add(1500 * time.Millisecond)

	obs.OnTaskCompleted(context.Background(), &task.Task{
		ID:          "task-1",
		Method:      "SetParameterValues",
		Status:      task.TaskStatusCompleted,
		DeviceSN:    "SN-001",
		Description: "下发配置",
		CreatorID:   operatorID.String(),
		SentAt:      &sent,
		CompletedAt: &done,
	})

	got := repo.waitOne(t)
	assert.Equal(t, "SetParameterValues", got.TaskType)
	assert.Equal(t, "task-1", got.TaskID)
	assert.Equal(t, "completed", got.Status)
	assert.Equal(t, "SN-001", got.Target)
	assert.Equal(t, "下发配置", got.Detail)
	require.NotNil(t, got.OperatorID)
	assert.Equal(t, operatorID, *got.OperatorID)
	assert.Equal(t, operatorID.String(), got.Operator)
	assert.Equal(t, 1500, got.CostMs)
}

func TestTaskLogObserver_WritesFailedTaskWithError(t *testing.T) {
	repo := newStubLogRepo()
	obs := newTaskLogObserver(repo, zap.NewNop())

	obs.OnTaskCompleted(context.Background(), &task.Task{
		ID:           "task-2",
		Method:       "Reboot",
		Status:       task.TaskStatusFailed,
		ErrorMessage: "device offline",
	})

	got := repo.waitOne(t)
	assert.Equal(t, "failed", got.Status)
	assert.Equal(t, "device offline", got.ErrorMsg)
	assert.Nil(t, got.OperatorID, "无 CreatorID → operator_id NULL")
	assert.Equal(t, 0, got.CostMs, "无 sent/completed 时间戳 → cost 0")
}

func TestTaskLogObserver_DoesNotTreatParamSyncCorrelationAsOperator(t *testing.T) {
	repo := newStubLogRepo()
	obs := newTaskLogObserver(repo, zap.NewNop())
	now := time.Now()

	obs.OnTaskCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), DeviceSN: "SN-1", Method: "GetParameterValues",
		Source: task.TaskSourceParamSync, CreatorID: uuid.NewString(),
		Status: task.TaskStatusFailed, CompletedAt: &now,
	})

	got := repo.waitOne(t)
	assert.Nil(t, got.OperatorID)
}

func TestTaskLogObserver_SkipsNonTerminalStatus(t *testing.T) {
	repo := newStubLogRepo()
	obs := newTaskLogObserver(repo, zap.NewNop())

	// 在途态（sent/pending）不应写日志（防御性过滤）。
	obs.OnTaskCompleted(context.Background(), &task.Task{ID: "task-3", Status: task.TaskStatusSent})
	obs.OnTaskCompleted(context.Background(), &task.Task{ID: "task-4", Status: task.TaskStatusPending})

	repo.assertNoCall(t)
}

func TestTaskLogObserver_RecordsExpiredAndCancelled(t *testing.T) {
	repo := newStubLogRepo()
	obs := newTaskLogObserver(repo, zap.NewNop())

	obs.OnTaskCompleted(context.Background(), &task.Task{ID: "task-5", Status: task.TaskStatusExpired})
	assert.Equal(t, "expired", repo.waitOne(t).Status)

	obs.OnTaskCompleted(context.Background(), &task.Task{ID: "task-6", Status: task.TaskStatusCancelled})
	assert.Equal(t, "cancelled", repo.waitOne(t).Status)
}

func TestTaskLogObserver_NilTaskNoop(t *testing.T) {
	repo := newStubLogRepo()
	obs := newTaskLogObserver(repo, zap.NewNop())
	assert.NotPanics(t, func() { obs.OnTaskCompleted(context.Background(), nil) })
	repo.assertNoCall(t)
}

func TestParseOperatorID(t *testing.T) {
	id := uuid.New()
	assert.Nil(t, parseOperatorID(""))
	assert.Nil(t, parseOperatorID("not-a-uuid"))
	got := parseOperatorID(id.String())
	require.NotNil(t, got)
	assert.Equal(t, id, *got)
}
