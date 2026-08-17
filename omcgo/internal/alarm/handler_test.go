package alarm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupHandlerTest creates a Handler wired to a mockAlarmStore and a minimal
// AlarmEngine (no Redis, no carrier registry, no event bus) for HTTP-level
// handler testing. It returns the handler, mock store, engine, and a gin
// router with the handler routes registered.
func setupHandlerTest() (*Handler, *mockAlarmStore, *AlarmEngine, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	store := newMockAlarmStore()
	engine := newTestEngine(store)
	logger := zap.NewNop()
	handler := NewHandler(engine, store, nil, logger)

	router := gin.New()
	handler.RegisterRoutes(router.Group(""))
	return handler, store, engine, router
}

// seedActiveAlarm inserts an alarm into the mock store and returns it.
func seedActiveAlarm(store *mockAlarmStore, opts ...func(*model.Alarm)) *model.Alarm {
	now := time.Now()
	alarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-TEST-001",
		Carrier:         model.CarrierCMCC,
		Severity:        model.AlarmMajor,
		AlarmType:       "equipment",
		AlarmIdentifier: "ALM001",
		Status:          model.AlarmActive,
		RaisedAt:        now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	for _, fn := range opts {
		fn(alarm)
	}
	store.active[alarm.ID] = alarm
	return alarm
}

// ---------- ListActive ----------

func TestHandler_ListActive_OK(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	seedActiveAlarm(store)
	seedActiveAlarm(store, func(a *model.Alarm) {
		a.AlarmIdentifier = "ALM002"
		a.DeviceSN = "SN-TEST-002"
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/active", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[model.Alarm]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestHandler_ListActive_WithSeverityFilter(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	seedActiveAlarm(store, func(a *model.Alarm) { a.Severity = model.AlarmCritical })
	seedActiveAlarm(store, func(a *model.Alarm) { a.Severity = model.AlarmWarning })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/active?severity=1", nil)
	router.ServeHTTP(w, req)

	// The mock store does not filter by severity; it returns all active alarms.
	// We only verify the handler parsed the query without error (200 OK).
	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, store.lastActiveFilter.Severity) {
		assert.Equal(t, model.AlarmCritical, *store.lastActiveFilter.Severity)
	}
	assert.Empty(t, store.lastActiveFilter.Severities)
}

func TestHandler_ListActive_WithMultipleSeverityFilter(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/active?severity=1,3,1,9", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, store.lastActiveFilter.Severity)
	assert.Equal(t, []model.AlarmSeverity{model.AlarmCritical, model.AlarmMinor}, store.lastActiveFilter.Severities)
}

func TestHandler_ListActive_InvalidDeviceID(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/active?device_id=not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListActive_WithTimeFilter(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	start := time.Now().Add(-24 * time.Hour).UTC().Truncate(time.Second)
	end := time.Now().UTC().Truncate(time.Second)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/alarms/active?start_time=%s&end_time=%s", start.Format(time.RFC3339), end.Format(time.RFC3339)),
		nil,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, store.lastActiveFilter.StartTime) {
		assert.True(t, store.lastActiveFilter.StartTime.Equal(start))
	}
	if assert.NotNil(t, store.lastActiveFilter.EndTime) {
		assert.True(t, store.lastActiveFilter.EndTime.Equal(end))
	}
}

func TestHandler_ListActive_WithNeTypeFilter(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/active?ne_type=gNB", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"gNB"}, store.lastActiveFilter.Technologies)
}

// ---------- ListHistory ----------

func TestHandler_ListHistory_OK(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	now := time.Now()
	cleared := &model.Alarm{
		ID:              uuid.New(),
		DeviceSN:        "SN-TEST-001",
		AlarmIdentifier: "ALM001",
		Status:          model.AlarmCleared,
		ClearedAt:       &now,
	}
	store.history = append(store.history, cleared)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/history", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[model.Alarm]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)
}

func TestHandler_ListHistory_WithTimeFilter(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/alarms/history?start_time=%s&end_time=%s", start, end), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListHistory_WithDeviceSNFilter(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/history?device_sn=SN-HISTORY-001", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, store.lastHistoryFilter.DeviceSN) {
		assert.Equal(t, "SN-HISTORY-001", *store.lastHistoryFilter.DeviceSN)
	}
}

// ---------- Statistics ----------

func TestHandler_Statistics_OK(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	seedActiveAlarm(store, func(a *model.Alarm) { a.Severity = model.AlarmCritical })
	seedActiveAlarm(store, func(a *model.Alarm) { a.Severity = model.AlarmMajor })
	seedActiveAlarm(store, func(a *model.Alarm) { a.Severity = model.AlarmMajor })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/statistics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats AlarmStatistics
	response.DecodeData(t, w.Body, &stats)
	assert.Equal(t, int64(3), stats.TotalActive)
}

func TestHandler_Statistics_WithDeviceFilter(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	deviceID := uuid.New()
	seedActiveAlarm(store, func(a *model.Alarm) { a.DeviceID = deviceID })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/alarms/statistics?device_id=%s", deviceID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- GetByID ----------

func TestHandler_GetByID_Found(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	alarm := seedActiveAlarm(store)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/alarms/%s", alarm.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got model.Alarm
	response.DecodeData(t, w.Body, &got)
	assert.Equal(t, alarm.ID, got.ID)
	assert.Equal(t, alarm.AlarmIdentifier, got.AlarmIdentifier)
}

func TestHandler_GetByID_FallsBackToHistory(t *testing.T) {
	_, store, _, router := setupHandlerTest()
	now := time.Now()
	historyAlarm := &model.Alarm{
		ID:              uuid.New(),
		DeviceID:        uuid.New(),
		DeviceSN:        "SN-HISTORY-001",
		Carrier:         model.CarrierCMCC,
		Severity:        model.AlarmMajor,
		AlarmType:       "equipment",
		AlarmIdentifier: "ALM-HISTORY-001",
		Status:          model.AlarmCleared,
		RaisedAt:        now.Add(-time.Hour),
		ClearedAt:       &now,
		UpdatedAt:       now,
		AdditionalInfo:  map[string]string{"additional_information": "slot=1", "additional_text": "LTE0"},
	}
	store.history = append(store.history, historyAlarm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/alarms/%s", historyAlarm.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got model.Alarm
	response.DecodeData(t, w.Body, &got)
	assert.Equal(t, historyAlarm.ID, got.ID)
	assert.Equal(t, "slot=1", got.AdditionalInfo["additional_information"])
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/alarms/%s", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetByID_InvalidUUID(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/alarms/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------- Acknowledge ----------

func TestHandler_Acknowledge_OK(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	alarm := seedActiveAlarm(store)

	body := `{"acknowledged_by":"admin@test.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/acknowledge", alarm.ID.String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	stored := store.active[alarm.ID]
	assert.Equal(t, model.AlarmAcknowledged, stored.Status)
	require.NotNil(t, stored.AcknowledgedBy)
	assert.Equal(t, "admin@test.com", *stored.AcknowledgedBy)
	assert.NotNil(t, stored.AcknowledgedAt)
}

func TestHandler_Acknowledge_InvalidUUID(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	body := `{"acknowledged_by":"admin"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		"/alarms/not-a-uuid/acknowledge",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Acknowledge_MissingBody(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	alarm := seedActiveAlarm(store)

	// Empty JSON body — acknowledged_by is required.
	body := `{}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/acknowledge", alarm.ID.String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Acknowledge_NotFoundAlarm(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	body := `{"acknowledged_by":"admin"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/acknowledge", uuid.New().String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAcknowledge_NotFound_Returns404 ensures that POST /alarms/:id/acknowledge
// against a non-existent alarm ID returns HTTP 404 (not 500), so the
// pgx.ErrNoRows from the repository is mapped to commonerrors.ErrNotFound and
// then translated to 404 by HTTPStatusFromError. (T-0057 / W2.D.1.b)
func TestAcknowledge_NotFound_Returns404(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	body := `{"acknowledged_by":"admin"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/acknowledge", uuid.New().String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code,
		"missing alarm should map ErrNotFound to 404, got body=%s", w.Body.String())
}

// ---------- ClearAlarm ----------

func TestHandler_ClearAlarm_OK(t *testing.T) {
	_, store, _, router := setupHandlerTest()

	alarm := seedActiveAlarm(store)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/clear", alarm.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// After clearing, the alarm should be archived and removed from active.
	assert.Len(t, store.active, 0)
	assert.Len(t, store.history, 1)
	assert.Equal(t, model.AlarmCleared, store.history[0].Status)
}

func TestHandler_ClearAlarm_InvalidUUID(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/alarms/bad-id/clear", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ClearAlarm_NotFound(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/clear", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_BatchClearPublishesEmailEventAndPreservesMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMockAlarmStore()
	bus := &recordingAlarmEmailEventBus{}
	engine := NewAlarmEngine(store, nil, nil, bus, zap.NewNop())
	handler := NewHandler(engine, store, nil, zap.NewNop())
	router := gin.New()
	handler.RegisterRoutes(router.Group(""))

	alarm := seedActiveAlarm(store)
	body := fmt.Sprintf(`{"ids":["%s"],"clear_note":"UAT recovery notification"}`, alarm.ID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/alarms/active/batch/clear", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, store.history, 1)
	require.NotNil(t, store.history[0].ClearedBy)
	assert.Equal(t, "operator", *store.history[0].ClearedBy)
	require.NotNil(t, store.history[0].ClearNote)
	assert.Equal(t, "UAT recovery notification", *store.history[0].ClearNote)
	assert.Equal(t, []string{event.SubjectAlarmCleared, event.SubjectAlarmEmailCleared}, bus.published)
}

func TestHandler_BatchClearSkipsMissingAlarm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMockAlarmStore()
	bus := &recordingAlarmEmailEventBus{}
	engine := NewAlarmEngine(store, nil, nil, bus, zap.NewNop())
	handler := NewHandler(engine, store, nil, zap.NewNop())
	router := gin.New()
	handler.RegisterRoutes(router.Group(""))

	alarm := seedActiveAlarm(store)
	body := fmt.Sprintf(`{"ids":["%s","%s"],"clear_note":"batch clear"}`, uuid.New(), alarm.ID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/alarms/active/batch/clear", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, store.history, 1)
	assert.Equal(t, []string{event.SubjectAlarmCleared, event.SubjectAlarmEmailCleared}, bus.published)
}

// TestClear_NotFound_Returns404 ensures that POST /alarms/:id/clear against a
// non-existent alarm ID returns HTTP 404 instead of leaking pgx.ErrNoRows
// through as 500. (T-0057 / W2.D.1.b)
func TestClear_NotFound_Returns404(t *testing.T) {
	_, _, _, router := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("/alarms/%s/clear", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code,
		"missing alarm should map ErrNotFound to 404, got body=%s", w.Body.String())
}

// ---------- parseSeverity ----------

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected model.AlarmSeverity
	}{
		{"1", model.AlarmCritical},
		{"31001", model.AlarmCritical},
		{"2", model.AlarmMajor},
		{"31002", model.AlarmMajor},
		{"3", model.AlarmMinor},
		{"31003", model.AlarmMinor},
		{"4", model.AlarmWarning},
		{"31004", model.AlarmWarning},
		{"0", 0},
		{"", 0},
		{"unknown", 0},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("input=%q", tc.input), func(t *testing.T) {
			assert.Equal(t, tc.expected, parseSeverity(tc.input))
		})
	}
}

func TestParseSeverities(t *testing.T) {
	assert.Equal(
		t,
		[]model.AlarmSeverity{model.AlarmCritical, model.AlarmMinor, model.AlarmWarning},
		parseSeverities("1, 31001, 3, 31003,4,31004,3,bad"),
	)
	assert.Empty(t, parseSeverities("bad"))
}
