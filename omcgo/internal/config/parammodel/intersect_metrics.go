package parammodel

import "github.com/prometheus/client_golang/prometheus"

// intersectMetrics 聚合 IntersectService 的 Prometheus 指标（设计 §1.8 + 实施计划 §2.A.P2-03）。
//
// 命名约定 param_intersect_*：与 param_registry_* / param_translator_* 平行。
type intersectMetrics struct {
	intersectTotal         *prometheus.CounterVec // labels: outcome=ok|err_*
	duration               prometheus.Histogram
	matched                prometheus.Counter
	defaultsMissing        prometheus.Counter
	uploadedExtras         prometheus.Counter
	dataTypeOverrideTotal  prometheus.Counter
	invalidateErrTotal     prometheus.Counter
}

// NewIntersectMetrics 注册并返回 IntersectService 指标集合。
//
// 传入 nil 时使用内部匿名 prometheus.Registry（测试 / 退化）。
func NewIntersectMetrics(reg prometheus.Registerer) *intersectMetrics {
	m := &intersectMetrics{
		intersectTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "param_intersect_total",
				Help: "Total ParamIntersect runs by outcome.",
			},
			[]string{"outcome"},
		),
		duration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "param_intersect_duration_seconds",
				Help:    "ParamIntersect end-to-end duration in seconds.",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
			},
		),
		matched: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "param_intersect_matched_rows_total",
				Help: "Total rows written to discovered_param_mappings via intersect.",
			},
		),
		defaultsMissing: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "param_intersect_defaults_missing_total",
				Help: "Default mappings whose privatePath is absent from CPE upload.",
			},
		),
		uploadedExtras: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "param_intersect_uploaded_extras_total",
				Help: "CPE-uploaded entries with no matching default mapping (discarded).",
			},
		),
		dataTypeOverrideTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "param_intersect_datatype_override_rejected_total",
				Help: "device_attrs_override.data_type=true requests rejected at intersect.",
			},
		),
		invalidateErrTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "param_intersect_invalidate_err_total",
				Help: "Cache invalidation failures after intersect (non-fatal).",
			},
		),
	}

	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	reg.MustRegister(
		m.intersectTotal,
		m.duration,
		m.matched,
		m.defaultsMissing,
		m.uploadedExtras,
		m.dataTypeOverrideTotal,
		m.invalidateErrTotal,
	)
	return m
}

func (m *intersectMetrics) outcome(label string) {
	m.intersectTotal.WithLabelValues(label).Inc()
}

func (m *intersectMetrics) dataTypeOverrideRejected() {
	m.dataTypeOverrideTotal.Inc()
}

func (m *intersectMetrics) invalidateErr() {
	m.invalidateErrTotal.Inc()
}
