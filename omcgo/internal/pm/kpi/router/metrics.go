package router

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics 是 Router 的可选 Prometheus 指标。
// 调用 NewMetrics(nil) 得到匿名 Registry，仍能消费但不导出到 /metrics。
type Metrics struct {
	hits   *prometheus.CounterVec // by tier=L1/L2/DB
	misses *prometheus.CounterVec // by reason=orphan/invalid_metadata
}

// NewMetrics 构造指标。reg 传 nil 会落入匿名 Registry（仅供测试）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	m := &Metrics{
		hits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "omc",
			Subsystem: "kpi_router",
			Name:      "lookup_hits_total",
			Help:      "KPI Router lookups by cache tier (L1/L2/DB).",
		}, []string{"tier"}),
		misses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "omc",
			Subsystem: "kpi_router",
			Name:      "lookup_miss_total",
			Help:      "KPI Router lookups that returned no route (orphan / invalid metadata).",
		}, []string{"reason"}),
	}
	reg.MustRegister(m.hits, m.misses)
	return m
}

func (m *Metrics) hit(tier string) {
	if m == nil {
		return
	}
	m.hits.WithLabelValues(tier).Inc()
}

func (m *Metrics) miss(reason string) {
	if m == nil {
		return
	}
	m.misses.WithLabelValues(reason).Inc()
}
