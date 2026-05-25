package device

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mocks (infMock prefix to avoid conflicts with service_test / handler_test)
// ---------------------------------------------------------------------------

type infMockDeviceRepo struct {
	createFn            func(ctx context.Context, device *model.Device) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
	updateFn            func(ctx context.Context, device *model.Device) error
	deleteFn            func(ctx context.Context, id uuid.UUID) error
	listFn              func(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	updateLastInformFn  func(ctx context.Context, sn string, at time.Time, events []string) error
	recordBootFn        func(ctx context.Context, sn string, at time.Time) (int, error)
	countByStatusFn     func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
}

func (m *infMockDeviceRepo) Create(ctx context.Context, device *model.Device) error {
	if m.createFn != nil {
		return m.createFn(ctx, device)
	}
	return nil
}
func (m *infMockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *infMockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if m.getBySerialNumberFn != nil {
		return m.getBySerialNumberFn(ctx, sn)
	}
	return nil, nil
}
func (m *infMockDeviceRepo) Update(ctx context.Context, device *model.Device) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, device)
	}
	return nil
}
func (m *infMockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *infMockDeviceRepo) BatchDelete(_ context.Context, _ []uuid.UUID, _ string) (int64, error) {
	return 0, nil
}
func (m *infMockDeviceRepo) List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[model.Device]{Items: []model.Device{}}, nil
}
func (m *infMockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

// T-0162 新接口方法
func (m *infMockDeviceRepo) UpdateLifecycle(_ context.Context, _ uuid.UUID, _ model.DeviceLifecycle) error {
	return nil
}

func (m *infMockDeviceRepo) UpdateOnlineStatus(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *infMockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.updateLastInformFn != nil {
		return m.updateLastInformFn(ctx, sn, at, events)
	}
	return nil
}
func (m *infMockDeviceRepo) RecordBoot(ctx context.Context, sn string, at time.Time) (int, error) {
	if m.recordBootFn != nil {
		return m.recordBootFn(ctx, sn, at)
	}
	return 0, nil
}
func (m *infMockDeviceRepo) CountByStatus(ctx context.Context, c *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, c)
	}
	return map[model.DeviceStatus]int64{}, nil
}
func (m *infMockDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *infMockDeviceRepo) ListGeo(_ context.Context, _ GeoDeviceFilter) ([]GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *infMockDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*GeoStats, error) {
	return &GeoStats{}, nil
}
func (m *infMockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]GeoDevice, error) {
	return nil, nil
}
func (m *infMockDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}

func (m *infMockDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (m *infMockDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

func (m *infMockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *infMockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *infMockDeviceRepo) ListRecycleBin(_ context.Context, _ RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *infMockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *infMockDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *infMockDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

type infMockParamRepo struct {
	batchUpsertFn func(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
	getByDeviceFn func(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
	getByPathFn   func(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
	deleteByDevFn func(ctx context.Context, deviceID uuid.UUID) error
}

func (m *infMockParamRepo) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	if m.batchUpsertFn != nil {
		return m.batchUpsertFn(ctx, deviceID, params)
	}
	return nil
}
func (m *infMockParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	if m.getByDeviceFn != nil {
		return m.getByDeviceFn(ctx, deviceID)
	}
	return nil, nil
}
func (m *infMockParamRepo) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	if m.getByPathFn != nil {
		return m.getByPathFn(ctx, deviceID, path)
	}
	return nil, nil
}
func (m *infMockParamRepo) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error {
	if m.deleteByDevFn != nil {
		return m.deleteByDevFn(ctx, deviceID)
	}
	return nil
}

func (m *infMockParamRepo) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}

func (m *infMockParamRepo) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (m *infMockParamRepo) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}

func (m *infMockParamRepo) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (m *infMockParamRepo) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (m *infMockParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *infMockParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *infMockParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

// Mock carrier for registry tests
type infMockCarrier struct {
	code         model.CarrierCode
	technologies []model.Technology
	ouiProducts  map[model.Technology][]carrier.OUIProductClassInfo
}

func (c *infMockCarrier) Code() model.CarrierCode                              { return c.code }
func (c *infMockCarrier) Name() string                                         { return string(c.code) }
func (c *infMockCarrier) SupportedTechnologies() []model.Technology            { return c.technologies }
func (c *infMockCarrier) DefaultDataModelVersions(_ model.Technology) []string { return nil }
func (c *infMockCarrier) KnownOUIProductClasses(tech model.Technology) []carrier.OUIProductClassInfo {
	return c.ouiProducts[tech]
}
func (c *infMockCarrier) MapParameterToUnified(_ string) string { return "" }
func (c *infMockCarrier) MapUnifiedToParameter(_ string) string { return "" }
func (c *infMockCarrier) ProvisioningTemplates(_ model.Technology) []*carrier.ProvisionTemplate {
	return nil
}
func (c *infMockCarrier) AlarmSeverityMapping(_ string) model.AlarmSeverity          { return 0 }
func (c *infMockCarrier) ValidateParameter(_ string, _ string) error                 { return nil }
func (c *infMockCarrier) GetInfoParamMapping(_ model.Technology) map[string]string   { return nil }
func (c *infMockCarrier) RFControlPath(_ model.Technology) string                    { return "" }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newInfTestDeviceService(deviceRepo *infMockDeviceRepo, paramRepo *infMockParamRepo) *DeviceService {
	return NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())
}

func sampleInformPayload(sn string) InformEventPayload {
	return InformEventPayload{
		DeviceId: tr069.DeviceId{
			Manufacturer: "TestVendor",
			OUI:          "AABBCC",
			ProductClass: "SmallCell",
			SerialNumber: sn,
		},
		Events: []string{tr069.EventBootstrap},
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.DeviceInfo.SoftwareVersion", Value: "1.0.0"},
			{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://192.168.1.1:7547"},
			{Name: "Device.DeviceInfo.ModelName", Value: "PicoCell-LTE"},
		},
		CurrentTime: "2024-01-15T10:30:00Z",
		RetryCount:  0,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewInformHandler(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	require.NotNil(t, h)
	assert.Equal(t, svc, h.service)
	assert.Equal(t, registry, h.carrierRegistry)
	assert.Equal(t, model.CarrierCMCC, h.defaultCarrier)
}

func TestPayloadToInform(t *testing.T) {
	payload := InformEventPayload{
		DeviceId: tr069.DeviceId{
			Manufacturer: "Acme",
			OUI:          "112233",
			ProductClass: "Femto",
			SerialNumber: "SN-PAY-001",
		},
		Events: []string{"0 BOOTSTRAP", "2 PERIODIC"},
		ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.DeviceInfo.SoftwareVersion", Value: "2.0.0"},
		},
	}

	inform := payloadToInform(payload)
	require.NotNil(t, inform)
	assert.Equal(t, payload.DeviceId, inform.DeviceId)
	require.Len(t, inform.Event, 2)
	assert.Equal(t, "0 BOOTSTRAP", inform.Event[0].EventCode)
	assert.Equal(t, "2 PERIODIC", inform.Event[1].EventCode)
	require.Len(t, inform.ParameterList, 1)
	assert.Equal(t, "Device.DeviceInfo.SoftwareVersion", inform.ParameterList[0].Name)
}

func TestResolveCarrier_RegistryMatch(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	registry.Register(&infMockCarrier{
		code:         model.CarrierCTCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
			model.TechLTE: {{OUI: "AABBCC", ProductClass: "SmallCell"}},
		},
	})
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	result := h.resolveCarrier("AABBCC")
	assert.Equal(t, model.CarrierCTCC, result)
}

func TestResolveCarrier_DefaultFallback(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	registry.Register(&infMockCarrier{
		code:         model.CarrierCTCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
			model.TechLTE: {{OUI: "XXYYZZ", ProductClass: "Other"}},
		},
	})
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	result := h.resolveCarrier("UNKNOWN_OUI")
	assert.Equal(t, model.CarrierCMCC, result)
}

func TestResolveCarrier_NilRegistry(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCUCC, zap.NewNop())

	result := h.resolveCarrier("AABBCC")
	assert.Equal(t, model.CarrierCUCC, result)
}

func TestHandleBootstrap_Success(t *testing.T) {
	var createdDevice *model.Device
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil // device doesn't exist yet
		},
		createFn: func(_ context.Context, device *model.Device) error {
			createdDevice = device
			return nil
		},
	}
	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-BOOT-001")
	evt, err := event.NewEvent(event.SubjectDeviceBootstrap, payload)
	require.NoError(t, err)

	err = h.handleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, createdDevice)
	assert.Equal(t, "SN-BOOT-001", createdDevice.SerialNumber)
	assert.Equal(t, "AABBCC", createdDevice.OUI)
	assert.Equal(t, model.CarrierCMCC, createdDevice.Carrier)
	assert.Equal(t, model.DeviceActive, createdDevice.Status)
}

func TestHandleRebootComplete_NormalReboot(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device
	var recordedSN string
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				OUI:            "AABBCC",
				Carrier:        model.CarrierCMCC,
				Status:         model.DeviceActive,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
		recordBootFn: func(_ context.Context, sn string, _ time.Time) (int, error) {
			recordedSN = sn
			return 3, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-REBOOT-001")
	payload.Events = []string{tr069.EventBoot, tr069.EventMReboot}
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, updatedDevice)
	assert.Equal(t, "SN-REBOOT-001", recordedSN)
}

func TestHandleRebootComplete_AbnormalReboot_PublishesAbnormalEvent(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				OUI:            "AABBCC",
				Carrier:        model.CarrierCMCC,
				Status:         model.DeviceActive,
				InformInterval: 300,
			}, nil
		},
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 7, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	abnormalReceived := make(chan event.Event, 1)
	_, subErr := bus.Subscribe(event.SubjectDeviceRebootAbnormal, func(_ context.Context, evt event.Event) error {
		abnormalReceived <- evt
		return nil
	})
	require.NoError(t, subErr)

	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-REBOOT-ABN-001")
	payload.Events = []string{tr069.EventBoot} // only "1 BOOT", no "M Reboot"
	// T-0158: 异常重启判断改为基于 HaltReason.MainReason 非空
	payload.ParameterList = append(payload.ParameterList,
		tr069.ParameterValueStruct{Name: HaltReasonMainPath, Value: "halt_reboot"},
		tr069.ParameterValueStruct{Name: HaltReasonDetailPath, Value: "watchdog_timeout"},
	)
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)

	select {
	case got := <-abnormalReceived:
		assert.Equal(t, event.SubjectDeviceRebootAbnormal, got.Subject)
		// 校验新增字段透传到 payload
		var p map[string]interface{}
		require.NoError(t, got.DecodePayload(&p))
		assert.Equal(t, "halt_reboot", p["halt_main_reason"])
		assert.Equal(t, "watchdog_timeout", p["halt_detail_reason"])
	case <-time.After(time.Second):
		t.Fatal("expected device.reboot.abnormal event, got none")
	}
}

// T-0158: 1 BOOT 但 HaltReason.MainReason 为空 → 正常重启，不发异常事件。
// 这是新判断规则相对旧规则（"1 BOOT && !M Reboot"）的关键差异：
// 受控的重启 / 看门狗内部恢复 CPE 不带 M Reboot 也不带 HaltReason，
// 现在不会再被误判为异常。
func TestHandleRebootComplete_BootWithoutHaltReason_NoAbnormalEvent(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				OUI:            "AABBCC",
				Carrier:        model.CarrierCMCC,
				Status:         model.DeviceActive,
				InformInterval: 300,
			}, nil
		},
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 3, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	abnormalReceived := make(chan event.Event, 1)
	_, subErr := bus.Subscribe(event.SubjectDeviceRebootAbnormal, func(_ context.Context, evt event.Event) error {
		abnormalReceived <- evt
		return nil
	})
	require.NoError(t, subErr)

	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-REBOOT-NORMAL-001")
	payload.Events = []string{tr069.EventBoot}
	// 故意不带 HaltReason 参数（sampleInformPayload 默认就没有）
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)

	select {
	case got := <-abnormalReceived:
		t.Fatalf("unexpected abnormal event: %s", got.Subject)
	case <-time.After(200 * time.Millisecond):
		// expected: no event published
	}
}

func TestHandleRebootComplete_AutoRegisterWhenMissing(t *testing.T) {
	var created bool
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil // device missing
		},
		createFn: func(_ context.Context, _ *model.Device) error {
			created = true
			return nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-REBOOT-ORPHAN-001")
	payload.Events = []string{tr069.EventBoot}
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)
	assert.True(t, created, "expected auto-register to call Create")
}

func TestHandlePeriodic_Success(t *testing.T) {
	deviceID := uuid.New()
	var updatedDevice *model.Device
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				OUI:            "AABBCC",
				Status:         model.DeviceActive,
				InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			updatedDevice = device
			return nil
		},
	}
	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-PER-001")
	payload.Events = []string{"2 PERIODIC"}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)

	err = h.handlePeriodic(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, updatedDevice)
	assert.Equal(t, deviceID, updatedDevice.ID)
	assert.NotNil(t, updatedDevice.LastInformAt)
}
