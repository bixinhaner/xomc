package ops

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

	"github.com/omcgo/omcgo/internal/core/model"
)

// ---------------------------------------------------------------------------
// Function-field mock repositories
// ---------------------------------------------------------------------------

type hTemplateRepo struct {
	CreateFn            func(ctx context.Context, tmpl *OpsTemplate) error
	GetByIDFn           func(ctx context.Context, id uuid.UUID) (*OpsTemplate, error)
	UpdateFn            func(ctx context.Context, tmpl *OpsTemplate) error
	DeleteFn            func(ctx context.Context, id uuid.UUID) error
	ListFn              func(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error)
	IncrementUseCountFn func(ctx context.Context, id uuid.UUID) error
}

func (m *hTemplateRepo) Create(ctx context.Context, tmpl *OpsTemplate) error {
	return m.CreateFn(ctx, tmpl)
}
func (m *hTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*OpsTemplate, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *hTemplateRepo) Update(ctx context.Context, tmpl *OpsTemplate) error {
	return m.UpdateFn(ctx, tmpl)
}
func (m *hTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *hTemplateRepo) List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error) {
	return m.ListFn(ctx, filter)
}
func (m *hTemplateRepo) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	if m.IncrementUseCountFn != nil {
		return m.IncrementUseCountFn(ctx, id)
	}
	return nil
}

type mockOpsTaskRepo struct {
	CreateFn       func(ctx context.Context, task *OpsTask) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*OpsTask, error)
	UpdateStatusFn func(ctx context.Context, task *OpsTask) error
	ListFn         func(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error)
}

func (m *mockOpsTaskRepo) Create(ctx context.Context, task *OpsTask) error {
	return m.CreateFn(ctx, task)
}
func (m *mockOpsTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*OpsTask, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockOpsTaskRepo) UpdateStatus(ctx context.Context, task *OpsTask) error {
	return m.UpdateStatusFn(ctx, task)
}
func (m *mockOpsTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error) {
	return m.ListFn(ctx, filter)
}

type mockCmdRecordRepo struct {
	CreateFn func(ctx context.Context, record *OpsCommandRecord) error
	ListFn   func(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error)
}

func (m *mockCmdRecordRepo) Create(ctx context.Context, record *OpsCommandRecord) error {
	return m.CreateFn(ctx, record)
}
func (m *mockCmdRecordRepo) List(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error) {
	return m.ListFn(ctx, filter)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupOpsRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func mustMarshalOps(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func newOpsTestHandler(
	tmplRepo *hTemplateRepo,
	taskRepo *mockOpsTaskRepo,
	cmdRepo *mockCmdRecordRepo,
) *Handler {
	logger := zap.NewNop()
	svc := NewService(tmplRepo, taskRepo, cmdRepo, logger)
	return NewHandler(svc, logger)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListTemplates(t *testing.T) {
	now := time.Now()
	tmplID := uuid.New()

	tmplRepo := &hTemplateRepo{
		ListFn: func(_ context.Context, _ TemplateFilter) (*model.ListResponse[OpsTemplate], error) {
			items := []OpsTemplate{
				{
					ID:                tmplID,
					TemplateName:      "Firmware Upgrade",
					Description:       "Standard firmware upgrade procedure",
					Category:          "upgrade",
					TargetDeviceTypes: json.RawMessage(`["eNB","gNB"]`),
					Steps:             json.RawMessage(`[{"name":"backup"},{"name":"upload"},{"name":"reboot"}]`),
					EstimatedDuration: 300,
					Creator:           "admin",
					UseCount:          5,
					Tags:              json.RawMessage(`["firmware","upgrade"]`),
					CreatedAt:         now,
					UpdatedAt:         now,
				},
			}
			return model.NewListResponse(items, 1, 1, 20), nil
		},
	}
	taskRepo := &mockOpsTaskRepo{}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/templates?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[OpsTemplate]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Firmware Upgrade", resp.Items[0].TemplateName)
	assert.Equal(t, "upgrade", resp.Items[0].Category)
}

func TestHandler_CreateTemplate(t *testing.T) {
	tmplRepo := &hTemplateRepo{
		CreateFn: func(_ context.Context, tmpl *OpsTemplate) error {
			tmpl.ID = uuid.New()
			tmpl.CreatedAt = time.Now()
			tmpl.UpdatedAt = time.Now()
			return nil
		},
	}
	taskRepo := &mockOpsTaskRepo{}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	body := CreateTemplateRequest{
		TemplateName:      "Config Sync",
		Description:       "Synchronize configuration",
		Category:          "config",
		TargetDeviceTypes: json.RawMessage(`["eNB"]`),
		Steps:             json.RawMessage(`[{"name":"download"},{"name":"apply"}]`),
		EstimatedDuration: 120,
		Creator:           "admin",
		Tags:              json.RawMessage(`["config"]`),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ops/templates", bytes.NewReader(mustMarshalOps(t, body)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp OpsTemplate
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "Config Sync", resp.TemplateName)
	assert.Equal(t, "config", resp.Category)
	assert.Equal(t, 0, resp.UseCount)
}

func TestHandler_GetTemplate(t *testing.T) {
	now := time.Now()
	tmplID := uuid.New()

	tmplRepo := &hTemplateRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTemplate, error) {
			return &OpsTemplate{
				ID:                tmplID,
				TemplateName:      "Firmware Upgrade",
				Description:       "Standard firmware upgrade procedure",
				Category:          "upgrade",
				TargetDeviceTypes: json.RawMessage(`["eNB","gNB"]`),
				Steps:             json.RawMessage(`[{"name":"backup"},{"name":"upload"}]`),
				EstimatedDuration: 300,
				Creator:           "admin",
				UseCount:          10,
				Tags:              json.RawMessage(`["firmware"]`),
				CreatedAt:         now,
				UpdatedAt:         now,
			}, nil
		},
	}
	taskRepo := &mockOpsTaskRepo{}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/templates/"+tmplID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp OpsTemplate
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, tmplID, resp.ID)
	assert.Equal(t, "Firmware Upgrade", resp.TemplateName)
	assert.Equal(t, 10, resp.UseCount)
}

func TestHandler_DeleteTemplate(t *testing.T) {
	tmplID := uuid.New()

	tmplRepo := &hTemplateRepo{
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, tmplID, id)
			return nil
		},
	}
	taskRepo := &mockOpsTaskRepo{}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/ops/templates/"+tmplID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestHandler_CreateTask(t *testing.T) {
	tmplID := uuid.New()

	tmplRepo := &hTemplateRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTemplate, error) {
			return &OpsTemplate{
				ID:           tmplID,
				TemplateName: "Firmware Upgrade",
				Steps:        json.RawMessage(`[{"name":"backup"},{"name":"upload"},{"name":"reboot"}]`),
			}, nil
		},
		IncrementUseCountFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}
	taskRepo := &mockOpsTaskRepo{
		CreateFn: func(_ context.Context, task *OpsTask) error {
			task.ID = uuid.New()
			task.CreatedAt = time.Now()
			task.UpdatedAt = time.Now()
			return nil
		},
	}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	body := CreateTaskRequest{
		TaskName:   "Upgrade Batch 1",
		TemplateID: &tmplID,
		DeviceSNs:  json.RawMessage(`["SN-001","SN-002","SN-003"]`),
		Creator:    "admin",
		Message:    "Batch firmware upgrade",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ops/tasks", bytes.NewReader(mustMarshalOps(t, body)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp OpsTask
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, OpsTaskPending, resp.Status)
	assert.Equal(t, "Upgrade Batch 1", resp.TaskName)
	assert.Equal(t, 3, resp.TotalSteps)
	assert.Equal(t, 3, resp.TotalCount)
	assert.Equal(t, 0, resp.CurrentStep)
	assert.Equal(t, 0, resp.Progress)
}

func TestHandler_CancelTask(t *testing.T) {
	taskID := uuid.New()

	tmplRepo := &hTemplateRepo{}
	taskRepo := &mockOpsTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID:       taskID,
				TaskName: "Running Task",
				Status:   OpsTaskRunning,
			}, nil
		},
		UpdateStatusFn: func(_ context.Context, task *OpsTask) error {
			assert.Equal(t, OpsTaskCancelled, task.Status)
			assert.NotNil(t, task.CompletedAt)
			return nil
		},
	}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/tasks/"+taskID.String()+"/cancel", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", resp["status"])
}

func TestHandler_PauseTask(t *testing.T) {
	taskID := uuid.New()

	tmplRepo := &hTemplateRepo{}
	taskRepo := &mockOpsTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID:       taskID,
				TaskName: "Running Task",
				Status:   OpsTaskRunning,
			}, nil
		},
		UpdateStatusFn: func(_ context.Context, task *OpsTask) error {
			assert.Equal(t, OpsTaskPaused, task.Status)
			return nil
		},
	}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/tasks/"+taskID.String()+"/pause", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "paused", resp["status"])
}

func TestHandler_ResumeTask(t *testing.T) {
	taskID := uuid.New()

	tmplRepo := &hTemplateRepo{}
	taskRepo := &mockOpsTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*OpsTask, error) {
			return &OpsTask{
				ID:       taskID,
				TaskName: "Paused Task",
				Status:   OpsTaskPaused,
			}, nil
		},
		UpdateStatusFn: func(_ context.Context, task *OpsTask) error {
			assert.Equal(t, OpsTaskRunning, task.Status)
			return nil
		},
	}
	cmdRepo := &mockCmdRecordRepo{}

	h := newOpsTestHandler(tmplRepo, taskRepo, cmdRepo)
	router := setupOpsRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/tasks/"+taskID.String()+"/resume", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "running", resp["status"])
}
