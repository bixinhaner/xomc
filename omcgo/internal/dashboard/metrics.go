package dashboard

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	queryDuration *prometheus.HistogramVec
	inflight      prometheus.Gauge
	timeout       *prometheus.CounterVec
	rejected      *prometheus.CounterVec
	cache         *prometheus.CounterVec
	coalesced     *prometheus.CounterVec
	missing       *prometheus.CounterVec
	incomplete    *prometheus.CounterVec
	rollupLag     *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		return &Metrics{}
	}
	metrics := &Metrics{
		queryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dashboard_kpi_query_duration_seconds",
			Help:    "Dashboard KPI rollup query duration.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1, 2, 3},
		}, []string{"endpoint", "granularity", "status"}),
		inflight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "dashboard_kpi_query_inflight",
			Help: "Current Dashboard KPI queries executing against TSDB.",
		}),
		timeout: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dashboard_kpi_query_timeout_total",
			Help: "Dashboard KPI queries that exceeded their deadline.",
		}, []string{"endpoint"}),
		rejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dashboard_kpi_query_rejected_total",
			Help: "Dashboard KPI queries rejected by the concurrency guard.",
		}, []string{"endpoint"}),
		cache: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dashboard_kpi_query_cache_total",
			Help: "Dashboard KPI query cache outcomes.",
		}, []string{"endpoint", "result"}),
		coalesced: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dashboard_kpi_query_coalesced_total",
			Help: "Dashboard KPI requests coalesced with an in-flight query.",
		}, []string{"endpoint"}),
		missing: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dashboard_kpi_missing_result_total",
			Help: "Missing Dashboard final published network rollup results.",
		}, []string{"technology", "granularity"}),
		incomplete: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dashboard_kpi_incomplete_window_total",
			Help: "Incomplete Dashboard network rollup windows.",
		}, []string{"technology", "granularity"}),
		rollupLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "pm_network_rollup_lag_seconds",
			Help: "Age of the latest published network rollup window.",
		}, []string{"technology", "granularity"}),
	}
	reg.MustRegister(
		metrics.queryDuration,
		metrics.inflight,
		metrics.timeout,
		metrics.rejected,
		metrics.cache,
		metrics.coalesced,
		metrics.missing,
		metrics.incomplete,
		metrics.rollupLag,
	)
	return metrics
}

func (m *Metrics) ObserveQuery(endpoint, granularity, status string, duration time.Duration) {
	if m == nil || m.queryDuration == nil {
		return
	}
	m.queryDuration.WithLabelValues(endpoint, granularity, status).Observe(duration.Seconds())
}

func (m *Metrics) ObserveGuardResult(key, result string) {
	if m == nil {
		return
	}
	endpoint := key
	if index := strings.IndexByte(key, ':'); index >= 0 {
		endpoint = key[:index]
	}
	switch result {
	case "timeout":
		if m.timeout != nil {
			m.timeout.WithLabelValues(endpoint).Inc()
		}
	case "rejected":
		if m.rejected != nil {
			m.rejected.WithLabelValues(endpoint).Inc()
		}
	case "coalesced":
		if m.coalesced != nil {
			m.coalesced.WithLabelValues(endpoint).Inc()
		}
	default:
		if m.cache != nil {
			m.cache.WithLabelValues(endpoint, result).Inc()
		}
	}
}

func (m *Metrics) ObserveGuardQuery(key, status string, duration time.Duration) {
	if m == nil {
		return
	}
	parts := strings.Split(key, ":")
	endpoint := parts[0]
	granularity := "hourly"
	if endpoint == "series" && len(parts) > 2 {
		granularity = parts[2]
	}
	m.ObserveQuery(endpoint, granularity, status, duration)
}

func (m *Metrics) SetInflight(value float64) {
	if m != nil && m.inflight != nil {
		m.inflight.Set(value)
	}
}

func (m *Metrics) ObserveMissing(technology, granularity string, count float64) {
	if m != nil && m.missing != nil && count > 0 {
		m.missing.WithLabelValues(technology, granularity).Add(count)
	}
}

func (m *Metrics) ObserveIncomplete(technology, granularity string, count float64) {
	if m != nil && m.incomplete != nil && count > 0 {
		m.incomplete.WithLabelValues(technology, granularity).Add(count)
	}
}

func (m *Metrics) SetRollupLag(technology, granularity string, seconds float64) {
	if m != nil && m.rollupLag != nil {
		m.rollupLag.WithLabelValues(technology, granularity).Set(seconds)
	}
}
