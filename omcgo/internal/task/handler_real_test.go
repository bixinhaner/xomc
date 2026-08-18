package task

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 这个文件用真实 *Handler + 真实 *TaskService，但 TaskService 的 repo=nil。
// 仅适合测试不会触达 repo 的 handler 代码路径——主要是参数校验、
// queue 命中分支以及 wrong-status 分支。

// newRealHandler 构造真实 Handler，queue 用 miniredis 后端。
func newRealHandler(t *testing.T) (*Handler, *RedisTaskQueue, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	q := NewRedisTaskQueue(client)
	svc := &TaskService{queue: q, repo: nil, logger: zap.NewNop()}
	h := NewHandler(svc)
	return h, q, m
}

func setupRealRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r
}

// ---- 参数校验失败路径 ----

func TestHandler_CreateTask_MissingDeviceSN(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	body := `{"method":"Reboot"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateTask_BadJSON(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	body := `{not-json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN-A", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateTask_AdmissionDeniedReturnsConflict(t *testing.T) {
	h, _, _ := newRealHandler(t)
	h.service.SetAdmissionGuard(TaskAdmissionGuardFunc(func(context.Context, TaskAdmissionRequest) (bool, string, error) {
		return false, "device is not accepted", nil
	}))
	r := setupRealRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN-REVIEW", bytes.NewBufferString(`{"method":"Reboot"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), ErrTaskAdmissionDenied.Error())
}

func TestHandler_BatchCreate_AdmissionDeniedReturnsConflict(t *testing.T) {
	h, _, _ := newRealHandler(t)
	h.service.SetAdmissionGuard(TaskAdmissionGuardFunc(func(context.Context, TaskAdmissionRequest) (bool, string, error) {
		return false, "device is not accepted", nil
	}))
	r := setupRealRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks/batch?device_sn=SN-REVIEW", bytes.NewBufferString(`[{"method":"Reboot"}]`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), ErrTaskAdmissionDenied.Error())
}

func TestHandler_GetTaskHistory_MissingDeviceSN(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetPendingTasks_MissingDeviceSN(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/pending", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTaskStats_MissingDeviceSN(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BatchCreate_MissingDeviceSN(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	body := `[{"method":"Reboot"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BatchCreate_BadJSON(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	body := `not-json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks/batch?device_sn=SN-X", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- 真正命中 queue 的成功路径 ----

func TestHandler_GetTask_FoundInQueue(t *testing.T) {
	h, q, _ := newRealHandler(t)
	r := setupRealRouter(h)

	tk := newTaskForQueue("t-h-get", "SN-HG", "Reboot")
	require.NoError(t, q.Push(context.Background(), tk))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/t-h-get", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "t-h-get", data["id"])
}

// ---- wrong-status cancel 路径（不调用 repo） ----

func TestHandler_CancelTask_WrongStatus(t *testing.T) {
	h, q, _ := newRealHandler(t)
	r := setupRealRouter(h)
	ctx := context.Background()

	tk := newTaskForQueue("t-h-can", "SN-HC", "Reboot")
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.MarkTaskSent(ctx, "t-h-can", "cwmp-h-1"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/tasks/t-h-can", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// service.CancelTask 返回 "cannot cancel..." → handler 返回 500 InternalServerError
	// （只有 "task not found" 才映射 404）
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- PurgeOldTasks 测试 retention_days 解析路径（管理员接口）----
// 这个会调 repo.PurgeOldTasks 所以 repo=nil 会 panic。我们改测 RegisterRoutes 注册了路由。

func TestHandler_RegisterRoutesRegistersAllPaths(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	routes := r.Routes()
	pathSet := map[string]bool{}
	for _, ri := range routes {
		pathSet[ri.Method+" "+ri.Path] = true
	}
	assert.True(t, pathSet["POST /api/v1/devices/tasks"])
	assert.True(t, pathSet["GET /api/v1/devices/tasks"])
	assert.True(t, pathSet["GET /api/v1/devices/tasks/pending"])
	assert.True(t, pathSet["GET /api/v1/devices/tasks/:task_id"])
	assert.True(t, pathSet["DELETE /api/v1/devices/tasks/:task_id"])
	assert.True(t, pathSet["GET /api/v1/devices/tasks/stats"])
	assert.True(t, pathSet["POST /api/v1/devices/tasks/batch"])
	assert.True(t, pathSet["POST /api/v1/devices/tasks/:task_id/retry"])
	assert.True(t, pathSet["POST /api/v1/tasks/purge"])
}

// ---- parseTimeParam 边界测试已在 handler_test.go 覆盖 ----

// ---- CreateTask: 缺少 method （bind required 校验）----

func TestHandler_CreateTask_MissingMethod(t *testing.T) {
	h, _, _ := newRealHandler(t)
	r := setupRealRouter(h)

	body := `{"params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN-MM", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- GetTaskHistory：device_sn 存在但 query 解析失败 — 这个分支较难触发，跳过 ----

// ---- 默认分页测试（query 校验通过但 service 会调 repo） ----
// → 调 service.GetTaskHistory 会 panic，所以这条路径不能完整跑。
