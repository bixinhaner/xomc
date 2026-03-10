package syslog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockSyslogRepo struct {
	systemLogs    *model.ListResponse[SystemLog]
	systemLogsErr error

	neMsgLogs    *model.ListResponse[NEMessageLog]
	neMsgLogsErr error

	// Capture the filters for assertions.
	lastSystemLogFilter    SystemLogFilter
	lastNEMessageLogFilter NEMessageLogFilter
}

func (m *mockSyslogRepo) ListSystemLogs(_ context.Context, f SystemLogFilter) (*model.ListResponse[SystemLog], error) {
	m.lastSystemLogFilter = f
	return m.systemLogs, m.systemLogsErr
}

func (m *mockSyslogRepo) ListNEMessageLogs(_ context.Context, f NEMessageLogFilter) (*model.ListResponse[NEMessageLog], error) {
	m.lastNEMessageLogFilter = f
	return m.neMsgLogs, m.neMsgLogsErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupRouter(repo SyslogRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repo, zap.NewNop())
	h.RegisterRoutes(r.Group(""))
	return r
}

func sampleSystemLogs() *model.ListResponse[SystemLog] {
	return &model.ListResponse[SystemLog]{
		Items: []SystemLog{
			{
				ID:        uuid.New(),
				Level:     "info",
				Source:    "acs",
				Message:   "device connected",
				CreatedAt: time.Now(),
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
}

func sampleNEMessageLogs() *model.ListResponse[NEMessageLog] {
	return &model.ListResponse[NEMessageLog]{
		Items: []NEMessageLog{
			{
				ID:          uuid.New(),
				DeviceSN:    "DEV001",
				MessageType: "Inform",
				Direction:   "inbound",
				CreatedAt:   time.Now(),
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
}

// ---------------------------------------------------------------------------
// Tests — ListSystemLogs
// ---------------------------------------------------------------------------

func TestHandler_ListSystemLogs_Default(t *testing.T) {
	repo := &mockSyslogRepo{systemLogs: sampleSystemLogs()}
	router := setupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/logs/system", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[SystemLog]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)

	// Default filter should have no level/source set.
	assert.Nil(t, repo.lastSystemLogFilter.Level)
	assert.Nil(t, repo.lastSystemLogFilter.Source)
}

func TestHandler_ListSystemLogs_WithFilter(t *testing.T) {
	repo := &mockSyslogRepo{systemLogs: sampleSystemLogs()}
	router := setupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/logs/system?level=error&source=acs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	require.NotNil(t, repo.lastSystemLogFilter.Level)
	assert.Equal(t, "error", *repo.lastSystemLogFilter.Level)
	require.NotNil(t, repo.lastSystemLogFilter.Source)
	assert.Equal(t, "acs", *repo.lastSystemLogFilter.Source)
}

// ---------------------------------------------------------------------------
// Tests — ListNEMessageLogs
// ---------------------------------------------------------------------------

func TestHandler_ListNEMessageLogs_Default(t *testing.T) {
	repo := &mockSyslogRepo{neMsgLogs: sampleNEMessageLogs()}
	router := setupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/logs/ne-messages", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[NEMessageLog]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Items, 1)

	// Default filter should have no device_sn set.
	assert.Nil(t, repo.lastNEMessageLogFilter.DeviceSN)
}

func TestHandler_ListNEMessageLogs_WithFilter(t *testing.T) {
	repo := &mockSyslogRepo{neMsgLogs: sampleNEMessageLogs()}
	router := setupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/logs/ne-messages?device_sn=DEV001", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	require.NotNil(t, repo.lastNEMessageLogFilter.DeviceSN)
	assert.Equal(t, "DEV001", *repo.lastNEMessageLogFilter.DeviceSN)
}

func TestHandler_ListNEMessageLogs_InvalidDeviceID(t *testing.T) {
	repo := &mockSyslogRepo{neMsgLogs: sampleNEMessageLogs()}
	router := setupRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/logs/ne-messages?device_id=invalid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
