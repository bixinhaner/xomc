package backup

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (prefixed with exec to avoid collision with service_test.go)
// ---------------------------------------------------------------------------

type execTaskRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*BackupTask, error)
	updateFn  func(ctx context.Context, task *BackupTask) error
}

func (m *execTaskRepo) Create(_ context.Context, _ *BackupTask) error { return nil }
func (m *execTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *execTaskRepo) Update(ctx context.Context, task *BackupTask) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, task)
	}
	return nil
}
func (m *execTaskRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *execTaskRepo) List(_ context.Context, _ TaskFilter) (*model.ListResponse[BackupTask], error) {
	return nil, nil
}

type execDeviceRepo struct {
	getBySNFn func(ctx context.Context, sn string) (*model.Device, error)
}

func (m *execDeviceRepo) Create(_ context.Context, _ *model.Device) error { return nil }
func (m *execDeviceRepo) Update(_ context.Context, _ *model.Device) error { return nil }
func (m *execDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *execDeviceRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySNFn != nil {
		return m.getBySNFn(ctx, sn)
	}
	return nil, nil
}
func (m *execDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return nil, nil
}
func (m *execDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}
func (m *execDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *execDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}
func (m *execDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *execDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *execDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *execDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *execDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}

type execCmdQueue struct {
	pushed []struct {
		DeviceSN string
		Cmd      *cmdqueue.Command
	}
}

func (m *execCmdQueue) Push(_ context.Context, deviceSN string, cmd *cmdqueue.Command) error {
	m.pushed = append(m.pushed, struct {
		DeviceSN string
		Cmd      *cmdqueue.Command
	}{deviceSN, cmd})
	return nil
}
func (m *execCmdQueue) Pop(_ context.Context, _ string) (*cmdqueue.Command, error)  { return nil, nil }
func (m *execCmdQueue) Peek(_ context.Context, _ string) (*cmdqueue.Command, error) { return nil, nil }
func (m *execCmdQueue) Len(_ context.Context, _ string) (int64, error)              { return 0, nil }
func (m *execCmdQueue) Clear(_ context.Context, _ string) error                     { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func newTestExecutor(taskRepo *execTaskRepo, deviceRepo *execDeviceRepo, cmdQ *execCmdQueue) *BackupExecutor {
	return &BackupExecutor{
		taskRepo:   taskRepo,
		deviceRepo: deviceRepo,
		cmdQueue:   cmdQ,
		connReq:    nil,
		eventBus:   event.NewChannelEventBus(16, zap.NewNop()),
		logger:     zap.NewNop(),
	}
}

func TestHandleTask_PendingToRunning(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN001"},
	}

	var statusUpdates []TaskStatus
	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*BackupTask, error) {
			assert.Equal(t, taskID, id)
			return task, nil
		},
		updateFn: func(_ context.Context, t *BackupTask) error {
			statusUpdates = append(statusUpdates, t.Status)
			return nil
		},
	}

	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn}, nil
		},
	}

	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{
		ID:        uuid.New().String(),
		Subject:   event.SubjectBackupTaskCreated,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// First update: running, last update: completed
	require.GreaterOrEqual(t, len(statusUpdates), 2)
	assert.Equal(t, TaskRunning, statusUpdates[0])
	assert.Equal(t, TaskCompleted, statusUpdates[len(statusUpdates)-1])
	assert.NotNil(t, task.StartedAt)
}

func TestHandleTask_PushUploadCommand(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN001", "SN002"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// Should have pushed 2 Upload commands
	require.Len(t, cmdQ.pushed, 2)
	assert.Equal(t, "SN001", cmdQ.pushed[0].DeviceSN)
	assert.Equal(t, "Upload", cmdQ.pushed[0].Cmd.Method)
	assert.Equal(t, "SN002", cmdQ.pushed[1].DeviceSN)
	assert.Equal(t, "Upload", cmdQ.pushed[1].Cmd.Method)

	// Verify file_type in params
	var params map[string]interface{}
	_ = json.Unmarshal(cmdQ.pushed[0].Cmd.Params, &params)
	assert.Equal(t, "2", params["file_type"])
}

func TestHandleTask_ProgressUpdate(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"SN001", "SN002", "SN003", "SN004"},
	}

	var progressValues []int
	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn: func(_ context.Context, t *BackupTask) error {
			progressValues = append(progressValues, t.Progress)
			return nil
		},
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn}, nil
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// Final progress should be 100
	assert.Equal(t, 100, progressValues[len(progressValues)-1])
}

func TestHandleTask_DeviceNotFound(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:        taskID,
		Status:    TaskPending,
		TargetIDs: []string{"MISSING_SN"},
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
		updateFn:  func(_ context.Context, _ *BackupTask) error { return nil },
	}
	deviceRepo := &execDeviceRepo{
		getBySNFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, assert.AnError
		},
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, deviceRepo, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	// No commands pushed
	assert.Len(t, cmdQ.pushed, 0)
	// Task should be failed
	assert.Equal(t, TaskFailed, task.Status)
	assert.NotNil(t, task.ErrorMessage)
}

func TestHandleTask_SkipsNonPending(t *testing.T) {
	taskID := uuid.New()
	task := &BackupTask{
		ID:     taskID,
		Status: TaskCompleted,
	}

	taskRepo := &execTaskRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return task, nil },
	}
	cmdQ := &execCmdQueue{}
	executor := newTestExecutor(taskRepo, &execDeviceRepo{}, cmdQ)

	payload, _ := json.Marshal(backupTaskPayload{TaskID: taskID.String()})
	evt := event.Event{ID: uuid.New().String(), Subject: event.SubjectBackupTaskCreated, Payload: payload, Timestamp: time.Now()}

	err := executor.handleTaskCreated(context.Background(), evt)
	require.NoError(t, err)

	assert.Len(t, cmdQ.pushed, 0)
}
