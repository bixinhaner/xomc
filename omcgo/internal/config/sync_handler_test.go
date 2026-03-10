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

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
)

// --- mock command queue ---

type mockCommandQueue struct {
	pushFn  func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error
	popFn   func(ctx context.Context, deviceSN string) (*cmdqueue.Command, error)
	peekFn  func(ctx context.Context, deviceSN string) (*cmdqueue.Command, error)
	lenFn   func(ctx context.Context, deviceSN string) (int64, error)
	clearFn func(ctx context.Context, deviceSN string) error
}

func (m *mockCommandQueue) Push(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
	if m.pushFn != nil {
		return m.pushFn(ctx, deviceSN, cmd)
	}
	return nil
}

func (m *mockCommandQueue) Pop(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	if m.popFn != nil {
		return m.popFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockCommandQueue) Peek(ctx context.Context, deviceSN string) (*cmdqueue.Command, error) {
	if m.peekFn != nil {
		return m.peekFn(ctx, deviceSN)
	}
	return nil, nil
}

func (m *mockCommandQueue) Len(ctx context.Context, deviceSN string) (int64, error) {
	if m.lenFn != nil {
		return m.lenFn(ctx, deviceSN)
	}
	return 0, nil
}

func (m *mockCommandQueue) Clear(ctx context.Context, deviceSN string) error {
	if m.clearFn != nil {
		return m.clearFn(ctx, deviceSN)
	}
	return nil
}

// helper: creates a gin engine with the sync handler routes registered.
func setupRouter(queue cmdqueue.CommandQueue) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewSyncHandler(queue, zap.NewNop())
	handler.RegisterRoutes(&r.RouterGroup)
	return r
}

// --- push config tests ---

func TestSyncHandler_PushConfig_Success(t *testing.T) {
	var capturedDeviceSN string
	var capturedCmd *cmdqueue.Command

	queue := &mockCommandQueue{
		pushFn: func(_ context.Context, deviceSN string, cmd *cmdqueue.Command) error {
			capturedDeviceSN = deviceSN
			capturedCmd = cmd
			return nil
		},
	}

	router := setupRouter(queue)

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
	assert.Equal(t, "DEV001", capturedDeviceSN)
	require.NotNil(t, capturedCmd)
	assert.Equal(t, "SetParameterValues", capturedCmd.Method)

	var respBody map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, "configuration push queued", respBody["message"])
	assert.Equal(t, "DEV001", respBody["device_id"])
}

func TestSyncHandler_PushConfig_MissingDevice(t *testing.T) {
	queue := &mockCommandQueue{}
	router := setupRouter(queue)

	body := PushConfigRequest{
		Parameters: []ParameterValue{
			{Name: "Device.Param", Value: "1", Type: "int"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	// POST to /config/sync/push/ without device ID -- the router won't match this route
	req := httptest.NewRequest(http.MethodPost, "/config/sync/push/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Gin returns 404 when no route matches (trailing slash without deviceId param)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSyncHandler_PushConfig_InvalidBody(t *testing.T) {
	queue := &mockCommandQueue{}
	router := setupRouter(queue)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/config/sync/push/DEV001", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- pull config tests ---

func TestSyncHandler_PullConfig_Success(t *testing.T) {
	var capturedDeviceSN string
	var capturedCmd *cmdqueue.Command

	queue := &mockCommandQueue{
		pushFn: func(_ context.Context, deviceSN string, cmd *cmdqueue.Command) error {
			capturedDeviceSN = deviceSN
			capturedCmd = cmd
			return nil
		},
	}

	router := setupRouter(queue)

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
	assert.Equal(t, "DEV001", capturedDeviceSN)
	require.NotNil(t, capturedCmd)
	assert.Equal(t, "GetParameterValues", capturedCmd.Method)

	var respBody map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, "configuration pull queued", respBody["message"])
	assert.Equal(t, "DEV001", respBody["device_id"])
}

func TestSyncHandler_PullConfig_InvalidBody(t *testing.T) {
	queue := &mockCommandQueue{}
	router := setupRouter(queue)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/config/sync/pull/DEV001", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- get sync status tests ---

func TestSyncHandler_GetSyncStatus_Success(t *testing.T) {
	queue := &mockCommandQueue{
		lenFn: func(_ context.Context, deviceSN string) (int64, error) {
			assert.Equal(t, "DEV001", deviceSN)
			return 5, nil
		},
	}

	router := setupRouter(queue)

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
