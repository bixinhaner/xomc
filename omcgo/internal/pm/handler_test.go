package pm

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
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock repositories (pmH prefix to avoid collision with threshold_handler_test)
// ---------------------------------------------------------------------------

type pmHCounterRepo struct {
	queryFn           func(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error)
	queryAggregatedFn func(ctx context.Context, filter counter.CounterFilter) ([]counter.AggregatedCounter, error)
}

func (m *pmHCounterRepo) BatchInsert(_ context.Context, _ []model.PMCounter) error { return nil }
func (m *pmHCounterRepo) Query(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, filter)
	}
	return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
}
func (m *pmHCounterRepo) QueryAggregated(ctx context.Context, filter counter.CounterFilter) ([]counter.AggregatedCounter, error) {
	if m.queryAggregatedFn != nil {
		return m.queryAggregatedFn(ctx, filter)
	}
	return nil, nil
}
func (m *pmHCounterRepo) QueryForKPI(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
	return nil, nil
}

type pmHKPIRepo struct {
	queryFn func(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error)
}

func (m *pmHKPIRepo) BatchInsert(_ context.Context, _ []model.KPIValue) error { return nil }
func (m *pmHKPIRepo) Query(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, filter)
	}
	return model.NewListResponse([]model.KPIValue{}, 0, 1, 20), nil
}
func (m *pmHKPIRepo) ListDefinitions(_ context.Context, _ *model.CarrierCode, _ *model.Technology) ([]model.KPIDefinition, error) {
	return nil, nil
}
func (m *pmHKPIRepo) SyncDefinitions(_ context.Context, _ []model.KPIDefinition) error { return nil }

type pmHTaskRepo struct {
	listFn   func(ctx context.Context, filter TaskFilter) (*model.ListResponse[PerformanceTask], error)
	createFn func(ctx context.Context, task *PerformanceTask) error
}

func (m *pmHTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[PerformanceTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return model.NewListResponse([]PerformanceTask{}, 0, 1, 20), nil
}
func (m *pmHTaskRepo) Create(ctx context.Context, task *PerformanceTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func pmHNewEngine() *kpi.KPIEngine {
	return kpi.NewKPIEngine(&pmHCounterRepo{}, &pmHKPIRepo{}, carrier.NewRegistry(), zap.NewNop())
}

func pmHSetupRouter(cr counter.CounterRepository, kr kpi.KPIRepository, engine *kpi.KPIEngine, tr TaskRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(cr, kr, engine, tr, nil, nil, "pm-files", zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListCounters_Default(t *testing.T) {
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 20, f.PageSize)
			return model.NewListResponse([]model.PMCounter{
				{CounterName: "rrc_conn_setup_att", CounterValue: 1000},
			}, 1, 1, 20), nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[model.PMCounter]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
}

func TestHandler_ListCounters_WithDeviceFilter(t *testing.T) {
	deviceID := uuid.New()
	cr := &pmHCounterRepo{
		queryFn: func(_ context.Context, f counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
			require.NotNil(t, f.DeviceID)
			assert.Equal(t, deviceID, *f.DeviceID)
			return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id="+deviceID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListCounters_InvalidDeviceID(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters?device_id=bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAggregatedCounters(t *testing.T) {
	cr := &pmHCounterRepo{
		queryAggregatedFn: func(_ context.Context, _ counter.CounterFilter) ([]counter.AggregatedCounter, error) {
			return []counter.AggregatedCounter{
				{CounterName: "rrc_conn_setup_att", SumValue: 5000, AvgValue: 1000},
			}, nil
		},
	}
	router := pmHSetupRouter(cr, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/counters/aggregated?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []counter.AggregatedCounter `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	assert.Equal(t, float64(5000), body.Items[0].SumValue)
}

func TestHandler_ListKPIValues(t *testing.T) {
	kr := &pmHKPIRepo{
		queryFn: func(_ context.Context, _ kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
			return model.NewListResponse([]model.KPIValue{
				{KPIName: "rrc_success_rate", KPIValue: 95.0},
			}, 1, 1, 20), nil
		},
	}
	router := pmHSetupRouter(&pmHCounterRepo{}, kr, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[model.KPIValue]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Total)
}

func TestHandler_ListKPIDefinitions(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/kpi/definitions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []model.KPIDefinition `json:"items"`
		Total int                   `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Total)
}

func TestHandler_CalculateKPI_BadRequest(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"device_id": "not-a-uuid"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/pm/kpi/calculate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListTasks(t *testing.T) {
	tr := &pmHTaskRepo{
		listFn: func(_ context.Context, f TaskFilter) (*model.ListResponse[PerformanceTask], error) {
			assert.Nil(t, f.Status)
			assert.Nil(t, f.TaskType)
			return model.NewListResponse([]PerformanceTask{
				{TaskName: "Daily PM", TaskType: PMTaskExtraction, Status: PMTaskPending},
			}, 1, 1, 20), nil
		},
	}
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), tr)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/pm/tasks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[PerformanceTask]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Total)
}

func TestHandler_CreateTask_Success(t *testing.T) {
	var created *PerformanceTask
	tr := &pmHTaskRepo{
		createFn: func(_ context.Context, task *PerformanceTask) error {
			task.ID = uuid.New()
			task.Status = PMTaskPending
			created = task
			return nil
		},
	}
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), tr)

	body, _ := json.Marshal(CreatePerformanceTaskRequest{
		TaskName:    "Nightly Report",
		TaskType:    PMTaskReport,
		Granularity: "15min",
		Creator:     "admin",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/pm/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, created)
	assert.Equal(t, "Nightly Report", created.TaskName)
}

func TestHandler_CreateTask_MissingName(t *testing.T) {
	router := pmHSetupRouter(&pmHCounterRepo{}, &pmHKPIRepo{}, pmHNewEngine(), &pmHTaskRepo{})

	body, _ := json.Marshal(map[string]string{"task_type": "report"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/pm/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
