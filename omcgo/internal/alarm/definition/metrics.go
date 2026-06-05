package definition

import "github.com/prometheus/client_golang/prometheus"

// registryMetrics 聚合 AlarmDefinition Registry 的 Prometheus 指标（设计 §3.3 + §3.5）。
//
// 命名约定 alarm_def_registry_* + 接收路径 alarm_unknown_total。
type registryMetrics struct {
	lookupTotal  *prometheus.CounterVec // labels: result=hit|miss
	refreshTotal *prometheus.CounterVec // labels: result=ok|err
	unknownTotal *prometheus.CounterVec // labels: action=dropped|kept_as_unknown
}

// NewRegistryMetrics 注册并返回 Registry 与 fallback 路径所用的指标集合。
func NewRegistryMetrics(reg prometheus.Registerer) *registryMetrics {
	m := &registryMetrics{
		lookupTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alarm_def_registry_lookup_total",
				Help: "Total alarm-definition Registry lookups by result.",
			},
			[]string{"result"},
		),
		refreshTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alarm_def_registry_refresh_total",
				Help: "AlarmDefinition Registry Refresh outcomes.",
			},
			[]string{"result"},
		),
		unknownTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alarm_unknown_total",
				Help: "Alarms whose identifier missed AlarmDefinition Registry; labelled by fallback action.",
			},
			[]string{"action"},
		),
	}

	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	reg.MustRegister(m.lookupTotal, m.refreshTotal, m.unknownTotal)
	return m
}

// 热路径捷径
func (m *registryMetrics) lookupHit()  { m.lookupTotal.WithLabelValues("hit").Inc() }
func (m *registryMetrics) lookupMiss() { m.lookupTotal.WithLabelValues("miss").Inc() }
func (m *registryMetrics) refreshOK()  { m.refreshTotal.WithLabelValues("ok").Inc() }
func (m *registryMetrics) refreshErr() { m.refreshTotal.WithLabelValues("err").Inc() }

// UnknownDropped 与 UnknownKept 由 receiver 在 fallback 路径调用（导出供包外消费）。
func (m *registryMetrics) UnknownDropped() { m.unknownTotal.WithLabelValues("dropped").Inc() }
func (m *registryMetrics) UnknownKept()    { m.unknownTotal.WithLabelValues("kept_as_unknown").Inc() }

// Metrics 把内部 *registryMetrics 暴露给 receiver 等外部消费方使用 fallback 计数器。
//
// 不直接暴露字段以保持封装；只暴露 fallback 路径需要的两个动作。
func (r *Registry) Metrics() *registryMetrics { return r.metrics }
