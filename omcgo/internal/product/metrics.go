package product

import "github.com/prometheus/client_golang/prometheus"

// registryMetrics 聚合 ProductRegistry 的 Prometheus 指标（设计 §4.3 + 实施计划 §2.A）。
//
// 命名遵循已有 P1-06 模块约定（device / alarm / task），全部以 product_registry_ 前缀。
// nil 实例由 newRegistryMetrics(nil) 返回安全空实现，避免单元测试强依赖 prometheus.Registerer。
type registryMetrics struct {
	matchTotal         *prometheus.CounterVec // labels: result=hit|orphan
	matchDuration      prometheus.Histogram
	cacheHitTotal      *prometheus.CounterVec // labels: layer=L1|L2|miss（GetProductByID 三级缓存）
	classCacheHitTotal *prometheus.CounterVec // labels: layer=L1|L2|miss（MatchProductClass 二级缓存，T-0173）
	refreshTotal       *prometheus.CounterVec // labels: result=ok|err
	patternSkipTotal   prometheus.Counter     // #17: Refresh 期正则编译失败被跳过的 pattern 数（坏正则可观测）
}

// NewRegistryMetrics 注册并返回 Registry 用的指标集合。
//
// 传入 nil 时返回一个内部已注册到匿名 Registry 的实例（零依赖、可丢弃），
// 测试场景与单元 main 启动均可用。
func NewRegistryMetrics(reg prometheus.Registerer) *registryMetrics {
	m := &registryMetrics{
		matchTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "product_registry_match_total",
				Help: "Total ProductRegistry productClass routing attempts.",
			},
			[]string{"result"},
		),
		matchDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "product_registry_match_duration_seconds",
				Help:    "ProductRegistry productClass match duration in seconds.",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
			},
		),
		cacheHitTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "product_registry_cache_hit_total",
				Help: "ProductRegistry GetProductByID cache hit by layer.",
			},
			[]string{"layer"},
		),
		classCacheHitTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "product_registry_class_cache_hit_total",
				Help: "ProductRegistry MatchProductClass cache hit by layer (T-0173).",
			},
			[]string{"layer"},
		),
		refreshTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "product_registry_refresh_total",
				Help: "ProductRegistry Refresh outcomes.",
			},
			[]string{"result"},
		),
		patternSkipTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "product_registry_pattern_skip_total",
				Help: "ProductRegistry patterns skipped during Refresh due to regex compile failure (#17).",
			},
		),
	}

	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	reg.MustRegister(m.matchTotal, m.matchDuration, m.cacheHitTotal, m.classCacheHitTotal, m.refreshTotal, m.patternSkipTotal)
	return m
}

// matchHit / matchOrphan / cacheHit / refreshOK / refreshErr 是热路径捷径，
// 避免在 registry.go 处处写字符串字面量并防笔误。

func (m *registryMetrics) matchHit()    { m.matchTotal.WithLabelValues("hit").Inc() }
func (m *registryMetrics) matchOrphan() { m.matchTotal.WithLabelValues("orphan").Inc() }
func (m *registryMetrics) cacheHit(layer string) {
	m.cacheHitTotal.WithLabelValues(layer).Inc()
}
func (m *registryMetrics) classCacheHit(layer string) {
	m.classCacheHitTotal.WithLabelValues(layer).Inc()
}
func (m *registryMetrics) refreshOK()  { m.refreshTotal.WithLabelValues("ok").Inc() }
func (m *registryMetrics) refreshErr() { m.refreshTotal.WithLabelValues("err").Inc() }

// patternSkip 记录一条因正则编译失败被跳过的 pattern（#17：坏正则不再静默吞掉）。
func (m *registryMetrics) patternSkip() { m.patternSkipTotal.Inc() }
