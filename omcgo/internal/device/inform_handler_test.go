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
	getDeletedFn        func(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error)
	updateFn            func(ctx context.Context, device *model.Device) error
	deleteFn            func(ctx context.Context, id uuid.UUID) error
	listFn              func(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	updateLastInformFn  func(ctx context.Context, sn string, at time.Time, events []string) error
	recordBootFn        func(ctx context.Context, sn string, at time.Time) (int, error)
	countByStatusFn     func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
	updateOnlineFn      func(ctx context.Context, id uuid.UUID, isOnline bool) error
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
func (m *infMockDeviceRepo) GetDeletedBySerialNumber(ctx context.Context, sn string, carrier model.CarrierCode) (*model.Device, error) {
	if m.getDeletedFn != nil {
		return m.getDeletedFn(ctx, sn, carrier)
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

func (m *infMockDeviceRepo) UpdateOnlineStatus(ctx context.Context, id uuid.UUID, isOnline bool) error {
	if m.updateOnlineFn != nil {
		return m.updateOnlineFn(ctx, id, isOnline)
	}
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
func (m *infMockDeviceRepo) GetGeoStats(_ context.Context, _ GeoStatsFilter) (*GeoStats, error) {
	return &GeoStats{}, nil
}
func (m *infMockDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, _ []model.DeviceVisibilityGrant) ([]GeoDevice, error) {
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

func (m *infMockDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *infMockDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *infMockDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *infMockDeviceRepo) ListRecycleBin(_ context.Context, _ RecycleBinFilter) (*model.ListResponse[DeviceWithInfo], error) {
	return model.NewListResponse([]DeviceWithInfo{}, 0, 1, 20), nil
}
func (m *infMockDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (*RestoreResult, error) {
	return &RestoreResult{}, nil
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
func (c *infMockCarrier) AlarmSeverityMapping(_ string) model.AlarmSeverity        { return 0 }
func (c *infMockCarrier) ValidateParameter(_ string, _ string) error               { return nil }
func (c *infMockCarrier) GetInfoParamMapping(_ model.Technology) map[string]string { return nil }
func (c *infMockCarrier) RFControlPath(_ model.Technology) string                  { return "" }
func (c *infMockCarrier) SupportsMRType(_ model.MRType) bool                       { return true }

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

func TestInformHandlerSubscribeUsesKeyedPeriodicConsumer(t *testing.T) {
	bus := &rpcRespRecordingBus{}
	h := NewInformHandler(nil, nil, model.CarrierCMCC, zap.NewNop())

	require.NoError(t, h.Subscribe(bus))

	require.Len(t, bus.keyedCalls, 1)
	assert.Equal(t, event.SubjectDevicePeriodic, bus.keyedCalls[0].subject)
	assert.Equal(t, "device-mgr-periodic", bus.keyedCalls[0].config.Durable)
	assert.Equal(t, periodicConsumerConcurrency, bus.keyedCalls[0].config.Concurrency)
	assert.Equal(t, periodicConsumerQueueDepth, bus.keyedCalls[0].config.QueueDepth)
	assert.GreaterOrEqual(t, bus.keyedCalls[0].config.Concurrency, 64,
		"20k devices reporting every minute require enough parallel handlers to sustain the production rate")
	assert.GreaterOrEqual(t, bus.keyedCalls[0].config.QueueDepth, 64,
		"64 shards x 64 entries absorb short reconnect bursts without an unbounded in-process buffer")
	for _, call := range bus.queueCalls {
		assert.NotEqual(t, event.SubjectDevicePeriodic, call.subject)
	}
}

func TestPeriodicDeviceKeyUsesSerialNumber(t *testing.T) {
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, sampleInformPayload(" SN-KEYED-001 "))
	require.NoError(t, err)

	key, err := periodicDeviceKey(evt)

	require.NoError(t, err)
	assert.Equal(t, "SN-KEYED-001", key)
}

func TestPeriodicDeviceKeyRejectsMissingSerialNumber(t *testing.T) {
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, sampleInformPayload(""))
	require.NoError(t, err)

	_, err = periodicDeviceKey(evt)

	require.Error(t, err)
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

	result, err := h.resolveCarrier("AABBCC", "SmallCell")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCTCC, result)
}

func TestResolveCarrier_DisambiguatesSharedOUIByProductClass(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	registry.Register(&infMockCarrier{
		code:         model.CarrierCMCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
			model.TechLTE: {{OUI: "AABBCC", ProductClass: "SmallCell-LTE"}},
		},
	})
	registry.Register(&infMockCarrier{
		code:         model.CarrierCTCC,
		technologies: []model.Technology{model.TechLTE},
		ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
			model.TechLTE: {{OUI: "AABBCC", ProductClass: "eSmallCell-LTE"}},
		},
	})
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	result, err := h.resolveCarrier("AABBCC", "eSmallCell-LTE")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCTCC, result)
}

func TestResolveCarrier_DoesNotFallbackForAmbiguousSharedOUI(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	for _, code := range []model.CarrierCode{model.CarrierCMCC, model.CarrierCTCC} {
		registry.Register(&infMockCarrier{
			code:         code,
			technologies: []model.Technology{model.TechLTE},
			ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
				model.TechLTE: {{OUI: "AABBCC", ProductClass: string(code) + "-product"}},
			},
		})
	}
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	result, err := h.resolveCarrier("AABBCC", "")
	assert.Empty(t, result)
	assert.ErrorIs(t, err, carrier.ErrAmbiguousCarrier)
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

	result, err := h.resolveCarrier("UNKNOWN_OUI", "Unknown")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCMCC, result)
}

func TestResolveCarrier_NilRegistry(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCUCC, zap.NewNop())

	result, err := h.resolveCarrier("AABBCC", "SmallCell")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCUCC, result)
}

// #17: 默认运营商由 CarrierRegistry.DefaultCarrier() 注入，而非 wiring 处硬编码
// CarrierCMCC。本测试以注册表派生的默认值构造 handler，验证 OUI 未命中时
// resolveCarrier 回退到该派生默认值（注册表未注册 CMCC 时不再强行落到 CMCC）。
func TestResolveCarrier_RegistryDerivedDefault(t *testing.T) {
	svc := newInfTestDeviceService(&infMockDeviceRepo{}, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	// 注册表里没有 CMCC：默认应回退到字典序最小的已注册运营商（ctcc < cucc）。
	registry.Register(&infMockCarrier{code: model.CarrierCUCC})
	registry.Register(&infMockCarrier{code: model.CarrierCTCC})

	defaultCarrier := registry.DefaultCarrier()
	require.Equal(t, model.CarrierCTCC, defaultCarrier,
		"registry without CMCC must derive a deterministic default, not hardcode CMCC")

	h := NewInformHandler(svc, registry, defaultCarrier, zap.NewNop())
	resolved, err := h.resolveCarrier("UNKNOWN_OUI", "Unknown")
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCTCC, resolved,
		"unresolved OUI must fall back to the registry-derived default carrier")
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

func TestHandleBootstrap_NewDevicePublishesCreated(t *testing.T) {
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	defer bus.Close()
	received := make(chan event.Event, 1)
	_, err := bus.Subscribe(event.SubjectDeviceRegistered, func(_ context.Context, evt event.Event) error {
		received <- evt
		return nil
	})
	require.NoError(t, err)

	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	evt, err := event.NewEvent(event.SubjectDeviceBootstrap, sampleInformPayload("SN-BOOT-CREATED"))
	require.NoError(t, err)

	require.NoError(t, h.handleBootstrap(context.Background(), evt))

	select {
	case published := <-received:
		assert.Equal(t, evt.ID, published.ID)
		var payload struct {
			DeviceID uuid.UUID `json:"device_id"`
			Created  bool      `json:"created"`
		}
		require.NoError(t, published.DecodePayload(&payload))
		assert.NotEqual(t, uuid.Nil, payload.DeviceID)
		assert.True(t, payload.Created)
	case <-time.After(time.Second):
		t.Fatal("expected device.registered event")
	}
}

func TestHandleBootstrap_DeviceRegisteredPublishFailureIsRetryable(t *testing.T) {
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	require.NoError(t, bus.Close())
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	evt, err := event.NewEvent(event.SubjectDeviceBootstrap, sampleInformPayload("SN-BOOT-PUBLISH-FAIL"))
	require.NoError(t, err)

	err = h.handleBootstrap(context.Background(), evt)

	require.ErrorIs(t, err, event.ErrBusClosed)
}

func TestHandleBootstrap_AmbiguousSharedOUIDoesNotRegisterDevice(t *testing.T) {
	var createCalled bool
	deviceRepo := &infMockDeviceRepo{
		createFn: func(_ context.Context, _ *model.Device) error {
			createCalled = true
			return nil
		},
	}
	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	for _, code := range []model.CarrierCode{model.CarrierCMCC, model.CarrierCTCC} {
		registry.Register(&infMockCarrier{
			code:         code,
			technologies: []model.Technology{model.TechLTE},
			ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
				model.TechLTE: {{OUI: "AABBCC", ProductClass: string(code) + "-product"}},
			},
		})
	}
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-SHARED-AMBIGUOUS")
	payload.DeviceId.ProductClass = ""
	evt, err := event.NewEvent(event.SubjectDeviceBootstrap, payload)
	require.NoError(t, err)

	err = h.handleBootstrap(context.Background(), evt)
	assert.ErrorIs(t, err, carrier.ErrAmbiguousCarrier)
	assert.False(t, createCalled)
}

func TestHandleBootstrap_ExistingDeviceKeepsStoredCarrierForAmbiguousIdentity(t *testing.T) {
	existing := &model.Device{
		ID:           uuid.New(),
		SerialNumber: "SN-SHARED-EXISTING",
		OUI:          "AABBCC",
		ProductClass: "legacy-product",
		Carrier:      model.CarrierCTCC,
		Technology:   model.TechLTE,
		Status:       model.DeviceActive,
	}
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return existing, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			assert.Equal(t, model.CarrierCTCC, device.Carrier)
			return nil
		},
	}
	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	registry := carrier.NewRegistry()
	for _, code := range []model.CarrierCode{model.CarrierCMCC, model.CarrierCTCC} {
		registry.Register(&infMockCarrier{
			code:         code,
			technologies: []model.Technology{model.TechLTE},
			ouiProducts: map[model.Technology][]carrier.OUIProductClassInfo{
				model.TechLTE: {{OUI: "AABBCC", ProductClass: string(code) + "-product"}},
			},
		})
	}
	h := NewInformHandler(svc, registry, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload(existing.SerialNumber)
	payload.DeviceId.ProductClass = "legacy-product"
	evt, err := event.NewEvent(event.SubjectDeviceBootstrap, payload)
	require.NoError(t, err)

	err = h.handleBootstrap(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, model.CarrierCTCC, existing.Carrier)
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
	// T-0158: 非 gNB 设备的异常重启判断基于 HaltReason.MainReason 非空
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

func TestHandleRebootComplete_AutoRegisterPublishFailureIsRetryable(t *testing.T) {
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	require.NoError(t, bus.Close())
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	payload := sampleInformPayload("SN-REBOOT-PUBLISH-FAIL")
	payload.Events = []string{tr069.EventBoot}
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)

	require.ErrorIs(t, err, event.ErrBusClosed)
}

func TestHandleRebootComplete_NormalRecordUsesPreRebootDeviceSnapshot(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				DeviceName:      "BSC-before-reboot",
				OUI:             "AABBCC",
				Carrier:         model.CarrierCMCC,
				Technology:      model.TechGSM,
				Status:          model.DeviceActive,
				LifecycleState:  model.LifecycleCommissioned,
				IsOnline:        true,
				IPAddress:       "172.21.172.109",
				FirmwareVersion: "BaiBS_before",
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			assert.Equal(t, "BaiBS_after", device.FirmwareVersion)
			return nil
		},
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 4, nil
		},
	}

	rec := &infMockBootEventRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetBootEventRecorder(rec)
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-PRE-REBOOT-NORMAL")
	payload.Events = []string{tr069.EventBoot}
	payload.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "BaiBS_after"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://10.0.0.2:7547"},
		{Name: "Device.DeviceInfo.ModelName", Value: "PicoCell-LTE"},
	}
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "BaiBS_before", rec.calls[0].SoftwareVersion)
	assert.Equal(t, "172.21.172.109", rec.calls[0].OperateIP)
	assert.Equal(t, "GSM", rec.calls[0].DeviceType)
	assert.Equal(t, "BSC-before-reboot", rec.calls[0].DeviceName)
}

func TestHandleRebootComplete_AbnormalRecordUsesPreRebootDeviceSnapshot(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:              deviceID,
				SerialNumber:    sn,
				DeviceName:      "BSC-before-abnormal",
				OUI:             "AABBCC",
				Carrier:         model.CarrierCMCC,
				Technology:      model.TechGSM,
				Status:          model.DeviceActive,
				LifecycleState:  model.LifecycleCommissioned,
				IsOnline:        true,
				IPAddress:       "172.21.172.110",
				FirmwareVersion: "BaiBS_before_abnormal",
				InformInterval:  300,
			}, nil
		},
		updateFn: func(_ context.Context, device *model.Device) error {
			assert.Equal(t, "BaiBS_after_abnormal", device.FirmwareVersion)
			return nil
		},
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 6, nil
		},
	}

	rec := &infMockAbnormalRebootRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetAbnormalRebootRecorder(rec)
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-PRE-REBOOT-ABNORMAL")
	payload.Events = []string{tr069.EventBoot}
	payload.ParameterList = []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "BaiBS_after_abnormal"},
		{Name: "Device.ManagementServer.ConnectionRequestURL", Value: "http://10.0.0.3:7547"},
		{Name: "Device.DeviceInfo.ModelName", Value: "PicoCell-LTE"},
		{Name: HaltReasonMainPath, Value: "halt_reboot"},
	}
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "BaiBS_before_abnormal", rec.calls[0].SoftwareVersion)
	assert.Equal(t, "172.21.172.110", rec.calls[0].OperateIP)
	assert.Equal(t, "GSM", rec.calls[0].DeviceType)
	assert.Equal(t, "BSC-before-abnormal", rec.calls[0].DeviceName)
}

type infMockAbnormalRebootRecorder struct {
	calls []AbnormalRebootSnapshot
}

func (m *infMockAbnormalRebootRecorder) RecordAbnormalReboot(_ context.Context, snap AbnormalRebootSnapshot) error {
	m.calls = append(m.calls, snap)
	return nil
}

func TestRecordBootFromInform_GSMAbnormalSnapshotUsesGSMDeviceType(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 9, nil
		},
	}
	rec := &infMockAbnormalRebootRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetAbnormalRebootRecorder(rec)

	params := []tr069.ParameterValueStruct{
		{Name: HaltReasonMainPath, Value: "halt_reboot"},
	}
	_, err := svc.RecordBootFromInform(context.Background(), &model.Device{
		ID:              deviceID,
		SerialNumber:    "SN-GSM-ABNORMAL",
		Technology:      model.TechGSM,
		DeviceName:      "BSC-1",
		IPAddress:       "172.21.172.109",
		FirmwareVersion: "BaiBS_AGC_2.1.7.9",
	}, []string{tr069.EventBoot}, params, 300)
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "GSM", rec.calls[0].DeviceType)
	assert.False(t, rec.calls[0].IsGNB)
	assert.Equal(t, "172.21.172.109", rec.calls[0].OperateIP)
}

func TestRecordBootFromInform_GNBRequiresHaltRebootMainReason(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 8, nil
		},
	}
	abnormalRec := &infMockAbnormalRebootRecorder{}
	normalRec := &infMockBootEventRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetAbnormalRebootRecorder(abnormalRec)
	svc.SetBootEventRecorder(normalRec)

	params := []tr069.ParameterValueStruct{
		{Name: HaltReasonMainPath, Value: "power_lost"},
	}
	_, err := svc.RecordBootFromInform(context.Background(), &model.Device{
		ID:           deviceID,
		SerialNumber: "SN-GNB-NORMAL",
		Technology:   model.TechNR,
		DeviceName:   "GNB-1",
		IPAddress:    "10.0.0.5",
	}, []string{tr069.EventBoot}, params, 180)
	require.NoError(t, err)

	assert.Empty(t, abnormalRec.calls)
	require.Len(t, normalRec.calls, 1)
	assert.Equal(t, "gNB", normalRec.calls[0].DeviceType)
	assert.True(t, normalRec.calls[0].IsGNB)
}

func TestRecordBootFromInform_GNBHaltRebootMainReasonIsAbnormal(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 9, nil
		},
	}
	rec := &infMockAbnormalRebootRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetAbnormalRebootRecorder(rec)

	params := []tr069.ParameterValueStruct{
		{Name: HaltReasonMainPath, Value: "halt_reboot"},
	}
	_, err := svc.RecordBootFromInform(context.Background(), &model.Device{
		ID:           deviceID,
		SerialNumber: "SN-GNB-ABNORMAL",
		Technology:   model.TechNR,
		DeviceName:   "GNB-2",
		IPAddress:    "10.0.0.6",
	}, []string{tr069.EventBoot}, params, 240)
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "gNB", rec.calls[0].DeviceType)
	assert.True(t, rec.calls[0].IsGNB)
	assert.Equal(t, "halt_reboot", rec.calls[0].HaltMainReason)
}

func TestRecordBootFromInform_LTENonEmptyMainReasonStillAbnormal(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 10, nil
		},
	}
	rec := &infMockAbnormalRebootRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetAbnormalRebootRecorder(rec)

	params := []tr069.ParameterValueStruct{
		{Name: HaltReasonMainPath, Value: "power_lost"},
	}
	_, err := svc.RecordBootFromInform(context.Background(), &model.Device{
		ID:           deviceID,
		SerialNumber: "SN-LTE-ABNORMAL",
		Technology:   model.TechLTE,
		DeviceName:   "ENB-1",
		IPAddress:    "10.0.0.7",
	}, []string{tr069.EventBoot}, params, 300)
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "eNB", rec.calls[0].DeviceType)
	assert.False(t, rec.calls[0].IsGNB)
	assert.Equal(t, "power_lost", rec.calls[0].HaltMainReason)
}

// infMockBootEventRecorder 记录 RecordBootEvent 被调用次数与最后一次快照，
// 供 issue #212 死判用例断言 "写一条重启记录"。
type infMockBootEventRecorder struct {
	calls []BootEventSnapshot
}

func (m *infMockBootEventRecorder) RecordBootEvent(_ context.Context, snap BootEventSnapshot) error {
	m.calls = append(m.calls, snap)
	return nil
}

func TestRecordBootFromInform_GSMNormalSnapshotUsesGSMDeviceType(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 5, nil
		},
	}
	rec := &infMockBootEventRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, nil, zap.NewNop())
	svc.SetBootEventRecorder(rec)

	params := []tr069.ParameterValueStruct{
		{Name: "Device.DeviceInfo.SoftwareVersion", Value: "BSC_2.0.5"},
	}
	_, err := svc.RecordBootFromInform(context.Background(), &model.Device{
		ID:              deviceID,
		SerialNumber:    "SN-GSM-NORMAL",
		Technology:      model.TechGSM,
		DeviceName:      "BSC-2",
		IPAddress:       "10.10.3.48",
		FirmwareVersion: "BSC_2.0.5",
	}, []string{tr069.EventBoot}, params, 100)
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, "GSM", rec.calls[0].DeviceType)
	assert.False(t, rec.calls[0].IsGNB)
	assert.Equal(t, "10.10.3.48", rec.calls[0].OperateIP)
}

// issue #212 死判：设备初始在线，收到 BOOT（1 BOOT）时必须无条件强制走出
// "下线 → 上线" 翻转（先发 device.offline，再发 device.online），且写一条重启记录。
//
// 与被动超时离线探测彻底解耦：本用例从不触发任何心跳超时，设备一直显示在线，
// 仅凭 BOOT 信号就应驱动翻转 —— 验证 "当前仍显示在线" 不再挡住翻转。
func TestHandleRebootComplete_OnlineDevice_ForcesOfflineThenOnlineFlip(t *testing.T) {
	deviceID := uuid.New()
	// stateful 在线标记：ForceBootStateFlip 写 false 后，后续 GetBySerialNumber
	// 必须反映出离线，UpdateFromInform 才能判定 oldStatus==Offline → 发 device.online。
	online := true
	var forcedOfflineCalled bool

	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			status := model.DeviceActive
			if !online {
				status = model.DeviceOffline
			}
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				OUI:            "AABBCC",
				Carrier:        model.CarrierCMCC,
				Technology:     model.TechLTE,
				Status:         status,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       online,
				InformInterval: 300,
			}, nil
		},
		updateOnlineFn: func(_ context.Context, _ uuid.UUID, isOnline bool) error {
			if !isOnline {
				forcedOfflineCalled = true
				online = false
			} else {
				online = true
			}
			return nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
		recordBootFn: func(_ context.Context, _ string, _ time.Time) (int, error) {
			return 5, nil
		},
	}

	bus := event.NewChannelEventBus(16, zap.NewNop())
	offlineCh := make(chan event.Event, 1)
	onlineCh := make(chan event.Event, 1)
	_, subErr := bus.Subscribe(event.SubjectDeviceOffline, func(_ context.Context, evt event.Event) error {
		offlineCh <- evt
		return nil
	})
	require.NoError(t, subErr)
	_, subErr = bus.Subscribe(event.SubjectDeviceOnline, func(_ context.Context, evt event.Event) error {
		onlineCh <- evt
		return nil
	})
	require.NoError(t, subErr)

	rec := &infMockBootEventRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	svc.SetBootEventRecorder(rec)
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-212-FLIP-001")
	payload.Events = []string{tr069.EventBoot} // 1 BOOT，无 HaltReason → 正常重启
	evt, err := event.NewEvent(event.SubjectDeviceRebootComplete, payload)
	require.NoError(t, err)

	err = h.handleRebootComplete(context.Background(), evt)
	require.NoError(t, err)

	// 1) 强制置离线被调用（即便设备初始在线）
	assert.True(t, forcedOfflineCalled, "BOOT 必须无条件先把设备强制置离线")

	// 2) device.offline 事件已发（"下线" 半程可见），reason=reboot 区分被动超时
	select {
	case got := <-offlineCh:
		assert.Equal(t, event.SubjectDeviceOffline, got.Subject)
		var p DeviceOfflineEvent
		require.NoError(t, got.DecodePayload(&p))
		assert.Equal(t, OfflineReasonReboot, p.Reason,
			"BOOT 驱动的离线 reason 应为 reboot，与超时离线 heartbeat_timeout 区分")
	case <-time.After(time.Second):
		t.Fatal("expected device.offline event, got none")
	}

	// 3) device.online 事件已发（"上线" 半程走出，触发 PM 配置重发）
	select {
	case got := <-onlineCh:
		assert.Equal(t, event.SubjectDeviceOnline, got.Subject)
		var p DeviceOnlineEvent
		require.NoError(t, got.DecodePayload(&p))
		assert.Equal(t, "SN-212-FLIP-001", p.SerialNumber)
	case <-time.After(time.Second):
		t.Fatal("expected device.online event after forced flip, got none")
	}

	// 4) 写了一条重启记录
	require.Len(t, rec.calls, 1, "BOOT 必须写一条重启记录")
	assert.Equal(t, "SN-212-FLIP-001", rec.calls[0].DeviceSN)
	assert.Equal(t, 5, rec.calls[0].BootCount)
}

// 对照（非 BOOT 不强制翻转）：纯 PERIODIC 心跳走 handlePeriodic，不应触发
// ForceBootStateFlip / device.offline，也不写重启记录 —— 强制翻转只属 BOOT 路径。
func TestHandlePeriodic_NoForcedFlipNoRebootRecord(t *testing.T) {
	deviceID := uuid.New()
	var forcedOfflineCalled bool
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID:             deviceID,
				SerialNumber:   sn,
				OUI:            "AABBCC",
				Carrier:        model.CarrierCMCC,
				Status:         model.DeviceActive,
				LifecycleState: model.LifecycleCommissioned,
				IsOnline:       true,
				InformInterval: 300,
			}, nil
		},
		updateOnlineFn: func(_ context.Context, _ uuid.UUID, isOnline bool) error {
			if !isOnline {
				forcedOfflineCalled = true
			}
			return nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}

	bus := event.NewChannelEventBus(16, zap.NewNop())
	offlineCh := make(chan event.Event, 1)
	_, subErr := bus.Subscribe(event.SubjectDeviceOffline, func(_ context.Context, evt event.Event) error {
		offlineCh <- evt
		return nil
	})
	require.NoError(t, subErr)

	rec := &infMockBootEventRecorder{}
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	svc.SetBootEventRecorder(rec)
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-212-PERIODIC-001")
	payload.Events = []string{"2 PERIODIC"}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)

	err = h.handlePeriodic(context.Background(), evt)
	require.NoError(t, err)

	assert.False(t, forcedOfflineCalled, "PERIODIC 不应触发强制置离线")
	assert.Empty(t, rec.calls, "PERIODIC 不应写重启记录")
	select {
	case <-offlineCh:
		t.Fatal("PERIODIC 不应发 device.offline 事件")
	case <-time.After(150 * time.Millisecond):
		// expected: no offline event
	}
}

func TestHandlePeriodic_DeletedDeviceSkipsAutoRegister(t *testing.T) {
	deletedID := uuid.New()
	createCalled := false
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil
		},
		getDeletedFn: func(_ context.Context, sn string, carrier model.CarrierCode) (*model.Device, error) {
			return &model.Device{ID: deletedID, SerialNumber: sn, Carrier: carrier}, nil
		},
		createFn: func(_ context.Context, _ *model.Device) error {
			createCalled = true
			return nil
		},
	}

	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())

	payload := sampleInformPayload("SN-RECYCLE-001")
	payload.Events = []string{"2 PERIODIC"}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)

	err = h.handlePeriodic(context.Background(), evt)
	require.NoError(t, err)
	assert.False(t, createCalled, "recycle-bin device must not be auto-registered")
}

func TestHandlePeriodic_AutoRegisterPublishFailureIsRetryable(t *testing.T) {
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return nil, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	require.NoError(t, bus.Close())
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	payload := sampleInformPayload("SN-PERIODIC-PUBLISH-FAIL")
	payload.Events = []string{"2 PERIODIC"}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)

	err = h.handlePeriodic(context.Background(), evt)

	require.ErrorIs(t, err, event.ErrBusClosed)
}

func TestHandlePeriodic_StaleCacheAutoRegisterPublishFailureIsRetryable(t *testing.T) {
	lookupCount := 0
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			lookupCount++
			if lookupCount == 1 {
				return &model.Device{ID: uuid.New(), SerialNumber: sn}, nil
			}
			return nil, nil
		},
	}
	bus := event.NewChannelEventBus(16, zap.NewNop())
	require.NoError(t, bus.Close())
	svc := NewDeviceService(deviceRepo, &infMockParamRepo{}, nil, bus, zap.NewNop())
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	payload := sampleInformPayload("SN-PERIODIC-STALE-PUBLISH-FAIL")
	payload.Events = []string{"2 PERIODIC"}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)

	err = h.handlePeriodic(context.Background(), evt)

	require.ErrorIs(t, err, event.ErrBusClosed)
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

type recordingAccessGate struct {
	observation AccessObservation
	decision    AccessDecision
	err         error
}

func (g *recordingAccessGate) Admit(_ context.Context, observation AccessObservation) (AccessDecision, error) {
	g.observation = observation
	return g.decision, g.err
}

func TestHandlePeriodic_ReevaluatesExistingDeviceWithoutFalsifyingHeartbeat(t *testing.T) {
	deviceID := uuid.New()
	updated := false
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID: deviceID, SerialNumber: sn, OUI: "48BF74", Carrier: model.CarrierCMCC,
				Status: model.DeviceActive, InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error {
			updated = true
			return nil
		},
	}
	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	gate := &recordingAccessGate{decision: AccessDecision{
		State: AccessDecisionRejected, ReasonCode: "rule_mismatch_confirmed",
	}}
	h.SetAccessGate(gate)
	payload := sampleInformPayload("SN-PER-ACCESS")
	payload.Events = []string{"2 PERIODIC"}
	payload.RemoteIP = "198.51.100.20"
	payload.Authenticated = true
	payload.AuthMethod = "digest"
	payload.CredentialID = "cpe"
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)

	err = h.handlePeriodic(context.Background(), evt)

	require.NoError(t, err)
	assert.True(t, updated, "access rejection must not rewrite the heartbeat as offline")
	assert.Equal(t, "SN-PER-ACCESS", gate.observation.SerialNumber)
	assert.Equal(t, model.CarrierCMCC, gate.observation.Carrier)
	assert.Equal(t, "198.51.100.20", gate.observation.RemoteIP)
	assert.True(t, gate.observation.Authenticated)
	assert.Equal(t, "digest", gate.observation.AuthMethod)
	assert.Equal(t, "cpe", gate.observation.CredentialID)
}

type recordingHeartbeatGroupAssigner struct {
	calls chan GroupAssignRequest
}

func (a *recordingHeartbeatGroupAssigner) AssignDeviceToGroup(
	_ context.Context,
	req GroupAssignRequest,
) error {
	a.calls <- req
	return nil
}

func TestHandlePeriodic_StableDeviceDoesNotRunGroupMatching(t *testing.T) {
	deviceID := uuid.New()
	deviceRepo := &infMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, sn string) (*model.Device, error) {
			return &model.Device{
				ID: deviceID, SerialNumber: sn, OUI: "AABBCC",
				Status: model.DeviceActive, InformInterval: 300,
			}, nil
		},
		updateFn: func(_ context.Context, _ *model.Device) error { return nil },
	}
	svc := newInfTestDeviceService(deviceRepo, &infMockParamRepo{})
	h := NewInformHandler(svc, nil, model.CarrierCMCC, zap.NewNop())
	assigner := &recordingHeartbeatGroupAssigner{calls: make(chan GroupAssignRequest, 1)}
	h.SetGroupAssigner(assigner)

	payload := sampleInformPayload("SN-PER-NO-GROUP")
	payload.Events = []string{"2 PERIODIC"}
	evt, err := event.NewEvent(event.SubjectDevicePeriodic, payload)
	require.NoError(t, err)
	require.NoError(t, h.handlePeriodic(context.Background(), evt))

	select {
	case req := <-assigner.calls:
		t.Fatalf("stable Periodic Inform must not run group matching: %+v", req)
	case <-time.After(50 * time.Millisecond):
	}
}
