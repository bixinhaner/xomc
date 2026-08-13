package quicksettings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
)

// ── mock fixtures ───────────────────────────────────────────────

type mockDeviceLookup struct {
	device *model.Device
	err    error
}

func (m *mockDeviceLookup) GetDevice(_ context.Context, _ uuid.UUID) (*model.Device, error) {
	return m.device, m.err
}

type mockProductMatcher struct {
	result *product.MatchResult
	err    error
}

func (m *mockProductMatcher) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	return m.result, m.err
}

type mockPMNameLookup struct {
	name string
	err  error
}

func (m *mockPMNameLookup) LookupParamModelNameByID(_ context.Context, _ uuid.UUID) (string, error) {
	return m.name, m.err
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r
}

func TestEnrichGroupsWithMappingsUsesProductEnumWireValues(t *testing.T) {
	values := "25,50,75,100"
	labels := "5MHz,10MHz,15MHz,20MHz"
	groups := []Group{{ID: "enb-cell", Params: []Param{{
		Name: "DLBandWidth", Type: "enum",
		StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth",
		EnumOptions:  []EnumOption{{Value: "n50", Label: "10MHz"}},
	}}}}

	got := enrichGroupsWithMappings(groups, []parammodel.ParamMapping{{
		StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth",
		EntryType:    "parameter", Access: "READ_WRITE", DataType: "INT", IsSupported: true,
		EnumValues: &values, EnumLabels: &labels,
	}})

	require.Len(t, got, 1)
	require.Len(t, got[0].Params, 1)
	assert.Equal(t, "int", got[0].Params[0].Type)
	assert.Equal(t, []EnumOption{
		{Value: "25", Label: "5MHz"}, {Value: "50", Label: "10MHz"},
		{Value: "75", Label: "15MHz"}, {Value: "100", Label: "20MHz"},
	}, got[0].Params[0].EnumOptions)
}

// makeMatchResult 构造一个含有 ParamModelID 的 MatchResult fixture。
func makeMatchResult(t *testing.T, productClass string, paramModelID *uuid.UUID) *product.MatchResult {
	t.Helper()
	return &product.MatchResult{
		Product: &product.Product{
			ID:           uuid.New(),
			Name:         "TestProduct",
			ParamModelID: paramModelID,
		},
		MatchedPattern: productClass,
	}
}

// ── tests ───────────────────────────────────────────────

func TestHandler_GetGroups_HappyPath(t *testing.T) {
	pmID := uuid.New()
	deviceID := uuid.New()

	reg := NewRegistry()
	reg.Replace("BLQ", []Group{
		{ID: "enb-cell", TitleZh: "小区参数", Params: []Param{{Name: "ECI", StandardPath: "Device.X.CellIdentity"}}},
	})

	h := NewHandler(
		reg,
		&mockDeviceLookup{device: &model.Device{ProductClass: "FAP/mBS31001/SC"}},
		&mockProductMatcher{result: makeMatchResult(t, "FAP/mBS31001/SC", &pmID)},
		&mockPMNameLookup{name: "BLQ"},
	)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		ParamModel string  `json:"param_model"`
		Groups     []Group `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "BLQ", resp.ParamModel)
	require.Len(t, resp.Groups, 1)
	assert.Equal(t, "enb-cell", resp.Groups[0].ID)
}

func TestHandler_GetGroups_UnknownParamModel_ReturnsEmptyGroups(t *testing.T) {
	// quicksettings/<name>.xml 缺失场景:Registry 没该 paramModel,GetByParamModel 返空
	pmID := uuid.New()
	deviceID := uuid.New()

	reg := NewRegistry() // 不预填任何 paramModel
	h := NewHandler(
		reg,
		&mockDeviceLookup{device: &model.Device{ProductClass: "Unknown-Product"}},
		&mockProductMatcher{result: makeMatchResult(t, "Unknown-Product", &pmID)},
		&mockPMNameLookup{name: "BM"}, // BM 的 XML 不存在
	)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		ParamModel string  `json:"param_model"`
		Groups     []Group `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "BM", resp.ParamModel)
	assert.Len(t, resp.Groups, 0, "前端凭 groups.length==0 决定是否显示 tab")
}

func TestHandler_GetGroups_MissingDeviceID_400(t *testing.T) {
	h := NewHandler(NewRegistry(), &mockDeviceLookup{}, &mockProductMatcher{}, &mockPMNameLookup{})
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetGroups_InvalidDeviceID_400(t *testing.T) {
	h := NewHandler(NewRegistry(), &mockDeviceLookup{}, &mockProductMatcher{}, &mockPMNameLookup{})
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id=not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetGroups_DeviceNotFound_404(t *testing.T) {
	deviceID := uuid.New()
	h := NewHandler(
		NewRegistry(),
		&mockDeviceLookup{err: errors.New("not found")},
		&mockProductMatcher{},
		&mockPMNameLookup{},
	)
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestHandler_GetGroups_EmptyDevice_404 锁 issue #180 回归:
// 设备查询带「排除已删除」过滤,软删/不存在的设备查出来是 (nil, nil)。
// 修复前代码紧接着解引用 device.ProductClass 触发空指针 panic → 兜成 500,前端拿 500 即空白。
// 断言:查到空设备返回 404 而非 500/panic。
func TestHandler_GetGroups_EmptyDevice_404(t *testing.T) {
	deviceID := uuid.New()
	h := NewHandler(
		NewRegistry(),
		&mockDeviceLookup{device: nil, err: nil}, // 软删/不存在:返回 (nil, nil)
		&mockProductMatcher{},
		&mockPMNameLookup{},
	)
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestHandler_GetGroups_DeviceLookupInfraError_404 钉住「GetDevice 任意 error 一律 404」
// 的当前契约(blanket err->404)。
//
// 此处模拟的是「瞬时基础设施错误」(如 DB 连接超时 / 网络抖动),而非业务语义上的
// 「设备不存在」。handler.go GetGroups 中的 `if err != nil { 404 }` 不区分两者:
// 不论是 sql.ErrNoRows 还是 context deadline / 连接失败,都兜成 404。
//
// 本测试的目的是 *记录并锁定当前行为*——并非主张 404 是基础设施错误的理想响应
// (严格来说瞬时故障更宜 503/500 让前端可重试)。任何未来想把瞬时错误改判为
// 5xx 的改动都会撞红此断言,从而被迫是一次「有意识」的契约变更而非无声漂移。
// 不修改生产代码。
func TestHandler_GetGroups_DeviceLookupInfraError_404(t *testing.T) {
	deviceID := uuid.New()
	h := NewHandler(
		NewRegistry(),
		// device:nil + 瞬时基础设施错误(连接超时),区别于 DeviceNotFound 用例的语义化 "not found"。
		&mockDeviceLookup{device: nil, err: errors.New("dial tcp: i/o timeout")},
		&mockProductMatcher{},
		&mockPMNameLookup{},
	)
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 当前契约:任何 GetDevice error(含瞬时故障)→ 404。改判前请先确认这是有意为之。
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetGroups_ProductClassUnmatched_404(t *testing.T) {
	deviceID := uuid.New()
	h := NewHandler(
		NewRegistry(),
		&mockDeviceLookup{device: &model.Device{ProductClass: "Unknown"}},
		&mockProductMatcher{err: errors.New("orphan")},
		&mockPMNameLookup{},
	)
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetGroups_NoParamModel_422(t *testing.T) {
	deviceID := uuid.New()
	h := NewHandler(
		NewRegistry(),
		&mockDeviceLookup{device: &model.Device{ProductClass: "FAP"}},
		&mockProductMatcher{result: makeMatchResult(t, "FAP", nil)}, // ParamModelID = nil
		&mockPMNameLookup{},
	)
	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?device_id="+deviceID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
