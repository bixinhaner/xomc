package pm

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
	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockThresholdRepo struct {
	createFn  func(ctx context.Context, threshold *KPIThreshold) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*KPIThreshold, error)
	updateFn  func(ctx context.Context, threshold *KPIThreshold) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter KPIThresholdFilter) (*model.ListResponse[KPIThreshold], error)
}

func (m *mockThresholdRepo) Create(ctx context.Context, threshold *KPIThreshold) error {
	if m.createFn != nil {
		return m.createFn(ctx, threshold)
	}
	return nil
}

func (m *mockThresholdRepo) GetByID(ctx context.Context, id uuid.UUID) (*KPIThreshold, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockThresholdRepo) Update(ctx context.Context, threshold *KPIThreshold) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, threshold)
	}
	return nil
}

func (m *mockThresholdRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockThresholdRepo) List(ctx context.Context, filter KPIThresholdFilter) (*model.ListResponse[KPIThreshold], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return &model.ListResponse[KPIThreshold]{Items: []KPIThreshold{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupThresholdRouter(repo ThresholdRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewThresholdHandler(repo, zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

func sampleThreshold() *KPIThreshold {
	now := time.Now()
	warn := 80.0
	minor := 85.0
	major := 90.0
	critical := 95.0
	return &KPIThreshold{
		ID:                uuid.New(),
		KPIName:           "RSRP",
		Carrier:           "cmcc",
		Technology:        "lte",
		WarningThreshold:  &warn,
		MinorThreshold:    &minor,
		MajorThreshold:    &major,
		CriticalThreshold: &critical,
		Comparison:        "gt",
		Enabled:           true,
		Description:       "RSRP threshold for LTE cells",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func sampleThresholdList() *model.ListResponse[KPIThreshold] {
	t1 := sampleThreshold()
	return &model.ListResponse[KPIThreshold]{
		Items:      []KPIThreshold{*t1},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
}

// ---------------------------------------------------------------------------
// Tests — ListThresholds
// ---------------------------------------------------------------------------

func TestThresholdHandler_ListThresholds_Default(t *testing.T) {
	repo := &mockThresholdRepo{
		listFn: func(_ context.Context, f KPIThresholdFilter) (*model.ListResponse[KPIThreshold], error) {
			// Verify defaults are applied.
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 20, f.PageSize)
			assert.Nil(t, f.KPIName)
			assert.Nil(t, f.Carrier)
			assert.Nil(t, f.Technology)
			assert.Nil(t, f.Enabled)
			return sampleThresholdList(), nil
		},
	}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/thresholds", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[KPIThreshold]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
}

func TestThresholdHandler_ListThresholds_WithFilters(t *testing.T) {
	repo := &mockThresholdRepo{
		listFn: func(_ context.Context, f KPIThresholdFilter) (*model.ListResponse[KPIThreshold], error) {
			require.NotNil(t, f.KPIName)
			assert.Equal(t, "RSRP", *f.KPIName)
			require.NotNil(t, f.Carrier)
			assert.Equal(t, "cmcc", *f.Carrier)
			require.NotNil(t, f.Technology)
			assert.Equal(t, "lte", *f.Technology)
			require.NotNil(t, f.Enabled)
			assert.True(t, *f.Enabled)
			return sampleThresholdList(), nil
		},
	}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/thresholds?kpi_name=RSRP&carrier=cmcc&technology=lte&enabled=true", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Tests — GetThreshold
// ---------------------------------------------------------------------------

func TestThresholdHandler_GetThreshold_Found(t *testing.T) {
	threshold := sampleThreshold()
	repo := &mockThresholdRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*KPIThreshold, error) {
			if id == threshold.ID {
				return threshold, nil
			}
			return nil, commonerrors.ErrNotFound
		},
	}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/thresholds/%s", threshold.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp KPIThreshold
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, threshold.ID, resp.ID)
	assert.Equal(t, "RSRP", resp.KPIName)
}

func TestThresholdHandler_GetThreshold_NotFound(t *testing.T) {
	repo := &mockThresholdRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*KPIThreshold, error) {
			return nil, commonerrors.ErrNotFound
		},
	}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/thresholds/%s", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestThresholdHandler_GetThreshold_InvalidUUID(t *testing.T) {
	repo := &mockThresholdRepo{}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/thresholds/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tests — CreateThreshold
// ---------------------------------------------------------------------------

func TestThresholdHandler_CreateThreshold_Success(t *testing.T) {
	var created *KPIThreshold
	repo := &mockThresholdRepo{
		createFn: func(_ context.Context, th *KPIThreshold) error {
			th.ID = uuid.New()
			th.CreatedAt = time.Now()
			th.UpdatedAt = time.Now()
			created = th
			return nil
		},
	}
	router := setupThresholdRouter(repo)

	warn := 80.0
	body := CreateKPIThresholdRequest{
		KPIName:          "RSRP",
		Carrier:          "cmcc",
		Technology:       "lte",
		WarningThreshold: &warn,
		Comparison:       "gt",
		Description:      "Test threshold",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/thresholds", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, created)
	assert.Equal(t, "RSRP", created.KPIName)
	assert.True(t, created.Enabled) // default is true
	assert.Equal(t, "gt", created.Comparison)
}

func TestThresholdHandler_CreateThreshold_MissingKPIName(t *testing.T) {
	repo := &mockThresholdRepo{}
	router := setupThresholdRouter(repo)

	// kpi_name is required via binding:"required".
	body := map[string]interface{}{
		"carrier": "cmcc",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/thresholds", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestThresholdHandler_CreateThreshold_DefaultComparison(t *testing.T) {
	var created *KPIThreshold
	repo := &mockThresholdRepo{
		createFn: func(_ context.Context, th *KPIThreshold) error {
			created = th
			return nil
		},
	}
	router := setupThresholdRouter(repo)

	// Omit "comparison" — handler defaults to "gt".
	body := CreateKPIThresholdRequest{
		KPIName: "SINR",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/thresholds", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, created)
	assert.Equal(t, "gt", created.Comparison)
}

// ---------------------------------------------------------------------------
// Tests — UpdateThreshold
// ---------------------------------------------------------------------------

func TestThresholdHandler_UpdateThreshold_Success(t *testing.T) {
	existing := sampleThreshold()
	var updated *KPIThreshold
	repo := &mockThresholdRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*KPIThreshold, error) {
			if id == existing.ID {
				// Return a copy so mutation in handler doesn't change our reference.
				cp := *existing
				return &cp, nil
			}
			return nil, commonerrors.ErrNotFound
		},
		updateFn: func(_ context.Context, th *KPIThreshold) error {
			updated = th
			return nil
		},
	}
	router := setupThresholdRouter(repo)

	newName := "RSRQ"
	disabled := false
	body := UpdateKPIThresholdRequest{
		KPIName: &newName,
		Enabled: &disabled,
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/thresholds/%s", existing.ID.String()), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, updated)
	assert.Equal(t, "RSRQ", updated.KPIName)
	assert.False(t, updated.Enabled)
	// Unchanged fields should remain.
	assert.Equal(t, "cmcc", updated.Carrier)
}

func TestThresholdHandler_UpdateThreshold_NotFound(t *testing.T) {
	repo := &mockThresholdRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*KPIThreshold, error) {
			return nil, commonerrors.ErrNotFound
		},
	}
	router := setupThresholdRouter(repo)

	newName := "RSRQ"
	body := UpdateKPIThresholdRequest{KPIName: &newName}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/thresholds/%s", uuid.New().String()), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestThresholdHandler_UpdateThreshold_InvalidUUID(t *testing.T) {
	repo := &mockThresholdRepo{}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/thresholds/bad-id", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tests — DeleteThreshold
// ---------------------------------------------------------------------------

func TestThresholdHandler_DeleteThreshold_Success(t *testing.T) {
	thresholdID := uuid.New()
	deleted := false
	repo := &mockThresholdRepo{
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, thresholdID, id)
			deleted = true
			return nil
		},
	}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/thresholds/%s", thresholdID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, deleted)
}

func TestThresholdHandler_DeleteThreshold_NotFound(t *testing.T) {
	repo := &mockThresholdRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return commonerrors.ErrNotFound
		},
	}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/thresholds/%s", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestThresholdHandler_DeleteThreshold_InvalidUUID(t *testing.T) {
	repo := &mockThresholdRepo{}
	router := setupThresholdRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/thresholds/bad-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
