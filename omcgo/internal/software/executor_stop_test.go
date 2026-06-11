package software

import (
	"context"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// stopSubTaskRepo lets a test drive GetByID's returned status so we can exercise
// ExecuteOne's pre-dispatch DB re-check (#59 Problem 3).
type stopSubTaskRepo struct {
	svcMockSubTaskRepo
	mu        sync.Mutex
	getStatus UpgradeState
	getErr    error
}

func (r *stopSubTaskRepo) GetByID(_ context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getErr != nil {
		return nil, r.getErr
	}
	return &UpgradeSubTask{ID: id, Status: r.getStatus}, nil
}

// stopTaskRepo lets a test drive the parent task status returned by GetByID.
type stopTaskRepo struct {
	svcMockTaskRepo
	status TaskStatus
	err    error
}

func (r *stopTaskRepo) GetByID(_ context.Context, id uuid.UUID) (*UpgradeTask, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &UpgradeTask{ID: id, Status: r.status}, nil
}

func newStopExecutor(t *testing.T, subRepo SubTaskRepository, taskRepo TaskRepository, cmdQueue devtask.Enqueuer) *UpgradeExecutor {
	t.Helper()
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: id, SerialNumber: "SN-STOP-001", Status: model.DeviceActive, FirmwareVersion: "V1.0.0"}, nil
		},
	}
	return NewUpgradeExecutor(
		taskRepo, subRepo, deviceRepo, &svcMockFirmwareRepo{},
		cmdQueue, nil, redisClient, &svcMockEventBus{}, zap.NewNop(),
	)
}

func goodFirmware() *FirmwareVersion {
	return &FirmwareVersion{
		ID: uuid.New(), Version: "V2.0.0", FileName: "fw.bin", MinIOPath: "p/fw.bin",
		SHA256Val: "deadbeefdeadbeef", // passes integrity verification
	}
}

// TestExecuteOne_AbortsWhenSubTaskSuspended is the core Problem 3 failure-path:
// a sub-task whose DB status is already Suspended (operator hit Suspend after the
// goroutine launched) must NOT dispatch a Download command.
func TestExecuteOne_AbortsWhenSubTaskSuspended(t *testing.T) {
	subRepo := &stopSubTaskRepo{getStatus: UpgradeSuspended}
	taskRepo := &stopTaskRepo{status: TaskInProgress}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newStopExecutor(t, subRepo, taskRepo, cmdQueue)

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, goodFirmware(), false, "")

	assert.False(t, dispatched, "Download must NOT dispatch when sub-task already suspended in DB")
}

// TestExecuteOne_AbortsWhenParentTaskSuspended covers the task-level halt: the
// sub-task row may not be flipped yet, but the parent task is Suspended.
func TestExecuteOne_AbortsWhenParentTaskSuspended(t *testing.T) {
	subRepo := &stopSubTaskRepo{getStatus: UpgradePending}
	taskRepo := &stopTaskRepo{status: TaskSuspended}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newStopExecutor(t, subRepo, taskRepo, cmdQueue)

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, goodFirmware(), false, "")

	assert.False(t, dispatched, "Download must NOT dispatch when parent task suspended in DB")
}

// TestExecuteOne_AbortsWhenContextCanceled covers the cancelable-context path:
// SuspendUpgrade/abort cancels the execution ctx; an in-flight goroutine entering
// ExecuteOne with a canceled ctx must abort before dispatch.
func TestExecuteOne_AbortsWhenContextCanceled(t *testing.T) {
	subRepo := &stopSubTaskRepo{getStatus: UpgradePending}
	taskRepo := &stopTaskRepo{status: TaskInProgress}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newStopExecutor(t, subRepo, taskRepo, cmdQueue)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate急停 before the goroutine got to run

	exec.ExecuteOne(ctx, &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, goodFirmware(), false, "")

	assert.False(t, dispatched, "Download must NOT dispatch when execution context already canceled")
}

// TestExecuteOne_DispatchesWhenPendingAndRunning is the success-path companion:
// a pending sub-task under an in-progress task with a live ctx still dispatches.
// Guards against the pre-dispatch re-check over-blocking the happy path.
func TestExecuteOne_DispatchesWhenPendingAndRunning(t *testing.T) {
	subRepo := &stopSubTaskRepo{getStatus: UpgradePending}
	taskRepo := &stopTaskRepo{status: TaskInProgress}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			assert.Equal(t, "Download", req.Method)
			return devtask.NewTask(req), nil
		},
	}
	exec := newStopExecutor(t, subRepo, taskRepo, cmdQueue)

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, goodFirmware(), false, "")

	assert.True(t, dispatched, "happy path: pending sub-task under running task must still dispatch")
}

// TestExecuteOne_DispatchesWhenReCheckReadFails confirms fail-open on the guard:
// a transient DB read error during the re-check must not block a legitimate
// dispatch (authz/terminal correctness is enforced elsewhere; we don't drop a
// valid upgrade on a flaky read).
func TestExecuteOne_DispatchesWhenReCheckReadFails(t *testing.T) {
	subRepo := &stopSubTaskRepo{getErr: commonerrors.ErrNotFound}
	taskRepo := &stopTaskRepo{status: TaskInProgress}
	var dispatched bool
	cmdQueue := &svcMockCmdQueue{
		createFn: func(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
			dispatched = true
			return devtask.NewTask(req), nil
		},
	}
	exec := newStopExecutor(t, subRepo, taskRepo, cmdQueue)

	exec.ExecuteOne(context.Background(), &UpgradeSubTask{
		ID: uuid.New(), TaskID: uuid.New(), DeviceID: uuid.New(),
	}, goodFirmware(), false, "")

	assert.True(t, dispatched, "transient re-check read error must fail-open (still dispatch)")
}
