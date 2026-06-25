package adhoc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handlerStubRepo 仅实现 Create/List/Get/Cancel/Update，不依赖 DB。
type handlerStubRepo struct {
	mu     sync.Mutex
	tasks  map[uuid.UUID]*Task
	create func(CreateRequest) (uuid.UUID, error)
	cancel func(uuid.UUID) error
	get      func(uuid.UUID) (*Task, error)       // T-0194：注入既有任务（含 is_builtin/mode/technology）
	update   func(uuid.UUID, UpdateRequest) error // T-0194：捕获更新入参
	deleteFn func(uuid.UUID) error                // #392：注入删除结果（区分终态/内置/非终态）
}

func (s *handlerStubRepo) Create(_ context.Context, req CreateRequest) (uuid.UUID, error) {
	if s.create != nil {
		return s.create(req)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New()
	if s.tasks == nil {
		s.tasks = map[uuid.UUID]*Task{}
	}
	s.tasks[id] = &Task{
		ID: id, Name: req.Name, Mode: req.Mode, CronExpr: req.CronExpr,
		DeviceSNs: req.DeviceSNs, MetricPaths: req.MetricPaths, Granularities: req.Granularities,
		WindowStart: req.WindowStart, WindowEnd: req.WindowEnd, Status: StatusPending,
		Creator: req.Creator, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	return id, nil
}
func (s *handlerStubRepo) Get(_ context.Context, id uuid.UUID) (*Task, error) {
	if s.get != nil {
		return s.get(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok {
		return t, nil
	}
	return nil, ErrNotFound
}
func (s *handlerStubRepo) Update(_ context.Context, id uuid.UUID, req UpdateRequest) error {
	if s.update != nil {
		return s.update(id, req)
	}
	return nil
}
func (s *handlerStubRepo) List(_ context.Context, _ ListFilter) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Task{}
	for _, t := range s.tasks {
		out = append(out, *t)
	}
	return out, nil
}
func (s *handlerStubRepo) Cancel(_ context.Context, id uuid.UUID) error {
	if s.cancel != nil {
		return s.cancel(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok {
		t.Status = StatusCanceled
		return nil
	}
	return ErrNotFound
}
func (s *handlerStubRepo) Delete(_ context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; ok {
		delete(s.tasks, id)
		return nil
	}
	return ErrNotFound
}
func (s *handlerStubRepo) LockNextPending(context.Context, string) (*Task, error) { return nil, nil }
func (s *handlerStubRepo) UpdateStatus(context.Context, uuid.UUID, Status, *int, string) error {
	return nil
}
func (s *handlerStubRepo) InsertResults(context.Context, []ResultRow) error { return nil }
func (s *handlerStubRepo) NextRunSeq(context.Context, uuid.UUID) (int, error) { return 1, nil }
func (s *handlerStubRepo) InsertRun(context.Context, TaskRun) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *handlerStubRepo) FinishRun(context.Context, uuid.UUID, Status, int, string) error {
	return nil
}
func (s *handlerStubRepo) ListRuns(context.Context, uuid.UUID, int, int) ([]TaskRun, error) {
	return nil, nil
}

func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repo, nil, nil, nil)
	g := r.Group("")
	h.RegisterRoutes(g)
	return r
}

// newTestRouterWithUser 构造注入了用户身份的 test router（#652 权限测试用）。
func newTestRouterWithUser(repo Repository, username string, superAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("username", username)
		c.Set("is_super_admin", superAdmin)
		c.Next()
	})
	h := NewHandler(repo, nil, nil, nil)
	g := r.Group("")
	h.RegisterRoutes(g)
	return r
}

func Test_Handler_Create_Success(t *testing.T) {
	repo := &handlerStubRepo{}
	r := newTestRouter(repo)

	body := map[string]any{
		"name": "test", "mode": "oneshot",
		"device_sns":     []string{"S1"},
		"metric_paths":   []string{"M1"},
		"granularities":  []string{"hourly"},
		"window_start":   "2026-05-22T10:00:00Z",
		"window_end":     "2026-05-22T11:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/adhoc/tasks", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	dataMap, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, dataMap["id"])
}

func Test_Handler_Create_BadWindow(t *testing.T) {
	r := newTestRouter(&handlerStubRepo{})

	body := map[string]any{
		"name": "x", "mode": "oneshot",
		"device_sns": []string{"S1"}, "metric_paths": []string{"M1"}, "granularities": []string{"hourly"},
		"window_start": "2026-05-22T12:00:00Z", "window_end": "2026-05-22T11:00:00Z", // end<start
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pm/adhoc/tasks", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Handler_List_Empty(t *testing.T) {
	r := newTestRouter(&handlerStubRepo{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pm/adhoc/tasks", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total"])
}

func Test_Handler_Get_NotFound(t *testing.T) {
	r := newTestRouter(&handlerStubRepo{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pm/adhoc/tasks/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_Handler_Cancel_Conflict(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{ID: taskID, IsBuiltin: false, Creator: "anonymous"}, nil
		},
		cancel: func(_ uuid.UUID) error { return ErrTerminalState },
	}
	r := newTestRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+taskID.String(), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ── #392 删除端点 ─────────────────────────────────────────────────────────

// 成功路径：终态自建任务硬删，返回 200 + deleted=true，且任务从 stub map 消失。
func Test_Handler_Delete_TerminalSelfBuilt_Success(t *testing.T) {
	id := uuid.New()
	repo := &handlerStubRepo{
		tasks: map[uuid.UUID]*Task{
			id: {ID: id, Status: StatusSucceeded, IsBuiltin: false, Creator: "anonymous"},
		},
	}
	r := newTestRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+id.String()+"/definition", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, true, data["deleted"])
	// 任务确实从 map 删除。
	_, ok := repo.tasks[id]
	assert.False(t, ok)
}

// 失败路径之一：内置任务被拒，返回 403，行仍在。
func Test_Handler_Delete_Builtin_Forbidden(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{ID: taskID, IsBuiltin: true, Creator: "system"}, nil
		},
		deleteFn: func(_ uuid.UUID) error { return ErrBuiltinNotDeletable },
	}
	r := newTestRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+taskID.String()+"/definition", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// 失败路径之二：非终态任务被拒，返回 409 冲突，行仍在。
func Test_Handler_Delete_NonTerminal_Conflict(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{ID: taskID, IsBuiltin: false, Creator: "anonymous", Status: StatusRunning}, nil
		},
		deleteFn: func(_ uuid.UUID) error { return ErrNotTerminal },
	}
	r := newTestRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+uuid.New().String()+"/definition", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// 失败路径之三：不存在任务返回 404。
func Test_Handler_Delete_NotFound(t *testing.T) {
	repo := &handlerStubRepo{
		deleteFn: func(_ uuid.UUID) error { return ErrNotFound },
	}
	r := newTestRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+uuid.New().String()+"/definition", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── #652 权限控制 ─────────────────────────────────────────────────────────

// Update：非 owner 普通用户编辑他人自建任务 → 403。
func Test_Handler_Update_NonOwner_Forbidden(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{
				ID: taskID, Name: "other-user-task", Mode: ModeOneshot,
				Dimension: DimensionDevice, IsBuiltin: false, Creator: "alice",
				MetricPaths: []string{"M1"}, Granularities: []string{"hourly"},
				DeviceSNs: []string{"S1"},
			}, nil
		},
	}
	r := newTestRouterWithUser(repo, "bob", false) // bob 不是 owner

	body := map[string]any{
		"metric_paths":  []string{"M2"},
		"granularities": []string{"hourly"},
		"device_sns":    []string{"S1"},
		"window_start":  "2026-05-22T10:00:00Z",
		"window_end":    "2026-05-22T11:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/pm/adhoc/tasks/"+taskID.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Update：超管可编辑他人自建任务 → 通过。
func Test_Handler_Update_Admin_Allowed(t *testing.T) {
	taskID := uuid.New()
	var updated bool
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{
				ID: taskID, Name: "other-user-task", Mode: ModeOneshot,
				Dimension: DimensionDevice, IsBuiltin: false, Creator: "alice",
				MetricPaths: []string{"M1"}, Granularities: []string{"hourly"},
				DeviceSNs: []string{"S1"},
			}, nil
		},
		update: func(_ uuid.UUID, _ UpdateRequest) error {
			updated = true
			return nil
		},
	}
	r := newTestRouterWithUser(repo, "admin-user", true) // 超管

	body := map[string]any{
		"metric_paths":  []string{"M2"},
		"granularities": []string{"hourly"},
		"device_sns":    []string{"S1"},
		"window_start":  "2026-05-22T10:00:00Z",
		"window_end":    "2026-05-22T11:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/pm/adhoc/tasks/"+taskID.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, updated)
}

// Cancel：非 owner 取消他人自建任务 → 403。
func Test_Handler_Cancel_NonOwner_Forbidden(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{
				ID: taskID, IsBuiltin: false, Creator: "alice",
			}, nil
		},
	}
	r := newTestRouterWithUser(repo, "bob", false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+taskID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Delete：非 owner 删除他人自建任务 → 403。
func Test_Handler_Delete_NonOwner_Forbidden(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{
				ID: taskID, IsBuiltin: false, Creator: "alice", Status: StatusSucceeded,
			}, nil
		},
	}
	r := newTestRouterWithUser(repo, "bob", false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+taskID.String()+"/definition", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Cancel：owner 取消自己的任务 → 通过。
func Test_Handler_Cancel_Owner_Allowed(t *testing.T) {
	taskID := uuid.New()
	var cancelled bool
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{
				ID: taskID, IsBuiltin: false, Creator: "alice",
			}, nil
		},
		cancel: func(_ uuid.UUID) error {
			cancelled = true
			return nil
		},
	}
	r := newTestRouterWithUser(repo, "alice", false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+taskID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, cancelled)
}

// Cancel：内置任务不做归属权校验，直接走 repo 逻辑（内置任务的取消由 repo 层守门）。
func Test_Handler_Cancel_BuiltinTask_SkipsOwnerCheck(t *testing.T) {
	taskID := uuid.New()
	var cancelled bool
	repo := &handlerStubRepo{
		get: func(id uuid.UUID) (*Task, error) {
			return &Task{
				ID: taskID, IsBuiltin: true, Creator: "system",
			}, nil
		},
		cancel: func(_ uuid.UUID) error {
			cancelled = true
			return nil
		},
	}
	r := newTestRouterWithUser(repo, "bob", false) // 普通用户

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/pm/adhoc/tasks/"+taskID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, cancelled)
}
