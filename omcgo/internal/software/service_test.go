package software

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	devtask "github.com/omcgo/omcgo/internal/task"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (svcMock prefix to avoid conflict with handler_test.go swH mocks)
// ---------------------------------------------------------------------------

type svcMockFirmwareRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error)
	createFn  func(ctx context.Context, fw *FirmwareVersion) error
}

func (m *svcMockFirmwareRepo) Create(ctx context.Context, fw *FirmwareVersion) error {
	if m.createFn != nil {
		return m.createFn(ctx, fw)
	}
	fw.ID = uuid.New()
	return nil
}
func (m *svcMockFirmwareRepo) GetByID(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockFirmwareRepo) List(_ context.Context, _ FirmwareFilter) (*model.ListResponse[FirmwareVersion], error) {
	return model.NewListResponse([]FirmwareVersion{}, 0, 1, 20), nil
}
func (m *svcMockFirmwareRepo) Update(_ context.Context, _ *FirmwareVersion) error { return nil }
func (m *svcMockFirmwareRepo) Delete(_ context.Context, _ uuid.UUID) error         { return nil }

type svcMockTaskRepo struct {
	createFn         func(ctx context.Context, task *UpgradeTask) error
	getByIDFn        func(ctx context.Context, id uuid.UUID) (*UpgradeTask, error)
	updateStatusFn   func(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error
	incrementCountsFn func(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error
}

func (m *svcMockTaskRepo) Create(ctx context.Context, task *UpgradeTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	task.ID = uuid.New()
	return nil
}
func (m *svcMockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockTaskRepo) Update(_ context.Context, _ *UpgradeTask) error { return nil }
func (m *svcMockTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, result)
	}
	return nil
}
func (m *svcMockTaskRepo) List(_ context.Context, _ UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	return model.NewListResponse([]UpgradeTask{}, 0, 1, 20), nil
}
func (m *svcMockTaskRepo) IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error {
	if m.incrementCountsFn != nil {
		return m.incrementCountsFn(ctx, taskID, successDelta, failDelta)
	}
	return nil
}

type svcMockSubTaskRepo struct {
	createFn            func(ctx context.Context, task *UpgradeSubTask) error
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status UpgradeState, msg string) error
	getActiveByDeviceFn func(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error)
	getByCommandKeyFn   func(ctx context.Context, commandKey string) (*UpgradeSubTask, error)
	batchCreateFn       func(ctx context.Context, tasks []*UpgradeSubTask) error
}

func (m *svcMockSubTaskRepo) Create(ctx context.Context, task *UpgradeSubTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	task.ID = uuid.New()
	return nil
}
func (m *svcMockSubTaskRepo) GetByID(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockSubTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, msg string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, msg)
	}
	return nil
}
func (m *svcMockSubTaskRepo) Update(_ context.Context, _ *UpgradeSubTask) error { return nil }
func (m *svcMockSubTaskRepo) List(_ context.Context, _ SubTaskFilter) (*model.ListResponse[UpgradeSubTask], error) {
	return model.NewListResponse([]UpgradeSubTask{}, 0, 1, 20), nil
}
func (m *svcMockSubTaskRepo) ListByTaskID(_ context.Context, _ uuid.UUID, _ SubTaskFilter) (*model.ListResponse[UpgradeSubTask], error) {
	return model.NewListResponse([]UpgradeSubTask{}, 0, 1, 20), nil
}
func (m *svcMockSubTaskRepo) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error) {
	if m.getActiveByDeviceFn != nil {
		return m.getActiveByDeviceFn(ctx, deviceID)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockSubTaskRepo) GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error) {
	if m.getByCommandKeyFn != nil {
		return m.getByCommandKeyFn(ctx, commandKey)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockSubTaskRepo) BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, tasks)
	}
	for _, t := range tasks {
		t.ID = uuid.New()
	}
	return nil
}
func (m *svcMockSubTaskRepo) FailStale(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

type svcMockDeviceRepo struct {
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
}

func (m *svcMockDeviceRepo) Create(_ context.Context, _ *model.Device) error { return nil }
func (m *svcMockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySerialNumberFn != nil {
		return m.getBySerialNumberFn(ctx, sn)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockDeviceRepo) Update(_ context.Context, _ *model.Device) error { return nil }
func (m *svcMockDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (m *svcMockDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return &model.ListResponse[model.Device]{Items: []model.Device{}}, nil
}
func (m *svcMockDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}
func (m *svcMockDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *svcMockDeviceRepo) RecordBoot(_ context.Context, _ string, _ time.Time) (int, error) {
	return 0, nil
}
func (m *svcMockDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return map[model.DeviceStatus]int64{}, nil
}
func (m *svcMockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) ListGeo(_ context.Context, _ device.GeoDeviceFilter) ([]device.GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *svcMockDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*device.GeoStats, error) {
	return &device.GeoStats{}, nil
}
func (m *svcMockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]device.GeoDevice, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *svcMockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *svcMockDeviceRepo) ListRecycleBin(_ context.Context, _ device.RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *svcMockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *svcMockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}

type svcMockCmdQueue struct {
	createFn func(ctx context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error)
}

func (m *svcMockCmdQueue) CreateTask(ctx context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return devtask.NewTask(req), nil
}

func (m *svcMockCmdQueue) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type svcMockEventBus struct {
	publishFn func(ctx context.Context, subject string, evt event.Event) error
}

func (m *svcMockEventBus) Publish(ctx context.Context, subject string, evt event.Event) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, subject, evt)
	}
	return nil
}
func (m *svcMockEventBus) Subscribe(_ string, _ event.EventHandler) (event.Subscription, error) {
	return &svcMockSub{}, nil
}
func (m *svcMockEventBus) QueueSubscribe(_ string, _ string, _ event.EventHandler) (event.Subscription, error) {
	return &svcMockSub{}, nil
}
func (m *svcMockEventBus) Close() error { return nil }

type svcMockSub struct{}

func (s *svcMockSub) Unsubscribe() error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestService_BatchUpgrade_Success(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()

	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           id,
				SerialNumber: "SN-UPGRADE-001",
			}, nil
		},
	}

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:           id,
				Version:      "V1.0.0",
				FileName:     "fw.bin",
				MinIOPath:    "firmware/cmcc/SC/V1.0.0/fw.bin",
				FileSize:     1024,
				ProductClass: "SmallCell-LTE",
			}, nil
		},
	}

	cmdQueue := &svcMockCmdQueue{}

	svc := NewSoftwareService(
		fwRepo,
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		deviceRepo,
		cmdQueue,
		nil, // connReq
		nil, // minioClient
		"test-bucket",
		&svcMockEventBus{},
		nil, // redis
		zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{deviceID},
		FirmwareID: firmwareID,
		TaskName:   "test-upgrade",
		TaskType:   TaskTypeUpgrade,
	}

	task, err := svc.BatchUpgrade(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, TaskInProgress, task.Status)
	assert.Equal(t, "test-upgrade", task.TaskName)
	assert.Equal(t, 1, task.TotalCount)
	assert.Equal(t, "SmallCell-LTE", task.ProductClass)
}

func TestService_BatchUpgrade_FirmwareNotFound(t *testing.T) {
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{},
		nil, // redis
		zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{uuid.New()},
		FirmwareID: uuid.New(),
		TaskName:   "test-upgrade",
	}

	_, err := svc.BatchUpgrade(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get firmware")
}

func TestService_HandleTransferComplete_Success(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()

	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			assert.Equal(t, "SN-TC-001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	var updatedStatus UpgradeState
	subTaskRepo := &svcMockSubTaskRepo{
		getActiveByDeviceFn: func(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
			return &UpgradeSubTask{
				ID:     uuid.New(),
				TaskID: taskID,
				Status: UpgradeDownloading,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, _ string) error {
			updatedStatus = status
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]string{
		"device_sn": "SN-TC-001",
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, UpgradeRebooting, updatedStatus) // downloading → rebooting
}

func TestService_HandleTransferComplete_NoActiveUpgrade(t *testing.T) {
	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return &model.Device{ID: uuid.New(), SerialNumber: "SN-001"}, nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		&svcMockSubTaskRepo{}, // GetActiveByDeviceID returns ErrNotFound
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]string{
		"device_sn": "SN-001",
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	assert.NoError(t, err) // should silently return nil
}

func TestService_HandleTransferComplete_InvalidPayload(t *testing.T) {
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, &svcMockTaskRepo{}, &svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	// Empty device_sn
	evt := event.Event{
		Payload: json.RawMessage(`{"device_sn":""}`),
	}

	err := svc.HandleTransferComplete(context.Background(), evt)
	assert.NoError(t, err) // should silently return nil
}

// ---------------------------------------------------------------------------
// Executor event handling tests
// ---------------------------------------------------------------------------

func TestService_HandleDownloadResponse_FaultCode(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()

	var capturedStatus UpgradeState
	var capturedMsg string
	incrementCalled := false

	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, commandKey string) (*UpgradeSubTask, error) {
			assert.Equal(t, subTaskID.String(), commandKey)
			return &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				Status:   UpgradeDownloading,
				DeviceSN: "SN-DL-001",
			}, nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status UpgradeState, msg string) error {
			capturedStatus = status
			capturedMsg = msg
			assert.Equal(t, subTaskID, id)
			return nil
		},
	}

	taskRepo := &svcMockTaskRepo{
		incrementCountsFn: func(_ context.Context, id uuid.UUID, successDelta, failDelta int) error {
			assert.Equal(t, taskID, id)
			assert.Equal(t, 0, successDelta)
			assert.Equal(t, 1, failDelta)
			incrementCalled = true
			return nil
		},
	}

	// Use miniredis so executor.releaseDeviceLock does not panic on nil redis
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		taskRepo,
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, redisClient, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectCommandDownloadResponse, map[string]interface{}{
		"device_sn":    "SN-DL-001",
		"command_key":  subTaskID.String(),
		"fault_code":   9001,
		"fault_string": "device busy",
	})
	require.NoError(t, err)

	err = svc.executor.HandleDownloadResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, UpgradeFailed, capturedStatus)
	assert.Contains(t, capturedMsg, "download fault 9001")
	assert.Contains(t, capturedMsg, "device busy")
	assert.True(t, incrementCalled, "IncrementCounts should have been called")
}

func TestService_HandleDownloadResponse_Success(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()

	updateCalled := false
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, commandKey string) (*UpgradeSubTask, error) {
			assert.Equal(t, subTaskID.String(), commandKey)
			return &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				Status:   UpgradeDownloading,
				DeviceSN: "SN-DL-002",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectCommandDownloadResponse, map[string]interface{}{
		"device_sn":    "SN-DL-002",
		"command_key":  subTaskID.String(),
		"fault_code":   0,
		"fault_string": "",
	})
	require.NoError(t, err)

	err = svc.executor.HandleDownloadResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, updateCalled, "updateStatus should NOT be called on success (waiting for TC)")
}

func TestService_HandleDownloadResponse_Idempotent_WrongStatus(t *testing.T) {
	taskID := uuid.New()
	subTaskID := uuid.New()

	updateCalled := false
	subTaskRepo := &svcMockSubTaskRepo{
		getByCommandKeyFn: func(_ context.Context, commandKey string) (*UpgradeSubTask, error) {
			return &UpgradeSubTask{
				ID:       subTaskID,
				TaskID:   taskID,
				Status:   UpgradeRebooting, // not "downloading"
				DeviceSN: "SN-DL-003",
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, err := event.NewEvent(event.SubjectCommandDownloadResponse, map[string]interface{}{
		"device_sn":   "SN-DL-003",
		"command_key": subTaskID.String(),
		"fault_code":  9001,
	})
	require.NoError(t, err)

	err = svc.executor.HandleDownloadResponse(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, updateCalled, "updateStatus should NOT be called for non-downloading sub-task")
}

func TestService_HandleTransferComplete_Idempotent(t *testing.T) {
	deviceID := uuid.New()
	taskID := uuid.New()

	updateCalled := false
	deviceRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			assert.Equal(t, "SN-IDEMPOTENT-001", sn)
			return &model.Device{ID: deviceID, SerialNumber: sn}, nil
		},
	}

	subTaskRepo := &svcMockSubTaskRepo{
		getActiveByDeviceFn: func(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
			return &UpgradeSubTask{
				ID:     uuid.New(),
				TaskID: taskID,
				Status: UpgradeCompleted, // not "downloading"
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, _ UpgradeState, _ string) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockTaskRepo{},
		subTaskRepo,
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]string{
		"device_sn": "SN-IDEMPOTENT-001",
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, updateCalled, "updateStatus should NOT be called for non-downloading sub-task")
}

// ---------------------------------------------------------------------------
// State transition tests
// ---------------------------------------------------------------------------

func TestService_BatchUpgrade_CreatesMainTask(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()

	var capturedTask *UpgradeTask
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:           id,
				Version:      "V2.0.0",
				FileName:     "fw-v2.bin",
				MinIOPath:    "firmware/cmcc/SC/V2.0.0/fw-v2.bin",
				FileSize:     2048,
				ProductClass: "SmallCell-NR",
				MD5Val:       "abc123",
			}, nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		taskRepo,
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{deviceID},
		FirmwareID: firmwareID,
		TaskName:   "batch-main-task-test",
		TaskType:   TaskTypePatch,
	}

	task, err := svc.BatchUpgrade(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedTask)

	assert.Equal(t, "batch-main-task-test", capturedTask.TaskName)
	assert.Equal(t, TaskTypePatch, capturedTask.TaskType)
	assert.Equal(t, 1, capturedTask.TotalCount)
	assert.Equal(t, "SmallCell-NR", capturedTask.ProductClass)
	assert.Equal(t, "fw-v2.bin", capturedTask.FileName)
	assert.Equal(t, "abc123", capturedTask.FileMD5)

	// After BatchUpgrade returns, status should be updated to in_progress
	assert.Equal(t, TaskInProgress, task.Status)
}

func TestService_BatchUpgrade_CreatesSubTasks(t *testing.T) {
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()
	deviceID3 := uuid.New()
	firmwareID := uuid.New()

	var capturedSubTasks []*UpgradeSubTask
	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:           id,
				Version:      "V3.0.0",
				FileName:     "fw-v3.bin",
				MinIOPath:    "firmware/cmcc/SC/V3.0.0/fw-v3.bin",
				FileSize:     4096,
				ProductClass: "SmallCell-LTE",
			}, nil
		},
	}

	subTaskRepo := &svcMockSubTaskRepo{
		batchCreateFn: func(_ context.Context, tasks []*UpgradeSubTask) error {
			capturedSubTasks = make([]*UpgradeSubTask, len(tasks))
			copy(capturedSubTasks, tasks)
			for _, t := range tasks {
				t.ID = uuid.New()
			}
			return nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		&svcMockTaskRepo{},
		subTaskRepo,
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	req := BatchUpgradeRequest{
		DeviceIDs:  []uuid.UUID{deviceID1, deviceID2, deviceID3},
		FirmwareID: firmwareID,
		TaskName:   "batch-subtask-test",
		TaskType:   TaskTypeUpgrade,
	}

	_, err := svc.BatchUpgrade(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedSubTasks)

	assert.Len(t, capturedSubTasks, 3)

	expectedDeviceIDs := map[uuid.UUID]bool{deviceID1: true, deviceID2: true, deviceID3: true}
	for _, st := range capturedSubTasks {
		assert.Equal(t, UpgradePending, st.Status, "each sub-task should be in pending state")
		assert.True(t, expectedDeviceIDs[st.DeviceID], "sub-task DeviceID should match one of the requested devices")
		assert.NotNil(t, st.FirmwareID, "sub-task should have FirmwareID set")
		assert.Equal(t, firmwareID, *st.FirmwareID)
		delete(expectedDeviceIDs, st.DeviceID)
	}
	assert.Empty(t, expectedDeviceIDs, "all device IDs should be accounted for")
}

// ---------------------------------------------------------------------------
// Model constants tests
// ---------------------------------------------------------------------------

func TestModelConstants_TaskType(t *testing.T) {
	assert.Equal(t, TaskType(1), TaskTypeUpgrade)
	assert.Equal(t, TaskType(2), TaskTypeRollback)
	assert.Equal(t, TaskType(4), TaskTypePatch)
	assert.Equal(t, TaskType(6), TaskTypeFPGA)
	assert.Equal(t, TaskType(8), TaskTypeReserved)
}

func TestModelConstants_FileType(t *testing.T) {
	assert.Equal(t, FileType(0), FileTypeIMG)
	assert.Equal(t, FileType(1), FileTypePATCH)
	assert.Equal(t, FileType(6), FileTypeFPGA)
}

func TestModelConstants_TaskStatus(t *testing.T) {
	assert.Equal(t, TaskStatus("pending"), TaskPending)
	assert.Equal(t, TaskStatus("in_progress"), TaskInProgress)
	assert.Equal(t, TaskStatus("suspended"), TaskSuspended)
	assert.Equal(t, TaskStatus("ended"), TaskEnded)
}

func TestModelConstants_TaskResult(t *testing.T) {
	assert.Equal(t, TaskResult("success"), TaskResultSuccess)
	assert.Equal(t, TaskResult("partial"), TaskResultPartial)
	assert.Equal(t, TaskResult("failed"), TaskResultFailed)
	assert.Equal(t, TaskResult("terminated"), TaskResultTerminated)
}

// ---------------------------------------------------------------------------
// Rollback tests
// ---------------------------------------------------------------------------

func TestService_RollbackDevices_CreatesTask(t *testing.T) {
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()

	var capturedTask *UpgradeTask
	taskRepo := &svcMockTaskRepo{
		createFn: func(_ context.Context, task *UpgradeTask) error {
			capturedTask = task
			task.ID = uuid.New()
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		taskRepo,
		&svcMockSubTaskRepo{},
		&svcMockDeviceRepo{},
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{}, nil, zap.NewNop(),
	)

	req := RollbackRequest{
		DeviceIDs:    []uuid.UUID{deviceID1, deviceID2},
		TaskName:     "rollback-test",
		CreateUser:   "admin",
	}

	task, err := svc.RollbackDevices(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedTask)

	assert.Equal(t, TaskTypeRollback, capturedTask.TaskType, "rollback task type should be 2")
	assert.Equal(t, "rollback-test", capturedTask.TaskName)
	assert.Equal(t, 2, capturedTask.TotalCount)
	assert.Equal(t, "admin", capturedTask.CreateUser)

	// After RollbackDevices returns, status should be updated to in_progress
	assert.Equal(t, TaskInProgress, task.Status)
}
