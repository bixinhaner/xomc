package notification

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailSettingsHandler_SendTest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	transport := &fakeTransport{}
	mailer, history, _ := newTestMailer(t, transport)
	handler := NewEmailSettingsHandler(mailer)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/admin"))

	req := httptest.NewRequest(http.MethodPost, "/admin/notification/email/test",
		strings.NewReader(`{"recipient":"operator@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	require.Equal(t, 1, transport.calls)
	assert.Equal(t, []string{"operator@example.test"}, transport.lastTo)
	items := historyAll(t, history)
	require.Len(t, items, 1)
	assert.Equal(t, HistoryStatusSent, items[0].Status)
}

func TestEmailSettingsHandler_SendTestRejectsMultipleRecipients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	transport := &fakeTransport{}
	mailer, _, _ := newTestMailer(t, transport)
	handler := NewEmailSettingsHandler(mailer)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/admin"))

	req := httptest.NewRequest(http.MethodPost, "/admin/notification/email/test",
		strings.NewReader(`{"recipient":"a@example.test,b@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Zero(t, transport.calls)
}

func TestEmailSettingsHandler_SendTestRequiresHistoryRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	transport := &fakeTransport{}
	mailer, history, _ := newTestMailer(t, transport)
	history.insertErr = assert.AnError
	handler := NewEmailSettingsHandler(mailer)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/admin"))

	req := httptest.NewRequest(http.MethodPost, "/admin/notification/email/test",
		strings.NewReader(`{"recipient":"operator@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadGateway, res.Code)
	assert.Zero(t, transport.calls)
}
