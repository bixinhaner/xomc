package parammodel

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// registryMetrics 聚合 ParamRegistry / Translator 的 Prometheus 指标（设计 §1.4-§1.6 + 实施计划 §2.A.P2-02）。
//
// 命名遵循 P2-01 ProductRegistry 既有约定：param_registry_* 与 param_translator_*。
// nil reg 由 NewRegistryMetrics(nil) 创建匿名 Registry，安全可丢弃，单元测试与 main 启动均可用。
type registryMetrics struct {
	lookupTotal             *prometheus.CounterVec // labels: source=discovered|default|miss
	lookupDuration          prometheus.Histogram
	cacheHitTotal           *prometheus.CounterVec // labels: layer=L1|L2|DB|miss, set=default|discovered
	refreshTotal            *prometheus.CounterVec // labels: result=ok|err
	translateTotal          *prometheus.CounterVec // labels: direction=to_private|to_standard, result=hit|miss
	invalidPlaceholderTotal *prometheus.CounterVec // labels: param_model
}

// NewRegistryMetrics 注册并返回 ParamRegistry/Translator 用的指标集合。
//
// 传入 nil 时返回内部匿名 prometheus.Registry，零依赖、可丢弃。
func NewRegistryMetrics(reg prometheus.Registerer) *registryMetrics {
	m := &registryMetrics{
		lookupTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "param_registry_lookup_total",
				Help: "Total ParamRegistry mapping lookups.",
			},
			[]string{"source"},
		),
		lookupDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "param_registry_lookup_duration_seconds",
				Help:    "ParamRegistry mapping lookup duration in seconds.",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
			},
		),
		cacheHitTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "param_registry_cache_hit_total",
				Help: "ParamRegistry cache hits by layer and mapping set.",
			},
			[]string{"layer", "set"},
		),
		refreshTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "param_registry_refresh_total",
				Help: "ParamRegistry Refresh outcomes.",
			},
			[]string{"result"},
		),
		translateTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "param_translator_translate_total",
				Help: "Total Translator path translations.",
			},
			[]string{"direction", "result"},
		),
		invalidPlaceholderTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "param_translator_invalid_placeholder_total",
				Help: "Mappings skipped due to {i} placeholder count mismatch.",
			},
			[]string{"param_model"},
		),
	}

	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	m.lookupTotal = registerCounterVec(reg, m.lookupTotal)
	m.lookupDuration = registerHistogram(reg, m.lookupDuration)
	m.cacheHitTotal = registerCounterVec(reg, m.cacheHitTotal)
	m.refreshTotal = registerCounterVec(reg, m.refreshTotal)
	m.translateTotal = registerCounterVec(reg, m.translateTotal)
	m.invalidPlaceholderTotal = registerCounterVec(reg, m.invalidPlaceholderTotal)
	return m
}

func registerCounterVec(reg prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := reg.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return collector
}

func registerHistogram(reg prometheus.Registerer, collector prometheus.Histogram) prometheus.Histogram {
	if err := reg.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(prometheus.Histogram); ok {
				return existing
			}
		}
		panic(err)
	}
	return collector
}

// 热路径捷径——避免在调用点反复字符串字面量并防笔误。

func (m *registryMetrics) lookupDiscovered() { m.lookupTotal.WithLabelValues("discovered").Inc() }
func (m *registryMetrics) lookupDefault()    { m.lookupTotal.WithLabelValues("default").Inc() }
func (m *registryMetrics) lookupMiss()       { m.lookupTotal.WithLabelValues("miss").Inc() }

func (m *registryMetrics) cacheHit(layer, set string) {
	m.cacheHitTotal.WithLabelValues(layer, set).Inc()
}

func (m *registryMetrics) refreshOK()  { m.refreshTotal.WithLabelValues("ok").Inc() }
func (m *registryMetrics) refreshErr() { m.refreshTotal.WithLabelValues("err").Inc() }

func (m *registryMetrics) translateHit(direction string) {
	m.translateTotal.WithLabelValues(direction, "hit").Inc()
}
func (m *registryMetrics) translateMiss(direction string) {
	m.translateTotal.WithLabelValues(direction, "miss").Inc()
}

func (m *registryMetrics) invalidPlaceholder(paramModel string) {
	m.invalidPlaceholderTotal.WithLabelValues(paramModel).Inc()
}
