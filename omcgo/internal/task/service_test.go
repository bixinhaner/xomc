package task

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mock implementations ---

// mockTaskQueue implements the methods of RedisTaskQueue used by TaskService.
// We embed no interface; we just provide methods that match what TaskService calls.
type mockTaskQueue struct {
	pushFn              func(ctx context.Context, task *Task) error
	popFn               func(ctx context.Context, deviceSN string) (*Task, error)
	getByIDFn           func(ctx context.Context, taskID string) (*Task, error)
	getByCWMPIDFn       func(ctx context.Context, cwmpID string) (*Task, error)
	updateFn            func(ctx context.Context, task *Task) error
	deleteFn            func(ctx context.Context, taskID string) error
	lenFn               func(ctx context.Context, deviceSN string) (int64, error)
	markTaskSentFn      func(ctx context.Context, taskID, cwmpID string) error
	markTaskCompletedFn func(ctx context.Context, taskID string, result json.RawMessage) error
	markTaskFailedFn    func(ctx context.Context, taskID string, errorCode int, errorMsg string) error
	getStaleSentTasksFn func(ctx context.Context, deviceSN string, staleDuration string) ([]*Task, error)
}

func (m *mockTaskQueue) Push(ctx context.Context, task *Task) error {
	if m.pushFn != nil {
		return m.pushFn(ctx, task)
	}
	return nil
}

func (m *mockTaskQueue) Pop(ctx context.Context, deviceSN string) (*Task, error) {
	if m.popFn != nil {
		return m.popFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockTaskQueue) GetByID(ctx context.Context, taskID string) (*Task, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, taskID)
	}
	return nil, nil
}

func (m *mockTaskQueue) GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error) {
	if m.getByCWMPIDFn != nil {
		return m.getByCWMPIDFn(ctx, cwmpID)
	}
	return nil, nil
}

func (m *mockTaskQueue) Update(ctx context.Context, task *Task) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskQueue) Delete(ctx context.Context, taskID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, taskID)
	}
	return nil
}

func (m *mockTaskQueue) Len(ctx context.Context, deviceSN string) (int64, error) {
	if m.lenFn != nil {
		return m.lenFn(ctx, deviceSN)
	}
	return 0, nil
}

func (m *mockTaskQueue) MarkTaskSent(ctx context.Context, taskID, cwmpID string) error {
	if m.markTaskSentFn != nil {
		return m.markTaskSentFn(ctx, taskID, cwmpID)
	}
	return nil
}

func (m *mockTaskQueue) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	if m.markTaskCompletedFn != nil {
		return m.markTaskCompletedFn(ctx, taskID, result)
	}
	return nil
}

func (m *mockTaskQueue) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	if m.markTaskFailedFn != nil {
		return m.markTaskFailedFn(ctx, taskID, errorCode, errorMsg)
	}
	return nil
}

func (m *mockTaskQueue) GetStaleSentTasks(ctx context.Context, deviceSN string, staleDuration string) ([]*Task, error) {
	if m.getStaleSentTasksFn != nil {
		return m.getStaleSentTasksFn(ctx, deviceSN, staleDuration)
	}
	return nil, nil
}

// mockTaskRepository implements the methods of PgTaskRepository used by TaskService.
type mockTaskRepository struct {
	createFn          func(ctx context.Context, task *Task) error
	updateFn          func(ctx context.Context, task *Task) error
	getByIDFn         func(ctx context.Context, id string) (*Task, error)
	getByCWMPIDFn     func(ctx context.Context, cwmpID string) (*Task, error)
	deleteFn          func(ctx context.Context, id string) error
	batchCreateFn     func(ctx context.Context, tasks []*Task) error
	getHistoryFn      func(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) ([]*Task, int64, error)
	getPendingByDevFn func(ctx context.Context, deviceSN string) ([]*Task, error)
	listSentByDevFn   func(ctx context.Context, deviceSN string, sentBefore time.Time, limit int) ([]*Task, error)
	countByStatusFn   func(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error)
	purgeOldTasksFn   func(ctx context.Context, before string) (int64, error)
}

func (m *mockTaskRepository) Create(ctx context.Context, task *Task) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepository) Update(ctx context.Context, task *Task) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepository) GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error) {
	if m.getByCWMPIDFn != nil {
		return m.getByCWMPIDFn(ctx, cwmpID)
	}
	return nil, nil
}

func (m *mockTaskRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTaskRepository) BatchCreate(ctx context.Context, tasks []*Task) error {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, tasks)
	}
	return nil
}

func (m *mockTaskRepository) GetHistory(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) ([]*Task, int64, error) {
	if m.getHistoryFn != nil {
		return m.getHistoryFn(ctx, deviceSN, opts)
	}
	return nil, 0, nil
}

func (m *mockTaskRepository) GetPendingByDevice(ctx context.Context, deviceSN string) ([]*Task, error) {
	if m.getPendingByDevFn != nil {
		return m.getPendingByDevFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockTaskRepository) ListSentByDeviceBefore(ctx context.Context, deviceSN string, sentBefore time.Time, limit int) ([]*Task, error) {
	if m.listSentByDevFn != nil {
		return m.listSentByDevFn(ctx, deviceSN, sentBefore, limit)
	}
	return nil, nil
}

func (m *mockTaskRepository) CountByStatus(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockTaskRepository) PurgeOldTasks(ctx context.Context, before string) (int64, error) {
	if m.purgeOldTasksFn != nil {
		return m.purgeOldTasksFn(ctx, before)
	}
	return 0, nil
}

// --- testableTaskService wraps TaskService for testing with mocks ---

type testableTaskService struct {
	queue *mockTaskQueue
	repo  *mockTaskRepository
	svc   *TaskService
}

func newTestableService() *testableTaskService {
	q := &mockTaskQueue{}
	r := &mockTaskRepository{}
	logger := zap.NewNop()

	// We need to create a TaskService that uses our mocks.
	// Since TaskService uses concrete types *RedisTaskQueue and *PgTaskRepository,
	// we create a testableTaskService that delegates to the mocks.
	// We'll test through the testable wrapper methods that mirror the service logic.
	return &testableTaskService{
		queue: q,
		repo:  r,
		svc: &TaskService{
			logger: logger,
		},
	}
}

// The following methods mirror TaskService methods but use mock queue/repo.
// This is necessary because TaskService uses concrete types.

func (ts *testableTaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	// T-0157 C1: 同步真实 TaskService.CreateTask 的默认超时兜底行为。
	if req.ExpiresIn == 0 && ts.svc.defaultExpiresIn > 0 {
		req.ExpiresIn = ts.svc.defaultExpiresIn
	}
	task := NewTask(req)
	if err := ts.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task: %w", err)
	}
	if req.FailImmediately {
		ts.svc.notifyCompletion(ctx, task)
		return task, nil
	}
	if err := ts.queue.Push(ctx, task); err != nil {
		_ = deleteCreatedTaskAfterEnqueueFailure(ctx, ts.repo, task.ID)
		return nil, fmt.Errorf("enqueue task: %w", err)
	}
	return task, nil
}

func (ts *testableTaskService) GetTask(ctx context.Context, taskID string) (*Task, error) {
	task, err := ts.queue.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task != nil {
		return task, nil
	}
	return ts.repo.GetByID(ctx, taskID)
}

func TestResolveTaskDetails_TerminalTombstoneUsesDurableTask(t *testing.T) {
	ctx := context.Background()
	tombstone := &Task{ID: "task-terminal", Status: TaskStatusCompleted}
	durable := &Task{
		ID:     tombstone.ID,
		Status: TaskStatusCompleted,
		Params: json.RawMessage(`{"names":["Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth"]}`),
		Result: json.RawMessage(`{"values":[{"name":"DLBandwidth","value":"n100"}]}`),
	}

	got, err := resolveTaskDetails(ctx, tombstone, func(_ context.Context, id string) (*Task, error) {
		require.Equal(t, tombstone.ID, id)
		return durable, nil
	})
	require.NoError(t, err)
	require.Same(t, durable, got)
}

func TestResolveTaskDetails_NonTerminalTaskStaysOnRedis(t *testing.T) {
	pending := &Task{ID: "task-pending", Status: TaskStatusPending}
	got, err := resolveTaskDetails(context.Background(), pending,
		func(context.Context, string) (*Task, error) {
			t.Fatal("non-terminal task must not query PostgreSQL")
			return nil, nil
		})
	require.NoError(t, err)
	require.Same(t, pending, got)
}

func TestResolveTaskDetails_MissingDurableRowKeepsTerminalFence(t *testing.T) {
	tombstone := &Task{ID: "task-missing", Status: TaskStatusFailed}
	got, err := resolveTaskDetails(context.Background(), tombstone,
		func(context.Context, string) (*Task, error) { return nil, nil })
	require.NoError(t, err)
	require.Same(t, tombstone, got)
}

func TestResolveTaskDetails_DurableLoadErrorPropagates(t *testing.T) {
	wantErr := fmt.Errorf("postgres unavailable")
	got, err := resolveTaskDetails(context.Background(),
		&Task{ID: "task-error", Status: TaskStatusExpired},
		func(context.Context, string) (*Task, error) { return nil, wantErr })
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
}

func (ts *testableTaskService) GetTaskByCWMPID(ctx context.Context, cwmpID string) (*Task, error) {
	task, err := ts.queue.GetByCWMPID(ctx, cwmpID)
	if err != nil {
		return nil, err
	}
	if task != nil {
		return task, nil
	}
	return ts.repo.GetByCWMPID(ctx, cwmpID)
}

func (ts *testableTaskService) MarkTaskSent(ctx context.Context, taskID, cwmpID string) error {
	if err := ts.queue.MarkTaskSent(ctx, taskID, cwmpID); err != nil {
		return err
	}
	task, err := ts.queue.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task != nil {
		ts.repo.Update(ctx, task)
	}
	return nil
}

func (ts *testableTaskService) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	if err := ts.queue.MarkTaskCompleted(ctx, taskID, result); err != nil {
		return err
	}
	task, err := ts.queue.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task != nil {
		ts.repo.Update(ctx, task)
	}
	return nil
}

func (ts *testableTaskService) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	if err := ts.queue.MarkTaskFailed(ctx, taskID, errorCode, errorMsg); err != nil {
		return err
	}
	task, err := ts.queue.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task != nil {
		ts.repo.Update(ctx, task)
	}
	return nil
}

func (ts *testableTaskService) CancelTask(ctx context.Context, taskID string) error {
	task, err := ts.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		// 与生产 TaskService.CancelTask 保持一致：返回哨兵错误，便于
		// errors.Is(err, ErrTaskNotFound) 判定 + handler 映射 404。
		return ErrTaskNotFound
	}
	if task.Status != TaskStatusPending {
		return fmt.Errorf("cannot cancel task with status: %s", task.Status)
	}
	now := time.Now()
	task.Status = TaskStatusCancelled
	task.CompletedAt = &now

	if err := ts.queue.Delete(ctx, taskID); err != nil {
		return err
	}
	return ts.repo.Update(ctx, task)
}

func (ts *testableTaskService) GetPendingTasks(ctx context.Context, deviceSN string, limit int) ([]*Task, error) {
	tasks, err := ts.repo.GetPendingByDevice(ctx, deviceSN)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(tasks) > limit {
		tasks = tasks[:limit]
	}
	return tasks, nil
}

func (ts *testableTaskService) GetTaskHistory(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) (*TaskListResponse, error) {
	tasks, total, err := ts.repo.GetHistory(ctx, deviceSN, opts)
	if err != nil {
		return nil, err
	}
	return &TaskListResponse{
		Tasks:    tasks,
		Total:    total,
		Page:     opts.Page,
		PageSize: opts.PageSize,
	}, nil
}

func (ts *testableTaskService) GetTaskStats(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error) {
	return ts.repo.CountByStatus(ctx, deviceSN)
}

func (ts *testableTaskService) RecoverPendingTasks(ctx context.Context, deviceSN string) error {
	staleTasks, err := ts.repo.ListSentByDeviceBefore(ctx, deviceSN, time.Now(), recoverSentTaskBatchSize)
	if err != nil {
		return err
	}
	for _, task := range staleTasks {
		if !task.CanRetry() {
			task.MarkFailed(0, "exceeded max retries")
			ts.queue.Update(ctx, task)
			ts.repo.Update(ctx, task)
			continue
		}
		task.ResetForRetry()
		if err := ts.queue.Update(ctx, task); err != nil {
			continue
		}
		ts.repo.Update(ctx, task)
	}
	return nil
}

func (ts *testableTaskService) BatchCreateTasks(ctx context.Context, reqs []*CreateTaskRequest) ([]*Task, error) {
	var tasks []*Task
	for _, req := range reqs {
		ts.svc.applyDefaultExpiresIn(req)
		tasks = append(tasks, NewTask(req))
	}
	if err := ts.repo.BatchCreate(ctx, tasks); err != nil {
		return nil, fmt.Errorf("batch persist tasks: %w", err)
	}
	var pushed []*Task
	for _, task := range tasks {
		if task.Status == TaskStatusFailed {
			ts.svc.notifyCompletion(ctx, task)
			continue
		}
		if err := ts.queue.Push(ctx, task); err != nil {
			continue
		}
		pushed = append(pushed, task)
	}
	return pushed, nil
}

func (ts *testableTaskService) PurgeOldTasks(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays).Format(time.RFC3339)
	return ts.repo.PurgeOldTasks(ctx, before)
}

type captureCompletionCallback struct {
	tasks []*Task
}

func (c *captureCompletionCallback) OnTaskCompleted(_ context.Context, task *Task) {
	c.tasks = append(c.tasks, task)
}

// --- Tests ---

func Test_CreateTask_Success(t *testing.T) {
	ts := newTestableService()
	var createdTask *Task
	ts.repo.createFn = func(ctx context.Context, task *Task) error {
		createdTask = task
		return nil
	}
	var pushedTask *Task
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		pushedTask = task
		return nil
	}

	ctx := context.Background()
	req := &CreateTaskRequest{
		DeviceSN: "SN001",
		Method:   "GetParameterValues",
		Params:   json.RawMessage(`{"names":["Device."]}`),
	}

	task, err := ts.CreateTask(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "SN001", task.DeviceSN)
	assert.Equal(t, "GetParameterValues", task.Method)
	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Equal(t, createdTask, pushedTask, "same task should be persisted and enqueued")
}

func Test_CreateTask_RepoFailure(t *testing.T) {
	ts := newTestableService()
	ts.repo.createFn = func(ctx context.Context, task *Task) error {
		return fmt.Errorf("db connection refused")
	}

	ctx := context.Background()
	req := &CreateTaskRequest{DeviceSN: "SN001", Method: "Reboot"}

	task, err := ts.CreateTask(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "persist task")
}

func Test_CreateTask_QueueFailure_DeletesDBRecord(t *testing.T) {
	ts := newTestableService()

	var deletedID string
	ts.repo.createFn = func(ctx context.Context, task *Task) error {
		return nil
	}
	ts.repo.deleteFn = func(ctx context.Context, id string) error {
		deletedID = id
		return nil
	}
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		return fmt.Errorf("redis unavailable")
	}

	ctx := context.Background()
	req := &CreateTaskRequest{DeviceSN: "SN001", Method: "Reboot"}

	task, err := ts.CreateTask(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "enqueue task")
	assert.NotEmpty(t, deletedID, "should attempt to delete the DB record on queue failure")
}

func Test_CreateTask_QueueFailureUsesIndependentRollbackContext(t *testing.T) {
	ts := newTestableService()
	rollbackContextUsable := false
	ts.repo.createFn = func(context.Context, *Task) error {
		return nil
	}
	ts.repo.deleteFn = func(ctx context.Context, _ string) error {
		rollbackContextUsable = ctx.Err() == nil
		return nil
	}
	ts.queue.pushFn = func(context.Context, *Task) error {
		return fmt.Errorf("redis push deadline exceeded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	task, err := ts.CreateTask(ctx, &CreateTaskRequest{DeviceSN: "SN001", Method: "Reboot"})

	require.ErrorContains(t, err, "enqueue task")
	require.Nil(t, task)
	require.True(t, rollbackContextUsable,
		"PG compensation must not inherit the failed Redis request cancellation")
}

func Test_GetTask_FoundInQueue(t *testing.T) {
	ts := newTestableService()

	expected := &Task{ID: "task-1", DeviceSN: "SN001", Method: "Reboot", Status: TaskStatusPending}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		if taskID == "task-1" {
			return expected, nil
		}
		return nil, nil
	}

	repoCalled := false
	ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		repoCalled = true
		return nil, nil
	}

	ctx := context.Background()
	task, err := ts.GetTask(ctx, "task-1")
	require.NoError(t, err)
	assert.Equal(t, expected, task)
	assert.False(t, repoCalled, "should not fall back to repo when found in queue")
}

func Test_GetTask_FallbackToRepo(t *testing.T) {
	ts := newTestableService()

	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return nil, nil // not found in queue
	}

	expected := &Task{ID: "task-1", DeviceSN: "SN001", Method: "Reboot", Status: TaskStatusCompleted}
	ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		if id == "task-1" {
			return expected, nil
		}
		return nil, nil
	}

	ctx := context.Background()
	task, err := ts.GetTask(ctx, "task-1")
	require.NoError(t, err)
	assert.Equal(t, expected, task)
}

func Test_GetTask_NotFound(t *testing.T) {
	ts := newTestableService()

	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return nil, nil
	}
	ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		return nil, nil
	}

	ctx := context.Background()
	task, err := ts.GetTask(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, task)
}

func Test_GetTaskByCWMPID_FoundInQueue(t *testing.T) {
	ts := newTestableService()

	expected := &Task{ID: "task-1", CWMPID: "cwmp-123"}
	ts.queue.getByCWMPIDFn = func(ctx context.Context, cwmpID string) (*Task, error) {
		if cwmpID == "cwmp-123" {
			return expected, nil
		}
		return nil, nil
	}

	ctx := context.Background()
	task, err := ts.GetTaskByCWMPID(ctx, "cwmp-123")
	require.NoError(t, err)
	assert.Equal(t, expected, task)
}

func Test_GetTaskByCWMPID_FallbackToRepo(t *testing.T) {
	ts := newTestableService()

	ts.queue.getByCWMPIDFn = func(ctx context.Context, cwmpID string) (*Task, error) {
		return nil, nil
	}

	expected := &Task{ID: "task-1", CWMPID: "cwmp-123"}
	ts.repo.getByCWMPIDFn = func(ctx context.Context, cwmpID string) (*Task, error) {
		return expected, nil
	}

	ctx := context.Background()
	task, err := ts.GetTaskByCWMPID(ctx, "cwmp-123")
	require.NoError(t, err)
	assert.Equal(t, expected, task)
}

func Test_MarkTaskSent_Success(t *testing.T) {
	ts := newTestableService()

	markSentCalled := false
	ts.queue.markTaskSentFn = func(ctx context.Context, taskID, cwmpID string) error {
		assert.Equal(t, "task-1", taskID)
		assert.Equal(t, "cwmp-abc", cwmpID)
		markSentCalled = true
		return nil
	}

	sentTask := &Task{ID: "task-1", Status: TaskStatusSent, CWMPID: "cwmp-abc"}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return sentTask, nil
	}

	repoUpdated := false
	ts.repo.updateFn = func(ctx context.Context, task *Task) error {
		repoUpdated = true
		assert.Equal(t, "task-1", task.ID)
		return nil
	}

	ctx := context.Background()
	err := ts.MarkTaskSent(ctx, "task-1", "cwmp-abc")
	require.NoError(t, err)
	assert.True(t, markSentCalled)
	assert.True(t, repoUpdated, "should sync to PostgreSQL")
}

func Test_MarkTaskCompleted_Success(t *testing.T) {
	ts := newTestableService()

	completedCalled := false
	ts.queue.markTaskCompletedFn = func(ctx context.Context, taskID string, result json.RawMessage) error {
		completedCalled = true
		return nil
	}

	completedTask := &Task{ID: "task-1", Status: TaskStatusCompleted}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return completedTask, nil
	}

	repoUpdated := false
	ts.repo.updateFn = func(ctx context.Context, task *Task) error {
		repoUpdated = true
		return nil
	}

	ctx := context.Background()
	err := ts.MarkTaskCompleted(ctx, "task-1", json.RawMessage(`{"ok":true}`))
	require.NoError(t, err)
	assert.True(t, completedCalled)
	assert.True(t, repoUpdated)
}

func Test_MarkTaskFailed_Success(t *testing.T) {
	ts := newTestableService()

	failedCalled := false
	ts.queue.markTaskFailedFn = func(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
		failedCalled = true
		assert.Equal(t, 9001, errorCode)
		assert.Equal(t, "device error", errorMsg)
		return nil
	}

	failedTask := &Task{ID: "task-1", Status: TaskStatusFailed}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return failedTask, nil
	}

	ts.repo.updateFn = func(ctx context.Context, task *Task) error {
		return nil
	}

	ctx := context.Background()
	err := ts.MarkTaskFailed(ctx, "task-1", 9001, "device error")
	require.NoError(t, err)
	assert.True(t, failedCalled)
}

func Test_CancelTask_Success(t *testing.T) {
	ts := newTestableService()

	pendingTask := &Task{ID: "task-1", DeviceSN: "SN001", Status: TaskStatusPending}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		if taskID == "task-1" {
			return pendingTask, nil
		}
		return nil, nil
	}

	deleteCalled := false
	ts.queue.deleteFn = func(ctx context.Context, taskID string) error {
		deleteCalled = true
		return nil
	}

	var updatedTask *Task
	ts.repo.updateFn = func(ctx context.Context, task *Task) error {
		updatedTask = task
		return nil
	}

	ctx := context.Background()
	err := ts.CancelTask(ctx, "task-1")
	require.NoError(t, err)
	assert.True(t, deleteCalled, "should delete from queue")
	require.NotNil(t, updatedTask)
	assert.Equal(t, TaskStatusCancelled, updatedTask.Status)
	assert.NotNil(t, updatedTask.CompletedAt)
}

func Test_CancelTask_NotPending(t *testing.T) {
	ts := newTestableService()

	sentTask := &Task{ID: "task-1", Status: TaskStatusSent}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return sentTask, nil
	}

	ctx := context.Background()
	err := ts.CancelTask(ctx, "task-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel task with status: sent")
}

func Test_CancelTask_NotFound(t *testing.T) {
	ts := newTestableService()

	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return nil, nil
	}
	ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		return nil, nil
	}

	ctx := context.Background()
	err := ts.CancelTask(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func Test_CancelTask_CompletedStatus(t *testing.T) {
	ts := newTestableService()

	completedTask := &Task{ID: "task-1", Status: TaskStatusCompleted}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return completedTask, nil
	}

	ctx := context.Background()
	err := ts.CancelTask(ctx, "task-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel task with status: completed")
}

func Test_CancelTask_FailedStatus(t *testing.T) {
	ts := newTestableService()

	failedTask := &Task{ID: "task-1", Status: TaskStatusFailed}
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return failedTask, nil
	}

	ctx := context.Background()
	err := ts.CancelTask(ctx, "task-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel task with status: failed")
}

func Test_GetPendingTasks_DelegatesToRepo(t *testing.T) {
	ts := newTestableService()

	expected := []*Task{
		{ID: "task-1", Status: TaskStatusPending, Priority: 1},
		{ID: "task-2", Status: TaskStatusPending, Priority: 5},
		{ID: "task-3", Status: TaskStatusPending, Priority: 10},
	}
	ts.repo.getPendingByDevFn = func(ctx context.Context, deviceSN string) ([]*Task, error) {
		assert.Equal(t, "SN001", deviceSN)
		return expected, nil
	}

	ctx := context.Background()
	tasks, err := ts.GetPendingTasks(ctx, "SN001", 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func Test_GetPendingTasks_WithLimit(t *testing.T) {
	ts := newTestableService()

	allTasks := []*Task{
		{ID: "task-1"}, {ID: "task-2"}, {ID: "task-3"}, {ID: "task-4"}, {ID: "task-5"},
	}
	ts.repo.getPendingByDevFn = func(ctx context.Context, deviceSN string) ([]*Task, error) {
		return allTasks, nil
	}

	ctx := context.Background()
	tasks, err := ts.GetPendingTasks(ctx, "SN001", 2)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "task-1", tasks[0].ID)
	assert.Equal(t, "task-2", tasks[1].ID)
}

func Test_GetTaskHistory_DelegatesToRepo(t *testing.T) {
	ts := newTestableService()

	expectedTasks := []*Task{
		{ID: "task-1", Status: TaskStatusCompleted},
		{ID: "task-2", Status: TaskStatusFailed},
	}
	ts.repo.getHistoryFn = func(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) ([]*Task, int64, error) {
		assert.Equal(t, "SN001", deviceSN)
		assert.Equal(t, 1, opts.Page)
		assert.Equal(t, 20, opts.PageSize)
		return expectedTasks, 2, nil
	}

	ctx := context.Background()
	opts := &TaskHistoryOptions{Page: 1, PageSize: 20}
	resp, err := ts.GetTaskHistory(ctx, "SN001", opts)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Tasks, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

func Test_GetTaskStats_DelegatesToRepo(t *testing.T) {
	ts := newTestableService()

	expected := map[TaskStatus]int64{
		TaskStatusPending:   5,
		TaskStatusSent:      2,
		TaskStatusCompleted: 10,
		TaskStatusFailed:    1,
	}
	ts.repo.countByStatusFn = func(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error) {
		assert.Equal(t, "SN001", deviceSN)
		return expected, nil
	}

	ctx := context.Background()
	stats, err := ts.GetTaskStats(ctx, "SN001")
	require.NoError(t, err)
	assert.Equal(t, expected, stats)
}

func Test_RecoverPendingTasks_ResetsStale(t *testing.T) {
	ts := newTestableService()

	sentAt := time.Now().Add(-10 * time.Minute)
	staleTask := &Task{
		ID:         "task-1",
		DeviceSN:   "SN001",
		Status:     TaskStatusSent,
		SentAt:     &sentAt,
		RetryCount: 0,
		MaxRetries: 3,
	}

	ts.repo.listSentByDevFn = func(ctx context.Context, deviceSN string, sentBefore time.Time, limit int) ([]*Task, error) {
		assert.Equal(t, "SN001", deviceSN)
		assert.Equal(t, recoverSentTaskBatchSize, limit)
		return []*Task{staleTask}, nil
	}

	updateCalled := false
	ts.queue.updateFn = func(ctx context.Context, task *Task) error {
		updateCalled = true
		assert.Equal(t, TaskStatusPending, task.Status)
		assert.Equal(t, 1, task.RetryCount)
		return nil
	}

	repoUpdated := false
	ts.repo.updateFn = func(ctx context.Context, task *Task) error {
		repoUpdated = true
		return nil
	}

	ctx := context.Background()
	err := ts.RecoverPendingTasks(ctx, "SN001")
	require.NoError(t, err)
	assert.True(t, updateCalled, "should update task in queue")
	assert.True(t, repoUpdated, "should sync to repo")
}

func Test_RecoverPendingTasks_MarkExhaustedAsFailed(t *testing.T) {
	ts := newTestableService()

	sentAt := time.Now().Add(-10 * time.Minute)
	exhaustedTask := &Task{
		ID:         "task-1",
		DeviceSN:   "SN001",
		Status:     TaskStatusSent,
		SentAt:     &sentAt,
		RetryCount: 3,
		MaxRetries: 3, // CanRetry() => false
	}

	ts.repo.listSentByDevFn = func(ctx context.Context, deviceSN string, sentBefore time.Time, limit int) ([]*Task, error) {
		return []*Task{exhaustedTask}, nil
	}

	queueUpdated := false
	ts.queue.updateFn = func(ctx context.Context, task *Task) error {
		queueUpdated = true
		assert.Equal(t, TaskStatusFailed, task.Status)
		assert.Contains(t, task.ErrorMessage, "exceeded max retries")
		return nil
	}
	ts.repo.updateFn = func(ctx context.Context, task *Task) error {
		assert.Equal(t, TaskStatusFailed, task.Status)
		return nil
	}

	pushCalled := false
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		pushCalled = true
		return nil
	}

	ctx := context.Background()
	err := ts.RecoverPendingTasks(ctx, "SN001")
	require.NoError(t, err)
	assert.True(t, queueUpdated, "should mark exhausted task as failed")
	assert.False(t, pushCalled, "should not re-push exhausted task")
}

func Test_BatchCreateTasks_Success(t *testing.T) {
	ts := newTestableService()

	batchCreated := false
	ts.repo.batchCreateFn = func(ctx context.Context, tasks []*Task) error {
		batchCreated = true
		assert.Len(t, tasks, 3)
		return nil
	}

	pushCount := 0
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		pushCount++
		return nil
	}

	ctx := context.Background()
	reqs := []*CreateTaskRequest{
		{DeviceSN: "SN001", Method: "GetParameterValues"},
		{DeviceSN: "SN001", Method: "SetParameterValues"},
		{DeviceSN: "SN001", Method: "Reboot"},
	}

	tasks, err := ts.BatchCreateTasks(ctx, reqs)
	require.NoError(t, err)
	assert.True(t, batchCreated)
	assert.Len(t, tasks, 3)
	assert.Equal(t, 3, pushCount)
}

func Test_BatchCreateTasks_FailImmediatelyPersistsTerminalTaskWithoutQueuePush(t *testing.T) {
	ts := newTestableService()
	var persisted []*Task
	ts.repo.batchCreateFn = func(ctx context.Context, tasks []*Task) error {
		persisted = append(persisted, tasks...)
		return nil
	}

	pushCount := 0
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		pushCount++
		return nil
	}

	callback := &captureCompletionCallback{}
	ts.svc.AddCompletionCallback(callback)

	ctx := context.Background()
	reqs := []*CreateTaskRequest{
		{
			DeviceSN:        "SN-OFF",
			Method:          "Reboot",
			FailImmediately: true,
			FailReason:      "device offline",
			SourceID:        "mml-task-1",
			Source:          TaskSourceMML,
		},
	}

	tasks, err := ts.BatchCreateTasks(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, persisted, 1)
	require.Len(t, tasks, 0, "immediate failures are persisted and completed, but not pushed to Redis")
	assert.Equal(t, TaskStatusFailed, persisted[0].Status)
	assert.Equal(t, "device offline", persisted[0].ErrorMessage)
	assert.NotNil(t, persisted[0].CompletedAt)
	assert.Equal(t, 0, pushCount)
	require.Len(t, callback.tasks, 1)
	assert.Equal(t, TaskStatusFailed, callback.tasks[0].Status)
}

func Test_BatchCreateTasks_MixedImmediateFailurePushesOnlyRunnableTasks(t *testing.T) {
	ts := newTestableService()
	var persisted []*Task
	ts.repo.batchCreateFn = func(ctx context.Context, tasks []*Task) error {
		persisted = append(persisted, tasks...)
		return nil
	}

	var pushed []*Task
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		pushed = append(pushed, task)
		return nil
	}

	callback := &captureCompletionCallback{}
	ts.svc.AddCompletionCallback(callback)

	ctx := context.Background()
	reqs := []*CreateTaskRequest{
		{
			DeviceSN: "SN-ON",
			Method:   "Reboot",
			SourceID: "mml-task-1",
			Source:   TaskSourceMML,
		},
		{
			DeviceSN:        "SN-OFF",
			Method:          "Reboot",
			FailImmediately: true,
			FailReason:      "device offline",
			SourceID:        "mml-task-1",
			Source:          TaskSourceMML,
		},
	}

	tasks, err := ts.BatchCreateTasks(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, persisted, 2)
	require.Len(t, tasks, 1, "only runnable tasks should be returned to fanout as queued work")
	require.Len(t, pushed, 1, "online task should still be queued")
	assert.Equal(t, "SN-ON", pushed[0].DeviceSN)
	assert.Equal(t, TaskStatusPending, pushed[0].Status)

	var offline *Task
	for _, persistedTask := range persisted {
		if persistedTask.DeviceSN == "SN-OFF" {
			offline = persistedTask
			break
		}
	}
	require.NotNil(t, offline)
	assert.Equal(t, TaskStatusFailed, offline.Status)
	assert.Equal(t, "device offline", offline.ErrorMessage)
	require.Len(t, callback.tasks, 1)
	assert.Equal(t, "SN-OFF", callback.tasks[0].DeviceSN)
	assert.Equal(t, TaskStatusFailed, callback.tasks[0].Status)
}

func Test_BatchCreateTasks_RepoFailure(t *testing.T) {
	ts := newTestableService()

	ts.repo.batchCreateFn = func(ctx context.Context, tasks []*Task) error {
		return fmt.Errorf("db error")
	}

	ctx := context.Background()
	reqs := []*CreateTaskRequest{
		{DeviceSN: "SN001", Method: "Reboot"},
	}

	tasks, err := ts.BatchCreateTasks(ctx, reqs)
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "batch persist tasks")
}

func Test_BatchCreateTasks_PartialQueueFailure(t *testing.T) {
	ts := newTestableService()

	ts.repo.batchCreateFn = func(ctx context.Context, tasks []*Task) error {
		return nil
	}

	callCount := 0
	ts.queue.pushFn = func(ctx context.Context, task *Task) error {
		callCount++
		if callCount == 2 {
			return fmt.Errorf("redis error")
		}
		return nil
	}

	ctx := context.Background()
	reqs := []*CreateTaskRequest{
		{DeviceSN: "SN001", Method: "GetParameterValues"},
		{DeviceSN: "SN001", Method: "SetParameterValues"},
		{DeviceSN: "SN001", Method: "Reboot"},
	}

	tasks, err := ts.BatchCreateTasks(ctx, reqs)
	require.NoError(t, err)
	// Second task fails to push, so only 2 returned
	assert.Len(t, tasks, 2)
}

func Test_PurgeOldTasks_DelegatesToRepo(t *testing.T) {
	ts := newTestableService()

	ts.repo.purgeOldTasksFn = func(ctx context.Context, before string) (int64, error) {
		// Verify the before time is approximately 30 days ago
		parsedTime, err := time.Parse(time.RFC3339, before)
		require.NoError(t, err)
		expectedTime := time.Now().AddDate(0, 0, -30)
		assert.WithinDuration(t, expectedTime, parsedTime, 5*time.Second)
		return 42, nil
	}

	ctx := context.Background()
	count, err := ts.PurgeOldTasks(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, int64(42), count)
}
