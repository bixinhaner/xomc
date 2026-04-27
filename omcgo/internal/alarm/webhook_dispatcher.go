package alarm

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// WebhookDispatcher 告警 webhook 派发接口。
// W1.5 冒烟版本仅支持单次 POST + JSON。retry / dead-letter / HMAC / 自定义 header / 模板渲染
// 留给 Wave 2 Block A.3 T-0011 全量实现。
type WebhookDispatcher interface {
	// Dispatch 把 payload 以 application/json POST 到 url。
	// 失败应返回 error 由调用方决定是否上报，但调用方不应阻塞业务流程。
	Dispatch(ctx context.Context, url string, payload []byte) error
}

// WebhookMetrics 维度：result = success | failure | skipped。
type WebhookMetrics struct {
	DispatchTotal *prometheus.CounterVec
}

// NewWebhookMetrics 创建并注册 webhook 指标。
func NewWebhookMetrics(reg prometheus.Registerer) *WebhookMetrics {
	m := &WebhookMetrics{
		DispatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alarm_webhook_dispatches_total",
			Help: "Total alarm webhook dispatches by result (success/failure/skipped)",
		}, []string{"result"}),
	}
	if reg != nil {
		reg.MustRegister(m.DispatchTotal)
	}
	return m
}

// HTTPWebhookDispatcher 基于标准 http.Client 的 webhook 派发实现。
type HTTPWebhookDispatcher struct {
	client  *http.Client
	logger  *zap.Logger
	metrics *WebhookMetrics
}

// NewHTTPWebhookDispatcher 默认 5 秒超时，无重试。
func NewHTTPWebhookDispatcher(logger *zap.Logger, metrics *WebhookMetrics) *HTTPWebhookDispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HTTPWebhookDispatcher{
		client:  &http.Client{Timeout: 5 * time.Second},
		logger:  logger,
		metrics: metrics,
	}
}

// Dispatch 实现 WebhookDispatcher 接口。
func (d *HTTPWebhookDispatcher) Dispatch(ctx context.Context, url string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		d.recordFailure()
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "omcgo-alarm-webhook/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		d.logger.Warn("webhook dispatch failed",
			zap.String("webhook_url", url),
			zap.Error(err))
		d.recordFailure()
		return fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		d.logger.Warn("webhook returned non-2xx",
			zap.String("webhook_url", url),
			zap.Int("status", resp.StatusCode))
		d.recordFailure()
		return fmt.Errorf("webhook non-2xx: %d", resp.StatusCode)
	}

	d.recordSuccess()
	return nil
}

func (d *HTTPWebhookDispatcher) recordSuccess() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("success").Inc()
	}
}

func (d *HTTPWebhookDispatcher) recordFailure() {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues("failure").Inc()
	}
}

// noopWebhookDispatcher 默认占位实现：无 dispatcher 注入时使用，调用即跳过。
// FilterEngine 在 dispatcher 未注入时回退到这个实现，避免 nil-check 散落各处。
type noopWebhookDispatcher struct{}

func (noopWebhookDispatcher) Dispatch(_ context.Context, _ string, _ []byte) error {
	return nil
}
