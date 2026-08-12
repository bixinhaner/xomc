package device

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
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
	devices           map[uuid.UUID]*model.Device
	bySN              map[string]*model.Device
	lastListFilter    DeviceFilter
	lastRecycleFilter RecycleBinFilter

	// #64 GIS 数据权限测试捕获位：记录最近一次 geo 查询收到的可见分组。
	geoListVisible   []uuid.UUID
	geoStatsVisible  []uuid.UUID
	geoSearchVisible []uuid.UUID
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
func (m *fakeDeviceRepo) GetDeletedBySerialNumber(_ context.Context, _ string, _ model.CarrierCode) (*model.Device, error) {
	return nil, nil
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
	m.lastListFilter = filter
	items := make([]model.Device, 0, len(m.devices))
	for _, d := range m.devices {
		if filter.Carrier != nil && d.Carrier != *filter.Carrier {
			continue
		}
		if filter.Technology != nil && d.Technology != *filter.Technology {
			continue
		}
		if len(filter.Technologies) > 0 {
			matched := false
			for _, tech := range filter.Technologies {
				if d.Technology == tech {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		items = append(items, *d)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

// T-0162 新接口方法
func (m *fakeDeviceRepo) UpdateLifecycle(_ context.Context, id uuid.UUID, lifecycle model.DeviceLifecycle) error {
	if d, ok := m.devices[id]; ok {
		d.LifecycleState = lifecycle
	}
	return nil
}

func (m *fakeDeviceRepo) UpdateOnlineStatus(_ context.Context, id uuid.UUID, isOnline bool) error {
	if d, ok := m.devices[id]; ok {
		d.IsOnline = isOnline
	}
	return nil
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
func (m *fakeDeviceRepo) ListGeo(_ context.Context, f GeoDeviceFilter) ([]GeoDevice, int64, error) {
	m.geoListVisible = f.VisibleGroups
	return nil, 0, nil
}
func (m *fakeDeviceRepo) GetGeoStats(_ context.Context, filter GeoStatsFilter) (*GeoStats, error) {
	m.geoStatsVisible = filter.VisibleGroups
	return &GeoStats{}, nil
}
func (m *fakeDeviceRepo) SearchDevices(_ context.Context, _ string, _ int, visibleGrants []model.DeviceVisibilityGrant) ([]GeoDevice, error) {
	if visibleGrants == nil {
		m.geoSearchVisible = nil
		return nil, nil
	}
	m.geoSearchVisible = []uuid.UUID{}
	for _, grant := range visibleGrants {
		m.geoSearchVisible = append(m.geoSearchVisible, grant.GroupIDs...)
	}
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

func (m *fakeDeviceRepo) UpdateLastParamSyncFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

func (m *fakeDeviceRepo) UpdateSiteName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *fakeDeviceRepo) ListSerialsByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}

func (m *fakeDeviceRepo) ListRecycleBin(_ context.Context, filter RecycleBinFilter) (*model.ListResponse[DeviceWithInfo], error) {
	m.lastRecycleFilter = filter
	items := make([]DeviceWithInfo, 0, len(m.devices))
	for _, d := range m.devices {
		if filter.Carrier != nil && d.Carrier != *filter.Carrier {
			continue
		}
		if filter.Technology != nil && d.Technology != *filter.Technology {
			continue
		}
		items = append(items, DeviceWithInfo{Device: *d})
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}
func (m *fakeDeviceRepo) RestoreDevices(_ context.Context, _ []uuid.UUID) (*RestoreResult, error) {
	return &RestoreResult{}, nil
}
func (m *fakeDeviceRepo) PermanentDelete(_ context.Context, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *fakeDeviceRepo) ListProductClasses(_ context.Context) ([]string, error) {
	return nil, nil
}

func TestGeoPaginationState(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		itemCount    int
		total        int64
		wantHasMore  bool
		wantComplete bool
	}{
		{
			name:         "第一页覆盖全部坐标",
			page:         1,
			pageSize:     500,
			itemCount:    200,
			total:        200,
			wantComplete: true,
		},
		{
			name:        "第一页仍有后续分页",
			page:        1,
			pageSize:    500,
			itemCount:   500,
			total:       750,
			wantHasMore: true,
		},
		{
			name:         "尾页不能冒充完整集合",
			page:         2,
			pageSize:     500,
			itemCount:    250,
			total:        750,
			wantHasMore:  false,
			wantComplete: false,
		},
		{
			name:         "空第一页是完整空集合",
			page:         1,
			pageSize:     500,
			itemCount:    0,
			total:        0,
			wantComplete: true,
		},
		{
			name:         "仓库条数与总数不一致时保守标记不完整",
			page:         1,
			pageSize:     500,
			itemCount:    199,
			total:        200,
			wantComplete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMore, complete := geoPaginationState(tt.page, tt.pageSize, tt.itemCount, tt.total)
			assert.Equal(t, tt.wantHasMore, hasMore)
			assert.Equal(t, tt.wantComplete, complete)
		})
	}
}

// ---------------------------------------------------------------------------

type fakeParamRepo struct {
	params      map[uuid.UUID][]model.DeviceParameter
	groupParams map[uuid.UUID]map[string][]model.DeviceParameter
	lastGroup   string
}

func newFakeParamRepo() *fakeParamRepo {
	return &fakeParamRepo{
		params:      make(map[uuid.UUID][]model.DeviceParameter),
		groupParams: make(map[uuid.UUID]map[string][]model.DeviceParameter),
	}
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

func (m *fakeParamRepo) DeleteByPathPrefix(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
	return 0, nil
}

func (m *fakeParamRepo) GetByPathPrefix(_ context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error) {
	if prefix == "" {
		out := make([]model.DeviceParameter, len(m.params[deviceID]))
		copy(out, m.params[deviceID])
		return out, nil
	}
	var out []model.DeviceParameter
	for _, p := range m.params[deviceID] {
		if strings.HasPrefix(p.ParameterPath, prefix) {
			out = append(out, p)
		}
	}
	return out, nil
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

func (m *fakeParamRepo) GetByGroup(_ context.Context, deviceID uuid.UUID, group string) ([]model.DeviceParameter, error) {
	m.lastGroup = group
	return m.groupParams[deviceID][group], nil
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
	isOnline := status == model.DeviceActive
	d := &model.Device{
		ID:             id,
		SerialNumber:   sn,
		OUI:            "AABBCC",
		Manufacturer:   "TestVendor",
		ModelName:      "PicoCell-100",
		Carrier:        carrier,
		Technology:     tech,
		LifecycleState: model.LifecycleCommissioned,
		IsOnline:       isOnline,
		Status:         status,
		DeviceName:     "Site-A",
		SiteID:         "SITE-001",
		Latitude:       f64p(39.9042),
		Longitude:      f64p(116.4074),
		CreatedAt:      now,
		UpdatedAt:      now,
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
		DeviceName:   "Site-A",
		SiteID:       "SITE-001",
		Latitude:     f64p(39.9042),
		Longitude:    f64p(116.4074),
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
		DeviceName: &newSite,
		Latitude:   &newLat,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/devices/"+deviceID.String(), bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.Device
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "Site-B", resp.DeviceName)
	if assert.NotNil(t, resp.Latitude) {
		assert.Equal(t, 31.2304, *resp.Latitude)
	}
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

func TestHandler_ListDevices_TechnologyNormalization(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	seedDevice(deviceRepo, uuid.New(), "SN-LIST-GSM-001", model.CarrierCMCC, model.TechGSM, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-LIST-LTE-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices?page=1&page_size=20&technology=GSM", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[DeviceWithInfo]
	response.DecodeData(t, w.Body, &resp)
	if assert.Len(t, resp.Items, 1) {
		assert.Equal(t, model.TechGSM, resp.Items[0].Technology)
	}
	if assert.NotNil(t, deviceRepo.lastListFilter.Technology) {
		assert.Equal(t, model.TechGSM, *deviceRepo.lastListFilter.Technology)
	}
}

func TestHandler_ListRecycleBin_TechnologyNormalization(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)

	seedDevice(deviceRepo, uuid.New(), "SN-RECYCLE-GSM-001", model.CarrierCMCC, model.TechGSM, model.DeviceActive)
	seedDevice(deviceRepo, uuid.New(), "SN-RECYCLE-LTE-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/recycle?page=1&page_size=20&technology=GSM", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[DeviceWithInfo]
	response.DecodeData(t, w.Body, &resp)
	if assert.Len(t, resp.Items, 1) {
		assert.Equal(t, model.TechGSM, resp.Items[0].Technology)
	}
	if assert.NotNil(t, deviceRepo.lastRecycleFilter.Technology) {
		assert.Equal(t, model.TechGSM, *deviceRepo.lastRecycleFilter.Technology)
	}
}

// f64p returns a pointer to v (model.Device.Latitude/Longitude are *float64).
func f64p(v float64) *float64 { return &v }

// TestHandler_ListDevices_ProductIDFilter verifies the new product_id query
// param is accepted (valid UUID → 200) and rejected when malformed (→ 400).
func TestHandler_ListDevices_ProductIDFilter(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	router := setupRouter(h)
	seedDevice(deviceRepo, uuid.New(), "SN-PID-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	// Valid product_id UUID → 200.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/devices?page=1&page_size=20&product_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Malformed product_id → 400.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/devices?page=1&page_size=20&product_id=not-a-uuid", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	calls               []syncStarterCall
	defaultUsed         bool
	defaultGPVTaskCount int
	defaultErr          error
}

type syncStarterCall struct {
	deviceID       uuid.UUID
	sourceID       string
	parameterPaths []string
}

type fakeDetailedParamSyncStarter struct {
	*fakeParamSyncStarter
	result *ManualParamSyncStart
}

type recordingConnectionRequester struct {
	called chan struct{}
}

func (r *recordingConnectionRequester) Send(context.Context, string, string) error {
	select {
	case r.called <- struct{}{}:
	default:
	}
	return nil
}

func (f *fakeDetailedParamSyncStarter) StartManualSyncDetailed(context.Context, *model.Device, string, []string) (*ManualParamSyncStart, error) {
	return f.result, nil
}

func (f *fakeParamSyncStarter) StartManualSync(_ context.Context, dev *model.Device, sourceID string, parameterPaths []string) (bool, int, error) {
	f.calls = append(f.calls, syncStarterCall{deviceID: dev.ID, sourceID: sourceID, parameterPaths: parameterPaths})
	return f.defaultUsed, f.defaultGPVTaskCount, f.defaultErr
}

func (f *fakeParamSyncStarter) StartManualSyncDetailed(_ context.Context, dev *model.Device, sourceID string, parameterPaths []string) (*ManualParamSyncStart, error) {
	f.calls = append(f.calls, syncStarterCall{deviceID: dev.ID, sourceID: sourceID, parameterPaths: parameterPaths})
	return &ManualParamSyncStart{Used: f.defaultUsed, TaskCount: f.defaultGPVTaskCount, Status: "queued"}, f.defaultErr
}

func TestHandler_SyncDeviceParams_Success(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	starter := &fakeParamSyncStarter{defaultUsed: true, defaultGPVTaskCount: 7}
	h.service.SetParamSyncStarter(starter)
	connReq := &recordingConnectionRequester{called: make(chan struct{}, 1)}
	h.service.SetConnectionRequester(connReq)
	router := setupRouter(h)

	id := uuid.New()
	dev := seedDevice(deviceRepo, id, "SN-SYNC-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	dev.ConnectionRequestURL = "http://192.0.2.10:7547"

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params",
		bytes.NewReader([]byte(`{"force": false, "parameter_paths": ["Device.DeviceInfo.SoftwareVersion"]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "queued", resp["status"])
	assert.Equal(t, id.String(), resp["device_id"])
	assert.Equal(t, "SN-SYNC-001", resp["serial_number"])
	assert.EqualValues(t, 7, resp["gpv_task_count"])
	assert.Contains(t, resp["source_id"].(string), "manual:", "source_id 应以 manual: 前缀")
	_, hasRequestID := resp["request_id"]
	assert.False(t, hasRequestID, "legacy sync must not expose a zero durable request id")
	_, hasRunID := resp["run_id"]
	assert.False(t, hasRunID, "legacy sync must not expose a zero durable run id")

	require.Len(t, starter.calls, 1, "应调一次 StartManualSync")
	assert.Equal(t, id, starter.calls[0].deviceID)
	// 传给 service 的 sourceID 是裸 UUID（写入 device_tasks.source_id UUID 列），
	// 响应里的 source_id 才是 manual:<uuid> display 形式
	_, parseErr := uuid.Parse(starter.calls[0].sourceID)
	assert.NoError(t, parseErr, "传给 service 的 sourceID 必须是合法 UUID")
	assert.Equal(t, []string{"Device.DeviceInfo.SoftwareVersion"}, starter.calls[0].parameterPaths)
	select {
	case <-connReq.called:
		t.Fatal("manual endpoint sent a duplicate device wake; durable task outbox owns the single wake")
	case <-time.After(100 * time.Millisecond):
	}
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

func TestHandler_SyncDeviceParams_DurableIDsAreReturned(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	requestID, runID := uuid.New(), uuid.New()
	h.service.SetParamSyncStarter(&fakeDetailedParamSyncStarter{
		fakeParamSyncStarter: &fakeParamSyncStarter{},
		result: &ManualParamSyncStart{
			Used: true, TaskCount: 2, RequestID: requestID, RunID: &runID, Status: "running",
		},
	})
	router := setupRouter(h)
	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-DURABLE", model.CarrierCMCC, model.TechLTE, model.DeviceActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]interface{}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, requestID.String(), resp["request_id"])
	assert.Equal(t, runID.String(), resp["run_id"])
}

func TestHandler_SyncDeviceParams_OfflineLegacyRejectedBeforeStart(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	starter := &fakeParamSyncStarter{defaultUsed: true, defaultGPVTaskCount: 5}
	h.service.SetParamSyncStarter(starter)
	router := setupRouter(h)
	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-OFFLINE-SYNC", model.CarrierCMCC, model.TechLTE, model.DeviceOffline)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.EqualValues(t, global.ErrCodeDeviceOffline, resp["biz_code"])
	assert.Contains(t, resp["msg"].(string), "offline")
	assert.Empty(t, starter.calls, "offline device must not create a parameter sync request/run")
}

func TestHandler_SyncDeviceParams_OfflineDurableQueuedWhenConfigured(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	requestID := uuid.New()
	h.service.SetParamSyncManualOfflineMode("queue")
	h.service.SetParamSyncStarter(&fakeDetailedParamSyncStarter{
		fakeParamSyncStarter: &fakeParamSyncStarter{},
		result: &ManualParamSyncStart{
			Used: true, RequestID: requestID, Status: "queued", ResultCode: "DEVICE_OFFLINE",
		},
	})
	router := setupRouter(h)
	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-OFFLINE-DURABLE", model.CarrierCMCC, model.TechLTE, model.DeviceOffline)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]interface{}
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "queued", resp["status"])
	assert.Equal(t, requestID.String(), resp["request_id"])
}

func TestHandler_SyncDeviceParams_OfflineDurableRejectedByDefault(t *testing.T) {
	h, deviceRepo, _ := newTestHandler()
	requestID := uuid.New()
	h.service.SetParamSyncStarter(&fakeDetailedParamSyncStarter{
		fakeParamSyncStarter: &fakeParamSyncStarter{},
		result:               &ManualParamSyncStart{Used: true, RequestID: requestID, Status: "queued"},
	})
	router := setupRouter(h)
	id := uuid.New()
	seedDevice(deviceRepo, id, "SN-OFFLINE-REJECT", model.CarrierCMCC, model.TechLTE, model.DeviceOffline)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id.String()+"/sync-params", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.EqualValues(t, global.ErrCodeDeviceOffline, resp["biz_code"])
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
