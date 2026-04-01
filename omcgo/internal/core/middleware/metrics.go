package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics 返回一个 Gin 中间件，自动统计每个接口的请求次数和延迟分布。
// 指标名称：http_requests_total 和 http_request_duration_seconds，标签为 method/path/status。
// 使用场景：全局中间件，自动为所有路由采集 Prometheus 指标，由 /metrics 端口暴露。
// registerer 为空时使用默认 prometheus.DefaultRegisterer，
// 传入自定义 Registry 则将指标隔离到单独 Registry（推荐）。
func PrometheusMetrics(registerers ...prometheus.Registerer) gin.HandlerFunc {
	var reg prometheus.Registerer
	if len(registerers) > 0 && registerers[0] != nil {
		reg = registerers[0]
	} else {
		reg = prometheus.DefaultRegisterer
	}

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	reg.MustRegister(requestsTotal, requestDuration)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		status := strconv.Itoa(c.Writer.Status())
		requestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		requestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
