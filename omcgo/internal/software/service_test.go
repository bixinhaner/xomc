package software

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
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
func (m *svcMockFirmwareRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type svcMockUpgradeRepo struct {
	createFn            func(ctx context.Context, task *UpgradeTask) error
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status UpgradeState, msg string) error
	getActiveByDeviceFn func(ctx context.Context, deviceID uuid.UUID) (*UpgradeTask, error)
}

func (m *svcMockUpgradeRepo) Create(ctx context.Context, task *UpgradeTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	task.ID = uuid.New()
	return nil
}
func (m *svcMockUpgradeRepo) GetByID(_ context.Context, _ uuid.UUID) (*UpgradeTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockUpgradeRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, msg string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, msg)
	}
	return nil
}
func (m *svcMockUpgradeRepo) List(_ context.Context, _ UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	return model.NewListResponse([]UpgradeTask{}, 0, 1, 20), nil
}
func (m *svcMockUpgradeRepo) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeTask, error) {
	if m.getActiveByDeviceFn != nil {
		return m.getActiveByDeviceFn(ctx, deviceID)
	}
	return nil, commonerrors.ErrNotFound
}
func (m *svcMockUpgradeRepo) CountByBatchStatus(_ context.Context, _ uuid.UUID) (map[UpgradeState]int64, error) {
	return nil, nil
}

type svcMockDeviceRepo struct {
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
}

func (m *svcMockDeviceRepo) Create(_ context.Context, _ *model.Device) error  { return nil }
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
func (m *svcMockDeviceRepo) Update(_ context.Context, _ *model.Device) error  { return nil }
func (m *svcMockDeviceRepo) Delete(_ context.Context, _ uuid.UUID) error      { return nil }
func (m *svcMockDeviceRepo) List(_ context.Context, _ device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return &model.ListResponse[model.Device]{Items: []model.Device{}}, nil
}
func (m *svcMockDeviceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.DeviceStatus) error {
	return nil
}
func (m *svcMockDeviceRepo) UpdateLastInform(_ context.Context, _ string, _ time.Time, _ []string) error {
	return nil
}
func (m *svcMockDeviceRepo) CountByStatus(_ context.Context, _ *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return map[model.DeviceStatus]int64{}, nil
}
func (m *svcMockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}

type svcMockCmdQueue struct {
	pushFn func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error
}

func (m *svcMockCmdQueue) Push(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
	if m.pushFn != nil {
		return m.pushFn(ctx, deviceSN, cmd)
	}
	return nil
}
func (m *svcMockCmdQueue) Pop(_ context.Context, _ string) (*cmdqueue.Command, error) {
	return nil, nil
}
func (m *svcMockCmdQueue) Peek(_ context.Context, _ string) (*cmdqueue.Command, error) {
	return nil, nil
}
func (m *svcMockCmdQueue) Len(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *svcMockCmdQueue) Clear(_ context.Context, _ string) error        { return nil }

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

func TestService_StartUpgrade_Success(t *testing.T) {
	deviceID := uuid.New()
	firmwareID := uuid.New()

	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           id,
				SerialNumber: "SN-UPGRADE-001",
				// Empty ConnectionRequestURL to avoid nil connReq.Send call
			}, nil
		},
	}

	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{
				ID:        id,
				Version:   "V1.0.0",
				FileName:  "fw.bin",
				MinIOPath: "firmware/cmcc/SC/V1.0.0/fw.bin",
				FileSize:  1024,
			}, nil
		},
	}

	var pushedCmd *cmdqueue.Command
	cmdQueue := &svcMockCmdQueue{
		pushFn: func(_ context.Context, deviceSN string, cmd *cmdqueue.Command) error {
			assert.Equal(t, "SN-UPGRADE-001", deviceSN)
			pushedCmd = cmd
			return nil
		},
	}

	var publishedSubject string
	eventBus := &svcMockEventBus{
		publishFn: func(_ context.Context, subject string, _ event.Event) error {
			publishedSubject = subject
			return nil
		},
	}

	svc := NewSoftwareService(
		fwRepo,
		&svcMockUpgradeRepo{},
		deviceRepo,
		cmdQueue,
		nil, // connReq — not called when ConnectionRequestURL is empty
		nil, // minioClient — not used in StartUpgrade
		"test-bucket",
		eventBus,
		zap.NewNop(),
	)

	task, err := svc.StartUpgrade(context.Background(), deviceID, firmwareID)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, UpgradeDownloading, task.Status)
	assert.Equal(t, deviceID, task.DeviceID)
	assert.Equal(t, firmwareID, task.FirmwareID)
	assert.NotNil(t, pushedCmd)
	assert.Equal(t, "Download", pushedCmd.Method)
	assert.Equal(t, event.SubjectUpgradeStarted, publishedSubject)
}

func TestService_StartUpgrade_DeviceNotFound(t *testing.T) {
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{},
		&svcMockUpgradeRepo{},
		&svcMockDeviceRepo{}, // default returns ErrNotFound
		&svcMockCmdQueue{},
		nil, nil, "test-bucket",
		&svcMockEventBus{},
		zap.NewNop(),
	)

	_, err := svc.StartUpgrade(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get device")
}

func TestService_StartUpgrade_ActiveUpgradeExists(t *testing.T) {
	deviceID := uuid.New()
	fwID := uuid.New()

	deviceRepo := &svcMockDeviceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-001"}, nil
		},
	}
	fwRepo := &svcMockFirmwareRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*FirmwareVersion, error) {
			return &FirmwareVersion{ID: fwID, Version: "V1.0"}, nil
		},
	}
	upgradeRepo := &svcMockUpgradeRepo{
		getActiveByDeviceFn: func(_ context.Context, _ uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{Status: UpgradeDownloading}, nil // active upgrade exists
		},
	}

	svc := NewSoftwareService(
		fwRepo, upgradeRepo, deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, zap.NewNop(),
	)

	_, err := svc.StartUpgrade(context.Background(), deviceID, fwID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "active upgrade")
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
	upgradeRepo := &svcMockUpgradeRepo{
		getActiveByDeviceFn: func(_ context.Context, _ uuid.UUID) (*UpgradeTask, error) {
			return &UpgradeTask{
				ID:       taskID,
				DeviceID: deviceID,
				Status:   UpgradeDownloading,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status UpgradeState, _ string) error {
			updatedStatus = status
			return nil
		},
	}

	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, upgradeRepo, deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, zap.NewNop(),
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
		&svcMockUpgradeRepo{}, // GetActiveByDeviceID returns ErrNotFound
		deviceRepo,
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, zap.NewNop(),
	)

	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, map[string]string{
		"device_sn": "SN-001",
	})

	err := svc.HandleTransferComplete(context.Background(), evt)
	assert.NoError(t, err) // should silently return nil
}

func TestService_HandleTransferComplete_InvalidPayload(t *testing.T) {
	svc := NewSoftwareService(
		&svcMockFirmwareRepo{}, &svcMockUpgradeRepo{}, &svcMockDeviceRepo{},
		&svcMockCmdQueue{}, nil, nil, "test-bucket",
		&svcMockEventBus{}, zap.NewNop(),
	)

	// Empty device_sn
	evt := event.Event{
		Payload: json.RawMessage(`{"device_sn":""}`),
	}

	err := svc.HandleTransferComplete(context.Background(), evt)
	assert.NoError(t, err) // should silently return nil
}
