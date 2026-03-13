package mr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/mr/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockMRStore struct {
	saveFileFn           func(ctx context.Context, file *MRFileInfo) error
	updateFileParsedFn   func(ctx context.Context, fileID uuid.UUID, recordCount int) error
	batchInsertRecordsFn func(ctx context.Context, fileID, deviceID uuid.UUID, mrType string, records []parser.MRRecord) error
	listFilesFn          func(ctx context.Context, filter MRFileFilter) (*model.ListResponse[MRFileInfo], error)
	getFileByIDFn        func(ctx context.Context, fileID uuid.UUID) (*MRFileInfo, error)
	queryRecordsFn       func(ctx context.Context, filter MRRecordFilter) (*model.ListResponse[MRRecordEntry], error)
}

func (m *mockMRStore) SaveFile(ctx context.Context, file *MRFileInfo) error {
	if m.saveFileFn != nil {
		return m.saveFileFn(ctx, file)
	}
	return nil
}

func (m *mockMRStore) UpdateFileParsed(ctx context.Context, fileID uuid.UUID, recordCount int) error {
	if m.updateFileParsedFn != nil {
		return m.updateFileParsedFn(ctx, fileID, recordCount)
	}
	return nil
}

func (m *mockMRStore) BatchInsertRecords(ctx context.Context, fileID, deviceID uuid.UUID, mrType string, records []parser.MRRecord) error {
	if m.batchInsertRecordsFn != nil {
		return m.batchInsertRecordsFn(ctx, fileID, deviceID, mrType, records)
	}
	return nil
}

func (m *mockMRStore) ListFiles(ctx context.Context, filter MRFileFilter) (*model.ListResponse[MRFileInfo], error) {
	if m.listFilesFn != nil {
		return m.listFilesFn(ctx, filter)
	}
	return &model.ListResponse[MRFileInfo]{Items: []MRFileInfo{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

func (m *mockMRStore) GetFileByID(ctx context.Context, fileID uuid.UUID) (*MRFileInfo, error) {
	if m.getFileByIDFn != nil {
		return m.getFileByIDFn(ctx, fileID)
	}
	return nil, nil
}

func (m *mockMRStore) QueryRecords(ctx context.Context, filter MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
	if m.queryRecordsFn != nil {
		return m.queryRecordsFn(ctx, filter)
	}
	return &model.ListResponse[MRRecordEntry]{Items: []MRRecordEntry{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

// ---------------------------------------------------------------------------

type mockIndicatorRepo struct {
	listFn      func(ctx context.Context, filter IndicatorFilter) (*model.ListResponse[MRIndicator], error)
	listAllFn   func(ctx context.Context) ([]MRIndicator, error)
	getByCodeFn func(ctx context.Context, code string) (*MRIndicator, error)
}

func (m *mockIndicatorRepo) List(ctx context.Context, filter IndicatorFilter) (*model.ListResponse[MRIndicator], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[MRIndicator]{Items: []MRIndicator{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

func (m *mockIndicatorRepo) ListAll(ctx context.Context) ([]MRIndicator, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx)
	}
	return []MRIndicator{}, nil
}

func (m *mockIndicatorRepo) GetByCode(ctx context.Context, code string) (*MRIndicator, error) {
	if m.getByCodeFn != nil {
		return m.getByCodeFn(ctx, code)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------

type mockMappingRepo struct {
	listFn          func(ctx context.Context, filter MappingFilter) (*model.ListResponse[MRDeviceMapping], error)
	updateFn        func(ctx context.Context, mapping *MRDeviceMapping) error
	toggleEnabledFn func(ctx context.Context, id uuid.UUID, enabled bool) (*MRDeviceMapping, error)
}

func (m *mockMappingRepo) List(ctx context.Context, filter MappingFilter) (*model.ListResponse[MRDeviceMapping], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[MRDeviceMapping]{Items: []MRDeviceMapping{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

func (m *mockMappingRepo) Update(ctx context.Context, mapping *MRDeviceMapping) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, mapping)
	}
	return nil
}

func (m *mockMappingRepo) ToggleEnabled(ctx context.Context, id uuid.UUID, enabled bool) (*MRDeviceMapping, error) {
	if m.toggleEnabledFn != nil {
		return m.toggleEnabledFn(ctx, id, enabled)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupMRRouter(store MRStore, indRepo IndicatorRepository, mapRepo MappingRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(store, indRepo, mapRepo, nil, "", zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

func sampleIndicators() []MRIndicator {
	desc := "Reference Signal Received Power"
	unit := "dBm"
	cat := "signal_quality"
	min := -140.0
	max := -44.0
	return []MRIndicator{
		{
			ID:            uuid.New(),
			IndicatorName: "RSRP",
			IndicatorCode: "MR.RSRP",
			Description:   &desc,
			Unit:          &unit,
			Category:      &cat,
			ValueRangeMin: &min,
			ValueRangeMax: &max,
			CreatedAt:     time.Now(),
		},
	}
}

func sampleIndicatorList() *model.ListResponse[MRIndicator] {
	items := sampleIndicators()
	return &model.ListResponse[MRIndicator]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
}

func sampleMRDataList() *model.ListResponse[MRRecordEntry] {
	return &model.ListResponse[MRRecordEntry]{
		Items: []MRRecordEntry{
			{
				Time:     time.Now(),
				FileID:   uuid.New(),
				DeviceID: uuid.New(),
				CellID:   "CELL-001",
				MRType:   "MRO",
				MeasurementData: map[string]interface{}{
					"RSRP": -85.5,
					"RSRQ": -10.2,
				},
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
}

// ---------------------------------------------------------------------------
// Tests — ListIndicators
// ---------------------------------------------------------------------------

func TestHandler_ListIndicators_Default(t *testing.T) {
	indRepo := &mockIndicatorRepo{
		listFn: func(_ context.Context, f IndicatorFilter) (*model.ListResponse[MRIndicator], error) {
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 20, f.PageSize)
			assert.Nil(t, f.Category)
			assert.Nil(t, f.Keyword)
			return sampleIndicatorList(), nil
		},
	}
	router := setupMRRouter(&mockMRStore{}, indRepo, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/indicators", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MRIndicator]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "MR.RSRP", resp.Items[0].IndicatorCode)
}

func TestHandler_ListIndicators_WithCategoryFilter(t *testing.T) {
	indRepo := &mockIndicatorRepo{
		listFn: func(_ context.Context, f IndicatorFilter) (*model.ListResponse[MRIndicator], error) {
			require.NotNil(t, f.Category)
			assert.Equal(t, "signal_quality", *f.Category)
			return sampleIndicatorList(), nil
		},
	}
	router := setupMRRouter(&mockMRStore{}, indRepo, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/indicators?category=signal_quality", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListIndicators_WithKeywordFilter(t *testing.T) {
	indRepo := &mockIndicatorRepo{
		listFn: func(_ context.Context, f IndicatorFilter) (*model.ListResponse[MRIndicator], error) {
			require.NotNil(t, f.Keyword)
			assert.Equal(t, "RSRP", *f.Keyword)
			return sampleIndicatorList(), nil
		},
	}
	router := setupMRRouter(&mockMRStore{}, indRepo, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/indicators?keyword=RSRP", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Tests — GetIndicatorStats
// ---------------------------------------------------------------------------

func TestHandler_GetIndicatorStats_Found(t *testing.T) {
	indicators := sampleIndicators()
	indRepo := &mockIndicatorRepo{
		getByCodeFn: func(_ context.Context, code string) (*MRIndicator, error) {
			if code == "MR.RSRP" {
				return &indicators[0], nil
			}
			return nil, nil
		},
	}
	router := setupMRRouter(&mockMRStore{}, indRepo, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/indicators/MR.RSRP/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "MR.RSRP", resp["indicator_code"])
	// min should be -140, max should be -44, avg = (-140 + -44)/2 = -92
	assert.InDelta(t, -92.0, resp["avg"], 0.01)
	assert.InDelta(t, -140.0, resp["min"], 0.01)
	assert.InDelta(t, -44.0, resp["max"], 0.01)
}

func TestHandler_GetIndicatorStats_NotFound(t *testing.T) {
	indRepo := &mockIndicatorRepo{
		getByCodeFn: func(_ context.Context, code string) (*MRIndicator, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	router := setupMRRouter(&mockMRStore{}, indRepo, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/indicators/UNKNOWN/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Tests — QueryMRData
// ---------------------------------------------------------------------------

func TestHandler_QueryMRData_Default(t *testing.T) {
	store := &mockMRStore{
		queryRecordsFn: func(_ context.Context, f MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 20, f.PageSize)
			assert.Nil(t, f.DeviceID)
			assert.Nil(t, f.FileID)
			assert.Nil(t, f.MRType)
			return sampleMRDataList(), nil
		},
	}
	router := setupMRRouter(store, &mockIndicatorRepo{}, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/data", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MRRecordEntry]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
}

func TestHandler_QueryMRData_WithDeviceID(t *testing.T) {
	deviceID := uuid.New()
	store := &mockMRStore{
		queryRecordsFn: func(_ context.Context, f MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
			require.NotNil(t, f.DeviceID)
			assert.Equal(t, deviceID, *f.DeviceID)
			return sampleMRDataList(), nil
		},
	}
	router := setupMRRouter(store, &mockIndicatorRepo{}, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/data?device_id="+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_QueryMRData_InvalidDeviceID(t *testing.T) {
	router := setupMRRouter(&mockMRStore{}, &mockIndicatorRepo{}, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/data?device_id=not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_QueryMRData_WithMRTypeFilter(t *testing.T) {
	store := &mockMRStore{
		queryRecordsFn: func(_ context.Context, f MRRecordFilter) (*model.ListResponse[MRRecordEntry], error) {
			require.NotNil(t, f.MRType)
			assert.Equal(t, "MRO", *f.MRType)
			return sampleMRDataList(), nil
		},
	}
	router := setupMRRouter(store, &mockIndicatorRepo{}, &mockMappingRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/mr/data?mr_type=MRO", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
