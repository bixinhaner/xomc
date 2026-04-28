package alarm

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// ErrDeadLetter 表示 webhook 派发已耗尽全部重试后仍失败，
// 调用方应将 payload 写入 dead-letter 存储，避免业务流程阻塞。
var ErrDeadLetter = errors.New("webhook dispatch exhausted retries")

// WebhookDispatcher 告警 webhook 派发接口。
// W2 T-0011 升级到生产级：retry / dead-letter / HMAC 已就位；
// 自定义 header / 模板渲染 / 密钥轮换留后续任务。
type WebhookDispatcher interface {
	// Dispatch 把 payload 以 application/json POST 到 url。
	// 当 secret 非空时使用 HMAC-SHA256 签名 payload，并通过 X-OMC-Signature 头携带（GitHub webhook 风格）。
	// 失败语义：
	//   - 4xx（400/401/403/404）：业务错，立刻返回 ErrDeadLetter，不重试。
	//   - 5xx 与传输错误：按指数退避重试至多 3 次（共 4 次尝试）；最终仍失败返 ErrDeadLetter。
	//   - URL 不合法 / payload 构造失败：返回 ErrDeadLetter（不可恢复，无需重试）。
	Dispatch(ctx context.Context, url, secret string, payload []byte) error
}

// WebhookMetrics 维度：result = success | retry | dead_letter | skipped。
type WebhookMetrics struct {
	DispatchTotal *prometheus.CounterVec
}

// NewWebhookMetrics 创建并注册 webhook 指标。
func NewWebhookMetrics(reg prometheus.Registerer) *WebhookMetrics {
	m := &WebhookMetrics{
		DispatchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alarm_webhook_dispatches_total",
			Help: "Total alarm webhook dispatches by result (success/retry/dead_letter/skipped)",
		}, []string{"result"}),
	}
	if reg != nil {
		reg.MustRegister(m.DispatchTotal)
	}
	return m
}

// 重试策略常量（章程 W2.A.2 指定）：base=500ms, factor=2, max attempts=4（1 次首发 + 3 次重试）, cap=5s。
const (
	webhookRetryBase     = 500 * time.Millisecond
	webhookRetryFactor   = 2
	webhookRetryMaxRetry = 3
	webhookRetryCap      = 5 * time.Second
	webhookSignatureHdr  = "X-OMC-Signature"
)

// HTTPWebhookDispatcher 基于标准 http.Client 的 webhook 派发实现。
type HTTPWebhookDispatcher struct {
	client  *http.Client
	logger  *zap.Logger
	metrics *WebhookMetrics
}

// NewHTTPWebhookDispatcher 默认 5 秒超时（每次尝试），重试与签名内置。
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
func (d *HTTPWebhookDispatcher) Dispatch(ctx context.Context, url, secret string, payload []byte) error {
	signature := ""
	if secret != "" {
		signature = computeHMACSignature(secret, payload)
	}

	var lastErr error
	for attempt := 0; attempt <= webhookRetryMaxRetry; attempt++ {
		// 每次尝试构造一份 reader——payload 短，复制成本可忽略
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			d.logger.Warn("build webhook request failed",
				zap.String("webhook_url", url),
				zap.Error(err))
			d.recordResult("dead_letter")
			return fmt.Errorf("%w: build request: %v", ErrDeadLetter, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "omcgo-alarm-webhook/1.0")
		if signature != "" {
			req.Header.Set(webhookSignatureHdr, "sha256="+signature)
		}

		resp, err := d.client.Do(req)
		if err == nil {
			// 必须读完 body 才能复用连接
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				d.recordResult("success")
				return nil
			}
			// 4xx 业务错：客户端配置/认证不对，重试无意义
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				d.logger.Warn("webhook returned 4xx, no retry",
					zap.String("webhook_url", url),
					zap.Int("status", resp.StatusCode))
				d.recordResult("dead_letter")
				return fmt.Errorf("%w: webhook 4xx status %d", ErrDeadLetter, resp.StatusCode)
			}
			// 5xx 走重试
			lastErr = fmt.Errorf("webhook %d", resp.StatusCode)
			d.logger.Warn("webhook returned 5xx, will retry",
				zap.String("webhook_url", url),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt))
		} else {
			// 传输层错误 / context 取消：context 取消立即返回，避免无意义重试
			if ctx.Err() != nil {
				d.recordResult("dead_letter")
				return fmt.Errorf("%w: context cancelled: %v", ErrDeadLetter, ctx.Err())
			}
			lastErr = err
			d.logger.Warn("webhook transport error, will retry",
				zap.String("webhook_url", url),
				zap.Error(err),
				zap.Int("attempt", attempt))
		}

		// 还有重试机会：等待退避时间
		if attempt < webhookRetryMaxRetry {
			d.recordResult("retry")
			delay := backoffDelay(attempt)
			select {
			case <-ctx.Done():
				d.recordResult("dead_letter")
				return fmt.Errorf("%w: context cancelled during backoff: %v", ErrDeadLetter, ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	d.recordResult("dead_letter")
	return fmt.Errorf("%w: last error: %v", ErrDeadLetter, lastErr)
}

func (d *HTTPWebhookDispatcher) recordResult(result string) {
	if d.metrics != nil {
		d.metrics.DispatchTotal.WithLabelValues(result).Inc()
	}
}

// backoffDelay 第 N 次重试前的等待时间：base * factor^N，最大不超过 cap。
func backoffDelay(attempt int) time.Duration {
	d := webhookRetryBase
	for i := 0; i < attempt; i++ {
		d *= time.Duration(webhookRetryFactor)
		if d > webhookRetryCap {
			return webhookRetryCap
		}
	}
	if d > webhookRetryCap {
		return webhookRetryCap
	}
	return d
}

// computeHMACSignature 用 HMAC-SHA256 给 payload 签名，返回 hex 字符串。
func computeHMACSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// noopWebhookDispatcher 默认占位实现：无 dispatcher 注入时使用，调用即跳过。
// FilterEngine 在 dispatcher 未注入时回退到这个实现，避免 nil-check 散落各处。
type noopWebhookDispatcher struct{}

func (noopWebhookDispatcher) Dispatch(_ context.Context, _ string, _ string, _ []byte) error {
	return nil
}
