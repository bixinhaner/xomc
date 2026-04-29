package backup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// --- Mock Repositories ---

type mockTaskRepo struct {
	createFn  func(ctx context.Context, task *BackupTask) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*BackupTask, error)
	updateFn  func(ctx context.Context, task *BackupTask) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error)
}

func (m *mockTaskRepo) Create(ctx context.Context, task *BackupTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepo) Update(ctx context.Context, task *BackupTask) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTaskRepo) CleanupOldRows(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
	return nil, 0, nil
}
func (m *mockTaskRepo) UpdateFilePath(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockTaskRepo) FindByIDPrefix(_ context.Context, _ string, _ int) ([]*BackupTask, error) {
	return nil, nil
}

type mockScheduleRepo struct {
	createFn  func(ctx context.Context, schedule *BackupSchedule) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*BackupSchedule, error)
	updateFn  func(ctx context.Context, schedule *BackupSchedule) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter ScheduleFilter) (*model.ListResponse[BackupSchedule], error)
}

func (m *mockScheduleRepo) Create(ctx context.Context, schedule *BackupSchedule) error {
	if m.createFn != nil {
		return m.createFn(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleRepo) GetByID(ctx context.Context, id uuid.UUID) (*BackupSchedule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockScheduleRepo) Update(ctx context.Context, schedule *BackupSchedule) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockScheduleRepo) List(ctx context.Context, filter ScheduleFilter) (*model.ListResponse[BackupSchedule], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

// --- Helper ---

func newTestService(taskRepo *mockTaskRepo, scheduleRepo *mockScheduleRepo) *Service {
	return NewService(taskRepo, scheduleRepo, nil, zap.NewNop())
}

// --- Tests: CreateTask ---

func TestService_CreateTask(t *testing.T) {
	var captured *BackupTask
	taskRepo := &mockTaskRepo{
		createFn: func(ctx context.Context, task *BackupTask) error {
			captured = task
			task.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})

	task := &BackupTask{
		TaskType:   TaskFull,
		TargetType: "device",
		TargetIDs:  nil, // should default to []
	}

	result, err := svc.CreateTask(context.Background(), task)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, TaskPending, captured.Status)
	assert.Equal(t, 0, captured.Progress)
	assert.NotNil(t, captured.TargetIDs)
	assert.Empty(t, captured.TargetIDs, "nil TargetIDs should default to empty slice")
}

func TestService_CreateTask_WithTargetIDs(t *testing.T) {
	taskRepo := &mockTaskRepo{
		createFn: func(ctx context.Context, task *BackupTask) error {
			task.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})

	ids := []string{"device-1", "device-2"}
	task := &BackupTask{
		TaskType:   TaskIncremental,
		TargetType: "device",
		TargetIDs:  ids,
	}

	result, err := svc.CreateTask(context.Background(), task)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ids, result.TargetIDs, "non-nil TargetIDs should be preserved")
}

// --- Tests: GetTask ---

func TestService_GetTask(t *testing.T) {
	taskID := uuid.New()
	expected := &BackupTask{
		ID:       taskID,
		TaskType: TaskFull,
		Status:   TaskRunning,
	}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
			assert.Equal(t, taskID, id)
			return expected, nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})
	result, err := svc.GetTask(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: CancelTask ---

func TestService_CancelTask_FromPending(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:     taskID,
		Status: TaskPending,
	}

	var updatedTask *BackupTask
	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
			return task, nil
		},
		updateFn: func(ctx context.Context, t *BackupTask) error {
			updatedTask = t
			return nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})
	err := svc.CancelTask(context.Background(), taskID)

	require.NoError(t, err)
	require.NotNil(t, updatedTask)
	assert.Equal(t, TaskCancelled, updatedTask.Status)
}

func TestService_CancelTask_FromCompleted(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:     taskID,
		Status: TaskCompleted,
	}

	taskRepo := &mockTaskRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
			return task, nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})
	err := svc.CancelTask(context.Background(), taskID)

	require.Error(t, err)
	var bErr *commonerrors.BusinessError
	assert.True(t, errors.As(err, &bErr))
	assert.Equal(t, 8100, bErr.Code)
}

// --- Tests: DeleteTask ---

func TestService_DeleteTask(t *testing.T) {
	taskID := uuid.New()
	var deletedID uuid.UUID

	taskRepo := &mockTaskRepo{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})
	err := svc.DeleteTask(context.Background(), taskID)

	require.NoError(t, err)
	assert.Equal(t, taskID, deletedID)
}

// --- Tests: ListTasks ---

func TestService_ListTasks(t *testing.T) {
	expected := &model.ListResponse[BackupTask]{
		Items:      []BackupTask{{ID: uuid.New(), Status: TaskPending}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	filter := TaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	}

	taskRepo := &mockTaskRepo{
		listFn: func(ctx context.Context, f TaskFilter) (*model.ListResponse[BackupTask], error) {
			assert.Equal(t, filter, f)
			return expected, nil
		},
	}

	svc := newTestService(taskRepo, &mockScheduleRepo{})
	result, err := svc.ListTasks(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: CreateSchedule ---

func TestService_CreateSchedule(t *testing.T) {
	var captured *BackupSchedule
	scheduleRepo := &mockScheduleRepo{
		createFn: func(ctx context.Context, schedule *BackupSchedule) error {
			captured = schedule
			schedule.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(&mockTaskRepo{}, scheduleRepo)

	schedule := &BackupSchedule{
		Name:      "Daily Backup",
		CronExpr:  "0 2 * * *",
		Enabled:   true,
		TaskType:  TaskFull,
		TargetIDs: nil, // should default to []
	}

	result, err := svc.CreateSchedule(context.Background(), schedule)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, captured.TargetIDs)
	assert.Empty(t, captured.TargetIDs, "nil TargetIDs should default to empty slice")
}

// --- Tests: UpdateSchedule ---

func TestService_UpdateSchedule(t *testing.T) {
	scheduleID := uuid.New()
	existing := &BackupSchedule{
		ID:        scheduleID,
		Name:      "Old Name",
		CronExpr:  "0 2 * * *",
		Enabled:   true,
		TaskType:  TaskFull,
		TargetIDs: []string{"device-1"},
	}

	var updatedSchedule *BackupSchedule
	scheduleRepo := &mockScheduleRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*BackupSchedule, error) {
			assert.Equal(t, scheduleID, id)
			return existing, nil
		},
		updateFn: func(ctx context.Context, s *BackupSchedule) error {
			updatedSchedule = s
			return nil
		},
	}

	svc := newTestService(&mockTaskRepo{}, scheduleRepo)

	update := &BackupSchedule{
		Name:       "New Name",
		CronExpr:   "0 3 * * *",
		Enabled:    false,
		TaskType:   TaskIncremental,
		TargetType: ptrString("group"),
		TargetIDs:  []string{"group-1", "group-2"},
	}

	result, err := svc.UpdateSchedule(context.Background(), scheduleID, update)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", updatedSchedule.Name)
	assert.Equal(t, "0 3 * * *", updatedSchedule.CronExpr)
	assert.Equal(t, false, updatedSchedule.Enabled)
	assert.Equal(t, TaskIncremental, updatedSchedule.TaskType)
	assert.Equal(t, []string{"group-1", "group-2"}, updatedSchedule.TargetIDs)
}

func TestService_UpdateSchedule_NilTargetIDsPreservesExisting(t *testing.T) {
	scheduleID := uuid.New()
	existing := &BackupSchedule{
		ID:        scheduleID,
		Name:      "Old Name",
		CronExpr:  "0 2 * * *",
		Enabled:   true,
		TaskType:  TaskFull,
		TargetIDs: []string{"device-1"},
	}

	var updatedSchedule *BackupSchedule
	scheduleRepo := &mockScheduleRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*BackupSchedule, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, s *BackupSchedule) error {
			updatedSchedule = s
			return nil
		},
	}

	svc := newTestService(&mockTaskRepo{}, scheduleRepo)

	update := &BackupSchedule{
		Name:      "Updated Name",
		CronExpr:  "0 4 * * *",
		Enabled:   true,
		TaskType:  TaskFull,
		TargetIDs: nil, // should preserve existing
	}

	result, err := svc.UpdateSchedule(context.Background(), scheduleID, update)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Updated Name", updatedSchedule.Name)
	assert.Equal(t, []string{"device-1"}, updatedSchedule.TargetIDs, "nil TargetIDs should preserve existing")
}

// --- Helper functions ---

func ptrString(s string) *string {
	return &s
}
