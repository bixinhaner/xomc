package alarm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	assert.NoError(t, c.Write(&m))
	return m.Counter.GetValue()
}

// 成功路径：dispatcher 收到 2xx 响应，metric.success 自增 1。
func TestHTTPWebhookDispatcher_Success(t *testing.T) {
	var (
		gotMethod      string
		gotContentType string
		gotBody        []byte
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)

	err := d.Dispatch(context.Background(), ts.URL, "", []byte(`{"alarm_identifier":"X"}`))
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, `{"alarm_identifier":"X"}`, string(gotBody))

	assert.Equal(t, 1.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("success")))
	assert.Equal(t, 0.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")))
}

// 失败路径：服务持续返回 5xx，dispatcher 重试耗尽后返回 ErrDeadLetter。
func TestHTTPWebhookDispatcher_Non2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)
	// 缩短重试上限以加速测试
	d.client.Timeout = 1 * time.Second

	err := d.Dispatch(context.Background(), ts.URL, "", []byte(`{}`))
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeadLetter)
	// success 计数为 0，dead_letter 至少 1
	assert.Equal(t, 0.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("success")))
	assert.GreaterOrEqual(t, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")), 1.0)
}

// 网络层错误：URL 不可达时多次重试后返 ErrDeadLetter。
func TestHTTPWebhookDispatcher_TransportError(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)
	d.client.Timeout = 200 * time.Millisecond

	// 127.0.0.1:1（保留端口）几乎不会有服务监听，触发 connect 拒绝
	err := d.Dispatch(context.Background(), "http://127.0.0.1:1/never", "", []byte(`{}`))
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeadLetter)
	assert.GreaterOrEqual(t, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")), 1.0)
}

// 构造非法 URL 触发 NewRequest 错误，立即 dead-letter，不重试。
func TestHTTPWebhookDispatcher_BadURL(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)

	err := d.Dispatch(context.Background(), "://malformed", "", []byte(`{}`))
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeadLetter)
	assert.Equal(t, 1.0, counterValue(t, metrics.DispatchTotal.WithLabelValues("dead_letter")))
}

// Context 取消应被传播到 HTTP 请求，立即 dead-letter。
func TestHTTPWebhookDispatcher_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	reg := prometheus.NewRegistry()
	metrics := NewWebhookMetrics(reg)
	d := NewHTTPWebhookDispatcher(zap.NewNop(), metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := d.Dispatch(ctx, ts.URL, "", []byte(`{}`))
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeadLetter)
}

// noopWebhookDispatcher：无实参注入时的兜底。
func TestNoopWebhookDispatcher_NeverErrors(t *testing.T) {
	d := noopWebhookDispatcher{}
	assert.NoError(t, d.Dispatch(context.Background(), "https://nowhere", "", []byte(`{}`)))
}
