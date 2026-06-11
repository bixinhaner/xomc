package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/task"
)

// mustJSON marshals v or fails the test.
func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// fakeEnqueuer is an in-memory stand-in for task.TaskService suitable for
// handler tests: it captures CreateTask calls and reports a configurable
// queue depth. Only what task.Enqueuer requires is implemented.
type fakeEnqueuer struct {
	createFn func(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error)
	lenFn    func(ctx context.Context, deviceSN string) (int64, error)
}

func (f *fakeEnqueuer) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return task.NewTask(req), nil
}

func (f *fakeEnqueuer) GetQueueLength(ctx context.Context, deviceSN string) (int64, error) {
	if f.lenFn != nil {
		return f.lenFn(ctx, deviceSN)
	}
	return 0, nil
}

// fakeDeviceChecker is an in-memory stand-in for DeviceExistenceChecker.
// existsFn drives the existence/lookup-error behavior per test.
type fakeDeviceChecker struct {
	existsFn func(ctx context.Context, sn string) (bool, error)
}

func (f *fakeDeviceChecker) ExistsBySerialNumber(ctx context.Context, sn string) (bool, error) {
	if f.existsFn != nil {
		return f.existsFn(ctx, sn)
	}
	return true, nil
}

// alwaysExistsChecker returns true for any SN — used so the existing happy-path
// tests pass the new existence pre-check without asserting on it.
func alwaysExistsChecker() DeviceExistenceChecker {
	return &fakeDeviceChecker{existsFn: func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}}
}

func setupRouter(enq task.Enqueuer) *gin.Engine {
	return setupRouterWithChecker(enq, alwaysExistsChecker())
}

func setupRouterWithChecker(enq task.Enqueuer, chk DeviceExistenceChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 测试场景模拟 AutoSync 未启用，gpvBatcher=nil → PullConfig 走兜底单 task 路径。
	handler := NewSyncHandler(enq, nil, chk, zap.NewNop())
	handler.RegisterRoutes(&r.RouterGroup)
	return r
}

// --- push config tests ---

func TestSyncHandler_PushConfig_Success(t *testing.T) {
	var capturedReq *task.CreateTaskRequest

	enq := &fakeEnqueuer{
		createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
			capturedReq = req
			return task.NewTask(req), nil
		},
	}

	router := setupRouter(enq)

	body := PushConfigRequest{
		Parameters: []ParameterValue{
			{Name: "Device.ManagementServer.PeriodicInformInterval", Value: "300", Type: "int"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/config/sync/push/DEV001", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedReq)
	assert.Equal(t, "DEV001", capturedReq.DeviceSN)
	assert.Equal(t, "SetParameterValues", capturedReq.Method)

	var respBody map[string]interface{}
	_, msg := response.DecodeData(t, w.Body, &respBody)
	assert.Equal(t, "configuration push queued", msg)
	assert.Equal(t, "DEV001", respBody["device_id"])
}

func TestSyncHandler_PushConfig_MissingDevice(t *testing.T) {
	router := setupRouter(&fakeEnqueuer{})

	body := PushConfigRequest{
		Parameters: []ParameterValue{
			{Name: "Device.Param", Value: "1", Type: "int"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/config/sync/push/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Gin returns 404 when no route matches (trailing slash without deviceId param)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSyncHandler_PushConfig_InvalidBody(t *testing.T) {
	router := setupRouter(&fakeEnqueuer{})

	req := httptest.NewRequest(http.MethodPost, "/config/sync/push/DEV001", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- pull config tests ---

func TestSyncHandler_PullConfig_Success(t *testing.T) {
	var capturedReq *task.CreateTaskRequest

	enq := &fakeEnqueuer{
		createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
			capturedReq = req
			return task.NewTask(req), nil
		},
	}

	router := setupRouter(enq)

	body := PullConfigRequest{
		ParameterNames: []string{
			"Device.ManagementServer.PeriodicInformInterval",
			"Device.DeviceInfo.ModelName",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/config/sync/pull/DEV001", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedReq)
	assert.Equal(t, "DEV001", capturedReq.DeviceSN)
	assert.Equal(t, "GetParameterValues", capturedReq.Method)

	var respBody map[string]interface{}
	_, msg := response.DecodeData(t, w.Body, &respBody)
	assert.Equal(t, "configuration pull queued", msg)
	assert.Equal(t, "DEV001", respBody["device_id"])
}

func TestSyncHandler_PullConfig_InvalidBody(t *testing.T) {
	router := setupRouter(&fakeEnqueuer{})

	req := httptest.NewRequest(http.MethodPost, "/config/sync/pull/DEV001", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- device existence pre-check tests (issue #126 item 3) ---

// TestSyncHandler_PushPull_DeviceNotFound verifies that a legal body targeting a
// non-existent device returns 404 and never reaches CreateTask (no orphan task).
func TestSyncHandler_PushPull_DeviceNotFound(t *testing.T) {
	cases := []struct {
		name string
		path string
		body []byte
	}{
		{
			name: "push",
			path: "/config/sync/push/NO-SUCH-DEV",
			body: mustJSON(t, PushConfigRequest{
				Parameters: []ParameterValue{{Name: "Device.Param", Value: "1", Type: "int"}},
			}),
		},
		{
			name: "pull",
			path: "/config/sync/pull/NO-SUCH-DEV",
			body: mustJSON(t, PullConfigRequest{
				ParameterNames: []string{"Device.DeviceInfo.ModelName"},
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			createCalled := false
			enq := &fakeEnqueuer{
				createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
					createCalled = true
					return task.NewTask(req), nil
				},
			}
			chk := &fakeDeviceChecker{existsFn: func(_ context.Context, sn string) (bool, error) {
				assert.Equal(t, "NO-SUCH-DEV", sn)
				return false, nil
			}}
			router := setupRouterWithChecker(enq, chk)

			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.False(t, createCalled, "CreateTask must not be called when device is absent")
		})
	}
}

// TestSyncHandler_PushPull_DeviceLookupError verifies that a lookup failure maps
// to 500 (distinguished from the 404 not-found case) and skips enqueue.
func TestSyncHandler_PushPull_DeviceLookupError(t *testing.T) {
	cases := []struct {
		name string
		path string
		body []byte
	}{
		{
			name: "push",
			path: "/config/sync/push/DEV001",
			body: mustJSON(t, PushConfigRequest{
				Parameters: []ParameterValue{{Name: "Device.Param", Value: "1", Type: "int"}},
			}),
		},
		{
			name: "pull",
			path: "/config/sync/pull/DEV001",
			body: mustJSON(t, PullConfigRequest{
				ParameterNames: []string{"Device.DeviceInfo.ModelName"},
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			createCalled := false
			enq := &fakeEnqueuer{
				createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
					createCalled = true
					return task.NewTask(req), nil
				},
			}
			chk := &fakeDeviceChecker{existsFn: func(_ context.Context, _ string) (bool, error) {
				return false, errors.New("db down")
			}}
			router := setupRouterWithChecker(enq, chk)

			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.False(t, createCalled, "CreateTask must not be called on lookup error")
		})
	}
}

// TestSyncHandler_PushConfig_NilCheckerSkipsPrecheck verifies backward
// compatibility: when no checker is injected, push proceeds and enqueues
// (existence pre-check is a no-op).
func TestSyncHandler_PushConfig_NilCheckerSkipsPrecheck(t *testing.T) {
	createCalled := false
	enq := &fakeEnqueuer{
		createFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
			createCalled = true
			return task.NewTask(req), nil
		},
	}
	router := setupRouterWithChecker(enq, nil)

	body := mustJSON(t, PushConfigRequest{
		Parameters: []ParameterValue{{Name: "Device.Param", Value: "1", Type: "int"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/config/sync/push/DEV001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, createCalled, "CreateTask should run when no checker is injected")
}

// --- get sync status tests ---

func TestSyncHandler_GetSyncStatus_Success(t *testing.T) {
	enq := &fakeEnqueuer{
		lenFn: func(_ context.Context, deviceSN string) (int64, error) {
			assert.Equal(t, "DEV001", deviceSN)
			return 5, nil
		},
	}

	router := setupRouter(enq)

	req := httptest.NewRequest(http.MethodGet, "/config/sync/status/DEV001", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SyncStatusResponse
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, "DEV001", resp.DeviceID)
	assert.Equal(t, int64(5), resp.PendingCount)
}
