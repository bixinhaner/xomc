package mml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/task"
)

// ---------------------------------------------------------------------------
// Function-field mock repositories
// ---------------------------------------------------------------------------

type hCmdRepo struct {
	ListFn          func(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error)
	GetByIDFn       func(ctx context.Context, id uuid.UUID) (*MMLCommand, error)
	GetByCodeFn     func(ctx context.Context, code string) (*MMLCommand, error)
	ListByGroupIDFn func(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error)
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
func (m *hCmdRepo) ListByGroupID(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error) {
	if m.ListByGroupIDFn != nil {
		return m.ListByGroupIDFn(ctx, groupID)
	}
	return nil, nil
}

type hScriptRepo struct {
	CreateFn          func(ctx context.Context, script *MMLScript) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*MMLScript, error)
	UpdateFn          func(ctx context.Context, script *MMLScript) error
	UpdateLifecycleFn func(ctx context.Context, script *MMLScript) error
	UpdateLastRunFn   func(ctx context.Context, id uuid.UUID, status string, at time.Time) error
	DeleteFn          func(ctx context.Context, id uuid.UUID) error
	ListFn            func(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error)
}

func (m *hScriptRepo) Create(ctx context.Context, script *MMLScript) error {
	return m.CreateFn(ctx, script)
}
func (m *hScriptRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *hScriptRepo) NameExistsForCreator(context.Context, string, string, *uuid.UUID) (bool, error) {
	return false, nil
}
func (m *hScriptRepo) Update(ctx context.Context, script *MMLScript) error {
	return m.UpdateFn(ctx, script)
}
func (m *hScriptRepo) UpdateLifecycle(ctx context.Context, script *MMLScript) error {
	if m.UpdateLifecycleFn != nil {
		return m.UpdateLifecycleFn(ctx, script)
	}
	return nil
}
func (m *hScriptRepo) UpdateLastRun(ctx context.Context, id uuid.UUID, status string, at time.Time) error {
	if m.UpdateLastRunFn != nil {
		return m.UpdateLastRunFn(ctx, id, status, at)
	}
	return nil
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
func (m *hTaskRepo) GetByRequestID(context.Context, string, string) (*MMLTask, error) {
	return nil, commonerrors.ErrNotFound
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

func (m *hTaskRepo) IncrementStats(ctx context.Context, id uuid.UUID, successDelta, failedDelta int) error {
	return nil
}

func (m *hTaskRepo) ListByScriptID(ctx context.Context, scriptID uuid.UUID, req model.ListRequest) (*model.ListResponse[MMLTask], error) {
	return model.NewListResponse([]MMLTask{}, 0, req.Page, req.PageSize), nil
}

func (m *hTaskRepo) GetActiveByScriptID(context.Context, uuid.UUID) (*MMLTask, error) {
	return nil, commonerrors.ErrNotFound
}

func (m *hTaskRepo) UpdateExportAggregate(ctx context.Context, id uuid.UUID, objectKey string, t time.Time) error {
	return nil
}

func (m *hTaskRepo) UpdateExportDevice(ctx context.Context, id uuid.UUID, deviceSN, objectKey string, t time.Time) error {
	return nil
}

type hCustomCommandRepo struct {
	CreateFn     func(ctx context.Context, tmpl *MMLCustomCommand) error
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error)
	UpdateFn     func(ctx context.Context, tmpl *MMLCustomCommand) error
	DeleteFn     func(ctx context.Context, id uuid.UUID) error
	ListFn       func(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error)
	NameExistsFn func(ctx context.Context, ownerID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
	PublicNameFn func(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error)
}

func (m *hCustomCommandRepo) Create(ctx context.Context, tmpl *MMLCustomCommand) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, tmpl)
	}
	return nil
}
func (m *hCustomCommandRepo) GetByID(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *hCustomCommandRepo) Update(ctx context.Context, tmpl *MMLCustomCommand) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, tmpl)
	}
	return nil
}
func (m *hCustomCommandRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *hCustomCommandRepo) List(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}
func (m *hCustomCommandRepo) NameExistsForPrivate(ctx context.Context, ownerID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	if m.NameExistsFn != nil {
		return m.NameExistsFn(ctx, ownerID, name, excludeID)
	}
	return false, nil
}
func (m *hCustomCommandRepo) NameExistsForPublic(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	if m.PublicNameFn != nil {
		return m.PublicNameFn(ctx, name, excludeID)
	}
	return false, nil
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/commands?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MMLCommand]
	response.DecodeData(t, w.Body, &resp)
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
				ID:          cmdID,
				CommandName: "Query Cell",
				CommandCode: "LST CELL",
				Category:    "query",
				Description: "List cell information",
				RPCMethod:   "GetParameterValues",
				TargetPaths: []string{"Device.Services.FAPService.{i}.CellConfig.{i}"},
				CreatedAt:   now,
			}, nil
		},
	}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/commands/"+cmdID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MMLCommand
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, cmdID, resp.ID)
	assert.Equal(t, "LST CELL", resp.CommandCode)
	assert.Equal(t, "GetParameterValues", resp.RPCMethod)
}

func TestHandler_GetCommandParamPaths(t *testing.T) {
	cmdID := uuid.New()
	cmdRepo := &hCmdRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLCommand, error) {
			assert.Equal(t, cmdID, id)
			return &MMLCommand{
				ID:            cmdID,
				CommandCode:   "MOD CELL",
				OperationType: "MOD",
				TargetPaths: []string{
					"Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF",
				},
			}, nil
		},
	}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/commands/"+cmdID.String()+"/param-paths", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var data CommandParamPathsResponse
	response.DecodeData(t, w.Body, &data)
	assert.Equal(t, "MOD CELL", data.CommandCode)
	assert.Equal(t, "MOD", data.OperationType)
	// 000090 后 supported_operations 由 operation_type 派生（单值数组），label=path 兜底
	assert.Equal(t, []string{"MOD"}, data.SupportedOperations)
	require.Len(t, data.ParamPaths, 1)
	assert.Equal(t, "Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF", data.ParamPaths[0].Path)
	assert.Equal(t, data.ParamPaths[0].Path, data.ParamPaths[0].Label)
	assert.True(t, data.ParamPaths[0].Writable, "MOD operation is writable")
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
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
	response.DecodeData(t, w.Body, &resp)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, TaskPending, resp.Status)
	assert.Equal(t, "Query cells", resp.TaskName)
	assert.Equal(t, []string{"SN-001", "SN-002"}, resp.DeviceSNs)
}

// TestHandler_ExecuteGroup_NotFound 覆盖 issue #125-mml 问题 2：
// POST /mml/groups/:id/execute 对不存在的 group 应返回 404 + 如实文案，
// 而非旧的 500 + 误导性「group_id required」。两条路径：
//   - 全零 UUID（合法格式但绝不对应真实 group）
//   - 随机 UUID 但 group 下无任何命令（Service 无独立 group 仓储,同归 not-found）
func TestHandler_ExecuteGroup_NotFound(t *testing.T) {
	cases := []struct {
		name    string
		groupID string
		// listFn 控制 ListByGroupID 返回（仅随机 UUID 用例触达）。
		listFn func(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error)
	}{
		{
			name:    "nil uuid",
			groupID: uuid.Nil.String(),
			listFn:  func(_ context.Context, _ uuid.UUID) ([]MMLCommand, error) { return nil, nil },
		},
		{
			name:    "unknown group has no commands",
			groupID: uuid.New().String(),
			listFn:  func(_ context.Context, _ uuid.UUID) ([]MMLCommand, error) { return []MMLCommand{}, nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmdRepo := &hCmdRepo{ListByGroupIDFn: tc.listFn}
			scriptRepo := &hScriptRepo{}
			taskRepo := &hTaskRepo{}

			logger := zap.NewNop()
			svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
			h := NewHandler(svc, logger)
			router := setupMMLRouter(h)

			body := ExecuteGroupHTTPRequest{DeviceSNs: []string{"SMK-NO-SUCH-DEV"}}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost,
				"/api/v1/mml/groups/"+tc.groupID+"/execute",
				bytes.NewReader(mustMarshalMML(t, body)))
			r.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, r)

			// 精确 404（非 500）。
			assert.Equal(t, http.StatusNotFound, w.Code,
				"not-found group 应映射 404，实际 %d body=%s", w.Code, w.Body.String())
			// 文案如实：含 not found，不得误导为缺参 group_id required。
			respBody := w.Body.String()
			assert.Contains(t, respBody, "not found",
				"文案应如实告知 group not found，实际 %s", respBody)
			assert.NotContains(t, respBody, "group_id required",
				"不得保留误导性的 group_id required 文案")
		})
	}
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/scripts?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MMLScript]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Batch Query Script", resp.Items[0].ScriptName)
}

func TestHandler_CreateScriptRouteRemoved(t *testing.T) {
	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{
		CreateFn: func(_ context.Context, _ *MMLScript) error {
			t.Fatal("legacy POST /mml/scripts route must not call CreateScript")
			return nil
		},
	}
	taskRepo := &hTaskRepo{}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts", bytes.NewReader([]byte(`{"script_name":"legacy"}`)))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mml/scripts/"+scriptID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/tasks?page=1&page_size=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[MMLTask]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "Batch Query", resp.Items[0].TaskName)
	assert.Equal(t, TaskCompleted, resp.Items[0].Status)
}

func TestHandler_ListTasksBindsTaskOrigin(t *testing.T) {
	tests := []struct {
		name       string
		queryValue string
		want       TaskOrigin
	}{
		{name: "script", queryValue: "script", want: TaskOriginScript},
		{name: "console", queryValue: "console", want: TaskOriginConsole},
		{name: "localized console", queryValue: "控制台执行", want: TaskOriginConsole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &hTaskRepo{
				ListFn: func(_ context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
					require.NotNil(t, filter.TaskOrigin)
					assert.Equal(t, tt.want, *filter.TaskOrigin)
					return model.NewListResponse([]MMLTask{}, 0, 1, 20), nil
				},
			}

			logger := zap.NewNop()
			svc := NewService(&hCmdRepo{}, &hScriptRepo{}, taskRepo, &hCustomCommandRepo{}, nil, logger)
			h := NewHandler(svc, logger)
			router := setupMMLRouter(h)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/tasks?page=1&page_size=20&task_origin="+url.QueryEscape(tt.queryValue), nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestHandler_ListTasksBindsScriptName(t *testing.T) {
	taskRepo := &hTaskRepo{
		ListFn: func(_ context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
			require.NotNil(t, filter.ScriptName)
			assert.Equal(t, "巡检脚本", *filter.ScriptName)
			return model.NewListResponse([]MMLTask{}, 0, 1, 20), nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(&hCmdRepo{}, &hScriptRepo{}, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/tasks?page=1&page_size=20&script_name="+url.QueryEscape("巡检脚本"), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListTasksRejectsInvalidTaskOrigin(t *testing.T) {
	taskRepo := &hTaskRepo{
		ListFn: func(_ context.Context, _ TaskFilter) (*model.ListResponse[MMLTask], error) {
			t.Fatal("List must not be called for invalid task_origin")
			return nil, nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(&hCmdRepo{}, &hScriptRepo{}, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mml/tasks?page=1&page_size=20&task_origin=bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/tasks/"+taskID.String()+"/start", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MMLTask
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, taskID, resp.ID)
}

func TestHandler_StartTask_AdmissionDeniedReturnsConflict(t *testing.T) {
	taskID := uuid.New()
	taskRepo := &hTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			return &MMLTask{
				ID:           id,
				Status:       TaskPending,
				ExecuteType:  ExecuteImmediate,
				DeviceSNs:    []string{"SN-REJECTED"},
				TotalDevices: 1,
				Commands: []map[string]interface{}{{
					"command_code": "REBOOT",
					"rpc_method":   "Reboot",
				}},
			}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ TaskStatus) error { return nil },
		UpdateFn:       func(_ context.Context, _ *MMLTask) error { return nil },
	}

	logger := zap.NewNop()
	svc := NewService(&hCmdRepo{}, &hScriptRepo{}, taskRepo, &hCustomCommandRepo{}, nil, logger)
	svc.SetFanouter(NewFanouter(&stubDeviceTaskCreator{
		err: fmt.Errorf("device SN-REJECTED: %w: normal_tasks_frozen_by_access_state", task.ErrTaskAdmissionDenied),
	}, nil, nil, nil, logger))
	router := setupMMLRouter(NewHandler(svc, logger))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/tasks/"+taskID.String()+"/start", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
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
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mml/tasks/"+taskID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response.DecodeData(t, w.Body, nil)
}

func TestHandler_DeleteTask_RunningReturnsConflict(t *testing.T) {
	taskID := uuid.New()

	cmdRepo := &hCmdRepo{}
	scriptRepo := &hScriptRepo{}
	taskRepo := &hTaskRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*MMLTask, error) {
			assert.Equal(t, taskID, id)
			return &MMLTask{ID: id, Status: TaskRunning}, nil
		},
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			t.Fatalf("running task %s must not be deleted", id)
			return nil
		},
	}

	logger := zap.NewNop()
	svc := NewService(cmdRepo, scriptRepo, taskRepo, &hCustomCommandRepo{}, nil, logger)
	h := NewHandler(svc, logger)
	router := setupMMLRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mml/tasks/"+taskID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "cannot delete a running task")
}
