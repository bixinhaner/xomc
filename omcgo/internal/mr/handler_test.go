package mr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
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

// DeleteFilesBefore — F05 cleaner 集成新增的接口方法（P4.1）。
// mock 默认无副作用，返 (0, nil)；如需断言可在此 struct 加 deleteFilesBeforeFn 字段。
func (m *mockMRStore) DeleteFilesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}

// ListFileDeviceAggregates — File Management MR Tab 新增接口方法。
func (m *mockMRStore) ListFileDeviceAggregates(ctx context.Context, filter MRFileDeviceFilter) (*model.ListResponse[MRFileDeviceAggregate], error) {
	return &model.ListResponse[MRFileDeviceAggregate]{Items: []MRFileDeviceAggregate{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

// ListFilesBySN — 自选设备下钻新增接口方法，mock 默认返空切片。
func (m *mockMRStore) ListFilesBySN(ctx context.Context, sn string) ([]MRFileInfo, error) {
	return nil, nil
}

// DeleteFilesBySN — 自选设备下钻新增接口方法，mock 默认返 (0, nil)。
func (m *mockMRStore) DeleteFilesBySN(ctx context.Context, sn string) (int64, error) {
	return 0, nil
}

func (m *mockMRStore) ListUncompressed(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	return nil, nil
}

func (m *mockMRStore) MarkCompressed(ctx context.Context, renames map[string]string) error {
	return nil
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
	response.DecodeData(t, w.Body, &resp)
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
	response.DecodeData(t, w.Body, &resp)
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
	response.DecodeData(t, w.Body, &resp)
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

// ---------------------------------------------------------------------------
// Tests — UpdateMapping / ToggleMapping NotFound mapping (T-0057)
// ---------------------------------------------------------------------------

// TestUpdateMapping_NotFound_Returns404 verifies that PUT /mr/mappings/:id
// returns 404 (not 500) when the mapping does not exist. Repository wraps
// pgx.ErrNoRows / RowsAffected==0 as coreerrors.ErrNotFound; handler must
// translate via HTTPStatusFromError + AbortWithError.
func TestUpdateMapping_NotFound_Returns404(t *testing.T) {
	mapRepo := &mockMappingRepo{
		updateFn: func(_ context.Context, _ *MRDeviceMapping) error {
			return fmt.Errorf("mr mapping not found: %w", coreerrors.ErrNotFound)
		},
	}
	router := setupMRRouter(&mockMRStore{}, &mockIndicatorRepo{}, mapRepo)

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"sampling_interval":30}`)
	req, _ := http.NewRequest(http.MethodPut, "/mr/mappings/00000000-0000-0000-0000-000000000001", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateMapping_BareErrNotFound_Returns404 covers repositories that return
// a bare ErrNotFound sentinel (no fmt.Errorf wrap). HTTPStatusFromError must
// still resolve via errors.Is.
func TestUpdateMapping_BareErrNotFound_Returns404(t *testing.T) {
	mapRepo := &mockMappingRepo{
		updateFn: func(_ context.Context, _ *MRDeviceMapping) error {
			return coreerrors.ErrNotFound
		},
	}
	router := setupMRRouter(&mockMRStore{}, &mockIndicatorRepo{}, mapRepo)

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"sampling_interval":30}`)
	req, _ := http.NewRequest(http.MethodPut, "/mr/mappings/00000000-0000-0000-0000-000000000099", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateMapping_OtherError_Returns500 verifies non-NotFound errors fall
// back to 500 (default branch of HTTPStatusFromError).
func TestUpdateMapping_OtherError_Returns500(t *testing.T) {
	mapRepo := &mockMappingRepo{
		updateFn: func(_ context.Context, _ *MRDeviceMapping) error {
			return fmt.Errorf("db connection lost")
		},
	}
	router := setupMRRouter(&mockMRStore{}, &mockIndicatorRepo{}, mapRepo)

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"sampling_interval":30}`)
	req, _ := http.NewRequest(http.MethodPut, "/mr/mappings/00000000-0000-0000-0000-000000000001", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestToggleMapping_NotFound_Returns404 covers PUT /mr/mappings/:id/toggle
// where ToggleEnabled returned ErrNotFound (pgx.ErrNoRows on RETURNING).
func TestToggleMapping_NotFound_Returns404(t *testing.T) {
	mapRepo := &mockMappingRepo{
		toggleEnabledFn: func(_ context.Context, _ uuid.UUID, _ bool) (*MRDeviceMapping, error) {
			return nil, fmt.Errorf("mr mapping not found: %w", coreerrors.ErrNotFound)
		},
	}
	router := setupMRRouter(&mockMRStore{}, &mockIndicatorRepo{}, mapRepo)

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"enabled":false}`)
	req, _ := http.NewRequest(http.MethodPut, "/mr/mappings/00000000-0000-0000-0000-000000000001/toggle", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
