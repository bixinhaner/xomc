package acs

import "github.com/prometheus/client_golang/prometheus"

// ACSMetrics holds all Prometheus metrics for the ACS engine.
type ACSMetrics struct {
	ActiveSessions         prometheus.Gauge
	GlobalActiveSessions   prometheus.Gauge
	LocalTrackedSessions   prometheus.Gauge
	InformTotal            *prometheus.CounterVec
	RPCDuration            *prometheus.HistogramVec
	RPCErrorsTotal         *prometheus.CounterVec
	SessionDuration        prometheus.Histogram
	RateLimitRejected      prometheus.Counter
	AdmissionRejected      prometheus.Counter
	RateLimitDeviceCount   prometheus.Gauge
	PostSessionWakeTotal   prometheus.Counter
	UECountQueueDepth      prometheus.Gauge
	UECountQueueCapacity   prometheus.Gauge
	UECountEnqueueTotal    *prometheus.CounterVec
	UECountProcessTotal    *prometheus.CounterVec
	UECountProcessDuration prometheus.Histogram
}

// NewACSMetrics creates and registers ACS metrics.
func NewACSMetrics(reg prometheus.Registerer) *ACSMetrics {
	m := &ACSMetrics{
		ActiveSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_active_sessions",
			Help: "Deprecated compatibility alias for acs_global_active_sessions; current number of globally admitted TR069 sessions",
		}),
		GlobalActiveSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_global_active_sessions",
			Help: "Current number of globally admitted TR069 sessions from the shared admission controller",
		}),
		LocalTrackedSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_local_tracked_sessions",
			Help: "Current number of session IDs retained by this ACS process for up to five minutes; not real-time concurrency",
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
		RateLimitRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acs_rate_limit_rejected_total",
			Help: "Total number of requests rejected by per-device rate limiter",
		}),
		AdmissionRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acs_admission_rejected_total",
			Help: "Total number of Inform requests rejected by the global ACS session admission limit",
		}),
		RateLimitDeviceCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_rate_limit_device_count",
			Help: "Number of devices tracked by the rate limiter",
		}),
		PostSessionWakeTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acs_post_session_wake_total",
			Help: "Total number of post-session Connection Requests sent to wake devices with remaining commands",
		}),
		UECountQueueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_ue_count_queue_depth",
			Help: "Current number of devices waiting for asynchronous UE count scheduling",
		}),
		UECountQueueCapacity: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_ue_count_queue_capacity",
			Help: "Maximum number of devices accepted by the asynchronous UE count scheduler",
		}),
		UECountEnqueueTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_ue_count_enqueue_total",
			Help: "Total UE count scheduling requests by admission result",
		}, []string{"result"}),
		UECountProcessTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_ue_count_process_total",
			Help: "Total asynchronous UE count scheduling attempts by result",
		}, []string{"result"}),
		UECountProcessDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "acs_ue_count_process_duration_seconds",
			Help:    "Duration of asynchronous UE count scheduling attempts",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 3},
		}),
	}

	reg.MustRegister(
		m.ActiveSessions,
		m.GlobalActiveSessions,
		m.LocalTrackedSessions,
		m.InformTotal,
		m.RPCDuration,
		m.RPCErrorsTotal,
		m.SessionDuration,
		m.RateLimitRejected,
		m.AdmissionRejected,
		m.RateLimitDeviceCount,
		m.PostSessionWakeTotal,
		m.UECountQueueDepth,
		m.UECountQueueCapacity,
		m.UECountEnqueueTotal,
		m.UECountProcessTotal,
		m.UECountProcessDuration,
	)

	return m
}
