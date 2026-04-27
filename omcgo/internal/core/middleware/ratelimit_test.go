package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// newRateLimitTestEngine 构造一个安装了 RateLimit 中间件的 Gin 引擎，
// 每个测试用独立 Registry 以避免 prometheus 重复注册导致的副作用。
func newRateLimitTestEngine(cfg RateLimitConfig) *gin.Engine {
	if cfg.Registerer == nil {
		cfg.Registerer = prometheus.NewRegistry()
	}
	r := gin.New()
	r.Use(RateLimit(cfg))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "alive") })
	return r
}

func doGetWithIP(r *gin.Engine, path, ip string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	// Gin 的 c.ClientIP() 优先使用 RemoteAddr（在受信代理场景下会先看 X-Forwarded-For）。
	// 这里直接设置 RemoteAddr，让 ClientIP 返回我们指定的值。
	req.RemoteAddr = ip + ":12345"
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 100,
		Burst:         5,
	})

	// 5 次请求处于 burst 容量内，全部应该通过。
	for i := 0; i < 5; i++ {
		w := doGetWithIP(r, "/test", "10.0.0.1")
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass", i)
	}
}

func TestRateLimit_RejectsBeyondBurst(t *testing.T) {
	// rate 极低 + burst=2，确保第 3 个请求会被拒绝。
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 0.5, // 每 2 秒补充 1 个 token
		Burst:         2,
	})

	w1 := doGetWithIP(r, "/test", "10.0.0.2")
	w2 := doGetWithIP(r, "/test", "10.0.0.2")
	w3 := doGetWithIP(r, "/test", "10.0.0.2")

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Contains(t, w3.Body.String(), "RATE_LIMITED")
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	// IP A 用尽 burst 后 IP B 应仍能通过，证明 limiter 按 IP 隔离。
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 0.5,
		Burst:         1,
	})

	wA1 := doGetWithIP(r, "/test", "10.0.0.10")
	wA2 := doGetWithIP(r, "/test", "10.0.0.10") // A 第 2 次 → 被拒
	wB1 := doGetWithIP(r, "/test", "10.0.0.11") // B 第 1 次 → 通过
	wB2 := doGetWithIP(r, "/test", "10.0.0.11") // B 第 2 次 → 被拒

	assert.Equal(t, http.StatusOK, wA1.Code)
	assert.Equal(t, http.StatusTooManyRequests, wA2.Code)
	assert.Equal(t, http.StatusOK, wB1.Code)
	assert.Equal(t, http.StatusTooManyRequests, wB2.Code)
}

func TestRateLimit_SkipperBypassesLimit(t *testing.T) {
	// /healthz 走 Skipper 直接放行，多次访问也不应被拒。
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 0.1,
		Burst:         1,
		Skipper: func(c *gin.Context) bool {
			return c.Request.URL.Path == "/healthz"
		},
	})

	for i := 0; i < 5; i++ {
		w := doGetWithIP(r, "/healthz", "10.0.0.20")
		assert.Equal(t, http.StatusOK, w.Code, "healthz request %d should pass", i)
	}

	// /test 仍受限：burst=1，第 2 个被拒。
	wT1 := doGetWithIP(r, "/test", "10.0.0.20")
	wT2 := doGetWithIP(r, "/test", "10.0.0.20")
	assert.Equal(t, http.StatusOK, wT1.Code)
	assert.Equal(t, http.StatusTooManyRequests, wT2.Code)
}

func TestRateLimit_BurstAllowsImmediateConcurrency(t *testing.T) {
	// burst=10 应允许 10 个 token 同时被立即消耗。
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 0.01, // 几乎不补充新 token，便于精确观察 burst 行为
		Burst:         10,
	})

	passed := 0
	rejected := 0
	for i := 0; i < 12; i++ {
		w := doGetWithIP(r, "/test", "10.0.0.30")
		switch w.Code {
		case http.StatusOK:
			passed++
		case http.StatusTooManyRequests:
			rejected++
		default:
			t.Fatalf("unexpected status %d", w.Code)
		}
	}

	// 前 10 请求应通过 burst，后 2 个请求应被拒（rate 太低，不足以补充新 token）。
	assert.Equal(t, 10, passed)
	assert.Equal(t, 2, rejected)
}

func TestRateLimit_ZeroRateActsAsNoOp(t *testing.T) {
	// RatePerSecond <= 0 视为关闭限流：所有请求都应放行。
	r := newRateLimitTestEngine(RateLimitConfig{RatePerSecond: 0, Burst: 1})

	for i := 0; i < 50; i++ {
		w := doGetWithIP(r, "/test", "10.0.0.40")
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass under no-op config", i)
	}
}

func TestRateLimit_DefaultBurstFromRate(t *testing.T) {
	// Burst 缺省时按 RatePerSecond 取整 + 1：rate=3 → burst=4。
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 3,
		Burst:         0,
	})

	passed := 0
	for i := 0; i < 4; i++ {
		w := doGetWithIP(r, "/test", "10.0.0.50")
		if w.Code == http.StatusOK {
			passed++
		}
	}
	assert.Equal(t, 4, passed, "default burst should equal int(rate)+1 = 4")

	// 第 5 个请求超过 burst，期望被拒。
	w5 := doGetWithIP(r, "/test", "10.0.0.50")
	assert.Equal(t, http.StatusTooManyRequests, w5.Code)
}

func TestRateLimit_RejectionResponseShape(t *testing.T) {
	r := newRateLimitTestEngine(RateLimitConfig{RatePerSecond: 0.1, Burst: 1})

	_ = doGetWithIP(r, "/test", "10.0.0.60") // 用掉 burst
	w := doGetWithIP(r, "/test", "10.0.0.60")

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"code":"RATE_LIMITED"`)
	assert.Contains(t, body, `"message"`)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestRateLimit_PrometheusRejectionsCounterIncrements(t *testing.T) {
	reg := prometheus.NewRegistry()
	r := newRateLimitTestEngine(RateLimitConfig{
		RatePerSecond: 0.1,
		Burst:         1,
		Registerer:    reg,
	})

	doGetWithIP(r, "/test", "10.0.0.70") // pass
	doGetWithIP(r, "/test", "10.0.0.70") // reject 1
	doGetWithIP(r, "/test", "10.0.0.70") // reject 2

	mfs, err := reg.Gather()
	assert.NoError(t, err)
	var found bool
	for _, mf := range mfs {
		if mf.GetName() != "http_ratelimit_rejections_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if m.GetCounter().GetValue() >= 2 {
				found = true
			}
		}
	}
	assert.True(t, found, "expected counter http_ratelimit_rejections_total to record at least 2 rejections")
}

func TestRateLimit_RecoversAfterTokenRefill(t *testing.T) {
	// 较高 rate + burst=1，等待 refill 后下一次访问应再次通过。
	r := newRateLimitTestEngine(RateLimitConfig{RatePerSecond: 50, Burst: 1})

	w1 := doGetWithIP(r, "/test", "10.0.0.80")
	w2 := doGetWithIP(r, "/test", "10.0.0.80") // 立即第二次 → 大概率被拒（burst=1，refill 还没完成）
	assert.Equal(t, http.StatusOK, w1.Code)
	// rate=50/s 意味着每 20ms 补充一个 token；睡 50ms 足够。
	time.Sleep(50 * time.Millisecond)
	w3 := doGetWithIP(r, "/test", "10.0.0.80")
	assert.Equal(t, http.StatusOK, w3.Code, "should recover after token refill")
	// w2 不强断言，因 limiter 可能在极快机器上恰好补充了 token；记录用便于调试。
	_ = w2
}
