package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func newTemplateTestRouter(t *testing.T) (*gin.Engine, *TemplateService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	h := NewTemplateHandler(svc, nil)

	r := gin.New()
	rg := r.Group("/api/v1/notifications")
	h.RegisterRoutes(rg)
	return r, svc
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTemplateHandler_Create_Success(t *testing.T) {
	r, _ := newTemplateTestRouter(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/notifications/templates", CreateTemplateRequest{
		Name:    "alarm-major",
		Channel: TemplateChannelEmail,
		Subject: "Alert",
		Body:    "Body",
	})
	assert.Equal(t, http.StatusCreated, w.Code)

	var got NotificationTemplate
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "alarm-major", got.Name)
	assert.NotEqual(t, uuid.Nil, got.ID)
}

func TestTemplateHandler_Create_BadRequest(t *testing.T) {
	r, _ := newTemplateTestRouter(t)

	// Missing required fields (channel + subject + body).
	w := doJSON(t, r, http.MethodPost, "/api/v1/notifications/templates", map[string]any{
		"name": "no-channel",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTemplateHandler_Create_InvalidChannel(t *testing.T) {
	r, _ := newTemplateTestRouter(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/notifications/templates", CreateTemplateRequest{
		Name:    "bad",
		Channel: "telegram",
		Subject: "s",
		Body:    "b",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTemplateHandler_GetByID(t *testing.T) {
	r, svc := newTemplateTestRouter(t)
	tpl, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "g",
		Channel: TemplateChannelSMS,
		Subject: "s",
		Body:    "b",
	})
	require.NoError(t, err)

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/templates/"+tpl.ID.String(), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var got NotificationTemplate
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, tpl.ID, got.ID)
}

func TestTemplateHandler_GetByID_NotFound(t *testing.T) {
	r, _ := newTemplateTestRouter(t)
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/templates/"+uuid.NewString(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTemplateHandler_GetByID_BadUUID(t *testing.T) {
	r, _ := newTemplateTestRouter(t)
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/templates/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTemplateHandler_Update(t *testing.T) {
	r, svc := newTemplateTestRouter(t)
	tpl, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "upd",
		Channel: TemplateChannelEmail,
		Subject: "old",
		Body:    "old",
	})
	require.NoError(t, err)

	newSubj := "new subject"
	disabled := false
	w := doJSON(t, r, http.MethodPut, "/api/v1/notifications/templates/"+tpl.ID.String(), UpdateTemplateRequest{
		Subject: &newSubj,
		Enabled: &disabled,
	})
	assert.Equal(t, http.StatusOK, w.Code)

	var got NotificationTemplate
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "new subject", got.Subject)
	assert.False(t, got.Enabled)
}

func TestTemplateHandler_Delete(t *testing.T) {
	r, svc := newTemplateTestRouter(t)
	tpl, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "dl",
		Channel: TemplateChannelWebhook,
		Subject: "s",
		Body:    "b",
	})
	require.NoError(t, err)

	w := doJSON(t, r, http.MethodDelete, "/api/v1/notifications/templates/"+tpl.ID.String(), nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	w2 := doJSON(t, r, http.MethodGet, "/api/v1/notifications/templates/"+tpl.ID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestTemplateHandler_List_FilterByChannel(t *testing.T) {
	r, svc := newTemplateTestRouter(t)
	for _, ch := range []string{TemplateChannelEmail, TemplateChannelSMS, TemplateChannelEmail} {
		_, err := svc.Create(context.Background(), CreateTemplateRequest{
			Name:    "n-" + ch + "-" + uuid.NewString(),
			Channel: ch,
			Subject: "s",
			Body:    "b",
		})
		require.NoError(t, err)
	}

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/templates?channel=email", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[NotificationTemplate]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Total)
	for _, item := range resp.Items {
		assert.Equal(t, TemplateChannelEmail, item.Channel)
	}
}
