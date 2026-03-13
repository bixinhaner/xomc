package monitor

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InfraMetrics holds infrastructure-level Prometheus metrics.
type InfraMetrics struct {
	DBQueryDuration    *prometheus.HistogramVec
	RedisOpDuration    *prometheus.HistogramVec
	NATSPublishTotal   *prometheus.CounterVec
	NATSConsumeTotal   *prometheus.CounterVec
}

// NewInfraMetrics registers and returns infrastructure metrics.
func NewInfraMetrics(reg prometheus.Registerer) *InfraMetrics {
	m := &InfraMetrics{
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		RedisOpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_operation_duration_seconds",
				Help:    "Redis operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		NATSPublishTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nats_publish_total",
				Help: "Total NATS messages published",
			},
			[]string{"subject"},
		),
		NATSConsumeTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nats_consume_total",
				Help: "Total NATS messages consumed",
			},
			[]string{"subject"},
		),
	}

	reg.MustRegister(m.DBQueryDuration, m.RedisOpDuration, m.NATSPublishTotal, m.NATSConsumeTotal)
	return m
}

// NewMetricsServer creates an HTTP server that serves Prometheus metrics.
func NewMetricsServer(port int) *http.Server {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}

// DefaultRegistry returns the default Prometheus registry, suitable for
// components that register metrics globally.
func DefaultRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return reg
}
