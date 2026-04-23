package config

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

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

func setupRouter(enq task.Enqueuer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewSyncHandler(enq, zap.NewNop())
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
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, "configuration push queued", respBody["message"])
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
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, "configuration pull queued", respBody["message"])
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
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "DEV001", resp.DeviceID)
	assert.Equal(t, int64(5), resp.PendingCount)
}
