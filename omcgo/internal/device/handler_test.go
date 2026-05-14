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
	"github.com/omcgo/omcgo/internal/core/response"
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

func (m *fakeDeviceRepo) BatchDelete(ctx context.Context, ids []uuid.UUID, _ string) (int64, error) {
	var deleted int64
	for _, id := range ids {
		if d, ok := m.devices[id]; ok {
			delete(m.bySN, d.SerialNumber)
			delete(m.devices, id)
			deleted++
		}
	}
	return deleted, nil
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

func (m *fakeDeviceRepo) RecordBoot(ctx context.Context, sn string, at time.Time) (int, error) {
	return 0, nil
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
func (m *fakeDeviceRepo) ListActiveByLastInform(_ context.Context, _ *time.Time, _ *uuid.UUID, _ int) ([]model.Device, error) {
	return nil, nil
}
func (m *fakeDeviceRepo) ListGeo(_ context.Context, _ GeoDeviceFilter) ([]GeoDevice, int64, error) {
	return nil, 0, nil
}
func (m *fakeDeviceRepo) GetGeoStats(_ context.Context, _ []string) (*GeoStats, error) {
	return &GeoStats{}, nil
}
func (m *fakeDeviceRepo) SearchDevices(_ context.Context, _ string, _ int) ([]GeoDevice, error) {
	return nil, nil
}
func (m *fakeDeviceRepo) FindStaleDevices(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *fakeDeviceRepo) ListStaleForParamSync(_ context.Context, _ time.Time, _ int) ([]*model.Device, error) {
	return nil, nil
}
func (m *fakeDeviceRepo) UpdateLastParamSyncAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *fakeDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *fakeDeviceRepo) ListRecycleBin(_ context.Context, _ RecycleBinFilter) (*model.ListResponse[model.Device], error) {
	return model.NewListResponse([]model.Device{}, 0, 1, 20), nil
}
func (m *fakeDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *fakeDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *fakeDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
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

func (m *fakeParamRepo) GetByPathPrefix(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (m *fakeParamRepo) CountByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int, error) {
	return 0, nil
}

func (m *fakeParamRepo) SearchByKeyword(_ context.Context, _ uuid.UUID, _ string, _ int) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (m *fakeParamRepo) GetDirectChildLeaves(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (m *fakeParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *fakeParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
}

func (m *fakeParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
	return []model.DeviceParameter{}, nil
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
	response.DecodeData(t, w.Body, &resp)
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
	response.DecodeData(t, w.Body, &resp)
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
	response.DecodeData(t, w.Body, &resp)
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

	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)

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
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

// TestRebootDevice_NotFound_Returns404 verifies that POST /devices/:id/reboot
// returns 404 (not 500) when the device does not exist. This guards against
// the prior bug where service-layer ErrNotFound was mapped to 500 by the
// handler's hard-coded http.StatusInternalServerError.
func TestRebootDevice_NotFound_Returns404(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	nonExistentID := uuid.New()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+nonExistentID.String()+"/reboot", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRebootDevice_InvalidUUID_Returns400 verifies that an unparseable UUID
// path parameter still returns 400 Bad Request.
func TestRebootDevice_InvalidUUID_Returns400(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/not-a-uuid/reboot", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	response.DecodeData(t, w.Body, &resp)

	counts, ok := resp["counts"]
	require.True(t, ok, "response should contain 'counts' key")
	assert.Equal(t, float64(2), counts["active"])
	assert.Equal(t, float64(1), counts["offline"])
}

// ---------------------------------------------------------------------------
// Tests: T-0126 SyncDeviceParams（手动 Path B 同步端点）
// ---------------------------------------------------------------------------

// fakeParamSyncStarter 实现 ParamSyncStarter 接口供 SyncDeviceParams 单测使用。
type fakeParamSyncStarter struct {
	calls      []syncStarterCall
	defaultUsed bool
	defaultErr  error
}

type syncStarterCall struct {
	deviceID uuid.UUID
	sourceID string
}

func (f *fakeParamSyncStarter) StartManualSync(_ context.Context, dev *model.Device, sourceID string) (bool, error) {
	f.calls = append(f.calls, syncStarterCall{deviceID: dev.ID, sourceID: sourceID})
	return f.defaultUsed, f.defaultErr
}

func TestHandler_SyncDeviceParams_Success(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	starter := &fakeParamSyncStarter{defaultUsed: true}
	h.service.SetParamSyncStarter(starter)
	router := setupRouter(h)

	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-SYNC-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params",
		bytes.NewReader([]byte(`{"force": false}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "queued", resp["status"])
	assert.Equal(t, id.String(), resp["device_id"])
	assert.Equal(t, "SN-SYNC-001", resp["serial_number"])
	assert.Contains(t, resp["source_id"].(string), "manual:", "source_id 应以 manual: 前缀")

	require.Len(t, starter.calls, 1, "应调一次 StartManualSync")
	assert.Equal(t, id, starter.calls[0].deviceID)
	assert.Contains(t, starter.calls[0].sourceID, "manual:")
}

func TestHandler_SyncDeviceParams_NoBody_OK(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	h.service.SetParamSyncStarter(&fakeParamSyncStarter{defaultUsed: true})
	router := setupRouter(h)

	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-NOBODY", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code, "body 为空应仍返 202（force 字段容错）")
}

func TestHandler_SyncDeviceParams_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()
	h.service.SetParamSyncStarter(&fakeParamSyncStarter{defaultUsed: true})
	router := setupRouter(h)

	id := uuid.New() // 未 seed → 不存在
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SyncDeviceParams_InvalidUUID(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/not-a-uuid/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SyncDeviceParams_PathBUnavailable_503(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	// starter 返 used=false（MappingSet 缺失模拟）
	h.service.SetParamSyncStarter(&fakeParamSyncStarter{defaultUsed: false})
	router := setupRouter(h)

	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-NO-MAPPING", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "unavailable", resp["status"])
	assert.Contains(t, resp["message"].(string), "path-b sync unavailable")
}

func TestHandler_SyncDeviceParams_NilStarter_500(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	// 不注入 starter — h.service.paramSyncStarter == nil
	router := setupRouter(h)

	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-NIL-STARTER", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_SyncDeviceParams_RoutesOldEndpointGone(t *testing.T) {
	h, _, _ := newTestHandler()
	router := setupRouter(h)

	// 旧端点 /param-sync 已下线，应返 404
	id := uuid.New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/param-sync", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "旧 /param-sync 路由应已下线")
}
