package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics returns a Gin middleware that records request metrics.
// If a custom registerer is provided, metrics are registered there;
// otherwise they use the default prometheus registry.
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
