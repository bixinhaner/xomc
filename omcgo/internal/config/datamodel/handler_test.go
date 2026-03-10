package datamodel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: DataModelRepository (dmH prefix to avoid conflict with importer_test.go)
// ---------------------------------------------------------------------------

type dmHDataModelRepo struct {
	createFn     func(ctx context.Context, dm *DataModel) error
	getByIDFn    func(ctx context.Context, id uuid.UUID) (*DataModel, error)
	updateFn     func(ctx context.Context, dm *DataModel) error
	deleteFn     func(ctx context.Context, id uuid.UUID) error
	listFn       func(ctx context.Context, filter DataModelFilter) (*model.ListResponse[DataModel], error)
	findActiveFn func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, oui, pc string, scope model.DataModelScope) (*DataModel, error)
	activateFn   func(ctx context.Context, id uuid.UUID) error
	deprecateFn  func(ctx context.Context, id uuid.UUID) error
	statisticsFn func(ctx context.Context) (*DataModelStats, error)
}

func (m *dmHDataModelRepo) Create(ctx context.Context, dm *DataModel) error {
	if m.createFn != nil {
		return m.createFn(ctx, dm)
	}
	dm.ID = uuid.New()
	return nil
}
func (m *dmHDataModelRepo) GetByID(ctx context.Context, id uuid.UUID) (*DataModel, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}
func (m *dmHDataModelRepo) Update(ctx context.Context, dm *DataModel) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, dm)
	}
	return nil
}
func (m *dmHDataModelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *dmHDataModelRepo) List(ctx context.Context, filter DataModelFilter) (*model.ListResponse[DataModel], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]DataModel{}, 0, 1, 20), nil
}
func (m *dmHDataModelRepo) FindActive(ctx context.Context, carrier model.CarrierCode, tech model.Technology, oui, pc string, scope model.DataModelScope) (*DataModel, error) {
	if m.findActiveFn != nil {
		return m.findActiveFn(ctx, carrier, tech, oui, pc, scope)
	}
	return nil, nil
}
func (m *dmHDataModelRepo) Activate(ctx context.Context, id uuid.UUID) error {
	if m.activateFn != nil {
		return m.activateFn(ctx, id)
	}
	return nil
}
func (m *dmHDataModelRepo) Deprecate(ctx context.Context, id uuid.UUID) error {
	if m.deprecateFn != nil {
		return m.deprecateFn(ctx, id)
	}
	return nil
}
func (m *dmHDataModelRepo) Statistics(ctx context.Context) (*DataModelStats, error) {
	if m.statisticsFn != nil {
		return m.statisticsFn(ctx)
	}
	return &DataModelStats{}, nil
}

// ---------------------------------------------------------------------------
// Mock: OUIRepository (dmH prefix)
// ---------------------------------------------------------------------------

type dmHOUIRepo struct {
	listFn   func(ctx context.Context) ([]OUIEntry, error)
	createFn func(ctx context.Context, entry *OUIEntry) error
}

func (m *dmHOUIRepo) GetByOUI(_ context.Context, _ string) (*OUIEntry, error) {
	return nil, fmt.Errorf("not found")
}
func (m *dmHOUIRepo) List(ctx context.Context) ([]OUIEntry, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []OUIEntry{}, nil
}
func (m *dmHOUIRepo) Create(ctx context.Context, entry *OUIEntry) error {
	if m.createFn != nil {
		return m.createFn(ctx, entry)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: ImportLogRepository (dmH prefix)
// ---------------------------------------------------------------------------

type dmHImportLogRepo struct{}

func (m *dmHImportLogRepo) Create(_ context.Context, _ *ImportLogEntry) error        { return nil }
func (m *dmHImportLogRepo) ListByModel(_ context.Context, _ uuid.UUID) ([]ImportLogEntry, error) {
	return []ImportLogEntry{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func dmHSetupRouter(repo DataModelRepository, ouiRepo OUIRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registry := NewDataModelRegistry(repo, nil, zap.NewNop())
	importer := NewDataModelImporter(repo, &dmHImportLogRepo{})
	h := NewHandler(repo, ouiRepo, registry, importer)
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func dmHSampleModel(id uuid.UUID) *DataModel {
	return &DataModel{
		ID:            id,
		Carrier:       "cmcc",
		Technology:    "lte",
		Version:       "V2.3",
		Scope:         model.ScopeCarrierDefault,
		Status:        StatusActive,
		IsActive:      true,
		RootObject:    "Device.",
		ParameterTree: json.RawMessage(`{"Device":{}}`),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestDmHandler_List_Default(t *testing.T) {
	repo := &dmHDataModelRepo{
		listFn: func(_ context.Context, _ DataModelFilter) (*model.ListResponse[DataModel], error) {
			return model.NewListResponse([]DataModel{
				*dmHSampleModel(uuid.New()),
				*dmHSampleModel(uuid.New()),
			}, 2, 1, 20), nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datamodels", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[DataModel]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Total)
}

func TestDmHandler_Get_Success(t *testing.T) {
	id := uuid.New()
	repo := &dmHDataModelRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*DataModel, error) {
			assert.Equal(t, id, gotID)
			return dmHSampleModel(id), nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datamodels/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp DataModel
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, id, resp.ID)
}

func TestDmHandler_Get_InvalidUUID(t *testing.T) {
	router := dmHSetupRouter(&dmHDataModelRepo{}, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datamodels/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDmHandler_Create_Success(t *testing.T) {
	repo := &dmHDataModelRepo{
		createFn: func(_ context.Context, dm *DataModel) error {
			dm.ID = uuid.New()
			dm.CreatedAt = time.Now()
			dm.UpdatedAt = time.Now()
			return nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	body, _ := json.Marshal(createDataModelRequest{
		Carrier:       "cmcc",
		Technology:    "lte",
		Version:       "V2.3",
		ParameterTree: json.RawMessage(`{"Device":{}}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datamodels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp DataModel
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, StatusDraft, resp.Status)
	assert.Equal(t, "Device.", resp.RootObject)
}

func TestDmHandler_Create_BadRequest(t *testing.T) {
	router := dmHSetupRouter(&dmHDataModelRepo{}, &dmHOUIRepo{})

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"carrier": "cmcc"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datamodels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDmHandler_Delete_Success(t *testing.T) {
	deleteCalled := false
	repo := &dmHDataModelRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/datamodels/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleteCalled)
}

func TestDmHandler_Activate_Success(t *testing.T) {
	id := uuid.New()
	repo := &dmHDataModelRepo{
		activateFn: func(_ context.Context, gotID uuid.UUID) error {
			assert.Equal(t, id, gotID)
			return nil
		},
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*DataModel, error) {
			return dmHSampleModel(id), nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datamodels/"+id.String()+"/activate", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDmHandler_Deprecate_Success(t *testing.T) {
	id := uuid.New()
	repo := &dmHDataModelRepo{
		deprecateFn: func(_ context.Context, gotID uuid.UUID) error {
			assert.Equal(t, id, gotID)
			return nil
		},
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*DataModel, error) {
			return dmHSampleModel(id), nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/datamodels/"+id.String()+"/deprecate", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDmHandler_Statistics(t *testing.T) {
	repo := &dmHDataModelRepo{
		statisticsFn: func(_ context.Context) (*DataModelStats, error) {
			return &DataModelStats{
				Total:  10,
				Active: 5,
				Draft:  3,
				Deprecated: 2,
				ByCarrier: map[string]int64{"cmcc": 6, "cucc": 4},
				ByScope:   map[string]int64{"carrier_default": 8, "product": 2},
			}, nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datamodels/statistics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var stats DataModelStats
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))
	assert.Equal(t, int64(10), stats.Total)
}

func TestDmHandler_Resolve_MissingParams(t *testing.T) {
	router := dmHSetupRouter(&dmHDataModelRepo{}, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datamodels/resolve", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDmHandler_Resolve_NotFound(t *testing.T) {
	// FindActive returns nil → registry returns nil → 404
	repo := &dmHDataModelRepo{
		findActiveFn: func(_ context.Context, _ model.CarrierCode, _ model.Technology, _, _ string, _ model.DataModelScope) (*DataModel, error) {
			return nil, nil
		},
	}
	router := dmHSetupRouter(repo, &dmHOUIRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/datamodels/resolve?carrier=cmcc&technology=lte", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDmHandler_ListOUI(t *testing.T) {
	ouiRepo := &dmHOUIRepo{
		listFn: func(_ context.Context) ([]OUIEntry, error) {
			return []OUIEntry{
				{OUI: "00E0FC", Manufacturer: "Huawei", ShortName: "HW"},
				{OUI: "001234", Manufacturer: "ZTE", ShortName: "ZTE"},
			}, nil
		},
	}
	router := dmHSetupRouter(&dmHDataModelRepo{}, ouiRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/oui", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []OUIEntry `json:"items"`
		Total int        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 2, body.Total)
	assert.Len(t, body.Items, 2)
}

func TestDmHandler_CreateOUI_Success(t *testing.T) {
	ouiRepo := &dmHOUIRepo{
		createFn: func(_ context.Context, entry *OUIEntry) error {
			assert.Equal(t, "00E0FC", entry.OUI)
			return nil
		},
	}
	router := dmHSetupRouter(&dmHDataModelRepo{}, ouiRepo)

	body, _ := json.Marshal(createOUIRequest{
		OUI:          "00E0FC",
		Manufacturer: "Huawei",
		ShortName:    "HW",
		Country:      "CN",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/oui", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestDmHandler_CreateOUI_BadRequest(t *testing.T) {
	router := dmHSetupRouter(&dmHDataModelRepo{}, &dmHOUIRepo{})

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"oui": "00E0FC"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/oui", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
