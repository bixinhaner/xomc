package notification

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

func newHistoryTestRouter(t *testing.T) (*gin.Engine, *HistoryService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := NewHistoryService(newMemHistoryRepo(), nil)
	h := NewHistoryHandler(svc, nil)

	r := gin.New()
	rg := r.Group("/api/v1/notifications")
	h.RegisterRoutes(rg)
	return r, svc
}

func TestHistoryHandler_GetByID(t *testing.T) {
	r, svc := newHistoryTestRouter(t)
	entry := &NotificationHistory{
		Channel:    TemplateChannelEmail,
		Recipients: []string{"a@example.com"},
		Subject:    "s",
		Body:       "b",
		Status:     HistoryStatusSent,
	}
	require.NoError(t, svc.Insert(context.Background(), entry))

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history/"+entry.ID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var got NotificationHistory
	response.DecodeData(t, w.Body, &got)
	assert.Equal(t, entry.ID, got.ID)
}

func TestHistoryHandler_GetByID_NotFound(t *testing.T) {
	r, _ := newHistoryTestRouter(t)
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history/"+uuid.NewString(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHistoryHandler_GetByID_BadUUID(t *testing.T) {
	r, _ := newHistoryTestRouter(t)
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHistoryHandler_List_FilterByStatus(t *testing.T) {
	r, svc := newHistoryTestRouter(t)
	entries := []NotificationHistory{
		{Channel: TemplateChannelEmail, Recipients: []string{"a"}, Status: HistoryStatusSent},
		{Channel: TemplateChannelEmail, Recipients: []string{"b"}, Status: HistoryStatusFailed},
		{Channel: TemplateChannelSMS, Recipients: []string{"c"}, Status: HistoryStatusFailed},
	}
	for i := range entries {
		require.NoError(t, svc.Insert(context.Background(), &entries[i]))
	}

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history?status=failed", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[NotificationHistory]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(2), resp.Total)
	for _, item := range resp.Items {
		assert.Equal(t, HistoryStatusFailed, item.Status)
	}
}

func TestHistoryHandler_List_BadTemplateID(t *testing.T) {
	r, _ := newHistoryTestRouter(t)
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history?template_id=not-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHistoryHandler_List_BadAlarmID(t *testing.T) {
	r, _ := newHistoryTestRouter(t)
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history?alarm_id=not-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHistoryHandler_List_FilterByTemplateAndAlarm(t *testing.T) {
	r, svc := newHistoryTestRouter(t)
	tplID := uuid.New()
	alarmID := uuid.New()

	entries := []NotificationHistory{
		{Channel: TemplateChannelEmail, Recipients: []string{"a"}, Status: HistoryStatusSent, TemplateID: &tplID, AlarmID: &alarmID},
		{Channel: TemplateChannelEmail, Recipients: []string{"b"}, Status: HistoryStatusFailed},
	}
	for i := range entries {
		require.NoError(t, svc.Insert(context.Background(), &entries[i]))
	}

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history?template_id="+tplID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListResponse[NotificationHistory]
	response.DecodeData(t, w.Body, &resp)
	assert.Equal(t, int64(1), resp.Total)

	w2 := doJSON(t, r, http.MethodGet, "/api/v1/notifications/history?alarm_id="+alarmID.String(), nil)
	assert.Equal(t, http.StatusOK, w2.Code)
	var resp2 model.ListResponse[NotificationHistory]
	response.DecodeData(t, w2.Body, &resp2)
	assert.Equal(t, int64(1), resp2.Total)
}
