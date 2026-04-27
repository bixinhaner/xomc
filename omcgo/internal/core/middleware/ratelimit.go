package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimitConfig 配置每客户端限流中间件。
// 限流粒度为 per-IP（取自 c.ClientIP()，已经过 Gin 的 ProxyTrust 处理）。
// 每个 IP 维护独立 token bucket，互不干扰；空闲 IP 会按 IdleTTL 自动回收，
// 防止长尾内存泄漏。
//
// 字段：
//   - RatePerSecond ：每秒补充令牌数（rate.Limit）；<=0 表示不限流（中间件成为 no-op）。
//   - Burst         ：桶容量（瞬时最大并发）；<=0 时按 RatePerSecond 取整后再 +1（保底）。
//   - IdleTTL       ：limiter 闲置多久后被清扫；<=0 时使用默认 5 分钟。
//   - SweepInterval ：清扫 goroutine 周期；<=0 时使用默认 1 分钟。
//   - Skipper       ：返回 true 的请求跳过限流（如 /healthz、/metrics）。
//   - Registerer    ：Prometheus 注册器；nil 时使用 prometheus.DefaultRegisterer。
type RateLimitConfig struct {
	RatePerSecond float64
	Burst         int
	IdleTTL       time.Duration
	SweepInterval time.Duration
	Skipper       func(c *gin.Context) bool
	Registerer    prometheus.Registerer
}

const (
	defaultRateLimitIdleTTL       = 5 * time.Minute
	defaultRateLimitSweepInterval = 1 * time.Minute
)

// rateLimitEntry 保存单个客户端的 token bucket 与最近一次访问时间。
// lastSeen 使用 atomic 存储 unix-nano，读路径无锁。
type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

// RateLimit 返回一个 Gin 中间件，按客户端 IP 进行 token bucket 限流。
// 超限请求返回 429 + JSON `{"code":"RATE_LIMITED","message":"..."}`，
// 同时累加 Prometheus counter http_ratelimit_rejections_total{path}
// 并写一行 zap warn（含 client_ip / path）。
//
// 使用场景：App 服务全局中间件，紧跟 RequestID/Recovery/Logging 之后注册，
// 防止单 IP 洪泛拖垮后端。需要分布式协同的限流（多实例聚合）应另行实现
// Redis 版本，本中间件仅做单实例进程内防护。
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	// 0 / 负数 rate 视为关闭限流：返回 pass-through 中间件，避免上游错配置直接全拒。
	if cfg.RatePerSecond <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	burst := cfg.Burst
	if burst <= 0 {
		burst = int(cfg.RatePerSecond) + 1
	}

	idleTTL := cfg.IdleTTL
	if idleTTL <= 0 {
		idleTTL = defaultRateLimitIdleTTL
	}

	sweepInterval := cfg.SweepInterval
	if sweepInterval <= 0 {
		sweepInterval = defaultRateLimitSweepInterval
	}

	rejections := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_ratelimit_rejections_total",
			Help: "Total number of HTTP requests rejected by per-IP rate limiter",
		},
		[]string{"path"},
	)

	reg := cfg.Registerer
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	// 使用 Register 而非 MustRegister：在多实例 / 测试场景下若 counter 已被注册，
	// 复用已注册的 *CounterVec，避免 panic。
	if err := reg.Register(rejections); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
				rejections = existing
			}
		}
	}

	limit := rate.Limit(cfg.RatePerSecond)
	limiters := &sync.Map{}

	getLimiter := func(key string, now time.Time) *rate.Limiter {
		if v, ok := limiters.Load(key); ok {
			entry := v.(*rateLimitEntry)
			entry.lastSeen.Store(now.UnixNano())
			return entry.limiter
		}
		entry := &rateLimitEntry{limiter: rate.NewLimiter(limit, burst)}
		entry.lastSeen.Store(now.UnixNano())
		actual, loaded := limiters.LoadOrStore(key, entry)
		if loaded {
			existing := actual.(*rateLimitEntry)
			existing.lastSeen.Store(now.UnixNano())
			return existing.limiter
		}
		return entry.limiter
	}

	// 周期清扫：移除超过 idleTTL 未访问的条目。简单方案，未来可换 LRU。
	// goroutine 在进程生命周期内常驻；模块化单体进程关闭即随之退出。
	go func() {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for now := range ticker.C {
			cutoff := now.Add(-idleTTL).UnixNano()
			limiters.Range(func(k, v any) bool {
				entry := v.(*rateLimitEntry)
				if entry.lastSeen.Load() < cutoff {
					limiters.Delete(k)
				}
				return true
			})
		}
	}()

	return func(c *gin.Context) {
		if cfg.Skipper != nil && cfg.Skipper(c) {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if clientIP == "" {
			// 取不到客户端 IP 时直接放行，避免误伤（例如内部 healthcheck 走 unix socket）。
			c.Next()
			return
		}

		limiter := getLimiter(clientIP, time.Now())
		if !limiter.Allow() {
			path := c.FullPath()
			if path == "" {
				path = c.Request.URL.Path
			}
			rejections.WithLabelValues(path).Inc()

			log := logger.L(c.Request.Context())
			log.Warn("ratelimit rejected",
				zap.String("client_ip", clientIP),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Float64("rate_per_second", cfg.RatePerSecond),
				zap.Int("burst", burst),
			)

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMITED",
				"message": "too many requests, please retry later",
			})
			return
		}

		c.Next()
	}
}
