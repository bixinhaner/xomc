package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: DeviceRepository
// ---------------------------------------------------------------------------

type mockDeviceRepo struct {
	createFn            func(ctx context.Context, device *model.Device) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	getBySerialNumberFn func(ctx context.Context, sn string) (*model.Device, error)
	updateFn            func(ctx context.Context, device *model.Device) error
	deleteFn            func(ctx context.Context, id uuid.UUID) error
	listFn              func(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
	updateLastInformFn  func(ctx context.Context, sn string, at time.Time, events []string) error
	countByStatusFn     func(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
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

func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	if m.updateLastInformFn != nil {
		return m.updateLastInformFn(ctx, sn, at, events)
	}
	return nil
}

func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx, carrier)
	}
	return map[model.DeviceStatus]int64{}, nil
}

// ---------------------------------------------------------------------------
// Mock: DeviceParameterRepository
// ---------------------------------------------------------------------------

type mockParamRepo struct {
	batchUpsertFn  func(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
	getByDeviceFn  func(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
	getByPathFn    func(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
	deleteByDevFn  func(ctx context.Context, deviceID uuid.UUID) error
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

	device, err := svc.RegisterFromInform(context.Background(), inform, model.CarrierCMCC)
	require.NoError(t, err)
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
	assert.NotNil(t, device.LastInformAt)
	assert.Equal(t, model.TechLTE, device.Technology)

	// Verify repo.Create was called
	assert.NotNil(t, createdDevice)

	// Verify parameters were stored
	assert.Len(t, upsertedParams, 3)
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

	device, err := svc.RegisterFromInform(context.Background(), inform, model.CarrierCMCC)
	require.NoError(t, err)
	require.NotNil(t, device)

	// Should have delegated to UpdateFromInform, which calls repo.Update
	assert.True(t, updateCalled, "expected Update to be called for existing device")
	assert.Equal(t, existingID, device.ID)
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
	assert.Equal(t, "1.0.0", updatedDevice.FirmwareVersion)
	assert.Equal(t, "http://192.168.1.1:7547", updatedDevice.ConnectionRequestURL)
	assert.Equal(t, "10.0.0.5", updatedDevice.IPAddress)
	assert.NotNil(t, updatedDevice.LastInformAt)

	// Auto-transition: offline -> active
	assert.Equal(t, model.DeviceActive, updatedDevice.Status)
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
		SiteName:     "SiteA",
		SiteID:       "SITE-001",
		Latitude:     31.23,
		Longitude:    121.47,
	}

	device, err := svc.CreateDevice(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, device)

	assert.Equal(t, "NEW-SN-100", device.SerialNumber)
	assert.Equal(t, model.DeviceRegistered, device.Status, "new API-created device should be in registered status")
	assert.Equal(t, model.CarrierCTCC, device.Carrier)
	assert.Equal(t, model.TechNR, device.Technology)
	assert.Equal(t, "SiteA", device.SiteName)
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
				SiteName:     "OldSite",
				Latitude:     0.0,
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
		SiteName: &newSite,
		Latitude: &newLat,
	}

	device, err := svc.UpdateDevice(context.Background(), deviceID, req)
	require.NoError(t, err)
	require.NotNil(t, device)

	assert.Equal(t, "NewSite", device.SiteName)
	assert.InDelta(t, 39.9, device.Latitude, 0.001)
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

// ---------------------------------------------------------------------------
// Tests: DeleteDevice
// ---------------------------------------------------------------------------

func TestDeviceService_DeleteDevice(t *testing.T) {
	deleteCalled := false
	targetID := uuid.New()

	deviceRepo := &mockDeviceRepo{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleteCalled = true
			assert.Equal(t, targetID, id)
			return nil
		},
	}

	svc := newTestDeviceService(deviceRepo, &mockParamRepo{})

	err := svc.DeleteDevice(context.Background(), targetID)
	require.NoError(t, err)
	assert.True(t, deleteCalled)
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectTechnology(tc.params)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: containsAny (package-level helper)
// ---------------------------------------------------------------------------

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		substrs []string
		want    bool
	}{
		{
			name:    "exact match",
			s:       "NR",
			substrs: []string{"NR"},
			want:    true,
		},
		{
			name:    "substring match",
			s:       "SmallCell-NR-SA",
			substrs: []string{"NR"},
			want:    true,
		},
		{
			name:    "no match",
			s:       "LTE device",
			substrs: []string{"NR", "5G", "gNB"},
			want:    false,
		},
		{
			name:    "empty string",
			s:       "",
			substrs: []string{"NR"},
			want:    false,
		},
		{
			name:    "empty substrs",
			s:       "anything",
			substrs: nil,
			want:    false,
		},
		{
			name:    "multiple substrs second matches",
			s:       "Indoor 5G pico",
			substrs: []string{"NR", "5G"},
			want:    true,
		},
		{
			name:    "match at end",
			s:       "model-gNB",
			substrs: []string{"gNB"},
			want:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := containsAny(tc.s, tc.substrs...)
			assert.Equal(t, tc.want, got)
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
