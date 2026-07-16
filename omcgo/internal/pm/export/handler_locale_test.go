package export

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestHandler_Create_CapturesRequestLocaleInParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskID := uuid.New()
	repo := &stubRepo{
		createID: taskID,
		getTask:  &Task{ID: taskID, SourceType: SourceAdhoc, Status: StatusPending},
	}
	h := NewHandler(NewService(repo, &stubEnqueuer{insertID: uuid.New()}), nil, zap.NewNop())
	router := gin.New()
	h.RegisterRoutes(router.Group(""))

	body := []byte(`{"source_type":"adhoc","params":{"task_id":"11111111-1111-1111-1111-111111111111"}}`)
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(appcontext.WithLocale(context.Background(), appcontext.LocaleEN))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, repo.created)
	assert.JSONEq(t,
		`{"task_id":"11111111-1111-1111-1111-111111111111","locale":"en-US"}`,
		string(repo.created.Params),
	)
}

func TestHandler_Create_DeviceViewDefaultTaskName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskID := uuid.New()
	repo := &stubRepo{
		createID: taskID,
		getTask:  &Task{ID: taskID, SourceType: SourceDeviceView, Status: StatusPending},
	}
	h := NewHandler(NewService(repo, &stubEnqueuer{insertID: uuid.New()}), nil, zap.NewNop())
	router := gin.New()
	h.RegisterRoutes(router.Group(""))

	body := []byte(`{"source_type":"device_view","params":{"granularity":"hourly"}}`)
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, repo.created)
	assert.Equal(t, SourceDeviceView, repo.created.SourceType)
	assert.Contains(t, repo.created.TaskName, "KPI导出_设备性能查看_")
	assert.NotContains(t, repo.created.TaskName, "仪表盘")
}

func TestHandler_Create_DeviceViewDefaultTaskNameUsesEnglishLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskID := uuid.New()
	repo := &stubRepo{
		createID: taskID,
		getTask:  &Task{ID: taskID, SourceType: SourceDeviceView, Status: StatusPending},
	}
	h := NewHandler(NewService(repo, &stubEnqueuer{insertID: uuid.New()}), nil, zap.NewNop())
	router := gin.New()
	h.RegisterRoutes(router.Group(""))

	body := []byte(`{"source_type":"device_view","params":{"granularity":"hourly"}}`)
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(appcontext.WithLocale(context.Background(), appcontext.LocaleEN))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, repo.created)
	assert.Contains(t, repo.created.TaskName, "KPI_Export_Device_Performance_View_")
	assert.NotContains(t, repo.created.TaskName, "仪表盘")
	assert.NotContains(t, repo.created.TaskName, "设备性能查看")
}

func TestHandler_Create_RejectsTooManyExportDeviceSNs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubRepo{createID: uuid.New()}
	h := NewHandler(NewService(repo, &stubEnqueuer{insertID: uuid.New()}), nil, zap.NewNop())
	router := gin.New()
	h.RegisterRoutes(router.Group(""))

	body := mustJSON(t, map[string]any{
		"source_type": "kpi_query",
		"params": map[string]any{
			"granularity":  "hourly",
			"dimension":    "device",
			"device_sns":   makeStrings("SN", 51),
			"metric_paths": []string{"K1"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "device_sns exceeds maximum of 50")
	assert.Nil(t, repo.created)
}

func TestHandler_Create_RejectsTooManyExportMetricPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubRepo{createID: uuid.New()}
	h := NewHandler(NewService(repo, &stubEnqueuer{insertID: uuid.New()}), nil, zap.NewNop())
	router := gin.New()
	h.RegisterRoutes(router.Group(""))

	body := mustJSON(t, map[string]any{
		"source_type": "dashboard",
		"params": map[string]any{
			"granularity":  "hourly",
			"dimension":    "device",
			"device_sns":   []string{"SN-1"},
			"metric_paths": makeStrings("K", 51),
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "metric_paths exceeds maximum of 50")
	assert.Nil(t, repo.created)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	out, err := json.Marshal(v)
	require.NoError(t, err)
	return out
}

func makeStrings(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = prefix + "-" + strconv.Itoa(i+1)
	}
	return out
}
