package notification

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestWebhook 装配一个 AlertWebhookHandler（Mailer 内存 repo + 给定 transport）。
func newTestWebhook(t *testing.T, opts AlertWebhookOptions, tr EmailTransport) (*AlertWebhookHandler, *memHistoryRepo) {
	t.Helper()
	histRepo := newMemHistoryRepo()
	m := NewMailer(
		NewTemplateService(newMemTemplateRepo(), nil),
		NewHistoryService(histRepo, nil),
		tr,
		nil,
	)
	return NewAlertWebhookHandler(m, opts, nil), histRepo
}

// doWebhook 发一个 POST /api/v1/alerts/webhook 请求。
func doWebhook(h *AlertWebhookHandler, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/webhook", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const firingPayload = `{"status":"firing","alerts":[
  {"status":"firing","labels":{"alertname":"OMCAppDown","severity":"critical","service":"omcgo-app"},
   "annotations":{"summary":"app 不可用","description":"2 分钟无法抓取"},"startsAt":"2026-05-18T10:00:00Z"}
]}`

func TestAlertWebhook_Firing_SendsEmail(t *testing.T) {
	tr := &fakeTransport{}
	h, histRepo := newTestWebhook(t, AlertWebhookOptions{Recipients: []string{"ops@x.com"}}, tr)

	w := doWebhook(h, firingPayload, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, tr.calls)
	require.Equal(t, []string{"ops@x.com"}, tr.lastTo)
	require.Contains(t, tr.lastSubject, "触发")
	require.Contains(t, tr.lastBody, "OMCAppDown")
	require.Contains(t, tr.lastBody, "severity=critical")

	hist := historyAll(t, histRepo)
	require.Len(t, hist, 1)
	require.Equal(t, HistoryStatusSent, hist[0].Status)
}

func TestAlertWebhook_EmptyAlerts(t *testing.T) {
	tr := &fakeTransport{}
	h, _ := newTestWebhook(t, AlertWebhookOptions{Recipients: []string{"ops@x.com"}}, tr)

	w := doWebhook(h, `{"status":"firing","alerts":[]}`, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, tr.calls)
}

func TestAlertWebhook_NoRecipients(t *testing.T) {
	tr := &fakeTransport{}
	h, _ := newTestWebhook(t, AlertWebhookOptions{Recipients: nil}, tr)

	w := doWebhook(h, firingPayload, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, tr.calls)
	require.Contains(t, w.Body.String(), "no_recipients")
}

func TestAlertWebhook_InvalidPayload(t *testing.T) {
	tr := &fakeTransport{}
	h, _ := newTestWebhook(t, AlertWebhookOptions{Recipients: []string{"ops@x.com"}}, tr)

	w := doWebhook(h, `not json`, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, 0, tr.calls)
}

func TestAlertWebhook_TokenMismatch(t *testing.T) {
	tr := &fakeTransport{}
	h, _ := newTestWebhook(t, AlertWebhookOptions{
		Token: "secret-token", Recipients: []string{"ops@x.com"},
	}, tr)

	w := doWebhook(h, firingPayload, map[string]string{"Authorization": "Bearer wrong"})
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, 0, tr.calls)
}

func TestAlertWebhook_TokenMatch(t *testing.T) {
	tr := &fakeTransport{}
	h, _ := newTestWebhook(t, AlertWebhookOptions{
		Token: "secret-token", Recipients: []string{"ops@x.com"},
	}, tr)

	w := doWebhook(h, firingPayload, map[string]string{"Authorization": "Bearer secret-token"})
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, tr.calls)
}

func TestAlertWebhook_DispatchFailureReturns502(t *testing.T) {
	tr := &fakeTransport{err: errors.New("smtp unreachable")}
	h, _ := newTestWebhook(t, AlertWebhookOptions{Recipients: []string{"ops@x.com"}}, tr)

	w := doWebhook(h, firingPayload, nil)
	// 5xx 让 Alertmanager 按其重试策略重投
	require.Equal(t, http.StatusBadGateway, w.Code)
}

func TestRenderAlerts_FiringAndResolved(t *testing.T) {
	t.Parallel()
	p := alertmanagerPayload{Alerts: []alertmanagerAlert{
		{Status: "firing", Labels: map[string]string{"alertname": "A"}},
		{Status: "resolved", Labels: map[string]string{"alertname": "B"}},
	}}
	subject, body := renderAlerts(p)
	require.Contains(t, subject, "1 条触发")
	require.Contains(t, subject, "1 条恢复")
	require.Contains(t, body, "[firing] A")
	require.Contains(t, body, "[resolved] B")
}
