package connreq

import "github.com/prometheus/client_golang/prometheus"

// DispatcherMetrics holds Prometheus metrics for Connection Request dispatching.
type DispatcherMetrics struct {
	SentTotal       *prometheus.CounterVec
	DurationSeconds *prometheus.HistogramVec
}

// NewDispatcherMetrics creates and registers Connection Request dispatcher metrics.
func NewDispatcherMetrics(reg prometheus.Registerer) *DispatcherMetrics {
	m := &DispatcherMetrics{
		SentTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_connection_request_sent_total",
			Help: "Total Connection Request dispatch outcomes; method=none,result=unavailable means the device will use its next periodic Inform",
		}, []string{"method", "result"}),
		DurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "acs_connection_request_duration_seconds",
			Help:    "Duration of Connection Request send operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"method"}),
	}

	reg.MustRegister(m.SentTotal, m.DurationSeconds)
	return m
}
