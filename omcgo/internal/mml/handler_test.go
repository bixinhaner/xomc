package mml

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

type hCmdRepo struct {
	ListFn      func(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error)
	GetByIDFn   func(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
	GetByCodeFn func(ctx context.Context, code string) (*MMLCommand, error)
}

func (m *hCmdRepo) List(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	return m.ListFn(ctx, filter)
}
func (m *hCmdRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *hCmdRepo) GetByCode(ctx context.Context, code string) (*MMLCommand, error) {
	return m.GetByCodeFn(ctx, code)
}

type hScriptRepo struct {
	CreateFn  func(ctx context.Context, script *MMLScript) error
	GetByIDFn func(ctx context.Context, id uuid.UUID) (*MMLScript, error)
	UpdateFn  func(ctx context.Context, script *MMLScript) error
	DeleteFn  func(ctx context.Context, id uuid.UUID) error
	ListFn    func(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error)
}

func (m *hScriptRepo) Create(ctx context.Context, script *MMLScript) error {
	return m.CreateFn(ctx, script)
}
func (m *hScriptRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *hScriptRepo) Update(ctx context.Context, script *MMLScript) error {
	return m.UpdateFn(ctx, script)
}
func (m *hScriptRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *hScriptRepo) List(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error) {
	return m.ListFn(ctx, filter)
}

type hTaskRepo struct {
	CreateFn       func(ctx context.Context, task *MMLTask) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*MMLTask, error)
	UpdateFn       func(ctx context.Context, task *MMLTask) error
	UpdateStatusFn func(ctx context.Context, id uuid.UUID, status TaskStatus) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	ListFn         func(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error)
}

func (m *hTaskRepo) Create(ctx context.Context, task *MMLTask) error {
	return m.CreateFn(ctx, task)
}
func (m *hTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *hTaskRepo) Update(ctx context.Context, task *MMLTask) error {
	return m.UpdateFn(ctx, task)
}
func (m *hTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}
func (m *hTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *hTaskRepo) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
	return m.ListFn(ctx, filter)
}

type hTemplateRepo struct {
	CreateFn  func(ctx context.Context, tmpl *MMLTemplate) error
	GetByIDFn func(ctx context.Context, id uuid.UUID) (*MMLTemplate, error)
	UpdateFn  func(ctx context.Context, tmpl *MMLTemplate) error
	DeleteFn  func(ctx context.Context, id uuid.UUID) error
	ListFn    func(ctx context.Context, filter TemplateFilter) (*model.ListResponse[MMLTemplate], error)
}

func (m *hTemplateRepo) Create(ctx context.Context, tmpl *MMLTemplate) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, tmpl)
	}
	return nil
}
func (m *hTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLTemplate, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *hTemplateRepo) Update(ctx context.Context, tmpl *MMLTemplate) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, tmpl)
	}
	return nil
}
func (m *hTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *hTemplateRepo) List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[MMLTemplate], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupMMLRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func mustMarshalMML(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ListCommands(t *testing.T) {
	now := time.Now()
	cmdID := uuid.New()

	cmdRepo := &hCmdRepo{
		ListFn: func(_ context.Context, _ CommandFilter) (*model.ListResponse[MMLCommand], error) {
			items := []MMLCommand{
				{
					ID:          cmdID,
					CommandName: "Query Cell",
					CommandCode: "LST CELL",
					Category:    "query",
					Description: "List cell information",
					RPCMethod:   "GetParameterValues",
					CreatedAt:   now,
				},
			}
			return model.NewListResponse(items, 1, 1, 20), nil
		},
	}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/commands?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MMLCommand]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "LST CELL", resp.Items[0].CommandCode)
	assert.Equal(t, "query", resp.Items[0].Category)
}

func TestHandler_GetCommand(t *testing.T) {
	now := time.Now()
	cmdID := uuid.New()

	cmdRepo := &hCmdRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLCommand, error) {
			return &MMLCommand{
				ID:            cmdID,
				CommandName:   "Query Cell",
				CommandCode:   "LST CELL",
				Category:      "query",
				Description:   "List cell information",
				RPCMethod:     "GetParameterValues",
				ParamTemplate: map[string]interface{}{"cell_id": ""},
				ProductTypes:  []string{"eNB", "gNB"},
				CreatedAt:     now,
			}, nil
		},
	}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/commands/"+cmdID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MMLCommand
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, cmdID, resp.ID)
	assert.Equal(t, "LST CELL", resp.CommandCode)
	assert.Equal(t, "GetParameterValues", resp.RPCMethod)
}

func TestHandler_Execute(t *testing.T) {
	cmdID := uuid.New()

	cmdRepo := &hCmdRepo{
		GetByCodeFn: func(_ context.Context, code string) (*MMLCommand, error) {
			return &MMLCommand{
				ID:          cmdID,
				CommandCode: code,
				RPCMethod:   "GetParameterValues",
			}, nil
		},
	}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		CreateFn: func(_ context.Context, task *MMLTask) error {
			task.ID = uuid.New()
			task.CreatedAt = time.Now()
			task.UpdatedAt = time.Now()
			return nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	body := ExecuteHTTPRequest{
		CommandCode: "LST CELL",
		DeviceSNs:   []string{"SN-001", "SN-002"},
		Parameters:  map[string]interface{}{"cell_id": "1"},
		TaskName:    "Query cells",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/mml/execute", bytes.NewReader(mustMarshalMML(t, body)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp MMLTask
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, TaskPending, resp.Status)
	assert.Equal(t, "Query cells", resp.TaskName)
	assert.Equal(t, []string{"SN-001", "SN-002"}, resp.DeviceSNs)
}

func TestHandler_ListScripts(t *testing.T) {
	now := time.Now()
	scriptID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{
		ListFn: func(_ context.Context, _ ScriptFilter) (*model.ListResponse[MMLScript], error) {
			items := []MMLScript{
				{
					ID:          scriptID,
					ScriptName:  "Batch Query Script",
					Description: "Batch cell query",
					Content:     "LST CELL;\nDSP CELLALGO;",
					DeviceType:  "eNB",
					Creator:     "admin",
					Tags:        []string{"batch", "query"},
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}
			return model.NewListResponse(items, 1, 1, 20), nil
		},
	}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/scripts?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MMLScript]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Batch Query Script", resp.Items[0].ScriptName)
	assert.Equal(t, "eNB", resp.Items[0].DeviceType)
}

func TestHandler_CreateScript(t *testing.T) {
	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{
		CreateFn: func(_ context.Context, script *MMLScript) error {
			script.ID = uuid.New()
			script.CreatedAt = time.Now()
			script.UpdatedAt = time.Now()
			return nil
		},
	}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	body := CreateScriptRequest{
		ScriptName:  "New Script",
		Description: "A new MML script",
		Content:     "LST CELL;",
		DeviceType:  "gNB",
		Tags:        []string{"5g", "cell"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts", bytes.NewReader(mustMarshalMML(t, body)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp MMLScript
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "New Script", resp.ScriptName)
	assert.Equal(t, "LST CELL;", resp.Content)
	assert.Equal(t, "gNB", resp.DeviceType)
	assert.Equal(t, []string{"5g", "cell"}, resp.Tags)
}

func TestHandler_DeleteScript(t *testing.T) {
	scriptID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, scriptID, id)
			return nil
		},
	}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mml/scripts/"+scriptID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestHandler_ListTasks(t *testing.T) {
	now := time.Now()
	taskID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		ListFn: func(_ context.Context, _ TaskFilter) (*model.ListResponse[MMLTask], error) {
			items := []MMLTask{
				{
					ID:        taskID,
					TaskName:  "Batch Query",
					DeviceSNs: []string{"SN-001"},
					Commands:  []map[string]interface{}{{"command_code": "LST CELL"}},
					Status:    TaskCompleted,
					Results:   []map[string]interface{}{{"status": "success"}},
					Creator:   "admin",
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			return model.NewListResponse(items, 1, 1, 20), nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/tasks?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MMLTask]
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Batch Query", resp.Items[0].TaskName)
	assert.Equal(t, TaskCompleted, resp.Items[0].Status)
}

// ---- Task control handler tests ----

func TestHandler_StartTask(t *testing.T) {
	taskID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			return &MMLTask{ID: id, Status: TaskPending}, nil
		},
		UpdateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus) error {
			assert.Equal(t, TaskRunning, status)
			return nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/tasks/"+taskID.String()+"/start", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MMLTask
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, taskID, resp.ID)
}

func TestHandler_PauseTask(t *testing.T) {
	taskID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			return &MMLTask{ID: id, Status: TaskRunning}, nil
		},
		UpdateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus) error {
			assert.Equal(t, TaskPaused, status)
			return nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/tasks/"+taskID.String()+"/pause", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CancelTask(t *testing.T) {
	taskID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			return &MMLTask{ID: id, Status: TaskRunning}, nil
		},
		UpdateStatusFn: func(_ context.Context, id uuid.UUID, status TaskStatus) error {
			assert.Equal(t, TaskCancelled, status)
			return nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/tasks/"+taskID.String()+"/cancel", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_DeleteTask(t *testing.T) {
	taskID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			return &MMLTask{ID: id, Status: TaskCompleted}, nil
		},
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, taskID, id)
			return nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hTemplateRepo{}, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mml/tasks/"+taskID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
