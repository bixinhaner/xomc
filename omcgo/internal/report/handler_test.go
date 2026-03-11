package report

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/model"
)

// ---------------------------------------------------------------------------
// Function-field mock repositories
// ---------------------------------------------------------------------------

type mockDefinitionRepo struct {
	CreateFn  func(ctx context.Context, def *ReportDefinition) error
	GetByIDFn func(ctx context.Context, id uuid.UUID) (*ReportDefinition, error)
	UpdateFn  func(ctx context.Context, def *ReportDefinition) error
	DeleteFn  func(ctx context.Context, id uuid.UUID) error
	ListFn    func(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error)
}

func (m *mockDefinitionRepo) Create(ctx context.Context, def *ReportDefinition) error {
	return m.CreateFn(ctx, def)
}
func (m *mockDefinitionRepo) GetByID(ctx context.Context, id uuid.UUID) (*ReportDefinition, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockDefinitionRepo) Update(ctx context.Context, def *ReportDefinition) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, def)
	}
	return nil
}
func (m *mockDefinitionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *mockDefinitionRepo) List(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error) {
	return m.ListFn(ctx, filter)
}

type hRecordRepo struct {
	CreateFn  func(ctx context.Context, record *ReportRecord) error
	GetByIDFn func(ctx context.Context, id uuid.UUID) (*ReportRecord, error)
	ListFn    func(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error)
}

func (m *hRecordRepo) Create(ctx context.Context, record *ReportRecord) error {
	return m.CreateFn(ctx, record)
}
func (m *hRecordRepo) GetByID(ctx context.Context, id uuid.UUID) (*ReportRecord, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *hRecordRepo) List(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error) {
	return m.ListFn(ctx, filter)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupReportRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func mustMarshalReport(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func newReportTestHandler(
	defRepo *mockDefinitionRepo,
	recordRepo *hRecordRepo,
) *Handler {
	logger := zap.NewNop()
	svc := NewService(defRepo, recordRepo, logger)
	return NewHandler(svc, logger)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListDefinitions(t *testing.T) {
	now := time.Now()
	defID := uuid.New()

	defRepo := &mockDefinitionRepo{
		ListFn: func(_ context.Context, _ DefinitionFilter) (*model.ListResponse[ReportDefinition], error) {
			items := []ReportDefinition{
				{
					ID:           defID,
					ReportName:   "Daily KPI Report",
					ReportType:   ReportPerformance,
					Description:  "Daily KPI performance summary",
					Format:       []string{"pdf", "xlsx"},
					Period:       PeriodDaily,
					KPICodes:     []string{"rrc_succ_rate", "erab_succ_rate"},
					DeviceGroups: []string{"region-north"},
					AutoGenerate: true,
					Status:       ReportPublished,
					Creator:      "admin",
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			}
			return model.NewListResponse(items, 1, 1, 20), nil
		},
	}
	recordRepo := &hRecordRepo{}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/definitions?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[ReportDefinition]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Daily KPI Report", resp.Items[0].ReportName)
	assert.Equal(t, ReportPerformance, resp.Items[0].ReportType)
	assert.Equal(t, ReportPublished, resp.Items[0].Status)
	assert.Equal(t, []string{"pdf", "xlsx"}, resp.Items[0].Format)
	assert.Equal(t, []string{"rrc_succ_rate", "erab_succ_rate"}, resp.Items[0].KPICodes)
}

func TestHandler_CreateDefinition(t *testing.T) {
	defRepo := &mockDefinitionRepo{
		CreateFn: func(_ context.Context, def *ReportDefinition) error {
			def.ID = uuid.New()
			def.CreatedAt = time.Now()
			def.UpdatedAt = time.Now()
			return nil
		},
	}
	recordRepo := &hRecordRepo{}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	body := CreateDefinitionRequest{
		ReportName:   "Weekly Alarm Report",
		ReportType:   ReportAlarm,
		Description:  "Weekly alarm statistics",
		Format:       []string{"pdf"},
		Period:       PeriodWeekly,
		KPICodes:     []string{"alarm_count", "alarm_clear_time"},
		DeviceGroups: []string{"all"},
		AutoGenerate: false,
		Status:       ReportDraft,
		Creator:      "admin",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/reports/definitions", bytes.NewReader(mustMarshalReport(t, body)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp ReportDefinition
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "Weekly Alarm Report", resp.ReportName)
	assert.Equal(t, ReportAlarm, resp.ReportType)
	assert.Equal(t, ReportDraft, resp.Status)
	assert.Equal(t, PeriodWeekly, resp.Period)
}

func TestHandler_GetDefinition(t *testing.T) {
	now := time.Now()
	defID := uuid.New()

	defRepo := &mockDefinitionRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*ReportDefinition, error) {
			return &ReportDefinition{
				ID:             defID,
				ReportName:     "Monthly Device Report",
				ReportType:     ReportDevice,
				Description:    "Monthly device summary",
				Format:         []string{"xlsx"},
				Period:         PeriodMonthly,
				KPICodes:       []string{"device_online_rate"},
				DeviceGroups:   []string{"region-south"},
				AutoGenerate:   true,
				CronExpression: "0 0 1 * *",
				Status:         ReportPublished,
				Creator:        "admin",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}
	recordRepo := &hRecordRepo{}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/definitions/"+defID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ReportDefinition
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, defID, resp.ID)
	assert.Equal(t, "Monthly Device Report", resp.ReportName)
	assert.Equal(t, ReportDevice, resp.ReportType)
	assert.True(t, resp.AutoGenerate)
	assert.Equal(t, "0 0 1 * *", resp.CronExpression)
}

func TestHandler_DeleteDefinition(t *testing.T) {
	defID := uuid.New()

	defRepo := &mockDefinitionRepo{
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, defID, id)
			return nil
		},
	}
	recordRepo := &hRecordRepo{}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/reports/definitions/"+defID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestHandler_ListRecords(t *testing.T) {
	now := time.Now()
	recordID := uuid.New()
	defID := uuid.New()

	defRepo := &mockDefinitionRepo{}
	recordRepo := &hRecordRepo{
		ListFn: func(_ context.Context, _ RecordFilter) (*model.ListResponse[ReportRecord], error) {
			items := []ReportRecord{
				{
					ID:                 recordID,
					ReportDefinitionID: defID,
					ReportName:         "Daily KPI Report-2026-03-10",
					Period:             "2026-03-10",
					GenerateTime:       now,
					FileSize:           1024000,
					DownloadURL:        "https://minio.local/reports/daily-kpi-2026-03-10.pdf",
					Format:             "pdf",
					Status:             RecordReady,
					CreatedAt:          now,
				},
			}
			return model.NewListResponse(items, 1, 1, 20), nil
		},
	}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/records?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[ReportRecord]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Daily KPI Report-2026-03-10", resp.Items[0].ReportName)
	assert.Equal(t, RecordReady, resp.Items[0].Status)
	assert.Equal(t, "pdf", resp.Items[0].Format)
}

func TestHandler_GenerateReport(t *testing.T) {
	defID := uuid.New()

	defRepo := &mockDefinitionRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*ReportDefinition, error) {
			return &ReportDefinition{
				ID:         defID,
				ReportName: "Daily KPI Report",
				ReportType: ReportPerformance,
				Format:     []string{"pdf", "xlsx"},
				Status:     ReportPublished,
			}, nil
		},
		UpdateFn: func(_ context.Context, def *ReportDefinition) error {
			return nil
		},
	}
	recordRepo := &hRecordRepo{
		CreateFn: func(_ context.Context, record *ReportRecord) error {
			record.ID = uuid.New()
			record.CreatedAt = time.Now()
			return nil
		},
	}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	body := GenerateReportRequest{
		DefinitionID: defID.String(),
		Period:       "2026-03-10",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", bytes.NewReader(mustMarshalReport(t, body)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp ReportRecord
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, defID, resp.ReportDefinitionID)
	assert.Equal(t, "Daily KPI Report-2026-03-10", resp.ReportName)
	assert.Equal(t, "2026-03-10", resp.Period)
	assert.Equal(t, RecordGenerating, resp.Status)
	assert.Equal(t, "pdf", resp.Format)
}

func TestHandler_GetSampleData(t *testing.T) {
	defRepo := &mockDefinitionRepo{}
	recordRepo := &hRecordRepo{}

	h := newReportTestHandler(defRepo, recordRepo)
	router := setupReportRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sample-data", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// Verify the three top-level keys exist
	assert.Contains(t, resp, "kpi_summary")
	assert.Contains(t, resp, "alarm_summary")
	assert.Contains(t, resp, "device_summary")

	// Verify some nested data
	kpiSummary, ok := resp["kpi_summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, kpiSummary, "rrc_succ_rate")
	assert.Contains(t, kpiSummary, "dl_throughput")

	deviceSummary, ok := resp["device_summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(200), deviceSummary["total_devices"])
	assert.Equal(t, float64(170), deviceSummary["online"])
}
