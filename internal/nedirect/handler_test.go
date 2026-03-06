package nedirect

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/common/event"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/omcr/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type mockDeviceRepo struct {
	devices map[string]*model.Device
}

func newMockDeviceRepo() *mockDeviceRepo {
	return &mockDeviceRepo{devices: make(map[string]*model.Device)}
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	m.devices[d.SerialNumber] = d
	return nil
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	for _, d := range m.devices {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}
func (m *mockDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	d, ok := m.devices[sn]
	if !ok {
		return nil, nil
	}
	return d, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, d *model.Device) error   { return nil }
func (m *mockDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (m *mockDeviceRepo) List(ctx context.Context, filter device.DeviceFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	return nil
}
func (m *mockDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	return nil
}
func (m *mockDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	return nil, nil
}

type mockParamRepo struct {
	params map[uuid.UUID][]model.DeviceParameter
}

func newMockParamRepo() *mockParamRepo {
	return &mockParamRepo{params: make(map[uuid.UUID][]model.DeviceParameter)}
}

func (m *mockParamRepo) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	m.params[deviceID] = params
	return nil
}
func (m *mockParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return m.params[deviceID], nil
}
func (m *mockParamRepo) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	return nil, nil
}
func (m *mockParamRepo) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error { return nil }

type mockAlarmStore struct {
	alarms []*model.Alarm
}

func (m *mockAlarmStore) SaveActive(ctx context.Context, a *model.Alarm) error {
	m.alarms = append(m.alarms, a)
	return nil
}
func (m *mockAlarmStore) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) GetActiveByDeviceAndCode(ctx context.Context, deviceSN, alarmCode string) (*model.Alarm, error) {
	return nil, nil
}
func (m *mockAlarmStore) UpdateActive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) RemoveActive(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *mockAlarmStore) ListActive(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse([]model.Alarm{}, 0, 1, 20), nil
}
func (m *mockAlarmStore) Archive(ctx context.Context, a *model.Alarm) error { return nil }
func (m *mockAlarmStore) ListHistory(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	return model.NewListResponse([]model.Alarm{}, 0, 1, 20), nil
}
func (m *mockAlarmStore) Statistics(ctx context.Context, filter alarm.AlarmFilter) (*alarm.AlarmStatistics, error) {
	return &alarm.AlarmStatistics{}, nil
}

type mockEventBus struct {
	published []event.Event
}

func (m *mockEventBus) Publish(ctx context.Context, subject string, e event.Event) error {
	m.published = append(m.published, e)
	return nil
}
func (m *mockEventBus) Subscribe(subject string, handler event.EventHandler) (event.Subscription, error) {
	return &mockSubscription{}, nil
}
func (m *mockEventBus) QueueSubscribe(subject, queue string, handler event.EventHandler) (event.Subscription, error) {
	return &mockSubscription{}, nil
}
func (m *mockEventBus) Close() error { return nil }

type mockSubscription struct{}

func (m *mockSubscription) Unsubscribe() error { return nil }

// --- Test helpers ---

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func setupHandler(existingDevices map[string]*model.Device, params map[uuid.UUID][]model.DeviceParameter) (*Handler, *mockEventBus) {
	deviceRepo := newMockDeviceRepo()
	if existingDevices != nil {
		deviceRepo.devices = existingDevices
	}

	paramRepo := newMockParamRepo()
	if params != nil {
		paramRepo.params = params
	}

	alarmStore := &mockAlarmStore{}
	alarmEngine := alarm.NewAlarmEngine(alarmStore, nil, nil, nil, testLogger())
	eventBus := &mockEventBus{}

	deviceService := device.NewDeviceService(deviceRepo, paramRepo, nil, testLogger())
	handler := NewHandler(deviceService, alarmEngine, eventBus, testLogger())

	return handler, eventBus
}

// --- Tests ---

func TestHandleRegister_NewDevice(t *testing.T) {
	handler, eventBus := setupHandler(nil, nil)

	body := RegisterRequest{
		SerialNumber: "SN-NEW-001",
		IPAddress:    "10.0.0.1",
		Manufacturer: "TestVendor",
		ModelName:    "LTE-Pico",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleRegister(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "registration accepted", resp["message"])

	// Event should have been published
	assert.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectRegister, eventBus.published[0].Subject)
}

func TestHandleRegister_ExistingDevice(t *testing.T) {
	deviceID := uuid.New()
	existingDevices := map[string]*model.Device{
		"SN-EXISTING": {
			ID:           deviceID,
			SerialNumber: "SN-EXISTING",
			Status:       model.DeviceActive,
		},
	}

	handler, eventBus := setupHandler(existingDevices, nil)

	body := RegisterRequest{SerialNumber: "SN-EXISTING"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/register", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleRegister(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "device already registered", resp["message"])
	assert.Equal(t, deviceID.String(), resp["device_id"])

	// No event published for existing device
	assert.Empty(t, eventBus.published)
}

func TestHandleRegister_MissingSerialNumber(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	body := RegisterRequest{IPAddress: "10.0.0.1"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/register", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleRegister(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRegister_WrongMethod(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/register", nil)
	w := httptest.NewRecorder()

	handler.HandleRegister(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleConfig_Success(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-CFG": {
			ID:           deviceID,
			SerialNumber: "SN-CFG",
			Status:       model.DeviceActive,
		},
	}
	params := map[uuid.UUID][]model.DeviceParameter{
		deviceID: {
			{
				DeviceID:       deviceID,
				ParameterPath:  "Device.DeviceInfo.Manufacturer",
				ParameterValue: "TestVendor",
				ParameterType:  model.ParamString,
			},
		},
	}

	handler, _ := setupHandler(devices, params)

	body := ConfigRequest{SerialNumber: "SN-CFG"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/config", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, deviceID.String(), resp["device_id"])
	assert.Equal(t, float64(1), resp["total"])
}

func TestHandleConfig_DeviceNotFound(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	body := ConfigRequest{SerialNumber: "NONEXISTENT"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/config", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleConfig(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleStatus_Success(t *testing.T) {
	deviceID := uuid.New()
	now := time.Now()
	devices := map[string]*model.Device{
		"SN-STATUS": {
			ID:              deviceID,
			SerialNumber:    "SN-STATUS",
			Status:          model.DeviceActive,
			FirmwareVersion: "v1.2.3",
			IPAddress:       "10.0.0.5",
			LastInformAt:    &now,
		},
	}

	handler, _ := setupHandler(devices, nil)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/status?serial_number=SN-STATUS", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "SN-STATUS", resp["serial_number"])
	assert.Equal(t, "active", resp["status"])
	assert.Equal(t, "v1.2.3", resp["firmware_version"])
}

func TestHandleStatus_MissingSerialNumber(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/status", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleStatus_DeviceNotFound(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/status?serial_number=UNKNOWN", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleFault_Success(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-FAULT": {
			ID:           deviceID,
			SerialNumber: "SN-FAULT",
			Status:       model.DeviceActive,
		},
	}

	handler, eventBus := setupHandler(devices, nil)

	body := FaultReport{
		SerialNumber: "SN-FAULT",
		AlarmCode:    "TEMP_HIGH",
		Severity:     1,
		Description:  "Temperature exceeded threshold",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/fault", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleFault(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "fault reported", resp["message"])

	// NEDirect fault event should have been published
	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectFault, eventBus.published[0].Subject)
}

func TestHandleFault_MissingFields(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	body := FaultReport{SerialNumber: "SN-FAULT"} // missing alarm_code
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/fault", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleFault(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleFault_WrongMethod(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/nedirect/fault", nil)
	w := httptest.NewRecorder()

	handler.HandleFault(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestRegisterRoutes(t *testing.T) {
	handler, _ := setupHandler(nil, nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Verify all routes are registered by making requests
	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/nedirect/register"},
		{http.MethodPost, "/nedirect/config"},
		{http.MethodGet, "/nedirect/status?serial_number=test"},
		{http.MethodPost, "/nedirect/fault"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, bytes.NewReader([]byte("{}")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Our handlers always set Content-Type: application/json.
		// Go's default 404 handler does not, so this verifies the route is registered.
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"),
			"handler not invoked for: %s %s", ep.method, ep.path)
	}
}
