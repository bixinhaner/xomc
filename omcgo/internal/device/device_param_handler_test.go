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

	t.Run("schema only includes actual parameters", func(t *testing.T) {
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
		require.Len(t, payload.Parameters, 1)
		assert.Empty(t, payload.Objects)
		assert.Equal(t, 1, payload.Total)
		assert.Equal(t, "Device.DeviceInfo.SerialNumber", payload.Parameters[0].Path)
		require.NotNil(t, payload.Parameters[0].CurrentValue)
		assert.Equal(t, "SN-PARAM-001", *payload.Parameters[0].CurrentValue)
		assert.False(t, payload.Parameters[0].Writable)
		assert.Equal(t, string(model.ParamString), payload.Parameters[0].Type)
	})
}

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

func TestBuildObjectSchema_KeepsAllReadOnlyObjectInstances(t *testing.T) {
	mv := parammodel.NewMappingValidator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{
			StandardPath: "DeviceGSM.Bts.{i}.Band",
			PrivatePath:  "DeviceGSM.Bts.{i}.Band",
			EntryType:    "parameter",
			Access:       "READ_ONLY",
			IsStorable:   true,
			IsActive:     true,
			IsSupported:  true,
		},
		{
			StandardPath: "DeviceGSM.Bts.{i}.CellId",
			PrivatePath:  "DeviceGSM.Bts.{i}.CellId",
			EntryType:    "parameter",
			Access:       "READ_ONLY",
			IsStorable:   true,
			IsActive:     true,
			IsSupported:  true,
		},
	}})
	params := []model.DeviceParameter{
		{ParameterPath: "DeviceGSM.Bts.1.Band", LastUpdatedAt: time.Date(2026, 6, 18, 12, 30, 0, 0, time.UTC)},
		{ParameterPath: "DeviceGSM.Bts.2.Band", LastUpdatedAt: time.Date(2026, 6, 18, 12, 30, 1, 0, time.UTC)},
		{ParameterPath: "DeviceGSM.Bts.3.CellId", LastUpdatedAt: time.Date(2026, 6, 18, 12, 30, 2, 0, time.UTC)},
	}

	objects := buildObjectSchema(mv, params, "DeviceGSM.Bts.")
	require.Len(t, objects, 1)
	assert.Equal(t, "DeviceGSM.Bts.", objects[0].Path)
	assert.Equal(t, []int{1, 2, 3}, objects[0].CurrentInstances)
}
