package device

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeParamModelRepoForDevice struct {
	defaultByModel map[uuid.UUID][]parammodel.ParamMapping
}

func (f *fakeParamModelRepoForDevice) ListMappingsByParamModel(_ context.Context, paramModelID uuid.UUID) ([]parammodel.ParamMapping, error) {
	out := f.defaultByModel[paramModelID]
	cp := make([]parammodel.ParamMapping, len(out))
	copy(cp, out)
	return cp, nil
}

func (f *fakeParamModelRepoForDevice) ListDiscoveredMappings(_ context.Context, _ uuid.UUID, _ string) ([]parammodel.ParamMapping, error) {
	return nil, nil
}

type fakeProductRepoForDevice struct {
	products map[uuid.UUID]*product.Product
	patterns []product.ProductClassPattern
}

func (f *fakeProductRepoForDevice) ListActivePatterns(_ context.Context) ([]product.ProductClassPattern, error) {
	out := make([]product.ProductClassPattern, len(f.patterns))
	copy(out, f.patterns)
	return out, nil
}

func (f *fakeProductRepoForDevice) GetProductByID(_ context.Context, id uuid.UUID) (*product.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (f *fakeProductRepoForDevice) ListProducts(_ context.Context) ([]*product.Product, error) {
	return nil, nil
}

func (f *fakeProductRepoForDevice) FetchIndicatorPlatformsByDeviceType(_ context.Context, _ string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func (f *fakeProductRepoForDevice) FetchAlarmNeTypes(_ context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func setupParamTreeRouter(h *ParameterTreeHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestParameterTreeHandler_UsesDefaultParamModelForTreeAndChildren(t *testing.T) {
	deviceRepo := newFakeDeviceRepo()
	paramRepo := newFakeParamRepo()
	logger := zap.NewNop()
	service := NewDeviceService(deviceRepo, paramRepo, nil, nil, logger)

	deviceID := uuid.New()
	productID := uuid.New()
	paramModelID := uuid.New()
	seeded := seedDevice(deviceRepo, deviceID, "SN-PARAM-001", model.CarrierCMCC, model.TechLTE, model.DeviceActive)
	seeded.ProductClass = "PicoCell-LTE"

	paramRepo.params[deviceID] = []model.DeviceParameter{
		{
			DeviceID:       deviceID,
			ParameterPath:  "Device.DeviceInfo.SerialNumber",
			ParameterValue: "SN-PARAM-001",
			ParameterType:  model.ParamString,
			LastUpdatedAt:  time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		},
	}

	productRepo := &fakeProductRepoForDevice{
		products: map[uuid.UUID]*product.Product{
			productID: {
				ID:           productID,
				Name:         "PicoCell LTE",
				ParamModelID: &paramModelID,
			},
		},
		patterns: []product.ProductClassPattern{{
			ID:           uuid.New(),
			ProductID:    productID,
			ProductClass: "^PicoCell-LTE$",
			SortOrder:    1,
			IsActive:     true,
		}},
	}

	paramModelRepo := &fakeParamModelRepoForDevice{
		defaultByModel: map[uuid.UUID][]parammodel.ParamMapping{
			paramModelID: {
				{
					ID:           uuid.New(),
					ParamModelID: paramModelID,
					PrivatePath:  "Device.DeviceInfo.SerialNumber",
					StandardPath: "Device.DeviceInfo.SerialNumber",
					EntryType:    "parameter",
					DataType:     string(model.ParamString),
					Access:       "readOnly",
				},
				{
					ID:           uuid.New(),
					ParamModelID: paramModelID,
					PrivatePath:  "Device.DeviceInfo.ManufacturerOUI",
					StandardPath: "Device.DeviceInfo.ManufacturerOUI",
					EntryType:    "parameter",
					DataType:     string(model.ParamString),
					Access:       "readWrite",
				},
			},
		},
	}

	productRegistry := product.NewRegistry(productRepo, product.NopCache{}, product.NewRegistryMetrics(nil), logger)
	require.NoError(t, productRegistry.Refresh(context.Background()))
	paramRegistry := parammodel.NewRegistry(paramModelRepo, parammodel.NopCache{}, productRegistry, parammodel.NewRegistryMetrics(nil), logger)

	handler := NewParameterTreeHandler(service, paramRepo, paramRegistry, productRegistry, logger)
	router := setupParamTreeRouter(handler)

	t.Run("tree includes model-only parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID.String()+"/parameters/tree", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var payload struct {
			Tree  []ParameterTreeNode `json:"tree"`
			Total int                 `json:"total"`
		}
		response.DecodeData(t, w.Body, &payload)
		assert.Equal(t, 2, payload.Total)

		deviceNode := payload.Tree[0]
		require.Equal(t, "Device", deviceNode.Name)
		infoNode := deviceNode.Children[0]
		require.Equal(t, "DeviceInfo", infoNode.Name)
		require.Len(t, infoNode.Children, 2)
		assert.Equal(t, "ManufacturerOUI", infoNode.Children[0].Name)
		assert.Equal(t, "", infoNode.Children[0].Value)
		assert.True(t, infoNode.Children[0].Writable)
		assert.Equal(t, "SerialNumber", infoNode.Children[1].Name)
		assert.Equal(t, "SN-PARAM-001", infoNode.Children[1].Value)
	})

	t.Run("children includes model-only parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/devices/"+deviceID.String()+"/parameters/children?path_prefix=Device.DeviceInfo.",
			nil,
		)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var payload DirectChildrenResponse
		response.DecodeData(t, w.Body, &payload)
		require.Len(t, payload.Items, 2)
		assert.Equal(t, "Device.DeviceInfo.ManufacturerOUI", payload.Items[0].ParameterPath)
		assert.Equal(t, "", payload.Items[0].ParameterValue)
		assert.True(t, payload.Items[0].Writable)
		assert.Equal(t, "Device.DeviceInfo.SerialNumber", payload.Items[1].ParameterPath)
		assert.Equal(t, "SN-PARAM-001", payload.Items[1].ParameterValue)
	})

	t.Run("schema includes model-only parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/devices/"+deviceID.String()+"/parameters/schema?path_prefix=Device.DeviceInfo",
			nil,
		)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var payload struct {
			Parameters []ParameterSchemaItem `json:"parameters"`
			Objects    []ObjectSchemaItem    `json:"objects"`
			Total      int                   `json:"total"`
		}
		response.DecodeData(t, w.Body, &payload)
		require.Len(t, payload.Parameters, 2)
		assert.Empty(t, payload.Objects)
		assert.Equal(t, 2, payload.Total)
		byPath := map[string]ParameterSchemaItem{}
		for _, item := range payload.Parameters {
			byPath[item.Path] = item
		}
		manufacturer, ok := byPath["Device.DeviceInfo.ManufacturerOUI"]
		require.True(t, ok)
		assert.Nil(t, manufacturer.CurrentValue)
		assert.True(t, manufacturer.Writable)
		assert.Equal(t, string(model.ParamString), manufacturer.Type)

		serial, ok := byPath["Device.DeviceInfo.SerialNumber"]
		require.True(t, ok)
		require.NotNil(t, serial.CurrentValue)
		assert.Equal(t, "SN-PARAM-001", *serial.CurrentValue)
		assert.False(t, serial.Writable)
		assert.Equal(t, string(model.ParamString), serial.Type)
	})
}

func TestMergeSchemaWithValues_IncludesMissingModelParameterForResolvedInstance(t *testing.T) {
	mv := parammodel.NewMappingValidator(&parammodel.MappingSet{
		Source: parammodel.MappingSourceDefault,
		Mappings: []parammodel.ParamMapping{
			{
				PrivatePath:    "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC",
				StandardPath:   "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC",
				EntryType:      "parameter",
				DataType:       string(model.ParamUint),
				Access:         "READ_WRITE",
				ChangeApplies:  "Immediate",
				MinValue:       int64Ptr(0),
				MaxValue:       int64Ptr(65535),
			},
			{
				PrivatePath:   "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID",
				StandardPath:  "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID",
				EntryType:     "parameter",
				DataType:      string(model.ParamUint),
				Access:        "READ_WRITE",
				ChangeApplies: "Immediate",
			},
		},
	})
	params := []model.DeviceParameter{
		{
			ParameterPath:  "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.PhyCellID",
			ParameterValue: "301",
			ParameterType:  model.ParamUint,
			Writable:       true,
			LastUpdatedAt:  time.Now(),
		},
	}

	items := mergeSchemaWithValues(mv, params, "Device.Services.FAPService.2.CellConfig.LTE.")
	byPath := map[string]ParameterSchemaItem{}
	for _, item := range items {
		byPath[item.Path] = item
	}

	tac, ok := byPath["Device.Services.FAPService.2.CellConfig.LTE.EPC.TAC"]
	if !ok {
		t.Fatalf("expected TAC placeholder schema item for instance 2")
	}
	assert.Nil(t, tac.CurrentValue)
	assert.True(t, tac.Writable)
	require.NotNil(t, tac.Constraints)
	assert.EqualValues(t, 0, *tac.Constraints.MinValue)
	assert.EqualValues(t, 65535, *tac.Constraints.MaxValue)
}

func int64Ptr(v int64) *int64 { return &v }

func TestBuildObjectSchema_IgnoresModelOnlyPlaceholderInstances(t *testing.T) {
	params := []model.DeviceParameter{
		{
			ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth",
			LastUpdatedAt: time.Date(2026, 5, 29, 15, 55, 40, 0, time.UTC),
		},
		{
			ParameterPath: "Device.Services.FAPService.6.FAPControl.LTE.InUse",
			LastUpdatedAt: time.Date(2026, 5, 29, 15, 55, 41, 0, time.UTC),
		},
		{
			ParameterPath: "Device.Services.FAPService.7.CellConfig.LTE.RAN.RF.DLBandwidth",
		},
		{
			ParameterPath: "Device.Services.FAPService.8.CellConfig.LTE.RAN.RF.DLBandwidth",
		},
		{
			ParameterPath: "Device.Services.FAPService.9.CellConfig.LTE.RAN.RF.DLBandwidth",
		},
	}

	objects := buildObjectSchema(nil, params, "Device.Services.FAPService.")
	require.Len(t, objects, 1)
	assert.Equal(t, "Device.Services.FAPService.", objects[0].Path)
	assert.Equal(t, []int{1, 6}, objects[0].CurrentInstances)
}

// CPE 固件 phantom：未真正配置的 cell 槽位也回若干只读硬件指示器（NR Tx Vswr / Qos 容量）。
// 实例下若全是 readonly，应被 buildObjectSchema 当作 phantom 隐藏，避免 UI 出现伪实例。
// 同时合法的稀疏邻区（含 writable 配置项的 inst.1/3/4）必须保留。
func TestBuildObjectSchema_FiltersPhantomReadonlyOnlyInstances(t *testing.T) {
	mv := parammodel.NewMappingValidator(&parammodel.MappingSet{
		Source: parammodel.MappingSourceDefault,
		Mappings: []parammodel.ParamMapping{
			// CellConfig.{i}.NR.RAN.Tx0VswrStatus —— 只读硬件指示器
			{PrivatePath: "Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.Tx0VswrStatus", EntryType: "parameter", Access: "readOnly", DataType: "string", IsStorable: true},
			// CellConfig.{i}.NR.RAN.PHY.BW —— 真实可写配置项（仅 inst.1 上报）
			{PrivatePath: "Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.PHY.BW", EntryType: "parameter", Access: "readWrite", DataType: "unsignedInt", IsStorable: true},
			// LTECell.{i}.PhyCellID —— 邻区，多实例都可写（合法稀疏 1/3/4）
			{PrivatePath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.PhyCellID", EntryType: "parameter", Access: "readWrite", DataType: "unsignedInt", IsStorable: true},
		},
	})
	now := time.Now()
	params := []model.DeviceParameter{
		// CellConfig.1：含 writable 项 → 保留
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.BW", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.1.NR.RAN.Tx0VswrStatus", Writable: false, LastUpdatedAt: now},
		// CellConfig.2/3/4：phantom，全 readonly
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.2.NR.RAN.Tx0VswrStatus", Writable: false, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.3.NR.RAN.Tx0VswrStatus", Writable: false, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.4.NR.RAN.Tx0VswrStatus", Writable: false, LastUpdatedAt: now},
		// LTECell.1/3/4：合法稀疏邻区，writable
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.1.PhyCellID", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.3.PhyCellID", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.4.PhyCellID", Writable: true, LastUpdatedAt: now},
	}

	objects := buildObjectSchema(mv, params, "")
	gotInstances := map[string][]int{}
	for _, o := range objects {
		gotInstances[o.Path] = o.CurrentInstances
	}

	cellPath := "Device.Services.FAPService.1.CellConfig."
	require.Contains(t, gotInstances, cellPath)
	assert.Equal(t, []int{1}, gotInstances[cellPath], "NR phantom inst.2/3/4 should be filtered")

	ltePath := "Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell."
	require.Contains(t, gotInstances, ltePath)
	assert.Equal(t, []int{1, 3, 4}, gotInstances[ltePath], "LTE sparse neighbor instances must be preserved")
}

// 单标量实例（如 BAIBLQ FAPService.{3..12} 仅回 1 个 NumOfCells writable 标量）：
// 后端不再做启发式 phantom 过滤，全部保留；UI 层结合 NumOfCells + FAPControl.LTE.InUse 精确判定。
func TestBuildObjectSchema_KeepsSingleScalarInstancesForFrontendFiltering(t *testing.T) {
	mv := parammodel.NewMappingValidator(&parammodel.MappingSet{
		Source: parammodel.MappingSourceDefault,
		Mappings: []parammodel.ParamMapping{
			{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", EntryType: "parameter", Access: "readWrite", DataType: "unsignedInt", IsStorable: true},
			{PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID", EntryType: "parameter", Access: "readWrite", DataType: "unsignedInt", IsStorable: true},
			{PrivatePath: "Device.Services.FAPService.{i}.FAPControl.LTE.InUse", EntryType: "parameter", Access: "readWrite", DataType: "boolean", IsStorable: true},
		},
	})
	now := time.Now()
	params := []model.DeviceParameter{
		// inst.1: 配置完整的真主 FAP
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.1.FAPControl.LTE.InUse", Writable: true, LastUpdatedAt: now},
		// inst.2..5: 仅回单个 writable 标量（前端会结合 InUse 过滤；本层全部保留）
		{ParameterPath: "Device.Services.FAPService.2.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.3.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.4.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", Writable: true, LastUpdatedAt: now},
		{ParameterPath: "Device.Services.FAPService.5.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells", Writable: true, LastUpdatedAt: now},
	}

	objects := buildObjectSchema(mv, params, "")
	gotInstances := map[string][]int{}
	for _, o := range objects {
		gotInstances[o.Path] = o.CurrentInstances
	}

	fapPath := "Device.Services.FAPService."
	require.Contains(t, gotInstances, fapPath)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, gotInstances[fapPath], "all instances retained; UI layer filters with NumOfCells + InUse")
}
