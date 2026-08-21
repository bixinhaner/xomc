package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: DeviceRepository
// ---------------------------------------------------------------------------

type mockDeviceRepo struct {
	createFn                   func(ctx context.Context, device *model.Device) error
	getByIDFn                  func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn        func(ctx context.Context, sn string) (*model.Device, error)
	getDeletedBySerialNumberFn func(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error)
	updateFn                   func(ctx context.Context, device *model.Device) error
	deleteFn                   func(ctx context.Context, id uuid.UUID) error
	listFn                     func(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	updateStatusFn             func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	updateLastInformFn         func(ctx context.Context, sn string, at time.Time, events []string) error
	recordBootFn               func(ctx context.Context, sn string, at time.Time) (int, error)
	countByStatusFn            func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
	restoreDevicesFn           func(ctx context.Context, ids []uuid.UUID) (*RestoreResult, error)
	batchDeleteFn              func(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error)
}

func (m *mockDeviceRepo) Create(ctx context.Context, device *model.Device) error {
	if m.createFn != nil {
		return m.createFn(ctx, device)
	}
	return nil
}

func (m *mockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySerialNumberFn != nil {
		return m.getBySerialNumberFn(ctx, sn)
	}
	return nil, nil
}

func (m *mockDeviceRepo) GetDeletedBySerialNumber(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error) {
	if m.getDeletedBySerialNumberFn != nil {
		return m.getDeletedBySerialNumberFn(ctx, sn, carrier)
	}
	return nil, nil
}

func (m *mockDeviceRepo) Update(ctx context.Context, device *model.Device) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, device)
	}
	return nil
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockDeviceRepo) BatchDelete(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error) {
	if m.batchDeleteFn != nil {
		return m.batchDeleteFn(ctx, ids, deletedBy)
	}
	return int64(len(ids)), nil
}

func (m *mockDeviceRepo) List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[model.Device]{Items: []model.Device{}}, nil
}

func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

// T-0162: 新接口方法，测试中默认 no-op
func (m *mockDeviceRepo) UpdateLifecycle(ctx context.Context, id uuid.UUID, lifecycle model.DeviceLifecycle) error {
	return nil
}

func (m *mockDeviceRepo) UpdateOnlineStatus(ctx context.Context, id uuid.UUID, isOnline bool) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.updateLastInformFn != nil {
		return m.updateLastInformFn(ctx, sn, at, events)
	}
	return nil
}

func (m *mockDeviceRepo) RecordBoot(ctx context.Context, sn string, at time.Time) (int, error) {
	if m.recordBootFn != nil {
		return m.recordBootFn(ctx, sn, at)
	}
	return 0, nil
}

func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, carrier)
	}
	return map[model.DeviceStatus]int64{}, nil
}

type mockDisconnectedAlarmCleaner struct {
	active  map[string]*model.Alarm
	cleared []*model.Alarm
	lookups int
}

func (m *mockDisconnectedAlarmCleaner) GetActiveByDeviceAndIdentifier(_ context.Context, deviceSN string, alarmIdentifier string) (*model.Alarm, error) {
	m.lookups++
	return m.active[deviceSN+":"+alarmIdentifier], nil
}

func (m *mockDisconnectedAlarmCleaner) ClearBySync(_ context.Context, alarm *model.Alarm) error {
	m.cleared = append(m.cleared, alarm)
	return nil
}

func (m *mockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return []model.Device{}, nil
}
func (m *mockDeviceRepo) ListGeo(_ context.Context, _ GeoDeviceFilter) ([]GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *mockDeviceRepo) GetGeoStats(_ context.Context, _ GeoStatsFilter) (*GeoStats, error) {
	return &GeoStats{}, nil
}
func (m *mockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []model.DeviceVisibilityGrant) ([]GeoDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *mockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

func (m *mockDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *mockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockDeviceRepo) ListRecycleBin(_ context.Context, _ RecycleBinFilter) (*model.ListResponse[DeviceWithInfo], error) {
	return model.NewListResponse([]DeviceWithInfo{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) RestoreDevices(ctx context.Context, ids []uuid.UUID) (*RestoreResult, error) {
	if m.restoreDevicesFn != nil {
		return m.restoreDevicesFn(ctx, ids)
	}
	return &RestoreResult{}, nil
}
func (m *mockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: DeviceParameterRepository
// ---------------------------------------------------------------------------

type mockParamRepo struct {
	batchUpsertFn func(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
	getByDeviceFn func(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
	getByPathFn   func(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
	deleteByDevFn func(ctx context.Context, deviceID uuid.UUID) error
}

type mockGroupAssigner struct {
	batchAddFn func(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error)
}

func (m *mockGroupAssigner) BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if m.batchAddFn != nil {
		return m.batchAddFn(ctx, groupID, deviceIDs)
	}
	return int64(len(deviceIDs)), nil
}

func (m *mockParamRepo) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	if m.batchUpsertFn != nil {
		return m.batchUpsertFn(ctx, deviceID, params)
	}
	return nil
}

func (m *mockParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	if m.getByDeviceFn != nil {
		return m.getByDeviceFn(ctx, deviceID)
	}
	return nil, nil
}

func (m *mockParamRepo) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	if m.getByPathFn != nil {
		return m.getByPathFn(ctx, deviceID, path)
	}
	return nil, nil
}

func (m *mockParamRepo) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error {
	if m.deleteByDevFn != nil {
		return m.deleteByDevFn(ctx, deviceID)
	}
	return nil
}

func (m *mockParamRepo) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}

func (m *mockParamRepo) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (m *mockParamRepo) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}

func (m *mockParamRepo) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (m *mockParamRepo) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (m *mockParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *mockParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

// ---------------------------------------------------------------------------
// Helper: build a DeviceService wired to the given mocks
// ---------------------------------------------------------------------------

func newTestDeviceService(deviceRepo *mockDeviceRepo, paramRepo *mockParamRepo) *DeviceService {
	return NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())
}

// sampleInform returns a minimal InformMessage useful for most tests.
func sampleInform(sn string) *tr069.InformMessage {
	return &tr069.InformMessage{
		DeviceId: tr069.DeviceId{
			Manufacturer: "TestVendor",
			OUI:          "AABBCC",
			ProductClass: "SmallCell",
			SerialNumber: sn,
		},
		Event: []tr069.EventStruct{
			{EventCode: tr069.EventBootstrap},
		},
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
			{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://192.168.1.1:7547"},
			{Name: "Device.DeviceInfo.ModelName", Value: "PicoCell-LTE"},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests: RegisterFromInform
// ---------------------------------------------------------------------------

func TestDeviceService_RegisterFromInform_NewDevice(t *testing.T) {
	var createdDevice *model.Device
	var upsertedParams []model.DeviceParameter

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil // device does not exist yet
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createdDevice = device
			return nil
		},
	}

	paramRepo := &mockParamRepo{
		batchUpsertFn: func(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
			upsertedParams = params
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, paramRepo)
	inform := sampleInform("SN001")

	registration, err := svc.RegisterFromInform(context.Background(), inform, model.CarrierCMCC)
	require.NoError(t, err)
	require.NotNil(t, registration)
	assert.True(t, registration.Created)
	device := registration.Device
	require.NotNil(t, device)

	// Verify created device fields
	assert.Equal(t, "SN001", device.SerialNumber)
	assert.Equal(t, "AABBCC", device.OUI)
	assert.Equal(t, "SmallCell", device.ProductClass)
	assert.Equal(t, "TestVendor", device.Manufacturer)
	assert.Equal(t, model.CarrierCMCC, device.Carrier)
	assert.Equal(t, model.DeviceActive, device.Status)
	assert.Equal(t, "1.0.0", device.FirmwareVersion)
	assert.Equal(t, "http://192.168.1.1:7547", device.ConnectionRequestURL)
	assert.Equal(t, "192.168.1.1", device.IPAddress)
	assert.NotNil(t, device.LastInformAt)
	assert.Equal(t, model.TechLTE, device.Technology)

	// Verify repo.Create was called
	assert.NotNil(t, createdDevice)

	// Verify parameters were stored
	assert.Len(t, upsertedParams, 3)
}

func TestDeviceService_RegisterFromInform_NewDevice_AssignsDefaultGroup(t *testing.T) {
	var createdID uuid.UUID
	var assignedGroup uuid.UUID
	var assignedDevices []uuid.UUID

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createdID = device.ID
			return nil
		},
	}
	paramRepo := &mockParamRepo{}

	svc := newTestDeviceService(deviceRepo, paramRepo)
	svc.SetGroupAssigner(&mockGroupAssigner{
		batchAddFn: func(_ context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
			assignedGroup = groupID
			assignedDevices = append([]uuid.UUID(nil), deviceIDs...)
			return int64(len(deviceIDs)), nil
		},
	})

	registration, err := svc.RegisterFromInform(context.Background(), sampleInform("SN-DEFAULT-GROUP"), model.CarrierCMCC)
	require.NoError(t, err)
	require.NotNil(t, registration)
	device := registration.Device
	require.NotNil(t, device)

	assert.Equal(t, createdID, device.ID)
	assert.Equal(t, uuid.MustParse(global.DefaultLevel2GroupID), assignedGroup)
	assert.Equal(t, []uuid.UUID{device.ID}, assignedDevices)
}

func TestDeviceService_RegisterFromInform_ExistingDevice(t *testing.T) {
	existingID := uuid.New()
	updateCalled := false

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:           existingID,
				SerialNumber: sn,
				Status:       model.DeviceOffline,
			}, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			updateCalled = true
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN001")

	registration, err := svc.RegisterFromInform(context.Background(), inform, model.CarrierCMCC)
	require.NoError(t, err)
	require.NotNil(t, registration)
	device := registration.Device
	require.NotNil(t, device)
	assert.False(t, registration.Created)

	// Should have delegated to UpdateFromInform, which calls repo.Update
	assert.True(t, updateCalled, "expected Update to be called for existing device")
	assert.Equal(t, existingID, device.ID)
}

func TestDeviceService_RegisterFromInformEvent_RedeliveryPreservesCreated(t *testing.T) {
	var stored *model.Device
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return stored, nil
		},
		createFn: func(_ context.Context, device *model.Device) error {
			stored = device
			return nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			stored = device
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN-EVENT-RETRY")

	first, err := svc.RegisterFromInformEvent(
		context.Background(),
		inform,
		model.CarrierCMCC,
		"bootstrap-event-1",
	)
	require.NoError(t, err)
	require.True(t, first.Created)

	redelivered, err := svc.RegisterFromInformEvent(
		context.Background(),
		inform,
		model.CarrierCMCC,
		"bootstrap-event-1",
	)
	require.NoError(t, err)
	assert.True(t, redelivered.Created)

	differentEvent, err := svc.RegisterFromInformEvent(
		context.Background(),
		inform,
		model.CarrierCMCC,
		"bootstrap-event-2",
	)
	require.NoError(t, err)
	assert.False(t, differentEvent.Created)
}

func TestDeviceService_RegisterFromInform_DeletedDeviceSkipped(t *testing.T) {
	deletedID := uuid.New()
	createCalled := false
	updateCalled := false

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil
		},
		getDeletedBySerialNumberFn: func(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error) {
			assert.Equal(t, model.CarrierCMCC, carrier)
			return &model.Device{
				ID:           deletedID,
				SerialNumber: sn,
				Status:       model.DeviceOffline,
			}, nil
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createCalled = true
			return nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			updateCalled = true
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN001")

	device, err := svc.RegisterFromInform(context.Background(), inform, model.CarrierCMCC)
	require.ErrorIs(t, err, commonerrors.ErrNotFound)
	require.Nil(t, device)

	assert.False(t, updateCalled, "expected Update not to be called for recycle-bin device")
	assert.False(t, createCalled, "expected Create not to be called for recycle-bin device")
}

// mockLicenseEnforcer 是 device.LicenseEnforcer 的测试 stub。
type mockLicenseEnforcer struct {
	capacityErr  error
	expiryErr    error
	capacityArg  int
	capacityType string
	expiryOp     string
}

func (m *mockLicenseEnforcer) EnforceCapacity(_ context.Context, deviceType string, additional int) error {
	m.capacityType = deviceType
	m.capacityArg = additional
	return m.capacityErr
}

func (m *mockLicenseEnforcer) EnforceExpiry(_ context.Context, operation string) error {
	m.expiryOp = operation
	return m.expiryErr
}

// fixedNETypeMatcher 固定返回某个网元类型的 ProductClassMatcher stub，
// 让 resolveNEType 在 per-type 容量校验测试中能成功解析出 ne_type。
type fixedNETypeMatcher struct{ neType string }

func (m *fixedNETypeMatcher) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	return &product.MatchResult{Product: &product.Product{AlarmNeType: m.neType}}, nil
}

func TestDeviceService_RegisterFromInform_LicenseCapacityExceeded(t *testing.T) {
	createCalled := false
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil // 全新设备
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createCalled = true
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetProductMatcher(&fixedNETypeMatcher{neType: "eNB"})
	enforcer := &mockLicenseEnforcer{capacityErr: commonerrors.ErrLicenseCapacityExceeded}
	svc.SetLicenseEnforcer(enforcer)

	_, err := svc.RegisterFromInform(context.Background(), sampleInform("SN-CAP"), model.CarrierCMCC)
	require.ErrorIs(t, err, commonerrors.ErrLicenseCapacityExceeded)
	assert.False(t, createCalled, "Create 不应在容量超限时被调用")
	assert.Equal(t, 1, enforcer.capacityArg)
	assert.Equal(t, "eNB", enforcer.capacityType)
}

func TestDeviceService_RegisterFromInform_UPSUsesUPSLicenseCapacity(t *testing.T) {
	createCalled := false
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createCalled = true
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetProductMatcher(&fixedNETypeMatcher{neType: "UPS"})
	enforcer := &mockLicenseEnforcer{}
	svc.SetLicenseEnforcer(enforcer)

	inform := sampleInform("SN-UPS-LIC")
	inform.DeviceId.ProductClass = "UPS_M3_BMU"
	inform.ParameterList = []tr069.ParameterValueStruct{
		{Name: "InternetGatewayDevice.DeviceInfo.SoftwareVersion", Value: "UPS-AP-V1.0.0"},
		{Name: "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress", Value: "192.168.2.10"},
	}

	registration, err := svc.RegisterFromInform(context.Background(), inform, model.CarrierCMCC)
	require.NoError(t, err)
	require.NotNil(t, registration)
	assert.True(t, createCalled)
	assert.Equal(t, "device.inform.register", enforcer.expiryOp)
	assert.Equal(t, 1, enforcer.capacityArg)
	assert.Equal(t, "UPS", enforcer.capacityType)
}

func TestDeviceService_RegisterFromInform_LicenseCapacity_ProductNotRegistered(t *testing.T) {
	// issue #316：productClass 未登记产品（resolveNEType 失败）→ 注册被拒。
	createCalled := false
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createCalled = true
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	// 不注入 productMatcher → resolveNEType 返回 ErrLicenseCapacityExceeded
	svc.SetLicenseEnforcer(&mockLicenseEnforcer{})

	_, err := svc.RegisterFromInform(context.Background(), sampleInform("SN-ORPHAN"), model.CarrierCMCC)
	require.ErrorIs(t, err, commonerrors.ErrLicenseCapacityExceeded)
	assert.False(t, createCalled, "Create 不应在产品未登记时被调用")
}

func TestDeviceService_RegisterFromInform_LicenseExpired(t *testing.T) {
	createCalled := false
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			createCalled = true
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetLicenseEnforcer(&mockLicenseEnforcer{expiryErr: commonerrors.ErrLicenseExpired})

	_, err := svc.RegisterFromInform(context.Background(), sampleInform("SN-EXP"), model.CarrierCMCC)
	require.ErrorIs(t, err, commonerrors.ErrLicenseExpired)
	assert.False(t, createCalled, "Create 不应在 license 过期时被调用")
}

func TestDeviceService_RegisterFromInform_LicenseOk_ExistingDeviceNotBlocked(t *testing.T) {
	// 已注册设备的 Inform 更新不应被 license enforcer 拦截。
	existing := &model.Device{ID: uuid.New(), SerialNumber: "SN-EXIST", Status: model.DeviceActive}
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return existing, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	enforcer := &mockLicenseEnforcer{
		capacityErr: commonerrors.ErrLicenseCapacityExceeded, // 即使容量满也不应拦已注册设备
	}
	svc.SetLicenseEnforcer(enforcer)

	registration, err := svc.RegisterFromInform(context.Background(), sampleInform("SN-EXIST"), model.CarrierCMCC)
	require.NoError(t, err)
	require.NotNil(t, registration)
	assert.Equal(t, 0, enforcer.capacityArg, "已注册设备不应触发容量检查")
}

// ---------------------------------------------------------------------------
// Tests: UpdateFromInform
// ---------------------------------------------------------------------------

func TestDeviceService_UpdateFromInform_Success(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				Technology:     model.TechLTE,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       false,
				Status:         model.DeviceOffline, // should auto-transition to active
				InformInterval: 300,
			}, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002")
	inform.ParameterList = append(inform.ParameterList, tr069.ParameterValueStruct{
		Name:  "Device.ManagementServer.UDPConnectionRequestAddress",
		Value: "10.0.0.5",
	})

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)

	// Fields updated from Inform
	assert.Equal(t, "AABBCC", updatedDevice.OUI)
	assert.Equal(t, "SmallCell", updatedDevice.ProductClass)
	assert.Equal(t, "TestVendor", updatedDevice.Manufacturer)
	assert.Equal(t, model.TechLTE, updatedDevice.Technology)
	assert.Equal(t, "1.0.0", updatedDevice.FirmwareVersion)
	assert.Equal(t, "http://192.168.1.1:7547", updatedDevice.ConnectionRequestURL)
	assert.Equal(t, "10.0.0.5", updatedDevice.IPAddress)
	assert.NotNil(t, updatedDevice.LastInformAt)

	// Auto-transition: offline -> active
	assert.Equal(t, model.DeviceActive, updatedDevice.Status)
	assert.True(t, updatedDevice.IsOnline)
}

func TestDeviceService_UpdateFromInform_FallsBackToConnectionRequestURLHost(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:                   deviceID,
				SerialNumber:         sn,
				Technology:           model.TechLTE,
				LifecycleState:       model.LifecycleCommissioned,
				IsOnline:             false,
				Status:               model.DeviceOffline,
				InformInterval:       300,
				ConnectionRequestURL: "http://172.19.3.81:7547/69F5E1319CDAF252",
			}, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002-FALLBACK")
	inform.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.19.3.81:7547/69F5E1319CDAF252"},
	}

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)

	assert.Equal(t, "http://172.19.3.81:7547/69F5E1319CDAF252", updatedDevice.ConnectionRequestURL)
	assert.Equal(t, "172.19.3.81", updatedDevice.IPAddress)
	assert.Empty(t, updatedDevice.UDPConnectionRequestAddress)
	assert.False(t, updatedDevice.NatDetected)
	assert.True(t, updatedDevice.IsOnline)
}

func TestDeviceService_UpdateFromInform_PreservesConnectionRequestSummaryWhenURLIsNotValid(t *testing.T) {
	for _, tt := range []struct {
		name   string
		params []tr069.ParameterValueStruct
	}{
		{name: "parameter missing", params: []tr069.ParameterValueStruct{{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"}}},
		{name: "parameter empty", params: []tr069.ParameterValueStruct{{Name: "Device.ManagementServer.ConnectionRequestURL", Value: ""}}},
		{name: "parameter invalid", params: []tr069.ParameterValueStruct{{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "://invalid"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var updatedDevice *model.Device
			deviceRepo := &mockDeviceRepo{
				getBySerialNumberFn: func(context.Context, string) (*model.Device, error) {
					return &model.Device{
						ID:                   uuid.New(),
						SerialNumber:         "BM-SN",
						Technology:           model.TechLTE,
						LifecycleState:       model.LifecycleCommissioned,
						Status:               model.DeviceActive,
						ConnectionRequestURL: "http://172.19.9.217:7547",
						IPAddress:            "172.19.9.217",
					}, nil
				},
				updateFn: func(_ context.Context, device *model.Device) error {
					updatedDevice = device
					return nil
				},
			}
			svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
			inform := sampleInform("BM-SN")
			inform.ParameterList = tt.params

			_, err := svc.UpdateFromInform(context.Background(), inform)
			require.NoError(t, err)
			require.NotNil(t, updatedDevice)
			assert.Equal(t, "http://172.19.9.217:7547", updatedDevice.ConnectionRequestURL)
			assert.Equal(t, "172.19.9.217", updatedDevice.IPAddress)
		})
	}
}

func TestDeviceService_UpdateFromInform_IgnoresUnspecifiedUDPAddress(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				Technology:     model.TechLTE,
				LifecycleState: model.LifecycleCommissioned,
				Status:         model.DeviceActive,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002-STUN")
	inform.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.17.1.14:7547/AB48F15B575B64B43D3F8830CFCF8277"},
		{Name: "Device.ManagementServer.UDPConnectionRequestAddress", Value: "0.0.0.0"},
		{Name: "Device.ManagementServer.STUNServerPort", Value: "3478"},
	}

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)

	assert.Empty(t, updatedDevice.UDPConnectionRequestAddress)
	assert.Equal(t, "172.17.1.14", updatedDevice.IPAddress)
	assert.False(t, updatedDevice.NatDetected)
	assert.True(t, updatedDevice.IsOnline)
}

func TestDeviceService_UpdateFromInform_DoesNotDeriveUDPAddressFromCachedSTUNPort(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				Technology:     model.TechLTE,
				LifecycleState: model.LifecycleCommissioned,
				Status:         model.DeviceActive,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002-STUN-CACHED")
	inform.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.17.1.14:7547/AB48F15B575B64B43D3F8830CFCF8277"},
		{Name: "Device.ManagementServer.UDPConnectionRequestAddress", Value: "0.0.0.0"},
	}

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)

	assert.Empty(t, updatedDevice.UDPConnectionRequestAddress)
	assert.Equal(t, "172.17.1.14", updatedDevice.IPAddress)
	assert.False(t, updatedDevice.NatDetected)
	assert.True(t, updatedDevice.IsOnline)
}

func TestDeviceService_UpdateFromInform_NotFound(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil // not found
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("UNKNOWN_SN")

	// Device not found returns (nil, nil) — caller decides whether to auto-register
	device, err := svc.UpdateFromInform(context.Background(), inform)
	assert.NoError(t, err)
	assert.Nil(t, device)
}

func TestDeviceService_UpdateFromInform_CorrectsTechnologyFromStrongPath(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				Technology:     model.TechLTE,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       false,
				Status:         model.DeviceOffline,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002-NR")
	inform.ParameterList = append(inform.ParameterList,
		tr069.ParameterValueStruct{Name: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.Common.gNBId", Value: "123"},
	)

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)
	assert.Equal(t, model.TechNR, updatedDevice.Technology)
}

func TestDeviceService_UpdateFromInform_DoesNotDowngradeWithoutStrongPath(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				Technology:     model.TechNR,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       false,
				Status:         model.DeviceOffline,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002-NR-KEEP")
	inform.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://192.168.1.1:7547"},
		{Name: "Device.DeviceInfo.ModelName", Value: "Legacy-LTE-Name"},
	}

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)
	assert.Equal(t, model.TechNR, updatedDevice.Technology)
}

func TestDeviceService_UpdateFromInform_ReconnectRecordsLastOnlineTimeWhenStatusAlreadyActive(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device
	recordCalls := 0

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				Technology:     model.TechLTE,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       false,              // 关键场景：离线后恢复
				Status:         model.DeviceActive, // 但状态本身已是 active
				InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	registry := carrier.NewRegistry()
	registry.Register(testCarrier{})
	svc.infoSyncer = &InfoSyncer{
		logger:          zap.NewNop(),
		paramRepo:       &mockParamRepo{},
		carrierRegistry: registry,
		infoRepo: stubDeviceInfoRepo{
			updateSyncFields: func(_ context.Context, gotID uuid.UUID, fields map[string]interface{}) error {
				recordCalls++
				assert.Equal(t, deviceID, gotID)
				_, hasLast := fields["last_online_time"]
				assert.True(t, hasLast)
				return nil
			},
		},
	}

	device, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-RECONNECT-001"))
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)

	assert.True(t, updatedDevice.IsOnline)
	assert.Equal(t, 1, recordCalls, "offline->online edge should refresh last_online_time exactly once")
}

// ---------------------------------------------------------------------------
// Tests: CreateDevice
// ---------------------------------------------------------------------------

func TestDeviceService_CreateDevice_Success(t *testing.T) {
	var created *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return nil, nil // no existing device
		},
		createFn: func(ctx context.Context, device *model.Device) error {
			created = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	req := CreateDeviceRequest{
		SerialNumber: "NEW-SN-100",
		OUI:          "112233",
		ProductClass: "Femto",
		Manufacturer: "Acme",
		ModelName:    "Femto-X1",
		Carrier:      model.CarrierCTCC,
		Technology:   model.TechNR,
		DeviceName:   "SiteA",
		SiteID:       "SITE-001",
		Latitude:     f64p(31.23),
		Longitude:    f64p(121.47),
	}

	device, err := svc.CreateDevice(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, device)

	assert.Equal(t, "NEW-SN-100", device.SerialNumber)
	assert.Equal(t, model.DeviceRegistered, device.Status, "new API-created device should be in registered status")
	assert.Equal(t, model.CarrierCTCC, device.Carrier)
	assert.Equal(t, model.TechNR, device.Technology)
	assert.Equal(t, "SiteA", device.DeviceName)
	assert.NotNil(t, created)
}

func TestDeviceService_CreateDevice_Duplicate(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(ctx context.Context, sn string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn}, nil // already exists
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	_, err := svc.CreateDevice(context.Background(), CreateDeviceRequest{
		SerialNumber: "DUPLICATE-SN",
		OUI:          "AABBCC",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists))
}

// ---------------------------------------------------------------------------
// Tests: UpdateDevice
// ---------------------------------------------------------------------------

func TestDeviceService_UpdateDevice_Success(t *testing.T) {
	deviceID := uuid.New()

	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:           deviceID,
				SerialNumber: "SN-UPD",
				DeviceName:   "OldSite",
				Latitude:     f64p(0.0),
			}, nil
		},
		updateFn: func(ctx context.Context, device *model.Device) error {
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	newSite := "NewSite"
	newLat := 39.9
	req := UpdateDeviceRequest{
		DeviceName: &newSite,
		Latitude:   &newLat,
	}

	device, err := svc.UpdateDevice(context.Background(), deviceID, req)
	require.NoError(t, err)
	require.NotNil(t, device)

	assert.Equal(t, "NewSite", device.DeviceName)
	if assert.NotNil(t, device.Latitude) {
		assert.InDelta(t, 39.9, *device.Latitude, 0.001)
	}
	// Unchanged field stays the same
	assert.Equal(t, "SN-UPD", device.SerialNumber)
}

func TestDeviceService_UpdateDevice_NotFound(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return nil, nil // not found
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	device, err := svc.UpdateDevice(context.Background(), uuid.New(), UpdateDeviceRequest{})
	assert.NoError(t, err)
	assert.Nil(t, device, "expected nil device when not found")
}

func TestDeviceService_UpdateDevice_ValidatesLocationSourceMode(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(context.Context, uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, SerialNumber: "SN-SOURCE"}, nil
		},
		updateFn: func(context.Context, *model.Device) error {
			t.Fatal("invalid source mode must not be persisted")
			return nil
		},
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	invalid := model.LocationSourceMode("auto")

	_, err := svc.UpdateDevice(context.Background(), deviceID, UpdateDeviceRequest{
		LocationSourceMode: &invalid,
	})

	require.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

// ---------------------------------------------------------------------------
// Tests: DeleteDevice
// ---------------------------------------------------------------------------

func TestDeviceService_DeleteDevice(t *testing.T) {
	batchDeleteCalled := false
	targetID := uuid.New()

	deviceRepo := &mockDeviceRepo{
		batchDeleteFn: func(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error) {
			batchDeleteCalled = true
			assert.Equal(t, []uuid.UUID{targetID}, ids)
			assert.Equal(t, "alice", deletedBy)
			return int64(len(ids)), nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	ctx := context.WithValue(context.Background(), admin.CtxKeyUsername, "alice")
	err := svc.DeleteDevice(ctx, targetID)
	require.NoError(t, err)
	assert.True(t, batchDeleteCalled)
}

func TestDeviceService_BatchDeleteDevices_FillsDeletedByFromContext(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}

	deviceRepo := &mockDeviceRepo{
		batchDeleteFn: func(ctx context.Context, gotIDs []uuid.UUID, deletedBy string) (int64, error) {
			assert.Equal(t, ids, gotIDs)
			assert.Equal(t, "bob", deletedBy)
			return int64(len(gotIDs)), nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	ctx := context.WithValue(context.Background(), admin.CtxKeyUsername, "bob")

	result := svc.BatchDeleteDevices(ctx, ids, "")

	assert.Equal(t, len(ids), result.Succeeded)
	assert.Equal(t, 0, result.Failed)
}

func TestDeviceService_BatchDeleteDevices_ExplicitDeletedByWins(t *testing.T) {
	ids := []uuid.UUID{uuid.New()}

	deviceRepo := &mockDeviceRepo{
		batchDeleteFn: func(ctx context.Context, gotIDs []uuid.UUID, deletedBy string) (int64, error) {
			assert.Equal(t, ids, gotIDs)
			assert.Equal(t, "system:auto_recycle", deletedBy)
			return int64(len(gotIDs)), nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	ctx := context.WithValue(context.Background(), admin.CtxKeyUsername, "bob")

	result := svc.BatchDeleteDevices(ctx, ids, "system:auto_recycle")

	assert.Equal(t, 1, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
}

type recordingDeviceGroupCountsInvalidator struct {
	calls int
}

func (r *recordingDeviceGroupCountsInvalidator) InvalidateDeviceGroupCounts() {
	r.calls++
}

func TestDeviceService_BatchDeleteDevices_InvalidatesDeviceGroupCounts(t *testing.T) {
	ids := []uuid.UUID{uuid.New()}
	deviceRepo := &mockDeviceRepo{
		batchDeleteFn: func(context.Context, []uuid.UUID, string) (int64, error) {
			return 1, nil
		},
	}
	invalidator := &recordingDeviceGroupCountsInvalidator{}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	svc.SetDeviceGroupCountsInvalidator(invalidator)

	result := svc.BatchDeleteDevices(context.Background(), ids, "alice")

	assert.Equal(t, 1, result.Succeeded)
	assert.Equal(t, 1, invalidator.calls)
}

// ---------------------------------------------------------------------------
// Tests: TransitionStatus
// ---------------------------------------------------------------------------

func TestDeviceService_TransitionStatus_Valid(t *testing.T) {
	deviceID := uuid.New()
	var newStatus model.DeviceStatus

	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:     deviceID,
				Status: model.DeviceActive,
			}, nil
		},
		updateStatusFn: func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
			newStatus = status
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	err := svc.TransitionStatus(context.Background(), deviceID, model.DeviceMaintenance)
	require.NoError(t, err)
	assert.Equal(t, model.DeviceMaintenance, newStatus)
}

func TestDeviceService_TransitionStatus_Invalid(t *testing.T) {
	deviceID := uuid.New()

	deviceRepo := &mockDeviceRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{
				ID:     deviceID,
				Status: model.DeviceDiscovered,
			}, nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	// discovered -> decommissioned is NOT a valid transition
	err := svc.TransitionStatus(context.Background(), deviceID, model.DeviceDecommissioned)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transition")
}

// ---------------------------------------------------------------------------
// Tests: detectTechnology (package-level helper)
// ---------------------------------------------------------------------------

func TestDetectTechnology(t *testing.T) {
	tests := []struct {
		name   string
		params []tr069.ParameterValueStruct
		want   model.Technology
	}{
		{
			name:   "empty params default to LTE",
			params: nil,
			want:   model.TechLTE,
		},
		{
			name: "LTE path wins without model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", Value: "123"},
			},
			want: model.TechLTE,
		},
		{
			name: "NR path with instance wins without model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.Common.gNBId", Value: "456"},
			},
			want: model.TechNR,
		},
		{
			name: "NR path on second instance wins",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.Common.NRARFCN", Value: "789"},
			},
			want: model.TechNR,
		},
		{
			name: "NR wins when LTE and NR paths coexist",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.Services.FAPService.1.FAPControl.LTE.OpState", Value: "1"},
				{Name: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.Common.gNBId", Value: "456"},
			},
			want: model.TechNR,
		},
		{
			name: "actual dengyo nr paths infer NR without model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.Antenna.NumOfRxAntenna", Value: "2"},
				{Name: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.rftxEnable", Value: "1"},
				{Name: "Device.Services.FAPService.1.FAPControl.NR.LgwFwdCfg.LgwEnable", Value: "0"},
			},
			want: model.TechNR,
		},
		{
			name: "LTE model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.ModelName", Value: "PicoCell-LTE-3000"},
			},
			want: model.TechLTE,
		},
		{
			name: "NR keyword in model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.ModelName", Value: "SmallCell-NR-SA"},
			},
			want: model.TechNR,
		},
		{
			name: "5G keyword in description",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.Description", Value: "Indoor 5G small cell"},
			},
			want: model.TechNR,
		},
		{
			name: "gNB keyword in model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.ModelName", Value: "gNB-DU-100"},
			},
			want: model.TechNR,
		},
		{
			name: "irrelevant param path ignored",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.SomeOther.Path", Value: "NR"},
			},
			want: model.TechLTE,
		},
		{
			name: "path detection beats misleading model name",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.ModelName", Value: "Legacy-LTE-Name"},
				{Name: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.Common.gNBId", Value: "999"},
			},
			want: model.TechNR,
		},
		{
			name: "firmware bnq alone does not affect legacy helper",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.SoftwareVersion", Value: "DENGYO_BNQ_2.6.12.47.5"},
			},
			want: model.TechLTE,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectTechnology(tc.params)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDetectTechnologyFromIdentity(t *testing.T) {
	tests := []struct {
		name            string
		productClass    string
		modelName       string
		firmwareVersion string
		want            model.Technology
		ok              bool
	}{
		{
			name:            "BSC product class maps to NR",
			productClass:    "FAP/BSC7079B243",
			firmwareVersion: "DENGYO_BNQ_2.6.12.47.5",
			want:            model.TechNR,
			ok:              true,
		},
		{
			name:      "BNQ model name maps to NR",
			modelName: "Dengyo-BNQ-Indoor",
			want:      model.TechNR,
			ok:        true,
		},
		{
			name:            "irrelevant identity returns false",
			productClass:    "SmallCell",
			modelName:       "Legacy-LTE-Name",
			firmwareVersion: "1.0.0",
			want:            "",
			ok:              false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := detectTechnologyFromIdentity(tc.productClass, tc.modelName, tc.firmwareVersion)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDeviceService_UpdateFromInform_CorrectsTechnologyFromProductIdentity(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device

	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				ProductClass:   "FAP/BSC7079B243",
				Technology:     model.TechLTE,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       false,
				Status:         model.DeviceOffline,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})
	inform := sampleInform("SN002-BSC-NR")
	inform.DeviceId.ProductClass = "FAP/BSC7079B243"
	inform.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "DENGYO_BNQ_2.6.12.47.5"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://172.19.3.81:7547/69F5E1319CDAF252"},
	}

	device, err := svc.UpdateFromInform(context.Background(), inform)
	require.NoError(t, err)
	require.NotNil(t, device)
	require.NotNil(t, updatedDevice)
	assert.Equal(t, model.TechNR, updatedDevice.Technology)
}

// ---------------------------------------------------------------------------
// Tests: detectTechnology — case-insensitive 补充用例
// ---------------------------------------------------------------------------

// TestDetectTechnology_CaseInsensitive 锁住 detectTechnology 对 model name /
// description 大小写不敏感的契约。设备厂商命名风格不统一（华为大写、Comba
// 全小写、Ericsson 混写），不能假设固定大小写。
func TestDetectTechnology_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		params []tr069.ParameterValueStruct
		want   model.Technology
	}{
		{
			name: "lowercase 5g in description",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.Description", Value: "indoor 5g pico"},
			},
			want: model.TechNR,
		},
		{
			name: "mixed case gNB",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.ModelName", Value: "gNB-DU-100"},
			},
			want: model.TechNR,
		},
		{
			name: "mixed-case nr in middle of string",
			params: []tr069.ParameterValueStruct{
				{Name: "Device.DeviceInfo.Description", Value: "5G nr base station"},
			},
			want: model.TechNR,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, detectTechnology(tc.params))
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: findParamValue (package-level helper)
// ---------------------------------------------------------------------------

func TestFindParamValue(t *testing.T) {
	params := []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "2.3.1"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://10.0.0.1:7547"},
	}

	t.Run("found", func(t *testing.T) {
		val := findParamValue(params, "Device.DeviceInfo.SoftwareVersion")
		assert.Equal(t, "2.3.1", val)
	})

	t.Run("not found returns empty", func(t *testing.T) {
		val := findParamValue(params, "Device.NonExistent.Path")
		assert.Equal(t, "", val)
	})

	t.Run("empty params returns empty", func(t *testing.T) {
		val := findParamValue(nil, "Device.DeviceInfo.SoftwareVersion")
		assert.Equal(t, "", val)
	})
}

// ---------------------------------------------------------------------------
// Tests: T-0123 device.online 事件 + firmware 二选一挡板
// ---------------------------------------------------------------------------

// newTestDeviceServiceWithBus 构造一个挂事件总线的 DeviceService（T-0123 测试用）。
func newTestDeviceServiceWithBus(deviceRepo *mockDeviceRepo, paramRepo *mockParamRepo, bus event.EventBus) *DeviceService {
	return NewDeviceService(deviceRepo, paramRepo, nil, bus, zap.NewNop())
}

func TestPublishDeviceRegistered_UsesStableSourceEventID(t *testing.T) {
	bus := event.NewChannelEventBus(16, zap.NewNop())
	t.Cleanup(func() { _ = bus.Close() })
	received := make(chan event.Event, 1)
	_, err := bus.Subscribe(event.SubjectDeviceRegistered, func(_ context.Context, evt event.Event) error {
		received <- evt
		return nil
	})
	require.NoError(t, err)
	svc := newTestDeviceServiceWithBus(&mockDeviceRepo{}, &mockParamRepo{}, bus)

	err = svc.PublishDeviceRegistered(
		context.Background(),
		&model.Device{ID: uuid.New(), SerialNumber: "SN-STABLE-EVENT"},
		true,
		"bootstrap-event-1",
	)
	require.NoError(t, err)

	select {
	case evt := <-received:
		assert.Equal(t, "bootstrap-event-1", evt.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for device.registered event")
	}
}

// captureOnlineEvent 订阅 device.online 主题，把收到的事件压进 channel。
func captureOnlineEvent(t *testing.T, bus event.EventBus) chan event.Event {
	t.Helper()
	ch := make(chan event.Event, 1)
	_, err := bus.Subscribe(event.SubjectDeviceOnline, func(_ context.Context, evt event.Event) error {
		ch <- evt
		return nil
	})
	require.NoError(t, err)
	return ch
}

func TestUpdateFromInform_OfflineToActive_PublishesOnlineEvent(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				ProductClass:    "SmallCell",
				Status:          model.DeviceOffline,
				FirmwareVersion: "1.0.0", // same as Inform → no firmware change
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	onlineCh := captureOnlineEvent(t, bus)
	svc := newTestDeviceServiceWithBus(deviceRepo, &mockParamRepo{}, bus)

	dev, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-ONLINE-001"))
	require.NoError(t, err)
	require.NotNil(t, dev)
	assert.Equal(t, model.DeviceActive, dev.Status)

	select {
	case evt := <-onlineCh:
		assert.Equal(t, event.SubjectDeviceOnline, evt.Subject)
		var payload DeviceOnlineEvent
		require.NoError(t, evt.DecodePayload(&payload))
		assert.Equal(t, deviceID, payload.DeviceID)
		assert.Equal(t, "SN-ONLINE-001", payload.SerialNumber)
		assert.Equal(t, "SmallCell", payload.ProductClass)
		assert.Equal(t, "1.0.0", payload.SwVersion)
	case <-time.After(2 * time.Second):
		t.Fatal("expected device.online event not published within 2s")
	}
}

func TestUpdateFromInform_OfflineToActive_ClearsDisconnectedAlarm(t *testing.T) {
	deviceID := uuid.New()
	omc := "OMC"
	disconnectedAlarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		DeviceSN:        "SN-ONLINE-ALARM",
		AlarmIdentifier: "7",
		AlarmSource:     &omc,
	}
	cleaner := &mockDisconnectedAlarmCleaner{active: map[string]*model.Alarm{
		"SN-ONLINE-ALARM:7": disconnectedAlarm,
	}}
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				Status:          model.DeviceOffline,
				Technology:      model.TechLTE,
				FirmwareVersion: "1.0.0",
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc := NewDeviceService(deviceRepo, &mockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetDisconnectedAlarmCleaner(cleaner, cleaner)

	dev, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-ONLINE-ALARM"))

	require.NoError(t, err)
	require.NotNil(t, dev)
	require.Equal(t, 1, cleaner.lookups)
	require.Len(t, cleaner.cleared, 1)
	assert.Equal(t, disconnectedAlarm.ID, cleaner.cleared[0].ID)
	require.NotNil(t, cleaner.cleared[0].ClearedBy)
	assert.Equal(t, "system:device_online", *cleaner.cleared[0].ClearedBy)
	require.NotNil(t, cleaner.cleared[0].ClearNote)
	assert.Equal(t, "device reported online", *cleaner.cleared[0].ClearNote)
}

func TestUpdateFromInform_ActiveStaysActive_DoesNotQueryDisconnectedAlarm(t *testing.T) {
	cleaner := &mockDisconnectedAlarmCleaner{active: map[string]*model.Alarm{
		"SN-STILL-ACTIVE-ALARM:7": {ID: uuid.New(), DeviceSN: "SN-STILL-ACTIVE-ALARM", AlarmIdentifier: "7"},
	}}
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              uuid.New(),
				SerialNumber:    sn,
				Status:          model.DeviceActive,
				Technology:      model.TechLTE,
				FirmwareVersion: "1.0.0",
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc := NewDeviceService(deviceRepo, &mockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetDisconnectedAlarmCleaner(cleaner, cleaner)

	_, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-STILL-ACTIVE-ALARM"))

	require.NoError(t, err)
	require.Zero(t, cleaner.lookups)
	require.Empty(t, cleaner.cleared)
}

func TestUpdateFromInform_FirmwareChangedSuppressesOnlineEvent(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				ProductClass:    "SmallCell",
				Status:          model.DeviceOffline,
				FirmwareVersion: "0.9.0", // != Inform "1.0.0" → firmware changed
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	onlineCh := captureOnlineEvent(t, bus)
	firmwareCh := make(chan event.Event, 1)
	_, subErr := bus.Subscribe(event.SubjectDeviceFirmwareChanged, func(_ context.Context, e event.Event) error {
		firmwareCh <- e
		return nil
	})
	require.NoError(t, subErr)
	svc := newTestDeviceServiceWithBus(deviceRepo, &mockParamRepo{}, bus)

	dev, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-FW-001"))
	require.NoError(t, err)
	require.NotNil(t, dev)
	assert.Equal(t, "1.0.0", dev.FirmwareVersion)
	assert.Equal(t, model.DeviceActive, dev.Status)

	// T-0125: firmware 变化 → 发 firmware.changed 含 oldVersion/newVersion + becameOnline=true
	select {
	case evt := <-firmwareCh:
		assert.Equal(t, event.SubjectDeviceFirmwareChanged, evt.Subject)
		var payload DeviceFirmwareChangedEvent
		require.NoError(t, evt.DecodePayload(&payload))
		assert.Equal(t, deviceID, payload.DeviceID)
		assert.Equal(t, "0.9.0", payload.OldVersion)
		assert.Equal(t, "1.0.0", payload.NewVersion)
		assert.True(t, payload.BecameOnline, "同 Inform 也满足 offline→active，BecameOnline 应为 true")
	case <-time.After(2 * time.Second):
		t.Fatal("expected firmware.changed event not published within 2s")
	}

	// 二选一：firmware 变化时不发 device.online
	select {
	case evt := <-onlineCh:
		t.Fatalf("expected NO device.online event when firmware changed, got %v", evt.Subject)
	case <-time.After(200 * time.Millisecond):
		// 期望路径
	}
}

func TestUpdateFromInform_FirmwareChanged_ActiveStaysActive_BecameOnlineFalse(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				ProductClass:    "SmallCell",
				Status:          model.DeviceActive, // 一直在线
				FirmwareVersion: "0.9.0",            // != Inform "1.0.0"
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	firmwareCh := make(chan event.Event, 1)
	_, subErr := bus.Subscribe(event.SubjectDeviceFirmwareChanged, func(_ context.Context, e event.Event) error {
		firmwareCh <- e
		return nil
	})
	require.NoError(t, subErr)
	svc := newTestDeviceServiceWithBus(deviceRepo, &mockParamRepo{}, bus)

	_, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-FW-ACTIVE"))
	require.NoError(t, err)

	select {
	case evt := <-firmwareCh:
		var payload DeviceFirmwareChangedEvent
		require.NoError(t, evt.DecodePayload(&payload))
		assert.False(t, payload.BecameOnline, "active→active 时 BecameOnline 应为 false")
	case <-time.After(2 * time.Second):
		t.Fatal("expected firmware.changed event even when status didn't change")
	}
}

func TestUpdateFromInform_ActiveStaysActive_NoOnlineEvent(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				ProductClass:    "SmallCell",
				Status:          model.DeviceActive, // already active
				FirmwareVersion: "1.0.0",
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	onlineCh := captureOnlineEvent(t, bus)
	svc := newTestDeviceServiceWithBus(deviceRepo, &mockParamRepo{}, bus)

	_, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-STILL-ACTIVE"))
	require.NoError(t, err)

	select {
	case evt := <-onlineCh:
		t.Fatalf("expected NO device.online event when active stays active, got %v", evt.Subject)
	case <-time.After(200 * time.Millisecond):
		// 期望路径
	}
}

func TestUpdateFromInform_NilEventBus_NoCrash(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              uuid.New(),
				SerialNumber:    sn,
				Status:          model.DeviceOffline,
				FirmwareVersion: "1.0.0",
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc := newTestDeviceService(deviceRepo, &mockParamRepo{}) // eventBus = nil

	_, err := svc.UpdateFromInform(context.Background(), sampleInform("SN-NIL-BUS"))
	assert.NoError(t, err)
}
