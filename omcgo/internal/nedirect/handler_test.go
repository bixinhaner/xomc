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
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Test helpers ---

func setupHandler(
	existingDevices map[string]*model.Device,
	params map[uuid.UUID][]model.DeviceParameter,
) (*Handler, *mockEventBus) {
	deviceRepo := newMockDeviceRepo()
	if existingDevices != nil {
		deviceRepo.devices = existingDevices
	}

	paramRepo := newMockParamRepo()
	if params != nil {
		paramRepo.params = params
	}

	alarmStore := &mockAlarmStore{}
	alarmEngine := alarm.NewAlarmEngine(alarmStore, nil, nil, nil, zap.NewNop())
	eventBus := &mockEventBus{}
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())

	sessionRepo := &mockSessionRepo{}
	commandRepo := &mockCommandRepo{}

	svc := NewService(sessionRepo, commandRepo, deviceService, alarmEngine, eventBus, zap.NewNop())
	handler := NewHandler(svc, nil, zap.NewNop())

	return handler, eventBus
}

func setupHandlerWithSession(
	existingDevices map[string]*model.Device,
	sessionRepo *mockSessionRepo,
	commandRepo *mockCommandRepo,
) (*Handler, *mockEventBus) {
	deviceRepo := newMockDeviceRepo()
	if existingDevices != nil {
		deviceRepo.devices = existingDevices
	}

	paramRepo := newMockParamRepo()
	alarmStore := &mockAlarmStore{}
	alarmEngine := alarm.NewAlarmEngine(alarmStore, nil, nil, nil, zap.NewNop())
	eventBus := &mockEventBus{}
	deviceService := device.NewDeviceService(deviceRepo, paramRepo, nil, nil, zap.NewNop())

	svc := NewService(sessionRepo, commandRepo, deviceService, alarmEngine, eventBus, zap.NewNop())
	handler := NewHandler(svc, nil, zap.NewNop())

	return handler, eventBus
}

// --- Tests: HandleRegister ---

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

// --- Tests: HandleConfig ---

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

// --- Tests: HandleStatus ---

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

// --- Tests: HandleFault ---

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

// --- Tests: HandleConnect ---

func TestHandleConnect_Success(t *testing.T) {
	deviceID := uuid.New()
	devices := map[string]*model.Device{
		"SN-CONN": {
			ID:           deviceID,
			SerialNumber: "SN-CONN",
			Status:       model.DeviceActive,
			IPAddress:    "10.0.0.10",
		},
	}

	sessionRepo := &mockSessionRepo{}
	handler, eventBus := setupHandlerWithSession(devices, sessionRepo, &mockCommandRepo{})

	body := ConnectRequest{
		DeviceSN: "SN-CONN",
		UserID:   "user1",
		Username: "admin",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/connect", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleConnect(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Session
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "SN-CONN", resp.DeviceSN)
	assert.Equal(t, SessionActive, resp.Status)

	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectConnect, eventBus.published[0].Subject)
}

func TestHandleConnect_MissingFields(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	body := ConnectRequest{DeviceSN: "SN-001"} // missing user_id
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/connect", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleConnect(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Tests: HandleDisconnect ---

func TestHandleDisconnect_Success(t *testing.T) {
	sessionID := uuid.New()
	activeSession := &Session{
		ID:       sessionID,
		DeviceSN: "SN-001",
		Status:   SessionActive,
	}

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Session, error) {
			if id == sessionID {
				return activeSession, nil
			}
			return nil, commonerrors.ErrNotFound
		},
	}

	handler, _ := setupHandlerWithSession(nil, sessionRepo, &mockCommandRepo{})

	body := map[string]string{"session_id": sessionID.String()}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/disconnect", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleDisconnect(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleDisconnect_InvalidID(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	body := map[string]string{"session_id": "not-a-uuid"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/disconnect", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleDisconnect(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Tests: HandleCommand ---

func TestHandleCommand_Success(t *testing.T) {
	sessionID := uuid.New()
	activeSession := &Session{
		ID:       sessionID,
		DeviceSN: "SN-001",
		Status:   SessionActive,
	}

	sessionRepo := &mockSessionRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Session, error) {
			if id == sessionID {
				return activeSession, nil
			}
			return nil, commonerrors.ErrNotFound
		},
	}

	handler, eventBus := setupHandlerWithSession(nil, sessionRepo, &mockCommandRepo{})

	body := SendCommandRequest{
		SessionID: sessionID.String(),
		Command:   "DSP CELLINFO;",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/command", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleCommand(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Command
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "DSP CELLINFO;", resp.CommandStr)
	assert.Equal(t, CommandPending, resp.Status)

	require.Len(t, eventBus.published, 1)
	assert.Equal(t, event.SubjectNEDirectCommand, eventBus.published[0].Subject)
}

func TestHandleCommand_MissingCommand(t *testing.T) {
	handler, _ := setupHandler(nil, nil)

	body := SendCommandRequest{
		SessionID: uuid.New().String(),
		Command:   "", // empty
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/nedirect/command", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.HandleCommand(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Tests: HandleListSessions ---

func TestHandleListSessions_Success(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		listFn: func(ctx context.Context, filter SessionFilter) (*model.ListResponse[Session], error) {
			return model.NewListResponse([]Session{}, 0, 1, 20), nil
		},
	}

	handler, _ := setupHandlerWithSession(nil, sessionRepo, &mockCommandRepo{})

	req := httptest.NewRequest(http.MethodGet, "/nedirect/sessions", nil)
	w := httptest.NewRecorder()

	handler.HandleListSessions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Tests: RegisterRoutes ---

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
		{http.MethodPost, "/nedirect/connect"},
		{http.MethodPost, "/nedirect/disconnect"},
		{http.MethodPost, "/nedirect/command"},
		{http.MethodGet, "/nedirect/sessions"},
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
