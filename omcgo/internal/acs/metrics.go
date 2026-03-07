package acs

import "github.com/prometheus/client_golang/prometheus"

// ACSMetrics holds all Prometheus metrics for the ACS engine.
type ACSMetrics struct {
	ActiveSessions   prometheus.Gauge
	InformTotal      *prometheus.CounterVec
	RPCDuration      *prometheus.HistogramVec
	RPCErrorsTotal   *prometheus.CounterVec
	SessionDuration  prometheus.Histogram
}

// NewACSMetrics creates and registers ACS metrics.
func NewACSMetrics(reg prometheus.Registerer) *ACSMetrics {
	m := &ACSMetrics{
		ActiveSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_active_sessions",
			Help: "Current number of active TR069 sessions",
		}),
		InformTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_inform_total",
			Help: "Total number of Inform messages received",
		}, []string{"event_type"}),
		RPCDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "acs_rpc_duration_seconds",
			Help:    "Duration of RPC method execution",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		RPCErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_rpc_errors_total",
			Help: "Total number of RPC errors",
		}, []string{"method"}),
		SessionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "acs_session_duration_seconds",
			Help:    "Duration of complete TR069 sessions",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		}),
	}

	reg.MustRegister(
		m.ActiveSessions,
		m.InformTotal,
		m.RPCDuration,
		m.RPCErrorsTotal,
		m.SessionDuration,
	)

	return m
}
