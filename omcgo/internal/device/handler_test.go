package device

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type fakeDeviceRepo struct {
	devices map[uuid.UUID]*model.Device
	bySN    map[string]*model.Device
}

func newFakeDeviceRepo() *fakeDeviceRepo {
	return &fakeDeviceRepo{
		devices: make(map[uuid.UUID]*model.Device),
		bySN:    make(map[string]*model.Device),
	}
}

func (m *fakeDeviceRepo) Create(ctx context.Context, d *model.Device) error {
	m.devices[d.ID] = d
	m.bySN[d.SerialNumber] = d
	return nil
}

func (m *fakeDeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	d, ok := m.devices[id]
	if !ok {
		return nil, nil
	}
	return d, nil
}

func (m *fakeDeviceRepo) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	d, ok := m.bySN[sn]
	if !ok {
		return nil, nil
	}
	return d, nil
}

func (m *fakeDeviceRepo) Update(ctx context.Context, d *model.Device) error {
	m.devices[d.ID] = d
	m.bySN[d.SerialNumber] = d
	return nil
}

func (m *fakeDeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if d, ok := m.devices[id]; ok {
		delete(m.bySN, d.SerialNumber)
		delete(m.devices, id)
	}
	return nil
}

func (m *fakeDeviceRepo) List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error) {
	items := make([]model.Device, 0, len(m.devices))
	for _, d := range m.devices {
		items = append(items, *d)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

func (m *fakeDeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error {
	if d, ok := m.devices[id]; ok {
		d.Status = status
	}
	return nil
}

func (m *fakeDeviceRepo) UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error {
	return nil
}

func (m *fakeDeviceRepo) CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error) {
	counts := make(map[model.DeviceStatus]int64)
	for _, d := range m.devices {
		if carrier != nil && d.Carrier != *carrier {
			continue
		}
		counts[d.Status]++
	}
	return counts, nil
}

// ---------------------------------------------------------------------------

type fakeParamRepo struct {
	params map[uuid.UUID][]model.DeviceParameter
}

func newFakeParamRepo() *fakeParamRepo {
	return &fakeParamRepo{params: make(map[uuid.UUID][]model.DeviceParameter)}
}

func (m *fakeParamRepo) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
	m.params[deviceID] = params
	return nil
}

func (m *fakeParamRepo) GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	return m.params[deviceID], nil
}

func (m *fakeParamRepo) GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	for _, p := range m.params[deviceID] {
		if p.ParameterPath == path {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *fakeParamRepo) DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error {
	delete(m.params, deviceID)
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newTestHandler() (*Handler, *fakeDeviceRepo, *fakeParamRepo) {
	deviceRepo := newFakeDeviceRepo()
	paramRepo := newFakeParamRepo()
	logger := zap.NewNop()
	svc := NewDeviceService(deviceRepo, paramRepo, nil, nil, logger)
	h := NewHandler(svc)
	return h, deviceRepo, paramRepo
}

func seedDevice(repo *fakeDeviceRepo, id uuid.UUID, sn string, carrier model.CarrierCode, tech model.Technology, status model.DeviceStatus) *model.Device {
	now := time.Now()
	d := &model.Device{
		ID:           id,
		SerialNumber: sn,
		OUI:          "AABBCC",
		Manufacturer: "TestVendor",
		ModelName:    "PicoCell-100",
		Carrier:      carrier,
		Technology:   tech,
		Status:       status,
		SiteName:     "Site-A",
		SiteID:       "SITE-001",
		Latitude:     39.9042,
		Longitude:    116.4074,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.devices[id] = d
	repo.bySN[sn] = d
	return d
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_CreateDevice_Success(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	body := CreateDeviceRequest{
		SerialNumber: "SN-CREATE-001",
		OUI:          "AABBCC",
		ProductClass: "PicoCell",
		Manufacturer: "TestVendor",
		ModelName:    "PicoCell-100",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		SiteName:     "Site-A",
		SiteID:       "SITE-001",
		Latitude:     39.9042,
		Longitude:    116.4074,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp model.Device
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "SN-CREATE-001", resp.SerialNumber)
	assert.Equal(t, "AABBCC", resp.OUI)
	assert.Equal(t, model.CarrierCMCC, resp.Carrier)
	assert.Equal(t, model.TechLTE, resp.Technology)
	assert.Equal(t, model.DeviceRegistered, resp.Status)
	assert.NotEqual(t, uuid.Nil, resp.ID)
}

func TestHandler_CreateDevice_BadRequest(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	// Missing required fields (serial_number, oui, carrier, technology).
	body := map[string]string{
		"model_name": "PicoCell-100",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDevice_Duplicate(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	// Pre-seed a device with the same serial number.
	seedDevice(deviceRepo, uuid.New(), "SN-DUP-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	body := CreateDeviceRequest{
		SerialNumber: "SN-DUP-001",
		OUI:          "AABBCC",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_GetDevice_Found(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	deviceID := uuid.New()
	seedDevice(deviceRepo, deviceID, "SN-GET-001", model.CarrierCTCC, model.TechNR, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.Device
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, deviceID, resp.ID)
	assert.Equal(t, "SN-GET-001", resp.SerialNumber)
	assert.Equal(t, model.CarrierCTCC, resp.Carrier)
	assert.Equal(t, model.TechNR, resp.Technology)
}

func TestHandler_GetDevice_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	nonExistentID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+nonExistentID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetDevice_InvalidUUID(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/invalid-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateDevice_Success(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	deviceID := uuid.New()
	seedDevice(deviceRepo, deviceID, "SN-UPD-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	newSite := "Site-B"
	newLat := 31.2304
	body := UpdateDeviceRequest{
		SiteName: &newSite,
		Latitude: &newLat,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/devices/"+deviceID.String(), bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.Device
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Site-B", resp.SiteName)
	assert.Equal(t, 31.2304, resp.Latitude)
	// Unchanged fields should remain the same.
	assert.Equal(t, "SN-UPD-001", resp.SerialNumber)
	assert.Equal(t, model.DeviceActive, resp.Status)
}

func TestHandler_DeleteDevice_Success(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	deviceID := uuid.New()
	seedDevice(deviceRepo, deviceID, "SN-DEL-001", model.CarrierCUCC, model.TechLTE, model.DeviceOffline)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/"+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	// Verify device was removed from the repo.
	_, exists := deviceRepo.devices[deviceID]
	assert.False(t, exists)
}

func TestHandler_ListDevices(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	// Seed two devices.
	seedDevice(deviceRepo, uuid.New(), "SN-LIST-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-LIST-002", model.CarrierCMCC, model.TechNR, model.DeviceRegistered)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[model.Device]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

func TestHandler_GetStats(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	// Seed devices with different statuses.
	seedDevice(deviceRepo, uuid.New(), "SN-STAT-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-STAT-002", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-STAT-003", model.CarrierCMCC, model.TechNR, model.DeviceOffline)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]map[string]float64
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	counts, ok := resp["counts"]
	require.True(t, ok, "response should contain 'counts' key")
	assert.Equal(t, float64(2), counts["active"])
	assert.Equal(t, float64(1), counts["offline"])
}
