package task

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHandler wraps a testableTaskService to provide handler methods
// that can be registered with gin for HTTP-level testing.
type testHandler struct {
	ts *testableTaskService
}

func newTestHandler() *testHandler {
	return &testHandler{ts: newTestableService()}
}

func (th *testHandler) setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")

	tasks := api.Group("/devices/tasks")
	{
		tasks.POST("", th.createTask)
		tasks.GET("/:task_id", th.getTask)
		tasks.DELETE("/:task_id", th.cancelTask)
		tasks.GET("/stats", th.getTaskStats)
	}

	return r
}

func (th *testHandler) createTask(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Bad Request",
			"details": "invalid input",
		})
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Bad Request",
			"details": "invalid input",
		})
		return
	}

	req.DeviceSN = deviceSN
	if req.Source == "" {
		req.Source = TaskSourceAPI
	}

	task, err := th.ts.CreateTask(c.Request.Context(), &req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": task,
	})
}

func (th *testHandler) getTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Bad Request",
		})
		return
	}

	task, err := th.ts.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error",
		})
		return
	}

	if task == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "Not Found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": task,
	})
}

func (th *testHandler) cancelTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Bad Request",
		})
		return
	}

	err := th.ts.CancelTask(c.Request.Context(), taskID)
	if err != nil {
		// 镜像生产 handler.CancelTask（#125）：用哨兵 errors.Is 判定 not-found，
		// 不再做易碎的精确字符串比对。
		if stderrors.Is(err, ErrTaskNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "Not Found",
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "task cancelled",
	})
}

func (th *testHandler) getTaskStats(c *gin.Context) {
	deviceSN := c.Query("device_sn")
	if deviceSN == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Bad Request",
		})
		return
	}

	stats, err := th.ts.GetTaskStats(c.Request.Context(), deviceSN)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"by_status": stats,
		},
	})
}

// --- Handler HTTP Tests ---

func Test_Handler_CreateTask_Success(t *testing.T) {
	th := newTestHandler()
	th.ts.repo.createFn = func(ctx context.Context, task *Task) error { return nil }
	th.ts.queue.pushFn = func(ctx context.Context, task *Task) error { return nil }

	router := th.setupRouter()

	body := `{"device_sn":"SN001","method":"GetParameterValues","params":{"names":["Device.DeviceInfo."]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "SN001", data["device_sn"])
	assert.Equal(t, "GetParameterValues", data["method"])
	assert.Equal(t, "pending", data["status"])
	assert.NotEmpty(t, data["id"])
}

func Test_Handler_CreateTask_MissingDeviceSN(t *testing.T) {
	th := newTestHandler()
	router := th.setupRouter()

	body := `{"method":"GetParameterValues"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Handler_CreateTask_MissingMethod(t *testing.T) {
	th := newTestHandler()
	router := th.setupRouter()

	// method is required per binding:"required"
	body := `{"params":{"names":["Device."]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Handler_CreateTask_InvalidJSON(t *testing.T) {
	th := newTestHandler()
	router := th.setupRouter()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_Handler_CreateTask_ServiceError(t *testing.T) {
	th := newTestHandler()
	th.ts.repo.createFn = func(ctx context.Context, task *Task) error {
		return fmt.Errorf("db error")
	}
	router := th.setupRouter()

	body := `{"device_sn":"SN001","method":"Reboot"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/tasks?device_sn=SN001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func Test_Handler_GetTask_Success(t *testing.T) {
	th := newTestHandler()
	expected := &Task{
		ID:       "task-123",
		DeviceSN: "SN001",
		Method:   "Reboot",
		Status:   TaskStatusPending,
		Source:   TaskSourceAPI,
	}
	th.ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		if taskID == "task-123" {
			return expected, nil
		}
		return nil, nil
	}

	router := th.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/task-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "task-123", data["id"])
	assert.Equal(t, "Reboot", data["method"])
}

func Test_Handler_GetTask_NotFound(t *testing.T) {
	th := newTestHandler()
	th.ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return nil, nil
	}
	th.ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		return nil, nil
	}

	router := th.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_Handler_CancelTask_Success(t *testing.T) {
	th := newTestHandler()

	pendingTask := &Task{ID: "task-123", DeviceSN: "SN001", Status: TaskStatusPending}
	th.ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		if taskID == "task-123" {
			return pendingTask, nil
		}
		return nil, nil
	}
	th.ts.queue.deleteFn = func(ctx context.Context, taskID string) error { return nil }
	th.ts.repo.updateFn = func(ctx context.Context, task *Task) error { return nil }

	router := th.setupRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/tasks/task-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "task cancelled", resp["message"])
}

func Test_Handler_CancelTask_NotFound(t *testing.T) {
	th := newTestHandler()
	th.ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return nil, nil
	}
	th.ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		return nil, nil
	}

	router := th.setupRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/devices/tasks/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func Test_Handler_GetTaskStats_Success(t *testing.T) {
	th := newTestHandler()

	th.ts.repo.countByStatusFn = func(ctx context.Context, deviceSN string) (map[TaskStatus]int64, error) {
		return map[TaskStatus]int64{
			TaskStatusPending:   5,
			TaskStatusCompleted: 10,
		}, nil
	}

	router := th.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/stats?device_sn=SN001", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	byStatus, ok := data["by_status"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(5), byStatus["pending"])
	assert.Equal(t, float64(10), byStatus["completed"])
}

func Test_Handler_GetTaskStats_MissingDeviceSN(t *testing.T) {
	th := newTestHandler()
	router := th.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/tasks/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test parseTimeParam utility function
func Test_parseTimeParam(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
	}{
		{"empty string returns nil", "", true},
		{"invalid time returns nil", "not-a-time", true},
		{"valid RFC3339 returns time", "2024-01-15T10:30:00Z", false},
		{"valid RFC3339 with timezone", "2024-01-15T10:30:00+08:00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimeParam(tt.input)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}
