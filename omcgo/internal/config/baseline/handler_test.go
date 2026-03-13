package baseline

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
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock: BaselineRepository (handler-level, in-memory store)
// ---------------------------------------------------------------------------

type handlerMockBaselineRepo struct {
	baselines map[uuid.UUID]*BaselineConfig
}

func newHandlerMockBaselineRepo() *handlerMockBaselineRepo {
	return &handlerMockBaselineRepo{baselines: make(map[uuid.UUID]*BaselineConfig)}
}

func (m *handlerMockBaselineRepo) Create(_ context.Context, b *BaselineConfig) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now
	m.baselines[b.ID] = b
	return nil
}

func (m *handlerMockBaselineRepo) GetByID(_ context.Context, id uuid.UUID) (*BaselineConfig, error) {
	b, ok := m.baselines[id]
	if !ok {
		return nil, errHandlerBaselineNotFound
	}
	return b, nil
}

func (m *handlerMockBaselineRepo) Update(_ context.Context, b *BaselineConfig) error {
	if _, ok := m.baselines[b.ID]; !ok {
		return errHandlerBaselineNotFound
	}
	b.UpdatedAt = time.Now()
	m.baselines[b.ID] = b
	return nil
}

func (m *handlerMockBaselineRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.baselines[id]; !ok {
		return errHandlerBaselineNotFound
	}
	delete(m.baselines, id)
	return nil
}

func (m *handlerMockBaselineRepo) List(_ context.Context, filter BaselineFilter) (*model.ListResponse[BaselineConfig], error) {
	items := make([]BaselineConfig, 0, len(m.baselines))
	for _, b := range m.baselines {
		items = append(items, *b)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

// ---------------------------------------------------------------------------
// Mock: ConfigTaskRepository (handler-level, in-memory store)
// ---------------------------------------------------------------------------

type handlerMockTaskRepo struct {
	tasks map[uuid.UUID]*ConfigTask
}

func newHandlerMockTaskRepo() *handlerMockTaskRepo {
	return &handlerMockTaskRepo{tasks: make(map[uuid.UUID]*ConfigTask)}
}

func (m *handlerMockTaskRepo) Create(_ context.Context, task *ConfigTask) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	m.tasks[task.ID] = task
	return nil
}

func (m *handlerMockTaskRepo) List(_ context.Context, filter ConfigTaskFilter) (*model.ListResponse[ConfigTask], error) {
	items := make([]ConfigTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		items = append(items, *t)
	}
	return model.NewListResponse(items, int64(len(items)), filter.Page, filter.PageSize), nil
}

// ---------------------------------------------------------------------------
// Mock: NeighborRepository (handler-level, in-memory store)
// ---------------------------------------------------------------------------

type handlerMockNeighborRepo struct {
	neighbors []NeighborParam
}

func newHandlerMockNeighborRepo() *handlerMockNeighborRepo {
	return &handlerMockNeighborRepo{}
}

func (m *handlerMockNeighborRepo) List(_ context.Context, filter NeighborFilter) (*model.ListResponse[NeighborParam], error) {
	return model.NewListResponse(m.neighbors, int64(len(m.neighbors)), filter.Page, filter.PageSize), nil
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var errHandlerBaselineNotFound = fmt.Errorf("baseline not found")

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupHandlerRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newTestHandler() (*Handler, *handlerMockBaselineRepo, *handlerMockTaskRepo, *handlerMockNeighborRepo) {
	baselineRepo := newHandlerMockBaselineRepo()
	taskRepo := newHandlerMockTaskRepo()
	neighborRepo := newHandlerMockNeighborRepo()
	logger := zap.NewNop()
	svc := NewService(baselineRepo, taskRepo, neighborRepo, logger)
	h := NewHandler(svc, logger)
	return h, baselineRepo, taskRepo, neighborRepo
}

func seedHandlerBaseline(repo *handlerMockBaselineRepo, id uuid.UUID, name string, status BaselineStatus) *BaselineConfig {
	now := time.Now()
	desc := "Test baseline"
	b := &BaselineConfig{
		ID:           id,
		BaselineName: name,
		Description:  &desc,
		Params:       json.RawMessage(`[{"key":"value"}]`),
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.baselines[id] = b
	return b
}

func seedHandlerConfigTask(repo *handlerMockTaskRepo, id uuid.UUID, name string, taskType ConfigTaskType) *ConfigTask {
	now := time.Now()
	t := &ConfigTask{
		ID:        id,
		TaskName:  name,
		TaskType:  taskType,
		DeviceSns: json.RawMessage(`["SN001","SN002"]`),
		Status:    ConfigTaskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.tasks[id] = t
	return t
}

func handlerMustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests: Baselines
// ---------------------------------------------------------------------------

func TestHandler_ListBaselines(t *testing.T) {
	h, baselineRepo, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	seedHandlerBaseline(baselineRepo, uuid.New(), "Baseline-A", BaselineDraft)
	seedHandlerBaseline(baselineRepo, uuid.New(), "Baseline-B", BaselineActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/baselines?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[BaselineConfig]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestHandler_CreateBaseline(t *testing.T) {
	h, _, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	desc := "New baseline"
	body := CreateBaselineRequest{
		BaselineName: "New Baseline",
		Description:  &desc,
		Params:       json.RawMessage(`[{"param":"val"}]`),
		Status:       BaselineDraft,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/baselines", bytes.NewReader(handlerMustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp BaselineConfig
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "New Baseline", resp.BaselineName)
	assert.Equal(t, BaselineDraft, resp.Status)
	assert.NotEqual(t, uuid.Nil, resp.ID)
}

func TestHandler_GetBaseline(t *testing.T) {
	h, baselineRepo, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	blID := uuid.New()
	seedHandlerBaseline(baselineRepo, blID, "Get-Baseline", BaselineActive)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/baselines/"+blID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp BaselineConfig
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, blID, resp.ID)
	assert.Equal(t, "Get-Baseline", resp.BaselineName)
}

func TestHandler_GetBaseline_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/baselines/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	// Service returns errHandlerBaselineNotFound which is a plain error,
	// so HTTPStatusFromError maps it to 500 (internal server error).
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UpdateBaseline(t *testing.T) {
	h, baselineRepo, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	blID := uuid.New()
	seedHandlerBaseline(baselineRepo, blID, "Old-Baseline", BaselineDraft)

	desc := "Updated desc"
	body := UpdateBaselineRequest{
		BaselineName: "Updated-Baseline",
		Description:  &desc,
		Params:       json.RawMessage(`[{"updated":true}]`),
		Status:       BaselineActive,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/baselines/"+blID.String(), bytes.NewReader(handlerMustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp BaselineConfig
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Updated-Baseline", resp.BaselineName)
	assert.Equal(t, BaselineActive, resp.Status)
}

func TestHandler_DeleteBaseline(t *testing.T) {
	h, baselineRepo, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	blID := uuid.New()
	seedHandlerBaseline(baselineRepo, blID, "ToDelete", BaselineDraft)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/baselines/"+blID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify baseline removed.
	_, exists := baselineRepo.baselines[blID]
	assert.False(t, exists)
}

// ---------------------------------------------------------------------------
// Tests: Config Tasks
// ---------------------------------------------------------------------------

func TestHandler_CreateConfigTask(t *testing.T) {
	h, _, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	body := CreateConfigTaskRequest{
		TaskName:   "Batch Config Task",
		TaskType:   ConfigTaskBatchConfig,
		DeviceSns:  json.RawMessage(`["SN001","SN002"]`),
		TotalCount: 2,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/tasks", bytes.NewReader(handlerMustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp ConfigTask
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Batch Config Task", resp.TaskName)
	assert.Equal(t, ConfigTaskBatchConfig, resp.TaskType)
	assert.Equal(t, ConfigTaskPending, resp.Status)
	assert.Equal(t, 0, resp.Progress)
	assert.NotEqual(t, uuid.Nil, resp.ID)
}

func TestHandler_ListConfigTasks(t *testing.T) {
	h, _, taskRepo, _ := newTestHandler()
	router := setupHandlerRouter(h)

	seedHandlerConfigTask(taskRepo, uuid.New(), "Task-A", ConfigTaskParamSync)
	seedHandlerConfigTask(taskRepo, uuid.New(), "Task-B", ConfigTaskBaselineApply)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/tasks?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[ConfigTask]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

// ---------------------------------------------------------------------------
// Tests: Neighbors
// ---------------------------------------------------------------------------

func TestHandler_ListNeighbors(t *testing.T) {
	h, _, _, neighborRepo := newTestHandler()
	router := setupHandlerRouter(h)

	neighborRepo.neighbors = []NeighborParam{
		{
			ID:           uuid.New(),
			SourceCellID: "CELL-001",
			TargetCellID: "CELL-002",
			NeighborType: NeighborIntraFreq,
			Params:       json.RawMessage(`{}`),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           uuid.New(),
			SourceCellID: "CELL-001",
			TargetCellID: "CELL-003",
			NeighborType: NeighborInterFreq,
			Params:       json.RawMessage(`{}`),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/neighbors?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[NeighborParam]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

// ---------------------------------------------------------------------------
// Tests: Bad Requests
// ---------------------------------------------------------------------------

func TestHandler_CreateBaseline_BadRequest(t *testing.T) {
	h, _, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	// Missing required field (baseline_name).
	body := map[string]string{
		"description": "incomplete",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/baselines", bytes.NewReader(handlerMustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateConfigTask_BadRequest(t *testing.T) {
	h, _, _, _ := newTestHandler()
	router := setupHandlerRouter(h)

	// Missing required fields (task_name, task_type).
	body := map[string]string{
		"message": "incomplete",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/tasks", bytes.NewReader(handlerMustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
